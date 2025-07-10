package llm_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/memtensor/memgos/pkg/llm"
	"github.com/memtensor/memgos/pkg/types"
)

// ExampleLLMFactory demonstrates how to use the LLM factory
func ExampleLLMFactory() {
	// Create a new LLM factory
	factory := llm.NewLLMFactory()
	
	// List available providers
	providers := factory.ListProviders()
	// Filter out aliases for consistent output
	mainProviders := []string{}
	for _, p := range providers {
		if p != "hf" { // Skip alias
			mainProviders = append(mainProviders, p)
		}
	}
	fmt.Printf("Available providers: %v\n", mainProviders)
	
	// Create an OpenAI LLM instance
	config := &llm.LLMConfig{
		Provider:    "openai",
		Model:       "gpt-3.5-turbo",
		APIKey:      "your-api-key",
		MaxTokens:   1024,
		Temperature: 0.7,
		TopP:        0.9,
		Timeout:     30 * time.Second,
	}
	
	llmInstance, err := factory.CreateLLM(config)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}
	defer llmInstance.Close()
	
	fmt.Printf("Created LLM provider: %s\n", llmInstance.GetProviderName())
	// Output: Available providers: [openai ollama huggingface]
	// Created LLM provider: openai
}

// ExampleOpenAILLM demonstrates how to use OpenAI LLM
func ExampleOpenAILLM() {
	config := &llm.LLMConfig{
		Provider:    "openai",
		Model:       "gpt-3.5-turbo",
		APIKey:      "your-api-key",
		MaxTokens:   1024,
		Temperature: 0.7,
		TopP:        0.9,
	}
	
	llmInstance, err := llm.CreateLLM(config)
	if err != nil {
		log.Fatalf("Failed to create OpenAI LLM: %v", err)
	}
	defer llmInstance.Close()
	
	// Create messages for conversation
	messages := types.MessageList{
		{Role: types.MessageRoleSystem, Content: "You are a helpful assistant."},
		{Role: types.MessageRoleUser, Content: "Hello! How are you?"},
	}
	
	// Generate response
	ctx := context.Background()
	response, err := llmInstance.Generate(ctx, messages)
	if err != nil {
		log.Fatalf("Failed to generate response: %v", err)
	}
	
	fmt.Printf("Response: %s\n", response)
}

// ExampleOllamaLLM demonstrates how to use Ollama LLM
func ExampleOllamaLLM() {
	config := &llm.LLMConfig{
		Provider:    "ollama",
		Model:       "llama2",
		BaseURL:     "http://localhost:11434",
		MaxTokens:   1024,
		Temperature: 0.7,
		TopP:        0.9,
	}
	
	llmInstance, err := llm.CreateLLM(config)
	if err != nil {
		log.Fatalf("Failed to create Ollama LLM: %v", err)
	}
	defer llmInstance.Close()
	
	// Check if Ollama is healthy
	ctx := context.Background()
	if err := llmInstance.HealthCheck(ctx); err != nil {
		log.Printf("Ollama health check failed: %v", err)
		return
	}
	
	// Create messages for conversation
	messages := types.MessageList{
		{Role: types.MessageRoleUser, Content: "What is the capital of France?"},
	}
	
	// Generate response
	response, err := llmInstance.Generate(ctx, messages)
	if err != nil {
		log.Fatalf("Failed to generate response: %v", err)
	}
	
	fmt.Printf("Response: %s\n", response)
}

// ExampleHuggingFaceLLM demonstrates how to use HuggingFace LLM
func ExampleHuggingFaceLLM() {
	config := &llm.LLMConfig{
		Provider:    "huggingface",
		Model:       "microsoft/DialoGPT-small",
		APIKey:      "your-hf-token", // Optional for public models
		MaxTokens:   512,
		Temperature: 0.7,
		TopP:        0.9,
	}
	
	llmInstance, err := llm.CreateLLM(config)
	if err != nil {
		log.Fatalf("Failed to create HuggingFace LLM: %v", err)
	}
	defer llmInstance.Close()
	
	// Create messages for conversation
	messages := types.MessageList{
		{Role: types.MessageRoleUser, Content: "Hello, how can I help you today?"},
	}
	
	// Generate response
	ctx := context.Background()
	response, err := llmInstance.Generate(ctx, messages)
	if err != nil {
		log.Fatalf("Failed to generate response: %v", err)
	}
	
	fmt.Printf("Response: %s\n", response)
}

// Example_streamingGeneration demonstrates streaming text generation
func Example_streamingGeneration() {
	config := &llm.LLMConfig{
		Provider:    "openai",
		Model:       "gpt-3.5-turbo",
		APIKey:      "your-api-key",
		MaxTokens:   1024,
		Temperature: 0.7,
	}
	
	llmInstance, err := llm.CreateLLM(config)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}
	defer llmInstance.Close()
	
	messages := types.MessageList{
		{Role: types.MessageRoleUser, Content: "Write a short story about a robot."},
	}
	
	// Create a channel for streaming
	streamChan := make(chan string, 100)
	
	// Generate response with streaming
	ctx := context.Background()
	go func() {
		defer close(streamChan)
		if err := llmInstance.GenerateStream(ctx, messages, streamChan); err != nil {
			log.Printf("Streaming error: %v", err)
		}
	}()
	
	// Read streaming response
	fmt.Print("Streaming response: ")
	for chunk := range streamChan {
		fmt.Print(chunk)
	}
	fmt.Println()
}

// Example_embeddings demonstrates generating embeddings
func Example_embeddings() {
	config := &llm.LLMConfig{
		Provider: "openai",
		Model:    "text-embedding-ada-002",
		APIKey:   "your-api-key",
	}
	
	llmInstance, err := llm.CreateLLM(config)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}
	defer llmInstance.Close()
	
	// Generate embeddings
	ctx := context.Background()
	text := "This is a sample text for embedding generation."
	embedding, err := llmInstance.Embed(ctx, text)
	if err != nil {
		log.Fatalf("Failed to generate embedding: %v", err)
	}
	
	fmt.Printf("Generated embedding with %d dimensions\n", len(embedding))
	fmt.Printf("First 5 values: %v\n", embedding[:5])
}

// Example_batchConfiguration demonstrates using configuration maps
func Example_batchConfiguration() {
	// Create multiple LLM instances from configuration maps
	configs := []map[string]interface{}{
		{
			"provider":    "openai",
			"model":       "gpt-3.5-turbo",
			"api_key":     "your-openai-key",
			"max_tokens":  1024,
			"temperature": 0.7,
		},
		{
			"provider":    "ollama",
			"model":       "llama2",
			"base_url":    "http://localhost:11434",
			"max_tokens":  1024,
			"temperature": 0.8,
		},
		{
			"provider":    "huggingface",
			"model":       "microsoft/DialoGPT-medium",
			"api_key":     "your-hf-token",
			"max_tokens":  512,
			"temperature": 0.6,
		},
	}
	
	var llmInstances []llm.LLMProvider
	defer func() {
		for _, instance := range llmInstances {
			instance.Close()
		}
	}()
	
	for i, configMap := range configs {
		instance, err := llm.CreateLLMFromMap(configMap)
		if err != nil {
			log.Printf("Failed to create LLM %d: %v", i, err)
			continue
		}
		llmInstances = append(llmInstances, instance)
		fmt.Printf("Created %s LLM with model %s\n", 
			instance.GetProviderName(), 
			instance.(*llm.OpenAILLM).GetModelName())
	}
}

// Example_healthChecks demonstrates health checking multiple providers
func Example_healthChecks() {
	factory := llm.NewLLMFactory()
	ctx := context.Background()
	
	// Perform health checks on all providers
	results := factory.HealthCheck(ctx)
	
	fmt.Println("Health check results:")
	for provider, err := range results {
		if err != nil {
			fmt.Printf("  %s: UNHEALTHY (%v)\n", provider, err)
		} else {
			fmt.Printf("  %s: HEALTHY\n", provider)
		}
	}
}

// Example_providerDefaults demonstrates using provider defaults
func Example_providerDefaults() {
	providers := []string{"openai", "ollama", "huggingface"}
	
	for _, provider := range providers {
		defaults := llm.ProviderDefaults(provider)
		fmt.Printf("%s defaults:\n", provider)
		for key, value := range defaults {
			fmt.Printf("  %s: %v\n", key, value)
		}
		fmt.Println()
	}
}

// Example_customProvider demonstrates registering a custom provider
func Example_customProvider() {
	// Register a custom provider that wraps Ollama
	llm.RegisterProvider("custom-ollama", func(config *llm.LLMConfig) (llm.LLMProvider, error) {
		// Modify config for custom behavior
		config.BaseURL = "http://custom-ollama:11434"
		return llm.NewOllamaLLM(config)
	})
	
	// Use the custom provider
	config := &llm.LLMConfig{
		Provider: "custom-ollama",
		Model:    "llama2",
	}
	
	instance, err := llm.CreateLLM(config)
	if err != nil {
		log.Fatalf("Failed to create custom LLM: %v", err)
	}
	defer instance.Close()
	
	fmt.Printf("Created custom provider: %s\n", instance.GetProviderName())
}

// Example_errorHandling demonstrates error handling patterns
func Example_errorHandling() {
	config := &llm.LLMConfig{
		Provider: "openai",
		Model:    "gpt-3.5-turbo",
		APIKey:   "invalid-key",
	}
	
	llmInstance, err := llm.CreateLLM(config)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}
	defer llmInstance.Close()
	
	messages := types.MessageList{
		{Role: types.MessageRoleUser, Content: "Hello"},
	}
	
	ctx := context.Background()
	response, err := llmInstance.Generate(ctx, messages)
	if err != nil {
		// Check for specific error types
		if llmErr, ok := err.(*llm.LLMError); ok {
			fmt.Printf("LLM Error [%s]: %s\n", llmErr.Code, llmErr.Message)
		} else {
			fmt.Printf("General error: %v\n", err)
		}
		return
	}
	
	fmt.Printf("Response: %s\n", response)
}

// Example_modelValidation demonstrates model validation
func Example_modelValidation() {
	config := &llm.LLMConfig{
		Provider:    "openai",
		Model:       "gpt-3.5-turbo",
		APIKey:      "your-api-key",
		MaxTokens:   1024,
		Temperature: 0.7,
	}
	
	llmInstance, err := llm.CreateLLM(config)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}
	defer llmInstance.Close()
	
	openaiLLM := llmInstance.(*llm.OpenAILLM)
	
	// Check if model is supported
	supportedModels := openaiLLM.GetSupportedModels()
	fmt.Printf("Supported models: %v\n", supportedModels)
	
	// Validate specific model
	if openaiLLM.IsModelSupported("gpt-4") {
		fmt.Println("GPT-4 is supported")
	} else {
		fmt.Println("GPT-4 is not supported")
	}
	
	// Get model capabilities
	capabilities := openaiLLM.GetModelCapabilities("gpt-3.5-turbo")
	fmt.Printf("Model capabilities: %v\n", capabilities)
}

// Example_metricsAndMonitoring demonstrates metrics collection
func Example_metricsAndMonitoring() {
	config := &llm.LLMConfig{
		Provider: "openai",
		Model:    "gpt-3.5-turbo",
		APIKey:   "your-api-key",
	}
	
	llmInstance, err := llm.CreateLLM(config)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}
	defer llmInstance.Close()
	
	messages := types.MessageList{
		{Role: types.MessageRoleUser, Content: "Hello"},
	}
	
	ctx := context.Background()
	response, err := llmInstance.Generate(ctx, messages)
	if err != nil {
		log.Fatalf("Failed to generate response: %v", err)
	}
	
	// Get model information and metrics
	modelInfo := llmInstance.GetModelInfo()
	fmt.Printf("Model info: %v\n", modelInfo)
	
	// Get token usage for OpenAI
	if openaiLLM, ok := llmInstance.(*llm.OpenAILLM); ok {
		tokenUsage := openaiLLM.GetTokenUsage()
		fmt.Printf("Token usage: %v\n", tokenUsage)
	}
	
	fmt.Printf("Response: %s\n", response)
}