package llm

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/avast/retry-go"

	"github.com/memtensor/memgos/pkg/types"
)

// OllamaLLM implements the LLM interface for Ollama models
type OllamaLLM struct {
	*BaseLLM
	client  *resty.Client
	config  *LLMConfig
	baseURL string
}

// OllamaRequest represents a request to Ollama API
type OllamaRequest struct {
	Model       string                 `json:"model"`
	Messages    []map[string]interface{} `json:"messages,omitempty"`
	Prompt      string                 `json:"prompt,omitempty"`
	Stream      bool                   `json:"stream"`
	Options     map[string]interface{} `json:"options,omitempty"`
	KeepAlive   string                 `json:"keep_alive,omitempty"`
	Format      string                 `json:"format,omitempty"`
}

// OllamaResponse represents a response from Ollama API
type OllamaResponse struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	Context            []int  `json:"context,omitempty"`
	TotalDuration      int64  `json:"total_duration,omitempty"`
	LoadDuration       int64  `json:"load_duration,omitempty"`
	PromptEvalCount    int    `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64  `json:"prompt_eval_duration,omitempty"`
	EvalCount          int    `json:"eval_count,omitempty"`
	EvalDuration       int64  `json:"eval_duration,omitempty"`
}

// OllamaChatResponse represents a chat response from Ollama API
type OllamaChatResponse struct {
	Model      string                 `json:"model"`
	CreatedAt  string                 `json:"created_at"`
	Message    map[string]interface{} `json:"message"`
	Done       bool                   `json:"done"`
	Context    []int                  `json:"context,omitempty"`
	TotalDuration      int64  `json:"total_duration,omitempty"`
	LoadDuration       int64  `json:"load_duration,omitempty"`
	PromptEvalCount    int    `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64  `json:"prompt_eval_duration,omitempty"`
	EvalCount          int    `json:"eval_count,omitempty"`
	EvalDuration       int64  `json:"eval_duration,omitempty"`
}

// OllamaModelInfo represents model information
type OllamaModelInfo struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	Digest     string `json:"digest"`
	ModifiedAt string `json:"modified_at"`
}

// OllamaModelsResponse represents the models list response
type OllamaModelsResponse struct {
	Models []OllamaModelInfo `json:"models"`
}

// NewOllamaLLM creates a new Ollama LLM instance
func NewOllamaLLM(config *LLMConfig) (LLMProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	
	if config.Model == "" {
		return nil, fmt.Errorf("model name is required")
	}
	
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	
	// Create HTTP client
	client := resty.New()
	client.SetBaseURL(baseURL)
	client.SetTimeout(config.Timeout)
	client.SetRetryCount(3)
	client.SetRetryWaitTime(1 * time.Second)
	client.SetRetryMaxWaitTime(10 * time.Second)
	
	// Set headers
	client.SetHeader("Content-Type", "application/json")
	client.SetHeader("User-Agent", "MemGOS/1.0")
	
	llm := &OllamaLLM{
		BaseLLM: NewBaseLLM(config.Model),
		client:  client,
		config:  config,
		baseURL: baseURL,
	}
	
	// Apply configuration
	llm.SetMaxTokens(config.MaxTokens)
	llm.SetTemperature(config.Temperature)
	llm.SetTopP(config.TopP)
	llm.SetTimeout(config.Timeout)
	
	return llm, nil
}

// Generate generates text based on messages
func (o *OllamaLLM) Generate(ctx context.Context, messages types.MessageList) (string, error) {
	if err := o.ValidateMessages(messages); err != nil {
		return "", fmt.Errorf("invalid messages: %w", err)
	}
	
	// Build request
	req := OllamaRequest{
		Model:    o.GetModelName(),
		Messages: o.FormatMessages(messages),
		Stream:   false,
		Options: map[string]interface{}{
			"num_predict":   o.GetMaxTokens(),
			"temperature":   o.GetTemperature(),
			"top_p":         o.GetTopP(),
		},
		KeepAlive: "5m",
	}
	
	// Make request with retry
	var resp OllamaChatResponse
	var err error
	
	err = retry.Do(
		func() error {
			response, reqErr := o.client.R().
				SetContext(ctx).
				SetBody(req).
				SetResult(&resp).
				Post("/api/chat")
			
			if reqErr != nil {
				return reqErr
			}
			
			if response.StatusCode() != 200 {
				return fmt.Errorf("HTTP %d: %s", response.StatusCode(), response.String())
			}
			
			return nil
		},
		retry.Attempts(3),
		retry.Delay(time.Second),
		retry.DelayType(retry.BackOffDelay),
	)
	
	if err != nil {
		return "", fmt.Errorf("Ollama API request failed: %w", err)
	}
	
	// Extract content from response
	if message, ok := resp.Message["content"].(string); ok {
		// Record metrics
		o.RecordMetrics("eval_count", resp.EvalCount)
		o.RecordMetrics("eval_duration", resp.EvalDuration)
		o.RecordMetrics("prompt_eval_count", resp.PromptEvalCount)
		o.RecordMetrics("prompt_eval_duration", resp.PromptEvalDuration)
		o.RecordMetrics("total_duration", resp.TotalDuration)
		
		return message, nil
	}
	
	return "", fmt.Errorf("no content in response message")
}

// GenerateWithCache generates text with KV cache support
func (o *OllamaLLM) GenerateWithCache(ctx context.Context, messages types.MessageList, cache interface{}) (string, error) {
	if err := o.ValidateMessages(messages); err != nil {
		return "", fmt.Errorf("invalid messages: %w", err)
	}
	
	// Build request with context for caching
	req := OllamaRequest{
		Model:    o.GetModelName(),
		Messages: o.FormatMessages(messages),
		Stream:   false,
		Options: map[string]interface{}{
			"num_predict":   o.GetMaxTokens(),
			"temperature":   o.GetTemperature(),
			"top_p":         o.GetTopP(),
		},
		KeepAlive: "5m",
	}
	
	// Add context from cache if available
	if contextCache, ok := cache.([]int); ok {
		req.Options["context"] = contextCache
	}
	
	var resp OllamaChatResponse
	response, err := o.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&resp).
		Post("/api/chat")
	
	if err != nil {
		return "", fmt.Errorf("Ollama API request failed: %w", err)
	}
	
	if response.StatusCode() != 200 {
		return "", fmt.Errorf("HTTP %d: %s", response.StatusCode(), response.String())
	}
	
	// Extract content from response
	if message, ok := resp.Message["content"].(string); ok {
		// Record metrics and context
		o.RecordMetrics("eval_count", resp.EvalCount)
		o.RecordMetrics("context", resp.Context)
		
		return message, nil
	}
	
	return "", fmt.Errorf("no content in response message")
}

// GenerateStream generates text with streaming support
func (o *OllamaLLM) GenerateStream(ctx context.Context, messages types.MessageList, stream chan<- string) error {
	if err := o.ValidateMessages(messages); err != nil {
		return fmt.Errorf("invalid messages: %w", err)
	}
	
	// Build streaming request
	req := OllamaRequest{
		Model:    o.GetModelName(),
		Messages: o.FormatMessages(messages),
		Stream:   true,
		Options: map[string]interface{}{
			"num_predict":   o.GetMaxTokens(),
			"temperature":   o.GetTemperature(),
			"top_p":         o.GetTopP(),
		},
		KeepAlive: "5m",
	}
	
	// Make streaming request
	response, err := o.client.R().
		SetContext(ctx).
		SetBody(req).
		SetDoNotParseResponse(true).
		Post("/api/chat")
	
	if err != nil {
		return fmt.Errorf("failed to create streaming request: %w", err)
	}
	defer response.RawBody().Close()
	
	if response.StatusCode() != 200 {
		return fmt.Errorf("HTTP %d: %s", response.StatusCode(), response.String())
	}
	
	// Read streaming response
	scanner := bufio.NewScanner(response.RawBody())
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		
		var resp OllamaChatResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			continue // Skip invalid JSON
		}
		
		if message, ok := resp.Message["content"].(string); ok && message != "" {
			select {
			case stream <- message:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		
		if resp.Done {
			break
		}
	}
	
	return scanner.Err()
}

// Embed generates embeddings for text
func (o *OllamaLLM) Embed(ctx context.Context, text string) (types.EmbeddingVector, error) {
	if text == "" {
		return nil, fmt.Errorf("empty text")
	}
	
	// Build embedding request
	req := map[string]interface{}{
		"model":  o.GetModelName(),
		"prompt": text,
	}
	
	var resp map[string]interface{}
	response, err := o.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&resp).
		Post("/api/embeddings")
	
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}
	
	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", response.StatusCode(), response.String())
	}
	
	// Extract embedding from response
	if embeddings, ok := resp["embedding"].([]interface{}); ok {
		embedding := make(types.EmbeddingVector, len(embeddings))
		for i, v := range embeddings {
			if f, ok := v.(float64); ok {
				embedding[i] = float32(f)
			}
		}
		return embedding, nil
	}
	
	return nil, fmt.Errorf("no embedding in response")
}

// GetProviderName returns the provider name
func (o *OllamaLLM) GetProviderName() string {
	return "ollama"
}

// GetSupportedModels returns supported models
func (o *OllamaLLM) GetSupportedModels() []string {
	models, err := o.ListModels(context.Background())
	if err != nil {
		// Return common models if API call fails
		return []string{
			"llama2",
			"llama2:7b",
			"llama2:13b",
			"llama2:70b",
			"codellama",
			"codellama:7b",
			"codellama:13b",
			"codellama:34b",
			"mistral",
			"mistral:7b",
			"mixtral",
			"mixtral:8x7b",
			"phi",
			"phi:2.7b",
			"orca-mini",
			"vicuna",
			"neural-chat",
		}
	}
	return models
}

// HealthCheck performs health check
func (o *OllamaLLM) HealthCheck(ctx context.Context) error {
	// Check if Ollama is running
	response, err := o.client.R().
		SetContext(ctx).
		Get("/api/tags")
	
	if err != nil {
		return fmt.Errorf("Ollama health check failed: %w", err)
	}
	
	if response.StatusCode() != 200 {
		return fmt.Errorf("Ollama health check failed: HTTP %d", response.StatusCode())
	}
	
	return nil
}

// SetConfig sets the configuration
func (o *OllamaLLM) SetConfig(config *LLMConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	
	o.config = config
	
	// Update base configuration
	o.SetMaxTokens(config.MaxTokens)
	o.SetTemperature(config.Temperature)
	o.SetTopP(config.TopP)
	o.SetTimeout(config.Timeout)
	
	return nil
}

// GetConfig returns the configuration
func (o *OllamaLLM) GetConfig() *LLMConfig {
	return o.config
}

// Close closes the Ollama client
func (o *OllamaLLM) Close() error {
	// Nothing to close for HTTP client
	return o.BaseLLM.Close()
}

// ListModels lists available models
func (o *OllamaLLM) ListModels(ctx context.Context) ([]string, error) {
	var resp OllamaModelsResponse
	response, err := o.client.R().
		SetContext(ctx).
		SetResult(&resp).
		Get("/api/tags")
	
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	
	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", response.StatusCode(), response.String())
	}
	
	var modelNames []string
	for _, model := range resp.Models {
		modelNames = append(modelNames, model.Name)
	}
	
	return modelNames, nil
}

// PullModel pulls a model from Ollama registry
func (o *OllamaLLM) PullModel(ctx context.Context, modelName string) error {
	req := map[string]interface{}{
		"name": modelName,
	}
	
	response, err := o.client.R().
		SetContext(ctx).
		SetBody(req).
		SetDoNotParseResponse(true).
		Post("/api/pull")
	
	if err != nil {
		return fmt.Errorf("failed to pull model: %w", err)
	}
	defer response.RawBody().Close()
	
	if response.StatusCode() != 200 {
		return fmt.Errorf("HTTP %d: %s", response.StatusCode(), response.String())
	}
	
	// Read streaming response to track progress
	scanner := bufio.NewScanner(response.RawBody())
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		
		var status map[string]interface{}
		if err := json.Unmarshal([]byte(line), &status); err != nil {
			continue
		}
		
		// Check if pulling is complete
		if completed, ok := status["completed"].(bool); ok && completed {
			break
		}
	}
	
	return scanner.Err()
}

// DeleteModel deletes a model from local storage
func (o *OllamaLLM) DeleteModel(ctx context.Context, modelName string) error {
	req := map[string]interface{}{
		"name": modelName,
	}
	
	response, err := o.client.R().
		SetContext(ctx).
		SetBody(req).
		Delete("/api/delete")
	
	if err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}
	
	if response.StatusCode() != 200 {
		return fmt.Errorf("HTTP %d: %s", response.StatusCode(), response.String())
	}
	
	return nil
}

// GetModelInfo returns detailed model information
func (o *OllamaLLM) GetModelInfo() map[string]interface{} {
	info := o.BaseLLM.GetModelInfo()
	info["provider"] = o.GetProviderName()
	info["base_url"] = o.baseURL
	info["keep_alive"] = "5m"
	
	// Add model-specific info
	models, err := o.ListModels(context.Background())
	if err == nil {
		info["available_models"] = models
	}
	
	return info
}

// CheckModelExists checks if a model exists locally
func (o *OllamaLLM) CheckModelExists(ctx context.Context, modelName string) (bool, error) {
	models, err := o.ListModels(ctx)
	if err != nil {
		return false, err
	}
	
	for _, model := range models {
		if model == modelName {
			return true, nil
		}
	}
	
	return false, nil
}

// GenerateCompletion generates text completion (non-chat format)
func (o *OllamaLLM) GenerateCompletion(ctx context.Context, prompt string) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("empty prompt")
	}
	
	req := OllamaRequest{
		Model:  o.GetModelName(),
		Prompt: prompt,
		Stream: false,
		Options: map[string]interface{}{
			"num_predict": o.GetMaxTokens(),
			"temperature": o.GetTemperature(),
			"top_p":       o.GetTopP(),
		},
		KeepAlive: "5m",
	}
	
	var resp OllamaResponse
	response, err := o.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&resp).
		Post("/api/generate")
	
	if err != nil {
		return "", fmt.Errorf("completion request failed: %w", err)
	}
	
	if response.StatusCode() != 200 {
		return "", fmt.Errorf("HTTP %d: %s", response.StatusCode(), response.String())
	}
	
	// Record metrics
	o.RecordMetrics("eval_count", resp.EvalCount)
	o.RecordMetrics("eval_duration", resp.EvalDuration)
	o.RecordMetrics("prompt_eval_count", resp.PromptEvalCount)
	o.RecordMetrics("prompt_eval_duration", resp.PromptEvalDuration)
	o.RecordMetrics("total_duration", resp.TotalDuration)
	
	return resp.Response, nil
}

// GetModelStats returns model statistics
func (o *OllamaLLM) GetModelStats() map[string]interface{} {
	metrics := o.GetMetrics()
	
	stats := map[string]interface{}{
		"eval_count":            metrics["eval_count"],
		"eval_duration":         metrics["eval_duration"],
		"prompt_eval_count":     metrics["prompt_eval_count"],
		"prompt_eval_duration":  metrics["prompt_eval_duration"],
		"total_duration":        metrics["total_duration"],
	}
	
	// Calculate derived metrics
	if evalCount, ok := metrics["eval_count"].(int); ok && evalCount > 0 {
		if evalDuration, ok := metrics["eval_duration"].(int64); ok {
			stats["tokens_per_second"] = float64(evalCount) / (float64(evalDuration) / 1e9)
		}
	}
	
	return stats
}

// SetKeepAlive sets the keep-alive duration for model loading
func (o *OllamaLLM) SetKeepAlive(duration string) {
	// This would be used in subsequent requests
	o.RecordMetrics("keep_alive", duration)
}

// GetVersion returns Ollama version information
func (o *OllamaLLM) GetVersion(ctx context.Context) (map[string]interface{}, error) {
	var resp map[string]interface{}
	response, err := o.client.R().
		SetContext(ctx).
		SetResult(&resp).
		Get("/api/version")
	
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}
	
	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", response.StatusCode(), response.String())
	}
	
	return resp, nil
}

// IsModelSupported checks if a model is supported
func (o *OllamaLLM) IsModelSupported(model string) bool {
	supportedModels := o.GetSupportedModels()
	for _, supported := range supportedModels {
		if strings.EqualFold(supported, model) {
			return true
		}
	}
	return false
}

// GetDefaultModel returns the default model for Ollama
func (o *OllamaLLM) GetDefaultModel() string {
	return "llama2"
}

// CreateModelfile creates a custom model from a Modelfile
func (o *OllamaLLM) CreateModelfile(ctx context.Context, name, modelfile string) error {
	req := map[string]interface{}{
		"name":      name,
		"modelfile": modelfile,
	}
	
	response, err := o.client.R().
		SetContext(ctx).
		SetBody(req).
		SetDoNotParseResponse(true).
		Post("/api/create")
	
	if err != nil {
		return fmt.Errorf("failed to create model: %w", err)
	}
	defer response.RawBody().Close()
	
	if response.StatusCode() != 200 {
		return fmt.Errorf("HTTP %d: %s", response.StatusCode(), response.String())
	}
	
	// Read streaming response
	scanner := bufio.NewScanner(response.RawBody())
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		
		var status map[string]interface{}
		if err := json.Unmarshal([]byte(line), &status); err != nil {
			continue
		}
		
		// Check if creation is complete
		if success, ok := status["success"].(bool); ok && success {
			break
		}
	}
	
	return scanner.Err()
}