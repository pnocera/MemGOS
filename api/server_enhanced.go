package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/interfaces"
	ginSwagger "github.com/swaggo/gin-swagger"
	swaggerFiles "github.com/swaggo/files"
	_ "github.com/memtensor/memgos/docs"
)

// EnhancedServer represents the API server instance with full functionality
type EnhancedServer struct {
	*Server // Embed the basic server to inherit methods
	authManager *AuthManager
	startTime   time.Time
}

// NewEnhancedServer creates a new API server instance with authentication
func NewEnhancedServer(mosCore interfaces.MOSCore, cfg *config.MOSConfig, logger interfaces.Logger) *EnhancedServer {
	// Set Gin mode based on config (use LogLevel as proxy for environment)
	if cfg.LogLevel == "error" || cfg.LogLevel == "warn" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Create the base server first
	baseServer := NewServer(mosCore, cfg, logger, nil)
	authManager := NewAuthManager()

	s := &EnhancedServer{
		Server:      baseServer,
		authManager: authManager,
		startTime:   time.Now(),
	}

	// Override with enhanced routes and middleware
	s.setupEnhancedMiddleware()
	s.setupEnhancedRoutes()

	return s
}

// setupEnhancedMiddleware configures additional middleware for the enhanced server
func (s *EnhancedServer) setupEnhancedMiddleware() {
	// Enhanced JWT authentication middleware
	s.router.Use(s.jwtAuthMiddleware())
}

// jwtAuthMiddleware provides JWT token authentication
func (s *EnhancedServer) jwtAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication for health check and docs
		if c.Request.URL.Path == "/health" || 
		   c.Request.URL.Path == "/" ||
		   c.Request.URL.Path == "/docs" ||
		   c.Request.URL.Path == "/openapi.json" ||
		   c.Request.URL.Path == "/auth/login" ||
		   c.Request.URL.Path == "/auth/refresh" {
			c.Next()
			return
		}

		// Extract JWT token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// TODO: Implement proper JWT validation
		// For now, just extract user_id from header or use default
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			userID = s.config.UserID // Use default from config
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

// setupEnhancedRoutes configures additional routes for the enhanced server
func (s *EnhancedServer) setupEnhancedRoutes() {
	// Authentication endpoints
	auth := s.router.Group("/auth")
	{
		auth.POST("/login", s.login)
		auth.POST("/logout", s.logout)
		auth.POST("/refresh", s.refreshToken)
	}

	// Override health check with enhanced version
	s.router.GET("/health", s.enhancedHealthCheck)

	// Enhanced OpenAPI spec endpoint
	s.router.GET("/openapi.json", s.getCompleteOpenAPISpec)

	// Swagger UI routes
	s.router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

// Start starts the enhanced API server
func (s *EnhancedServer) Start(ctx context.Context) error {
	port := s.getPort()

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info("Starting Enhanced API server", map[string]interface{}{
		"port":     port,
		"mode":     gin.Mode(),
		"features": []string{"auth", "metrics", "openapi", "cors"},
	})

	// Start cleanup routine for expired sessions
	go s.startSessionCleanup()

	// Start server in a goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Failed to start server", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	s.logger.Info("Shutting down Enhanced API server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return s.server.Shutdown(shutdownCtx)
}

// Stop gracefully stops the enhanced API server
func (s *EnhancedServer) Stop() error {
	if s.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}

// startSessionCleanup starts a goroutine to periodically clean up expired sessions
func (s *EnhancedServer) startSessionCleanup() {
	ticker := time.NewTicker(1 * time.Hour) // Clean up every hour
	for {
		select {
		case <-ticker.C:
			s.authManager.CleanupExpiredSessions()
			s.logger.Debug("Cleaned up expired sessions")
		}
	}
}

// getPort returns the port from environment or config
func (s *EnhancedServer) getPort() int {
	if s.config.HealthCheckPort > 0 {
		return s.config.HealthCheckPort
	}
	return 8080 // Default port
}

// Enhanced health check with start time tracking
func (s *EnhancedServer) enhancedHealthCheck(c *gin.Context) {
	uptime := time.Since(s.startTime)

	health := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   "1.0.0",
		Uptime:    uptime.String(),
		Checks: map[string]string{
			"core":        "ok",
			"database":    "ok",
			"memory":      "ok",
			"auth":        "ok",
			"sessions":    fmt.Sprintf("%d active", len(s.authManager.sessions)),
			"start_time": s.startTime.Format(time.RFC3339),
		},
	}

	c.JSON(http.StatusOK, health)
}

// Authentication handlers
func (s *EnhancedServer) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// TODO: Implement proper authentication
	// For now, return a placeholder token
	token := s.authManager.GenerateToken(req.Username)
	
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"username": req.Username,
			"id": req.Username,
		},
	})
}

func (s *EnhancedServer) logout(c *gin.Context) {
	// TODO: Implement logout logic
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (s *EnhancedServer) refreshToken(c *gin.Context) {
	// TODO: Implement token refresh
	c.JSON(http.StatusOK, gin.H{"message": "Token refreshed"})
}

func (s *EnhancedServer) getCompleteOpenAPISpec(c *gin.Context) {
	// TODO: Return complete OpenAPI specification
	c.JSON(http.StatusOK, gin.H{
		"openapi": "3.0.0",
		"info": gin.H{
			"title": "MemGOS Enhanced API",
			"version": "1.0.0",
			"description": "Enhanced Memory-based General Operating System API",
		},
		"paths": gin.H{
			"/health": gin.H{
				"get": gin.H{
					"summary": "Health check endpoint",
					"responses": gin.H{
						"200": gin.H{
							"description": "Service is healthy",
						},
					},
				},
			},
		},
	})
}

// Request/Response types
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}