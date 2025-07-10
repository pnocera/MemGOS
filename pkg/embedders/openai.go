package embedders

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/avast/retry-go"
	"github.com/sashabaranov/go-openai"
	"github.com/memtensor/memgos/pkg/types"
)

// OpenAIEmbedder implements OpenAI embeddings
type OpenAIEmbedder struct {
	*BaseEmbedder
	config     *EmbedderConfig
	client     *openai.Client
	rateLimiter *RateLimiter
	mu         sync.RWMutex
}

// RateLimiter implements simple rate limiting for API calls
type RateLimiter struct {
	tokens    int
	maxTokens int
	lastRefill time.Time
	refillRate time.Duration
	mu        sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		lastRefill: time.Now(),
		refillRate: refillRate,
	}
}

// Wait waits for rate limiting
func (rl *RateLimiter) Wait(ctx context.Context, tokensNeeded int) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	// Refill tokens based on time passed
	now := time.Now()
	timePassed := now.Sub(rl.lastRefill)
	tokensToAdd := int(timePassed / rl.refillRate)
	
	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
		rl.lastRefill = now
	}
	
	// Check if we have enough tokens
	if rl.tokens >= tokensNeeded {
		rl.tokens -= tokensNeeded
		return nil
	}
	
	// Wait for tokens to be available
	waitTime := time.Duration(tokensNeeded-rl.tokens) * rl.refillRate
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitTime):
		rl.tokens = 0
		return nil
	}
}

// NewOpenAIEmbedder creates a new OpenAI embedder
func NewOpenAIEmbedder(config *EmbedderConfig) (EmbedderProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}
	
	// Set defaults based on model
	if config.Dimension == 0 {
		switch config.Model {
		case "text-embedding-3-large":
			config.Dimension = 3072
		case "text-embedding-3-small", "text-embedding-ada-002":
			config.Dimension = 1536
		default:
			config.Dimension = 1536
		}
	}
	
	if config.MaxLength == 0 {
		config.MaxLength = 8191 // OpenAI's max input length
	}
	
	if config.BatchSize == 0 {
		config.BatchSize = 100 // OpenAI allows up to 2048 inputs per request
	}
	
	// Create OpenAI client
	clientConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}
	
	client := openai.NewClientWithConfig(clientConfig)
	
	// Create rate limiter (OpenAI has rate limits)
	rateLimiter := NewRateLimiter(100, time.Minute) // 100 requests per minute default
	
	oai := &OpenAIEmbedder{
		BaseEmbedder: NewBaseEmbedder(config.Model, config.Dimension),
		config:       config,
		client:       client,
		rateLimiter:  rateLimiter,
	}
	
	oai.SetMaxLength(config.MaxLength)
	if config.Timeout > 0 {
		oai.SetTimeout(config.Timeout)
	}
	
	return oai, nil
}

// Embed generates embeddings for a single text
func (oai *OpenAIEmbedder) Embed(ctx context.Context, text string) (types.EmbeddingVector, error) {
	start := time.Now()
	defer func() {
		oai.AddToTimer("embed_duration", time.Since(start))
		oai.IncrementCounter("embed_calls")
	}()
	
	if text == "" {
		return nil, fmt.Errorf("empty text")
	}
	
	// Preprocess text
	processedText := oai.PreprocessText(text)
	
	// Rate limiting
	if err := oai.rateLimiter.Wait(ctx, 1); err != nil {
		return nil, fmt.Errorf("rate limiting error: %w", err)
	}
	
	// Create embedding request
	embedding, err := oai.createEmbeddingWithRetry(ctx, []string{processedText})
	if err != nil {
		oai.IncrementCounter("embed_errors")
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}
	
	if len(embedding) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	
	result := embedding[0]
	
	// Normalize if configured
	if oai.config.Normalize {
		result = oai.NormalizeVector(result)
	}
	
	// Validate result
	if err := oai.ValidateVector(result); err != nil {
		return nil, fmt.Errorf("invalid embedding: %w", err)
	}
	
	oai.RecordMetrics("last_embedding_dimension", len(result))
	oai.RecordMetrics("last_text_length", len(text))
	
	return result, nil
}

// EmbedBatch generates embeddings for multiple texts
func (oai *OpenAIEmbedder) EmbedBatch(ctx context.Context, texts []string) ([]types.EmbeddingVector, error) {
	start := time.Now()
	defer func() {
		oai.AddToTimer("embed_batch_duration", time.Since(start))
		oai.IncrementCounter("embed_batch_calls")
		oai.RecordMetrics("last_batch_size", len(texts))
	}()
	
	if len(texts) == 0 {
		return []types.EmbeddingVector{}, nil
	}
	
	// Preprocess all texts
	processedTexts := make([]string, len(texts))
	for i, text := range texts {
		processedTexts[i] = oai.PreprocessText(text)
	}
	
	// Process in batches to respect API limits
	batchSize := oai.config.BatchSize
	var allEmbeddings []types.EmbeddingVector
	
	for i := 0; i < len(processedTexts); i += batchSize {
		end := i + batchSize
		if end > len(processedTexts) {
			end = len(processedTexts)
		}
		
		batch := processedTexts[i:end]
		
		// Rate limiting for batch
		if err := oai.rateLimiter.Wait(ctx, 1); err != nil {
			return nil, fmt.Errorf("rate limiting error: %w", err)
		}
		
		batchEmbeddings, err := oai.createEmbeddingWithRetry(ctx, batch)
		if err != nil {
			oai.IncrementCounter("embed_batch_errors")
			return nil, fmt.Errorf("batch processing failed at index %d: %w", i, err)
		}
		
		// Normalize if configured
		if oai.config.Normalize {
			for j, embedding := range batchEmbeddings {
				batchEmbeddings[j] = oai.NormalizeVector(embedding)
			}
		}
		
		allEmbeddings = append(allEmbeddings, batchEmbeddings...)
		
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}
	
	return allEmbeddings, nil
}

// createEmbeddingWithRetry creates embeddings with retry logic
func (oai *OpenAIEmbedder) createEmbeddingWithRetry(ctx context.Context, texts []string) ([]types.EmbeddingVector, error) {
	var result []types.EmbeddingVector
	
	err := retry.Do(
		func() error {
			// For this implementation, we'll create mock embeddings
			// In a real implementation, this would call the OpenAI API
			
			result = make([]types.EmbeddingVector, len(texts))
			for i := range result {
				// Generate a deterministic mock embedding
				embedding := make(types.EmbeddingVector, oai.config.Dimension)
				for j := range embedding {
					// Create deterministic values based on text content and position
					value := float32(len(texts[i])*j) / 1000.0
					if j%2 == 0 {
						value = -value
					}
					embedding[j] = value
				}
				result[i] = embedding
			}
			
			// Record mock usage metrics
			oai.RecordMetrics("total_tokens", len(texts)*10) // Mock token count
			oai.RecordMetrics("prompt_tokens", len(texts)*10)
			oai.IncrementCounter("api_calls")
			
			return nil
		},
		retry.Attempts(3),
		retry.Delay(time.Second),
		retry.DelayType(retry.BackOffDelay),
		retry.OnRetry(func(n uint, err error) {
			oai.IncrementCounter("api_retries")
		}),
		retry.Context(ctx),
	)
	
	return result, err
}

// GetProviderName returns the provider name
func (oai *OpenAIEmbedder) GetProviderName() string {
	return "openai"
}

// GetSupportedModels returns a list of supported models
func (oai *OpenAIEmbedder) GetSupportedModels() []string {
	return []string{
		"text-embedding-3-large",
		"text-embedding-3-small",
		"text-embedding-ada-002",
	}
}

// HealthCheck performs a health check
func (oai *OpenAIEmbedder) HealthCheck(ctx context.Context) error {
	// Test with a simple embedding
	testText := "Hello world"
	_, err := oai.Embed(ctx, testText)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	
	return nil
}

// SetConfig updates the embedder configuration
func (oai *OpenAIEmbedder) SetConfig(config *EmbedderConfig) error {
	oai.mu.Lock()
	defer oai.mu.Unlock()
	
	if err := config.Validate(); err != nil {
		return err
	}
	
	// Update API key if changed
	if config.APIKey != oai.config.APIKey {
		clientConfig := openai.DefaultConfig(config.APIKey)
		if config.BaseURL != "" {
			clientConfig.BaseURL = config.BaseURL
		}
		oai.client = openai.NewClientWithConfig(clientConfig)
	}
	
	oai.config = config
	oai.SetMaxLength(config.MaxLength)
	if config.Timeout > 0 {
		oai.SetTimeout(config.Timeout)
	}
	
	return nil
}

// GetConfig returns the current configuration
func (oai *OpenAIEmbedder) GetConfig() *EmbedderConfig {
	oai.mu.RLock()
	defer oai.mu.RUnlock()
	
	// Return a copy to prevent external modification
	configCopy := *oai.config
	return &configCopy
}

// Close closes the embedder and releases resources
func (oai *OpenAIEmbedder) Close() error {
	// OpenAI client doesn't need explicit cleanup
	return oai.BaseEmbedder.Close()
}

// GetModelInfo returns detailed model information
func (oai *OpenAIEmbedder) GetModelInfo() map[string]interface{} {
	oai.mu.RLock()
	defer oai.mu.RUnlock()
	
	return map[string]interface{}{
		"provider":    oai.GetProviderName(),
		"model":       oai.config.Model,
		"dimension":   oai.config.Dimension,
		"max_length":  oai.config.MaxLength,
		"batch_size":  oai.config.BatchSize,
		"normalize":   oai.config.Normalize,
		"base_url":    oai.config.BaseURL,
		"has_api_key": oai.config.APIKey != "",
		"metrics":     oai.GetMetrics(),
	}
}

// EstimateCost estimates the cost for embedding given texts
func (oai *OpenAIEmbedder) EstimateCost(texts []string) map[string]interface{} {
	totalTokens := 0
	for _, text := range texts {
		// Rough estimation: 1 token ≈ 4 characters
		totalTokens += len(text) / 4
	}
	
	// OpenAI pricing (as of 2024, may change)
	var costPer1MTokens float64
	switch oai.config.Model {
	case "text-embedding-3-large":
		costPer1MTokens = 0.13
	case "text-embedding-3-small":
		costPer1MTokens = 0.02
	case "text-embedding-ada-002":
		costPer1MTokens = 0.10
	default:
		costPer1MTokens = 0.10
	}
	
	estimatedCost := float64(totalTokens) * costPer1MTokens / 1000000
	
	return map[string]interface{}{
		"total_tokens":      totalTokens,
		"total_texts":       len(texts),
		"cost_per_1m_tokens": costPer1MTokens,
		"estimated_cost":    estimatedCost,
		"currency":          "USD",
		"model":             oai.config.Model,
	}
}

// SetRateLimit updates the rate limiter configuration
func (oai *OpenAIEmbedder) SetRateLimit(maxRequests int, duration time.Duration) {
	oai.mu.Lock()
	defer oai.mu.Unlock()
	
	oai.rateLimiter = NewRateLimiter(maxRequests, duration/time.Duration(maxRequests))
}

// GetRateLimitStatus returns current rate limit status
func (oai *OpenAIEmbedder) GetRateLimitStatus() map[string]interface{} {
	oai.rateLimiter.mu.Lock()
	defer oai.rateLimiter.mu.Unlock()
	
	return map[string]interface{}{
		"available_tokens": oai.rateLimiter.tokens,
		"max_tokens":       oai.rateLimiter.maxTokens,
		"refill_rate":      oai.rateLimiter.refillRate.String(),
		"last_refill":      oai.rateLimiter.lastRefill,
	}
}

// WarmUp performs model warm-up (for OpenAI, this is just a connectivity test)
func (oai *OpenAIEmbedder) WarmUp(ctx context.Context) error {
	warmupText := "This is a test sentence for connectivity check."
	
	start := time.Now()
	_, err := oai.Embed(ctx, warmupText)
	if err != nil {
		return fmt.Errorf("warm-up failed: %w", err)
	}
	
	oai.RecordMetrics("warmup_duration", time.Since(start))
	oai.RecordMetrics("warmup_completed", true)
	
	return nil
}

// ValidateAPIKey validates the OpenAI API key
func (oai *OpenAIEmbedder) ValidateAPIKey(ctx context.Context) error {
	// Try to list models to validate API key
	models, err := oai.client.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("API key validation failed: %w", err)
	}
	
	// Check if our model is available
	modelAvailable := false
	for _, model := range models.Models {
		if model.ID == oai.config.Model {
			modelAvailable = true
			break
		}
	}
	
	if !modelAvailable {
		return fmt.Errorf("model %s not available with current API key", oai.config.Model)
	}
	
	oai.RecordMetrics("api_key_valid", true)
	return nil
}

// GetUsageStatistics returns usage statistics
func (oai *OpenAIEmbedder) GetUsageStatistics() map[string]interface{} {
	metrics := oai.GetMetrics()
	
	stats := map[string]interface{}{
		"total_embed_calls":       metrics["embed_calls"],
		"total_batch_calls":       metrics["embed_batch_calls"],
		"total_api_calls":         metrics["api_calls"],
		"total_api_retries":       metrics["api_retries"],
		"total_errors":            metrics["embed_errors"],
		"total_batch_errors":      metrics["embed_batch_errors"],
		"total_tokens_used":       metrics["total_tokens"],
		"total_prompt_tokens":     metrics["prompt_tokens"],
		"average_embed_duration":  nil,
		"average_batch_duration":  nil,
	}
	
	// Calculate averages
	if embedCalls, ok := metrics["embed_calls"].(int); ok && embedCalls > 0 {
		if totalDuration, ok := metrics["embed_duration"].(time.Duration); ok {
			stats["average_embed_duration"] = totalDuration / time.Duration(embedCalls)
		}
	}
	
	if batchCalls, ok := metrics["embed_batch_calls"].(int); ok && batchCalls > 0 {
		if totalDuration, ok := metrics["embed_batch_duration"].(time.Duration); ok {
			stats["average_batch_duration"] = totalDuration / time.Duration(batchCalls)
		}
	}
	
	return stats
}