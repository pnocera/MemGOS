package chunkers

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
)

// QualityMetricsCalculator provides comprehensive quality assessment for chunks
type QualityMetricsCalculator struct {
	config            *QualityMetricsConfig
	embeddingProvider EmbeddingProvider
	llmProvider       interfaces.LLM
	similarityCalc    SimilarityCalculator
	
	// Metrics history for trending analysis
	metricsHistory []*QualityAssessment
}

// QualityAssessment contains comprehensive quality metrics for a chunk or set of chunks
type QualityAssessment struct {
	// Timestamp of assessment
	Timestamp time.Time `json:"timestamp"`
	
	// Overall quality score (0-1)
	OverallScore float64 `json:"overall_score"`
	
	// Individual metric scores
	Coherence         float64 `json:"coherence"`
	Completeness      float64 `json:"completeness"`
	Relevance         float64 `json:"relevance"`
	InformationDensity float64 `json:"information_density"`
	Readability       float64 `json:"readability"`
	SemanticIntegrity float64 `json:"semantic_integrity"`
	StructurePreservation float64 `json:"structure_preservation"`
	OverlapOptimization float64 `json:"overlap_optimization"`
	
	// Detailed analysis
	Issues     []QualityIssue     `json:"issues,omitempty"`
	Suggestions []QualitySuggestion `json:"suggestions,omitempty"`
	
	// Metadata
	ChunkCount    int                    `json:"chunk_count"`
	TotalTokens   int                    `json:"total_tokens"`
	ProcessingTime time.Duration         `json:"processing_time"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// QualityIssue represents a detected quality issue
type QualityIssue struct {
	Type        QualityIssueType `json:"type"`
	Severity    IssueSeverity    `json:"severity"`
	Description string           `json:"description"`
	ChunkIndex  int              `json:"chunk_index,omitempty"`
	Position    *TextPosition    `json:"position,omitempty"`
	Impact      float64          `json:"impact"` // Impact on overall quality (0-1)
}

// QualitySuggestion represents an improvement suggestion
type QualitySuggestion struct {
	Type           SuggestionType `json:"type"`
	Description    string         `json:"description"`
	ExpectedImprovement float64   `json:"expected_improvement"` // Expected improvement (0-1)
	Implementation string         `json:"implementation,omitempty"`
}

// QualityIssueType represents different types of quality issues
type QualityIssueType string

const (
	IssueTypeCoherence       QualityIssueType = "coherence"
	IssueTypeCompleteness    QualityIssueType = "completeness"
	IssueTypeRedundancy      QualityIssueType = "redundancy"
	IssueTypeFragmentation   QualityIssueType = "fragmentation"
	IssueTypeContextLoss     QualityIssueType = "context_loss"
	IssueTypeStructuralBreak QualityIssueType = "structural_break"
	IssueTypeInformationLoss QualityIssueType = "information_loss"
)

// IssueSeverity represents the severity of quality issues
type IssueSeverity string

const (
	SeverityLow      IssueSeverity = "low"
	SeverityModerate IssueSeverity = "moderate"
	SeverityHigh     IssueSeverity = "high"
	SeverityCritical IssueSeverity = "critical"
)

// SuggestionType represents different types of improvement suggestions
type SuggestionType string

const (
	SuggestionAdjustChunkSize    SuggestionType = "adjust_chunk_size"
	SuggestionAdjustOverlap      SuggestionType = "adjust_overlap"
	SuggestionImproveCoherence   SuggestionType = "improve_coherence"
	SuggestionReduceRedundancy   SuggestionType = "reduce_redundancy"
	SuggestionPreserveStructure  SuggestionType = "preserve_structure"
	SuggestionEnhanceContext     SuggestionType = "enhance_context"
)

// TextPosition represents a position in text
type TextPosition struct {
	Start int `json:"start"`
	End   int `json:"end"`
	Line  int `json:"line,omitempty"`
}

// NewQualityMetricsCalculator creates a new quality metrics calculator
func NewQualityMetricsCalculator(config *QualityMetricsConfig, embeddingProvider EmbeddingProvider, llmProvider interfaces.LLM) *QualityMetricsCalculator {
	if config == nil {
		config = DefaultQualityConfig()
	}
	
	return &QualityMetricsCalculator{
		config:            config,
		embeddingProvider: embeddingProvider,
		llmProvider:       llmProvider,
		similarityCalc:    NewCosineSimilarityCalculator(),
		metricsHistory:    make([]*QualityAssessment, 0),
	}
}

// AssessChunks performs comprehensive quality assessment on a set of chunks
func (qmc *QualityMetricsCalculator) AssessChunks(ctx context.Context, chunks []*Chunk, originalText string) (*QualityAssessment, error) {
	if len(chunks) == 0 {
		return &QualityAssessment{
			Timestamp:    time.Now(),
			OverallScore: 0.0,
			ChunkCount:   0,
		}, nil
	}
	
	startTime := time.Now()
	
	assessment := &QualityAssessment{
		Timestamp:      startTime,
		ChunkCount:     len(chunks),
		TotalTokens:    qmc.calculateTotalTokens(chunks),
		Issues:         make([]QualityIssue, 0),
		Suggestions:    make([]QualitySuggestion, 0),
		Metadata:       make(map[string]interface{}),
	}
	
	// Calculate individual metrics based on configuration
	for _, metric := range qmc.config.MetricsToCollect {
		score, issues, suggestions, err := qmc.calculateMetric(ctx, metric, chunks, originalText)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate metric %s: %w", metric, err)
		}
		
		qmc.setMetricScore(assessment, metric, score)
		assessment.Issues = append(assessment.Issues, issues...)
		assessment.Suggestions = append(assessment.Suggestions, suggestions...)
	}
	
	// Calculate overall score
	assessment.OverallScore = qmc.calculateOverallScore(assessment)
	
	// Add processing time
	assessment.ProcessingTime = time.Since(startTime)
	
	// Store in history
	qmc.addToHistory(assessment)
	
	return assessment, nil
}

// calculateMetric calculates a specific quality metric
func (qmc *QualityMetricsCalculator) calculateMetric(ctx context.Context, metric QualityMetricType, chunks []*Chunk, originalText string) (float64, []QualityIssue, []QualitySuggestion, error) {
	switch metric {
	case MetricCoherence:
		return qmc.calculateCoherence(ctx, chunks)
	case MetricCompleteness:
		return qmc.calculateCompleteness(ctx, chunks, originalText)
	case MetricRelevance:
		return qmc.calculateRelevance(ctx, chunks)
	case MetricInformation:
		return qmc.calculateInformationDensity(ctx, chunks)
	case MetricReadability:
		return qmc.calculateReadability(ctx, chunks)
	case MetricSemanticIntegrity:
		return qmc.calculateSemanticIntegrity(ctx, chunks)
	case MetricStructure:
		return qmc.calculateStructurePreservation(ctx, chunks, originalText)
	case MetricOverlap:
		return qmc.calculateOverlapOptimization(ctx, chunks)
	default:
		return 0.0, nil, nil, fmt.Errorf("unknown metric: %s", metric)
	}
}

// calculateCoherence measures semantic coherence within and between chunks
func (qmc *QualityMetricsCalculator) calculateCoherence(ctx context.Context, chunks []*Chunk) (float64, []QualityIssue, []QualitySuggestion, error) {
	if len(chunks) <= 1 {
		return 1.0, nil, nil, nil
	}
	
	var issues []QualityIssue
	var suggestions []QualitySuggestion
	
	// Calculate intra-chunk coherence (within chunks)
	intraCoherenceSum := 0.0
	for i, chunk := range chunks {
		coherence := qmc.calculateIntraChunkCoherence(chunk)
		intraCoherenceSum += coherence
		
		if coherence < 0.6 {
			issues = append(issues, QualityIssue{
				Type:        IssueTypeCoherence,
				Severity:    SeverityModerate,
				Description: fmt.Sprintf("Low intra-chunk coherence (%.2f) in chunk %d", coherence, i),
				ChunkIndex:  i,
				Impact:      (0.6 - coherence) * 0.5,
			})
		}
	}
	intraCoherence := intraCoherenceSum / float64(len(chunks))
	
	// Calculate inter-chunk coherence (between chunks)
	interCoherence := 0.0
	if qmc.embeddingProvider != nil {
		var err error
		interCoherence, err = qmc.calculateInterChunkCoherence(ctx, chunks)
		if err != nil {
			// Fallback to text-based coherence
			interCoherence = qmc.calculateTextBasedInterCoherence(chunks)
		}
	} else {
		interCoherence = qmc.calculateTextBasedInterCoherence(chunks)
	}
	
	// Check for coherence breaks
	if interCoherence < 0.5 {
		issues = append(issues, QualityIssue{
			Type:        IssueTypeCoherence,
			Severity:    SeverityHigh,
			Description: fmt.Sprintf("Low inter-chunk coherence (%.2f)", interCoherence),
			Impact:      (0.5 - interCoherence) * 0.8,
		})
		
		suggestions = append(suggestions, QualitySuggestion{
			Type:                SuggestionAdjustOverlap,
			Description:         "Increase chunk overlap to improve semantic coherence between chunks",
			ExpectedImprovement: 0.2,
		})
	}
	
	// Combined coherence score (weighted average)
	overallCoherence := 0.7*intraCoherence + 0.3*interCoherence
	
	return overallCoherence, issues, suggestions, nil
}

// calculateIntraChunkCoherence measures coherence within a single chunk
func (qmc *QualityMetricsCalculator) calculateIntraChunkCoherence(chunk *Chunk) float64 {
	if len(chunk.Sentences) <= 1 {
		return 1.0
	}
	
	// Simple approach: measure vocabulary consistency
	wordCounts := make(map[string]int)
	totalWords := 0
	
	for _, sentence := range chunk.Sentences {
		words := strings.Fields(strings.ToLower(sentence))
		for _, word := range words {
			word = strings.Trim(word, ".,!?;:\"'()[]{}...")
			if len(word) > 2 {
				wordCounts[word]++
				totalWords++
			}
		}
	}
	
	if totalWords == 0 {
		return 1.0
	}
	
	// Calculate vocabulary consistency (repeated words indicate coherence)
	repeatedWords := 0
	for _, count := range wordCounts {
		if count > 1 {
			repeatedWords += count - 1
		}
	}
	
	coherence := float64(repeatedWords) / float64(totalWords)
	return math.Min(1.0, coherence*2) // Scale to 0-1 range
}

// calculateInterChunkCoherence measures coherence between chunks using embeddings
func (qmc *QualityMetricsCalculator) calculateInterChunkCoherence(ctx context.Context, chunks []*Chunk) (float64, error) {
	if len(chunks) <= 1 {
		return 1.0, nil
	}
	
	// Get embeddings for all chunks
	embeddings := make([][]float64, len(chunks))
	for i, chunk := range chunks {
		embedding, err := qmc.embeddingProvider.GetEmbedding(ctx, chunk.Text)
		if err != nil {
			return 0, err
		}
		embeddings[i] = embedding
	}
	
	// Calculate average similarity between adjacent chunks
	totalSimilarity := 0.0
	for i := 0; i < len(embeddings)-1; i++ {
		similarity, err := qmc.similarityCalc.CosineSimilarity(embeddings[i], embeddings[i+1])
		if err != nil {
			return 0, err
		}
		totalSimilarity += similarity
	}
	
	return totalSimilarity / float64(len(embeddings)-1), nil
}

// calculateTextBasedInterCoherence calculates coherence using text analysis
func (qmc *QualityMetricsCalculator) calculateTextBasedInterCoherence(chunks []*Chunk) float64 {
	if len(chunks) <= 1 {
		return 1.0
	}
	
	totalCoherence := 0.0
	for i := 0; i < len(chunks)-1; i++ {
		coherence := qmc.calculateTextSimilarity(chunks[i].Text, chunks[i+1].Text)
		totalCoherence += coherence
	}
	
	return totalCoherence / float64(len(chunks)-1)
}

// calculateTextSimilarity calculates similarity between two texts
func (qmc *QualityMetricsCalculator) calculateTextSimilarity(text1, text2 string) float64 {
	words1 := qmc.extractKeywords(text1)
	words2 := qmc.extractKeywords(text2)
	
	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}
	
	// Calculate Jaccard similarity
	intersection := 0
	union := make(map[string]bool)
	
	for word := range words1 {
		union[word] = true
	}
	for word := range words2 {
		union[word] = true
		if words1[word] {
			intersection++
		}
	}
	
	if len(union) == 0 {
		return 0.0
	}
	
	return float64(intersection) / float64(len(union))
}

// extractKeywords extracts keywords from text
func (qmc *QualityMetricsCalculator) extractKeywords(text string) map[string]bool {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "is": true,
		"are": true, "was": true, "were": true, "be": true, "been": true,
	}
	
	words := strings.Fields(strings.ToLower(text))
	keywords := make(map[string]bool)
	
	for _, word := range words {
		word = strings.Trim(word, ".,!?;:\"'()[]{}...")
		if len(word) > 2 && !stopWords[word] {
			keywords[word] = true
		}
	}
	
	return keywords
}

// calculateCompleteness measures how completely the original information is preserved
func (qmc *QualityMetricsCalculator) calculateCompleteness(ctx context.Context, chunks []*Chunk, originalText string) (float64, []QualityIssue, []QualitySuggestion, error) {
	if originalText == "" {
		return 1.0, nil, nil, nil
	}
	
	var issues []QualityIssue
	var suggestions []QualitySuggestion
	
	// Combine all chunk texts
	combinedText := ""
	for _, chunk := range chunks {
		combinedText += chunk.Text + " "
	}
	
	// Calculate character preservation ratio
	charPreservation := float64(len(combinedText)) / float64(len(originalText))
	
	// Calculate word preservation ratio
	originalWords := len(strings.Fields(originalText))
	combinedWords := len(strings.Fields(combinedText))
	wordPreservation := float64(combinedWords) / float64(originalWords)
	
	// Calculate semantic preservation (if embeddings available)
	semanticPreservation := 1.0
	if qmc.embeddingProvider != nil {
		var err error
		semanticPreservation, err = qmc.calculateSemanticPreservation(ctx, originalText, combinedText)
		if err != nil {
			// Fallback to word-based calculation
			semanticPreservation = wordPreservation
		}
	} else {
		semanticPreservation = wordPreservation
	}
	
	// Check for significant information loss
	if wordPreservation < 0.8 {
		issues = append(issues, QualityIssue{
			Type:        IssueTypeInformationLoss,
			Severity:    SeverityHigh,
			Description: fmt.Sprintf("Significant word loss: %.1f%% preserved", wordPreservation*100),
			Impact:      (0.8 - wordPreservation) * 0.9,
		})
		
		suggestions = append(suggestions, QualitySuggestion{
			Type:                SuggestionAdjustChunkSize,
			Description:         "Increase chunk size to reduce information loss",
			ExpectedImprovement: 0.15,
		})
	}
	
	// Combined completeness score
	completeness := 0.3*charPreservation + 0.3*wordPreservation + 0.4*semanticPreservation
	return math.Min(1.0, completeness), issues, suggestions, nil
}

// calculateSemanticPreservation measures semantic preservation using embeddings
func (qmc *QualityMetricsCalculator) calculateSemanticPreservation(ctx context.Context, originalText, combinedText string) (float64, error) {
	originalEmbedding, err := qmc.embeddingProvider.GetEmbedding(ctx, originalText)
	if err != nil {
		return 0, err
	}
	
	combinedEmbedding, err := qmc.embeddingProvider.GetEmbedding(ctx, combinedText)
	if err != nil {
		return 0, err
	}
	
	similarity, err := qmc.similarityCalc.CosineSimilarity(originalEmbedding, combinedEmbedding)
	if err != nil {
		return 0, err
	}
	
	return similarity, nil
}

// calculateRelevance measures relevance of chunk content
func (qmc *QualityMetricsCalculator) calculateRelevance(ctx context.Context, chunks []*Chunk) (float64, []QualityIssue, []QualitySuggestion, error) {
	// For relevance, we need a query or topic context
	// In absence of that, we measure content informativeness
	
	var issues []QualityIssue
	var suggestions []QualitySuggestion
	
	totalRelevance := 0.0
	for i, chunk := range chunks {
		relevance := qmc.calculateContentInformativeness(chunk)
		totalRelevance += relevance
		
		if relevance < 0.4 {
			issues = append(issues, QualityIssue{
				Type:        IssueTypeInformationLoss,
				Severity:    SeverityModerate,
				Description: fmt.Sprintf("Low content informativeness (%.2f) in chunk %d", relevance, i),
				ChunkIndex:  i,
				Impact:      (0.4 - relevance) * 0.3,
			})
		}
	}
	
	avgRelevance := totalRelevance / float64(len(chunks))
	
	if avgRelevance < 0.5 {
		suggestions = append(suggestions, QualitySuggestion{
			Type:                SuggestionEnhanceContext,
			Description:         "Consider adding more context to improve content relevance",
			ExpectedImprovement: 0.1,
		})
	}
	
	return avgRelevance, issues, suggestions, nil
}

// calculateContentInformativeness measures how informative content is
func (qmc *QualityMetricsCalculator) calculateContentInformativeness(chunk *Chunk) float64 {
	text := strings.ToLower(chunk.Text)
	words := strings.Fields(text)
	
	if len(words) == 0 {
		return 0.0
	}
	
	// Count informative words (longer than 3 characters, not stop words)
	stopWords := map[string]bool{
		"the": true, "and": true, "that": true, "with": true, "have": true,
		"this": true, "will": true, "you": true, "they": true, "but": true,
		"not": true, "are": true, "can": true, "had": true, "her": true,
		"was": true, "one": true, "our": true, "out": true, "day": true,
	}
	
	informativeWords := 0
	uniqueWords := make(map[string]bool)
	
	for _, word := range words {
		word = strings.Trim(word, ".,!?;:\"'()[]{}...")
		if len(word) > 3 && !stopWords[word] {
			informativeWords++
			uniqueWords[word] = true
		}
	}
	
	// Calculate informativeness as ratio of informative words and vocabulary diversity
	informativeness := float64(informativeWords) / float64(len(words))
	diversity := float64(len(uniqueWords)) / float64(len(words))
	
	return (informativeness + diversity) / 2
}

// calculateInformationDensity measures information density in chunks
func (qmc *QualityMetricsCalculator) calculateInformationDensity(ctx context.Context, chunks []*Chunk) (float64, []QualityIssue, []QualitySuggestion, error) {
	var issues []QualityIssue
	var suggestions []QualitySuggestion
	
	totalDensity := 0.0
	for i, chunk := range chunks {
		density := qmc.calculateChunkInformationDensity(chunk)
		totalDensity += density
		
		if density < 0.3 {
			issues = append(issues, QualityIssue{
				Type:        IssueTypeInformationLoss,
				Severity:    SeverityLow,
				Description: fmt.Sprintf("Low information density (%.2f) in chunk %d", density, i),
				ChunkIndex:  i,
				Impact:      (0.3 - density) * 0.2,
			})
		}
	}
	
	avgDensity := totalDensity / float64(len(chunks))
	return avgDensity, issues, suggestions, nil
}

// calculateChunkInformationDensity calculates information density for a single chunk
func (qmc *QualityMetricsCalculator) calculateChunkInformationDensity(chunk *Chunk) float64 {
	if chunk.TokenCount == 0 {
		return 0.0
	}
	
	// Count different types of informative content
	text := chunk.Text
	
	// Count entities (capitalized words that might be names, places, etc.)
	entities := 0
	words := strings.Fields(text)
	for _, word := range words {
		if len(word) > 1 && strings.ToUpper(word[:1]) == word[:1] && strings.ToLower(word[1:]) == word[1:] {
			entities++
		}
	}
	
	// Count numbers and quantitative information
	numbers := strings.Count(text, "0") + strings.Count(text, "1") + strings.Count(text, "2") + 
	          strings.Count(text, "3") + strings.Count(text, "4") + strings.Count(text, "5") + 
	          strings.Count(text, "6") + strings.Count(text, "7") + strings.Count(text, "8") + 
	          strings.Count(text, "9")
	
	// Calculate density based on informative content per token
	informativeness := float64(entities+numbers/10) / float64(chunk.TokenCount)
	
	return math.Min(1.0, informativeness*5) // Scale to 0-1 range
}

// calculateReadability measures text readability
func (qmc *QualityMetricsCalculator) calculateReadability(ctx context.Context, chunks []*Chunk) (float64, []QualityIssue, []QualitySuggestion, error) {
	var issues []QualityIssue
	var suggestions []QualitySuggestion
	
	totalReadability := 0.0
	for i, chunk := range chunks {
		readability := qmc.calculateFleschReadingEase(chunk.Text)
		totalReadability += readability
		
		if readability < 30 { // Very difficult to read
			issues = append(issues, QualityIssue{
				Type:        IssueTypeCoherence,
				Severity:    SeverityModerate,
				Description: fmt.Sprintf("Low readability score (%.1f) in chunk %d", readability, i),
				ChunkIndex:  i,
				Impact:      (30 - readability) / 100 * 0.3,
			})
		}
	}
	
	avgReadability := totalReadability / float64(len(chunks))
	
	// Convert Flesch score (0-100) to 0-1 scale
	return avgReadability / 100, issues, suggestions, nil
}

// calculateFleschReadingEase calculates Flesch Reading Ease score
func (qmc *QualityMetricsCalculator) calculateFleschReadingEase(text string) float64 {
	if text == "" {
		return 0
	}
	
	sentences := strings.Count(text, ".") + strings.Count(text, "!") + strings.Count(text, "?")
	if sentences == 0 {
		sentences = 1
	}
	
	words := len(strings.Fields(text))
	if words == 0 {
		return 0
	}
	
	// Simple syllable counting (approximation)
	syllables := 0
	for _, word := range strings.Fields(strings.ToLower(text)) {
		syllables += qmc.countSyllables(word)
	}
	
	if syllables == 0 {
		syllables = words // Fallback
	}
	
	// Flesch Reading Ease formula
	score := 206.835 - (1.015 * float64(words)/float64(sentences)) - (84.6 * float64(syllables)/float64(words))
	
	// Clamp to 0-100 range
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	
	return score
}

// countSyllables counts syllables in a word (approximation)
func (qmc *QualityMetricsCalculator) countSyllables(word string) int {
	word = strings.ToLower(word)
	vowels := "aeiouy"
	syllableCount := 0
	previousWasVowel := false
	
	for _, char := range word {
		isVowel := strings.ContainsRune(vowels, char)
		if isVowel && !previousWasVowel {
			syllableCount++
		}
		previousWasVowel = isVowel
	}
	
	// Handle silent 'e'
	if strings.HasSuffix(word, "e") && syllableCount > 1 {
		syllableCount--
	}
	
	// Every word has at least one syllable
	if syllableCount == 0 {
		syllableCount = 1
	}
	
	return syllableCount
}

// calculateSemanticIntegrity measures preservation of semantic meaning
func (qmc *QualityMetricsCalculator) calculateSemanticIntegrity(ctx context.Context, chunks []*Chunk) (float64, []QualityIssue, []QualitySuggestion, error) {
	if len(chunks) <= 1 {
		return 1.0, nil, nil, nil
	}
	
	var issues []QualityIssue
	var suggestions []QualitySuggestion
	
	// Check for semantic breaks between chunks
	if qmc.embeddingProvider != nil {
		integrity, err := qmc.calculateEmbeddingBasedIntegrity(ctx, chunks)
		if err == nil {
			if integrity < 0.6 {
				issues = append(issues, QualityIssue{
					Type:        IssueTypeContextLoss,
					Severity:    SeverityHigh,
					Description: fmt.Sprintf("Low semantic integrity (%.2f) between chunks", integrity),
					Impact:      (0.6 - integrity) * 0.8,
				})
				
				suggestions = append(suggestions, QualitySuggestion{
					Type:                SuggestionAdjustOverlap,
					Description:         "Increase overlap to preserve semantic continuity",
					ExpectedImprovement: 0.2,
				})
			}
			return integrity, issues, suggestions, nil
		}
	}
	
	// Fallback to text-based integrity
	integrity := qmc.calculateTextBasedIntegrity(chunks)
	return integrity, issues, suggestions, nil
}

// calculateEmbeddingBasedIntegrity calculates semantic integrity using embeddings
func (qmc *QualityMetricsCalculator) calculateEmbeddingBasedIntegrity(ctx context.Context, chunks []*Chunk) (float64, error) {
	similarities := make([]float64, len(chunks)-1)
	
	for i := 0; i < len(chunks)-1; i++ {
		emb1, err := qmc.embeddingProvider.GetEmbedding(ctx, chunks[i].Text)
		if err != nil {
			return 0, err
		}
		
		emb2, err := qmc.embeddingProvider.GetEmbedding(ctx, chunks[i+1].Text)
		if err != nil {
			return 0, err
		}
		
		similarity, err := qmc.similarityCalc.CosineSimilarity(emb1, emb2)
		if err != nil {
			return 0, err
		}
		
		similarities[i] = similarity
	}
	
	// Calculate average similarity
	total := 0.0
	for _, sim := range similarities {
		total += sim
	}
	
	return total / float64(len(similarities)), nil
}

// calculateTextBasedIntegrity calculates integrity using text analysis
func (qmc *QualityMetricsCalculator) calculateTextBasedIntegrity(chunks []*Chunk) float64 {
	if len(chunks) <= 1 {
		return 1.0
	}
	
	totalIntegrity := 0.0
	for i := 0; i < len(chunks)-1; i++ {
		integrity := qmc.calculateTextSimilarity(chunks[i].Text, chunks[i+1].Text)
		totalIntegrity += integrity
	}
	
	return totalIntegrity / float64(len(chunks)-1)
}

// calculateStructurePreservation measures how well document structure is preserved
func (qmc *QualityMetricsCalculator) calculateStructurePreservation(ctx context.Context, chunks []*Chunk, originalText string) (float64, []QualityIssue, []QualitySuggestion, error) {
	var issues []QualityIssue
	var suggestions []QualitySuggestion
	
	// Analyze structural elements preservation
	originalStructure := qmc.analyzeStructure(originalText)
	preservedStructure := qmc.analyzePreservedStructure(chunks)
	
	preservation := qmc.compareStructures(originalStructure, preservedStructure)
	
	if preservation < 0.7 {
		issues = append(issues, QualityIssue{
			Type:        IssueTypeStructuralBreak,
			Severity:    SeverityModerate,
			Description: fmt.Sprintf("Document structure poorly preserved (%.1f%%)", preservation*100),
			Impact:      (0.7 - preservation) * 0.6,
		})
		
		suggestions = append(suggestions, QualitySuggestion{
			Type:                SuggestionPreserveStructure,
			Description:         "Use structure-aware chunking to preserve document organization",
			ExpectedImprovement: 0.25,
		})
	}
	
	return preservation, issues, suggestions, nil
}

// DocumentStructure represents document structural elements
type DocumentStructure struct {
	Headers     int `json:"headers"`
	Lists       int `json:"lists"`
	Paragraphs  int `json:"paragraphs"`
	CodeBlocks  int `json:"code_blocks"`
	Tables      int `json:"tables"`
}

// analyzeStructure analyzes structural elements in text
func (qmc *QualityMetricsCalculator) analyzeStructure(text string) *DocumentStructure {
	return &DocumentStructure{
		Headers:    strings.Count(text, "#") + strings.Count(text, "<h"),
		Lists:      strings.Count(text, "- ") + strings.Count(text, "* ") + strings.Count(text, "<li>"),
		Paragraphs: strings.Count(text, "\n\n") + 1,
		CodeBlocks: strings.Count(text, "```") / 2,
		Tables:     strings.Count(text, "|") / 3, // Rough estimate
	}
}

// analyzePreservedStructure analyzes preserved structure in chunks
func (qmc *QualityMetricsCalculator) analyzePreservedStructure(chunks []*Chunk) *DocumentStructure {
	structure := &DocumentStructure{}
	
	for _, chunk := range chunks {
		chunkStructure := qmc.analyzeStructure(chunk.Text)
		structure.Headers += chunkStructure.Headers
		structure.Lists += chunkStructure.Lists
		structure.Paragraphs += chunkStructure.Paragraphs
		structure.CodeBlocks += chunkStructure.CodeBlocks
		structure.Tables += chunkStructure.Tables
	}
	
	return structure
}

// compareStructures compares original and preserved structures
func (qmc *QualityMetricsCalculator) compareStructures(original, preserved *DocumentStructure) float64 {
	totalOriginal := original.Headers + original.Lists + original.Paragraphs + original.CodeBlocks + original.Tables
	totalPreserved := preserved.Headers + preserved.Lists + preserved.Paragraphs + preserved.CodeBlocks + preserved.Tables
	
	if totalOriginal == 0 {
		return 1.0
	}
	
	return math.Min(1.0, float64(totalPreserved)/float64(totalOriginal))
}

// calculateOverlapOptimization measures optimization of chunk overlaps
func (qmc *QualityMetricsCalculator) calculateOverlapOptimization(ctx context.Context, chunks []*Chunk) (float64, []QualityIssue, []QualitySuggestion, error) {
	if len(chunks) <= 1 {
		return 1.0, nil, nil, nil
	}
	
	var issues []QualityIssue
	var suggestions []QualitySuggestion
	
	// Analyze overlap efficiency
	totalOverlap := 0
	redundantOverlap := 0
	
	for i := 0; i < len(chunks)-1; i++ {
		overlap := qmc.calculateOverlapBetweenChunks(chunks[i], chunks[i+1])
		totalOverlap += overlap
		
		// Check if overlap is meaningful or just redundant
		if qmc.isRedundantOverlap(chunks[i], chunks[i+1]) {
			redundantOverlap += overlap
		}
	}
	
	if totalOverlap == 0 {
		return 1.0, issues, suggestions, nil
	}
	
	optimization := 1.0 - (float64(redundantOverlap) / float64(totalOverlap))
	
	if optimization < 0.7 {
		issues = append(issues, QualityIssue{
			Type:        IssueTypeRedundancy,
			Severity:    SeverityModerate,
			Description: fmt.Sprintf("High redundant overlap (%.1f%%)", (1-optimization)*100),
			Impact:      (0.7 - optimization) * 0.4,
		})
		
		suggestions = append(suggestions, QualitySuggestion{
			Type:                SuggestionReduceRedundancy,
			Description:         "Optimize overlap to reduce redundancy while maintaining context",
			ExpectedImprovement: 0.15,
		})
	}
	
	return optimization, issues, suggestions, nil
}

// calculateOverlapBetweenChunks calculates overlap between two chunks
func (qmc *QualityMetricsCalculator) calculateOverlapBetweenChunks(chunk1, chunk2 *Chunk) int {
	// Simple approach: find common sentences or text segments
	sentences1 := chunk1.Sentences
	sentences2 := chunk2.Sentences
	
	overlap := 0
	for _, s1 := range sentences1 {
		for _, s2 := range sentences2 {
			if strings.TrimSpace(s1) == strings.TrimSpace(s2) {
				overlap++
				break
			}
		}
	}
	
	return overlap
}

// isRedundantOverlap checks if overlap between chunks is redundant
func (qmc *QualityMetricsCalculator) isRedundantOverlap(chunk1, chunk2 *Chunk) bool {
	// Simple heuristic: if chunks share more than 50% of sentences, it's likely redundant
	overlap := qmc.calculateOverlapBetweenChunks(chunk1, chunk2)
	minSentences := len(chunk1.Sentences)
	if len(chunk2.Sentences) < minSentences {
		minSentences = len(chunk2.Sentences)
	}
	
	if minSentences == 0 {
		return false
	}
	
	overlapRatio := float64(overlap) / float64(minSentences)
	return overlapRatio > 0.5
}

// setMetricScore sets the score for a specific metric in the assessment
func (qmc *QualityMetricsCalculator) setMetricScore(assessment *QualityAssessment, metric QualityMetricType, score float64) {
	switch metric {
	case MetricCoherence:
		assessment.Coherence = score
	case MetricCompleteness:
		assessment.Completeness = score
	case MetricRelevance:
		assessment.Relevance = score
	case MetricInformation:
		assessment.InformationDensity = score
	case MetricReadability:
		assessment.Readability = score
	case MetricSemanticIntegrity:
		assessment.SemanticIntegrity = score
	case MetricStructure:
		assessment.StructurePreservation = score
	case MetricOverlap:
		assessment.OverlapOptimization = score
	}
}

// calculateOverallScore calculates overall quality score from individual metrics
func (qmc *QualityMetricsCalculator) calculateOverallScore(assessment *QualityAssessment) float64 {
	// Weighted combination of metrics
	weights := map[QualityMetricType]float64{
		MetricCoherence:         0.20,
		MetricCompleteness:      0.15,
		MetricRelevance:         0.15,
		MetricInformation:       0.10,
		MetricReadability:       0.10,
		MetricSemanticIntegrity: 0.20,
		MetricStructure:         0.05,
		MetricOverlap:           0.05,
	}
	
	totalWeight := 0.0
	weightedSum := 0.0
	
	for _, metric := range qmc.config.MetricsToCollect {
		weight := weights[metric]
		score := qmc.getMetricScore(assessment, metric)
		
		weightedSum += weight * score
		totalWeight += weight
	}
	
	if totalWeight == 0 {
		return 0.0
	}
	
	return weightedSum / totalWeight
}

// getMetricScore retrieves the score for a specific metric
func (qmc *QualityMetricsCalculator) getMetricScore(assessment *QualityAssessment, metric QualityMetricType) float64 {
	switch metric {
	case MetricCoherence:
		return assessment.Coherence
	case MetricCompleteness:
		return assessment.Completeness
	case MetricRelevance:
		return assessment.Relevance
	case MetricInformation:
		return assessment.InformationDensity
	case MetricReadability:
		return assessment.Readability
	case MetricSemanticIntegrity:
		return assessment.SemanticIntegrity
	case MetricStructure:
		return assessment.StructurePreservation
	case MetricOverlap:
		return assessment.OverlapOptimization
	default:
		return 0.0
	}
}

// calculateTotalTokens calculates total tokens across all chunks
func (qmc *QualityMetricsCalculator) calculateTotalTokens(chunks []*Chunk) int {
	total := 0
	for _, chunk := range chunks {
		total += chunk.TokenCount
	}
	return total
}

// addToHistory adds assessment to metrics history
func (qmc *QualityMetricsCalculator) addToHistory(assessment *QualityAssessment) {
	qmc.metricsHistory = append(qmc.metricsHistory, assessment)
	
	// Limit history size
	maxHistory := 100
	if len(qmc.metricsHistory) > maxHistory {
		qmc.metricsHistory = qmc.metricsHistory[len(qmc.metricsHistory)-maxHistory:]
	}
}

// GetMetricsHistory returns the metrics history
func (qmc *QualityMetricsCalculator) GetMetricsHistory() []*QualityAssessment {
	return qmc.metricsHistory
}

// GetQualityTrends analyzes quality trends over time
func (qmc *QualityMetricsCalculator) GetQualityTrends() *QualityTrends {
	if len(qmc.metricsHistory) < 2 {
		return &QualityTrends{}
	}
	
	recent := qmc.metricsHistory[len(qmc.metricsHistory)-10:] // Last 10 assessments
	
	trends := &QualityTrends{
		Period:           fmt.Sprintf("Last %d assessments", len(recent)),
		AverageScore:     qmc.calculateAverageScore(recent),
		ScoreTrend:       qmc.calculateScoreTrend(recent),
		IssueFrequency:   qmc.calculateIssueFrequency(recent),
		ImprovementAreas: qmc.identifyImprovementAreas(recent),
	}
	
	return trends
}

// QualityTrends represents quality trends analysis
type QualityTrends struct {
	Period           string                  `json:"period"`
	AverageScore     float64                 `json:"average_score"`
	ScoreTrend       string                  `json:"score_trend"` // "improving", "declining", "stable"
	IssueFrequency   map[QualityIssueType]int `json:"issue_frequency"`
	ImprovementAreas []string                `json:"improvement_areas"`
}

// calculateAverageScore calculates average quality score
func (qmc *QualityMetricsCalculator) calculateAverageScore(assessments []*QualityAssessment) float64 {
	if len(assessments) == 0 {
		return 0.0
	}
	
	total := 0.0
	for _, assessment := range assessments {
		total += assessment.OverallScore
	}
	
	return total / float64(len(assessments))
}

// calculateScoreTrend determines if scores are improving, declining, or stable
func (qmc *QualityMetricsCalculator) calculateScoreTrend(assessments []*QualityAssessment) string {
	if len(assessments) < 3 {
		return "insufficient_data"
	}
	
	// Calculate trend using simple linear regression slope
	n := len(assessments)
	sumX, sumY, sumXY, sumXX := 0.0, 0.0, 0.0, 0.0
	
	for i, assessment := range assessments {
		x := float64(i)
		y := assessment.OverallScore
		
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}
	
	slope := (float64(n)*sumXY - sumX*sumY) / (float64(n)*sumXX - sumX*sumX)
	
	if slope > 0.01 {
		return "improving"
	} else if slope < -0.01 {
		return "declining"
	}
	return "stable"
}

// calculateIssueFrequency calculates frequency of different issue types
func (qmc *QualityMetricsCalculator) calculateIssueFrequency(assessments []*QualityAssessment) map[QualityIssueType]int {
	frequency := make(map[QualityIssueType]int)
	
	for _, assessment := range assessments {
		for _, issue := range assessment.Issues {
			frequency[issue.Type]++
		}
	}
	
	return frequency
}

// identifyImprovementAreas identifies areas that need improvement
func (qmc *QualityMetricsCalculator) identifyImprovementAreas(assessments []*QualityAssessment) []string {
	var areas []string
	
	// Calculate average scores for each metric
	avgCoherence := 0.0
	avgCompleteness := 0.0
	avgRelevance := 0.0
	
	for _, assessment := range assessments {
		avgCoherence += assessment.Coherence
		avgCompleteness += assessment.Completeness
		avgRelevance += assessment.Relevance
	}
	
	n := float64(len(assessments))
	avgCoherence /= n
	avgCompleteness /= n
	avgRelevance /= n
	
	// Identify areas below threshold
	threshold := 0.7
	
	if avgCoherence < threshold {
		areas = append(areas, "coherence")
	}
	if avgCompleteness < threshold {
		areas = append(areas, "completeness")
	}
	if avgRelevance < threshold {
		areas = append(areas, "relevance")
	}
	
	return areas
}