package chunkers

import (
	"testing"
	"time"
)

func TestTokenizerFactory(t *testing.T) {
	factory := NewTokenizerFactory()
	
	// Test supported providers
	providers := factory.GetSupportedProviders()
	expectedProviders := []string{"tiktoken", "openai", "huggingface", "simple"}
	
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

func TestTokenizerConfig(t *testing.T) {
	config := DefaultTokenizerConfig()
	
	if config.Provider != "tiktoken" {
		t.Errorf("Expected default provider to be 'tiktoken', got %s", config.Provider)
	}
	
	if config.ModelName != "gpt-4o" {
		t.Errorf("Expected default model to be 'gpt-4o', got %s", config.ModelName)
	}
	
	if config.RequestTimeout != 10*time.Second {
		t.Errorf("Expected default timeout to be 10s, got %v", config.RequestTimeout)
	}
	
	if !config.CacheEnabled {
		t.Error("Expected cache to be enabled by default")
	}
}

func TestTokenizerFactoryValidation(t *testing.T) {
	factory := NewTokenizerFactory()
	
	// Test nil config
	err := factory.ValidateConfig(nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}
	
	// Test empty provider
	config := &TokenizerConfig{
		Provider: "",
	}
	err = factory.ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for empty provider")
	}
	
	// Test empty model name
	config = &TokenizerConfig{
		Provider:  "tiktoken",
		ModelName: "",
	}
	err = factory.ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for empty model name")
	}
	
	// Test invalid timeout
	config = &TokenizerConfig{
		Provider:       "tiktoken",
		ModelName:      "gpt-4",
		RequestTimeout: 0,
	}
	err = factory.ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for invalid timeout")
	}
	
	// Test valid config
	config = DefaultTokenizerConfig()
	err = factory.ValidateConfig(config)
	if err != nil {
		t.Errorf("Expected no error for valid config, got %v", err)
	}
}

func TestSimpleTokenizerProvider(t *testing.T) {
	provider, err := NewSimpleTokenizerProvider(nil)
	if err != nil {
		t.Errorf("Expected no error creating simple tokenizer, got %v", err)
	}
	
	// Test basic tokenization
	testCases := []struct {
		text        string
		expectedMin int
	}{
		{"", 0},
		{"hello", 1},
		{"hello world", 2},
		{"Hello, world!", 3}, // At least 3 tokens
		{"This is a test.\nNew line.", 7}, // At least 7 tokens
	}
	
	for _, tc := range testCases {
		count, err := provider.CountTokens(tc.text)
		if err != nil {
			t.Errorf("Expected no error tokenizing '%s', got %v", tc.text, err)
		}
		if count < tc.expectedMin {
			t.Errorf("Expected at least %d tokens for '%s', got %d", tc.expectedMin, tc.text, count)
		}
	}
}

func TestSimpleTokenizerBatch(t *testing.T) {
	provider, err := NewSimpleTokenizerProvider(nil)
	if err != nil {
		t.Errorf("Expected no error creating simple tokenizer, got %v", err)
	}
	
	texts := []string{
		"hello",
		"hello world",
		"Hello, world!",
	}
	
	counts, err := provider.CountTokensBatch(texts)
	if err != nil {
		t.Errorf("Expected no error in batch tokenization, got %v", err)
	}
	
	if len(counts) != len(texts) {
		t.Errorf("Expected %d counts, got %d", len(texts), len(counts))
	}
	
	// Verify individual counts match
	for i, text := range texts {
		expected, _ := provider.CountTokens(text)
		if counts[i] != expected {
			t.Errorf("Expected count %d for text %d, got %d", expected, i, counts[i])
		}
	}
}

func TestTikTokenEncoder(t *testing.T) {
	encoder, err := NewTikTokenEncoder("gpt-4")
	if err != nil {
		t.Errorf("Expected no error creating encoder, got %v", err)
	}
	
	// Test basic tokenization
	testCases := []struct {
		text     string
		minCount int // Minimum expected tokens
	}{
		{"", 0},
		{"hello", 1},
		{"hello world", 2},
		{"The quick brown fox jumps over the lazy dog.", 10},
		{"I can't believe it's not butter!", 7},
	}
	
	for _, tc := range testCases {
		count, err := encoder.CountTokens(tc.text)
		if err != nil {
			t.Errorf("Expected no error tokenizing '%s', got %v", tc.text, err)
		}
		if count < tc.minCount {
			t.Errorf("Expected at least %d tokens for '%s', got %d", tc.minCount, tc.text, count)
		}
	}
}

func TestTikTokenProvider(t *testing.T) {
	provider, err := NewTikTokenProvider(nil)
	if err != nil {
		t.Errorf("Expected no error creating tiktoken provider, got %v", err)
	}
	
	// Test model info
	info := provider.GetModelInfo()
	if info.Provider != "tiktoken" {
		t.Errorf("Expected provider to be 'tiktoken', got %s", info.Provider)
	}
	
	if info.MaxTokens <= 0 {
		t.Errorf("Expected positive max tokens, got %d", info.MaxTokens)
	}
	
	// Test tokenization
	count, err := provider.CountTokens("Hello, world!")
	if err != nil {
		t.Errorf("Expected no error tokenizing, got %v", err)
	}
	
	if count <= 0 {
		t.Errorf("Expected positive token count, got %d", count)
	}
}

func TestOpenAITokenizerProvider(t *testing.T) {
	config := &TokenizerConfig{
		Provider:       "openai",
		ModelName:      "gpt-4",
		APIKey:         "test-key",
		RequestTimeout: 10 * time.Second,
	}
	
	provider, err := NewOpenAITokenizerProvider(config)
	if err != nil {
		t.Errorf("Expected no error creating OpenAI tokenizer, got %v", err)
	}
	
	// Test model info
	info := provider.GetModelInfo()
	if info.Provider != "tiktoken" { // OpenAI uses tiktoken internally
		t.Errorf("Expected provider to be 'tiktoken', got %s", info.Provider)
	}
	
	// Test tokenization (uses estimation)
	count, err := provider.CountTokens("Hello, world!")
	if err != nil {
		t.Errorf("Expected no error tokenizing, got %v", err)
	}
	
	if count <= 0 {
		t.Errorf("Expected positive token count, got %d", count)
	}
}

func TestHuggingFaceTokenizerProvider(t *testing.T) {
	provider, err := NewHuggingFaceTokenizerProvider(nil)
	if err != nil {
		t.Errorf("Expected no error creating HuggingFace tokenizer, got %v", err)
	}
	
	// Test model info
	info := provider.GetModelInfo()
	if info.Provider != "huggingface" {
		t.Errorf("Expected provider to be 'huggingface', got %s", info.Provider)
	}
	
	// Test BERT-style tokenization
	testCases := []struct {
		text        string
		expectedMin int
	}{
		{"", 0},     // Special case - empty gets 0
		{"hello", 3}, // [CLS] + hello + [SEP]
		{"hello world", 4}, // [CLS] + hello + world + [SEP]
		{"supercalifragilisticexpialidocious", 5}, // [CLS] + multiple subwords + [SEP]
	}
	
	for _, tc := range testCases {
		count, err := provider.CountTokens(tc.text)
		if err != nil {
			t.Errorf("Expected no error tokenizing '%s', got %v", tc.text, err)
		}
		if tc.text == "" && count != 0 {
			t.Errorf("Expected 0 tokens for empty string, got %d", count)
		} else if tc.text != "" && count < tc.expectedMin {
			t.Errorf("Expected at least %d tokens for '%s', got %d", tc.expectedMin, tc.text, count)
		}
	}
}

func TestTokenizerModelInfo(t *testing.T) {
	testCases := []struct {
		modelName      string
		expectedTokens int
	}{
		{"gpt-4o", 128000},
		{"gpt-4o-mini", 128000},
		{"gpt-4", 8192},
		{"gpt-4-turbo", 8192},
		{"gpt-3.5-turbo", 4096},
		{"unknown-model", 4096}, // Default
	}
	
	for _, tc := range testCases {
		info := getTikTokenModelInfo(tc.modelName)
		if info.MaxTokens != tc.expectedTokens {
			t.Errorf("Expected max tokens %d for model %s, got %d", 
				tc.expectedTokens, tc.modelName, info.MaxTokens)
		}
		if info.Provider != "tiktoken" {
			t.Errorf("Expected provider to be 'tiktoken', got %s", info.Provider)
		}
	}
}

func TestTokenizerFactoryCreateTokenizer(t *testing.T) {
	factory := NewTokenizerFactory()
	
	// Test simple tokenizer
	config := &TokenizerConfig{
		Provider:       "simple",
		ModelName:      "simple",
		RequestTimeout: 10 * time.Second,
	}
	
	tokenizer, err := factory.CreateTokenizer(config)
	if err != nil {
		t.Errorf("Expected no error creating simple tokenizer, got %v", err)
	}
	
	if tokenizer.GetModelInfo().Provider != "simple" {
		t.Errorf("Expected simple provider, got %s", tokenizer.GetModelInfo().Provider)
	}
	
	// Test tiktoken tokenizer
	config.Provider = "tiktoken"
	config.ModelName = "gpt-4"
	
	tokenizer, err = factory.CreateTokenizer(config)
	if err != nil {
		t.Errorf("Expected no error creating tiktoken tokenizer, got %v", err)
	}
	
	if tokenizer.GetModelInfo().Provider != "tiktoken" {
		t.Errorf("Expected tiktoken provider, got %s", tokenizer.GetModelInfo().Provider)
	}
	
	// Test unsupported provider
	config.Provider = "unsupported"
	
	_, err = factory.CreateTokenizer(config)
	if err == nil {
		t.Error("Expected error for unsupported provider")
	}
}