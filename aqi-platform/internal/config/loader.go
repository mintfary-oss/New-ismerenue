// Package config — загрузчик конфигурации через Viper.
// Приоритет: переменные окружения > config.yaml > значения по умолчанию.
package config

import (
	"fmt"
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

	// Чтение файла (не обязательно — ENV достаточно).
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Файл найден, но содержит ошибку.
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
