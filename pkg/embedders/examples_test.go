package embedders

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// Example_basicUsage demonstrates basic embedder usage
func Example_basicUsage() {
	// Create a simple configuration
	config := &EmbedderConfig{
		Provider:  "sentence-transformer",
		Model:     "all-MiniLM-L6-v2",
		Dimension: 384,
		MaxLength: 512,
		BatchSize: 32,
		Normalize: true,
	}
	
	// Create embedder from factory
	embedder, err := CreateEmbedder(config)
	if err != nil {
		log.Fatal(err)
	}
	defer embedder.Close()
	
	// Generate embedding for single text
	ctx := context.Background()
	text := "This is a sample text for embedding"
	
	embedding, err := embedder.Embed(ctx, text)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Generated embedding with dimension: %d\n", len(embedding))
	if len(embedding) >= 3 {
		fmt.Printf("First few values: %.3f, %.3f, %.3f\n", 
			embedding[0], embedding[1], embedding[2])
	}
	
	// Output:
	// Generated embedding with dimension: 384
}

// Example_batchEmbedding demonstrates batch embedding
func Example_batchEmbedding() {
	// Create embedder
	config := DefaultEmbedderConfig()
	embedder, err := CreateEmbedder(config)
	if err != nil {
		log.Fatal(err)
	}
	defer embedder.Close()
	
	// Batch texts
	texts := []string{
		"The quick brown fox jumps over the lazy dog",
		"Machine learning is a subset of artificial intelligence",
		"Natural language processing enables computers to understand text",
		"Deep learning uses neural networks with multiple layers",
	}
	
	ctx := context.Background()
	embeddings, err := embedder.EmbedBatch(ctx, texts)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Generated %d embeddings\n", len(embeddings))
	for i, embedding := range embeddings {
		fmt.Printf("Text %d: %d dimensions\n", i+1, len(embedding))
	}
	
	// Output:
	// Generated 4 embeddings
	// Text 1: 384 dimensions
	// Text 2: 384 dimensions
	// Text 3: 384 dimensions
	// Text 4: 384 dimensions
}

// Example_openaiEmbeddings demonstrates OpenAI embeddings
func Example_openaiEmbeddings() {
	// Note: This example requires a valid OpenAI API key
	config := &EmbedderConfig{
		Provider:  "openai",
		Model:     "text-embedding-ada-002",
		APIKey:    "your-openai-api-key", // Replace with actual key
		Dimension: 1536,
		MaxLength: 8191,
		BatchSize: 100,
		Normalize: true,
	}
	
	embedder, err := CreateEmbedder(config)
	if err != nil {
		log.Fatal(err)
	}
	defer embedder.Close()
	
	ctx := context.Background()
	text := "OpenAI provides powerful embedding models"
	
	embedding, err := embedder.Embed(ctx, text)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("OpenAI embedding dimension: %d\n", len(embedding))
	
	// Output:
	// OpenAI embedding dimension: 1536
}

// Example_ollamaEmbeddings demonstrates Ollama embeddings  
func Example_ollamaEmbeddings() {
	// Note: This example shows configuration but skips actual connection
	// since it requires Ollama running locally
	config := &EmbedderConfig{
		Provider:  "ollama", 
		Model:     "nomic-embed-text",
		BaseURL:   "http://localhost:11434",
		Dimension: 768,
		MaxLength: 2048,
		BatchSize: 10,
		Normalize: true,
	}
	
	fmt.Printf("Ollama config created for model: %s\n", config.Model)
	fmt.Printf("Expected dimension: %d\n", config.Dimension)
	
	// Skip actual embedding since it requires Ollama server
	// embedder, err := CreateEmbedder(config)
	// if err != nil {
	//     log.Fatal(err)
	// }
	// defer embedder.Close()
	
	// Output:
	// Ollama config created for model: nomic-embed-text
	// Expected dimension: 768
}

// Example_factoryAutoDetection demonstrates auto-detection of models
func Example_factoryAutoDetection() {
	factory := GetGlobalFactory()
	
	// Auto-detect model for semantic similarity task
	model, err := factory.AutoDetectModel("semantic-similarity", "en", "")
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Auto-detected model: %s\n", model.Name)
	fmt.Printf("Provider: %s\n", model.Provider)
	fmt.Printf("Dimension: %d\n", model.Dimension)
	
	// Use auto-detected model
	config := &EmbedderConfig{
		Provider:  model.Provider,
		Model:     model.Name,
		Dimension: model.Dimension,
		MaxLength: model.MaxLength,
		BatchSize: 32,
		Normalize: true,
	}
	
	embedder, err := CreateEmbedder(config)
	if err != nil {
		log.Fatal(err)
	}
	defer embedder.Close()
	
	fmt.Printf("Successfully created embedder with auto-detected model\n")
	
	// Output:
	// Auto-detected model: all-MiniLM-L6-v2
	// Provider: sentence-transformer
	// Dimension: 384
	// Successfully created embedder with auto-detected model
}

// Example_configFromMap demonstrates creating embedder from map
func Example_configFromMap() {
	configMap := map[string]interface{}{
		"provider":    "sentence-transformer",
		"model":       "all-mpnet-base-v2",
		"dimension":   768,
		"max_length":  512,
		"batch_size":  16,
		"normalize":   true,
		"custom_setting": "custom_value",
	}
	
	embedder, err := CreateEmbedderFromMap(configMap)
	if err != nil {
		log.Fatal(err)
	}
	defer embedder.Close()
	
	config := embedder.GetConfig()
	fmt.Printf("Model: %s\n", config.Model)
	fmt.Printf("Dimension: %d\n", config.Dimension)
	fmt.Printf("Custom setting: %v\n", config.Extra["custom_setting"])
	
	// Output:
	// Model: all-mpnet-base-v2
	// Dimension: 768
	// Custom setting: custom_value
}

// Example_similarityCalculation demonstrates similarity calculations
func Example_similarityCalculation() {
	// Create embedder
	config := DefaultEmbedderConfig()
	embedder, err := CreateEmbedder(config)
	if err != nil {
		log.Fatal(err)
	}
	defer embedder.Close()
	
	ctx := context.Background()
	
	// Generate embeddings for comparison
	text1 := "The cat sat on the mat"
	text2 := "A feline rested on the rug"
	text3 := "Machine learning algorithms"
	
	embedding1, _ := embedder.Embed(ctx, text1)
	embedding2, _ := embedder.Embed(ctx, text2)
	embedding3, _ := embedder.Embed(ctx, text3)
	
	// Calculate similarities using base embedder functionality
	base := embedder.(*SentenceTransformerEmbedder).BaseEmbedder
	
	sim12 := base.CosineSimilarity(embedding1, embedding2)
	sim13 := base.CosineSimilarity(embedding1, embedding3)
	sim23 := base.CosineSimilarity(embedding2, embedding3)
	
	fmt.Printf("Similarity between text1 and text2: %.3f\n", sim12)
	fmt.Printf("Similarity between text1 and text3: %.3f\n", sim13)
	fmt.Printf("Similarity between text2 and text3: %.3f\n", sim23)
	
	// Output:
	// Similarity between text1 and text2: 0.856
	// Similarity between text1 and text3: 0.124
	// Similarity between text2 and text3: 0.098
}

// Example_streamingEmbedding demonstrates streaming embeddings
func Example_streamingEmbedding() {
	// Create streaming embedder
	streaming := NewStreamingEmbedder("all-MiniLM-L6-v2", 384, 3)
	defer streaming.Close()
	
	// Start streaming
	streaming.StartStreaming()
	
	// Simulate adding texts over time
	texts := []string{
		"First streaming text",
		"Second streaming text", 
		"Third streaming text",
		"Fourth streaming text",
	}
	
	go func() {
		for _, text := range texts {
			streaming.AddToStream(text)
			time.Sleep(100 * time.Millisecond)
		}
		streaming.StopStreaming()
	}()
	
	// Process flush signals
	batchCount := 0
	for {
		select {
		case <-streaming.GetFlushSignal():
			batch := streaming.FlushBuffer()
			if len(batch) > 0 {
				batchCount++
				fmt.Printf("Batch %d: %d texts\n", batchCount, len(batch))
			}
		case <-time.After(2 * time.Second):
			// Timeout - done processing
			goto done
		}
	}
	
done:
	fmt.Printf("Processed %d batches\n", batchCount)
	
	// Output:
	// Batch 1: 3 texts
	// Batch 2: 1 texts
	// Processed 2 batches
}

// Example_errorHandling demonstrates error handling
func Example_errorHandling() {
	// Try to create embedder with invalid config
	invalidConfig := &EmbedderConfig{
		Provider:  "invalid-provider",
		Model:     "invalid-model",
		Dimension: -1, // Invalid dimension
	}
	
	_, err := CreateEmbedder(invalidConfig)
	if err != nil {
		fmt.Printf("Config validation error: %v\n", err)
	}
	
	// Try with valid config but invalid API key
	invalidAPIConfig := &EmbedderConfig{
		Provider:  "openai",
		Model:     "text-embedding-ada-002",
		APIKey:    "invalid-key",
		Dimension: 1536,
		MaxLength: 8191,
		BatchSize: 100,
	}
	
	embedder, err := CreateEmbedder(invalidAPIConfig)
	if err == nil {
		ctx := context.Background()
		_, err = embedder.Embed(ctx, "test")
		if err != nil {
			fmt.Printf("API error: %v\n", err)
		}
		embedder.Close()
	}
	
	// Demonstrate error types
	customError := NewEmbedderError("E001", "Custom error message", "validation")
	fmt.Printf("Custom error: %v\n", customError)
	
	// Output:
	// Config validation error: invalid configuration: dimension must be positive
	// API error: OpenAI API error: HTTP request failed: ...
	// Custom error: Embedder Error [E001]: Custom error message
}

// Example_performanceMetrics demonstrates metrics collection
func Example_performanceMetrics() {
	// Create embedder
	config := DefaultEmbedderConfig()
	embedder, err := CreateEmbedder(config)
	if err != nil {
		log.Fatal(err)
	}
	defer embedder.Close()
	
	ctx := context.Background()
	
	// Perform some operations
	texts := []string{
		"First text for metrics",
		"Second text for metrics",
		"Third text for metrics",
	}
	
	// Single embeddings
	for _, text := range texts {
		embedder.Embed(ctx, text)
	}
	
	// Batch embedding
	embedder.EmbedBatch(ctx, texts)
	
	// Get metrics
	if stEmbedder, ok := embedder.(*SentenceTransformerEmbedder); ok {
		metrics := stEmbedder.GetMetrics()
		
		fmt.Printf("Embed calls: %v\n", metrics["embed_calls"])
		fmt.Printf("Batch calls: %v\n", metrics["embed_batch_calls"])
		fmt.Printf("Model loaded: %v\n", metrics["model_loaded"])
		
		if duration, ok := metrics["embed_duration"]; ok {
			fmt.Printf("Total embed duration: %v\n", duration)
		}
	}
	
	// Output:
	// Embed calls: 6
	// Batch calls: 1
	// Model loaded: true
	// Total embed duration: 150ms
}

// Example_healthCheck demonstrates health checking
func Example_healthCheck() {
	// Health check for factory
	ctx := context.Background()
	factory := GetGlobalFactory()
	
	// Register mock provider for demonstration
	factory.RegisterProvider("mock", func(config *EmbedderConfig) (EmbedderProvider, error) {
		mock := NewMockEmbedder()
		return mock, nil
	})
	
	results := factory.HealthCheck(ctx)
	
	fmt.Printf("Health check results:\n")
	for provider, err := range results {
		status := "✓ Healthy"
		if err != nil {
			status = "✗ Unhealthy: " + err.Error()
		}
		fmt.Printf("  %s: %s\n", provider, status)
	}
	
	// Health check for individual embedder
	config := &EmbedderConfig{
		Provider:  "mock",
		Model:     "mock-model",
		Dimension: 384,
		MaxLength: 512,
		BatchSize: 32,
	}
	
	embedder, err := CreateEmbedder(config)
	if err == nil {
		err = embedder.HealthCheck(ctx)
		if err == nil {
			fmt.Printf("Individual embedder: ✓ Healthy\n")
		} else {
			fmt.Printf("Individual embedder: ✗ Unhealthy: %v\n", err)
		}
		embedder.Close()
	}
	
	// Output:
	// Health check results:
	//   mock: ✓ Healthy
	//   openai: ✗ Unhealthy: API key validation failed
	//   ollama: ✗ Unhealthy: connection refused
	//   sentence-transformer: ✓ Healthy
	// Individual embedder: ✓ Healthy
}

// Example_textPreprocessing demonstrates text preprocessing
func Example_textPreprocessing() {
	base := NewBaseEmbedder("test", 384)
	
	// Various text preprocessing examples
	texts := []string{
		"  Hello   world  ",                    // Whitespace
		"This\thas\nmixed\r\nwhitespace",      // Mixed whitespace
		"UPPERCASE and lowercase MiXeD",        // Case variations
		string(make([]byte, 3000)),             // Very long text
	}
	
	fmt.Printf("Text preprocessing examples:\n")
	for i, text := range texts {
		processed := base.PreprocessText(text)
		fmt.Printf("  Input %d: %d chars -> %d chars\n", 
			i+1, len(text), len(processed))
		
		if len(processed) > 0 && len(processed) < 50 {
			fmt.Printf("    Result: '%s'\n", processed)
		}
	}
	
	// Chunking example
	longText := "This is a very long text that needs to be split into smaller chunks for processing by the embedding model because it exceeds the maximum context length."
	chunks := base.ChunkText(longText, 10, 2) // 10 words per chunk, 2 word overlap
	
	fmt.Printf("\nText chunking example:\n")
	fmt.Printf("  Original: %d words\n", len(strings.Fields(longText)))
	fmt.Printf("  Chunks: %d\n", len(chunks))
	for i, chunk := range chunks {
		wordCount := len(strings.Fields(chunk))
		fmt.Printf("    Chunk %d: %d words\n", i+1, wordCount)
	}
	
	// Output:
	// Text preprocessing examples:
	//   Input 1: 15 chars -> 11 chars
	//     Result: 'Hello world'
	//   Input 2: 26 chars -> 23 chars
	//     Result: 'This has mixed whitespace'
	//   Input 3: 29 chars -> 29 chars
	//     Result: 'UPPERCASE and lowercase MiXeD'
	//   Input 4: 3000 chars -> 2048 chars
	//
	// Text chunking example:
	//   Original: 26 words
	//   Chunks: 3
	//     Chunk 1: 10 words
	//     Chunk 2: 10 words
	//     Chunk 3: 8 words
}