// Package main — точка входа AQI Platform.
// Запуск: aqi-platform server | agent | migrate
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
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
	// Контекст с отменой по сигналам ОС.
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// TODO Sprint 2: загрузка конфига, инициализация БД, запуск сервера.
	fmt.Printf("AQI Platform %s запускается...\n", version)
	fmt.Printf("Конфиг: %s\n", configPath)

	<-ctx.Done()
	fmt.Println("Остановка сервера...")
	return nil
}

// runMigrate применяет миграции к базе данных.
func runMigrate(_ context.Context, configPath, direction string) error {
	// TODO Sprint 2: загрузка конфига и запуск golang-migrate.
	fmt.Printf("Миграции (%s), конфиг: %s\n", direction, configPath)
	return nil
}
