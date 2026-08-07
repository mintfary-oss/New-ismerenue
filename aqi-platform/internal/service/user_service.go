package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/mintfary/aqi-platform/internal/domain"
)

// FullUserRepository — расширенный интерфейс для UserService (CRUD).
type FullUserRepository interface {
	UserRepository // GetByEmail, GetByID, Create, UpdatePassword
	List(ctx context.Context, limit, offset int) ([]domain.User, error)
	Update(ctx context.Context, id uuid.UUID, in domain.UpdateUserInput) (*domain.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context) (int, error)
}

// UserService — сервис управления пользователями.
type UserService struct {
	repo   FullUserRepository
	auth   *AuthService // для хэширования паролей
	logger *slog.Logger
}

// NewUserService создаёт сервис пользователей.
func NewUserService(repo FullUserRepository, auth *AuthService, logger *slog.Logger) *UserService {
	return &UserService{repo: repo, auth: auth, logger: logger}
}

// List возвращает постраничный список пользователей.
func (s *UserService) List(ctx context.Context, limit, offset int) ([]domain.User, int, error) {
	users, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("UserService.List: %w", err)
	}
	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("UserService.List count: %w", err)
	}
	return users, total, nil
}

// GetByID возвращает пользователя по ID.
func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("UserService.GetByID: %w", err)
	}
	return u, nil
}

// Create создаёт нового пользователя с хэшированием пароля.
func (s *UserService) Create(ctx context.Context, in domain.CreateUserInput) (*domain.User, error) {
	// Валидация роли.
	if !in.Role.IsValid() {
		return nil, domain.ErrBadRequest("недопустимая роль: "+string(in.Role), nil)
	}

	// Проверяем уникальность email.
	existing, err := s.repo.GetByEmail(ctx, in.Email)
	if err == nil && existing != nil {
		return nil, domain.ErrConflict("пользователь с таким email уже существует")
	}

	// Хэшируем пароль.
	hash, err := s.auth.HashPassword(in.Password)
	if err != nil {
		return nil, fmt.Errorf("UserService.Create hash: %w", err)
	}
	in.Password = hash

	u, err := s.repo.Create(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("UserService.Create: %w", err)
	}
	s.logger.Info("пользователь создан", "id", u.ID, "email", u.Email, "role", u.Role)
	return u, nil
}

// Update обновляет данные пользователя.
func (s *UserService) Update(ctx context.Context, id uuid.UUID, in domain.UpdateUserInput) (*domain.User, error) {
	if in.Role != nil && !in.Role.IsValid() {
		return nil, domain.ErrBadRequest("недопустимая роль", nil)
	}
	u, err := s.repo.Update(ctx, id, in)
	if err != nil {
		return nil, fmt.Errorf("UserService.Update: %w", err)
	}
	return u, nil
}

// Delete деактивирует пользователя (soft delete).
func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("UserService.Delete: %w", err)
	}
	s.logger.Info("пользователь деактивирован", "id", id)
	return nil
}

// ChangePassword меняет пароль после проверки старого.
func (s *UserService) ChangePassword(ctx context.Context, id uuid.UUID, in domain.ChangePasswordInput) error {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("UserService.ChangePassword get: %w", err)
	}

	if !s.auth.verifyPassword(in.OldPassword, u.Password) {
		return domain.NewAppError(400, "неверный текущий пароль", domain.ErrInvalidCredentials)
	}

	hash, err := s.auth.HashPassword(in.NewPassword)
	if err != nil {
		return fmt.Errorf("UserService.ChangePassword hash: %w", err)
	}

	return s.repo.UpdatePassword(ctx, id, hash)
}
