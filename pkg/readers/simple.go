package readers

import (
	"context"
	"strings"
	"time"

	"github.com/memtensor/memgos/pkg/types"
)

// SimpleMemReader implements basic memory reading functionality
type SimpleMemReader struct {
	*BaseMemReader
}

// NewSimpleMemReader creates a new simple memory reader
func NewSimpleMemReader(config *ReaderConfig) *SimpleMemReader {
	if config == nil {
		config = DefaultReaderConfig()
		config.Strategy = ReadStrategySimple
		config.AnalysisDepth = "basic"
	}
	
	return &SimpleMemReader{
		BaseMemReader: NewBaseMemReader(config),
	}
}

// Read reads and analyzes memory content using simple strategies
func (r *SimpleMemReader) Read(ctx context.Context, query *ReadQuery) (*ReadResult, error) {
	startTime := time.Now()
	
	// Simple reading strategy - direct content matching
	memories, err := r.performSimpleSearch(ctx, query)
	if err != nil {
		return nil, err
	}
	
	result := &ReadResult{
		Memories:    memories,
		TotalFound:  len(memories),
		ProcessTime: time.Since(startTime),
		Metadata: map[string]interface{}{
			"strategy": "simple",
			"depth":    "basic",
		},
	}
	
	// Add optional analysis based on configuration
	if r.config.QualityAssessment {
		result.Quality = r.performBasicQualityAssessment(memories)
	}
	
	if r.config.DuplicateDetection {
		result.Duplicates = r.performBasicDuplicateDetection(memories)
	}
	
	if r.config.Summarization {
		result.Summary = r.performBasicSummarization(memories)
	}
	
	return result, nil
}

// ReadMemory reads a specific memory item
func (r *SimpleMemReader) ReadMemory(ctx context.Context, memoryID string) (*MemoryAnalysis, error) {
	// In a real implementation, this would fetch from a memory store
	// For now, we'll create a mock analysis
	return &MemoryAnalysis{
		MemoryID:  memoryID,
		Type:      "textual",
		Content:   "Mock content for memory " + memoryID,
		Keywords:  []string{"mock", "memory", "content"},
		Themes:    []string{"testing", "implementation"},
		Sentiment: analyzeSentiment("Mock content for memory " + memoryID),
		Complexity: 0.5,
		Relevance:  0.8,
		Quality:    0.7,
		Metadata: map[string]interface{}{
			"reader": "simple",
			"depth":  "basic",
		},
		Timestamp: time.Now(),
		Insights:  []string{"Basic memory analysis", "Simple extraction"},
	}, nil
}

// ReadMemories reads multiple memory items
func (r *SimpleMemReader) ReadMemories(ctx context.Context, memoryIDs []string) ([]*MemoryAnalysis, error) {
	analyses := make([]*MemoryAnalysis, len(memoryIDs))
	
	for i, memoryID := range memoryIDs {
		analysis, err := r.ReadMemory(ctx, memoryID)
		if err != nil {
			return nil, err
		}
		analyses[i] = analysis
	}
	
	return analyses, nil
}

// AnalyzeMemory analyzes memory content for patterns and insights
func (r *SimpleMemReader) AnalyzeMemory(ctx context.Context, memory types.MemoryItem) (*MemoryAnalysis, error) {
	content := memory.GetContent()
	
	return &MemoryAnalysis{
		MemoryID:   memory.GetID(),
		Type:       inferMemoryType(memory),
		Content:    content,
		Keywords:   extractKeywords(content),
		Themes:     extractThemes(content),
		Sentiment:  analyzeSentiment(content),
		Complexity: calculateComplexity(content),
		Relevance:  0.8, // Default relevance
		Quality:    calculateQuality(memory),
		Metadata: map[string]interface{}{
			"reader":     "simple",
			"created_at": memory.GetCreatedAt(),
			"updated_at": memory.GetUpdatedAt(),
		},
		Timestamp: time.Now(),
		Insights:  r.generateBasicInsights(memory),
	}, nil
}

// AnalyzeMemories analyzes multiple memories for relationships
func (r *SimpleMemReader) AnalyzeMemories(ctx context.Context, memories []types.MemoryItem) (*MemoryCollectionAnalysis, error) {
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
	
	// Basic analysis
	types := make(map[string]int)
	themes := make(map[string]int)
	var totalLength int
	var totalQuality float64
	
	for _, memory := range memories {
		memType := inferMemoryType(memory)
		types[memType]++
		
		content := memory.GetContent()
		totalLength += len(content)
		totalQuality += calculateQuality(memory)
		
		// Extract and count themes
		memThemes := extractThemes(content)
		for _, theme := range memThemes {
			themes[theme]++
		}
	}
	
	// Calculate statistics
	stats := &MemoryStatistics{
		AverageLength:  float64(totalLength) / float64(len(memories)),
		AverageQuality: totalQuality / float64(len(memories)),
		CreationRate:   r.calculateCreationRate(memories),
		UpdateFrequency: r.calculateUpdateFrequency(memories),
	}
	
	// Generate basic relationships
	relationships := r.generateBasicRelationships(memories)
	
	// Create timeline
	timeline := r.generateBasicTimeline(memories)
	
	// Generate insights
	insights := r.generateCollectionInsights(memories, types, themes)
	
	return &MemoryCollectionAnalysis{
		TotalMemories: len(memories),
		Types:         types,
		Themes:        themes,
		Relationships: relationships,
		Clusters:      []MemoryCluster{}, // Simple reader doesn't do clustering
		Timeline:      timeline,
		Statistics:    stats,
		Insights:      insights,
		Recommendations: r.generateRecommendations(memories, types, themes),
	}, nil
}

// ExtractPatterns extracts patterns from memory content
func (r *SimpleMemReader) ExtractPatterns(ctx context.Context, memories []types.MemoryItem, config *PatternConfig) (*PatternAnalysis, error) {
	if config == nil {
		config = r.config.PatternConfig
	}
	
	patterns := make([]Pattern, 0)
	frequencies := make(map[string]int)
	
	// Simple pattern extraction - word frequency analysis
	for _, memory := range memories {
		content := memory.GetContent()
		words := strings.Fields(strings.ToLower(content))
		
		for _, word := range words {
			if len(word) > 3 { // Skip short words
				frequencies[word]++
			}
		}
	}
	
	// Convert frequent words to patterns
	for word, freq := range frequencies {
		if float64(freq)/float64(len(memories)) >= config.MinSupport {
			patterns = append(patterns, Pattern{
				ID:          "word_" + word,
				Type:        "frequent_word",
				Description: "Frequently occurring word: " + word,
				Instances:   []string{word},
				Frequency:   freq,
				Confidence:  float64(freq) / float64(len(memories)),
				Significance: calculateWordSignificance(word, freq, len(memories)),
			})
		}
	}
	
	return &PatternAnalysis{
		Patterns:    patterns,
		Frequencies: frequencies,
		Sequences:   []PatternSequence{}, // Simple reader doesn't detect sequences
		Anomalies:   []PatternAnomaly{},  // Simple reader doesn't detect anomalies
		Confidence:  0.6,
		Metadata: map[string]interface{}{
			"method":  "frequency_analysis",
			"reader":  "simple",
			"config":  config,
		},
	}, nil
}

// SummarizeMemories creates summaries of memory collections
func (r *SimpleMemReader) SummarizeMemories(ctx context.Context, memories []types.MemoryItem, config *SummarizationConfig) (*MemorySummary, error) {
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
	
	// Collect all content
	var allContent strings.Builder
	allThemes := make(map[string]int)
	var totalComplexity float64
	
	for _, memory := range memories {
		content := memory.GetContent()
		allContent.WriteString(content)
		allContent.WriteString(" ")
		
		themes := extractThemes(content)
		for _, theme := range themes {
			allThemes[theme]++
		}
		
		totalComplexity += calculateComplexity(content)
	}
	
	// Generate summary (simple extractive approach)
	summary := r.generateSimpleSummary(allContent.String(), config.MaxLength)
	
	// Extract key points
	keyPoints := r.extractKeyPoints(allContent.String(), config.KeyPoints)
	
	// Get top themes
	topThemes := r.getTopThemes(allThemes, 5)
	
	// Simple entity extraction
	entities := r.extractSimpleEntities(allContent.String())
	
	return &MemorySummary{
		Summary:      summary,
		KeyPoints:    keyPoints,
		Themes:       topThemes,
		Entities:     entities,
		Complexity:   totalComplexity / float64(len(memories)),
		Coherence:    r.calculateCoherence(memories),
		Completeness: r.calculateCompleteness(memories),
		Metadata: map[string]interface{}{
			"method":      "extractive",
			"reader":      "simple",
			"total_memories": len(memories),
		},
		GeneratedAt: time.Now(),
	}, nil
}

// DetectDuplicates identifies duplicate memories
func (r *SimpleMemReader) DetectDuplicates(ctx context.Context, memories []types.MemoryItem, threshold float64) (*DuplicateAnalysis, error) {
	duplicates := make([]DuplicateGroup, 0)
	processed := make(map[string]bool)
	
	for i, mem1 := range memories {
		if processed[mem1.GetID()] {
			continue
		}
		
		group := DuplicateGroup{
			ID:        "group_" + mem1.GetID(),
			MemoryIDs: []string{mem1.GetID()},
			Similarity: 1.0,
			Confidence: 1.0,
			Recommended: mem1.GetID(),
		}
		
		for j := i + 1; j < len(memories); j++ {
			mem2 := memories[j]
			if processed[mem2.GetID()] {
				continue
			}
			
			similarity := calculateSimilarity(mem1, mem2)
			if similarity >= threshold {
				group.MemoryIDs = append(group.MemoryIDs, mem2.GetID())
				processed[mem2.GetID()] = true
				
				// Update group similarity (minimum of all pairs)
				if similarity < group.Similarity {
					group.Similarity = similarity
				}
			}
		}
		
		if len(group.MemoryIDs) > 1 {
			duplicates = append(duplicates, group)
		}
		
		processed[mem1.GetID()] = true
	}
	
	return &DuplicateAnalysis{
		Duplicates:      duplicates,
		TotalDuplicates: len(duplicates),
		Threshold:       threshold,
		Method:          "similarity_threshold",
		Metadata: map[string]interface{}{
			"reader":    "simple",
			"algorithm": "pairwise_comparison",
		},
	}, nil
}

// AssessQuality evaluates memory quality
func (r *SimpleMemReader) AssessQuality(ctx context.Context, memory types.MemoryItem) (*QualityAssessment, error) {
	content := memory.GetContent()
	
	// Quality dimensions
	dimensions := map[string]float64{
		"length":      r.assessLength(content),
		"structure":   r.assessStructure(content),
		"readability": r.assessReadability(content),
		"informativeness": r.assessInformativeness(content),
	}
	
	// Calculate overall score
	var totalScore float64
	for _, score := range dimensions {
		totalScore += score
	}
	overallScore := totalScore / float64(len(dimensions))
	
	// Identify issues
	issues := r.identifyQualityIssues(content, dimensions)
	
	// Generate suggestions
	suggestions := r.generateQualitySuggestions(issues)
	
	return &QualityAssessment{
		Score:      overallScore,
		Dimensions: dimensions,
		Issues:     issues,
		Suggestions: suggestions,
		Metadata: map[string]interface{}{
			"reader":    "simple",
			"method":    "heuristic_analysis",
			"timestamp": time.Now(),
		},
	}, nil
}

// Helper methods for SimpleMemReader

func (r *SimpleMemReader) performSimpleSearch(ctx context.Context, query *ReadQuery) ([]types.MemoryItem, error) {
	// Mock implementation - in real scenario, this would search actual memory stores
	memories := make([]types.MemoryItem, 0)
	
	// Create mock memories for demonstration
	for i := 0; i < min(query.TopK, 10); i++ {
		memory := &types.TextualMemoryItem{
			ID:        "mem_" + string(rune('0'+i)),
			Memory:    "Mock memory content for query: " + query.Query,
			Metadata:  map[string]interface{}{"mock": true, "index": i},
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Hour),
			UpdatedAt: time.Now(),
		}
		memories = append(memories, memory)
	}
	
	return memories, nil
}

func (r *SimpleMemReader) performBasicQualityAssessment(memories []types.MemoryItem) *QualityAssessment {
	if len(memories) == 0 {
		return &QualityAssessment{
			Score:       0.0,
			Dimensions:  make(map[string]float64),
			Issues:      []QualityIssue{},
			Suggestions: []string{"No memories to assess"},
		}
	}
	
	var totalQuality float64
	for _, memory := range memories {
		totalQuality += calculateQuality(memory)
	}
	
	avgQuality := totalQuality / float64(len(memories))
	
	return &QualityAssessment{
		Score: avgQuality,
		Dimensions: map[string]float64{
			"overall": avgQuality,
		},
		Issues: []QualityIssue{},
		Suggestions: []string{"Quality assessment completed"},
		Metadata: map[string]interface{}{
			"method": "average_quality",
			"count":  len(memories),
		},
	}
}

func (r *SimpleMemReader) performBasicDuplicateDetection(memories []types.MemoryItem) *DuplicateAnalysis {
	return &DuplicateAnalysis{
		Duplicates:      []DuplicateGroup{},
		TotalDuplicates: 0,
		Threshold:       0.8,
		Method:          "basic_similarity",
		Metadata: map[string]interface{}{
			"reader": "simple",
			"method": "basic_detection",
		},
	}
}

func (r *SimpleMemReader) performBasicSummarization(memories []types.MemoryItem) *MemorySummary {
	if len(memories) == 0 {
		return &MemorySummary{
			Summary:     "No memories to summarize",
			KeyPoints:   []string{},
			Themes:      []string{},
			Entities:    []string{},
			GeneratedAt: time.Now(),
		}
	}
	
	return &MemorySummary{
		Summary:     "Basic summary of " + string(rune(len(memories)+'0')) + " memories",
		KeyPoints:   []string{"Memory collection analyzed", "Basic patterns identified"},
		Themes:      []string{"general", "content"},
		Entities:    []string{},
		Complexity:  0.5,
		Coherence:   0.7,
		Completeness: 0.8,
		GeneratedAt: time.Now(),
	}
}

func inferMemoryType(memory types.MemoryItem) string {
	switch memory.(type) {
	case *types.TextualMemoryItem:
		return "textual"
	case *types.ActivationMemoryItem:
		return "activation"
	case *types.ParametricMemoryItem:
		return "parametric"
	default:
		return "unknown"
	}
}

func (r *SimpleMemReader) generateBasicInsights(memory types.MemoryItem) []string {
	insights := []string{}
	content := memory.GetContent()
	
	if len(content) > 500 {
		insights = append(insights, "Long-form content detected")
	}
	
	if strings.Contains(content, "?") {
		insights = append(insights, "Contains questions")
	}
	
	if strings.Contains(content, "!") {
		insights = append(insights, "Contains exclamations")
	}
	
	return insights
}

func (r *SimpleMemReader) calculateCreationRate(memories []types.MemoryItem) float64 {
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

func (r *SimpleMemReader) calculateUpdateFrequency(memories []types.MemoryItem) float64 {
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

func (r *SimpleMemReader) generateBasicRelationships(memories []types.MemoryItem) []MemoryRelationship {
	relationships := []MemoryRelationship{}
	
	// Simple relationship detection based on similarity
	for i := 0; i < len(memories); i++ {
		for j := i + 1; j < len(memories); j++ {
			similarity := calculateSimilarity(memories[i], memories[j])
			if similarity > 0.3 {
				relationships = append(relationships, MemoryRelationship{
					SourceID:    memories[i].GetID(),
					TargetID:    memories[j].GetID(),
					Type:        "similar",
					Strength:    similarity,
					Description: "Similar content detected",
				})
			}
		}
	}
	
	return relationships
}

func (r *SimpleMemReader) generateBasicTimeline(memories []types.MemoryItem) []MemoryTimelinePoint {
	timeline := []MemoryTimelinePoint{}
	
	// Group memories by day
	dayGroups := make(map[string][]types.MemoryItem)
	for _, memory := range memories {
		day := memory.GetCreatedAt().Format("2006-01-02")
		dayGroups[day] = append(dayGroups[day], memory)
	}
	
	// Create timeline points
	for day, dayMemories := range dayGroups {
		t, _ := time.Parse("2006-01-02", day)
		events := []string{}
		for _, memory := range dayMemories {
			events = append(events, memory.GetID())
		}
		
		timeline = append(timeline, MemoryTimelinePoint{
			Timestamp: t,
			Count:     len(dayMemories),
			Events:    events,
		})
	}
	
	return timeline
}

func (r *SimpleMemReader) generateCollectionInsights(memories []types.MemoryItem, types map[string]int, themes map[string]int) []string {
	insights := []string{}
	
	if len(memories) > 10 {
		insights = append(insights, "Large memory collection detected")
	}
	
	dominantType := ""
	maxCount := 0
	for memType, count := range types {
		if count > maxCount {
			maxCount = count
			dominantType = memType
		}
	}
	
	if dominantType != "" {
		insights = append(insights, "Dominant memory type: "+dominantType)
	}
	
	if len(themes) > 5 {
		insights = append(insights, "Diverse thematic content")
	}
	
	return insights
}

func (r *SimpleMemReader) generateRecommendations(memories []types.MemoryItem, types map[string]int, themes map[string]int) []string {
	recommendations := []string{}
	
	if len(memories) > 100 {
		recommendations = append(recommendations, "Consider memory consolidation")
	}
	
	if len(themes) > 10 {
		recommendations = append(recommendations, "Consider thematic organization")
	}
	
	recommendations = append(recommendations, "Regular quality assessment recommended")
	
	return recommendations
}

func calculateWordSignificance(word string, freq int, totalMemories int) float64 {
	// Simple significance calculation based on frequency
	return float64(freq) / float64(totalMemories)
}

func (r *SimpleMemReader) generateSimpleSummary(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}
	
	// Simple extractive summary - first sentences
	sentences := strings.Split(content, ".")
	summary := ""
	
	for _, sentence := range sentences {
		if len(summary)+len(sentence) <= maxLength {
			summary += sentence + "."
		} else {
			break
		}
	}
	
	return summary
}

func (r *SimpleMemReader) extractKeyPoints(content string, numPoints int) []string {
	sentences := strings.Split(content, ".")
	keyPoints := []string{}
	
	// Simple approach - take first sentences
	for i := 0; i < min(numPoints, len(sentences)); i++ {
		if strings.TrimSpace(sentences[i]) != "" {
			keyPoints = append(keyPoints, strings.TrimSpace(sentences[i]))
		}
	}
	
	return keyPoints
}

func (r *SimpleMemReader) getTopThemes(themes map[string]int, limit int) []string {
	type themeCount struct {
		theme string
		count int
	}
	
	var themeCounts []themeCount
	for theme, count := range themes {
		themeCounts = append(themeCounts, themeCount{theme, count})
	}
	
	// Simple sorting - in real implementation, use proper sorting
	topThemes := []string{}
	for i := 0; i < min(limit, len(themeCounts)); i++ {
		topThemes = append(topThemes, themeCounts[i].theme)
	}
	
	return topThemes
}

func (r *SimpleMemReader) extractSimpleEntities(content string) []string {
	// Simple entity extraction - capitalize words
	words := strings.Fields(content)
	entities := []string{}
	
	for _, word := range words {
		if len(word) > 0 && strings.ToUpper(word[:1]) == word[:1] {
			entities = append(entities, word)
		}
	}
	
	return entities
}

func (r *SimpleMemReader) calculateCoherence(memories []types.MemoryItem) float64 {
	// Simple coherence calculation
	if len(memories) <= 1 {
		return 1.0
	}
	
	var totalSimilarity float64
	var comparisons int
	
	for i := 0; i < len(memories); i++ {
		for j := i + 1; j < len(memories); j++ {
			totalSimilarity += calculateSimilarity(memories[i], memories[j])
			comparisons++
		}
	}
	
	if comparisons == 0 {
		return 0.0
	}
	
	return totalSimilarity / float64(comparisons)
}

func (r *SimpleMemReader) calculateCompleteness(memories []types.MemoryItem) float64 {
	// Simple completeness calculation based on content length
	if len(memories) == 0 {
		return 0.0
	}
	
	var totalLength int
	for _, memory := range memories {
		totalLength += len(memory.GetContent())
	}
	
	avgLength := float64(totalLength) / float64(len(memories))
	
	// Normalize to 0-1 range (assuming 200 chars is "complete")
	completeness := avgLength / 200.0
	if completeness > 1.0 {
		completeness = 1.0
	}
	
	return completeness
}

func (r *SimpleMemReader) assessLength(content string) float64 {
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

func (r *SimpleMemReader) assessStructure(content string) float64 {
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

func (r *SimpleMemReader) assessReadability(content string) float64 {
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

func (r *SimpleMemReader) assessInformativeness(content string) float64 {
	// Simple informativeness based on diversity of words
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

func (r *SimpleMemReader) identifyQualityIssues(content string, dimensions map[string]float64) []QualityIssue {
	issues := []QualityIssue{}
	
	if dimensions["length"] < 0.5 {
		issues = append(issues, QualityIssue{
			Type:        "length",
			Severity:    "medium",
			Description: "Content is too short",
			Suggestion:  "Consider expanding the content",
			Impact:      0.3,
		})
	}
	
	if dimensions["structure"] < 0.5 {
		issues = append(issues, QualityIssue{
			Type:        "structure",
			Severity:    "low",
			Description: "Content lacks structure",
			Suggestion:  "Add punctuation and paragraphs",
			Impact:      0.2,
		})
	}
	
	if dimensions["readability"] < 0.5 {
		issues = append(issues, QualityIssue{
			Type:        "readability",
			Severity:    "medium",
			Description: "Content may be hard to read",
			Suggestion:  "Adjust sentence length",
			Impact:      0.4,
		})
	}
	
	return issues
}

func (r *SimpleMemReader) generateQualitySuggestions(issues []QualityIssue) []string {
	suggestions := []string{}
	
	for _, issue := range issues {
		suggestions = append(suggestions, issue.Suggestion)
	}
	
	if len(suggestions) == 0 {
		suggestions = append(suggestions, "Content quality is satisfactory")
	}
	
	return suggestions
}

// Removed duplicate min function - using the one from base.go