-- Миграция 004: Таблица сгенерированных отчётов
-- CSV-содержимое хранится в БД как TEXT для простоты (подходит для отчётов до ~10 МБ).

CREATE TABLE IF NOT EXISTS reports (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        REFERENCES users (id) ON DELETE SET NULL,
    name         TEXT        NOT NULL,
    report_type  TEXT        NOT NULL
                             CHECK (report_type IN ('measurements', 'forecasts', 'availability')),
    params       JSONB       NOT NULL DEFAULT '{}',
    status       TEXT        NOT NULL DEFAULT 'pending'
                             CHECK (status IN ('pending', 'ready', 'error')),
    row_count    INTEGER,
    file_data    TEXT,        -- CSV-содержимое (NULL пока не готово)
    error_msg    TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS reports_user_idx    ON reports (user_id);
CREATE INDEX IF NOT EXISTS reports_status_idx  ON reports (status);
CREATE INDEX IF NOT EXISTS reports_created_idx ON reports (created_at DESC);
