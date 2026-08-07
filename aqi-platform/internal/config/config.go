// Package config содержит структуры конфигурации приложения.
// Значения загружаются из переменных окружения и файла config.yaml.
package config

import "time"

// Config — корневая конфигурация приложения.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Email    EmailConfig    `mapstructure:"email"`
	Forecast ForecastConfig `mapstructure:"forecast"`
	Log      LogConfig      `mapstructure:"log"`
}

// ServerConfig — параметры HTTP-сервера.
type ServerConfig struct {
	// Port — порт, на котором слушает сервер (default: 8080).
	Port int `mapstructure:"port" validate:"required,min=1,max=65535"`

	// Host — адрес привязки (default: 0.0.0.0).
	Host string `mapstructure:"host"`

	// ReadTimeout — таймаут чтения запроса.
	ReadTimeout time.Duration `mapstructure:"read_timeout"`

	// WriteTimeout — таймаут записи ответа.
	WriteTimeout time.Duration `mapstructure:"write_timeout"`

	// IdleTimeout — таймаут idle соединений.
	IdleTimeout time.Duration `mapstructure:"idle_timeout"`

	// TLSCertFile — путь к TLS сертификату (опционально).
	TLSCertFile string `mapstructure:"tls_cert_file"`

	// TLSKeyFile — путь к TLS ключу (опционально).
	TLSKeyFile string `mapstructure:"tls_key_file"`

	// BaseURL — внешний URL платформы (нужен для email-ссылок).
	BaseURL string `mapstructure:"base_url"`

	// WidgetAllowOrigins — список доменов, которым разрешено встраивать виджет.
	// Пустой список = разрешено всем (публичный виджет).
	WidgetAllowOrigins []string `mapstructure:"widget_allow_origins"`
}

// DatabaseConfig — параметры подключения к PostgreSQL + TimescaleDB.
type DatabaseConfig struct {
	Host     string `mapstructure:"host"     validate:"required"`
	Port     int    `mapstructure:"port"     validate:"required"`
	Name     string `mapstructure:"name"     validate:"required"`
	User     string `mapstructure:"user"     validate:"required"`
	Password string `mapstructure:"password" validate:"required"`

	// SSLMode — режим SSL: disable | require | verify-full.
	SSLMode string `mapstructure:"ssl_mode"`

	// MaxOpenConns — максимальное число открытых соединений.
	MaxOpenConns int `mapstructure:"max_open_conns"`

	// MaxIdleConns — максимальное число idle соединений.
	MaxIdleConns int `mapstructure:"max_idle_conns"`

	// ConnMaxLifetime — максимальное время жизни соединения.
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`

	// MigrationsPath — путь к директории с SQL-миграциями.
	MigrationsPath string `mapstructure:"migrations_path"`
}

// DSN возвращает строку подключения PostgreSQL.
func (d DatabaseConfig) DSN() string {
	return "host=" + d.Host +
		" port=" + itoa(d.Port) +
		" dbname=" + d.Name +
		" user=" + d.User +
		" password=" + d.Password +
		" sslmode=" + d.SSLMode
}

// RedisConfig — параметры подключения к Redis.
type RedisConfig struct {
	Addr     string `mapstructure:"addr"     validate:"required"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`

	// TokenBlacklistTTL — TTL записи в блеклисте JWT токенов.
	TokenBlacklistTTL time.Duration `mapstructure:"token_blacklist_ttl"`
}

// AuthConfig — параметры аутентификации.
type AuthConfig struct {
	// JWTSecret — секрет для подписи JWT токенов (минимум 32 байта).
	JWTSecret string `mapstructure:"jwt_secret" validate:"required,min=32"`

	// AccessTokenTTL — время жизни access token (default: 15m).
	AccessTokenTTL time.Duration `mapstructure:"access_token_ttl"`

	// RefreshTokenTTL — время жизни refresh token (default: 720h = 30 дней).
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`

	// PasswordResetTTL — время жизни токена сброса пароля (default: 1h).
	PasswordResetTTL time.Duration `mapstructure:"password_reset_ttl"`

	// MaxLoginAttempts — максимум попыток входа до блокировки.
	MaxLoginAttempts int `mapstructure:"max_login_attempts"`

	// LockoutDuration — длительность блокировки после превышения попыток.
	LockoutDuration time.Duration `mapstructure:"lockout_duration"`

	// Argon2 параметры — настройки хеширования паролей.
	Argon2Time    uint32 `mapstructure:"argon2_time"`
	Argon2Memory  uint32 `mapstructure:"argon2_memory"`
	Argon2Threads uint8  `mapstructure:"argon2_threads"`
	Argon2KeyLen  uint32 `mapstructure:"argon2_key_len"`
}

// EmailConfig — параметры email-шлюза.
type EmailConfig struct {
	// IMAPHost — хост IMAP-сервера для приёма данных.
	IMAPHost string `mapstructure:"imap_host"`
	IMAPPort int    `mapstructure:"imap_port"`

	// IMAPUser — логин (email Ecology@kemerovo.ru).
	IMAPUser string `mapstructure:"imap_user"`

	// IMAPPassword — пароль IMAP.
	IMAPPassword string `mapstructure:"imap_password"`

	// PollInterval — интервал опроса почтового ящика.
	PollInterval time.Duration `mapstructure:"poll_interval"`

	// SMTPHost — хост для отправки уведомлений (опционально).
	SMTPHost string `mapstructure:"smtp_host"`
	SMTPPort int    `mapstructure:"smtp_port"`
	SMTPUser string `mapstructure:"smtp_user"`
	SMTPPass string `mapstructure:"smtp_pass"`
	FromAddr string `mapstructure:"from_addr"`
}

// ForecastConfig — параметры прогнозного движка.
type ForecastConfig struct {
	// UpdateInterval — как часто пересчитывать прогноз (default: 20m).
	UpdateInterval time.Duration `mapstructure:"update_interval"`

	// HorizonHours — горизонт прогноза в часах (default: 6, мин по ТЗ).
	HorizonHours int `mapstructure:"horizon_hours"`

	// EWMAAlpha — коэффициент сглаживания EWMA (0 < alpha < 1).
	EWMAAlpha float64 `mapstructure:"ewma_alpha"`

	// IDWPower — степень обратных расстояний для IDW (default: 2).
	IDWPower float64 `mapstructure:"idw_power"`

	// MinSensorsForForecast — минимальное число активных датчиков.
	MinSensorsForForecast int `mapstructure:"min_sensors_for_forecast"`
}

// LogConfig — параметры логирования.
type LogConfig struct {
	// Level — уровень: debug | info | warn | error.
	Level string `mapstructure:"level"`

	// Format — формат: json | text.
	Format string `mapstructure:"format"`
}

// itoa — минимальный int → string без fmt.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		buf = append(buf, byte('0'+n%10))
		n /= 10
	}
	if neg {
		buf = append(buf, '-')
	}
	// reverse
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
