package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"

	"github.com/sv410/Distributed-Real-Time-Push-Notification-Service/internal/config"
	"github.com/sv410/Distributed-Real-Time-Push-Notification-Service/internal/kafka"
	"github.com/sv410/Distributed-Real-Time-Push-Notification-Service/internal/provider"
	redisLib "github.com/sv410/Distributed-Real-Time-Push-Notification-Service/internal/redis"
	"github.com/sv410/Distributed-Real-Time-Push-Notification-Service/internal/worker"
	"github.com/sv410/Distributed-Real-Time-Push-Notification-Service/pkg"
)

// Service represents the main notification service
type Service struct {
	config          *config.Config
	workerPool      *worker.Pool
	kafkaConsumer   *kafka.Consumer
	kafkaProducer   *kafka.Producer
	rateLimiter     *redisLib.RateLimiter
	redisClient     *redis.Client
	providerManager *provider.ProviderManager
	httpServer      *http.Server

	// Channels
	messageChan chan *pkg.NotificationMessage
	errorChan   chan error

	// Context and cancellation
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewService creates a new notification service
func NewService() (*Service, error) {
	cfg := config.LoadConfig()

	// Create context
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize Redis client
	redisClient := redisLib.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)

	// Test Redis connection
	if err := redisLib.HealthCheck(ctx, redisClient); err != nil {
		cancel() // Clean up context
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}
	log.Println("Redis connection established")

	// Initialize rate limiter
	rateLimiter := redisLib.NewRateLimiter(redisClient, cfg.RateLimitPerUser, cfg.RateLimitWindow)

	// Initialize provider manager with mock providers
	providerManager := provider.NewProviderManager(provider.Random)

	// Add some mock providers with different characteristics
	providerManager.AddProvider(provider.NewMockProvider("firebase", 0.95, 100*time.Millisecond, 50*time.Millisecond))
	providerManager.AddProvider(provider.NewMockProvider("apns", 0.98, 150*time.Millisecond, 75*time.Millisecond))
	providerManager.AddProvider(provider.NewMockProvider("fcm", 0.92, 80*time.Millisecond, 40*time.Millisecond))

	log.Printf("Initialized %d mock providers", len(providerManager.GetAllProviders()))

	// Initialize worker pool
	workerPool := worker.NewPool(
		cfg.WorkerCount,
		cfg.MaxQueueSize,
		rateLimiter,
		providerManager,
		cfg.RetryAttempts,
		cfg.RetryDelay,
	)

	// Create channels
	messageChan := make(chan *pkg.NotificationMessage, cfg.MaxQueueSize)
	errorChan := make(chan error, 100)

	// Initialize Kafka consumer
	kafkaConsumer, err := kafka.NewConsumer(
		cfg.KafkaBrokers,
		cfg.ConsumerGroup,
		[]string{cfg.KafkaTopic},
		messageChan,
		errorChan,
	)
	if err != nil {
		cancel() // Clean up context
		return nil, fmt.Errorf("failed to create kafka consumer: %w", err)
	}

	// Initialize Kafka producer (for testing purposes)
	kafkaProducer, err := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopic)
	if err != nil {
		log.Printf("Warning: failed to create kafka producer: %v", err)
		kafkaProducer = nil // Non-critical for the service
	}

	service := &Service{
		config:          cfg,
		workerPool:      workerPool,
		kafkaConsumer:   kafkaConsumer,
		kafkaProducer:   kafkaProducer,
		rateLimiter:     rateLimiter,
		redisClient:     redisClient,
		providerManager: providerManager,
		messageChan:     messageChan,
		errorChan:       errorChan,
		ctx:             ctx,
		cancel:          cancel,
	}

	// Initialize HTTP server
	service.setupHTTPServer()

	return service, nil
}

// Start starts the notification service
func (s *Service) Start() error {
	log.Println("Starting notification service...")

	// Start worker pool
	s.workerPool.Start(s.ctx)

	// Start Kafka consumer
	if err := s.kafkaConsumer.Start(); err != nil {
		return fmt.Errorf("failed to start kafka consumer: %w", err)
	}

	// Start message processor
	s.wg.Add(1)
	go s.processMessages()

	// Start result processor
	s.wg.Add(1)
	go s.processResults()

	// Start error processor
	s.wg.Add(1)
	go s.processErrors()

	// Start HTTP server
	s.wg.Add(1)
	go s.startHTTPServer()

	log.Println("Notification service started successfully")
	return nil
}

// Stop stops the notification service gracefully
func (s *Service) Stop() {
	log.Println("Shutting down notification service...")

	// Cancel context
	s.cancel()

	// Stop HTTP server
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
		defer cancel()
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}

	// Stop Kafka consumer
	if err := s.kafkaConsumer.Stop(); err != nil {
		log.Printf("Kafka consumer stop error: %v", err)
	}

	// Stop worker pool
	s.workerPool.Stop()

	// Close Redis client
	if err := s.redisClient.Close(); err != nil {
		log.Printf("Redis client close error: %v", err)
	}

	// Close Kafka producer
	if s.kafkaProducer != nil {
		if err := s.kafkaProducer.Close(); err != nil {
			log.Printf("Kafka producer close error: %v", err)
		}
	}

	// Wait for goroutines
	s.wg.Wait()

	log.Println("Notification service stopped")
}

// processMessages processes incoming messages from Kafka
func (s *Service) processMessages() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case msg := <-s.messageChan:
			if msg == nil {
				continue
			}

			// Submit to worker pool
			if err := s.workerPool.Submit(msg); err != nil {
				log.Printf("Failed to submit message to worker pool: %v", err)
			}
		}
	}
}

// processResults processes results from worker pool
func (s *Service) processResults() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case result := <-s.workerPool.Results():
			if result == nil {
				continue
			}

			// Log result
			if result.Success {
				log.Printf("Successfully processed notification %s for user %s via %s (attempts: %d)",
					result.MessageID, result.UserID, result.Provider, result.Attempts)
			} else {
				log.Printf("Failed to process notification %s for user %s: %v (attempts: %d)",
					result.MessageID, result.UserID, result.Error, result.Attempts)
			}
		}
	}
}

// processErrors processes errors from various components
func (s *Service) processErrors() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case err := <-s.errorChan:
			if err == nil {
				continue
			}
			log.Printf("Service error: %v", err)
		case err := <-s.workerPool.Errors():
			if err == nil {
				continue
			}
			log.Printf("Worker pool error: %v", err)
		}
	}
}

// setupHTTPServer sets up the HTTP server for health checks and metrics
func (s *Service) setupHTTPServer() {
	router := mux.NewRouter()

	// Health check endpoint
	router.HandleFunc("/health", s.healthHandler).Methods("GET")

	// Metrics endpoint
	router.HandleFunc("/metrics", s.metricsHandler).Methods("GET")

	// Rate limit status endpoint
	router.HandleFunc("/ratelimit/{userID}", s.rateLimitHandler).Methods("GET")

	// Test endpoint to send a notification (for testing)
	if s.kafkaProducer != nil {
		router.HandleFunc("/send", s.sendNotificationHandler).Methods("POST")
	}

	s.httpServer = &http.Server{
		Addr:         ":" + s.config.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

// startHTTPServer starts the HTTP server
func (s *Service) startHTTPServer() {
	defer s.wg.Done()

	log.Printf("HTTP server starting on port %s", s.config.Port)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("HTTP server error: %v", err)
	}
}

// HTTP Handlers

// healthHandler provides health check endpoint
func (s *Service) healthHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"service":   "notification-service",
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	}

	// Check worker pool health
	if err := s.workerPool.IsHealthy(r.Context()); err != nil {
		status["status"] = "unhealthy"
		status["worker_pool_error"] = err.Error()
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	// Check Redis health
	if err := redisLib.HealthCheck(r.Context(), s.redisClient); err != nil {
		status["status"] = "unhealthy"
		status["redis_error"] = err.Error()
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	// Check Kafka health
	if err := kafka.HealthCheck(s.config.KafkaBrokers); err != nil {
		status["status"] = "unhealthy"
		status["kafka_error"] = err.Error()
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	// Check provider health
	providerHealth := s.providerManager.HealthCheckAll(r.Context())
	healthyProviders := 0
	for _, err := range providerHealth {
		if err == nil {
			healthyProviders++
		}
	}

	status["healthy_providers"] = healthyProviders
	status["total_providers"] = len(providerHealth)

	if healthyProviders == 0 {
		status["status"] = "unhealthy"
		status["provider_error"] = "no healthy providers"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// metricsHandler provides metrics endpoint
func (s *Service) metricsHandler(w http.ResponseWriter, r *http.Request) {
	processed, failed, rateLimited := s.workerPool.GetMetrics()

	metrics := map[string]interface{}{
		"processed_messages":    processed,
		"failed_messages":       failed,
		"rate_limited_messages": rateLimited,
		"queue_size":            s.workerPool.QueueSize(),
		"worker_count":          s.config.WorkerCount,
		"timestamp":             time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// rateLimitHandler provides rate limit status for a user
func (s *Service) rateLimitHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	if userID == "" {
		http.Error(w, "userID is required", http.StatusBadRequest)
		return
	}

	current, err := s.rateLimiter.GetCurrentCount(r.Context(), userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting rate limit: %v", err), http.StatusInternalServerError)
		return
	}

	remaining, err := s.rateLimiter.GetRemainingCount(r.Context(), userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting remaining count: %v", err), http.StatusInternalServerError)
		return
	}

	ttl, err := s.rateLimiter.GetTTL(r.Context(), userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting TTL: %v", err), http.StatusInternalServerError)
		return
	}

	status := map[string]interface{}{
		"user_id":          userID,
		"limit":            s.config.RateLimitPerUser,
		"current":          current,
		"remaining":        remaining,
		"window_seconds":   int(s.config.RateLimitWindow.Seconds()),
		"reset_in_seconds": int(ttl.Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// sendNotificationHandler provides a test endpoint to send notifications
func (s *Service) sendNotificationHandler(w http.ResponseWriter, r *http.Request) {
	if s.kafkaProducer == nil {
		http.Error(w, "Kafka producer not available", http.StatusServiceUnavailable)
		return
	}

	var notification pkg.NotificationMessage
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Set defaults if not provided
	if notification.ID == "" {
		notification.ID = fmt.Sprintf("test_%d", time.Now().UnixNano())
	}
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = time.Now()
	}
	if notification.Priority == 0 {
		notification.Priority = pkg.PriorityNormal
	}

	// Send to Kafka
	if err := s.kafkaProducer.Send(&notification); err != nil {
		http.Error(w, fmt.Sprintf("Failed to send notification: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":         "Notification sent successfully",
		"notification_id": notification.ID,
		"user_id":         notification.UserID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// main function
func main() {
	// Create service
	service, err := NewService()
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// Start service
	if err := service.Start(); err != nil {
		log.Fatalf("Failed to start service: %v", err)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive a signal
	sig := <-sigChan
	log.Printf("Received signal: %v", sig)

	// Graceful shutdown
	service.Stop()
}
