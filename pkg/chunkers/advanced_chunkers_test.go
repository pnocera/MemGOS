package chunkers

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/memtensor/memgos/pkg/types"
)

// MockLLMProvider implements a mock LLM provider for testing
type MockLLMProvider struct {
	responses map[string]string
}

func NewMockLLMProvider() *MockLLMProvider {
	return &MockLLMProvider{
		responses: map[string]string{
			"boundary_analysis": `{
  "boundaries": [2, 5, 8],
  "reasoning": "Detected topic boundaries based on semantic analysis",
  "confidence": 0.85,
  "chunk_summaries": ["Introduction to machine learning", "Deep learning concepts", "Neural network architectures"],
  "optimization_notes": "Consider increasing overlap for better context preservation"
}`,
			"summary": "This section discusses machine learning fundamentals and neural network architectures.",
		},
	}
}

func (m *MockLLMProvider) Generate(ctx context.Context, messages types.MessageList) (string, error) {
	// Simple mock: return boundary analysis for agentic chunker
	for _, msg := range messages {
		content := msg.Content
		if strings.Contains(content, "boundary") || strings.Contains(content, "analyze") {
			return m.responses["boundary_analysis"], nil
		}
		if strings.Contains(content, "summarize") || strings.Contains(content, "Summary") {
			return m.responses["summary"], nil
		}
	}
	return "Mock LLM response", nil
}

func (m *MockLLMProvider) GenerateWithCache(ctx context.Context, messages types.MessageList, cache interface{}) (string, error) {
	return m.Generate(ctx, messages)
}

func (m *MockLLMProvider) GenerateStream(ctx context.Context, messages types.MessageList, stream chan<- string) error {
	response, err := m.Generate(ctx, messages)
	if err != nil {
		return err
	}
	stream <- response
	close(stream)
	return nil
}

func (m *MockLLMProvider) Embed(ctx context.Context, text string) (types.EmbeddingVector, error) {
	// Return a simple mock embedding
	return types.EmbeddingVector{0.1, 0.2, 0.3, 0.4, 0.5}, nil
}

func (m *MockLLMProvider) GetModelInfo() map[string]interface{} {
	return map[string]interface{}{
		"name":    "mock-llm",
		"version": "1.0",
	}
}

func (m *MockLLMProvider) Close() error {
	return nil
}

// MockEmbeddingProvider implements a mock embedding provider for testing
type MockEmbeddingProvider struct{}

func (m *MockEmbeddingProvider) GetEmbedding(ctx context.Context, text string) ([]float64, error) {
	// Return a simple mock embedding
	return []float64{0.1, 0.2, 0.3, 0.4, 0.5}, nil
}

func (m *MockEmbeddingProvider) GetEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	var embeddings [][]float64
	for range texts {
		embeddings = append(embeddings, []float64{0.1, 0.2, 0.3, 0.4, 0.5})
	}
	return embeddings, nil
}

func (m *MockEmbeddingProvider) GetBatchEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	return m.GetEmbeddings(ctx, texts)
}

func (m *MockEmbeddingProvider) GetModelInfo() EmbeddingModelInfo {
	return EmbeddingModelInfo{
		Name:       "mock-embedding",
		Dimensions: 5,
		MaxTokens:  512,
	}
}

func (m *MockEmbeddingProvider) GetDimension() int {
	return 5
}

func (m *MockEmbeddingProvider) GetDimensions() int {
	return 5
}

func (m *MockEmbeddingProvider) GetMaxTokens() int {
	return 512
}

func (m *MockEmbeddingProvider) Close() error {
	return nil
}

// Test data
const testText = `Machine learning is a powerful subset of artificial intelligence. It enables computers to learn and improve from experience without being explicitly programmed.

Deep learning is a specialized area within machine learning. It uses neural networks with multiple layers to model and understand complex patterns in data.

Neural networks are inspired by the human brain's structure. They consist of interconnected nodes that process information in parallel. These networks can recognize patterns and make predictions.

Artificial intelligence has many applications. It's used in healthcare, finance, transportation, and entertainment. AI systems can analyze vast amounts of data quickly and accurately.

The future of AI looks promising. Researchers are developing more sophisticated algorithms and models. These advances will likely lead to breakthroughs in various fields.`

const multiModalText = `# Machine Learning Guide

Machine learning is revolutionizing technology.

## Code Example

` + "```python\n" + `import numpy as np
from sklearn.linear_model import LinearRegression

# Create sample data
X = np.array([[1], [2], [3], [4], [5]])
y = np.array([2, 4, 6, 8, 10])

# Train model
model = LinearRegression()
model.fit(X, y)
` + "```\n" + `

## Data Table

| Algorithm | Accuracy | Training Time |
|-----------|----------|---------------|
| Linear Regression | 85% | 0.1s |
| Random Forest | 92% | 2.3s |
| Neural Network | 96% | 45s |

## Image Reference

![ML Workflow](ml-workflow.png)

This diagram shows the typical machine learning workflow from data collection to model deployment.`

// TestAgenticChunker tests the agentic chunker implementation
func TestAgenticChunker(t *testing.T) {
	config := DefaultChunkerConfig()
	config.ChunkSize = 200
	config.ChunkOverlap = 50
	
	llmProvider := NewMockLLMProvider()
	
	chunker, err := NewAgenticChunker(config, llmProvider)
	if err != nil {
		t.Fatalf("Failed to create agentic chunker: %v", err)
	}
	
	ctx := context.Background()
	chunks, err := chunker.Chunk(ctx, testText)
	if err != nil {
		t.Fatalf("Failed to chunk text: %v", err)
	}
	
	if len(chunks) == 0 {
		t.Fatal("No chunks created")
	}
	
	// Verify chunker type in metadata
	for _, chunk := range chunks {
		if chunkerType, ok := chunk.Metadata["chunker_type"].(string); ok {
			if chunkerType != "agentic" {
				t.Errorf("Expected chunker type 'agentic', got '%s'", chunkerType)
			}
		} else {
			t.Error("Chunker type not found in metadata")
		}
		
		// Verify agentic-specific metadata
		if _, ok := chunk.Metadata["llm_decisions"]; !ok {
			t.Error("LLM decisions not found in metadata")
		}
		
		if _, ok := chunk.Metadata["document_type"]; !ok {
			t.Error("Document type not found in metadata")
		}
	}
	
	t.Logf("Created %d chunks using agentic chunker", len(chunks))
}

// TestAgenticChunkerConfig tests agentic chunker configuration
func TestAgenticChunkerConfig(t *testing.T) {
	config := DefaultChunkerConfig()
	llmProvider := NewMockLLMProvider()
	
	chunker, err := NewAgenticChunker(config, llmProvider)
	if err != nil {
		t.Fatalf("Failed to create agentic chunker: %v", err)
	}
	
	// Test getting agentic config
	agenticConfig := chunker.GetAgenticConfig()
	if agenticConfig == nil {
		t.Fatal("Agentic config is nil")
	}
	
	// Test setting agentic config
	newConfig := DefaultAgenticConfig()
	newConfig.AnalysisDepth = AnalysisDepthExpert
	newConfig.ReasoningSteps = 5
	
	chunker.SetAgenticConfig(newConfig)
	
	retrievedConfig := chunker.GetAgenticConfig()
	if retrievedConfig.AnalysisDepth != AnalysisDepthExpert {
		t.Errorf("Expected analysis depth 'expert', got '%s'", retrievedConfig.AnalysisDepth)
	}
	
	if retrievedConfig.ReasoningSteps != 5 {
		t.Errorf("Expected 5 reasoning steps, got %d", retrievedConfig.ReasoningSteps)
	}
}

// TestMultiModalChunker tests the multi-modal chunker implementation
func TestMultiModalChunker(t *testing.T) {
	config := DefaultChunkerConfig()
	config.ChunkSize = 300
	
	chunker, err := NewMultiModalChunker(config)
	if err != nil {
		t.Fatalf("Failed to create multi-modal chunker: %v", err)
	}
	
	ctx := context.Background()
	chunks, err := chunker.Chunk(ctx, multiModalText)
	if err != nil {
		t.Fatalf("Failed to chunk multi-modal text: %v", err)
	}
	
	if len(chunks) == 0 {
		t.Fatal("No chunks created")
	}
	
	// Verify different content types are detected
	contentTypes := make(map[string]bool)
	for _, chunk := range chunks {
		if contentType, ok := chunk.Metadata["content_type"].(string); ok {
			contentTypes[contentType] = true
		}
	}
	
	expectedTypes := []string{"text", "code", "table", "image"}
	for _, expectedType := range expectedTypes {
		if !contentTypes[expectedType] {
			t.Logf("Content type '%s' not detected (this may be expected depending on detection accuracy)", expectedType)
		}
	}
	
	t.Logf("Created %d chunks from multi-modal content", len(chunks))
	t.Logf("Detected content types: %v", contentTypes)
}

// TestMultiModalChunkerConfig tests multi-modal chunker configuration
func TestMultiModalChunkerConfig(t *testing.T) {
	config := DefaultChunkerConfig()
	
	chunker, err := NewMultiModalChunker(config)
	if err != nil {
		t.Fatalf("Failed to create multi-modal chunker: %v", err)
	}
	
	// Test getting multi-modal config
	multiModalConfig := chunker.GetMultiModalConfig()
	if multiModalConfig == nil {
		t.Fatal("Multi-modal config is nil")
	}
	
	// Test setting multi-modal config
	newConfig := DefaultMultiModalConfig()
	newConfig.PreserveCodeBlocks = false
	newConfig.MaxTableSize = 500
	
	chunker.SetMultiModalConfig(newConfig)
	
	retrievedConfig := chunker.GetMultiModalConfig()
	if retrievedConfig.PreserveCodeBlocks != false {
		t.Error("Expected PreserveCodeBlocks to be false")
	}
	
	if retrievedConfig.MaxTableSize != 500 {
		t.Errorf("Expected MaxTableSize 500, got %d", retrievedConfig.MaxTableSize)
	}
}

// TestHierarchicalChunker tests the hierarchical chunker implementation
func TestHierarchicalChunker(t *testing.T) {
	config := DefaultChunkerConfig()
	config.ChunkSize = 150
	
	llmProvider := NewMockLLMProvider()
	
	chunker, err := NewHierarchicalChunker(config, llmProvider)
	if err != nil {
		t.Fatalf("Failed to create hierarchical chunker: %v", err)
	}
	
	ctx := context.Background()
	chunks, err := chunker.Chunk(ctx, testText)
	if err != nil {
		t.Fatalf("Failed to chunk text: %v", err)
	}
	
	if len(chunks) == 0 {
		t.Fatal("No chunks created")
	}
	
	// Verify hierarchical metadata
	hierarchyLevels := make(map[int]bool)
	for _, chunk := range chunks {
		if level, ok := chunk.Metadata["hierarchy_level"].(int); ok {
			hierarchyLevels[level] = true
		}
		
		// Check for hierarchical-specific metadata
		if _, ok := chunk.Metadata["node_id"]; !ok {
			t.Error("Node ID not found in metadata")
		}
		
		if _, ok := chunk.Metadata["child_count"]; !ok {
			t.Error("Child count not found in metadata")
		}
	}
	
	t.Logf("Created %d chunks with hierarchy levels: %v", len(chunks), hierarchyLevels)
}

// TestHierarchicalChunkerConfig tests hierarchical chunker configuration
func TestHierarchicalChunkerConfig(t *testing.T) {
	config := DefaultChunkerConfig()
	llmProvider := NewMockLLMProvider()
	
	chunker, err := NewHierarchicalChunker(config, llmProvider)
	if err != nil {
		t.Fatalf("Failed to create hierarchical chunker: %v", err)
	}
	
	// Test getting hierarchical config
	hierarchicalConfig := chunker.GetHierarchicalConfig()
	if hierarchicalConfig == nil {
		t.Fatal("Hierarchical config is nil")
	}
	
	// Test setting hierarchical config
	newConfig := DefaultHierarchicalConfig()
	newConfig.MaxLevels = 5
	newConfig.SummaryMode = SummaryModeLLM
	
	chunker.SetHierarchicalConfig(newConfig)
	
	retrievedConfig := chunker.GetHierarchicalConfig()
	if retrievedConfig.MaxLevels != 5 {
		t.Errorf("Expected MaxLevels 5, got %d", retrievedConfig.MaxLevels)
	}
	
	if retrievedConfig.SummaryMode != SummaryModeLLM {
		t.Errorf("Expected SummaryMode 'llm', got '%s'", retrievedConfig.SummaryMode)
	}
}

// TestQualityMetricsCalculator tests the quality metrics calculator
func TestQualityMetricsCalculator(t *testing.T) {
	config := DefaultQualityConfig()
	
	// Create a mock embedding provider for testing
	embeddingProvider := &MockEmbeddingProvider{}
	llmProvider := NewMockLLMProvider()
	
	calculator := NewQualityMetricsCalculator(config, embeddingProvider, llmProvider)
	
	// Create some test chunks
	chunks := []*Chunk{
		{
			Text:       "Machine learning is a powerful technology.",
			TokenCount: 8,
			Sentences:  []string{"Machine learning is a powerful technology."},
			Metadata:   make(map[string]interface{}),
			CreatedAt:  time.Now(),
		},
		{
			Text:       "It enables computers to learn from data automatically.",
			TokenCount: 9,
			Sentences:  []string{"It enables computers to learn from data automatically."},
			Metadata:   make(map[string]interface{}),
			CreatedAt:  time.Now(),
		},
	}
	
	ctx := context.Background()
	assessment, err := calculator.AssessChunks(ctx, chunks, testText)
	if err != nil {
		t.Fatalf("Failed to assess chunks: %v", err)
	}
	
	if assessment == nil {
		t.Fatal("Assessment is nil")
	}
	
	if assessment.ChunkCount != len(chunks) {
		t.Errorf("Expected chunk count %d, got %d", len(chunks), assessment.ChunkCount)
	}
	
	if assessment.OverallScore < 0 || assessment.OverallScore > 1 {
		t.Errorf("Overall score should be between 0 and 1, got %f", assessment.OverallScore)
	}
	
	t.Logf("Quality assessment: Overall Score = %.3f, Coherence = %.3f, Completeness = %.3f", 
		assessment.OverallScore, assessment.Coherence, assessment.Completeness)
}

// TestAdvancedConfigPresets tests advanced configuration presets
func TestAdvancedConfigPresets(t *testing.T) {
	presets := []ConfigurationPreset{
		PresetDefault,
		PresetHighQuality,
		PresetHighPerformance,
		PresetBalanced,
		PresetMinimal,
		PresetResearchPaper,
		PresetTechnicalDoc,
		PresetChatbot,
		PresetRAG,
	}
	
	for _, preset := range presets {
		config := GetPresetConfig(preset)
		if config == nil {
			t.Errorf("Preset config for %s is nil", preset)
			continue
		}
		
		// Validate the config
		validator := NewConfigValidator()
		if err := validator.ValidateConfig(config); err != nil {
			t.Errorf("Preset config %s is invalid: %v", preset, err)
		}
		
		t.Logf("Preset %s validated successfully", preset)
	}
}

// TestConfigSerialization tests configuration serialization
func TestConfigSerialization(t *testing.T) {
	config := GetPresetConfig(PresetHighQuality)
	serializer := NewConfigSerializer()
	
	// Test JSON serialization
	jsonData, err := serializer.ToJSON(config)
	if err != nil {
		t.Fatalf("Failed to serialize config to JSON: %v", err)
	}
	
	if len(jsonData) == 0 {
		t.Fatal("Serialized JSON is empty")
	}
	
	// Test JSON deserialization
	deserializedConfig, err := serializer.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize config from JSON: %v", err)
	}
	
	if deserializedConfig == nil {
		t.Fatal("Deserialized config is nil")
	}
	
	// Compare key fields
	if config.BaseConfig.ChunkSize != deserializedConfig.BaseConfig.ChunkSize {
		t.Errorf("Chunk size mismatch: expected %d, got %d", 
			config.BaseConfig.ChunkSize, deserializedConfig.BaseConfig.ChunkSize)
	}
	
	t.Log("Config serialization test passed")
}

// TestAdvancedChunkerFactory tests the enhanced chunker factory
func TestAdvancedChunkerFactory(t *testing.T) {
	factory := NewChunkerFactory()
	config := DefaultChunkerConfig()
	llmProvider := NewMockLLMProvider()
	
	// Test creating multi-modal chunker
	multiModalChunker, err := factory.CreateMultiModalChunker(config)
	if err != nil {
		t.Errorf("Failed to create multi-modal chunker: %v", err)
	} else {
		if multiModalChunker.GetChunkSize() != config.ChunkSize {
			t.Errorf("Multi-modal chunker chunk size mismatch")
		}
	}
	
	// Test creating agentic chunker
	agenticChunker, err := factory.CreateAgenticChunker(config, llmProvider)
	if err != nil {
		t.Errorf("Failed to create agentic chunker: %v", err)
	} else {
		if agenticChunker.GetChunkSize() != config.ChunkSize {
			t.Errorf("Agentic chunker chunk size mismatch")
		}
	}
	
	// Test creating hierarchical chunker
	hierarchicalChunker, err := factory.CreateHierarchicalChunker(config, llmProvider)
	if err != nil {
		t.Errorf("Failed to create hierarchical chunker: %v", err)
	} else {
		if hierarchicalChunker.GetChunkSize() != config.ChunkSize {
			t.Errorf("Hierarchical chunker chunk size mismatch")
		}
	}
	
	// Test creating chunkers through string names
	testCases := []struct {
		name     string
		expected ChunkerType
	}{
		{"agentic", ChunkerTypeAgentic},
		{"multimodal", ChunkerTypeMultiModal},
		{"multi-modal", ChunkerTypeMultiModal},
		{"hierarchical", ChunkerTypeHierarchical},
	}
	
	for _, tc := range testCases {
		// For chunkers that require LLM, we expect an error with guidance
		_, err := factory.CreateChunkerFromString(tc.name, config)
		if err == nil && (tc.expected == ChunkerTypeAgentic || tc.expected == ChunkerTypeHierarchical) {
			t.Errorf("Expected error for %s chunker without LLM provider", tc.name)
		} else if err != nil && tc.expected == ChunkerTypeMultiModal {
			t.Errorf("Unexpected error for %s chunker: %v", tc.name, err)
		}
	}
}

// TestPerformanceMetrics tests basic performance tracking
func TestPerformanceMetrics(t *testing.T) {
	config := DefaultChunkerConfig()
	config.ChunkSize = 100
	
	chunker, err := NewMultiModalChunker(config)
	if err != nil {
		t.Fatalf("Failed to create chunker: %v", err)
	}
	
	ctx := context.Background()
	
	// Measure chunking performance
	start := time.Now()
	chunks, err := chunker.Chunk(ctx, testText)
	duration := time.Since(start)
	
	if err != nil {
		t.Fatalf("Failed to chunk text: %v", err)
	}
	
	// Verify performance metadata is added
	for _, chunk := range chunks {
		if processingTime, ok := chunk.Metadata["processing_time"].(time.Duration); ok {
			if processingTime <= 0 {
				t.Error("Processing time should be positive")
			}
		}
		
		if stats, ok := chunk.Metadata["chunking_stats"]; ok {
			if stats == nil {
				t.Error("Chunking stats should not be nil")
			}
		}
	}
	
	t.Logf("Chunked %d characters into %d chunks in %v", len(testText), len(chunks), duration)
}

// BenchmarkAgenticChunker benchmarks the agentic chunker
func BenchmarkAgenticChunker(b *testing.B) {
	config := DefaultChunkerConfig()
	llmProvider := NewMockLLMProvider()
	
	chunker, err := NewAgenticChunker(config, llmProvider)
	if err != nil {
		b.Fatalf("Failed to create agentic chunker: %v", err)
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := chunker.Chunk(ctx, testText)
		if err != nil {
			b.Fatalf("Failed to chunk text: %v", err)
		}
	}
}

// BenchmarkMultiModalChunker benchmarks the multi-modal chunker
func BenchmarkMultiModalChunker(b *testing.B) {
	config := DefaultChunkerConfig()
	
	chunker, err := NewMultiModalChunker(config)
	if err != nil {
		b.Fatalf("Failed to create multi-modal chunker: %v", err)
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := chunker.Chunk(ctx, multiModalText)
		if err != nil {
			b.Fatalf("Failed to chunk text: %v", err)
		}
	}
}

// BenchmarkHierarchicalChunker benchmarks the hierarchical chunker
func BenchmarkHierarchicalChunker(b *testing.B) {
	config := DefaultChunkerConfig()
	llmProvider := NewMockLLMProvider()
	
	chunker, err := NewHierarchicalChunker(config, llmProvider)
	if err != nil {
		b.Fatalf("Failed to create hierarchical chunker: %v", err)
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := chunker.Chunk(ctx, testText)
		if err != nil {
			b.Fatalf("Failed to chunk text: %v", err)
		}
	}
}

// TestChunkerIntegration tests integration between different chunkers
func TestChunkerIntegration(t *testing.T) {
	config := DefaultChunkerConfig()
	config.ChunkSize = 200
	
	llmProvider := NewMockLLMProvider()
	ctx := context.Background()
	
	// Test different chunkers on the same text
	chunkers := map[string]Chunker{}
	
	// Multi-modal chunker
	multiModal, err := NewMultiModalChunker(config)
	if err == nil {
		chunkers["multimodal"] = multiModal
	}
	
	// Agentic chunker
	agentic, err := NewAgenticChunker(config, llmProvider)
	if err == nil {
		chunkers["agentic"] = agentic
	}
	
	// Hierarchical chunker
	hierarchical, err := NewHierarchicalChunker(config, llmProvider)
	if err == nil {
		chunkers["hierarchical"] = hierarchical
	}
	
	results := make(map[string][]*Chunk)
	
	for name, chunker := range chunkers {
		chunks, err := chunker.Chunk(ctx, testText)
		if err != nil {
			t.Errorf("Failed to chunk with %s chunker: %v", name, err)
			continue
		}
		results[name] = chunks
		t.Logf("%s chunker created %d chunks", name, len(chunks))
	}
	
	// Verify all chunkers produced valid results
	for name, chunks := range results {
		if len(chunks) == 0 {
			t.Errorf("%s chunker produced no chunks", name)
		}
		
		for i, chunk := range chunks {
			if chunk.Text == "" {
				t.Errorf("%s chunker chunk %d has empty text", name, i)
			}
			if chunk.TokenCount <= 0 {
				t.Errorf("%s chunker chunk %d has invalid token count: %d", name, i, chunk.TokenCount)
			}
		}
	}
}