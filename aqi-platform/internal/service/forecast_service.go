// Package service — прогнозный движок AQI Platform.
//
// Алгоритм (двухэтапный):
//  1. EWMA (Exponentially Weighted Moving Average) — сглаживает шумы датчика
//     и экстраполирует значение вперёд на horizon_hours.
//  2. IDW (Inverse Distance Weighting) — интерполирует значения с датчиков
//     на 4 фиксированные точки мониторинга Кемерово.
//
// Формулы:
//
//	EWMA:   S_t = α × X_t + (1−α) × S_{t−1}
//	Тренд:  ΔS = (S_last − S_prev) / period
//	IDW:    V_p = Σ(w_i × V_i) / Σ(w_i), где w_i = 1/d_i^p
package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/mintfary/aqi-platform/internal/config"
	"github.com/mintfary/aqi-platform/internal/domain"
)

// ForecastMeasurementReader читает последние измерения для прогноза.
type ForecastMeasurementReader interface {
	Latest(ctx context.Context) ([]domain.LatestMeasurement, error)
	List(ctx context.Context, f domain.MeasurementFilter) ([]domain.Measurement, error)
}

// ForecastWriter сохраняет рассчитанные прогнозы.
type ForecastWriter interface {
	InsertBatch(ctx context.Context, forecasts []domain.Forecast) error
	Latest(ctx context.Context) ([]domain.Forecast, error)
	LatestByPoint(ctx context.Context, pointID string) ([]domain.Forecast, error)
	ByDistrict(ctx context.Context, district string) ([]domain.Forecast, error)
	CityAverage(ctx context.Context) (*domain.CityForecast, error)
}

// ForecastService управляет расчётом и хранением прогнозов.
type ForecastService struct {
	measurements ForecastMeasurementReader
	forecasts    ForecastWriter
	cfg          config.ForecastConfig
	logger       *slog.Logger
}

// NewForecastService создаёт сервис прогнозирования.
func NewForecastService(
	measurements ForecastMeasurementReader,
	forecasts ForecastWriter,
	cfg config.ForecastConfig,
	logger *slog.Logger,
) *ForecastService {
	return &ForecastService{
		measurements: measurements,
		forecasts:    forecasts,
		cfg:          cfg,
		logger:       logger,
	}
}

// ── Публичные методы чтения ────────────────────────────────────────────────

// Current возвращает текущий прогноз для всех точек (horizon=0).
func (s *ForecastService) Current(ctx context.Context) ([]domain.Forecast, error) {
	return s.forecasts.Latest(ctx)
}

// ByPoint возвращает все горизонты прогноза для конкретной точки.
func (s *ForecastService) ByPoint(ctx context.Context, pointID string) ([]domain.Forecast, error) {
	if domain.PointByID(pointID) == nil {
		return nil, domain.ErrNotFound
	}
	return s.forecasts.LatestByPoint(ctx, pointID)
}

// ByDistrict возвращает прогноз по всем точкам района.
func (s *ForecastService) ByDistrict(ctx context.Context, district string) ([]domain.Forecast, error) {
	return s.forecasts.ByDistrict(ctx, district)
}

// CityAverage возвращает агрегированный прогноз по городу и районам.
func (s *ForecastService) CityAverage(ctx context.Context) (*domain.CityForecast, error) {
	return s.forecasts.CityAverage(ctx)
}

// Points возвращает список всех точек мониторинга (статические данные).
func (s *ForecastService) Points() []domain.ForecastPoint {
	return domain.KemerovoPoints
}

// ── Расчёт прогноза ────────────────────────────────────────────────────────

// Run выполняет один цикл расчёта прогноза.
// Вызывается планировщиком каждые cfg.UpdateInterval.
func (s *ForecastService) Run(ctx context.Context) error {
	s.logger.Info("расчёт прогноза: старт")

	// 1. Читаем последние измерения со всех датчиков.
	latest, err := s.measurements.Latest(ctx)
	if err != nil {
		return fmt.Errorf("ForecastService.Run: чтение измерений: %w", err)
	}

	if len(latest) < s.cfg.MinSensorsForForecast {
		s.logger.Warn("недостаточно датчиков для прогноза",
			"available", len(latest),
			"required", s.cfg.MinSensorsForForecast,
		)
		return nil
	}

	// 2. Для каждого датчика считаем тренд (EWMA по последним 6 часам).
	sensorValues, err := s.computeSensorTrends(ctx, latest)
	if err != nil {
		s.logger.Warn("расчёт трендов", "err", err)
		// Используем текущие значения без тренда.
		for i := range latest {
			if _, ok := sensorValues[latest[i].Sensor.ID.String()]; !ok {
				sensorValues[latest[i].Sensor.ID.String()] = sensorValue{
					pm25: latest[i].Measurement.PM25,
					no2:  latest[i].Measurement.NO2,
					so2:  latest[i].Measurement.SO2,
					lat:  latest[i].Sensor.Lat,
					lng:  latest[i].Sensor.Lng,
				}
			}
		}
	}

	// 3. Рассчитываем прогнозы для каждой точки на каждый горизонт.
	now := time.Now().UTC().Truncate(time.Hour)
	horizons := buildHorizons(s.cfg.HorizonHours)

	var allForecasts []domain.Forecast

	for _, point := range domain.KemerovoPoints {
		for _, h := range horizons {
			fc, err := s.computePointForecast(sensorValues, point, h, now)
			if err != nil {
				s.logger.Warn("расчёт для точки", "point", point.ID, "horizon", h, "err", err)
				continue
			}
			allForecasts = append(allForecasts, *fc)
		}
	}

	if len(allForecasts) == 0 {
		return fmt.Errorf("ForecastService.Run: нет данных для сохранения")
	}

	// 4. Сохраняем в БД.
	if err := s.forecasts.InsertBatch(ctx, allForecasts); err != nil {
		return fmt.Errorf("ForecastService.Run: сохранение: %w", err)
	}

	s.logger.Info("расчёт прогноза: завершён",
		"points", len(domain.KemerovoPoints),
		"horizons", len(horizons),
		"records", len(allForecasts),
	)
	return nil
}

// ── EWMA: экстраполяция тренда датчика ────────────────────────────────────

// sensorValue — сглаженные значения одного датчика на moment T.
type sensorValue struct {
	pm25     *float64
	no2      *float64
	so2      *float64
	lat, lng float64
	trendPM  float64 // Δpm25/час
	trendNO2 float64 // Δno2/час
}

// computeSensorTrends вычисляет EWMA-тренд для каждого датчика.
// Читает последние 6 часов измерений и применяет EWMA.
func (s *ForecastService) computeSensorTrends(
	ctx context.Context,
	latest []domain.LatestMeasurement,
) (map[string]sensorValue, error) {
	result := make(map[string]sensorValue, len(latest))

	now := time.Now().UTC()
	filter := domain.MeasurementFilter{
		From:   now.Add(-6 * time.Hour),
		To:     now,
		Period: "raw",
		Limit:  360, // 6ч × 60 мин = 360 записей макс
	}

	allMeasurements, err := s.measurements.List(ctx, filter)
	if err != nil {
		return result, fmt.Errorf("computeSensorTrends: %w", err)
	}

	// Группируем по датчику.
	byID := make(map[string][]domain.Measurement)
	for _, m := range allMeasurements {
		id := m.SensorID.String()
		byID[id] = append(byID[id], m)
	}

	α := s.cfg.EWMAAlpha
	if α <= 0 || α >= 1 {
		α = 0.3 // безопасное значение по умолчанию
	}

	for _, lm := range latest {
		id := lm.Sensor.ID.String()
		measurements := byID[id]

		sv := sensorValue{
			lat: lm.Sensor.Lat,
			lng: lm.Sensor.Lng,
			pm25: lm.Measurement.PM25,
			no2:  lm.Measurement.NO2,
			so2:  lm.Measurement.SO2,
		}

		if len(measurements) >= 2 {
			// Применяем EWMA к временному ряду.
			ewmaPM, ewmaNO2 := computeEWMA(measurements, α)
			sv.pm25 = ewmaPM.last
			sv.no2 = ewmaNO2.last
			sv.trendPM = ewmaPM.trendPerHour
			sv.trendNO2 = ewmaNO2.trendPerHour
		}

		result[id] = sv
	}

	return result, nil
}

// ewmaResult — результат EWMA для одного параметра.
type ewmaResult struct {
	last         *float64
	trendPerHour float64
}

// computeEWMA вычисляет сглаженное значение и почасовой тренд через EWMA.
func computeEWMA(measurements []domain.Measurement, α float64) (pm, no2 ewmaResult) {
	// Сортировка по времени (ascending) не гарантирована из БД — используем поступающий порядок.
	// measurements предполагаются отсортированными по time DESC из репозитория.
	// Разворачиваем для forward EWMA.
	n := len(measurements)

	var (
		smPM  float64
		smNO2 float64
		hasPM bool
		hasNO2 bool
	)

	// Проходим от старых к новым (reverse, т.к. из БД DESC).
	for i := n - 1; i >= 0; i-- {
		m := measurements[i]
		if m.PM25 != nil {
			if !hasPM {
				smPM = *m.PM25
				hasPM = true
			} else {
				smPM = α**m.PM25 + (1-α)*smPM
			}
		}
		if m.NO2 != nil {
			if !hasNO2 {
				smNO2 = *m.NO2
				hasNO2 = true
			} else {
				smNO2 = α**m.NO2 + (1-α)*smNO2
			}
		}
	}

	// Тренд = (последнее EWMA − первое EWMA) / duration_hours.
	var trendPM, trendNO2 float64
	if n >= 2 && !measurements[0].Time.IsZero() && !measurements[n-1].Time.IsZero() {
		durationH := measurements[n-1].Time.Sub(measurements[0].Time).Hours()
		if durationH > 0 {
			// Приблизительный тренд через начальное/конечное EWMA.
			if hasPM {
				var firstPM float64
				if measurements[n-1].PM25 != nil {
					firstPM = *measurements[n-1].PM25
				}
				trendPM = (smPM - firstPM) / durationH
			}
			if hasNO2 {
				var firstNO2 float64
				if measurements[n-1].NO2 != nil {
					firstNO2 = *measurements[n-1].NO2
				}
				trendNO2 = (smNO2 - firstNO2) / durationH
			}
		}
	}

	if hasPM {
		v := smPM
		pm = ewmaResult{last: &v, trendPerHour: trendPM}
	}
	if hasNO2 {
		v := smNO2
		no2 = ewmaResult{last: &v, trendPerHour: trendNO2}
	}
	return
}

// ── IDW: интерполяция на контрольные точки ─────────────────────────────────

// computePointForecast вычисляет прогноз для одной точки на один горизонт.
// Использует IDW (обратно-взвешенную интерполяцию расстояний).
func (s *ForecastService) computePointForecast(
	sensors map[string]sensorValue,
	point domain.ForecastPoint,
	horizonHours int,
	baseTime time.Time,
) (*domain.Forecast, error) {
	if len(sensors) == 0 {
		return nil, fmt.Errorf("нет данных датчиков")
	}

	p := s.cfg.IDWPower
	if p <= 0 {
		p = 2.0
	}

	// Вычисляем взвешенные значения через IDW.
	var (
		sumW   float64
		sumPM  float64
		sumNO2 float64
		sumSO2 float64
		hasPM  bool
		hasNO2 bool
		hasSO2 bool
	)

	for _, sv := range sensors {
		dist := haversine(point.Lat, point.Lng, sv.lat, sv.lng)
		if dist < 0.001 {
			dist = 0.001 // защита от деления на ноль (датчик прямо в точке)
		}
		w := 1.0 / math.Pow(dist, p)
		sumW += w

		// Экстраполяция тренда на горизонт.
		hf := float64(horizonHours)

		if sv.pm25 != nil {
			val := *sv.pm25 + sv.trendPM*hf
			if val < 0 {
				val = 0
			}
			sumPM += w * val
			hasPM = true
		}
		if sv.no2 != nil {
			val := *sv.no2 + sv.trendNO2*hf
			if val < 0 {
				val = 0
			}
			sumNO2 += w * val
			hasNO2 = true
		}
		if sv.so2 != nil {
			sumSO2 += w * *sv.so2
			hasSO2 = true
		}
	}

	if sumW == 0 {
		return nil, fmt.Errorf("нулевой суммарный вес IDW")
	}

	fc := &domain.Forecast{
		Time:         baseTime.Add(time.Duration(horizonHours) * time.Hour),
		PointID:      point.ID,
		Lat:          point.Lat,
		Lng:          point.Lng,
		HorizonHours: horizonHours,
		ModelVersion: "v2-ewma-idw",
		CreatedAt:    time.Now().UTC(),
	}

	// Собираем измерение для расчёта AQI.
	var m domain.Measurement
	if hasPM {
		v := sumPM / sumW
		m.PM25 = &v
		fc.PM25Forecast = &v
	}
	if hasNO2 {
		v := sumNO2 / sumW
		m.NO2 = &v
		fc.NO2Forecast = &v
	}
	if hasSO2 {
		v := sumSO2 / sumW
		fc.SO2Forecast = &v
	}

	fc.AQI = domain.CalcOverallAQI(&m)
	fc.AQICategory = domain.AQIToCategory(fc.AQI)

	return fc, nil
}

// ── Вспомогательные функции ────────────────────────────────────────────────

// haversine вычисляет расстояние в км между двумя точками на сфере.
func haversine(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371.0 // радиус Земли в км
	φ1 := lat1 * math.Pi / 180
	φ2 := lat2 * math.Pi / 180
	Δφ := (lat2 - lat1) * math.Pi / 180
	Δλ := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(Δφ/2)*math.Sin(Δφ/2) +
		math.Cos(φ1)*math.Cos(φ2)*
			math.Sin(Δλ/2)*math.Sin(Δλ/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// buildHorizons строит список горизонтов прогноза.
// Для горизонта ≤ 6ч: [0, 1, 2, 3, 6]; для больших — добавляем промежуточные.
func buildHorizons(maxH int) []int {
	if maxH <= 0 {
		maxH = 6
	}
	base := []int{0, 1, 2, 3}
	for _, h := range []int{6, 12, 24} {
		if h <= maxH {
			base = append(base, h)
		}
	}
	// Если maxH не покрыт — добавляем его явно.
	last := base[len(base)-1]
	if last < maxH {
		base = append(base, maxH)
	}
	return base
}
