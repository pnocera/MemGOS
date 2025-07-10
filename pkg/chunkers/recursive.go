package chunkers

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// RecursiveChunker implements recursive text chunking that preserves hierarchical structure
type RecursiveChunker struct {
	config *ChunkerConfig
	
	// Token estimation function
	tokenEstimator func(string) int
	
	// Separators in order of preference (from largest to smallest structures)
	separators []string
}

// NewRecursiveChunker creates a new recursive chunker
func NewRecursiveChunker(config *ChunkerConfig) (*RecursiveChunker, error) {
	if config == nil {
		config = DefaultChunkerConfig()
	}
	
	// Define separators in order of preference
	// Start with larger structures and work down to smaller ones
	separators := []string{
		"\n\n\n",    // Multiple blank lines (major sections)
		"\n\n",      // Double newlines (paragraphs)
		"\n",        // Single newlines (lines)
		". ",        // Sentences
		"! ",        // Exclamations
		"? ",        // Questions
		"; ",        // Semicolons
		", ",        // Commas
		" ",         // Spaces (words)
		"",          // Character level (last resort)
	}
	
	chunker := &RecursiveChunker{
		config:         config,
		tokenEstimator: defaultTokenEstimator,
		separators:     separators,
	}
	
	return chunker, nil
}

// Chunk splits text into chunks using recursive strategy
func (rc *RecursiveChunker) Chunk(ctx context.Context, text string) ([]*Chunk, error) {
	return rc.ChunkWithMetadata(ctx, text, nil)
}

// ChunkWithMetadata splits text into chunks with additional metadata
func (rc *RecursiveChunker) ChunkWithMetadata(ctx context.Context, text string, metadata map[string]interface{}) ([]*Chunk, error) {
	if text == "" {
		return []*Chunk{}, nil
	}
	
	startTime := time.Now()
	
	// Start recursive chunking process
	chunks := rc.recursiveChunk(text, 0, 0, metadata)
	
	// Add chunking statistics to metadata
	processingTime := time.Since(startTime)
	stats := CalculateStats(chunks, len(text), processingTime)
	
	for _, chunk := range chunks {
		chunk.Metadata["chunking_stats"] = stats
		chunk.Metadata["chunker_type"] = string(ChunkerTypeRecursive)
		chunk.Metadata["chunk_config"] = rc.config
	}
	
	return chunks, nil
}

// recursiveChunk recursively splits text using hierarchical separators
func (rc *RecursiveChunker) recursiveChunk(text string, level int, textOffset int, metadata map[string]interface{}) []*Chunk {
	// Check if text is small enough to be a single chunk
	if rc.EstimateTokens(text) <= rc.config.ChunkSize {
		chunk := &Chunk{
			Text:       text,
			TokenCount: rc.EstimateTokens(text),
			Sentences:  []string{text},
			StartIndex: textOffset,
			EndIndex:   textOffset + len(text),
			Metadata:   make(map[string]interface{}),
			CreatedAt:  time.Now(),
		}
		
		// Copy metadata
		if metadata != nil {
			for k, v := range metadata {
				chunk.Metadata[k] = v
			}
		}
		
		// Add hierarchical metadata
		chunk.Metadata["hierarchy_level"] = level
		chunk.Metadata["separator_used"] = rc.getSeparatorName(level)
		
		return []*Chunk{chunk}
	}
	
	// Find the best separator for this level
	separator := rc.getSeparatorForLevel(level)
	if separator == "" {
		// No more separators, split by character count
		return rc.splitByCharacterCount(text, textOffset, level, metadata)
	}
	
	// Split text by the current separator
	parts := rc.splitTextBySeparator(text, separator)
	if len(parts) <= 1 {
		// Current separator didn't split the text, try next level
		return rc.recursiveChunk(text, level+1, textOffset, metadata)
	}
	
	// Process parts and combine them into chunks
	chunks := []*Chunk{}
	currentParts := []string{}
	currentTokenCount := 0
	currentOffset := textOffset
	
	for _, part := range parts {
		if part == "" {
			continue
		}
		
		partTokens := rc.EstimateTokens(part)
		
		// Check if adding this part would exceed chunk size
		if currentTokenCount+partTokens > rc.config.ChunkSize && len(currentParts) > 0 {
			// Process current accumulated parts
			combinedText := strings.Join(currentParts, separator)
			if rc.EstimateTokens(combinedText) > rc.config.ChunkSize {
				// Still too large, recursively chunk the combined text
				subChunks := rc.recursiveChunk(combinedText, level+1, currentOffset, metadata)
				chunks = append(chunks, subChunks...)
			} else {
				// Create chunk from combined parts
				chunk := rc.createChunkFromParts(currentParts, separator, currentOffset, level, metadata)
				chunks = append(chunks, chunk)
			}
			
			// Start new accumulation with overlap if configured
			if rc.config.ChunkOverlap > 0 && len(currentParts) > 0 {
				overlapParts := rc.calculatePartsOverlap(currentParts, separator)
				currentParts = overlapParts
				currentTokenCount = rc.estimateTokensForParts(overlapParts, separator)
			} else {
				currentParts = []string{}
				currentTokenCount = 0
			}
			
			// Update offset
			currentOffset = rc.findPartPosition(text, part, currentOffset)
		}
		
		// Add current part
		currentParts = append(currentParts, part)
		currentTokenCount += partTokens
	}
	
	// Process remaining parts
	if len(currentParts) > 0 {
		combinedText := strings.Join(currentParts, separator)
		if rc.EstimateTokens(combinedText) > rc.config.ChunkSize {
			// Still too large, recursively chunk
			subChunks := rc.recursiveChunk(combinedText, level+1, currentOffset, metadata)
			chunks = append(chunks, subChunks...)
		} else {
			// Create final chunk
			chunk := rc.createChunkFromParts(currentParts, separator, currentOffset, level, metadata)
			chunks = append(chunks, chunk)
		}
	}
	
	return chunks
}

// getSeparatorForLevel returns the appropriate separator for the given level
func (rc *RecursiveChunker) getSeparatorForLevel(level int) string {
	if level >= len(rc.separators) {
		return "" // No more separators available
	}
	return rc.separators[level]
}

// getSeparatorName returns a human-readable name for the separator at the given level
func (rc *RecursiveChunker) getSeparatorName(level int) string {
	names := []string{
		"major_sections",
		"paragraphs",
		"lines",
		"sentences",
		"exclamations",
		"questions",
		"semicolons",
		"commas",
		"words",
		"characters",
	}
	
	if level >= len(names) {
		return "unknown"
	}
	return names[level]
}

// splitTextBySeparator splits text by a separator while preserving the separator
func (rc *RecursiveChunker) splitTextBySeparator(text, separator string) []string {
	if separator == "" {
		// Character-level split
		return strings.Split(text, "")
	}
	
	// Split while preserving separator context
	parts := strings.Split(text, separator)
	
	// For meaningful separators, add them back to maintain structure
	if separator != " " && separator != "" && len(parts) > 1 {
		result := make([]string, 0, len(parts)*2-1)
		for idx, part := range parts {
			if idx > 0 && part != "" {
				result = append(result, separator)
			}
			if part != "" {
				result = append(result, part)
			}
		}
		return result
	}
	
	return parts
}

// splitByCharacterCount splits text by character count as last resort
func (rc *RecursiveChunker) splitByCharacterCount(text string, textOffset, level int, metadata map[string]interface{}) []*Chunk {
	chunks := []*Chunk{}
	chunkSize := rc.config.ChunkSize * 4 // Rough character estimate (4 chars per token)
	overlapSize := rc.config.ChunkOverlap * 4
	
	for i := 0; i < len(text); i += chunkSize - overlapSize {
		end := i + chunkSize
		if end > len(text) {
			end = len(text)
		}
		
		chunkText := text[i:end]
		chunk := &Chunk{
			Text:       chunkText,
			TokenCount: rc.EstimateTokens(chunkText),
			Sentences:  []string{chunkText},
			StartIndex: textOffset + i,
			EndIndex:   textOffset + end,
			Metadata:   make(map[string]interface{}),
			CreatedAt:  time.Now(),
		}
		
		// Copy metadata
		if metadata != nil {
			for k, v := range metadata {
				chunk.Metadata[k] = v
			}
		}
		
		// Add hierarchical metadata
		chunk.Metadata["hierarchy_level"] = level
		chunk.Metadata["separator_used"] = "character_count"
		chunk.Metadata["forced_split"] = true
		
		chunks = append(chunks, chunk)
		
		// Break if we've reached the end
		if end >= len(text) {
			break
		}
	}
	
	return chunks
}

// createChunkFromParts creates a chunk from text parts
func (rc *RecursiveChunker) createChunkFromParts(parts []string, separator string, startOffset, level int, metadata map[string]interface{}) *Chunk {
	chunkText := strings.Join(parts, separator)
	
	chunk := &Chunk{
		Text:       chunkText,
		TokenCount: rc.EstimateTokens(chunkText),
		Sentences:  make([]string, len(parts)),
		StartIndex: startOffset,
		EndIndex:   startOffset + len(chunkText),
		Metadata:   make(map[string]interface{}),
		CreatedAt:  time.Now(),
	}
	
	copy(chunk.Sentences, parts)
	
	// Copy metadata
	if metadata != nil {
		for k, v := range metadata {
			chunk.Metadata[k] = v
		}
	}
	
	// Add hierarchical metadata
	chunk.Metadata["hierarchy_level"] = level
	chunk.Metadata["separator_used"] = rc.getSeparatorName(level)
	chunk.Metadata["part_count"] = len(parts)
	
	return chunk
}

// calculatePartsOverlap determines which parts to include in overlap
func (rc *RecursiveChunker) calculatePartsOverlap(parts []string, separator string) []string {
	if rc.config.ChunkOverlap <= 0 || len(parts) == 0 {
		return []string{}
	}
	
	overlapTokenTarget := rc.config.ChunkOverlap
	overlapParts := []string{}
	currentOverlapTokens := 0
	
	// Start from the end and work backwards
	for i := len(parts) - 1; i >= 0; i-- {
		partTokens := rc.EstimateTokens(parts[i])
		if currentOverlapTokens+partTokens <= overlapTokenTarget {
			overlapParts = append([]string{parts[i]}, overlapParts...)
			currentOverlapTokens += partTokens
		} else {
			break
		}
	}
	
	return overlapParts
}

// estimateTokensForParts estimates total tokens for a slice of parts
func (rc *RecursiveChunker) estimateTokensForParts(parts []string, separator string) int {
	combinedText := strings.Join(parts, separator)
	return rc.EstimateTokens(combinedText)
}

// findPartPosition finds the position of a part in the original text
func (rc *RecursiveChunker) findPartPosition(text, part string, startFrom int) int {
	part = strings.TrimSpace(part)
	if part == "" {
		return startFrom
	}
	
	pos := strings.Index(text[startFrom:], part)
	if pos == -1 {
		return startFrom
	}
	return startFrom + pos
}

// EstimateTokens estimates the number of tokens in text
func (rc *RecursiveChunker) EstimateTokens(text string) int {
	return rc.tokenEstimator(text)
}

// GetConfig returns the current chunker configuration
func (rc *RecursiveChunker) GetConfig() *ChunkerConfig {
	config := *rc.config
	return &config
}

// SetConfig updates the chunker configuration
func (rc *RecursiveChunker) SetConfig(config *ChunkerConfig) error {
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
	
	rc.config = config
	return nil
}

// GetChunkSize returns the configured chunk size
func (rc *RecursiveChunker) GetChunkSize() int {
	return rc.config.ChunkSize
}

// GetChunkOverlap returns the configured chunk overlap
func (rc *RecursiveChunker) GetChunkOverlap() int {
	return rc.config.ChunkOverlap
}

// GetSupportedLanguages returns supported languages for this chunker
func (rc *RecursiveChunker) GetSupportedLanguages() []string {
	return []string{"*"} // Universal - works with any language due to hierarchical approach
}

// SetSeparators allows customization of the separator hierarchy
func (rc *RecursiveChunker) SetSeparators(separators []string) error {
	if len(separators) == 0 {
		return fmt.Errorf("separators list cannot be empty")
	}
	
	rc.separators = make([]string, len(separators))
	copy(rc.separators, separators)
	return nil
}

// GetSeparators returns the current separator hierarchy
func (rc *RecursiveChunker) GetSeparators() []string {
	result := make([]string, len(rc.separators))
	copy(result, rc.separators)
	return result
}