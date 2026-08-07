package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/google/uuid"
	"github.com/mintfary/aqi-platform/internal/domain"
)

// MeasurementRepo реализует доступ к TimescaleDB hypertable measurements.
type MeasurementRepo struct {
	db *pgxpool.Pool
}

// NewMeasurementRepo создаёт репозиторий измерений.
func NewMeasurementRepo(db *pgxpool.Pool) *MeasurementRepo {
	return &MeasurementRepo{db: db}
}

// Insert сохраняет одно измерение с датчика.
// Дублирующиеся записи (sensor_id + time) обновляются (upsert).
func (r *MeasurementRepo) Insert(ctx context.Context, in domain.MeasurementInput) error {
	const q = `
		INSERT INTO measurements
			(time, sensor_id, no2, o3, co, h2s, so2, pm25,
			 temperature, humidity, pressure, wind_speed, wind_dir)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (time, sensor_id) DO UPDATE SET
			no2         = EXCLUDED.no2,
			o3          = EXCLUDED.o3,
			co          = EXCLUDED.co,
			h2s         = EXCLUDED.h2s,
			so2         = EXCLUDED.so2,
			pm25        = EXCLUDED.pm25,
			temperature = EXCLUDED.temperature,
			humidity    = EXCLUDED.humidity,
			pressure    = EXCLUDED.pressure,
			wind_speed  = EXCLUDED.wind_speed,
			wind_dir    = EXCLUDED.wind_dir`

	_, err := r.db.Exec(ctx, q,
		in.Time, in.SensorID,
		in.NO2, in.O3, in.CO, in.H2S, in.SO2, in.PM25,
		in.Temperature, in.Humidity, in.Pressure, in.WindSpeed, in.WindDir,
	)
	if err != nil {
		return fmt.Errorf("MeasurementRepo.Insert: %w", err)
	}
	return nil
}

// InsertBatch сохраняет пакет измерений в одной транзакции.
// Используется для bulk-загрузки данных.
func (r *MeasurementRepo) InsertBatch(ctx context.Context, items []domain.MeasurementInput) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("MeasurementRepo.InsertBatch begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	const q = `
		INSERT INTO measurements
			(time, sensor_id, no2, o3, co, h2s, so2, pm25,
			 temperature, humidity, pressure, wind_speed, wind_dir)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (time, sensor_id) DO NOTHING`

	for _, in := range items {
		if _, err := tx.Exec(ctx, q,
			in.Time, in.SensorID,
			in.NO2, in.O3, in.CO, in.H2S, in.SO2, in.PM25,
			in.Temperature, in.Humidity, in.Pressure, in.WindSpeed, in.WindDir,
		); err != nil {
			return fmt.Errorf("MeasurementRepo.InsertBatch exec: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("MeasurementRepo.InsertBatch commit: %w", err)
	}
	return nil
}

// List возвращает сырые измерения по фильтру.
// Для больших диапазонов рекомендуется использовать Aggregate.
func (r *MeasurementRepo) List(ctx context.Context, f domain.MeasurementFilter) ([]domain.Measurement, error) {
	limit := f.Limit
	if limit <= 0 || limit > 10000 {
		limit = 1000
	}

	// Строим WHERE условия динамически.
	conds := []string{"time >= $1", "time <= $2"}
	args := []any{f.From, f.To}
	argN := 3

	if f.SensorID != nil {
		conds = append(conds, fmt.Sprintf("sensor_id = $%d", argN))
		args = append(args, *f.SensorID)
		argN++
	}

	q := fmt.Sprintf(`
		SELECT time, sensor_id, no2, o3, co, h2s, so2, pm25,
		       temperature, humidity, pressure, wind_speed, wind_dir
		FROM measurements
		WHERE %s
		ORDER BY time DESC
		LIMIT $%d`,
		strings.Join(conds, " AND "), argN,
	)
	args = append(args, limit)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("MeasurementRepo.List: %w", err)
	}
	defer rows.Close()

	var result []domain.Measurement
	for rows.Next() {
		var m domain.Measurement
		if err := rows.Scan(
			&m.Time, &m.SensorID,
			&m.NO2, &m.O3, &m.CO, &m.H2S, &m.SO2, &m.PM25,
			&m.Temperature, &m.Humidity, &m.Pressure, &m.WindSpeed, &m.WindDir,
		); err != nil {
			return nil, fmt.Errorf("MeasurementRepo.List scan: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// Aggregate возвращает агрегированные данные через TimescaleDB time_bucket.
// bucket: "1 hour", "1 day", "1 week" и т.д.
func (r *MeasurementRepo) Aggregate(ctx context.Context, f domain.MeasurementFilter, bucket string) ([]domain.AggregatedMeasurement, error) {
	if bucket == "" {
		bucket = "1 hour"
	}

	conds := []string{"time >= $1", "time <= $2"}
	args := []any{f.From, f.To}
	argN := 3

	if f.SensorID != nil {
		conds = append(conds, fmt.Sprintf("sensor_id = $%d", argN))
		args = append(args, *f.SensorID)
		argN++
	}

	// time_bucket — функция TimescaleDB для группировки по временным интервалам.
	q := fmt.Sprintf(`
		SELECT
			time_bucket($%d, time) AS bucket,
			sensor_id,
			AVG(no2)   AS avg_no2,
			AVG(o3)    AS avg_o3,
			AVG(co)    AS avg_co,
			AVG(h2s)   AS avg_h2s,
			AVG(so2)   AS avg_so2,
			AVG(pm25)  AS avg_pm25,
			MAX(no2)   AS max_no2,
			MAX(pm25)  AS max_pm25,
			COUNT(*)   AS data_points
		FROM measurements
		WHERE %s
		GROUP BY bucket, sensor_id
		ORDER BY bucket DESC, sensor_id`,
		argN,
		strings.Join(conds, " AND "),
	)
	args = append(args, bucket)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("MeasurementRepo.Aggregate: %w", err)
	}
	defer rows.Close()

	var result []domain.AggregatedMeasurement
	for rows.Next() {
		var m domain.AggregatedMeasurement
		if err := rows.Scan(
			&m.Bucket, &m.SensorID,
			&m.AvgNO2, &m.AvgO3, &m.AvgCO, &m.AvgH2S, &m.AvgSO2, &m.AvgPM25,
			&m.MaxNO2, &m.MaxPM25, &m.DataPoints,
		); err != nil {
			return nil, fmt.Errorf("MeasurementRepo.Aggregate scan: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// Latest возвращает последнее измерение для каждого активного датчика.
// Используется для дашборда и виджета.
func (r *MeasurementRepo) Latest(ctx context.Context) ([]domain.LatestMeasurement, error) {
	// DISTINCT ON — PostgreSQL-расширение для выборки одной строки на группу.
	// Эффективно с индексом (sensor_id, time DESC).
	const q = `
		SELECT DISTINCT ON (m.sensor_id)
			m.time, m.sensor_id, m.no2, m.o3, m.co, m.h2s, m.so2, m.pm25,
			m.temperature, m.humidity, m.pressure, m.wind_speed, m.wind_dir,
			s.id, s.name, s.address, s.lat, s.lng, s.type, s.is_active, s.last_seen, s.created_at
		FROM measurements m
		JOIN sensors s ON s.id = m.sensor_id
		WHERE s.is_active = true
		  AND m.time >= NOW() - INTERVAL '24 hours'
		ORDER BY m.sensor_id, m.time DESC`

	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("MeasurementRepo.Latest: %w", err)
	}
	defer rows.Close()

	var result []domain.LatestMeasurement
	for rows.Next() {
		var lm domain.LatestMeasurement
		m := &lm.Measurement
		s := &lm.Sensor
		if err := rows.Scan(
			&m.Time, &m.SensorID,
			&m.NO2, &m.O3, &m.CO, &m.H2S, &m.SO2, &m.PM25,
			&m.Temperature, &m.Humidity, &m.Pressure, &m.WindSpeed, &m.WindDir,
			&s.ID, &s.Name, &s.Address, &s.Lat, &s.Lng,
			&s.Type, &s.IsActive, &s.LastSeen, &s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("MeasurementRepo.Latest scan: %w", err)
		}
		// Рассчитываем AQI по актуальным данным.
		lm.AQI = domain.CalcOverallAQI(m)
		lm.AQICategory = domain.AQIToCategory(lm.AQI)
		result = append(result, lm)
	}
	return result, rows.Err()
}

// LatestBySensor возвращает последнее измерение для конкретного датчика.
func (r *MeasurementRepo) LatestBySensor(ctx context.Context, sensorID uuid.UUID) (*domain.Measurement, error) {
	const q = `
		SELECT time, sensor_id, no2, o3, co, h2s, so2, pm25,
		       temperature, humidity, pressure, wind_speed, wind_dir
		FROM measurements
		WHERE sensor_id = $1
		ORDER BY time DESC
		LIMIT 1`

	var m domain.Measurement
	err := r.db.QueryRow(ctx, q, sensorID).Scan(
		&m.Time, &m.SensorID,
		&m.NO2, &m.O3, &m.CO, &m.H2S, &m.SO2, &m.PM25,
		&m.Temperature, &m.Humidity, &m.Pressure, &m.WindSpeed, &m.WindDir,
	)
	if err != nil {
		return nil, fmt.Errorf("MeasurementRepo.LatestBySensor: %w", err)
	}
	return &m, nil
}

// DeleteOlderThan удаляет измерения старше заданного времени (data retention).
// По ТЗ: хранение 60 месяцев (5 лет). Вызывается планировщиком.
func (r *MeasurementRepo) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM measurements WHERE time < $1`, before,
	)
	if err != nil {
		return 0, fmt.Errorf("MeasurementRepo.DeleteOlderThan: %w", err)
	}
	return tag.RowsAffected(), nil
}
