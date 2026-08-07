// Package service содержит бизнес-логику приложения.
// Слой service не зависит от HTTP — только от domain и repository.
package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"

	"github.com/mintfary/aqi-platform/internal/config"
	"github.com/mintfary/aqi-platform/internal/domain"
	"github.com/mintfary/aqi-platform/internal/middleware"
)

// UserRepository — интерфейс доступа к данным пользователей.
// Позволяет подменять реализацию в тестах (mock).
type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	Create(ctx context.Context, in domain.CreateUserInput) (*domain.User, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error
}

// TokenStore — хранилище отозванных токенов (Redis).
type TokenStore interface {
	Add(ctx context.Context, tokenID string, expiry time.Time) error
	IsBlacklisted(ctx context.Context, tokenID string) (bool, error)
}

// LoginAttemptStore — счётчик попыток входа (Redis).
type LoginAttemptStore interface {
	IsLocked(ctx context.Context, email string) (bool, error)
	Increment(ctx context.Context, email string) (int, error)
	Reset(ctx context.Context, email string) error
}

// AuthService — сервис аутентификации и авторизации.
type AuthService struct {
	users    UserRepository
	tokens   TokenStore
	attempts LoginAttemptStore
	cfg      config.AuthConfig
	logger   *slog.Logger
}

// NewAuthService создаёт сервис аутентификации.
func NewAuthService(
	users UserRepository,
	tokens TokenStore,
	attempts LoginAttemptStore,
	cfg config.AuthConfig,
	logger *slog.Logger,
) *AuthService {
	return &AuthService{
		users:    users,
		tokens:   tokens,
		attempts: attempts,
		cfg:      cfg,
		logger:   logger,
	}
}

// ── Типы токенов ────────────────────────────────────────────────────────────

// TokenPair — пара access + refresh токенов.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// ── Вход / выход ────────────────────────────────────────────────────────────

// LoginInput — данные для входа.
type LoginInput struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=1"`
}

// Login выполняет аутентификацию пользователя.
// Реализует защиту от timing-атак через constant-time сравнение.
func (s *AuthService) Login(ctx context.Context, in LoginInput) (*TokenPair, error) {
	// Проверяем блокировку аккаунта (brute force защита).
	locked, err := s.attempts.IsLocked(ctx, in.Email)
	if err != nil {
		s.logger.Error("проверка блокировки", "email", in.Email, "err", err)
	}
	if locked {
		return nil, domain.NewAppError(429, "аккаунт временно заблокирован из-за множества неудачных попыток входа", domain.ErrAccountLocked)
	}

	// Ищем пользователя по email.
	user, err := s.users.GetByEmail(ctx, in.Email)
	if err != nil || user == nil {
		// Выполняем «холостое» хеширование для защиты от timing-атаки.
		// Это скрывает факт существования/отсутствия пользователя.
		_ = s.verifyPassword("dummy_password_for_timing_protection",
			"$argon2id$v=19$m=65536,t=3,p=4$AAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
		_, _ = s.attempts.Increment(ctx, in.Email)
		return nil, domain.NewAppError(401, "неверный логин или пароль", domain.ErrInvalidCredentials)
	}

	if !user.IsActive {
		return nil, domain.NewAppError(401, "аккаунт отключён", domain.ErrAccountDisabled)
	}

	// Проверяем пароль.
	if !s.verifyPassword(in.Password, user.Password) {
		count, _ := s.attempts.Increment(ctx, in.Email)
		s.logger.Warn("неудачная попытка входа",
			"email", in.Email,
			"attempts", count,
			"max", s.cfg.MaxLoginAttempts,
		)
		return nil, domain.NewAppError(401, "неверный логин или пароль", domain.ErrInvalidCredentials)
	}

	// Сброс счётчика после успешного входа.
	if err = s.attempts.Reset(ctx, in.Email); err != nil {
		s.logger.Error("сброс счётчика попыток", "err", err)
	}

	s.logger.Info("успешный вход", "user_id", user.ID, "role", user.Role)

	return s.generateTokenPair(user)
}

// Logout добавляет текущий access token в блеклист.
func (s *AuthService) Logout(ctx context.Context, claims *middleware.Claims) error {
	if claims == nil {
		return nil
	}
	expiry, err := claims.GetExpirationTime()
	if err != nil || expiry == nil {
		return nil
	}
	return s.tokens.Add(ctx, claims.TokenID, expiry.Time)
}

// ── Управление токенами ─────────────────────────────────────────────────────

// RefreshTokens обновляет пару токенов по refresh token.
func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := s.parseToken(refreshToken, "refresh")
	if err != nil {
		return nil, domain.NewAppError(401, "недействительный refresh token", err)
	}

	// Проверяем блеклист.
	blacklisted, err := s.tokens.IsBlacklisted(ctx, claims.TokenID)
	if err != nil {
		s.logger.Error("проверка блеклиста при refresh", "err", err)
	}
	if blacklisted {
		return nil, domain.NewAppError(401, "токен отозван", domain.ErrTokenInvalid)
	}

	// Отзываем старый refresh token.
	exp, _ := claims.GetExpirationTime()
	if exp != nil {
		_ = s.tokens.Add(ctx, claims.TokenID, exp.Time)
	}

	// Загружаем актуальные данные пользователя.
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, domain.NewAppError(401, "некорректный идентификатор пользователя", err)
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, domain.NewAppError(401, "пользователь не найден", domain.ErrNotFound)
	}

	if !user.IsActive {
		return nil, domain.NewAppError(401, "аккаунт отключён", domain.ErrAccountDisabled)
	}

	return s.generateTokenPair(user)
}

// ValidateAccessToken проверяет access token и возвращает claims.
// Реализует интерфейс middleware.TokenValidator.
func (s *AuthService) ValidateAccessToken(tokenString string) (*middleware.Claims, error) {
	return s.parseToken(tokenString, "access")
}

// IsBlacklisted проверяет блеклист.
// Реализует интерфейс middleware.TokenValidator.
func (s *AuthService) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	return s.tokens.IsBlacklisted(ctx, tokenID)
}

// ── Пароли ──────────────────────────────────────────────────────────────────

// HashPassword создаёт Argon2id хеш пароля с случайной солью.
// Формат: $argon2id$v=19$m=<memory>,t=<time>,p=<threads>$<salt_b64>$<hash_b64>
func (s *AuthService) HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("генерация соли: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		s.cfg.Argon2Time,
		s.cfg.Argon2Memory,
		s.cfg.Argon2Threads,
		s.cfg.Argon2KeyLen,
	)

	// Кодируем в стандартный формат Argon2.
	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		s.cfg.Argon2Memory,
		s.cfg.Argon2Time,
		s.cfg.Argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encoded, nil
}

// verifyPassword проверяет пароль против Argon2id хеша.
// Constant-time сравнение предотвращает timing-атаки.
func (s *AuthService) verifyPassword(password, encodedHash string) bool {
	params, salt, hash, err := decodeArgon2Hash(encodedHash)
	if err != nil {
		return false
	}

	otherHash := argon2.IDKey([]byte(password), salt, params.time, params.memory, params.threads, params.keyLen)

	// subtle.ConstantTimeCompare — защита от timing-атак.
	return subtle.ConstantTimeCompare(hash, otherHash) == 1
}

// argon2Params — параметры Argon2id из закодированного хеша.
type argon2Params struct {
	memory  uint32
	time    uint32
	threads uint8
	keyLen  uint32
}

// decodeArgon2Hash разбирает строку формата Argon2id PHC.
func decodeArgon2Hash(encoded string) (*argon2Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return nil, nil, nil, fmt.Errorf("неверный формат хеша: %d частей", len(parts))
	}

	if parts[1] != "argon2id" {
		return nil, nil, nil, fmt.Errorf("неверный алгоритм: %s", parts[1])
	}

	var p argon2Params
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.time, &p.threads)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("разбор параметров: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("декодирование соли: %w", err)
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("декодирование хеша: %w", err)
	}

	p.keyLen = uint32(len(hash))
	return &p, salt, hash, nil
}

// ── Внутренние методы ───────────────────────────────────────────────────────

// generateTokenPair создаёт новую пару access + refresh токенов.
func (s *AuthService) generateTokenPair(user *domain.User) (*TokenPair, error) {
	accessID := uuid.New().String()
	refreshID := uuid.New().String()
	now := time.Now().UTC()

	// Access token.
	accessClaims := middleware.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.AccessTokenTTL)),
			ID:        accessID,
		},
		UserID:  user.ID.String(),
		Role:    string(user.Role),
		TokenID: accessID,
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).
		SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("подпись access token: %w", err)
	}

	// Refresh token — содержит только ID пользователя и тип.
	refreshClaims := middleware.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.RefreshTokenTTL)),
			ID:        refreshID,
			Audience:  jwt.ClaimStrings{"refresh"},
		},
		UserID:  user.ID.String(),
		Role:    string(user.Role),
		TokenID: refreshID,
	}

	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).
		SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("подпись refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    now.Add(s.cfg.AccessTokenTTL),
		TokenType:    "Bearer",
	}, nil
}

// parseToken разбирает и валидирует JWT токен.
func (s *AuthService) parseToken(tokenString, expectedType string) (*middleware.Claims, error) {
	claims := &middleware.Claims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(t *jwt.Token) (any, error) {
			// Проверяем алгоритм подписи.
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("неожиданный метод подписи: %v", t.Header["alg"])
			}
			return []byte(s.cfg.JWTSecret), nil
		},
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, domain.ErrTokenInvalid
	}

	// Проверка типа токена (access vs refresh).
	if expectedType == "refresh" {
		aud, _ := claims.GetAudience()
		isRefresh := false
		for _, a := range aud {
			if a == "refresh" {
				isRefresh = true
				break
			}
		}
		if !isRefresh {
			return nil, fmt.Errorf("ожидается refresh token")
		}
	}

	return claims, nil
}

// ResetPassword меняет пароль пользователя по email.
// Вызывается после валидации one-time токена сброса.
func (s *AuthService) ResetPassword(ctx context.Context, email, newPassword string) error {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return domain.ErrNotFoundErr("пользователь")
	}

	hash, err := s.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("AuthService.ResetPassword hash: %w", err)
	}

	if err := s.users.UpdatePassword(ctx, user.ID, hash); err != nil {
		return fmt.Errorf("AuthService.ResetPassword update: %w", err)
	}

	s.logger.Info("пароль сброшен", "email", email)
	return nil
}
