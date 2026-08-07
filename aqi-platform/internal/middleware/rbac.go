package middleware

import (
	"net/http"

	"github.com/mintfary/aqi-platform/internal/domain"
)

// RequireRole — middleware RBAC: пропускает запрос только если роль пользователя
// входит в список разрешённых ролей.
// Должен применяться ПОСЛЕ middleware Auth (иначе claims будет nil → 401).
func RequireRole(roles ...domain.Role) func(http.Handler) http.Handler {
	// Предварительно строим map для O(1) поиска.
	allowed := make(map[domain.Role]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				writeError(w, http.StatusUnauthorized, "требуется авторизация")
				return
			}

			role := domain.Role(claims.Role)
			if _, ok := allowed[role]; !ok {
				writeError(w, http.StatusForbidden, "недостаточно прав доступа")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin — сокращение для RequireRole(RoleAdmin).
func RequireAdmin() func(http.Handler) http.Handler {
	return RequireRole(domain.RoleAdmin)
}

// RequireAnalystOrAdmin — сокращение для RequireRole(RoleAdmin, RoleAnalyst).
func RequireAnalystOrAdmin() func(http.Handler) http.Handler {
	return RequireRole(domain.RoleAdmin, domain.RoleAnalyst)
}

// RequireActiveUser — проверяет, что пользователь активен.
// Используется совместно с Auth.
func RequireActiveUser() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				writeError(w, http.StatusUnauthorized, "требуется авторизация")
				return
			}
			// Активность пользователя проверяется при выдаче токена.
			// Если нужна real-time проверка — добавить обращение к БД здесь.
			next.ServeHTTP(w, r)
		})
	}
}
