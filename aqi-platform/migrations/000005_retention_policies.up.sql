-- Миграция 005: политики хранения данных (retention policies)
-- Требование ТЗ: сырые измерения — 60 месяцев.
-- Прогнозы и отчёты хранятся меньше — они занимают место и теряют актуальность.

-- ============================================================
-- Retention policy для forecasts: 30 дней
-- Прогнозы актуальны ненадолго; исторические данные прогнозов не нужны.
-- ============================================================
SELECT add_retention_policy(
    'forecasts',
    INTERVAL '30 days',
    if_not_exists => TRUE
);

-- ============================================================
-- Дополнительная почасовая агрегация: суточные бакеты
-- Используется для графиков за недели и месяцы (меньше данных).
-- ============================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS measurements_1d
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 day', time)    AS bucket,
    sensor_id,
    AVG(no2)                      AS avg_no2,
    AVG(o3)                       AS avg_o3,
    AVG(co)                       AS avg_co,
    AVG(h2s)                      AS avg_h2s,
    AVG(so2)                      AS avg_so2,
    AVG(pm25)                     AS avg_pm25,
    AVG(temperature)              AS avg_temperature,
    AVG(humidity)                 AS avg_humidity,
    MIN(pm25)                     AS min_pm25,
    MAX(pm25)                     AS max_pm25,
    MIN(no2)                      AS min_no2,
    MAX(no2)                      AS max_no2,
    COUNT(*)                      AS data_points
FROM measurements
GROUP BY bucket, sensor_id
WITH NO DATA;

-- Политика автообновления суточной агрегации.
SELECT add_continuous_aggregate_policy(
    'measurements_1d',
    start_offset      => INTERVAL '2 days',
    end_offset        => INTERVAL '1 day',
    schedule_interval => INTERVAL '1 day',
    if_not_exists     => TRUE
);

-- ============================================================
-- Автоматическая очистка старых отчётов старше 12 месяцев
-- Используем TimescaleDB job (background_job) для PostgreSQL-таблицы.
-- ============================================================
SELECT add_job(
    'DELETE FROM reports WHERE created_at < NOW() - INTERVAL ''12 months''',
    INTERVAL '1 day',
    if_not_exists => TRUE
);

-- ============================================================
-- Сводная информация о политиках (для отладки)
-- ============================================================
COMMENT ON TABLE measurements IS
    'Сырые измерения датчиков. Retention: 60 месяцев (ТЗ).';
COMMENT ON TABLE forecasts IS
    'Прогнозы качества воздуха. Retention: 30 дней.';
