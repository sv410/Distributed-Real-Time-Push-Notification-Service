// Package middleware provides HTTP middleware functions
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Logger returns a gin.HandlerFunc that logs requests using logrus
func Logger(logger *logrus.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		fields := logrus.Fields{
			"client_ip":   param.ClientIP,
			"timestamp":   param.TimeStamp.Format(time.RFC3339),
			"method":      param.Method,
			"path":        param.Path,
			"protocol":    param.Request.Proto,
			"status_code": param.StatusCode,
			"latency":     param.Latency,
			"user_agent":  param.Request.UserAgent(),
		}

		if param.ErrorMessage != "" {
			fields["error"] = param.ErrorMessage
		}

		if param.StatusCode >= 500 {
			logger.WithFields(fields).Error("Server error")
		} else if param.StatusCode >= 400 {
			logger.WithFields(fields).Warn("Client error")
		} else {
			logger.WithFields(fields).Info("Request completed")
		}

		return ""
	})
}

// Recovery returns a gin.HandlerFunc that recovers from panics
func Recovery(logger *logrus.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.WithFields(logrus.Fields{
			"path":   c.Request.URL.Path,
			"method": c.Request.Method,
			"panic":  recovered,
		}).Error("Panic recovered")

		c.AbortWithStatus(500)
	})
}

// CORS returns a gin.HandlerFunc that handles CORS
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RequestID returns a gin.HandlerFunc that adds a request ID to the context
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

func generateRequestID() string {
	// Simple timestamp-based request ID
	return time.Now().Format("20060102150405") + "-" + time.Now().Format("000000")
}