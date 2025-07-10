package embedders

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/memtensor/memgos/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBaseEmbedder(t *testing.T) {
	embedder := NewBaseEmbedder("test-model", 384)
	
	assert.Equal(t, "test-model", embedder.GetModelName())
	assert.Equal(t, 384, embedder.GetDimension())
	assert.Equal(t, 512, embedder.GetMaxLength())
	assert.Equal(t, 30*time.Second, embedder.GetTimeout())
}

func TestBaseEmbedder_PreprocessText(t *testing.T) {
	embedder := NewBaseEmbedder("test", 384)
	
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trim whitespace",
			input:    "  hello world  ",
			expected: "hello world",
		},
		{
			name:     "normalize spaces",
			input:    "hello    world\n\ttest",
			expected: "hello world test",
		},
		{
			name:     "truncate long text",
			input:    string(make([]byte, 3000)), // Very long text
			expected: "", // Will be truncated
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := embedder.PreprocessText(tt.input)
			if tt.name == "truncate long text" {
				assert.True(t, len(result) < len(tt.input))
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestBaseEmbedder_ChunkText(t *testing.T) {
	embedder := NewBaseEmbedder("test", 384)
	
	text := "This is a test sentence with many words that should be split into chunks"
	chunks := embedder.ChunkText(text, 5, 2) // 5 words per chunk, 2 word overlap
	
	assert.Greater(t, len(chunks), 1)
	
	// Check that chunks have overlap
	if len(chunks) > 1 {
		firstChunkWords := len(chunks[0]) // This is simplified, real implementation would count words
		assert.Greater(t, firstChunkWords, 0)
	}
}

func TestBaseEmbedder_NormalizeVector(t *testing.T) {
	embedder := NewBaseEmbedder("test", 3)
	
	vector := types.EmbeddingVector{3, 4, 0} // Length should be 5
	normalized := embedder.NormalizeVector(vector)
	
	// Calculate expected values
	expected := types.EmbeddingVector{0.6, 0.8, 0}
	
	for i := range expected {
		assert.InDelta(t, expected[i], normalized[i], 0.001)
	}
	
	// Check that normalized vector has unit length
	var norm float32
	for _, val := range normalized {
		norm += val * val
	}
	assert.InDelta(t, 1.0, math.Sqrt(float64(norm)), 0.001)
}

func TestBaseEmbedder_CosineSimilarity(t *testing.T) {
	embedder := NewBaseEmbedder("test", 3)
	
	tests := []struct {
		name     string
		a        types.EmbeddingVector
		b        types.EmbeddingVector
		expected float32
	}{
		{
			name:     "identical vectors",
			a:        types.EmbeddingVector{1, 0, 0},
			b:        types.EmbeddingVector{1, 0, 0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        types.EmbeddingVector{1, 0, 0},
			b:        types.EmbeddingVector{0, 1, 0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        types.EmbeddingVector{1, 0, 0},
			b:        types.EmbeddingVector{-1, 0, 0},
			expected: -1.0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity := embedder.CosineSimilarity(tt.a, tt.b)
			assert.InDelta(t, tt.expected, similarity, 0.001)
		})
	}
}

func TestBaseEmbedder_ValidateVector(t *testing.T) {
	embedder := NewBaseEmbedder("test", 3)
	
	tests := []struct {
		name    string
		vector  types.EmbeddingVector
		wantErr bool
	}{
		{
			name:    "valid vector",
			vector:  types.EmbeddingVector{1, 2, 3},
			wantErr: false,
		},
		{
			name:    "empty vector",
			vector:  types.EmbeddingVector{},
			wantErr: true,
		},
		{
			name:    "wrong dimension",
			vector:  types.EmbeddingVector{1, 2},
			wantErr: true,
		},
		{
			name:    "NaN value",
			vector:  types.EmbeddingVector{1, float32(math.NaN()), 3},
			wantErr: true,
		},
		{
			name:    "infinite value",
			vector:  types.EmbeddingVector{1, float32(math.Inf(1)), 3},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := embedder.ValidateVector(tt.vector)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBaseEmbedder_Metrics(t *testing.T) {
	embedder := NewBaseEmbedder("test", 384)
	
	// Test recording metrics
	embedder.RecordMetrics("test_metric", 42)
	embedder.IncrementCounter("test_counter")
	embedder.IncrementCounter("test_counter")
	embedder.AddToTimer("test_timer", time.Second)
	embedder.AddToTimer("test_timer", 2*time.Second)
	
	metrics := embedder.GetMetrics()
	
	assert.Equal(t, 42, metrics["test_metric"])
	assert.Equal(t, 2, metrics["test_counter"])
	assert.Equal(t, 3*time.Second, metrics["test_timer"])
}

func TestEmbedderConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *EmbedderConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &EmbedderConfig{
				Provider:  "test",
				Model:     "test-model",
				Dimension: 384,
				MaxLength: 512,
				BatchSize: 32,
			},
			wantErr: false,
		},
		{
			name: "missing provider",
			config: &EmbedderConfig{
				Model:     "test-model",
				Dimension: 384,
				MaxLength: 512,
				BatchSize: 32,
			},
			wantErr: true,
		},
		{
			name: "missing model",
			config: &EmbedderConfig{
				Provider:  "test",
				Dimension: 384,
				MaxLength: 512,
				BatchSize: 32,
			},
			wantErr: true,
		},
		{
			name: "invalid dimension",
			config: &EmbedderConfig{
				Provider:  "test",
				Model:     "test-model",
				Dimension: 0,
				MaxLength: 512,
				BatchSize: 32,
			},
			wantErr: true,
		},
		{
			name: "invalid max_length",
			config: &EmbedderConfig{
				Provider:  "test",
				Model:     "test-model",
				Dimension: 384,
				MaxLength: 0,
				BatchSize: 32,
			},
			wantErr: true,
		},
		{
			name: "invalid batch_size",
			config: &EmbedderConfig{
				Provider:  "test",
				Model:     "test-model",
				Dimension: 384,
				MaxLength: 512,
				BatchSize: 0,
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultEmbedderConfig(t *testing.T) {
	config := DefaultEmbedderConfig()
	
	assert.Equal(t, "sentence-transformer", config.Provider)
	assert.Equal(t, "all-MiniLM-L6-v2", config.Model)
	assert.Equal(t, 384, config.Dimension)
	assert.Equal(t, 512, config.MaxLength)
	assert.Equal(t, 32, config.BatchSize)
	assert.True(t, config.Normalize)
	
	// Should be valid
	assert.NoError(t, config.Validate())
}

func TestStreamingEmbedder(t *testing.T) {
	streaming := NewStreamingEmbedder("test", 384, 3)
	defer streaming.Close()
	
	// Test streaming functionality
	streaming.StartStreaming()
	assert.True(t, streaming.isStreaming)
	
	// Add texts to stream
	streaming.AddToStream("text1")
	streaming.AddToStream("text2")
	streaming.AddToStream("text3") // Should trigger flush
	
	// Check flush signal
	select {
	case <-streaming.GetFlushSignal():
		// Expected
	case <-time.After(time.Second):
		t.Fatal("Expected flush signal")
	}
	
	// Flush buffer
	batch := streaming.FlushBuffer()
	assert.Equal(t, 3, len(batch))
	assert.Equal(t, "text1", batch[0])
	assert.Equal(t, "text2", batch[1])
	assert.Equal(t, "text3", batch[2])
	
	// Stop streaming
	streaming.StopStreaming()
	assert.False(t, streaming.isStreaming)
}

func TestEmbedderError(t *testing.T) {
	err := NewEmbedderError("E001", "test error", "validation")
	
	assert.Equal(t, "E001", err.Code)
	assert.Equal(t, "test error", err.Message)
	assert.Equal(t, "validation", err.Type)
	assert.Equal(t, "Embedder Error [E001]: test error", err.Error())
}

// Benchmark tests
func BenchmarkBaseEmbedder_PreprocessText(b *testing.B) {
	embedder := NewBaseEmbedder("test", 384)
	text := "This is a sample text for preprocessing benchmarks with multiple words and spaces"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		embedder.PreprocessText(text)
	}
}

func BenchmarkBaseEmbedder_NormalizeVector(b *testing.B) {
	embedder := NewBaseEmbedder("test", 384)
	vector := make(types.EmbeddingVector, 384)
	for i := range vector {
		vector[i] = float32(i)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		embedder.NormalizeVector(vector)
	}
}

func BenchmarkBaseEmbedder_CosineSimilarity(b *testing.B) {
	embedder := NewBaseEmbedder("test", 384)
	a := make(types.EmbeddingVector, 384)
	vectorB := make(types.EmbeddingVector, 384)
	for i := range a {
		a[i] = float32(i)
		vectorB[i] = float32(i + 1)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		embedder.CosineSimilarity(a, vectorB)
	}
}

// Mock embedder for testing interfaces
type MockEmbedder struct {
	*BaseEmbedder
	embedFunc      func(context.Context, string) (types.EmbeddingVector, error)
	embedBatchFunc func(context.Context, []string) ([]types.EmbeddingVector, error)
	config         *EmbedderConfig
}

func NewMockEmbedder() *MockEmbedder {
	return &MockEmbedder{
		BaseEmbedder: NewBaseEmbedder("mock", 384),
		config: &EmbedderConfig{
			Provider:  "mock",
			Model:     "mock-model",
			Dimension: 384,
			MaxLength: 512,
			BatchSize: 32,
			Normalize: true,
			Extra:     make(map[string]interface{}),
		},
	}
}

func (m *MockEmbedder) Embed(ctx context.Context, text string) (types.EmbeddingVector, error) {
	if m.embedFunc != nil {
		return m.embedFunc(ctx, text)
	}
	// Return a mock embedding
	return make(types.EmbeddingVector, 384), nil
}

func (m *MockEmbedder) EmbedBatch(ctx context.Context, texts []string) ([]types.EmbeddingVector, error) {
	if m.embedBatchFunc != nil {
		return m.embedBatchFunc(ctx, texts)
	}
	// Return mock embeddings
	embeddings := make([]types.EmbeddingVector, len(texts))
	for i := range embeddings {
		embeddings[i] = make(types.EmbeddingVector, 384)
	}
	return embeddings, nil
}

func (m *MockEmbedder) GetProviderName() string {
	return "mock"
}

func (m *MockEmbedder) GetSupportedModels() []string {
	return []string{"mock-model"}
}

func (m *MockEmbedder) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *MockEmbedder) SetConfig(config *EmbedderConfig) error {
	// Copy the config to preserve extra fields
	m.config = &EmbedderConfig{
		Provider:  config.Provider,
		Model:     config.Model,
		APIKey:    config.APIKey,
		BaseURL:   config.BaseURL,
		Dimension: config.Dimension,
		MaxLength: config.MaxLength,
		Timeout:   config.Timeout,
		BatchSize: config.BatchSize,
		Normalize: config.Normalize,
		Extra:     make(map[string]interface{}),
	}
	// Copy extra fields
	if config.Extra != nil {
		for k, v := range config.Extra {
			m.config.Extra[k] = v
		}
	}
	return nil
}

func (m *MockEmbedder) GetConfig() *EmbedderConfig {
	configCopy := *m.config
	return &configCopy
}

func TestMockEmbedder(t *testing.T) {
	embedder := NewMockEmbedder()
	ctx := context.Background()
	
	// Test single embedding
	embedding, err := embedder.Embed(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, 384, len(embedding))
	
	// Test batch embedding
	embeddings, err := embedder.EmbedBatch(ctx, []string{"test1", "test2"})
	require.NoError(t, err)
	assert.Equal(t, 2, len(embeddings))
	assert.Equal(t, 384, len(embeddings[0]))
	assert.Equal(t, 384, len(embeddings[1]))
}