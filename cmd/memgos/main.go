// Package main provides the main entry point for MemGOS
// @title MemGOS API
// @version 1.0.0
// @description Memory-based General Operating System API with comprehensive swagger documentation
// @contact.name MemGOS API Support
// @contact.url https://github.com/pnocera/MemGOS
// @contact.email support@memgos.io
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
// @host localhost:8080
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT token authentication. Use 'Bearer {token}' format.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/memtensor/memgos/api"
	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/core"
	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/logger"
	"github.com/memtensor/memgos/pkg/metrics"
	"github.com/memtensor/memgos/pkg/types"
)

// Version information (set by build process)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// Command line flags
var (
	configFile  = flag.String("config", "", "Path to configuration file")
	logLevel    = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	logFile     = flag.String("log-file", "", "Log file path (default: stdout)")
	showVersion = flag.Bool("version", false, "Show version information")
	apiMode     = flag.Bool("api", false, "Run in API server mode")
	interactive = flag.Bool("interactive", false, "Run in interactive mode")
	userID      = flag.String("user", "", "User ID for operations")
	sessionID   = flag.String("session", "", "Session ID for operations")
)

func main() {
	flag.Parse()
	
	// Show version if requested
	if *showVersion {
		fmt.Printf("MemGOS %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		os.Exit(0)
	}
	
	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal, gracefully shutting down...")
		cancel()
	}()
	
	// Initialize and run application
	if err := run(ctx); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}

func run(ctx context.Context) error {
	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Initialize logger
	logger, err := initializeLogger(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	
	logger.Info("Starting MemGOS", map[string]interface{}{
		"version":    Version,
		"build_time": BuildTime,
		"git_commit": GitCommit,
	})
	
	// Initialize metrics
	metrics, err := initializeMetrics(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize metrics: %w", err)
	}
	
	// Initialize MOS Core
	mosCore, err := core.NewMOSCore(cfg, logger, metrics)
	if err != nil {
		return fmt.Errorf("failed to create MOS core: %w", err)
	}
	
	if err := mosCore.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize MOS core: %w", err)
	}
	defer func() {
		if closeErr := mosCore.Close(); closeErr != nil {
			logger.Error("Failed to close MOS core", closeErr)
		}
	}()
	
	// Run based on mode
	switch {
	case *apiMode:
		return runAPIServer(ctx, mosCore, cfg, logger)
	case *interactive:
		return runInteractiveMode(ctx, mosCore, cfg, logger)
	default:
		return runCLIMode(ctx, mosCore, cfg, logger)
	}
}

func loadConfig() (*config.MOSConfig, error) {
	cfg := config.NewMOSConfig()
	
	// Load from file if specified
	if *configFile != "" {
		ext := filepath.Ext(*configFile)
		switch ext {
		case ".json":
			if err := cfg.FromJSONFile(*configFile); err != nil {
				return nil, fmt.Errorf("failed to load JSON config: %w", err)
			}
		case ".yaml", ".yml":
			if err := cfg.FromYAMLFile(*configFile); err != nil {
				return nil, fmt.Errorf("failed to load YAML config: %w", err)
			}
		default:
			return nil, fmt.Errorf("unsupported config file format: %s", ext)
		}
	}
	
	// Override with command line flags
	if *logLevel != "" {
		cfg.LogLevel = *logLevel
	}
	if *logFile != "" {
		cfg.LogFile = *logFile
	}
	if *userID != "" {
		cfg.UserID = *userID
	}
	if *sessionID != "" {
		cfg.SessionID = *sessionID
	}
	
	// Set defaults if not provided
	if cfg.UserID == "" {
		cfg.UserID = "default-user"
	}
	if cfg.SessionID == "" {
		cfg.SessionID = fmt.Sprintf("session-%d", time.Now().Unix())
	}
	
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	return cfg, nil
}

func initializeLogger(cfg *config.MOSConfig) (interfaces.Logger, error) {
	// TODO: Initialize proper logger implementation
	// For now, return a placeholder
	return &logger.PlaceholderLogger{
		Level: cfg.LogLevel,
		File:  cfg.LogFile,
	}, nil
}

func initializeMetrics(cfg *config.MOSConfig) (interfaces.Metrics, error) {
	if !cfg.MetricsEnabled {
		return &metrics.NoOpMetrics{}, nil
	}
	
	// TODO: Initialize proper metrics implementation
	return &metrics.PlaceholderMetrics{
		Port: cfg.MetricsPort,
	}, nil
}

func runAPIServer(ctx context.Context, mosCore interfaces.MOSCore, cfg *config.MOSConfig, logger interfaces.Logger) error {
	logger.Info("Starting Enhanced API server mode")
	
	// Create the Enhanced API server instance
	server := api.NewEnhancedServer(mosCore, cfg, logger)
	
	// Start the server (this will block until context is cancelled)
	return server.Start(ctx)
}

func runInteractiveMode(ctx context.Context, mosCore interfaces.MOSCore, cfg *config.MOSConfig, logger interfaces.Logger) error {
	logger.Info("Starting interactive mode")
	
	// TODO: Implement interactive CLI
	// This would involve:
	// 1. Creating a command-line interface
	// 2. Implementing commands for all MOS operations
	// 3. Providing help and autocomplete
	// 4. Handling user input and displaying results
	
	fmt.Println("Welcome to MemGOS Interactive Mode")
	fmt.Println("Type 'help' for available commands or 'exit' to quit")
	
	// Simple placeholder interactive loop
	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nGoodbye!")
			return nil
		default:
			// TODO: Implement actual interactive command processing
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func runCLIMode(ctx context.Context, mosCore interfaces.MOSCore, cfg *config.MOSConfig, logger interfaces.Logger) error {
	logger.Info("Starting CLI mode")
	
	// Parse remaining command line arguments
	args := flag.Args()
	if len(args) == 0 {
		return fmt.Errorf("no command specified, use --help for usage information")
	}
	
	command := args[0]
	commandArgs := args[1:]
	
	// Execute command
	switch command {
	case "search":
		return executeSearchCommand(ctx, mosCore, commandArgs, logger)
	case "add":
		return executeAddCommand(ctx, mosCore, commandArgs, logger)
	case "register":
		return executeRegisterCommand(ctx, mosCore, commandArgs, logger)
	case "list":
		return executeListCommand(ctx, mosCore, commandArgs, logger)
	case "chat":
		return executeChatCommand(ctx, mosCore, commandArgs, logger)
	case "user":
		return executeUserCommand(ctx, mosCore, commandArgs, logger)
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

func executeSearchCommand(ctx context.Context, mosCore interfaces.MOSCore, args []string, logger interfaces.Logger) error {
	if len(args) == 0 {
		return fmt.Errorf("search query required")
	}
	
	query := args[0]
	searchQuery := &types.SearchQuery{
		Query: query,
		TopK:  5,
	}
	
	result, err := mosCore.Search(ctx, searchQuery)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}
	
	fmt.Printf("Search Results for '%s':\n", query)
	fmt.Printf("Textual Memories: %d cubes\n", len(result.TextMem))
	fmt.Printf("Activation Memories: %d cubes\n", len(result.ActMem))
	fmt.Printf("Parametric Memories: %d cubes\n", len(result.ParaMem))
	
	// TODO: Display detailed results
	
	return nil
}

func executeAddCommand(ctx context.Context, mosCore interfaces.MOSCore, args []string, logger interfaces.Logger) error {
	if len(args) == 0 {
		return fmt.Errorf("content to add required")
	}
	
	content := args[0]
	request := &types.AddMemoryRequest{
		MemoryContent: &content,
	}
	
	if err := mosCore.Add(ctx, request); err != nil {
		return fmt.Errorf("add failed: %w", err)
	}
	
	fmt.Printf("Successfully added content to memory\n")
	return nil
}

func executeRegisterCommand(ctx context.Context, mosCore interfaces.MOSCore, args []string, logger interfaces.Logger) error {
	if len(args) == 0 {
		return fmt.Errorf("cube path required")
	}
	
	cubePath := args[0]
	cubeID := ""
	if len(args) > 1 {
		cubeID = args[1]
	}
	
	if err := mosCore.RegisterMemCube(ctx, cubePath, cubeID, ""); err != nil {
		return fmt.Errorf("register failed: %w", err)
	}
	
	fmt.Printf("Successfully registered memory cube: %s\n", cubePath)
	return nil
}

func executeListCommand(ctx context.Context, mosCore interfaces.MOSCore, args []string, logger interfaces.Logger) error {
	// TODO: Implement list cubes command
	fmt.Println("List command not yet implemented")
	return nil
}

func executeChatCommand(ctx context.Context, mosCore interfaces.MOSCore, args []string, logger interfaces.Logger) error {
	if len(args) == 0 {
		return fmt.Errorf("chat query required")
	}
	
	query := args[0]
	request := &types.ChatRequest{
		Query: query,
	}
	
	response, err := mosCore.Chat(ctx, request)
	if err != nil {
		return fmt.Errorf("chat failed: %w", err)
	}
	
	fmt.Printf("Assistant: %s\n", response.Response)
	return nil
}

func executeUserCommand(ctx context.Context, mosCore interfaces.MOSCore, args []string, logger interfaces.Logger) error {
	if len(args) == 0 {
		// Show current user info
		userInfo, err := mosCore.GetUserInfo(ctx, "")
		if err != nil {
			return fmt.Errorf("failed to get user info: %w", err)
		}
		
		fmt.Printf("User Information:\n")
		for key, value := range userInfo {
			fmt.Printf("  %s: %v\n", key, value)
		}
		return nil
	}
	
	subcommand := args[0]
	switch subcommand {
	case "create":
		if len(args) < 2 {
			return fmt.Errorf("user name required")
		}
		userName := args[1]
		userID, err := mosCore.CreateUser(ctx, userName, types.UserRoleUser, "")
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
		fmt.Printf("Created user: %s (ID: %s)\n", userName, userID)
		return nil
	default:
		return fmt.Errorf("unknown user subcommand: %s", subcommand)
	}
}