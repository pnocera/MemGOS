// Package main provides the MemGOS MCP server command-line interface
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/memtensor/memgos/mcp"
	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/interfaces"
)

var (
	configFile = flag.String("config", "", "Path to configuration file")
	apiToken   = flag.String("token", "", "API token for authentication (required)")
	userID     = flag.String("user", "default-user", "User ID for the session")
	sessionID  = flag.String("session", "mcp-session", "Session ID for the MCP server")
	apiURL     = flag.String("api-url", "http://localhost:8080", "MemGOS API base URL")
	verbose    = flag.Bool("verbose", false, "Enable verbose logging")
	version    = flag.Bool("version", false, "Show version information")
)

// SimpleLogger implements interfaces.Logger for basic logging
type SimpleLogger struct {
	verbose bool
}

func (l *SimpleLogger) Debug(msg string, fields ...map[string]interface{}) {
	if l.verbose {
		log.Printf("[DEBUG] %s %v", msg, fields)
	}
}

func (l *SimpleLogger) Info(msg string, fields ...map[string]interface{}) {
	log.Printf("[INFO] %s %v", msg, fields)
}

func (l *SimpleLogger) Warn(msg string, fields ...map[string]interface{}) {
	log.Printf("[WARN] %s %v", msg, fields)
}

func (l *SimpleLogger) Error(msg string, err error, fields ...map[string]interface{}) {
	log.Printf("[ERROR] %s: %v %v", msg, err, fields)
}

func (l *SimpleLogger) Fatal(msg string, err error, fields ...map[string]interface{}) {
	log.Fatalf("[FATAL] %s: %v %v", msg, err, fields)
}

func (l *SimpleLogger) WithFields(fields map[string]interface{}) interfaces.Logger {
	return l // Simple implementation returns self
}

// SimpleMetrics implements interfaces.Metrics for basic metrics
type SimpleMetrics struct{}

func (m *SimpleMetrics) Counter(name string, value float64, tags map[string]string)   {}
func (m *SimpleMetrics) Gauge(name string, value float64, tags map[string]string)     {}
func (m *SimpleMetrics) Timer(name string, value float64, tags map[string]string)     {}
func (m *SimpleMetrics) Histogram(name string, value float64, tags map[string]string) {}

func main() {
	flag.Parse()

	if *version {
		fmt.Println("MemGOS MCP Server v1.0.0")
		fmt.Println("Model Context Protocol server for MemGOS")
		fmt.Println("Connects to MemGOS API using token authentication")
		os.Exit(0)
	}

	// Check for token from environment if not provided via flag
	if *apiToken == "" {
		if envToken := os.Getenv("MEMGOS_API_TOKEN"); envToken != "" {
			*apiToken = envToken
		} else {
			fmt.Fprintf(os.Stderr, "Error: API token is required. Use --token flag or set MEMGOS_API_TOKEN environment variable.\n")
			fmt.Fprintf(os.Stderr, "Usage: %s --token <api-token> [options]\n", os.Args[0])
			os.Exit(1)
		}
	}

	// Create logger  
	logger := &SimpleLogger{verbose: *verbose}

	// Create configuration for MCP client mode (API-based)
	cfg := config.NewMOSConfig()
	cfg.UserID = *userID
	cfg.SessionID = *sessionID
	cfg.LogLevel = "info"
	
	// Configure for API token authentication
	cfg.APIToken = *apiToken
	cfg.APIURL = *apiURL
	cfg.MCPMode = true // Enable MCP client mode
	
	// Disable local components since we're using API
	cfg.EnableTextualMemory = false
	cfg.EnableActivationMemory = false
	cfg.EnableParametricMemory = false
	cfg.EnableMemScheduler = false
	cfg.MetricsEnabled = false
	cfg.HealthCheckEnabled = false

	if *verbose {
		cfg.LogLevel = "debug"
	}

	// Load config file if provided
	if *configFile != "" {
		// For now, just use basic config - TODO: implement config loading
		logger.Info("Config file specified but not implemented yet", map[string]interface{}{
			"config_file": *configFile,
		})
		
		// Override with command line flags
		if *userID != "default-user" {
			cfg.UserID = *userID
		}
		if *sessionID != "mcp-session" {
			cfg.SessionID = *sessionID
		}
	}

	logger.Info("Starting MemGOS MCP Server", map[string]interface{}{
		"user_id":    cfg.UserID,
		"session_id": cfg.SessionID,
		"api_url":    cfg.APIURL,
		"mcp_mode":   cfg.MCPMode,
	})

	// Create API client for MCP mode
	apiClient := NewAPIClient(cfg.APIURL, cfg.APIToken, logger)
	
	// Test API connection
	ctx := context.Background()
	if err := apiClient.TestConnection(ctx); err != nil {
		logger.Error("Failed to connect to MemGOS API", err, map[string]interface{}{
			"api_url": cfg.APIURL,
		})
		os.Exit(1)
	}
	
	logger.Info("Connected to MemGOS API successfully", map[string]interface{}{
		"api_url": cfg.APIURL,
	})

	// Create MCP server with API client
	mcpServer := mcp.NewServerWithAPIClient(apiClient, cfg, logger)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		sig := <-sigChan
		logger.Info("Received shutdown signal", map[string]interface{}{
			"signal": sig.String(),
		})
		cancel()
	}()

	// Start MCP server
	logger.Info("MCP Server ready - communicating via stdio", nil)
	
	if err := mcpServer.Start(ctx); err != nil {
		logger.Error("MCP Server error", err)
		os.Exit(1)
	}

	logger.Info("MemGOS MCP Server shutdown complete", nil)
}