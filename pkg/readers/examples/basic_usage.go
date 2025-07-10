package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/memtensor/memgos/pkg/readers"
	"github.com/memtensor/memgos/pkg/types"
)

// MockMemory implements types.MemoryItem for demonstration
type MockMemory struct {
	ID        string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (m *MockMemory) GetID() string                        { return m.ID }
func (m *MockMemory) GetContent() string                   { return m.Content }
func (m *MockMemory) GetMetadata() map[string]interface{}  { return make(map[string]interface{}) }
func (m *MockMemory) GetCreatedAt() time.Time              { return m.CreatedAt }
func (m *MockMemory) GetUpdatedAt() time.Time              { return m.UpdatedAt }

func main() {
	fmt.Println("MemGOS Memory Reader - Basic Usage Example")
	fmt.Println("==========================================")

	// Example 1: Simple Memory Reading
	basicReaderExample()

	// Example 2: Memory Analysis
	memoryAnalysisExample()

	// Example 3: Pattern Detection
	patternDetectionExample()

	// Example 4: Quality Assessment
	qualityAssessmentExample()

	// Example 5: Memory Summarization
	summarizationExample()
}

func basicReaderExample() {
	fmt.Println("\n1. Basic Memory Reading")
	fmt.Println("-----------------------")

	// Create a simple reader with default configuration
	reader := readers.NewSimpleMemReader(nil)
	defer reader.Close()

	// Create a read query
	query := &readers.ReadQuery{
		Query: "software development best practices",
		TopK:  5,
		Filters: map[string]string{
			"type": "technical",
		},
	}

	// Perform the read operation
	ctx := context.Background()
	result, err := reader.Read(ctx, query)
	if err != nil {
		log.Fatalf("Read operation failed: %v", err)
	}

	// Display results
	fmt.Printf("Found %d memories in %v\n", result.TotalFound, result.ProcessTime)
	fmt.Printf("Strategy used: %s\n", result.Metadata["strategy"])

	// Show basic analysis if available
	if result.Quality != nil {
		fmt.Printf("Overall quality score: %.2f\n", result.Quality.Score)
	}
}

func memoryAnalysisExample() {
	fmt.Println("\n2. Memory Analysis")
	fmt.Println("------------------")

	// Create an advanced reader for detailed analysis
	config := readers.DefaultReaderConfig()
	config.Strategy = readers.ReadStrategyAdvanced
	config.AnalysisDepth = "deep"

	reader := readers.NewSimpleMemReader(config)
	defer reader.Close()

	// Create sample memories
	memories := []types.MemoryItem{
		&MockMemory{
			ID:        "mem-1",
			Content:   "Software development requires careful planning, systematic testing, and continuous integration to ensure high-quality deliverables.",
			CreatedAt: time.Now().Add(-24 * time.Hour),
			UpdatedAt: time.Now(),
		},
		&MockMemory{
			ID:        "mem-2",
			Content:   "Machine learning algorithms have revolutionized data analysis and predictive modeling in various industries.",
			CreatedAt: time.Now().Add(-12 * time.Hour),
			UpdatedAt: time.Now(),
		},
		&MockMemory{
			ID:        "mem-3",
			Content:   "User experience design focuses on creating intuitive and accessible interfaces that enhance user satisfaction.",
			CreatedAt: time.Now().Add(-6 * time.Hour),
			UpdatedAt: time.Now(),
		},
	}

	ctx := context.Background()

	// Analyze individual memory
	analysis, err := reader.AnalyzeMemory(ctx, memories[0])
	if err != nil {
		log.Printf("Memory analysis failed: %v", err)
		return
	}

	fmt.Printf("Memory Analysis for %s:\n", analysis.MemoryID)
	fmt.Printf("  Type: %s\n", analysis.Type)
	fmt.Printf("  Keywords: %v\n", analysis.Keywords)
	fmt.Printf("  Themes: %v\n", analysis.Themes)
	fmt.Printf("  Complexity: %.2f\n", analysis.Complexity)
	fmt.Printf("  Quality: %.2f\n", analysis.Quality)
	fmt.Printf("  Sentiment: %s (%.2f)\n", analysis.Sentiment.Emotion, analysis.Sentiment.Polarity)

	// Analyze memory collection
	collectionAnalysis, err := reader.AnalyzeMemories(ctx, memories)
	if err != nil {
		log.Printf("Collection analysis failed: %v", err)
		return
	}

	fmt.Printf("\nCollection Analysis:\n")
	fmt.Printf("  Total memories: %d\n", collectionAnalysis.TotalMemories)
	fmt.Printf("  Memory types: %v\n", collectionAnalysis.Types)
	fmt.Printf("  Common themes: %v\n", collectionAnalysis.Themes)
	fmt.Printf("  Average quality: %.2f\n", collectionAnalysis.Statistics.AverageQuality)
	fmt.Printf("  Insights: %v\n", collectionAnalysis.Insights)
}

func patternDetectionExample() {
	fmt.Println("\n3. Pattern Detection")
	fmt.Println("--------------------")

	reader := readers.NewSimpleMemReader(nil)
	defer reader.Close()

	// Create memories with patterns
	memories := []types.MemoryItem{
		&MockMemory{ID: "pat-1", Content: "Testing is essential for software quality assurance and reliability."},
		&MockMemory{ID: "pat-2", Content: "Quality software requires thorough testing and validation procedures."},
		&MockMemory{ID: "pat-3", Content: "Automated testing improves software development efficiency and quality."},
		&MockMemory{ID: "pat-4", Content: "Code review and testing are fundamental practices in software engineering."},
	}

	// Configure pattern detection
	patternConfig := &readers.PatternConfig{
		MinSupport:    0.25, // 25% of memories must contain the pattern
		MinConfidence: 0.6,  // 60% confidence threshold
		MaxPatterns:   20,   // Maximum 20 patterns to return
		PatternTypes:  []string{"frequent", "sequential"},
	}

	ctx := context.Background()
	patterns, err := reader.ExtractPatterns(ctx, memories, patternConfig)
	if err != nil {
		log.Printf("Pattern extraction failed: %v", err)
		return
	}

	fmt.Printf("Pattern Analysis Results:\n")
	fmt.Printf("  Patterns found: %d\n", len(patterns.Patterns))
	fmt.Printf("  Overall confidence: %.2f\n", patterns.Confidence)

	// Display top patterns
	for i, pattern := range patterns.Patterns {
		if i >= 5 { // Show only top 5
			break
		}
		fmt.Printf("  Pattern %d: %s (freq: %d, conf: %.2f)\n",
			i+1, pattern.Description, pattern.Frequency, pattern.Confidence)
	}

	// Display word frequencies
	fmt.Printf("  Top word frequencies:\n")
	count := 0
	for word, freq := range patterns.Frequencies {
		if count >= 5 {
			break
		}
		fmt.Printf("    %s: %d\n", word, freq)
		count++
	}
}

func qualityAssessmentExample() {
	fmt.Println("\n4. Quality Assessment")
	fmt.Println("---------------------")

	reader := readers.NewSimpleMemReader(nil)
	defer reader.Close()

	// Create memories with different quality levels
	memories := []types.MemoryItem{
		&MockMemory{
			ID:      "quality-high",
			Content: "This is a well-structured, comprehensive document about software architecture patterns. It includes detailed explanations, examples, and best practices that provide significant value to developers working on complex systems.",
		},
		&MockMemory{
			ID:      "quality-medium",
			Content: "Software patterns are useful. They help with design and make code better.",
		},
		&MockMemory{
			ID:      "quality-low",
			Content: "Code good.",
		},
	}

	ctx := context.Background()

	fmt.Printf("Quality Assessment Results:\n")

	for _, memory := range memories {
		assessment, err := reader.AssessQuality(ctx, memory)
		if err != nil {
			log.Printf("Quality assessment failed for %s: %v", memory.GetID(), err)
			continue
		}

		fmt.Printf("\nMemory: %s\n", memory.GetID())
		fmt.Printf("  Overall Score: %.2f\n", assessment.Score)
		fmt.Printf("  Dimensions:\n")
		for dimension, score := range assessment.Dimensions {
			fmt.Printf("    %s: %.2f\n", dimension, score)
		}

		if len(assessment.Issues) > 0 {
			fmt.Printf("  Issues:\n")
			for _, issue := range assessment.Issues {
				fmt.Printf("    %s (%s): %s\n", issue.Type, issue.Severity, issue.Description)
			}
		}

		if len(assessment.Suggestions) > 0 {
			fmt.Printf("  Suggestions:\n")
			for _, suggestion := range assessment.Suggestions {
				fmt.Printf("    - %s\n", suggestion)
			}
		}
	}
}

func summarizationExample() {
	fmt.Println("\n5. Memory Summarization")
	fmt.Println("-----------------------")

	reader := readers.NewSimpleMemReader(nil)
	defer reader.Close()

	// Create a collection of related memories
	memories := []types.MemoryItem{
		&MockMemory{
			ID: "sum-1",
			Content: "Microservices architecture is a design approach where applications are built as a collection of loosely coupled services. Each service is independently deployable and scalable.",
		},
		&MockMemory{
			ID: "sum-2",
			Content: "The main benefits of microservices include improved fault isolation, technology diversity, and easier scaling of individual components based on demand.",
		},
		&MockMemory{
			ID: "sum-3",
			Content: "However, microservices also introduce complexity in terms of service communication, data consistency, and operational overhead requiring robust monitoring and deployment strategies.",
		},
		&MockMemory{
			ID: "sum-4",
			Content: "Container technologies like Docker and orchestration platforms like Kubernetes have made microservices deployment and management more practical and efficient.",
		},
	}

	// Configure summarization
	summaryConfig := &readers.SummarizationConfig{
		MaxLength:        300,
		MinLength:        100,
		Strategy:         "extractive",
		KeyPoints:        3,
		IncludeThemes:    true,
		IncludeEntities:  true,
		IncludeSentiment: true,
	}

	ctx := context.Background()
	summary, err := reader.SummarizeMemories(ctx, memories, summaryConfig)
	if err != nil {
		log.Printf("Summarization failed: %v", err)
		return
	}

	fmt.Printf("Memory Collection Summary:\n")
	fmt.Printf("  Summary: %s\n", summary.Summary)
	fmt.Printf("  Key Points:\n")
	for i, point := range summary.KeyPoints {
		fmt.Printf("    %d. %s\n", i+1, point)
	}
	fmt.Printf("  Themes: %v\n", summary.Themes)
	fmt.Printf("  Entities: %v\n", summary.Entities)
	fmt.Printf("  Complexity: %.2f\n", summary.Complexity)
	fmt.Printf("  Coherence: %.2f\n", summary.Coherence)
	fmt.Printf("  Completeness: %.2f\n", summary.Completeness)

	if summary.Sentiment != nil {
		fmt.Printf("  Overall Sentiment: %s (%.2f)\n",
			summary.Sentiment.Emotion, summary.Sentiment.Polarity)
	}

	fmt.Printf("  Generated: %s\n", summary.GeneratedAt.Format(time.RFC3339))
}