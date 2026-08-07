-- Запросы для работы с датчиками

-- name: GetSensorByID :one
SELECT id, name, address, lat, lng, type, is_active, last_seen, created_at
FROM sensors
WHERE id = $1
LIMIT 1;

-- name: ListSensors :many
SELECT id, name, address, lat, lng, type, is_active, last_seen, created_at
FROM sensors
ORDER BY name ASC;

-- name: ListActiveSensors :many
SELECT id, name, address, lat, lng, type, is_active, last_seen, created_at
FROM sensors
WHERE is_active = true
ORDER BY name ASC;

-- name: CreateSensor :one
INSERT INTO sensors (name, address, lat, lng, type)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, address, lat, lng, type, is_active, last_seen, created_at;

-- name: UpdateSensor :one
UPDATE sensors
SET
    name      = COALESCE($2, name),
    address   = COALESCE($3, address),
    lat       = COALESCE($4, lat),
    lng       = COALESCE($5, lng),
    type      = COALESCE($6, type),
    is_active = COALESCE($7, is_active)
WHERE id = $1
RETURNING id, name, address, lat, lng, type, is_active, last_seen, created_at;

-- name: UpdateSensorLastSeen :exec
UPDATE sensors SET last_seen = NOW() WHERE id = $1;

-- name: DeleteSensor :exec
DELETE FROM sensors WHERE id = $1;
