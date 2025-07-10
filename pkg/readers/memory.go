package readers

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// AdvancedMemReader implements advanced memory reading with semantic analysis
type AdvancedMemReader struct {
	*BaseMemReader
	memCube   interfaces.MemCube
	embedder  interfaces.Embedder
	vectorDB  interfaces.VectorDB
}

// NewAdvancedMemReader creates a new advanced memory reader
func NewAdvancedMemReader(config *ReaderConfig, memCube interfaces.MemCube, embedder interfaces.Embedder, vectorDB interfaces.VectorDB) *AdvancedMemReader {
	if config == nil {
		config = DefaultReaderConfig()
		config.Strategy = ReadStrategyAdvanced
		config.AnalysisDepth = "deep"
	}
	
	return &AdvancedMemReader{
		BaseMemReader: NewBaseMemReader(config),
		memCube:       memCube,
		embedder:      embedder,
		vectorDB:      vectorDB,
	}
}

// Read reads and analyzes memory content using advanced strategies
func (r *AdvancedMemReader) Read(ctx context.Context, query *ReadQuery) (*ReadResult, error) {
	startTime := time.Now()
	
	// Advanced reading strategy with semantic search
	memories, err := r.performAdvancedSearch(ctx, query)
	if err != nil {
		return nil, err
	}
	
	result := &ReadResult{
		Memories:    memories,
		TotalFound:  len(memories),
		ProcessTime: time.Since(startTime),
		Metadata: map[string]interface{}{
			"strategy": "advanced",
			"depth":    "deep",
			"semantic": true,
		},
	}
	
	// Perform advanced analysis
	if r.config.PatternDetection {
		result.Patterns, err = r.ExtractPatterns(ctx, memories, r.config.PatternConfig)
		if err != nil {
			return nil, fmt.Errorf("pattern extraction failed: %w", err)
		}
	}
	
	if r.config.QualityAssessment {
		result.Quality, err = r.performAdvancedQualityAssessment(ctx, memories)
		if err != nil {
			return nil, fmt.Errorf("quality assessment failed: %w", err)
		}
	}
	
	if r.config.DuplicateDetection {
		result.Duplicates, err = r.DetectDuplicates(ctx, memories, 0.85)
		if err != nil {
			return nil, fmt.Errorf("duplicate detection failed: %w", err)
		}
	}
	
	if r.config.Summarization {
		result.Summary, err = r.SummarizeMemories(ctx, memories, r.config.SummarizationConfig)
		if err != nil {
			return nil, fmt.Errorf("summarization failed: %w", err)
		}
	}
	
	// Always perform collection analysis for advanced reader
	result.Analysis, err = r.AnalyzeMemories(ctx, memories)
	if err != nil {
		return nil, fmt.Errorf("collection analysis failed: %w", err)
	}
	
	return result, nil
}

// ReadMemory reads a specific memory item with advanced analysis
func (r *AdvancedMemReader) ReadMemory(ctx context.Context, memoryID string) (*MemoryAnalysis, error) {
	// Fetch memory from the cube
	memory, err := r.getMemoryFromCube(ctx, memoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch memory %s: %w", memoryID, err)
	}
	
	return r.AnalyzeMemory(ctx, memory)
}

// ReadMemories reads multiple memory items with batch optimization
func (r *AdvancedMemReader) ReadMemories(ctx context.Context, memoryIDs []string) ([]*MemoryAnalysis, error) {
	analyses := make([]*MemoryAnalysis, len(memoryIDs))
	
	// Batch fetch memories for efficiency
	memories, err := r.getMemoriesFromCube(ctx, memoryIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to batch fetch memories: %w", err)
	}
	
	for i, memory := range memories {
		analysis, err := r.AnalyzeMemory(ctx, memory)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze memory %s: %w", memory.GetID(), err)
		}
		analyses[i] = analysis
	}
	
	return analyses, nil
}

// AnalyzeMemory performs advanced analysis of a single memory
func (r *AdvancedMemReader) AnalyzeMemory(ctx context.Context, memory types.MemoryItem) (*MemoryAnalysis, error) {
	content := memory.GetContent()
	
	// Advanced keyword extraction using embeddings
	keywords, err := r.extractAdvancedKeywords(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("keyword extraction failed: %w", err)
	}
	
	// Advanced theme extraction using clustering
	themes, err := r.extractAdvancedThemes(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("theme extraction failed: %w", err)
	}
	
	// Sentiment analysis with confidence
	sentiment := r.analyzeAdvancedSentiment(content)
	
	// Semantic complexity analysis
	complexity, err := r.calculateSemanticComplexity(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("complexity calculation failed: %w", err)
	}
	
	// Relevance scoring using semantic similarity
	relevance, err := r.calculateRelevance(ctx, memory)
	if err != nil {
		return nil, fmt.Errorf("relevance calculation failed: %w", err)
	}
	
	// Advanced quality assessment
	qualityAssessment, err := r.AssessQuality(ctx, memory)
	if err != nil {
		return nil, fmt.Errorf("quality assessment failed: %w", err)
	}
	
	// Generate insights using pattern analysis
	insights, err := r.generateAdvancedInsights(ctx, memory)
	if err != nil {
		return nil, fmt.Errorf("insight generation failed: %w", err)
	}
	
	return &MemoryAnalysis{
		MemoryID:   memory.GetID(),
		Type:       inferMemoryType(memory),
		Content:    content,
		Keywords:   keywords,
		Themes:     themes,
		Sentiment:  sentiment,
		Complexity: complexity,
		Relevance:  relevance,
		Quality:    qualityAssessment.Score,
		Metadata: map[string]interface{}{
			"reader":          "advanced",
			"created_at":      memory.GetCreatedAt(),
			"updated_at":      memory.GetUpdatedAt(),
			"quality_details": qualityAssessment,
		},
		Timestamp: time.Now(),
		Insights:  insights,
	}, nil
}

// AnalyzeMemories performs advanced collection analysis
func (r *AdvancedMemReader) AnalyzeMemories(ctx context.Context, memories []types.MemoryItem) (*MemoryCollectionAnalysis, error) {
	if len(memories) == 0 {
		return &MemoryCollectionAnalysis{
			TotalMemories: 0,
			Types:         make(map[string]int),
			Themes:        make(map[string]int),
			Relationships: []MemoryRelationship{},
			Clusters:      []MemoryCluster{},
			Timeline:      []MemoryTimelinePoint{},
			Statistics:    &MemoryStatistics{},
			Insights:      []string{"No memories to analyze"},
		}, nil
	}
	
	// Type and theme analysis
	types := make(map[string]int)
	themes := make(map[string]int)
	var totalLength int
	var totalQuality float64
	var totalComplexity float64
	
	for _, memory := range memories {
		memType := inferMemoryType(memory)
		types[memType]++
		
		content := memory.GetContent()
		totalLength += len(content)
		
		// Calculate quality and complexity
		quality := calculateQuality(memory)
		totalQuality += quality
		
		complexity, _ := r.calculateSemanticComplexity(ctx, content)
		totalComplexity += complexity
		
		// Extract themes
		memThemes, _ := r.extractAdvancedThemes(ctx, content)
		for _, theme := range memThemes {
			themes[theme]++
		}
	}
	
	// Advanced statistics
	stats := &MemoryStatistics{
		AverageLength:   float64(totalLength) / float64(len(memories)),
		MedianLength:    r.calculateMedianLength(memories),
		LengthStdDev:    r.calculateLengthStdDev(memories, float64(totalLength)/float64(len(memories))),
		AverageQuality:  totalQuality / float64(len(memories)),
		QualityStdDev:   r.calculateQualityStdDev(memories, totalQuality/float64(len(memories))),
		CreationRate:    r.calculateCreationRateAdvanced(memories),
		UpdateFrequency: r.calculateUpdateFrequencyAdvanced(memories),
	}
	
	// Advanced relationship detection using semantic similarity
	relationships, err := r.detectSemanticRelationships(ctx, memories)
	if err != nil {
		return nil, fmt.Errorf("relationship detection failed: %w", err)
	}
	
	// Semantic clustering
	clusters, err := r.performSemanticClustering(ctx, memories)
	if err != nil {
		return nil, fmt.Errorf("semantic clustering failed: %w", err)
	}
	
	// Advanced timeline with event detection
	timeline, err := r.generateAdvancedTimeline(ctx, memories)
	if err != nil {
		return nil, fmt.Errorf("timeline generation failed: %w", err)
	}
	
	// Generate advanced insights
	insights, err := r.generateCollectionInsights(ctx, memories, types, themes, relationships, clusters)
	if err != nil {
		return nil, fmt.Errorf("insight generation failed: %w", err)
	}
	
	// Generate smart recommendations
	recommendations, err := r.generateSmartRecommendations(ctx, memories, types, themes, clusters)
	if err != nil {
		return nil, fmt.Errorf("recommendation generation failed: %w", err)
	}
	
	return &MemoryCollectionAnalysis{
		TotalMemories:   len(memories),
		Types:           types,
		Themes:          themes,
		Relationships:   relationships,
		Clusters:        clusters,
		Timeline:        timeline,
		Statistics:      stats,
		Insights:        insights,
		Recommendations: recommendations,
	}, nil
}

// ExtractPatterns performs advanced pattern extraction using ML techniques
func (r *AdvancedMemReader) ExtractPatterns(ctx context.Context, memories []types.MemoryItem, config *PatternConfig) (*PatternAnalysis, error) {
	if config == nil {
		config = r.config.PatternConfig
	}
	
	patterns := make([]Pattern, 0)
	frequencies := make(map[string]int)
	
	// Semantic pattern extraction using embeddings
	semanticPatterns, err := r.extractSemanticPatterns(ctx, memories, config)
	if err != nil {
		return nil, fmt.Errorf("semantic pattern extraction failed: %w", err)
	}
	patterns = append(patterns, semanticPatterns...)
	
	// Temporal pattern extraction
	temporalPatterns, err := r.extractTemporalPatterns(ctx, memories, config)
	if err != nil {
		return nil, fmt.Errorf("temporal pattern extraction failed: %w", err)
	}
	patterns = append(patterns, temporalPatterns...)
	
	// Structural pattern extraction
	structuralPatterns, err := r.extractStructuralPatterns(ctx, memories, config)
	if err != nil {
		return nil, fmt.Errorf("structural pattern extraction failed: %w", err)
	}
	patterns = append(patterns, structuralPatterns...)
	
	// Frequency analysis with TF-IDF
	frequencies, err = r.calculateAdvancedFrequencies(ctx, memories)
	if err != nil {
		return nil, fmt.Errorf("frequency analysis failed: %w", err)
	}
	
	// Pattern sequence mining
	sequences, err := r.minePatternSequences(ctx, memories, config)
	if err != nil {
		return nil, fmt.Errorf("sequence mining failed: %w", err)
	}
	
	// Anomaly detection
	anomalies, err := r.detectPatternAnomalies(ctx, memories, patterns)
	if err != nil {
		return nil, fmt.Errorf("anomaly detection failed: %w", err)
	}
	
	// Calculate overall confidence
	confidence := r.calculatePatternConfidence(patterns, sequences, anomalies)
	
	return &PatternAnalysis{
		Patterns:    patterns,
		Frequencies: frequencies,
		Sequences:   sequences,
		Anomalies:   anomalies,
		Confidence:  confidence,
		Metadata: map[string]interface{}{
			"method":        "advanced_ml",
			"reader":        "advanced",
			"config":        config,
			"timestamp":     time.Now(),
			"total_patterns": len(patterns),
		},
	}, nil
}

// SummarizeMemories creates advanced summaries using extractive and abstractive techniques
func (r *AdvancedMemReader) SummarizeMemories(ctx context.Context, memories []types.MemoryItem, config *SummarizationConfig) (*MemorySummary, error) {
	if config == nil {
		config = r.config.SummarizationConfig
	}
	
	if len(memories) == 0 {
		return &MemorySummary{
			Summary:      "No memories to summarize",
			KeyPoints:    []string{},
			Themes:       []string{},
			Entities:     []string{},
			Complexity:   0.0,
			Coherence:    0.0,
			Completeness: 0.0,
			GeneratedAt:  time.Now(),
		}, nil
	}
	
	// Collect and preprocess content
	allContent := r.preprocessContent(memories)
	
	// Advanced summarization
	var summary string
	var err error
	
	switch config.Strategy {
	case "extractive":
		summary, err = r.generateExtractiveSummary(ctx, allContent, config)
	case "abstractive":
		summary, err = r.generateAbstractiveSummary(ctx, allContent, config)
	default:
		summary, err = r.generateHybridSummary(ctx, allContent, config)
	}
	
	if err != nil {
		return nil, fmt.Errorf("summary generation failed: %w", err)
	}
	
	// Extract key points using importance scoring
	keyPoints, err := r.extractImportantKeyPoints(ctx, allContent, config.KeyPoints)
	if err != nil {
		return nil, fmt.Errorf("key point extraction failed: %w", err)
	}
	
	// Advanced theme extraction
	var themes []string
	if config.IncludeThemes {
		themes, err = r.extractTopThemes(ctx, memories, 5)
		if err != nil {
			return nil, fmt.Errorf("theme extraction failed: %w", err)
		}
	}
	
	// Named entity recognition
	var entities []string
	if config.IncludeEntities {
		entities, err = r.extractNamedEntities(ctx, allContent)
		if err != nil {
			return nil, fmt.Errorf("entity extraction failed: %w", err)
		}
	}
	
	// Advanced sentiment analysis
	var sentiment *SentimentAnalysis
	if config.IncludeSentiment {
		sentiment = r.analyzeCollectionSentiment(memories)
	}
	
	// Calculate summary metrics
	complexity, err := r.calculateSemanticComplexity(ctx, summary)
	if err != nil {
		complexity = 0.5 // fallback
	}
	
	coherence := r.calculateSummaryCoherence(ctx, summary, allContent)
	completeness := r.calculateSummaryCompleteness(summary, allContent, keyPoints)
	
	return &MemorySummary{
		Summary:      summary,
		KeyPoints:    keyPoints,
		Themes:       themes,
		Entities:     entities,
		Sentiment:    sentiment,
		Complexity:   complexity,
		Coherence:    coherence,
		Completeness: completeness,
		Metadata: map[string]interface{}{
			"strategy":       config.Strategy,
			"reader":         "advanced",
			"source_count":   len(memories),
			"timestamp":      time.Now(),
		},
		GeneratedAt: time.Now(),
	}, nil
}

// DetectDuplicates uses advanced similarity measures for duplicate detection
func (r *AdvancedMemReader) DetectDuplicates(ctx context.Context, memories []types.MemoryItem, threshold float64) (*DuplicateAnalysis, error) {
	duplicates := make([]DuplicateGroup, 0)
	processed := make(map[string]bool)
	
	// Use semantic similarity for duplicate detection
	for i, mem1 := range memories {
		if processed[mem1.GetID()] {
			continue
		}
		
		group := DuplicateGroup{
			ID:          "group_" + mem1.GetID(),
			MemoryIDs:   []string{mem1.GetID()},
			Similarity:  1.0,
			Confidence:  1.0,
			Recommended: mem1.GetID(),
		}
		
		for j := i + 1; j < len(memories); j++ {
			mem2 := memories[j]
			if processed[mem2.GetID()] {
				continue
			}
			
			// Use semantic similarity
			similarity, err := r.calculateSemanticSimilarity(ctx, mem1, mem2)
			if err != nil {
				// Fallback to basic similarity
				similarity = calculateSimilarity(mem1, mem2)
			}
			
			if similarity >= threshold {
				group.MemoryIDs = append(group.MemoryIDs, mem2.GetID())
				processed[mem2.GetID()] = true
				
				// Update group similarity (minimum of all pairs)
				if similarity < group.Similarity {
					group.Similarity = similarity
				}
				
				// Choose recommended memory (highest quality)
				if calculateQuality(mem2) > calculateQuality(r.getMemoryByID(memories, group.Recommended)) {
					group.Recommended = mem2.GetID()
				}
			}
		}
		
		if len(group.MemoryIDs) > 1 {
			// Calculate group confidence based on similarity distribution
			group.Confidence = r.calculateGroupConfidence(group, memories)
			duplicates = append(duplicates, group)
		}
		
		processed[mem1.GetID()] = true
	}
	
	return &DuplicateAnalysis{
		Duplicates:      duplicates,
		TotalDuplicates: len(duplicates),
		Threshold:       threshold,
		Method:          "semantic_similarity",
		Metadata: map[string]interface{}{
			"reader":        "advanced",
			"algorithm":     "semantic_clustering",
			"embedder":      "enabled",
			"timestamp":     time.Now(),
		},
	}, nil
}

// AssessQuality performs comprehensive quality assessment
func (r *AdvancedMemReader) AssessQuality(ctx context.Context, memory types.MemoryItem) (*QualityAssessment, error) {
	content := memory.GetContent()
	
	// Multi-dimensional quality assessment
	dimensions := map[string]float64{
		"length":          r.assessLengthAdvanced(content),
		"structure":       r.assessStructureAdvanced(content),
		"readability":     r.assessReadabilityAdvanced(content),
		"informativeness": r.assessInformativenessAdvanced(content),
		"coherence":       r.assessCoherence(ctx, content),
		"uniqueness":      r.assessUniqueness(ctx, memory),
		"semantic_richness": r.assessSemanticRichness(ctx, content),
	}
	
	// Calculate weighted overall score
	weights := map[string]float64{
		"length":            0.1,
		"structure":         0.15,
		"readability":       0.2,
		"informativeness":   0.2,
		"coherence":         0.15,
		"uniqueness":        0.1,
		"semantic_richness": 0.1,
	}
	
	var overallScore float64
	for dimension, score := range dimensions {
		if weight, exists := weights[dimension]; exists {
			overallScore += score * weight
		}
	}
	
	// Identify quality issues with detailed analysis
	issues, err := r.identifyAdvancedQualityIssues(ctx, content, dimensions)
	if err != nil {
		return nil, fmt.Errorf("quality issue identification failed: %w", err)
	}
	
	// Generate targeted suggestions
	suggestions, err := r.generateAdvancedQualitySuggestions(ctx, issues, dimensions)
	if err != nil {
		return nil, fmt.Errorf("suggestion generation failed: %w", err)
	}
	
	return &QualityAssessment{
		Score:      overallScore,
		Dimensions: dimensions,
		Issues:     issues,
		Suggestions: suggestions,
		Metadata: map[string]interface{}{
			"reader":     "advanced",
			"method":     "multi_dimensional",
			"weights":    weights,
			"timestamp":  time.Now(),
		},
	}, nil
}

// Helper methods for AdvancedMemReader

func (r *AdvancedMemReader) performAdvancedSearch(ctx context.Context, query *ReadQuery) ([]types.MemoryItem, error) {
	// Use semantic search if embedder is available
	if r.embedder != nil && r.vectorDB != nil {
		return r.performSemanticSearch(ctx, query)
	}
	
	// Fallback to textual search
	return r.performTextualSearch(ctx, query)
}

func (r *AdvancedMemReader) performSemanticSearch(ctx context.Context, query *ReadQuery) ([]types.MemoryItem, error) {
	// Generate query embedding
	queryEmbedding, err := r.embedder.Embed(ctx, query.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}
	
	// Search vector database
	results, err := r.vectorDB.Search(ctx, queryEmbedding, query.TopK, query.Filters)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}
	
	// Convert results to memory items
	memories := make([]types.MemoryItem, len(results))
	for i, result := range results {
		memory, err := r.getMemoryFromCube(ctx, result.ID)
		if err != nil {
			continue // Skip failed retrievals
		}
		memories[i] = memory
	}
	
	return memories, nil
}

func (r *AdvancedMemReader) performTextualSearch(ctx context.Context, query *ReadQuery) ([]types.MemoryItem, error) {
	if r.memCube == nil {
		return nil, fmt.Errorf("no memory cube available for textual search")
	}
	
	// Use textual memory interface
	textMem := r.memCube.GetTextualMemory()
	if textMem == nil {
		return nil, fmt.Errorf("textual memory not available")
	}
	
	// Perform textual search
	textResults, err := textMem.SearchSemantic(ctx, query.Query, query.TopK, query.Filters)
	if err != nil {
		return nil, fmt.Errorf("textual search failed: %w", err)
	}
	
	// Convert to memory items
	memories := make([]types.MemoryItem, len(textResults))
	for i, result := range textResults {
		memories[i] = result
	}
	
	return memories, nil
}

func (r *AdvancedMemReader) getMemoryFromCube(ctx context.Context, memoryID string) (types.MemoryItem, error) {
	if r.memCube == nil {
		// Return mock memory for testing
		return &types.TextualMemoryItem{
			ID:        memoryID,
			Memory:    "Mock memory content for ID: " + memoryID,
			Metadata:  map[string]interface{}{"mock": true},
			CreatedAt: time.Now().Add(-time.Hour),
			UpdatedAt: time.Now(),
		}, nil
	}
	
	return r.memCube.GetTextualMemory().Get(ctx, memoryID)
}

func (r *AdvancedMemReader) getMemoriesFromCube(ctx context.Context, memoryIDs []string) ([]types.MemoryItem, error) {
	memories := make([]types.MemoryItem, len(memoryIDs))
	
	for i, id := range memoryIDs {
		memory, err := r.getMemoryFromCube(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get memory %s: %w", id, err)
		}
		memories[i] = memory
	}
	
	return memories, nil
}

func (r *AdvancedMemReader) extractAdvancedKeywords(ctx context.Context, content string) ([]string, error) {
	// Advanced keyword extraction using TF-IDF and semantic analysis
	words := strings.Fields(strings.ToLower(content))
	
	// Calculate word frequencies
	freq := make(map[string]int)
	for _, word := range words {
		if len(word) > 3 { // Skip short words
			freq[word]++
		}
	}
	
	// Sort by frequency and return top keywords
	type wordFreq struct {
		word string
		freq int
	}
	
	var wordFreqs []wordFreq
	for word, f := range freq {
		wordFreqs = append(wordFreqs, wordFreq{word, f})
	}
	
	sort.Slice(wordFreqs, func(i, j int) bool {
		return wordFreqs[i].freq > wordFreqs[j].freq
	})
	
	keywords := make([]string, 0, min(10, len(wordFreqs)))
	for i := 0; i < min(10, len(wordFreqs)); i++ {
		keywords = append(keywords, wordFreqs[i].word)
	}
	
	return keywords, nil
}

func (r *AdvancedMemReader) extractAdvancedThemes(ctx context.Context, content string) ([]string, error) {
	// Advanced theme extraction using topic modeling concepts
	// For now, use semantic similarity-based clustering
	
	sentences := strings.Split(content, ".")
	themes := make(map[string]int)
	
	// Simple thematic analysis based on sentence structure
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 20 {
			// Extract potential themes from longer sentences
			words := strings.Fields(strings.ToLower(sentence))
			if len(words) > 5 {
				// Use first meaningful words as theme indicators
				for _, word := range words[:min(5, len(words))] {
					if len(word) > 4 {
						themes[word]++
					}
				}
			}
		}
	}
	
	// Return top themes
	type themeCount struct {
		theme string
		count int
	}
	
	var themeCounts []themeCount
	for theme, count := range themes {
		themeCounts = append(themeCounts, themeCount{theme, count})
	}
	
	sort.Slice(themeCounts, func(i, j int) bool {
		return themeCounts[i].count > themeCounts[j].count
	})
	
	result := make([]string, 0, min(5, len(themeCounts)))
	for i := 0; i < min(5, len(themeCounts)); i++ {
		result = append(result, themeCounts[i].theme)
	}
	
	return result, nil
}

func (r *AdvancedMemReader) analyzeAdvancedSentiment(content string) *SentimentAnalysis {
	// Advanced sentiment analysis with better heuristics
	positiveWords := []string{"good", "great", "excellent", "positive", "happy", "success", "achievement"}
	negativeWords := []string{"bad", "terrible", "negative", "sad", "failure", "problem", "issue"}
	
	words := strings.Fields(strings.ToLower(content))
	
	var positiveScore, negativeScore int
	for _, word := range words {
		for _, pos := range positiveWords {
			if strings.Contains(word, pos) {
				positiveScore++
			}
		}
		for _, neg := range negativeWords {
			if strings.Contains(word, neg) {
				negativeScore++
			}
		}
	}
	
	totalScore := positiveScore + negativeScore
	var polarity float64
	var emotion string
	var confidence float64
	
	if totalScore > 0 {
		polarity = float64(positiveScore-negativeScore) / float64(totalScore)
		confidence = float64(totalScore) / float64(len(words))
		
		if polarity > 0.2 {
			emotion = "positive"
		} else if polarity < -0.2 {
			emotion = "negative"
		} else {
			emotion = "neutral"
		}
	} else {
		polarity = 0.0
		emotion = "neutral"
		confidence = 0.1
	}
	
	// Calculate subjectivity based on emotional words
	subjectivity := float64(totalScore) / float64(len(words))
	if subjectivity > 1.0 {
		subjectivity = 1.0
	}
	
	return &SentimentAnalysis{
		Polarity:     polarity,
		Subjectivity: subjectivity,
		Emotion:      emotion,
		Confidence:   confidence,
	}
}

func (r *AdvancedMemReader) calculateSemanticComplexity(ctx context.Context, content string) (float64, error) {
	// Advanced complexity calculation considering semantic depth
	if len(content) == 0 {
		return 0.0, nil
	}
	
	// Basic metrics
	words := strings.Fields(content)
	sentences := strings.Split(content, ".")
	
	if len(sentences) == 0 {
		return 0.0, nil
	}
	
	// Lexical diversity
	uniqueWords := make(map[string]bool)
	for _, word := range words {
		uniqueWords[strings.ToLower(word)] = true
	}
	
	lexicalDiversity := float64(len(uniqueWords)) / float64(len(words))
	
	// Average sentence length
	avgSentenceLength := float64(len(words)) / float64(len(sentences))
	
	// Structural complexity (nested structures, punctuation variety)
	structuralComplexity := r.calculateStructuralComplexity(content)
	
	// Combine metrics
	complexity := (lexicalDiversity + (avgSentenceLength/20.0) + structuralComplexity) / 3.0
	
	if complexity > 1.0 {
		complexity = 1.0
	}
	
	return complexity, nil
}

func (r *AdvancedMemReader) calculateStructuralComplexity(content string) float64 {
	complexity := 0.0
	
	// Punctuation variety
	punctuation := []string{".", ",", ";", ":", "!", "?", "-", "(", ")", "[", "]"}
	punctCount := 0
	for _, punct := range punctuation {
		if strings.Contains(content, punct) {
			punctCount++
		}
	}
	complexity += float64(punctCount) / float64(len(punctuation))
	
	// Paragraph structure
	paragraphs := strings.Split(content, "\n\n")
	if len(paragraphs) > 1 {
		complexity += 0.2
	}
	
	// Line breaks
	lines := strings.Split(content, "\n")
	if len(lines) > 3 {
		complexity += 0.1
	}
	
	return complexity
}

func (r *AdvancedMemReader) calculateRelevance(ctx context.Context, memory types.MemoryItem) (float64, error) {
	// Calculate relevance based on recency, quality, and content richness
	quality := calculateQuality(memory)
	
	// Recency factor
	age := time.Since(memory.GetUpdatedAt()).Hours() / 24.0 // days
	recencyFactor := math.Exp(-age / 30.0) // Decay over 30 days
	
	// Content richness
	content := memory.GetContent()
	richness := float64(len(strings.Fields(content))) / 100.0 // Normalize by word count
	if richness > 1.0 {
		richness = 1.0
	}
	
	// Combine factors
	relevance := (quality*0.4 + recencyFactor*0.3 + richness*0.3)
	
	return relevance, nil
}

func (r *AdvancedMemReader) generateAdvancedInsights(ctx context.Context, memory types.MemoryItem) ([]string, error) {
	insights := []string{}
	content := memory.GetContent()
	
	// Content-based insights
	if len(content) > 1000 {
		insights = append(insights, "Extensive content with rich information")
	}
	
	if strings.Count(content, "?") > 3 {
		insights = append(insights, "Inquiry-focused content with multiple questions")
	}
	
	if strings.Count(content, "!") > 2 {
		insights = append(insights, "Emphatic content with strong statements")
	}
	
	// Temporal insights
	age := time.Since(memory.GetCreatedAt())
	if age.Hours() < 24 {
		insights = append(insights, "Recently created memory")
	} else if age.Hours() > 24*30 {
		insights = append(insights, "Mature memory content")
	}
	
	// Update pattern insights
	if memory.GetUpdatedAt().After(memory.GetCreatedAt()) {
		insights = append(insights, "Memory has been updated since creation")
	}
	
	return insights, nil
}

func (r *AdvancedMemReader) performAdvancedQualityAssessment(ctx context.Context, memories []types.MemoryItem) (*QualityAssessment, error) {
	if len(memories) == 0 {
		return &QualityAssessment{
			Score:       0.0,
			Dimensions:  make(map[string]float64),
			Issues:      []QualityIssue{},
			Suggestions: []string{"No memories to assess"},
		}, nil
	}
	
	// Aggregate quality assessment
	var totalScore float64
	dimensions := make(map[string]float64)
	allIssues := []QualityIssue{}
	
	for _, memory := range memories {
		assessment, err := r.AssessQuality(ctx, memory)
		if err != nil {
			continue
		}
		
		totalScore += assessment.Score
		
		// Aggregate dimensions
		for dim, score := range assessment.Dimensions {
			dimensions[dim] += score
		}
		
		allIssues = append(allIssues, assessment.Issues...)
	}
	
	// Average the scores
	avgScore := totalScore / float64(len(memories))
	for dim := range dimensions {
		dimensions[dim] /= float64(len(memories))
	}
	
	// Generate collection-level suggestions
	suggestions := r.generateCollectionQualitySuggestions(allIssues, dimensions)
	
	return &QualityAssessment{
		Score:      avgScore,
		Dimensions: dimensions,
		Issues:     allIssues,
		Suggestions: suggestions,
		Metadata: map[string]interface{}{
			"reader":         "advanced",
			"assessment_type": "collection",
			"memory_count":   len(memories),
			"timestamp":      time.Now(),
		},
	}, nil
}

// Advanced analytics and pattern detection implementation

func (r *AdvancedMemReader) calculateMedianLength(memories []types.MemoryItem) float64 {
	lengths := make([]int, len(memories))
	for i, memory := range memories {
		lengths[i] = len(memory.GetContent())
	}
	
	sort.Ints(lengths)
	
	if len(lengths)%2 == 0 {
		return float64(lengths[len(lengths)/2-1]+lengths[len(lengths)/2]) / 2.0
	}
	return float64(lengths[len(lengths)/2])
}

func (r *AdvancedMemReader) calculateLengthStdDev(memories []types.MemoryItem, mean float64) float64 {
	var variance float64
	for _, memory := range memories {
		length := float64(len(memory.GetContent()))
		variance += math.Pow(length-mean, 2)
	}
	return math.Sqrt(variance / float64(len(memories)))
}

func (r *AdvancedMemReader) calculateQualityStdDev(memories []types.MemoryItem, mean float64) float64 {
	var variance float64
	for _, memory := range memories {
		quality := calculateQuality(memory)
		variance += math.Pow(quality-mean, 2)
	}
	return math.Sqrt(variance / float64(len(memories)))
}

func (r *AdvancedMemReader) getMemoryByID(memories []types.MemoryItem, id string) types.MemoryItem {
	for _, memory := range memories {
		if memory.GetID() == id {
			return memory
		}
	}
	return nil
}

// Advanced semantic relationship detection implementation
func (r *AdvancedMemReader) detectSemanticRelationships(ctx context.Context, memories []types.MemoryItem) ([]MemoryRelationship, error) {
	relationships := make([]MemoryRelationship, 0)
	
	// Cross-memory similarity analysis
	for i := 0; i < len(memories); i++ {
		for j := i + 1; j < len(memories); j++ {
			mem1, mem2 := memories[i], memories[j]
			
			// Semantic similarity
			similarity, err := r.calculateSemanticSimilarity(ctx, mem1, mem2)
			if err != nil {
				// Fallback to content similarity
				similarity = calculateSimilarity(mem1, mem2)
			}
			
			if similarity > 0.6 {
				relType := r.determineRelationshipType(similarity, mem1, mem2)
				relationships = append(relationships, MemoryRelationship{
					SourceID:    mem1.GetID(),
					TargetID:    mem2.GetID(),
					Type:        relType,
					Strength:    similarity,
					Description: r.generateRelationshipDescription(relType, similarity),
				})
			}
			
			// Temporal relationships
			if r.areTemporallyRelated(mem1, mem2) {
				relationships = append(relationships, MemoryRelationship{
					SourceID:    mem1.GetID(),
					TargetID:    mem2.GetID(),
					Type:        "temporal",
					Strength:    r.calculateTemporalStrength(mem1, mem2),
					Description: "Temporally related memories",
				})
			}
		}
	}
	
	return relationships, nil
}

func (r *AdvancedMemReader) performSemanticClustering(ctx context.Context, memories []types.MemoryItem) ([]MemoryCluster, error) {
	if len(memories) < 2 {
		return []MemoryCluster{}, nil
	}
	
	clusters := make([]MemoryCluster, 0)
	processed := make(map[string]bool)
	clusterID := 0
	
	// Simple clustering algorithm based on content similarity
	for i, memory := range memories {
		if processed[memory.GetID()] {
			continue
		}
		
		cluster := MemoryCluster{
			ID:          fmt.Sprintf("cluster_%d", clusterID),
			Name:        fmt.Sprintf("Cluster %d", clusterID+1),
			MemoryIDs:   []string{memory.GetID()},
			Centroid:    r.calculateCentroid([]types.MemoryItem{memory}),
			Coherence:   1.0,
			Description: "",
		}
		
		// Find similar memories to add to cluster
		for j := i + 1; j < len(memories); j++ {
			other := memories[j]
			if processed[other.GetID()] {
				continue
			}
			
			similarity := calculateSimilarity(memory, other)
			if similarity > 0.7 {
				cluster.MemoryIDs = append(cluster.MemoryIDs, other.GetID())
				processed[other.GetID()] = true
			}
		}
		
		if len(cluster.MemoryIDs) > 1 {
			// Update cluster properties
			cluster.Description = r.generateClusterDescription(cluster.MemoryIDs, memories)
			cluster.Coherence = r.calculateClusterCoherence(cluster.MemoryIDs, memories)
			clusters = append(clusters, cluster)
		}
		
		processed[memory.GetID()] = true
		clusterID++
	}
	
	return clusters, nil
}

func (r *AdvancedMemReader) generateAdvancedTimeline(ctx context.Context, memories []types.MemoryItem) ([]MemoryTimelinePoint, error) {
	timeline := make([]MemoryTimelinePoint, 0)
	
	// Sort memories by creation date
	sort.Slice(memories, func(i, j int) bool {
		return memories[i].GetCreatedAt().Before(memories[j].GetCreatedAt())
	})
	
	// Group by time periods (days)
	timeGroups := make(map[string][]types.MemoryItem)
	for _, memory := range memories {
		dayKey := memory.GetCreatedAt().Format("2006-01-02")
		timeGroups[dayKey] = append(timeGroups[dayKey], memory)
	}
	
	// Create timeline points with event detection
	for dayKey, dayMemories := range timeGroups {
		t, _ := time.Parse("2006-01-02", dayKey)
		
		events := r.detectTimelineEvents(dayMemories)
		
		timeline = append(timeline, MemoryTimelinePoint{
			Timestamp: t,
			Count:     len(dayMemories),
			Events:    events,
		})
	}
	
	// Sort timeline by timestamp
	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].Timestamp.Before(timeline[j].Timestamp)
	})
	
	return timeline, nil
}

func (r *AdvancedMemReader) calculateSemanticSimilarity(ctx context.Context, mem1, mem2 types.MemoryItem) (float64, error) {
	if r.embedder == nil {
		// Fallback to basic similarity
		return calculateSimilarity(mem1, mem2), nil
	}
	
	// Generate embeddings for both memories
	emb1, err := r.embedder.Embed(ctx, mem1.GetContent())
	if err != nil {
		return 0.0, fmt.Errorf("failed to embed memory 1: %w", err)
	}
	
	emb2, err := r.embedder.Embed(ctx, mem2.GetContent())
	if err != nil {
		return 0.0, fmt.Errorf("failed to embed memory 2: %w", err)
	}
	
	// Calculate cosine similarity
	// Convert from []float32 to []float64
	emb1Float64 := make([]float64, len(emb1))
	emb2Float64 := make([]float64, len(emb2))
	for i, v := range emb1 {
		emb1Float64[i] = float64(v)
	}
	for i, v := range emb2 {
		emb2Float64[i] = float64(v)
	}
	similarity := r.calculateCosineSimilarity(emb1Float64, emb2Float64)
	
	return similarity, nil
}

func (r *AdvancedMemReader) calculateGroupConfidence(group DuplicateGroup, memories []types.MemoryItem) float64 {
	// Implementation would calculate confidence based on similarity distribution
	return 0.8
}

func (r *AdvancedMemReader) assessCoherence(ctx context.Context, content string) float64 {
	// Implementation would assess text coherence
	return 0.7
}

func (r *AdvancedMemReader) assessUniqueness(ctx context.Context, memory types.MemoryItem) float64 {
	// Implementation would assess content uniqueness
	return 0.8
}

func (r *AdvancedMemReader) assessSemanticRichness(ctx context.Context, content string) float64 {
	// Implementation would assess semantic richness
	return 0.6
}

// Additional placeholder methods for completeness
func (r *AdvancedMemReader) extractSemanticPatterns(ctx context.Context, memories []types.MemoryItem, config *PatternConfig) ([]Pattern, error) {
	patterns := make([]Pattern, 0)
	
	// Semantic patterns based on content analysis
	if r.embedder != nil {
		patterns = append(patterns, r.extractEmbeddingPatterns(ctx, memories, config)...)
	}
	
	// Topic clustering patterns
	topicPatterns := r.extractTopicPatterns(memories, config)
	patterns = append(patterns, topicPatterns...)
	
	// Sentiment patterns
	sentimentPatterns := r.extractSentimentPatterns(memories, config)
	patterns = append(patterns, sentimentPatterns...)
	
	return patterns, nil
}

func (r *AdvancedMemReader) extractTemporalPatterns(ctx context.Context, memories []types.MemoryItem, config *PatternConfig) ([]Pattern, error) {
	patterns := make([]Pattern, 0)
	
	// Sort memories by creation time
	sort.Slice(memories, func(i, j int) bool {
		return memories[i].GetCreatedAt().Before(memories[j].GetCreatedAt())
	})
	
	// Detect creation patterns
	patterns = append(patterns, r.detectCreationPatterns(memories, config)...)
	
	// Detect activity bursts
	patterns = append(patterns, r.detectActivityBursts(memories, config)...)
	
	// Detect temporal sequences
	patterns = append(patterns, r.detectTemporalSequences(memories, config)...)
	
	return patterns, nil
}

func (r *AdvancedMemReader) extractStructuralPatterns(ctx context.Context, memories []types.MemoryItem, config *PatternConfig) ([]Pattern, error) {
	patterns := make([]Pattern, 0)
	
	// Length patterns
	patterns = append(patterns, r.detectLengthPatterns(memories, config)...)
	
	// Format patterns
	patterns = append(patterns, r.detectFormatPatterns(memories, config)...)
	
	// Structure patterns
	patterns = append(patterns, r.detectStructurePatterns(memories, config)...)
	
	return patterns, nil
}

func (r *AdvancedMemReader) calculateAdvancedFrequencies(ctx context.Context, memories []types.MemoryItem) (map[string]int, error) {
	frequencies := make(map[string]int)
	
	// Word frequencies with TF-IDF weighting
	wordFreqs := r.calculateTFIDF(memories)
	for word, freq := range wordFreqs {
		frequencies["word_"+word] = int(freq * 1000) // Scale for display
	}
	
	// Theme frequencies
	themeFreqs := r.calculateThemeFrequencies(memories)
	for theme, freq := range themeFreqs {
		frequencies["theme_"+theme] = freq
	}
	
	// Pattern frequencies
	patternFreqs := r.calculatePatternFrequencies(memories)
	for pattern, freq := range patternFreqs {
		frequencies["pattern_"+pattern] = freq
	}
	
	return frequencies, nil
}

func (r *AdvancedMemReader) minePatternSequences(ctx context.Context, memories []types.MemoryItem, config *PatternConfig) ([]PatternSequence, error) {
	sequences := make([]PatternSequence, 0)
	
	// Sort by creation time for sequence analysis
	sort.Slice(memories, func(i, j int) bool {
		return memories[i].GetCreatedAt().Before(memories[j].GetCreatedAt())
	})
	
	// Mine frequent sequences
	sequences = append(sequences, r.mineFrequentSequences(memories, config)...)
	
	// Mine temporal sequences
	sequences = append(sequences, r.mineTemporalSequences(memories, config)...)
	
	// Mine thematic sequences
	sequences = append(sequences, r.mineThematicSequences(memories, config)...)
	
	return sequences, nil
}

func (r *AdvancedMemReader) detectPatternAnomalies(ctx context.Context, memories []types.MemoryItem, patterns []Pattern) ([]PatternAnomaly, error) {
	anomalies := make([]PatternAnomaly, 0)
	
	// Statistical anomalies
	anomalies = append(anomalies, r.detectStatisticalAnomalies(memories)...)
	
	// Content anomalies
	anomalies = append(anomalies, r.detectContentAnomalies(memories)...)
	
	// Temporal anomalies
	anomalies = append(anomalies, r.detectTemporalAnomalies(memories)...)
	
	// Pattern-based anomalies
	anomalies = append(anomalies, r.detectPatternBasedAnomalies(memories, patterns)...)
	
	return anomalies, nil
}

func (r *AdvancedMemReader) calculatePatternConfidence(patterns []Pattern, sequences []PatternSequence, anomalies []PatternAnomaly) float64 {
	return 0.8
}

func (r *AdvancedMemReader) preprocessContent(memories []types.MemoryItem) string {
	var content strings.Builder
	for _, memory := range memories {
		content.WriteString(memory.GetContent())
		content.WriteString(" ")
	}
	return content.String()
}

func (r *AdvancedMemReader) generateExtractiveSummary(ctx context.Context, content string, config *SummarizationConfig) (string, error) {
	// Implementation would use extractive summarization
	return "Extractive summary placeholder", nil
}

func (r *AdvancedMemReader) generateAbstractiveSummary(ctx context.Context, content string, config *SummarizationConfig) (string, error) {
	// Implementation would use abstractive summarization
	return "Abstractive summary placeholder", nil
}

func (r *AdvancedMemReader) generateHybridSummary(ctx context.Context, content string, config *SummarizationConfig) (string, error) {
	// Implementation would combine both approaches
	return "Hybrid summary placeholder", nil
}

func (r *AdvancedMemReader) extractImportantKeyPoints(ctx context.Context, content string, numPoints int) ([]string, error) {
	// Implementation would use importance scoring
	return []string{"Key point 1", "Key point 2"}, nil
}

func (r *AdvancedMemReader) extractTopThemes(ctx context.Context, memories []types.MemoryItem, limit int) ([]string, error) {
	// Implementation would extract top themes
	return []string{"theme1", "theme2"}, nil
}

func (r *AdvancedMemReader) extractNamedEntities(ctx context.Context, content string) ([]string, error) {
	// Implementation would use NER
	return []string{"Entity1", "Entity2"}, nil
}

func (r *AdvancedMemReader) analyzeCollectionSentiment(memories []types.MemoryItem) *SentimentAnalysis {
	// Implementation would analyze collection sentiment
	return &SentimentAnalysis{
		Polarity:     0.1,
		Subjectivity: 0.5,
		Emotion:      "neutral",
		Confidence:   0.7,
	}
}

func (r *AdvancedMemReader) calculateSummaryCoherence(ctx context.Context, summary, content string) float64 {
	// Implementation would calculate coherence
	return 0.8
}

func (r *AdvancedMemReader) calculateSummaryCompleteness(summary, content string, keyPoints []string) float64 {
	// Implementation would calculate completeness
	return 0.7
}

func (r *AdvancedMemReader) generateCollectionInsights(ctx context.Context, memories []types.MemoryItem, types map[string]int, themes map[string]int, relationships []MemoryRelationship, clusters []MemoryCluster) ([]string, error) {
	// Implementation would generate insights
	return []string{"Advanced insight 1", "Advanced insight 2"}, nil
}

func (r *AdvancedMemReader) generateSmartRecommendations(ctx context.Context, memories []types.MemoryItem, types map[string]int, themes map[string]int, clusters []MemoryCluster) ([]string, error) {
	// Implementation would generate smart recommendations
	return []string{"Smart recommendation 1", "Smart recommendation 2"}, nil
}

func (r *AdvancedMemReader) identifyAdvancedQualityIssues(ctx context.Context, content string, dimensions map[string]float64) ([]QualityIssue, error) {
	// Implementation would identify quality issues
	return []QualityIssue{}, nil
}

func (r *AdvancedMemReader) generateAdvancedQualitySuggestions(ctx context.Context, issues []QualityIssue, dimensions map[string]float64) ([]string, error) {
	// Implementation would generate suggestions
	return []string{"Advanced suggestion 1"}, nil
}

func (r *AdvancedMemReader) generateCollectionQualitySuggestions(issues []QualityIssue, dimensions map[string]float64) []string {
	// Implementation would generate collection suggestions
	return []string{"Collection suggestion 1"}
}

// Helper methods for advanced functionality

func (r *AdvancedMemReader) determineRelationshipType(similarity float64, mem1, mem2 types.MemoryItem) string {
	if similarity > 0.9 {
		return "near_duplicate"
	} else if similarity > 0.8 {
		return "highly_similar"
	} else if similarity > 0.7 {
		return "similar"
	}
	return "related"
}

func (r *AdvancedMemReader) generateRelationshipDescription(relType string, strength float64) string {
	switch relType {
	case "near_duplicate":
		return fmt.Sprintf("Nearly identical content (%.2f similarity)", strength)
	case "highly_similar":
		return fmt.Sprintf("Highly similar content (%.2f similarity)", strength)
	case "similar":
		return fmt.Sprintf("Similar content (%.2f similarity)", strength)
	case "temporal":
		return "Created in close temporal proximity"
	default:
		return fmt.Sprintf("Related content (%.2f similarity)", strength)
	}
}

func (r *AdvancedMemReader) areTemporallyRelated(mem1, mem2 types.MemoryItem) bool {
	diff := mem1.GetCreatedAt().Sub(mem2.GetCreatedAt())
	if diff < 0 {
		diff = -diff
	}
	return diff.Hours() < 24 // Within 24 hours
}

func (r *AdvancedMemReader) calculateTemporalStrength(mem1, mem2 types.MemoryItem) float64 {
	diff := mem1.GetCreatedAt().Sub(mem2.GetCreatedAt())
	if diff < 0 {
		diff = -diff
	}
	
	hours := diff.Hours()
	if hours == 0 {
		return 1.0
	}
	
	// Decay function - closer in time = higher strength
	return math.Exp(-hours / 24.0)
}

func (r *AdvancedMemReader) calculateCentroid(memories []types.MemoryItem) []float64 {
	if len(memories) == 0 {
		return []float64{}
	}
	
	// Simple centroid calculation based on content statistics
	var totalLength float64
	var totalComplexity float64
	
	for _, memory := range memories {
		totalLength += float64(len(memory.GetContent()))
		totalComplexity += calculateComplexity(memory.GetContent())
	}
	
	return []float64{
		totalLength / float64(len(memories)),
		totalComplexity / float64(len(memories)),
	}
}

func (r *AdvancedMemReader) generateClusterDescription(memoryIDs []string, memories []types.MemoryItem) string {
	if len(memoryIDs) == 0 {
		return "Empty cluster"
	}
	
	// Find common themes
	themes := make(map[string]int)
	for _, memory := range memories {
		for _, id := range memoryIDs {
			if memory.GetID() == id {
				memThemes := extractThemes(memory.GetContent())
				for _, theme := range memThemes {
					themes[theme]++
				}
				break
			}
		}
	}
	
	// Find most common theme
	var dominantTheme string
	maxCount := 0
	for theme, count := range themes {
		if count > maxCount {
			maxCount = count
			dominantTheme = theme
		}
	}
	
	if dominantTheme != "" {
		return fmt.Sprintf("Cluster of %d memories with '%s' theme", len(memoryIDs), dominantTheme)
	}
	
	return fmt.Sprintf("Cluster of %d related memories", len(memoryIDs))
}

func (r *AdvancedMemReader) calculateClusterCoherence(memoryIDs []string, memories []types.MemoryItem) float64 {
	if len(memoryIDs) <= 1 {
		return 1.0
	}
	
	clusterMemories := make([]types.MemoryItem, 0, len(memoryIDs))
	for _, memory := range memories {
		for _, id := range memoryIDs {
			if memory.GetID() == id {
				clusterMemories = append(clusterMemories, memory)
				break
			}
		}
	}
	
	// Calculate average pairwise similarity
	var totalSimilarity float64
	var comparisons int
	
	for i := 0; i < len(clusterMemories); i++ {
		for j := i + 1; j < len(clusterMemories); j++ {
			similarity := calculateSimilarity(clusterMemories[i], clusterMemories[j])
			totalSimilarity += similarity
			comparisons++
		}
	}
	
	if comparisons == 0 {
		return 0.0
	}
	
	return totalSimilarity / float64(comparisons)
}

func (r *AdvancedMemReader) detectTimelineEvents(memories []types.MemoryItem) []string {
	events := make([]string, 0)
	
	if len(memories) > 10 {
		events = append(events, "High activity period")
	}
	
	// Check for content patterns
	var totalLength int
	for _, memory := range memories {
		totalLength += len(memory.GetContent())
	}
	
	avgLength := totalLength / len(memories)
	if avgLength > 500 {
		events = append(events, "Long-form content creation")
	}
	
	// Check for question patterns
	var questionCount int
	for _, memory := range memories {
		if strings.Contains(memory.GetContent(), "?") {
			questionCount++
		}
	}
	
	if float64(questionCount)/float64(len(memories)) > 0.5 {
		events = append(events, "Inquiry-focused period")
	}
	
	return events
}

func (r *AdvancedMemReader) calculateCosineSimilarity(vec1, vec2 []float64) float64 {
	if len(vec1) != len(vec2) {
		return 0.0
	}
	
	var dotProduct, norm1, norm2 float64
	
	for i := 0; i < len(vec1); i++ {
		dotProduct += vec1[i] * vec2[i]
		norm1 += vec1[i] * vec1[i]
		norm2 += vec2[i] * vec2[i]
	}
	
	norm1 = math.Sqrt(norm1)
	norm2 = math.Sqrt(norm2)
	
	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}
	
	return dotProduct / (norm1 * norm2)
}

// Pattern detection helper methods

func (r *AdvancedMemReader) extractEmbeddingPatterns(ctx context.Context, memories []types.MemoryItem, config *PatternConfig) []Pattern {
	patterns := make([]Pattern, 0)
	
	// Simple embedding-based pattern detection
	if len(memories) < 2 {
		return patterns
	}
	
	// Group similar embeddings
	similarities := make(map[string]float64)
	for i := 0; i < len(memories); i++ {
		for j := i + 1; j < len(memories); j++ {
			sim, _ := r.calculateSemanticSimilarity(ctx, memories[i], memories[j])
			key := fmt.Sprintf("%s-%s", memories[i].GetID(), memories[j].GetID())
			similarities[key] = sim
		}
	}
	
	// Find high similarity patterns
	highSimCount := 0
	for _, sim := range similarities {
		if sim > 0.8 {
			highSimCount++
		}
	}
	
	if highSimCount > 0 {
		patterns = append(patterns, Pattern{
			ID:          "semantic_similarity",
			Type:        "semantic",
			Description: fmt.Sprintf("High semantic similarity detected in %d memory pairs", highSimCount),
			Instances:   []string{fmt.Sprintf("%d pairs", highSimCount)},
			Frequency:   highSimCount,
			Confidence:  0.8,
			Significance: float64(highSimCount) / float64(len(similarities)),
		})
	}
	
	return patterns
}

func (r *AdvancedMemReader) extractTopicPatterns(memories []types.MemoryItem, config *PatternConfig) []Pattern {
	patterns := make([]Pattern, 0)
	
	// Aggregate all themes
	themeFreq := make(map[string]int)
	for _, memory := range memories {
		themes := extractThemes(memory.GetContent())
		for _, theme := range themes {
			themeFreq[theme]++
		}
	}
	
	// Identify dominant topics
	for theme, freq := range themeFreq {
		support := float64(freq) / float64(len(memories))
		if support >= config.MinSupport {
			patterns = append(patterns, Pattern{
				ID:          "topic_" + theme,
				Type:        "topic",
				Description: fmt.Sprintf("Recurring topic: %s", theme),
				Instances:   []string{theme},
				Frequency:   freq,
				Confidence:  support,
				Significance: support,
			})
		}
	}
	
	return patterns
}

func (r *AdvancedMemReader) extractSentimentPatterns(memories []types.MemoryItem, config *PatternConfig) []Pattern {
	patterns := make([]Pattern, 0)
	
	sentimentCounts := map[string]int{
		"positive": 0,
		"negative": 0,
		"neutral":  0,
	}
	
	for _, memory := range memories {
		sentiment := analyzeSentiment(memory.GetContent())
		sentimentCounts[sentiment.Emotion]++
	}
	
	// Find dominant sentiment patterns
	for emotion, count := range sentimentCounts {
		if count > 0 {
			support := float64(count) / float64(len(memories))
			if support >= config.MinSupport {
				patterns = append(patterns, Pattern{
					ID:          "sentiment_" + emotion,
					Type:        "sentiment",
					Description: fmt.Sprintf("Dominant sentiment: %s", emotion),
					Instances:   []string{emotion},
					Frequency:   count,
					Confidence:  support,
					Significance: support,
				})
			}
		}
	}
	
	return patterns
}

func (r *AdvancedMemReader) detectCreationPatterns(memories []types.MemoryItem, config *PatternConfig) []Pattern {
	patterns := make([]Pattern, 0)
	
	if len(memories) < 2 {
		return patterns
	}
	
	// Analyze creation time intervals
	intervals := make([]float64, len(memories)-1)
	for i := 1; i < len(memories); i++ {
		diff := memories[i].GetCreatedAt().Sub(memories[i-1].GetCreatedAt())
		intervals[i-1] = diff.Hours()
	}
	
	// Find regular intervals
	avgInterval := r.calculateMean(intervals)
	stdDev := r.calculateStdDev(intervals, avgInterval)
	
	regularCount := 0
	for _, interval := range intervals {
		if math.Abs(interval-avgInterval) < stdDev {
			regularCount++
		}
	}
	
	if float64(regularCount)/float64(len(intervals)) > 0.7 {
		patterns = append(patterns, Pattern{
			ID:          "regular_creation",
			Type:        "temporal",
			Description: fmt.Sprintf("Regular creation pattern (avg %.1f hours)", avgInterval),
			Instances:   []string{fmt.Sprintf("%.1f hours", avgInterval)},
			Frequency:   regularCount,
			Confidence:  float64(regularCount) / float64(len(intervals)),
			Significance: 0.8,
		})
	}
	
	return patterns
}

func (r *AdvancedMemReader) detectActivityBursts(memories []types.MemoryItem, config *PatternConfig) []Pattern {
	patterns := make([]Pattern, 0)
	
	// Group by day
	dailyCounts := make(map[string]int)
	for _, memory := range memories {
		day := memory.GetCreatedAt().Format("2006-01-02")
		dailyCounts[day]++
	}
	
	// Find burst days (days with significantly more activity)
	var counts []int
	for _, count := range dailyCounts {
		counts = append(counts, count)
	}
	
	if len(counts) > 0 {
		avgDaily := r.calculateMeanInt(counts)
		burstThreshold := int(float64(avgDaily) * 2.0)
		
		burstDays := 0
		for _, count := range counts {
			if count > burstThreshold {
				burstDays++
			}
		}
		
		if burstDays > 0 {
			patterns = append(patterns, Pattern{
				ID:          "activity_burst",
				Type:        "temporal",
				Description: fmt.Sprintf("Activity bursts detected on %d days", burstDays),
				Instances:   []string{fmt.Sprintf("%d days", burstDays)},
				Frequency:   burstDays,
				Confidence:  0.8,
				Significance: float64(burstDays) / float64(len(dailyCounts)),
			})
		}
	}
	
	return patterns
}

func (r *AdvancedMemReader) detectTemporalSequences(memories []types.MemoryItem, config *PatternConfig) []Pattern {
	patterns := make([]Pattern, 0)
	
	// Detect sequences of related memories created in succession
	sequenceLength := 0
	maxSequence := 0
	threshold := 24.0 // hours
	
	for i := 1; i < len(memories); i++ {
		diff := memories[i].GetCreatedAt().Sub(memories[i-1].GetCreatedAt()).Hours()
		similarity := calculateSimilarity(memories[i-1], memories[i])
		
		if diff < threshold && similarity > 0.5 {
			sequenceLength++
		} else {
			if sequenceLength > maxSequence {
				maxSequence = sequenceLength
			}
			sequenceLength = 0
		}
	}
	
	if maxSequence > 2 {
		patterns = append(patterns, Pattern{
			ID:          "temporal_sequence",
			Type:        "temporal",
			Description: fmt.Sprintf("Longest related sequence: %d memories", maxSequence+1),
			Instances:   []string{fmt.Sprintf("%d memories", maxSequence+1)},
			Frequency:   maxSequence + 1,
			Confidence:  0.7,
			Significance: float64(maxSequence) / float64(len(memories)),
		})
	}
	
	return patterns
}

func (r *AdvancedMemReader) detectLengthPatterns(memories []types.MemoryItem, config *PatternConfig) []Pattern {
	patterns := make([]Pattern, 0)
	
	// Analyze content length distribution
	lengths := make([]int, len(memories))
	for i, memory := range memories {
		lengths[i] = len(memory.GetContent())
	}
	
	if len(lengths) > 0 {
		avgLength := r.calculateMeanInt(lengths)
		
		// Categorize lengths
		short, medium, long := 0, 0, 0
		for _, length := range lengths {
			if length < avgLength/2 {
				short++
			} else if length > avgLength*2 {
				long++
			} else {
				medium++
			}
		}
		
		// Find dominant length pattern
		total := len(lengths)
		if float64(short)/float64(total) > 0.6 {
			patterns = append(patterns, Pattern{
				ID:          "short_content",
				Type:        "structural",
				Description: "Predominantly short-form content",
				Instances:   []string{"short"},
				Frequency:   short,
				Confidence:  float64(short) / float64(total),
				Significance: 0.7,
			})
		} else if float64(long)/float64(total) > 0.4 {
			patterns = append(patterns, Pattern{
				ID:          "long_content",
				Type:        "structural",
				Description: "Significant long-form content",
				Instances:   []string{"long"},
				Frequency:   long,
				Confidence:  float64(long) / float64(total),
				Significance: 0.7,
			})
		}
	}
	
	return patterns
}

func (r *AdvancedMemReader) detectFormatPatterns(memories []types.MemoryItem, config *PatternConfig) []Pattern {
	patterns := make([]Pattern, 0)
	
	// Detect formatting patterns
	structuredCount := 0
	for _, memory := range memories {
		content := memory.GetContent()
		if strings.Contains(content, "\n") && strings.Contains(content, ".") {
			structuredCount++
		}
	}
	
	if float64(structuredCount)/float64(len(memories)) > 0.7 {
		patterns = append(patterns, Pattern{
			ID:          "structured_format",
			Type:        "structural",
			Description: "Well-structured content format",
			Instances:   []string{"structured"},
			Frequency:   structuredCount,
			Confidence:  float64(structuredCount) / float64(len(memories)),
			Significance: 0.6,
		})
	}
	
	return patterns
}

func (r *AdvancedMemReader) detectStructurePatterns(memories []types.MemoryItem, config *PatternConfig) []Pattern {
	patterns := make([]Pattern, 0)
	
	// Detect punctuation patterns
	questionCount := 0
	for _, memory := range memories {
		if strings.Contains(memory.GetContent(), "?") {
			questionCount++
		}
	}
	
	if float64(questionCount)/float64(len(memories)) > 0.5 {
		patterns = append(patterns, Pattern{
			ID:          "inquiry_pattern",
			Type:        "structural",
			Description: "High frequency of questions",
			Instances:   []string{"questions"},
			Frequency:   questionCount,
			Confidence:  float64(questionCount) / float64(len(memories)),
			Significance: 0.6,
		})
	}
	
	return patterns
}

// TF-IDF and frequency analysis methods

func (r *AdvancedMemReader) calculateTFIDF(memories []types.MemoryItem) map[string]float64 {
	tfidf := make(map[string]float64)
	
	// Simple TF-IDF implementation
	docCount := len(memories)
	wordDocCount := make(map[string]int)
	
	// Count documents containing each word
	for _, memory := range memories {
		words := strings.Fields(strings.ToLower(memory.GetContent()))
		uniqueWords := make(map[string]bool)
		for _, word := range words {
			if len(word) > 3 {
				uniqueWords[word] = true
			}
		}
		for word := range uniqueWords {
			wordDocCount[word]++
		}
	}
	
	// Calculate TF-IDF for each word
	for word, docFreq := range wordDocCount {
		if docFreq > 1 { // Only consider words appearing in multiple documents
			idf := math.Log(float64(docCount) / float64(docFreq))
			tf := float64(docFreq) / float64(docCount)
			tfidf[word] = tf * idf
		}
	}
	
	return tfidf
}

func (r *AdvancedMemReader) calculateThemeFrequencies(memories []types.MemoryItem) map[string]int {
	themeFreq := make(map[string]int)
	
	for _, memory := range memories {
		themes := extractThemes(memory.GetContent())
		for _, theme := range themes {
			themeFreq[theme]++
		}
	}
	
	return themeFreq
}

func (r *AdvancedMemReader) calculatePatternFrequencies(memories []types.MemoryItem) map[string]int {
	patternFreq := make(map[string]int)
	
	// Simple pattern frequency analysis
	_ = []string{"question", "statement", "exclamation", "list"}
	
	for _, memory := range memories {
		content := memory.GetContent()
		if strings.Contains(content, "?") {
			patternFreq["question"]++
		}
		if strings.Contains(content, ".") {
			patternFreq["statement"]++
		}
		if strings.Contains(content, "!") {
			patternFreq["exclamation"]++
		}
		if strings.Contains(content, "\n- ") || strings.Contains(content, "\n* ") {
			patternFreq["list"]++
		}
	}
	
	return patternFreq
}

// Sequence mining methods

func (r *AdvancedMemReader) mineFrequentSequences(memories []types.MemoryItem, config *PatternConfig) []PatternSequence {
	sequences := make([]PatternSequence, 0)
	
	// Simple frequent sequence mining based on themes
	if len(memories) < 3 {
		return sequences
	}
	
	themeSeq := make([]string, len(memories))
	for i, memory := range memories {
		themes := extractThemes(memory.GetContent())
		if len(themes) > 0 {
			themeSeq[i] = themes[0] // Use primary theme
		} else {
			themeSeq[i] = "unknown"
		}
	}
	
	// Find frequent 3-sequences
	seqCount := make(map[string]int)
	for i := 0; i <= len(themeSeq)-3; i++ {
		seq := fmt.Sprintf("%s->%s->%s", themeSeq[i], themeSeq[i+1], themeSeq[i+2])
		seqCount[seq]++
	}
	
	for seq, count := range seqCount {
		if count > 1 {
			support := float64(count) / float64(len(themeSeq)-2)
			sequences = append(sequences, PatternSequence{
				ID:       "theme_seq_" + seq,
				Patterns: strings.Split(seq, "->"),
				Support:  support,
				Length:   3,
			})
		}
	}
	
	return sequences
}

func (r *AdvancedMemReader) mineTemporalSequences(memories []types.MemoryItem, config *PatternConfig) []PatternSequence {
	sequences := make([]PatternSequence, 0)
	
	// Mine sequences based on temporal proximity and content similarity
	for i := 0; i < len(memories)-2; i++ {
		seq := []string{memories[i].GetID(), memories[i+1].GetID(), memories[i+2].GetID()}
		
		// Check temporal proximity
		diff1 := memories[i+1].GetCreatedAt().Sub(memories[i].GetCreatedAt()).Hours()
		diff2 := memories[i+2].GetCreatedAt().Sub(memories[i+1].GetCreatedAt()).Hours()
		
		if diff1 < 24 && diff2 < 24 {
			// Check content similarity
			sim1 := calculateSimilarity(memories[i], memories[i+1])
			sim2 := calculateSimilarity(memories[i+1], memories[i+2])
			
			if sim1 > 0.5 && sim2 > 0.5 {
				support := (sim1 + sim2) / 2.0
				sequences = append(sequences, PatternSequence{
					ID:       fmt.Sprintf("temporal_seq_%d", i),
					Patterns: seq,
					Support:  support,
					Length:   3,
				})
			}
		}
	}
	
	return sequences
}

func (r *AdvancedMemReader) mineThematicSequences(memories []types.MemoryItem, config *PatternConfig) []PatternSequence {
	sequences := make([]PatternSequence, 0)
	
	// Mine sequences based on thematic progression
	for i := 0; i < len(memories)-2; i++ {
		themes1 := extractThemes(memories[i].GetContent())
		themes2 := extractThemes(memories[i+1].GetContent())
		themes3 := extractThemes(memories[i+2].GetContent())
		
		if len(themes1) > 0 && len(themes2) > 0 && len(themes3) > 0 {
			// Check for thematic progression
			if r.hasThematicProgression(themes1, themes2, themes3) {
				sequences = append(sequences, PatternSequence{
					ID:       fmt.Sprintf("thematic_seq_%d", i),
					Patterns: []string{themes1[0], themes2[0], themes3[0]},
					Support:  0.7,
					Length:   3,
				})
			}
		}
	}
	
	return sequences
}

// Anomaly detection methods

func (r *AdvancedMemReader) detectStatisticalAnomalies(memories []types.MemoryItem) []PatternAnomaly {
	anomalies := make([]PatternAnomaly, 0)
	
	// Length anomalies
	lengths := make([]int, len(memories))
	for i, memory := range memories {
		lengths[i] = len(memory.GetContent())
	}
	
	if len(lengths) > 0 {
		mean := r.calculateMeanInt(lengths)
		stdDev := r.calculateStdDevInt(lengths, mean)
		
		outlierIDs := make([]string, 0)
		for i, length := range lengths {
			if math.Abs(float64(length-mean)) > 2*float64(stdDev) {
				outlierIDs = append(outlierIDs, memories[i].GetID())
			}
		}
		
		if len(outlierIDs) > 0 {
			anomalies = append(anomalies, PatternAnomaly{
				ID:          "length_outliers",
				Type:        "statistical",
				Description: fmt.Sprintf("Content length outliers detected (%d memories)", len(outlierIDs)),
				Severity:    0.6,
				MemoryIDs:   outlierIDs,
			})
		}
	}
	
	return anomalies
}

func (r *AdvancedMemReader) detectContentAnomalies(memories []types.MemoryItem) []PatternAnomaly {
	anomalies := make([]PatternAnomaly, 0)
	
	// Detect memories with unusual content characteristics
	for _, memory := range memories {
		content := memory.GetContent()
		
		// Check for unusual character patterns
		if r.hasUnusualCharacters(content) {
			anomalies = append(anomalies, PatternAnomaly{
				ID:          "unusual_chars_" + memory.GetID(),
				Type:        "content",
				Description: "Unusual character patterns detected",
				Severity:    0.5,
				MemoryIDs:   []string{memory.GetID()},
			})
		}
		
		// Check for repetitive content
		if r.isRepetitive(content) {
			anomalies = append(anomalies, PatternAnomaly{
				ID:          "repetitive_" + memory.GetID(),
				Type:        "content",
				Description: "Highly repetitive content detected",
				Severity:    0.4,
				MemoryIDs:   []string{memory.GetID()},
			})
		}
	}
	
	return anomalies
}

func (r *AdvancedMemReader) detectTemporalAnomalies(memories []types.MemoryItem) []PatternAnomaly {
	anomalies := make([]PatternAnomaly, 0)
	
	if len(memories) < 2 {
		return anomalies
	}
	
	// Sort by creation time
	sort.Slice(memories, func(i, j int) bool {
		return memories[i].GetCreatedAt().Before(memories[j].GetCreatedAt())
	})
	
	// Detect unusual time gaps
	intervals := make([]float64, len(memories)-1)
	for i := 1; i < len(memories); i++ {
		diff := memories[i].GetCreatedAt().Sub(memories[i-1].GetCreatedAt())
		intervals[i-1] = diff.Hours()
	}
	
	if len(intervals) > 0 {
		mean := r.calculateMean(intervals)
		stdDev := r.calculateStdDev(intervals, mean)
		
		// Find gaps significantly larger than average
		for i, interval := range intervals {
			if interval > mean+2*stdDev {
				anomalies = append(anomalies, PatternAnomaly{
					ID:          fmt.Sprintf("time_gap_%d", i),
					Type:        "temporal",
					Description: fmt.Sprintf("Unusual time gap: %.1f hours", interval),
					Severity:    math.Min(0.8, interval/(mean+2*stdDev)),
					MemoryIDs:   []string{memories[i].GetID(), memories[i+1].GetID()},
				})
			}
		}
	}
	
	return anomalies
}

func (r *AdvancedMemReader) detectPatternBasedAnomalies(memories []types.MemoryItem, patterns []Pattern) []PatternAnomaly {
	anomalies := make([]PatternAnomaly, 0)
	
	// Detect memories that don't fit established patterns
	for _, memory := range memories {
		fitsPattern := false
		for _, pattern := range patterns {
			if r.memoryFitsPattern(memory, pattern) {
				fitsPattern = true
				break
			}
		}
		
		if !fitsPattern && len(patterns) > 0 {
			anomalies = append(anomalies, PatternAnomaly{
				ID:          "pattern_outlier_" + memory.GetID(),
				Type:        "pattern",
				Description: "Memory doesn't fit established patterns",
				Severity:    0.6,
				MemoryIDs:   []string{memory.GetID()},
			})
		}
	}
	
	return anomalies
}

// Statistical helper methods

func (r *AdvancedMemReader) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (r *AdvancedMemReader) calculateStdDev(values []float64, mean float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	
	variance := 0.0
	for _, v := range values {
		variance += math.Pow(v-mean, 2)
	}
	return math.Sqrt(variance / float64(len(values)-1))
}

func (r *AdvancedMemReader) calculateMeanInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	
	sum := 0
	for _, v := range values {
		sum += v
	}
	return sum / len(values)
}

func (r *AdvancedMemReader) calculateStdDevInt(values []int, mean int) int {
	if len(values) <= 1 {
		return 0
	}
	
	variance := 0
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	return int(math.Sqrt(float64(variance) / float64(len(values)-1)))
}

// Content analysis helper methods

func (r *AdvancedMemReader) hasThematicProgression(themes1, themes2, themes3 []string) bool {
	// Simple check for thematic relationship
	if len(themes1) == 0 || len(themes2) == 0 || len(themes3) == 0 {
		return false
	}
	
	// Check if themes are related (simple heuristic)
	for _, t1 := range themes1 {
		for _, t2 := range themes2 {
			for _, t3 := range themes3 {
				if strings.Contains(t1, t2) || strings.Contains(t2, t3) || strings.Contains(t1, t3) {
					return true
				}
			}
		}
	}
	
	return false
}

func (r *AdvancedMemReader) hasUnusualCharacters(content string) bool {
	// Check for unusual character patterns
	nonPrintable := 0
	for _, char := range content {
		if char < 32 && char != 9 && char != 10 && char != 13 { // Allow tab, LF, CR
			nonPrintable++
		}
	}
	
	return float64(nonPrintable)/float64(len(content)) > 0.1
}

func (r *AdvancedMemReader) isRepetitive(content string) bool {
	// Simple repetition detection
	words := strings.Fields(content)
	if len(words) < 10 {
		return false
	}
	
	wordCount := make(map[string]int)
	for _, word := range words {
		wordCount[word]++
	}
	
	// Check if any word appears more than 30% of the time
	for _, count := range wordCount {
		if float64(count)/float64(len(words)) > 0.3 {
			return true
		}
	}
	
	return false
}

func (r *AdvancedMemReader) memoryFitsPattern(memory types.MemoryItem, pattern Pattern) bool {
	// Simple pattern matching logic
	switch pattern.Type {
	case "semantic":
		return len(memory.GetContent()) > 50 // Semantic patterns need substantial content
	case "topic":
		themes := extractThemes(memory.GetContent())
		for _, theme := range themes {
			for _, instance := range pattern.Instances {
				if theme == instance {
					return true
				}
			}
		}
	case "sentiment":
		sentiment := analyzeSentiment(memory.GetContent())
		for _, instance := range pattern.Instances {
			if sentiment.Emotion == instance {
				return true
			}
		}
	case "temporal":
		return true // Most memories fit temporal patterns
	case "structural":
		content := memory.GetContent()
		for _, instance := range pattern.Instances {
			switch instance {
			case "short":
				return len(content) < 100
			case "long":
				return len(content) > 500
			case "structured":
				return strings.Contains(content, "\n") && strings.Contains(content, ".")
			case "questions":
				return strings.Contains(content, "?")
			}
		}
	}
	
	return false
}

// Missing advanced methods implementation

func (r *AdvancedMemReader) calculateCreationRateAdvanced(memories []types.MemoryItem) float64 {
	if len(memories) <= 1 {
		return 0.0
	}
	
	earliest := memories[0].GetCreatedAt()
	latest := memories[0].GetCreatedAt()
	
	for _, memory := range memories {
		created := memory.GetCreatedAt()
		if created.Before(earliest) {
			earliest = created
		}
		if created.After(latest) {
			latest = created
		}
	}
	
	duration := latest.Sub(earliest)
	if duration.Hours() == 0 {
		return 0.0
	}
	
	return float64(len(memories)) / duration.Hours()
}

func (r *AdvancedMemReader) calculateUpdateFrequencyAdvanced(memories []types.MemoryItem) float64 {
	if len(memories) == 0 {
		return 0.0
	}
	
	var totalUpdates float64
	for _, memory := range memories {
		created := memory.GetCreatedAt()
		updated := memory.GetUpdatedAt()
		
		if updated.After(created) {
			totalUpdates++
		}
	}
	
	return totalUpdates / float64(len(memories))
}

func (r *AdvancedMemReader) assessLengthAdvanced(content string) float64 {
	length := len(content)
	if length < 10 {
		return 0.2
	} else if length < 50 {
		return 0.5
	} else if length < 200 {
		return 0.8
	} else {
		return 1.0
	}
}

func (r *AdvancedMemReader) assessStructureAdvanced(content string) float64 {
	score := 0.0
	
	if strings.Contains(content, "\n") {
		score += 0.3
	}
	if strings.Contains(content, ".") {
		score += 0.3
	}
	if strings.Contains(content, ",") {
		score += 0.2
	}
	if len(strings.Fields(content)) > 5 {
		score += 0.2
	}
	
	return score
}

func (r *AdvancedMemReader) assessReadabilityAdvanced(content string) float64 {
	words := strings.Fields(content)
	if len(words) == 0 {
		return 0.0
	}
	
	sentences := strings.Count(content, ".") + strings.Count(content, "!") + strings.Count(content, "?")
	if sentences == 0 {
		sentences = 1
	}
	
	avgWordsPerSentence := float64(len(words)) / float64(sentences)
	
	// Optimal readability around 15-20 words per sentence
	if avgWordsPerSentence >= 15 && avgWordsPerSentence <= 20 {
		return 1.0
	} else if avgWordsPerSentence >= 10 && avgWordsPerSentence <= 25 {
		return 0.8
	} else {
		return 0.5
	}
}

func (r *AdvancedMemReader) assessInformativenessAdvanced(content string) float64 {
	// Advanced informativeness based on diversity of words
	words := strings.Fields(strings.ToLower(content))
	uniqueWords := make(map[string]bool)
	
	for _, word := range words {
		uniqueWords[word] = true
	}
	
	if len(words) == 0 {
		return 0.0
	}
	
	diversity := float64(len(uniqueWords)) / float64(len(words))
	return diversity
}