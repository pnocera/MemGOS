package chunkers

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// TokenizerProvider defines the interface for tokenizers
type TokenizerProvider interface {
	// CountTokens returns the number of tokens in the text
	CountTokens(text string) (int, error)

	// CountTokensBatch returns token counts for multiple texts
	CountTokensBatch(texts []string) ([]int, error)

	// GetModelInfo returns information about the tokenizer model
	GetModelInfo() TokenizerModelInfo

	// GetMaxTokens returns the maximum number of tokens supported
	GetMaxTokens() int

	// Close releases any resources held by the tokenizer
	Close() error
}

// TokenizerModelInfo contains information about a tokenizer model
type TokenizerModelInfo struct {
	// Name of the model
	Name string `json:"name"`

	// Provider of the tokenizer (e.g., "openai", "huggingface", "local")
	Provider string `json:"provider"`

	// MaxTokens supported by the model
	MaxTokens int `json:"max_tokens"`

	// Version of the tokenizer
	Version string `json:"version"`

	// Description of the tokenizer
	Description string `json:"description"`

	// VocabularySize is the size of the tokenizer vocabulary
	VocabularySize int `json:"vocabulary_size"`

	// SupportedLanguages lists languages supported by the tokenizer
	SupportedLanguages []string `json:"supported_languages"`
}

// TokenizerConfig contains configuration for tokenizers
type TokenizerConfig struct {
	// Provider specifies which tokenizer provider to use
	Provider string `json:"provider"`

	// ModelName specifies the model to use
	ModelName string `json:"model_name"`

	// APIKey for API-based tokenizers
	APIKey string `json:"api_key,omitempty"`

	// APIEndpoint for custom API endpoints
	APIEndpoint string `json:"api_endpoint,omitempty"`

	// LocalModelPath for local tokenizer files
	LocalModelPath string `json:"local_model_path,omitempty"`

	// RequestTimeout for API requests
	RequestTimeout time.Duration `json:"request_timeout"`

	// RetryAttempts for failed requests
	RetryAttempts int `json:"retry_attempts"`

	// CacheEnabled enables token count caching
	CacheEnabled bool `json:"cache_enabled"`

	// CacheSize is the maximum number of token counts to cache
	CacheSize int `json:"cache_size"`

	// CacheTTL is the time-to-live for cached token counts
	CacheTTL time.Duration `json:"cache_ttl"`
}

// DefaultTokenizerConfig returns a default tokenizer configuration
func DefaultTokenizerConfig() *TokenizerConfig {
	return &TokenizerConfig{
		Provider:       "tiktoken",
		ModelName:      "gpt-4o",
		RequestTimeout: 10 * time.Second,
		RetryAttempts:  2,
		CacheEnabled:   true,
		CacheSize:      10000,
		CacheTTL:       1 * time.Hour,
	}
}

// TokenizerFactory creates tokenizer providers
type TokenizerFactory struct{}

// NewTokenizerFactory creates a new tokenizer factory
func NewTokenizerFactory() *TokenizerFactory {
	return &TokenizerFactory{}
}

// CreateTokenizer creates a tokenizer based on configuration
func (tf *TokenizerFactory) CreateTokenizer(config *TokenizerConfig) (TokenizerProvider, error) {
	if config == nil {
		config = DefaultTokenizerConfig()
	}

	switch config.Provider {
	case "tiktoken":
		return NewTikTokenProvider(config)
	case "openai":
		return NewOpenAITokenizerProvider(config)
	case "huggingface":
		return NewHuggingFaceTokenizerProvider(config)
	case "simple":
		return NewSimpleTokenizerProvider(config)
	default:
		return nil, fmt.Errorf("unsupported tokenizer provider: %s", config.Provider)
	}
}

// GetSupportedProviders returns a list of supported tokenizer providers
func (tf *TokenizerFactory) GetSupportedProviders() []string {
	return []string{"tiktoken", "openai", "huggingface", "simple"}
}

// ValidateConfig validates a tokenizer configuration
func (tf *TokenizerFactory) ValidateConfig(config *TokenizerConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.Provider == "" {
		return fmt.Errorf("provider cannot be empty")
	}

	if config.ModelName == "" {
		return fmt.Errorf("model name cannot be empty")
	}

	if config.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout must be positive")
	}

	if config.RetryAttempts < 0 {
		return fmt.Errorf("retry attempts cannot be negative")
	}

	return nil
}

// TikTokenProvider implements TokenizerProvider using tiktoken-style tokenization
type TikTokenProvider struct {
	config    *TokenizerConfig
	modelInfo TokenizerModelInfo
	encoder   *TikTokenEncoder
}

// NewTikTokenProvider creates a new tiktoken provider
func NewTikTokenProvider(config *TokenizerConfig) (*TikTokenProvider, error) {
	if config == nil {
		config = DefaultTokenizerConfig()
	}

	encoder, err := NewTikTokenEncoder(config.ModelName)
	if err != nil {
		return nil, fmt.Errorf("failed to create tiktoken encoder: %w", err)
	}

	modelInfo := getTikTokenModelInfo(config.ModelName)

	return &TikTokenProvider{
		config:    config,
		modelInfo: modelInfo,
		encoder:   encoder,
	}, nil
}

// CountTokens returns the number of tokens in the text
func (p *TikTokenProvider) CountTokens(text string) (int, error) {
	return p.encoder.CountTokens(text)
}

// CountTokensBatch returns token counts for multiple texts
func (p *TikTokenProvider) CountTokensBatch(texts []string) ([]int, error) {
	counts := make([]int, len(texts))
	for i, text := range texts {
		count, err := p.CountTokens(text)
		if err != nil {
			return nil, fmt.Errorf("failed to count tokens for text %d: %w", i, err)
		}
		counts[i] = count
	}
	return counts, nil
}

// GetModelInfo returns information about the tokenizer model
func (p *TikTokenProvider) GetModelInfo() TokenizerModelInfo {
	return p.modelInfo
}

// GetMaxTokens returns the maximum number of tokens supported
func (p *TikTokenProvider) GetMaxTokens() int {
	return p.modelInfo.MaxTokens
}

// Close releases any resources held by the tokenizer
func (p *TikTokenProvider) Close() error {
	return nil
}

// TikTokenEncoder implements a simplified tiktoken-style encoder
type TikTokenEncoder struct {
	modelName string
	patterns  []*regexp.Regexp
}

// NewTikTokenEncoder creates a new tiktoken encoder
func NewTikTokenEncoder(modelName string) (*TikTokenEncoder, error) {
	patterns := []*regexp.Regexp{
		// Common patterns for tokenization
		regexp.MustCompile(`'s|'t|'re|'ve|'m|'ll|'d`), // Contractions
		regexp.MustCompile(`[^\s\w]`),                 // Punctuation
		regexp.MustCompile(`\w+`),                     // Words
		regexp.MustCompile(`\s+`),                     // Whitespace
	}

	return &TikTokenEncoder{
		modelName: modelName,
		patterns:  patterns,
	}, nil
}

// CountTokens counts tokens using tiktoken-style logic
func (e *TikTokenEncoder) CountTokens(text string) (int, error) {
	if text == "" {
		return 0, nil
	}

	// Use a more sophisticated tokenization approach
	tokens := e.tokenize(text)
	return len(tokens), nil
}

// tokenize splits text into tokens
func (e *TikTokenEncoder) tokenize(text string) []string {
	var tokens []string

	// Handle different types of content
	for _, pattern := range e.patterns {
		matches := pattern.FindAllString(text, -1)
		for _, match := range matches {
			if strings.TrimSpace(match) != "" {
				tokens = append(tokens, match)
			}
		}
	}

	// If no patterns matched, fall back to simple word splitting
	if len(tokens) == 0 {
		tokens = strings.Fields(text)
	}

	return tokens
}

// SimpleTokenizerProvider implements a simple word-based tokenizer
type SimpleTokenizerProvider struct {
	config    *TokenizerConfig
	modelInfo TokenizerModelInfo
}

// NewSimpleTokenizerProvider creates a new simple tokenizer provider
func NewSimpleTokenizerProvider(config *TokenizerConfig) (*SimpleTokenizerProvider, error) {
	if config == nil {
		config = DefaultTokenizerConfig()
	}

	modelInfo := TokenizerModelInfo{
		Name:               "simple",
		Provider:           "simple",
		MaxTokens:          100000,
		Version:            "1.0",
		Description:        "Simple word-based tokenizer",
		VocabularySize:     -1, // Unknown
		SupportedLanguages: []string{"en"},
	}

	return &SimpleTokenizerProvider{
		config:    config,
		modelInfo: modelInfo,
	}, nil
}

// CountTokens returns the number of tokens using simple word counting
func (p *SimpleTokenizerProvider) CountTokens(text string) (int, error) {
	if text == "" {
		return 0, nil
	}

	// Enhanced simple tokenization
	tokenCount := 0

	// Split by whitespace and count words
	words := strings.Fields(text)
	tokenCount += len(words)

	// Add tokens for punctuation (roughly)
	punctuationCount := 0
	for _, r := range text {
		if unicode.IsPunct(r) {
			punctuationCount++
		}
	}
	tokenCount += punctuationCount / 2 // Rough estimate

	// Add tokens for special characters
	specialCount := strings.Count(text, "\n") + strings.Count(text, "\t")
	tokenCount += specialCount

	return tokenCount, nil
}

// CountTokensBatch returns token counts for multiple texts
func (p *SimpleTokenizerProvider) CountTokensBatch(texts []string) ([]int, error) {
	counts := make([]int, len(texts))
	for i, text := range texts {
		count, err := p.CountTokens(text)
		if err != nil {
			return nil, fmt.Errorf("failed to count tokens for text %d: %w", i, err)
		}
		counts[i] = count
	}
	return counts, nil
}

// GetModelInfo returns information about the tokenizer model
func (p *SimpleTokenizerProvider) GetModelInfo() TokenizerModelInfo {
	return p.modelInfo
}

// GetMaxTokens returns the maximum number of tokens supported
func (p *SimpleTokenizerProvider) GetMaxTokens() int {
	return p.modelInfo.MaxTokens
}

// Close releases any resources held by the tokenizer
func (p *SimpleTokenizerProvider) Close() error {
	return nil
}

// OpenAITokenizerProvider implements tokenizer using OpenAI's API
type OpenAITokenizerProvider struct {
	config     *TokenizerConfig
	httpClient *http.Client
	modelInfo  TokenizerModelInfo
}

// NewOpenAITokenizerProvider creates a new OpenAI tokenizer provider
func NewOpenAITokenizerProvider(config *TokenizerConfig) (*OpenAITokenizerProvider, error) {
	if config == nil {
		config = DefaultTokenizerConfig()
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required for OpenAI tokenizer")
	}

	httpClient := &http.Client{
		Timeout: config.RequestTimeout,
	}

	modelInfo := getOpenAITokenizerModelInfo(config.ModelName)

	return &OpenAITokenizerProvider{
		config:     config,
		httpClient: httpClient,
		modelInfo:  modelInfo,
	}, nil
}

// CountTokens returns the number of tokens using OpenAI's API
func (p *OpenAITokenizerProvider) CountTokens(text string) (int, error) {
	// This would make an API call to OpenAI's tokenizer endpoint
	// For now, we'll use a fallback estimation
	return p.estimateTokens(text), nil
}

// estimateTokens provides a rough token estimation
func (p *OpenAITokenizerProvider) estimateTokens(text string) int {
	if text == "" {
		return 0
	}

	// More sophisticated estimation than the simple 4-char rule
	words := strings.Fields(text)
	tokenCount := len(words)

	// Account for subword tokens
	for _, word := range words {
		if len(word) > 6 {
			tokenCount += (len(word) - 6) / 3 // Rough subword estimation
		}
	}

	// Add tokens for punctuation
	for _, r := range text {
		if unicode.IsPunct(r) {
			tokenCount++
		}
	}

	return tokenCount
}

// CountTokensBatch returns token counts for multiple texts
func (p *OpenAITokenizerProvider) CountTokensBatch(texts []string) ([]int, error) {
	counts := make([]int, len(texts))
	for i, text := range texts {
		count, err := p.CountTokens(text)
		if err != nil {
			return nil, fmt.Errorf("failed to count tokens for text %d: %w", i, err)
		}
		counts[i] = count
	}
	return counts, nil
}

// GetModelInfo returns information about the tokenizer model
func (p *OpenAITokenizerProvider) GetModelInfo() TokenizerModelInfo {
	return p.modelInfo
}

// GetMaxTokens returns the maximum number of tokens supported
func (p *OpenAITokenizerProvider) GetMaxTokens() int {
	return p.modelInfo.MaxTokens
}

// Close releases any resources held by the tokenizer
func (p *OpenAITokenizerProvider) Close() error {
	if p.httpClient != nil {
		p.httpClient.CloseIdleConnections()
	}
	return nil
}

// HuggingFaceTokenizerProvider implements tokenizer using Hugging Face tokenizers
type HuggingFaceTokenizerProvider struct {
	config    *TokenizerConfig
	modelInfo TokenizerModelInfo
}

// NewHuggingFaceTokenizerProvider creates a new Hugging Face tokenizer provider
func NewHuggingFaceTokenizerProvider(config *TokenizerConfig) (*HuggingFaceTokenizerProvider, error) {
	if config == nil {
		config = DefaultTokenizerConfig()
	}

	modelInfo := TokenizerModelInfo{
		Name:               config.ModelName,
		Provider:           "huggingface",
		MaxTokens:          512,
		Version:            "1.0",
		Description:        "Hugging Face tokenizer",
		VocabularySize:     30000,
		SupportedLanguages: []string{"en", "multi"},
	}

	return &HuggingFaceTokenizerProvider{
		config:    config,
		modelInfo: modelInfo,
	}, nil
}

// CountTokens returns the number of tokens using Hugging Face tokenizer
func (p *HuggingFaceTokenizerProvider) CountTokens(text string) (int, error) {
	// This would use the Hugging Face tokenizers library
	// For now, we'll use a BERT-style estimation
	return p.estimateBERTTokens(text), nil
}

// estimateBERTTokens provides BERT-style token estimation
func (p *HuggingFaceTokenizerProvider) estimateBERTTokens(text string) int {
	if text == "" {
		return 0
	}

	// BERT uses WordPiece tokenization
	words := strings.Fields(strings.ToLower(text))
	tokenCount := 0

	for _, word := range words {
		// Estimate subword tokens
		if len(word) <= 4 {
			tokenCount += 1
		} else if len(word) <= 8 {
			tokenCount += 2
		} else {
			tokenCount += (len(word) + 3) / 4 // Rough WordPiece estimation
		}
	}

	// Add special tokens
	tokenCount += 2 // [CLS] and [SEP]

	return tokenCount
}

// CountTokensBatch returns token counts for multiple texts
func (p *HuggingFaceTokenizerProvider) CountTokensBatch(texts []string) ([]int, error) {
	counts := make([]int, len(texts))
	for i, text := range texts {
		count, err := p.CountTokens(text)
		if err != nil {
			return nil, fmt.Errorf("failed to count tokens for text %d: %w", i, err)
		}
		counts[i] = count
	}
	return counts, nil
}

// GetModelInfo returns information about the tokenizer model
func (p *HuggingFaceTokenizerProvider) GetModelInfo() TokenizerModelInfo {
	return p.modelInfo
}

// GetMaxTokens returns the maximum number of tokens supported
func (p *HuggingFaceTokenizerProvider) GetMaxTokens() int {
	return p.modelInfo.MaxTokens
}

// Close releases any resources held by the tokenizer
func (p *HuggingFaceTokenizerProvider) Close() error {
	return nil
}

// Model info helper functions
func getTikTokenModelInfo(modelName string) TokenizerModelInfo {
	switch modelName {
	case "gpt-4o", "gpt-4o-mini":
		return TokenizerModelInfo{
			Name:               modelName,
			Provider:           "tiktoken",
			MaxTokens:          128000,
			Version:            "o200k_base",
			Description:        "GPT-4o tokenizer",
			VocabularySize:     200000,
			SupportedLanguages: []string{"en", "multi"},
		}
	case "gpt-4", "gpt-4-turbo":
		return TokenizerModelInfo{
			Name:               modelName,
			Provider:           "tiktoken",
			MaxTokens:          8192,
			Version:            "cl100k_base",
			Description:        "GPT-4 tokenizer",
			VocabularySize:     100000,
			SupportedLanguages: []string{"en", "multi"},
		}
	case "gpt-3.5-turbo":
		return TokenizerModelInfo{
			Name:               modelName,
			Provider:           "tiktoken",
			MaxTokens:          4096,
			Version:            "cl100k_base",
			Description:        "GPT-3.5 tokenizer",
			VocabularySize:     100000,
			SupportedLanguages: []string{"en", "multi"},
		}
	default:
		return TokenizerModelInfo{
			Name:               modelName,
			Provider:           "tiktoken",
			MaxTokens:          4096,
			Version:            "cl100k_base",
			Description:        "Default tiktoken tokenizer",
			VocabularySize:     100000,
			SupportedLanguages: []string{"en"},
		}
	}
}

func getOpenAITokenizerModelInfo(modelName string) TokenizerModelInfo {
	return getTikTokenModelInfo(modelName)
}
