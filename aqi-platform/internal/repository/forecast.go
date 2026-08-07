package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mintfary/aqi-platform/internal/domain"
)

// ForecastRepo реализует доступ к TimescaleDB hypertable forecasts.
type ForecastRepo struct {
	db *pgxpool.Pool
}

// NewForecastRepo создаёт репозиторий прогнозов.
func NewForecastRepo(db *pgxpool.Pool) *ForecastRepo {
	return &ForecastRepo{db: db}
}

// InsertBatch сохраняет пакет прогнозов в одной транзакции.
// Дублирующиеся записи (time + point_id + horizon_hours) обновляются (upsert).
func (r *ForecastRepo) InsertBatch(ctx context.Context, forecasts []domain.Forecast) error {
	if len(forecasts) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ForecastRepo.InsertBatch begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	const q = `
		INSERT INTO forecasts
			(time, point_id, lat, lng, horizon_hours, aqi, aqi_category,
			 no2_forecast, pm25_forecast, so2_forecast, model_version)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (time, point_id, horizon_hours) DO UPDATE SET
			aqi           = EXCLUDED.aqi,
			aqi_category  = EXCLUDED.aqi_category,
			no2_forecast  = EXCLUDED.no2_forecast,
			pm25_forecast = EXCLUDED.pm25_forecast,
			so2_forecast  = EXCLUDED.so2_forecast,
			model_version = EXCLUDED.model_version`

	for _, f := range forecasts {
		if _, err := tx.Exec(ctx, q,
			f.Time, f.PointID, f.Lat, f.Lng, f.HorizonHours,
			f.AQI, string(f.AQICategory),
			f.NO2Forecast, f.PM25Forecast, f.SO2Forecast,
			f.ModelVersion,
		); err != nil {
			return fmt.Errorf("ForecastRepo.InsertBatch exec: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("ForecastRepo.InsertBatch commit: %w", err)
	}
	return nil
}

// LatestByPoint возвращает последний прогноз для конкретной точки (все горизонты).
func (r *ForecastRepo) LatestByPoint(ctx context.Context, pointID string) ([]domain.Forecast, error) {
	// Выбираем все горизонты для самого свежего времени создания прогноза.
	const q = `
		WITH latest AS (
			SELECT MAX(created_at) AS ts
			FROM forecasts
			WHERE point_id = $1
		)
		SELECT f.time, f.point_id, f.lat, f.lng, f.horizon_hours,
		       f.aqi, f.aqi_category, f.no2_forecast, f.pm25_forecast,
		       f.so2_forecast, f.model_version, f.created_at
		FROM forecasts f, latest
		WHERE f.point_id = $1
		  AND f.created_at = latest.ts
		ORDER BY f.horizon_hours ASC`

	rows, err := r.db.Query(ctx, q, pointID)
	if err != nil {
		return nil, fmt.Errorf("ForecastRepo.LatestByPoint: %w", err)
	}
	defer rows.Close()

	return scanForecasts(rows)
}

// Latest возвращает текущий прогноз для всех точек (горизонт = 0 = сейчас).
func (r *ForecastRepo) Latest(ctx context.Context) ([]domain.Forecast, error) {
	const q = `
		SELECT DISTINCT ON (point_id)
			time, point_id, lat, lng, horizon_hours,
			aqi, aqi_category, no2_forecast, pm25_forecast,
			so2_forecast, model_version, created_at
		FROM forecasts
		WHERE horizon_hours = 0
		ORDER BY point_id, created_at DESC`

	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("ForecastRepo.Latest: %w", err)
	}
	defer rows.Close()

	return scanForecasts(rows)
}

// ByDistrict возвращает последний прогноз для всех точек в заданном районе.
func (r *ForecastRepo) ByDistrict(ctx context.Context, district string) ([]domain.Forecast, error) {
	// Ищем точки района из статических данных.
	points := domain.PointsByDistrict(district)
	if len(points) == 0 {
		return nil, domain.ErrNotFound
	}

	pointIDs := make([]string, len(points))
	for i, p := range points {
		pointIDs[i] = p.ID
	}

	// Для каждой точки берём последний прогноз (horizon_hours=0).
	const q = `
		SELECT DISTINCT ON (point_id)
			time, point_id, lat, lng, horizon_hours,
			aqi, aqi_category, no2_forecast, pm25_forecast,
			so2_forecast, model_version, created_at
		FROM forecasts
		WHERE point_id = ANY($1) AND horizon_hours = 0
		ORDER BY point_id, created_at DESC`

	rows, err := r.db.Query(ctx, q, pointIDs)
	if err != nil {
		return nil, fmt.Errorf("ForecastRepo.ByDistrict: %w", err)
	}
	defer rows.Close()

	return scanForecasts(rows)
}

// CityAverage вычисляет средний AQI по всем точкам на основе последних прогнозов.
func (r *ForecastRepo) CityAverage(ctx context.Context) (*domain.CityForecast, error) {
	forecasts, err := r.Latest(ctx)
	if err != nil {
		return nil, err
	}
	if len(forecasts) == 0 {
		return nil, domain.ErrNotFound
	}

	// Вычисляем средний AQI по городу.
	totalAQI := 0
	for _, f := range forecasts {
		totalAQI += f.AQI
	}
	avgAQI := totalAQI / len(forecasts)
	cityCategory := domain.AQIToCategory(avgAQI)

	// Группируем по районам.
	districtMap := make(map[string][]int) // district → []aqi
	for _, f := range forecasts {
		pt := domain.PointByID(f.PointID)
		if pt != nil {
			districtMap[pt.District] = append(districtMap[pt.District], f.AQI)
		}
	}

	districts := make([]domain.DistrictForecast, 0, len(districtMap))
	for name, aqis := range districtMap {
		sum := 0
		for _, v := range aqis {
			sum += v
		}
		avg := sum / len(aqis)
		districts = append(districts, domain.DistrictForecast{
			DistrictName: name,
			AQI:          avg,
			AQICategory:  domain.AQIToCategory(avg),
		})
	}

	return &domain.CityForecast{
		Time:         forecasts[0].Time,
		CityAQI:      avgAQI,
		CityCategory: cityCategory,
		Districts:    districts,
		Points:       forecasts,
	}, nil
}

// DeleteOlderThan удаляет прогнозы старше заданного времени (retention).
func (r *ForecastRepo) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM forecasts WHERE created_at < $1`, before,
	)
	if err != nil {
		return 0, fmt.Errorf("ForecastRepo.DeleteOlderThan: %w", err)
	}
	return tag.RowsAffected(), nil
}

// ── helpers ────────────────────────────────────────────────────────────────

type forecastRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}

func scanForecasts(rows forecastRows) ([]domain.Forecast, error) {
	defer rows.Close()
	var result []domain.Forecast
	for rows.Next() {
		var f domain.Forecast
		var cat string
		if err := rows.Scan(
			&f.Time, &f.PointID, &f.Lat, &f.Lng, &f.HorizonHours,
			&f.AQI, &cat,
			&f.NO2Forecast, &f.PM25Forecast, &f.SO2Forecast,
			&f.ModelVersion, &f.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanForecasts: %w", err)
		}
		f.AQICategory = domain.AQICategory(cat)
		result = append(result, f)
	}
	return result, rows.Err()
}
