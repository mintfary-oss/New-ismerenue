package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/mintfary/aqi-platform/internal/config"
)

// NewRedisClient создаёт и проверяет клиент Redis.
// Используется для: блеклист JWT токенов, кеш последних измерений, rate limit counters.
func NewRedisClient(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	})

	// Проверка соединения.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("ping Redis: %w", err)
	}

	return client, nil
}

// TokenBlacklist управляет блеклистом отозванных JWT токенов в Redis.
type TokenBlacklist struct {
	client *redis.Client
	ttl    time.Duration
}

// NewTokenBlacklist создаёт хранилище отозванных токенов.
func NewTokenBlacklist(client *redis.Client, ttl time.Duration) *TokenBlacklist {
	return &TokenBlacklist{client: client, ttl: ttl}
}

// Add добавляет токен в блеклист с TTL, равным времени до его истечения.
func (b *TokenBlacklist) Add(ctx context.Context, tokenID string, expiry time.Time) error {
	ttl := time.Until(expiry)
	if ttl <= 0 {
		return nil // токен уже истёк, не нужно хранить
	}
	key := "blacklist:token:" + tokenID
	return b.client.Set(ctx, key, "1", ttl).Err()
}

// IsBlacklisted проверяет, находится ли токен в блеклисте.
func (b *TokenBlacklist) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	key := "blacklist:token:" + tokenID
	result, err := b.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("проверка блеклиста: %w", err)
	}
	return result > 0, nil
}

// LoginAttemptTracker отслеживает попытки входа для защиты от brute force.
type LoginAttemptTracker struct {
	client          *redis.Client
	maxAttempts     int
	lockoutDuration time.Duration
}

// NewLoginAttemptTracker создаёт трекер попыток входа.
func NewLoginAttemptTracker(client *redis.Client, maxAttempts int, lockout time.Duration) *LoginAttemptTracker {
	return &LoginAttemptTracker{
		client:          client,
		maxAttempts:     maxAttempts,
		lockoutDuration: lockout,
	}
}

// Increment увеличивает счётчик неудачных попыток входа.
// Возвращает текущее число попыток.
func (t *LoginAttemptTracker) Increment(ctx context.Context, email string) (int, error) {
	key := "login_attempts:" + email
	count, err := t.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	// Устанавливаем TTL только при первой попытке.
	if count == 1 {
		t.client.Expire(ctx, key, t.lockoutDuration)
	}
	return int(count), nil
}

// IsLocked проверяет, заблокирован ли аккаунт.
func (t *LoginAttemptTracker) IsLocked(ctx context.Context, email string) (bool, error) {
	key := "login_attempts:" + email
	count, err := t.client.Get(ctx, key).Int()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return count >= t.maxAttempts, nil
}

// Reset сбрасывает счётчик после успешного входа.
func (t *LoginAttemptTracker) Reset(ctx context.Context, email string) error {
	return t.client.Del(ctx, "login_attempts:"+email).Err()
}
