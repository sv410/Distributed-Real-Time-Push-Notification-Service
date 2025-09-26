package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/sv410/Distributed-Real-Time-Push-Notification-Service/internal/provider"
	"github.com/sv410/Distributed-Real-Time-Push-Notification-Service/internal/redis"
	"github.com/sv410/Distributed-Real-Time-Push-Notification-Service/pkg"
)

// Pool represents a worker pool for processing notifications
type Pool struct {
	workers     int
	jobQueue    chan *pkg.NotificationMessage
	resultQueue chan *pkg.ProcessingResult
	errorQueue  chan error
	quit        chan bool
	wg          sync.WaitGroup

	rateLimiter     *redis.RateLimiter
	providerManager *provider.ProviderManager

	retryAttempts int
	retryDelay    time.Duration

	// Metrics
	processed   int64
	failed      int64
	rateLimited int64
	mu          sync.RWMutex
}

// NewPool creates a new worker pool
func NewPool(workers, maxQueueSize int, rateLimiter *redis.RateLimiter, providerManager *provider.ProviderManager, retryAttempts int, retryDelay time.Duration) *Pool {
	return &Pool{
		workers:         workers,
		jobQueue:        make(chan *pkg.NotificationMessage, maxQueueSize),
		resultQueue:     make(chan *pkg.ProcessingResult, maxQueueSize),
		errorQueue:      make(chan error, maxQueueSize),
		quit:            make(chan bool),
		rateLimiter:     rateLimiter,
		providerManager: providerManager,
		retryAttempts:   retryAttempts,
		retryDelay:      retryDelay,
	}
}

// Start starts the worker pool
func (p *Pool) Start(ctx context.Context) {
	log.Printf("Starting worker pool with %d workers", p.workers)

	// Start workers
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}

	log.Printf("Worker pool started with %d workers", p.workers)
}

// Stop stops the worker pool
func (p *Pool) Stop() {
	log.Println("Stopping worker pool...")
	close(p.quit)
	p.wg.Wait()
	close(p.jobQueue)
	close(p.resultQueue)
	close(p.errorQueue)
	log.Println("Worker pool stopped")
}

// Submit submits a job to the worker pool
func (p *Pool) Submit(notification *pkg.NotificationMessage) error {
	select {
	case p.jobQueue <- notification:
		return nil
	default:
		return fmt.Errorf("job queue is full")
	}
}

// Results returns the result channel
func (p *Pool) Results() <-chan *pkg.ProcessingResult {
	return p.resultQueue
}

// Errors returns the error channel
func (p *Pool) Errors() <-chan error {
	return p.errorQueue
}

// GetMetrics returns current metrics
func (p *Pool) GetMetrics() (processed, failed, rateLimited int64) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.processed, p.failed, p.rateLimited
}

// worker is the main worker function
func (p *Pool) worker(ctx context.Context, workerID int) {
	defer p.wg.Done()

	log.Printf("Worker %d started", workerID)
	defer log.Printf("Worker %d stopped", workerID)

	for {
		select {
		case <-p.quit:
			return
		case <-ctx.Done():
			return
		case job := <-p.jobQueue:
			if job == nil {
				return
			}
			p.processNotification(ctx, workerID, job)
		}
	}
}

// processNotification processes a single notification
func (p *Pool) processNotification(ctx context.Context, workerID int, notification *pkg.NotificationMessage) {
	startTime := time.Now()

	log.Printf("Worker %d processing notification %s for user %s", workerID, notification.ID, notification.UserID)

	// Check if notification has expired
	if notification.ExpiresAt != nil && time.Now().After(*notification.ExpiresAt) {
		p.sendError(fmt.Errorf("notification %s expired", notification.ID))
		return
	}

	// Check rate limiting
	allowed, err := p.rateLimiter.IsAllowed(ctx, notification.UserID)
	if err != nil {
		p.sendError(fmt.Errorf("rate limiter error for user %s: %w", notification.UserID, err))
		return
	}

	if !allowed {
		p.mu.Lock()
		p.rateLimited++
		p.mu.Unlock()

		result := &pkg.ProcessingResult{
			MessageID:   notification.ID,
			UserID:      notification.UserID,
			Success:     false,
			Error:       fmt.Errorf("rate limit exceeded for user %s", notification.UserID),
			ProcessedAt: time.Now(),
			Attempts:    notification.Retry + 1,
		}
		p.sendResult(result)
		return
	}

	// Get a provider
	selectedProvider, err := p.providerManager.GetProvider(ctx)
	if err != nil {
		p.sendError(fmt.Errorf("failed to get provider: %w", err))
		return
	}

	// Attempt to send notification with retries
	var lastErr error
	maxAttempts := p.retryAttempts + 1 // +1 for initial attempt

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			// Add exponential backoff for retries
			delay := p.retryDelay * time.Duration(attempt-1)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return
			}
		}

		// Create a timeout context for the provider call
		providerCtx, cancel := context.WithTimeout(ctx, 10*time.Second)

		response, err := selectedProvider.Send(providerCtx, notification)
		cancel()

		if err != nil {
			lastErr = err
			log.Printf("Worker %d: Attempt %d failed for notification %s: %v", workerID, attempt, notification.ID, err)
			continue
		}

		// Process provider response
		result := &pkg.ProcessingResult{
			MessageID:   notification.ID,
			UserID:      notification.UserID,
			Success:     response.Success,
			Provider:    selectedProvider.Name(),
			ProcessedAt: time.Now(),
			Attempts:    attempt,
		}

		if response.Success {
			p.mu.Lock()
			p.processed++
			p.mu.Unlock()

			log.Printf("Worker %d: Successfully sent notification %s via %s (took %v)",
				workerID, notification.ID, selectedProvider.Name(), time.Since(startTime))
		} else {
			result.Error = fmt.Errorf("provider error: %s", response.Error)

			p.mu.Lock()
			p.failed++
			p.mu.Unlock()

			log.Printf("Worker %d: Failed to send notification %s via %s: %s",
				workerID, notification.ID, selectedProvider.Name(), response.Error)
		}

		p.sendResult(result)
		return
	}

	// All attempts failed
	result := &pkg.ProcessingResult{
		MessageID:   notification.ID,
		UserID:      notification.UserID,
		Success:     false,
		Provider:    selectedProvider.Name(),
		Error:       fmt.Errorf("all %d attempts failed, last error: %w", maxAttempts, lastErr),
		ProcessedAt: time.Now(),
		Attempts:    maxAttempts,
	}

	p.mu.Lock()
	p.failed++
	p.mu.Unlock()

	p.sendResult(result)
}

// sendResult sends a result to the result channel without blocking
func (p *Pool) sendResult(result *pkg.ProcessingResult) {
	select {
	case p.resultQueue <- result:
	default:
		log.Printf("Result queue full, dropping result for message %s", result.MessageID)
	}
}

// sendError sends an error to the error channel without blocking
func (p *Pool) sendError(err error) {
	select {
	case p.errorQueue <- err:
	default:
		log.Printf("Error queue full, dropping error: %v", err)
	}
}

// QueueSize returns the current size of the job queue
func (p *Pool) QueueSize() int {
	return len(p.jobQueue)
}

// IsHealthy performs a basic health check
func (p *Pool) IsHealthy(ctx context.Context) error {
	// Check if workers are running
	select {
	case <-p.quit:
		return fmt.Errorf("worker pool is stopped")
	default:
	}

	// Check provider health
	healthResults := p.providerManager.HealthCheckAll(ctx)
	healthyCount := 0
	for _, err := range healthResults {
		if err == nil {
			healthyCount++
		}
	}

	if healthyCount == 0 {
		return fmt.Errorf("no healthy providers available")
	}

	return nil
}
