// Package config — загрузчик конфигурации через Viper.
// Приоритет: переменные окружения > config.yaml > значения по умолчанию.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Load читает конфигурацию из файла и переменных окружения.
// Переменные окружения имеют формат AQI_SECTION_KEY (например AQI_SERVER_PORT).
func Load(path string) (*Config, error) {
	v := viper.New()

	// Настройка источников.
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.SetEnvPrefix("AQI")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Значения по умолчанию.
	setDefaults(v)

	// Явная привязка переменных окружения.
	// AutomaticEnv() не работает с Unmarshal для вложенных ключей — BindEnv
	// обязателен чтобы переменные окружения попали в структуру Config.
	bindEnvs(v)

	// Чтение файла (не обязательно — ENV достаточно).
	// При использовании SetConfigFile() Viper возвращает *os.PathError (не
	// ConfigFileNotFoundError) когда файл отсутствует — проверяем оба случая.
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !errors.Is(err, fs.ErrNotExist) {
			// Файл найден, но содержит ошибку синтаксиса.
			return nil, fmt.Errorf("чтение конфигурации: %w", err)
		}
		// Файл не найден — продолжаем с ENV и дефолтами.
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("разбор конфигурации: %w", err)
	}

	// Простая валидация критичных полей.
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("конфигурация невалидна: %w", err)
	}

	return &cfg, nil
}

// setDefaults устанавливает безопасные значения по умолчанию.
func setDefaults(v *viper.Viper) {
	// Server
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", 15*time.Second)
	v.SetDefault("server.write_timeout", 15*time.Second)
	v.SetDefault("server.idle_timeout", 60*time.Second)
	v.SetDefault("server.base_url", "http://localhost:8080")

	// Database
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.name", "aqi")
	v.SetDefault("database.user", "aqi")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", 5*time.Minute)
	v.SetDefault("database.migrations_path", "migrations")

	// Redis
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.token_blacklist_ttl", 24*time.Hour)

	// Auth
	v.SetDefault("auth.access_token_ttl", 15*time.Minute)
	v.SetDefault("auth.refresh_token_ttl", 720*time.Hour) // 30 дней
	v.SetDefault("auth.password_reset_ttl", time.Hour)
	v.SetDefault("auth.max_login_attempts", 5)
	v.SetDefault("auth.lockout_duration", 15*time.Minute)
	// Argon2id — параметры по OWASP 2025
	v.SetDefault("auth.argon2_time", 3)
	v.SetDefault("auth.argon2_memory", 65536) // 64 MB
	v.SetDefault("auth.argon2_threads", 4)
	v.SetDefault("auth.argon2_key_len", 32)

	// Email
	v.SetDefault("email.imap_port", 993)
	v.SetDefault("email.poll_interval", 5*time.Minute)
	v.SetDefault("email.smtp_port", 587)

	// Forecast
	v.SetDefault("forecast.update_interval", 20*time.Minute)
	v.SetDefault("forecast.horizon_hours", 6)
	v.SetDefault("forecast.ewma_alpha", 0.3)
	v.SetDefault("forecast.idw_power", 2.0)
	v.SetDefault("forecast.min_sensors_for_forecast", 1)

	// Alert
	v.SetDefault("alert.enabled", false)
	v.SetDefault("alert.threshold", 101)               // Unhealthy for Sensitive Groups
	v.SetDefault("alert.cooldown_duration", 4*time.Hour)
	v.SetDefault("alert.check_interval", 20*time.Minute)

	// Log
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
}

// validate проверяет критичные поля конфигурации.
func validate(cfg *Config) error {
	if cfg.Auth.JWTSecret == "" {
		return fmt.Errorf("auth.jwt_secret (AQI_AUTH_JWT_SECRET) обязателен")
	}
	if len(cfg.Auth.JWTSecret) < 32 {
		return fmt.Errorf("auth.jwt_secret должен быть минимум 32 символа")
	}
	if cfg.Database.Password == "" {
		return fmt.Errorf("database.password (AQI_DATABASE_PASSWORD) обязателен")
	}
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("server.port должен быть в диапазоне 1-65535")
	}
	return nil
}

// bindEnvs явно привязывает переменные окружения к ключам Viper.
// Это необходимо потому что AutomaticEnv() не работает с v.Unmarshal()
// для вложенных ключей: без BindEnv значения из ENV не попадают в структуру.
func bindEnvs(v *viper.Viper) {
	pairs := [][2]string{
		// Server
		{"server.host", "AQI_SERVER_HOST"},
		{"server.port", "AQI_SERVER_PORT"},
		{"server.base_url", "AQI_SERVER_BASE_URL"},
		{"server.read_timeout", "AQI_SERVER_READ_TIMEOUT"},
		{"server.write_timeout", "AQI_SERVER_WRITE_TIMEOUT"},
		{"server.idle_timeout", "AQI_SERVER_IDLE_TIMEOUT"},
		// Database
		{"database.host", "AQI_DATABASE_HOST"},
		{"database.port", "AQI_DATABASE_PORT"},
		{"database.name", "AQI_DATABASE_NAME"},
		{"database.user", "AQI_DATABASE_USER"},
		{"database.password", "AQI_DATABASE_PASSWORD"},
		{"database.ssl_mode", "AQI_DATABASE_SSL_MODE"},
		{"database.max_open_conns", "AQI_DATABASE_MAX_OPEN_CONNS"},
		{"database.max_idle_conns", "AQI_DATABASE_MAX_IDLE_CONNS"},
		{"database.conn_max_lifetime", "AQI_DATABASE_CONN_MAX_LIFETIME"},
		{"database.migrations_path", "AQI_DATABASE_MIGRATIONS_PATH"},
		// Redis
		{"redis.addr", "AQI_REDIS_ADDR"},
		{"redis.password", "AQI_REDIS_PASSWORD"},
		{"redis.db", "AQI_REDIS_DB"},
		{"redis.token_blacklist_ttl", "AQI_REDIS_TOKEN_BLACKLIST_TTL"},
		// Auth
		{"auth.jwt_secret", "AQI_AUTH_JWT_SECRET"},
		{"auth.access_token_ttl", "AQI_AUTH_ACCESS_TOKEN_TTL"},
		{"auth.refresh_token_ttl", "AQI_AUTH_REFRESH_TOKEN_TTL"},
		{"auth.password_reset_ttl", "AQI_AUTH_PASSWORD_RESET_TTL"},
		{"auth.max_login_attempts", "AQI_AUTH_MAX_LOGIN_ATTEMPTS"},
		{"auth.lockout_duration", "AQI_AUTH_LOCKOUT_DURATION"},
		// Email
		{"email.imap_host", "AQI_EMAIL_IMAP_HOST"},
		{"email.imap_port", "AQI_EMAIL_IMAP_PORT"},
		{"email.imap_user", "AQI_EMAIL_IMAP_USER"},
		{"email.imap_password", "AQI_EMAIL_IMAP_PASSWORD"},
		{"email.poll_interval", "AQI_EMAIL_POLL_INTERVAL"},
		{"email.smtp_host", "AQI_EMAIL_SMTP_HOST"},
		{"email.smtp_port", "AQI_EMAIL_SMTP_PORT"},
		{"email.smtp_user", "AQI_EMAIL_SMTP_USER"},
		{"email.smtp_pass", "AQI_EMAIL_SMTP_PASS"},
		{"email.from_addr", "AQI_EMAIL_FROM_ADDR"},
		// Alert
		{"alert.enabled", "AQI_ALERT_ENABLED"},
		{"alert.threshold", "AQI_ALERT_THRESHOLD"},
		{"alert.cooldown_duration", "AQI_ALERT_COOLDOWN_DURATION"},
		{"alert.check_interval", "AQI_ALERT_CHECK_INTERVAL"},
		// Log
		{"log.level", "AQI_LOG_LEVEL"},
		{"log.format", "AQI_LOG_FORMAT"},
	}
	for _, p := range pairs {
		_ = v.BindEnv(p[0], p[1])
	}
}
