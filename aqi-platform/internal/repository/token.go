// Package repository — реализация доступа к api_tokens.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/google/uuid"
	"github.com/mintfary/aqi-platform/internal/domain"
)

// TokenRepo реализует доступ к таблице api_tokens.
type TokenRepo struct {
	db *pgxpool.Pool
}

// NewTokenRepo создаёт репозиторий API-токенов.
func NewTokenRepo(db *pgxpool.Pool) *TokenRepo {
	return &TokenRepo{db: db}
}

// scanToken сканирует строку БД в структуру APIToken (без token_hash).
func scanToken(row pgx.Row) (*domain.APIToken, error) {
	var t domain.APIToken
	err := row.Scan(&t.ID, &t.UserID, &t.Name, &t.LastUsed, &t.ExpiresAt, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan token: %w", err)
	}
	return &t, nil
}

// List возвращает все API-токены пользователя (без hash).
func (r *TokenRepo) List(ctx context.Context, userID uuid.UUID) ([]domain.APIToken, error) {
	const q = `
		SELECT id, user_id, name, last_used, expires_at, created_at
		FROM api_tokens
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("TokenRepo.List: %w", err)
	}
	defer rows.Close()

	var tokens []domain.APIToken
	for rows.Next() {
		var t domain.APIToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.LastUsed, &t.ExpiresAt, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("TokenRepo.List scan: %w", err)
		}
		tokens = append(tokens, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("TokenRepo.List rows: %w", err)
	}
	return tokens, nil
}

// Create создаёт новый API-токен с HMAC-хешем.
func (r *TokenRepo) Create(
	ctx context.Context,
	userID uuid.UUID,
	name, tokenHash string,
	expiresAt *time.Time,
) (*domain.APIToken, error) {
	const q = `
		INSERT INTO api_tokens (user_id, name, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, name, last_used, expires_at, created_at`

	t, err := scanToken(r.db.QueryRow(ctx, q, userID, name, tokenHash, expiresAt))
	if err != nil {
		return nil, fmt.Errorf("TokenRepo.Create: %w", err)
	}
	return t, nil
}

// Delete удаляет токен пользователя (только свой).
func (r *TokenRepo) Delete(ctx context.Context, userID, tokenID uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM api_tokens WHERE id = $1 AND user_id = $2`,
		tokenID, userID,
	)
	if err != nil {
		return fmt.Errorf("TokenRepo.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// GetByHash возвращает токен по HMAC-хешу (для аутентификации внешних систем).
func (r *TokenRepo) GetByHash(ctx context.Context, tokenHash string) (*domain.APIToken, error) {
	const q = `
		SELECT id, user_id, name, last_used, expires_at, created_at
		FROM api_tokens
		WHERE token_hash = $1`

	t, err := scanToken(r.db.QueryRow(ctx, q, tokenHash))
	if err != nil {
		return nil, fmt.Errorf("TokenRepo.GetByHash: %w", err)
	}
	return t, nil
}

// UpdateLastUsed обновляет поле last_used (вызывается асинхронно).
func (r *TokenRepo) UpdateLastUsed(ctx context.Context, id uuid.UUID, t time.Time) error {
	_, err := r.db.Exec(ctx, `UPDATE api_tokens SET last_used = $2 WHERE id = $1`, id, t)
	if err != nil {
		return fmt.Errorf("TokenRepo.UpdateLastUsed: %w", err)
	}
	return nil
}
