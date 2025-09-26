// Package redis provides Redis client functionality for caching and session management
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"notification-service/pkg/models"
)

// Client wraps a Redis client
type Client struct {
	client *redis.Client
	logger *logrus.Logger
}

// NewClient creates a new Redis client
func NewClient(host, port, password string, db int, logger *logrus.Logger) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{
		client: rdb,
		logger: logger,
	}, nil
}

// SetUserSession stores a user session in Redis
func (c *Client) SetUserSession(ctx context.Context, userID string, session *models.UserSession, expiration time.Duration) error {
	key := fmt.Sprintf("session:%s", userID)
	
	sessionData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := c.client.Set(ctx, key, sessionData, expiration).Err(); err != nil {
		return fmt.Errorf("failed to set session: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"key":     key,
	}).Debug("User session stored successfully")

	return nil
}

// GetUserSession retrieves a user session from Redis
func (c *Client) GetUserSession(ctx context.Context, userID string) (*models.UserSession, error) {
	key := fmt.Sprintf("session:%s", userID)
	
	sessionData, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session not found for user %s", userID)
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session models.UserSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// DeleteUserSession removes a user session from Redis
func (c *Client) DeleteUserSession(ctx context.Context, userID string) error {
	key := fmt.Sprintf("session:%s", userID)
	
	result := c.client.Del(ctx, key)
	if err := result.Err(); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"key":     key,
	}).Debug("User session deleted successfully")

	return nil
}

// SetNotificationStatus stores notification status in Redis for tracking
func (c *Client) SetNotificationStatus(ctx context.Context, notificationID string, status models.NotificationStatus, expiration time.Duration) error {
	key := fmt.Sprintf("notification_status:%s", notificationID)
	
	if err := c.client.Set(ctx, key, string(status), expiration).Err(); err != nil {
		return fmt.Errorf("failed to set notification status: %w", err)
	}

	return nil
}

// GetNotificationStatus retrieves notification status from Redis
func (c *Client) GetNotificationStatus(ctx context.Context, notificationID string) (models.NotificationStatus, error) {
	key := fmt.Sprintf("notification_status:%s", notificationID)
	
	status, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("notification status not found")
		}
		return "", fmt.Errorf("failed to get notification status: %w", err)
	}

	return models.NotificationStatus(status), nil
}

// IncrementCounter increments a counter in Redis (for rate limiting, metrics)
func (c *Client) IncrementCounter(ctx context.Context, key string, expiration time.Duration) (int64, error) {
	pipe := c.client.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, expiration)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to increment counter: %w", err)
	}

	return incr.Val(), nil
}

// GetCounter gets a counter value from Redis
func (c *Client) GetCounter(ctx context.Context, key string) (int64, error) {
	val, err := c.client.Get(ctx, key).Int64()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get counter: %w", err)
	}
	return val, nil
}

// SetCache sets a generic cache value
func (c *Client) SetCache(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value: %w", err)
	}

	if err := c.client.Set(ctx, key, data, expiration).Err(); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// GetCache gets a generic cache value
func (c *Client) GetCache(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("cache key not found: %s", key)
		}
		return fmt.Errorf("failed to get cache: %w", err)
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return fmt.Errorf("failed to unmarshal cache value: %w", err)
	}

	return nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.client.Close()
}

// Ping tests the Redis connection
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}