package pkg

import (
	"testing"
	"time"
)

func TestNotificationMessage(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(1 * time.Hour)

	msg := &NotificationMessage{
		ID:        "test-123",
		UserID:    "user-456",
		Type:      "push",
		Title:     "Test Notification",
		Body:      "This is a test notification",
		Data:      map[string]interface{}{"key": "value"},
		Priority:  PriorityHigh,
		CreatedAt: now,
		ExpiresAt: &expiresAt,
		Retry:     0,
	}

	if msg.ID != "test-123" {
		t.Errorf("Expected ID to be 'test-123', got %s", msg.ID)
	}

	if msg.UserID != "user-456" {
		t.Errorf("Expected UserID to be 'user-456', got %s", msg.UserID)
	}

	if msg.Priority != PriorityHigh {
		t.Errorf("Expected Priority to be PriorityHigh, got %v", msg.Priority)
	}

	if msg.Data["key"] != "value" {
		t.Errorf("Expected Data['key'] to be 'value', got %v", msg.Data["key"])
	}
}

func TestPriorityString(t *testing.T) {
	tests := []struct {
		priority Priority
		expected string
	}{
		{PriorityLow, "low"},
		{PriorityNormal, "normal"},
		{PriorityHigh, "high"},
		{PriorityUrgent, "urgent"},
		{Priority(999), "normal"}, // Unknown priority should default to normal
	}

	for _, test := range tests {
		result := test.priority.String()
		if result != test.expected {
			t.Errorf("Expected priority %d to be '%s', got '%s'", test.priority, test.expected, result)
		}
	}
}

func TestProviderResponse(t *testing.T) {
	// Test successful response
	successResponse := &ProviderResponse{
		Success:   true,
		MessageID: "msg-123",
	}

	if !successResponse.Success {
		t.Errorf("Expected Success to be true")
	}

	if successResponse.MessageID != "msg-123" {
		t.Errorf("Expected MessageID to be 'msg-123', got %s", successResponse.MessageID)
	}

	// Test error response
	errorResponse := &ProviderResponse{
		Success: false,
		Error:   "network timeout",
	}

	if errorResponse.Success {
		t.Errorf("Expected Success to be false")
	}

	if errorResponse.Error != "network timeout" {
		t.Errorf("Expected Error to be 'network timeout', got %s", errorResponse.Error)
	}
}

func TestProcessingResult(t *testing.T) {
	now := time.Now()

	result := &ProcessingResult{
		MessageID:   "msg-123",
		UserID:      "user-456",
		Success:     true,
		Provider:    "firebase",
		ProcessedAt: now,
		Attempts:    2,
	}

	if result.MessageID != "msg-123" {
		t.Errorf("Expected MessageID to be 'msg-123', got %s", result.MessageID)
	}

	if result.UserID != "user-456" {
		t.Errorf("Expected UserID to be 'user-456', got %s", result.UserID)
	}

	if !result.Success {
		t.Errorf("Expected Success to be true")
	}

	if result.Provider != "firebase" {
		t.Errorf("Expected Provider to be 'firebase', got %s", result.Provider)
	}

	if result.Attempts != 2 {
		t.Errorf("Expected Attempts to be 2, got %d", result.Attempts)
	}
}
