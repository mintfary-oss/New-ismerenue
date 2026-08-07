// Package service — сервис управления API-токенами.
package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/mintfary/aqi-platform/internal/domain"
)

// APITokenRepository — интерфейс репозитория api_tokens.
type APITokenRepository interface {
	List(ctx context.Context, userID uuid.UUID) ([]domain.APIToken, error)
	Create(ctx context.Context, userID uuid.UUID, name, tokenHash string, expiresAt *time.Time) (*domain.APIToken, error)
	Delete(ctx context.Context, userID, tokenID uuid.UUID) error
	GetByHash(ctx context.Context, tokenHash string) (*domain.APIToken, error)
	UpdateLastUsed(ctx context.Context, id uuid.UUID, t time.Time) error
}

// APITokenCreateResult — ответ на создание токена.
// Token возвращается только один раз — при создании.
type APITokenCreateResult struct {
	Token    string          `json:"token"` // raw, показывается один раз
	APIToken domain.APIToken `json:"api_token"`
}

// TokenService управляет API-токенами пользователей.
type TokenService struct {
	repo       APITokenRepository
	userRepo   UserRepository
	hmacSecret []byte
	logger     *slog.Logger
}

// NewTokenService создаёт сервис API-токенов.
// hmacSecret — секрет для HMAC-SHA256 (используется jwt_secret из конфига).
func NewTokenService(
	repo APITokenRepository,
	userRepo UserRepository,
	hmacSecret string,
	logger *slog.Logger,
) *TokenService {
	return &TokenService{
		repo:       repo,
		userRepo:   userRepo,
		hmacSecret: []byte(hmacSecret),
		logger:     logger,
	}
}

// List возвращает все API-токены текущего пользователя.
func (s *TokenService) List(ctx context.Context, userID uuid.UUID) ([]domain.APIToken, error) {
	tokens, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("TokenService.List: %w", err)
	}
	return tokens, nil
}

// Create генерирует новый API-токен и сохраняет его HMAC-хеш.
func (s *TokenService) Create(
	ctx context.Context,
	userID uuid.UUID,
	in domain.CreateAPITokenInput,
) (*APITokenCreateResult, error) {
	if in.Name == "" {
		return nil, domain.ErrBadRequest("имя токена обязательно", nil)
	}

	// Генерируем 32 криптостойких случайных байта → hex-строка (64 символа).
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return nil, fmt.Errorf("TokenService.Create rand: %w", err)
	}
	rawToken := hex.EncodeToString(rawBytes)

	// Сохраняем только HMAC-SHA256 хеш — сам токен нигде не хранится.
	tokenHash := s.hashToken(rawToken)

	token, err := s.repo.Create(ctx, userID, in.Name, tokenHash, in.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("TokenService.Create repo: %w", err)
	}

	s.logger.Info("API-токен создан", "user_id", userID, "token_name", in.Name)

	return &APITokenCreateResult{
		Token:    rawToken,
		APIToken: *token,
	}, nil
}

// Delete удаляет API-токен пользователя.
func (s *TokenService) Delete(ctx context.Context, userID, tokenID uuid.UUID) error {
	if err := s.repo.Delete(ctx, userID, tokenID); err != nil {
		return fmt.Errorf("TokenService.Delete: %w", err)
	}
	s.logger.Info("API-токен удалён", "user_id", userID, "token_id", tokenID)
	return nil
}

// ValidateAPIToken проверяет raw-токен и возвращает связанного пользователя.
// Реализует интерфейс middleware.APITokenChecker.
func (s *TokenService) ValidateAPIToken(ctx context.Context, rawToken string) (*domain.User, error) {
	tokenHash := s.hashToken(rawToken)

	apiToken, err := s.repo.GetByHash(ctx, tokenHash)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	// Проверка срока действия.
	if apiToken.ExpiresAt != nil && time.Now().After(*apiToken.ExpiresAt) {
		return nil, domain.ErrTokenExpired
	}

	// Обновляем last_used асинхронно, чтобы не блокировать запрос.
	go func() {
		_ = s.repo.UpdateLastUsed(context.Background(), apiToken.ID, time.Now().UTC())
	}()

	user, err := s.userRepo.GetByID(ctx, apiToken.UserID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	if !user.IsActive {
		return nil, domain.ErrAccountDisabled
	}
	return user, nil
}

// hashToken вычисляет HMAC-SHA256 от raw-токена.
func (s *TokenService) hashToken(raw string) string {
	h := hmac.New(sha256.New, s.hmacSecret)
	h.Write([]byte(raw))
	return hex.EncodeToString(h.Sum(nil))
}
