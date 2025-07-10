# LLM Package Documentation

The LLM package provides a comprehensive integration layer for Large Language Models in MemGOS, supporting OpenAI, Ollama, and HuggingFace providers with a unified interface.

## Features

- **Multi-Provider Support**: OpenAI, Ollama, and HuggingFace integrations
- **Factory Pattern**: Easy creation and management of LLM instances
- **Streaming Support**: Real-time text generation streaming
- **Embeddings**: Text embedding generation for semantic search
- **Error Handling**: Comprehensive error handling with retry logic
- **Configuration**: Flexible configuration management
- **Health Checks**: Provider health monitoring
- **Metrics**: Usage statistics and performance monitoring
- **Thread Safety**: Concurrent access support

## Supported Providers

### OpenAI
- **Models**: GPT-4, GPT-3.5-turbo, and all OpenAI text models
- **Features**: Chat completion, streaming, embeddings, function calling
- **Configuration**: API key required, optional base URL override

### Ollama
- **Models**: Llama2, CodeLlama, Mistral, Mixtral, and all locally available models
- **Features**: Local model inference, streaming, model management
- **Configuration**: Base URL (default: http://localhost:11434)

### HuggingFace
- **Models**: All HuggingFace Hub models, transformers, sentence-transformers
- **Features**: API and local inference, embeddings, code generation
- **Configuration**: Optional API token for private models

## Quick Start

### Basic Usage

```go
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
    // Create LLM configuration
    config := &llm.LLMConfig{
        Provider:    "openai",
        Model:       "gpt-3.5-turbo",
        APIKey:      "your-api-key",
        MaxTokens:   1024,
        Temperature: 0.7,
        TopP:        0.9,
        Timeout:     30 * time.Second,
    }

    // Create LLM instance
    llmInstance, err := llm.CreateLLM(config)
    if err != nil {
        log.Fatalf("Failed to create LLM: %v", err)
    }
    defer llmInstance.Close()

    // Prepare messages
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
```

### Factory Pattern

```go
// Create factory
factory := llm.NewLLMFactory()

// List available providers
providers := factory.ListProviders()
fmt.Printf("Available providers: %v\n", providers)

// Create from configuration map
configMap := map[string]interface{}{
    "provider":    "ollama",
    "model":       "llama2",
    "base_url":    "http://localhost:11434",
    "max_tokens":  1024,
    "temperature": 0.7,
}

llmInstance, err := factory.CreateLLMFromMap(configMap)
if err != nil {
    log.Fatalf("Failed to create LLM: %v", err)
}
defer llmInstance.Close()
```

### Streaming Generation

```go
// Create streaming channel
streamChan := make(chan string, 100)

// Generate with streaming
go func() {
    defer close(streamChan)
    err := llmInstance.GenerateStream(ctx, messages, streamChan)
    if err != nil {
        log.Printf("Streaming error: %v", err)
    }
}()

// Read streaming response
fmt.Print("Streaming response: ")
for chunk := range streamChan {
    fmt.Print(chunk)
}
fmt.Println()
```

## Configuration

### LLMConfig Structure

```go
type LLMConfig struct {
    Provider    string            `json:"provider"`     // "openai", "ollama", "huggingface"
    Model       string            `json:"model"`        // Model name
    APIKey      string            `json:"api_key"`      // API key (optional for some providers)
    BaseURL     string            `json:"base_url"`     // Custom base URL
    MaxTokens   int               `json:"max_tokens"`   // Maximum tokens to generate
    Temperature float64           `json:"temperature"`  // Randomness (0.0-2.0)
    TopP        float64           `json:"top_p"`        // Nucleus sampling (0.0-1.0)
    Timeout     time.Duration     `json:"timeout"`      // Request timeout
    Extra       map[string]interface{} `json:"extra"`   // Provider-specific options
}
```

### Provider-Specific Configuration

#### OpenAI
```go
config := &llm.LLMConfig{
    Provider:    "openai",
    Model:       "gpt-3.5-turbo",
    APIKey:      "sk-...",                           // Required
    BaseURL:     "https://api.openai.com/v1",       // Optional
    MaxTokens:   1024,
    Temperature: 0.7,
    TopP:        0.9,
}
```

#### Ollama
```go
config := &llm.LLMConfig{
    Provider:    "ollama",
    Model:       "llama2",
    BaseURL:     "http://localhost:11434",  // Default
    MaxTokens:   1024,
    Temperature: 0.7,
    TopP:        0.9,
}
```

#### HuggingFace
```go
config := &llm.LLMConfig{
    Provider:    "huggingface",
    Model:       "microsoft/DialoGPT-small",
    APIKey:      "hf_...",                          // Optional
    BaseURL:     "",                                // Use "local" for local inference
    MaxTokens:   512,
    Temperature: 0.7,
    TopP:        0.9,
}
```

## Advanced Features

### Health Monitoring

```go
// Check individual provider health
ctx := context.Background()
err := llmInstance.HealthCheck(ctx)
if err != nil {
    log.Printf("Health check failed: %v", err)
}

// Check all providers
factory := llm.NewLLMFactory()
results := factory.HealthCheck(ctx)
for provider, err := range results {
    if err != nil {
        fmt.Printf("%s: UNHEALTHY (%v)\n", provider, err)
    } else {
        fmt.Printf("%s: HEALTHY\n", provider)
    }
}
```

### Embeddings

```go
// Generate embeddings
text := "This is a sample text for embedding."
embedding, err := llmInstance.Embed(ctx, text)
if err != nil {
    log.Fatalf("Failed to generate embedding: %v", err)
}

fmt.Printf("Generated %d-dimensional embedding\n", len(embedding))
```

### Metrics and Monitoring

```go
// Get model information
info := llmInstance.GetModelInfo()
fmt.Printf("Model info: %v\n", info)

// Get provider-specific metrics
if openaiLLM, ok := llmInstance.(*llm.OpenAILLM); ok {
    tokenUsage := openaiLLM.GetTokenUsage()
    fmt.Printf("Token usage: %v\n", tokenUsage)
}
```

### Custom Providers

```go
// Register custom provider
llm.RegisterProvider("custom", func(config *llm.LLMConfig) (llm.LLMProvider, error) {
    // Custom implementation
    return customImplementation(config)
})

// Use custom provider
config := &llm.LLMConfig{
    Provider: "custom",
    Model:    "custom-model",
}
llmInstance, err := llm.CreateLLM(config)
```

## Provider-Specific Features

### OpenAI Features

```go
openaiLLM := llmInstance.(*llm.OpenAILLM)

// List available models
models, err := openaiLLM.ListModels(ctx)

// Function calling
functions := []openai.FunctionDefinition{
    // Define functions
}
response, err := openaiLLM.CreateFunctionCall(ctx, messages, functions)

// Custom completion options
req := openai.ChatCompletionRequest{
    Model:    "gpt-4",
    Messages: openaiMessages,
    // Custom options
}
response, err := openaiLLM.CreateChatCompletion(ctx, req)
```

### Ollama Features

```go
ollamaLLM := llmInstance.(*llm.OllamaLLM)

// Pull new model
err := ollamaLLM.PullModel(ctx, "llama2:13b")

// Delete model
err := ollamaLLM.DeleteModel(ctx, "old-model")

// Generate completion (non-chat)
response, err := ollamaLLM.GenerateCompletion(ctx, "Complete this sentence:")

// Create custom model
modelfile := `FROM llama2
PARAMETER temperature 0.8`
err := ollamaLLM.CreateModelfile(ctx, "custom-model", modelfile)

// Get version information
version, err := ollamaLLM.GetVersion(ctx)
```

### HuggingFace Features

```go
hfLLM := llmInstance.(*llm.HuggingFaceLLM)

// Search models
models, err := hfLLM.SearchModels(ctx, "conversational", 10)

// Get model details
details, err := hfLLM.GetModelDetails(ctx, "microsoft/DialoGPT-large")

// Generate code
code, err := hfLLM.GenerateCode(ctx, "Create a hello world function", "python")

// Batch generation
prompts := []string{"Hello", "World", "Test"}
responses, err := hfLLM.BatchGenerate(ctx, prompts)

// Set local mode
hfLLM.SetUseLocal(true)
```

## Error Handling

```go
response, err := llmInstance.Generate(ctx, messages)
if err != nil {
    // Check for LLM-specific errors
    if llmErr, ok := err.(*llm.LLMError); ok {
        switch llmErr.Type {
        case "authentication":
            log.Printf("Authentication error: %s", llmErr.Message)
        case "rate_limit":
            log.Printf("Rate limit exceeded: %s", llmErr.Message)
        case "model_not_found":
            log.Printf("Model not found: %s", llmErr.Message)
        default:
            log.Printf("LLM error [%s]: %s", llmErr.Code, llmErr.Message)
        }
    } else {
        log.Printf("General error: %v", err)
    }
    return
}
```

## Configuration Management

### Environment Variables

```bash
# OpenAI
export OPENAI_API_KEY="sk-..."
export OPENAI_BASE_URL="https://api.openai.com/v1"

# Ollama
export OLLAMA_BASE_URL="http://localhost:11434"

# HuggingFace
export HUGGINGFACE_API_TOKEN="hf_..."
```

### Configuration Files

```yaml
# config.yaml
llm:
  provider: "openai"
  model: "gpt-3.5-turbo"
  api_key: "${OPENAI_API_KEY}"
  max_tokens: 1024
  temperature: 0.7
  top_p: 0.9
  timeout: "30s"
```

### Provider Defaults

```go
// Get provider defaults
defaults := llm.ProviderDefaults("openai")
fmt.Printf("OpenAI defaults: %v\n", defaults)

// Merge with user config
userConfig := map[string]interface{}{
    "model": "gpt-4",
    "temperature": 0.5,
}
merged := llm.MergeConfigWithDefaults("openai", userConfig)
```

## Performance Optimization

### Connection Pooling

```go
// HTTP client settings are automatically configured
// with connection pooling and keep-alive
```

### Retry Logic

```go
// Automatic retry with exponential backoff
// is built into all providers
```

### Caching

```go
// KV cache support (where available)
response, err := llmInstance.GenerateWithCache(ctx, messages, cache)
```

### Batching

```go
// Batch multiple requests where supported
if hfLLM.SupportsBatching() {
    responses, err := hfLLM.BatchGenerate(ctx, prompts)
}
```

## Testing

### Unit Tests

```bash
go test ./pkg/llm/...
```

### Integration Tests

```bash
# Run with actual services
go test -tags=integration ./pkg/llm/...
```

### Mock Testing

```go
// Mock LLM for testing
type MockLLM struct {
    *llm.BaseLLM
}

func (m *MockLLM) Generate(ctx context.Context, messages types.MessageList) (string, error) {
    return "mock response", nil
}

// Use in tests
llmInstance := &MockLLM{BaseLLM: llm.NewBaseLLM("mock")}
```

## Best Practices

1. **Always close LLM instances** to free resources:
   ```go
   defer llmInstance.Close()
   ```

2. **Use context for cancellation**:
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
   defer cancel()
   ```

3. **Handle errors appropriately**:
   ```go
   if err != nil {
       // Log error and handle gracefully
   }
   ```

4. **Configure timeouts**:
   ```go
   config.Timeout = 30 * time.Second
   ```

5. **Monitor health**:
   ```go
   go func() {
       ticker := time.NewTicker(5 * time.Minute)
       for range ticker.C {
           if err := llmInstance.HealthCheck(ctx); err != nil {
               log.Printf("Health check failed: %v", err)
           }
       }
   }()
   ```

6. **Use appropriate models** for your use case:
   - Chat: GPT-3.5-turbo, GPT-4, Llama2
   - Embeddings: text-embedding-ada-002, sentence-transformers
   - Code: CodeLlama, Codegen models

## Troubleshooting

### Common Issues

1. **Authentication errors**: Check API keys and permissions
2. **Connection timeouts**: Verify network connectivity and service availability
3. **Model not found**: Ensure model exists and is accessible
4. **Rate limiting**: Implement backoff strategies
5. **Memory issues**: Monitor token usage and model size

### Debugging

```go
// Enable debug logging
config.Extra["debug"] = true

// Check model information
info := llmInstance.GetModelInfo()
log.Printf("Model info: %+v", info)

// Monitor metrics
metrics := llmInstance.(*llm.OpenAILLM).GetTokenUsage()
log.Printf("Token usage: %+v", metrics)
```

## Contributing

1. Add new providers by implementing the `LLMProvider` interface
2. Follow existing patterns for error handling and configuration
3. Include comprehensive tests
4. Update documentation
5. Ensure thread safety

## License

This package is part of MemGOS and follows the same license terms.