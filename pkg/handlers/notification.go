// Package handlers provides HTTP request handlers
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"notification-service/pkg/models"
	"notification-service/pkg/services"
)

// NotificationHandler handles notification-related HTTP requests
type NotificationHandler struct {
	notificationService *services.NotificationService
	logger              *logrus.Logger
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(notificationService *services.NotificationService, logger *logrus.Logger) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
		logger:              logger,
	}
}

// SendNotification handles POST /api/v1/notifications
func (h *NotificationHandler) SendNotification(c *gin.Context) {
	var req models.NotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid request payload")
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request payload",
			Error:   err.Error(),
		})
		return
	}

	notification, err := h.notificationService.SendNotification(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to send notification")
		statusCode := http.StatusInternalServerError
		
		// Handle specific error cases
		if err.Error() == "user session not found" || err.Error() == "user session is not active" {
			statusCode = http.StatusBadRequest
		}
		if err.Error() == "rate limit exceeded" {
			statusCode = http.StatusTooManyRequests
		}

		c.JSON(statusCode, models.APIResponse{
			Success: false,
			Message: "Failed to send notification",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "Notification queued successfully",
		Data:    notification,
	})
}

// GetNotificationStatus handles GET /api/v1/notifications/:id/status
func (h *NotificationHandler) GetNotificationStatus(c *gin.Context) {
	notificationID := c.Param("id")
	if notificationID == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Notification ID is required",
		})
		return
	}

	status, err := h.notificationService.GetNotificationStatus(c.Request.Context(), notificationID)
	if err != nil {
		h.logger.WithError(err).WithField("notification_id", notificationID).Error("Failed to get notification status")
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Message: "Notification status not found",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Notification status retrieved successfully",
		Data: map[string]interface{}{
			"notification_id": notificationID,
			"status":         status,
		},
	})
}

// RegisterSession handles POST /api/v1/sessions
func (h *NotificationHandler) RegisterSession(c *gin.Context) {
	var req struct {
		UserID      string `json:"user_id" binding:"required"`
		DeviceToken string `json:"device_token" binding:"required"`
		Platform    string `json:"platform" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid session registration payload")
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request payload",
			Error:   err.Error(),
		})
		return
	}

	if err := h.notificationService.RegisterUserSession(c.Request.Context(), req.UserID, req.DeviceToken, req.Platform); err != nil {
		h.logger.WithError(err).Error("Failed to register user session")
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to register session",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "Session registered successfully",
		Data: map[string]interface{}{
			"user_id":  req.UserID,
			"platform": req.Platform,
		},
	})
}

// UnregisterSession handles DELETE /api/v1/sessions/:user_id
func (h *NotificationHandler) UnregisterSession(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "User ID is required",
		})
		return
	}

	if err := h.notificationService.UnregisterUserSession(c.Request.Context(), userID); err != nil {
		h.logger.WithError(err).WithField("user_id", userID).Error("Failed to unregister user session")
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to unregister session",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Session unregistered successfully",
		Data: map[string]interface{}{
			"user_id": userID,
		},
	})
}

// HealthCheck handles GET /health
func (h *NotificationHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "API Gateway is healthy",
		Data: map[string]interface{}{
			"service": "notification-api-gateway",
			"status":  "running",
		},
	})
}