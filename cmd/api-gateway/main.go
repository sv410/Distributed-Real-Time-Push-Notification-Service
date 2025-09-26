// API Gateway - Entry point for push notification requests
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"notification-service/internal/kafka"
	"notification-service/internal/redis"
	"notification-service/pkg/config"
	"notification-service/pkg/handlers"
	"notification-service/pkg/middleware"
	"notification-service/pkg/services"
)

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

	logger.Info("Starting Notification API Gateway")

	// Initialize Redis client
	redisClient, err := redis.NewClient(cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.Password, cfg.Redis.DB, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize Redis client")
	}
	defer redisClient.Close()

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx); err != nil {
		logger.WithError(err).Fatal("Failed to connect to Redis")
	}
	logger.Info("Connected to Redis successfully")

	// Initialize Kafka producer
	producer, err := kafka.NewProducer(cfg.Kafka.BootstrapServers, cfg.Kafka.Topic, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize Kafka producer")
	}
	defer producer.Close()
	logger.Info("Kafka producer initialized successfully")

	// Initialize services
	notificationService := services.NewNotificationService(producer, redisClient, logger)

	// Initialize handlers
	notificationHandler := handlers.NewNotificationHandler(notificationService, logger)

	// Setup Gin router
	if cfg.Log.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	
	router := gin.New()
	
	// Add middleware
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.Logger(logger))
	router.Use(middleware.CORS())
	router.Use(middleware.RequestID())

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Notification routes
		v1.POST("/notifications", notificationHandler.SendNotification)
		v1.GET("/notifications/:id/status", notificationHandler.GetNotificationStatus)
		
		// Session management routes
		v1.POST("/sessions", notificationHandler.RegisterSession)
		v1.DELETE("/sessions/:user_id", notificationHandler.UnregisterSession)
	}

	// Health check route
	router.GET("/health", notificationHandler.HealthCheck)

	// Setup HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.WithField("address", server.Addr).Info("Starting HTTP server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start HTTP server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.WithError(err).Fatal("Server forced to shutdown")
	}

	logger.Info("Server exited")
}