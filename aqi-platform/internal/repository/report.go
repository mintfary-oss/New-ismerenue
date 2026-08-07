// Package repository — доступ к таблице reports.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/google/uuid"
)

// Report — метаданные сгенерированного отчёта.
type Report struct {
	ID          uuid.UUID        `json:"id"`
	UserID      *uuid.UUID       `json:"user_id"`
	Name        string           `json:"name"`
	ReportType  string           `json:"report_type"`
	Params      json.RawMessage  `json:"params"`
	Status      string           `json:"status"`
	RowCount    *int             `json:"row_count"`
	ErrorMsg    *string          `json:"error_msg,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	CompletedAt *time.Time       `json:"completed_at"`
}

// ReportRepo реализует доступ к таблице reports.
type ReportRepo struct {
	db *pgxpool.Pool
}

// NewReportRepo создаёт репозиторий отчётов.
func NewReportRepo(db *pgxpool.Pool) *ReportRepo {
	return &ReportRepo{db: db}
}

// Create создаёт запись отчёта со статусом pending.
func (r *ReportRepo) Create(
	ctx context.Context,
	userID *uuid.UUID,
	name, reportType string,
	params json.RawMessage,
) (*Report, error) {
	const q = `
		INSERT INTO reports (user_id, name, report_type, params)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, name, report_type, params, status,
		          row_count, error_msg, created_at, completed_at`

	rep := &Report{}
	err := r.db.QueryRow(ctx, q, userID, name, reportType, params).Scan(
		&rep.ID, &rep.UserID, &rep.Name, &rep.ReportType, &rep.Params,
		&rep.Status, &rep.RowCount, &rep.ErrorMsg, &rep.CreatedAt, &rep.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("ReportRepo.Create: %w", err)
	}
	return rep, nil
}

// SetReady обновляет статус на ready и сохраняет CSV.
func (r *ReportRepo) SetReady(ctx context.Context, id uuid.UUID, fileData string, rowCount int) error {
	const q = `
		UPDATE reports
		SET status = 'ready', file_data = $2, row_count = $3, completed_at = NOW()
		WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id, fileData, rowCount)
	if err != nil {
		return fmt.Errorf("ReportRepo.SetReady: %w", err)
	}
	return nil
}

// SetError обновляет статус на error.
func (r *ReportRepo) SetError(ctx context.Context, id uuid.UUID, errMsg string) error {
	const q = `
		UPDATE reports
		SET status = 'error', error_msg = $2, completed_at = NOW()
		WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id, errMsg)
	if err != nil {
		return fmt.Errorf("ReportRepo.SetError: %w", err)
	}
	return nil
}

// List возвращает отчёты пользователя (без file_data).
func (r *ReportRepo) List(ctx context.Context, userID *uuid.UUID, limit int) ([]Report, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var (
		q    string
		args []any
	)
	if userID == nil {
		// Admin — все отчёты.
		q = `
			SELECT id, user_id, name, report_type, params, status,
			       row_count, error_msg, created_at, completed_at
			FROM reports
			ORDER BY created_at DESC
			LIMIT $1`
		args = []any{limit}
	} else {
		q = `
			SELECT id, user_id, name, report_type, params, status,
			       row_count, error_msg, created_at, completed_at
			FROM reports
			WHERE user_id = $1
			ORDER BY created_at DESC
			LIMIT $2`
		args = []any{userID, limit}
	}

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("ReportRepo.List: %w", err)
	}
	defer rows.Close()

	var result []Report
	for rows.Next() {
		var rep Report
		if err := rows.Scan(
			&rep.ID, &rep.UserID, &rep.Name, &rep.ReportType, &rep.Params,
			&rep.Status, &rep.RowCount, &rep.ErrorMsg, &rep.CreatedAt, &rep.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("ReportRepo.List scan: %w", err)
		}
		result = append(result, rep)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ReportRepo.List rows: %w", err)
	}
	return result, nil
}

// GetFileData возвращает CSV-содержимое отчёта по ID.
func (r *ReportRepo) GetFileData(ctx context.Context, id uuid.UUID) (string, error) {
	var data *string
	err := r.db.QueryRow(ctx, `SELECT file_data FROM reports WHERE id = $1`, id).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrReportNotFound
	}
	if err != nil {
		return "", fmt.Errorf("ReportRepo.GetFileData: %w", err)
	}
	if data == nil {
		return "", ErrReportNotReady
	}
	return *data, nil
}

// Sentinel-ошибки для отчётов.
var (
	ErrReportNotFound = fmt.Errorf("отчёт не найден")
	ErrReportNotReady = fmt.Errorf("отчёт ещё не готов")
)
