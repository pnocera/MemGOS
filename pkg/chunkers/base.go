// Package chunkers provides text chunking functionality for MemGOS
package chunkers

import (
	"context"
	"time"
)

// Chunk represents a processed text chunk with metadata
type Chunk struct {
	// Text content of the chunk
	Text string `json:"text"`
	
	// TokenCount is the number of tokens in this chunk
	TokenCount int `json:"token_count"`
	
	// Sentences contains the sentences that make up this chunk
	Sentences []string `json:"sentences"`
	
	// StartIndex is the starting position in the original text
	StartIndex int `json:"start_index"`
	
	// EndIndex is the ending position in the original text
	EndIndex int `json:"end_index"`
	
	// Metadata contains additional information about the chunk
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	
	// CreatedAt is when this chunk was created
	CreatedAt time.Time `json:"created_at"`
}

// ChunkerConfig represents base configuration for text chunkers
type ChunkerConfig struct {
	// ChunkSize is the maximum number of tokens per chunk
	ChunkSize int `json:"chunk_size"`
	
	// ChunkOverlap is the number of tokens to overlap between chunks
	ChunkOverlap int `json:"chunk_overlap"`
	
	// MinSentencesPerChunk is the minimum sentences required per chunk
	MinSentencesPerChunk int `json:"min_sentences_per_chunk"`
	
	// MaxSentencesPerChunk is the maximum sentences allowed per chunk
	MaxSentencesPerChunk int `json:"max_sentences_per_chunk"`
	
	// PreserveFormatting indicates whether to preserve original formatting
	PreserveFormatting bool `json:"preserve_formatting"`
	
	// Language specifies the language for sentence detection
	Language string `json:"language"`
}

// DefaultChunkerConfig returns a sensible default configuration
func DefaultChunkerConfig() *ChunkerConfig {
	return &ChunkerConfig{
		ChunkSize:            512,
		ChunkOverlap:         128,
		MinSentencesPerChunk: 1,
		MaxSentencesPerChunk: 100,
		PreserveFormatting:   true,
		Language:             "en",
	}
}

// Chunker defines the interface for all text chunking implementations
type Chunker interface {
	// Chunk splits text into chunks based on the chunker's strategy
	Chunk(ctx context.Context, text string) ([]*Chunk, error)
	
	// ChunkWithMetadata splits text into chunks with additional metadata
	ChunkWithMetadata(ctx context.Context, text string, metadata map[string]interface{}) ([]*Chunk, error)
	
	// EstimateTokens estimates the number of tokens in text
	EstimateTokens(text string) int
	
	// GetConfig returns the current chunker configuration
	GetConfig() *ChunkerConfig
	
	// SetConfig updates the chunker configuration
	SetConfig(config *ChunkerConfig) error
	
	// GetChunkSize returns the configured chunk size
	GetChunkSize() int
	
	// GetChunkOverlap returns the configured chunk overlap
	GetChunkOverlap() int
	
	// GetSupportedLanguages returns supported languages for this chunker
	GetSupportedLanguages() []string
}

// ChunkerType represents different chunking strategies
type ChunkerType string

const (
	// ChunkerTypeSentence splits text by sentences
	ChunkerTypeSentence ChunkerType = "sentence"
	
	// ChunkerTypeParagraph splits text by paragraphs
	ChunkerTypeParagraph ChunkerType = "paragraph"
	
	// ChunkerTypeSemantic splits text by semantic boundaries
	ChunkerTypeSemantic ChunkerType = "semantic"
	
	// ChunkerTypeFixed splits text by fixed token counts
	ChunkerTypeFixed ChunkerType = "fixed"
	
	// ChunkerTypeRecursive recursively splits text preserving structure
	ChunkerTypeRecursive ChunkerType = "recursive"
)

// SupportedChunkerTypes returns all supported chunker types
func SupportedChunkerTypes() []ChunkerType {
	return []ChunkerType{
		ChunkerTypeSentence,
		ChunkerTypeParagraph,
		ChunkerTypeSemantic,
		ChunkerTypeFixed,
		ChunkerTypeRecursive,
	}
}

// IsValidChunkerType checks if a chunker type is supported
func IsValidChunkerType(chunkerType ChunkerType) bool {
	for _, supported := range SupportedChunkerTypes() {
		if supported == chunkerType {
			return true
		}
	}
	return false
}

// ChunkingStats provides statistics about the chunking process
type ChunkingStats struct {
	// OriginalTextLength is the length of the original text
	OriginalTextLength int `json:"original_text_length"`
	
	// TotalChunks is the number of chunks created
	TotalChunks int `json:"total_chunks"`
	
	// AverageChunkSize is the average size of chunks in tokens
	AverageChunkSize float64 `json:"average_chunk_size"`
	
	// MinChunkSize is the size of the smallest chunk
	MinChunkSize int `json:"min_chunk_size"`
	
	// MaxChunkSize is the size of the largest chunk
	MaxChunkSize int `json:"max_chunk_size"`
	
	// TotalTokens is the total number of tokens across all chunks
	TotalTokens int `json:"total_tokens"`
	
	// OverlapTokens is the number of tokens that are overlapped
	OverlapTokens int `json:"overlap_tokens"`
	
	// ProcessingTime is the time taken to perform chunking
	ProcessingTime time.Duration `json:"processing_time"`
}

// CalculateStats computes statistics for a set of chunks
func CalculateStats(chunks []*Chunk, originalLength int, processingTime time.Duration) *ChunkingStats {
	if len(chunks) == 0 {
		return &ChunkingStats{
			OriginalTextLength: originalLength,
			ProcessingTime:     processingTime,
		}
	}
	
	stats := &ChunkingStats{
		OriginalTextLength: originalLength,
		TotalChunks:        len(chunks),
		ProcessingTime:     processingTime,
	}
	
	totalTokens := 0
	minSize := chunks[0].TokenCount
	maxSize := chunks[0].TokenCount
	
	for _, chunk := range chunks {
		totalTokens += chunk.TokenCount
		if chunk.TokenCount < minSize {
			minSize = chunk.TokenCount
		}
		if chunk.TokenCount > maxSize {
			maxSize = chunk.TokenCount
		}
	}
	
	stats.TotalTokens = totalTokens
	stats.MinChunkSize = minSize
	stats.MaxChunkSize = maxSize
	stats.AverageChunkSize = float64(totalTokens) / float64(len(chunks))
	
	// Calculate overlap tokens (simplified estimation)
	if len(chunks) > 1 {
		// Estimate overlap based on configuration and actual chunk positions
		// This is a simplified calculation
		stats.OverlapTokens = (len(chunks) - 1) * (totalTokens / len(chunks) / 4) // Rough estimate
	}
	
	return stats
}