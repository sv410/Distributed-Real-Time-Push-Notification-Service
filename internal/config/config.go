package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the notification service
type Config struct {
	// Kafka configuration
	KafkaBrokers  []string
	KafkaTopic    string
	ConsumerGroup string

	// Redis configuration
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// Rate limiting configuration
	RateLimitPerUser int           // notifications per user per window
	RateLimitWindow  time.Duration // rate limit window duration

	// Worker pool configuration
	WorkerCount   int
	MaxQueueSize  int
	RetryAttempts int
	RetryDelay    time.Duration

	// External provider configuration
	ProviderTimeout time.Duration
	ProviderRetries int

	// Service configuration
	Port            string
	LogLevel        string
	ShutdownTimeout time.Duration
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *Config {
	cfg := &Config{
		// Kafka defaults
		KafkaBrokers:  getStringSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
		KafkaTopic:    getEnv("KAFKA_TOPIC", "notifications"),
		ConsumerGroup: getEnv("CONSUMER_GROUP", "notification-service"),

		// Redis defaults
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),

		// Rate limiting defaults
		RateLimitPerUser: getEnvAsInt("RATE_LIMIT_PER_USER", 10),
		RateLimitWindow:  getEnvAsDuration("RATE_LIMIT_WINDOW", 1*time.Minute),

		// Worker pool defaults
		WorkerCount:   getEnvAsInt("WORKER_COUNT", 10),
		MaxQueueSize:  getEnvAsInt("MAX_QUEUE_SIZE", 1000),
		RetryAttempts: getEnvAsInt("RETRY_ATTEMPTS", 3),
		RetryDelay:    getEnvAsDuration("RETRY_DELAY", 1*time.Second),

		// External provider defaults
		ProviderTimeout: getEnvAsDuration("PROVIDER_TIMEOUT", 10*time.Second),
		ProviderRetries: getEnvAsInt("PROVIDER_RETRIES", 2),

		// Service defaults
		Port:            getEnv("PORT", "8080"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		ShutdownTimeout: getEnvAsDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple comma-separated parsing - could be enhanced
		return []string{value}
	}
	return defaultValue
}
