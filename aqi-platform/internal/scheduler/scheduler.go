// Package scheduler содержит фоновые задачи платформы.
// Все задачи запускаются как горутины и корректно завершаются при отмене контекста.
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/mintfary/aqi-platform/internal/config"
	"github.com/mintfary/aqi-platform/internal/metrics"
)

// ForecastRunner — интерфейс для запуска расчёта прогноза.
type ForecastRunner interface {
	Run(ctx context.Context) error
}

// RetentionCleaner — интерфейс удаления устаревших данных.
type RetentionCleaner interface {
	// DeleteOlderThan удаляет записи старше заданного момента времени.
	// Возвращает количество удалённых строк.
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// Scheduler управляет фоновыми задачами платформы.
type Scheduler struct {
	forecast        ForecastRunner
	measurements    RetentionCleaner
	forecastsCleaner RetentionCleaner
	cfg             config.ForecastConfig
	logger          *slog.Logger
}

// New создаёт планировщик задач.
func New(
	forecast ForecastRunner,
	measurements RetentionCleaner,
	forecastsCleaner RetentionCleaner,
	cfg config.ForecastConfig,
	logger *slog.Logger,
) *Scheduler {
	return &Scheduler{
		forecast:        forecast,
		measurements:    measurements,
		forecastsCleaner: forecastsCleaner,
		cfg:             cfg,
		logger:          logger,
	}
}

// Start запускает все фоновые задачи в отдельных горутинах.
// Блокируется до отмены ctx.
func (s *Scheduler) Start(ctx context.Context) {
	// ── Задача 1: пересчёт прогноза ──────────────────────────────────────
	interval := s.cfg.UpdateInterval
	if interval <= 0 {
		interval = 20 * time.Minute
	}

	s.logger.Info("планировщик запущен",
		"forecast_interval", interval,
		"retention_interval", "24h",
	)

	// Первый запуск сразу при старте (не ждать первого тика).
	go func() {
		s.runForecast(ctx)
	}()

	go s.forecastLoop(ctx, interval)
	go s.retentionLoop(ctx)

	<-ctx.Done()
	s.logger.Info("планировщик остановлен")
}

// forecastLoop выполняет пересчёт прогноза по расписанию.
func (s *Scheduler) forecastLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runForecast(ctx)
		}
	}
}

// runForecast выполняет один цикл расчёта прогноза.
func (s *Scheduler) runForecast(ctx context.Context) {
	start := time.Now()
	metrics.ForecastRunsTotal.Inc()

	if err := s.forecast.Run(ctx); err != nil {
		s.logger.Error("расчёт прогноза: ошибка", "err", err)
		metrics.ForecastErrors.Inc()
		return
	}

	dur := time.Since(start)
	metrics.ForecastRunDuration.Observe(dur.Seconds())
	s.logger.Info("расчёт прогноза: успешно", "duration", dur.Round(time.Millisecond))
}

// retentionLoop выполняет удаление устаревших данных раз в сутки.
// По ТЗ: хранение данных — 60 месяцев (5 лет).
func (s *Scheduler) retentionLoop(ctx context.Context) {
	// Первый запуск через 1 час после старта (не мешать инициализации).
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Hour):
	}

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		s.runRetention(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// runRetention удаляет данные старше 60 месяцев (5 лет по ТЗ).
func (s *Scheduler) runRetention(ctx context.Context) {
	// 5 лет = 60 месяцев — минимальный срок хранения по ТЗ.
	const retentionPeriod = 5 * 365 * 24 * time.Hour
	before := time.Now().UTC().Add(-retentionPeriod)

	s.logger.Info("retention: удаление устаревших данных", "before", before.Format("2006-01-02"))

	// Удаляем старые измерения.
	if n, err := s.measurements.DeleteOlderThan(ctx, before); err != nil {
		s.logger.Error("retention: удаление измерений", "err", err)
	} else if n > 0 {
		s.logger.Info("retention: измерения удалены", "count", n)
	}

	// Удаляем старые прогнозы (храним последние 30 дней — прогнозы не нужно хранить долго).
	forecastBefore := time.Now().UTC().Add(-30 * 24 * time.Hour)
	if n, err := s.forecastsCleaner.DeleteOlderThan(ctx, forecastBefore); err != nil {
		s.logger.Error("retention: удаление прогнозов", "err", err)
	} else if n > 0 {
		s.logger.Info("retention: прогнозы удалены", "count", n)
	}
}
