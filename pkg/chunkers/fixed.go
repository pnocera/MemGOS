package chunkers

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// FixedChunker implements fixed-size text chunking
type FixedChunker struct {
	config *ChunkerConfig
	
	// Token estimation function
	tokenEstimator func(string) int
}

// NewFixedChunker creates a new fixed-size chunker
func NewFixedChunker(config *ChunkerConfig) (*FixedChunker, error) {
	if config == nil {
		config = DefaultChunkerConfig()
	}
	
	chunker := &FixedChunker{
		config:         config,
		tokenEstimator: defaultTokenEstimator,
	}
	
	return chunker, nil
}

// Chunk splits text into fixed-size chunks
func (fc *FixedChunker) Chunk(ctx context.Context, text string) ([]*Chunk, error) {
	return fc.ChunkWithMetadata(ctx, text, nil)
}

// ChunkWithMetadata splits text into chunks with additional metadata
func (fc *FixedChunker) ChunkWithMetadata(ctx context.Context, text string, metadata map[string]interface{}) ([]*Chunk, error) {
	if text == "" {
		return []*Chunk{}, nil
	}
	
	startTime := time.Now()
	
	// Tokenize text into words for more accurate chunking
	words := strings.Fields(text)
	if len(words) == 0 {
		// If no words found, treat entire text as one chunk
		chunk := &Chunk{
			Text:       text,
			TokenCount: fc.EstimateTokens(text),
			Sentences:  []string{text},
			StartIndex: 0,
			EndIndex:   len(text),
			Metadata:   make(map[string]interface{}),
			CreatedAt:  time.Now(),
		}
		
		if metadata != nil {
			for k, v := range metadata {
				chunk.Metadata[k] = v
			}
		}
		
		return []*Chunk{chunk}, nil
	}
	
	chunks := []*Chunk{}
	currentWords := []string{}
	currentTokenCount := 0
	textPosition := 0
	overlapWords := []string{}
	
	for _, word := range words {
		wordTokens := fc.EstimateTokens(word)
		
		// Check if adding this word would exceed chunk size
		if currentTokenCount+wordTokens > fc.config.ChunkSize && len(currentWords) > 0 {
			// Create chunk from current words
			chunk := fc.createChunk(currentWords, text, textPosition, metadata)
			chunks = append(chunks, chunk)
			
			// Calculate overlap for next chunk
			if fc.config.ChunkOverlap > 0 {
				overlapWords = fc.calculateWordOverlap(currentWords)
			} else {
				overlapWords = []string{}
			}
			
			// Start new chunk with overlap
			currentWords = make([]string, len(overlapWords))
			copy(currentWords, overlapWords)
			currentTokenCount = fc.estimateTokensForWords(currentWords)
			
			// Update text position (simplified calculation)
			if len(overlapWords) > 0 {
				textPosition = fc.findWordPosition(text, overlapWords[0], textPosition)
			} else {
				textPosition = fc.findWordPosition(text, word, textPosition)
			}
		}
		
		// Add current word to chunk
		currentWords = append(currentWords, word)
		currentTokenCount += wordTokens
	}
	
	// Add final chunk if it has content
	if len(currentWords) > 0 {
		chunk := fc.createChunk(currentWords, text, textPosition, metadata)
		chunks = append(chunks, chunk)
	}
	
	// Add chunking statistics to metadata
	processingTime := time.Since(startTime)
	stats := CalculateStats(chunks, len(text), processingTime)
	
	for _, chunk := range chunks {
		chunk.Metadata["chunking_stats"] = stats
		chunk.Metadata["chunker_type"] = string(ChunkerTypeFixed)
		chunk.Metadata["chunk_config"] = fc.config
		chunk.Metadata["word_count"] = len(chunk.Sentences) // Store words in sentences field
	}
	
	return chunks, nil
}

// createChunk creates a chunk from a list of words
func (fc *FixedChunker) createChunk(words []string, originalText string, startPos int, metadata map[string]interface{}) *Chunk {
	chunkText := strings.Join(words, " ")
	
	chunk := &Chunk{
		Text:       chunkText,
		TokenCount: fc.EstimateTokens(chunkText),
		Sentences:  make([]string, len(words)), // Store words in sentences field for consistency
		StartIndex: startPos,
		EndIndex:   startPos + len(chunkText),
		Metadata:   make(map[string]interface{}),
		CreatedAt:  time.Now(),
	}
	
	copy(chunk.Sentences, words)
	
	// Copy metadata
	if metadata != nil {
		for k, v := range metadata {
			chunk.Metadata[k] = v
		}
	}
	
	// Ensure end index doesn't exceed original text length
	if chunk.EndIndex > len(originalText) {
		chunk.EndIndex = len(originalText)
	}
	
	return chunk
}

// calculateWordOverlap determines which words to include in overlap
func (fc *FixedChunker) calculateWordOverlap(words []string) []string {
	if fc.config.ChunkOverlap <= 0 || len(words) == 0 {
		return []string{}
	}
	
	overlapTokenTarget := fc.config.ChunkOverlap
	overlapWords := []string{}
	currentOverlapTokens := 0
	
	// Start from the end and work backwards
	for i := len(words) - 1; i >= 0; i-- {
		wordTokens := fc.EstimateTokens(words[i])
		if currentOverlapTokens+wordTokens <= overlapTokenTarget {
			overlapWords = append([]string{words[i]}, overlapWords...)
			currentOverlapTokens += wordTokens
		} else {
			break
		}
	}
	
	return overlapWords
}

// estimateTokensForWords estimates total tokens for a slice of words
func (fc *FixedChunker) estimateTokensForWords(words []string) int {
	total := 0
	for _, word := range words {
		total += fc.EstimateTokens(word)
	}
	return total
}

// findWordPosition finds the position of a word in the original text
func (fc *FixedChunker) findWordPosition(text, word string, startFrom int) int {
	word = strings.TrimSpace(word)
	if word == "" {
		return startFrom
	}
	
	// Search for the word starting from the given position
	pos := strings.Index(text[startFrom:], word)
	if pos == -1 {
		return startFrom
	}
	return startFrom + pos
}

// EstimateTokens estimates the number of tokens in text
func (fc *FixedChunker) EstimateTokens(text string) int {
	return fc.tokenEstimator(text)
}

// GetConfig returns the current chunker configuration
func (fc *FixedChunker) GetConfig() *ChunkerConfig {
	config := *fc.config
	return &config
}

// SetConfig updates the chunker configuration
func (fc *FixedChunker) SetConfig(config *ChunkerConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	
	if config.ChunkSize <= 0 {
		return fmt.Errorf("chunk size must be positive")
	}
	
	if config.ChunkOverlap < 0 {
		return fmt.Errorf("chunk overlap cannot be negative")
	}
	
	if config.ChunkOverlap >= config.ChunkSize {
		return fmt.Errorf("chunk overlap must be less than chunk size")
	}
	
	fc.config = config
	return nil
}

// GetChunkSize returns the configured chunk size
func (fc *FixedChunker) GetChunkSize() int {
	return fc.config.ChunkSize
}

// GetChunkOverlap returns the configured chunk overlap
func (fc *FixedChunker) GetChunkOverlap() int {
	return fc.config.ChunkOverlap
}

// GetSupportedLanguages returns supported languages for this chunker
func (fc *FixedChunker) GetSupportedLanguages() []string {
	return []string{"*"} // Universal - works with any language
}