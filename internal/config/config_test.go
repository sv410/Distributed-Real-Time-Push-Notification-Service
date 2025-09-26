package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envKeys := []string{
		"KAFKA_BROKERS", "KAFKA_TOPIC", "CONSUMER_GROUP",
		"REDIS_ADDR", "REDIS_PASSWORD", "REDIS_DB",
		"RATE_LIMIT_PER_USER", "RATE_LIMIT_WINDOW",
		"WORKER_COUNT", "MAX_QUEUE_SIZE", "RETRY_ATTEMPTS", "RETRY_DELAY",
		"PROVIDER_TIMEOUT", "PROVIDER_RETRIES",
		"PORT", "LOG_LEVEL", "SHUTDOWN_TIMEOUT",
	}

	for _, key := range envKeys {
		originalEnv[key] = os.Getenv(key)
		os.Unsetenv(key)
	}

	// Restore environment after test
	defer func() {
		for key, value := range originalEnv {
			if value != "" {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	// Test default configuration
	cfg := LoadConfig()

	if len(cfg.KafkaBrokers) != 1 || cfg.KafkaBrokers[0] != "localhost:9092" {
		t.Errorf("Expected KafkaBrokers to be [localhost:9092], got %v", cfg.KafkaBrokers)
	}

	if cfg.KafkaTopic != "notifications" {
		t.Errorf("Expected KafkaTopic to be 'notifications', got %s", cfg.KafkaTopic)
	}

	if cfg.ConsumerGroup != "notification-service" {
		t.Errorf("Expected ConsumerGroup to be 'notification-service', got %s", cfg.ConsumerGroup)
	}

	if cfg.RedisAddr != "localhost:6379" {
		t.Errorf("Expected RedisAddr to be 'localhost:6379', got %s", cfg.RedisAddr)
	}

	if cfg.RateLimitPerUser != 10 {
		t.Errorf("Expected RateLimitPerUser to be 10, got %d", cfg.RateLimitPerUser)
	}

	if cfg.RateLimitWindow != 1*time.Minute {
		t.Errorf("Expected RateLimitWindow to be 1m, got %v", cfg.RateLimitWindow)
	}

	if cfg.WorkerCount != 10 {
		t.Errorf("Expected WorkerCount to be 10, got %d", cfg.WorkerCount)
	}

	if cfg.MaxQueueSize != 1000 {
		t.Errorf("Expected MaxQueueSize to be 1000, got %d", cfg.MaxQueueSize)
	}

	if cfg.Port != "8080" {
		t.Errorf("Expected Port to be '8080', got %s", cfg.Port)
	}
}

func TestLoadConfigWithEnvironment(t *testing.T) {
	// Set environment variables
	os.Setenv("KAFKA_TOPIC", "test-topic")
	os.Setenv("RATE_LIMIT_PER_USER", "20")
	os.Setenv("WORKER_COUNT", "5")
	os.Setenv("PORT", "9090")

	defer func() {
		os.Unsetenv("KAFKA_TOPIC")
		os.Unsetenv("RATE_LIMIT_PER_USER")
		os.Unsetenv("WORKER_COUNT")
		os.Unsetenv("PORT")
	}()

	cfg := LoadConfig()

	if cfg.KafkaTopic != "test-topic" {
		t.Errorf("Expected KafkaTopic to be 'test-topic', got %s", cfg.KafkaTopic)
	}

	if cfg.RateLimitPerUser != 20 {
		t.Errorf("Expected RateLimitPerUser to be 20, got %d", cfg.RateLimitPerUser)
	}

	if cfg.WorkerCount != 5 {
		t.Errorf("Expected WorkerCount to be 5, got %d", cfg.WorkerCount)
	}

	if cfg.Port != "9090" {
		t.Errorf("Expected Port to be '9090', got %s", cfg.Port)
	}
}
