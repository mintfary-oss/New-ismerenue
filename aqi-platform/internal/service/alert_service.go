// Package service — сервис AQI-алертов.
//
// AlertService проверяет текущие значения AQI по всем точкам мониторинга
// и отправляет email-уведомления получателям, когда AQI превышает порог.
//
// Антиспам: для каждой точки хранится время последней отправки алерта.
// Повторный алерт по той же точке отправляется не раньше cooldown_duration.
package service

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"sync"
	"time"

	"github.com/mintfary/aqi-platform/internal/config"
	"github.com/mintfary/aqi-platform/internal/domain"
)

// AlertEmailSender — интерфейс для отправки email-алертов.
// Реализуется *email.Sender.
type AlertEmailSender interface {
	// SendAQIAlert отправляет уведомление о превышении порога AQI.
	SendAQIAlert(to []string, data AQIAlertData) error
	// IsConfigured возвращает true если SMTP настроен.
	IsConfigured() bool
}

// AlertMeasurementReader читает последние измерения для проверки порогов.
type AlertMeasurementReader interface {
	Latest(ctx context.Context) ([]domain.LatestMeasurement, error)
}

// AQIAlertData — данные для email-шаблона алерта.
type AQIAlertData struct {
	// PointID — идентификатор точки мониторинга.
	PointID string
	// District — название района.
	District string
	// AQI — текущий числовой AQI.
	AQI int
	// Category — категория AQI (например "Нездоровое для чувствительных групп").
	Category string
	// CategoryColor — hex-цвет категории.
	CategoryColor string
	// Recommendation — рекомендация для населения.
	Recommendation string
	// MeasuredAt — время измерения.
	MeasuredAt time.Time
	// DashboardURL — ссылка на дашборд платформы.
	DashboardURL string
}

// AlertService проверяет AQI-значения и отправляет email-алерты.
type AlertService struct {
	measurements AlertMeasurementReader
	sender       AlertEmailSender
	cfg          config.AlertConfig
	baseURL      string
	logger       *slog.Logger

	// cooldown: pointID → время последней отправки алерта
	mu       sync.Mutex
	lastSent map[string]time.Time
}

// NewAlertService создаёт сервис алертов.
func NewAlertService(
	measurements AlertMeasurementReader,
	sender AlertEmailSender,
	cfg config.AlertConfig,
	baseURL string,
	logger *slog.Logger,
) *AlertService {
	return &AlertService{
		measurements: measurements,
		sender:       sender,
		cfg:          cfg,
		baseURL:      baseURL,
		logger:       logger,
		lastSent:     make(map[string]time.Time),
	}
}

// Check проверяет текущие AQI-значения и отправляет алерты при необходимости.
// Вызывается планировщиком после каждого расчёта прогноза.
func (s *AlertService) Check(ctx context.Context) {
	if !s.cfg.Enabled {
		return
	}
	if len(s.cfg.Recipients) == 0 {
		return
	}
	if !s.sender.IsConfigured() {
		s.logger.Warn("alert: SMTP не настроен, алерты отключены")
		return
	}

	latest, err := s.measurements.Latest(ctx)
	if err != nil {
		s.logger.Error("alert: ошибка чтения измерений", "err", err)
		return
	}

	now := time.Now()

	for _, m := range latest {
		aqi := domain.CalcOverallAQI(&m.Measurement)
		if aqi < s.cfg.Threshold {
			continue // ниже порога — не тревожим
		}

		pointID := m.Sensor.ID.String()
		if s.isOnCooldown(pointID, now) {
			s.logger.Debug("alert: кулдаун активен, пропускаем",
				"point_id", pointID, "aqi", aqi)
			continue
		}

		cat := domain.AQIToCategory(aqi)
		data := AQIAlertData{
			PointID:        pointID,
			District:       m.Sensor.Name,
			AQI:            aqi,
			Category:       domain.AQICategoryLabel(cat),
			CategoryColor:  domain.AQICategoryColor(cat),
			Recommendation: aqiRecommendation(cat),
			MeasuredAt:     m.Measurement.Time,
			DashboardURL:   s.baseURL + "/dashboard",
		}

		if err := s.sender.SendAQIAlert(s.cfg.Recipients, data); err != nil {
			s.logger.Error("alert: ошибка отправки",
				"point_id", pointID, "err", err)
			continue
		}

		s.setCooldown(pointID, now)
		s.logger.Info("alert: отправлен",
			"point_id", pointID,
			"district", data.District,
			"aqi", aqi,
			"category", data.Category,
			"recipients", len(s.cfg.Recipients),
		)
	}
}

// isOnCooldown проверяет, находится ли точка на кулдауне (без блокировки —
// вызывать только при удержании mu или до первого вызова setCooldown).
func (s *AlertService) isOnCooldown(pointID string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	last, ok := s.lastSent[pointID]
	if !ok {
		return false
	}
	return now.Sub(last) < s.cfg.CooldownDuration
}

// setCooldown фиксирует время последней отправки для точки.
func (s *AlertService) setCooldown(pointID string, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastSent[pointID] = now
}

// aqiRecommendation возвращает краткую рекомендацию для населения.
func aqiRecommendation(cat domain.AQICategory) string {
	switch cat {
	case domain.AQICategoryGood:
		return "Качество воздуха удовлетворительное, активность на свежем воздухе безопасна."
	case domain.AQICategoryModerate:
		return "Качество воздуха приемлемое. Чувствительным людям рекомендуется ограничить длительные нагрузки на свежем воздухе."
	case domain.AQICategoryUnhealthy:
		return "Чувствительным группам (дети, пожилые, люди с заболеваниями) рекомендуется ограничить пребывание на улице."
	case domain.AQICategoryBad:
		return "Всем рекомендуется сократить пребывание на улице. Чувствительным группам — оставаться дома."
	case domain.AQICategoryVeryBad:
		return "Избегайте длительного нахождения на улице. Используйте маски FFP2/N95 при выходе на улицу."
	case domain.AQICategoryHazardous:
		return "ОПАСНО. Оставайтесь дома, закройте окна. При необходимости выхода используйте респиратор."
	default:
		return "Следите за обновлениями данных о качестве воздуха."
	}
}

// ── HTML-шаблон email-алерта ───────────────────────────────────────────────

const aqiAlertHTML = `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Алерт качества воздуха — AQI Кемерово</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
         background: #f4f6f9; margin: 0; padding: 20px; }
  .card { max-width: 560px; margin: 0 auto; background: #fff;
          border-radius: 10px; overflow: hidden; box-shadow: 0 2px 12px rgba(0,0,0,.12); }
  .header { background: {{.CategoryColor}}; padding: 24px 32px; color: #fff; }
  .header h1 { margin: 0 0 6px; font-size: 22px; }
  .header p  { margin: 0; opacity: .9; font-size: 14px; }
  .aqi-badge { display: inline-block; background: rgba(255,255,255,.25);
               border-radius: 50px; padding: 6px 18px; font-size: 28px;
               font-weight: 700; margin-top: 12px; }
  .body { padding: 28px 32px; }
  .meta { display: flex; gap: 24px; margin-bottom: 20px; }
  .meta-item label { font-size: 11px; color: #999; text-transform: uppercase; display: block; }
  .meta-item value { font-size: 15px; font-weight: 600; color: #1a1d27; }
  .rec { background: #f8f9fc; border-left: 4px solid {{.CategoryColor}};
         border-radius: 0 6px 6px 0; padding: 14px 18px; margin: 20px 0;
         font-size: 14px; color: #444; line-height: 1.5; }
  .btn { display: inline-block; padding: 12px 28px; background: #5b8dee;
         color: #fff !important; text-decoration: none; border-radius: 7px;
         font-weight: 600; font-size: 14px; margin-top: 8px; }
  .footer { padding: 16px 32px; border-top: 1px solid #eee; font-size: 12px; color: #999; }
</style>
</head>
<body>
<div class="card">
  <div class="header">
    <h1>🌫️ Предупреждение о качестве воздуха</h1>
    <p>{{.District}} · {{.MeasuredAt.Format "02.01.2006 15:04"}}</p>
    <div class="aqi-badge">AQI {{.AQI}}</div>
  </div>
  <div class="body">
    <div class="meta">
      <div class="meta-item">
        <label>Статус</label>
        <value>{{.Category}}</value>
      </div>
      <div class="meta-item">
        <label>Точка мониторинга</label>
        <value>{{.District}}</value>
      </div>
    </div>
    <div class="rec">{{.Recommendation}}</div>
    <a href="{{.DashboardURL}}" class="btn">Открыть дашборд</a>
  </div>
  <div class="footer">
    Это автоматическое уведомление от AQI Кемерово.
    Чтобы отписаться — обратитесь к администратору платформы.
  </div>
</div>
</body>
</html>`

// RenderAQIAlertEmail рендерит HTML-тело email-алерта.
func RenderAQIAlertEmail(data AQIAlertData) (string, error) {
	tmpl, err := template.New("alert").Parse(aqiAlertHTML)
	if err != nil {
		return "", fmt.Errorf("alert: parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("alert: execute template: %w", err)
	}
	return buf.String(), nil
}
