package middleware

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter — token bucket лимитер на основе IP-адреса.
// Для продакшна рекомендуется использовать Redis-based limiter (распределённый).
// Текущая реализация — in-memory, работает в рамках одного процесса.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     int           // максимум запросов за окно
	window   time.Duration // размер окна
	cleanup  time.Duration // интервал очистки старых buckets
}

type bucket struct {
	count    int
	resetAt  time.Time
}

// NewRateLimiter создаёт in-memory лимитер.
func NewRateLimiter(rate int, window, cleanup time.Duration) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*bucket),
		rate:    rate,
		window:  window,
		cleanup: cleanup,
	}
	go rl.runCleanup()
	return rl
}

// Allow проверяет и уменьшает доступный лимит для ключа (IP или user_id).
// Возвращает true, если запрос разрешён.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.buckets[key]
	if !exists || now.After(b.resetAt) {
		rl.buckets[key] = &bucket{count: 1, resetAt: now.Add(rl.window)}
		return true
	}

	if b.count >= rl.rate {
		return false
	}
	b.count++
	return true
}

// runCleanup периодически удаляет истёкшие buckets (предотвращает утечку памяти).
func (rl *RateLimiter) runCleanup() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, b := range rl.buckets {
			if now.After(b.resetAt) {
				delete(rl.buckets, key)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware возвращает Chi-совместимый middleware.
// key — функция извлечения ключа из запроса (IP, user ID, и т.д.).
func (rl *RateLimiter) Middleware(key func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			k := key(r)
			if !rl.Allow(k) {
				w.Header().Set("Retry-After", "60")
				writeError(w, http.StatusTooManyRequests, "превышен лимит запросов, повторите через минуту")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// KeyByIP извлекает IP-адрес клиента как ключ для rate limiting.
// Предполагает, что RealIP middleware уже применён.
func KeyByIP(r *http.Request) string {
	return r.RemoteAddr
}

// Предустановленные лимитеры для различных эндпоинтов.
var (
	// LoginLimiter — строгий лимит для /auth/login: 10 попыток в 15 минут.
	LoginLimiter = NewRateLimiter(10, 15*time.Minute, 30*time.Minute)

	// APILimiter — стандартный лимит для API: 100 запросов в минуту.
	APILimiter = NewRateLimiter(100, time.Minute, 5*time.Minute)

	// WidgetLimiter — публичный виджет: 300 запросов в минуту (больше трафика).
	WidgetLimiter = NewRateLimiter(300, time.Minute, 5*time.Minute)
)
