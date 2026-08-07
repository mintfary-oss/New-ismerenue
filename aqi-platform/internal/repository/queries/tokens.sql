-- Запросы для API-токенов

-- name: CreateAPIToken :one
INSERT INTO api_tokens (user_id, name, token_hash, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, name, token_hash, last_used, expires_at, created_at;

-- name: GetAPITokenByHash :one
SELECT id, user_id, name, token_hash, last_used, expires_at, created_at
FROM api_tokens
WHERE token_hash = $1
  AND (expires_at IS NULL OR expires_at > NOW())
LIMIT 1;

-- name: ListAPITokensByUser :many
SELECT id, user_id, name, last_used, expires_at, created_at
FROM api_tokens
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: UpdateAPITokenLastUsed :exec
UPDATE api_tokens SET last_used = NOW() WHERE id = $1;

-- name: DeleteAPIToken :exec
DELETE FROM api_tokens WHERE id = $1 AND user_id = $2;

-- name: DeleteExpiredAPITokens :exec
DELETE FROM api_tokens WHERE expires_at IS NOT NULL AND expires_at < NOW();
