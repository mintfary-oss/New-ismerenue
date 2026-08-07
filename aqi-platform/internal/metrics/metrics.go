// Package metrics регистрирует Prometheus-метрики платформы.
//
// Метрики разбиты на группы:
//   - HTTP: счётчики запросов, latency гистограммы, активные соединения
//   - Business: текущий AQI по районам, счётчик прогнозных расчётов
//   - DB: размер пула соединений PostgreSQL
//
// Использование:
//
//	metrics.HTTPRequests.WithLabelValues("GET", "/api/v1/sensors", "200").Inc()
//	metrics.ForecastRunsTotal.Inc()
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ── HTTP метрики ──────────────────────────────────────────────────────────────

// HTTPRequests — счётчик HTTP-запросов с метками: method, path, status.
var HTTPRequests = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "aqi",
		Subsystem: "http",
		Name:      "requests_total",
		Help:      "Общее число HTTP-запросов",
	},
	[]string{"method", "path", "status"},
)

// HTTPDuration — гистограмма времени обработки HTTP-запросов (в секундах).
var HTTPDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "aqi",
		Subsystem: "http",
		Name:      "request_duration_seconds",
		Help:      "Время обработки HTTP-запроса",
		Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
	},
	[]string{"method", "path"},
)

// HTTPActiveConnections — текущее число активных HTTP-соединений.
var HTTPActiveConnections = promauto.NewGauge(
	prometheus.GaugeOpts{
		Namespace: "aqi",
		Subsystem: "http",
		Name:      "active_connections",
		Help:      "Число активных HTTP-соединений в данный момент",
	},
)

// ── Бизнес-метрики ────────────────────────────────────────────────────────────

// CurrentAQI — текущий AQI по точкам мониторинга.
var CurrentAQI = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: "aqi",
		Subsystem: "air",
		Name:      "current_aqi",
		Help:      "Текущий индекс AQI для контрольной точки",
	},
	[]string{"point_id", "district"},
)

// MeasurementsIngested — счётчик принятых измерений (от датчиков).
var MeasurementsIngested = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "aqi",
		Subsystem: "ingest",
		Name:      "measurements_total",
		Help:      "Число принятых измерений от датчиков",
	},
	[]string{"source"}, // "api" | "imap" | "batch"
)

// ForecastRunsTotal — счётчик запусков прогнозного движка.
var ForecastRunsTotal = promauto.NewCounter(
	prometheus.CounterOpts{
		Namespace: "aqi",
		Subsystem: "forecast",
		Name:      "runs_total",
		Help:      "Число запусков прогнозного движка EWMA+IDW",
	},
)

// ForecastRunDuration — время одного расчёта прогноза (в секундах).
var ForecastRunDuration = promauto.NewHistogram(
	prometheus.HistogramOpts{
		Namespace: "aqi",
		Subsystem: "forecast",
		Name:      "run_duration_seconds",
		Help:      "Время одного расчёта прогноза",
		Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.0},
	},
)

// ForecastErrors — счётчик ошибок прогнозного движка.
var ForecastErrors = promauto.NewCounter(
	prometheus.CounterOpts{
		Namespace: "aqi",
		Subsystem: "forecast",
		Name:      "errors_total",
		Help:      "Число ошибок прогнозного движка",
	},
)

// ActiveSensors — число активных датчиков.
var ActiveSensors = promauto.NewGauge(
	prometheus.GaugeOpts{
		Namespace: "aqi",
		Subsystem: "sensor",
		Name:      "active_total",
		Help:      "Число активных датчиков",
	},
)

// OnlineSensors — число датчиков онлайн (последний сигнал < 30 мин назад).
var OnlineSensors = promauto.NewGauge(
	prometheus.GaugeOpts{
		Namespace: "aqi",
		Subsystem: "sensor",
		Name:      "online_total",
		Help:      "Число датчиков с сигналом за последние 30 минут",
	},
)

// IMAPPollTotal — счётчик проверок IMAP почтового ящика.
var IMAPPollTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "aqi",
		Subsystem: "imap",
		Name:      "polls_total",
		Help:      "Число проверок IMAP почтового ящика",
	},
	[]string{"result"}, // "ok" | "error" | "empty"
)

// ── DB метрики ────────────────────────────────────────────────────────────────

// DBPoolSize — текущий размер пула соединений PostgreSQL.
var DBPoolSize = promauto.NewGauge(
	prometheus.GaugeOpts{
		Namespace: "aqi",
		Subsystem: "db",
		Name:      "pool_total_conns",
		Help:      "Текущий размер пула соединений PostgreSQL",
	},
)

// DBPoolIdle — число idle соединений в пуле.
var DBPoolIdle = promauto.NewGauge(
	prometheus.GaugeOpts{
		Namespace: "aqi",
		Subsystem: "db",
		Name:      "pool_idle_conns",
		Help:      "Число idle соединений PostgreSQL в пуле",
	},
)

// ── Инициализация ─────────────────────────────────────────────────────────────

func init() {
	// Регистрируем стандартные Go runtime метрики (memory, GC, goroutines).
	// Используем Register (не MustRegister) — допускаем повторную регистрацию
	// в тестах, где init() вызывается несколько раз из разных пакетов.
	registerIgnoreDup(collectors.NewGoCollector())
	registerIgnoreDup(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
}

// registerIgnoreDup регистрирует коллектор и молча игнорирует AlreadyRegisteredError.
func registerIgnoreDup(c prometheus.Collector) {
	if err := prometheus.Register(c); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			panic(err)
		}
	}
}

// Handler возвращает HTTP-обработчик для эндпоинта /metrics.
func Handler() http.Handler {
	return promhttp.Handler()
}

// ── Middleware ────────────────────────────────────────────────────────────────

// responseWriter — обёртка для захвата HTTP статус-кода.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Status() int {
	if rw.status == 0 {
		return http.StatusOK
	}
	return rw.status
}

// Middleware возвращает Chi-совместимый middleware для сбора HTTP-метрик.
// Записывает requests_total и request_duration_seconds по каждому маршруту.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		HTTPActiveConnections.Inc()
		defer HTTPActiveConnections.Dec()

		// Нормализуем path: убираем параметры UUID для группировки.
		path := normalizePath(r.URL.Path)

		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(rw.Status())

		HTTPRequests.WithLabelValues(r.Method, path, status).Inc()
		HTTPDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

// normalizePath нормализует URL path для метрик — заменяет UUID и числовые
// сегменты на плейсхолдеры, чтобы избежать взрыва cardinality.
//
// Например:
//
//	/api/v1/sensors/550e8400-e29b-41d4-a716-446655440000 → /api/v1/sensors/{id}
//	/api/v1/users/42 → /api/v1/users/{id}
func normalizePath(p string) string {
	// Быстрая нормализация: всё что похоже на UUID/число в path → {id}
	result := make([]byte, 0, len(p))
	i := 0
	for i < len(p) {
		if p[i] == '/' {
			result = append(result, '/')
			i++
			// Читаем следующий сегмент.
			j := i
			for j < len(p) && p[j] != '/' {
				j++
			}
			seg := p[i:j]
			if isUUID(seg) || isNumeric(seg) {
				result = append(result, []byte("{id}")...)
			} else {
				result = append(result, []byte(seg)...)
			}
			i = j
		} else {
			result = append(result, p[i])
			i++
		}
	}
	return string(result)
}

// isUUID проверяет, является ли строка UUID (8-4-4-4-12 hex).
func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}

// isNumeric проверяет, является ли строка числом (ID из целых чисел).
func isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
