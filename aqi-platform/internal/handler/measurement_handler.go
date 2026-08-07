package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mintfary/aqi-platform/internal/domain"
	"github.com/mintfary/aqi-platform/internal/service"
)

// MeasurementHandler реализует HTTP-обработчики работы с измерениями.
type MeasurementHandler struct {
	svc    *service.MeasurementService
	logger *slog.Logger
}

// NewMeasurementHandler создаёт обработчик измерений.
func NewMeasurementHandler(svc *service.MeasurementService, logger *slog.Logger) *MeasurementHandler {
	return &MeasurementHandler{svc: svc, logger: logger}
}

// List godoc
// @Summary     Список измерений
// @Description Возвращает измерения за период. Параметр period: raw | 1h | 1d.
// @Tags        measurements
// @Security    BearerAuth
// @Param       from      query string false "Начало периода (RFC3339)"
// @Param       to        query string false "Конец периода (RFC3339)"
// @Param       sensor_id query string false "UUID датчика"
// @Param       period    query string false "Агрегация: raw | 1h | 1d"
// @Param       limit     query int    false "Лимит записей (макс. 10000)"
// @Success     200 {object} map[string]any
// @Router      /measurements [get]
func (h *MeasurementHandler) List(w http.ResponseWriter, r *http.Request) {
	f, err := parseMeasurementFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	data, err := h.svc.List(r.Context(), f)
	if err != nil {
		handleError(w, h.logger, err)
		return
	}

	ok(w, map[string]any{
		"data":   data,
		"from":   f.From.Format(time.RFC3339),
		"to":     f.To.Format(time.RFC3339),
		"period": f.Period,
	})
}

// Latest godoc
// @Summary     Последние измерения по всем датчикам
// @Description Используется для дашборда и карты. Возвращает AQI для каждого активного датчика.
// @Tags        measurements
// @Security    BearerAuth
// @Success     200 {object} map[string]any
// @Router      /measurements/latest [get]
func (h *MeasurementHandler) Latest(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.Latest(r.Context())
	if err != nil {
		handleError(w, h.logger, err)
		return
	}

	type item struct {
		SensorID    uuid.UUID          `json:"sensor_id"`
		SensorName  string             `json:"sensor_name"`
		Lat         float64            `json:"lat"`
		Lng         float64            `json:"lng"`
		Time        string             `json:"time"`
		AQI         int                `json:"aqi"`
		AQICategory domain.AQICategory `json:"aqi_category"`
		AQILabel    string             `json:"aqi_label"`
		AQIColor    string             `json:"aqi_color"`
		PM25        *float64           `json:"pm25,omitempty"`
		NO2         *float64           `json:"no2,omitempty"`
		IsOnline    bool               `json:"is_online"`
	}

	resp := make([]item, 0, len(result))
	for _, lm := range result {
		resp = append(resp, item{
			SensorID:    lm.Sensor.ID,
			SensorName:  lm.Sensor.Name,
			Lat:         lm.Sensor.Lat,
			Lng:         lm.Sensor.Lng,
			Time:        lm.Measurement.Time.Format(time.RFC3339),
			AQI:         lm.AQI,
			AQICategory: lm.AQICategory,
			AQILabel:    domain.AQICategoryLabel(lm.AQICategory),
			AQIColor:    domain.AQICategoryColor(lm.AQICategory),
			PM25:        lm.Measurement.PM25,
			NO2:         lm.Measurement.NO2,
			IsOnline:    lm.Sensor.IsOnline(),
		})
	}

	ok(w, map[string]any{
		"sensors":    resp,
		"count":      len(resp),
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// Aggregate godoc
// @Summary     Агрегированные данные
// @Description Возвращает среднечасовые/суточные значения за период.
// @Tags        measurements
// @Security    BearerAuth
// @Param       from      query string false "Начало периода (RFC3339)"
// @Param       to        query string false "Конец периода (RFC3339)"
// @Param       sensor_id query string false "UUID датчика"
// @Param       bucket    query string false "Интервал: 1h | 1d | 1w"
// @Success     200 {object} map[string]any
// @Router      /measurements/aggregate [get]
func (h *MeasurementHandler) Aggregate(w http.ResponseWriter, r *http.Request) {
	f, err := parseMeasurementFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	bucket := r.URL.Query().Get("bucket")
	if bucket == "" {
		bucket = "1 hour"
	}

	result, err := h.svc.Aggregate(r.Context(), f, bucket)
	if err != nil {
		handleError(w, h.logger, err)
		return
	}

	ok(w, map[string]any{
		"data":   result,
		"bucket": bucket,
		"from":   f.From.Format(time.RFC3339),
		"to":     f.To.Format(time.RFC3339),
	})
}

// ── IngestHandler ─────────────────────────────────────────────────────────

// IngestHandler реализует приём данных от датчиков.
type IngestHandler struct {
	svc    *service.MeasurementService
	logger *slog.Logger
}

// NewIngestHandler создаёт обработчик загрузки данных.
func NewIngestHandler(svc *service.MeasurementService, logger *slog.Logger) *IngestHandler {
	return &IngestHandler{svc: svc, logger: logger}
}

// Upload godoc
// @Summary     Загрузить измерения от датчиков
// @Description Принимает одно или массив измерений. Аутентификация по API-токену.
// @Tags        ingest
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body []domain.MeasurementInput true "Измерения"
// @Success     202
// @Router      /ingest/data [post]
func (h *IngestHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// Поддерживаем как одиночный объект, так и массив.
	var items []domain.MeasurementInput

	dec := json.NewDecoder(r.Body)
	dec.UseNumber()

	// Пробуем декодировать как массив.
	if err := dec.Decode(&items); err != nil {
		// Попытка декодировать как одиночный объект.
		var single domain.MeasurementInput
		if err2 := json.NewDecoder(r.Body).Decode(&single); err2 != nil {
			writeError(w, http.StatusBadRequest, "некорректный JSON: ожидается объект или массив")
			return
		}
		items = []domain.MeasurementInput{single}
	}

	if len(items) == 0 {
		writeError(w, http.StatusBadRequest, "пустой пакет данных")
		return
	}

	if err := h.svc.IngestBatch(r.Context(), items); err != nil {
		handleError(w, h.logger, err)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, http.StatusAccepted, map[string]any{
		"accepted": len(items),
		"status":   "ok",
	})
}

// History возвращает историю загрузок (заглушка — Sprint 4).
func (h *IngestHandler) History(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotImplemented, "история загрузок — Sprint 4")
}

// GetRules возвращает правила валидации (заглушка — Sprint 4).
func (h *IngestHandler) GetRules(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotImplemented, "правила валидации — Sprint 4")
}

// UpdateRules обновляет правила валидации (заглушка — Sprint 4).
func (h *IngestHandler) UpdateRules(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotImplemented, "правила валидации — Sprint 4")
}

// ── Вспомогательные функции ────────────────────────────────────────────────

// parseMeasurementFilter читает параметры фильтра из query string.
func parseMeasurementFilter(r *http.Request) (domain.MeasurementFilter, error) {
	q := r.URL.Query()
	now := time.Now().UTC()

	f := domain.MeasurementFilter{
		From:   now.Add(-24 * time.Hour), // по умолчанию: последние 24ч
		To:     now,
		Period: q.Get("period"),
		Limit:  queryInt(r, "limit", 1000),
	}

	if fromStr := q.Get("from"); fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return f, domain.ErrInvalidInput
		}
		f.From = t
	}

	if toStr := q.Get("to"); toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			return f, domain.ErrInvalidInput
		}
		f.To = t
	}

	if sidStr := q.Get("sensor_id"); sidStr != "" {
		id, err := uuid.Parse(sidStr)
		if err != nil {
			return f, domain.ErrInvalidInput
		}
		f.SensorID = &id
	}

	if f.To.Before(f.From) {
		return f, domain.ErrInvalidInput
	}

	return f, nil
}
