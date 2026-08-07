-- Миграция 001: Начальная схема базы данных
-- AQI Platform — платформа прогнозирования качества атмосферного воздуха
-- Применяется автоматически при запуске: aqi-platform migrate

-- Расширение для UUID.
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- ПОЛЬЗОВАТЕЛИ
-- ============================================================
CREATE TABLE IF NOT EXISTS users (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT        NOT NULL,
    username   TEXT        NOT NULL,
    password   TEXT        NOT NULL,       -- Argon2id хеш
    role       TEXT        NOT NULL CHECK (role IN ('admin', 'analyst', 'viewer')),
    is_active  BOOLEAN     NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_idx    ON users (LOWER(email));
CREATE UNIQUE INDEX IF NOT EXISTS users_username_idx ON users (LOWER(username));
CREATE INDEX        IF NOT EXISTS users_role_idx     ON users (role);

-- ============================================================
-- ДАТЧИКИ
-- ============================================================
CREATE TABLE IF NOT EXISTS sensors (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL,
    address    TEXT        NOT NULL,
    lat        DOUBLE PRECISION NOT NULL,
    lng        DOUBLE PRECISION NOT NULL,
    type       TEXT        NOT NULL CHECK (type IN ('gas', 'dust', 'combo')),
    is_active  BOOLEAN     NOT NULL DEFAULT true,
    last_seen  TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS sensors_is_active_idx ON sensors (is_active);
CREATE INDEX IF NOT EXISTS sensors_type_idx      ON sensors (type);

-- ============================================================
-- ИЗМЕРЕНИЯ (будет конвертировано в hypertable в 002)
-- ============================================================
CREATE TABLE IF NOT EXISTS measurements (
    time         TIMESTAMPTZ      NOT NULL,
    sensor_id    UUID             NOT NULL REFERENCES sensors (id) ON DELETE CASCADE,
    no2          DOUBLE PRECISION,           -- мг/м³
    o3           DOUBLE PRECISION,           -- мг/м³
    co           DOUBLE PRECISION,           -- мг/м³
    h2s          DOUBLE PRECISION,           -- мг/м³
    so2          DOUBLE PRECISION,           -- мг/м³
    pm25         DOUBLE PRECISION,           -- мг/м³
    temperature  DOUBLE PRECISION,           -- °C
    humidity     DOUBLE PRECISION,           -- %
    pressure     DOUBLE PRECISION,           -- гПа
    wind_speed   DOUBLE PRECISION,           -- м/с
    wind_dir     DOUBLE PRECISION            -- градусы
);

-- ============================================================
-- ПРОГНОЗЫ (будет конвертировано в hypertable в 002)
-- ============================================================
CREATE TABLE IF NOT EXISTS forecasts (
    time           TIMESTAMPTZ      NOT NULL,
    point_id       TEXT             NOT NULL,
    lat            DOUBLE PRECISION NOT NULL,
    lng            DOUBLE PRECISION NOT NULL,
    horizon_hours  INTEGER          NOT NULL,
    aqi            INTEGER          NOT NULL,
    aqi_category   TEXT             NOT NULL,
    no2_forecast   DOUBLE PRECISION,
    pm25_forecast  DOUBLE PRECISION,
    so2_forecast   DOUBLE PRECISION,
    model_version  TEXT             NOT NULL DEFAULT 'v1',
    created_at     TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

-- ============================================================
-- API ТОКЕНЫ
-- ============================================================
CREATE TABLE IF NOT EXISTS api_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name       TEXT        NOT NULL,
    token_hash TEXT        NOT NULL,          -- HMAC-SHA256
    last_used  TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS api_tokens_hash_idx    ON api_tokens (token_hash);
CREATE        INDEX IF NOT EXISTS api_tokens_user_idx    ON api_tokens (user_id);

-- ============================================================
-- АУДИТ ЛОГ
-- ============================================================
CREATE TABLE IF NOT EXISTS audit_log (
    id          BIGSERIAL   PRIMARY KEY,
    user_id     UUID        REFERENCES users (id) ON DELETE SET NULL,
    action      TEXT        NOT NULL,         -- login, logout, create_user, ...
    resource    TEXT        NOT NULL,         -- users, sensors, ...
    resource_id TEXT,
    ip_address  INET,
    user_agent  TEXT,
    details     JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS audit_log_user_idx    ON audit_log (user_id);
CREATE INDEX IF NOT EXISTS audit_log_action_idx  ON audit_log (action);
CREATE INDEX IF NOT EXISTS audit_log_created_idx ON audit_log (created_at DESC);

-- ============================================================
-- ЗАГРУЖЕННЫЕ ДАННЫЕ (email / файл)
-- ============================================================
CREATE TABLE IF NOT EXISTS uploaded_data (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        REFERENCES users (id) ON DELETE SET NULL,
    filename    TEXT        NOT NULL,
    source      TEXT        NOT NULL CHECK (source IN ('email', 'api', 'manual')),
    status      TEXT        NOT NULL CHECK (status IN ('pending', 'valid', 'invalid', 'processed'))
                            DEFAULT 'pending',
    rows_total  INTEGER     NOT NULL DEFAULT 0,
    rows_valid  INTEGER     NOT NULL DEFAULT 0,
    error_msg   TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS uploaded_data_status_idx ON uploaded_data (status);

-- ============================================================
-- ОБРАТНАЯ СВЯЗЬ
-- ============================================================
CREATE TABLE IF NOT EXISTS feedback (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        REFERENCES users (id) ON DELETE SET NULL,
    email      TEXT,
    subject    TEXT        NOT NULL,
    message    TEXT        NOT NULL,
    status     TEXT        NOT NULL DEFAULT 'new'
                           CHECK (status IN ('new', 'in_progress', 'resolved')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- ТРИГГЕР: автообновление updated_at у users
-- ============================================================
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
