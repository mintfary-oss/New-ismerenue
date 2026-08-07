// Package server содержит HTTP-сервер с graceful shutdown.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/mintfary/aqi-platform/internal/config"
)

// Server — обёртка над стандартным net/http.Server.
type Server struct {
	http   *http.Server
	cfg    config.ServerConfig
	logger *slog.Logger
}

// New создаёт HTTP-сервер с заданным роутером и конфигурацией.
func New(cfg config.ServerConfig, handler http.Handler, logger *slog.Logger) *Server {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	httpSrv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
		// Скрываем версию Go из заголовка Server.
		// Заголовок будет перезаписан middleware security.
	}

	return &Server{
		http:   httpSrv,
		cfg:    cfg,
		logger: logger,
	}
}

// Start запускает сервер и блокируется до завершения.
// Graceful shutdown выполняется при отмене ctx.
func (s *Server) Start(ctx context.Context) error {
	// Горутина для graceful shutdown при отмене контекста.
	shutdownErr := make(chan error, 1)
	go func() {
		<-ctx.Done()
		s.logger.Info("начало graceful shutdown HTTP-сервера")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		shutdownErr <- s.http.Shutdown(shutdownCtx)
	}()

	// Запуск TLS или обычного HTTP.
	var listenErr error
	if s.cfg.TLSCertFile != "" && s.cfg.TLSKeyFile != "" {
		s.logger.Info("запуск HTTPS-сервера", "addr", s.http.Addr)
		listenErr = s.http.ListenAndServeTLS(s.cfg.TLSCertFile, s.cfg.TLSKeyFile)
	} else {
		s.logger.Info("запуск HTTP-сервера", "addr", s.http.Addr)
		listenErr = s.http.ListenAndServe()
	}

	// ErrServerClosed — нормальное завершение после Shutdown.
	if errors.Is(listenErr, http.ErrServerClosed) {
		return <-shutdownErr
	}
	return listenErr
}

// Addr возвращает адрес, на котором слушает сервер.
func (s *Server) Addr() string {
	return s.http.Addr
}
