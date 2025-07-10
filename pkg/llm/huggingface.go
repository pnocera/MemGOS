package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/avast/retry-go"

	"github.com/memtensor/memgos/pkg/types"
)

// HuggingFaceLLM implements the LLM interface for HuggingFace models
type HuggingFaceLLM struct {
	*BaseLLM
	client    *resty.Client
	config    *LLMConfig
	baseURL   string
	useLocal  bool
	modelPath string
}

// HuggingFaceRequest represents a request to HuggingFace API
type HuggingFaceRequest struct {
	Inputs     interface{}            `json:"inputs"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Options    map[string]interface{} `json:"options,omitempty"`
}

// HuggingFaceResponse represents a response from HuggingFace API
type HuggingFaceResponse struct {
	GeneratedText string                 `json:"generated_text,omitempty"`
	Text          string                 `json:"text,omitempty"`
	Score         float64                `json:"score,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Warnings      []string               `json:"warnings,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// HuggingFaceChatRequest represents a chat request
type HuggingFaceChatRequest struct {
	Model    string                   `json:"model"`
	Messages []map[string]interface{} `json:"messages"`
	Stream   bool                     `json:"stream"`
	Options  map[string]interface{}   `json:"options,omitempty"`
}

// HuggingFaceModelInfo represents model information
type HuggingFaceModelInfo struct {
	ID            string   `json:"id"`
	ModelId       string   `json:"modelId"`
	Author        string   `json:"author"`
	Pipeline      string   `json:"pipeline_tag"`
	Tags          []string `json:"tags"`
	Downloads     int      `json:"downloads"`
	Likes         int      `json:"likes"`
	LibraryName   string   `json:"library_name"`
	Private       bool     `json:"private"`
	Disabled      bool     `json:"disabled"`
	Gated         bool     `json:"gated"`
	LastModified  string   `json:"lastModified"`
	CreatedAt     string   `json:"createdAt"`
}

// NewHuggingFaceLLM creates a new HuggingFace LLM instance
func NewHuggingFaceLLM(config *LLMConfig) (LLMProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	
	if config.Model == "" {
		return nil, fmt.Errorf("model name is required")
	}
	
	// Determine if using local model or API
	useLocal := config.BaseURL == "" || config.BaseURL == "local"
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api-inference.huggingface.co"
	}
	
	// Create HTTP client
	client := resty.New()
	if !useLocal {
		client.SetBaseURL(baseURL)
	}
	client.SetTimeout(config.Timeout)
	client.SetRetryCount(3)
	client.SetRetryWaitTime(1 * time.Second)
	client.SetRetryMaxWaitTime(10 * time.Second)
	
	// Set headers
	client.SetHeader("Content-Type", "application/json")
	client.SetHeader("User-Agent", "MemGOS/1.0")
	
	// Set authorization header if API key is provided
	if config.APIKey != "" {
		client.SetHeader("Authorization", "Bearer "+config.APIKey)
	}
	
	llm := &HuggingFaceLLM{
		BaseLLM:   NewBaseLLM(config.Model),
		client:    client,
		config:    config,
		baseURL:   baseURL,
		useLocal:  useLocal,
		modelPath: config.Model,
	}
	
	// Apply configuration
	llm.SetMaxTokens(config.MaxTokens)
	llm.SetTemperature(config.Temperature)
	llm.SetTopP(config.TopP)
	llm.SetTimeout(config.Timeout)
	
	return llm, nil
}

// Generate generates text based on messages
func (h *HuggingFaceLLM) Generate(ctx context.Context, messages types.MessageList) (string, error) {
	if err := h.ValidateMessages(messages); err != nil {
		return "", fmt.Errorf("invalid messages: %w", err)
	}
	
	if h.useLocal {
		return h.generateLocal(ctx, messages)
	}
	
	return h.generateAPI(ctx, messages)
}

// generateAPI generates text using HuggingFace API
func (h *HuggingFaceLLM) generateAPI(ctx context.Context, messages types.MessageList) (string, error) {
	// Build prompt from messages
	prompt := h.BuildPrompt(messages)
	
	// Build request
	req := HuggingFaceRequest{
		Inputs: prompt,
		Parameters: map[string]interface{}{
			"max_new_tokens":    h.GetMaxTokens(),
			"temperature":       h.GetTemperature(),
			"top_p":             h.GetTopP(),
			"do_sample":         h.GetTemperature() > 0,
			"return_full_text":  false,
			"clean_up_tokenization_spaces": true,
		},
		Options: map[string]interface{}{
			"wait_for_model": true,
			"use_cache":      false,
		},
	}
	
	var resp []HuggingFaceResponse
	var err error
	
	// Make request with retry
	err = retry.Do(
		func() error {
			response, reqErr := h.client.R().
				SetContext(ctx).
				SetBody(req).
				SetResult(&resp).
				Post("/models/" + h.GetModelName())
			
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
		return "", fmt.Errorf("HuggingFace API request failed: %w", err)
	}
	
	if len(resp) == 0 {
		return "", fmt.Errorf("no response from HuggingFace API")
	}
	
	// Handle errors in response
	if resp[0].Error != "" {
		return "", fmt.Errorf("API error: %s", resp[0].Error)
	}
	
	// Extract generated text
	generatedText := resp[0].GeneratedText
	if generatedText == "" {
		generatedText = resp[0].Text
	}
	
	// Record metrics
	h.RecordMetrics("response_length", len(generatedText))
	h.RecordMetrics("warnings", resp[0].Warnings)
	
	return generatedText, nil
}

// generateLocal generates text using local model (placeholder)
func (h *HuggingFaceLLM) generateLocal(ctx context.Context, messages types.MessageList) (string, error) {
	// This would integrate with local ONNX runtime or transformers
	// For now, return a placeholder indicating local inference is not implemented
	return "", fmt.Errorf("local HuggingFace inference not yet implemented")
}

// GenerateWithCache generates text with cache support
func (h *HuggingFaceLLM) GenerateWithCache(ctx context.Context, messages types.MessageList, cache interface{}) (string, error) {
	// HuggingFace API doesn't expose KV cache directly
	// Use regular generation with caching options
	return h.Generate(ctx, messages)
}

// GenerateStream generates text with streaming support
func (h *HuggingFaceLLM) GenerateStream(ctx context.Context, messages types.MessageList, stream chan<- string) error {
	if err := h.ValidateMessages(messages); err != nil {
		return fmt.Errorf("invalid messages: %w", err)
	}
	
	// HuggingFace Inference API doesn't support streaming for most models
	// Fall back to regular generation and send as single chunk
	response, err := h.Generate(ctx, messages)
	if err != nil {
		return err
	}
	
	// Send response in chunks to simulate streaming
	words := strings.Fields(response)
	for _, word := range words {
		select {
		case stream <- word + " ":
		case <-ctx.Done():
			return ctx.Err()
		}
		
		// Small delay to simulate streaming
		time.Sleep(50 * time.Millisecond)
	}
	
	return nil
}

// Embed generates embeddings for text
func (h *HuggingFaceLLM) Embed(ctx context.Context, text string) (types.EmbeddingVector, error) {
	if text == "" {
		return nil, fmt.Errorf("empty text")
	}
	
	if h.useLocal {
		return h.embedLocal(ctx, text)
	}
	
	return h.embedAPI(ctx, text)
}

// embedAPI generates embeddings using HuggingFace API
func (h *HuggingFaceLLM) embedAPI(ctx context.Context, text string) (types.EmbeddingVector, error) {
	// Use sentence-transformers model for embeddings
	embedModel := "sentence-transformers/all-MiniLM-L6-v2"
	if strings.Contains(h.GetModelName(), "embed") {
		embedModel = h.GetModelName()
	}
	
	req := HuggingFaceRequest{
		Inputs: text,
		Options: map[string]interface{}{
			"wait_for_model": true,
			"use_cache":      true,
		},
	}
	
	var resp [][]float32
	response, err := h.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&resp).
		Post("/models/" + embedModel)
	
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}
	
	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", response.StatusCode(), response.String())
	}
	
	if len(resp) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	
	// Record metrics
	h.RecordMetrics("embedding_dimension", len(resp[0]))
	
	return types.EmbeddingVector(resp[0]), nil
}

// embedLocal generates embeddings using local model (placeholder)
func (h *HuggingFaceLLM) embedLocal(ctx context.Context, text string) (types.EmbeddingVector, error) {
	// This would integrate with local ONNX runtime or transformers
	return nil, fmt.Errorf("local HuggingFace embeddings not yet implemented")
}

// GetProviderName returns the provider name
func (h *HuggingFaceLLM) GetProviderName() string {
	return "huggingface"
}

// GetSupportedModels returns supported models
func (h *HuggingFaceLLM) GetSupportedModels() []string {
	return []string{
		// Text generation models
		"microsoft/DialoGPT-small",
		"microsoft/DialoGPT-medium",
		"microsoft/DialoGPT-large",
		"gpt2",
		"gpt2-medium",
		"gpt2-large",
		"gpt2-xl",
		"distilgpt2",
		"EleutherAI/gpt-neo-125M",
		"EleutherAI/gpt-neo-1.3B",
		"EleutherAI/gpt-neo-2.7B",
		"EleutherAI/gpt-j-6B",
		"facebook/opt-125m",
		"facebook/opt-350m",
		"facebook/opt-1.3b",
		"facebook/opt-2.7b",
		"facebook/opt-6.7b",
		"facebook/opt-13b",
		"facebook/opt-30b",
		"bigscience/bloom-560m",
		"bigscience/bloom-1b1",
		"bigscience/bloom-1b7",
		"bigscience/bloom-3b",
		"bigscience/bloom-7b1",
		"bigscience/bloomz-560m",
		"bigscience/bloomz-1b1",
		"bigscience/bloomz-1b7",
		"bigscience/bloomz-3b",
		"bigscience/bloomz-7b1",
		// Code generation models
		"codeparrot/codeparrot-small",
		"Salesforce/codegen-350M-mono",
		"Salesforce/codegen-2B-mono",
		"Salesforce/codegen-6B-mono",
		"Salesforce/codegen-16B-mono",
		// Embedding models
		"sentence-transformers/all-MiniLM-L6-v2",
		"sentence-transformers/all-mpnet-base-v2",
		"sentence-transformers/all-distilroberta-v1",
		"sentence-transformers/all-roberta-large-v1",
		"sentence-transformers/multi-qa-MiniLM-L6-cos-v1",
		"sentence-transformers/multi-qa-mpnet-base-cos-v1",
		"sentence-transformers/multi-qa-distilbert-cos-v1",
		"sentence-transformers/paraphrase-MiniLM-L6-v2",
		"sentence-transformers/paraphrase-mpnet-base-v2",
		"sentence-transformers/paraphrase-distilroberta-base-v1",
		// Specialized models
		"facebook/blenderbot-400M-distill",
		"facebook/blenderbot-1B-distill",
		"facebook/blenderbot-3B",
		"microsoft/GODEL-v1_1-base-seq2seq",
		"microsoft/GODEL-v1_1-large-seq2seq",
		"Salesforce/blip-image-captioning-base",
		"Salesforce/blip-image-captioning-large",
	}
}

// HealthCheck performs health check
func (h *HuggingFaceLLM) HealthCheck(ctx context.Context) error {
	if h.useLocal {
		// For local models, check if model path exists
		return fmt.Errorf("local model health check not implemented")
	}
	
	// For API, try to get model info
	response, err := h.client.R().
		SetContext(ctx).
		Get("/models/" + h.GetModelName())
	
	if err != nil {
		return fmt.Errorf("HuggingFace health check failed: %w", err)
	}
	
	if response.StatusCode() != 200 {
		return fmt.Errorf("HuggingFace health check failed: HTTP %d", response.StatusCode())
	}
	
	return nil
}

// SetConfig sets the configuration
func (h *HuggingFaceLLM) SetConfig(config *LLMConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	
	h.config = config
	
	// Update base configuration
	h.SetMaxTokens(config.MaxTokens)
	h.SetTemperature(config.Temperature)
	h.SetTopP(config.TopP)
	h.SetTimeout(config.Timeout)
	
	return nil
}

// GetConfig returns the configuration
func (h *HuggingFaceLLM) GetConfig() *LLMConfig {
	return h.config
}

// Close closes the HuggingFace client
func (h *HuggingFaceLLM) Close() error {
	// Nothing to close for HTTP client
	return h.BaseLLM.Close()
}

// GetModelInfo returns detailed model information
func (h *HuggingFaceLLM) GetModelInfo() map[string]interface{} {
	info := h.BaseLLM.GetModelInfo()
	info["provider"] = h.GetProviderName()
	info["base_url"] = h.baseURL
	info["use_local"] = h.useLocal
	info["model_path"] = h.modelPath
	info["api_key_set"] = h.config.APIKey != ""
	
	return info
}

// SearchModels searches for models in HuggingFace Hub
func (h *HuggingFaceLLM) SearchModels(ctx context.Context, query string, limit int) ([]HuggingFaceModelInfo, error) {
	if h.useLocal {
		return nil, fmt.Errorf("model search not available for local models")
	}
	
	params := map[string]string{
		"search": query,
		"limit":  fmt.Sprintf("%d", limit),
	}
	
	var models []HuggingFaceModelInfo
	response, err := h.client.R().
		SetContext(ctx).
		SetQueryParams(params).
		SetResult(&models).
		Get("/models")
	
	if err != nil {
		return nil, fmt.Errorf("failed to search models: %w", err)
	}
	
	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", response.StatusCode(), response.String())
	}
	
	return models, nil
}

// GetModelDetails gets detailed information about a specific model
func (h *HuggingFaceLLM) GetModelDetails(ctx context.Context, modelName string) (*HuggingFaceModelInfo, error) {
	if h.useLocal {
		return nil, fmt.Errorf("model details not available for local models")
	}
	
	var model HuggingFaceModelInfo
	response, err := h.client.R().
		SetContext(ctx).
		SetResult(&model).
		Get("/models/" + modelName)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get model details: %w", err)
	}
	
	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", response.StatusCode(), response.String())
	}
	
	return &model, nil
}

// IsModelSupported checks if a model is supported
func (h *HuggingFaceLLM) IsModelSupported(model string) bool {
	// For HuggingFace, we support any model that exists on the hub
	// This is a simplified check
	return true
}

// GetDefaultModel returns the default model for HuggingFace
func (h *HuggingFaceLLM) GetDefaultModel() string {
	return "microsoft/DialoGPT-small"
}

// GetModelCapabilities returns model capabilities
func (h *HuggingFaceLLM) GetModelCapabilities(model string) map[string]interface{} {
	capabilities := map[string]interface{}{
		"chat":         true,
		"completion":   true,
		"embeddings":   false,
		"streaming":    false, // Limited streaming support
		"local":        h.useLocal,
		"fine_tuning":  false,
	}
	
	// Model-specific capabilities
	if strings.Contains(model, "embed") || strings.Contains(model, "sentence-transformers") {
		capabilities["embeddings"] = true
		capabilities["chat"] = false
		capabilities["completion"] = false
	}
	
	if strings.Contains(model, "code") {
		capabilities["code_generation"] = true
	}
	
	if strings.Contains(model, "bloom") || strings.Contains(model, "opt") || strings.Contains(model, "gpt") {
		capabilities["multilingual"] = true
	}
	
	return capabilities
}

// GenerateCode generates code using a code-specific model
func (h *HuggingFaceLLM) GenerateCode(ctx context.Context, prompt string, language string) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("empty prompt")
	}
	
	// Use code-specific model if available
	codeModel := h.GetModelName()
	if !strings.Contains(codeModel, "code") {
		codeModel = "Salesforce/codegen-350M-mono"
	}
	
	// Build code generation request
	fullPrompt := fmt.Sprintf("# Language: %s\n# Task: %s\n\n", language, prompt)
	
	req := HuggingFaceRequest{
		Inputs: fullPrompt,
		Parameters: map[string]interface{}{
			"max_new_tokens":   h.GetMaxTokens(),
			"temperature":      h.GetTemperature(),
			"top_p":            h.GetTopP(),
			"do_sample":        true,
			"return_full_text": false,
		},
	}
	
	var resp []HuggingFaceResponse
	response, err := h.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&resp).
		Post("/models/" + codeModel)
	
	if err != nil {
		return "", fmt.Errorf("code generation request failed: %w", err)
	}
	
	if response.StatusCode() != 200 {
		return "", fmt.Errorf("HTTP %d: %s", response.StatusCode(), response.String())
	}
	
	if len(resp) == 0 {
		return "", fmt.Errorf("no response from code generation")
	}
	
	return resp[0].GeneratedText, nil
}

// SetUseLocal sets whether to use local model inference
func (h *HuggingFaceLLM) SetUseLocal(useLocal bool) {
	h.useLocal = useLocal
}

// GetTokenUsage returns token usage statistics
func (h *HuggingFaceLLM) GetTokenUsage() map[string]interface{} {
	metrics := h.GetMetrics()
	
	return map[string]interface{}{
		"response_length":      metrics["response_length"],
		"embedding_dimension":  metrics["embedding_dimension"],
		"warnings":             metrics["warnings"],
	}
}

// SupportsBatching returns whether the model supports batch processing
func (h *HuggingFaceLLM) SupportsBatching() bool {
	return !h.useLocal // API supports batching, local inference TBD
}

// BatchGenerate generates text for multiple prompts in batch
func (h *HuggingFaceLLM) BatchGenerate(ctx context.Context, prompts []string) ([]string, error) {
	if !h.SupportsBatching() {
		return nil, fmt.Errorf("batch generation not supported for local models")
	}
	
	if len(prompts) == 0 {
		return nil, fmt.Errorf("empty prompts")
	}
	
	// Build batch request
	req := HuggingFaceRequest{
		Inputs: prompts,
		Parameters: map[string]interface{}{
			"max_new_tokens":   h.GetMaxTokens(),
			"temperature":      h.GetTemperature(),
			"top_p":            h.GetTopP(),
			"do_sample":        h.GetTemperature() > 0,
			"return_full_text": false,
		},
	}
	
	var resp [][]HuggingFaceResponse
	response, err := h.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&resp).
		Post("/models/" + h.GetModelName())
	
	if err != nil {
		return nil, fmt.Errorf("batch generation request failed: %w", err)
	}
	
	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", response.StatusCode(), response.String())
	}
	
	// Extract responses
	results := make([]string, len(resp))
	for i, responseGroup := range resp {
		if len(responseGroup) > 0 {
			results[i] = responseGroup[0].GeneratedText
		}
	}
	
	return results, nil
}