// Package domain содержит доменные сущности и бизнес-правила.
// Пакет не зависит ни от каких внешних библиотек — только stdlib.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Role — роль пользователя в системе.
type Role string

const (
	RoleAdmin   Role = "admin"   // полный доступ
	RoleAnalyst Role = "analyst" // чтение + загрузка данных
	RoleViewer  Role = "viewer"  // только чтение
)

// IsValid проверяет, является ли роль допустимой.
func (r Role) IsValid() bool {
	return r == RoleAdmin || r == RoleAnalyst || r == RoleViewer
}

// String реализует fmt.Stringer.
func (r Role) String() string { return string(r) }

// User — пользователь платформы.
type User struct {
	ID        uuid.UUID `db:"id"`
	Email     string    `db:"email"`
	Username  string    `db:"username"`
	Password  string    `db:"password"` // Argon2id hash
	Role      Role      `db:"role"`
	IsActive  bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// IsAdmin проверяет, является ли пользователь администратором.
func (u *User) IsAdmin() bool { return u.Role == RoleAdmin }

// CanWrite проверяет право на запись данных.
func (u *User) CanWrite() bool { return u.Role == RoleAdmin || u.Role == RoleAnalyst }

// CreateUserInput — данные для создания нового пользователя.
type CreateUserInput struct {
	Email    string `json:"email"    validate:"required,email,max=255"`
	Username string `json:"username" validate:"required,min=3,max=50,alphanum"`
	Password string `json:"password" validate:"required,min=12,max=128"`
	Role     Role   `json:"role"     validate:"required"`
}

// UpdateUserInput — данные для обновления пользователя.
type UpdateUserInput struct {
	Email    *string `json:"email"    validate:"omitempty,email,max=255"`
	Username *string `json:"username" validate:"omitempty,min=3,max=50,alphanum"`
	Role     *Role   `json:"role"     validate:"omitempty"`
	IsActive *bool   `json:"is_active"`
}

// ChangePasswordInput — данные для смены пароля.
type ChangePasswordInput struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=12,max=128"`
}

// APIToken — токен для доступа к API без пароля (используется внешними системами).
// TokenHash намеренно отсутствует — никогда не возвращается клиенту.
type APIToken struct {
	ID        uuid.UUID  `db:"id"         json:"id"`
	UserID    uuid.UUID  `db:"user_id"    json:"user_id"`
	Name      string     `db:"name"       json:"name"`
	LastUsed  *time.Time `db:"last_used"  json:"last_used"`
	ExpiresAt *time.Time `db:"expires_at" json:"expires_at"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
}

// CreateAPITokenInput — данные для создания API-токена.
type CreateAPITokenInput struct {
	Name      string     `json:"name"       validate:"required,min=2,max=100"`
	ExpiresAt *time.Time `json:"expires_at"`
}

// Feedback — обращение/предложение от пользователя.
type Feedback struct {
	ID        uuid.UUID  `db:"id"         json:"id"`
	UserID    *uuid.UUID `db:"user_id"    json:"user_id"`
	Email     *string    `db:"email"      json:"email"`
	Subject   string     `db:"subject"    json:"subject"`
	Message   string     `db:"message"    json:"message"`
	Status    string     `db:"status"     json:"status"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
}

// CreateFeedbackInput — данные для создания обращения.
type CreateFeedbackInput struct {
	Email   string `json:"email"   validate:"omitempty,email,max=255"`
	Subject string `json:"subject" validate:"required,min=3,max=200"`
	Message string `json:"message" validate:"required,min=10,max=5000"`
}
