package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/mintfary/aqi-platform/internal/domain"
)

// SensorRepository — интерфейс доступа к данным датчиков.
type SensorRepository interface {
	List(ctx context.Context, onlyActive bool) ([]domain.Sensor, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Sensor, error)
	Create(ctx context.Context, in domain.CreateSensorInput) (*domain.Sensor, error)
	Update(ctx context.Context, id uuid.UUID, in domain.UpdateSensorInput) (*domain.Sensor, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// SensorService — сервис управления датчиками.
type SensorService struct {
	repo   SensorRepository
	logger *slog.Logger
}

// NewSensorService создаёт сервис датчиков.
func NewSensorService(repo SensorRepository, logger *slog.Logger) *SensorService {
	return &SensorService{repo: repo, logger: logger}
}

// List возвращает список датчиков.
func (s *SensorService) List(ctx context.Context, onlyActive bool) ([]domain.Sensor, error) {
	sensors, err := s.repo.List(ctx, onlyActive)
	if err != nil {
		return nil, fmt.Errorf("SensorService.List: %w", err)
	}
	return sensors, nil
}

// GetByID возвращает датчик по ID.
func (s *SensorService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Sensor, error) {
	sensor, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("SensorService.GetByID: %w", err)
	}
	return sensor, nil
}

// Create создаёт новый датчик.
func (s *SensorService) Create(ctx context.Context, in domain.CreateSensorInput) (*domain.Sensor, error) {
	if !isValidSensorType(in.Type) {
		return nil, domain.ErrBadRequest("недопустимый тип датчика: "+string(in.Type), nil)
	}
	sensor, err := s.repo.Create(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("SensorService.Create: %w", err)
	}
	s.logger.Info("датчик создан", "id", sensor.ID, "name", sensor.Name, "type", sensor.Type)
	return sensor, nil
}

// Update обновляет данные датчика.
func (s *SensorService) Update(ctx context.Context, id uuid.UUID, in domain.UpdateSensorInput) (*domain.Sensor, error) {
	if in.Type != nil && !isValidSensorType(*in.Type) {
		return nil, domain.ErrBadRequest("недопустимый тип датчика", nil)
	}
	sensor, err := s.repo.Update(ctx, id, in)
	if err != nil {
		return nil, fmt.Errorf("SensorService.Update: %w", err)
	}
	return sensor, nil
}

// Delete удаляет датчик. Запрещено, если есть данные измерений.
func (s *SensorService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("SensorService.Delete: %w", err)
	}
	s.logger.Info("датчик удалён", "id", id)
	return nil
}

// SensorStatus возвращает статус датчика (онлайн/оффлайн).
func (s *SensorService) SensorStatus(ctx context.Context, id uuid.UUID) (*domain.SensorStatusResponse, error) {
	sensor, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("SensorService.SensorStatus: %w", err)
	}
	return &domain.SensorStatusResponse{
		SensorID: sensor.ID,
		Name:     sensor.Name,
		IsActive: sensor.IsActive,
		IsOnline: sensor.IsOnline(),
		LastSeen: sensor.LastSeen,
	}, nil
}

// isValidSensorType проверяет тип датчика.
func isValidSensorType(t domain.SensorType) bool {
	return t == domain.SensorTypeGas || t == domain.SensorTypeDust || t == domain.SensorTypeCombo
}
