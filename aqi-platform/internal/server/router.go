package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/mintfary/aqi-platform/internal/domain"
	"github.com/mintfary/aqi-platform/internal/handler"
	"github.com/mintfary/aqi-platform/internal/middleware"
)

// NewRouter создаёт и настраивает Chi-роутер со всеми маршрутами.
// auth — валидатор JWT токенов (AuthService реализует этот интерфейс).
func NewRouter(h *handler.Handlers, auth middleware.TokenValidator) http.Handler {
	r := chi.NewRouter()

	// ── Базовый стек middleware ────────────────────────────────────────────
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Compress(5))
	r.Use(chimiddleware.Timeout(60 * time.Second))
	r.Use(middleware.SecurityHeaders())

	// CORS — разрешаем все origins (публичный виджет + SPA).
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-Id"},
		ExposedHeaders:   []string{"X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// ── Публичные маршруты (без авторизации) ──────────────────────────────
	r.Get("/health", h.Health.Live)
	r.Get("/ready", h.Health.Ready)

	r.Route("/widget", func(r chi.Router) {
		r.Use(middleware.WidgetSecurityHeaders())
		r.Get("/", h.Widget.Index)
		r.Get("/data", h.Widget.Data)
		r.Get("/forecast", h.Widget.Forecast)
		r.Get("/weather", h.Widget.Weather)
	})

	r.Get("/api/v1/docs", h.Health.APIDocs)
	r.Get("/api/v1/openapi.yaml", h.Health.OpenAPISpec)

	// ── Аутентификация ────────────────────────────────────────────────────
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/login", h.Auth.Login)
		r.Post("/refresh", h.Auth.Refresh)
		r.Post("/logout", h.Auth.Logout)
		r.Post("/forgot-password", h.Auth.ForgotPassword)
		r.Post("/reset-password", h.Auth.ResetPassword)
	})

	// ── Публичные прогнозы (чтение без авторизации) ───────────────────────
	r.Route("/api/v1/public", func(r chi.Router) {
		r.Get("/forecast/current", h.Forecast.Current)
		r.Get("/forecast/city-average", h.Forecast.CityAverage)
		r.Get("/forecast/points", h.Forecast.Points)
	})

	// ── Защищённые маршруты (JWT обязателен) ──────────────────────────────
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Auth(auth))

		// Пользователи — только Admin.
		r.Route("/users", func(r chi.Router) {
			r.Use(middleware.RequireAdmin())
			r.Get("/", h.User.List)
			r.Post("/", h.User.Create)
			r.Get("/{id}", h.User.Get)
			r.Patch("/{id}", h.User.Update)
			r.Delete("/{id}", h.User.Delete)
		})

		// Датчики: чтение — все авторизованные; запись — только Admin.
		r.Route("/sensors", func(r chi.Router) {
			r.Get("/", h.Sensor.List)
			r.Get("/{id}", h.Sensor.Get)
			r.Get("/{id}/status", h.Sensor.Status)
			r.With(middleware.RequireAdmin()).Post("/", h.Sensor.Create)
			r.With(middleware.RequireAdmin()).Patch("/{id}", h.Sensor.Update)
			r.With(middleware.RequireAdmin()).Delete("/{id}", h.Sensor.Delete)
		})

		// Измерения — все авторизованные, только чтение.
		r.Route("/measurements", func(r chi.Router) {
			r.Get("/", h.Measurement.List)
			r.Get("/latest", h.Measurement.Latest)
			r.Get("/aggregate", h.Measurement.Aggregate)
		})

		// Прогнозы — все авторизованные, только чтение.
		r.Route("/forecast", func(r chi.Router) {
			r.Get("/points", h.Forecast.Points)
			r.Get("/current", h.Forecast.Current)
			r.Get("/city-average", h.Forecast.CityAverage)
			r.Get("/{point_id}", h.Forecast.ByPoint)
			r.Get("/district/{id}", h.Forecast.ByDistrict)
		})

		// API-токены — пользователь управляет своими токенами.
		r.Route("/tokens", func(r chi.Router) {
			r.Get("/", h.Token.List)
			r.Post("/", h.Token.Create)
			r.Delete("/{id}", h.Token.Delete)
		})

		// Загрузка данных — Analyst + Admin.
		r.Route("/ingest", func(r chi.Router) {
			r.Use(middleware.RequireAnalystOrAdmin())
			r.Post("/data", h.Ingest.Upload)
			r.Get("/history", h.Ingest.History)
			r.Get("/validation-rules", h.Ingest.GetRules)
			r.With(middleware.RequireAdmin()).Put("/validation-rules", h.Ingest.UpdateRules)
		})

		// Отчёты — Analyst + Admin.
		r.Route("/reports", func(r chi.Router) {
			r.Use(middleware.RequireRole(domain.RoleAdmin, domain.RoleAnalyst))
			r.Get("/", h.Report.List)
			r.Post("/generate", h.Report.Generate)
			r.Get("/{id}/download", h.Report.Download)
		})

		// Статистика — только Admin.
		r.Route("/stats", func(r chi.Router) {
			r.Use(middleware.RequireAdmin())
			r.Get("/availability", h.Stats.Availability)
			r.Get("/data-coverage", h.Stats.DataCoverage)
		})

		// Обратная связь: создание — все авторизованные; список — только Admin.
		r.Route("/feedback", func(r chi.Router) {
			r.Post("/", h.Feedback.Create)
			r.With(middleware.RequireAdmin()).Get("/", h.Feedback.List)
		})
	})

	return r
}
