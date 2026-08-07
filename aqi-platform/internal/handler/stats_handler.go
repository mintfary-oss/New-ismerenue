// Package handler — HTTP-обработчики статистики платформы (Admin-only).
package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/mintfary/aqi-platform/internal/repository"
)

// StatsRepository — минимальный интерфейс для аналитических запросов.
type StatsRepository interface {
	Availability(ctx context.Context, from, to time.Time) ([]repository.SensorAvailability, error)
	DataCoverage(ctx context.Context, from, to time.Time) ([]repository.ParameterCoverage, error)
}

// StatsHandler обрабатывает аналитические запросы (только Admin).
type StatsHandler struct {
	repo   StatsRepository
	logger *slog.Logger
}

// NewStatsHandler создаёт обработчик статистики.
func NewStatsHandler(repo StatsRepository, logger *slog.Logger) *StatsHandler {
	return &StatsHandler{repo: repo, logger: logger}
}

// Availability godoc
// @Summary     Доступность датчиков
// @Description Процент доступности каждого активного датчика за период.
// @Tags        stats
// @Security    BearerAuth
// @Produce     json
// @Param       from  query string true  "Начало периода (RFC3339)"
// @Param       to    query string false "Конец периода (RFC3339, default: now)"
// @Success     200 {array} repository.SensorAvailability
// @Failure     400 {object} map[string]string
// @Router      /stats/availability [get]
func (h *StatsHandler) Availability(w http.ResponseWriter, r *http.Request) {
	from, to, err := parsePeriod(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.repo.Availability(r.Context(), from, to)
	if err != nil {
		h.logger.Error("StatsHandler.Availability", "err", err)
		writeError(w, http.StatusInternalServerError, "ошибка вычисления доступности")
		return
	}

	if result == nil {
		result = []repository.SensorAvailability{}
	}
	ok(w, result)
}

// DataCoverage godoc
// @Summary     Покрытие данными
// @Description Процент непустых значений по каждому параметру за период.
// @Tags        stats
// @Security    BearerAuth
// @Produce     json
// @Param       from  query string true  "Начало периода (RFC3339)"
// @Param       to    query string false "Конец периода (RFC3339, default: now)"
// @Success     200 {array} repository.ParameterCoverage
// @Failure     400 {object} map[string]string
// @Router      /stats/data-coverage [get]
func (h *StatsHandler) DataCoverage(w http.ResponseWriter, r *http.Request) {
	from, to, err := parsePeriod(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.repo.DataCoverage(r.Context(), from, to)
	if err != nil {
		h.logger.Error("StatsHandler.DataCoverage", "err", err)
		writeError(w, http.StatusInternalServerError, "ошибка вычисления покрытия данными")
		return
	}

	if result == nil {
		result = []repository.ParameterCoverage{}
	}
	ok(w, result)
}

// parsePeriod читает параметры from/to из query string запроса.
// from — обязателен (RFC3339). to — опционален (default: now).
func parsePeriod(r *http.Request) (from, to time.Time, err error) {
	fromStr := r.URL.Query().Get("from")
	if fromStr == "" {
		return time.Time{}, time.Time{}, errorf("параметр from обязателен")
	}
	from, err = time.Parse(time.RFC3339, fromStr)
	if err != nil {
		return time.Time{}, time.Time{}, errorf("from: некорректный формат RFC3339")
	}

	to = time.Now().UTC()
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			return time.Time{}, time.Time{}, errorf("to: некорректный формат RFC3339")
		}
	}

	if !to.After(from) {
		return time.Time{}, time.Time{}, errorf("to должно быть позже from")
	}
	return from, to, nil
}

// errorf — мини-хелпер для ошибок парсинга (обёртка над errors.New).
func errorf(msg string) error {
	return errors.New(msg)
}
