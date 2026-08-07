package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/mintfary/aqi-platform/internal/config"
	"github.com/mintfary/aqi-platform/internal/domain"
)

// newTestAuthService — вспомогательная функция для создания AuthService в тестах.
func newTestAuthService(repo *mockUserRepo) *AuthService {
	cfg := config.AuthConfig{
		JWTSecret:       "test-secret-key-32-bytes-minimum!",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		// Минимальные параметры Argon2id для быстрых тестов (не для продакшна!)
		Argon2Time:    1,
		Argon2Memory:  64 * 1024, // 64 MB
		Argon2Threads: 2,
		Argon2KeyLen:  32,
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError, // молчим в тестах
	}))
	return NewAuthService(
		repo,
		newMockTokenStore(),
		newMockLoginAttemptStore(),
		cfg,
		logger,
	)
}

// TestHashAndVerifyPassword проверяет хэширование и проверку паролей (Argon2id).
func TestHashAndVerifyPassword(t *testing.T) {
	svc := newTestAuthService(newMockUserRepo())

	password := "SecurePass1234!"
	hash, err := svc.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword() вернул пустой хэш")
	}
	if hash == password {
		t.Fatal("HashPassword() вернул пароль в открытом виде")
	}
	// Хэш должен содержать признак Argon2id
	if len(hash) < 20 {
		t.Errorf("HashPassword() хэш слишком короткий: %q", hash)
	}

	// Верный пароль должен проходить проверку
	if !svc.verifyPassword(password, hash) {
		t.Error("verifyPassword() = false для верного пароля")
	}

	// Неверный пароль не должен проходить
	if svc.verifyPassword("WrongPassword", hash) {
		t.Error("verifyPassword() = true для неверного пароля")
	}
}

// TestHashesAreUnique проверяет что два хэша одного пароля не совпадают (соль).
func TestHashesAreUnique(t *testing.T) {
	svc := newTestAuthService(newMockUserRepo())

	h1, err := svc.HashPassword("SamePassword!")
	mustNotError(err)
	h2, err := svc.HashPassword("SamePassword!")
	mustNotError(err)

	if h1 == h2 {
		t.Error("Два хэша одного пароля не должны совпадать (разные соли)")
	}
}

// TestLoginSuccess проверяет успешный вход.
func TestLoginSuccess(t *testing.T) {
	ctx := context.Background()
	repo := newMockUserRepo()
	svc := newTestAuthService(repo)

	// Создаём пользователя с хэшированным паролем
	hash, err := svc.HashPassword("MyPassword123!")
	mustNotError(err)

	_, err = repo.Create(ctx, domain.CreateUserInput{
		Email:    "test@example.com",
		Username: "testuser",
		Password: hash,
		Role:     domain.RoleAnalyst,
	})
	mustNotError(err)

	// Логинимся
	pair, err := svc.Login(ctx, LoginInput{
		Email:    "test@example.com",
		Password: "MyPassword123!",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if pair == nil {
		t.Fatal("Login() вернул nil TokenPair")
	}
	if pair.AccessToken == "" {
		t.Error("Login() вернул пустой AccessToken")
	}
	if pair.RefreshToken == "" {
		t.Error("Login() вернул пустой RefreshToken")
	}
}

// TestLoginWrongPassword проверяет отказ при неверном пароле.
func TestLoginWrongPassword(t *testing.T) {
	ctx := context.Background()
	repo := newMockUserRepo()
	svc := newTestAuthService(repo)

	hash, _ := svc.HashPassword("CorrectPassword!")
	_, _ = repo.Create(ctx, domain.CreateUserInput{
		Email:    "user@example.com",
		Username: "user1",
		Password: hash,
		Role:     domain.RoleAnalyst,
	})

	_, err := svc.Login(ctx, LoginInput{
		Email:    "user@example.com",
		Password: "WrongPassword",
	})
	if err == nil {
		t.Fatal("Login() с неверным паролем должен вернуть ошибку")
	}
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("Login() ошибка = %v, хотим ErrInvalidCredentials", err)
	}
}

// TestLoginNonExistentUser проверяет отказ для несуществующего пользователя.
func TestLoginNonExistentUser(t *testing.T) {
	ctx := context.Background()
	svc := newTestAuthService(newMockUserRepo())

	_, err := svc.Login(ctx, LoginInput{
		Email:    "nobody@example.com",
		Password: "password",
	})
	if err == nil {
		t.Fatal("Login() для несуществующего пользователя должен вернуть ошибку")
	}
}

// TestLoginAccountLockAfterFailures проверяет блокировку после 5 неудач.
func TestLoginAccountLockAfterFailures(t *testing.T) {
	ctx := context.Background()
	repo := newMockUserRepo()
	attempts := newMockLoginAttemptStore()

	cfg := config.AuthConfig{
		JWTSecret:       "test-secret-key-32-bytes-minimum!",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		Argon2Time:    1,
		Argon2Memory:  64 * 1024,
		Argon2Threads: 2,
		Argon2KeyLen:  32,
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	svc := NewAuthService(repo, newMockTokenStore(), attempts, cfg, logger)

	hash, _ := svc.HashPassword("RealPass!")
	_, _ = repo.Create(ctx, domain.CreateUserInput{
		Email:    "lock@example.com",
		Username: "lockuser",
		Password: hash,
		Role:     domain.RoleAnalyst,
	})

	// 5 неудачных попыток → блокировка в mockLoginAttemptStore
	for range 5 {
		_, _ = svc.Login(ctx, LoginInput{
			Email:    "lock@example.com",
			Password: "WrongPassword",
		})
	}

	// Следующая попытка должна вернуть ErrAccountLocked
	_, err := svc.Login(ctx, LoginInput{
		Email:    "lock@example.com",
		Password: "RealPass!",
	})
	if err == nil {
		t.Fatal("Login() должен быть заблокирован после 5 неудач")
	}
	if !errors.Is(err, domain.ErrAccountLocked) {
		t.Errorf("Login() ошибка = %v, хотим ErrAccountLocked", err)
	}
}
