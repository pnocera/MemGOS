// Package main demonstrates LLM integration usage in MemGOS
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/memtensor/memgos/pkg/llm"
	"github.com/memtensor/memgos/pkg/types"
)

func main() {
	fmt.Println("MemGOS LLM Integration Demo")
	fmt.Println("===========================")

	// Demo 1: OpenAI Integration
	demoOpenAI()

	// Demo 2: Ollama Integration  
	demoOllama()

	// Demo 3: HuggingFace Integration
	demoHuggingFace()

	// Demo 4: Factory Pattern
	demoFactory()

	// Demo 5: Health Monitoring
	demoHealthCheck()
}

func demoOpenAI() {
	fmt.Println("\n1. OpenAI Integration Demo")
	fmt.Println("--------------------------")

	config := &llm.LLMConfig{
		Provider:    "openai",
		Model:       "gpt-3.5-turbo",
		APIKey:      "your-api-key-here", // Replace with actual key
		MaxTokens:   100,
		Temperature: 0.7,
		TopP:        0.9,
		Timeout:     30 * time.Second,
	}

	llmInstance, err := llm.CreateLLM(config)
	if err != nil {
		log.Printf("Failed to create OpenAI LLM: %v", err)
		return
	}
	defer llmInstance.Close()

	messages := types.MessageList{
		{Role: types.MessageRoleSystem, Content: "You are a helpful assistant."},
		{Role: types.MessageRoleUser, Content: "What is the capital of France?"},
	}

	ctx := context.Background()
	response, err := llmInstance.Generate(ctx, messages)
	if err != nil {
		log.Printf("OpenAI generation failed: %v", err)
		return
	}

	fmt.Printf("OpenAI Response: %s\n", response)

	// Get model info
	info := llmInstance.GetModelInfo()
	fmt.Printf("Model Info: %+v\n", info)
}

func demoOllama() {
	fmt.Println("\n2. Ollama Integration Demo")
	fmt.Println("--------------------------")

	config := &llm.LLMConfig{
		Provider:    "ollama",
		Model:       "llama2",
		BaseURL:     "http://localhost:11434",
		MaxTokens:   100,
		Temperature: 0.7,
		TopP:        0.9,
		Timeout:     30 * time.Second,
	}

	llmInstance, err := llm.CreateLLM(config)
	if err != nil {
		log.Printf("Failed to create Ollama LLM: %v", err)
		return
	}
	defer llmInstance.Close()

	// Check health first
	ctx := context.Background()
	if err := llmInstance.HealthCheck(ctx); err != nil {
		log.Printf("Ollama health check failed: %v", err)
		return
	}

	messages := types.MessageList{
		{Role: types.MessageRoleUser, Content: "Explain quantum computing in simple terms."},
	}

	response, err := llmInstance.Generate(ctx, messages)
	if err != nil {
		log.Printf("Ollama generation failed: %v", err)
		return
	}

	fmt.Printf("Ollama Response: %s\n", response)

	// Demo Ollama-specific features
	if ollamaLLM, ok := llmInstance.(*llm.OllamaLLM); ok {
		models, err := ollamaLLM.ListModels(ctx)
		if err == nil {
			fmt.Printf("Available Ollama models: %v\n", models)
		}

		stats := ollamaLLM.GetModelStats()
		fmt.Printf("Model stats: %+v\n", stats)
	}
}

func demoHuggingFace() {
	fmt.Println("\n3. HuggingFace Integration Demo")
	fmt.Println("-------------------------------")

	config := &llm.LLMConfig{
		Provider:    "huggingface",
		Model:       "microsoft/DialoGPT-small",
		APIKey:      "your-hf-token-here", // Optional for public models
		MaxTokens:   50,
		Temperature: 0.7,
		TopP:        0.9,
		Timeout:     30 * time.Second,
	}

	llmInstance, err := llm.CreateLLM(config)
	if err != nil {
		log.Printf("Failed to create HuggingFace LLM: %v", err)
		return
	}
	defer llmInstance.Close()

	messages := types.MessageList{
		{Role: types.MessageRoleUser, Content: "Hello, how are you?"},
	}

	ctx := context.Background()
	response, err := llmInstance.Generate(ctx, messages)
	if err != nil {
		log.Printf("HuggingFace generation failed: %v", err)
		return
	}

	fmt.Printf("HuggingFace Response: %s\n", response)

	// Demo HuggingFace-specific features
	if hfLLM, ok := llmInstance.(*llm.HuggingFaceLLM); ok {
		capabilities := hfLLM.GetModelCapabilities(config.Model)
		fmt.Printf("Model capabilities: %+v\n", capabilities)

		if hfLLM.SupportsBatching() {
			prompts := []string{"Hello", "How are you?", "Goodbye"}
			batchResponses, err := hfLLM.BatchGenerate(ctx, prompts)
			if err == nil {
				fmt.Printf("Batch responses: %v\n", batchResponses)
			}
		}
	}
}

func demoFactory() {
	fmt.Println("\n4. Factory Pattern Demo")
	fmt.Println("-----------------------")

	factory := llm.NewLLMFactory()

	// List all providers
	providers := factory.ListProviders()
	fmt.Printf("Available providers: %v\n", providers)

	// Create multiple LLMs from configuration maps
	configs := []map[string]interface{}{
		{
			"provider":    "ollama",
			"model":       "llama2",
			"base_url":    "http://localhost:11434",
			"max_tokens":  50,
			"temperature": 0.7,
		},
		{
			"provider":    "huggingface",
			"model":       "microsoft/DialoGPT-small",
			"max_tokens":  50,
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
		instance, err := factory.CreateLLMFromMap(configMap)
		if err != nil {
			log.Printf("Failed to create LLM %d: %v", i, err)
			continue
		}
		llmInstances = append(llmInstances, instance)
		fmt.Printf("Created %s LLM with model %s\n", 
			instance.GetProviderName(), 
			instance.GetConfig().Model)
	}

	// Get provider defaults
	for _, provider := range []string{"openai", "ollama", "huggingface"} {
		defaults := llm.ProviderDefaults(provider)
		fmt.Printf("%s defaults: model=%s, max_tokens=%v\n", 
			provider, defaults["model"], defaults["max_tokens"])
	}
}

func demoHealthCheck() {
	fmt.Println("\n5. Health Monitoring Demo")
	fmt.Println("-------------------------")

	factory := llm.NewLLMFactory()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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

	// Demo individual provider info
	for _, provider := range []string{"openai", "ollama", "huggingface"} {
		info, err := factory.GetProviderInfo(provider)
		if err != nil {
			log.Printf("Failed to get %s info: %v", provider, err)
			continue
		}
		fmt.Printf("%s info: name=%s, models_count=%d\n", 
			provider, 
			info["name"], 
			len(info["supported_models"].([]string)))
	}
}

func demoStreaming() {
	fmt.Println("\n6. Streaming Generation Demo")
	fmt.Println("----------------------------")

	config := &llm.LLMConfig{
		Provider:    "openai",
		Model:       "gpt-3.5-turbo",
		APIKey:      "your-api-key-here",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	llmInstance, err := llm.CreateLLM(config)
	if err != nil {
		log.Printf("Failed to create LLM: %v", err)
		return
	}
	defer llmInstance.Close()

	messages := types.MessageList{
		{Role: types.MessageRoleUser, Content: "Write a short poem about technology."},
	}

	// Create streaming channel
	streamChan := make(chan string, 100)

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

func demoEmbeddings() {
	fmt.Println("\n7. Embeddings Demo")
	fmt.Println("------------------")

	config := &llm.LLMConfig{
		Provider: "openai",
		Model:    "text-embedding-ada-002",
		APIKey:   "your-api-key-here",
	}

	llmInstance, err := llm.CreateLLM(config)
	if err != nil {
		log.Printf("Failed to create LLM: %v", err)
		return
	}
	defer llmInstance.Close()

	texts := []string{
		"Machine learning is a subset of artificial intelligence.",
		"Deep learning uses neural networks with multiple layers.",
		"Natural language processing enables computers to understand text.",
	}

	ctx := context.Background()
	for i, text := range texts {
		embedding, err := llmInstance.Embed(ctx, text)
		if err != nil {
			log.Printf("Embedding generation failed: %v", err)
			continue
		}
		fmt.Printf("Text %d embedding: %d dimensions, first 5 values: %v\n", 
			i+1, len(embedding), embedding[:5])
	}
}

func demoErrorHandling() {
	fmt.Println("\n8. Error Handling Demo")
	fmt.Println("----------------------")

	// Demo with invalid API key
	config := &llm.LLMConfig{
		Provider: "openai",
		Model:    "gpt-3.5-turbo",
		APIKey:   "invalid-key",
	}

	llmInstance, err := llm.CreateLLM(config)
	if err != nil {
		log.Printf("Failed to create LLM: %v", err)
		return
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
			fmt.Printf("LLM Error [%s]: %s (Type: %s)\n", 
				llmErr.Code, llmErr.Message, llmErr.Type)
		} else {
			fmt.Printf("General error: %v\n", err)
		}
		return
	}

	fmt.Printf("Response: %s\n", response)
}