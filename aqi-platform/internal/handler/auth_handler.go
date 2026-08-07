package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/mintfary/aqi-platform/internal/domain"
	"github.com/mintfary/aqi-platform/internal/middleware"
	"github.com/mintfary/aqi-platform/internal/service"
)

// AuthHandler реализует HTTP-обработчики аутентификации.
// Тонкий слой: декодирует запрос → вызывает AuthService → формирует ответ.
type AuthHandler struct {
	svc    *service.AuthService
	logger *slog.Logger
}

// NewAuthHandler создаёт обработчик аутентификации.
func NewAuthHandler(svc *service.AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{svc: svc, logger: logger}
}

// Login godoc
// @Summary     Вход в систему
// @Description Проверяет email + пароль, возвращает JWT пару (access + refresh).
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body service.LoginInput true "Учётные данные"
// @Success     200 {object} service.TokenPair
// @Failure     400 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Failure     429 {object} map[string]string
// @Router      /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var in service.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}
	if in.Email == "" || in.Password == "" {
		writeError(w, http.StatusBadRequest, "email и password обязательны")
		return
	}

	pair, err := h.svc.Login(r.Context(), in)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	ok(w, pair)
}

// Refresh godoc
// @Summary     Обновление токенов
// @Description Принимает refresh token, возвращает новую пару access + refresh.
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body object{refresh_token=string} true "Refresh token"
// @Success     200 {object} service.TokenPair
// @Failure     401 {object} map[string]string
// @Router      /auth/refresh [post]
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "поле refresh_token обязательно")
		return
	}

	pair, err := h.svc.RefreshTokens(r.Context(), body.RefreshToken)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	ok(w, pair)
}

// Logout godoc
// @Summary     Выход из системы
// @Description Добавляет текущий access token в блеклист.
// @Tags        auth
// @Security    BearerAuth
// @Success     204
// @Failure     401 {object} map[string]string
// @Router      /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if err := h.svc.Logout(r.Context(), claims); err != nil {
		h.logger.Error("logout error", "err", err)
		// Не возвращаем ошибку клиенту — он всё равно должен удалить токен локально.
	}
	noContent(w)
}

// ForgotPassword — заглушка (требует SMTP-интеграции, Sprint 4).
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "сброс пароля через email будет реализован в Sprint 4")
}

// ResetPassword — заглушка (требует SMTP-интеграции, Sprint 4).
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "сброс пароля через email будет реализован в Sprint 4")
}

// handleServiceError транслирует доменные ошибки в HTTP-коды.
func (h *AuthHandler) handleServiceError(w http.ResponseWriter, err error) {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		writeError(w, appErr.HTTPCode(), appErr.Message)
		return
	}
	h.logger.Error("auth service error", "err", err)
	writeError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
}

// ValidateAPIToken реализует middleware.APITokenChecker для API-токенов.
// Используется датчиками при загрузке данных.
func (h *AuthHandler) ValidateAPIToken(_ context.Context, _ string) (*domain.User, error) {
	// Sprint 4: проверка по таблице api_tokens.
	return nil, domain.ErrUnauthorized
}
