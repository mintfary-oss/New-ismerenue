package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mintfary/aqi-platform/internal/docs"
)

// swaggerUIHTML — самодостаточная страница Swagger UI (CDN-версия).
// Позволяет просматривать и тестировать API без сборки фронтенда.
var swaggerUIHTML = []byte(`<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>AQI Platform — API Docs</title>
  <link rel="stylesheet"
        href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  <style>
    body { margin: 0; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    .topbar { background: #1a3a52; padding: 12px 20px; display: flex; align-items: center; gap: 16px; }
    .topbar h1 { color: #fff; margin: 0; font-size: 1.2rem; font-weight: 600; }
    .topbar .badge { background: #26a65b; color: #fff; border-radius: 4px; padding: 2px 8px; font-size: 0.75rem; }
    #swagger-ui { max-width: 1400px; margin: 0 auto; padding: 16px; }
  </style>
</head>
<body>
  <div class="topbar">
    <h1>🌬 AQI Platform</h1>
    <span class="badge">v1.0.0</span>
    <span style="color:#aaa;font-size:0.85rem">Платформа мониторинга качества воздуха • Кемерово</span>
  </div>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: "/api/v1/openapi.yaml",
      dom_id: "#swagger-ui",
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: "BaseLayout",
      deepLinking: true,
      displayRequestDuration: true,
      filter: true,
      tryItOutEnabled: true,
      persistAuthorization: true,
    });
  </script>
</body>
</html>
`)

// HealthHandler обрабатывает запросы /health, /ready и документацию API.
type HealthHandler struct {
	db     *pgxpool.Pool
	logger *slog.Logger
	start  time.Time
}

func init() {
	// Фиксируем время запуска для uptime.
	_ = time.Now()
}

// Live — GET /health
// Liveness probe: возвращает 200 если процесс работает.
func (h *HealthHandler) Live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// Ready — GET /ready
// Readiness probe: возвращает 200 если все зависимости (БД, Redis) готовы.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	type depStatus struct {
		Status  string `json:"status"`
		Latency string `json:"latency,omitempty"`
		Error   string `json:"error,omitempty"`
	}
	type response struct {
		Status       string               `json:"status"`
		Deps         map[string]depStatus `json:"dependencies"`
		Uptime       string               `json:"uptime"`
		GoVer        string               `json:"go_version"`
		NumCPU       int                  `json:"num_cpu"`
		NumGoroutine int                  `json:"goroutines"`
	}

	deps := make(map[string]depStatus)
	allOK := true

	// Проверка PostgreSQL.
	if h.db != nil {
		start := time.Now()
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := h.db.Ping(ctx); err != nil {
			deps["postgres"] = depStatus{Status: "error", Error: err.Error()}
			allOK = false
		} else {
			deps["postgres"] = depStatus{
				Status:  "ok",
				Latency: time.Since(start).String(),
			}
		}
	} else {
		deps["postgres"] = depStatus{Status: "not_configured"}
	}

	uptime := "unknown"
	if !h.start.IsZero() {
		uptime = fmt.Sprintf("%.0fs", time.Since(h.start).Seconds())
	}

	status := "ok"
	code := http.StatusOK
	if !allOK {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	writeJSON(w, code, response{
		Status:       status,
		Deps:         deps,
		Uptime:       uptime,
		GoVer:        runtime.Version(),
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
	})
}

// APIDocs — GET /api/v1/docs
// Отдаёт Swagger UI — интерактивную документацию API.
func (h *HealthHandler) APIDocs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(swaggerUIHTML)
}

// OpenAPISpec — GET /api/v1/openapi.yaml
// Отдаёт встроенный OpenAPI 3.1 YAML-файл.
func (h *HealthHandler) OpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(docs.OpenAPISpec)
}
