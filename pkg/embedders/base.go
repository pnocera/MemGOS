// Package embedders provides embedding implementations for MemGOS
package embedders

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// BaseEmbedder provides common functionality for all embedder implementations
type BaseEmbedder struct {
	modelName   string
	dimension   int
	maxLength   int
	timeout     time.Duration
	metrics     map[string]interface{}
	mu          sync.RWMutex
}

// NewBaseEmbedder creates a new base embedder instance
func NewBaseEmbedder(modelName string, dimension int) *BaseEmbedder {
	return &BaseEmbedder{
		modelName: modelName,
		dimension: dimension,
		maxLength: 512, // Default max length for most models
		timeout:   30 * time.Second,
		metrics:   make(map[string]interface{}),
	}
}

// GetDimension returns the embedding dimension
func (b *BaseEmbedder) GetDimension() int {
	return b.dimension
}

// GetModelName returns the model name
func (b *BaseEmbedder) GetModelName() string {
	return b.modelName
}

// SetMaxLength sets the maximum input length
func (b *BaseEmbedder) SetMaxLength(maxLength int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.maxLength = maxLength
}

// GetMaxLength returns the maximum input length
func (b *BaseEmbedder) GetMaxLength() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.maxLength
}

// SetTimeout sets the request timeout
func (b *BaseEmbedder) SetTimeout(timeout time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.timeout = timeout
}

// GetTimeout returns the request timeout
func (b *BaseEmbedder) GetTimeout() time.Duration {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.timeout
}

// PreprocessText preprocesses text before embedding
func (b *BaseEmbedder) PreprocessText(text string) string {
	// Trim whitespace
	text = strings.TrimSpace(text)
	
	// Replace multiple whitespaces with single space
	text = strings.Join(strings.Fields(text), " ")
	
	// Truncate if too long (simple character-based truncation)
	if len(text) > b.maxLength*4 { // Rough estimate: 4 chars per token
		text = text[:b.maxLength*4]
		// Try to cut at word boundary
		if lastSpace := strings.LastIndex(text, " "); lastSpace > b.maxLength*3 {
			text = text[:lastSpace]
		}
	}
	
	return text
}

// ChunkText splits text into chunks that fit the model's context window
func (b *BaseEmbedder) ChunkText(text string, chunkSize int, overlap int) []string {
	if chunkSize <= 0 {
		chunkSize = b.maxLength
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= chunkSize {
		overlap = chunkSize / 2
	}
	
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}
	
	var chunks []string
	start := 0
	
	for start < len(words) {
		end := start + chunkSize
		if end > len(words) {
			end = len(words)
		}
		
		chunk := strings.Join(words[start:end], " ")
		chunks = append(chunks, chunk)
		
		if end == len(words) {
			break
		}
		
		start = end - overlap
		if start < 0 {
			start = 0
		}
	}
	
	return chunks
}

// NormalizeVector normalizes an embedding vector to unit length
func (b *BaseEmbedder) NormalizeVector(vector types.EmbeddingVector) types.EmbeddingVector {
	var norm float32
	for _, val := range vector {
		norm += val * val
	}
	norm = float32(math.Sqrt(float64(norm)))
	
	if norm == 0 {
		return vector
	}
	
	normalized := make(types.EmbeddingVector, len(vector))
	for i, val := range vector {
		normalized[i] = val / norm
	}
	
	return normalized
}

// CosineSimilarity calculates cosine similarity between two vectors
func (b *BaseEmbedder) CosineSimilarity(a, vectorB types.EmbeddingVector) float32 {
	if len(a) != len(vectorB) {
		return 0
	}
	
	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * vectorB[i]
		normA += a[i] * a[i]
		normB += vectorB[i] * vectorB[i]
	}
	
	if normA == 0 || normB == 0 {
		return 0
	}
	
	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// ValidateVector validates an embedding vector
func (b *BaseEmbedder) ValidateVector(vector types.EmbeddingVector) error {
	if len(vector) == 0 {
		return fmt.Errorf("embedding vector is empty")
	}
	
	if len(vector) != b.dimension {
		return fmt.Errorf("embedding dimension mismatch: expected %d, got %d", b.dimension, len(vector))
	}
	
	// Check for NaN or Inf values
	for i, val := range vector {
		if math.IsNaN(float64(val)) || math.IsInf(float64(val), 0) {
			return fmt.Errorf("invalid value at index %d: %f", i, val)
		}
	}
	
	return nil
}

// RecordMetrics records usage metrics
func (b *BaseEmbedder) RecordMetrics(metric string, value interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.metrics[metric] = value
}

// GetMetrics returns accumulated metrics
func (b *BaseEmbedder) GetMetrics() map[string]interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	// Return a copy to prevent concurrent modification
	metrics := make(map[string]interface{})
	for k, v := range b.metrics {
		metrics[k] = v
	}
	return metrics
}

// IncrementCounter increments a counter metric
func (b *BaseEmbedder) IncrementCounter(metric string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if count, ok := b.metrics[metric].(int); ok {
		b.metrics[metric] = count + 1
	} else {
		b.metrics[metric] = 1
	}
}

// AddToTimer adds a duration to a timer metric
func (b *BaseEmbedder) AddToTimer(metric string, duration time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if total, ok := b.metrics[metric].(time.Duration); ok {
		b.metrics[metric] = total + duration
	} else {
		b.metrics[metric] = duration
	}
}

// Close provides default close implementation
func (b *BaseEmbedder) Close() error {
	// Base implementation - nothing to close
	return nil
}

// EmbedderConfig represents configuration for embedder instances
type EmbedderConfig struct {
	Provider   string            `json:"provider"`
	Model      string            `json:"model"`
	APIKey     string            `json:"api_key"`
	BaseURL    string            `json:"base_url"`
	Dimension  int               `json:"dimension"`
	MaxLength  int               `json:"max_length"`
	Timeout    time.Duration     `json:"timeout"`
	BatchSize  int               `json:"batch_size"`
	Normalize  bool              `json:"normalize"`
	Extra      map[string]interface{} `json:"extra"`
}

// Validate validates the embedder configuration
func (c *EmbedderConfig) Validate() error {
	if c.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	if c.Dimension <= 0 {
		return fmt.Errorf("dimension must be positive")
	}
	if c.MaxLength <= 0 {
		return fmt.Errorf("max_length must be positive")
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("batch_size must be positive")
	}
	return nil
}

// DefaultEmbedderConfig returns default embedder configuration
func DefaultEmbedderConfig() *EmbedderConfig {
	return &EmbedderConfig{
		Provider:  "sentence-transformer",
		Model:     "all-MiniLM-L6-v2",
		Dimension: 384,
		MaxLength: 512,
		Timeout:   30 * time.Second,
		BatchSize: 32,
		Normalize: true,
		Extra:     make(map[string]interface{}),
	}
}

// EmbedderResponse represents a generic embedder response
type EmbedderResponse struct {
	Embeddings   []types.EmbeddingVector `json:"embeddings"`
	Model        string                  `json:"model"`
	Dimension    int                     `json:"dimension"`
	TokensUsed   int                     `json:"tokens_used"`
	ProcessTime  time.Duration           `json:"process_time"`
	Metadata     map[string]interface{}  `json:"metadata"`
	RequestID    string                  `json:"request_id"`
}

// EmbedderError represents an embedder-specific error
type EmbedderError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

// Error implements the error interface
func (e *EmbedderError) Error() string {
	return fmt.Sprintf("Embedder Error [%s]: %s", e.Code, e.Message)
}

// NewEmbedderError creates a new embedder error
func NewEmbedderError(code, message, errType string) *EmbedderError {
	return &EmbedderError{
		Code:    code,
		Message: message,
		Type:    errType,
	}
}

// EmbedderProvider defines the interface for embedder provider implementations
type EmbedderProvider interface {
	interfaces.Embedder
	GetProviderName() string
	GetSupportedModels() []string
	HealthCheck(ctx context.Context) error
	SetConfig(config *EmbedderConfig) error
	GetConfig() *EmbedderConfig
}

// StreamingEmbedder provides streaming capabilities for embedder implementations
type StreamingEmbedder struct {
	*BaseEmbedder
	batchBuffer []string
	batchSize   int
	flushTimer  *time.Timer
	flushChan   chan struct{}
	resultChan  chan []types.EmbeddingVector
	isStreaming bool
}

// NewStreamingEmbedder creates a new streaming embedder instance
func NewStreamingEmbedder(modelName string, dimension int, batchSize int) *StreamingEmbedder {
	return &StreamingEmbedder{
		BaseEmbedder: NewBaseEmbedder(modelName, dimension),
		batchBuffer:  make([]string, 0, batchSize),
		batchSize:    batchSize,
		flushChan:    make(chan struct{}, 1),
		resultChan:   make(chan []types.EmbeddingVector, 10),
		isStreaming:  false,
	}
}

// StartStreaming starts streaming mode
func (s *StreamingEmbedder) StartStreaming() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isStreaming = true
}

// StopStreaming stops streaming mode and flushes remaining data
func (s *StreamingEmbedder) StopStreaming() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.isStreaming {
		s.isStreaming = false
		if s.flushTimer != nil {
			s.flushTimer.Stop()
		}
		
		// Flush remaining buffer
		if len(s.batchBuffer) > 0 {
			select {
			case s.flushChan <- struct{}{}:
			default:
			}
		}
	}
}

// AddToStream adds text to the streaming buffer
func (s *StreamingEmbedder) AddToStream(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.isStreaming {
		return
	}
	
	s.batchBuffer = append(s.batchBuffer, text)
	
	// Trigger flush if buffer is full
	if len(s.batchBuffer) >= s.batchSize {
		select {
		case s.flushChan <- struct{}{}:
		default:
		}
	} else {
		// Reset flush timer
		if s.flushTimer != nil {
			s.flushTimer.Stop()
		}
		s.flushTimer = time.AfterFunc(time.Second, func() {
			select {
			case s.flushChan <- struct{}{}:
			default:
			}
		})
	}
}

// GetResults returns the result channel for streaming embeddings
func (s *StreamingEmbedder) GetResults() <-chan []types.EmbeddingVector {
	return s.resultChan
}

// GetFlushSignal returns the flush signal channel
func (s *StreamingEmbedder) GetFlushSignal() <-chan struct{} {
	return s.flushChan
}

// FlushBuffer returns and clears the current buffer
func (s *StreamingEmbedder) FlushBuffer() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if len(s.batchBuffer) == 0 {
		return nil
	}
	
	batch := make([]string, len(s.batchBuffer))
	copy(batch, s.batchBuffer)
	s.batchBuffer = s.batchBuffer[:0]
	
	return batch
}

// Close closes the streaming embedder
func (s *StreamingEmbedder) Close() error {
	s.StopStreaming()
	close(s.flushChan)
	close(s.resultChan)
	return s.BaseEmbedder.Close()
}