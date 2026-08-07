-- Запросы для измерений (TimescaleDB hypertable)

-- name: InsertMeasurement :exec
INSERT INTO measurements
    (time, sensor_id, no2, o3, co, h2s, so2, pm25, temperature, humidity, pressure, wind_speed, wind_dir)
VALUES
    ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: GetLatestMeasurements :many
-- Последнее измерение для каждого активного датчика.
SELECT DISTINCT ON (m.sensor_id)
    m.time, m.sensor_id, m.no2, m.o3, m.co, m.h2s, m.so2, m.pm25,
    m.temperature, m.humidity, m.pressure, m.wind_speed, m.wind_dir
FROM measurements m
JOIN sensors s ON s.id = m.sensor_id
WHERE s.is_active = true
ORDER BY m.sensor_id, m.time DESC;

-- name: GetMeasurementsBySensor :many
SELECT time, sensor_id, no2, o3, co, h2s, so2, pm25, temperature, humidity, pressure, wind_speed, wind_dir
FROM measurements
WHERE sensor_id = $1
  AND time >= $2
  AND time <  $3
ORDER BY time ASC
LIMIT $4;

-- name: GetAggregatedMeasurements1h :many
-- Часовые агрегаты из continuous aggregate view.
SELECT
    bucket, sensor_id,
    avg_no2, avg_o3, avg_co, avg_h2s, avg_so2, avg_pm25,
    avg_temperature, avg_humidity,
    max_pm25, max_no2, data_points
FROM measurements_1h
WHERE sensor_id = $1
  AND bucket >= $2
  AND bucket <  $3
ORDER BY bucket ASC;

-- name: GetDataAvailabilityStats :one
-- Процент доступности данных за указанный период.
SELECT
    COUNT(*)::FLOAT  AS total_points,
    COUNT(pm25)::FLOAT AS pm25_points,
    COUNT(no2)::FLOAT  AS no2_points,
    (COUNT(pm25)::FLOAT / NULLIF(COUNT(*), 0)) * 100 AS pm25_coverage_pct
FROM measurements
WHERE sensor_id = $1
  AND time >= $2
  AND time <  $3;
