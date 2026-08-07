package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/mintfary/aqi-platform/internal/handler"
)

// NewRouter создаёт и настраивает Chi-роутер со всеми маршрутами.
func NewRouter(h *handler.Handlers) http.Handler {
	r := chi.NewRouter()

	// ── Базовый стек middleware (применяется ко всем маршрутам) ────────────
	r.Use(chimiddleware.RequestID)            // X-Request-Id
	r.Use(chimiddleware.RealIP)              // берём IP из X-Forwarded-For
	r.Use(chimiddleware.Logger)              // структурированный лог запросов
	r.Use(chimiddleware.Recoverer)           // перехват паники
	r.Use(chimiddleware.Compress(5))         // gzip для ответов > 1KB
	r.Use(chimiddleware.Timeout(60 * time.Second))

	// CORS — разные политики для платформы и виджета.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // TODO Sprint 4: взять из cfg
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-Id"},
		ExposedHeaders:   []string{"X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// ── Публичные маршруты ─────────────────────────────────────────────────
	r.Get("/health", h.Health.Live)
	r.Get("/ready",  h.Health.Ready)

	// Публичный виджет (без авторизации — открытый доступ по ТЗ).
	r.Route("/widget", func(r chi.Router) {
		r.Get("/",         h.Widget.Index)
		r.Get("/data",     h.Widget.Data)
		r.Get("/forecast", h.Widget.Forecast)
		r.Get("/weather",  h.Widget.Weather)
	})

	// OpenAPI документация (SwaggerUI).
	r.Get("/api/v1/docs",       h.Health.APIDocs)
	r.Get("/api/v1/openapi.yaml", h.Health.OpenAPISpec)

	// ── Аутентификация (rate-limit применяется внутри handler) ────────────
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/login",          h.Auth.Login)
		r.Post("/refresh",        h.Auth.Refresh)
		r.Post("/logout",         h.Auth.Logout)
		r.Post("/forgot-password", h.Auth.ForgotPassword)
		r.Post("/reset-password",  h.Auth.ResetPassword)
	})

	// ── Защищённые маршруты (JWT обязателен) ──────────────────────────────
	r.Route("/api/v1", func(r chi.Router) {
		// TODO Sprint 4: r.Use(h.Middleware.Auth)

		// Пользователи.
		r.Route("/users", func(r chi.Router) {
			r.Get("/",        h.User.List)
			r.Post("/",       h.User.Create)
			r.Get("/{id}",    h.User.Get)
			r.Patch("/{id}",  h.User.Update)
			r.Delete("/{id}", h.User.Delete)
		})

		// Датчики.
		r.Route("/sensors", func(r chi.Router) {
			r.Get("/",           h.Sensor.List)
			r.Post("/",          h.Sensor.Create)
			r.Get("/{id}",       h.Sensor.Get)
			r.Patch("/{id}",     h.Sensor.Update)
			r.Delete("/{id}",    h.Sensor.Delete)
			r.Get("/{id}/status", h.Sensor.Status)
		})

		// Измерения.
		r.Route("/measurements", func(r chi.Router) {
			r.Get("/",          h.Measurement.List)
			r.Get("/latest",    h.Measurement.Latest)
			r.Get("/aggregate", h.Measurement.Aggregate)
		})

		// Прогнозы.
		r.Route("/forecast", func(r chi.Router) {
			r.Get("/points",          h.Forecast.Points)
			r.Get("/current",         h.Forecast.Current)
			r.Get("/city-average",    h.Forecast.CityAverage)
			r.Get("/{point_id}",      h.Forecast.ByPoint)
			r.Get("/district/{id}",   h.Forecast.ByDistrict)
		})

		// API-токены.
		r.Route("/tokens", func(r chi.Router) {
			r.Get("/",     h.Token.List)
			r.Post("/",    h.Token.Create)
			r.Delete("/{id}", h.Token.Delete)
		})

		// Загрузка данных.
		r.Route("/ingest", func(r chi.Router) {
			r.Post("/data",             h.Ingest.Upload)
			r.Get("/history",           h.Ingest.History)
			r.Get("/validation-rules",  h.Ingest.GetRules)
			r.Put("/validation-rules",  h.Ingest.UpdateRules)
		})

		// Отчёты.
		r.Route("/reports", func(r chi.Router) {
			r.Get("/",              h.Report.List)
			r.Post("/generate",     h.Report.Generate)
			r.Get("/{id}/download", h.Report.Download)
		})

		// Статистика (только Admin).
		r.Route("/stats", func(r chi.Router) {
			r.Get("/availability", h.Stats.Availability)
			r.Get("/data-coverage", h.Stats.DataCoverage)
		})

		// Обратная связь.
		r.Route("/feedback", func(r chi.Router) {
			r.Post("/", h.Feedback.Create)
			r.Get("/",  h.Feedback.List)
		})
	})

	return r
}
