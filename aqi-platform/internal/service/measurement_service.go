package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/mintfary/aqi-platform/internal/domain"
)

// MeasurementRepository — интерфейс доступа к данным измерений.
type MeasurementRepository interface {
	Insert(ctx context.Context, in domain.MeasurementInput) error
	InsertBatch(ctx context.Context, items []domain.MeasurementInput) error
	List(ctx context.Context, f domain.MeasurementFilter) ([]domain.Measurement, error)
	Aggregate(ctx context.Context, f domain.MeasurementFilter, bucket string) ([]domain.AggregatedMeasurement, error)
	Latest(ctx context.Context) ([]domain.LatestMeasurement, error)
	LatestBySensor(ctx context.Context, sensorID uuid.UUID) (*domain.Measurement, error)
}

// SensorLastSeenUpdater — интерфейс для обновления last_seen датчика.
type SensorLastSeenUpdater interface {
	UpdateLastSeen(ctx context.Context, id uuid.UUID, t time.Time) error
}

// MeasurementService — сервис работы с измерениями.
type MeasurementService struct {
	measurements MeasurementRepository
	sensors      SensorLastSeenUpdater
	logger       *slog.Logger
}

// NewMeasurementService создаёт сервис измерений.
func NewMeasurementService(
	measurements MeasurementRepository,
	sensors SensorLastSeenUpdater,
	logger *slog.Logger,
) *MeasurementService {
	return &MeasurementService{
		measurements: measurements,
		sensors:      sensors,
		logger:       logger,
	}
}

// Ingest принимает одно измерение от датчика и сохраняет его.
// Также обновляет last_seen датчика.
func (s *MeasurementService) Ingest(ctx context.Context, in domain.MeasurementInput) error {
	// Нормализуем время: если в будущем — отклоняем.
	if in.Time.After(time.Now().Add(5 * time.Minute)) {
		return domain.ErrBadRequest("время измерения в будущем", nil)
	}
	// Если время не задано — используем текущее.
	if in.Time.IsZero() {
		in.Time = time.Now().UTC()
	}

	if err := s.measurements.Insert(ctx, in); err != nil {
		return fmt.Errorf("MeasurementService.Ingest: %w", err)
	}

	// Обновляем last_seen асинхронно, чтобы не блокировать ответ датчику.
	go func() {
		if err := s.sensors.UpdateLastSeen(context.Background(), in.SensorID, in.Time); err != nil {
			s.logger.Warn("обновление last_seen датчика", "sensor_id", in.SensorID, "err", err)
		}
	}()

	return nil
}

// IngestBatch принимает пакет измерений (bulk upload).
func (s *MeasurementService) IngestBatch(ctx context.Context, items []domain.MeasurementInput) error {
	if len(items) == 0 {
		return nil
	}
	if len(items) > 1000 {
		return domain.ErrBadRequest("максимальный размер пакета — 1000 записей", nil)
	}

	now := time.Now().UTC()
	// Нормализуем время и находим самое свежее для last_seen по каждому датчику.
	latestBySensor := make(map[uuid.UUID]time.Time)
	for i := range items {
		if items[i].Time.IsZero() {
			items[i].Time = now
		}
		if t, ok := latestBySensor[items[i].SensorID]; !ok || items[i].Time.After(t) {
			latestBySensor[items[i].SensorID] = items[i].Time
		}
	}

	if err := s.measurements.InsertBatch(ctx, items); err != nil {
		return fmt.Errorf("MeasurementService.IngestBatch: %w", err)
	}

	// Обновляем last_seen для каждого датчика.
	go func() {
		for sensorID, t := range latestBySensor {
			if err := s.sensors.UpdateLastSeen(context.Background(), sensorID, t); err != nil {
				s.logger.Warn("обновление last_seen (batch)", "sensor_id", sensorID, "err", err)
			}
		}
	}()

	return nil
}

// List возвращает измерения по фильтру.
// Если period == "1h" или "1d" — используется агрегация.
func (s *MeasurementService) List(ctx context.Context, f domain.MeasurementFilter) (any, error) {
	switch f.Period {
	case "1h":
		return s.measurements.Aggregate(ctx, f, "1 hour")
	case "1d":
		return s.measurements.Aggregate(ctx, f, "1 day")
	default:
		return s.measurements.List(ctx, f)
	}
}

// Latest возвращает последние измерения по всем активным датчикам.
func (s *MeasurementService) Latest(ctx context.Context) ([]domain.LatestMeasurement, error) {
	result, err := s.measurements.Latest(ctx)
	if err != nil {
		return nil, fmt.Errorf("MeasurementService.Latest: %w", err)
	}
	return result, nil
}

// Aggregate возвращает агрегированные данные за период.
func (s *MeasurementService) Aggregate(ctx context.Context, f domain.MeasurementFilter, bucket string) ([]domain.AggregatedMeasurement, error) {
	result, err := s.measurements.Aggregate(ctx, f, bucket)
	if err != nil {
		return nil, fmt.Errorf("MeasurementService.Aggregate: %w", err)
	}
	return result, nil
}
