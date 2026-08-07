// Package service — вспомогательные mock-реализации для unit-тестов.
// Не используют БД или Redis — только in-memory структуры.
package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mintfary/aqi-platform/internal/domain"
)

// ── mockUserRepo ─────────────────────────────────────────────────────────────

type mockUserRepo struct {
	mu    sync.RWMutex
	users map[uuid.UUID]*domain.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[uuid.UUID]*domain.User)}
}

func (r *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.users {
		if u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("%w", domain.ErrNotFound)
}

func (r *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.users[id]
	if !ok {
		return nil, fmt.Errorf("%w", domain.ErrNotFound)
	}
	cp := *u
	return &cp, nil
}

func (r *mockUserRepo) Create(ctx context.Context, in domain.CreateUserInput) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u := &domain.User{
		ID:        uuid.New(),
		Email:     in.Email,
		Username:  in.Username,
		Password:  in.Password,
		Role:      in.Role,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	r.users[u.ID] = u
	cp := *u
	return &cp, nil
}

func (r *mockUserRepo) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return fmt.Errorf("%w", domain.ErrNotFound)
	}
	u.Password = hash
	return nil
}

func (r *mockUserRepo) List(ctx context.Context, limit, offset int) ([]domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.User, 0, len(r.users))
	for _, u := range r.users {
		out = append(out, *u)
	}
	// Простая пагинация
	if offset >= len(out) {
		return nil, nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return out[offset:end], nil
}

func (r *mockUserRepo) Update(ctx context.Context, id uuid.UUID, in domain.UpdateUserInput) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return nil, fmt.Errorf("%w", domain.ErrNotFound)
	}
	if in.Username != nil {
		u.Username = *in.Username
	}
	if in.Role != nil {
		u.Role = *in.Role
	}
	if in.IsActive != nil {
		u.IsActive = *in.IsActive
	}
	cp := *u
	return &cp, nil
}

func (r *mockUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.users[id]; !ok {
		return fmt.Errorf("%w", domain.ErrNotFound)
	}
	// Soft delete — деактивируем
	r.users[id].IsActive = false
	return nil
}

func (r *mockUserRepo) CountActive(ctx context.Context) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n := 0
	for _, u := range r.users {
		if u.IsActive {
			n++
		}
	}
	return n, nil
}

func (r *mockUserRepo) Count(ctx context.Context) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.users), nil
}

// ── mockTokenStore ───────────────────────────────────────────────────────────

type mockTokenStore struct {
	mu          sync.RWMutex
	blacklisted map[string]time.Time
}

func newMockTokenStore() *mockTokenStore {
	return &mockTokenStore{blacklisted: make(map[string]time.Time)}
}

func (s *mockTokenStore) Add(ctx context.Context, tokenID string, expiry time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blacklisted[tokenID] = expiry
	return nil
}

func (s *mockTokenStore) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	exp, ok := s.blacklisted[tokenID]
	if !ok {
		return false, nil
	}
	// Если время вышло — токен больше не в блеклисте
	if time.Now().After(exp) {
		return false, nil
	}
	return true, nil
}

// ── mockLoginAttemptStore ────────────────────────────────────────────────────

type mockLoginAttemptStore struct {
	mu       sync.Mutex
	attempts map[string]int
	locked   map[string]bool
}

func newMockLoginAttemptStore() *mockLoginAttemptStore {
	return &mockLoginAttemptStore{
		attempts: make(map[string]int),
		locked:   make(map[string]bool),
	}
}

func (s *mockLoginAttemptStore) IsLocked(ctx context.Context, email string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.locked[email], nil
}

func (s *mockLoginAttemptStore) Increment(ctx context.Context, email string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attempts[email]++
	if s.attempts[email] >= 5 {
		s.locked[email] = true
	}
	return s.attempts[email], nil
}

func (s *mockLoginAttemptStore) Reset(ctx context.Context, email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attempts[email] = 0
	s.locked[email] = false
	return nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

// mustNotError паникует если err != nil — удобно для подготовки тестов.
func mustNotError(err error) {
	if err != nil {
		panic(fmt.Sprintf("unexpected error in test setup: %v", err))
	}
}
