// Package chunkers provides embedding interfaces for semantic chunking
package chunkers

import (
	"context"
	"fmt"
	"time"
)

// EmbeddingProvider defines the interface for embedding providers
type EmbeddingProvider interface {
	// GetEmbedding returns the embedding vector for a given text
	GetEmbedding(ctx context.Context, text string) ([]float64, error)
	
	// GetEmbeddings returns embedding vectors for multiple texts
	GetEmbeddings(ctx context.Context, texts []string) ([][]float64, error)
	
	// GetModelInfo returns information about the embedding model
	GetModelInfo() EmbeddingModelInfo
	
	// GetDimensions returns the dimensionality of the embedding vectors
	GetDimensions() int
	
	// GetMaxTokens returns the maximum number of tokens supported
	GetMaxTokens() int
	
	// Close releases any resources held by the provider
	Close() error
}

// EmbeddingModelInfo contains information about an embedding model
type EmbeddingModelInfo struct {
	// Name of the model
	Name string `json:"name"`
	
	// Provider of the model (e.g., "openai", "sentence-transformers", "local")
	Provider string `json:"provider"`
	
	// Dimensions of the embedding vectors
	Dimensions int `json:"dimensions"`
	
	// MaxTokens supported by the model
	MaxTokens int `json:"max_tokens"`
	
	// Version of the model
	Version string `json:"version"`
	
	// Description of the model
	Description string `json:"description"`
	
	// Languages supported by the model
	SupportedLanguages []string `json:"supported_languages"`
}

// EmbeddingConfig contains configuration for embedding providers
type EmbeddingConfig struct {
	// Provider specifies which embedding provider to use
	Provider string `json:"provider"`
	
	// ModelName specifies the model to use
	ModelName string `json:"model_name"`
	
	// APIKey for API-based providers
	APIKey string `json:"api_key,omitempty"`
	
	// APIEndpoint for custom API endpoints
	APIEndpoint string `json:"api_endpoint,omitempty"`
	
	// LocalModelPath for local model files
	LocalModelPath string `json:"local_model_path,omitempty"`
	
	// BatchSize for batch processing
	BatchSize int `json:"batch_size"`
	
	// RequestTimeout for API requests
	RequestTimeout time.Duration `json:"request_timeout"`
	
	// RetryAttempts for failed requests
	RetryAttempts int `json:"retry_attempts"`
	
	// CacheEnabled enables embedding caching
	CacheEnabled bool `json:"cache_enabled"`
	
	// CacheSize is the maximum number of embeddings to cache
	CacheSize int `json:"cache_size"`
	
	// CacheTTL is the time-to-live for cached embeddings
	CacheTTL time.Duration `json:"cache_ttl"`
}

// DefaultEmbeddingConfig returns a default embedding configuration
func DefaultEmbeddingConfig() *EmbeddingConfig {
	return &EmbeddingConfig{
		Provider:       "openai",
		ModelName:      "text-embedding-3-small",
		BatchSize:      100,
		RequestTimeout: 30 * time.Second,
		RetryAttempts:  3,
		CacheEnabled:   true,
		CacheSize:      1000,
		CacheTTL:       1 * time.Hour,
	}
}

// EmbeddingCache interface for caching embeddings
type EmbeddingCache interface {
	// Get retrieves an embedding from cache
	Get(ctx context.Context, key string) ([]float64, bool)
	
	// Set stores an embedding in cache
	Set(ctx context.Context, key string, embedding []float64, ttl time.Duration) error
	
	// Delete removes an embedding from cache
	Delete(ctx context.Context, key string) error
	
	// Clear removes all embeddings from cache
	Clear(ctx context.Context) error
	
	// Size returns the current cache size
	Size() int
	
	// Stats returns cache statistics
	Stats() EmbeddingCacheStats
}

// EmbeddingCacheStats contains cache statistics
type EmbeddingCacheStats struct {
	// Size is the current number of cached embeddings
	Size int `json:"size"`
	
	// MaxSize is the maximum cache size
	MaxSize int `json:"max_size"`
	
	// HitCount is the number of cache hits
	HitCount int64 `json:"hit_count"`
	
	// MissCount is the number of cache misses
	MissCount int64 `json:"miss_count"`
	
	// HitRate is the cache hit rate
	HitRate float64 `json:"hit_rate"`
	
	// MemoryUsage is the estimated memory usage in bytes
	MemoryUsage int64 `json:"memory_usage"`
}

// EmbeddingFactory creates embedding providers
type EmbeddingFactory struct {
	cache EmbeddingCache
}

// NewEmbeddingFactory creates a new embedding factory
func NewEmbeddingFactory(cache EmbeddingCache) *EmbeddingFactory {
	return &EmbeddingFactory{
		cache: cache,
	}
}

// CreateProvider creates an embedding provider based on configuration
func (ef *EmbeddingFactory) CreateProvider(config *EmbeddingConfig) (EmbeddingProvider, error) {
	if config == nil {
		config = DefaultEmbeddingConfig()
	}
	
	switch config.Provider {
	case "openai":
		return NewOpenAIEmbeddingProvider(config, ef.cache)
	case "sentence-transformers":
		return NewSentenceTransformerProvider(config, ef.cache)
	case "local":
		return NewLocalEmbeddingProvider(config, ef.cache)
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", config.Provider)
	}
}

// GetSupportedProviders returns a list of supported embedding providers
func (ef *EmbeddingFactory) GetSupportedProviders() []string {
	return []string{"openai", "sentence-transformers", "local"}
}

// ValidateConfig validates an embedding configuration
func (ef *EmbeddingFactory) ValidateConfig(config *EmbeddingConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	
	if config.Provider == "" {
		return fmt.Errorf("provider cannot be empty")
	}
	
	if config.ModelName == "" {
		return fmt.Errorf("model name cannot be empty")
	}
	
	if config.BatchSize <= 0 {
		return fmt.Errorf("batch size must be positive")
	}
	
	if config.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout must be positive")
	}
	
	if config.RetryAttempts < 0 {
		return fmt.Errorf("retry attempts cannot be negative")
	}
	
	if config.CacheEnabled && config.CacheSize <= 0 {
		return fmt.Errorf("cache size must be positive when cache is enabled")
	}
	
	if config.CacheEnabled && config.CacheTTL <= 0 {
		return fmt.Errorf("cache TTL must be positive when cache is enabled")
	}
	
	return nil
}

// EmbeddingProviderDescriptor provides information about embedding providers
type EmbeddingProviderDescriptor struct {
	Provider     string   `json:"provider"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Models       []string `json:"models"`
	Features     []string `json:"features"`
	Requirements []string `json:"requirements"`
}

// GetProviderDescriptors returns descriptive information about all embedding providers
func (ef *EmbeddingFactory) GetProviderDescriptors() []EmbeddingProviderDescriptor {
	return []EmbeddingProviderDescriptor{
		{
			Provider:    "openai",
			Name:        "OpenAI Embeddings",
			Description: "OpenAI's text embedding models via API",
			Models:      []string{"text-embedding-3-large", "text-embedding-3-small", "text-embedding-ada-002"},
			Features:    []string{"High quality", "Fast API", "Multiple model sizes", "Batch processing"},
			Requirements: []string{"API key", "Internet connection"},
		},
		{
			Provider:    "sentence-transformers",
			Name:        "Sentence Transformers",
			Description: "Hugging Face Sentence Transformers models",
			Models:      []string{"all-MiniLM-L6-v2", "all-mpnet-base-v2", "multi-qa-mpnet-base-dot-v1"},
			Features:    []string{"Local inference", "Multiple languages", "Various specialized models"},
			Requirements: []string{"Python environment", "Sentence transformers library"},
		},
		{
			Provider:    "local",
			Name:        "Local Model",
			Description: "Custom local embedding models",
			Models:      []string{"custom"},
			Features:    []string{"Full control", "Privacy", "No API costs"},
			Requirements: []string{"Model files", "Compatible inference engine"},
		},
	}
}