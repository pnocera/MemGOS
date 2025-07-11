package llm

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/avast/retry-go"
	"github.com/cenkalti/backoff/v4"
	"github.com/go-resty/resty/v2"
	"github.com/sashabaranov/go-openai"

	"github.com/memtensor/memgos/pkg/types"
)

// OpenAILLM implements the LLM interface for OpenAI models
type OpenAILLM struct {
	*BaseLLM
	client     *openai.Client
	config     *LLMConfig
	httpClient *resty.Client
}

// NewOpenAILLM creates a new OpenAI LLM instance
func NewOpenAILLM(config *LLMConfig) (LLMProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	// Create OpenAI client
	openaiConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		openaiConfig.BaseURL = config.BaseURL
	}

	client := openai.NewClientWithConfig(openaiConfig)

	// Create HTTP client for custom requests
	httpClient := resty.New()
	httpClient.SetTimeout(config.Timeout)
	httpClient.SetRetryCount(3)
	httpClient.SetRetryWaitTime(1 * time.Second)
	httpClient.SetRetryMaxWaitTime(5 * time.Second)

	// Set headers
	httpClient.SetHeader("Authorization", "Bearer "+config.APIKey)
	httpClient.SetHeader("Content-Type", "application/json")
	httpClient.SetHeader("User-Agent", "MemGOS/1.0")

	llm := &OpenAILLM{
		BaseLLM:    NewBaseLLM(config.Model),
		client:     client,
		config:     config,
		httpClient: httpClient,
	}

	// Apply configuration
	llm.SetMaxTokens(config.MaxTokens)
	llm.SetTemperature(config.Temperature)
	llm.SetTopP(config.TopP)
	llm.SetTimeout(config.Timeout)

	return llm, nil
}

// Generate generates text based on messages
func (o *OpenAILLM) Generate(ctx context.Context, messages types.MessageList) (string, error) {
	if err := o.ValidateMessages(messages); err != nil {
		return "", fmt.Errorf("invalid messages: %w", err)
	}

	// Convert messages to OpenAI format
	openaiMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		openaiMessages[i] = openai.ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	// Create request
	req := openai.ChatCompletionRequest{
		Model:       o.GetModelName(),
		Messages:    openaiMessages,
		MaxTokens:   o.GetMaxTokens(),
		Temperature: float32(o.GetTemperature()),
		TopP:        float32(o.GetTopP()),
		Stream:      false,
	}

	// Make request with retry
	var resp openai.ChatCompletionResponse
	var err error

	err = retry.Do(
		func() error {
			resp, err = o.client.CreateChatCompletion(ctx, req)
			return err
		},
		retry.Attempts(3),
		retry.Delay(time.Second),
		retry.DelayType(retry.BackOffDelay),
	)

	if err != nil {
		return "", fmt.Errorf("OpenAI API request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	// Record metrics
	o.RecordMetrics("tokens_used", resp.Usage.TotalTokens)
	o.RecordMetrics("prompt_tokens", resp.Usage.PromptTokens)
	o.RecordMetrics("completion_tokens", resp.Usage.CompletionTokens)
	o.RecordMetrics("model", resp.Model)

	return resp.Choices[0].Message.Content, nil
}

// GenerateWithCache generates text with KV cache support
func (o *OpenAILLM) GenerateWithCache(ctx context.Context, messages types.MessageList, cache interface{}) (string, error) {
	// OpenAI doesn't expose KV cache directly, so we fall back to regular generation
	// In a real implementation, you might cache responses locally
	return o.Generate(ctx, messages)
}

// GenerateStream generates text with streaming support
func (o *OpenAILLM) GenerateStream(ctx context.Context, messages types.MessageList, stream chan<- string) error {
	if err := o.ValidateMessages(messages); err != nil {
		return fmt.Errorf("invalid messages: %w", err)
	}

	// Convert messages to OpenAI format
	openaiMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		openaiMessages[i] = openai.ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	// Create streaming request
	req := openai.ChatCompletionRequest{
		Model:       o.GetModelName(),
		Messages:    openaiMessages,
		MaxTokens:   o.GetMaxTokens(),
		Temperature: float32(o.GetTemperature()),
		TopP:        float32(o.GetTopP()),
		Stream:      true,
	}

	// Create streaming response
	streamResp, err := o.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create streaming response: %w", err)
	}
	defer streamResp.Close()

	// Read stream
	for {
		response, err := streamResp.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("stream error: %w", err)
		}

		if len(response.Choices) > 0 {
			content := response.Choices[0].Delta.Content
			if content != "" {
				select {
				case stream <- content:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}

	return nil
}

// Embed generates embeddings for text
func (o *OpenAILLM) Embed(ctx context.Context, text string) (types.EmbeddingVector, error) {
	if text == "" {
		return nil, fmt.Errorf("empty text")
	}

	// Use text-embedding-ada-002 model for embeddings
	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.AdaEmbeddingV2,
	}

	resp, err := o.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	// Convert to our format
	embedding := make(types.EmbeddingVector, len(resp.Data[0].Embedding))
	for i, v := range resp.Data[0].Embedding {
		embedding[i] = float32(v)
	}

	// Record metrics
	o.RecordMetrics("embedding_tokens", resp.Usage.TotalTokens)

	return embedding, nil
}

// GetProviderName returns the provider name
func (o *OpenAILLM) GetProviderName() string {
	return "openai"
}

// GetSupportedModels returns supported models
func (o *OpenAILLM) GetSupportedModels() []string {
	return []string{
		"gpt-4",
		"gpt-4-32k",
		"gpt-4-1106-preview",
		"gpt-4-turbo-preview",
		"gpt-3.5-turbo",
		"gpt-3.5-turbo-16k",
		"gpt-3.5-turbo-1106",
		"text-davinci-003",
		"text-davinci-002",
		"text-curie-001",
		"text-babbage-001",
		"text-ada-001",
	}
}

// HealthCheck performs health check
func (o *OpenAILLM) HealthCheck(ctx context.Context) error {
	// Try to list models as a health check
	_, err := o.client.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("OpenAI health check failed: %w", err)
	}
	return nil
}

// SetConfig sets the configuration
func (o *OpenAILLM) SetConfig(config *LLMConfig) error {
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
func (o *OpenAILLM) GetConfig() *LLMConfig {
	return o.config
}

// Close closes the OpenAI client
func (o *OpenAILLM) Close() error {
	// Nothing to close for OpenAI client
	return o.BaseLLM.Close()
}

// ListModels lists available models
func (o *OpenAILLM) ListModels(ctx context.Context) ([]string, error) {
	models, err := o.client.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	var modelNames []string
	for _, model := range models.Models {
		modelNames = append(modelNames, model.ID)
	}

	return modelNames, nil
}

// GetModelInfo returns detailed model information
func (o *OpenAILLM) GetModelInfo() map[string]interface{} {
	info := o.BaseLLM.GetModelInfo()
	info["provider"] = o.GetProviderName()
	info["supported_models"] = o.GetSupportedModels()
	info["api_key_set"] = o.config.APIKey != ""
	info["base_url"] = o.config.BaseURL

	return info
}

// EstimateTokens estimates token count for messages
func (o *OpenAILLM) EstimateTokens(messages types.MessageList) int {
	totalTokens := 0
	for _, msg := range messages {
		// Rough estimate: 4 chars per token + role overhead
		totalTokens += len(msg.Content)/4 + 10
	}
	return totalTokens
}

// ValidateModel validates if model is supported
func (o *OpenAILLM) ValidateModel(model string) error {
	supportedModels := o.GetSupportedModels()
	for _, supported := range supportedModels {
		if supported == model {
			return nil
		}
	}
	return fmt.Errorf("unsupported model: %s", model)
}

// CreateChatCompletion creates a chat completion with custom options
func (o *OpenAILLM) CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	// Apply default settings if not specified
	if req.Model == "" {
		req.Model = o.GetModelName()
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = o.GetMaxTokens()
	}
	if req.Temperature == 0 {
		req.Temperature = float32(o.GetTemperature())
	}
	if req.TopP == 0 {
		req.TopP = float32(o.GetTopP())
	}

	return o.client.CreateChatCompletion(ctx, req)
}

// CreateEmbedding creates embeddings with custom options
func (o *OpenAILLM) CreateEmbedding(ctx context.Context, req openai.EmbeddingRequest) (openai.EmbeddingResponse, error) {
	// Set default model if not specified
	if req.Model == openai.EmbeddingModel("") {
		req.Model = openai.AdaEmbeddingV2
	}

	return o.client.CreateEmbeddings(ctx, req)
}

// GetTokenUsage returns token usage statistics
func (o *OpenAILLM) GetTokenUsage() map[string]interface{} {
	metrics := o.GetMetrics()

	return map[string]interface{}{
		"total_tokens":      metrics["tokens_used"],
		"prompt_tokens":     metrics["prompt_tokens"],
		"completion_tokens": metrics["completion_tokens"],
		"embedding_tokens":  metrics["embedding_tokens"],
	}
}

// SetCustomHeaders sets custom HTTP headers
func (o *OpenAILLM) SetCustomHeaders(headers map[string]string) {
	for key, value := range headers {
		o.httpClient.SetHeader(key, value)
	}
}

// WithRetry configures retry settings
func (o *OpenAILLM) WithRetry(maxRetries int, backoffStrategy backoff.BackOff) *OpenAILLM {
	o.httpClient.SetRetryCount(maxRetries)
	return o
}

// WithTimeout sets request timeout
func (o *OpenAILLM) WithTimeout(timeout time.Duration) *OpenAILLM {
	o.httpClient.SetTimeout(timeout)
	o.SetTimeout(timeout)
	return o
}

// IsModelSupported checks if a model is supported
func (o *OpenAILLM) IsModelSupported(model string) bool {
	supportedModels := o.GetSupportedModels()
	for _, supported := range supportedModels {
		if strings.EqualFold(supported, model) {
			return true
		}
	}
	return false
}

// GetDefaultModel returns the default model for OpenAI
func (o *OpenAILLM) GetDefaultModel() string {
	return "gpt-3.5-turbo"
}

// GetModelCapabilities returns model capabilities
func (o *OpenAILLM) GetModelCapabilities(model string) map[string]interface{} {
	capabilities := map[string]interface{}{
		"chat":          true,
		"embeddings":    false,
		"streaming":     true,
		"function_call": false,
	}

	// Model-specific capabilities
	switch model {
	case "gpt-4", "gpt-4-32k", "gpt-4-1106-preview", "gpt-4-turbo-preview":
		capabilities["function_call"] = true
		capabilities["max_tokens"] = 8192
	case "gpt-3.5-turbo", "gpt-3.5-turbo-16k", "gpt-3.5-turbo-1106":
		capabilities["function_call"] = true
		capabilities["max_tokens"] = 4096
	case "text-davinci-003", "text-davinci-002":
		capabilities["chat"] = false
		capabilities["completion"] = true
		capabilities["max_tokens"] = 4097
	}

	return capabilities
}

// CreateFunctionCall creates a function call completion
func (o *OpenAILLM) CreateFunctionCall(ctx context.Context, messages types.MessageList, functions []openai.FunctionDefinition) (openai.ChatCompletionResponse, error) {
	if err := o.ValidateMessages(messages); err != nil {
		return openai.ChatCompletionResponse{}, fmt.Errorf("invalid messages: %w", err)
	}

	// Convert messages to OpenAI format
	openaiMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		openaiMessages[i] = openai.ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	req := openai.ChatCompletionRequest{
		Model:       o.GetModelName(),
		Messages:    openaiMessages,
		Functions:   functions,
		MaxTokens:   o.GetMaxTokens(),
		Temperature: float32(o.GetTemperature()),
		TopP:        float32(o.GetTopP()),
	}

	return o.client.CreateChatCompletion(ctx, req)
}
