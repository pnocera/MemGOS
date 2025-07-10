package examples_usage

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/memtensor/memgos/pkg/readers"
	"github.com/memtensor/memgos/pkg/types"
)

// ExampleBasicMemoryReading demonstrates basic memory reading and analysis
func ExampleBasicMemoryReading() {
	ctx := context.Background()
	
	// Create a simple memory reader
	config := readers.DefaultReaderConfig()
	config.Strategy = readers.ReadStrategySimple
	
	reader := readers.NewSimpleMemReader(config)
	defer reader.Close()
	
	// Create test memories
	memories := createSampleMemories()
	
	// Analyze individual memory
	analysis, err := reader.AnalyzeMemory(ctx, memories[0])
	if err != nil {
		log.Fatalf("Failed to analyze memory: %v", err)
	}
	
	fmt.Printf("Memory Analysis:\n")
	fmt.Printf("  ID: %s\n", analysis.MemoryID)
	fmt.Printf("  Type: %s\n", analysis.Type)
	fmt.Printf("  Keywords: %v\n", analysis.Keywords)
	fmt.Printf("  Themes: %v\n", analysis.Themes)
	fmt.Printf("  Quality Score: %.2f\n", analysis.Quality)
	fmt.Printf("  Complexity: %.2f\n", analysis.Complexity)
	fmt.Printf("  Sentiment: %s (%.2f)\n", analysis.Sentiment.Emotion, analysis.Sentiment.Polarity)
	
	// Analyze collection of memories
	collectionAnalysis, err := reader.AnalyzeMemories(ctx, memories)
	if err != nil {
		log.Fatalf("Failed to analyze memory collection: %v", err)
	}
	
	fmt.Printf("\nCollection Analysis:\n")
	fmt.Printf("  Total Memories: %d\n", collectionAnalysis.TotalMemories)
	fmt.Printf("  Memory Types: %v\n", collectionAnalysis.Types)
	fmt.Printf("  Common Themes: %v\n", collectionAnalysis.Themes)
	fmt.Printf("  Average Quality: %.2f\n", collectionAnalysis.Statistics.AverageQuality)
	fmt.Printf("  Relationships Found: %d\n", len(collectionAnalysis.Relationships))
}

// ExampleAdvancedPatternDetection demonstrates advanced pattern detection
func ExampleAdvancedPatternDetection() {
	ctx := context.Background()
	
	// Create advanced memory reader
	config := readers.DefaultReaderConfig()
	config.Strategy = readers.ReadStrategyAdvanced
	config.PatternDetection = true
	config.AnalysisDepth = "deep"
	
	reader := readers.NewAdvancedMemReader(config, nil, nil, nil)
	defer reader.Close()
	
	// Create memories with patterns
	memories := createPatternMemories()
	
	// Extract patterns
	patterns, err := reader.ExtractPatterns(ctx, memories, config.PatternConfig)
	if err != nil {
		log.Fatalf("Failed to extract patterns: %v", err)
	}
	
	fmt.Printf("Pattern Analysis Results:\n")
	fmt.Printf("  Total Patterns Found: %d\n", len(patterns.Patterns))
	fmt.Printf("  Pattern Confidence: %.2f\n", patterns.Confidence)
	
	// Display detected patterns
	for i, pattern := range patterns.Patterns {
		fmt.Printf("\n  Pattern %d:\n", i+1)
		fmt.Printf("    Type: %s\n", pattern.Type)
		fmt.Printf("    Description: %s\n", pattern.Description)
		fmt.Printf("    Frequency: %d\n", pattern.Frequency)
		fmt.Printf("    Confidence: %.2f\n", pattern.Confidence)
		fmt.Printf("    Significance: %.2f\n", pattern.Significance)
	}
	
	// Display pattern sequences
	if len(patterns.Sequences) > 0 {
		fmt.Printf("\n  Pattern Sequences:\n")
		for i, sequence := range patterns.Sequences {
			fmt.Printf("    Sequence %d: %v (Support: %.2f)\n", 
				i+1, sequence.Patterns, sequence.Support)
		}
	}
	
	// Display anomalies
	if len(patterns.Anomalies) > 0 {
		fmt.Printf("\n  Anomalies Detected:\n")
		for i, anomaly := range patterns.Anomalies {
			fmt.Printf("    Anomaly %d: %s (Severity: %.2f)\n", 
				i+1, anomaly.Description, anomaly.Severity)
		}
	}
}

// ExampleMemoryQualityAssessment demonstrates comprehensive quality assessment
func ExampleMemoryQualityAssessment() {
	ctx := context.Background()
	
	// Create reader with quality assessment enabled
	config := readers.DefaultReaderConfig()
	config.QualityAssessment = true
	
	reader := readers.NewAdvancedMemReader(config, nil, nil, nil)
	defer reader.Close()
	
	// Create memories with varying quality
	memories := createQualityTestMemories()
	
	fmt.Printf("Memory Quality Assessment:\n")
	
	for i, memory := range memories {
		quality, err := reader.AssessQuality(ctx, memory)
		if err != nil {
			log.Printf("Failed to assess quality for memory %d: %v", i, err)
			continue
		}
		
		fmt.Printf("\n  Memory %d (%s):\n", i+1, memory.GetID())
		fmt.Printf("    Overall Score: %.2f\n", quality.Score)
		fmt.Printf("    Dimensions:\n")
		
		for dimension, score := range quality.Dimensions {
			fmt.Printf("      %s: %.2f\n", dimension, score)
		}
		
		if len(quality.Issues) > 0 {
			fmt.Printf("    Issues:\n")
			for _, issue := range quality.Issues {
				fmt.Printf("      - %s (%s): %s\n", 
					issue.Type, issue.Severity, issue.Description)
			}
		}
		
		if len(quality.Suggestions) > 0 {
			fmt.Printf("    Suggestions:\n")
			for _, suggestion := range quality.Suggestions {
				fmt.Printf("      - %s\n", suggestion)
			}
		}
	}
}

// ExampleMemorySummarization demonstrates different summarization strategies
func ExampleMemorySummarization() {
	ctx := context.Background()
	
	reader := readers.NewAdvancedMemReader(readers.DefaultReaderConfig(), nil, nil, nil)
	defer reader.Close()
	
	memories := createLongFormMemories()
	
	// Test different summarization strategies
	strategies := []string{"extractive", "abstractive", "hybrid"}
	
	for _, strategy := range strategies {
		config := &readers.SummarizationConfig{
			MaxLength:        200,
			MinLength:        50,
			Strategy:         strategy,
			KeyPoints:        3,
			IncludeThemes:    true,
			IncludeEntities:  true,
			IncludeSentiment: true,
		}
		
		summary, err := reader.SummarizeMemories(ctx, memories, config)
		if err != nil {
			log.Printf("Failed to create %s summary: %v", strategy, err)
			continue
		}
		
		fmt.Printf("\n%s Summary:\n", strategy)
		fmt.Printf("  Summary: %s\n", summary.Summary)
		fmt.Printf("  Key Points: %v\n", summary.KeyPoints)
		fmt.Printf("  Themes: %v\n", summary.Themes)
		fmt.Printf("  Entities: %v\n", summary.Entities)
		fmt.Printf("  Complexity: %.2f\n", summary.Complexity)
		fmt.Printf("  Coherence: %.2f\n", summary.Coherence)
		fmt.Printf("  Completeness: %.2f\n", summary.Completeness)
		
		if summary.Sentiment != nil {
			fmt.Printf("  Overall Sentiment: %s (%.2f)\n", 
				summary.Sentiment.Emotion, summary.Sentiment.Polarity)
		}
	}
}

// ExampleDuplicateDetection demonstrates duplicate memory detection
func ExampleDuplicateDetection() {
	ctx := context.Background()
	
	reader := readers.NewAdvancedMemReader(readers.DefaultReaderConfig(), nil, nil, nil)
	defer reader.Close()
	
	// Create memories with duplicates and near-duplicates
	memories := createDuplicateTestMemories()
	
	// Test different similarity thresholds
	thresholds := []float64{0.9, 0.8, 0.7, 0.6}
	
	for _, threshold := range thresholds {
		duplicates, err := reader.DetectDuplicates(ctx, memories, threshold)
		if err != nil {
			log.Printf("Failed to detect duplicates at threshold %.2f: %v", threshold, err)
			continue
		}
		
		fmt.Printf("\nDuplicate Detection (Threshold: %.2f):\n", threshold)
		fmt.Printf("  Total Duplicate Groups: %d\n", duplicates.TotalDuplicates)
		fmt.Printf("  Detection Method: %s\n", duplicates.Method)
		
		for i, group := range duplicates.Duplicates {
			fmt.Printf("\n  Group %d:\n", i+1)
			fmt.Printf("    Memory IDs: %v\n", group.MemoryIDs)
			fmt.Printf("    Similarity: %.2f\n", group.Similarity)
			fmt.Printf("    Confidence: %.2f\n", group.Confidence)
			fmt.Printf("    Recommended: %s\n", group.Recommended)
		}
	}
}

// ExampleConfigurationProfiles demonstrates different reader configurations
func ExampleConfigurationProfiles() {
	ctx := context.Background()
	
	configManager := readers.NewConfigManager("")
	factory := readers.NewReaderFactory(nil)
	
	// Test different configuration profiles
	profiles := configManager.GetAvailableProfiles()
	
	fmt.Printf("Testing Different Configuration Profiles:\n")
	
	for _, profile := range profiles {
		config := configManager.CreateProfiledConfig(profile)
		
		fmt.Printf("\n  Profile: %s\n", profile)
		fmt.Printf("    Strategy: %s\n", config.Strategy)
		fmt.Printf("    Analysis Depth: %s\n", config.AnalysisDepth)
		fmt.Printf("    Pattern Detection: %t\n", config.PatternDetection)
		fmt.Printf("    Quality Assessment: %t\n", config.QualityAssessment)
		fmt.Printf("    Duplicate Detection: %t\n", config.DuplicateDetection)
		fmt.Printf("    Sentiment Analysis: %t\n", config.SentimentAnalysis)
		fmt.Printf("    Summarization: %t\n", config.Summarization)
		
		// Create reader with this profile
		factory.SetConfig(profile, config)
		reader, err := factory.CreateReaderFromConfig()
		if err != nil {
			log.Printf("Failed to create reader for profile %s: %v", profile, err)
			continue
		}
		
		// Test basic functionality
		memories := createSampleMemories()
		query := &readers.ReadQuery{
			Query:  "test query",
			TopK:   5,
			UserID: "test_user",
		}
		
		result, err := reader.Read(ctx, query)
		if err != nil {
			log.Printf("Failed to read with profile %s: %v", profile, err)
		} else {
			fmt.Printf("    Processed %d memories in %v\n", 
				len(result.Memories), result.ProcessTime)
		}
		
		reader.Close()
	}
}

// ExampleReaderFactory demonstrates factory pattern usage
func ExampleReaderFactory() {
	// Create factory with default configuration
	factory := readers.CreateDefaultFactory()
	
	// Get available reader types
	types := factory.GetAvailableReaderTypes()
	fmt.Printf("Available Reader Types: %v\n", types)
	
	// Test each reader type
	for _, readerType := range types {
		fmt.Printf("\nTesting Reader Type: %s\n", readerType)
		
		// Get capabilities
		capabilities, err := factory.GetReaderCapabilities(readerType)
		if err != nil {
			log.Printf("Failed to get capabilities: %v", err)
			continue
		}
		
		fmt.Printf("  Capabilities:\n")
		fmt.Printf("    Pattern Detection: %t\n", capabilities.SupportsPatternDetection)
		fmt.Printf("    Semantic Search: %t\n", capabilities.SupportsSemanticSearch)
		fmt.Printf("    Advanced Analytics: %t\n", capabilities.SupportsAdvancedAnalytics)
		fmt.Printf("    Clustering: %t\n", capabilities.SupportsClustering)
		fmt.Printf("    Anomaly Detection: %t\n", capabilities.SupportsAnomalyDetection)
		fmt.Printf("    Max Memories: %d\n", capabilities.MaxMemoriesPerQuery)
		fmt.Printf("    Performance Level: %s\n", capabilities.PerformanceLevel)
		fmt.Printf("    Accuracy Level: %s\n", capabilities.AccuracyLevel)
		
		// Create reader
		reader, err := factory.CreateReader(readerType)
		if err != nil {
			log.Printf("Failed to create reader: %v", err)
			continue
		}
		
		fmt.Printf("  Reader created successfully\n")
		reader.Close()
	}
	
	// Test reader recommendation
	requirements := &readers.ReaderRequirements{
		NeedsSemanticSearch:    false,
		NeedsAdvancedAnalytics: true,
		NeedsClustering:        false,
		MaxMemoryCount:         1000,
		PerformancePriority:    8,
		AccuracyPriority:       6,
		ResourceConstraints:    true,
	}
	
	recommendedType, err := factory.RecommendReaderType(requirements)
	if err != nil {
		log.Printf("Failed to recommend reader: %v", err)
	} else {
		fmt.Printf("\nRecommended Reader Type: %s\n", recommendedType)
		
		// Create auto reader
		reader, err := factory.CreateAutoReader(requirements)
		if err != nil {
			log.Printf("Failed to create auto reader: %v", err)
		} else {
			fmt.Printf("Auto reader created successfully\n")
			reader.Close()
		}
	}
}

// Helper functions to create test data

func createSampleMemories() []types.MemoryItem {
	now := time.Now()
	
	return []types.MemoryItem{
		&types.TextualMemoryItem{
			ID:        "sample1",
			Memory:    "This is a comprehensive guide to machine learning algorithms. It covers supervised learning, unsupervised learning, and reinforcement learning techniques with practical examples.",
			Metadata:  map[string]interface{}{"category": "education", "subject": "ai"},
			CreatedAt: now.Add(-2 * time.Hour),
			UpdatedAt: now.Add(-2 * time.Hour),
		},
		&types.TextualMemoryItem{
			ID:        "sample2",
			Memory:    "Market analysis shows strong growth in technology sector. Companies are investing heavily in artificial intelligence and machine learning solutions.",
			Metadata:  map[string]interface{}{"category": "business", "subject": "market"},
			CreatedAt: now.Add(-1 * time.Hour),
			UpdatedAt: now.Add(-1 * time.Hour),
		},
		&types.TextualMemoryItem{
			ID:        "sample3",
			Memory:    "Personal reflection on my learning journey. I feel excited about the progress I've made in understanding complex algorithms and data structures.",
			Metadata:  map[string]interface{}{"category": "personal", "subject": "reflection"},
			CreatedAt: now.Add(-30 * time.Minute),
			UpdatedAt: now.Add(-30 * time.Minute),
		},
	}
}

func createPatternMemories() []types.MemoryItem {
	now := time.Now()
	
	return []types.MemoryItem{
		&types.TextualMemoryItem{
			ID:        "pattern1",
			Memory:    "What are neural networks? How do they learn? Can we improve their performance?",
			Metadata:  map[string]interface{}{"pattern_type": "questions"},
			CreatedAt: now.Add(-3 * time.Hour),
			UpdatedAt: now.Add(-3 * time.Hour),
		},
		&types.TextualMemoryItem{
			ID:        "pattern2",
			Memory:    "Neural networks are powerful computational models. They mimic biological neural systems. We use them for pattern recognition.",
			Metadata:  map[string]interface{}{"pattern_type": "statements"},
			CreatedAt: now.Add(-2 * time.Hour),
			UpdatedAt: now.Add(-2 * time.Hour),
		},
		&types.TextualMemoryItem{
			ID:        "pattern3",
			Memory:    "Amazing breakthrough in AI research! Incredible results achieved! Outstanding performance metrics!",
			Metadata:  map[string]interface{}{"pattern_type": "exclamations"},
			CreatedAt: now.Add(-1 * time.Hour),
			UpdatedAt: now.Add(-1 * time.Hour),
		},
		&types.TextualMemoryItem{
			ID:        "pattern4",
			Memory:    "Deep learning components:\n- Neural layers\n- Activation functions\n- Backpropagation\n- Gradient descent\n- Loss functions",
			Metadata:  map[string]interface{}{"pattern_type": "structured_list"},
			CreatedAt: now.Add(-30 * time.Minute),
			UpdatedAt: now.Add(-30 * time.Minute),
		},
	}
}

func createQualityTestMemories() []types.MemoryItem {
	now := time.Now()
	
	return []types.MemoryItem{
		&types.TextualMemoryItem{
			ID:        "quality_high",
			Memory:    "This is a well-structured document with clear paragraphs, proper punctuation, and comprehensive coverage of the topic. It demonstrates good organization and readability.",
			Metadata:  map[string]interface{}{"expected_quality": "high"},
			CreatedAt: now.Add(-1 * time.Hour),
			UpdatedAt: now.Add(-1 * time.Hour),
		},
		&types.TextualMemoryItem{
			ID:        "quality_medium",
			Memory:    "decent content but lacking structure some punctuation issues and could use better organization",
			Metadata:  map[string]interface{}{"expected_quality": "medium"},
			CreatedAt: now.Add(-30 * time.Minute),
			UpdatedAt: now.Add(-30 * time.Minute),
		},
		&types.TextualMemoryItem{
			ID:        "quality_low",
			Memory:    "short",
			Metadata:  map[string]interface{}{"expected_quality": "low"},
			CreatedAt: now.Add(-10 * time.Minute),
			UpdatedAt: now.Add(-10 * time.Minute),
		},
	}
}

func createLongFormMemories() []types.MemoryItem {
	now := time.Now()
	
	return []types.MemoryItem{
		&types.TextualMemoryItem{
			ID:        "long1",
			Memory:    "Artificial intelligence has revolutionized numerous industries and continues to shape our technological landscape. Machine learning algorithms enable computers to learn from data without explicit programming. Deep learning, a subset of machine learning, uses neural networks with multiple layers to process complex patterns. Natural language processing allows machines to understand and generate human language. Computer vision enables systems to interpret visual information. These technologies are transforming healthcare, finance, transportation, and entertainment industries.",
			Metadata:  map[string]interface{}{"type": "technology"},
			CreatedAt: now.Add(-2 * time.Hour),
			UpdatedAt: now.Add(-2 * time.Hour),
		},
		&types.TextualMemoryItem{
			ID:        "long2",
			Memory:    "The future of artificial intelligence holds immense promise and challenges. Ethical considerations include algorithmic bias, privacy concerns, and job displacement. Responsible AI development requires transparency, fairness, and accountability. Collaboration between technologists, policymakers, and society is essential. Education and reskilling programs will help workers adapt to AI-driven changes. International cooperation is needed to establish AI governance frameworks.",
			Metadata:  map[string]interface{}{"type": "ethics"},
			CreatedAt: now.Add(-1 * time.Hour),
			UpdatedAt: now.Add(-1 * time.Hour),
		},
	}
}

func createDuplicateTestMemories() []types.MemoryItem {
	now := time.Now()
	
	return []types.MemoryItem{
		&types.TextualMemoryItem{
			ID:        "original",
			Memory:    "Machine learning is a subset of artificial intelligence that enables computers to learn from data.",
			Metadata:  map[string]interface{}{"group": "ml_definition"},
			CreatedAt: now.Add(-3 * time.Hour),
			UpdatedAt: now.Add(-3 * time.Hour),
		},
		&types.TextualMemoryItem{
			ID:        "exact_duplicate",
			Memory:    "Machine learning is a subset of artificial intelligence that enables computers to learn from data.",
			Metadata:  map[string]interface{}{"group": "ml_definition"},
			CreatedAt: now.Add(-2 * time.Hour),
			UpdatedAt: now.Add(-2 * time.Hour),
		},
		&types.TextualMemoryItem{
			ID:        "near_duplicate",
			Memory:    "Machine learning is a subset of AI that allows computers to learn from data.",
			Metadata:  map[string]interface{}{"group": "ml_definition"},
			CreatedAt: now.Add(-1 * time.Hour),
			UpdatedAt: now.Add(-1 * time.Hour),
		},
		&types.TextualMemoryItem{
			ID:        "similar",
			Memory:    "ML enables systems to automatically learn and improve from experience without being explicitly programmed.",
			Metadata:  map[string]interface{}{"group": "ml_definition"},
			CreatedAt: now.Add(-30 * time.Minute),
			UpdatedAt: now.Add(-30 * time.Minute),
		},
		&types.TextualMemoryItem{
			ID:        "different",
			Memory:    "Deep learning uses neural networks with multiple layers to process complex patterns in data.",
			Metadata:  map[string]interface{}{"group": "dl_definition"},
			CreatedAt: now.Add(-10 * time.Minute),
			UpdatedAt: now.Add(-10 * time.Minute),
		},
	}
}