package embedders

import (
	"context"
	"testing"

	"github.com/memtensor/memgos/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmbedderFactory(t *testing.T) {
	factory := NewEmbedderFactory()
	
	// Check that default providers are registered
	providers := factory.ListProviders()
	assert.Contains(t, providers, "sentence-transformer")
	assert.Contains(t, providers, "openai")
	assert.Contains(t, providers, "ollama")
	
	// Check that default models are registered
	models := factory.ListModels()
	assert.Greater(t, len(models), 0)
	
	// Check specific models
	modelNames := make([]string, len(models))
	for i, model := range models {
		modelNames[i] = model.Name
	}
	assert.Contains(t, modelNames, "all-MiniLM-L6-v2")
	assert.Contains(t, modelNames, "text-embedding-ada-002")
}

func TestEmbedderFactory_RegisterProvider(t *testing.T) {
	factory := NewEmbedderFactory()
	
	// Register a test provider
	factory.RegisterProvider("test-provider", func(config *EmbedderConfig) (EmbedderProvider, error) {
		mock := NewMockEmbedder()
		return mock, nil
	})
	
	providers := factory.ListProviders()
	assert.Contains(t, providers, "test-provider")
	
	// Test getting the provider
	constructor, exists := factory.GetProvider("test-provider")
	assert.True(t, exists)
	assert.NotNil(t, constructor)
}

func TestEmbedderFactory_RegisterModel(t *testing.T) {
	factory := NewEmbedderFactory()
	
	testModel := &ModelInfo{
		Name:      "test-model",
		Provider:  "test-provider",
		Dimension: 512,
		MaxLength: 1024,
		Languages: []string{"en"},
		TaskTypes: []string{"test"},
	}
	
	factory.RegisterModel(testModel)
	
	// Check that model is registered
	model, exists := factory.GetModelInfo("test-model")
	assert.True(t, exists)
	assert.Equal(t, testModel.Name, model.Name)
	assert.Equal(t, testModel.Provider, model.Provider)
	assert.Equal(t, testModel.Dimension, model.Dimension)
}

func TestEmbedderFactory_ListModelsByProvider(t *testing.T) {
	factory := NewEmbedderFactory()
	
	// Get OpenAI models
	openaiModels := factory.ListModelsByProvider("openai")
	assert.Greater(t, len(openaiModels), 0)
	
	for _, model := range openaiModels {
		assert.Equal(t, "openai", model.Provider)
	}
	
	// Get sentence-transformer models
	stModels := factory.ListModelsByProvider("sentence-transformer")
	assert.Greater(t, len(stModels), 0)
	
	for _, model := range stModels {
		assert.Equal(t, "sentence-transformer", model.Provider)
	}
}

func TestEmbedderFactory_CreateEmbedder(t *testing.T) {
	factory := NewEmbedderFactory()
	
	// Register mock provider for testing
	factory.RegisterProvider("mock", func(config *EmbedderConfig) (EmbedderProvider, error) {
		mock := NewMockEmbedder()
		return mock, nil
	})
	
	config := &EmbedderConfig{
		Provider:  "mock",
		Model:     "mock-model",
		Dimension: 384,
		MaxLength: 512,
		BatchSize: 32,
		Normalize: true,
	}
	
	embedder, err := factory.CreateEmbedder(config)
	require.NoError(t, err)
	assert.NotNil(t, embedder)
}

func TestEmbedderFactory_CreateEmbedderFromBackend(t *testing.T) {
	factory := NewEmbedderFactory()
	
	// Register mock providers
	factory.RegisterProvider("openai", func(config *EmbedderConfig) (EmbedderProvider, error) {
		mock := NewMockEmbedder()
		return mock, nil
	})
	
	factory.RegisterProvider("sentence-transformer", func(config *EmbedderConfig) (EmbedderProvider, error) {
		mock := NewMockEmbedder()
		return mock, nil
	})
	
	tests := []struct {
		backend  types.BackendType
		model    string
		wantErr  bool
	}{
		{types.BackendOpenAI, "text-embedding-ada-002", false},
		{types.BackendHuggingFace, "all-MiniLM-L6-v2", false},
		{types.BackendRedis, "unsupported", true}, // Unsupported backend
	}
	
	for _, tt := range tests {
		t.Run(string(tt.backend), func(t *testing.T) {
			embedder, err := factory.CreateEmbedderFromBackend(tt.backend, tt.model)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, embedder)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, embedder)
			}
		})
	}
}

func TestEmbedderFactory_CreateEmbedderFromMap(t *testing.T) {
	factory := NewEmbedderFactory()
	
	// Register mock provider
	factory.RegisterProvider("mock", func(config *EmbedderConfig) (EmbedderProvider, error) {
		mock := NewMockEmbedder()
		mock.SetConfig(config)
		return mock, nil
	})
	
	configMap := map[string]interface{}{
		"provider":   "mock",
		"model":      "mock-model",
		"dimension":  384,
		"max_length": 512,
		"batch_size": 32,
		"normalize":  true,
		"extra_field": "extra_value",
	}
	
	embedder, err := factory.CreateEmbedderFromMap(configMap)
	require.NoError(t, err)
	assert.NotNil(t, embedder)
	
	// Check that extra fields are preserved
	config := embedder.GetConfig()
	if config.Extra != nil {
		assert.Equal(t, "extra_value", config.Extra["extra_field"])
	}
}

func TestEmbedderFactory_AutoDetectModel(t *testing.T) {
	factory := NewEmbedderFactory()
	
	tests := []struct {
		name             string
		taskType         string
		language         string
		preferredProvider string
		expectModel      bool
	}{
		{
			name:        "semantic similarity task",
			taskType:    "semantic-similarity",
			language:    "en",
			expectModel: true,
		},
		{
			name:        "multilingual task",
			taskType:    "",
			language:    "fr",
			expectModel: true,
		},
		{
			name:             "openai provider preference",
			taskType:         "",
			language:         "",
			preferredProvider: "openai",
			expectModel:      true,
		},
		{
			name:             "unsupported provider",
			taskType:         "",
			language:         "",
			preferredProvider: "unsupported",
			expectModel:      false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := factory.AutoDetectModel(tt.taskType, tt.language, tt.preferredProvider)
			if tt.expectModel {
				assert.NoError(t, err)
				assert.NotNil(t, model)
				
				if tt.preferredProvider != "" {
					assert.Equal(t, tt.preferredProvider, model.Provider)
				}
			} else {
				assert.Error(t, err)
				assert.Nil(t, model)
			}
		})
	}
}

func TestEmbedderFactory_ValidateProviderConfig(t *testing.T) {
	factory := NewEmbedderFactory()
	
	tests := []struct {
		name     string
		provider string
		config   map[string]interface{}
		wantErr  bool
	}{
		{
			name:     "valid openai config",
			provider: "openai",
			config:   map[string]interface{}{"api_key": "test-key"},
			wantErr:  false,
		},
		{
			name:     "invalid openai config",
			provider: "openai",
			config:   map[string]interface{}{},
			wantErr:  true,
		},
		{
			name:     "valid ollama config",
			provider: "ollama",
			config:   map[string]interface{}{"base_url": "http://localhost:11434"},
			wantErr:  false,
		},
		{
			name:     "ollama config with defaults",
			provider: "ollama",
			config:   map[string]interface{}{},
			wantErr:  false,
		},
		{
			name:     "valid sentence-transformer config",
			provider: "sentence-transformer",
			config:   map[string]interface{}{"model": "all-MiniLM-L6-v2"},
			wantErr:  false,
		},
		{
			name:     "invalid sentence-transformer config",
			provider: "sentence-transformer",
			config:   map[string]interface{}{},
			wantErr:  true,
		},
		{
			name:     "unsupported provider",
			provider: "unsupported",
			config:   map[string]interface{}{},
			wantErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := factory.ValidateProviderConfig(tt.provider, tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEmbedderFactory_ProviderDefaults(t *testing.T) {
	factory := NewEmbedderFactory()
	
	// Test OpenAI defaults
	openaiDefaults := factory.ProviderDefaults("openai")
	assert.Equal(t, "openai", openaiDefaults["provider"])
	assert.Equal(t, "text-embedding-ada-002", openaiDefaults["model"])
	assert.Equal(t, 1536, openaiDefaults["dimension"])
	
	// Test Ollama defaults
	ollamaDefaults := factory.ProviderDefaults("ollama")
	assert.Equal(t, "ollama", ollamaDefaults["provider"])
	assert.Equal(t, "nomic-embed-text", ollamaDefaults["model"])
	assert.Equal(t, "http://localhost:11434", ollamaDefaults["base_url"])
	
	// Test sentence-transformer defaults
	stDefaults := factory.ProviderDefaults("sentence-transformer")
	assert.Equal(t, "sentence-transformer", stDefaults["provider"])
	assert.Equal(t, "all-MiniLM-L6-v2", stDefaults["model"])
	assert.Equal(t, 384, stDefaults["dimension"])
}

func TestEmbedderFactory_MergeConfigWithDefaults(t *testing.T) {
	factory := NewEmbedderFactory()
	
	userConfig := map[string]interface{}{
		"model":      "custom-model",
		"dimension":  512,
		"custom_key": "custom_value",
	}
	
	merged := factory.MergeConfigWithDefaults("openai", userConfig)
	
	// User config should override defaults
	assert.Equal(t, "custom-model", merged["model"])
	assert.Equal(t, 512, merged["dimension"])
	assert.Equal(t, "custom_value", merged["custom_key"])
	
	// Defaults should be preserved where not overridden
	assert.Equal(t, "openai", merged["provider"])
	assert.Equal(t, 100, merged["batch_size"])
}

func TestGlobalFactory(t *testing.T) {
	// Test that global factory functions work
	factory1 := GetGlobalFactory()
	factory2 := GetGlobalFactory()
	
	// Should return the same instance
	assert.Same(t, factory1, factory2)
	
	// Test global functions
	providers := ListProviders()
	assert.Greater(t, len(providers), 0)
	
	models := ListModels()
	assert.Greater(t, len(models), 0)
	
	// Register a test provider globally
	RegisterProvider("global-test", func(config *EmbedderConfig) (EmbedderProvider, error) {
		return NewMockEmbedder(), nil
	})
	
	providers = ListProviders()
	assert.Contains(t, providers, "global-test")
}


func TestEmbedderFactory_HealthCheck(t *testing.T) {
	factory := NewEmbedderFactory()
	
	// Register mock provider
	factory.RegisterProvider("mock", func(config *EmbedderConfig) (EmbedderProvider, error) {
		mock := NewMockEmbedder()
		return mock, nil
	})
	
	ctx := context.Background()
	results := factory.HealthCheck(ctx)
	
	// Should have results for all providers
	assert.Contains(t, results, "mock")
	
	// Mock provider should be healthy
	assert.NoError(t, results["mock"])
}

func TestEmbedderFactory_GetProviderInfo(t *testing.T) {
	factory := NewEmbedderFactory()
	
	// Register mock provider
	factory.RegisterProvider("mock", func(config *EmbedderConfig) (EmbedderProvider, error) {
		mock := NewMockEmbedder()
		return mock, nil
	})
	
	info, err := factory.GetProviderInfo("mock")
	require.NoError(t, err)
	
	assert.Equal(t, "mock", info["name"])
	assert.Contains(t, info, "supported_models")
	assert.Contains(t, info, "config")
}

// Benchmark tests
func BenchmarkEmbedderFactory_CreateEmbedder(b *testing.B) {
	factory := NewEmbedderFactory()
	
	// Register mock provider
	factory.RegisterProvider("mock", func(config *EmbedderConfig) (EmbedderProvider, error) {
		mock := NewMockEmbedder()
		return mock, nil
	})
	
	config := &EmbedderConfig{
		Provider:  "mock",
		Model:     "mock-model",
		Dimension: 384,
		MaxLength: 512,
		BatchSize: 32,
		Normalize: true,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		embedder, _ := factory.CreateEmbedder(config)
		embedder.Close()
	}
}

func BenchmarkEmbedderFactory_AutoDetectModel(b *testing.B) {
	factory := NewEmbedderFactory()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		factory.AutoDetectModel("semantic-similarity", "en", "")
	}
}