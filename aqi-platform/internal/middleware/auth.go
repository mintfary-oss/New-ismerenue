// Package middleware содержит HTTP-middleware для Chi-роутера.
// Каждый middleware — самодостаточная функция, не зависящая от других.
package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mintfary/aqi-platform/internal/domain"
)

// contextKey — тип ключей для context.WithValue (предотвращает коллизии).
type contextKey string

const (
	ctxKeyClaims contextKey = "claims"
	ctxKeyUserID contextKey = "user_id"
)

// Claims — payload JWT токена.
type Claims struct {
	jwt.RegisteredClaims
	UserID   string `json:"uid"`
	Role     string `json:"role"`
	TokenID  string `json:"jti"` // уникальный ID токена (для блеклиста)
}

// TokenValidator — интерфейс для проверки JWT и блеклиста.
// Позволяет подменять реализацию в тестах.
type TokenValidator interface {
	ValidateAccessToken(tokenString string) (*Claims, error)
	IsBlacklisted(ctx context.Context, tokenID string) (bool, error)
}

// Auth — middleware обязательной JWT-аутентификации.
// Если токен невалиден или отсутствует → 401.
func Auth(validator TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				writeError(w, http.StatusUnauthorized, "требуется авторизация")
				return
			}

			claims, err := validator.ValidateAccessToken(token)
			if err != nil {
				switch {
				case errors.Is(err, jwt.ErrTokenExpired):
					writeError(w, http.StatusUnauthorized, "токен истёк")
				case errors.Is(err, jwt.ErrTokenSignatureInvalid):
					writeError(w, http.StatusUnauthorized, "недействительный токен")
				default:
					writeError(w, http.StatusUnauthorized, "ошибка авторизации")
				}
				return
			}

			// Проверка блеклиста отозванных токенов.
			blacklisted, err := validator.IsBlacklisted(r.Context(), claims.TokenID)
			if err != nil {
				slog.Error("проверка блеклиста токенов", "err", err)
				// При ошибке Redis — разрешаем доступ (fail-open), логируем.
			}
			if blacklisted {
				writeError(w, http.StatusUnauthorized, "токен отозван")
				return
			}

			// Передаём claims в контекст запроса.
			ctx := context.WithValue(r.Context(), ctxKeyClaims, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth — middleware необязательной аутентификации.
// Используется для публичных эндпоинтов, которые могут вести себя по-разному
// в зависимости от наличия авторизации (например, виджет с доп. данными для admin).
func OptionalAuth(validator TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token != "" {
				if claims, err := validator.ValidateAccessToken(token); err == nil {
					ctx := context.WithValue(r.Context(), ctxKeyClaims, claims)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// APITokenAuth — middleware для аутентификации по API-токену (Bearer в заголовке).
// Используется для интеграций внешних систем.
type APITokenChecker interface {
	ValidateAPIToken(ctx context.Context, rawToken string) (*domain.User, error)
}

func APITokenAuth(checker APITokenChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				writeError(w, http.StatusUnauthorized, "требуется API-токен")
				return
			}

			user, err := checker.ValidateAPIToken(r.Context(), token)
			if err != nil || user == nil {
				writeError(w, http.StatusUnauthorized, "недействительный API-токен")
				return
			}

			// Оборачиваем user как Claims для единого интерфейса.
			claims := &Claims{
				UserID: user.ID.String(),
				Role:   user.Role.String(),
			}
			ctx := context.WithValue(r.Context(), ctxKeyClaims, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext извлекает JWT claims из контекста запроса.
// Возвращает nil, если аутентификация не выполнена.
func ClaimsFromContext(ctx context.Context) *Claims {
	v, _ := ctx.Value(ctxKeyClaims).(*Claims)
	return v
}

// extractBearerToken извлекает токен из заголовка Authorization: Bearer <token>.
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return ""
	}
	return strings.TrimSpace(auth[len(prefix):])
}

// writeError записывает JSON-ошибку (локальная копия, чтобы не импортировать handler).
func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(`{"error":"` + escapeJSON(msg) + `","ts":"` + time.Now().UTC().Format(time.RFC3339) + `"}`))
}

// escapeJSON минимально экранирует строку для JSON-вставки без encoding/json.
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
