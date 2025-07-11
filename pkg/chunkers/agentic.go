package chunkers

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// AgenticChunker implements LLM-driven chunking with AI decision-making
// This chunker uses Large Language Models to make intelligent decisions about
// chunk boundaries based on semantic understanding and content analysis
type AgenticChunker struct {
	config    *ChunkerConfig
	llmProvider interfaces.LLM
	
	// Token estimation components
	tokenizer      TokenizerProvider
	tokenEstimator func(string) int
	
	// Embedding components for similarity analysis
	embeddingProvider    EmbeddingProvider
	similarityCalculator SimilarityCalculator
	
	// Sentence processing
	sentenceChunker *SentenceChunker
	
	// Agentic configuration
	agenticConfig *AgenticChunkerConfig
}

// AgenticChunkerConfig contains advanced configuration for agentic chunking
type AgenticChunkerConfig struct {
	// DecisionPromptTemplate is the template for LLM decision-making
	DecisionPromptTemplate string `json:"decision_prompt_template"`
	
	// AnalysisDepth controls how deeply the LLM analyzes content
	AnalysisDepth AnalysisDepthLevel `json:"analysis_depth"`
	
	// BoundaryDetectionMode controls the boundary detection strategy
	BoundaryDetectionMode BoundaryDetectionMode `json:"boundary_detection_mode"`
	
	// ReasoningSteps controls how many reasoning steps the LLM takes
	ReasoningSteps int `json:"reasoning_steps"`
	
	// ConfidenceThreshold for accepting LLM decisions
	ConfidenceThreshold float64 `json:"confidence_threshold"`
	
	// UseSemanticContext enables context-aware chunking
	UseSemanticContext bool `json:"use_semantic_context"`
	
	// MaxLLMCalls limits the number of LLM calls per document
	MaxLLMCalls int `json:"max_llm_calls"`
	
	// EnableMultiRoundReasoning enables iterative reasoning
	EnableMultiRoundReasoning bool `json:"enable_multi_round_reasoning"`
	
	// DocumentTypeHeuristics enables document type detection
	DocumentTypeHeuristics bool `json:"document_type_heuristics"`
}

// AnalysisDepthLevel defines how deeply the LLM analyzes content
type AnalysisDepthLevel string

const (
	// AnalysisDepthBasic performs basic content analysis
	AnalysisDepthBasic AnalysisDepthLevel = "basic"
	
	// AnalysisDepthIntermediate performs moderate content analysis
	AnalysisDepthIntermediate AnalysisDepthLevel = "intermediate"
	
	// AnalysisDepthDeep performs comprehensive content analysis
	AnalysisDepthDeep AnalysisDepthLevel = "deep"
	
	// AnalysisDepthExpert performs expert-level analysis with multiple passes
	AnalysisDepthExpert AnalysisDepthLevel = "expert"
)

// BoundaryDetectionMode defines the strategy for boundary detection
type BoundaryDetectionMode string

const (
	// BoundaryModeConservative prefers fewer, larger chunks
	BoundaryModeConservative BoundaryDetectionMode = "conservative"
	
	// BoundaryModeBalanced balances chunk size and semantic coherence
	BoundaryModeBalanced BoundaryDetectionMode = "balanced"
	
	// BoundaryModeAggressive prefers more, smaller chunks
	BoundaryModeAggressive BoundaryDetectionMode = "aggressive"
	
	// BoundaryModeAdaptive adapts strategy based on content type
	BoundaryModeAdaptive BoundaryDetectionMode = "adaptive"
)

// DefaultAgenticConfig returns a default configuration for agentic chunking
func DefaultAgenticConfig() *AgenticChunkerConfig {
	return &AgenticChunkerConfig{
		DecisionPromptTemplate: `You are an expert document analyst tasked with determining optimal chunk boundaries for text processing.

Analyze the following text segments and determine where to place chunk boundaries. Consider:
1. Semantic coherence and topic boundaries
2. Optimal chunk size for downstream processing
3. Preservation of important context and relationships
4. Document structure and logical flow

Text segments:
{{.Segments}}

Target chunk size: {{.TargetSize}} tokens
Current chunk sizes: {{.CurrentSizes}}
Document type: {{.DocumentType}}

Provide your analysis in the following JSON format:
{
  "boundaries": [list of sentence indices where chunks should end],
  "reasoning": "Your reasoning for each boundary decision",
  "confidence": 0.95,
  "chunk_summaries": ["brief summary of each chunk's content"],
  "optimization_notes": "any suggestions for improvement"
}`,
		AnalysisDepth:             AnalysisDepthIntermediate,
		BoundaryDetectionMode:     BoundaryModeBalanced,
		ReasoningSteps:            3,
		ConfidenceThreshold:       0.7,
		UseSemanticContext:        true,
		MaxLLMCalls:              10,
		EnableMultiRoundReasoning: true,
		DocumentTypeHeuristics:    true,
	}
}

// NewAgenticChunker creates a new agentic chunker
func NewAgenticChunker(config *ChunkerConfig, llmProvider interfaces.LLM) (*AgenticChunker, error) {
	if config == nil {
		config = DefaultChunkerConfig()
	}
	
	if llmProvider == nil {
		return nil, fmt.Errorf("LLM provider is required for agentic chunker")
	}
	
	// Create sentence chunker for initial processing
	sentenceChunker, err := NewSentenceChunker(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create sentence chunker: %w", err)
	}
	
	// Initialize embedding provider (optional)
	embeddingFactory := NewEmbeddingFactory(NewMemoryEmbeddingCache(1000))
	embeddingConfig := DefaultEmbeddingConfig()
	embeddingProvider, err := embeddingFactory.CreateProvider(embeddingConfig)
	if err != nil {
		fmt.Printf("Warning: embedding provider not available, using basic analysis: %v\n", err)
		embeddingProvider = nil
	}
	
	// Initialize tokenizer
	tokenizerFactory := NewTokenizerFactory()
	tokenizerConfig := DefaultTokenizerConfig()
	tokenizer, err := tokenizerFactory.CreateTokenizer(tokenizerConfig)
	if err != nil {
		fmt.Printf("Warning: advanced tokenizer not available, using fallback: %v\n", err)
		tokenizer = nil
	}
	
	similarityCalculator := NewCosineSimilarityCalculator()
	
	chunker := &AgenticChunker{
		config:               config,
		llmProvider:          llmProvider,
		tokenizer:            tokenizer,
		tokenEstimator:       defaultTokenEstimator,
		embeddingProvider:    embeddingProvider,
		similarityCalculator: similarityCalculator,
		sentenceChunker:      sentenceChunker,
		agenticConfig:        DefaultAgenticConfig(),
	}
	
	return chunker, nil
}

// SetAgenticConfig updates the agentic configuration
func (ac *AgenticChunker) SetAgenticConfig(config *AgenticChunkerConfig) {
	if config != nil {
		ac.agenticConfig = config
	}
}

// GetAgenticConfig returns the current agentic configuration
func (ac *AgenticChunker) GetAgenticConfig() *AgenticChunkerConfig {
	return ac.agenticConfig
}

// Chunk splits text using LLM-driven decision making
func (ac *AgenticChunker) Chunk(ctx context.Context, text string) ([]*Chunk, error) {
	return ac.ChunkWithMetadata(ctx, text, nil)
}

// ChunkWithMetadata splits text into chunks using AI-driven analysis
func (ac *AgenticChunker) ChunkWithMetadata(ctx context.Context, text string, metadata map[string]interface{}) ([]*Chunk, error) {
	if text == "" {
		return []*Chunk{}, nil
	}
	
	startTime := time.Now()
	
	// Step 1: Extract sentences and prepare for analysis
	sentences := ac.extractSentences(text)
	if len(sentences) == 0 {
		sentences = []string{strings.TrimSpace(text)}
	}
	
	// Step 2: Detect document type and adapt strategy
	documentType := ac.detectDocumentType(text)
	if ac.agenticConfig.DocumentTypeHeuristics {
		ac.adaptConfigForDocumentType(documentType)
	}
	
	// Step 3: Perform multi-round LLM analysis
	boundaries, decisions, err := ac.performLLMAnalysis(ctx, sentences, text, documentType)
	if err != nil {
		// Fallback to semantic chunking on LLM failure
		fmt.Printf("Warning: LLM analysis failed, falling back to semantic chunking: %v\n", err)
		return ac.fallbackToSemanticChunking(ctx, text, metadata)
	}
	
	// Step 4: Create chunks based on LLM decisions
	chunks := ac.createChunksFromBoundaries(sentences, boundaries, text, metadata, decisions)
	
	// Step 5: Add agentic metadata
	processingTime := time.Since(startTime)
	stats := CalculateStats(chunks, len(text), processingTime)
	
	for i, chunk := range chunks {
		if chunk.Metadata == nil {
			chunk.Metadata = make(map[string]interface{})
		}
		chunk.Metadata["chunking_stats"] = stats
		chunk.Metadata["chunker_type"] = string(ChunkerTypeAgentic)
		chunk.Metadata["chunk_config"] = ac.config
		chunk.Metadata["agentic_config"] = ac.agenticConfig
		chunk.Metadata["llm_decisions"] = decisions
		chunk.Metadata["document_type"] = documentType
		chunk.Metadata["boundary_confidence"] = ac.calculateBoundaryConfidence(i, decisions)
		chunk.Metadata["processing_time"] = processingTime
	}
	
	return chunks, nil
}

// DocumentType represents different types of documents
type DocumentType string

const (
	DocumentTypeAcademic    DocumentType = "academic"
	DocumentTypeTechnical   DocumentType = "technical"
	DocumentTypeNarrative   DocumentType = "narrative"
	DocumentTypeReference   DocumentType = "reference"
	DocumentTypeConversational DocumentType = "conversational"
	DocumentTypeLegal       DocumentType = "legal"
	DocumentTypeNews        DocumentType = "news"
	DocumentTypeUnknown     DocumentType = "unknown"
)

// detectDocumentType analyzes text to determine document type
func (ac *AgenticChunker) detectDocumentType(text string) DocumentType {
	// Academic indicators
	academicKeywords := []string{"abstract", "methodology", "conclusion", "hypothesis", "research", "study", "analysis", "findings", "literature review"}
	academicScore := ac.calculateKeywordScore(text, academicKeywords)
	
	// Technical indicators
	technicalKeywords := []string{"implementation", "algorithm", "function", "method", "system", "architecture", "configuration", "protocol"}
	technicalScore := ac.calculateKeywordScore(text, technicalKeywords)
	
	// Legal indicators
	legalKeywords := []string{"whereas", "therefore", "pursuant", "jurisdiction", "statute", "regulation", "contract", "agreement", "clause"}
	legalScore := ac.calculateKeywordScore(text, legalKeywords)
	
	// News indicators
	newsKeywords := []string{"breaking", "reported", "according to", "spokesperson", "announced", "breaking news"}
	newsScore := ac.calculateKeywordScore(text, newsKeywords)
	
	// Determine type based on highest score
	scores := map[DocumentType]float64{
		DocumentTypeAcademic:  academicScore,
		DocumentTypeTechnical: technicalScore,
		DocumentTypeLegal:     legalScore,
		DocumentTypeNews:      newsScore,
	}
	
	maxScore := 0.0
	detectedType := DocumentTypeUnknown
	
	for docType, score := range scores {
		if score > maxScore {
			maxScore = score
			detectedType = docType
		}
	}
	
	// Require minimum threshold for detection
	if maxScore < 0.1 {
		detectedType = DocumentTypeUnknown
	}
	
	return detectedType
}

// calculateKeywordScore calculates the score for keywords in text
func (ac *AgenticChunker) calculateKeywordScore(text string, keywords []string) float64 {
	textLower := strings.ToLower(text)
	words := strings.Fields(textLower)
	totalWords := len(words)
	
	if totalWords == 0 {
		return 0
	}
	
	matches := 0
	for _, keyword := range keywords {
		if strings.Contains(textLower, strings.ToLower(keyword)) {
			matches++
		}
	}
	
	return float64(matches) / float64(len(keywords))
}

// adaptConfigForDocumentType adapts chunking strategy based on document type
func (ac *AgenticChunker) adaptConfigForDocumentType(docType DocumentType) {
	switch docType {
	case DocumentTypeAcademic:
		// Academic papers benefit from preserving section boundaries
		ac.agenticConfig.BoundaryDetectionMode = BoundaryModeConservative
		ac.agenticConfig.AnalysisDepth = AnalysisDepthDeep
		
	case DocumentTypeTechnical:
		// Technical docs need to preserve code blocks and procedures
		ac.agenticConfig.BoundaryDetectionMode = BoundaryModeBalanced
		ac.agenticConfig.AnalysisDepth = AnalysisDepthIntermediate
		
	case DocumentTypeNarrative:
		// Narratives can be chunked more aggressively
		ac.agenticConfig.BoundaryDetectionMode = BoundaryModeAggressive
		ac.agenticConfig.AnalysisDepth = AnalysisDepthBasic
		
	case DocumentTypeLegal:
		// Legal docs require careful preservation of clauses
		ac.agenticConfig.BoundaryDetectionMode = BoundaryModeConservative
		ac.agenticConfig.AnalysisDepth = AnalysisDepthExpert
		
	default:
		// Use balanced approach for unknown types
		ac.agenticConfig.BoundaryDetectionMode = BoundaryModeBalanced
		ac.agenticConfig.AnalysisDepth = AnalysisDepthIntermediate
	}
}

// LLMDecision represents a decision made by the LLM
type LLMDecision struct {
	Boundaries       []int    `json:"boundaries"`
	Reasoning        string   `json:"reasoning"`
	Confidence       float64  `json:"confidence"`
	ChunkSummaries   []string `json:"chunk_summaries"`
	OptimizationNotes string   `json:"optimization_notes"`
	AnalysisRound    int      `json:"analysis_round"`
	ProcessingTime   time.Duration `json:"processing_time"`
}

// performLLMAnalysis conducts multi-round LLM analysis for boundary detection
func (ac *AgenticChunker) performLLMAnalysis(ctx context.Context, sentences []string, fullText string, docType DocumentType) ([]int, []*LLMDecision, error) {
	var allDecisions []*LLMDecision
	var finalBoundaries []int
	
	callCount := 0
	maxCalls := ac.agenticConfig.MaxLLMCalls
	
	// Initial analysis
	for round := 0; round < ac.agenticConfig.ReasoningSteps && callCount < maxCalls; round++ {
		decision, err := ac.analyzeBoundariesWithLLM(ctx, sentences, fullText, docType, round)
		if err != nil {
			return nil, nil, fmt.Errorf("LLM analysis failed in round %d: %w", round, err)
		}
		
		decision.AnalysisRound = round
		allDecisions = append(allDecisions, decision)
		callCount++
		
		// Check confidence threshold
		if decision.Confidence >= ac.agenticConfig.ConfidenceThreshold {
			finalBoundaries = decision.Boundaries
			break
		}
		
		// Multi-round reasoning: refine based on previous decisions
		if ac.agenticConfig.EnableMultiRoundReasoning && round < ac.agenticConfig.ReasoningSteps-1 {
			sentences = ac.refineSentencesBasedOnDecision(sentences, decision)
		}
	}
	
	// If no decision met confidence threshold, use the best one
	if len(finalBoundaries) == 0 && len(allDecisions) > 0 {
		bestDecision := ac.selectBestDecision(allDecisions)
		finalBoundaries = bestDecision.Boundaries
	}
	
	return finalBoundaries, allDecisions, nil
}

// analyzeBoundariesWithLLM uses LLM to analyze and determine chunk boundaries
func (ac *AgenticChunker) analyzeBoundariesWithLLM(ctx context.Context, sentences []string, fullText string, docType DocumentType, round int) (*LLMDecision, error) {
	startTime := time.Now()
	
	// Prepare context for LLM
	analysisContext := ac.prepareAnalysisContext(sentences, fullText, docType, round)
	
	// Create prompt
	prompt := ac.createAnalysisPrompt(analysisContext)
	
	// Convert to MessageList
	messageList := types.MessageList{}
	for _, msg := range prompt {
		messageDict := types.MessageDict{
			Role:    types.MessageRole(msg["role"].(string)),
			Content: msg["content"].(string),
		}
		messageList = append(messageList, messageDict)
	}
	
	// Call LLM
	response, err := ac.llmProvider.Generate(ctx, messageList)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}
	
	// Parse LLM response
	decision, err := ac.parseLLMResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}
	
	decision.ProcessingTime = time.Since(startTime)
	
	// Validate and adjust boundaries
	decision.Boundaries = ac.validateBoundaries(decision.Boundaries, len(sentences))
	
	return decision, nil
}

// AnalysisContext contains context information for LLM analysis
type AnalysisContext struct {
	Segments      []string     `json:"segments"`
	TargetSize    int          `json:"target_size"`
	CurrentSizes  []int        `json:"current_sizes"`
	DocumentType  DocumentType `json:"document_type"`
	AnalysisDepth AnalysisDepthLevel `json:"analysis_depth"`
	Round         int          `json:"round"`
	PreviousDecisions []*LLMDecision `json:"previous_decisions,omitempty"`
}

// prepareAnalysisContext prepares context information for LLM analysis
func (ac *AgenticChunker) prepareAnalysisContext(sentences []string, fullText string, docType DocumentType, round int) *AnalysisContext {
	context := &AnalysisContext{
		Segments:      sentences,
		TargetSize:    ac.config.ChunkSize,
		CurrentSizes:  make([]int, len(sentences)),
		DocumentType:  docType,
		AnalysisDepth: ac.agenticConfig.AnalysisDepth,
		Round:         round,
	}
	
	// Calculate current sentence sizes
	for i, sentence := range sentences {
		context.CurrentSizes[i] = ac.EstimateTokens(sentence)
	}
	
	return context
}

// createAnalysisPrompt creates the prompt for LLM analysis
func (ac *AgenticChunker) createAnalysisPrompt(context *AnalysisContext) []map[string]interface{} {
	// Template substitution
	prompt := ac.agenticConfig.DecisionPromptTemplate
	prompt = strings.ReplaceAll(prompt, "{{.Segments}}", ac.formatSegments(context.Segments))
	prompt = strings.ReplaceAll(prompt, "{{.TargetSize}}", fmt.Sprintf("%d", context.TargetSize))
	prompt = strings.ReplaceAll(prompt, "{{.CurrentSizes}}", ac.formatSizes(context.CurrentSizes))
	prompt = strings.ReplaceAll(prompt, "{{.DocumentType}}", string(context.DocumentType))
	
	// Add analysis depth specific instructions
	depthInstructions := ac.getDepthInstructions(context.AnalysisDepth)
	prompt += "\n\n" + depthInstructions
	
	// Add round-specific context
	if context.Round > 0 && len(context.PreviousDecisions) > 0 {
		prompt += "\n\nPrevious analysis results:\n" + ac.formatPreviousDecisions(context.PreviousDecisions)
	}
	
	return []map[string]interface{}{
		{
			"role":    "user",
			"content": prompt,
		},
	}
}

// formatSegments formats sentences for LLM consumption
func (ac *AgenticChunker) formatSegments(sentences []string) string {
	var formatted strings.Builder
	for i, sentence := range sentences {
		formatted.WriteString(fmt.Sprintf("[%d] %s\n", i, sentence))
	}
	return formatted.String()
}

// formatSizes formats token counts for display
func (ac *AgenticChunker) formatSizes(sizes []int) string {
	sizeStrs := make([]string, len(sizes))
	for i, size := range sizes {
		sizeStrs[i] = fmt.Sprintf("%d", size)
	}
	return "[" + strings.Join(sizeStrs, ", ") + "]"
}

// getDepthInstructions returns instructions based on analysis depth
func (ac *AgenticChunker) getDepthInstructions(depth AnalysisDepthLevel) string {
	switch depth {
	case AnalysisDepthBasic:
		return "Perform basic topic boundary detection focusing on major topic changes."
	case AnalysisDepthIntermediate:
		return "Perform detailed analysis considering semantic relationships, topic coherence, and document structure."
	case AnalysisDepthDeep:
		return "Perform comprehensive analysis including subtle topic transitions, argument flow, and contextual dependencies."
	case AnalysisDepthExpert:
		return "Perform expert-level analysis with attention to domain-specific patterns, rhetorical structure, and complex relationships."
	default:
		return "Perform standard boundary analysis."
	}
}

// formatPreviousDecisions formats previous decisions for context
func (ac *AgenticChunker) formatPreviousDecisions(decisions []*LLMDecision) string {
	var formatted strings.Builder
	for i, decision := range decisions {
		formatted.WriteString(fmt.Sprintf("Round %d: boundaries %v, confidence %.2f\n", 
			i, decision.Boundaries, decision.Confidence))
		formatted.WriteString(fmt.Sprintf("Reasoning: %s\n\n", decision.Reasoning))
	}
	return formatted.String()
}

// parseLLMResponse parses the LLM response into a structured decision
func (ac *AgenticChunker) parseLLMResponse(response string) (*LLMDecision, error) {
	// Try to extract JSON from response
	jsonPattern := regexp.MustCompile(`\{[\s\S]*\}`)
	jsonMatch := jsonPattern.FindString(response)
	
	if jsonMatch == "" {
		return nil, fmt.Errorf("no JSON found in LLM response")
	}
	
	var decision LLMDecision
	if err := json.Unmarshal([]byte(jsonMatch), &decision); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	
	// Validate decision
	if decision.Confidence < 0 || decision.Confidence > 1 {
		decision.Confidence = 0.5 // Default to medium confidence
	}
	
	return &decision, nil
}

// validateBoundaries ensures boundaries are valid for the given sentence count
func (ac *AgenticChunker) validateBoundaries(boundaries []int, sentenceCount int) []int {
	var valid []int
	
	for _, boundary := range boundaries {
		// Ensure boundary is within valid range
		if boundary > 0 && boundary < sentenceCount {
			// Check for duplicates
			isDuplicate := false
			for _, existing := range valid {
				if existing == boundary {
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				valid = append(valid, boundary)
			}
		}
	}
	
	return valid
}

// refineSentencesBasedOnDecision refines sentence analysis based on previous decision
func (ac *AgenticChunker) refineSentencesBasedOnDecision(sentences []string, decision *LLMDecision) []string {
	// For now, return original sentences
	// In a more sophisticated implementation, this could:
	// - Combine sentences that were identified as strongly related
	// - Split complex sentences identified as containing multiple topics
	// - Adjust sentence boundaries based on optimization notes
	return sentences
}

// selectBestDecision selects the best decision from multiple rounds
func (ac *AgenticChunker) selectBestDecision(decisions []*LLMDecision) *LLMDecision {
	if len(decisions) == 0 {
		return nil
	}
	
	// Select based on highest confidence
	best := decisions[0]
	for _, decision := range decisions[1:] {
		if decision.Confidence > best.Confidence {
			best = decision
		}
	}
	
	return best
}

// createChunksFromBoundaries creates chunks based on LLM-determined boundaries
func (ac *AgenticChunker) createChunksFromBoundaries(sentences []string, boundaries []int, originalText string, metadata map[string]interface{}, decisions []*LLMDecision) []*Chunk {
	if len(sentences) == 0 {
		return []*Chunk{}
	}
	
	// Sort boundaries
	sortedBoundaries := make([]int, len(boundaries))
	copy(sortedBoundaries, boundaries)
	for i := 0; i < len(sortedBoundaries)-1; i++ {
		for j := i + 1; j < len(sortedBoundaries); j++ {
			if sortedBoundaries[i] > sortedBoundaries[j] {
				sortedBoundaries[i], sortedBoundaries[j] = sortedBoundaries[j], sortedBoundaries[i]
			}
		}
	}
	
	var chunks []*Chunk
	currentStart := 0
	textPosition := 0
	
	// Add end boundary
	allBoundaries := append(sortedBoundaries, len(sentences))
	
	for i, boundary := range allBoundaries {
		if boundary <= currentStart {
			continue
		}
		
		// Extract sentences for this chunk
		chunkSentences := sentences[currentStart:boundary]
		if len(chunkSentences) == 0 {
			currentStart = boundary
			continue
		}
		
		chunkText := strings.Join(chunkSentences, " ")
		
		chunk := &Chunk{
			Text:       chunkText,
			TokenCount: ac.EstimateTokens(chunkText),
			Sentences:  make([]string, len(chunkSentences)),
			StartIndex: textPosition,
			EndIndex:   textPosition + len(chunkText),
			Metadata:   make(map[string]interface{}),
			CreatedAt:  time.Now(),
		}
		
		copy(chunk.Sentences, chunkSentences)
		
		// Copy base metadata
		if metadata != nil {
			for k, v := range metadata {
				chunk.Metadata[k] = v
			}
		}
		
		// Add agentic-specific metadata
		chunk.Metadata["chunk_index"] = i
		chunk.Metadata["sentence_range"] = fmt.Sprintf("%d-%d", currentStart, boundary-1)
		chunk.Metadata["llm_reasoning"] = ac.getChunkReasoning(i, decisions)
		chunk.Metadata["boundary_confidence"] = ac.calculateBoundaryConfidence(i, decisions)
		
		chunks = append(chunks, chunk)
		
		textPosition = ac.findTextPosition(originalText, chunkText, textPosition)
		currentStart = boundary
	}
	
	return chunks
}

// getChunkReasoning extracts reasoning for a specific chunk from LLM decisions
func (ac *AgenticChunker) getChunkReasoning(chunkIndex int, decisions []*LLMDecision) string {
	if len(decisions) == 0 {
		return "No LLM reasoning available"
	}
	
	// Use the best decision
	bestDecision := ac.selectBestDecision(decisions)
	if bestDecision == nil {
		return "No valid LLM decision found"
	}
	
	// Try to extract chunk-specific reasoning from summaries
	if chunkIndex < len(bestDecision.ChunkSummaries) {
		return bestDecision.ChunkSummaries[chunkIndex]
	}
	
	return bestDecision.Reasoning
}

// calculateBoundaryConfidence calculates confidence for a specific boundary
func (ac *AgenticChunker) calculateBoundaryConfidence(chunkIndex int, decisions []*LLMDecision) float64 {
	if len(decisions) == 0 {
		return 0.0
	}
	
	// Average confidence across all decisions
	totalConfidence := 0.0
	for _, decision := range decisions {
		totalConfidence += decision.Confidence
	}
	
	return totalConfidence / float64(len(decisions))
}

// fallbackToSemanticChunking provides fallback when LLM analysis fails
func (ac *AgenticChunker) fallbackToSemanticChunking(ctx context.Context, text string, metadata map[string]interface{}) ([]*Chunk, error) {
	// Create a semantic chunker as fallback
	semanticChunker, err := NewSemanticChunker(ac.config)
	if err != nil {
		// If semantic chunker also fails, use sentence chunker
		return ac.sentenceChunker.ChunkWithMetadata(ctx, text, metadata)
	}
	
	chunks, err := semanticChunker.ChunkWithMetadata(ctx, text, metadata)
	if err != nil {
		// Final fallback to sentence chunker
		return ac.sentenceChunker.ChunkWithMetadata(ctx, text, metadata)
	}
	
	// Add fallback metadata
	for _, chunk := range chunks {
		if chunk.Metadata == nil {
			chunk.Metadata = make(map[string]interface{})
		}
		chunk.Metadata["fallback_mode"] = "semantic"
		chunk.Metadata["agentic_failed"] = true
	}
	
	return chunks, nil
}

// extractSentences extracts sentences from text
func (ac *AgenticChunker) extractSentences(text string) []string {
	return ac.sentenceChunker.extractSentences(text)
}

// findTextPosition finds the position of text in the original document
func (ac *AgenticChunker) findTextPosition(originalText, searchText string, startFrom int) int {
	searchText = strings.TrimSpace(searchText)
	if searchText == "" {
		return startFrom
	}
	
	pos := strings.Index(originalText[startFrom:], searchText)
	if pos == -1 {
		return startFrom
	}
	return startFrom + pos
}

// EstimateTokens estimates the number of tokens in text
func (ac *AgenticChunker) EstimateTokens(text string) int {
	if ac.tokenizer != nil {
		count, err := ac.tokenizer.CountTokens(text)
		if err == nil {
			return count
		}
	}
	return ac.tokenEstimator(text)
}

// GetConfig returns the current chunker configuration
func (ac *AgenticChunker) GetConfig() *ChunkerConfig {
	config := *ac.config
	return &config
}

// SetConfig updates the chunker configuration
func (ac *AgenticChunker) SetConfig(config *ChunkerConfig) error {
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
	
	ac.config = config
	return nil
}

// GetChunkSize returns the configured chunk size
func (ac *AgenticChunker) GetChunkSize() int {
	return ac.config.ChunkSize
}

// GetChunkOverlap returns the configured chunk overlap
func (ac *AgenticChunker) GetChunkOverlap() int {
	return ac.config.ChunkOverlap
}

// GetSupportedLanguages returns supported languages for this chunker
func (ac *AgenticChunker) GetSupportedLanguages() []string {
	return []string{"en"} // Currently optimized for English
}

// Use the defaultTokenEstimator from sentence.go to avoid duplication