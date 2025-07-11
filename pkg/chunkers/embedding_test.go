package chunkers

import (
	"context"
	"testing"
	"time"
)

func TestEmbeddingFactory(t *testing.T) {
	factory := NewEmbeddingFactory(NewNoOpEmbeddingCache())
	
	// Test supported providers
	providers := factory.GetSupportedProviders()
	expectedProviders := []string{"openai", "sentence-transformers", "local"}
	
	if len(providers) != len(expectedProviders) {
		t.Errorf("Expected %d providers, got %d", len(expectedProviders), len(providers))
	}
	
	for _, expected := range expectedProviders {
		found := false
		for _, provider := range providers {
			if provider == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected provider %s not found", expected)
		}
	}
}

func TestEmbeddingConfig(t *testing.T) {
	config := DefaultEmbeddingConfig()
	
	if config.Provider != "openai" {
		t.Errorf("Expected default provider to be 'openai', got %s", config.Provider)
	}
	
	if config.ModelName != "text-embedding-3-small" {
		t.Errorf("Expected default model to be 'text-embedding-3-small', got %s", config.ModelName)
	}
	
	if config.BatchSize != 100 {
		t.Errorf("Expected default batch size to be 100, got %d", config.BatchSize)
	}
	
	if config.RequestTimeout != 30*time.Second {
		t.Errorf("Expected default timeout to be 30s, got %v", config.RequestTimeout)
	}
	
	if !config.CacheEnabled {
		t.Error("Expected cache to be enabled by default")
	}
}

func TestEmbeddingFactoryValidation(t *testing.T) {
	factory := NewEmbeddingFactory(NewNoOpEmbeddingCache())
	
	// Test nil config
	err := factory.ValidateConfig(nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}
	
	// Test empty provider
	config := &EmbeddingConfig{
		Provider: "",
	}
	err = factory.ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for empty provider")
	}
	
	// Test empty model name
	config = &EmbeddingConfig{
		Provider:  "openai",
		ModelName: "",
	}
	err = factory.ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for empty model name")
	}
	
	// Test invalid batch size
	config = &EmbeddingConfig{
		Provider:  "openai",
		ModelName: "text-embedding-3-small",
		BatchSize: 0,
	}
	err = factory.ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for invalid batch size")
	}
	
	// Test valid config
	config = DefaultEmbeddingConfig()
	err = factory.ValidateConfig(config)
	if err != nil {
		t.Errorf("Expected no error for valid config, got %v", err)
	}
}

func TestMemoryEmbeddingCache(t *testing.T) {
	cache := NewMemoryEmbeddingCache(3)
	ctx := context.Background()
	
	// Test empty cache
	_, found := cache.Get(ctx, "key1")
	if found {
		t.Error("Expected empty cache to return false")
	}
	
	if cache.Size() != 0 {
		t.Errorf("Expected cache size to be 0, got %d", cache.Size())
	}
	
	// Test set and get
	embedding := []float64{0.1, 0.2, 0.3}
	err := cache.Set(ctx, "key1", embedding, time.Hour)
	if err != nil {
		t.Errorf("Expected no error setting cache, got %v", err)
	}
	
	retrieved, found := cache.Get(ctx, "key1")
	if !found {
		t.Error("Expected to find cached embedding")
	}
	
	if len(retrieved) != len(embedding) {
		t.Errorf("Expected embedding length %d, got %d", len(embedding), len(retrieved))
	}
	
	for i, val := range embedding {
		if retrieved[i] != val {
			t.Errorf("Expected embedding[%d] = %f, got %f", i, val, retrieved[i])
		}
	}
	
	// Test cache eviction
	cache.Set(ctx, "key2", []float64{0.4, 0.5}, time.Hour)
	cache.Set(ctx, "key3", []float64{0.6, 0.7}, time.Hour)
	cache.Set(ctx, "key4", []float64{0.8, 0.9}, time.Hour) // Should evict key1
	
	if cache.Size() != 3 {
		t.Errorf("Expected cache size to be 3, got %d", cache.Size())
	}
	
	// key1 should be evicted
	_, found = cache.Get(ctx, "key1")
	if found {
		t.Error("Expected key1 to be evicted")
	}
	
	// key4 should be present
	_, found = cache.Get(ctx, "key4")
	if !found {
		t.Error("Expected key4 to be present")
	}
}

func TestEmbeddingCacheExpiration(t *testing.T) {
	cache := NewMemoryEmbeddingCache(10)
	ctx := context.Background()
	
	// Set with short TTL
	embedding := []float64{0.1, 0.2, 0.3}
	err := cache.Set(ctx, "key1", embedding, 10*time.Millisecond)
	if err != nil {
		t.Errorf("Expected no error setting cache, got %v", err)
	}
	
	// Should be available immediately
	_, found := cache.Get(ctx, "key1")
	if !found {
		t.Error("Expected to find cached embedding")
	}
	
	// Wait for expiration
	time.Sleep(20 * time.Millisecond)
	
	// Should be expired
	_, found = cache.Get(ctx, "key1")
	if found {
		t.Error("Expected cached embedding to be expired")
	}
}

func TestEmbeddingCacheStats(t *testing.T) {
	cache := NewMemoryEmbeddingCache(10)
	ctx := context.Background()
	
	// Initial stats
	stats := cache.Stats()
	if stats.Size != 0 {
		t.Errorf("Expected initial size to be 0, got %d", stats.Size)
	}
	if stats.HitCount != 0 {
		t.Errorf("Expected initial hit count to be 0, got %d", stats.HitCount)
	}
	if stats.MissCount != 0 {
		t.Errorf("Expected initial miss count to be 0, got %d", stats.MissCount)
	}
	
	// Add some data
	cache.Set(ctx, "key1", []float64{0.1, 0.2}, time.Hour)
	cache.Set(ctx, "key2", []float64{0.3, 0.4}, time.Hour)
	
	// Test miss
	cache.Get(ctx, "nonexistent")
	
	// Test hit
	cache.Get(ctx, "key1")
	
	stats = cache.Stats()
	if stats.Size != 2 {
		t.Errorf("Expected size to be 2, got %d", stats.Size)
	}
	if stats.HitCount != 1 {
		t.Errorf("Expected hit count to be 1, got %d", stats.HitCount)
	}
	if stats.MissCount != 1 {
		t.Errorf("Expected miss count to be 1, got %d", stats.MissCount)
	}
	if stats.HitRate != 0.5 {
		t.Errorf("Expected hit rate to be 0.5, got %f", stats.HitRate)
	}
}

func TestNoOpEmbeddingCache(t *testing.T) {
	cache := NewNoOpEmbeddingCache()
	ctx := context.Background()
	
	// Test that no-op cache never returns anything
	_, found := cache.Get(ctx, "key1")
	if found {
		t.Error("Expected no-op cache to never return true")
	}
	
	// Test that set doesn't error
	err := cache.Set(ctx, "key1", []float64{0.1, 0.2}, time.Hour)
	if err != nil {
		t.Errorf("Expected no error from no-op cache set, got %v", err)
	}
	
	// Test that size is always 0
	if cache.Size() != 0 {
		t.Errorf("Expected no-op cache size to be 0, got %d", cache.Size())
	}
	
	// Test that stats are empty
	stats := cache.Stats()
	if stats.Size != 0 || stats.HitCount != 0 || stats.MissCount != 0 {
		t.Error("Expected no-op cache stats to be empty")
	}
}

func TestOpenAIModelInfo(t *testing.T) {
	testCases := []struct {
		modelName      string
		expectedDims   int
		expectedTokens int
	}{
		{"text-embedding-3-large", 3072, 8192},
		{"text-embedding-3-small", 1536, 8192},
		{"text-embedding-ada-002", 1536, 8192},
		{"unknown-model", 1536, 8192}, // Default
	}
	
	for _, tc := range testCases {
		info := getOpenAIModelInfo(tc.modelName)
		if info.Dimensions != tc.expectedDims {
			t.Errorf("Expected dimensions %d for model %s, got %d", 
				tc.expectedDims, tc.modelName, info.Dimensions)
		}
		if info.MaxTokens != tc.expectedTokens {
			t.Errorf("Expected max tokens %d for model %s, got %d", 
				tc.expectedTokens, tc.modelName, info.MaxTokens)
		}
		if info.Provider != "openai" {
			t.Errorf("Expected provider to be 'openai', got %s", info.Provider)
		}
	}
}