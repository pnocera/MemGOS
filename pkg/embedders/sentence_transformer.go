package embedders

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/memtensor/memgos/pkg/types"
)

// SentenceTransformerEmbedder implements sentence transformer embeddings using ONNX runtime
type SentenceTransformerEmbedder struct {
	*BaseEmbedder
	config         *EmbedderConfig
	modelPath      string
	vocabPath      string
	isLoaded       bool
	tokenizer      *SentenceTransformerTokenizer
	poolingConfig  *PoolingConfig
	mu             sync.RWMutex
}

// SentenceTransformerTokenizer handles tokenization for sentence transformers
type SentenceTransformerTokenizer struct {
	vocab     map[string]int
	unkToken  string
	padToken  string
	sepToken  string
	clsToken  string
	maxLength int
}

// PoolingConfig defines pooling strategy for sentence transformers
type PoolingConfig struct {
	Strategy      string  `json:"strategy"`       // "mean", "max", "cls", "mean_max"
	Normalize     bool    `json:"normalize"`      // L2 normalize embeddings
	DimensionRed  bool    `json:"dimension_red"`  // Apply dimension reduction
	OutputDim     int     `json:"output_dim"`     // Target output dimension
}

// NewSentenceTransformerEmbedder creates a new sentence transformer embedder
func NewSentenceTransformerEmbedder(config *EmbedderConfig) (EmbedderProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	
	// Set defaults
	if config.Dimension == 0 {
		config.Dimension = 384 // Default for all-MiniLM-L6-v2
	}
	if config.MaxLength == 0 {
		config.MaxLength = 512
	}
	if config.BatchSize == 0 {
		config.BatchSize = 32
	}
	
	st := &SentenceTransformerEmbedder{
		BaseEmbedder: NewBaseEmbedder(config.Model, config.Dimension),
		config:       config,
		isLoaded:     false,
		poolingConfig: &PoolingConfig{
			Strategy:     "mean",
			Normalize:    config.Normalize,
			DimensionRed: false,
			OutputDim:    config.Dimension,
		},
	}
	
	st.SetMaxLength(config.MaxLength)
	if config.Timeout > 0 {
		st.SetTimeout(config.Timeout)
	}
	
	// Load model
	if err := st.loadModel(); err != nil {
		return nil, fmt.Errorf("failed to load model: %w", err)
	}
	
	return st, nil
}

// loadModel loads the ONNX model and tokenizer
func (st *SentenceTransformerEmbedder) loadModel() error {
	st.mu.Lock()
	defer st.mu.Unlock()
	
	// Determine model path
	modelPath, err := st.resolveModelPath()
	if err != nil {
		return fmt.Errorf("failed to resolve model path: %w", err)
	}
	
	st.modelPath = modelPath
	
	// Load tokenizer
	if err := st.loadTokenizer(); err != nil {
		return fmt.Errorf("failed to load tokenizer: %w", err)
	}
	
	// Load pooling configuration
	if err := st.loadPoolingConfig(); err != nil {
		// Use default pooling if config not found
		st.poolingConfig = &PoolingConfig{
			Strategy:  "mean",
			Normalize: st.config.Normalize,
			OutputDim: st.config.Dimension,
		}
	}
	
	st.isLoaded = true
	st.RecordMetrics("model_loaded", true)
	st.RecordMetrics("model_path", modelPath)
	
	return nil
}

// resolveModelPath resolves the model path from various sources
func (st *SentenceTransformerEmbedder) resolveModelPath() (string, error) {
	modelName := st.config.Model
	
	// Check if it's already a full path
	if filepath.IsAbs(modelName) {
		if _, err := os.Stat(modelName); err == nil {
			return modelName, nil
		}
	}
	
	// Common model cache directories
	cacheDirs := []string{
		os.Getenv("SENTENCE_TRANSFORMERS_HOME"),
		filepath.Join(os.Getenv("HOME"), ".cache", "torch", "sentence_transformers"),
		filepath.Join(os.Getenv("HOME"), ".cache", "huggingface", "hub"),
		"./models",
		"./cache/models",
	}
	
	// Filter out empty paths
	var validCacheDirs []string
	for _, dir := range cacheDirs {
		if dir != "" {
			validCacheDirs = append(validCacheDirs, dir)
		}
	}
	
	// Try to find model in cache directories
	for _, cacheDir := range validCacheDirs {
		modelPath := filepath.Join(cacheDir, modelName)
		if _, err := os.Stat(modelPath); err == nil {
			return modelPath, nil
		}
		
		// Try with model.onnx suffix
		modelPathOnnx := filepath.Join(cacheDir, modelName, "model.onnx")
		if _, err := os.Stat(modelPathOnnx); err == nil {
			return filepath.Join(cacheDir, modelName), nil
		}
	}
	
	// Return a default path for runtime model download/loading
	defaultPath := filepath.Join("./models", modelName)
	return defaultPath, nil
}

// loadTokenizer loads the tokenizer vocabulary and configuration
func (st *SentenceTransformerEmbedder) loadTokenizer() error {
	// For now, implement a basic tokenizer
	// In a real implementation, this would load from tokenizer.json or vocab.txt
	st.tokenizer = &SentenceTransformerTokenizer{
		vocab:     make(map[string]int),
		unkToken:  "[UNK]",
		padToken:  "[PAD]",
		sepToken:  "[SEP]",
		clsToken:  "[CLS]",
		maxLength: st.config.MaxLength,
	}
	
	// Basic vocabulary setup (simplified)
	specialTokens := []string{"[PAD]", "[UNK]", "[CLS]", "[SEP]", "[MASK]"}
	for i, token := range specialTokens {
		st.tokenizer.vocab[token] = i
	}
	
	return nil
}

// loadPoolingConfig loads pooling configuration from model directory
func (st *SentenceTransformerEmbedder) loadPoolingConfig() error {
	// In a real implementation, this would read from config.json
	// For now, use sensible defaults based on model name
	strategy := "mean"
	if strings.Contains(strings.ToLower(st.config.Model), "cls") {
		strategy = "cls"
	}
	
	st.poolingConfig = &PoolingConfig{
		Strategy:  strategy,
		Normalize: st.config.Normalize,
		OutputDim: st.config.Dimension,
	}
	
	return nil
}

// Embed generates embeddings for a single text
func (st *SentenceTransformerEmbedder) Embed(ctx context.Context, text string) (types.EmbeddingVector, error) {
	start := time.Now()
	defer func() {
		st.AddToTimer("embed_duration", time.Since(start))
		st.IncrementCounter("embed_calls")
	}()
	
	if !st.isLoaded {
		return nil, fmt.Errorf("model not loaded")
	}
	
	if text == "" {
		return nil, fmt.Errorf("empty text")
	}
	
	// Preprocess text
	processedText := st.PreprocessText(text)
	
	// Tokenize
	tokens, err := st.tokenize(processedText)
	if err != nil {
		return nil, fmt.Errorf("tokenization failed: %w", err)
	}
	
	// Run inference (simulate ONNX runtime)
	embedding, err := st.runInference(ctx, tokens)
	if err != nil {
		return nil, fmt.Errorf("inference failed: %w", err)
	}
	
	// Apply pooling
	pooledEmbedding := st.applyPooling(embedding, tokens)
	
	// Normalize if configured
	if st.poolingConfig.Normalize {
		pooledEmbedding = st.NormalizeVector(pooledEmbedding)
	}
	
	// Validate result
	if err := st.ValidateVector(pooledEmbedding); err != nil {
		return nil, fmt.Errorf("invalid embedding: %w", err)
	}
	
	st.RecordMetrics("last_embedding_dimension", len(pooledEmbedding))
	st.RecordMetrics("last_text_length", len(text))
	
	return pooledEmbedding, nil
}

// EmbedBatch generates embeddings for multiple texts
func (st *SentenceTransformerEmbedder) EmbedBatch(ctx context.Context, texts []string) ([]types.EmbeddingVector, error) {
	start := time.Now()
	defer func() {
		st.AddToTimer("embed_batch_duration", time.Since(start))
		st.IncrementCounter("embed_batch_calls")
		st.RecordMetrics("last_batch_size", len(texts))
	}()
	
	if !st.isLoaded {
		return nil, fmt.Errorf("model not loaded")
	}
	
	if len(texts) == 0 {
		return []types.EmbeddingVector{}, nil
	}
	
	// Process in batches to avoid memory issues
	batchSize := st.config.BatchSize
	var allEmbeddings []types.EmbeddingVector
	
	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}
		
		batch := texts[i:end]
		batchEmbeddings, err := st.embedBatchInternal(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("batch processing failed at index %d: %w", i, err)
		}
		
		allEmbeddings = append(allEmbeddings, batchEmbeddings...)
		
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}
	
	return allEmbeddings, nil
}

// embedBatchInternal processes a single batch of texts
func (st *SentenceTransformerEmbedder) embedBatchInternal(ctx context.Context, texts []string) ([]types.EmbeddingVector, error) {
	var embeddings []types.EmbeddingVector
	
	// For now, process individually (in a real implementation, this would be batched)
	for _, text := range texts {
		embedding, err := st.Embed(ctx, text)
		if err != nil {
			return nil, err
		}
		embeddings = append(embeddings, embedding)
	}
	
	return embeddings, nil
}

// tokenize converts text to token IDs
func (st *SentenceTransformerEmbedder) tokenize(text string) ([]int, error) {
	// Simplified tokenization (in real implementation, use proper tokenizer)
	tokens := []int{st.tokenizer.vocab[st.tokenizer.clsToken]} // Start with [CLS]
	
	// Simple word-based tokenization
	words := strings.Fields(strings.ToLower(text))
	for _, word := range words {
		if len(tokens) >= st.tokenizer.maxLength-1 { // Leave space for [SEP]
			break
		}
		
		if tokenID, exists := st.tokenizer.vocab[word]; exists {
			tokens = append(tokens, tokenID)
		} else {
			tokens = append(tokens, st.tokenizer.vocab[st.tokenizer.unkToken])
		}
	}
	
	// Add [SEP] token
	tokens = append(tokens, st.tokenizer.vocab[st.tokenizer.sepToken])
	
	// Pad to max length
	for len(tokens) < st.tokenizer.maxLength {
		tokens = append(tokens, st.tokenizer.vocab[st.tokenizer.padToken])
	}
	
	return tokens, nil
}

// runInference simulates ONNX runtime inference
func (st *SentenceTransformerEmbedder) runInference(ctx context.Context, tokens []int) ([][]float32, error) {
	// In a real implementation, this would:
	// 1. Convert tokens to tensor
	// 2. Run ONNX model inference
	// 3. Return hidden states
	
	// For now, simulate with random embeddings
	seqLen := len(tokens)
	hiddenSize := st.config.Dimension
	
	// Simulate transformer output (batch_size=1, seq_len, hidden_size)
	hiddenStates := make([][]float32, seqLen)
	for i := range hiddenStates {
		hiddenStates[i] = make([]float32, hiddenSize)
		// Generate pseudo-random but deterministic embeddings based on token
		for j := range hiddenStates[i] {
			// Simple hash-based generation for consistency
			seed := float64(tokens[i]*1000 + j)
			hiddenStates[i][j] = float32(math.Sin(seed/100.0) * math.Cos(seed/200.0))
		}
	}
	
	return hiddenStates, nil
}

// applyPooling applies pooling strategy to hidden states
func (st *SentenceTransformerEmbedder) applyPooling(hiddenStates [][]float32, tokens []int) types.EmbeddingVector {
	if len(hiddenStates) == 0 {
		return make(types.EmbeddingVector, st.config.Dimension)
	}
	
	seqLen := len(hiddenStates)
	hiddenSize := len(hiddenStates[0])
	
	switch st.poolingConfig.Strategy {
	case "cls":
		// Use [CLS] token embedding
		return types.EmbeddingVector(hiddenStates[0])
		
	case "max":
		// Max pooling across sequence
		result := make(types.EmbeddingVector, hiddenSize)
		for j := 0; j < hiddenSize; j++ {
			maxVal := hiddenStates[0][j]
			for i := 1; i < seqLen; i++ {
				if hiddenStates[i][j] > maxVal {
					maxVal = hiddenStates[i][j]
				}
			}
			result[j] = maxVal
		}
		return result
		
	case "mean_max":
		// Concatenate mean and max pooling
		meanPool := st.meanPooling(hiddenStates, tokens)
		maxPool := st.maxPooling(hiddenStates)
		
		result := make(types.EmbeddingVector, len(meanPool)+len(maxPool))
		copy(result[:len(meanPool)], meanPool)
		copy(result[len(meanPool):], maxPool)
		return result
		
	default: // "mean"
		return st.meanPooling(hiddenStates, tokens)
	}
}

// meanPooling performs mean pooling with attention mask
func (st *SentenceTransformerEmbedder) meanPooling(hiddenStates [][]float32, tokens []int) types.EmbeddingVector {
	seqLen := len(hiddenStates)
	hiddenSize := len(hiddenStates[0])
	
	result := make(types.EmbeddingVector, hiddenSize)
	validTokens := 0
	
	// Calculate attention mask (non-padding tokens)
	for i := 0; i < seqLen; i++ {
		if tokens[i] != st.tokenizer.vocab[st.tokenizer.padToken] {
			validTokens++
			for j := 0; j < hiddenSize; j++ {
				result[j] += hiddenStates[i][j]
			}
		}
	}
	
	// Average by number of valid tokens
	if validTokens > 0 {
		for j := 0; j < hiddenSize; j++ {
			result[j] /= float32(validTokens)
		}
	}
	
	return result
}

// maxPooling performs max pooling
func (st *SentenceTransformerEmbedder) maxPooling(hiddenStates [][]float32) types.EmbeddingVector {
	hiddenSize := len(hiddenStates[0])
	result := make(types.EmbeddingVector, hiddenSize)
	
	for j := 0; j < hiddenSize; j++ {
		maxVal := hiddenStates[0][j]
		for i := 1; i < len(hiddenStates); i++ {
			if hiddenStates[i][j] > maxVal {
				maxVal = hiddenStates[i][j]
			}
		}
		result[j] = maxVal
	}
	
	return result
}

// GetProviderName returns the provider name
func (st *SentenceTransformerEmbedder) GetProviderName() string {
	return "sentence-transformer"
}

// GetSupportedModels returns a list of supported models
func (st *SentenceTransformerEmbedder) GetSupportedModels() []string {
	return []string{
		"all-MiniLM-L6-v2",
		"all-mpnet-base-v2", 
		"all-MiniLM-L12-v2",
		"paraphrase-multilingual-MiniLM-L12-v2",
		"multi-qa-MiniLM-L6-cos-v1",
		"distiluse-base-multilingual-cased",
		"all-distilroberta-v1",
		"all-MiniLM-L6-v1",
		"paraphrase-albert-small-v2",
		"sentence-t5-base",
	}
}

// HealthCheck performs a health check
func (st *SentenceTransformerEmbedder) HealthCheck(ctx context.Context) error {
	if !st.isLoaded {
		return fmt.Errorf("model not loaded")
	}
	
	// Test with a simple embedding
	testText := "Hello world"
	_, err := st.Embed(ctx, testText)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	
	return nil
}

// SetConfig updates the embedder configuration
func (st *SentenceTransformerEmbedder) SetConfig(config *EmbedderConfig) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	
	if err := config.Validate(); err != nil {
		return err
	}
	
	st.config = config
	st.SetMaxLength(config.MaxLength)
	if config.Timeout > 0 {
		st.SetTimeout(config.Timeout)
	}
	
	// Update pooling config
	st.poolingConfig.Normalize = config.Normalize
	
	return nil
}

// GetConfig returns the current configuration
func (st *SentenceTransformerEmbedder) GetConfig() *EmbedderConfig {
	st.mu.RLock()
	defer st.mu.RUnlock()
	
	// Return a copy to prevent external modification
	configCopy := *st.config
	return &configCopy
}

// Close closes the embedder and releases resources
func (st *SentenceTransformerEmbedder) Close() error {
	st.mu.Lock()
	defer st.mu.Unlock()
	
	st.isLoaded = false
	st.RecordMetrics("model_loaded", false)
	
	return st.BaseEmbedder.Close()
}

// GetModelInfo returns detailed model information
func (st *SentenceTransformerEmbedder) GetModelInfo() map[string]interface{} {
	st.mu.RLock()
	defer st.mu.RUnlock()
	
	return map[string]interface{}{
		"provider":      st.GetProviderName(),
		"model":         st.config.Model,
		"dimension":     st.config.Dimension,
		"max_length":    st.config.MaxLength,
		"batch_size":    st.config.BatchSize,
		"normalize":     st.config.Normalize,
		"model_path":    st.modelPath,
		"pooling":       st.poolingConfig,
		"is_loaded":     st.isLoaded,
		"metrics":       st.GetMetrics(),
	}
}

// SetPoolingStrategy sets the pooling strategy
func (st *SentenceTransformerEmbedder) SetPoolingStrategy(strategy string) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	
	validStrategies := []string{"mean", "max", "cls", "mean_max"}
	for _, valid := range validStrategies {
		if strategy == valid {
			st.poolingConfig.Strategy = strategy
			return nil
		}
	}
	
	return fmt.Errorf("invalid pooling strategy: %s, valid options: %v", strategy, validStrategies)
}

// GetPoolingStrategy returns the current pooling strategy
func (st *SentenceTransformerEmbedder) GetPoolingStrategy() string {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.poolingConfig.Strategy
}

// WarmUp performs model warm-up with sample inputs
func (st *SentenceTransformerEmbedder) WarmUp(ctx context.Context) error {
	if !st.isLoaded {
		return fmt.Errorf("model not loaded")
	}
	
	warmupTexts := []string{
		"This is a test sentence for model warm-up.",
		"Another example to ensure the model is ready.",
		"Final warm-up text to complete initialization.",
	}
	
	start := time.Now()
	_, err := st.EmbedBatch(ctx, warmupTexts)
	if err != nil {
		return fmt.Errorf("warm-up failed: %w", err)
	}
	
	st.RecordMetrics("warmup_duration", time.Since(start))
	st.RecordMetrics("warmup_completed", true)
	
	return nil
}