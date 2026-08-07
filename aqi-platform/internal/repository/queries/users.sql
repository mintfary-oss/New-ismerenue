-- Запросы для работы с пользователями
-- Файл читается sqlc для генерации типобезопасного Go-кода

-- name: GetUserByID :one
SELECT id, email, username, password, role, is_active, created_at, updated_at
FROM users
WHERE id = $1 AND is_active = true
LIMIT 1;

-- name: GetUserByEmail :one
SELECT id, email, username, password, role, is_active, created_at, updated_at
FROM users
WHERE LOWER(email) = LOWER($1)
LIMIT 1;

-- name: ListUsers :many
SELECT id, email, username, role, is_active, created_at, updated_at
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CreateUser :one
INSERT INTO users (email, username, password, role)
VALUES ($1, $2, $3, $4)
RETURNING id, email, username, role, is_active, created_at, updated_at;

-- name: UpdateUser :one
UPDATE users
SET
    email      = COALESCE($2, email),
    username   = COALESCE($3, username),
    role       = COALESCE($4, role),
    is_active  = COALESCE($5, is_active)
WHERE id = $1
RETURNING id, email, username, role, is_active, created_at, updated_at;

-- name: UpdateUserPassword :exec
UPDATE users SET password = $2 WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: DeactivateUser :exec
UPDATE users SET is_active = false WHERE id = $1;
