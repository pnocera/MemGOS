package llm

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// LLMFactory provides factory methods for creating LLM instances
type LLMFactory struct {
	providers map[string]func(*LLMConfig) (LLMProvider, error)
	mu        sync.RWMutex
}

// NewLLMFactory creates a new LLM factory
func NewLLMFactory() *LLMFactory {
	factory := &LLMFactory{
		providers: make(map[string]func(*LLMConfig) (LLMProvider, error)),
	}
	
	// Register default providers
	factory.RegisterProvider("openai", NewOpenAILLM)
	factory.RegisterProvider("ollama", NewOllamaLLM)
	factory.RegisterProvider("huggingface", NewHuggingFaceLLM)
	factory.RegisterProvider("hf", NewHuggingFaceLLM) // Alias
	
	return factory
}

// RegisterProvider registers a new LLM provider
func (f *LLMFactory) RegisterProvider(name string, constructor func(*LLMConfig) (LLMProvider, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.providers[strings.ToLower(name)] = constructor
}

// GetProvider returns a provider constructor by name
func (f *LLMFactory) GetProvider(name string) (func(*LLMConfig) (LLMProvider, error), bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	constructor, exists := f.providers[strings.ToLower(name)]
	return constructor, exists
}

// ListProviders returns all registered provider names
func (f *LLMFactory) ListProviders() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	providers := make([]string, 0, len(f.providers))
	for name := range f.providers {
		providers = append(providers, name)
	}
	return providers
}

// CreateLLM creates an LLM instance based on configuration
func (f *LLMFactory) CreateLLM(config *LLMConfig) (LLMProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	constructor, exists := f.GetProvider(config.Provider)
	if !exists {
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}
	
	return constructor(config)
}

// CreateLLMFromBackend creates an LLM instance from backend type
func (f *LLMFactory) CreateLLMFromBackend(backend types.BackendType, model string) (LLMProvider, error) {
	config := &LLMConfig{
		Provider:    string(backend),
		Model:       model,
		MaxTokens:   1024,
		Temperature: 0.7,
		TopP:        0.9,
	}
	
	return f.CreateLLM(config)
}

// CreateLLMFromMap creates an LLM instance from a configuration map
func (f *LLMFactory) CreateLLMFromMap(configMap map[string]interface{}) (LLMProvider, error) {
	config := &LLMConfig{
		Extra: make(map[string]interface{}),
	}
	
	// Extract standard fields
	if provider, ok := configMap["provider"].(string); ok {
		config.Provider = provider
	}
	if model, ok := configMap["model"].(string); ok {
		config.Model = model
	}
	if apiKey, ok := configMap["api_key"].(string); ok {
		config.APIKey = apiKey
	}
	if baseURL, ok := configMap["base_url"].(string); ok {
		config.BaseURL = baseURL
	}
	if maxTokens, ok := configMap["max_tokens"].(int); ok {
		config.MaxTokens = maxTokens
	}
	if temperature, ok := configMap["temperature"].(float64); ok {
		config.Temperature = temperature
	}
	if topP, ok := configMap["top_p"].(float64); ok {
		config.TopP = topP
	}
	
	// Store extra fields
	for key, value := range configMap {
		switch key {
		case "provider", "model", "api_key", "base_url", "max_tokens", "temperature", "top_p":
			// Skip standard fields
		default:
			config.Extra[key] = value
		}
	}
	
	return f.CreateLLM(config)
}

// HealthCheck performs health check on all registered providers
func (f *LLMFactory) HealthCheck(ctx context.Context) map[string]error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	results := make(map[string]error)
	
	for name, constructor := range f.providers {
		// Create a test instance with minimal config
		config := &LLMConfig{
			Provider:    name,
			Model:       "test",
			MaxTokens:   1,
			Temperature: 0.7,
			TopP:        0.9,
		}
		
		llm, err := constructor(config)
		if err != nil {
			results[name] = fmt.Errorf("failed to create instance: %w", err)
			continue
		}
		
		// Perform health check
		if err := llm.HealthCheck(ctx); err != nil {
			results[name] = fmt.Errorf("health check failed: %w", err)
		} else {
			results[name] = nil
		}
		
		// Clean up
		llm.Close()
	}
	
	return results
}

// GetProviderInfo returns information about a provider
func (f *LLMFactory) GetProviderInfo(providerName string) (map[string]interface{}, error) {
	constructor, exists := f.GetProvider(providerName)
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", providerName)
	}
	
	// Create a test instance to get info
	config := &LLMConfig{
		Provider:    providerName,
		Model:       "test",
		MaxTokens:   1,
		Temperature: 0.7,
		TopP:        0.9,
	}
	
	llm, err := constructor(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider instance: %w", err)
	}
	defer llm.Close()
	
	return map[string]interface{}{
		"name":             llm.GetProviderName(),
		"supported_models": llm.GetSupportedModels(),
		"config":           llm.GetConfig(),
	}, nil
}

// Global factory instance
var globalFactory *LLMFactory
var factoryOnce sync.Once

// GetGlobalFactory returns the global LLM factory instance
func GetGlobalFactory() *LLMFactory {
	factoryOnce.Do(func() {
		globalFactory = NewLLMFactory()
	})
	return globalFactory
}

// CreateLLM creates an LLM instance using the global factory
func CreateLLM(config *LLMConfig) (LLMProvider, error) {
	return GetGlobalFactory().CreateLLM(config)
}

// CreateLLMFromBackend creates an LLM instance from backend type using the global factory
func CreateLLMFromBackend(backend types.BackendType, model string) (LLMProvider, error) {
	return GetGlobalFactory().CreateLLMFromBackend(backend, model)
}

// CreateLLMFromMap creates an LLM instance from a configuration map using the global factory
func CreateLLMFromMap(configMap map[string]interface{}) (LLMProvider, error) {
	return GetGlobalFactory().CreateLLMFromMap(configMap)
}

// RegisterProvider registers a provider with the global factory
func RegisterProvider(name string, constructor func(*LLMConfig) (LLMProvider, error)) {
	GetGlobalFactory().RegisterProvider(name, constructor)
}

// ListProviders returns all registered provider names from the global factory
func ListProviders() []string {
	return GetGlobalFactory().ListProviders()
}

// HealthCheckAll performs health check on all providers using the global factory
func HealthCheckAll(ctx context.Context) map[string]error {
	return GetGlobalFactory().HealthCheck(ctx)
}

// GetProviderInfo returns information about a provider using the global factory
func GetProviderInfo(providerName string) (map[string]interface{}, error) {
	return GetGlobalFactory().GetProviderInfo(providerName)
}

// DefaultLLMFromConfig creates a default LLM instance from package config
func DefaultLLMFromConfig(config *LLMConfig) (interfaces.LLM, error) {
	if config == nil {
		config = DefaultLLMConfig()
	}
	
	return CreateLLM(config)
}

// LLMFromConfigMap creates an LLM instance from a configuration map
func LLMFromConfigMap(configMap map[string]interface{}) (interfaces.LLM, error) {
	return CreateLLMFromMap(configMap)
}

// LLMFromBackendType creates an LLM instance from backend type
func LLMFromBackendType(backend types.BackendType, model string) (interfaces.LLM, error) {
	return CreateLLMFromBackend(backend, model)
}

// ValidateProviderConfig validates provider-specific configuration
func ValidateProviderConfig(provider string, config map[string]interface{}) error {
	switch strings.ToLower(provider) {
	case "openai":
		if _, ok := config["api_key"]; !ok {
			return fmt.Errorf("openai provider requires api_key")
		}
	case "ollama":
		if _, ok := config["base_url"]; !ok {
			// Use default if not provided
			config["base_url"] = "http://localhost:11434"
		}
	case "huggingface", "hf":
		if _, ok := config["model"]; !ok {
			return fmt.Errorf("huggingface provider requires model")
		}
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
	
	return nil
}

// ProviderDefaults returns default configuration for each provider
func ProviderDefaults(provider string) map[string]interface{} {
	switch strings.ToLower(provider) {
	case "openai":
		return map[string]interface{}{
			"provider":    "openai",
			"model":       "gpt-3.5-turbo",
			"base_url":    "https://api.openai.com/v1",
			"max_tokens":  1024,
			"temperature": 0.7,
			"top_p":       0.9,
		}
	case "ollama":
		return map[string]interface{}{
			"provider":    "ollama",
			"model":       "llama2",
			"base_url":    "http://localhost:11434",
			"max_tokens":  1024,
			"temperature": 0.7,
			"top_p":       0.9,
		}
	case "huggingface", "hf":
		return map[string]interface{}{
			"provider":    "huggingface",
			"model":       "microsoft/DialoGPT-small",
			"max_tokens":  512,
			"temperature": 0.7,
			"top_p":       0.9,
		}
	default:
		return map[string]interface{}{
			"provider":    provider,
			"max_tokens":  1024,
			"temperature": 0.7,
			"top_p":       0.9,
		}
	}
}

// MergeConfigWithDefaults merges user config with provider defaults
func MergeConfigWithDefaults(provider string, userConfig map[string]interface{}) map[string]interface{} {
	defaults := ProviderDefaults(provider)
	
	// Merge user config over defaults
	for key, value := range userConfig {
		defaults[key] = value
	}
	
	return defaults
}

// NewFromConfig creates an LLM instance from configuration - convenience function for chat package
func NewFromConfig(config *LLMConfig) (interfaces.LLM, error) {
	provider, err := CreateLLM(config)
	if err != nil {
		return nil, err
	}
	
	// Return as interfaces.LLM
	return provider, nil
}