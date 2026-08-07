// Package repository — SQL-запросы для статистики платформы (Admin-only).
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StatsRepo реализует аналитические запросы к измерениям и датчикам.
type StatsRepo struct {
	db *pgxpool.Pool
}

// NewStatsRepo создаёт репозиторий статистики.
func NewStatsRepo(db *pgxpool.Pool) *StatsRepo {
	return &StatsRepo{db: db}
}

// SensorAvailability — доступность одного датчика за период.
type SensorAvailability struct {
	SensorID           string  `json:"sensor_id"`
	SensorName         string  `json:"sensor_name"`
	ExpectedHours      float64 `json:"expected_hours"`        // часов в периоде
	ActualMeasurements int64   `json:"actual_measurements"`   // реальных записей
	AvailabilityPct    float64 `json:"availability_pct"`      // 0–100 %
}

// Availability возвращает процент доступности каждого активного датчика
// за период [from, to). Предполагается 1 измерение в час на датчик.
func (r *StatsRepo) Availability(ctx context.Context, from, to time.Time) ([]SensorAvailability, error) {
	const q = `
		SELECT
			s.id::text                                                       AS sensor_id,
			s.name                                                           AS sensor_name,
			EXTRACT(EPOCH FROM ($2 - $1))::float8 / 3600.0                  AS expected_hours,
			COUNT(m.sensor_id)                                               AS actual_measurements,
			CASE
				WHEN EXTRACT(EPOCH FROM ($2 - $1)) > 0
				THEN ROUND(
					COUNT(m.sensor_id)::numeric
					/ (EXTRACT(EPOCH FROM ($2 - $1))::numeric / 3600)
					* 100,
					2
				)::float8
				ELSE 0.0
			END                                                              AS availability_pct
		FROM sensors s
		LEFT JOIN measurements m
			ON m.sensor_id = s.id
			AND m.time >= $1
			AND m.time <  $2
		WHERE s.is_active = true
		GROUP BY s.id, s.name
		ORDER BY s.name ASC`

	rows, err := r.db.Query(ctx, q, from, to)
	if err != nil {
		return nil, fmt.Errorf("StatsRepo.Availability: %w", err)
	}
	defer rows.Close()

	var result []SensorAvailability
	for rows.Next() {
		var a SensorAvailability
		if err := rows.Scan(
			&a.SensorID, &a.SensorName,
			&a.ExpectedHours, &a.ActualMeasurements, &a.AvailabilityPct,
		); err != nil {
			return nil, fmt.Errorf("StatsRepo.Availability scan: %w", err)
		}
		result = append(result, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("StatsRepo.Availability rows: %w", err)
	}
	return result, nil
}

// ParameterCoverage — покрытие данными одного параметра.
type ParameterCoverage struct {
	Parameter   string  `json:"parameter"`
	Total       int64   `json:"total"`
	NonNull     int64   `json:"non_null"`
	CoveragePct float64 `json:"coverage_pct"` // 0–100 %
}

// DataCoverageRow — внутренняя строка агрегации по всем параметрам.
type DataCoverageRow struct {
	Total       int64
	NO2Count    int64
	O3Count     int64
	COCount     int64
	H2SCount    int64
	SO2Count    int64
	PM25Count   int64
	TempCount   int64
	HumidCount  int64
	PressCount  int64
	WindSCount  int64
	WindDCount  int64
}

// DataCoverage возвращает покрытие по каждому параметру за период [from, to).
func (r *StatsRepo) DataCoverage(ctx context.Context, from, to time.Time) ([]ParameterCoverage, error) {
	const q = `
		SELECT
			COUNT(*)               AS total,
			COUNT(no2)             AS no2_count,
			COUNT(o3)              AS o3_count,
			COUNT(co)              AS co_count,
			COUNT(h2s)             AS h2s_count,
			COUNT(so2)             AS so2_count,
			COUNT(pm25)            AS pm25_count,
			COUNT(temperature)     AS temp_count,
			COUNT(humidity)        AS hum_count,
			COUNT(pressure)        AS press_count,
			COUNT(wind_speed)      AS winds_count,
			COUNT(wind_dir)        AS windd_count
		FROM measurements
		WHERE time >= $1 AND time < $2`

	var row DataCoverageRow
	err := r.db.QueryRow(ctx, q, from, to).Scan(
		&row.Total,
		&row.NO2Count, &row.O3Count, &row.COCount, &row.H2SCount,
		&row.SO2Count, &row.PM25Count,
		&row.TempCount, &row.HumidCount, &row.PressCount,
		&row.WindSCount, &row.WindDCount,
	)
	if err != nil {
		return nil, fmt.Errorf("StatsRepo.DataCoverage: %w", err)
	}

	pct := func(n int64) float64 {
		if row.Total == 0 {
			return 0
		}
		return float64(n) / float64(row.Total) * 100
	}

	params := []struct {
		name  string
		count int64
	}{
		{"no2", row.NO2Count},
		{"o3", row.O3Count},
		{"co", row.COCount},
		{"h2s", row.H2SCount},
		{"so2", row.SO2Count},
		{"pm25", row.PM25Count},
		{"temperature", row.TempCount},
		{"humidity", row.HumidCount},
		{"pressure", row.PressCount},
		{"wind_speed", row.WindSCount},
		{"wind_dir", row.WindDCount},
	}

	result := make([]ParameterCoverage, 0, len(params))
	for _, p := range params {
		result = append(result, ParameterCoverage{
			Parameter:   p.name,
			Total:       row.Total,
			NonNull:     p.count,
			CoveragePct: pct(p.count),
		})
	}
	return result, nil
}
