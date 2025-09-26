package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// RateLimiter provides Redis-based rate limiting functionality
type RateLimiter struct {
	client    *redis.Client
	limit     int           // maximum requests per window
	window    time.Duration // time window
	keyPrefix string
}

// NewRateLimiter creates a new Redis-based rate limiter
func NewRateLimiter(client *redis.Client, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		client:    client,
		limit:     limit,
		window:    window,
		keyPrefix: "rate_limit:",
	}
}

// IsAllowed checks if a user is allowed to send a notification
func (rl *RateLimiter) IsAllowed(ctx context.Context, userID string) (bool, error) {
	key := fmt.Sprintf("%s%s", rl.keyPrefix, userID)

	// Use Redis pipeline for atomic operations
	pipe := rl.client.Pipeline()

	// Increment the counter
	incrCmd := pipe.Incr(ctx, key)

	// Set expiration if this is the first increment
	pipe.Expire(ctx, key, rl.window)

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("redis pipeline error: %w", err)
	}

	// Check if the count exceeds the limit
	count := incrCmd.Val()
	return count <= int64(rl.limit), nil
}

// GetCurrentCount returns the current count for a user
func (rl *RateLimiter) GetCurrentCount(ctx context.Context, userID string) (int, error) {
	key := fmt.Sprintf("%s%s", rl.keyPrefix, userID)

	count, err := rl.client.Get(ctx, key).Int()
	if err == redis.Nil {
		return 0, nil // Key doesn't exist, count is 0
	}
	if err != nil {
		return 0, fmt.Errorf("redis get error: %w", err)
	}

	return count, nil
}

// GetRemainingCount returns remaining notifications allowed for a user
func (rl *RateLimiter) GetRemainingCount(ctx context.Context, userID string) (int, error) {
	current, err := rl.GetCurrentCount(ctx, userID)
	if err != nil {
		return 0, err
	}

	remaining := rl.limit - current
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

// Reset resets the rate limit for a user (useful for testing)
func (rl *RateLimiter) Reset(ctx context.Context, userID string) error {
	key := fmt.Sprintf("%s%s", rl.keyPrefix, userID)

	err := rl.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("redis delete error: %w", err)
	}

	return nil
}

// GetTTL returns the remaining time until the rate limit resets
func (rl *RateLimiter) GetTTL(ctx context.Context, userID string) (time.Duration, error) {
	key := fmt.Sprintf("%s%s", rl.keyPrefix, userID)

	ttl, err := rl.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("redis ttl error: %w", err)
	}

	return ttl, nil
}

// NewRedisClient creates a new Redis client with the given configuration
func NewRedisClient(addr, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

// HealthCheck performs a health check on the Redis connection
func HealthCheck(ctx context.Context, client *redis.Client) error {
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}
	return nil
}
