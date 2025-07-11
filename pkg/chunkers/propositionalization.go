package chunkers

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// PropositionalizationChunker implements atomic proposition extraction from text
// This chunker creates self-contained, minimal semantic units suitable for complex reasoning
type PropositionalizationChunker struct {
	config *ChunkerConfig
	
	// LLM provider for proposition extraction
	llmProvider interfaces.LLM
	
	// Embedding provider for proposition validation
	embeddingProvider EmbeddingProvider
	
	// Similarity calculator for deduplication
	similarityCalculator SimilarityCalculator
	
	// Tokenizer for accurate token counting
	tokenizer TokenizerProvider
	
	// Proposition extraction configuration
	propositionConfig *PropositionConfig
	
	// Fallback token estimation
	tokenEstimator func(string) int
}

// PropositionConfig contains configuration for proposition extraction
type PropositionConfig struct {
	// EnableLLMExtraction uses LLM for proposition extraction
	EnableLLMExtraction bool `json:"enable_llm_extraction"`
	
	// ExtractionPrompt is the prompt template for extracting propositions
	ExtractionPrompt string `json:"extraction_prompt"`
	
	// MaxPropositionsPerChunk limits propositions extracted per input chunk
	MaxPropositionsPerChunk int `json:"max_propositions_per_chunk"`
	
	// MinPropositionLength is the minimum length for a valid proposition
	MinPropositionLength int `json:"min_proposition_length"`
	
	// MaxPropositionLength is the maximum length for a proposition
	MaxPropositionLength int `json:"max_proposition_length"`
	
	// EnableValidation validates propositions for completeness
	EnableValidation bool `json:"enable_validation"`
	
	// ValidationPrompt is used to validate proposition quality
	ValidationPrompt string `json:"validation_prompt"`
	
	// SimilarityThreshold for deduplication
	SimilarityThreshold float64 `json:"similarity_threshold"`
	
	// EnableDeduplication removes similar propositions
	EnableDeduplication bool `json:"enable_deduplication"`
	
	// PreserveCorePropositions keeps essential propositions even if similar
	PreserveCorePropositions bool `json:"preserve_core_propositions"`
	
	// ContextWindow size for maintaining coherence
	ContextWindow int `json:"context_window"`
	
	// MaxRetries for LLM calls
	MaxRetries int `json:"max_retries"`
	
	// RetryDelay between LLM retries
	RetryDelay time.Duration `json:"retry_delay"`
}

// DefaultPropositionConfig returns default proposition configuration
func DefaultPropositionConfig() *PropositionConfig {
	return &PropositionConfig{
		EnableLLMExtraction: true,
		ExtractionPrompt: `Extract atomic propositions from the following text. Each proposition should be:
1. A single, complete factual statement
2. Self-contained and understandable without additional context
3. Atomic (cannot be broken down further)
4. Factually accurate and verifiable

Text:
{{TEXT}}

Extract propositions in this format:
[PROPOSITION] Statement here
[PROPOSITION] Another statement here

Propositions:`,
		MaxPropositionsPerChunk:  10,
		MinPropositionLength:     20,
		MaxPropositionLength:     200,
		EnableValidation:         true,
		ValidationPrompt: `Evaluate if the following proposition is complete and self-contained:
"{{PROPOSITION}}"

Is this proposition:
1. Complete (contains subject, predicate, and necessary context)? YES/NO
2. Self-contained (understandable without external context)? YES/NO
3. Atomic (cannot be broken down further while maintaining meaning)? YES/NO

If all answers are YES, respond "VALID". Otherwise, respond "INVALID" and explain why.

Response:`,
		SimilarityThreshold:       0.85,
		EnableDeduplication:       true,
		PreserveCorePropositions:  true,
		ContextWindow:            3,
		MaxRetries:               3,
		RetryDelay:               2 * time.Second,
	}
}

// Proposition represents an atomic proposition
type Proposition struct {
	// Text content of the proposition
	Text string `json:"text"`
	
	// TokenCount in the proposition
	TokenCount int `json:"token_count"`
	
	// Confidence score for the proposition quality
	Confidence float64 `json:"confidence"`
	
	// IsValid indicates if the proposition passed validation
	IsValid bool `json:"is_valid"`
	
	// CoreConcepts extracted from the proposition
	CoreConcepts []string `json:"core_concepts"`
	
	// Entities mentioned in the proposition
	Entities []string `json:"entities"`
	
	// Relations expressed in the proposition
	Relations []string `json:"relations"`
	
	// SourceChunk reference to original chunk
	SourceChunk *Chunk `json:"source_chunk,omitempty"`
	
	// Position in the original text
	SourcePosition int `json:"source_position"`
	
	// Embedding vector for the proposition
	Embedding []float64 `json:"embedding,omitempty"`
	
	// Metadata for additional information
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	
	// CreatedAt timestamp
	CreatedAt time.Time `json:"created_at"`
}

// NewPropositionalizationChunker creates a new propositionalization chunker
func NewPropositionalizationChunker(config *ChunkerConfig, llmProvider interfaces.LLM) (*PropositionalizationChunker, error) {
	if config == nil {
		config = DefaultChunkerConfig()
	}
	
	propositionConfig := DefaultPropositionConfig()
	
	// Initialize embedding provider if available
	embeddingFactory := NewEmbeddingFactory(NewMemoryEmbeddingCache(1000))
	embeddingConfig := DefaultEmbeddingConfig()
	embeddingProvider, err := embeddingFactory.CreateProvider(embeddingConfig)
	if err != nil {
		// Log warning but continue without embeddings
		fmt.Printf("Warning: failed to create embedding provider for propositionalization: %v\n", err)
		embeddingProvider = nil
	}
	
	// Initialize similarity calculator
	similarityCalculator := NewCosineSimilarityCalculator()
	
	// Initialize tokenizer
	tokenizerFactory := NewTokenizerFactory()
	tokenizerConfig := DefaultTokenizerConfig()
	tokenizer, err := tokenizerFactory.CreateTokenizer(tokenizerConfig)
	if err != nil {
		// Log warning but continue with fallback
		fmt.Printf("Warning: failed to create tokenizer for propositionalization: %v\n", err)
		tokenizer = nil
	}
	
	chunker := &PropositionalizationChunker{
		config:               config,
		llmProvider:          llmProvider,
		embeddingProvider:    embeddingProvider,
		similarityCalculator: similarityCalculator,
		tokenizer:            tokenizer,
		propositionConfig:    propositionConfig,
		tokenEstimator:       defaultTokenEstimator,
	}
	
	return chunker, nil
}

// Chunk splits text into atomic propositions and returns them as chunks
func (pc *PropositionalizationChunker) Chunk(ctx context.Context, text string) ([]*Chunk, error) {
	return pc.ChunkWithMetadata(ctx, text, nil)
}

// ChunkWithMetadata splits text into atomic propositions with metadata
func (pc *PropositionalizationChunker) ChunkWithMetadata(ctx context.Context, text string, metadata map[string]interface{}) ([]*Chunk, error) {
	if text == "" {
		return []*Chunk{}, nil
	}
	
	startTime := time.Now()
	
	// Extract propositions from text
	propositions, err := pc.extractPropositions(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to extract propositions: %w", err)
	}
	
	// Validate propositions if enabled
	if pc.propositionConfig.EnableValidation {
		propositions = pc.validatePropositions(ctx, propositions)
	}
	
	// Deduplicate propositions if enabled
	if pc.propositionConfig.EnableDeduplication {
		propositions, err = pc.deduplicatePropositions(ctx, propositions)
		if err != nil {
			// Log error but continue with non-deduplicated propositions
			fmt.Printf("Warning: failed to deduplicate propositions: %v\n", err)
		}
	}
	
	// Convert propositions to chunks
	chunks := make([]*Chunk, len(propositions))
	for i, prop := range propositions {
		chunk := pc.propositionToChunk(prop, metadata)
		chunks[i] = chunk
	}
	
	// Add chunking statistics
	processingTime := time.Since(startTime)
	stats := CalculateStats(chunks, len(text), processingTime)
	
	for _, chunk := range chunks {
		chunk.Metadata["chunking_stats"] = stats
		chunk.Metadata["chunker_type"] = "propositionalization"
		chunk.Metadata["proposition_config"] = pc.propositionConfig
		chunk.Metadata["total_propositions"] = len(propositions)
	}
	
	return chunks, nil
}

// extractPropositions extracts atomic propositions from text
func (pc *PropositionalizationChunker) extractPropositions(ctx context.Context, text string) ([]*Proposition, error) {
	if pc.propositionConfig.EnableLLMExtraction && pc.llmProvider != nil {
		return pc.extractPropositionsWithLLM(ctx, text)
	}
	
	// Fallback to rule-based extraction
	return pc.extractPropositionsWithRules(text)
}

// extractPropositionsWithLLM uses LLM to extract propositions
func (pc *PropositionalizationChunker) extractPropositionsWithLLM(ctx context.Context, text string) ([]*Proposition, error) {
	// Prepare prompt
	prompt := strings.ReplaceAll(pc.propositionConfig.ExtractionPrompt, "{{TEXT}}", text)
	
	messages := types.MessageList{
		{
			Role:    types.MessageRoleUser,
			Content: prompt,
		},
	}
	
	// Retry logic for LLM calls
	var response string
	var err error
	
	for i := 0; i < pc.propositionConfig.MaxRetries; i++ {
		response, err = pc.llmProvider.Generate(ctx, messages)
		if err == nil {
			break
		}
		
		if i < pc.propositionConfig.MaxRetries-1 {
			time.Sleep(pc.propositionConfig.RetryDelay)
		}
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to extract propositions after %d retries: %w", pc.propositionConfig.MaxRetries, err)
	}
	
	// Parse propositions from response
	return pc.parsePropositionsFromResponse(response, text)
}

// parsePropositionsFromResponse parses propositions from LLM response
func (pc *PropositionalizationChunker) parsePropositionsFromResponse(response, originalText string) ([]*Proposition, error) {
	// Parse propositions using regex pattern
	propositionPattern := regexp.MustCompile(`\[PROPOSITION\]\s*(.+?)(?:\n|$)`)
	matches := propositionPattern.FindAllStringSubmatch(response, -1)
	
	propositions := make([]*Proposition, 0, len(matches))
	
	for i, match := range matches {
		if len(match) < 2 {
			continue
		}
		
		propText := strings.TrimSpace(match[1])
		
		// Validate proposition length
		if len(propText) < pc.propositionConfig.MinPropositionLength ||
		   len(propText) > pc.propositionConfig.MaxPropositionLength {
			continue
		}
		
		// Create proposition
		prop := &Proposition{
			Text:           propText,
			TokenCount:     pc.EstimateTokens(propText),
			Confidence:     0.8, // Default confidence for LLM-extracted propositions
			IsValid:        true, // Will be validated later if enabled
			SourcePosition: i,
			Metadata:       make(map[string]interface{}),
			CreatedAt:      time.Now(),
		}
		
		// Extract core concepts, entities, and relations
		pc.enrichProposition(prop)
		
		propositions = append(propositions, prop)
		
		// Limit number of propositions per chunk
		if len(propositions) >= pc.propositionConfig.MaxPropositionsPerChunk {
			break
		}
	}
	
	return propositions, nil
}

// extractPropositionsWithRules uses rule-based extraction as fallback
func (pc *PropositionalizationChunker) extractPropositionsWithRules(text string) ([]*Proposition, error) {
	// Split text into sentences
	sentences := pc.splitIntoSentences(text)
	
	propositions := make([]*Proposition, 0, len(sentences))
	
	for i, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		
		// Skip empty or too short sentences
		if len(sentence) < pc.propositionConfig.MinPropositionLength {
			continue
		}
		
		// Skip sentences that are likely not complete propositions
		if !pc.isLikelyProposition(sentence) {
			continue
		}
		
		// Create proposition from sentence
		prop := &Proposition{
			Text:           sentence,
			TokenCount:     pc.EstimateTokens(sentence),
			Confidence:     0.6, // Lower confidence for rule-based extraction
			IsValid:        true,
			SourcePosition: i,
			Metadata:       make(map[string]interface{}),
			CreatedAt:      time.Now(),
		}
		
		// Extract core concepts, entities, and relations
		pc.enrichProposition(prop)
		
		propositions = append(propositions, prop)
		
		// Limit number of propositions
		if len(propositions) >= pc.propositionConfig.MaxPropositionsPerChunk {
			break
		}
	}
	
	return propositions, nil
}

// splitIntoSentences splits text into sentences
func (pc *PropositionalizationChunker) splitIntoSentences(text string) []string {
	// Simple sentence splitting - can be enhanced with more sophisticated NLP
	sentencePattern := regexp.MustCompile(`[.!?]+\s+`)
	sentences := sentencePattern.Split(text, -1)
	
	// Clean up sentences
	cleanSentences := make([]string, 0, len(sentences))
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 0 {
			cleanSentences = append(cleanSentences, sentence)
		}
	}
	
	return cleanSentences
}

// isLikelyProposition checks if a sentence is likely to be a valid proposition
func (pc *PropositionalizationChunker) isLikelyProposition(sentence string) bool {
	// Check for basic proposition structure
	hasSubject := pc.hasSubject(sentence)
	hasPredicate := pc.hasPredicate(sentence)
	isComplete := pc.isCompleteSentence(sentence)
	
	return hasSubject && hasPredicate && isComplete
}

// hasSubject checks if sentence has a recognizable subject
func (pc *PropositionalizationChunker) hasSubject(sentence string) bool {
	// Simple heuristic: check for common subject patterns
	subjectPatterns := []string{
		`\b(the|a|an|this|that|these|those)\s+\w+`,
		`\b[A-Z]\w+`,  // Proper nouns
		`\b(he|she|it|they|we|you|I)\b`,
	}
	
	for _, pattern := range subjectPatterns {
		matched, _ := regexp.MatchString(pattern, sentence)
		if matched {
			return true
		}
	}
	
	return false
}

// hasPredicate checks if sentence has a recognizable predicate
func (pc *PropositionalizationChunker) hasPredicate(sentence string) bool {
	// Simple heuristic: check for common verb patterns
	verbPatterns := []string{
		`\b(is|are|was|were|be|been|being)\b`,
		`\b(has|have|had)\b`,
		`\b(does|do|did)\b`,
		`\b\w+ed\b`,  // Past tense verbs
		`\b\w+ing\b`, // Present participle
		`\b\w+s\b`,   // Third person singular
	}
	
	for _, pattern := range verbPatterns {
		matched, _ := regexp.MatchString(pattern, sentence)
		if matched {
			return true
		}
	}
	
	return false
}

// isCompleteSentence checks if sentence appears complete
func (pc *PropositionalizationChunker) isCompleteSentence(sentence string) bool {
	// Check for proper capitalization and punctuation
	hasCapital := len(sentence) > 0 && sentence[0] >= 'A' && sentence[0] <= 'Z'
	hasPunctuation := strings.HasSuffix(sentence, ".") || 
	                  strings.HasSuffix(sentence, "!") || 
	                  strings.HasSuffix(sentence, "?")
	
	return hasCapital && hasPunctuation
}

// enrichProposition extracts additional information from proposition
func (pc *PropositionalizationChunker) enrichProposition(prop *Proposition) {
	// Extract core concepts (simplified keyword extraction)
	prop.CoreConcepts = pc.extractConcepts(prop.Text)
	
	// Extract entities (simplified named entity recognition)
	prop.Entities = pc.extractEntities(prop.Text)
	
	// Extract relations (simplified relation extraction)
	prop.Relations = pc.extractRelations(prop.Text)
	
	// Add structural metadata
	prop.Metadata["word_count"] = len(strings.Fields(prop.Text))
	prop.Metadata["character_count"] = len(prop.Text)
	prop.Metadata["has_numbers"] = regexp.MustCompile(`\d+`).MatchString(prop.Text)
	prop.Metadata["has_dates"] = pc.hasDatePattern(prop.Text)
	prop.Metadata["complexity_score"] = pc.calculateComplexityScore(prop.Text)
}

// extractConcepts extracts key concepts from proposition text
func (pc *PropositionalizationChunker) extractConcepts(text string) []string {
	// Simple keyword extraction - remove stop words and keep important terms
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "is": true,
		"are": true, "was": true, "were": true, "be": true, "been": true,
		"have": true, "has": true, "had": true, "will": true, "would": true,
		"this": true, "that": true, "these": true, "those": true,
	}
	
	words := strings.Fields(strings.ToLower(text))
	concepts := make([]string, 0)
	
	for _, word := range words {
		// Clean word
		word = regexp.MustCompile(`[^\w]`).ReplaceAllString(word, "")
		
		// Skip stop words and short words
		if len(word) > 3 && !stopWords[word] {
			concepts = append(concepts, word)
		}
	}
	
	return concepts
}

// extractEntities extracts named entities from proposition text
func (pc *PropositionalizationChunker) extractEntities(text string) []string {
	// Simple named entity extraction using capitalization patterns
	entityPattern := regexp.MustCompile(`\b[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*\b`)
	matches := entityPattern.FindAllString(text, -1)
	
	// Deduplicate entities
	entitySet := make(map[string]bool)
	entities := make([]string, 0)
	
	for _, entity := range matches {
		if !entitySet[entity] {
			entities = append(entities, entity)
			entitySet[entity] = true
		}
	}
	
	return entities
}

// extractRelations extracts relational terms from proposition text
func (pc *PropositionalizationChunker) extractRelations(text string) []string {
	// Common relational terms
	relationPatterns := []string{
		`\b(is|are|was|were)\s+\w+`,
		`\b(has|have|had)\s+\w+`,
		`\b(contains|includes|involves)\b`,
		`\b(causes|results in|leads to)\b`,
		`\b(located|situated|positioned)\b`,
		`\b(belongs to|part of|member of)\b`,
	}
	
	relations := make([]string, 0)
	
	for _, pattern := range relationPatterns {
		regex := regexp.MustCompile(pattern)
		matches := regex.FindAllString(text, -1)
		relations = append(relations, matches...)
	}
	
	return relations
}

// hasDatePattern checks if text contains date patterns
func (pc *PropositionalizationChunker) hasDatePattern(text string) bool {
	datePatterns := []string{
		`\b\d{1,2}/\d{1,2}/\d{4}\b`,
		`\b\d{4}-\d{2}-\d{2}\b`,
		`\b(January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{1,2},?\s+\d{4}\b`,
	}
	
	for _, pattern := range datePatterns {
		matched, _ := regexp.MatchString(pattern, text)
		if matched {
			return true
		}
	}
	
	return false
}

// calculateComplexityScore calculates a complexity score for the proposition
func (pc *PropositionalizationChunker) calculateComplexityScore(text string) float64 {
	// Simple complexity calculation based on various factors
	wordCount := len(strings.Fields(text))
	clauseCount := strings.Count(text, ",") + strings.Count(text, ";") + 1
	avgWordLength := float64(len(text)) / float64(wordCount)
	
	// Normalize scores
	wordComplexity := float64(wordCount) / 20.0       // Normalize to ~20 words
	clauseComplexity := float64(clauseCount) / 3.0    // Normalize to ~3 clauses
	lengthComplexity := avgWordLength / 6.0           // Normalize to ~6 chars per word
	
	return (wordComplexity + clauseComplexity + lengthComplexity) / 3.0
}

// validatePropositions validates propositions for completeness and quality
func (pc *PropositionalizationChunker) validatePropositions(ctx context.Context, propositions []*Proposition) []*Proposition {
	if pc.llmProvider == nil {
		// Simple rule-based validation as fallback
		return pc.validatePropositionsWithRules(propositions)
	}
	
	validPropositions := make([]*Proposition, 0, len(propositions))
	
	for _, prop := range propositions {
		isValid := pc.validatePropositionWithLLM(ctx, prop)
		prop.IsValid = isValid
		
		if isValid {
			validPropositions = append(validPropositions, prop)
		}
	}
	
	return validPropositions
}

// validatePropositionWithLLM validates a proposition using LLM
func (pc *PropositionalizationChunker) validatePropositionWithLLM(ctx context.Context, prop *Proposition) bool {
	prompt := strings.ReplaceAll(pc.propositionConfig.ValidationPrompt, "{{PROPOSITION}}", prop.Text)
	
	messages := types.MessageList{
		{
			Role:    types.MessageRoleUser,
			Content: prompt,
		},
	}
	
	response, err := pc.llmProvider.Generate(ctx, messages)
	if err != nil {
		// If validation fails, assume valid
		return true
	}
	
	// Check if response indicates validity
	return strings.Contains(strings.ToUpper(response), "VALID")
}

// validatePropositionsWithRules validates propositions using simple rules
func (pc *PropositionalizationChunker) validatePropositionsWithRules(propositions []*Proposition) []*Proposition {
	validPropositions := make([]*Proposition, 0, len(propositions))
	
	for _, prop := range propositions {
		// Simple validation rules
		isValid := len(prop.Text) >= pc.propositionConfig.MinPropositionLength &&
		          len(prop.Text) <= pc.propositionConfig.MaxPropositionLength &&
		          pc.isLikelyProposition(prop.Text)
		
		prop.IsValid = isValid
		
		if isValid {
			validPropositions = append(validPropositions, prop)
		}
	}
	
	return validPropositions
}

// deduplicatePropositions removes similar propositions
func (pc *PropositionalizationChunker) deduplicatePropositions(ctx context.Context, propositions []*Proposition) ([]*Proposition, error) {
	if len(propositions) <= 1 {
		return propositions, nil
	}
	
	// Generate embeddings for propositions if available
	if pc.embeddingProvider != nil {
		for _, prop := range propositions {
			if prop.Embedding == nil {
				embedding, err := pc.embeddingProvider.GetEmbedding(ctx, prop.Text)
				if err == nil {
					prop.Embedding = embedding
				}
			}
		}
	}
	
	// Find duplicates
	duplicateIndices := make(map[int]bool)
	
	for i := 0; i < len(propositions); i++ {
		if duplicateIndices[i] {
			continue
		}
		
		for j := i + 1; j < len(propositions); j++ {
			if duplicateIndices[j] {
				continue
			}
			
			similar, err := pc.arePropositionsSimilar(propositions[i], propositions[j])
			if err != nil {
				continue
			}
			
			if similar {
				// Keep the proposition with higher confidence
				if propositions[i].Confidence >= propositions[j].Confidence {
					duplicateIndices[j] = true
				} else {
					duplicateIndices[i] = true
					break
				}
			}
		}
	}
	
	// Create deduplicated list
	deduplicatedPropositions := make([]*Proposition, 0, len(propositions))
	for i, prop := range propositions {
		if !duplicateIndices[i] {
			deduplicatedPropositions = append(deduplicatedPropositions, prop)
		}
	}
	
	return deduplicatedPropositions, nil
}

// arePropositionsSimilar checks if two propositions are similar
func (pc *PropositionalizationChunker) arePropositionsSimilar(prop1, prop2 *Proposition) (bool, error) {
	// Use embedding similarity if available
	if prop1.Embedding != nil && prop2.Embedding != nil && pc.similarityCalculator != nil {
		similarity, err := pc.similarityCalculator.CosineSimilarity(prop1.Embedding, prop2.Embedding)
		if err != nil {
			return false, err
		}
		return similarity >= pc.propositionConfig.SimilarityThreshold, nil
	}
	
	// Fallback to text similarity
	textSimilarity := pc.calculateTextSimilarity(prop1.Text, prop2.Text)
	return textSimilarity >= pc.propositionConfig.SimilarityThreshold, nil
}

// calculateTextSimilarity calculates similarity between two text strings
func (pc *PropositionalizationChunker) calculateTextSimilarity(text1, text2 string) float64 {
	// Simple Jaccard similarity based on words
	words1 := strings.Fields(strings.ToLower(text1))
	words2 := strings.Fields(strings.ToLower(text2))
	
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)
	
	for _, word := range words1 {
		set1[word] = true
	}
	
	for _, word := range words2 {
		set2[word] = true
	}
	
	// Calculate intersection
	intersection := 0
	for word := range set1 {
		if set2[word] {
			intersection++
		}
	}
	
	// Calculate union
	union := len(set1) + len(set2) - intersection
	
	if union == 0 {
		return 0
	}
	
	return float64(intersection) / float64(union)
}

// propositionToChunk converts a proposition to a chunk
func (pc *PropositionalizationChunker) propositionToChunk(prop *Proposition, metadata map[string]interface{}) *Chunk {
	chunk := &Chunk{
		Text:       prop.Text,
		TokenCount: prop.TokenCount,
		Sentences:  []string{prop.Text}, // Proposition is atomic, so it's a single "sentence"
		StartIndex: prop.SourcePosition,
		EndIndex:   prop.SourcePosition + len(prop.Text),
		Metadata:   make(map[string]interface{}),
		CreatedAt:  time.Now(),
	}
	
	// Copy original metadata
	if metadata != nil {
		for k, v := range metadata {
			chunk.Metadata[k] = v
		}
	}
	
	// Add proposition-specific metadata
	chunk.Metadata["proposition"] = prop
	chunk.Metadata["is_atomic_proposition"] = true
	chunk.Metadata["confidence"] = prop.Confidence
	chunk.Metadata["is_valid"] = prop.IsValid
	chunk.Metadata["core_concepts"] = prop.CoreConcepts
	chunk.Metadata["entities"] = prop.Entities
	chunk.Metadata["relations"] = prop.Relations
	chunk.Metadata["complexity_score"] = prop.Metadata["complexity_score"]
	
	if prop.Embedding != nil {
		chunk.Metadata["embedding"] = prop.Embedding
	}
	
	return chunk
}

// EstimateTokens estimates the number of tokens in text
func (pc *PropositionalizationChunker) EstimateTokens(text string) int {
	if pc.tokenizer != nil {
		count, err := pc.tokenizer.CountTokens(text)
		if err == nil {
			return count
		}
	}
	
	return pc.tokenEstimator(text)
}

// GetConfig returns the current chunker configuration
func (pc *PropositionalizationChunker) GetConfig() *ChunkerConfig {
	config := *pc.config
	return &config
}

// SetConfig updates the chunker configuration
func (pc *PropositionalizationChunker) SetConfig(config *ChunkerConfig) error {
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
func (pc *PropositionalizationChunker) GetChunkSize() int {
	return pc.config.ChunkSize
}

// GetChunkOverlap returns the configured chunk overlap
func (pc *PropositionalizationChunker) GetChunkOverlap() int {
	return pc.config.ChunkOverlap
}

// GetSupportedLanguages returns supported languages
func (pc *PropositionalizationChunker) GetSupportedLanguages() []string {
	return []string{"en"} // Currently optimized for English
}

// SetPropositionConfig updates the proposition configuration
func (pc *PropositionalizationChunker) SetPropositionConfig(config *PropositionConfig) {
	pc.propositionConfig = config
}

// GetPropositionConfig returns the current proposition configuration
func (pc *PropositionalizationChunker) GetPropositionConfig() *PropositionConfig {
	return pc.propositionConfig
}