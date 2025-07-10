package llm

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/memtensor/memgos/pkg/types"
)

func TestLLMFactory(t *testing.T) {
	factory := NewLLMFactory()
	
	t.Run("ListProviders", func(t *testing.T) {
		providers := factory.ListProviders()
		assert.Contains(t, providers, "openai")
		assert.Contains(t, providers, "ollama")
		assert.Contains(t, providers, "huggingface")
	})
	
	t.Run("CreateOpenAILLM", func(t *testing.T) {
		config := &LLMConfig{
			Provider:    "openai",
			Model:       "gpt-3.5-turbo",
			APIKey:      "test-key",
			MaxTokens:   1024,
			Temperature: 0.7,
			TopP:        0.9,
			Timeout:     30 * time.Second,
		}
		
		llm, err := factory.CreateLLM(config)
		require.NoError(t, err)
		assert.NotNil(t, llm)
		assert.Equal(t, "openai", llm.GetProviderName())
		assert.Equal(t, "gpt-3.5-turbo", llm.(*OpenAILLM).GetModelName())
		
		llm.Close()
	})
	
	t.Run("CreateOllamaLLM", func(t *testing.T) {
		config := &LLMConfig{
			Provider:    "ollama",
			Model:       "llama2",
			BaseURL:     "http://localhost:11434",
			MaxTokens:   1024,
			Temperature: 0.7,
			TopP:        0.9,
			Timeout:     30 * time.Second,
		}
		
		llm, err := factory.CreateLLM(config)
		require.NoError(t, err)
		assert.NotNil(t, llm)
		assert.Equal(t, "ollama", llm.GetProviderName())
		assert.Equal(t, "llama2", llm.(*OllamaLLM).GetModelName())
		
		llm.Close()
	})
	
	t.Run("CreateHuggingFaceLLM", func(t *testing.T) {
		config := &LLMConfig{
			Provider:    "huggingface",
			Model:       "microsoft/DialoGPT-small",
			APIKey:      "test-key",
			MaxTokens:   512,
			Temperature: 0.7,
			TopP:        0.9,
			Timeout:     30 * time.Second,
		}
		
		llm, err := factory.CreateLLM(config)
		require.NoError(t, err)
		assert.NotNil(t, llm)
		assert.Equal(t, "huggingface", llm.GetProviderName())
		assert.Equal(t, "microsoft/DialoGPT-small", llm.(*HuggingFaceLLM).GetModelName())
		
		llm.Close()
	})
	
	t.Run("CreateLLMFromBackend", func(t *testing.T) {
		llm, err := factory.CreateLLMFromBackend(types.BackendOpenAI, "gpt-3.5-turbo")
		require.Error(t, err) // Should fail without API key
		assert.Nil(t, llm)
		
		llm, err = factory.CreateLLMFromBackend(types.BackendOllama, "llama2")
		require.NoError(t, err)
		assert.NotNil(t, llm)
		llm.Close()
	})
	
	t.Run("CreateLLMFromMap", func(t *testing.T) {
		configMap := map[string]interface{}{
			"provider":    "ollama",
			"model":       "llama2",
			"base_url":    "http://localhost:11434",
			"max_tokens":  1024,
			"temperature": 0.7,
			"top_p":       0.9,
		}
		
		llm, err := factory.CreateLLMFromMap(configMap)
		require.NoError(t, err)
		assert.NotNil(t, llm)
		assert.Equal(t, "ollama", llm.GetProviderName())
		
		llm.Close()
	})
	
	t.Run("InvalidProvider", func(t *testing.T) {
		config := &LLMConfig{
			Provider: "invalid",
			Model:    "test",
		}
		
		llm, err := factory.CreateLLM(config)
		require.Error(t, err)
		assert.Nil(t, llm)
		assert.Contains(t, err.Error(), "unsupported provider")
	})
	
	t.Run("InvalidConfig", func(t *testing.T) {
		config := &LLMConfig{
			Provider:    "openai",
			Model:       "", // Missing model
			Temperature: -1, // Invalid temperature
		}
		
		llm, err := factory.CreateLLM(config)
		require.Error(t, err)
		assert.Nil(t, llm)
	})
	
	t.Run("GetProviderInfo", func(t *testing.T) {
		info, err := factory.GetProviderInfo("ollama")
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, "ollama", info["name"])
		assert.NotNil(t, info["supported_models"])
	})
	
	t.Run("RegisterCustomProvider", func(t *testing.T) {
		// Register a custom provider
		factory.RegisterProvider("custom", func(config *LLMConfig) (LLMProvider, error) {
			return NewOllamaLLM(config) // Use Ollama as base for testing
		})
		
		providers := factory.ListProviders()
		assert.Contains(t, providers, "custom")
		
		config := &LLMConfig{
			Provider: "custom",
			Model:    "test-model",
		}
		
		llm, err := factory.CreateLLM(config)
		require.NoError(t, err)
		assert.NotNil(t, llm)
		llm.Close()
	})
}

func TestFactoryLLMConfig(t *testing.T) {
	t.Run("ValidateValid", func(t *testing.T) {
		config := &LLMConfig{
			Provider:    "openai",
			Model:       "gpt-3.5-turbo",
			MaxTokens:   1024,
			Temperature: 0.7,
			TopP:        0.9,
		}
		
		err := config.Validate()
		assert.NoError(t, err)
	})
	
	t.Run("ValidateInvalid", func(t *testing.T) {
		// Missing provider
		config := &LLMConfig{
			Model: "gpt-3.5-turbo",
		}
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "provider is required")
		
		// Missing model
		config = &LLMConfig{
			Provider: "openai",
		}
		err = config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "model is required")
		
		// Invalid temperature
		config = &LLMConfig{
			Provider:    "openai",
			Model:       "gpt-3.5-turbo",
			Temperature: -1,
		}
		err = config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "temperature must be between")
		
		// Invalid top_p
		config = &LLMConfig{
			Provider: "openai",
			Model:    "gpt-3.5-turbo",
			TopP:     1.5,
		}
		err = config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "top_p must be between")
	})
}

func TestGlobalFactory(t *testing.T) {
	t.Run("GetGlobalFactory", func(t *testing.T) {
		factory1 := GetGlobalFactory()
		factory2 := GetGlobalFactory()
		assert.Same(t, factory1, factory2) // Should be the same instance
	})
	
	t.Run("GlobalFunctions", func(t *testing.T) {
		providers := ListProviders()
		assert.Contains(t, providers, "openai")
		assert.Contains(t, providers, "ollama")
		assert.Contains(t, providers, "huggingface")
		
		// Test provider defaults
		defaults := ProviderDefaults("openai")
		assert.Equal(t, "openai", defaults["provider"])
		assert.Equal(t, "gpt-3.5-turbo", defaults["model"])
		
		defaults = ProviderDefaults("ollama")
		assert.Equal(t, "ollama", defaults["provider"])
		assert.Equal(t, "llama2", defaults["model"])
		
		defaults = ProviderDefaults("huggingface")
		assert.Equal(t, "huggingface", defaults["provider"])
		assert.Equal(t, "microsoft/DialoGPT-small", defaults["model"])
	})
	
	t.Run("ValidateProviderConfig", func(t *testing.T) {
		// OpenAI requires API key
		config := map[string]interface{}{
			"model": "gpt-3.5-turbo",
		}
		err := ValidateProviderConfig("openai", config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "api_key")
		
		config["api_key"] = "test-key"
		err = ValidateProviderConfig("openai", config)
		assert.NoError(t, err)
		
		// Ollama should set default base_url
		config = map[string]interface{}{
			"model": "llama2",
		}
		err = ValidateProviderConfig("ollama", config)
		assert.NoError(t, err)
		assert.Equal(t, "http://localhost:11434", config["base_url"])
		
		// HuggingFace requires model
		config = map[string]interface{}{}
		err = ValidateProviderConfig("huggingface", config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "model")
		
		config["model"] = "test-model"
		err = ValidateProviderConfig("huggingface", config)
		assert.NoError(t, err)
	})
	
	t.Run("MergeConfigWithDefaults", func(t *testing.T) {
		userConfig := map[string]interface{}{
			"model":       "custom-model",
			"temperature": 0.5,
		}
		
		merged := MergeConfigWithDefaults("openai", userConfig)
		assert.Equal(t, "custom-model", merged["model"])
		assert.Equal(t, 0.5, merged["temperature"])
		assert.Equal(t, "openai", merged["provider"])
		assert.Equal(t, 1024, merged["max_tokens"]) // From defaults
	})
}

func TestDefaultLLMConfig(t *testing.T) {
	config := DefaultLLMConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "openai", config.Provider)
	assert.Equal(t, "gpt-3.5-turbo", config.Model)
	assert.Equal(t, 1024, config.MaxTokens)
	assert.Equal(t, 0.7, config.Temperature)
	assert.Equal(t, 0.9, config.TopP)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.NotNil(t, config.Extra)
}

func TestHealthCheck(t *testing.T) {
	// This test requires actual services running, so skip in CI
	if testing.Short() {
		t.Skip("Skipping health check test in short mode")
	}
	
	factory := NewLLMFactory()
	ctx := context.Background()
	
	results := factory.HealthCheck(ctx)
	assert.NotNil(t, results)
	
	// Check that all providers were tested
	assert.Contains(t, results, "openai")
	assert.Contains(t, results, "ollama")
	assert.Contains(t, results, "huggingface")
	
	// Most should fail without proper configuration
	for provider, err := range results {
		t.Logf("Provider %s health check: %v", provider, err)
	}
}