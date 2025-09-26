// Package models defines the data structures used throughout the notification service
package models

import (
	"time"

	"github.com/google/uuid"
)

// NotificationRequest represents an incoming push notification request
type NotificationRequest struct {
	UserID    string            `json:"user_id" binding:"required"`
	Title     string            `json:"title" binding:"required"`
	Message   string            `json:"message" binding:"required"`
	Data      map[string]string `json:"data,omitempty"`
	Priority  NotificationPriority `json:"priority,omitempty"`
	ScheduleAt *time.Time        `json:"schedule_at,omitempty"`
}

// Notification represents a processed notification
type Notification struct {
	ID         string               `json:"id"`
	UserID     string               `json:"user_id"`
	Title      string               `json:"title"`
	Message    string               `json:"message"`
	Data       map[string]string    `json:"data,omitempty"`
	Priority   NotificationPriority `json:"priority"`
	Status     NotificationStatus   `json:"status"`
	CreatedAt  time.Time            `json:"created_at"`
	SentAt     *time.Time           `json:"sent_at,omitempty"`
	ScheduleAt *time.Time           `json:"schedule_at,omitempty"`
	RetryCount int                  `json:"retry_count"`
	Error      string               `json:"error,omitempty"`
}

// NotificationPriority represents the priority level of a notification
type NotificationPriority string

const (
	PriorityHigh   NotificationPriority = "high"
	PriorityNormal NotificationPriority = "normal"
	PriorityLow    NotificationPriority = "low"
)

// NotificationStatus represents the status of a notification
type NotificationStatus string

const (
	StatusPending NotificationStatus = "pending"
	StatusSent    NotificationStatus = "sent"
	StatusFailed  NotificationStatus = "failed"
	StatusRetry   NotificationStatus = "retry"
)

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// UserSession represents a user session stored in Redis
type UserSession struct {
	UserID       string    `json:"user_id"`
	DeviceToken  string    `json:"device_token"`
	Platform     string    `json:"platform"` // ios, android, web
	IsActive     bool      `json:"is_active"`
	LastSeen     time.Time `json:"last_seen"`
	CreatedAt    time.Time `json:"created_at"`
}

// NewNotification creates a new notification from a request
func NewNotification(req *NotificationRequest) *Notification {
	now := time.Now()
	notification := &Notification{
		ID:        uuid.New().String(),
		UserID:    req.UserID,
		Title:     req.Title,
		Message:   req.Message,
		Data:      req.Data,
		Priority:  req.Priority,
		Status:    StatusPending,
		CreatedAt: now,
	}

	if req.Priority == "" {
		notification.Priority = PriorityNormal
	}

	if req.ScheduleAt != nil {
		notification.ScheduleAt = req.ScheduleAt
	}

	return notification
}

// IsScheduled checks if the notification is scheduled for future delivery
func (n *Notification) IsScheduled() bool {
	return n.ScheduleAt != nil && n.ScheduleAt.After(time.Now())
}

// CanRetry checks if the notification can be retried based on retry count
func (n *Notification) CanRetry(maxRetries int) bool {
	return n.RetryCount < maxRetries
}

// MarkAsSent marks the notification as successfully sent
func (n *Notification) MarkAsSent() {
	now := time.Now()
	n.Status = StatusSent
	n.SentAt = &now
}

// MarkAsFailed marks the notification as failed
func (n *Notification) MarkAsFailed(err string) {
	n.Status = StatusFailed
	n.Error = err
}

// IncrementRetry increments the retry count and sets status to retry
func (n *Notification) IncrementRetry() {
	n.RetryCount++
	n.Status = StatusRetry
}