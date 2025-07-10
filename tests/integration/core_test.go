// Package integration provides integration tests for MemGOS core functionality
package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/core"
	"github.com/memtensor/memgos/pkg/logger"
	"github.com/memtensor/memgos/pkg/memory"
	"github.com/memtensor/memgos/pkg/metrics"
	"github.com/memtensor/memgos/pkg/types"
)

func TestMOSCoreIntegration(t *testing.T) {
	// Setup test environment
	tempDir := t.TempDir()
	ctx := context.Background()

	// Create test configuration
	cfg := config.NewMOSConfig()
	cfg.UserID = "test-user-123"
	cfg.SessionID = "test-session-456"
	cfg.EnableTextualMemory = true
	cfg.EnableActivationMemory = false
	cfg.EnableParametricMemory = false
	cfg.EnableMemScheduler = false
	cfg.TopK = 3

	// Initialize logger and metrics
	testLogger := logger.NewTestLogger()
	testMetrics := metrics.NewTestMetrics()

	t.Run("MOS Core Initialization", func(t *testing.T) {
		mosCore, err := core.NewMOSCore(cfg, testLogger, testMetrics)
		require.NoError(t, err)
		assert.NotNil(t, mosCore)

		err = mosCore.Initialize(ctx)
		require.NoError(t, err)

		// Clean up
		err = mosCore.Close()
		assert.NoError(t, err)
	})

	t.Run("Memory Cube Registration and Operations", func(t *testing.T) {
		mosCore, err := core.NewMOSCore(cfg, testLogger, testMetrics)
		require.NoError(t, err)

		err = mosCore.Initialize(ctx)
		require.NoError(t, err)
		defer mosCore.Close()

		// Create a test memory cube directory
		cubeDir := filepath.Join(tempDir, "test_cube")
		err = os.MkdirAll(cubeDir, 0755)
		require.NoError(t, err)

		// Create cube configuration
		cubeConfig := config.NewMemCubeConfig()
		cubeConfig.TextMem.MemoryFilename = "textual_memory.json"
		err = cubeConfig.ToJSONFile(filepath.Join(cubeDir, "config.json"))
		require.NoError(t, err)

		// Register memory cube
		err = mosCore.RegisterMemCube(ctx, cubeDir, "test-cube", cfg.UserID)
		require.NoError(t, err)

		// Test adding memories
		request := &types.AddMemoryRequest{
			MemoryContent: stringPtr("This is a test memory about artificial intelligence"),
			CubeID:        stringPtr("test-cube"),
			UserID:        stringPtr(cfg.UserID),
		}

		err = mosCore.Add(ctx, request)
		require.NoError(t, err)

		// Test searching memories
		searchQuery := &types.SearchQuery{
			Query:     "artificial intelligence",
			TopK:      3,
			CubeIDs:   []string{"test-cube"},
			UserID:    cfg.UserID,
			SessionID: cfg.SessionID,
		}

		results, err := mosCore.Search(ctx, searchQuery)
		require.NoError(t, err)
		assert.NotNil(t, results)
		assert.Greater(t, len(results.TextMem), 0)

		// Test getting all memories
		allResults, err := mosCore.GetAll(ctx, "test-cube", cfg.UserID)
		require.NoError(t, err)
		assert.NotNil(t, allResults)
		assert.Greater(t, len(allResults.TextMem), 0)

		// Test unregistering memory cube
		err = mosCore.UnregisterMemCube(ctx, "test-cube", cfg.UserID)
		require.NoError(t, err)
	})

	t.Run("Chat Functionality", func(t *testing.T) {
		mosCore, err := core.NewMOSCore(cfg, testLogger, testMetrics)
		require.NoError(t, err)

		err = mosCore.Initialize(ctx)
		require.NoError(t, err)
		defer mosCore.Close()

		// Create and register a test cube with some memories
		cubeDir := filepath.Join(tempDir, "chat_test_cube")
		err = os.MkdirAll(cubeDir, 0755)
		require.NoError(t, err)

		cubeConfig := config.NewMemCubeConfig()
		err = cubeConfig.ToJSONFile(filepath.Join(cubeDir, "config.json"))
		require.NoError(t, err)

		err = mosCore.RegisterMemCube(ctx, cubeDir, "chat-cube", cfg.UserID)
		require.NoError(t, err)

		// Add some test memories
		addRequest := &types.AddMemoryRequest{
			MemoryContent: stringPtr("Go is a programming language developed by Google"),
			CubeID:        stringPtr("chat-cube"),
			UserID:        stringPtr(cfg.UserID),
		}
		err = mosCore.Add(ctx, addRequest)
		require.NoError(t, err)

		// Test chat functionality
		chatRequest := &types.ChatRequest{
			Query:  "Tell me about Go programming language",
			UserID: stringPtr(cfg.UserID),
		}

		response, err := mosCore.Chat(ctx, chatRequest)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.Response)
		assert.Equal(t, cfg.UserID, response.UserID)
		assert.Equal(t, cfg.SessionID, response.SessionID)
	})

	t.Run("User Management", func(t *testing.T) {
		mosCore, err := core.NewMOSCore(cfg, testLogger, testMetrics)
		require.NoError(t, err)

		err = mosCore.Initialize(ctx)
		require.NoError(t, err)
		defer mosCore.Close()

		// Test creating a user
		newUserID, err := mosCore.CreateUser(ctx, "Test User", types.UserRoleUser, "new-user-123")
		require.NoError(t, err)
		assert.Equal(t, "new-user-123", newUserID)

		// Test getting user info
		userInfo, err := mosCore.GetUserInfo(ctx, cfg.UserID)
		require.NoError(t, err)
		assert.NotNil(t, userInfo)
		assert.Equal(t, cfg.UserID, userInfo["user_id"])
	})
}

func TestMemCubeIntegration(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	// Initialize logger and metrics
	testLogger := logger.NewTestLogger()
	testMetrics := metrics.NewTestMetrics()

	t.Run("Memory Cube Factory Operations", func(t *testing.T) {
		factory := memory.NewMemCubeFactory(testLogger, testMetrics)

		// Test creating empty cube
		cube, err := factory.CreateEmpty("test-cube", "Test Cube")
		require.NoError(t, err)
		assert.NotNil(t, cube)
		assert.Equal(t, "test-cube", cube.GetID())
		assert.Equal(t, "Test Cube", cube.GetName())

		// Test dumping cube
		cubeDir := filepath.Join(tempDir, "dump_test")
		err = cube.Dump(ctx, cubeDir)
		require.NoError(t, err)

		// Verify config file was created
		configPath := filepath.Join(cubeDir, "config.json")
		assert.FileExists(t, configPath)

		// Test loading cube from directory
		loadedCube, err := factory.CreateFromDirectory(ctx, "loaded-cube", "Loaded Cube", cubeDir)
		require.NoError(t, err)
		assert.NotNil(t, loadedCube)

		// Clean up
		err = cube.Close()
		assert.NoError(t, err)
		err = loadedCube.Close()
		assert.NoError(t, err)
	})

	t.Run("Memory Operations", func(t *testing.T) {
		// Create memory configuration
		memConfig := config.NewMemoryConfig()
		memConfig.MemoryFilename = "test_memory.json"

		// Test textual memory
		textMem, err := memory.NewTextualMemory(memConfig, testLogger, testMetrics)
		require.NoError(t, err)

		// Add memories
		memories := []*types.TextualMemoryItem{
			{
				ID:     "mem-1",
				Memory: "The capital of France is Paris",
				Metadata: map[string]interface{}{
					"source": "test",
					"topic":  "geography",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:     "mem-2",
				Memory: "Go is a programming language",
				Metadata: map[string]interface{}{
					"source": "test",
					"topic":  "programming",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}

		err = textMem.AddTextual(ctx, memories)
		require.NoError(t, err)

		// Test search
		results, err := textMem.Search(ctx, "France", 10)
		require.NoError(t, err)
		assert.Greater(t, len(results), 0)

		// Test semantic search
		semanticResults, err := textMem.SearchSemantic(ctx, "programming", 10, nil)
		require.NoError(t, err)
		assert.Greater(t, len(semanticResults), 0)

		// Test get by ID
		retrieved, err := textMem.Get(ctx, "mem-1")
		require.NoError(t, err)
		assert.NotNil(t, retrieved)

		// Test update
		updatedMem := &types.TextualMemoryItem{
			ID:     "mem-1",
			Memory: "The capital of France is Paris, a beautiful city",
			Metadata: map[string]interface{}{
				"source":  "test",
				"topic":   "geography",
				"updated": true,
			},
			CreatedAt: memories[0].CreatedAt,
			UpdatedAt: time.Now(),
		}

		err = textMem.Update(ctx, "mem-1", updatedMem)
		require.NoError(t, err)

		// Test delete
		err = textMem.Delete(ctx, "mem-2")
		require.NoError(t, err)

		// Verify deletion
		_, err = textMem.Get(ctx, "mem-2")
		assert.Error(t, err)

		// Test dump and load
		memDir := filepath.Join(tempDir, "memory_test")
		err = os.MkdirAll(memDir, 0755)
		require.NoError(t, err)

		err = textMem.Dump(ctx, memDir)
		require.NoError(t, err)

		// Load memories
		newTextMem, err := memory.NewTextualMemory(memConfig, testLogger, testMetrics)
		require.NoError(t, err)

		err = newTextMem.Load(ctx, memDir)
		require.NoError(t, err)

		// Verify loaded memories
		allMemories, err := newTextMem.GetAll(ctx)
		require.NoError(t, err)
		assert.Greater(t, len(allMemories), 0)

		// Clean up
		err = textMem.Close()
		assert.NoError(t, err)
		err = newTextMem.Close()
		assert.NoError(t, err)
	})
}

func TestCubeRegistry(t *testing.T) {
	ctx := context.Background()
	testLogger := logger.NewTestLogger()
	testMetrics := metrics.NewTestMetrics()

	t.Run("Cube Registry Operations", func(t *testing.T) {
		registry := memory.NewCubeRegistry(testLogger, testMetrics)

		// Create test cubes
		factory := memory.NewMemCubeFactory(testLogger, testMetrics)
		cube1, err := factory.CreateEmpty("cube-1", "Test Cube 1")
		require.NoError(t, err)

		cube2, err := factory.CreateEmpty("cube-2", "Test Cube 2")
		require.NoError(t, err)

		// Register cubes
		err = registry.Register(cube1)
		require.NoError(t, err)

		err = registry.Register(cube2)
		require.NoError(t, err)

		// Test duplicate registration
		err = registry.Register(cube1)
		assert.Error(t, err)

		// Test retrieval
		retrieved, err := registry.Get("cube-1")
		require.NoError(t, err)
		assert.Equal(t, "cube-1", retrieved.GetID())

		// Test list
		cubes := registry.List()
		assert.Len(t, cubes, 2)

		// Test unregister
		err = registry.Unregister("cube-1")
		require.NoError(t, err)

		// Verify unregistration
		_, err = registry.Get("cube-1")
		assert.Error(t, err)

		// Clean up
		err = registry.Close()
		assert.NoError(t, err)
	})
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}