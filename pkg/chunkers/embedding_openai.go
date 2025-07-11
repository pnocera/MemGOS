package chunkers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAIEmbeddingProvider implements EmbeddingProvider for OpenAI embeddings
type OpenAIEmbeddingProvider struct {
	config     *EmbeddingConfig
	cache      EmbeddingCache
	httpClient *http.Client
	modelInfo  EmbeddingModelInfo
}

// NewOpenAIEmbeddingProvider creates a new OpenAI embedding provider
func NewOpenAIEmbeddingProvider(config *EmbeddingConfig, cache EmbeddingCache) (*OpenAIEmbeddingProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required for OpenAI provider")
	}

	// Set default endpoint if not provided
	if config.APIEndpoint == "" {
		config.APIEndpoint = "https://api.openai.com/v1/embeddings"
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.RequestTimeout,
	}

	// Get model info
	modelInfo := getOpenAIModelInfo(config.ModelName)

	return &OpenAIEmbeddingProvider{
		config:     config,
		cache:      cache,
		httpClient: httpClient,
		modelInfo:  modelInfo,
	}, nil
}

// GetEmbedding returns the embedding vector for a given text
func (p *OpenAIEmbeddingProvider) GetEmbedding(ctx context.Context, text string) ([]float64, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// Check cache first
	if p.cache != nil && p.config.CacheEnabled {
		cacheKey := p.generateCacheKey(text)
		if embedding, found := p.cache.Get(ctx, cacheKey); found {
			return embedding, nil
		}
	}

	// Prepare request
	embeddings, err := p.GetEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding: %w", err)
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return embeddings[0], nil
}

// GetEmbeddings returns embedding vectors for multiple texts
func (p *OpenAIEmbeddingProvider) GetEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return [][]float64{}, nil
	}

	// Check cache for all texts
	var uncachedTexts []string
	var uncachedIndices []int
	embeddings := make([][]float64, len(texts))

	if p.cache != nil && p.config.CacheEnabled {
		for i, text := range texts {
			cacheKey := p.generateCacheKey(text)
			if embedding, found := p.cache.Get(ctx, cacheKey); found {
				embeddings[i] = embedding
			} else {
				uncachedTexts = append(uncachedTexts, text)
				uncachedIndices = append(uncachedIndices, i)
			}
		}
	} else {
		uncachedTexts = texts
		for i := range texts {
			uncachedIndices = append(uncachedIndices, i)
		}
	}

	// If all texts were cached, return cached embeddings
	if len(uncachedTexts) == 0 {
		return embeddings, nil
	}

	// Process uncached texts in batches
	batchSize := p.config.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	for i := 0; i < len(uncachedTexts); i += batchSize {
		end := i + batchSize
		if end > len(uncachedTexts) {
			end = len(uncachedTexts)
		}

		batchTexts := uncachedTexts[i:end]
		batchIndices := uncachedIndices[i:end]

		// Get embeddings for batch
		batchEmbeddings, err := p.fetchEmbeddings(ctx, batchTexts)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch embeddings for batch: %w", err)
		}

		// Store in cache and result array
		for j, embedding := range batchEmbeddings {
			originalIndex := batchIndices[j]
			embeddings[originalIndex] = embedding

			// Cache the embedding
			if p.cache != nil && p.config.CacheEnabled {
				cacheKey := p.generateCacheKey(batchTexts[j])
				if err := p.cache.Set(ctx, cacheKey, embedding, p.config.CacheTTL); err != nil {
					// Log error but don't fail the request
					fmt.Printf("Warning: failed to cache embedding: %v\n", err)
				}
			}
		}
	}

	return embeddings, nil
}

// fetchEmbeddings makes the actual API call to OpenAI
func (p *OpenAIEmbeddingProvider) fetchEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	// Prepare request body
	requestBody := map[string]interface{}{
		"model": p.config.ModelName,
		"input": texts,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", p.config.APIEndpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	// Make request with retries
	var resp *http.Response
	var requestErr error

	for attempt := 0; attempt <= p.config.RetryAttempts; attempt++ {
		resp, requestErr = p.httpClient.Do(req)
		if requestErr == nil && resp.StatusCode == http.StatusOK {
			break
		}

		if resp != nil {
			resp.Body.Close()
		}

		if attempt < p.config.RetryAttempts {
			// Wait before retrying
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt+1) * time.Second):
				// Continue to next attempt
			}
		}
	}

	if requestErr != nil {
		return nil, fmt.Errorf("failed to make request after %d attempts: %w", p.config.RetryAttempts+1, requestErr)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response
	var response OpenAIEmbeddingResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract embeddings
	embeddings := make([][]float64, len(response.Data))
	for i, data := range response.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}

// GetModelInfo returns information about the embedding model
func (p *OpenAIEmbeddingProvider) GetModelInfo() EmbeddingModelInfo {
	return p.modelInfo
}

// GetDimensions returns the dimensionality of the embedding vectors
func (p *OpenAIEmbeddingProvider) GetDimensions() int {
	return p.modelInfo.Dimensions
}

// GetMaxTokens returns the maximum number of tokens supported
func (p *OpenAIEmbeddingProvider) GetMaxTokens() int {
	return p.modelInfo.MaxTokens
}

// Close releases any resources held by the provider
func (p *OpenAIEmbeddingProvider) Close() error {
	// Close HTTP client if needed
	if p.httpClient != nil {
		p.httpClient.CloseIdleConnections()
	}
	return nil
}

// generateCacheKey generates a cache key for a text
func (p *OpenAIEmbeddingProvider) generateCacheKey(text string) string {
	// Create cache key from model name and text hash
	h := sha256.New()
	h.Write([]byte(p.config.ModelName))
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil))
}

// OpenAI API response structures
type OpenAIEmbeddingResponse struct {
	Data  []OpenAIEmbeddingData `json:"data"`
	Model string                `json:"model"`
	Usage OpenAIUsage           `json:"usage"`
}

type OpenAIEmbeddingData struct {
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
	Object    string    `json:"object"`
}

type OpenAIUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// getOpenAIModelInfo returns model information for OpenAI models
func getOpenAIModelInfo(modelName string) EmbeddingModelInfo {
	switch modelName {
	case "text-embedding-3-large":
		return EmbeddingModelInfo{
			Name:               "text-embedding-3-large",
			Provider:           "openai",
			Dimensions:         3072,
			MaxTokens:          8192,
			Version:            "3",
			Description:        "OpenAI's most capable embedding model",
			SupportedLanguages: []string{"en", "es", "fr", "de", "pt", "it", "ja", "ko", "zh", "ar", "hi", "th", "vi"},
		}
	case "text-embedding-3-small":
		return EmbeddingModelInfo{
			Name:               "text-embedding-3-small",
			Provider:           "openai",
			Dimensions:         1536,
			MaxTokens:          8192,
			Version:            "3",
			Description:        "OpenAI's efficient embedding model",
			SupportedLanguages: []string{"en", "es", "fr", "de", "pt", "it", "ja", "ko", "zh", "ar", "hi", "th", "vi"},
		}
	case "text-embedding-ada-002":
		return EmbeddingModelInfo{
			Name:               "text-embedding-ada-002",
			Provider:           "openai",
			Dimensions:         1536,
			MaxTokens:          8192,
			Version:            "2",
			Description:        "OpenAI's previous generation embedding model",
			SupportedLanguages: []string{"en", "es", "fr", "de", "pt", "it", "ja", "ko", "zh", "ar", "hi", "th", "vi"},
		}
	default:
		// Return default for unknown models
		return EmbeddingModelInfo{
			Name:               modelName,
			Provider:           "openai",
			Dimensions:         1536,
			MaxTokens:          8192,
			Version:            "unknown",
			Description:        "OpenAI embedding model",
			SupportedLanguages: []string{"en"},
		}
	}
}
