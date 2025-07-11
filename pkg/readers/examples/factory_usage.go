package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/memtensor/memgos/pkg/readers"
)

func factoryMain() {
	fmt.Println("MemGOS Memory Reader - Factory Usage Examples")
	fmt.Println("==============================================")

	// Example 1: Basic Factory Usage
	basicFactoryExample()

	// Example 2: Reader Recommendation System
	readerRecommendationExample()

	// Example 3: Configuration Management
	configurationExample()

	// Example 4: Multi-Reader Strategy
	multiReaderExample()

	// Example 5: Reader Pool for Concurrency
	readerPoolExample()
}

func basicFactoryExample() {
	fmt.Println("\n1. Basic Factory Usage")
	fmt.Println("----------------------")

	// Create a factory with default configuration
	factory := readers.CreateDefaultFactory()

	// Get available reader types
	availableTypes := factory.GetAvailableReaderTypes()
	fmt.Printf("Available reader types: %v\n", availableTypes)

	// Create different types of readers
	for _, readerType := range availableTypes {
		fmt.Printf("\nCreating %s reader:\n", readerType)

		reader, err := factory.CreateReader(readerType)
		if err != nil {
			log.Printf("Failed to create %s reader: %v", readerType, err)
			continue
		}

		// Get reader capabilities
		capabilities, err := factory.GetReaderCapabilities(readerType)
		if err != nil {
			log.Printf("Failed to get capabilities: %v", err)
		} else {
			fmt.Printf("  Max memories per query: %d\n", capabilities.MaxMemoriesPerQuery)
			fmt.Printf("  Performance level: %s\n", capabilities.PerformanceLevel)
			fmt.Printf("  Accuracy level: %s\n", capabilities.AccuracyLevel)
			fmt.Printf("  Supports semantic search: %v\n", capabilities.SupportsSemanticSearch)
			fmt.Printf("  Supports clustering: %v\n", capabilities.SupportsClustering)
		}

		reader.Close()
	}
}

func readerRecommendationExample() {
	fmt.Println("\n2. Reader Recommendation System")
	fmt.Println("-------------------------------")

	factory := readers.CreateDefaultFactory()

	// Define different use case requirements
	useCases := []struct {
		name         string
		requirements *readers.ReaderRequirements
	}{
		{
			name: "Real-time Processing",
			requirements: &readers.ReaderRequirements{
				NeedsSemanticSearch:     false,
				NeedsAdvancedAnalytics:  false,
				MaxMemoryCount:          100,
				PerformancePriority:     9, // High performance priority
				AccuracyPriority:        3, // Low accuracy priority
				ResourceConstraints:     true,
			},
		},
		{
			name: "Research Analysis",
			requirements: &readers.ReaderRequirements{
				NeedsSemanticSearch:     true,
				NeedsAdvancedAnalytics:  true,
				NeedsClustering:         true,
				NeedsAnomalyDetection:   true,
				MaxMemoryCount:          5000,
				PerformancePriority:     3, // Low performance priority
				AccuracyPriority:        9, // High accuracy priority
				ResourceConstraints:     false,
			},
		},
		{
			name: "Balanced Processing",
			requirements: &readers.ReaderRequirements{
				NeedsSemanticSearch:     false,
				NeedsAdvancedAnalytics:  true,
				MaxMemoryCount:          1000,
				PerformancePriority:     5, // Medium performance priority
				AccuracyPriority:        6, // Medium accuracy priority
				ResourceConstraints:     false,
			},
		},
	}

	for _, useCase := range useCases {
		fmt.Printf("\nUse Case: %s\n", useCase.name)

		// Get recommendation
		recommendedType, err := factory.RecommendReaderType(useCase.requirements)
		if err != nil {
			log.Printf("Failed to get recommendation: %v", err)
			continue
		}

		fmt.Printf("  Recommended reader: %s\n", recommendedType)

		// Create the recommended reader
		reader, err := factory.CreateAutoReader(useCase.requirements)
		if err != nil {
			log.Printf("Failed to create auto reader: %v", err)
			continue
		}

		// Validate the recommendation
		capabilities, _ := factory.GetReaderCapabilities(recommendedType)
		fmt.Printf("  Performance level: %s\n", capabilities.PerformanceLevel)
		fmt.Printf("  Accuracy level: %s\n", capabilities.AccuracyLevel)
		fmt.Printf("  Can handle %d memories: %v\n",
			useCase.requirements.MaxMemoryCount,
			useCase.requirements.MaxMemoryCount <= capabilities.MaxMemoriesPerQuery)

		reader.Close()
	}
}

func configurationExample() {
	fmt.Println("\n3. Configuration Management")
	fmt.Println("---------------------------")

	// Create configuration manager
	cm := readers.NewConfigManager("")

	// Get available profiles
	profiles := cm.GetAvailableProfiles()
	fmt.Printf("Available configuration profiles: %v\n", profiles)

	// Demonstrate different profiles
	profileExamples := []readers.ConfigProfile{
		readers.ProfileHighPerformance,
		readers.ProfileHighAccuracy,
		readers.ProfileBalanced,
		readers.ProfileLowResource,
	}

	for _, profile := range profileExamples {
		fmt.Printf("\nProfile: %s\n", profile)

		config := cm.CreateProfiledConfig(profile)
		fmt.Printf("  Strategy: %s\n", config.Strategy)
		fmt.Printf("  Analysis depth: %s\n", config.AnalysisDepth)
		fmt.Printf("  Pattern detection: %v\n", config.PatternDetection)
		fmt.Printf("  Quality assessment: %v\n", config.QualityAssessment)

		// Create reader with this profile
		factory := readers.CreateFactoryFromConfig(config)
		reader, err := factory.CreateReaderFromConfig()
		if err != nil {
			log.Printf("Failed to create reader from config: %v", err)
			continue
		}

		fmt.Printf("  Created reader successfully\n")
		reader.Close()
	}

	// Custom configuration example
	fmt.Printf("\nCustom Configuration:\n")
	customOptions := map[string]interface{}{
		"strategy":             "advanced",
		"analysis_depth":       "deep",
		"pattern_detection":    true,
		"quality_assessment":   true,
		"duplicate_detection":  false,
		"sentiment_analysis":   true,
		"summarization":        true,
	}

	customConfig, err := cm.CreateCustomConfig(customOptions)
	if err != nil {
		log.Printf("Failed to create custom config: %v", err)
	} else {
		fmt.Printf("  Strategy: %s\n", customConfig.Strategy)
		fmt.Printf("  Analysis depth: %s\n", customConfig.AnalysisDepth)
		fmt.Printf("  Duplicate detection: %v\n", customConfig.DuplicateDetection)
	}
}

func multiReaderExample() {
	fmt.Println("\n4. Multi-Reader Strategy")
	fmt.Println("------------------------")

	factory := readers.CreateDefaultFactory()

	// Create multi-reader strategy for optimal processing
	strategy := readers.NewMultiReaderStrategy(factory)

	// Add readers with different weights
	err := strategy.AddReader("fast", readers.ReaderTypeSimple, 0.3)
	if err != nil {
		log.Printf("Failed to add fast reader: %v", err)
		return
	}

	err = strategy.AddReader("accurate", readers.ReaderTypeAdvanced, 0.7)
	if err != nil {
		log.Printf("Failed to add accurate reader: %v", err)
		return
	}

	// Get all readers and weights
	readers := strategy.GetReaders()
	weights := strategy.GetWeights()

	fmt.Printf("Multi-reader strategy configured:\n")
	for name, reader := range readers {
		weight := weights[name]
		fmt.Printf("  %s reader: weight %.1f\n", name, weight)

		// Demonstrate usage
		config := reader.GetConfiguration()
		fmt.Printf("    Strategy: %s\n", config.Strategy)
		fmt.Printf("    Analysis depth: %s\n", config.AnalysisDepth)
	}

	// Simulate processing with different readers based on requirements
	fmt.Printf("\nProcessing scenarios:\n")

	scenarios := []struct {
		name        string
		readerName  string
		description string
	}{
		{"Quick scan", "fast", "Use fast reader for initial filtering"},
		{"Deep analysis", "accurate", "Use accurate reader for detailed analysis"},
	}

	for _, scenario := range scenarios {
		fmt.Printf("  %s: %s\n", scenario.name, scenario.description)
		reader := readers[scenario.readerName]

		// Mock query processing
		mockQuery := &readers.ReadQuery{
			Query: "test query",
			TopK:  10,
		}

		ctx := context.Background()
		startTime := time.Now()
		result, err := reader.Read(ctx, mockQuery)
		processingTime := time.Since(startTime)

		if err != nil {
			log.Printf("    Processing failed: %v", err)
		} else {
			fmt.Printf("    Processed %d memories in %v\n",
				result.TotalFound, processingTime)
		}
	}

	strategy.Close()
}

func readerPoolExample() {
	fmt.Println("\n5. Reader Pool for Concurrency")
	fmt.Println("------------------------------")

	factory := readers.CreateDefaultFactory()

	// Create a reader pool for concurrent processing
	poolSize := 3
	pool := readers.NewReaderPool(factory, readers.ReaderTypeSimple, poolSize)

	fmt.Printf("Created reader pool with %d readers\n", poolSize)

	// Simulate concurrent processing
	tasks := []string{
		"Process user queries",
		"Analyze system logs",
		"Extract document insights",
		"Quality assessment batch",
		"Pattern detection job",
	}

	fmt.Printf("Processing %d concurrent tasks:\n", len(tasks))

	// Channel to collect results
	results := make(chan string, len(tasks))

	// Start concurrent workers
	for i, task := range tasks {
		go func(taskID int, taskName string) {
			// Get reader from pool
			reader, err := pool.GetReader()
			if err != nil {
				results <- fmt.Sprintf("Task %d failed to get reader: %v", taskID+1, err)
				return
			}
			defer pool.ReturnReader(reader)

			// Simulate processing
			mockQuery := &readers.ReadQuery{
				Query: taskName,
				TopK:  5,
			}

			ctx := context.Background()
			startTime := time.Now()
			result, err := reader.Read(ctx, mockQuery)
			processingTime := time.Since(startTime)

			if err != nil {
				results <- fmt.Sprintf("Task %d (%s) failed: %v", taskID+1, taskName, err)
			} else {
				results <- fmt.Sprintf("Task %d (%s): %d memories in %v",
					taskID+1, taskName, result.TotalFound, processingTime)
			}
		}(i, task)
	}

	// Collect and display results
	for i := 0; i < len(tasks); i++ {
		result := <-results
		fmt.Printf("  %s\n", result)
	}

	// Close pool
	err := pool.Close()
	if err != nil {
		log.Printf("Failed to close pool: %v", err)
	} else {
		fmt.Printf("Reader pool closed successfully\n")
	}
}