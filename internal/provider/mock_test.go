package provider

import (
	"context"
	"testing"
	"time"

	"github.com/sv410/Distributed-Real-Time-Push-Notification-Service/pkg"
)

func TestMockProvider(t *testing.T) {
	// Create a mock provider with 100% success rate
	provider := NewMockProvider("test-provider", 1.0, 50*time.Millisecond, 10*time.Millisecond)

	if provider.Name() != "test-provider" {
		t.Errorf("Expected provider name to be 'test-provider', got %s", provider.Name())
	}

	ctx := context.Background()
	notification := &pkg.NotificationMessage{
		ID:     "test-123",
		UserID: "user-456",
		Title:  "Test Notification",
		Body:   "This is a test",
	}

	// Test successful send
	response, err := provider.Send(ctx, notification)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !response.Success {
		t.Errorf("Expected successful response")
	}

	if response.MessageID == "" {
		t.Errorf("Expected message ID to be set")
	}

	// Test health check
	err = provider.HealthCheck(ctx)
	if err != nil {
		t.Errorf("Expected health check to pass, got %v", err)
	}
}

func TestMockProviderFailure(t *testing.T) {
	// Create a mock provider with 0% success rate
	provider := NewMockProvider("failing-provider", 0.0, 10*time.Millisecond, 0)

	ctx := context.Background()
	notification := &pkg.NotificationMessage{
		ID:     "test-123",
		UserID: "user-456",
		Title:  "Test Notification",
		Body:   "This is a test",
	}

	// Test failed send
	response, err := provider.Send(ctx, notification)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if response.Success {
		t.Errorf("Expected failed response")
	}

	if response.Error == "" {
		t.Errorf("Expected error message to be set")
	}
}

func TestProviderManager(t *testing.T) {
	manager := NewProviderManager(Random)

	// Test empty manager
	_, err := manager.GetProvider(context.Background())
	if err == nil {
		t.Errorf("Expected error when no providers available")
	}

	// Add providers
	provider1 := NewMockProvider("provider1", 1.0, 10*time.Millisecond, 0)
	provider2 := NewMockProvider("provider2", 1.0, 10*time.Millisecond, 0)

	manager.AddProvider(provider1)
	manager.AddProvider(provider2)

	// Test getting provider
	provider, err := manager.GetProvider(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if provider == nil {
		t.Errorf("Expected provider to be returned")
	}

	// Test health check all
	results := manager.HealthCheckAll(context.Background())
	if len(results) != 2 {
		t.Errorf("Expected 2 health check results, got %d", len(results))
	}

	for name, err := range results {
		if err != nil {
			t.Errorf("Expected provider %s to be healthy, got %v", name, err)
		}
	}
}

func TestProviderManagerHealthBased(t *testing.T) {
	manager := NewProviderManager(HealthBased)

	// Add healthy and unhealthy providers
	healthyProvider := NewMockProvider("healthy", 1.0, 10*time.Millisecond, 0)
	unhealthyProvider := NewMockProvider("unhealthy", 1.0, 10*time.Millisecond, 0)
	unhealthyProvider.SetHealthStatus(false)

	manager.AddProvider(unhealthyProvider) // Add unhealthy first
	manager.AddProvider(healthyProvider)   // Add healthy second

	// Should return healthy provider
	provider, err := manager.GetProvider(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if provider.Name() != "healthy" {
		t.Errorf("Expected healthy provider, got %s", provider.Name())
	}
}
