package models

import (
	"testing"
	"time"
)

func TestNewNotification(t *testing.T) {
	req := &NotificationRequest{
		UserID:   "user123",
		Title:    "Test Title",
		Message:  "Test Message",
		Priority: PriorityHigh,
		Data: map[string]string{
			"type": "test",
		},
	}

	notification := NewNotification(req)

	if notification.ID == "" {
		t.Error("Notification ID should not be empty")
	}

	if notification.UserID != req.UserID {
		t.Errorf("Expected UserID %s, got %s", req.UserID, notification.UserID)
	}

	if notification.Title != req.Title {
		t.Errorf("Expected Title %s, got %s", req.Title, notification.Title)
	}

	if notification.Message != req.Message {
		t.Errorf("Expected Message %s, got %s", req.Message, notification.Message)
	}

	if notification.Priority != req.Priority {
		t.Errorf("Expected Priority %s, got %s", req.Priority, notification.Priority)
	}

	if notification.Status != StatusPending {
		t.Errorf("Expected Status %s, got %s", StatusPending, notification.Status)
	}

	if notification.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestNotification_IsScheduled(t *testing.T) {
	// Test not scheduled
	notification := &Notification{}
	if notification.IsScheduled() {
		t.Error("Notification without ScheduleAt should not be scheduled")
	}

	// Test scheduled in past
	past := time.Now().Add(-1 * time.Hour)
	notification.ScheduleAt = &past
	if notification.IsScheduled() {
		t.Error("Notification scheduled in past should not be considered scheduled")
	}

	// Test scheduled in future
	future := time.Now().Add(1 * time.Hour)
	notification.ScheduleAt = &future
	if !notification.IsScheduled() {
		t.Error("Notification scheduled in future should be considered scheduled")
	}
}

func TestNotification_CanRetry(t *testing.T) {
	notification := &Notification{RetryCount: 0}
	maxRetries := 3

	// Test can retry
	if !notification.CanRetry(maxRetries) {
		t.Error("Notification with 0 retries should be able to retry")
	}

	// Test at max retries
	notification.RetryCount = maxRetries
	if notification.CanRetry(maxRetries) {
		t.Error("Notification at max retries should not be able to retry")
	}

	// Test beyond max retries
	notification.RetryCount = maxRetries + 1
	if notification.CanRetry(maxRetries) {
		t.Error("Notification beyond max retries should not be able to retry")
	}
}

func TestNotification_MarkAsSent(t *testing.T) {
	notification := &Notification{Status: StatusPending}
	notification.MarkAsSent()

	if notification.Status != StatusSent {
		t.Errorf("Expected Status %s, got %s", StatusSent, notification.Status)
	}

	if notification.SentAt == nil {
		t.Error("SentAt should be set when marked as sent")
	}
}

func TestNotification_MarkAsFailed(t *testing.T) {
	notification := &Notification{Status: StatusPending}
	errorMsg := "Test error"
	notification.MarkAsFailed(errorMsg)

	if notification.Status != StatusFailed {
		t.Errorf("Expected Status %s, got %s", StatusFailed, notification.Status)
	}

	if notification.Error != errorMsg {
		t.Errorf("Expected Error %s, got %s", errorMsg, notification.Error)
	}
}

func TestNotification_IncrementRetry(t *testing.T) {
	notification := &Notification{RetryCount: 0, Status: StatusPending}
	notification.IncrementRetry()

	if notification.RetryCount != 1 {
		t.Errorf("Expected RetryCount 1, got %d", notification.RetryCount)
	}

	if notification.Status != StatusRetry {
		t.Errorf("Expected Status %s, got %s", StatusRetry, notification.Status)
	}
}