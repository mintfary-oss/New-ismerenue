// Package handler содержит HTTP-обработчики (тонкий слой над service).
// Каждый handler: декодирует запрос → вызывает service → кодирует ответ.
package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/mintfary/aqi-platform/internal/service"
)

// Handlers — корневой контейнер всех HTTP-обработчиков.
// Внедряется в роутер через dependency injection.
type Handlers struct {
	Health      *HealthHandler
	Auth        *AuthHandler
	User        *UserHandler
	Sensor      *SensorHandler
	Measurement *MeasurementHandler
	Forecast    *ForecastHandler
	Token       *TokenHandler
	Ingest      *IngestHandler
	Report      *ReportHandler
	Stats       *StatsHandler
	Feedback    *FeedbackHandler
	Widget      *WidgetHandler
}

// Deps — зависимости для создания handlers.
type Deps struct {
	DB          *pgxpool.Pool
	Redis       *redis.Client
	Logger      *slog.Logger
	AuthSvc     *service.AuthService
	UserSvc     *service.UserService
	SensorSvc   *service.SensorService
	MeasureSvc  *service.MeasurementService
}

// NewHandlers создаёт все handlers с общими зависимостями.
func NewHandlers(deps Deps) *Handlers {
	return &Handlers{
		Health:      &HealthHandler{db: deps.DB, logger: deps.Logger},
		Auth:        NewAuthHandler(deps.AuthSvc, deps.Logger),
		User:        NewUserHandler(deps.UserSvc, deps.Logger),
		Sensor:      NewSensorHandler(deps.SensorSvc, deps.Logger),
		Measurement: NewMeasurementHandler(deps.MeasureSvc, deps.Logger),
		Forecast:    &ForecastHandler{logger: deps.Logger},
		Token:       &TokenHandler{logger: deps.Logger},
		Ingest:      NewIngestHandler(deps.MeasureSvc, deps.Logger),
		Report:      &ReportHandler{logger: deps.Logger},
		Stats:       &StatsHandler{logger: deps.Logger},
		Feedback:    &FeedbackHandler{logger: deps.Logger},
		Widget:      &WidgetHandler{logger: deps.Logger},
	}
}

// ── Вспомогательные функции ответа ──────────────────────────────────────

// writeJSON записывает JSON-ответ с заданным статус-кодом.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Ответ уже начат — можно только залогировать.
		slog.Error("writeJSON encode", "err", err)
	}
}

// writeError записывает стандартный JSON-ответ с ошибкой.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ok записывает ответ 200 с данными.
func ok(w http.ResponseWriter, v any) {
	writeJSON(w, http.StatusOK, v)
}

// created записывает ответ 201 с данными.
func created(w http.ResponseWriter, v any) {
	writeJSON(w, http.StatusCreated, v)
}

// noContent записывает ответ 204.
func noContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
