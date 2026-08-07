-- Откат миграции 002: TimescaleDB
SELECT remove_retention_policy('measurements', if_not_exists => TRUE);
SELECT remove_continuous_aggregate_policy('measurements_1h', if_not_exists => TRUE);
DROP MATERIALIZED VIEW IF EXISTS measurements_1h;
