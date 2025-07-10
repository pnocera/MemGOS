package chunkers

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ParagraphChunker implements paragraph-based text chunking
type ParagraphChunker struct {
	config *ChunkerConfig
	
	// paragraphBoundaryRegex for detecting paragraph boundaries
	paragraphBoundaryRegex *regexp.Regexp
	
	// Token estimation function
	tokenEstimator func(string) int
}

// NewParagraphChunker creates a new paragraph-based chunker
func NewParagraphChunker(config *ChunkerConfig) (*ParagraphChunker, error) {
	if config == nil {
		config = DefaultChunkerConfig()
	}
	
	// Initialize paragraph boundary regex (double newlines, markdown breaks, etc.)
	paragraphRegex, err := regexp.Compile(`\n\s*\n|\r\n\s*\r\n`)
	if err != nil {
		return nil, fmt.Errorf("failed to compile paragraph boundary regex: %w", err)
	}
	
	chunker := &ParagraphChunker{
		config:                 config,
		paragraphBoundaryRegex: paragraphRegex,
		tokenEstimator:         defaultTokenEstimator,
	}
	
	return chunker, nil
}

// Chunk splits text into paragraph-based chunks
func (pc *ParagraphChunker) Chunk(ctx context.Context, text string) ([]*Chunk, error) {
	return pc.ChunkWithMetadata(ctx, text, nil)
}

// ChunkWithMetadata splits text into chunks with additional metadata
func (pc *ParagraphChunker) ChunkWithMetadata(ctx context.Context, text string, metadata map[string]interface{}) ([]*Chunk, error) {
	if text == "" {
		return []*Chunk{}, nil
	}
	
	startTime := time.Now()
	
	// Extract paragraphs from text
	paragraphs := pc.extractParagraphs(text)
	if len(paragraphs) == 0 {
		paragraphs = []string{strings.TrimSpace(text)}
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
	
	var currentParagraphs []string
	currentTokenCount := 0
	textPosition := 0
	
	for _, paragraph := range paragraphs {
		paragraphTokens := pc.EstimateTokens(paragraph)
		
		// Check if adding this paragraph would exceed chunk size
		if currentTokenCount+paragraphTokens > pc.config.ChunkSize && len(currentParagraphs) > 0 {
			// Finalize current chunk
			pc.finalizeChunk(currentChunk, currentParagraphs, text, textPosition)
			chunks = append(chunks, currentChunk)
			
			// Start new chunk with overlap if configured
			overlapParagraphs := pc.calculateOverlap(currentParagraphs)
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
			
			currentParagraphs = overlapParagraphs
			currentTokenCount = pc.estimateTokensForParagraphs(overlapParagraphs)
			
			// Update text position for overlap
			if len(overlapParagraphs) > 0 {
				textPosition = pc.findTextPosition(text, overlapParagraphs[0])
			}
		}
		
		// Add current paragraph to chunk
		currentParagraphs = append(currentParagraphs, paragraph)
		currentTokenCount += paragraphTokens
	}
	
	// Add final chunk if it has content
	if len(currentParagraphs) > 0 {
		pc.finalizeChunk(currentChunk, currentParagraphs, text, textPosition)
		chunks = append(chunks, currentChunk)
	}
	
	// Add chunking statistics to metadata
	processingTime := time.Since(startTime)
	stats := CalculateStats(chunks, len(text), processingTime)
	
	for _, chunk := range chunks {
		chunk.Metadata["chunking_stats"] = stats
		chunk.Metadata["chunker_type"] = string(ChunkerTypeParagraph)
		chunk.Metadata["chunk_config"] = pc.config
		chunk.Metadata["paragraph_count"] = len(chunk.Sentences) // Reuse sentences field for paragraphs
	}
	
	return chunks, nil
}

// extractParagraphs extracts paragraphs from text
func (pc *ParagraphChunker) extractParagraphs(text string) []string {
	// Clean and normalize text
	text = strings.TrimSpace(text)
	
	// Split by paragraph boundaries
	parts := pc.paragraphBoundaryRegex.Split(text, -1)
	
	paragraphs := make([]string, 0, len(parts))
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		paragraphs = append(paragraphs, part)
	}
	
	// Handle edge case where no paragraph boundaries were found
	if len(paragraphs) == 0 && text != "" {
		paragraphs = []string{text}
	}
	
	return paragraphs
}

// finalizeChunk completes a chunk with calculated positions and text
func (pc *ParagraphChunker) finalizeChunk(chunk *Chunk, paragraphs []string, originalText string, startPos int) {
	chunk.Sentences = make([]string, len(paragraphs)) // Store paragraphs in sentences field
	copy(chunk.Sentences, paragraphs)
	chunk.Text = strings.Join(paragraphs, "\n\n") // Join with paragraph separators
	chunk.TokenCount = pc.EstimateTokens(chunk.Text)
	chunk.StartIndex = startPos
	chunk.EndIndex = startPos + len(chunk.Text)
	
	// Ensure end index doesn't exceed original text length
	if chunk.EndIndex > len(originalText) {
		chunk.EndIndex = len(originalText)
	}
}

// calculateOverlap determines which paragraphs to include in overlap
func (pc *ParagraphChunker) calculateOverlap(paragraphs []string) []string {
	if pc.config.ChunkOverlap <= 0 || len(paragraphs) == 0 {
		return []string{}
	}
	
	// Calculate how many paragraphs to include in overlap
	overlapTokenTarget := pc.config.ChunkOverlap
	
	overlapParagraphs := []string{}
	currentOverlapTokens := 0
	
	// Start from the end and work backwards
	for i := len(paragraphs) - 1; i >= 0; i-- {
		paragraphTokens := pc.EstimateTokens(paragraphs[i])
		if currentOverlapTokens+paragraphTokens <= overlapTokenTarget {
			overlapParagraphs = append([]string{paragraphs[i]}, overlapParagraphs...)
			currentOverlapTokens += paragraphTokens
		} else {
			break
		}
	}
	
	return overlapParagraphs
}

// estimateTokensForParagraphs estimates total tokens for a slice of paragraphs
func (pc *ParagraphChunker) estimateTokensForParagraphs(paragraphs []string) int {
	total := 0
	for _, paragraph := range paragraphs {
		total += pc.EstimateTokens(paragraph)
	}
	return total
}

// findTextPosition finds the position of a paragraph in the original text
func (pc *ParagraphChunker) findTextPosition(text, paragraph string) int {
	paragraph = strings.TrimSpace(paragraph)
	if paragraph == "" {
		return 0
	}
	
	pos := strings.Index(text, paragraph)
	if pos == -1 {
		return 0
	}
	return pos
}

// EstimateTokens estimates the number of tokens in text
func (pc *ParagraphChunker) EstimateTokens(text string) int {
	return pc.tokenEstimator(text)
}

// GetConfig returns the current chunker configuration
func (pc *ParagraphChunker) GetConfig() *ChunkerConfig {
	config := *pc.config
	return &config
}

// SetConfig updates the chunker configuration
func (pc *ParagraphChunker) SetConfig(config *ChunkerConfig) error {
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
	
	pc.config = config
	return nil
}

// GetChunkSize returns the configured chunk size
func (pc *ParagraphChunker) GetChunkSize() int {
	return pc.config.ChunkSize
}

// GetChunkOverlap returns the configured chunk overlap
func (pc *ParagraphChunker) GetChunkOverlap() int {
	return pc.config.ChunkOverlap
}

// GetSupportedLanguages returns supported languages for this chunker
func (pc *ParagraphChunker) GetSupportedLanguages() []string {
	return []string{"en", "es", "fr", "de", "it", "pt", "ru", "zh", "ja", "ko"} // Universal paragraph detection
}