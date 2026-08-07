-- Откат миграции 005: удаляем retention policies и агрегаты

-- Удаляем job очистки отчётов (номер job может отличаться — берём по имени процедуры)
SELECT delete_job(job_id)
FROM timescaledb_information.jobs
WHERE proc_name ILIKE '%DELETE FROM reports%'
   OR application_name ILIKE '%reports%';

-- Удаляем политику хранения для forecasts
SELECT remove_retention_policy('forecasts', if_not_exists => TRUE);

-- Удаляем суточный continuous aggregate
SELECT remove_continuous_aggregate_policy('measurements_1d', if_not_exists => TRUE);
DROP MATERIALIZED VIEW IF EXISTS measurements_1d;
