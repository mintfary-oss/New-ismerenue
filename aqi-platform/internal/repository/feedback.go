// Package repository — реализация доступа к таблице feedback.
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/google/uuid"
	"github.com/mintfary/aqi-platform/internal/domain"
)

// FeedbackRepo реализует доступ к таблице feedback.
type FeedbackRepo struct {
	db *pgxpool.Pool
}

// NewFeedbackRepo создаёт репозиторий обратной связи.
func NewFeedbackRepo(db *pgxpool.Pool) *FeedbackRepo {
	return &FeedbackRepo{db: db}
}

// Create сохраняет новое обращение.
func (r *FeedbackRepo) Create(ctx context.Context, in domain.CreateFeedbackInput, userID *uuid.UUID) (*domain.Feedback, error) {
	const q = `
		INSERT INTO feedback (user_id, email, subject, message)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, email, subject, message, status, created_at`

	var email *string
	if in.Email != "" {
		email = &in.Email
	}

	var f domain.Feedback
	err := r.db.QueryRow(ctx, q, userID, email, in.Subject, in.Message).Scan(
		&f.ID, &f.UserID, &f.Email, &f.Subject, &f.Message, &f.Status, &f.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("FeedbackRepo.Create: %w", err)
	}
	return &f, nil
}

// List возвращает список обращений. Admin видит все, остальные — только свои.
func (r *FeedbackRepo) List(ctx context.Context, userID *uuid.UUID, limit, offset int) ([]domain.Feedback, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var (
		rows interface{ Close(); Next() bool; Scan(dest ...any) error; Err() error }
		err  error
	)

	if userID == nil {
		// Admin — все обращения.
		const q = `
			SELECT id, user_id, email, subject, message, status, created_at
			FROM feedback
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2`
		rows, err = r.db.Query(ctx, q, limit, offset)
	} else {
		// Пользователь — только свои.
		const q = `
			SELECT id, user_id, email, subject, message, status, created_at
			FROM feedback
			WHERE user_id = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3`
		rows, err = r.db.Query(ctx, q, userID, limit, offset)
	}
	if err != nil {
		return nil, fmt.Errorf("FeedbackRepo.List: %w", err)
	}
	defer rows.Close()

	var items []domain.Feedback
	for rows.Next() {
		var f domain.Feedback
		if err := rows.Scan(
			&f.ID, &f.UserID, &f.Email, &f.Subject, &f.Message, &f.Status, &f.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("FeedbackRepo.List scan: %w", err)
		}
		items = append(items, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("FeedbackRepo.List rows: %w", err)
	}
	return items, nil
}
