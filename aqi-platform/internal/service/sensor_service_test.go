// Package service — тесты SensorService.
//
// Покрываем:
//   - List: все датчики / только активные
//   - GetByID: существующий / несуществующий
//   - Create: корректный / недопустимый тип
//   - Update: обновление полей / недопустимый тип
//   - Delete: существующий / несуществующий
//   - SensorStatus: онлайн / оффлайн / никогда не видели
package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mintfary/aqi-platform/internal/domain"
)

// ── mockSensorRepository ──────────────────────────────────────────────────────

type mockSensorRepository struct {
	mu      sync.RWMutex
	sensors map[uuid.UUID]*domain.Sensor
}

func newMockSensorRepository() *mockSensorRepository {
	return &mockSensorRepository{sensors: make(map[uuid.UUID]*domain.Sensor)}
}

func (r *mockSensorRepository) List(_ context.Context, onlyActive bool) ([]domain.Sensor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.Sensor, 0, len(r.sensors))
	for _, s := range r.sensors {
		if onlyActive && !s.IsActive {
			continue
		}
		cp := *s
		out = append(out, cp)
	}
	return out, nil
}

func (r *mockSensorRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.Sensor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.sensors[id]
	if !ok {
		return nil, fmt.Errorf("%w", domain.ErrNotFound)
	}
	cp := *s
	return &cp, nil
}

func (r *mockSensorRepository) Create(_ context.Context, in domain.CreateSensorInput) (*domain.Sensor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := &domain.Sensor{
		ID:        uuid.New(),
		Name:      in.Name,
		Address:   in.Address,
		Lat:       in.Lat,
		Lng:       in.Lng,
		Type:      in.Type,
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	r.sensors[s.ID] = s
	cp := *s
	return &cp, nil
}

func (r *mockSensorRepository) Update(_ context.Context, id uuid.UUID, in domain.UpdateSensorInput) (*domain.Sensor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sensors[id]
	if !ok {
		return nil, fmt.Errorf("%w", domain.ErrNotFound)
	}
	if in.Name != nil {
		s.Name = *in.Name
	}
	if in.Address != nil {
		s.Address = *in.Address
	}
	if in.Lat != nil {
		s.Lat = *in.Lat
	}
	if in.Lng != nil {
		s.Lng = *in.Lng
	}
	if in.Type != nil {
		s.Type = *in.Type
	}
	if in.IsActive != nil {
		s.IsActive = *in.IsActive
	}
	cp := *s
	return &cp, nil
}

func (r *mockSensorRepository) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.sensors[id]; !ok {
		return fmt.Errorf("%w", domain.ErrNotFound)
	}
	delete(r.sensors, id)
	return nil
}

// UpdateLastSeen — реализует SensorLastSeenUpdater для mockSensorRepository.
func (r *mockSensorRepository) UpdateLastSeen(_ context.Context, id uuid.UUID, t time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sensors[id]
	if !ok {
		return fmt.Errorf("%w", domain.ErrNotFound)
	}
	s.LastSeen = &t
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func createTestSensor(t *testing.T, repo *mockSensorRepository, sType domain.SensorType) *domain.Sensor {
	t.Helper()
	s, err := repo.Create(context.Background(), domain.CreateSensorInput{
		Name:    "Тест датчик",
		Address: "ул. Тестовая, 1",
		Lat:     55.35,
		Lng:     86.08,
		Type:    sType,
	})
	if err != nil {
		t.Fatalf("создание тестового датчика: %v", err)
	}
	return s
}

// ── SensorService.List ─────────────────────────────────────────────────────────

func TestSensorList_AllSensors(t *testing.T) {
	repo := newMockSensorRepository()
	svc := NewSensorService(repo, testLogger())

	createTestSensor(t, repo, domain.SensorTypeGas)
	createTestSensor(t, repo, domain.SensorTypeDust)

	sensors, err := svc.List(context.Background(), false)
	if err != nil {
		t.Fatalf("List вернул ошибку: %v", err)
	}
	if len(sensors) != 2 {
		t.Errorf("ожидали 2 датчика, получили %d", len(sensors))
	}
}

func TestSensorList_OnlyActive(t *testing.T) {
	repo := newMockSensorRepository()
	svc := NewSensorService(repo, testLogger())

	s1 := createTestSensor(t, repo, domain.SensorTypeGas)
	createTestSensor(t, repo, domain.SensorTypeDust)

	// Деактивируем первый датчик.
	active := false
	_, err := repo.Update(context.Background(), s1.ID, domain.UpdateSensorInput{IsActive: &active})
	if err != nil {
		t.Fatalf("деактивация датчика: %v", err)
	}

	sensors, err := svc.List(context.Background(), true)
	if err != nil {
		t.Fatalf("List(onlyActive) вернул ошибку: %v", err)
	}
	if len(sensors) != 1 {
		t.Errorf("ожидали 1 активный датчик, получили %d", len(sensors))
	}
}

func TestSensorList_Empty(t *testing.T) {
	repo := newMockSensorRepository()
	svc := NewSensorService(repo, testLogger())

	sensors, err := svc.List(context.Background(), false)
	if err != nil {
		t.Fatalf("List на пустом репо вернул ошибку: %v", err)
	}
	if len(sensors) != 0 {
		t.Errorf("ожидали 0 датчиков, получили %d", len(sensors))
	}
}

// ── SensorService.GetByID ─────────────────────────────────────────────────────

func TestSensorGetByID_Exists(t *testing.T) {
	repo := newMockSensorRepository()
	svc := NewSensorService(repo, testLogger())

	created := createTestSensor(t, repo, domain.SensorTypeCombo)

	got, err := svc.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID вернул ошибку: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID не совпадает: ожидали %v, получили %v", created.ID, got.ID)
	}
}

func TestSensorGetByID_NotFound(t *testing.T) {
	repo := newMockSensorRepository()
	svc := NewSensorService(repo, testLogger())

	_, err := svc.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("ожидали ошибку ErrNotFound")
	}
}

// ── SensorService.Create ──────────────────────────────────────────────────────

func TestSensorCreate_ValidTypes(t *testing.T) {
	validTypes := []domain.SensorType{
		domain.SensorTypeGas,
		domain.SensorTypeDust,
		domain.SensorTypeCombo,
	}

	for _, sType := range validTypes {
		t.Run(string(sType), func(t *testing.T) {
			repo := newMockSensorRepository()
			svc := NewSensorService(repo, testLogger())

			s, err := svc.Create(context.Background(), domain.CreateSensorInput{
				Name:    "Датчик",
				Address: "Адрес, 1",
				Lat:     55.35,
				Lng:     86.08,
				Type:    sType,
			})
			if err != nil {
				t.Fatalf("Create(%s) вернул ошибку: %v", sType, err)
			}
			if s.ID == uuid.Nil {
				t.Error("ID датчика не должен быть пустым")
			}
			if s.Type != sType {
				t.Errorf("ожидали тип %s, получили %s", sType, s.Type)
			}
		})
	}
}

func TestSensorCreate_InvalidType(t *testing.T) {
	repo := newMockSensorRepository()
	svc := NewSensorService(repo, testLogger())

	_, err := svc.Create(context.Background(), domain.CreateSensorInput{
		Name:    "Датчик",
		Address: "Адрес, 1",
		Lat:     55.35,
		Lng:     86.08,
		Type:    "unknown_type",
	})
	if err == nil {
		t.Fatal("ожидали ошибку при недопустимом типе датчика")
	}
}

func TestSensorCreate_AssignsID(t *testing.T) {
	repo := newMockSensorRepository()
	svc := NewSensorService(repo, testLogger())

	s, err := svc.Create(context.Background(), domain.CreateSensorInput{
		Name:    "Датчик А",
		Address: "ул. Весенняя, 5",
		Lat:     55.3,
		Lng:     86.1,
		Type:    domain.SensorTypeGas,
	})
	if err != nil {
		t.Fatalf("Create вернул ошибку: %v", err)
	}
	if !s.IsActive {
		t.Error("новый датчик должен быть активным по умолчанию")
	}
}

// ── SensorService.Update ──────────────────────────────────────────────────────

func TestSensorUpdate_Name(t *testing.T) {
	repo := newMockSensorRepository()
	svc := NewSensorService(repo, testLogger())

	s := createTestSensor(t, repo, domain.SensorTypeGas)
	newName := "Обновлённое имя"

	updated, err := svc.Update(context.Background(), s.ID, domain.UpdateSensorInput{Name: &newName})
	if err != nil {
		t.Fatalf("Update вернул ошибку: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("ожидали имя %q, получили %q", newName, updated.Name)
	}
}

func TestSensorUpdate_InvalidType(t *testing.T) {
	repo := newMockSensorRepository()
	svc := NewSensorService(repo, testLogger())

	s := createTestSensor(t, repo, domain.SensorTypeGas)
	badType := domain.SensorType("badtype")

	_, err := svc.Update(context.Background(), s.ID, domain.UpdateSensorInput{Type: &badType})
	if err == nil {
		t.Fatal("ожидали ошибку при недопустимом типе при обновлении")
	}
}

func TestSensorUpdate_Deactivate(t *testing.T) {
	repo := newMockSensorRepository()
	svc := NewSensorService(repo, testLogger())

	s := createTestSensor(t, repo, domain.SensorTypeDust)
	active := false

	updated, err := svc.Update(context.Background(), s.ID, domain.UpdateSensorInput{IsActive: &active})
	if err != nil {
		t.Fatalf("Update вернул ошибку: %v", err)
	}
	if updated.IsActive {
		t.Error("датчик должен быть деактивирован")
	}
}

// ── SensorService.Delete ──────────────────────────────────────────────────────

func TestSensorDelete_Exists(t *testing.T) {
	repo := newMockSensorRepository()
	svc := NewSensorService(repo, testLogger())

	s := createTestSensor(t, repo, domain.SensorTypeGas)

	err := svc.Delete(context.Background(), s.ID)
	if err != nil {
		t.Fatalf("Delete вернул ошибку: %v", err)
	}

	// Убеждаемся что датчик удалён.
	_, err = svc.GetByID(context.Background(), s.ID)
	if err == nil {
		t.Fatal("датчик должен быть удалён")
	}
}

func TestSensorDelete_NotFound(t *testing.T) {
	repo := newMockSensorRepository()
	svc := NewSensorService(repo, testLogger())

	err := svc.Delete(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("ожидали ошибку при удалении несуществующего датчика")
	}
}

// ── SensorService.SensorStatus ────────────────────────────────────────────────

func TestSensorStatus_NeverSeen(t *testing.T) {
	repo := newMockSensorRepository()
	svc := NewSensorService(repo, testLogger())

	s := createTestSensor(t, repo, domain.SensorTypeGas)

	status, err := svc.SensorStatus(context.Background(), s.ID)
	if err != nil {
		t.Fatalf("SensorStatus вернул ошибку: %v", err)
	}
	if status.IsOnline {
		t.Error("датчик без LastSeen не должен быть онлайн")
	}
	if status.LastSeen != nil {
		t.Error("LastSeen должен быть nil для нового датчика")
	}
}

func TestSensorStatus_Online(t *testing.T) {
	repo := newMockSensorRepository()
	svc := NewSensorService(repo, testLogger())

	s := createTestSensor(t, repo, domain.SensorTypeCombo)

	// Обновляем LastSeen на "только что".
	recentTime := time.Now().UTC().Add(-5 * time.Minute)
	mustNotError(repo.UpdateLastSeen(context.Background(), s.ID, recentTime))

	status, err := svc.SensorStatus(context.Background(), s.ID)
	if err != nil {
		t.Fatalf("SensorStatus вернул ошибку: %v", err)
	}
	if !status.IsOnline {
		t.Errorf("датчик с LastSeen=%v должен быть онлайн", recentTime)
	}
}

func TestSensorStatus_Offline(t *testing.T) {
	repo := newMockSensorRepository()
	svc := NewSensorService(repo, testLogger())

	s := createTestSensor(t, repo, domain.SensorTypeGas)

	// LastSeen — 2 часа назад (больше 30 минут → оффлайн).
	oldTime := time.Now().UTC().Add(-2 * time.Hour)
	mustNotError(repo.UpdateLastSeen(context.Background(), s.ID, oldTime))

	status, err := svc.SensorStatus(context.Background(), s.ID)
	if err != nil {
		t.Fatalf("SensorStatus вернул ошибку: %v", err)
	}
	if status.IsOnline {
		t.Errorf("датчик с LastSeen=%v должен быть оффлайн", oldTime)
	}
}

func TestSensorStatus_NotFound(t *testing.T) {
	repo := newMockSensorRepository()
	svc := NewSensorService(repo, testLogger())

	_, err := svc.SensorStatus(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("ожидали ошибку для несуществующего датчика")
	}
}

func TestSensorStatus_Fields(t *testing.T) {
	repo := newMockSensorRepository()
	svc := NewSensorService(repo, testLogger())

	s := createTestSensor(t, repo, domain.SensorTypeDust)

	status, err := svc.SensorStatus(context.Background(), s.ID)
	if err != nil {
		t.Fatalf("SensorStatus вернул ошибку: %v", err)
	}
	if status.SensorID != s.ID {
		t.Errorf("SensorID: ожидали %v, получили %v", s.ID, status.SensorID)
	}
	if status.Name != s.Name {
		t.Errorf("Name: ожидали %q, получили %q", s.Name, status.Name)
	}
	if !status.IsActive {
		t.Error("новый датчик должен быть активным")
	}
}
