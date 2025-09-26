// Package services provides business logic services
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"notification-service/internal/kafka"
	"notification-service/internal/redis"
	"notification-service/pkg/models"
)

// NotificationService handles notification business logic
type NotificationService struct {
	producer    *kafka.Producer
	redisClient *redis.Client
	logger      *logrus.Logger
}

// NewNotificationService creates a new notification service
func NewNotificationService(producer *kafka.Producer, redisClient *redis.Client, logger *logrus.Logger) *NotificationService {
	return &NotificationService{
		producer:    producer,
		redisClient: redisClient,
		logger:      logger,
	}
}

// SendNotification processes and sends a notification request
func (s *NotificationService) SendNotification(ctx context.Context, req *models.NotificationRequest) (*models.Notification, error) {
	// Validate user session exists
	session, err := s.redisClient.GetUserSession(ctx, req.UserID)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", req.UserID).Error("User session not found")
		return nil, fmt.Errorf("user session not found: %w", err)
	}

	if !session.IsActive {
		return nil, fmt.Errorf("user session is not active")
	}

	// Create notification
	notification := models.NewNotification(req)

	// Check rate limiting
	if err := s.checkRateLimit(ctx, req.UserID); err != nil {
		s.logger.WithError(err).WithField("user_id", req.UserID).Warn("Rate limit exceeded")
		return nil, err
	}

	// Store notification status in Redis for tracking
	if err := s.redisClient.SetNotificationStatus(ctx, notification.ID, notification.Status, 24*time.Hour); err != nil {
		s.logger.WithError(err).WithField("notification_id", notification.ID).Warn("Failed to store notification status")
	}

	// Send to Kafka for processing
	if err := s.producer.Produce(notification.UserID, notification); err != nil {
		notification.MarkAsFailed(err.Error())
		s.logger.WithError(err).WithField("notification_id", notification.ID).Error("Failed to send notification to Kafka")
		return notification, fmt.Errorf("failed to queue notification: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"notification_id": notification.ID,
		"user_id":         notification.UserID,
		"title":           notification.Title,
	}).Info("Notification queued successfully")

	return notification, nil
}

// GetNotificationStatus retrieves the status of a notification
func (s *NotificationService) GetNotificationStatus(ctx context.Context, notificationID string) (models.NotificationStatus, error) {
	status, err := s.redisClient.GetNotificationStatus(ctx, notificationID)
	if err != nil {
		return "", fmt.Errorf("failed to get notification status: %w", err)
	}
	return status, nil
}

// checkRateLimit checks if the user has exceeded the rate limit
func (s *NotificationService) checkRateLimit(ctx context.Context, userID string) error {
	key := fmt.Sprintf("rate_limit:%s", userID)
	
	// Allow 100 notifications per minute per user
	count, err := s.redisClient.IncrementCounter(ctx, key, time.Minute)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", userID).Warn("Failed to check rate limit")
		return nil // Don't block on rate limit errors
	}

	if count > 100 {
		return fmt.Errorf("rate limit exceeded: %d notifications in the last minute", count)
	}

	return nil
}

// RegisterUserSession registers a new user session
func (s *NotificationService) RegisterUserSession(ctx context.Context, userID, deviceToken, platform string) error {
	session := &models.UserSession{
		UserID:      userID,
		DeviceToken: deviceToken,
		Platform:    platform,
		IsActive:    true,
		LastSeen:    time.Now(),
		CreatedAt:   time.Now(),
	}

	// Store session for 24 hours
	if err := s.redisClient.SetUserSession(ctx, userID, session, 24*time.Hour); err != nil {
		return fmt.Errorf("failed to register user session: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":  userID,
		"platform": platform,
	}).Info("User session registered successfully")

	return nil
}

// UnregisterUserSession removes a user session
func (s *NotificationService) UnregisterUserSession(ctx context.Context, userID string) error {
	if err := s.redisClient.DeleteUserSession(ctx, userID); err != nil {
		return fmt.Errorf("failed to unregister user session: %w", err)
	}

	s.logger.WithField("user_id", userID).Info("User session unregistered successfully")
	return nil
}