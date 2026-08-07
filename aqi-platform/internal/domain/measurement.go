package domain

import (
	"time"

	"github.com/google/uuid"
)

// Measurement — одно измерение с датчика.
// Хранится в TimescaleDB hypertable, партиционированной по time.
type Measurement struct {
	Time       time.Time `db:"time"`
	SensorID   uuid.UUID `db:"sensor_id"`
	NO2        *float64  `db:"no2"`        // мг/м³
	O3         *float64  `db:"o3"`         // мг/м³
	CO         *float64  `db:"co"`         // мг/м³
	H2S        *float64  `db:"h2s"`        // мг/м³
	SO2        *float64  `db:"so2"`        // мг/м³
	PM25       *float64  `db:"pm25"`       // мг/м³
	Temperature *float64 `db:"temperature"` // °C
	Humidity   *float64  `db:"humidity"`   // %
	Pressure   *float64  `db:"pressure"`   // гПа
	WindSpeed  *float64  `db:"wind_speed"` // м/с
	WindDir    *float64  `db:"wind_dir"`   // градусы (0-360)
}

// MeasurementInput — входные данные при загрузке измерения (от датчика или API).
type MeasurementInput struct {
	SensorID    uuid.UUID `json:"sensor_id"   validate:"required"`
	Time        time.Time `json:"time"        validate:"required"`
	NO2         *float64  `json:"no2"         validate:"omitempty,min=0,max=10"`
	O3          *float64  `json:"o3"          validate:"omitempty,min=0,max=5"`
	CO          *float64  `json:"co"          validate:"omitempty,min=0,max=200"`
	H2S         *float64  `json:"h2s"         validate:"omitempty,min=0,max=1"`
	SO2         *float64  `json:"so2"         validate:"omitempty,min=0,max=20"`
	PM25        *float64  `json:"pm25"        validate:"omitempty,min=0,max=10"`
	Temperature *float64  `json:"temperature" validate:"omitempty,min=-60,max=60"`
	Humidity    *float64  `json:"humidity"    validate:"omitempty,min=0,max=100"`
	Pressure    *float64  `json:"pressure"    validate:"omitempty,min=800,max=1100"`
	WindSpeed   *float64  `json:"wind_speed"  validate:"omitempty,min=0,max=100"`
	WindDir     *float64  `json:"wind_dir"    validate:"omitempty,min=0,max=360"`
}

// MeasurementFilter — фильтр для запроса измерений.
type MeasurementFilter struct {
	SensorID *uuid.UUID `json:"sensor_id"`
	From     time.Time  `json:"from"    validate:"required"`
	To       time.Time  `json:"to"      validate:"required"`
	Period   string     `json:"period"  validate:"omitempty,oneof=raw 1h 1d"` // raw|1h|1d
	Limit    int        `json:"limit"   validate:"omitempty,min=1,max=10000"`
}

// AggregatedMeasurement — агрегированное измерение за период.
type AggregatedMeasurement struct {
	Bucket      time.Time  `db:"bucket"`
	SensorID    uuid.UUID  `db:"sensor_id"`
	AvgNO2      *float64   `db:"avg_no2"`
	AvgO3       *float64   `db:"avg_o3"`
	AvgCO       *float64   `db:"avg_co"`
	AvgH2S      *float64   `db:"avg_h2s"`
	AvgSO2      *float64   `db:"avg_so2"`
	AvgPM25     *float64   `db:"avg_pm25"`
	MaxNO2      *float64   `db:"max_no2"`
	MaxPM25     *float64   `db:"max_pm25"`
	DataPoints  int        `db:"data_points"`
}

// LatestMeasurement — последнее значение датчика (для дашборда).
type LatestMeasurement struct {
	Sensor      Sensor
	Measurement Measurement
	AQI         int
	AQICategory AQICategory
}
