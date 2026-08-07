// Package handler — HTTP-обработчики API-токенов.
package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mintfary/aqi-platform/internal/domain"
	"github.com/mintfary/aqi-platform/internal/middleware"
	"github.com/mintfary/aqi-platform/internal/service"
)

// TokenHandler управляет API-токенами текущего пользователя.
type TokenHandler struct {
	svc    *service.TokenService
	logger *slog.Logger
}

// NewTokenHandler создаёт обработчик API-токенов.
func NewTokenHandler(svc *service.TokenService, logger *slog.Logger) *TokenHandler {
	return &TokenHandler{svc: svc, logger: logger}
}

// List godoc
// @Summary     Список API-токенов
// @Description Возвращает все API-токены текущего пользователя (без значений токенов).
// @Tags        tokens
// @Security    BearerAuth
// @Produce     json
// @Success     200 {array}  domain.APIToken
// @Failure     401 {object} map[string]string
// @Router      /tokens [get]
func (h *TokenHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDFromCtx(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "требуется авторизация")
		return
	}

	tokens, err := h.svc.List(r.Context(), userID)
	if err != nil {
		h.logger.Error("TokenHandler.List", "err", err)
		writeError(w, http.StatusInternalServerError, "ошибка получения токенов")
		return
	}

	if tokens == nil {
		tokens = []domain.APIToken{}
	}
	ok(w, tokens)
}

// Create godoc
// @Summary     Создать API-токен
// @Description Генерирует новый API-токен. Значение токена возвращается ОДИН раз.
// @Tags        tokens
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body domain.CreateAPITokenInput true "Параметры токена"
// @Success     201 {object} service.APITokenCreateResult
// @Failure     400 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Router      /tokens [post]
func (h *TokenHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDFromCtx(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "требуется авторизация")
		return
	}

	var in domain.CreateAPITokenInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "поле name обязательно")
		return
	}

	result, err := h.svc.Create(r.Context(), userID, in)
	if err != nil {
		var appErr *domain.AppError
		if errors.As(err, &appErr) {
			writeError(w, appErr.Code, appErr.Message)
			return
		}
		h.logger.Error("TokenHandler.Create", "err", err)
		writeError(w, http.StatusInternalServerError, "ошибка создания токена")
		return
	}

	created(w, result)
}

// Delete godoc
// @Summary     Удалить API-токен
// @Description Удаляет API-токен по ID (только свой).
// @Tags        tokens
// @Security    BearerAuth
// @Param       id path string true "UUID токена"
// @Success     204
// @Failure     400 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /tokens/{id} [delete]
func (h *TokenHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDFromCtx(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "требуется авторизация")
		return
	}

	tokenID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный ID токена")
		return
	}

	if err := h.svc.Delete(r.Context(), userID, tokenID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "токен не найден")
			return
		}
		h.logger.Error("TokenHandler.Delete", "err", err)
		writeError(w, http.StatusInternalServerError, "ошибка удаления токена")
		return
	}

	noContent(w)
}

// ── helpers ──────────────────────────────────────────────────────────────────

// userIDFromCtx извлекает UUID текущего пользователя из JWT-claims контекста.
func userIDFromCtx(r *http.Request) (uuid.UUID, error) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		return uuid.Nil, domain.ErrUnauthorized
	}
	return uuid.Parse(claims.UserID)
}
