package chunkers

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/memtensor/memgos/pkg/interfaces"
)

// SentenceChunker implements sentence-based text chunking
type SentenceChunker struct {
	config *ChunkerConfig
	logger interfaces.Logger
	
	// sentenceBoundaryRegex for detecting sentence boundaries
	sentenceBoundaryRegex *regexp.Regexp
	
	// Simple token estimation function (approximation)
	tokenEstimator func(string) int
}

// NewSentenceChunker creates a new sentence-based chunker
func NewSentenceChunker(config *ChunkerConfig) (*SentenceChunker, error) {
	if config == nil {
		config = DefaultChunkerConfig()
	}
	
	// Initialize sentence boundary regex for English and basic punctuation
	sentenceRegex, err := regexp.Compile(`[.!?]+\s+`)
	if err != nil {
		return nil, fmt.Errorf("failed to compile sentence boundary regex: %w", err)
	}
	
	chunker := &SentenceChunker{
		config:                config,
		sentenceBoundaryRegex: sentenceRegex,
		tokenEstimator:        defaultTokenEstimator,
	}
	
	return chunker, nil
}

// Chunk splits text into sentence-based chunks
func (sc *SentenceChunker) Chunk(ctx context.Context, text string) ([]*Chunk, error) {
	return sc.ChunkWithMetadata(ctx, text, nil)
}

// ChunkWithMetadata splits text into chunks with additional metadata
func (sc *SentenceChunker) ChunkWithMetadata(ctx context.Context, text string, metadata map[string]interface{}) ([]*Chunk, error) {
	if text == "" {
		return []*Chunk{}, nil
	}
	
	startTime := time.Now()
	
	// Extract sentences from text
	sentences := sc.extractSentences(text)
	if len(sentences) == 0 {
		// If no sentences found, treat entire text as one sentence
		sentences = []string{strings.TrimSpace(text)}
	}
	
	chunks := []*Chunk{}
	currentChunk := &Chunk{
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
	}
	
	// Copy metadata to chunk
	if metadata != nil {
		for k, v := range metadata {
			currentChunk.Metadata[k] = v
		}
	}
	
	var currentSentences []string
	currentTokenCount := 0
	textPosition := 0
	
	for i, sentence := range sentences {
		sentenceTokens := sc.EstimateTokens(sentence)
		
		// Check if adding this sentence would exceed chunk size
		if currentTokenCount+sentenceTokens > sc.config.ChunkSize && len(currentSentences) >= sc.config.MinSentencesPerChunk {
			// Finalize current chunk
			sc.finalizeChunk(currentChunk, currentSentences, text, textPosition)
			chunks = append(chunks, currentChunk)
			
			// Start new chunk with overlap if configured
			overlapSentences := sc.calculateOverlap(currentSentences)
			currentChunk = &Chunk{
				Metadata:  make(map[string]interface{}),
				CreatedAt: time.Now(),
			}
			
			// Copy metadata to new chunk
			if metadata != nil {
				for k, v := range metadata {
					currentChunk.Metadata[k] = v
				}
			}
			
			currentSentences = overlapSentences
			currentTokenCount = sc.estimateTokensForSentences(overlapSentences)
			
			// Update text position for overlap
			if len(overlapSentences) > 0 {
				textPosition = sc.findTextPosition(text, overlapSentences[0])
			}
		}
		
		// Add current sentence to chunk
		currentSentences = append(currentSentences, sentence)
		currentTokenCount += sentenceTokens
		
		// Check if we've reached maximum sentences per chunk
		if len(currentSentences) >= sc.config.MaxSentencesPerChunk {
			sc.finalizeChunk(currentChunk, currentSentences, text, textPosition)
			chunks = append(chunks, currentChunk)
			
			// Start new chunk if there are more sentences
			if i < len(sentences)-1 {
				overlapSentences := sc.calculateOverlap(currentSentences)
				currentChunk = &Chunk{
					Metadata:  make(map[string]interface{}),
					CreatedAt: time.Now(),
				}
				
				// Copy metadata to new chunk
				if metadata != nil {
					for k, v := range metadata {
						currentChunk.Metadata[k] = v
					}
				}
				
				currentSentences = overlapSentences
				currentTokenCount = sc.estimateTokensForSentences(overlapSentences)
			} else {
				currentSentences = []string{}
				currentTokenCount = 0
			}
		}
	}
	
	// Add final chunk if it has content
	if len(currentSentences) > 0 {
		sc.finalizeChunk(currentChunk, currentSentences, text, textPosition)
		chunks = append(chunks, currentChunk)
	}
	
	// Add chunking statistics to metadata
	processingTime := time.Since(startTime)
	stats := CalculateStats(chunks, len(text), processingTime)
	
	for _, chunk := range chunks {
		chunk.Metadata["chunking_stats"] = stats
		chunk.Metadata["chunker_type"] = string(ChunkerTypeSentence)
		chunk.Metadata["chunk_config"] = sc.config
	}
	
	return chunks, nil
}

// extractSentences extracts sentences from text using regex and heuristics
func (sc *SentenceChunker) extractSentences(text string) []string {
	// Clean and normalize text
	text = strings.TrimSpace(text)
	text = sc.normalizeWhitespace(text)
	
	// Split by sentence boundaries
	parts := sc.sentenceBoundaryRegex.Split(text, -1)
	
	sentences := make([]string, 0, len(parts))
	
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		// Add back the sentence terminator if not the last part
		if i < len(parts)-1 {
			// Find the original terminator
			nextPartStart := strings.Index(text, part) + len(part)
			if nextPartStart < len(text) {
				for nextPartStart < len(text) && unicode.IsSpace(rune(text[nextPartStart])) {
					nextPartStart++
				}
				if nextPartStart > 0 {
					terminator := text[len(part):nextPartStart]
					terminator = strings.TrimSpace(terminator)
					if terminator != "" {
						part += terminator
					}
				}
			}
		}
		
		sentences = append(sentences, part)
	}
	
	// Handle edge case where no sentence boundaries were found
	if len(sentences) == 0 && text != "" {
		sentences = []string{text}
	}
	
	return sentences
}

// normalizeWhitespace normalizes whitespace in text
func (sc *SentenceChunker) normalizeWhitespace(text string) string {
	// Replace multiple whitespace with single space
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(text, " ")
}

// finalizeChunk completes a chunk with calculated positions and text
func (sc *SentenceChunker) finalizeChunk(chunk *Chunk, sentences []string, originalText string, startPos int) {
	chunk.Sentences = make([]string, len(sentences))
	copy(chunk.Sentences, sentences)
	chunk.Text = strings.Join(sentences, " ")
	chunk.TokenCount = sc.EstimateTokens(chunk.Text)
	chunk.StartIndex = startPos
	chunk.EndIndex = startPos + len(chunk.Text)
	
	// Ensure end index doesn't exceed original text length
	if chunk.EndIndex > len(originalText) {
		chunk.EndIndex = len(originalText)
	}
}

// calculateOverlap determines which sentences to include in overlap
func (sc *SentenceChunker) calculateOverlap(sentences []string) []string {
	if sc.config.ChunkOverlap <= 0 || len(sentences) == 0 {
		return []string{}
	}
	
	// Calculate how many sentences to include in overlap
	_ = sc.estimateTokensForSentences(sentences) // totalTokens (unused)
	overlapTokenTarget := sc.config.ChunkOverlap
	
	overlapSentences := []string{}
	currentOverlapTokens := 0
	
	// Start from the end and work backwards
	for i := len(sentences) - 1; i >= 0; i-- {
		sentenceTokens := sc.EstimateTokens(sentences[i])
		if currentOverlapTokens+sentenceTokens <= overlapTokenTarget {
			overlapSentences = append([]string{sentences[i]}, overlapSentences...)
			currentOverlapTokens += sentenceTokens
		} else {
			break
		}
	}
	
	return overlapSentences
}

// estimateTokensForSentences estimates total tokens for a slice of sentences
func (sc *SentenceChunker) estimateTokensForSentences(sentences []string) int {
	total := 0
	for _, sentence := range sentences {
		total += sc.EstimateTokens(sentence)
	}
	return total
}

// findTextPosition finds the position of a sentence in the original text
func (sc *SentenceChunker) findTextPosition(text, sentence string) int {
	sentence = strings.TrimSpace(sentence)
	if sentence == "" {
		return 0
	}
	
	// Simple string search - in a real implementation, you might want
	// more sophisticated position tracking
	pos := strings.Index(text, sentence)
	if pos == -1 {
		return 0
	}
	return pos
}

// EstimateTokens estimates the number of tokens in text
func (sc *SentenceChunker) EstimateTokens(text string) int {
	return sc.tokenEstimator(text)
}

// GetConfig returns the current chunker configuration
func (sc *SentenceChunker) GetConfig() *ChunkerConfig {
	// Return a copy to prevent external modification
	config := *sc.config
	return &config
}

// SetConfig updates the chunker configuration
func (sc *SentenceChunker) SetConfig(config *ChunkerConfig) error {
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
	
	sc.config = config
	return nil
}

// GetChunkSize returns the configured chunk size
func (sc *SentenceChunker) GetChunkSize() int {
	return sc.config.ChunkSize
}

// GetChunkOverlap returns the configured chunk overlap
func (sc *SentenceChunker) GetChunkOverlap() int {
	return sc.config.ChunkOverlap
}

// GetSupportedLanguages returns supported languages for this chunker
func (sc *SentenceChunker) GetSupportedLanguages() []string {
	return []string{"en", "es", "fr", "de", "it", "pt"} // Basic Latin script languages
}

// defaultTokenEstimator provides a simple token estimation
// This is a rough approximation - in practice, you'd want to use
// a proper tokenizer like tiktoken or similar
func defaultTokenEstimator(text string) int {
	if text == "" {
		return 0
	}
	
	// Simple heuristic: average of 4 characters per token
	// This is roughly accurate for English text with GPT-style tokenizers
	words := strings.Fields(text)
	
	// Count words and punctuation separately
	tokenCount := len(words)
	
	// Add extra tokens for punctuation
	punctuation := strings.Count(text, ".") + strings.Count(text, ",") + 
		strings.Count(text, "!") + strings.Count(text, "?") + 
		strings.Count(text, ";") + strings.Count(text, ":")
	
	return tokenCount + punctuation/2 // Rough approximation
}