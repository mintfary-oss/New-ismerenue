// Package repository содержит реализацию доступа к данным через pgx/v5.
// Все методы корректно обрабатывают pgx.ErrNoRows → domain.ErrNotFound.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/google/uuid"
	"github.com/mintfary/aqi-platform/internal/domain"
)

// UserRepo реализует доступ к таблице users.
type UserRepo struct {
	db *pgxpool.Pool
}

// NewUserRepo создаёт репозиторий пользователей.
func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

// GetByEmail возвращает пользователя по e-mail.
// Возвращает domain.ErrNotFound, если пользователь не найден.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	const q = `
		SELECT id, email, username, password, role, is_active, created_at, updated_at
		FROM users
		WHERE email = $1 AND is_active = true
		LIMIT 1`

	var u domain.User
	err := r.db.QueryRow(ctx, q, email).Scan(
		&u.ID, &u.Email, &u.Username, &u.Password,
		&u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("UserRepo.GetByEmail: %w", err)
	}
	return &u, nil
}

// GetByID возвращает пользователя по UUID.
func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const q = `
		SELECT id, email, username, password, role, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
		LIMIT 1`

	var u domain.User
	err := r.db.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.Email, &u.Username, &u.Password,
		&u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("UserRepo.GetByID: %w", err)
	}
	return &u, nil
}

// List возвращает постраничный список пользователей.
func (r *UserRepo) List(ctx context.Context, limit, offset int) ([]domain.User, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	const q = `
		SELECT id, email, username, password, role, is_active, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("UserRepo.List: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(
			&u.ID, &u.Email, &u.Username, &u.Password,
			&u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("UserRepo.List scan: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("UserRepo.List rows: %w", err)
	}
	return users, nil
}

// Create создаёт нового пользователя и возвращает его.
func (r *UserRepo) Create(ctx context.Context, in domain.CreateUserInput) (*domain.User, error) {
	// Хэш пароля должен быть уже установлен в поле Password.
	// Сервисный слой отвечает за хэширование до вызова репозитория.
	const q = `
		INSERT INTO users (email, username, password, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, username, password, role, is_active, created_at, updated_at`

	var u domain.User
	err := r.db.QueryRow(ctx, q,
		in.Email, in.Username, in.Password, in.Role,
	).Scan(
		&u.ID, &u.Email, &u.Username, &u.Password,
		&u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("UserRepo.Create: %w", err)
	}
	return &u, nil
}

// Update обновляет изменяемые поля пользователя.
// Поля со значением nil не обновляются (COALESCE-паттерн).
func (r *UserRepo) Update(ctx context.Context, id uuid.UUID, in domain.UpdateUserInput) (*domain.User, error) {
	const q = `
		UPDATE users SET
			email     = COALESCE($2, email),
			username  = COALESCE($3, username),
			role      = COALESCE($4, role),
			is_active = COALESCE($5, is_active),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, email, username, password, role, is_active, created_at, updated_at`

	var u domain.User
	err := r.db.QueryRow(ctx, q,
		id, in.Email, in.Username, in.Role, in.IsActive,
	).Scan(
		&u.ID, &u.Email, &u.Username, &u.Password,
		&u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("UserRepo.Update: %w", err)
	}
	return &u, nil
}

// Delete деактивирует пользователя (soft delete через is_active = false).
func (r *UserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE users SET is_active = false, updated_at = NOW() WHERE id = $1`
	tag, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("UserRepo.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// UpdatePassword обновляет хэш пароля пользователя.
func (r *UserRepo) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	const q = `UPDATE users SET password = $2, updated_at = NOW() WHERE id = $1`
	tag, err := r.db.Exec(ctx, q, id, hash)
	if err != nil {
		return fmt.Errorf("UserRepo.UpdatePassword: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Count возвращает общее число пользователей.
func (r *UserRepo) Count(ctx context.Context) (int, error) {
	var n int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return 0, fmt.Errorf("UserRepo.Count: %w", err)
	}
	return n, nil
}

// API-токены вынесены в repository/token.go (TokenRepo).
