package domain

import (
	"errors"
	"fmt"
	"net/http"
)

// Стандартные доменные ошибки.
var (
	ErrNotFound          = errors.New("ресурс не найден")
	ErrAlreadyExists     = errors.New("ресурс уже существует")
	ErrInvalidInput      = errors.New("некорректные входные данные")
	ErrUnauthorized      = errors.New("требуется аутентификация")
	ErrForbidden         = errors.New("недостаточно прав")
	ErrInvalidCredentials = errors.New("неверный логин или пароль")
	ErrTokenExpired      = errors.New("токен истёк")
	ErrTokenInvalid      = errors.New("недействительный токен")
	ErrAccountLocked     = errors.New("аккаунт временно заблокирован")
	ErrAccountDisabled   = errors.New("аккаунт отключён")
	ErrWeakPassword      = errors.New("пароль не соответствует требованиям сложности")
	ErrRateLimitExceeded = errors.New("превышен лимит запросов")
	ErrInternal          = errors.New("внутренняя ошибка сервера")
)

// AppError — структурированная ошибка с HTTP-кодом и деталями.
type AppError struct {
	Code    int    // HTTP статус-код
	Message string // Сообщение для пользователя (на русском)
	Err     error  // Оригинальная ошибка (для логирования)
}

// Error реализует интерфейс error.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap позволяет использовать errors.Is/As.
func (e *AppError) Unwrap() error { return e.Err }

// HTTPCode возвращает HTTP-код для ответа клиенту.
func (e *AppError) HTTPCode() int { return e.Code }

// NewAppError создаёт AppError с заданным кодом и сообщением.
func NewAppError(code int, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

// Вспомогательные конструкторы для частых случаев.

// ErrBadRequest создаёт ошибку 400.
func ErrBadRequest(msg string, err error) *AppError {
	return NewAppError(http.StatusBadRequest, msg, err)
}

// ErrUnauthorizedErr создаёт ошибку 401.
func ErrUnauthorizedErr(msg string) *AppError {
	return NewAppError(http.StatusUnauthorized, msg, ErrUnauthorized)
}

// ErrForbiddenErr создаёт ошибку 403.
func ErrForbiddenErr(msg string) *AppError {
	return NewAppError(http.StatusForbidden, msg, ErrForbidden)
}

// ErrNotFoundErr создаёт ошибку 404.
func ErrNotFoundErr(resource string) *AppError {
	return NewAppError(http.StatusNotFound, resource+" не найден", ErrNotFound)
}

// ErrConflict создаёт ошибку 409.
func ErrConflict(msg string) *AppError {
	return NewAppError(http.StatusConflict, msg, ErrAlreadyExists)
}

// ErrTooManyRequests создаёт ошибку 429.
func ErrTooManyRequests() *AppError {
	return NewAppError(http.StatusTooManyRequests, "превышен лимит запросов", ErrRateLimitExceeded)
}

// ErrInternalServer создаёт ошибку 500.
func ErrInternalServer(err error) *AppError {
	return NewAppError(http.StatusInternalServerError, "внутренняя ошибка сервера", err)
}

// IsNotFound проверяет, является ли ошибка "не найдено".
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsUnauthorized проверяет, является ли ошибка ошибкой аутентификации.
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrTokenExpired) || errors.Is(err, ErrTokenInvalid)
}

// ValidationError — ошибка валидации с деталями по полям.
type ValidationError struct {
	Field   string
	Message string
}

// ValidationErrors — список ошибок валидации.
type ValidationErrors []ValidationError

// Error реализует интерфейс error.
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "ошибка валидации"
	}
	return fmt.Sprintf("ошибка валидации поля '%s': %s", ve[0].Field, ve[0].Message)
}
