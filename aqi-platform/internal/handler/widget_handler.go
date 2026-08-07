package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/mintfary/aqi-platform/internal/domain"
	"github.com/mintfary/aqi-platform/internal/service"
)

// WidgetHandler реализует публичный виджет качества воздуха.
// Все эндпоинты без авторизации — открытый доступ для iframe-встраивания.
// CORS: разрешены все origins (по ТЗ: виджет публичный).
type WidgetHandler struct {
	measureSvc  *service.MeasurementService
	forecastSvc *service.ForecastService
	logger      *slog.Logger
}

// NewWidgetHandler создаёт обработчик виджета.
func NewWidgetHandler(
	measureSvc *service.MeasurementService,
	forecastSvc *service.ForecastService,
	logger *slog.Logger,
) *WidgetHandler {
	return &WidgetHandler{
		measureSvc:  measureSvc,
		forecastSvc: forecastSvc,
		logger:      logger,
	}
}

// Index возвращает HTML-страницу виджета для встраивания через <iframe>.
// Виджет самодостаточен: JS внутри HTML, нет внешних зависимостей.
func (h *WidgetHandler) Index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// X-Frame-Options: ALLOWALL — виджет специально предназначен для iframe.
	w.Header().Set("X-Frame-Options", "ALLOWALL")
	w.Header().Set("Content-Security-Policy", "frame-ancestors *")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(widgetHTML))
}

// Data возвращает JSON с актуальными данными AQI по всем датчикам.
// Используется JavaScript внутри виджета для обновления данных.
func (h *WidgetHandler) Data(w http.ResponseWriter, r *http.Request) {
	latest, err := h.measureSvc.Latest(r.Context())
	if err != nil {
		h.logger.Error("widget data: чтение измерений", "err", err)
		ok(w, widgetEmptyData())
		return
	}

	type sensorData struct {
		ID       string             `json:"id"`
		Name     string             `json:"name"`
		Lat      float64            `json:"lat"`
		Lng      float64            `json:"lng"`
		AQI      int                `json:"aqi"`
		Category domain.AQICategory `json:"category"`
		Label    string             `json:"label"`
		Color    string             `json:"color"`
		PM25     *float64           `json:"pm25,omitempty"`
		NO2      *float64           `json:"no2,omitempty"`
		IsOnline bool               `json:"is_online"`
		Time     string             `json:"time"`
	}

	sensors := make([]sensorData, 0, len(latest))
	for _, lm := range latest {
		sensors = append(sensors, sensorData{
			ID:       lm.Sensor.ID.String(),
			Name:     lm.Sensor.Name,
			Lat:      lm.Sensor.Lat,
			Lng:      lm.Sensor.Lng,
			AQI:      lm.AQI,
			Category: lm.AQICategory,
			Label:    domain.AQICategoryLabel(lm.AQICategory),
			Color:    domain.AQICategoryColor(lm.AQICategory),
			PM25:     lm.Measurement.PM25,
			NO2:      lm.Measurement.NO2,
			IsOnline: lm.Sensor.IsOnline(),
			Time:     lm.Measurement.Time.Format(time.RFC3339),
		})
	}

	// Вычисляем средний AQI по городу как максимум (консервативная оценка).
	cityAQI := 0
	for _, s := range sensors {
		if s.AQI > cityAQI {
			cityAQI = s.AQI
		}
	}
	cityCat := domain.AQIToCategory(cityAQI)

	ok(w, map[string]any{
		"city_aqi":      cityAQI,
		"city_category": cityCat,
		"city_label":    domain.AQICategoryLabel(cityCat),
		"city_color":    domain.AQICategoryColor(cityCat),
		"sensors":       sensors,
		"count":         len(sensors),
		"updated_at":    time.Now().UTC().Format(time.RFC3339),
	})
}

// Forecast возвращает текущий прогноз для виджета (упрощённый формат).
func (h *WidgetHandler) Forecast(w http.ResponseWriter, r *http.Request) {
	city, err := h.forecastSvc.CityAverage(r.Context())
	if err != nil {
		if isNotFound(err) {
			ok(w, map[string]any{
				"available":  false,
				"message":    "прогноз ещё не рассчитан",
				"updated_at": time.Now().UTC().Format(time.RFC3339),
			})
			return
		}
		h.logger.Error("widget forecast: ошибка", "err", err)
		ok(w, map[string]any{"available": false, "message": "ошибка прогноза"})
		return
	}

	type districtItem struct {
		Name  string `json:"name"`
		AQI   int    `json:"aqi"`
		Label string `json:"label"`
		Color string `json:"color"`
	}
	districts := make([]districtItem, 0, len(city.Districts))
	for _, d := range city.Districts {
		districts = append(districts, districtItem{
			Name:  d.DistrictName,
			AQI:   d.AQI,
			Label: domain.AQICategoryLabel(d.AQICategory),
			Color: domain.AQICategoryColor(d.AQICategory),
		})
	}

	ok(w, map[string]any{
		"available":     true,
		"city_aqi":      city.CityAQI,
		"city_label":    domain.AQICategoryLabel(city.CityCategory),
		"city_color":    domain.AQICategoryColor(city.CityCategory),
		"districts":     districts,
		"forecast_time": city.Time.Format(time.RFC3339),
		"updated_at":    time.Now().UTC().Format(time.RFC3339),
	})
}

// Weather возвращает текущие метеоданные (температура, влажность, ветер)
// из последних измерений датчиков.
func (h *WidgetHandler) Weather(w http.ResponseWriter, r *http.Request) {
	latest, err := h.measureSvc.Latest(r.Context())
	if err != nil || len(latest) == 0 {
		ok(w, map[string]any{
			"available":  false,
			"updated_at": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	// Берём среднее по всем датчикам с метеоданными.
	var sumTemp, sumHum, sumWind float64
	var cntTemp, cntHum, cntWind int

	for _, lm := range latest {
		m := lm.Measurement
		if m.Temperature != nil {
			sumTemp += *m.Temperature
			cntTemp++
		}
		if m.Humidity != nil {
			sumHum += *m.Humidity
			cntHum++
		}
		if m.WindSpeed != nil {
			sumWind += *m.WindSpeed
			cntWind++
		}
	}

	resp := map[string]any{
		"available":  true,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}
	if cntTemp > 0 {
		resp["temperature"] = round2(sumTemp / float64(cntTemp))
	}
	if cntHum > 0 {
		resp["humidity"] = round2(sumHum / float64(cntHum))
	}
	if cntWind > 0 {
		resp["wind_speed"] = round2(sumWind / float64(cntWind))
	}

	ok(w, resp)
}

// ── Вспомогательные функции ────────────────────────────────────────────────

func widgetEmptyData() map[string]any {
	return map[string]any{
		"city_aqi":      0,
		"city_category": domain.AQICategoryGood,
		"city_label":    "Нет данных",
		"city_color":    "#cccccc",
		"sensors":       []any{},
		"count":         0,
		"updated_at":    time.Now().UTC().Format(time.RFC3339),
	}
}

// round2 округляет до 2 знаков после запятой.
func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

// widgetHTML — самодостаточный HTML-виджет для встраивания через <iframe>.
// Данные запрашиваются у /widget/data через fetch() каждые 60 секунд.
const widgetHTML = `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Качество воздуха — Кемерово</title>
<style>
  *{box-sizing:border-box;margin:0;padding:0}
  body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
       background:#f5f7fa;color:#1a1a2e;min-height:100vh}
  .header{padding:16px 20px;background:#1a1a2e;color:#fff}
  .header h1{font-size:16px;font-weight:600;letter-spacing:.3px}
  .header p{font-size:12px;opacity:.7;margin-top:2px}
  .city-block{padding:20px;text-align:center}
  .city-aqi{font-size:72px;font-weight:800;line-height:1;transition:color .5s}
  .city-label{font-size:18px;font-weight:600;margin-top:8px}
  .city-sub{font-size:12px;opacity:.6;margin-top:4px}
  .sensors{padding:0 16px 16px}
  .sensor-card{background:#fff;border-radius:12px;padding:14px 16px;
               margin-bottom:10px;display:flex;align-items:center;
               box-shadow:0 2px 8px rgba(0,0,0,.06)}
  .dot{width:12px;height:12px;border-radius:50%;margin-right:12px;flex-shrink:0}
  .sensor-name{flex:1;font-size:14px;font-weight:500}
  .sensor-aqi{font-size:22px;font-weight:700}
  .sensor-pm{font-size:11px;opacity:.55;margin-top:2px;text-align:right}
  .offline{opacity:.45}
  .footer{text-align:center;font-size:10px;opacity:.4;padding-bottom:16px}
  .update-time{font-size:10px;opacity:.4;margin-top:4px}
</style>
</head>
<body>
<div class="header">
  <h1>Качество атмосферного воздуха</h1>
  <p id="city-name">г. Кемерово</p>
</div>
<div class="city-block">
  <div class="city-aqi" id="city-aqi">—</div>
  <div class="city-label" id="city-label">Загрузка...</div>
  <div class="update-time" id="update-time"></div>
</div>
<div class="sensors" id="sensors-list"></div>
<div class="footer">AQI Platform · Данные обновляются каждую минуту</div>
<script>
const BASE = window.location.origin;
function fetchData(){
  fetch(BASE+'/widget/data')
    .then(r=>r.json())
    .then(d=>{
      const aqiEl=document.getElementById('city-aqi');
      aqiEl.textContent=d.city_aqi||'—';
      aqiEl.style.color=d.city_color||'#333';
      document.getElementById('city-label').textContent=d.city_label||'';
      const t=new Date(d.updated_at);
      document.getElementById('update-time').textContent=
        'Обновлено: '+t.toLocaleTimeString('ru-RU',{hour:'2-digit',minute:'2-digit'});
      const list=document.getElementById('sensors-list');
      list.innerHTML='';
      (d.sensors||[]).forEach(s=>{
        const card=document.createElement('div');
        card.className='sensor-card'+(s.is_online?'':' offline');
        card.innerHTML=
          '<div class="dot" style="background:'+s.color+'"></div>'+
          '<div style="flex:1"><div class="sensor-name">'+s.name+'</div>'+
          '<div class="sensor-pm">'+(s.pm25!=null?'PM2.5: '+s.pm25.toFixed(3)+' мг/м³':'нет данных')+'</div></div>'+
          '<div><div class="sensor-aqi" style="color:'+s.color+'">'+s.aqi+'</div>'+
          '<div class="sensor-pm">'+s.label+'</div></div>';
        list.appendChild(card);
      });
    })
    .catch(()=>{
      document.getElementById('city-label').textContent='Нет соединения';
    });
}
fetchData();
setInterval(fetchData,60000);
</script>
</body>
</html>`
