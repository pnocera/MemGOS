package chunkers

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SemanticChunker implements semantic-aware text chunking
// This is a simplified implementation that uses sentence boundaries
// with semantic heuristics. A full implementation would use
// transformer models for semantic similarity
type SemanticChunker struct {
	config *ChunkerConfig
	
	// Token estimation function
	tokenEstimator func(string) int
	
	// Sentence chunker for initial splitting
	sentenceChunker *SentenceChunker
}

// NewSemanticChunker creates a new semantic-aware chunker
func NewSemanticChunker(config *ChunkerConfig) (*SemanticChunker, error) {
	if config == nil {
		config = DefaultChunkerConfig()
	}
	
	// Create a sentence chunker for initial processing
	sentenceChunker, err := NewSentenceChunker(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create sentence chunker: %w", err)
	}
	
	chunker := &SemanticChunker{
		config:          config,
		tokenEstimator:  defaultTokenEstimator,
		sentenceChunker: sentenceChunker,
	}
	
	return chunker, nil
}

// Chunk splits text into semantically coherent chunks
func (sc *SemanticChunker) Chunk(ctx context.Context, text string) ([]*Chunk, error) {
	return sc.ChunkWithMetadata(ctx, text, nil)
}

// ChunkWithMetadata splits text into chunks with additional metadata
func (sc *SemanticChunker) ChunkWithMetadata(ctx context.Context, text string, metadata map[string]interface{}) ([]*Chunk, error) {
	if text == "" {
		return []*Chunk{}, nil
	}
	
	startTime := time.Now()
	
	// Extract sentences from text
	sentences := sc.extractSentences(text)
	if len(sentences) == 0 {
		sentences = []string{strings.TrimSpace(text)}
	}
	
	// Group sentences into semantically coherent chunks
	chunks := sc.groupSentencesSemantics(sentences, text, metadata)
	
	// Add chunking statistics to metadata
	processingTime := time.Since(startTime)
	stats := CalculateStats(chunks, len(text), processingTime)
	
	for _, chunk := range chunks {
		chunk.Metadata["chunking_stats"] = stats
		chunk.Metadata["chunker_type"] = string(ChunkerTypeSemantic)
		chunk.Metadata["chunk_config"] = sc.config
		chunk.Metadata["semantic_grouping"] = true
	}
	
	return chunks, nil
}

// extractSentences extracts sentences from text
func (sc *SemanticChunker) extractSentences(text string) []string {
	return sc.sentenceChunker.extractSentences(text)
}

// groupSentencesSemantics groups sentences based on semantic heuristics
func (sc *SemanticChunker) groupSentencesSemantics(sentences []string, originalText string, metadata map[string]interface{}) []*Chunk {
	if len(sentences) == 0 {
		return []*Chunk{}
	}
	
	chunks := []*Chunk{}
	currentSentences := []string{}
	currentTokenCount := 0
	textPosition := 0
	
	for _, sentence := range sentences {
		sentenceTokens := sc.EstimateTokens(sentence)
		
		// Check if adding this sentence would exceed chunk size
		if currentTokenCount+sentenceTokens > sc.config.ChunkSize && len(currentSentences) >= sc.config.MinSentencesPerChunk {
			// Check semantic boundary before splitting
			if sc.isSemanticBoundary(currentSentences, sentence) || 
			   len(currentSentences) >= sc.config.MaxSentencesPerChunk {
				
				// Finalize current chunk
				chunk := sc.createSemanticChunk(currentSentences, originalText, textPosition, metadata)
				chunks = append(chunks, chunk)
				
				// Start new chunk with overlap if configured
				overlapSentences := sc.calculateSemanticOverlap(currentSentences, sentence)
				currentSentences = overlapSentences
				currentTokenCount = sc.estimateTokensForSentences(overlapSentences)
				
				// Update text position for overlap
				if len(overlapSentences) > 0 {
					textPosition = sc.findTextPosition(originalText, overlapSentences[0], textPosition)
				} else {
					textPosition = sc.findTextPosition(originalText, sentence, textPosition)
				}
			}
		}
		
		// Add current sentence to chunk
		currentSentences = append(currentSentences, sentence)
		currentTokenCount += sentenceTokens
	}
	
	// Add final chunk if it has content
	if len(currentSentences) > 0 {
		chunk := sc.createSemanticChunk(currentSentences, originalText, textPosition, metadata)
		chunks = append(chunks, chunk)
	}
	
	return chunks
}

// isSemanticBoundary determines if there's a semantic boundary between sentence groups
func (sc *SemanticChunker) isSemanticBoundary(currentSentences []string, nextSentence string) bool {
	if len(currentSentences) == 0 {
		return false
	}
	
	lastSentence := currentSentences[len(currentSentences)-1]
	
	// Simple heuristics for semantic boundaries
	// In a real implementation, you'd use embedding similarity
	
	// Check for topic transition indicators
	topicTransitionIndicators := []string{
		"however", "nevertheless", "on the other hand", "meanwhile",
		"furthermore", "moreover", "in addition", "subsequently",
		"consequently", "therefore", "thus", "hence", "in contrast",
		"alternatively", "similarly", "likewise", "conversely",
	}
	
	nextLower := strings.ToLower(nextSentence)
	for _, indicator := range topicTransitionIndicators {
		if strings.HasPrefix(nextLower, indicator) {
			return true
		}
	}
	
	// Check for paragraph-like structure (questions followed by explanations)
	if strings.HasSuffix(lastSentence, "?") && !strings.HasSuffix(nextSentence, "?") {
		return true
	}
	
	// Check for list-like structure
	listIndicators := []string{"first", "second", "third", "finally", "lastly", "next"}
	for _, indicator := range listIndicators {
		if strings.HasPrefix(nextLower, indicator) {
			return true
		}
	}
	
	// Check for significant vocabulary shift (simplified)
	return sc.hasVocabularyShift(lastSentence, nextSentence)
}

// hasVocabularyShift detects significant vocabulary changes between sentences
func (sc *SemanticChunker) hasVocabularyShift(sentence1, sentence2 string) bool {
	// Extract content words (simplified)
	words1 := sc.extractContentWords(sentence1)
	words2 := sc.extractContentWords(sentence2)
	
	if len(words1) == 0 || len(words2) == 0 {
		return false
	}
	
	// Calculate simple overlap ratio
	common := 0
	word1Map := make(map[string]bool)
	for _, word := range words1 {
		word1Map[word] = true
	}
	
	for _, word := range words2 {
		if word1Map[word] {
			common++
		}
	}
	
	overlapRatio := float64(common) / float64(len(words1)+len(words2)-common)
	
	// Consider it a vocabulary shift if overlap is very low
	return overlapRatio < 0.2
}

// extractContentWords extracts meaningful words from a sentence
func (sc *SemanticChunker) extractContentWords(sentence string) []string {
	// Simple stop word list
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "is": true,
		"are": true, "was": true, "were": true, "be": true, "been": true,
		"have": true, "has": true, "had": true, "will": true, "would": true,
		"could": true, "should": true, "may": true, "might": true, "can": true,
		"this": true, "that": true, "these": true, "those": true, "it": true,
		"he": true, "she": true, "they": true, "we": true, "you": true, "i": true,
	}
	
	words := strings.Fields(strings.ToLower(sentence))
	contentWords := []string{}
	
	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,!?;:\"'()[]{}...")
		if len(word) > 2 && !stopWords[word] {
			contentWords = append(contentWords, word)
		}
	}
	
	return contentWords
}

// calculateSemanticOverlap determines semantic overlap for next chunk
func (sc *SemanticChunker) calculateSemanticOverlap(currentSentences []string, nextSentence string) []string {
	if sc.config.ChunkOverlap <= 0 || len(currentSentences) == 0 {
		return []string{}
	}
	
	// For semantic chunking, we want to maintain semantic coherence in overlap
	// So we include sentences that are semantically related to the next sentence
	
	overlapSentences := []string{}
	overlapTokens := 0
	
	// Start from the end and work backwards
	for i := len(currentSentences) - 1; i >= 0; i-- {
		sentence := currentSentences[i]
		sentenceTokens := sc.EstimateTokens(sentence)
		
		if overlapTokens+sentenceTokens <= sc.config.ChunkOverlap {
			// Check if this sentence is semantically related to the next sentence
			if sc.areSemanticallySimilar(sentence, nextSentence) {
				overlapSentences = append([]string{sentence}, overlapSentences...)
				overlapTokens += sentenceTokens
			}
		} else {
			break
		}
	}
	
	return overlapSentences
}

// areSemanticallySimilar checks if two sentences are semantically similar
func (sc *SemanticChunker) areSemanticallySimilar(sentence1, sentence2 string) bool {
	words1 := sc.extractContentWords(sentence1)
	words2 := sc.extractContentWords(sentence2)
	
	if len(words1) == 0 || len(words2) == 0 {
		return false
	}
	
	// Calculate simple overlap ratio
	common := 0
	word1Map := make(map[string]bool)
	for _, word := range words1 {
		word1Map[word] = true
	}
	
	for _, word := range words2 {
		if word1Map[word] {
			common++
		}
	}
	
	overlapRatio := float64(common) / float64(len(words1)+len(words2)-common)
	
	// Consider similar if there's reasonable overlap
	return overlapRatio > 0.3
}

// createSemanticChunk creates a chunk from sentences with semantic metadata
func (sc *SemanticChunker) createSemanticChunk(sentences []string, originalText string, startPos int, metadata map[string]interface{}) *Chunk {
	chunkText := strings.Join(sentences, " ")
	
	chunk := &Chunk{
		Text:       chunkText,
		TokenCount: sc.EstimateTokens(chunkText),
		Sentences:  make([]string, len(sentences)),
		StartIndex: startPos,
		EndIndex:   startPos + len(chunkText),
		Metadata:   make(map[string]interface{}),
		CreatedAt:  time.Now(),
	}
	
	copy(chunk.Sentences, sentences)
	
	// Copy metadata
	if metadata != nil {
		for k, v := range metadata {
			chunk.Metadata[k] = v
		}
	}
	
	// Add semantic-specific metadata
	chunk.Metadata["semantic_coherence"] = sc.calculateSemanticCoherence(sentences)
	chunk.Metadata["content_words"] = sc.extractContentWordsFromChunk(sentences)
	
	// Ensure end index doesn't exceed original text length
	if chunk.EndIndex > len(originalText) {
		chunk.EndIndex = len(originalText)
	}
	
	return chunk
}

// calculateSemanticCoherence calculates a simple coherence score for the chunk
func (sc *SemanticChunker) calculateSemanticCoherence(sentences []string) float64 {
	if len(sentences) <= 1 {
		return 1.0
	}
	
	totalSimilarity := 0.0
	comparisons := 0
	
	for i := 0; i < len(sentences)-1; i++ {
		for j := i + 1; j < len(sentences); j++ {
			if sc.areSemanticallySimilar(sentences[i], sentences[j]) {
				totalSimilarity += 1.0
			}
			comparisons++
		}
	}
	
	if comparisons == 0 {
		return 1.0
	}
	
	return totalSimilarity / float64(comparisons)
}

// extractContentWordsFromChunk extracts all content words from chunk sentences
func (sc *SemanticChunker) extractContentWordsFromChunk(sentences []string) []string {
	allWords := []string{}
	wordSet := make(map[string]bool)
	
	for _, sentence := range sentences {
		words := sc.extractContentWords(sentence)
		for _, word := range words {
			if !wordSet[word] {
				allWords = append(allWords, word)
				wordSet[word] = true
			}
		}
	}
	
	return allWords
}

// estimateTokensForSentences estimates total tokens for a slice of sentences
func (sc *SemanticChunker) estimateTokensForSentences(sentences []string) int {
	total := 0
	for _, sentence := range sentences {
		total += sc.EstimateTokens(sentence)
	}
	return total
}

// findTextPosition finds the position of a sentence in the original text
func (sc *SemanticChunker) findTextPosition(text, sentence string, startFrom int) int {
	sentence = strings.TrimSpace(sentence)
	if sentence == "" {
		return startFrom
	}
	
	pos := strings.Index(text[startFrom:], sentence)
	if pos == -1 {
		return startFrom
	}
	return startFrom + pos
}

// EstimateTokens estimates the number of tokens in text
func (sc *SemanticChunker) EstimateTokens(text string) int {
	return sc.tokenEstimator(text)
}

// GetConfig returns the current chunker configuration
func (sc *SemanticChunker) GetConfig() *ChunkerConfig {
	config := *sc.config
	return &config
}

// SetConfig updates the chunker configuration
func (sc *SemanticChunker) SetConfig(config *ChunkerConfig) error {
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
func (sc *SemanticChunker) GetChunkSize() int {
	return sc.config.ChunkSize
}

// GetChunkOverlap returns the configured chunk overlap
func (sc *SemanticChunker) GetChunkOverlap() int {
	return sc.config.ChunkOverlap
}

// GetSupportedLanguages returns supported languages for this chunker
func (sc *SemanticChunker) GetSupportedLanguages() []string {
	return []string{"en"} // Currently optimized for English, expandable
}