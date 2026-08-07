// Package repository содержит реализацию доступа к данным.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // pgx v5 driver
	_ "github.com/golang-migrate/migrate/v4/source/file"     // file:// source
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mintfary/aqi-platform/internal/config"
)

// NewPostgresPool создаёт пул соединений к PostgreSQL через pgx/v5.
// Пул потокобезопасен и используется всеми репозиториями.
func NewPostgresPool(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("разбор DSN: %w", err)
	}

	// Настройка пула соединений.
	poolCfg.MaxConns = int32(cfg.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.MaxIdleConns)
	poolCfg.MaxConnLifetime = cfg.ConnMaxLifetime
	poolCfg.MaxConnIdleTime = 5 * time.Minute
	poolCfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("создание пула: %w", err)
	}

	// Проверка соединения при старте.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err = pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}

	return pool, nil
}

// RunMigrations применяет SQL-миграции из указанной директории через golang-migrate.
// migrationsPath — путь в формате file:///abs/path/to/migrations
// direction — "up" или "down"
func RunMigrations(databaseURL, migrationsPath, direction string) error {
	// golang-migrate требует pgx5 URL вида pgx5://user:pass@host/db
	m, err := migrate.New(migrationsPath, databaseURL)
	if err != nil {
		return fmt.Errorf("инициализация миграций: %w", err)
	}
	defer m.Close()

	switch direction {
	case "up":
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("применение миграций (up): %w", err)
		}
	case "down":
		if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("откат миграций (down): %w", err)
		}
	default:
		return fmt.Errorf("неизвестное направление: %s (ожидается up | down)", direction)
	}

	return nil
}
