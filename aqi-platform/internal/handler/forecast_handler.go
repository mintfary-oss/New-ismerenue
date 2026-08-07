package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mintfary/aqi-platform/internal/domain"
	"github.com/mintfary/aqi-platform/internal/service"
)

// ForecastHandler реализует HTTP-обработчики прогнозирования качества воздуха.
type ForecastHandler struct {
	svc    *service.ForecastService
	logger *slog.Logger
}

// NewForecastHandler создаёт обработчик прогнозов.
func NewForecastHandler(svc *service.ForecastService, logger *slog.Logger) *ForecastHandler {
	return &ForecastHandler{svc: svc, logger: logger}
}

// Points godoc
// @Summary     Список точек мониторинга
// @Description Возвращает 4 фиксированные точки мониторинга Кемерово.
// @Tags        forecast
// @Success     200 {object} map[string]any
// @Router      /forecast/points [get]
func (h *ForecastHandler) Points(w http.ResponseWriter, r *http.Request) {
	points := h.svc.Points()

	type pointResponse struct {
		ID       string  `json:"id"`
		Name     string  `json:"name"`
		Lat      float64 `json:"lat"`
		Lng      float64 `json:"lng"`
		District string  `json:"district"`
	}

	resp := make([]pointResponse, 0, len(points))
	for _, p := range points {
		resp = append(resp, pointResponse{
			ID:       p.ID,
			Name:     p.Name,
			Lat:      p.Lat,
			Lng:      p.Lng,
			District: p.District,
		})
	}

	ok(w, map[string]any{
		"points": resp,
		"count":  len(resp),
	})
}

// Current godoc
// @Summary     Текущий прогноз по всем точкам
// @Description Возвращает последний рассчитанный прогноз (horizon=0) для всех точек.
// @Tags        forecast
// @Success     200 {object} map[string]any
// @Router      /forecast/current [get]
func (h *ForecastHandler) Current(w http.ResponseWriter, r *http.Request) {
	forecasts, err := h.svc.Current(r.Context())
	if err != nil {
		if isNotFound(err) {
			ok(w, map[string]any{"forecasts": []any{}, "count": 0, "message": "прогнозы ещё не рассчитаны"})
			return
		}
		handleError(w, h.logger, err)
		return
	}

	ok(w, map[string]any{
		"forecasts":  forecastsToResponse(forecasts),
		"count":      len(forecasts),
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// CityAverage godoc
// @Summary     Средний AQI по городу
// @Description Агрегированный прогноз: средний AQI по городу и по каждому району.
// @Tags        forecast
// @Success     200 {object} map[string]any
// @Router      /forecast/city-average [get]
func (h *ForecastHandler) CityAverage(w http.ResponseWriter, r *http.Request) {
	city, err := h.svc.CityAverage(r.Context())
	if err != nil {
		if isNotFound(err) {
			ok(w, map[string]any{
				"city_aqi":      0,
				"city_category": domain.AQICategoryGood,
				"city_label":    "Нет данных",
				"city_color":    "#cccccc",
				"districts":     []any{},
				"message":       "прогнозы ещё не рассчитаны",
			})
			return
		}
		handleError(w, h.logger, err)
		return
	}

	type districtResp struct {
		Name     string             `json:"name"`
		AQI      int                `json:"aqi"`
		Category domain.AQICategory `json:"category"`
		Label    string             `json:"label"`
		Color    string             `json:"color"`
	}
	districts := make([]districtResp, 0, len(city.Districts))
	for _, d := range city.Districts {
		districts = append(districts, districtResp{
			Name:     d.DistrictName,
			AQI:      d.AQI,
			Category: d.AQICategory,
			Label:    domain.AQICategoryLabel(d.AQICategory),
			Color:    domain.AQICategoryColor(d.AQICategory),
		})
	}

	ok(w, map[string]any{
		"time":          city.Time.Format(time.RFC3339),
		"city_aqi":      city.CityAQI,
		"city_category": city.CityCategory,
		"city_label":    domain.AQICategoryLabel(city.CityCategory),
		"city_color":    domain.AQICategoryColor(city.CityCategory),
		"districts":     districts,
		"points_count":  len(city.Points),
		"updated_at":    time.Now().UTC().Format(time.RFC3339),
	})
}

// ByPoint godoc
// @Summary     Прогноз для конкретной точки
// @Description Возвращает все горизонты прогноза (0h, 1h, 2h, 3h, 6h) для точки.
// @Tags        forecast
// @Param       point_id path string true "ID точки (kirov | zavodsky | rudnichny | leninsky)"
// @Success     200 {object} map[string]any
// @Router      /forecast/{point_id} [get]
func (h *ForecastHandler) ByPoint(w http.ResponseWriter, r *http.Request) {
	pointID := chi.URLParam(r, "point_id")
	if pointID == "" {
		writeError(w, http.StatusBadRequest, "point_id обязателен")
		return
	}

	forecasts, err := h.svc.ByPoint(r.Context(), pointID)
	if err != nil {
		handleError(w, h.logger, err)
		return
	}

	point := domain.PointByID(pointID)
	resp := map[string]any{
		"forecasts": forecastsToResponse(forecasts),
		"count":     len(forecasts),
	}
	if point != nil {
		resp["point"] = map[string]any{
			"id":       point.ID,
			"name":     point.Name,
			"lat":      point.Lat,
			"lng":      point.Lng,
			"district": point.District,
		}
	}
	if len(forecasts) == 0 {
		resp["message"] = "прогнозы для точки ещё не рассчитаны"
	}

	ok(w, resp)
}

// ByDistrict godoc
// @Summary     Прогноз по административному району
// @Description Возвращает текущий прогноз для всех точек в районе.
// @Tags        forecast
// @Param       id path string true "Название района (Кировский | Заводский | Рудничный | Ленинский)"
// @Success     200 {object} map[string]any
// @Router      /forecast/district/{id} [get]
func (h *ForecastHandler) ByDistrict(w http.ResponseWriter, r *http.Request) {
	district := chi.URLParam(r, "id")
	if district == "" {
		writeError(w, http.StatusBadRequest, "id района обязателен")
		return
	}

	forecasts, err := h.svc.ByDistrict(r.Context(), district)
	if err != nil {
		if isNotFound(err) {
			writeError(w, http.StatusNotFound, "район не найден: "+district)
			return
		}
		handleError(w, h.logger, err)
		return
	}

	ok(w, map[string]any{
		"district":  district,
		"forecasts": forecastsToResponse(forecasts),
		"count":     len(forecasts),
	})
}

// ── Вспомогательные функции ────────────────────────────────────────────────

// forecastToResponse преобразует Forecast в JSON-представление.
func forecastToResponse(f domain.Forecast) map[string]any {
	resp := map[string]any{
		"time":          f.Time.Format(time.RFC3339),
		"point_id":      f.PointID,
		"lat":           f.Lat,
		"lng":           f.Lng,
		"horizon_hours": f.HorizonHours,
		"aqi":           f.AQI,
		"aqi_category":  f.AQICategory,
		"aqi_label":     domain.AQICategoryLabel(f.AQICategory),
		"aqi_color":     domain.AQICategoryColor(f.AQICategory),
		"model_version": f.ModelVersion,
		"created_at":    f.CreatedAt.Format(time.RFC3339),
	}
	if f.PM25Forecast != nil {
		resp["pm25_forecast"] = *f.PM25Forecast
	}
	if f.NO2Forecast != nil {
		resp["no2_forecast"] = *f.NO2Forecast
	}
	if f.SO2Forecast != nil {
		resp["so2_forecast"] = *f.SO2Forecast
	}
	return resp
}

func forecastsToResponse(forecasts []domain.Forecast) []map[string]any {
	result := make([]map[string]any, 0, len(forecasts))
	for _, f := range forecasts {
		result = append(result, forecastToResponse(f))
	}
	return result
}

// isNotFound проверяет ошибку "не найдено".
func isNotFound(err error) bool {
	return domain.IsNotFound(err)
}

// Compile-time check: context import used in handler signature.
var _ context.Context = (context.Context)(nil)
