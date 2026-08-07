package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mintfary/aqi-platform/internal/domain"
	"github.com/mintfary/aqi-platform/internal/service"
)

// UserHandler реализует HTTP-обработчики управления пользователями.
type UserHandler struct {
	svc    *service.UserService
	logger *slog.Logger
}

// NewUserHandler создаёт обработчик пользователей.
func NewUserHandler(svc *service.UserService, logger *slog.Logger) *UserHandler {
	return &UserHandler{svc: svc, logger: logger}
}

// List godoc
// @Summary     Список пользователей
// @Description Возвращает постраничный список пользователей (только Admin).
// @Tags        users
// @Security    BearerAuth
// @Param       limit  query int false "Количество записей (по умолчанию 50)"
// @Param       offset query int false "Смещение (по умолчанию 0)"
// @Success     200 {object} map[string]any
// @Router      /users [get]
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)

	users, total, err := h.svc.List(r.Context(), limit, offset)
	if err != nil {
		handleError(w, h.logger, err)
		return
	}

	// Скрываем хэши паролей в ответе.
	type userResponse struct {
		ID        uuid.UUID   `json:"id"`
		Email     string      `json:"email"`
		Username  string      `json:"username"`
		Role      domain.Role `json:"role"`
		IsActive  bool        `json:"is_active"`
		CreatedAt string      `json:"created_at"`
	}
	resp := make([]userResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, userResponse{
			ID:        u.ID,
			Email:     u.Email,
			Username:  u.Username,
			Role:      u.Role,
			IsActive:  u.IsActive,
			CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	ok(w, map[string]any{
		"users":  resp,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// Create godoc
// @Summary     Создать пользователя
// @Tags        users
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body domain.CreateUserInput true "Данные пользователя"
// @Success     201 {object} map[string]any
// @Router      /users [post]
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in domain.CreateUserInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}

	u, err := h.svc.Create(r.Context(), in)
	if err != nil {
		handleError(w, h.logger, err)
		return
	}

	created(w, userToResponse(u))
}

// Get godoc
// @Summary     Получить пользователя
// @Tags        users
// @Security    BearerAuth
// @Param       id path string true "UUID пользователя"
// @Success     200 {object} map[string]any
// @Router      /users/{id} [get]
func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный UUID")
		return
	}

	u, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		handleError(w, h.logger, err)
		return
	}

	ok(w, userToResponse(u))
}

// Update godoc
// @Summary     Обновить пользователя
// @Tags        users
// @Security    BearerAuth
// @Param       id   path string                 true "UUID пользователя"
// @Param       body body domain.UpdateUserInput true "Поля для обновления"
// @Success     200 {object} map[string]any
// @Router      /users/{id} [patch]
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный UUID")
		return
	}

	var in domain.UpdateUserInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}

	u, err := h.svc.Update(r.Context(), id, in)
	if err != nil {
		handleError(w, h.logger, err)
		return
	}

	ok(w, userToResponse(u))
}

// Delete godoc
// @Summary     Деактивировать пользователя
// @Tags        users
// @Security    BearerAuth
// @Param       id path string true "UUID пользователя"
// @Success     204
// @Router      /users/{id} [delete]
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

// ── Вспомогательные функции ────────────────────────────────────────────────

// userToResponse преобразует User в ответ без хэша пароля.
func userToResponse(u *domain.User) map[string]any {
	return map[string]any{
		"id":         u.ID,
		"email":      u.Email,
		"username":   u.Username,
		"role":       u.Role,
		"is_active":  u.IsActive,
		"created_at": u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		"updated_at": u.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// parseUUID извлекает UUID из параметра URL.
func parseUUID(r *http.Request, param string) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, param))
}

// queryInt читает целочисленный query-параметр с дефолтом.
func queryInt(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return def
	}
	return n
}

// handleError транслирует доменные ошибки в HTTP-ответы.
func handleError(w http.ResponseWriter, logger *slog.Logger, err error) {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		writeError(w, appErr.HTTPCode(), appErr.Message)
		return
	}
	if errors.Is(err, domain.ErrNotFound) {
		writeError(w, http.StatusNotFound, "ресурс не найден")
		return
	}
	if errors.Is(err, domain.ErrAlreadyExists) {
		writeError(w, http.StatusConflict, "ресурс уже существует")
		return
	}
	logger.Error("internal error", "err", err)
	writeError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
}
