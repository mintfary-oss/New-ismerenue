-- Миграция 002: TimescaleDB hypertables + continuous aggregates
-- Требует расширения timescaledb (доступно в образе timescale/timescaledb-ha)

CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

-- Конвертация measurements в hypertable (партиция по времени, интервал 1 день).
SELECT create_hypertable(
    'measurements',
    'time',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

-- Индексы для частых запросов.
CREATE INDEX IF NOT EXISTS measurements_sensor_time_idx
    ON measurements (sensor_id, time DESC);

-- Конвертация forecasts в hypertable.
SELECT create_hypertable(
    'forecasts',
    'time',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

CREATE INDEX IF NOT EXISTS forecasts_point_time_idx
    ON forecasts (point_id, time DESC);

-- ============================================================
-- Continuous Aggregate: почасовая агрегация измерений
-- Используется для быстрых запросов графиков за длинные периоды
-- ============================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS measurements_1h
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', time)   AS bucket,
    sensor_id,
    AVG(no2)                      AS avg_no2,
    AVG(o3)                       AS avg_o3,
    AVG(co)                       AS avg_co,
    AVG(h2s)                      AS avg_h2s,
    AVG(so2)                      AS avg_so2,
    AVG(pm25)                     AS avg_pm25,
    AVG(temperature)              AS avg_temperature,
    AVG(humidity)                 AS avg_humidity,
    MAX(pm25)                     AS max_pm25,
    MAX(no2)                      AS max_no2,
    COUNT(*)                      AS data_points
FROM measurements
GROUP BY bucket, sensor_id
WITH NO DATA;

-- Политика автообновления агрегации.
SELECT add_continuous_aggregate_policy(
    'measurements_1h',
    start_offset  => INTERVAL '3 hours',
    end_offset    => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists => TRUE
);

-- ============================================================
-- Retention policy: хранить сырые данные 60 месяцев (требование ТЗ)
-- ============================================================
SELECT add_retention_policy(
    'measurements',
    INTERVAL '60 months',
    if_not_exists => TRUE
);

-- Агрегированные данные хранить бессрочно (занимают мало места).
