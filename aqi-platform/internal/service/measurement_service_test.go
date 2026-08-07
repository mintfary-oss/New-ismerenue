// Package service — тесты MeasurementService.
//
// Покрываем:
//   - Ingest: нормальный путь, время в будущем, нулевое время
//   - IngestBatch: пустой пакет, превышение лимита, нормальный путь
//   - List: raw / агрегация
//   - AQI-расчёт через CalcOverallAQI (domain-уровень)
package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mintfary/aqi-platform/internal/domain"
)

// ── mockMeasurementRepository ─────────────────────────────────────────────────

type mockMeasurementRepository struct {
	mu      sync.Mutex
	records []domain.Measurement
}

func (r *mockMeasurementRepository) Insert(_ context.Context, in domain.MeasurementInput) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, domain.Measurement{
		Time:     in.Time,
		SensorID: in.SensorID,
		PM25:     in.PM25,
		NO2:      in.NO2,
	})
	return nil
}

func (r *mockMeasurementRepository) InsertBatch(_ context.Context, items []domain.MeasurementInput) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, in := range items {
		r.records = append(r.records, domain.Measurement{
			Time:     in.Time,
			SensorID: in.SensorID,
			PM25:     in.PM25,
			NO2:      in.NO2,
		})
	}
	return nil
}

func (r *mockMeasurementRepository) List(_ context.Context, _ domain.MeasurementFilter) ([]domain.Measurement, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]domain.Measurement, len(r.records))
	copy(cp, r.records)
	return cp, nil
}

func (r *mockMeasurementRepository) Aggregate(_ context.Context, _ domain.MeasurementFilter, _ string) ([]domain.AggregatedMeasurement, error) {
	return nil, nil
}

func (r *mockMeasurementRepository) Latest(_ context.Context) ([]domain.LatestMeasurement, error) {
	return nil, nil
}

func (r *mockMeasurementRepository) LatestBySensor(_ context.Context, _ uuid.UUID) (*domain.Measurement, error) {
	return nil, nil
}

// ── mockSensorLastSeenUpdater ────────────────────────────────────────────────

type mockSensorLastSeenUpdater struct {
	mu      sync.Mutex
	updates map[uuid.UUID]time.Time
}

func newMockSensorLastSeenUpdater() *mockSensorLastSeenUpdater {
	return &mockSensorLastSeenUpdater{updates: make(map[uuid.UUID]time.Time)}
}

func (u *mockSensorLastSeenUpdater) UpdateLastSeen(_ context.Context, id uuid.UUID, t time.Time) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.updates[id] = t
	return nil
}

func (u *mockSensorLastSeenUpdater) lastSeen(id uuid.UUID) (time.Time, bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	t, ok := u.updates[id]
	return t, ok
}

// ── MeasurementService.Ingest ─────────────────────────────────────────────────

func TestIngest_NormalPath(t *testing.T) {
	repo := &mockMeasurementRepository{}
	sensorUpd := newMockSensorLastSeenUpdater()
	svc := NewMeasurementService(repo, sensorUpd, testLogger())

	id := uuid.New()
	pm25 := 5.0
	in := domain.MeasurementInput{
		SensorID: id,
		Time:     time.Now().UTC().Add(-time.Minute),
		PM25:     &pm25,
	}

	err := svc.Ingest(context.Background(), in)
	if err != nil {
		t.Fatalf("Ingest вернул ошибку: %v", err)
	}

	repo.mu.Lock()
	n := len(repo.records)
	repo.mu.Unlock()
	if n != 1 {
		t.Errorf("ожидали 1 запись в репозитории, получили %d", n)
	}
}

func TestIngest_FutureTime(t *testing.T) {
	repo := &mockMeasurementRepository{}
	svc := NewMeasurementService(repo, newMockSensorLastSeenUpdater(), testLogger())

	id := uuid.New()
	in := domain.MeasurementInput{
		SensorID: id,
		Time:     time.Now().UTC().Add(10 * time.Minute), // далеко в будущем
	}

	err := svc.Ingest(context.Background(), in)
	if err == nil {
		t.Fatal("ожидали ошибку при времени в будущем")
	}
}

func TestIngest_ZeroTime_UsesNow(t *testing.T) {
	repo := &mockMeasurementRepository{}
	svc := NewMeasurementService(repo, newMockSensorLastSeenUpdater(), testLogger())

	id := uuid.New()
	in := domain.MeasurementInput{SensorID: id} // Time == zero

	err := svc.Ingest(context.Background(), in)
	if err != nil {
		t.Fatalf("Ingest с нулевым временем вернул ошибку: %v", err)
	}

	repo.mu.Lock()
	if len(repo.records) == 0 {
		repo.mu.Unlock()
		t.Fatal("запись не сохранена")
	}
	saved := repo.records[0]
	repo.mu.Unlock()

	if saved.Time.IsZero() {
		t.Error("время должно быть проставлено автоматически")
	}
}

func TestIngest_BoundaryFuture_5Min(t *testing.T) {
	// Время ровно на границе +5 мин — должно пройти (After → строго больше).
	repo := &mockMeasurementRepository{}
	svc := NewMeasurementService(repo, newMockSensorLastSeenUpdater(), testLogger())

	id := uuid.New()
	in := domain.MeasurementInput{
		SensorID: id,
		Time:     time.Now().UTC().Add(4 * time.Minute), // ≤ 5 мин → допустимо
	}

	err := svc.Ingest(context.Background(), in)
	if err != nil {
		t.Errorf("время в пределах допуска (+4 мин) должно приниматься, ошибка: %v", err)
	}
}

// ── MeasurementService.IngestBatch ───────────────────────────────────────────

func TestIngestBatch_EmptySlice(t *testing.T) {
	svc := NewMeasurementService(&mockMeasurementRepository{}, newMockSensorLastSeenUpdater(), testLogger())
	err := svc.IngestBatch(context.Background(), nil)
	if err != nil {
		t.Errorf("IngestBatch с nil не должен возвращать ошибку: %v", err)
	}

	err = svc.IngestBatch(context.Background(), []domain.MeasurementInput{})
	if err != nil {
		t.Errorf("IngestBatch с пустым slice не должен возвращать ошибку: %v", err)
	}
}

func TestIngestBatch_ExceedsLimit(t *testing.T) {
	svc := NewMeasurementService(&mockMeasurementRepository{}, newMockSensorLastSeenUpdater(), testLogger())

	items := make([]domain.MeasurementInput, 1001)
	for i := range items {
		items[i] = domain.MeasurementInput{SensorID: uuid.New(), Time: time.Now().UTC()}
	}

	err := svc.IngestBatch(context.Background(), items)
	if err == nil {
		t.Fatal("ожидали ошибку при превышении лимита 1000 записей")
	}
}

func TestIngestBatch_NormalPath(t *testing.T) {
	repo := &mockMeasurementRepository{}
	svc := NewMeasurementService(repo, newMockSensorLastSeenUpdater(), testLogger())

	id1 := uuid.New()
	id2 := uuid.New()
	items := []domain.MeasurementInput{
		{SensorID: id1, Time: time.Now().UTC().Add(-2 * time.Minute)},
		{SensorID: id1, Time: time.Now().UTC().Add(-1 * time.Minute)},
		{SensorID: id2, Time: time.Now().UTC().Add(-30 * time.Second)},
	}

	err := svc.IngestBatch(context.Background(), items)
	if err != nil {
		t.Fatalf("IngestBatch вернул ошибку: %v", err)
	}

	repo.mu.Lock()
	n := len(repo.records)
	repo.mu.Unlock()
	if n != 3 {
		t.Errorf("ожидали 3 записи, получили %d", n)
	}
}

func TestIngestBatch_ZeroTimeFilled(t *testing.T) {
	repo := &mockMeasurementRepository{}
	svc := NewMeasurementService(repo, newMockSensorLastSeenUpdater(), testLogger())

	items := []domain.MeasurementInput{
		{SensorID: uuid.New()}, // Time == zero
	}

	err := svc.IngestBatch(context.Background(), items)
	if err != nil {
		t.Fatalf("IngestBatch вернул ошибку: %v", err)
	}

	repo.mu.Lock()
	saved := repo.records[0]
	repo.mu.Unlock()

	if saved.Time.IsZero() {
		t.Error("нулевое время должно быть заполнено текущим")
	}
}

// ── MeasurementService.List ───────────────────────────────────────────────────

func TestList_RawPeriod(t *testing.T) {
	repo := &mockMeasurementRepository{}
	svc := NewMeasurementService(repo, newMockSensorLastSeenUpdater(), testLogger())

	// Предварительно добавляем запись.
	pm25 := 10.0
	mustNotError(repo.Insert(context.Background(), domain.MeasurementInput{
		SensorID: uuid.New(), Time: time.Now().UTC(), PM25: &pm25,
	}))

	f := domain.MeasurementFilter{
		From:   time.Now().UTC().Add(-time.Hour),
		To:     time.Now().UTC(),
		Period: "raw",
	}
	result, err := svc.List(context.Background(), f)
	if err != nil {
		t.Fatalf("List(raw) вернул ошибку: %v", err)
	}
	if result == nil {
		t.Error("List(raw) не должен вернуть nil")
	}
}

// ── AQI domain-функции ────────────────────────────────────────────────────────

func TestCalcAQI_GoodAir(t *testing.T) {
	// PM2.5 = 0.005 мг/м³ = 5 мкг/м³ → AQI ≈ 20 (Good)
	aqi := domain.CalcAQIforPM25(0.005)
	if aqi > 50 {
		t.Errorf("PM2.5=5 мкг/м³ должно давать AQI≤50 (Good), получили %d", aqi)
	}
}

func TestCalcAQI_ModeratePM25(t *testing.T) {
	// PM2.5 = 0.025 мг/м³ = 25 мкг/м³ → AQI 51–100 (Moderate)
	aqi := domain.CalcAQIforPM25(0.025)
	if aqi < 51 || aqi > 100 {
		t.Errorf("PM2.5=25 мкг/м³ должно давать AQI 51–100, получили %d", aqi)
	}
}

func TestCalcAQI_HighNO2(t *testing.T) {
	// NO2 = 0.3 мг/м³ = 0.3*532 = 159.6 ppb → AQI 101–150 (Unhealthy sensitive)
	aqi := domain.CalcAQIforNO2(0.3)
	if aqi < 101 || aqi > 150 {
		t.Errorf("NO2=0.3 мг/м³ должно давать AQI 101–150, получили %d", aqi)
	}
}

func TestCalcOverallAQI_MaxSubindex(t *testing.T) {
	// PM2.5 низкий, NO2 высокий — итоговый AQI = max из субиндексов.
	pm25 := 0.005 // 5 мкг/м³ → AQI ~20
	no2 := 0.3    // 159 ppb → AQI ~115
	m := &domain.Measurement{PM25: &pm25, NO2: &no2}
	overall := domain.CalcOverallAQI(m)
	aqiPM := domain.CalcAQIforPM25(pm25)
	aqiNO2 := domain.CalcAQIforNO2(no2)
	expected := aqiNO2
	if aqiPM > aqiNO2 {
		expected = aqiPM
	}
	if overall != expected {
		t.Errorf("CalcOverallAQI: ожидали %d, получили %d", expected, overall)
	}
}

func TestCalcOverallAQI_NilMeasurement(t *testing.T) {
	m := &domain.Measurement{}
	aqi := domain.CalcOverallAQI(m)
	if aqi != 0 {
		t.Errorf("AQI при нулевых данных должен быть 0, получили %d", aqi)
	}
}

func TestAQIToCategory(t *testing.T) {
	cases := []struct {
		aqi  int
		want domain.AQICategory
	}{
		{0, domain.AQICategoryGood},
		{50, domain.AQICategoryGood},
		{51, domain.AQICategoryModerate},
		{100, domain.AQICategoryModerate},
		{101, domain.AQICategoryUnhealthy},
		{150, domain.AQICategoryUnhealthy},
		{151, domain.AQICategoryBad},
		{200, domain.AQICategoryBad},
		{201, domain.AQICategoryVeryBad},
		{300, domain.AQICategoryVeryBad},
		{301, domain.AQICategoryHazardous},
		{500, domain.AQICategoryHazardous},
	}
	for _, c := range cases {
		got := domain.AQIToCategory(c.aqi)
		if got != c.want {
			t.Errorf("AQIToCategory(%d): ожидали %s, получили %s", c.aqi, c.want, got)
		}
	}
}
