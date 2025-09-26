package pkg

import (
	"time"
)

// NotificationMessage represents a notification to be processed
type NotificationMessage struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Type      string                 `json:"type"`
	Title     string                 `json:"title"`
	Body      string                 `json:"body"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Priority  Priority               `json:"priority"`
	CreatedAt time.Time              `json:"created_at"`
	ExpiresAt *time.Time             `json:"expires_at,omitempty"`
	Retry     int                    `json:"retry"`
}

// Priority defines notification priority levels
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityUrgent
)

// String returns string representation of priority
func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityNormal:
		return "normal"
	case PriorityHigh:
		return "high"
	case PriorityUrgent:
		return "urgent"
	default:
		return "normal"
	}
}

// ProviderResponse represents the response from external providers
type ProviderResponse struct {
	Success   bool   `json:"success"`
	MessageID string `json:"message_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

// ProcessingResult represents the result of processing a notification
type ProcessingResult struct {
	MessageID   string
	UserID      string
	Success     bool
	Provider    string
	Error       error
	ProcessedAt time.Time
	Attempts    int
}
