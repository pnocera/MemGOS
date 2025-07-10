// Package llm provides LLM (Large Language Model) implementations for MemGOS
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// BaseLLM provides common functionality for all LLM implementations
type BaseLLM struct {
	modelName   string
	maxTokens   int
	temperature float64
	topP        float64
	timeout     time.Duration
	metrics     map[string]interface{}
}

// NewBaseLLM creates a new base LLM instance
func NewBaseLLM(modelName string) *BaseLLM {
	return &BaseLLM{
		modelName:   modelName,
		maxTokens:   1024,
		temperature: 0.7,
		topP:        0.9,
		timeout:     30 * time.Second,
		metrics:     make(map[string]interface{}),
	}
}

// SetMaxTokens sets the maximum number of tokens
func (b *BaseLLM) SetMaxTokens(maxTokens int) {
	b.maxTokens = maxTokens
}

// SetTemperature sets the temperature for generation
func (b *BaseLLM) SetTemperature(temperature float64) {
	b.temperature = temperature
}

// SetTopP sets the top-p value for nucleus sampling
func (b *BaseLLM) SetTopP(topP float64) {
	b.topP = topP
}

// SetTimeout sets the request timeout
func (b *BaseLLM) SetTimeout(timeout time.Duration) {
	b.timeout = timeout
}

// GetMaxTokens returns the maximum number of tokens
func (b *BaseLLM) GetMaxTokens() int {
	return b.maxTokens
}

// GetTemperature returns the temperature
func (b *BaseLLM) GetTemperature() float64 {
	return b.temperature
}

// GetTopP returns the top-p value
func (b *BaseLLM) GetTopP() float64 {
	return b.topP
}

// GetTimeout returns the request timeout
func (b *BaseLLM) GetTimeout() time.Duration {
	return b.timeout
}

// GetModelName returns the model name
func (b *BaseLLM) GetModelName() string {
	return b.modelName
}

// GetModelInfo returns model information
func (b *BaseLLM) GetModelInfo() map[string]interface{} {
	return map[string]interface{}{
		"model":       b.modelName,
		"max_tokens":  b.maxTokens,
		"temperature": b.temperature,
		"top_p":       b.topP,
		"timeout":     b.timeout.String(),
		"metrics":     b.metrics,
	}
}

// ValidateMessages validates the message list
func (b *BaseLLM) ValidateMessages(messages types.MessageList) error {
	if len(messages) == 0 {
		return fmt.Errorf("empty message list")
	}

	for i, msg := range messages {
		if msg.Role == "" {
			return fmt.Errorf("message %d: role is required", i)
		}
		if msg.Content == "" {
			return fmt.Errorf("message %d: content is required", i)
		}
		if msg.Role != types.MessageRoleUser && 
		   msg.Role != types.MessageRoleAssistant && 
		   msg.Role != types.MessageRoleSystem {
			return fmt.Errorf("message %d: invalid role %s", i, msg.Role)
		}
	}

	return nil
}

// FormatMessages formats messages for API consumption
func (b *BaseLLM) FormatMessages(messages types.MessageList) []map[string]interface{} {
	formatted := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		formatted[i] = map[string]interface{}{
			"role":    string(msg.Role),
			"content": msg.Content,
		}
	}
	return formatted
}

// ExtractContent extracts text content from API response
func (b *BaseLLM) ExtractContent(response interface{}) (string, error) {
	switch v := response.(type) {
	case string:
		return v, nil
	case map[string]interface{}:
		// Try common response formats
		if content, ok := v["content"].(string); ok {
			return content, nil
		}
		if message, ok := v["message"].(map[string]interface{}); ok {
			if content, ok := message["content"].(string); ok {
				return content, nil
			}
		}
		if choices, ok := v["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if message, ok := choice["message"].(map[string]interface{}); ok {
					if content, ok := message["content"].(string); ok {
						return content, nil
					}
				}
			}
		}
		return "", fmt.Errorf("no content found in response")
	default:
		return "", fmt.Errorf("unsupported response type: %T", v)
	}
}

// RecordMetrics records usage metrics
func (b *BaseLLM) RecordMetrics(metric string, value interface{}) {
	b.metrics[metric] = value
}

// GetMetrics returns accumulated metrics
func (b *BaseLLM) GetMetrics() map[string]interface{} {
	return b.metrics
}

// TokenCount estimates token count for text (simple heuristic)
func (b *BaseLLM) TokenCount(text string) int {
	// Simple heuristic: ~4 characters per token
	return len(text) / 4
}

// TruncateToTokens truncates text to fit within token limit
func (b *BaseLLM) TruncateToTokens(text string, maxTokens int) string {
	if maxTokens <= 0 {
		return ""
	}
	
	maxChars := maxTokens * 4 // Simple heuristic
	if len(text) <= maxChars {
		return text
	}
	
	return text[:maxChars]
}

// GenerateRequestID generates a unique request ID
func (b *BaseLLM) GenerateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// ParseJSONResponse parses JSON response from API
func (b *BaseLLM) ParseJSONResponse(data []byte) (map[string]interface{}, error) {
	var response map[string]interface{}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}
	return response, nil
}

// BuildPrompt builds a prompt from message list
func (b *BaseLLM) BuildPrompt(messages types.MessageList) string {
	var builder strings.Builder
	
	for _, msg := range messages {
		switch msg.Role {
		case types.MessageRoleSystem:
			builder.WriteString(fmt.Sprintf("System: %s\n", msg.Content))
		case types.MessageRoleUser:
			builder.WriteString(fmt.Sprintf("User: %s\n", msg.Content))
		case types.MessageRoleAssistant:
			builder.WriteString(fmt.Sprintf("Assistant: %s\n", msg.Content))
		}
	}
	
	return builder.String()
}

// Close provides default close implementation
func (b *BaseLLM) Close() error {
	// Base implementation - nothing to close
	return nil
}

// StreamingLLM provides streaming capabilities for LLM implementations
type StreamingLLM struct {
	*BaseLLM
	streamBuffer chan string
	streamActive bool
}

// NewStreamingLLM creates a new streaming LLM instance
func NewStreamingLLM(modelName string) *StreamingLLM {
	return &StreamingLLM{
		BaseLLM:      NewBaseLLM(modelName),
		streamBuffer: make(chan string, 1000),
		streamActive: false,
	}
}

// StartStreaming starts streaming mode
func (s *StreamingLLM) StartStreaming() {
	s.streamActive = true
}

// StopStreaming stops streaming mode
func (s *StreamingLLM) StopStreaming() {
	s.streamActive = false
	if s.streamBuffer != nil {
		close(s.streamBuffer)
		s.streamBuffer = make(chan string, 1000)
	}
}

// IsStreaming returns whether streaming is active
func (s *StreamingLLM) IsStreaming() bool {
	return s.streamActive
}

// WriteToStream writes content to the stream
func (s *StreamingLLM) WriteToStream(content string) {
	if s.streamActive && s.streamBuffer != nil {
		select {
		case s.streamBuffer <- content:
		default:
			// Buffer full, skip
		}
	}
}

// ReadFromStream reads from the stream
func (s *StreamingLLM) ReadFromStream() <-chan string {
	return s.streamBuffer
}

// Close closes the streaming LLM
func (s *StreamingLLM) Close() error {
	s.StopStreaming()
	return s.BaseLLM.Close()
}

// LLMResponse represents a generic LLM response
type LLMResponse struct {
	Content      string                 `json:"content"`
	TokensUsed   int                    `json:"tokens_used"`
	Model        string                 `json:"model"`
	FinishReason string                 `json:"finish_reason"`
	Metadata     map[string]interface{} `json:"metadata"`
	RequestID    string                 `json:"request_id"`
}

// LLMError represents an LLM-specific error
type LLMError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

// Error implements the error interface
func (e *LLMError) Error() string {
	return fmt.Sprintf("LLM Error [%s]: %s", e.Code, e.Message)
}

// NewLLMError creates a new LLM error
func NewLLMError(code, message, errType string) *LLMError {
	return &LLMError{
		Code:    code,
		Message: message,
		Type:    errType,
	}
}

// LLMConfig represents configuration for LLM instances
type LLMConfig struct {
	Provider    string            `json:"provider"`
	Model       string            `json:"model"`
	APIKey      string            `json:"api_key"`
	BaseURL     string            `json:"base_url"`
	MaxTokens   int               `json:"max_tokens"`
	Temperature float64           `json:"temperature"`
	TopP        float64           `json:"top_p"`
	Timeout     time.Duration     `json:"timeout"`
	Extra       map[string]interface{} `json:"extra"`
}

// Validate validates the LLM configuration
func (c *LLMConfig) Validate() error {
	if c.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	if c.MaxTokens < 0 {
		return fmt.Errorf("max_tokens must be non-negative")
	}
	if c.Temperature < 0 || c.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	if c.TopP < 0 || c.TopP > 1 {
		return fmt.Errorf("top_p must be between 0 and 1")
	}
	return nil
}

// DefaultLLMConfig returns default LLM configuration
func DefaultLLMConfig() *LLMConfig {
	return &LLMConfig{
		Provider:    "openai",
		Model:       "gpt-3.5-turbo",
		MaxTokens:   1024,
		Temperature: 0.7,
		TopP:        0.9,
		Timeout:     30 * time.Second,
		Extra:       make(map[string]interface{}),
	}
}

// LLMProvider defines the interface for LLM provider implementations
type LLMProvider interface {
	interfaces.LLM
	GetProviderName() string
	GetSupportedModels() []string
	HealthCheck(ctx context.Context) error
	SetConfig(config *LLMConfig) error
	GetConfig() *LLMConfig
}