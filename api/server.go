// Package api provides HTTP REST API server functionality for MemGOS
package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/interfaces"
)

// Server represents the API server instance
type Server struct {
	mosCore     interfaces.MOSCore
	config      *config.MOSConfig
	logger      interfaces.Logger
	router      *gin.Engine
	server      *http.Server
	authManager interfaces.UserManager
}

// NewServer creates a new API server instance
func NewServer(mosCore interfaces.MOSCore, cfg *config.MOSConfig, logger interfaces.Logger, authManager interfaces.UserManager) *Server {
	// Set Gin mode based on config
	// Set Gin mode based on log level (use LogLevel as proxy for environment)
	if cfg.LogLevel == "error" || cfg.LogLevel == "warn" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	s := &Server{
		mosCore:     mosCore,
		config:      cfg,
		logger:      logger,
		router:      router,
		authManager: authManager,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// setupMiddleware configures middleware for the server
func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Custom logging middleware
	s.router.Use(s.loggingMiddleware())

	// CORS middleware
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"*"} // Configure properly for production
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	s.router.Use(cors.New(corsConfig))

	// Request ID middleware
	s.router.Use(s.requestIDMiddleware())

	// Authentication middleware for protected routes
	s.router.Use(s.authMiddleware())
}

// Start starts the API server
func (s *Server) Start(ctx context.Context) error {
	// Use HealthCheckPort as API port since APIPort doesn't exist
	port := s.config.HealthCheckPort
	if port == 0 {
		port = 8080 // Default port
	}

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info("Starting API server", map[string]interface{}{
		"port": port,
		"mode": gin.Mode(),
	})

	// Start server in a goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Failed to start server", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	s.logger.Info("Shutting down API server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return s.server.Shutdown(shutdownCtx)
}

// Stop gracefully stops the API server
func (s *Server) Stop() error {
	if s.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.router.GET("/health", s.healthCheck)
	s.router.GET("/", s.redirectToDocs)

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

	// API documentation endpoint
	s.router.GET("/docs/*any", s.serveDocs)
	s.router.GET("/openapi.json", s.getOpenAPISpec)
}

// getPort returns the port from environment or config
func (s *Server) getPort() int {
	if portStr := os.Getenv("PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			return port
		}
	}
	return s.config.HealthCheckPort
}