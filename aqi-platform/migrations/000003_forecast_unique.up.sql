-- Миграция 003: уникальный индекс на forecasts для upsert
-- Нужен для ON CONFLICT (time, point_id, horizon_hours) в ForecastRepo.InsertBatch

CREATE UNIQUE INDEX IF NOT EXISTS forecasts_time_point_horizon_idx
    ON forecasts (time, point_id, horizon_hours);
