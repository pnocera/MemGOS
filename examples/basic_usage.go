// Package main demonstrates basic usage of MemGOS core functionality
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/core"
	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/logger"
	"github.com/memtensor/memgos/pkg/memory"
	"github.com/memtensor/memgos/pkg/metrics"
	"github.com/memtensor/memgos/pkg/types"
)

func main() {
	fmt.Println("MemGOS Basic Usage Example")
	fmt.Println("==========================")

	// Setup context
	ctx := context.Background()

	// Create temporary directory for this example
	tempDir, err := os.MkdirTemp("", "memgos-example-")
	if err != nil {
		log.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)
	fmt.Printf("Using temporary directory: %s\n\n", tempDir)

	// Initialize logger and metrics
	logger := logger.NewConsoleLogger("info")
	metrics := metrics.NewPrometheusMetrics()

	// Example 1: Basic Memory Cube Operations
	fmt.Println("1. Basic Memory Cube Operations")
	fmt.Println("-------------------------------")
	
	if err := demonstrateMemoryCube(ctx, tempDir, logger, metrics); err != nil {
		log.Fatalf("Memory cube example failed: %v", err)
	}

	// Example 2: MOS Core Integration
	fmt.Println("\n2. MOS Core Integration")
	fmt.Println("----------------------")
	
	if err := demonstrateMOSCore(ctx, tempDir, logger, metrics); err != nil {
		log.Fatalf("MOS core example failed: %v", err)
	}

	// Example 3: Advanced Memory Operations
	fmt.Println("\n3. Advanced Memory Operations")
	fmt.Println("----------------------------")
	
	if err := demonstrateAdvancedOperations(ctx, tempDir, logger, metrics); err != nil {
		log.Fatalf("Advanced operations example failed: %v", err)
	}

	fmt.Println("\n✅ All examples completed successfully!")
}

func demonstrateMemoryCube(ctx context.Context, tempDir string, logger interfaces.Logger, metrics interfaces.Metrics) error {
	// Create memory cube factory
	factory := memory.NewMemCubeFactory(logger, metrics)

	// Create an empty memory cube
	cube, err := factory.CreateEmpty("my-cube", "My First Memory Cube")
	if err != nil {
		return fmt.Errorf("failed to create cube: %w", err)
	}
	defer cube.Close()

	fmt.Printf("✓ Created memory cube: %s (%s)\n", cube.GetID(), cube.GetName())

	// Add some textual memories
	if cube.GetTextualMemory() != nil {
		memories := []*types.TextualMemoryItem{
			{
				ID:     "golang-info",
				Memory: "Go is a programming language developed by Google. It's known for its simplicity and efficiency.",
				Metadata: map[string]interface{}{
					"topic":  "programming",
					"source": "manual",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:     "ai-info",
				Memory: "Artificial Intelligence (AI) is the simulation of human intelligence in machines.",
				Metadata: map[string]interface{}{
					"topic":  "technology",
					"source": "manual",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}

		err = cube.GetTextualMemory().AddTextual(ctx, memories)
		if err != nil {
			return fmt.Errorf("failed to add memories: %w", err)
		}
		fmt.Printf("✓ Added %d textual memories\n", len(memories))

		// Search memories
		results, err := cube.GetTextualMemory().Search(ctx, "programming", 5)
		if err != nil {
			return fmt.Errorf("failed to search memories: %w", err)
		}
		fmt.Printf("✓ Found %d memories for 'programming'\n", len(results))

		// Dump cube to directory
		cubeDir := filepath.Join(tempDir, "my-cube")
		err = cube.Dump(ctx, cubeDir)
		if err != nil {
			return fmt.Errorf("failed to dump cube: %w", err)
		}
		fmt.Printf("✓ Dumped cube to: %s\n", cubeDir)

		// Load cube from directory
		loadedCube, err := factory.CreateFromDirectory(ctx, "loaded-cube", "Loaded Cube", cubeDir)
		if err != nil {
			return fmt.Errorf("failed to load cube: %w", err)
		}
		defer loadedCube.Close()

		fmt.Printf("✓ Loaded cube: %s (%s)\n", loadedCube.GetID(), loadedCube.GetName())
	}

	return nil
}

func demonstrateMOSCore(ctx context.Context, tempDir string, logger interfaces.Logger, metrics interfaces.Metrics) error {
	// Create MOS configuration
	cfg := config.NewMOSConfig()
	cfg.UserID = "demo-user"
	cfg.SessionID = "demo-session"
	cfg.EnableTextualMemory = true
	cfg.EnableActivationMemory = false
	cfg.EnableParametricMemory = false
	cfg.TopK = 3

	// Create MOS core
	mosCore, err := core.NewMOSCore(cfg, logger, metrics)
	if err != nil {
		return fmt.Errorf("failed to create MOS core: %w", err)
	}
	defer mosCore.Close()

	// Initialize MOS core
	err = mosCore.Initialize(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize MOS core: %w", err)
	}
	fmt.Println("✓ Initialized MOS core")

	// Create a memory cube directory
	cubeDir := filepath.Join(tempDir, "demo-cube")
	err = os.MkdirAll(cubeDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create cube directory: %w", err)
	}

	// Create cube configuration
	cubeConfig := config.NewMemCubeConfig()
	configPath := filepath.Join(cubeDir, "config.json")
	err = cubeConfig.ToJSONFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to save cube config: %w", err)
	}

	// Register memory cube with MOS
	err = mosCore.RegisterMemCube(ctx, cubeDir, "demo-cube", cfg.UserID)
	if err != nil {
		return fmt.Errorf("failed to register cube: %w", err)
	}
	fmt.Println("✓ Registered memory cube with MOS")

	// Add memories through MOS
	addRequest := &types.AddMemoryRequest{
		MemoryContent: stringPtr("MemGOS is a memory operating system for AI applications. It provides unified memory management."),
		CubeID:        stringPtr("demo-cube"),
		UserID:        stringPtr(cfg.UserID),
	}

	err = mosCore.Add(ctx, addRequest)
	if err != nil {
		return fmt.Errorf("failed to add memory: %w", err)
	}
	fmt.Println("✓ Added memory through MOS")

	// Search memories through MOS
	searchQuery := &types.SearchQuery{
		Query:     "memory operating system",
		TopK:      3,
		CubeIDs:   []string{"demo-cube"},
		UserID:    cfg.UserID,
		SessionID: cfg.SessionID,
	}

	results, err := mosCore.Search(ctx, searchQuery)
	if err != nil {
		return fmt.Errorf("failed to search memories: %w", err)
	}
	fmt.Printf("✓ Found %d textual memory results\n", len(results.TextMem))

	// Test chat functionality (with mock LLM)
	chatRequest := &types.ChatRequest{
		Query:  "What is MemGOS?",
		UserID: stringPtr(cfg.UserID),
	}

	chatResponse, err := mosCore.Chat(ctx, chatRequest)
	if err != nil {
		return fmt.Errorf("failed to chat: %w", err)
	}
	fmt.Printf("✓ Chat response: %s\n", chatResponse.Response)

	// Get user info
	userInfo, err := mosCore.GetUserInfo(ctx, cfg.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}
	fmt.Printf("✓ User info retrieved for: %s\n", userInfo["user_id"])

	return nil
}

func demonstrateAdvancedOperations(ctx context.Context, tempDir string, logger interfaces.Logger, metrics interfaces.Metrics) error {
	// Create cube registry for advanced operations
	registry := memory.NewCubeRegistry(logger, metrics)
	defer registry.Close()

	// Create multiple cubes
	factory := memory.NewMemCubeFactory(logger, metrics)

	// Create knowledge base cube
	kbCube, err := factory.CreateEmpty("knowledge-base", "Knowledge Base")
	if err != nil {
		return fmt.Errorf("failed to create knowledge base cube: %w", err)
	}
	defer kbCube.Close()

	// Create conversation cube
	convCube, err := factory.CreateEmpty("conversations", "Conversation History")
	if err != nil {
		return fmt.Errorf("failed to create conversation cube: %w", err)
	}
	defer convCube.Close()

	// Register cubes
	err = registry.Register(kbCube)
	if err != nil {
		return fmt.Errorf("failed to register knowledge base cube: %w", err)
	}

	err = registry.Register(convCube)
	if err != nil {
		return fmt.Errorf("failed to register conversation cube: %w", err)
	}

	fmt.Println("✓ Created and registered multiple memory cubes")

	// Add diverse content to knowledge base
	if kbCube.GetTextualMemory() != nil {
		kbMemories := []*types.TextualMemoryItem{
			{
				ID:     "go-overview",
				Memory: "Go (Golang) is a statically typed, compiled programming language designed at Google. It features garbage collection, type safety, and CSP-style concurrency.",
				Metadata: map[string]interface{}{
					"category": "programming-languages",
					"language": "go",
					"level":    "overview",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:     "ai-ml-basics",
				Memory: "Machine Learning is a subset of AI that enables computers to learn and improve from experience without being explicitly programmed. It includes supervised, unsupervised, and reinforcement learning.",
				Metadata: map[string]interface{}{
					"category": "artificial-intelligence",
					"subcategory": "machine-learning",
					"level":    "basics",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:     "memory-systems",
				Memory: "Memory systems in computing include cache memory, RAM, storage, and virtual memory. Each has different characteristics in terms of speed, capacity, and persistence.",
				Metadata: map[string]interface{}{
					"category": "computer-science",
					"subcategory": "memory-systems",
					"level":    "intermediate",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}

		err = kbCube.GetTextualMemory().AddTextual(ctx, kbMemories)
		if err != nil {
			return fmt.Errorf("failed to add knowledge base memories: %w", err)
		}
		fmt.Printf("✓ Added %d memories to knowledge base\n", len(kbMemories))

		// Demonstrate semantic search with filters
		aiResults, err := kbCube.GetTextualMemory().SearchSemantic(ctx, "artificial intelligence machine learning", 2, map[string]string{
			"category": "artificial-intelligence",
		})
		if err != nil {
			return fmt.Errorf("failed to perform semantic search: %w", err)
		}
		fmt.Printf("✓ Semantic search found %d AI-related memories\n", len(aiResults))

		// Demonstrate memory update
		updatedMemory := &types.TextualMemoryItem{
			ID:     "go-overview",
			Memory: "Go (Golang) is a statically typed, compiled programming language designed at Google by Robert Griesemer, Rob Pike, and Ken Thompson. It features garbage collection, type safety, memory safety, and CSP-style concurrency.",
			Metadata: map[string]interface{}{
				"category": "programming-languages",
				"language": "go",
				"level":    "overview",
				"updated":  true,
				"authors":  "Griesemer, Pike, Thompson",
			},
			CreatedAt: kbMemories[0].CreatedAt,
			UpdatedAt: time.Now(),
		}

		err = kbCube.GetTextualMemory().Update(ctx, "go-overview", updatedMemory)
		if err != nil {
			return fmt.Errorf("failed to update memory: %w", err)
		}
		fmt.Println("✓ Updated Go overview memory with additional details")

		// Demonstrate memory retrieval and verification
		retrieved, err := kbCube.GetTextualMemory().Get(ctx, "go-overview")
		if err != nil {
			return fmt.Errorf("failed to retrieve updated memory: %w", err)
		}
		
		if textMem, ok := retrieved.(*types.TextualMemoryItem); ok {
			fmt.Printf("✓ Verified updated memory contains 'authors': %v\n", textMem.Metadata["authors"])
		}
	}

	// Add conversation history
	if convCube.GetTextualMemory() != nil {
		conversations := []*types.TextualMemoryItem{
			{
				ID:     "conv-1",
				Memory: "User asked about Go programming language. Assistant provided overview of language features and use cases.",
				Metadata: map[string]interface{}{
					"type":      "conversation",
					"user_id":   "user-123",
					"session":   "session-456",
					"timestamp": time.Now().Unix(),
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:     "conv-2",
				Memory: "User inquired about memory management in AI systems. Discussion covered memory hierarchies and optimization strategies.",
				Metadata: map[string]interface{}{
					"type":      "conversation",
					"user_id":   "user-123",
					"session":   "session-456",
					"timestamp": time.Now().Unix(),
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}

		err = convCube.GetTextualMemory().AddTextual(ctx, conversations)
		if err != nil {
			return fmt.Errorf("failed to add conversation memories: %w", err)
		}
		fmt.Printf("✓ Added %d conversation memories\n", len(conversations))
	}

	// Demonstrate cube operations
	cubes := registry.List()
	fmt.Printf("✓ Registry contains %d memory cubes:\n", len(cubes))
	for _, cube := range cubes {
		fmt.Printf("  - %s: %s\n", cube.GetID(), cube.GetName())
	}

	// Demonstrate memory persistence
	kbDir := filepath.Join(tempDir, "knowledge-base")
	err = kbCube.Dump(ctx, kbDir)
	if err != nil {
		return fmt.Errorf("failed to dump knowledge base: %w", err)
	}
	fmt.Printf("✓ Dumped knowledge base to: %s\n", kbDir)

	convDir := filepath.Join(tempDir, "conversations")
	err = convCube.Dump(ctx, convDir)
	if err != nil {
		return fmt.Errorf("failed to dump conversations: %w", err)
	}
	fmt.Printf("✓ Dumped conversations to: %s\n", convDir)

	// Verify file creation
	if _, err := os.Stat(filepath.Join(kbDir, "config.json")); os.IsNotExist(err) {
		return fmt.Errorf("knowledge base config file not created")
	}
	fmt.Println("✓ Verified configuration files were created")

	return nil
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}