// Package handler — HTTP-обработчики обратной связи.
package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/mintfary/aqi-platform/internal/domain"
	"github.com/mintfary/aqi-platform/internal/middleware"
)

// FeedbackRepository — минимальный интерфейс для хранения обращений.
type FeedbackRepository interface {
	Create(ctx context.Context, in domain.CreateFeedbackInput, userID *uuid.UUID) (*domain.Feedback, error)
	List(ctx context.Context, userID *uuid.UUID, limit, offset int) ([]domain.Feedback, error)
}

// FeedbackHandler обрабатывает обращения пользователей (форма обратной связи).
type FeedbackHandler struct {
	repo   FeedbackRepository
	logger *slog.Logger
}

// NewFeedbackHandler создаёт обработчик обратной связи.
func NewFeedbackHandler(repo FeedbackRepository, logger *slog.Logger) *FeedbackHandler {
	return &FeedbackHandler{repo: repo, logger: logger}
}

// Create godoc
// @Summary     Создать обращение
// @Description Сохраняет сообщение обратной связи. Доступно для всех авторизованных пользователей.
// @Tags        feedback
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body domain.CreateFeedbackInput true "Обращение"
// @Success     201 {object} domain.Feedback
// @Failure     400 {object} map[string]string
// @Router      /feedback [post]
func (h *FeedbackHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in domain.CreateFeedbackInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}
	if in.Subject == "" || in.Message == "" {
		writeError(w, http.StatusBadRequest, "поля subject и message обязательны")
		return
	}

	// Извлекаем ID текущего пользователя (если авторизован).
	var userID *uuid.UUID
	if claims := middleware.ClaimsFromContext(r.Context()); claims != nil {
		if id, err := uuid.Parse(claims.UserID); err == nil {
			userID = &id
		}
	}

	fb, err := h.repo.Create(r.Context(), in, userID)
	if err != nil {
		h.logger.Error("FeedbackHandler.Create", "err", err)
		writeError(w, http.StatusInternalServerError, "ошибка сохранения обращения")
		return
	}

	created(w, fb)
}

// List godoc
// @Summary     Список обращений
// @Description Admin видит все обращения, обычный пользователь — только свои.
// @Tags        feedback
// @Security    BearerAuth
// @Produce     json
// @Param       limit  query int false "Кол-во записей (default: 50)"
// @Param       offset query int false "Смещение (default: 0)"
// @Success     200 {array} domain.Feedback
// @Router      /feedback [get]
func (h *FeedbackHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 50
	}

	// Admin видит все; остальные — только свои.
	var filterUserID *uuid.UUID
	if claims := middleware.ClaimsFromContext(r.Context()); claims != nil {
		if claims.Role != string(domain.RoleAdmin) {
			if id, err := uuid.Parse(claims.UserID); err == nil {
				filterUserID = &id
			}
		}
	}

	items, err := h.repo.List(r.Context(), filterUserID, limit, offset)
	if err != nil {
		h.logger.Error("FeedbackHandler.List", "err", err)
		writeError(w, http.StatusInternalServerError, "ошибка получения обращений")
		return
	}

	if items == nil {
		items = []domain.Feedback{}
	}
	ok(w, items)
}
