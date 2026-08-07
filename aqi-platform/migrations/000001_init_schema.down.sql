-- Откат миграции 001
DROP TRIGGER  IF EXISTS users_updated_at   ON users;
DROP FUNCTION IF EXISTS update_updated_at;
DROP TABLE    IF EXISTS feedback;
DROP TABLE    IF EXISTS uploaded_data;
DROP TABLE    IF EXISTS audit_log;
DROP TABLE    IF EXISTS api_tokens;
DROP TABLE    IF EXISTS forecasts;
DROP TABLE    IF EXISTS measurements;
DROP TABLE    IF EXISTS sensors;
DROP TABLE    IF EXISTS users;
