package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/interfaces"
)

// EnhancedServer represents the API server instance with full functionality
type EnhancedServer struct {
	mosCore     interfaces.MOSCore
	config      *config.MOSConfig
	logger      interfaces.Logger
	router      *gin.Engine
	server      *http.Server
	authManager *AuthManager
	startTime   time.Time
}

// NewEnhancedServer creates a new API server instance with authentication
func NewEnhancedServer(mosCore interfaces.MOSCore, cfg *config.MOSConfig, logger interfaces.Logger) *EnhancedServer {
	// Set Gin mode based on config
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	authManager := NewAuthManager()

	s := &EnhancedServer{
		mosCore:     mosCore,
		config:      cfg,
		logger:      logger,
		router:      router,
		authManager: authManager,
		startTime:   time.Now(),
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// setupMiddleware configures middleware for the enhanced server
func (s *EnhancedServer) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Custom logging middleware
	s.router.Use(s.loggingMiddleware())

	// CORS middleware
	s.router.Use(s.corsMiddleware())

	// Request ID middleware
	s.router.Use(s.requestIDMiddleware())

	// Enhanced JWT authentication middleware
	s.router.Use(s.jwtAuthMiddleware())

	// Metrics middleware
	s.router.Use(s.metricsMiddleware())
}

// setupRoutes configures all API routes with authentication endpoints
func (s *EnhancedServer) setupRoutes() {
	// Health check endpoint
	s.router.GET("/health", s.healthCheck)
	s.router.GET("/", s.redirectToDocs)

	// Authentication endpoints
	auth := s.router.Group("/auth")
	{
		auth.POST("/login", s.login)
		auth.POST("/logout", s.logout)
		auth.POST("/refresh", s.refreshToken)
	}

	// Configuration endpoints
	s.router.POST("/configure", s.setConfig)

	// User management endpoints
	users := s.router.Group("/users")
	{
		users.POST("", s.createUser)
		users.GET("", s.listUsers)
		users.GET("/me", s.getUserInfo)
	}

	// Memory cube management endpoints
	memCubes := s.router.Group("/mem_cubes")
	{
		memCubes.POST("", s.registerMemCube)
		memCubes.DELETE("/:mem_cube_id", s.unregisterMemCube)
		memCubes.POST("/:cube_id/share", s.shareCube)
	}

	// Memory management endpoints
	memories := s.router.Group("/memories")
	{
		memories.POST("", s.addMemory)
		memories.GET("", s.getAllMemories)
		memories.GET("/:mem_cube_id/:memory_id", s.getMemory)
		memories.PUT("/:mem_cube_id/:memory_id", s.updateMemory)
		memories.DELETE("/:mem_cube_id/:memory_id", s.deleteMemory)
		memories.DELETE("/:mem_cube_id", s.deleteAllMemories)
	}

	// Search endpoint
	s.router.POST("/search", s.searchMemories)

	// Chat endpoint
	s.router.POST("/chat", s.chat)

	// Metrics endpoint
	s.router.GET("/metrics", s.getMetrics)

	// API documentation endpoints
	s.router.GET("/docs/*any", s.serveDocs)
	s.router.GET("/openapi.json", s.getCompleteOpenAPISpec)
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
	if s.config.APIPort > 0 {
		return s.config.APIPort
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