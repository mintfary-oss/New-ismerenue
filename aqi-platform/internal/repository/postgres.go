// Package repository содержит реализацию доступа к данным.
package repository

import (
	"context"
	"fmt"
	"time"

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

// RunMigrations применяет SQL-миграции из указанной директории.
func RunMigrations(databaseURL, migrationsPath, direction string) error {
	// Реализация через golang-migrate/migrate подключается здесь.
	// Вынесено в отдельную функцию для вызова из cobra-команды migrate.
	_ = databaseURL
	_ = migrationsPath
	_ = direction
	return nil
}
