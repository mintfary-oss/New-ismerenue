package middleware

import (
	"net/http"
)

// SecurityHeaders добавляет HTTP заголовки безопасности в каждый ответ.
// Реализует защиту согласно OWASP Top 10:2025 и рекомендациям securityheaders.com.
func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()

			// Предотвращает определение MIME-типа браузером (MIME sniffing).
			h.Set("X-Content-Type-Options", "nosniff")

			// Защита от XSS в старых браузерах.
			h.Set("X-XSS-Protection", "1; mode=block")

			// Контроль Referrer — не передавать origin на другие сайты.
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Запрет встраивания в iframe (для платформы; виджет переопределяет).
			h.Set("X-Frame-Options", "SAMEORIGIN")

			// Ограничение разрешений браузера (геолокация и т.д.).
			h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			// Удаляем заголовок Server (не раскрывать версию Go).
			h.Set("Server", "aqi-platform")

			// HSTS — включать только при HTTPS.
			if r.TLS != nil {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			}

			// Content-Security-Policy для API (не нужен для HTML страниц — для них отдельно).
			if isAPIPath(r.URL.Path) {
				h.Set("Content-Security-Policy", "default-src 'none'")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// WidgetSecurityHeaders — заголовки для публичного виджета.
// Разрешает встраивание в iframe на любом сайте (требование ТЗ).
func WidgetSecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()

			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "geolocation=(self)")

			// Разрешаем встраивание на любом сайте (iframe для виджета по ТЗ).
			h.Del("X-Frame-Options")

			// CSP для виджета: разрешаем MapLibre GL и внешние тайлы.
			h.Set("Content-Security-Policy",
				"default-src 'self'; "+
					"script-src 'self' 'unsafe-inline'; "+ // MapLibre требует inline scripts
					"style-src 'self' 'unsafe-inline'; "+
					"img-src 'self' data: https://*.tile.openstreetmap.org https://*.2gis.com; "+
					"connect-src 'self' https://*.openstreetmap.org; "+
					"frame-ancestors *",
			)

			next.ServeHTTP(w, r)
		})
	}
}

// isAPIPath проверяет, является ли путь API-эндпоинтом.
func isAPIPath(path string) bool {
	return len(path) > 4 && path[:4] == "/api"
}
