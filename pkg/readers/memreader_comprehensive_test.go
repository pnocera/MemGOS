package readers

import (
	"context"
	"testing"
	"time"

	"github.com/memtensor/memgos/pkg/types"
)

// TestMemReaderComprehensiveSystem tests the complete MemReader system
func TestMemReaderComprehensiveSystem(t *testing.T) {
	ctx := context.Background()
	
	// Test configuration management
	t.Run("ConfigurationManagement", func(t *testing.T) {
		testConfigurationManagement(t)
	})
	
	// Test factory patterns
	t.Run("FactoryPatterns", func(t *testing.T) {
		testFactoryPatterns(t)
	})
	
	// Test simple reader
	t.Run("SimpleReader", func(t *testing.T) {
		testSimpleReader(t, ctx)
	})
	
	// Test advanced reader
	t.Run("AdvancedReader", func(t *testing.T) {
		testAdvancedReader(t, ctx)
	})
	
	// Test pattern detection
	t.Run("PatternDetection", func(t *testing.T) {
		testPatternDetection(t, ctx)
	})
	
	// Test anomaly detection
	t.Run("AnomalyDetection", func(t *testing.T) {
		testAnomalyDetection(t, ctx)
	})
	
	// Test memory analytics
	t.Run("MemoryAnalytics", func(t *testing.T) {
		testMemoryAnalytics(t, ctx)
	})
}

func testConfigurationManagement(t *testing.T) {
	// Test default configuration
	config := DefaultReaderConfig()
	if config == nil {
		t.Error("Default configuration should not be nil")
	}
	
	if config.Strategy != ReadStrategyAdvanced {
		t.Errorf("Expected strategy %v, got %v", ReadStrategyAdvanced, config.Strategy)
	}
	
	// Test configuration manager
	cm := NewConfigManager("")
	
	// Test profiled configurations
	profiles := cm.GetAvailableProfiles()
	if len(profiles) == 0 {
		t.Error("Should have available profiles")
	}
	
	for _, profile := range profiles {
		config := cm.CreateProfiledConfig(profile)
		if config == nil {
			t.Errorf("Profile %s should create a valid configuration", profile)
		}
		
		err := cm.validateConfig(config)
		if err != nil {
			t.Errorf("Profile %s configuration should be valid: %v", profile, err)
		}
	}
	
	// Test custom configuration
	customOptions := map[string]interface{}{
		"strategy":           "advanced",
		"analysis_depth":     "deep",
		"pattern_detection":  true,
		"quality_assessment": true,
	}
	
	customConfig, err := cm.CreateCustomConfig(customOptions)
	if err != nil {
		t.Errorf("Failed to create custom configuration: %v", err)
	}
	
	if customConfig.Strategy != ReadStrategyAdvanced {
		t.Error("Custom configuration strategy not set correctly")
	}
}

func testFactoryPatterns(t *testing.T) {
	factory := NewReaderFactory(nil)
	
	// Test available reader types
	types := factory.GetAvailableReaderTypes()
	if len(types) == 0 {
		t.Error("Should have available reader types")
	}
	
	// Test reader creation
	for _, readerType := range types {
		reader, err := factory.CreateReader(readerType)
		if err != nil {
			t.Errorf("Failed to create reader type %s: %v", readerType, err)
		}
		
		if reader == nil {
			t.Errorf("Reader type %s should not be nil", readerType)
		}
		
		// Test reader capabilities
		capabilities, err := factory.GetReaderCapabilities(readerType)
		if err != nil {
			t.Errorf("Failed to get capabilities for %s: %v", readerType, err)
		}
		
		if capabilities == nil {
			t.Errorf("Capabilities for %s should not be nil", readerType)
		}
		
		reader.Close()
	}
	
	// Test reader recommendation
	requirements := &ReaderRequirements{
		NeedsSemanticSearch:    false,
		NeedsAdvancedAnalytics: true,
		MaxMemoryCount:         1000,
		PerformancePriority:    7,
		AccuracyPriority:       8,
		ResourceConstraints:    false,
	}
	
	recommendedType, err := factory.RecommendReaderType(requirements)
	if err != nil {
		t.Errorf("Failed to recommend reader type: %v", err)
	}
	
	if recommendedType == "" {
		t.Error("Should recommend a reader type")
	}
	
	// Test auto reader creation
	reader, err := factory.CreateAutoReader(requirements)
	if err != nil {
		t.Errorf("Failed to create auto reader: %v", err)
	}
	
	if reader == nil {
		t.Error("Auto reader should not be nil")
	}
	
	reader.Close()
}

func testSimpleReader(t *testing.T, ctx context.Context) {
	config := DefaultReaderConfig()
	config.Strategy = ReadStrategySimple
	config.AnalysisDepth = "basic"
	
	reader := NewSimpleMemReader(config)
	defer reader.Close()
	
	// Create test memories
	memories := createTestMemories()
	
	// Test memory analysis
	for _, memory := range memories {
		analysis, err := reader.AnalyzeMemory(ctx, memory)
		if err != nil {
			t.Errorf("Failed to analyze memory: %v", err)
		}
		
		if analysis == nil {
			t.Error("Analysis should not be nil")
		}
		
		if analysis.MemoryID != memory.GetID() {
			t.Error("Analysis memory ID should match")
		}
		
		if len(analysis.Keywords) == 0 {
			t.Error("Should extract keywords")
		}
	}
	
	// Test collection analysis
	collectionAnalysis, err := reader.AnalyzeMemories(ctx, memories)
	if err != nil {
		t.Errorf("Failed to analyze memory collection: %v", err)
	}
	
	if collectionAnalysis.TotalMemories != len(memories) {
		t.Error("Collection analysis count should match")
	}
	
	// Test pattern extraction
	patterns, err := reader.ExtractPatterns(ctx, memories, config.PatternConfig)
	if err != nil {
		t.Errorf("Failed to extract patterns: %v", err)
	}
	
	if patterns == nil {
		t.Error("Patterns should not be nil")
	}
	
	// Test summarization
	summary, err := reader.SummarizeMemories(ctx, memories, config.SummarizationConfig)
	if err != nil {
		t.Errorf("Failed to create summary: %v", err)
	}
	
	if summary == nil {
		t.Error("Summary should not be nil")
	}
	
	// Test duplicate detection
	duplicates, err := reader.DetectDuplicates(ctx, memories, 0.8)
	if err != nil {
		t.Errorf("Failed to detect duplicates: %v", err)
	}
	
	if duplicates == nil {
		t.Error("Duplicate analysis should not be nil")
	}
	
	// Test quality assessment
	quality, err := reader.AssessQuality(ctx, memories[0])
	if err != nil {
		t.Errorf("Failed to assess quality: %v", err)
	}
	
	if quality == nil {
		t.Error("Quality assessment should not be nil")
	}
	
	if quality.Score < 0 || quality.Score > 1 {
		t.Error("Quality score should be between 0 and 1")
	}
}

func testAdvancedReader(t *testing.T, ctx context.Context) {
	config := DefaultReaderConfig()
	config.Strategy = ReadStrategyAdvanced
	config.AnalysisDepth = "deep"
	
	reader := NewAdvancedMemReader(config, nil, nil, nil)
	defer reader.Close()
	
	// Create test memories
	memories := createTestMemories()
	
	// Test advanced memory analysis
	for _, memory := range memories {
		analysis, err := reader.AnalyzeMemory(ctx, memory)
		if err != nil {
			t.Errorf("Failed to analyze memory: %v", err)
		}
		
		if analysis == nil {
			t.Error("Analysis should not be nil")
		}
		
		// Advanced reader should provide more detailed analysis
		if len(analysis.Keywords) == 0 {
			t.Error("Should extract keywords")
		}
		
		if len(analysis.Themes) == 0 {
			t.Error("Should extract themes")
		}
		
		if analysis.Sentiment == nil {
			t.Error("Should have sentiment analysis")
		}
		
		if analysis.Complexity == 0 {
			t.Error("Should calculate complexity")
		}
	}
	
	// Test advanced collection analysis
	collectionAnalysis, err := reader.AnalyzeMemories(ctx, memories)
	if err != nil {
		t.Errorf("Failed to analyze memory collection: %v", err)
	}
	
	if collectionAnalysis == nil {
		t.Error("Collection analysis should not be nil")
	}
	
	if collectionAnalysis.Statistics == nil {
		t.Error("Should have statistics")
	}
	
	// Test advanced pattern extraction
	patterns, err := reader.ExtractPatterns(ctx, memories, config.PatternConfig)
	if err != nil {
		t.Errorf("Failed to extract patterns: %v", err)
	}
	
	if patterns == nil {
		t.Error("Patterns should not be nil")
	}
	
	// Advanced reader should detect more pattern types
	patternTypes := make(map[string]bool)
	for _, pattern := range patterns.Patterns {
		patternTypes[pattern.Type] = true
	}
	
	// Should have multiple types of patterns
	if len(patternTypes) == 0 {
		t.Error("Should detect pattern types")
	}
}

func testPatternDetection(t *testing.T, ctx context.Context) {
	config := DefaultReaderConfig()
	reader := NewAdvancedMemReader(config, nil, nil, nil)
	defer reader.Close()
	
	// Create memories with specific patterns
	patternMemories := createPatternTestMemories()
	
	// Test pattern extraction
	patterns, err := reader.ExtractPatterns(ctx, patternMemories, config.PatternConfig)
	if err != nil {
		t.Errorf("Failed to extract patterns: %v", err)
	}
	
	// Should detect various pattern types
	hasSemanticPattern := false
	hasTemporalPattern := false
	hasStructuralPattern := false
	
	for _, pattern := range patterns.Patterns {
		switch pattern.Type {
		case "semantic", "topic", "sentiment":
			hasSemanticPattern = true
		case "temporal":
			hasTemporalPattern = true
		case "structural":
			hasStructuralPattern = true
		}
	}
	
	// Should detect at least some patterns
	if !hasSemanticPattern && !hasTemporalPattern && !hasStructuralPattern {
		t.Error("Should detect some patterns")
	}
	
	// Test sequence mining
	if len(patterns.Sequences) == 0 {
		// Sequences might be empty for small datasets
		t.Log("No sequences detected (may be normal for small datasets)")
	}
	
	// Test pattern confidence
	if patterns.Confidence < 0 || patterns.Confidence > 1 {
		t.Error("Pattern confidence should be between 0 and 1")
	}
}

func testAnomalyDetection(t *testing.T, ctx context.Context) {
	config := DefaultReaderConfig()
	reader := NewAdvancedMemReader(config, nil, nil, nil)
	defer reader.Close()
	
	// Create memories with anomalies
	anomalyMemories := createAnomalyTestMemories()
	
	// First extract patterns to have a baseline
	patterns, err := reader.ExtractPatterns(ctx, anomalyMemories, config.PatternConfig)
	if err != nil {
		t.Errorf("Failed to extract patterns: %v", err)
	}
	
	// Test anomaly detection
	anomalies, err := reader.detectPatternAnomalies(ctx, anomalyMemories, patterns.Patterns)
	if err != nil {
		t.Errorf("Failed to detect anomalies: %v", err)
	}
	
	// Verify anomalies were detected
	if len(anomalies) == 0 {
		t.Log("No anomalies detected in test data (this is acceptable)")
	} else {
		t.Logf("Detected %d anomalies", len(anomalies))
	}
	
	// Test statistical anomalies
	statAnomalies := reader.detectStatisticalAnomalies(anomalyMemories)
	if len(statAnomalies) > 0 {
		for _, anomaly := range statAnomalies {
			if anomaly.Severity < 0 || anomaly.Severity > 1 {
				t.Error("Anomaly severity should be between 0 and 1")
			}
			
			if len(anomaly.MemoryIDs) == 0 {
				t.Error("Anomaly should reference memory IDs")
			}
		}
	}
	
	// Test temporal anomalies
	tempAnomalies := reader.detectTemporalAnomalies(anomalyMemories)
	// Temporal anomalies depend on time gaps
	t.Logf("Detected %d temporal anomalies", len(tempAnomalies))
	
	// Test content anomalies
	contentAnomalies := reader.detectContentAnomalies(anomalyMemories)
	// Content anomalies depend on content characteristics
	t.Logf("Detected %d content anomalies", len(contentAnomalies))
}

func testMemoryAnalytics(t *testing.T, ctx context.Context) {
	config := DefaultReaderConfig()
	reader := NewAdvancedMemReader(config, nil, nil, nil)
	defer reader.Close()
	
	memories := createTestMemories()
	
	// Test advanced frequencies
	frequencies, err := reader.calculateAdvancedFrequencies(ctx, memories)
	if err != nil {
		t.Errorf("Failed to calculate frequencies: %v", err)
	}
	
	if len(frequencies) == 0 {
		t.Error("Should calculate some frequencies")
	}
	
	// Test TF-IDF calculation
	tfidf := reader.calculateTFIDF(memories)
	if len(tfidf) == 0 {
		t.Error("Should calculate TF-IDF scores")
	}
	
	// Test theme frequencies
	themeFreq := reader.calculateThemeFrequencies(memories)
	if len(themeFreq) == 0 {
		t.Error("Should calculate theme frequencies")
	}
	
	// Test semantic relationships
	relationships, err := reader.detectSemanticRelationships(ctx, memories)
	if err != nil {
		t.Errorf("Failed to detect relationships: %v", err)
	}
	
	// Relationships depend on content similarity
	t.Logf("Detected %d relationships", len(relationships))
	
	// Test semantic clustering
	clusters, err := reader.performSemanticClustering(ctx, memories)
	if err != nil {
		t.Errorf("Failed to perform clustering: %v", err)
	}
	
	// Clusters depend on content similarity
	t.Logf("Created %d clusters", len(clusters))
	
	for _, cluster := range clusters {
		if cluster.Coherence < 0 || cluster.Coherence > 1 {
			t.Error("Cluster coherence should be between 0 and 1")
		}
		
		if len(cluster.MemoryIDs) == 0 {
			t.Error("Cluster should contain memory IDs")
		}
	}
}

// Helper functions to create test data

func createTestMemories() []types.MemoryItem {
	now := time.Now()
	
	return []types.MemoryItem{
		&types.TextualMemoryItem{
			ID:        "mem1",
			Memory:    "This is a technical document about software development. It discusses algorithms and data structures in computer science.",
			Metadata:  map[string]interface{}{"type": "technical"},
			CreatedAt: now.Add(-3 * time.Hour),
			UpdatedAt: now.Add(-3 * time.Hour),
		},
		&types.TextualMemoryItem{
			ID:        "mem2",
			Memory:    "A business analysis of market trends shows positive growth. Revenue and profit margins are improving consistently.",
			Metadata:  map[string]interface{}{"type": "business"},
			CreatedAt: now.Add(-2 * time.Hour),
			UpdatedAt: now.Add(-2 * time.Hour),
		},
		&types.TextualMemoryItem{
			ID:        "mem3",
			Memory:    "Personal reflection on learning and growth. I feel inspired by recent achievements and experiences in my journey.",
			Metadata:  map[string]interface{}{"type": "personal"},
			CreatedAt: now.Add(-1 * time.Hour),
			UpdatedAt: now.Add(-1 * time.Hour),
		},
		&types.TextualMemoryItem{
			ID:        "mem4",
			Memory:    "Research study on machine learning applications. The analysis concludes that neural networks show promising results.",
			Metadata:  map[string]interface{}{"type": "academic"},
			CreatedAt: now.Add(-30 * time.Minute),
			UpdatedAt: now.Add(-30 * time.Minute),
		},
		&types.TextualMemoryItem{
			ID:        "mem5",
			Memory:    "Creative writing exercise exploring imagination. The artistic process involves inspiration and creative visualization.",
			Metadata:  map[string]interface{}{"type": "creative"},
			CreatedAt: now.Add(-10 * time.Minute),
			UpdatedAt: now.Add(-10 * time.Minute),
		},
	}
}

func createPatternTestMemories() []types.MemoryItem {
	now := time.Now()
	
	return []types.MemoryItem{
		&types.TextualMemoryItem{
			ID:        "pattern1",
			Memory:    "What is machine learning? How does it work? Can we improve algorithms?",
			Metadata:  map[string]interface{}{"pattern": "questions"},
			CreatedAt: now.Add(-5 * time.Hour),
			UpdatedAt: now.Add(-5 * time.Hour),
		},
		&types.TextualMemoryItem{
			ID:        "pattern2",
			Memory:    "Machine learning algorithms are powerful tools. They enable data-driven decisions. We use them extensively.",
			Metadata:  map[string]interface{}{"pattern": "statements"},
			CreatedAt: now.Add(-4 * time.Hour),
			UpdatedAt: now.Add(-4 * time.Hour),
		},
		&types.TextualMemoryItem{
			ID:        "pattern3",
			Memory:    "Excellent results! Amazing performance! Outstanding achievements in machine learning research!",
			Metadata:  map[string]interface{}{"pattern": "exclamations"},
			CreatedAt: now.Add(-3 * time.Hour),
			UpdatedAt: now.Add(-3 * time.Hour),
		},
		&types.TextualMemoryItem{
			ID:        "pattern4",
			Memory:    "Key features of machine learning:\n- Data processing\n- Pattern recognition\n- Predictive modeling\n- Algorithm optimization",
			Metadata:  map[string]interface{}{"pattern": "structured"},
			CreatedAt: now.Add(-2 * time.Hour),
			UpdatedAt: now.Add(-2 * time.Hour),
		},
		&types.TextualMemoryItem{
			ID:        "pattern5",
			Memory:    "Short text.",
			Metadata:  map[string]interface{}{"pattern": "short"},
			CreatedAt: now.Add(-1 * time.Hour),
			UpdatedAt: now.Add(-1 * time.Hour),
		},
	}
}

func createAnomalyTestMemories() []types.MemoryItem {
	now := time.Now()
	
	return []types.MemoryItem{
		&types.TextualMemoryItem{
			ID:        "anomaly1",
			Memory:    "Normal length content with typical structure and readable text format.",
			Metadata:  map[string]interface{}{"type": "normal"},
			CreatedAt: now.Add(-24 * time.Hour),
			UpdatedAt: now.Add(-24 * time.Hour),
		},
		&types.TextualMemoryItem{
			ID:        "anomaly2",
			Memory:    "Short",
			Metadata:  map[string]interface{}{"type": "short_anomaly"},
			CreatedAt: now.Add(-23 * time.Hour),
			UpdatedAt: now.Add(-23 * time.Hour),
		},
		&types.TextualMemoryItem{
			ID:        "anomaly3",
			Memory:    "This is an extremely long content that goes on and on with lots of text and information that might be considered an outlier in terms of length compared to other memories in the collection. It contains many words and sentences that make it significantly longer than typical content. This excessive length might be detected as a statistical anomaly when analyzing the collection of memories for patterns and outliers.",
			Metadata:  map[string]interface{}{"type": "long_anomaly"},
			CreatedAt: now.Add(-22 * time.Hour),
			UpdatedAt: now.Add(-22 * time.Hour),
		},
		&types.TextualMemoryItem{
			ID:        "anomaly4",
			Memory:    "The same word repeated word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word word.",
			Metadata:  map[string]interface{}{"type": "repetitive_anomaly"},
			CreatedAt: now.Add(-1 * time.Hour),  // Large time gap
			UpdatedAt: now.Add(-1 * time.Hour),
		},
		&types.TextualMemoryItem{
			ID:        "anomaly5",
			Memory:    "Normal content again with standard structure.",
			Metadata:  map[string]interface{}{"type": "normal"},
			CreatedAt: now.Add(-30 * time.Minute),
			UpdatedAt: now.Add(-30 * time.Minute),
		},
	}
}