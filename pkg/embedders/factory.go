package embedders

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// EmbedderFactory provides factory methods for creating embedder instances
type EmbedderFactory struct {
	providers map[string]func(*EmbedderConfig) (EmbedderProvider, error)
	models    map[string]*ModelInfo
	mu        sync.RWMutex
}

// ModelInfo contains information about embedding models
type ModelInfo struct {
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	Dimension int    `json:"dimension"`
	MaxLength int    `json:"max_length"`
	Languages []string `json:"languages"`
	TaskTypes []string `json:"task_types"`
}

// NewEmbedderFactory creates a new embedder factory
func NewEmbedderFactory() *EmbedderFactory {
	factory := &EmbedderFactory{
		providers: make(map[string]func(*EmbedderConfig) (EmbedderProvider, error)),
		models:    make(map[string]*ModelInfo),
	}
	
	// Register default providers
	factory.RegisterProvider("sentence-transformer", NewSentenceTransformerEmbedder)
	factory.RegisterProvider("sentence-transformers", NewSentenceTransformerEmbedder) // Alias
	factory.RegisterProvider("st", NewSentenceTransformerEmbedder) // Short alias
	factory.RegisterProvider("openai", NewOpenAIEmbedder)
	factory.RegisterProvider("ollama", NewOllamaEmbedder)
	
	// Register default models
	factory.registerDefaultModels()
	
	return factory
}

// registerDefaultModels registers information about popular embedding models
func (f *EmbedderFactory) registerDefaultModels() {
	defaultModels := []*ModelInfo{
		// Sentence Transformer models
		{
			Name: "all-MiniLM-L6-v2", Provider: "sentence-transformer", Dimension: 384, MaxLength: 512,
			Languages: []string{"en"}, TaskTypes: []string{"semantic-similarity", "classification"},
		},
		{
			Name: "all-mpnet-base-v2", Provider: "sentence-transformer", Dimension: 768, MaxLength: 512,
			Languages: []string{"en"}, TaskTypes: []string{"semantic-similarity", "classification"},
		},
		{
			Name: "all-MiniLM-L12-v2", Provider: "sentence-transformer", Dimension: 384, MaxLength: 512,
			Languages: []string{"en"}, TaskTypes: []string{"semantic-similarity", "classification"},
		},
		{
			Name: "paraphrase-multilingual-MiniLM-L12-v2", Provider: "sentence-transformer", Dimension: 384, MaxLength: 512,
			Languages: []string{"multi"}, TaskTypes: []string{"semantic-similarity", "paraphrase"},
		},
		{
			Name: "multi-qa-MiniLM-L6-cos-v1", Provider: "sentence-transformer", Dimension: 384, MaxLength: 512,
			Languages: []string{"en"}, TaskTypes: []string{"question-answering", "retrieval"},
		},
		{
			Name: "distiluse-base-multilingual-cased", Provider: "sentence-transformer", Dimension: 512, MaxLength: 512,
			Languages: []string{"multi"}, TaskTypes: []string{"semantic-similarity"},
		},
		// OpenAI models
		{
			Name: "text-embedding-ada-002", Provider: "openai", Dimension: 1536, MaxLength: 8191,
			Languages: []string{"multi"}, TaskTypes: []string{"semantic-similarity", "classification", "clustering"},
		},
		{
			Name: "text-embedding-3-small", Provider: "openai", Dimension: 1536, MaxLength: 8191,
			Languages: []string{"multi"}, TaskTypes: []string{"semantic-similarity", "classification", "clustering"},
		},
		{
			Name: "text-embedding-3-large", Provider: "openai", Dimension: 3072, MaxLength: 8191,
			Languages: []string{"multi"}, TaskTypes: []string{"semantic-similarity", "classification", "clustering"},
		},
		// Ollama models (common ones)
		{
			Name: "nomic-embed-text", Provider: "ollama", Dimension: 768, MaxLength: 2048,
			Languages: []string{"en"}, TaskTypes: []string{"semantic-similarity", "retrieval"},
		},
		{
			Name: "mxbai-embed-large", Provider: "ollama", Dimension: 1024, MaxLength: 512,
			Languages: []string{"en"}, TaskTypes: []string{"semantic-similarity", "retrieval"},
		},
		{
			Name: "all-minilm", Provider: "ollama", Dimension: 384, MaxLength: 512,
			Languages: []string{"en"}, TaskTypes: []string{"semantic-similarity"},
		},
	}
	
	for _, model := range defaultModels {
		f.models[model.Name] = model
	}
}

// RegisterProvider registers a new embedder provider
func (f *EmbedderFactory) RegisterProvider(name string, constructor func(*EmbedderConfig) (EmbedderProvider, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.providers[strings.ToLower(name)] = constructor
}

// RegisterModel registers information about an embedding model
func (f *EmbedderFactory) RegisterModel(model *ModelInfo) {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.models[model.Name] = model
}

// GetProvider returns a provider constructor by name
func (f *EmbedderFactory) GetProvider(name string) (func(*EmbedderConfig) (EmbedderProvider, error), bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	constructor, exists := f.providers[strings.ToLower(name)]
	return constructor, exists
}

// GetModelInfo returns information about a model
func (f *EmbedderFactory) GetModelInfo(modelName string) (*ModelInfo, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	info, exists := f.models[modelName]
	return info, exists
}

// ListProviders returns all registered provider names
func (f *EmbedderFactory) ListProviders() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	providers := make([]string, 0, len(f.providers))
	for name := range f.providers {
		providers = append(providers, name)
	}
	return providers
}

// ListModels returns all registered model names
func (f *EmbedderFactory) ListModels() []*ModelInfo {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	models := make([]*ModelInfo, 0, len(f.models))
	for _, model := range f.models {
		models = append(models, model)
	}
	return models
}

// ListModelsByProvider returns models for a specific provider
func (f *EmbedderFactory) ListModelsByProvider(provider string) []*ModelInfo {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	var models []*ModelInfo
	for _, model := range f.models {
		if strings.EqualFold(model.Provider, provider) {
			models = append(models, model)
		}
	}
	return models
}

// CreateEmbedder creates an embedder instance based on configuration
func (f *EmbedderFactory) CreateEmbedder(config *EmbedderConfig) (EmbedderProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	// Auto-detect provider from model if not specified
	if config.Provider == "" {
		if modelInfo, exists := f.GetModelInfo(config.Model); exists {
			config.Provider = modelInfo.Provider
		} else {
			return nil, fmt.Errorf("provider not specified and model %s not found in registry", config.Model)
		}
	}
	
	// Auto-detect dimension from model if not specified
	if config.Dimension == 0 {
		if modelInfo, exists := f.GetModelInfo(config.Model); exists {
			config.Dimension = modelInfo.Dimension
		} else {
			return nil, fmt.Errorf("dimension not specified and model %s not found in registry", config.Model)
		}
	}
	
	constructor, exists := f.GetProvider(config.Provider)
	if !exists {
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}
	
	return constructor(config)
}

// CreateEmbedderFromBackend creates an embedder instance from backend type
func (f *EmbedderFactory) CreateEmbedderFromBackend(backend types.BackendType, model string) (EmbedderProvider, error) {
	var provider string
	switch backend {
	case types.BackendOpenAI:
		provider = "openai"
	case types.BackendOllama:
		provider = "ollama"
	case types.BackendHuggingFace:
		provider = "sentence-transformer"
	default:
		return nil, fmt.Errorf("unsupported backend for embeddings: %s", backend)
	}
	
	config := &EmbedderConfig{
		Provider:  provider,
		Model:     model,
		Dimension: 384, // Default dimension
		MaxLength: 512,
		BatchSize: 32,
		Normalize: true,
	}
	
	return f.CreateEmbedder(config)
}

// CreateEmbedderFromMap creates an embedder instance from a configuration map
func (f *EmbedderFactory) CreateEmbedderFromMap(configMap map[string]interface{}) (EmbedderProvider, error) {
	config := &EmbedderConfig{
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
	if dimension, ok := configMap["dimension"].(int); ok {
		config.Dimension = dimension
	}
	if maxLength, ok := configMap["max_length"].(int); ok {
		config.MaxLength = maxLength
	}
	if batchSize, ok := configMap["batch_size"].(int); ok {
		config.BatchSize = batchSize
	}
	if normalize, ok := configMap["normalize"].(bool); ok {
		config.Normalize = normalize
	}
	
	// Store extra fields
	for key, value := range configMap {
		switch key {
		case "provider", "model", "api_key", "base_url", "dimension", "max_length", "batch_size", "normalize":
			// Skip standard fields
		default:
			config.Extra[key] = value
		}
	}
	
	return f.CreateEmbedder(config)
}

// AutoDetectModel attempts to auto-detect the best model for a given task
func (f *EmbedderFactory) AutoDetectModel(taskType string, language string, preferredProvider string) (*ModelInfo, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	var candidates []*ModelInfo
	
	// Filter models by criteria
	for _, model := range f.models {
		// Filter by provider if specified
		if preferredProvider != "" && !strings.EqualFold(model.Provider, preferredProvider) {
			continue
		}
		
		// Filter by task type if specified
		if taskType != "" {
			found := false
			for _, task := range model.TaskTypes {
				if strings.Contains(strings.ToLower(task), strings.ToLower(taskType)) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		// Filter by language if specified
		if language != "" && language != "en" {
			found := false
			for _, lang := range model.Languages {
				if lang == "multi" || strings.EqualFold(lang, language) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		candidates = append(candidates, model)
	}
	
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no suitable model found for task:%s, language:%s, provider:%s", 
			taskType, language, preferredProvider)
	}
	
	// Return the first candidate (could implement more sophisticated selection)
	return candidates[0], nil
}

// HealthCheck performs health check on all registered providers
func (f *EmbedderFactory) HealthCheck(ctx context.Context) map[string]error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	results := make(map[string]error)
	
	for name, constructor := range f.providers {
		// Create a test instance with minimal config
		config := &EmbedderConfig{
			Provider:  name,
			Model:     "test",
			Dimension: 384,
			MaxLength: 512,
			BatchSize: 1,
			Normalize: true,
		}
		
		embedder, err := constructor(config)
		if err != nil {
			results[name] = fmt.Errorf("failed to create instance: %w", err)
			continue
		}
		
		// Perform health check
		if err := embedder.HealthCheck(ctx); err != nil {
			results[name] = fmt.Errorf("health check failed: %w", err)
		} else {
			results[name] = nil
		}
		
		// Clean up
		embedder.Close()
	}
	
	return results
}

// GetProviderInfo returns information about a provider
func (f *EmbedderFactory) GetProviderInfo(providerName string) (map[string]interface{}, error) {
	constructor, exists := f.GetProvider(providerName)
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", providerName)
	}
	
	// Create a test instance to get info
	config := &EmbedderConfig{
		Provider:  providerName,
		Model:     "test",
		Dimension: 384,
		MaxLength: 512,
		BatchSize: 1,
		Normalize: true,
	}
	
	embedder, err := constructor(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider instance: %w", err)
	}
	defer embedder.Close()
	
	models := f.ListModelsByProvider(providerName)
	
	return map[string]interface{}{
		"name":             embedder.GetProviderName(),
		"supported_models": embedder.GetSupportedModels(),
		"registered_models": models,
		"config":           embedder.GetConfig(),
	}, nil
}

// ValidateProviderConfig validates provider-specific configuration
func (f *EmbedderFactory) ValidateProviderConfig(provider string, config map[string]interface{}) error {
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
	case "sentence-transformer", "sentence-transformers", "st":
		if _, ok := config["model"]; !ok {
			return fmt.Errorf("sentence-transformer provider requires model")
		}
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
	
	return nil
}

// ProviderDefaults returns default configuration for each provider
func (f *EmbedderFactory) ProviderDefaults(provider string) map[string]interface{} {
	switch strings.ToLower(provider) {
	case "openai":
		return map[string]interface{}{
			"provider":   "openai",
			"model":      "text-embedding-ada-002",
			"dimension":  1536,
			"max_length": 8191,
			"batch_size": 100,
			"normalize":  true,
		}
	case "ollama":
		return map[string]interface{}{
			"provider":   "ollama",
			"model":      "nomic-embed-text",
			"base_url":   "http://localhost:11434",
			"dimension":  768,
			"max_length": 2048,
			"batch_size": 32,
			"normalize":  true,
		}
	case "sentence-transformer", "sentence-transformers", "st":
		return map[string]interface{}{
			"provider":   "sentence-transformer",
			"model":      "all-MiniLM-L6-v2",
			"dimension":  384,
			"max_length": 512,
			"batch_size": 32,
			"normalize":  true,
		}
	default:
		return map[string]interface{}{
			"provider":   provider,
			"dimension":  384,
			"max_length": 512,
			"batch_size": 32,
			"normalize":  true,
		}
	}
}

// MergeConfigWithDefaults merges user config with provider defaults
func (f *EmbedderFactory) MergeConfigWithDefaults(provider string, userConfig map[string]interface{}) map[string]interface{} {
	defaults := f.ProviderDefaults(provider)
	
	// Merge user config over defaults
	for key, value := range userConfig {
		defaults[key] = value
	}
	
	return defaults
}

// Global factory instance
var globalFactory *EmbedderFactory
var factoryOnce sync.Once

// GetGlobalFactory returns the global embedder factory instance
func GetGlobalFactory() *EmbedderFactory {
	factoryOnce.Do(func() {
		globalFactory = NewEmbedderFactory()
	})
	return globalFactory
}

// CreateEmbedder creates an embedder instance using the global factory
func CreateEmbedder(config *EmbedderConfig) (EmbedderProvider, error) {
	return GetGlobalFactory().CreateEmbedder(config)
}

// CreateEmbedderFromBackend creates an embedder instance from backend type using the global factory
func CreateEmbedderFromBackend(backend types.BackendType, model string) (EmbedderProvider, error) {
	return GetGlobalFactory().CreateEmbedderFromBackend(backend, model)
}

// CreateEmbedderFromMap creates an embedder instance from a configuration map using the global factory
func CreateEmbedderFromMap(configMap map[string]interface{}) (EmbedderProvider, error) {
	return GetGlobalFactory().CreateEmbedderFromMap(configMap)
}

// RegisterProvider registers a provider with the global factory
func RegisterProvider(name string, constructor func(*EmbedderConfig) (EmbedderProvider, error)) {
	GetGlobalFactory().RegisterProvider(name, constructor)
}

// RegisterModel registers a model with the global factory
func RegisterModel(model *ModelInfo) {
	GetGlobalFactory().RegisterModel(model)
}

// ListProviders returns all registered provider names from the global factory
func ListProviders() []string {
	return GetGlobalFactory().ListProviders()
}

// ListModels returns all registered models from the global factory
func ListModels() []*ModelInfo {
	return GetGlobalFactory().ListModels()
}

// AutoDetectModel auto-detects the best model using the global factory
func AutoDetectModel(taskType string, language string, preferredProvider string) (*ModelInfo, error) {
	return GetGlobalFactory().AutoDetectModel(taskType, language, preferredProvider)
}

// HealthCheckAll performs health check on all providers using the global factory
func HealthCheckAll(ctx context.Context) map[string]error {
	return GetGlobalFactory().HealthCheck(ctx)
}

// GetProviderInfo returns information about a provider using the global factory
func GetProviderInfo(providerName string) (map[string]interface{}, error) {
	return GetGlobalFactory().GetProviderInfo(providerName)
}

// DefaultEmbedderFromConfig creates a default embedder instance from package config
func DefaultEmbedderFromConfig(config *EmbedderConfig) (interfaces.Embedder, error) {
	if config == nil {
		config = DefaultEmbedderConfig()
	}
	
	return CreateEmbedder(config)
}

// EmbedderFromConfigMap creates an embedder instance from a configuration map
func EmbedderFromConfigMap(configMap map[string]interface{}) (interfaces.Embedder, error) {
	return CreateEmbedderFromMap(configMap)
}

// EmbedderFromBackendType creates an embedder instance from backend type
func EmbedderFromBackendType(backend types.BackendType, model string) (interfaces.Embedder, error) {
	return CreateEmbedderFromBackend(backend, model)
}