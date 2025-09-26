// Consumer Service - Processes push notifications from Kafka queue
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"notification-service/internal/kafka"
	"notification-service/internal/redis"
	"notification-service/pkg/config"
	"notification-service/pkg/models"
)

// Consumer handles notification processing
type Consumer struct {
	kafkaConsumer *kafka.Consumer
	redisClient   *redis.Client
	logger        *logrus.Logger
	maxRetries    int
}

// NewConsumer creates a new consumer instance
func NewConsumer(kafkaConsumer *kafka.Consumer, redisClient *redis.Client, logger *logrus.Logger) *Consumer {
	return &Consumer{
		kafkaConsumer: kafkaConsumer,
		redisClient:   redisClient,
		logger:        logger,
		maxRetries:    3,
	}
}

// Start begins processing messages
func (c *Consumer) Start(ctx context.Context, workerCount int) error {
	c.logger.WithField("workers", workerCount).Info("Starting notification consumer")

	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			c.worker(ctx, workerID)
		}(i + 1)
	}

	// Wait for all workers to finish
	wg.Wait()
	return nil
}

// worker processes messages from Kafka
func (c *Consumer) worker(ctx context.Context, workerID int) {
	logger := c.logger.WithField("worker_id", workerID)
	logger.Info("Worker started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("Worker shutting down")
			return
		default:
			// Process message
			if err := c.processMessage(ctx, logger); err != nil {
				logger.WithError(err).Error("Failed to process message")
				time.Sleep(1 * time.Second) // Brief pause on error
			}
		}
	}
}

// processMessage processes a single message from Kafka
func (c *Consumer) processMessage(ctx context.Context, logger *logrus.Entry) error {
	msg, err := c.kafkaConsumer.Consume()
	if err != nil {
		// Handle consume timeout gracefully
		if err.Error() == "consume timeout" {
			return nil
		}
		return fmt.Errorf("failed to consume message: %w", err)
	}

	// Parse notification
	var notification models.Notification
	if err := json.Unmarshal(msg.Value, &notification); err != nil {
		logger.WithError(err).Error("Failed to unmarshal notification")
		// Commit the message anyway to avoid reprocessing invalid data
		c.kafkaConsumer.Commit(msg)
		return nil
	}

	logger = logger.WithFields(logrus.Fields{
		"notification_id": notification.ID,
		"user_id":         notification.UserID,
		"title":           notification.Title,
	})

	// Check if notification is scheduled
	if notification.IsScheduled() {
		logger.Info("Notification is scheduled for future delivery")
		// In a real implementation, you would re-queue the message or use a delay queue
		// For now, we'll just skip it
		c.kafkaConsumer.Commit(msg)
		return nil
	}

	// Process the notification
	if err := c.processNotification(ctx, &notification, logger); err != nil {
		logger.WithError(err).Error("Failed to process notification")

		// Handle retry logic
		if notification.CanRetry(c.maxRetries) {
			notification.IncrementRetry()
			logger.WithField("retry_count", notification.RetryCount).Warn("Retrying notification")
			
			// In a real implementation, you would send back to a retry topic
			// For now, we'll just log and continue
		} else {
			notification.MarkAsFailed(err.Error())
			logger.Error("Max retries exceeded, marking as failed")
		}
	} else {
		notification.MarkAsSent()
		logger.Info("Notification processed successfully")
	}

	// Update notification status in Redis
	if err := c.redisClient.SetNotificationStatus(ctx, notification.ID, notification.Status, 24*time.Hour); err != nil {
		logger.WithError(err).Warn("Failed to update notification status in Redis")
	}

	// Commit the message
	if err := c.kafkaConsumer.Commit(msg); err != nil {
		logger.WithError(err).Error("Failed to commit message")
		return err
	}

	return nil
}

// processNotification handles the actual notification delivery
func (c *Consumer) processNotification(ctx context.Context, notification *models.Notification, logger *logrus.Entry) error {
	// Get user session
	session, err := c.redisClient.GetUserSession(ctx, notification.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user session: %w", err)
	}

	if !session.IsActive {
		return fmt.Errorf("user session is not active")
	}

	// Simulate notification delivery based on platform
	switch session.Platform {
	case "ios":
		return c.sendIOSNotification(notification, session, logger)
	case "android":
		return c.sendAndroidNotification(notification, session, logger)
	case "web":
		return c.sendWebNotification(notification, session, logger)
	default:
		return fmt.Errorf("unsupported platform: %s", session.Platform)
	}
}

// sendIOSNotification simulates sending notification to iOS device
func (c *Consumer) sendIOSNotification(notification *models.Notification, session *models.UserSession, logger *logrus.Entry) error {
	logger.Info("Sending iOS push notification")
	
	// Simulate processing time
	time.Sleep(50 * time.Millisecond)
	
	// In a real implementation, you would use Apple Push Notification service (APNs)
	logger.WithFields(logrus.Fields{
		"device_token": session.DeviceToken,
		"platform":     "ios",
	}).Info("iOS notification sent successfully")
	
	return nil
}

// sendAndroidNotification simulates sending notification to Android device
func (c *Consumer) sendAndroidNotification(notification *models.Notification, session *models.UserSession, logger *logrus.Entry) error {
	logger.Info("Sending Android push notification")
	
	// Simulate processing time
	time.Sleep(30 * time.Millisecond)
	
	// In a real implementation, you would use Firebase Cloud Messaging (FCM)
	logger.WithFields(logrus.Fields{
		"device_token": session.DeviceToken,
		"platform":     "android",
	}).Info("Android notification sent successfully")
	
	return nil
}

// sendWebNotification simulates sending notification to web browser
func (c *Consumer) sendWebNotification(notification *models.Notification, session *models.UserSession, logger *logrus.Entry) error {
	logger.Info("Sending web push notification")
	
	// Simulate processing time
	time.Sleep(20 * time.Millisecond)
	
	// In a real implementation, you would use Web Push Protocol
	logger.WithFields(logrus.Fields{
		"device_token": session.DeviceToken,
		"platform":     "web",
	}).Info("Web notification sent successfully")
	
	return nil
}

func main() {
	// Load configuration
	cfg := config.GetDefaultConfig()
	if configFile := os.Getenv("CONFIG_FILE"); configFile != "" {
		loadedCfg, err := config.Load(configFile)
		if err != nil {
			logrus.WithError(err).Warn("Failed to load config file, using defaults")
		} else {
			cfg = loadedCfg
		}
	}

	// Setup logger
	logger := logrus.New()
	level, err := logrus.ParseLevel(cfg.Log.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)
	
	if cfg.Log.File != "" {
		file, err := os.OpenFile(cfg.Log.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			logger.SetOutput(file)
		} else {
			logger.WithError(err).Warn("Failed to open log file, using stdout")
		}
	}

	logger.Info("Starting Notification Consumer")

	// Initialize Redis client
	redisClient, err := redis.NewClient(cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.Password, cfg.Redis.DB, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize Redis client")
	}
	defer redisClient.Close()

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx); err != nil {
		logger.WithError(err).Fatal("Failed to connect to Redis")
	}
	logger.Info("Connected to Redis successfully")

	// Initialize Kafka consumer
	kafkaConsumer, err := kafka.NewConsumer(cfg.Kafka.BootstrapServers, cfg.Kafka.Topic, cfg.Kafka.GroupID, cfg.Kafka.AutoOffsetReset, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize Kafka consumer")
	}
	defer kafkaConsumer.Close()
	logger.Info("Kafka consumer initialized successfully")

	// Create consumer
	consumer := NewConsumer(kafkaConsumer, redisClient, logger)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consumer in a goroutine
	go func() {
		// Use multiple workers for concurrent processing (supporting 10,000 notifications per minute)
		workerCount := 10
		if err := consumer.Start(ctx, workerCount); err != nil {
			logger.WithError(err).Fatal("Consumer failed")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down consumer...")
	cancel()

	// Give some time for workers to finish
	time.Sleep(5 * time.Second)
	logger.Info("Consumer exited")
}