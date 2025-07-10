package embedders

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/avast/retry-go"
	"github.com/go-resty/resty/v2"
	"github.com/memtensor/memgos/pkg/types"
)

// OllamaEmbedder implements Ollama embeddings
type OllamaEmbedder struct {
	*BaseEmbedder
	config       *EmbedderConfig
	client       *resty.Client
	baseURL      string
	mu           sync.RWMutex
}

// OllamaEmbeddingRequest represents an Ollama embedding request
type OllamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream,omitempty"`
}

// OllamaEmbeddingResponse represents an Ollama embedding response
type OllamaEmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
	Model     string    `json:"model"`
}

// OllamaModelsResponse represents the response from /api/tags
type OllamaModelsResponse struct {
	Models []OllamaModel `json:"models"`
}

// OllamaModel represents a model in Ollama
type OllamaModel struct {
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	Digest   string    `json:"digest"`
	Modified time.Time `json:"modified_at"`
	Details  OllamaModelDetails `json:"details,omitempty"`
}

// OllamaModelDetails represents model details
type OllamaModelDetails struct {
	Format     string   `json:"format"`
	Family     string   `json:"family"`
	Families   []string `json:"families"`
	Parameters string   `json:"parameter_size"`
	Quantization string `json:"quantization_level"`
}

// OllamaStreamResponse represents a streaming response chunk
type OllamaStreamResponse struct {
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	Response  string    `json:"response,omitempty"`
	Done      bool      `json:"done"`
	Embedding []float64 `json:"embedding,omitempty"`
}

// NewOllamaEmbedder creates a new Ollama embedder
func NewOllamaEmbedder(config *EmbedderConfig) (EmbedderProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	
	// Set defaults
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	
	if config.MaxLength == 0 {
		config.MaxLength = 2048 // Default for most Ollama models
	}
	
	if config.BatchSize == 0 {
		config.BatchSize = 10 // Smaller batch size for local models
	}
	
	if config.Dimension == 0 {
		// Auto-detect dimension from model (will be set after first embedding)
		config.Dimension = 768 // Default assumption
	}
	
	// Create HTTP client with timeout
	client := resty.New()
	client.SetTimeout(30 * time.Second)
	if config.Timeout > 0 {
		client.SetTimeout(config.Timeout)
	}
	
	ollama := &OllamaEmbedder{
		BaseEmbedder: NewBaseEmbedder(config.Model, config.Dimension),
		config:       config,
		client:       client,
		baseURL:      strings.TrimSuffix(baseURL, "/"),
	}
	
	ollama.SetMaxLength(config.MaxLength)
	if config.Timeout > 0 {
		ollama.SetTimeout(config.Timeout)
	}
	
	return ollama, nil
}

// Embed generates embeddings for a single text
func (ol *OllamaEmbedder) Embed(ctx context.Context, text string) (types.EmbeddingVector, error) {
	start := time.Now()
	defer func() {
		ol.AddToTimer("embed_duration", time.Since(start))
		ol.IncrementCounter("embed_calls")
	}()
	
	if text == "" {
		return nil, fmt.Errorf("empty text")
	}
	
	// Preprocess text
	processedText := ol.PreprocessText(text)
	
	// Create embedding with retry
	embedding, err := ol.createEmbeddingWithRetry(ctx, processedText)
	if err != nil {
		ol.IncrementCounter("embed_errors")
		return nil, fmt.Errorf("Ollama API error: %w", err)
	}
	
	// Convert float64 to float32
	result := make(types.EmbeddingVector, len(embedding))
	for i, val := range embedding {
		result[i] = float32(val)
	}
	
	// Auto-detect dimension on first call
	if ol.GetDimension() != len(result) {
		ol.mu.Lock()
		ol.dimension = len(result)
		ol.config.Dimension = len(result)
		ol.mu.Unlock()
		ol.RecordMetrics("auto_detected_dimension", len(result))
	}
	
	// Normalize if configured
	if ol.config.Normalize {
		result = ol.NormalizeVector(result)
	}
	
	// Validate result
	if err := ol.ValidateVector(result); err != nil {
		return nil, fmt.Errorf("invalid embedding: %w", err)
	}
	
	ol.RecordMetrics("last_embedding_dimension", len(result))
	ol.RecordMetrics("last_text_length", len(text))
	
	return result, nil
}

// EmbedBatch generates embeddings for multiple texts
func (ol *OllamaEmbedder) EmbedBatch(ctx context.Context, texts []string) ([]types.EmbeddingVector, error) {
	start := time.Now()
	defer func() {
		ol.AddToTimer("embed_batch_duration", time.Since(start))
		ol.IncrementCounter("embed_batch_calls")
		ol.RecordMetrics("last_batch_size", len(texts))
	}()
	
	if len(texts) == 0 {
		return []types.EmbeddingVector{}, nil
	}
	
	// Ollama doesn't support batch embedding, so process individually
	var allEmbeddings []types.EmbeddingVector
	
	for i, text := range texts {
		embedding, err := ol.Embed(ctx, text)
		if err != nil {
			ol.IncrementCounter("embed_batch_errors")
			return nil, fmt.Errorf("batch processing failed at index %d: %w", i, err)
		}
		
		allEmbeddings = append(allEmbeddings, embedding)
		
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
func (ol *OllamaEmbedder) createEmbeddingWithRetry(ctx context.Context, text string) ([]float64, error) {
	var result []float64
	
	err := retry.Do(
		func() error {
			embedding, err := ol.createEmbedding(ctx, text)
			if err != nil {
				return err
			}
			result = embedding
			return nil
		},
		retry.Attempts(3),
		retry.Delay(time.Second),
		retry.DelayType(retry.BackOffDelay),
		retry.OnRetry(func(n uint, err error) {
			ol.IncrementCounter("api_retries")
		}),
		retry.Context(ctx),
	)
	
	return result, err
}

// createEmbedding creates a single embedding via Ollama API
func (ol *OllamaEmbedder) createEmbedding(ctx context.Context, text string) ([]float64, error) {
	req := OllamaEmbeddingRequest{
		Model:  ol.config.Model,
		Prompt: text,
		Stream: false,
	}
	
	url := fmt.Sprintf("%s/api/embeddings", ol.baseURL)
	
	resp, err := ol.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&OllamaEmbeddingResponse{}).
		Post(url)
	
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode(), resp.String())
	}
	
	result, ok := resp.Result().(*OllamaEmbeddingResponse)
	if !ok || result == nil {
		return nil, fmt.Errorf("invalid response format")
	}
	
	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	
	ol.IncrementCounter("api_calls")
	
	return result.Embedding, nil
}

// EmbedStream creates embeddings using streaming API
func (ol *OllamaEmbedder) EmbedStream(ctx context.Context, text string, resultChan chan<- types.EmbeddingVector) error {
	req := OllamaEmbeddingRequest{
		Model:  ol.config.Model,
		Prompt: text,
		Stream: true,
	}
	
	url := fmt.Sprintf("%s/api/embeddings", ol.baseURL)
	
	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	
	httpClient := &http.Client{Timeout: ol.GetTimeout()}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	decoder := json.NewDecoder(resp.Body)
	
	for {
		var streamResp OllamaStreamResponse
		if err := decoder.Decode(&streamResp); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode streaming response: %w", err)
		}
		
		if len(streamResp.Embedding) > 0 {
			// Convert to EmbeddingVector
			embedding := make(types.EmbeddingVector, len(streamResp.Embedding))
			for i, val := range streamResp.Embedding {
				embedding[i] = float32(val)
			}
			
			// Normalize if configured
			if ol.config.Normalize {
				embedding = ol.NormalizeVector(embedding)
			}
			
			select {
			case resultChan <- embedding:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		
		if streamResp.Done {
			break
		}
	}
	
	return nil
}

// GetProviderName returns the provider name
func (ol *OllamaEmbedder) GetProviderName() string {
	return "ollama"
}

// GetSupportedModels returns a list of supported models
func (ol *OllamaEmbedder) GetSupportedModels() []string {
	return []string{
		"nomic-embed-text",
		"mxbai-embed-large", 
		"all-minilm",
		"snowflake-arctic-embed",
		"bge-large",
		"bge-base",
		"sentence-transformers",
	}
}

// ListAvailableModels queries Ollama for available models
func (ol *OllamaEmbedder) ListAvailableModels(ctx context.Context) ([]OllamaModel, error) {
	url := fmt.Sprintf("%s/api/tags", ol.baseURL)
	
	resp, err := ol.client.R().
		SetContext(ctx).
		SetResult(&OllamaModelsResponse{}).
		Get(url)
	
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode(), resp.String())
	}
	
	result, ok := resp.Result().(*OllamaModelsResponse)
	if !ok || result == nil {
		return nil, fmt.Errorf("invalid response format")
	}
	
	return result.Models, nil
}

// HealthCheck performs a health check
func (ol *OllamaEmbedder) HealthCheck(ctx context.Context) error {
	// First check if Ollama is running
	url := fmt.Sprintf("%s/api/tags", ol.baseURL)
	
	resp, err := ol.client.R().
		SetContext(ctx).
		Get(url)
	
	if err != nil {
		return fmt.Errorf("Ollama server not reachable: %w", err)
	}
	
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("Ollama server returned status %d", resp.StatusCode())
	}
	
	// Test with a simple embedding
	testText := "Hello world"
	_, err = ol.Embed(ctx, testText)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	
	return nil
}

// SetConfig updates the embedder configuration
func (ol *OllamaEmbedder) SetConfig(config *EmbedderConfig) error {
	ol.mu.Lock()
	defer ol.mu.Unlock()
	
	if err := config.Validate(); err != nil {
		return err
	}
	
	// Update base URL if changed
	if config.BaseURL != "" && config.BaseURL != ol.config.BaseURL {
		ol.baseURL = strings.TrimSuffix(config.BaseURL, "/")
	}
	
	ol.config = config
	ol.SetMaxLength(config.MaxLength)
	if config.Timeout > 0 {
		ol.SetTimeout(config.Timeout)
		ol.client.SetTimeout(config.Timeout)
	}
	
	return nil
}

// GetConfig returns the current configuration
func (ol *OllamaEmbedder) GetConfig() *EmbedderConfig {
	ol.mu.RLock()
	defer ol.mu.RUnlock()
	
	// Return a copy to prevent external modification
	configCopy := *ol.config
	return &configCopy
}

// Close closes the embedder and releases resources
func (ol *OllamaEmbedder) Close() error {
	// Resty client doesn't need explicit cleanup
	return ol.BaseEmbedder.Close()
}

// GetModelInfo returns detailed model information
func (ol *OllamaEmbedder) GetModelInfo() map[string]interface{} {
	ol.mu.RLock()
	defer ol.mu.RUnlock()
	
	return map[string]interface{}{
		"provider":    ol.GetProviderName(),
		"model":       ol.config.Model,
		"dimension":   ol.GetDimension(),
		"max_length":  ol.config.MaxLength,
		"batch_size":  ol.config.BatchSize,
		"normalize":   ol.config.Normalize,
		"base_url":    ol.baseURL,
		"metrics":     ol.GetMetrics(),
	}
}

// PullModel pulls a model from Ollama registry
func (ol *OllamaEmbedder) PullModel(ctx context.Context, modelName string) error {
	url := fmt.Sprintf("%s/api/pull", ol.baseURL)
	
	reqBody := map[string]interface{}{
		"name": modelName,
	}
	
	resp, err := ol.client.R().
		SetContext(ctx).
		SetBody(reqBody).
		Post(url)
	
	if err != nil {
		return fmt.Errorf("failed to pull model: %w", err)
	}
	
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("pull request failed with status %d: %s", resp.StatusCode(), resp.String())
	}
	
	ol.RecordMetrics("models_pulled", modelName)
	
	return nil
}

// DeleteModel deletes a model from Ollama
func (ol *OllamaEmbedder) DeleteModel(ctx context.Context, modelName string) error {
	url := fmt.Sprintf("%s/api/delete", ol.baseURL)
	
	reqBody := map[string]interface{}{
		"name": modelName,
	}
	
	resp, err := ol.client.R().
		SetContext(ctx).
		SetBody(reqBody).
		Delete(url)
	
	if err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}
	
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("delete request failed with status %d: %s", resp.StatusCode(), resp.String())
	}
	
	ol.RecordMetrics("models_deleted", modelName)
	
	return nil
}

// GetServerInfo returns information about the Ollama server
func (ol *OllamaEmbedder) GetServerInfo(ctx context.Context) (map[string]interface{}, error) {
	// Try to get version info
	url := fmt.Sprintf("%s/api/version", ol.baseURL)
	
	resp, err := ol.client.R().
		SetContext(ctx).
		Get(url)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get server info: %w", err)
	}
	
	info := map[string]interface{}{
		"base_url":    ol.baseURL,
		"status_code": resp.StatusCode(),
		"reachable":   resp.StatusCode() == http.StatusOK,
	}
	
	if resp.StatusCode() == http.StatusOK {
		var versionInfo map[string]interface{}
		if err := json.Unmarshal(resp.Body(), &versionInfo); err == nil {
			info["version"] = versionInfo
		}
	}
	
	// Get available models
	if models, err := ol.ListAvailableModels(ctx); err == nil {
		info["available_models"] = models
		info["model_count"] = len(models)
	}
	
	return info, nil
}

// WarmUp performs model warm-up
func (ol *OllamaEmbedder) WarmUp(ctx context.Context) error {
	warmupText := "This is a test sentence for model warm-up."
	
	start := time.Now()
	_, err := ol.Embed(ctx, warmupText)
	if err != nil {
		return fmt.Errorf("warm-up failed: %w", err)
	}
	
	ol.RecordMetrics("warmup_duration", time.Since(start))
	ol.RecordMetrics("warmup_completed", true)
	
	return nil
}

// EnsureModelExists checks if model exists and pulls it if necessary
func (ol *OllamaEmbedder) EnsureModelExists(ctx context.Context) error {
	models, err := ol.ListAvailableModels(ctx)
	if err != nil {
		return fmt.Errorf("failed to check available models: %w", err)
	}
	
	// Check if our model exists
	for _, model := range models {
		if model.Name == ol.config.Model || 
		   strings.HasPrefix(model.Name, ol.config.Model+":") {
			ol.RecordMetrics("model_exists", true)
			return nil
		}
	}
	
	// Model doesn't exist, try to pull it
	ol.RecordMetrics("model_exists", false)
	
	fmt.Printf("Model %s not found, attempting to pull...\n", ol.config.Model)
	if err := ol.PullModel(ctx, ol.config.Model); err != nil {
		return fmt.Errorf("failed to pull model %s: %w", ol.config.Model, err)
	}
	
	fmt.Printf("Successfully pulled model %s\n", ol.config.Model)
	return nil
}

// GetUsageStatistics returns usage statistics
func (ol *OllamaEmbedder) GetUsageStatistics() map[string]interface{} {
	metrics := ol.GetMetrics()
	
	stats := map[string]interface{}{
		"total_embed_calls":       metrics["embed_calls"],
		"total_batch_calls":       metrics["embed_batch_calls"],
		"total_api_calls":         metrics["api_calls"],
		"total_api_retries":       metrics["api_retries"],
		"total_errors":            metrics["embed_errors"],
		"total_batch_errors":      metrics["embed_batch_errors"],
		"average_embed_duration":  nil,
		"average_batch_duration":  nil,
		"models_pulled":           metrics["models_pulled"],
		"models_deleted":          metrics["models_deleted"],
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