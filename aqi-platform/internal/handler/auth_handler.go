package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mintfary/aqi-platform/internal/domain"
	"github.com/mintfary/aqi-platform/internal/middleware"
	"github.com/mintfary/aqi-platform/internal/service"
)

// PasswordResetTokenStore — интерфейс хранилища токенов сброса пароля (Redis).
type PasswordResetTokenStore interface {
	Set(ctx context.Context, token, email string) error
	GetEmail(ctx context.Context, token string) (string, error)
}

// EmailSender — интерфейс отправки email-уведомлений.
type EmailSender interface {
	SendPasswordReset(toEmail, resetToken, baseURL string) error
	IsConfigured() bool
}

// AuthHandler реализует HTTP-обработчики аутентификации.
// Тонкий слой: декодирует запрос → вызывает AuthService → формирует ответ.
type AuthHandler struct {
	svc        *service.AuthService
	resetStore PasswordResetTokenStore // nil если SMTP не настроен
	mailer     EmailSender             // nil если SMTP не настроен
	baseURL    string
	logger     *slog.Logger
}

// NewAuthHandler создаёт обработчик аутентификации.
func NewAuthHandler(svc *service.AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{svc: svc, logger: logger}
}

// WithPasswordReset добавляет поддержку сброса пароля через email.
func (h *AuthHandler) WithPasswordReset(store PasswordResetTokenStore, mailer EmailSender, baseURL string) {
	h.resetStore = store
	h.mailer = mailer
	h.baseURL = strings.TrimRight(baseURL, "/")
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
		// Не возвращаем ошибку клиенту — он должен удалить токен локально.
	}
	noContent(w)
}

// ForgotPassword godoc
// @Summary     Запрос сброса пароля
// @Description Отправляет письмо со ссылкой для сброса пароля (если SMTP настроен).
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body object{email=string} true "Email пользователя"
// @Success     204  "Письмо отправлено (или SMTP не настроен — ответ одинаков)"
// @Failure     400  {object} map[string]string
// @Router      /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	if h.resetStore == nil || h.mailer == nil {
		// SMTP не настроен — отвечаем 204 чтобы не раскрывать конфигурацию
		noContent(w)
		return
	}

	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}
	if body.Email == "" {
		writeError(w, http.StatusBadRequest, "email обязателен")
		return
	}

	ctx := r.Context()

	// Генерируем безопасный случайный токен (32 байта = 64 hex символа).
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		h.logger.Error("forgot password: rand.Read", "err", err)
		writeError(w, http.StatusInternalServerError, "внутренняя ошибка")
		return
	}
	token := hex.EncodeToString(tokenBytes)

	// Сохраняем токен → email в Redis.
	// Намеренно не проверяем существование пользователя здесь —
	// это предотвращает user enumeration (timing attack).
	if err := h.resetStore.Set(ctx, token, body.Email); err != nil {
		h.logger.Error("forgot password: store set", "email", body.Email, "err", err)
		// Тихий fail — не раскрываем информацию
		noContent(w)
		return
	}

	// Отправляем письмо в фоне (не блокируем ответ клиенту).
	go func() {
		if err := h.mailer.SendPasswordReset(body.Email, token, h.baseURL); err != nil {
			h.logger.Warn("forgot password: send email failed",
				"email", body.Email,
				"err", err,
			)
		} else {
			h.logger.Info("forgot password: email sent", "email", body.Email)
		}
	}()

	// Всегда отвечаем 204 — не раскрываем существует ли пользователь.
	noContent(w)
}

// ResetPassword godoc
// @Summary     Сброс пароля по токену
// @Description Устанавливает новый пароль если токен действителен (one-time use).
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body object{token=string,new_password=string} true "Токен и новый пароль"
// @Success     204  "Пароль изменён"
// @Failure     400  {object} map[string]string
// @Failure     404  {object} map[string]string "Токен недействителен или истёк"
// @Router      /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if h.resetStore == nil {
		writeError(w, http.StatusNotImplemented, "сброс пароля через email не настроен")
		return
	}

	var body struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}
	if body.Token == "" || body.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "поля token и new_password обязательны")
		return
	}
	if len(body.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "пароль должен содержать не менее 8 символов")
		return
	}

	ctx := r.Context()

	// Получаем email по токену (one-time: токен удаляется при GetEmail).
	email, err := h.resetStore.GetEmail(ctx, body.Token)
	if err != nil {
		h.logger.Error("reset password: get email", "err", err)
		writeError(w, http.StatusInternalServerError, "внутренняя ошибка")
		return
	}
	if email == "" {
		writeError(w, http.StatusBadRequest, "токен недействителен или истёк")
		return
	}

	// Меняем пароль через AuthService.
	if err := h.svc.ResetPassword(ctx, email, body.NewPassword); err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.logger.Info("password reset successful", "email", email)
	noContent(w)
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
