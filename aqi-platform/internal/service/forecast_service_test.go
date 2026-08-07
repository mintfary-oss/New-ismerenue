// Package service — тесты прогнозного движка.
//
// Покрываем:
//   - computeEWMA: сглаживание и тренд
//   - haversine: геодезическое расстояние
//   - buildHorizons: построение горизонтов
//   - computePointForecast: IDW-интерполяция
//   - ForecastService.Run: полный цикл (минимум датчиков / нормальный путь)
package service

import (
	"context"
	"log/slog"
	"math"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mintfary/aqi-platform/internal/config"
	"github.com/mintfary/aqi-platform/internal/domain"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func ptr(v float64) *float64 { return &v }

func testForecastCfg() config.ForecastConfig {
	return config.ForecastConfig{
		HorizonHours:          6,
		EWMAAlpha:             0.3,
		IDWPower:              2.0,
		MinSensorsForForecast: 1,
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// ── mockForecastMeasurementReader ────────────────────────────────────────────

type mockForecastMeasurementReader struct {
	latest []domain.LatestMeasurement
	list   []domain.Measurement
}

func (m *mockForecastMeasurementReader) Latest(_ context.Context) ([]domain.LatestMeasurement, error) {
	return m.latest, nil
}

func (m *mockForecastMeasurementReader) List(_ context.Context, _ domain.MeasurementFilter) ([]domain.Measurement, error) {
	return m.list, nil
}

// ── mockForecastWriter ────────────────────────────────────────────────────────

type mockForecastWriter struct {
	saved []domain.Forecast
}

func (m *mockForecastWriter) InsertBatch(_ context.Context, forecasts []domain.Forecast) error {
	m.saved = append(m.saved, forecasts...)
	return nil
}

func (m *mockForecastWriter) Latest(_ context.Context) ([]domain.Forecast, error) {
	return m.saved, nil
}

func (m *mockForecastWriter) LatestByPoint(_ context.Context, pointID string) ([]domain.Forecast, error) {
	var out []domain.Forecast
	for _, f := range m.saved {
		if f.PointID == pointID {
			out = append(out, f)
		}
	}
	return out, nil
}

func (m *mockForecastWriter) ByDistrict(_ context.Context, district string) ([]domain.Forecast, error) {
	var out []domain.Forecast
	for _, f := range m.saved {
		if p := domain.PointByID(f.PointID); p != nil && p.District == district {
			out = append(out, f)
		}
	}
	return out, nil
}

func (m *mockForecastWriter) CityAverage(_ context.Context) (*domain.CityForecast, error) {
	if len(m.saved) == 0 {
		return nil, domain.ErrNotFound
	}
	total := 0
	for _, f := range m.saved {
		total += f.AQI
	}
	avg := total / len(m.saved)
	return &domain.CityForecast{
		Time:         time.Now(),
		CityAQI:      avg,
		CityCategory: domain.AQIToCategory(avg),
	}, nil
}

// ── computeEWMA ──────────────────────────────────────────────────────────────

func TestComputeEWMA_SinglePoint(t *testing.T) {
	// С одной точкой EWMA = сама точка, тренд = 0.
	measurements := []domain.Measurement{
		{Time: time.Now(), PM25: ptr(10.0), NO2: ptr(0.05)},
	}
	pm, no2 := computeEWMA(measurements, 0.3)
	if pm.last == nil {
		t.Fatal("pm.last не должен быть nil")
	}
	if math.Abs(*pm.last-10.0) > 0.01 {
		t.Errorf("pm.last: ожидали 10.0, получили %v", *pm.last)
	}
	if pm.trendPerHour != 0 {
		t.Errorf("trendPerHour при одной точке должен быть 0, получили %v", pm.trendPerHour)
	}
	if no2.last == nil {
		t.Fatal("no2.last не должен быть nil")
	}
}

func TestComputeEWMA_TwoPoints_PositiveTrend(t *testing.T) {
	// DESC: новое [0] = 20, старое [1] = 10 → EWMA растёт.
	now := time.Now().UTC()
	measurements := []domain.Measurement{
		{Time: now, PM25: ptr(20.0)},           // новое (i=0 в DESC)
		{Time: now.Add(-1 * time.Hour), PM25: ptr(10.0)}, // старое (i=1 в DESC)
	}
	pm, _ := computeEWMA(measurements, 0.3)
	if pm.last == nil {
		t.Fatal("pm.last не должен быть nil")
	}
	// EWMA при α=0.3: начинаем с 10 (старый), затем α*20+(1-α)*10 = 6+7 = 13
	expected := 0.3*20.0 + 0.7*10.0
	if math.Abs(*pm.last-expected) > 0.001 {
		t.Errorf("pm.last: ожидали %.4f, получили %.4f", expected, *pm.last)
	}
}

func TestComputeEWMA_NilValues(t *testing.T) {
	// Если все PM25 == nil — last должен быть nil.
	measurements := []domain.Measurement{
		{Time: time.Now(), NO2: ptr(0.1)},
		{Time: time.Now().Add(-time.Hour), NO2: ptr(0.05)},
	}
	pm, no2 := computeEWMA(measurements, 0.3)
	if pm.last != nil {
		t.Errorf("pm.last должен быть nil при отсутствии PM25-данных")
	}
	if no2.last == nil {
		t.Errorf("no2.last не должен быть nil")
	}
}

func TestComputeEWMA_InvalidAlpha(t *testing.T) {
	// computeEWMA принимает alpha как параметр — проверяем граничные значения.
	measurements := []domain.Measurement{
		{Time: time.Now(), PM25: ptr(10.0)},
	}
	// alpha=0 → EWMA некорректен, но не должен паниковать.
	pm, _ := computeEWMA(measurements, 0.0)
	if pm.last == nil {
		t.Fatal("pm.last не должен быть nil при alpha=0")
	}
}

// ── haversine ────────────────────────────────────────────────────────────────

func TestHaversine_SamePoint(t *testing.T) {
	d := haversine(55.3557, 86.0867, 55.3557, 86.0867)
	if d != 0 {
		t.Errorf("расстояние до самой себя должно быть 0, получили %v", d)
	}
}

func TestHaversine_KnownDistance(t *testing.T) {
	// Кемерово → Новосибирск: ~230 км (приблизительно).
	kemerovo := [2]float64{55.3557, 86.0867}
	novosibirsk := [2]float64{54.9885, 82.9123}
	d := haversine(kemerovo[0], kemerovo[1], novosibirsk[0], novosibirsk[1])
	if d < 200 || d > 270 {
		t.Errorf("расстояние Кемерово-Новосибирск ожидали ~230 км, получили %.1f км", d)
	}
}

func TestHaversine_Symmetry(t *testing.T) {
	d1 := haversine(55.0, 86.0, 54.0, 85.0)
	d2 := haversine(54.0, 85.0, 55.0, 86.0)
	if math.Abs(d1-d2) > 0.001 {
		t.Errorf("haversine должна быть симметричной: d1=%.4f d2=%.4f", d1, d2)
	}
}

// ── buildHorizons ────────────────────────────────────────────────────────────

func TestBuildHorizons_Default6(t *testing.T) {
	h := buildHorizons(6)
	// Ожидаем: [0, 1, 2, 3, 6]
	want := []int{0, 1, 2, 3, 6}
	if len(h) != len(want) {
		t.Fatalf("ожидали %v, получили %v", want, h)
	}
	for i, v := range want {
		if h[i] != v {
			t.Errorf("[%d]: ожидали %d, получили %d", i, v, h[i])
		}
	}
}

func TestBuildHorizons_24h(t *testing.T) {
	h := buildHorizons(24)
	// Ожидаем: [0, 1, 2, 3, 6, 12, 24]
	want := []int{0, 1, 2, 3, 6, 12, 24}
	if len(h) != len(want) {
		t.Fatalf("ожидали %v, получили %v", want, h)
	}
}

func TestBuildHorizons_Zero(t *testing.T) {
	h := buildHorizons(0)
	// 0 → maxH принудительно = 6
	if len(h) == 0 {
		t.Fatal("buildHorizons(0) не должна вернуть пустой список")
	}
	if h[0] != 0 {
		t.Errorf("первый горизонт должен быть 0, получили %d", h[0])
	}
}

func TestBuildHorizons_NonStandard(t *testing.T) {
	h := buildHorizons(8)
	// Ожидаем: [0, 1, 2, 3, 6, 8] — 8 добавляется явно
	last := h[len(h)-1]
	if last != 8 {
		t.Errorf("последний горизонт должен быть 8, получили %d", last)
	}
}

// ── computePointForecast ─────────────────────────────────────────────────────

func TestComputePointForecast_SingleSensor(t *testing.T) {
	svc := &ForecastService{cfg: testForecastCfg(), logger: testLogger()}

	pm25 := 15.0
	sensors := map[string]sensorValue{
		"s1": {pm25: &pm25, lat: 55.35, lng: 86.08},
	}
	point := domain.ForecastPoint{ID: "center", Lat: 55.36, Lng: 86.09, District: "Центральный"}
	fc, err := svc.computePointForecast(sensors, point, 0, time.Now().UTC())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc.PM25Forecast == nil {
		t.Fatal("PM25Forecast не должен быть nil")
	}
	if math.Abs(*fc.PM25Forecast-15.0) > 0.01 {
		t.Errorf("PM25Forecast: ожидали 15.0, получили %.4f", *fc.PM25Forecast)
	}
	if fc.AQI <= 0 {
		t.Errorf("AQI должен быть > 0 при PM25=15 мг/м³")
	}
}

func TestComputePointForecast_NoSensors(t *testing.T) {
	svc := &ForecastService{cfg: testForecastCfg(), logger: testLogger()}
	_, err := svc.computePointForecast(map[string]sensorValue{}, domain.KemerovoPoints[0], 0, time.Now())
	if err == nil {
		t.Fatal("ожидали ошибку при пустом наборе датчиков")
	}
}

func TestComputePointForecast_PositiveTrend(t *testing.T) {
	svc := &ForecastService{cfg: testForecastCfg(), logger: testLogger()}

	pm25 := 10.0
	trendPM := 5.0 // +5 мкг/м³ в час
	sensors := map[string]sensorValue{
		"s1": {pm25: &pm25, trendPM: trendPM, lat: 55.35, lng: 86.08},
	}
	point := domain.KemerovoPoints[0]

	// Горизонт 2ч: ожидаем pm25 + trendPM*2 = 20
	fc, err := svc.computePointForecast(sensors, point, 2, time.Now().UTC())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc.PM25Forecast == nil {
		t.Fatal("PM25Forecast не должен быть nil")
	}
	expected := pm25 + trendPM*2
	if math.Abs(*fc.PM25Forecast-expected) > 0.01 {
		t.Errorf("ожидали %.1f, получили %.4f", expected, *fc.PM25Forecast)
	}
}

func TestComputePointForecast_NegativeTrendClampedToZero(t *testing.T) {
	svc := &ForecastService{cfg: testForecastCfg(), logger: testLogger()}

	pm25 := 5.0
	trendPM := -10.0 // уйдёт в отрицательное — должно быть зажато в 0
	sensors := map[string]sensorValue{
		"s1": {pm25: &pm25, trendPM: trendPM, lat: 55.35, lng: 86.08},
	}
	point := domain.KemerovoPoints[0]

	fc, err := svc.computePointForecast(sensors, point, 2, time.Now().UTC())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc.PM25Forecast == nil {
		t.Fatal("PM25Forecast не должен быть nil")
	}
	if *fc.PM25Forecast < 0 {
		t.Errorf("PM25Forecast не должен быть отрицательным, получили %v", *fc.PM25Forecast)
	}
}

// ── ForecastService.Run ───────────────────────────────────────────────────────

func TestForecastService_Run_TooFewSensors(t *testing.T) {
	reader := &mockForecastMeasurementReader{latest: []domain.LatestMeasurement{}}
	writer := &mockForecastWriter{}

	cfg := testForecastCfg()
	cfg.MinSensorsForForecast = 2 // требуем 2, передаём 0

	svc := NewForecastService(reader, writer, cfg, testLogger())
	err := svc.Run(context.Background())
	if err != nil {
		t.Errorf("Run при нехватке датчиков должен завершаться без ошибки (warn), получили: %v", err)
	}
	if len(writer.saved) != 0 {
		t.Errorf("не должно быть сохранённых прогнозов при нехватке датчиков")
	}
}

func TestForecastService_Run_NormalPath(t *testing.T) {
	sensorID := uuid.New()
	pm25 := 12.5
	no2 := 0.04

	reader := &mockForecastMeasurementReader{
		latest: []domain.LatestMeasurement{
			{
				Sensor:      domain.Sensor{ID: sensorID, Lat: 55.35, Lng: 86.08, IsActive: true},
				Measurement: domain.Measurement{SensorID: sensorID, PM25: &pm25, NO2: &no2},
			},
		},
		list: []domain.Measurement{}, // нет истории → тренд = 0
	}
	writer := &mockForecastWriter{}

	cfg := testForecastCfg()
	cfg.MinSensorsForForecast = 1

	svc := NewForecastService(reader, writer, cfg, testLogger())
	err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("Run вернул ошибку: %v", err)
	}

	// Ожидаем прогнозы для всех точек × горизонтов.
	wantPoints := len(domain.KemerovoPoints)
	wantHorizons := len(buildHorizons(cfg.HorizonHours))
	wantTotal := wantPoints * wantHorizons

	if len(writer.saved) != wantTotal {
		t.Errorf("ожидали %d прогнозов, получили %d", wantTotal, len(writer.saved))
	}

	// Проверяем что AQI корректный.
	for _, fc := range writer.saved {
		if fc.AQI < 0 {
			t.Errorf("AQI не должен быть отрицательным: %v", fc.AQI)
		}
		if fc.PointID == "" {
			t.Error("PointID не должен быть пустым")
		}
	}
}

func TestForecastService_Points(t *testing.T) {
	svc := NewForecastService(nil, nil, testForecastCfg(), testLogger())
	pts := svc.Points()
	if len(pts) == 0 {
		t.Fatal("ожидали точки Кемерово")
	}
	for _, p := range pts {
		if p.ID == "" {
			t.Error("ID точки не должен быть пустым")
		}
		if p.Lat == 0 || p.Lng == 0 {
			t.Errorf("точка %s: некорректные координаты Lat=%v Lng=%v", p.ID, p.Lat, p.Lng)
		}
	}
}
