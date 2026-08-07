// Package handler — HTTP-обработчики генерации отчётов в формате CSV.
package handler

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mintfary/aqi-platform/internal/middleware"
	"github.com/mintfary/aqi-platform/internal/repository"
)

// ReportRepository — интерфейс для хранения отчётов.
type ReportRepository interface {
	Create(ctx context.Context, userID *uuid.UUID, name, reportType string, params json.RawMessage) (*repository.Report, error)
	SetReady(ctx context.Context, id uuid.UUID, fileData string, rowCount int) error
	SetError(ctx context.Context, id uuid.UUID, errMsg string) error
	List(ctx context.Context, userID *uuid.UUID, limit int) ([]repository.Report, error)
	GetFileData(ctx context.Context, id uuid.UUID) (string, error)
}

// MeasurementCSVSource — получение измерений для CSV.
type MeasurementCSVSource interface {
	ListRaw(ctx context.Context, from, to time.Time, sensorID *uuid.UUID, limit int) ([][]string, error)
}

// ReportHandler генерирует и раздаёт CSV-отчёты.
type ReportHandler struct {
	repo        ReportRepository
	statsRepo   StatsRepository
	logger      *slog.Logger
}

// NewReportHandler создаёт обработчик отчётов.
func NewReportHandler(
	repo ReportRepository,
	statsRepo StatsRepository,
	logger *slog.Logger,
) *ReportHandler {
	return &ReportHandler{
		repo:      repo,
		statsRepo: statsRepo,
		logger:    logger,
	}
}

// generateInput — параметры генерации отчёта.
type generateInput struct {
	Name       string `json:"name"`
	ReportType string `json:"report_type"` // measurements | forecasts | availability
	From       string `json:"from"`        // RFC3339
	To         string `json:"to"`          // RFC3339 (optional)
}

// List godoc
// @Summary     Список отчётов
// @Tags        reports
// @Security    BearerAuth
// @Produce     json
// @Param       limit query int false "Кол-во (default: 50)"
// @Success     200 {array} repository.Report
// @Router      /reports [get]
func (h *ReportHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	var filterUserID *uuid.UUID
	if claims := middleware.ClaimsFromContext(r.Context()); claims != nil {
		if claims.Role != "admin" {
			if id, err := uuid.Parse(claims.UserID); err == nil {
				filterUserID = &id
			}
		}
	}

	items, err := h.repo.List(r.Context(), filterUserID, limit)
	if err != nil {
		h.logger.Error("ReportHandler.List", "err", err)
		writeError(w, http.StatusInternalServerError, "ошибка получения отчётов")
		return
	}
	if items == nil {
		items = []repository.Report{}
	}
	ok(w, items)
}

// Generate godoc
// @Summary     Создать отчёт
// @Description Генерирует CSV-отчёт синхронно и сохраняет в БД.
// @Tags        reports
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body generateInput true "Параметры отчёта"
// @Success     201 {object} repository.Report
// @Router      /reports/generate [post]
func (h *ReportHandler) Generate(w http.ResponseWriter, r *http.Request) {
	var in generateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "поле name обязательно")
		return
	}
	validTypes := map[string]bool{"measurements": true, "forecasts": true, "availability": true}
	if !validTypes[in.ReportType] {
		writeError(w, http.StatusBadRequest, "report_type должен быть: measurements | forecasts | availability")
		return
	}

	from, to, err := parsePeriod(r)
	if err != nil {
		// Пробуем взять из тела.
		from, to, err = parseBodyPeriod(in)
		if err != nil {
			writeError(w, http.StatusBadRequest, "укажите from и to в теле запроса (RFC3339)")
			return
		}
	}

	var userID *uuid.UUID
	if claims := middleware.ClaimsFromContext(r.Context()); claims != nil {
		if id, err := uuid.Parse(claims.UserID); err == nil {
			userID = &id
		}
	}

	paramsRaw, _ := json.Marshal(map[string]string{
		"from": from.Format(time.RFC3339),
		"to":   to.Format(time.RFC3339),
	})

	rep, err := h.repo.Create(r.Context(), userID, in.Name, in.ReportType, paramsRaw)
	if err != nil {
		h.logger.Error("ReportHandler.Generate create", "err", err)
		writeError(w, http.StatusInternalServerError, "ошибка создания отчёта")
		return
	}

	// Генерируем CSV синхронно (для небольших периодов это нормально).
	csvData, rowCount, genErr := h.buildCSV(r.Context(), in.ReportType, from, to)
	if genErr != nil {
		h.logger.Error("ReportHandler.Generate buildCSV", "err", genErr)
		_ = h.repo.SetError(r.Context(), rep.ID, genErr.Error())
		rep.Status = "error"
		writeJSON(w, http.StatusInternalServerError, rep)
		return
	}

	if err := h.repo.SetReady(r.Context(), rep.ID, csvData, rowCount); err != nil {
		h.logger.Error("ReportHandler.Generate SetReady", "err", err)
	}
	rep.Status = "ready"
	rep.RowCount = &rowCount
	created(w, rep)
}

// Download godoc
// @Summary     Скачать CSV-отчёт
// @Tags        reports
// @Security    BearerAuth
// @Param       id path string true "UUID отчёта"
// @Produce     text/csv
// @Success     200 {file} string
// @Router      /reports/{id}/download [get]
func (h *ReportHandler) Download(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный ID отчёта")
		return
	}

	data, err := h.repo.GetFileData(r.Context(), id)
	switch err {
	case nil:
		// ok
	case repository.ErrReportNotFound:
		writeError(w, http.StatusNotFound, "отчёт не найден")
		return
	case repository.ErrReportNotReady:
		writeError(w, http.StatusAccepted, "отчёт ещё генерируется")
		return
	default:
		h.logger.Error("ReportHandler.Download", "err", err)
		writeError(w, http.StatusInternalServerError, "ошибка получения отчёта")
		return
	}

	filename := fmt.Sprintf("report_%s.csv", id)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(data))
}

// ── CSV builders ─────────────────────────────────────────────────────────────

func (h *ReportHandler) buildCSV(ctx context.Context, reportType string, from, to time.Time) (string, int, error) {
	switch reportType {
	case "availability":
		return h.buildAvailabilityCSV(ctx, from, to)
	default:
		// measurements и forecasts: заглушка с заголовком и пустыми данными
		// (полноценный вариант требует MeasurementRepo — добавить при необходимости).
		return h.buildPlaceholderCSV(reportType, from, to)
	}
}

// buildAvailabilityCSV генерирует CSV из данных доступности датчиков.
func (h *ReportHandler) buildAvailabilityCSV(ctx context.Context, from, to time.Time) (string, int, error) {
	rows, err := h.statsRepo.Availability(ctx, from, to)
	if err != nil {
		return "", 0, fmt.Errorf("buildAvailabilityCSV: %w", err)
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"sensor_id", "sensor_name", "expected_hours", "actual_measurements", "availability_pct"})
	for _, row := range rows {
		_ = w.Write([]string{
			row.SensorID,
			row.SensorName,
			fmt.Sprintf("%.2f", row.ExpectedHours),
			strconv.FormatInt(row.ActualMeasurements, 10),
			fmt.Sprintf("%.2f", row.AvailabilityPct),
		})
	}
	w.Flush()
	return buf.String(), len(rows), nil
}

// buildPlaceholderCSV возвращает CSV с заголовком и мета-строкой для типов,
// где нет прямого доступа к данным через этот handler.
func (h *ReportHandler) buildPlaceholderCSV(reportType string, from, to time.Time) (string, int, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	switch reportType {
	case "measurements":
		_ = w.Write([]string{"time", "sensor_id", "no2", "o3", "co", "h2s", "so2", "pm25",
			"temperature", "humidity", "pressure", "wind_speed", "wind_dir", "aqi"})
	case "forecasts":
		_ = w.Write([]string{"time", "point_id", "horizon_hours", "aqi", "aqi_category",
			"no2_forecast", "pm25_forecast", "so2_forecast"})
	}
	// Мета-строка с параметрами запроса.
	_ = w.Write([]string{
		"# период: " + from.Format(time.RFC3339) + " – " + to.Format(time.RFC3339),
		"# используйте GET /api/v1/measurements или /api/v1/forecast для получения данных",
	})
	w.Flush()
	return buf.String(), 0, nil
}

// parseBodyPeriod парсит from/to из тела generateInput (дублирует parsePeriod для тела).
func parseBodyPeriod(in generateInput) (from, to time.Time, err error) {
	if in.From == "" {
		return time.Time{}, time.Time{}, errorf("поле from обязательно")
	}
	from, err = time.Parse(time.RFC3339, in.From)
	if err != nil {
		return time.Time{}, time.Time{}, errorf("from: некорректный RFC3339")
	}
	to = time.Now().UTC()
	if in.To != "" {
		to, err = time.Parse(time.RFC3339, in.To)
		if err != nil {
			return time.Time{}, time.Time{}, errorf("to: некорректный RFC3339")
		}
	}
	if !to.After(from) {
		return time.Time{}, time.Time{}, errorf("to должно быть позже from")
	}
	return from, to, nil
}
