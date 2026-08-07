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

// SensorRepo реализует доступ к таблице sensors.
type SensorRepo struct {
	db *pgxpool.Pool
}

// NewSensorRepo создаёт репозиторий датчиков.
func NewSensorRepo(db *pgxpool.Pool) *SensorRepo {
	return &SensorRepo{db: db}
}

// scanSensor сканирует строку БД в структуру Sensor.
func scanSensor(row pgx.Row) (*domain.Sensor, error) {
	var s domain.Sensor
	err := row.Scan(
		&s.ID, &s.Name, &s.Address, &s.Lat, &s.Lng,
		&s.Type, &s.IsActive, &s.LastSeen, &s.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan sensor: %w", err)
	}
	return &s, nil
}

// List возвращает список датчиков с опциональной фильтрацией по is_active.
func (r *SensorRepo) List(ctx context.Context, onlyActive bool) ([]domain.Sensor, error) {
	q := `
		SELECT id, name, address, lat, lng, type, is_active, last_seen, created_at
		FROM sensors`
	if onlyActive {
		q += ` WHERE is_active = true`
	}
	q += ` ORDER BY name ASC`

	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("SensorRepo.List: %w", err)
	}
	defer rows.Close()

	var sensors []domain.Sensor
	for rows.Next() {
		var s domain.Sensor
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Address, &s.Lat, &s.Lng,
			&s.Type, &s.IsActive, &s.LastSeen, &s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("SensorRepo.List scan: %w", err)
		}
		sensors = append(sensors, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("SensorRepo.List rows: %w", err)
	}
	return sensors, nil
}

// GetByID возвращает датчик по UUID.
func (r *SensorRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Sensor, error) {
	const q = `
		SELECT id, name, address, lat, lng, type, is_active, last_seen, created_at
		FROM sensors
		WHERE id = $1`

	s, err := scanSensor(r.db.QueryRow(ctx, q, id))
	if err != nil {
		return nil, fmt.Errorf("SensorRepo.GetByID: %w", err)
	}
	return s, nil
}

// Create создаёт новый датчик.
func (r *SensorRepo) Create(ctx context.Context, in domain.CreateSensorInput) (*domain.Sensor, error) {
	const q = `
		INSERT INTO sensors (name, address, lat, lng, type)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, address, lat, lng, type, is_active, last_seen, created_at`

	s, err := scanSensor(r.db.QueryRow(ctx, q,
		in.Name, in.Address, in.Lat, in.Lng, in.Type,
	))
	if err != nil {
		return nil, fmt.Errorf("SensorRepo.Create: %w", err)
	}
	return s, nil
}

// Update обновляет изменяемые поля датчика.
func (r *SensorRepo) Update(ctx context.Context, id uuid.UUID, in domain.UpdateSensorInput) (*domain.Sensor, error) {
	const q = `
		UPDATE sensors SET
			name      = COALESCE($2, name),
			address   = COALESCE($3, address),
			lat       = COALESCE($4, lat),
			lng       = COALESCE($5, lng),
			type      = COALESCE($6, type),
			is_active = COALESCE($7, is_active)
		WHERE id = $1
		RETURNING id, name, address, lat, lng, type, is_active, last_seen, created_at`

	s, err := scanSensor(r.db.QueryRow(ctx, q,
		id, in.Name, in.Address, in.Lat, in.Lng, in.Type, in.IsActive,
	))
	if err != nil {
		return nil, fmt.Errorf("SensorRepo.Update: %w", err)
	}
	return s, nil
}

// Delete удаляет датчик (hard delete, используется только администратором).
func (r *SensorRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM sensors WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("SensorRepo.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// UpdateLastSeen обновляет время последнего контакта с датчиком.
// Вызывается при каждом поступлении данных от датчика.
func (r *SensorRepo) UpdateLastSeen(ctx context.Context, id uuid.UUID, t time.Time) error {
	const q = `UPDATE sensors SET last_seen = $2 WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id, t)
	if err != nil {
		return fmt.Errorf("SensorRepo.UpdateLastSeen: %w", err)
	}
	return nil
}
