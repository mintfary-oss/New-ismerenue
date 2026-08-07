-- Запросы для аудит-лога

-- name: InsertAuditLog :exec
INSERT INTO audit_log (user_id, action, resource, resource_id, ip_address, user_agent, details)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: ListAuditLog :many
SELECT id, user_id, action, resource, resource_id, ip_address, user_agent, details, created_at
FROM audit_log
WHERE ($1::UUID IS NULL OR user_id = $1)
  AND ($2::TEXT IS NULL OR action = $2)
  AND created_at >= $3
  AND created_at <  $4
ORDER BY created_at DESC
LIMIT $5 OFFSET $6;
