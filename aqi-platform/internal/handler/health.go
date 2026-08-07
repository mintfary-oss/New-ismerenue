package handler

import (
	"context"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthHandler обрабатывает запросы /health и /ready.
type HealthHandler struct {
	db     *pgxpool.Pool
	logger *slog.Logger
	start  time.Time
}

func init() {
	// Записываем время старта приложения.
	_ = time.Now()
}

// Live — GET /health
// Возвращает 200 если процесс работает (liveness probe для Docker/K8s).
func (h *HealthHandler) Live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// Ready — GET /ready
// Возвращает 200 если все зависимости (БД, Redis) готовы (readiness probe).
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	type depStatus struct {
		Status string `json:"status"`
		Latency string `json:"latency,omitempty"`
		Error   string `json:"error,omitempty"`
	}
	type response struct {
		Status  string               `json:"status"`
		Deps    map[string]depStatus `json:"dependencies"`
		Uptime  string               `json:"uptime"`
		GoVer   string               `json:"go_version"`
		NumCPU  int                  `json:"num_cpu"`
		NumGoroutine int             `json:"goroutines"`
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

	status := "ok"
	code := http.StatusOK
	if !allOK {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	writeJSON(w, code, response{
		Status:       status,
		Deps:         deps,
		GoVer:        runtime.Version(),
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
	})
}

// APIDocs — GET /api/v1/docs — swagger UI.
func (h *HealthHandler) APIDocs(w http.ResponseWriter, _ *http.Request) {
	// TODO Sprint 7: отдать embedded swagger-ui HTML.
	http.Redirect(w, nil, "/api/v1/openapi.yaml", http.StatusFound)
}

// OpenAPISpec — GET /api/v1/openapi.yaml.
func (h *HealthHandler) OpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	// TODO Sprint 7: embed openapi.yaml.
	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("# AQI Platform OpenAPI spec\nopenapi: 3.1.0\n"))
}
