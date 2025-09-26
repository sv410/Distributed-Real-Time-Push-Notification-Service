package provider

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/sv410/Distributed-Real-Time-Push-Notification-Service/pkg"
)

// Provider defines the interface for external notification providers
type Provider interface {
	Name() string
	Send(ctx context.Context, notification *pkg.NotificationMessage) (*pkg.ProviderResponse, error)
	HealthCheck(ctx context.Context) error
}

// MockProvider simulates an external notification provider
type MockProvider struct {
	name          string
	successRate   float64 // 0.0 to 1.0
	avgLatency    time.Duration
	latencyJitter time.Duration
	healthStatus  bool
}

// NewMockProvider creates a new mock provider with configurable behavior
func NewMockProvider(name string, successRate float64, avgLatency, latencyJitter time.Duration) *MockProvider {
	return &MockProvider{
		name:          name,
		successRate:   successRate,
		avgLatency:    avgLatency,
		latencyJitter: latencyJitter,
		healthStatus:  true,
	}
}

// Name returns the provider name
func (mp *MockProvider) Name() string {
	return mp.name
}

// Send simulates sending a notification through the provider
func (mp *MockProvider) Send(ctx context.Context, notification *pkg.NotificationMessage) (*pkg.ProviderResponse, error) {
	// Simulate network latency
	latency := mp.avgLatency
	if mp.latencyJitter > 0 {
		jitter := time.Duration(rand.Int63n(int64(mp.latencyJitter)))
		latency += jitter
	}

	select {
	case <-time.After(latency):
		// Continue with processing
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Simulate success/failure based on success rate
	success := rand.Float64() < mp.successRate

	response := &pkg.ProviderResponse{
		Success: success,
	}

	if success {
		// Generate a mock message ID
		response.MessageID = fmt.Sprintf("%s_%d_%s", mp.name, time.Now().Unix(), notification.ID[:8])
	} else {
		// Simulate different types of failures
		failures := []string{
			"network timeout",
			"rate limit exceeded",
			"invalid token",
			"service unavailable",
			"message too large",
		}
		response.Error = failures[rand.Intn(len(failures))]
	}

	return response, nil
}

// HealthCheck simulates a health check for the provider
func (mp *MockProvider) HealthCheck(ctx context.Context) error {
	// Simulate some latency for health check
	select {
	case <-time.After(50 * time.Millisecond):
		// Continue
	case <-ctx.Done():
		return ctx.Err()
	}

	if !mp.healthStatus {
		return fmt.Errorf("provider %s is unhealthy", mp.name)
	}

	// Random health check failures (5% chance)
	if rand.Float64() < 0.05 {
		return fmt.Errorf("provider %s health check failed", mp.name)
	}

	return nil
}

// SetHealthStatus allows controlling the health status for testing
func (mp *MockProvider) SetHealthStatus(healthy bool) {
	mp.healthStatus = healthy
}

// ProviderManager manages multiple providers and provides load balancing
type ProviderManager struct {
	providers []Provider
	strategy  LoadBalanceStrategy
}

// LoadBalanceStrategy defines the load balancing strategy
type LoadBalanceStrategy int

const (
	RoundRobin LoadBalanceStrategy = iota
	Random
	HealthBased
)

// NewProviderManager creates a new provider manager
func NewProviderManager(strategy LoadBalanceStrategy) *ProviderManager {
	return &ProviderManager{
		providers: make([]Provider, 0),
		strategy:  strategy,
	}
}

// AddProvider adds a provider to the manager
func (pm *ProviderManager) AddProvider(provider Provider) {
	pm.providers = append(pm.providers, provider)
}

// GetProvider returns a provider based on the load balancing strategy
func (pm *ProviderManager) GetProvider(ctx context.Context) (Provider, error) {
	if len(pm.providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	switch pm.strategy {
	case RoundRobin:
		// Simple round-robin (not thread-safe, but OK for demo)
		index := rand.Intn(len(pm.providers))
		return pm.providers[index], nil

	case Random:
		index := rand.Intn(len(pm.providers))
		return pm.providers[index], nil

	case HealthBased:
		// Try to find a healthy provider
		for _, provider := range pm.providers {
			if err := provider.HealthCheck(ctx); err == nil {
				return provider, nil
			}
		}
		// If no healthy providers, return the first one
		return pm.providers[0], nil

	default:
		return pm.providers[0], nil
	}
}

// GetAllProviders returns all registered providers
func (pm *ProviderManager) GetAllProviders() []Provider {
	return pm.providers
}

// HealthCheckAll performs health checks on all providers
func (pm *ProviderManager) HealthCheckAll(ctx context.Context) map[string]error {
	results := make(map[string]error)

	for _, provider := range pm.providers {
		results[provider.Name()] = provider.HealthCheck(ctx)
	}

	return results
}
