package chunkers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// ContextualChunker implements Anthropic-style contextual chunking
// This chunker generates contextual descriptions for chunks using LLM providers
type ContextualChunker struct {
	config *ChunkerConfig
	
	// Base chunker for initial splitting
	baseChunker Chunker
	
	// LLM provider for context generation
	llmProvider interfaces.LLM
	
	// Embedding provider for semantic enhancement
	embeddingProvider EmbeddingProvider
	
	// Tokenizer for accurate token counting
	tokenizer TokenizerProvider
	
	// Context generation settings
	contextConfig *ContextualConfig
	
	// Fallback token estimation
	tokenEstimator func(string) int
}

// ContextualConfig contains configuration for contextual chunking
type ContextualConfig struct {
	// EnableContextGeneration enables LLM-based context generation
	EnableContextGeneration bool `json:"enable_context_generation"`
	
	// ContextPrompt is the prompt template for generating context
	ContextPrompt string `json:"context_prompt"`
	
	// MaxContextLength is the maximum length for generated context
	MaxContextLength int `json:"max_context_length"`
	
	// IncludeDocumentContext includes document-level context in chunks
	IncludeDocumentContext bool `json:"include_document_context"`
	
	// DocumentSummaryLength is the length of document summary to include
	DocumentSummaryLength int `json:"document_summary_length"`
	
	// SemanticEnhancement enables semantic metadata enrichment
	SemanticEnhancement bool `json:"semantic_enhancement"`
	
	// ContextRefreshThreshold determines when to regenerate context
	ContextRefreshThreshold float64 `json:"context_refresh_threshold"`
	
	// MaxRetries for LLM calls
	MaxRetries int `json:"max_retries"`
	
	// RetryDelay between LLM retries
	RetryDelay time.Duration `json:"retry_delay"`
}

// DefaultContextualConfig returns default contextual configuration
func DefaultContextualConfig() *ContextualConfig {
	return &ContextualConfig{
		EnableContextGeneration: true,
		ContextPrompt: `Analyze the following text chunk and provide a brief contextual description that captures:
1. The main topic or theme
2. Key concepts or entities mentioned
3. The purpose or intent of this content
4. How it might relate to broader document themes

Text chunk:
{{CHUNK_TEXT}}

Document context:
{{DOCUMENT_CONTEXT}}

Provide a concise contextual description (max 150 words):`,
		MaxContextLength:         150,
		IncludeDocumentContext:   true,
		DocumentSummaryLength:    300,
		SemanticEnhancement:      true,
		ContextRefreshThreshold:  0.7,
		MaxRetries:              3,
		RetryDelay:              2 * time.Second,
	}
}

// NewContextualChunker creates a new contextual chunker
func NewContextualChunker(config *ChunkerConfig, llmProvider interfaces.LLM) (*ContextualChunker, error) {
	if config == nil {
		config = DefaultChunkerConfig()
	}
	
	contextConfig := DefaultContextualConfig()
	
	// Create base chunker (semantic chunker by default)
	baseChunker, err := NewSemanticChunker(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create base chunker: %w", err)
	}
	
	// Initialize embedding provider if available
	embeddingFactory := NewEmbeddingFactory(NewMemoryEmbeddingCache(1000))
	embeddingConfig := DefaultEmbeddingConfig()
	embeddingProvider, err := embeddingFactory.CreateProvider(embeddingConfig)
	if err != nil {
		// Log warning but continue without embeddings
		fmt.Printf("Warning: failed to create embedding provider for contextual chunker: %v\n", err)
		embeddingProvider = nil
	}
	
	// Initialize tokenizer
	tokenizerFactory := NewTokenizerFactory()
	tokenizerConfig := DefaultTokenizerConfig()
	tokenizer, err := tokenizerFactory.CreateTokenizer(tokenizerConfig)
	if err != nil {
		// Log warning but continue with fallback
		fmt.Printf("Warning: failed to create tokenizer for contextual chunker: %v\n", err)
		tokenizer = nil
	}
	
	chunker := &ContextualChunker{
		config:            config,
		baseChunker:       baseChunker,
		llmProvider:       llmProvider,
		embeddingProvider: embeddingProvider,
		tokenizer:         tokenizer,
		contextConfig:     contextConfig,
		tokenEstimator:    defaultTokenEstimator,
	}
	
	return chunker, nil
}

// Chunk splits text into contextually enhanced chunks
func (cc *ContextualChunker) Chunk(ctx context.Context, text string) ([]*Chunk, error) {
	return cc.ChunkWithMetadata(ctx, text, nil)
}

// ChunkWithMetadata splits text into chunks with contextual metadata
func (cc *ContextualChunker) ChunkWithMetadata(ctx context.Context, text string, metadata map[string]interface{}) ([]*Chunk, error) {
	if text == "" {
		return []*Chunk{}, nil
	}
	
	startTime := time.Now()
	
	// Generate document-level context if enabled
	var documentContext string
	if cc.contextConfig.IncludeDocumentContext {
		var err error
		documentContext, err = cc.generateDocumentContext(ctx, text)
		if err != nil {
			// Log error but continue without document context
			fmt.Printf("Warning: failed to generate document context: %v\n", err)
			documentContext = ""
		}
	}
	
	// Use base chunker to get initial chunks
	baseChunks, err := cc.baseChunker.ChunkWithMetadata(ctx, text, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create base chunks: %w", err)
	}
	
	// Enhance chunks with contextual information
	enhancedChunks := make([]*Chunk, len(baseChunks))
	for i, chunk := range baseChunks {
		enhancedChunk, err := cc.enhanceChunkWithContext(ctx, chunk, documentContext, text)
		if err != nil {
			// Log error but continue with unenhanced chunk
			fmt.Printf("Warning: failed to enhance chunk %d with context: %v\n", i, err)
			enhancedChunk = chunk
		}
		enhancedChunks[i] = enhancedChunk
	}
	
	// Add chunking statistics
	processingTime := time.Since(startTime)
	stats := CalculateStats(enhancedChunks, len(text), processingTime)
	
	for _, chunk := range enhancedChunks {
		chunk.Metadata["chunking_stats"] = stats
		chunk.Metadata["chunker_type"] = "contextual"
		chunk.Metadata["context_config"] = cc.contextConfig
		chunk.Metadata["has_document_context"] = documentContext != ""
	}
	
	return enhancedChunks, nil
}

// generateDocumentContext generates a contextual summary of the entire document
func (cc *ContextualChunker) generateDocumentContext(ctx context.Context, text string) (string, error) {
	if cc.llmProvider == nil {
		return "", fmt.Errorf("LLM provider not available")
	}
	
	// Truncate text if it's too long for the LLM
	maxInputLength := 4000 // Conservative estimate for most LLMs
	truncatedText := text
	if cc.tokenEstimator(text) > maxInputLength {
		truncatedText = cc.truncateText(text, maxInputLength)
	}
	
	prompt := fmt.Sprintf(`Analyze the following document and provide a brief summary that captures:
1. Main themes and topics
2. Key concepts and terminology used
3. Document structure and organization
4. Purpose and intended audience

Document:
%s

Provide a concise summary (max %d words):`, truncatedText, cc.contextConfig.DocumentSummaryLength/4) // Rough word estimate
	
	messages := types.MessageList{
		{
			Role:    types.MessageRoleUser,
			Content: prompt,
		},
	}
	
	response, err := cc.llmProvider.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate document context: %w", err)
	}
	
	// Truncate response if needed
	if cc.tokenEstimator(response) > cc.contextConfig.DocumentSummaryLength {
		response = cc.truncateText(response, cc.contextConfig.DocumentSummaryLength)
	}
	
	return response, nil
}

// enhanceChunkWithContext enhances a chunk with contextual information
func (cc *ContextualChunker) enhanceChunkWithContext(ctx context.Context, chunk *Chunk, documentContext, fullText string) (*Chunk, error) {
	// Create enhanced chunk
	enhancedChunk := &Chunk{
		Text:       chunk.Text,
		TokenCount: chunk.TokenCount,
		Sentences:  make([]string, len(chunk.Sentences)),
		StartIndex: chunk.StartIndex,
		EndIndex:   chunk.EndIndex,
		Metadata:   make(map[string]interface{}),
		CreatedAt:  time.Now(),
	}
	
	copy(enhancedChunk.Sentences, chunk.Sentences)
	
	// Copy original metadata
	for k, v := range chunk.Metadata {
		enhancedChunk.Metadata[k] = v
	}
	
	// Generate contextual description if enabled
	if cc.contextConfig.EnableContextGeneration && cc.llmProvider != nil {
		contextualDescription, err := cc.generateChunkContext(ctx, chunk.Text, documentContext)
		if err != nil {
			// Log error but continue without context
			fmt.Printf("Warning: failed to generate chunk context: %v\n", err)
		} else {
			enhancedChunk.Metadata["contextual_description"] = contextualDescription
		}
	}
	
	// Add semantic enhancements if enabled
	if cc.contextConfig.SemanticEnhancement && cc.embeddingProvider != nil {
		err := cc.addSemanticEnhancements(ctx, enhancedChunk)
		if err != nil {
			// Log error but continue without semantic enhancements
			fmt.Printf("Warning: failed to add semantic enhancements: %v\n", err)
		}
	}
	
	// Add positional context
	enhancedChunk.Metadata["position_context"] = cc.generatePositionalContext(chunk, fullText)
	
	// Add structural metadata
	enhancedChunk.Metadata["chunk_structure"] = cc.analyzeChunkStructure(chunk.Text)
	
	return enhancedChunk, nil
}

// generateChunkContext generates contextual description for a chunk
func (cc *ContextualChunker) generateChunkContext(ctx context.Context, chunkText, documentContext string) (string, error) {
	if cc.llmProvider == nil {
		return "", fmt.Errorf("LLM provider not available")
	}
	
	// Prepare prompt by replacing placeholders
	prompt := strings.ReplaceAll(cc.contextConfig.ContextPrompt, "{{CHUNK_TEXT}}", chunkText)
	prompt = strings.ReplaceAll(prompt, "{{DOCUMENT_CONTEXT}}", documentContext)
	
	messages := types.MessageList{
		{
			Role:    types.MessageRoleUser,
			Content: prompt,
		},
	}
	
	// Retry logic for LLM calls
	var response string
	var err error
	
	for i := 0; i < cc.contextConfig.MaxRetries; i++ {
		response, err = cc.llmProvider.Generate(ctx, messages)
		if err == nil {
			break
		}
		
		if i < cc.contextConfig.MaxRetries-1 {
			time.Sleep(cc.contextConfig.RetryDelay)
		}
	}
	
	if err != nil {
		return "", fmt.Errorf("failed to generate chunk context after %d retries: %w", cc.contextConfig.MaxRetries, err)
	}
	
	// Truncate response if needed
	if cc.tokenEstimator(response) > cc.contextConfig.MaxContextLength {
		response = cc.truncateText(response, cc.contextConfig.MaxContextLength)
	}
	
	return response, nil
}

// addSemanticEnhancements adds semantic metadata to chunks
func (cc *ContextualChunker) addSemanticEnhancements(ctx context.Context, chunk *Chunk) error {
	if cc.embeddingProvider == nil {
		return fmt.Errorf("embedding provider not available")
	}
	
	// Generate embedding for the chunk
	embedding, err := cc.embeddingProvider.GetEmbedding(ctx, chunk.Text)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}
	
	// Store embedding in metadata
	chunk.Metadata["embedding"] = embedding
	chunk.Metadata["embedding_model"] = cc.embeddingProvider.GetModelInfo().Name
	chunk.Metadata["embedding_dimensions"] = len(embedding)
	
	// Generate embeddings for individual sentences for finer-grained analysis
	if len(chunk.Sentences) > 1 {
		sentenceEmbeddings := make([][]float64, len(chunk.Sentences))
		for i, sentence := range chunk.Sentences {
			sentEmb, err := cc.embeddingProvider.GetEmbedding(ctx, sentence)
			if err != nil {
				// Skip this sentence if embedding fails
				continue
			}
			sentenceEmbeddings[i] = sentEmb
		}
		chunk.Metadata["sentence_embeddings"] = sentenceEmbeddings
	}
	
	return nil
}

// generatePositionalContext generates context about chunk position in document
func (cc *ContextualChunker) generatePositionalContext(chunk *Chunk, fullText string) map[string]interface{} {
	totalLength := len(fullText)
	
	return map[string]interface{}{
		"relative_position":    float64(chunk.StartIndex) / float64(totalLength),
		"document_section":     cc.identifyDocumentSection(chunk.StartIndex, totalLength),
		"preceding_context":    cc.extractPrecedingContext(chunk.StartIndex, fullText),
		"following_context":    cc.extractFollowingContext(chunk.EndIndex, fullText),
		"paragraph_number":     cc.estimateParagraphNumber(chunk.StartIndex, fullText),
	}
}

// identifyDocumentSection identifies which section of the document the chunk belongs to
func (cc *ContextualChunker) identifyDocumentSection(startIndex, totalLength int) string {
	position := float64(startIndex) / float64(totalLength)
	
	switch {
	case position < 0.1:
		return "introduction"
	case position < 0.3:
		return "early_content"
	case position < 0.7:
		return "main_content"
	case position < 0.9:
		return "late_content"
	default:
		return "conclusion"
	}
}

// extractPrecedingContext extracts a small amount of preceding text for context
func (cc *ContextualChunker) extractPrecedingContext(startIndex int, fullText string) string {
	contextLength := 100 // characters
	start := startIndex - contextLength
	if start < 0 {
		start = 0
	}
	
	if start >= startIndex {
		return ""
	}
	
	return fullText[start:startIndex]
}

// extractFollowingContext extracts a small amount of following text for context
func (cc *ContextualChunker) extractFollowingContext(endIndex int, fullText string) string {
	contextLength := 100 // characters
	end := endIndex + contextLength
	if end > len(fullText) {
		end = len(fullText)
	}
	
	if endIndex >= end {
		return ""
	}
	
	return fullText[endIndex:end]
}

// estimateParagraphNumber estimates which paragraph the chunk starts in
func (cc *ContextualChunker) estimateParagraphNumber(startIndex int, fullText string) int {
	paragraphCount := strings.Count(fullText[:startIndex], "\n\n") + 1
	return paragraphCount
}

// analyzeChunkStructure analyzes the structural properties of a chunk
func (cc *ContextualChunker) analyzeChunkStructure(text string) map[string]interface{} {
	return map[string]interface{}{
		"sentence_count":    len(strings.Split(text, ".")),
		"paragraph_count":   len(strings.Split(text, "\n\n")),
		"word_count":        len(strings.Fields(text)),
		"character_count":   len(text),
		"has_questions":     strings.Contains(text, "?"),
		"has_lists":         strings.Contains(text, "•") || strings.Contains(text, "1.") || strings.Contains(text, "-"),
		"has_emphasis":      strings.Contains(text, "**") || strings.Contains(text, "*"),
		"avg_sentence_length": cc.calculateAverageSentenceLength(text),
	}
}

// calculateAverageSentenceLength calculates the average sentence length in the chunk
func (cc *ContextualChunker) calculateAverageSentenceLength(text string) float64 {
	sentences := strings.Split(text, ".")
	if len(sentences) == 0 {
		return 0
	}
	
	totalLength := 0
	validSentences := 0
	
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 0 {
			totalLength += len(sentence)
			validSentences++
		}
	}
	
	if validSentences == 0 {
		return 0
	}
	
	return float64(totalLength) / float64(validSentences)
}

// truncateText truncates text to fit within token limit
func (cc *ContextualChunker) truncateText(text string, maxTokens int) string {
	if cc.tokenizer != nil {
		// Use proper tokenizer if available
		tokens, err := cc.tokenizer.CountTokens(text)
		if err == nil && tokens <= maxTokens {
			return text
		}
	}
	
	// Fallback to character-based truncation
	maxChars := maxTokens * 4 // Rough estimate
	if len(text) <= maxChars {
		return text
	}
	
	return text[:maxChars]
}

// EstimateTokens estimates the number of tokens in text
func (cc *ContextualChunker) EstimateTokens(text string) int {
	if cc.tokenizer != nil {
		count, err := cc.tokenizer.CountTokens(text)
		if err == nil {
			return count
		}
	}
	
	return cc.tokenEstimator(text)
}

// GetConfig returns the current chunker configuration
func (cc *ContextualChunker) GetConfig() *ChunkerConfig {
	return cc.baseChunker.GetConfig()
}

// SetConfig updates the chunker configuration
func (cc *ContextualChunker) SetConfig(config *ChunkerConfig) error {
	return cc.baseChunker.SetConfig(config)
}

// GetChunkSize returns the configured chunk size
func (cc *ContextualChunker) GetChunkSize() int {
	return cc.baseChunker.GetChunkSize()
}

// GetChunkOverlap returns the configured chunk overlap
func (cc *ContextualChunker) GetChunkOverlap() int {
	return cc.baseChunker.GetChunkOverlap()
}

// GetSupportedLanguages returns supported languages
func (cc *ContextualChunker) GetSupportedLanguages() []string {
	return cc.baseChunker.GetSupportedLanguages()
}

// SetContextConfig updates the contextual configuration
func (cc *ContextualChunker) SetContextConfig(config *ContextualConfig) {
	cc.contextConfig = config
}

// GetContextConfig returns the current contextual configuration
func (cc *ContextualChunker) GetContextConfig() *ContextualConfig {
	return cc.contextConfig
}