// Package unit provides unit tests for MemGOS components
package unit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/logger"
	"github.com/memtensor/memgos/pkg/memory"
	"github.com/memtensor/memgos/pkg/metrics"
	"github.com/memtensor/memgos/pkg/types"
)

func TestTextualMemoryBasicOperations(t *testing.T) {
	cfg := config.NewMemoryConfig()
	logger := &logger.PlaceholderLogger{Level: "debug"}
	metrics := &metrics.NoOpMetrics{}
	
	textMem, err := memory.NewTextualMemory(cfg, logger, metrics)
	require.NoError(t, err)
	defer textMem.Close()
	
	ctx := context.Background()
	
	// Test adding memories
	item1 := &types.TextualMemoryItem{
		ID:        "test1",
		Memory:    "This is a test memory",
		Metadata:  map[string]interface{}{"source": "test"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	item2 := &types.TextualMemoryItem{
		ID:        "test2",
		Memory:    "Another test memory",
		Metadata:  map[string]interface{}{"source": "test"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	err = textMem.AddTextual(ctx, []*types.TextualMemoryItem{item1, item2})
	require.NoError(t, err)
	
	// Test getting specific memory
	retrieved, err := textMem.Get(ctx, "test1")
	require.NoError(t, err)
	
	textItem, ok := retrieved.(*types.TextualMemoryItem)
	require.True(t, ok)
	assert.Equal(t, "This is a test memory", textItem.Memory)
	
	// Test getting all memories
	allMemories, err := textMem.GetAll(ctx)
	require.NoError(t, err)
	assert.Len(t, allMemories, 2)
	
	// Test search
	searchResults, err := textMem.Search(ctx, "test", 10)
	require.NoError(t, err)
	assert.Len(t, searchResults, 2)
	
	// Test update
	item1.Memory = "Updated test memory"
	err = textMem.Update(ctx, "test1", item1)
	require.NoError(t, err)
	
	updated, err := textMem.Get(ctx, "test1")
	require.NoError(t, err)
	updatedItem, ok := updated.(*types.TextualMemoryItem)
	require.True(t, ok)
	assert.Equal(t, "Updated test memory", updatedItem.Memory)
	
	// Test delete
	err = textMem.Delete(ctx, "test1")
	require.NoError(t, err)
	
	_, err = textMem.Get(ctx, "test1")
	assert.Error(t, err)
	
	// Test delete all
	err = textMem.DeleteAll(ctx)
	require.NoError(t, err)
	
	allMemories, err = textMem.GetAll(ctx)
	require.NoError(t, err)
	assert.Len(t, allMemories, 0)
}

func TestMemoryCubeOperations(t *testing.T) {
	cfg := config.NewMemCubeConfig()
	logger := &logger.PlaceholderLogger{Level: "debug"}
	metrics := &metrics.NoOpMetrics{}
	
	cube, err := memory.NewGeneralMemCube("test-cube", "Test Cube", cfg, logger, metrics)
	require.NoError(t, err)
	defer cube.Close()
	
	assert.Equal(t, "test-cube", cube.GetID())
	assert.Equal(t, "Test Cube", cube.GetName())
	assert.NotNil(t, cube.GetTextualMemory())
	assert.NotNil(t, cube.GetActivationMemory())
	assert.NotNil(t, cube.GetParametricMemory())
}

func TestCubeRegistry(t *testing.T) {
	logger := &logger.PlaceholderLogger{Level: "debug"}
	metrics := &metrics.NoOpMetrics{}
	
	registry := memory.NewCubeRegistry(logger, metrics)
	defer registry.Close()
	
	// Create test cube
	cfg := config.NewMemCubeConfig()
	cube, err := memory.NewGeneralMemCube("test-cube", "Test Cube", cfg, logger, metrics)
	require.NoError(t, err)
	
	// Test registration
	err = registry.Register(cube)
	require.NoError(t, err)
	
	// Test duplicate registration
	err = registry.Register(cube)
	assert.Error(t, err)
	
	// Test retrieval
	retrieved, err := registry.Get("test-cube")
	require.NoError(t, err)
	assert.Equal(t, cube.GetID(), retrieved.GetID())
	
	// Test listing
	cubes := registry.List()
	assert.Len(t, cubes, 1)
	
	// Test unregistration
	err = registry.Unregister("test-cube")
	require.NoError(t, err)
	
	// Test retrieval after unregistration
	_, err = registry.Get("test-cube")
	assert.Error(t, err)
}

func TestMemorySearch(t *testing.T) {
	cfg := config.NewMemoryConfig()
	logger := &logger.PlaceholderLogger{Level: "debug"}
	metrics := &metrics.NoOpMetrics{}
	
	textMem, err := memory.NewTextualMemory(cfg, logger, metrics)
	require.NoError(t, err)
	defer textMem.Close()
	
	ctx := context.Background()
	
	// Add test memories
	memories := []*types.TextualMemoryItem{
		{
			ID:        "1",
			Memory:    "The quick brown fox jumps over the lazy dog",
			Metadata:  map[string]interface{}{"category": "animals"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "2",
			Memory:    "Python is a programming language",
			Metadata:  map[string]interface{}{"category": "programming"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "3",
			Memory:    "Go is also a programming language",
			Metadata:  map[string]interface{}{"category": "programming"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	
	err = textMem.AddTextual(ctx, memories)
	require.NoError(t, err)
	
	// Test exact match search
	results, err := textMem.Search(ctx, "fox", 5)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	
	// Test partial match search
	results, err = textMem.Search(ctx, "programming", 5)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	
	// Test case insensitive search
	results, err = textMem.Search(ctx, "PYTHON", 5)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	
	// Test topK limitation
	results, err = textMem.Search(ctx, "language", 1)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	
	// Test no match
	results, err = textMem.Search(ctx, "nonexistent", 5)
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestMemoryPersistence(t *testing.T) {
	cfg := config.NewMemoryConfig()
	cfg.MemoryFilename = "test_memory.json"
	logger := &logger.PlaceholderLogger{Level: "debug"}
	metrics := &metrics.NoOpMetrics{}
	
	ctx := context.Background()
	tempDir := t.TempDir()
	
	// Create and populate memory
	textMem1, err := memory.NewTextualMemory(cfg, logger, metrics)
	require.NoError(t, err)
	
	item := &types.TextualMemoryItem{
		ID:        "persist-test",
		Memory:    "This memory should persist",
		Metadata:  map[string]interface{}{"test": true},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	err = textMem1.AddTextual(ctx, []*types.TextualMemoryItem{item})
	require.NoError(t, err)
	
	// Save to disk
	err = textMem1.Dump(ctx, tempDir)
	require.NoError(t, err)
	textMem1.Close()
	
	// Create new memory instance and load
	textMem2, err := memory.NewTextualMemory(cfg, logger, metrics)
	require.NoError(t, err)
	defer textMem2.Close()
	
	err = textMem2.Load(ctx, tempDir)
	require.NoError(t, err)
	
	// Verify data was loaded
	retrieved, err := textMem2.Get(ctx, "persist-test")
	require.NoError(t, err)
	
	textItem, ok := retrieved.(*types.TextualMemoryItem)
	require.True(t, ok)
	assert.Equal(t, "This memory should persist", textItem.Memory)
	assert.True(t, textItem.Metadata["test"].(bool))
}

func TestMemoryConfiguration(t *testing.T) {
	// Test default configuration
	cfg := config.NewMemoryConfig()
	assert.Equal(t, types.MemoryBackendNaive, cfg.Backend)
	assert.Equal(t, "memory.json", cfg.MemoryFilename)
	assert.Equal(t, 5, cfg.TopK)
	
	// Test configuration validation
	err := cfg.Validate()
	require.NoError(t, err)
	
	// Test invalid configuration
	cfg.Backend = ""
	err = cfg.Validate()
	assert.Error(t, err)
}

func TestMemoryCubeConfiguration(t *testing.T) {
	cfg := config.NewMemCubeConfig()
	assert.NotNil(t, cfg.TextMem)
	assert.NotNil(t, cfg.ActMem)
	assert.NotNil(t, cfg.ParaMem)
	
	err := cfg.Validate()
	require.NoError(t, err)
}