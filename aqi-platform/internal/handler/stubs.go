package handler

import (
	"log/slog"
	"net/http"
)

// ForecastHandler — прогнозы качества воздуха. Реализация: Sprint 3.
type ForecastHandler struct{ logger *slog.Logger }

func (h *ForecastHandler) Points(w http.ResponseWriter, _ *http.Request)      { writeJSON(w, 501, map[string]string{"status": "not_implemented", "sprint": "3"}) }
func (h *ForecastHandler) Current(w http.ResponseWriter, _ *http.Request)     { writeJSON(w, 501, map[string]string{"status": "not_implemented", "sprint": "3"}) }
func (h *ForecastHandler) CityAverage(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 501, map[string]string{"status": "not_implemented", "sprint": "3"}) }
func (h *ForecastHandler) ByPoint(w http.ResponseWriter, _ *http.Request)     { writeJSON(w, 501, map[string]string{"status": "not_implemented", "sprint": "3"}) }
func (h *ForecastHandler) ByDistrict(w http.ResponseWriter, _ *http.Request)  { writeJSON(w, 501, map[string]string{"status": "not_implemented", "sprint": "3"}) }

// TokenHandler — управление API-токенами. Реализация: Sprint 4.
type TokenHandler struct{ logger *slog.Logger }

func (h *TokenHandler) List(w http.ResponseWriter, _ *http.Request)      { writeJSON(w, 501, map[string]string{"status": "not_implemented", "sprint": "4"}) }
func (h *TokenHandler) Create(w http.ResponseWriter, _ *http.Request)    { writeJSON(w, 501, map[string]string{"status": "not_implemented", "sprint": "4"}) }
func (h *TokenHandler) Delete(w http.ResponseWriter, _ *http.Request)    { writeJSON(w, 501, map[string]string{"status": "not_implemented", "sprint": "4"}) }

// ReportHandler — генерация отчётов. Реализация: Sprint 4.
type ReportHandler struct{ logger *slog.Logger }

func (h *ReportHandler) List(w http.ResponseWriter, _ *http.Request)     { writeJSON(w, 501, map[string]string{"status": "not_implemented", "sprint": "4"}) }
func (h *ReportHandler) Generate(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 501, map[string]string{"status": "not_implemented", "sprint": "4"}) }
func (h *ReportHandler) Download(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 501, map[string]string{"status": "not_implemented", "sprint": "4"}) }

// StatsHandler — статистика платформы. Реализация: Sprint 4.
type StatsHandler struct{ logger *slog.Logger }

func (h *StatsHandler) Availability(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 501, map[string]string{"status": "not_implemented", "sprint": "4"}) }
func (h *StatsHandler) DataCoverage(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 501, map[string]string{"status": "not_implemented", "sprint": "4"}) }

// FeedbackHandler — форма обратной связи. Реализация: Sprint 4.
type FeedbackHandler struct{ logger *slog.Logger }

func (h *FeedbackHandler) Create(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 501, map[string]string{"status": "not_implemented", "sprint": "4"}) }
func (h *FeedbackHandler) List(w http.ResponseWriter, _ *http.Request)   { writeJSON(w, 501, map[string]string{"status": "not_implemented", "sprint": "4"}) }

// WidgetHandler — публичный виджет. Реализация: Sprint 3.
type WidgetHandler struct{ logger *slog.Logger }

func (h *WidgetHandler) Index(w http.ResponseWriter, _ *http.Request)    { writeJSON(w, 501, map[string]string{"status": "not_implemented", "sprint": "3"}) }
func (h *WidgetHandler) Data(w http.ResponseWriter, _ *http.Request)     { writeJSON(w, 501, map[string]string{"status": "not_implemented", "sprint": "3"}) }
func (h *WidgetHandler) Forecast(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 501, map[string]string{"status": "not_implemented", "sprint": "3"}) }
func (h *WidgetHandler) Weather(w http.ResponseWriter, _ *http.Request)  { writeJSON(w, 501, map[string]string{"status": "not_implemented", "sprint": "3"}) }
