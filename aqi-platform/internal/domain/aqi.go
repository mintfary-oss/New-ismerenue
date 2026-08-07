package domain

import "time"

// AQICategory — категория качества воздуха (российская + международная методология).
type AQICategory string

const (
	AQICategoryGood      AQICategory = "good"       // Хорошее (0-50)
	AQICategoryModerate  AQICategory = "moderate"   // Умеренное (51-100)
	AQICategoryUnhealthy AQICategory = "unhealthy"  // Нездоровое для чувствительных (101-150)
	AQICategoryBad       AQICategory = "bad"        // Нездоровое (151-200)
	AQICategoryVeryBad   AQICategory = "very_bad"   // Очень нездоровое (201-300)
	AQICategoryHazardous AQICategory = "hazardous"  // Опасное (301+)
)

// AQIBreakpoint — диапазон для расчёта AQI по одному веществу (US EPA метод).
type AQIBreakpoint struct {
	ConcLo float64 // нижняя граница концентрации
	ConcHi float64 // верхняя граница концентрации
	AQILo  float64 // нижняя граница AQI
	AQIHi  float64 // верхняя граница AQI
}

// pm25Breakpoints — контрольные точки для PM2.5 (мкг/м³, 24-часовое среднее).
var pm25Breakpoints = []AQIBreakpoint{
	{0.0, 12.0, 0, 50},
	{12.1, 35.4, 51, 100},
	{35.5, 55.4, 101, 150},
	{55.5, 150.4, 151, 200},
	{150.5, 250.4, 201, 300},
	{250.5, 350.4, 301, 400},
	{350.5, 500.4, 401, 500},
}

// no2Breakpoints — контрольные точки для NO2 (ppb, 1-часовое среднее).
// Перевод из мг/м³: NO2 1 мг/м³ ≈ 532 ppb при 20°C.
var no2Breakpoints = []AQIBreakpoint{
	{0, 53, 0, 50},
	{54, 100, 51, 100},
	{101, 360, 101, 150},
	{361, 649, 151, 200},
	{650, 1249, 201, 300},
	{1250, 1649, 301, 400},
	{1650, 2049, 401, 500},
}

// CalcAQIforPM25 рассчитывает субиндекс AQI для PM2.5 (мг/м³ → AQI).
// Входное значение в мг/м³ конвертируется в мкг/м³ (×1000).
func CalcAQIforPM25(mgPerM3 float64) int {
	ugPerM3 := mgPerM3 * 1000
	return calcSubIndex(ugPerM3, pm25Breakpoints)
}

// CalcAQIforNO2 рассчитывает субиндекс AQI для NO2 (мг/м³ → AQI).
func CalcAQIforNO2(mgPerM3 float64) int {
	// 1 мг/м³ NO2 ≈ 532 ppb (при 20°C, 1 атм)
	ppb := mgPerM3 * 532
	return calcSubIndex(ppb, no2Breakpoints)
}

// CalcOverallAQI возвращает итоговый AQI как максимум субиндексов.
func CalcOverallAQI(m *Measurement) int {
	max := 0
	if m.PM25 != nil {
		if v := CalcAQIforPM25(*m.PM25); v > max {
			max = v
		}
	}
	if m.NO2 != nil {
		if v := CalcAQIforNO2(*m.NO2); v > max {
			max = v
		}
	}
	return max
}

// AQIToCategory переводит числовой AQI в категорию.
func AQIToCategory(aqi int) AQICategory {
	switch {
	case aqi <= 50:
		return AQICategoryGood
	case aqi <= 100:
		return AQICategoryModerate
	case aqi <= 150:
		return AQICategoryUnhealthy
	case aqi <= 200:
		return AQICategoryBad
	case aqi <= 300:
		return AQICategoryVeryBad
	default:
		return AQICategoryHazardous
	}
}

// AQICategoryLabel возвращает русскоязычное название категории.
func AQICategoryLabel(c AQICategory) string {
	switch c {
	case AQICategoryGood:
		return "Хорошее"
	case AQICategoryModerate:
		return "Умеренное"
	case AQICategoryUnhealthy:
		return "Нездоровое для чувствительных групп"
	case AQICategoryBad:
		return "Нездоровое"
	case AQICategoryVeryBad:
		return "Очень нездоровое"
	case AQICategoryHazardous:
		return "Опасное"
	default:
		return "Неизвестно"
	}
}

// AQICategoryColor возвращает hex-цвет для категории (для виджета).
func AQICategoryColor(c AQICategory) string {
	switch c {
	case AQICategoryGood:
		return "#00e400"
	case AQICategoryModerate:
		return "#ffff00"
	case AQICategoryUnhealthy:
		return "#ff7e00"
	case AQICategoryBad:
		return "#ff0000"
	case AQICategoryVeryBad:
		return "#8f3f97"
	case AQICategoryHazardous:
		return "#7e0023"
	default:
		return "#cccccc"
	}
}

// calcSubIndex вычисляет субиндекс AQI по формуле линейной интерполяции.
// Формула: AQI = ((AQIHi - AQILo) / (ConcHi - ConcLo)) * (Conc - ConcLo) + AQILo
func calcSubIndex(conc float64, breakpoints []AQIBreakpoint) int {
	for _, bp := range breakpoints {
		if conc >= bp.ConcLo && conc <= bp.ConcHi {
			aqi := ((bp.AQIHi-bp.AQILo)/(bp.ConcHi-bp.ConcLo))*(conc-bp.ConcLo) + bp.AQILo
			return int(aqi + 0.5) // округление
		}
	}
	// Выше верхней границы → максимум 500
	return 500
}

// ForecastPoint — контрольная точка для прогноза (может не совпадать с датчиком).
type ForecastPoint struct {
	ID       string  // уникальный идентификатор точки
	Name     string  // название (например "Площадь Советов")
	Lat      float64
	Lng      float64
	District string // название района
}

// Forecast — прогноз качества воздуха для одной точки.
type Forecast struct {
	Time          time.Time
	PointID       string
	Lat           float64
	Lng           float64
	HorizonHours  int
	AQI           int
	AQICategory   AQICategory
	NO2Forecast   *float64
	PM25Forecast  *float64
	SO2Forecast   *float64
	ModelVersion  string
	CreatedAt     time.Time
}

// CityForecast — агрегированный прогноз по городу и районам.
type CityForecast struct {
	Time          time.Time
	CityAQI       int
	CityCategory  AQICategory
	Districts     []DistrictForecast
	Points        []Forecast
}

// DistrictForecast — прогноз по одному административному району.
type DistrictForecast struct {
	DistrictID   string
	DistrictName string
	AQI          int
	AQICategory  AQICategory
}
