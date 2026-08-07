package handler

import (
	"log/slog"
	"net/http"
)

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


