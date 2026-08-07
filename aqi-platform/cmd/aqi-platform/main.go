// Package main — точка входа AQI Platform.
// Запуск: aqi-platform server | migrate | version
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/mintfary/aqi-platform/internal/config"
	"github.com/mintfary/aqi-platform/internal/email"
	"github.com/mintfary/aqi-platform/internal/handler"
	"github.com/mintfary/aqi-platform/internal/repository"
	"github.com/mintfary/aqi-platform/internal/scheduler"
	"github.com/mintfary/aqi-platform/internal/server"
	"github.com/mintfary/aqi-platform/internal/service"
)

// Значения проставляются при сборке через -ldflags.
var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	root := buildRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// buildRootCmd создаёт дерево cobra-команд.
func buildRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "aqi-platform",
		Short: "Платформа прогнозирования качества атмосферного воздуха",
		Long: `AQI Platform — self-hosted система мониторинга и прогнозирования
качества атмосферного воздуха. Разворачивается одной командой через Docker.`,
	}

	root.AddCommand(
		buildServerCmd(),
		buildMigrateCmd(),
		buildVersionCmd(),
	)

	return root
}

// buildServerCmd — команда запуска HTTP-сервера.
func buildServerCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Запустить HTTP-сервер платформы",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runServer(cmd.Context(), configPath)
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "config.yaml", "путь к файлу конфигурации")
	return cmd
}

// buildMigrateCmd — команда применения миграций БД.
func buildMigrateCmd() *cobra.Command {
	var configPath string
	var direction string

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Применить миграции базы данных",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMigrate(cmd.Context(), configPath, direction)
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "config.yaml", "путь к файлу конфигурации")
	cmd.Flags().StringVarP(&direction, "direction", "d", "up", "направление миграции: up | down")
	return cmd
}

// buildVersionCmd — команда вывода версии.
func buildVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Вывести версию приложения",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("aqi-platform %s (собран: %s)\n", version, buildTime)
		},
	}
}

// runServer инициализирует и запускает HTTP-сервер с graceful shutdown.
func runServer(ctx context.Context, configPath string) error {
	// ── 1. Конфигурация ───────────────────────────────────────────────────
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("загрузка конфигурации: %w", err)
	}

	// ── 2. Логирование ────────────────────────────────────────────────────
	logger := buildLogger(cfg.Log)

	logger.Info("AQI Platform запускается",
		"version", version,
		"build_time", buildTime,
		"config", configPath,
	)

	// ── 3. Контекст с отменой по сигналам ОС ─────────────────────────────
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ── 4. База данных ────────────────────────────────────────────────────
	db, err := repository.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("подключение к PostgreSQL: %w", err)
	}
	defer db.Close()
	logger.Info("PostgreSQL подключён")

	// ── 5. Redis ──────────────────────────────────────────────────────────
	redisClient, err := repository.NewRedisClient(ctx, cfg.Redis)
	if err != nil {
		return fmt.Errorf("подключение к Redis: %w", err)
	}
	defer redisClient.Close()
	logger.Info("Redis подключён")

	// ── 6. Репозитории ────────────────────────────────────────────────────
	userRepo := repository.NewUserRepo(db)
	sensorRepo := repository.NewSensorRepo(db)
	measurementRepo := repository.NewMeasurementRepo(db)
	forecastRepo := repository.NewForecastRepo(db)
	tokenRepo := repository.NewTokenRepo(db)
	feedbackRepo := repository.NewFeedbackRepo(db)
	statsRepo := repository.NewStatsRepo(db)
	reportRepo := repository.NewReportRepo(db)

	// Redis-хранилища.
	tokenBlacklist := repository.NewTokenBlacklist(redisClient, cfg.Redis.TokenBlacklistTTL)
	loginAttempts := repository.NewLoginAttemptTracker(
		redisClient,
		cfg.Auth.MaxLoginAttempts,
		cfg.Auth.LockoutDuration,
	)

	// ── 7. Сервисы ────────────────────────────────────────────────────────
	authSvc := service.NewAuthService(userRepo, tokenBlacklist, loginAttempts, cfg.Auth, logger)
	userSvc := service.NewUserService(userRepo, authSvc, logger)
	sensorSvc := service.NewSensorService(sensorRepo, logger)
	measureSvc := service.NewMeasurementService(measurementRepo, sensorRepo, logger)
	forecastSvc := service.NewForecastService(measurementRepo, forecastRepo, cfg.Forecast, logger)
	tokenSvc := service.NewTokenService(tokenRepo, userRepo, cfg.Auth.JWTSecret, logger)

	// ── 8. Планировщик фоновых задач ─────────────────────────────────────
	sched := scheduler.New(forecastSvc, measurementRepo, forecastRepo, cfg.Forecast, logger)
	go sched.Start(ctx)

	// ── 8a. IMAP-приёмник данных с датчиков ───────────────────────────────
	// Опрашивает почтовый ящик и загружает CSV-вложения с измерениями.
	imapReceiver := email.New(cfg.Email, measureSvc, logger)
	go imapReceiver.Start(ctx)

	// ── 9. HTTP-обработчики ───────────────────────────────────────────────
	handlers := handler.NewHandlers(handler.Deps{
		DB:          db,
		Redis:       redisClient,
		Logger:      logger,
		AuthSvc:     authSvc,
		UserSvc:     userSvc,
		SensorSvc:   sensorSvc,
		MeasureSvc:  measureSvc,
		ForecastSvc: forecastSvc,
		TokenSvc:    tokenSvc,
		FeedbackRepo: feedbackRepo,
		StatsRepo:    statsRepo,
		ReportRepo:   reportRepo,
	})

	// ── 10. Роутер и HTTP-сервер ──────────────────────────────────────────
	router := server.NewRouter(handlers, authSvc)
	srv := server.New(cfg.Server, router, logger)

	logger.Info("HTTP-сервер запущен", "addr", fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port))

	// ── 11. Запуск (блокируется до отмены контекста) ─────────────────────
	return srv.Start(ctx)
}

// runMigrate применяет миграции к базе данных.
func runMigrate(_ context.Context, configPath, direction string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("загрузка конфигурации: %w", err)
	}

	// golang-migrate требует URL формата pgx5://user:pass@host:port/db
	databaseURL := fmt.Sprintf("pgx5://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	// Путь к миграциям в формате file:///abs/path
	migrationsPath := cfg.Database.MigrationsPath
	if !strings.HasPrefix(migrationsPath, "file://") {
		absPath, absErr := filepath.Abs(migrationsPath)
		if absErr != nil {
			absPath = migrationsPath
		}
		migrationsPath = "file://" + absPath
	}

	fmt.Printf("Применение миграций (%s): %s\n", direction, migrationsPath)

	if err := repository.RunMigrations(databaseURL, migrationsPath, direction); err != nil {
		return fmt.Errorf("миграции: %w", err)
	}

	fmt.Println("Миграции применены успешно")
	return nil
}

// buildLogger создаёт slog.Logger с заданными параметрами.
func buildLogger(cfg config.LogConfig) *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler
	if cfg.Format == "text" {
		h = slog.NewTextHandler(os.Stdout, opts)
	} else {
		h = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(h)
}
