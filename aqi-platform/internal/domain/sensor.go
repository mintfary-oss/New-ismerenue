package domain

import (
	"time"

	"github.com/google/uuid"
)

// SensorType — тип датчика.
type SensorType string

const (
	SensorTypeGas   SensorType = "gas"   // газоанализатор (NO2, O3, CO, H2S, SO2)
	SensorTypeDust  SensorType = "dust"  // пылемер (PM2.5)
	SensorTypeCombo SensorType = "combo" // комбинированный
)

// Sensor — физический датчик измерения качества воздуха.
type Sensor struct {
	ID        uuid.UUID  `db:"id"`
	Name      string     `db:"name"`
	Address   string     `db:"address"` // физический адрес установки
	Lat       float64    `db:"lat"`
	Lng       float64    `db:"lng"`
	Type      SensorType `db:"type"`
	IsActive  bool       `db:"is_active"`
	LastSeen  *time.Time `db:"last_seen"` // nil = никогда не видели
	CreatedAt time.Time  `db:"created_at"`
}

// IsOnline возвращает true, если датчик передавал данные в течение последних 30 минут.
func (s *Sensor) IsOnline() bool {
	if s.LastSeen == nil {
		return false
	}
	return time.Since(*s.LastSeen) < 30*time.Minute
}

// CreateSensorInput — данные для создания датчика.
type CreateSensorInput struct {
	Name    string     `json:"name"    validate:"required,min=2,max=100"`
	Address string     `json:"address" validate:"required,min=5,max=255"`
	Lat     float64    `json:"lat"     validate:"required,min=-90,max=90"`
	Lng     float64    `json:"lng"     validate:"required,min=-180,max=180"`
	Type    SensorType `json:"type"    validate:"required"`
}

// UpdateSensorInput — данные для обновления датчика.
type UpdateSensorInput struct {
	Name     *string     `json:"name"      validate:"omitempty,min=2,max=100"`
	Address  *string     `json:"address"   validate:"omitempty,min=5,max=255"`
	Lat      *float64    `json:"lat"       validate:"omitempty,min=-90,max=90"`
	Lng      *float64    `json:"lng"       validate:"omitempty,min=-180,max=180"`
	Type     *SensorType `json:"type"      validate:"omitempty"`
	IsActive *bool       `json:"is_active"`
}

// SensorStatusResponse — ответ на запрос статуса датчика.
type SensorStatusResponse struct {
	SensorID uuid.UUID  `json:"sensor_id"`
	Name     string     `json:"name"`
	IsActive bool       `json:"is_active"`
	IsOnline bool       `json:"is_online"`
	LastSeen *time.Time `json:"last_seen"`
}
