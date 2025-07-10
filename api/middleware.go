package api

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// loggingMiddleware provides request logging
func (s *Server) loggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Log using our structured logger
		s.logger.Info("HTTP Request", map[string]interface{}{
			"method":      param.Method,
			"path":        param.Path,
			"status_code": param.StatusCode,
			"latency":     param.Latency,
			"client_ip":   param.ClientIP,
			"user_agent":  param.Request.UserAgent(),
			"request_id":  param.Keys["request_id"],
		})
		return ""
	})
}

// requestIDMiddleware adds a unique request ID to each request
func (s *Server) requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// authMiddleware provides authentication (placeholder for now)
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication for health check and docs
		if c.Request.URL.Path == "/health" || 
		   c.Request.URL.Path == "/" ||
		   c.Request.URL.Path == "/docs" ||
		   c.Request.URL.Path == "/openapi.json" {
			c.Next()
			return
		}

		// TODO: Implement proper authentication
		// For now, just extract user_id from header or query param
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			userID = c.Query("user_id")
		}
		if userID == "" {
			userID = s.config.UserID // Use default from config
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

// rateLimitMiddleware provides rate limiting (placeholder)
func (s *Server) rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement rate limiting
		// For now, just continue
		c.Next()
	}
}

// corsMiddleware provides CORS headers
func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-User-ID, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// timeoutMiddleware provides request timeout
func (s *Server) timeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement proper timeout handling
		c.Next()
	}
}

// metricsMiddleware collects request metrics
func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		// Record metrics
		status := strconv.Itoa(c.Writer.Status())
		
		// TODO: Use proper metrics implementation
		s.logger.Debug("Request metrics", map[string]interface{}{
			"method":   c.Request.Method,
			"path":     c.Request.URL.Path,
			"status":   status,
			"duration": duration.Milliseconds(),
		})
	}
}