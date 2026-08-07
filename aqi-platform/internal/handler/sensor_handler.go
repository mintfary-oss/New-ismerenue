package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/mintfary/aqi-platform/internal/domain"
	"github.com/mintfary/aqi-platform/internal/service"
)

// SensorHandler реализует HTTP-обработчики управления датчиками.
type SensorHandler struct {
	svc    *service.SensorService
	logger *slog.Logger
}

// NewSensorHandler создаёт обработчик датчиков.
func NewSensorHandler(svc *service.SensorService, logger *slog.Logger) *SensorHandler {
	return &SensorHandler{svc: svc, logger: logger}
}

// List godoc
// @Summary     Список датчиков
// @Description Возвращает все датчики. Параметр active=true — только активные.
// @Tags        sensors
// @Security    BearerAuth
// @Param       active query bool false "Только активные датчики"
// @Success     200 {object} map[string]any
// @Router      /sensors [get]
func (h *SensorHandler) List(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("active") == "true"

	sensors, err := h.svc.List(r.Context(), onlyActive)
	if err != nil {
		handleError(w, h.logger, err)
		return
	}

	ok(w, map[string]any{
		"sensors": sensorsToResponse(sensors),
		"count":   len(sensors),
	})
}

// Create godoc
// @Summary     Создать датчик
// @Tags        sensors
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body domain.CreateSensorInput true "Данные датчика"
// @Success     201 {object} map[string]any
// @Router      /sensors [post]
func (h *SensorHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in domain.CreateSensorInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}

	sensor, err := h.svc.Create(r.Context(), in)
	if err != nil {
		handleError(w, h.logger, err)
		return
	}

	created(w, sensorToResponse(sensor))
}

// Get godoc
// @Summary     Получить датчик
// @Tags        sensors
// @Security    BearerAuth
// @Param       id path string true "UUID датчика"
// @Success     200 {object} map[string]any
// @Router      /sensors/{id} [get]
func (h *SensorHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный UUID")
		return
	}

	sensor, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		handleError(w, h.logger, err)
		return
	}

	ok(w, sensorToResponse(sensor))
}

// Update godoc
// @Summary     Обновить датчик
// @Tags        sensors
// @Security    BearerAuth
// @Param       id   path string                   true "UUID датчика"
// @Param       body body domain.UpdateSensorInput true "Поля для обновления"
// @Success     200 {object} map[string]any
// @Router      /sensors/{id} [patch]
func (h *SensorHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный UUID")
		return
	}

	var in domain.UpdateSensorInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}

	sensor, err := h.svc.Update(r.Context(), id, in)
	if err != nil {
		handleError(w, h.logger, err)
		return
	}

	ok(w, sensorToResponse(sensor))
}

// Delete godoc
// @Summary     Удалить датчик
// @Tags        sensors
// @Security    BearerAuth
// @Param       id path string true "UUID датчика"
// @Success     204
// @Router      /sensors/{id} [delete]
func (h *SensorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный UUID")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		handleError(w, h.logger, err)
		return
	}

	noContent(w)
}

// Status godoc
// @Summary     Статус датчика
// @Description Возвращает онлайн/оффлайн статус и время последнего контакта.
// @Tags        sensors
// @Security    BearerAuth
// @Param       id path string true "UUID датчика"
// @Success     200 {object} domain.SensorStatusResponse
// @Router      /sensors/{id}/status [get]
func (h *SensorHandler) Status(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный UUID")
		return
	}

	status, err := h.svc.SensorStatus(r.Context(), id)
	if err != nil {
		handleError(w, h.logger, err)
		return
	}

	ok(w, status)
}

// ── Вспомогательные функции ────────────────────────────────────────────────

func sensorToResponse(s *domain.Sensor) map[string]any {
	resp := map[string]any{
		"id":         s.ID,
		"name":       s.Name,
		"address":    s.Address,
		"lat":        s.Lat,
		"lng":        s.Lng,
		"type":       s.Type,
		"is_active":  s.IsActive,
		"is_online":  s.IsOnline(),
		"created_at": s.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if s.LastSeen != nil {
		resp["last_seen"] = s.LastSeen.Format("2006-01-02T15:04:05Z")
	} else {
		resp["last_seen"] = nil
	}
	return resp
}

func sensorsToResponse(sensors []domain.Sensor) []map[string]any {
	result := make([]map[string]any, 0, len(sensors))
	for i := range sensors {
		result = append(result, sensorToResponse(&sensors[i]))
	}
	return result
}

// Compile-time check: SensorHandler implements Status endpoint.
var _ interface {
	Status(http.ResponseWriter, *http.Request)
} = (*SensorHandler)(nil)
