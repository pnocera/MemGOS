package parsers

import (
	"context"
	"strings"
	"testing"

	"github.com/memtensor/memgos/pkg/chunkers"
	"github.com/memtensor/memgos/pkg/logger"
)

// MockEmbedder for testing
type MockEmbedder struct{}

func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	// Return a simple mock embedding
	return []float64{0.1, 0.2, 0.3, 0.4, 0.5}, nil
}

func (m *MockEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings := make([][]float64, len(texts))
	for i := range texts {
		embedding, err := m.Embed(ctx, texts[i])
		if err != nil {
			return nil, err
		}
		embeddings[i] = embedding
	}
	return embeddings, nil
}

func (m *MockEmbedder) GetDimensions() int {
	return 5
}

func (m *MockEmbedder) GetMaxTokens() int {
	return 512
}

func TestTextProcessingPipeline(t *testing.T) {
	// Create mock logger
	logger := logger.NewLogger()
	
	// Create mock embedder
	embedder := &MockEmbedder{}
	
	// Create default config
	config := DefaultTextProcessingConfig()
	
	// Create text processing service
	service, err := NewTextProcessingService(config, embedder, logger)
	if err != nil {
		t.Fatalf("Failed to create text processing service: %v", err)
	}
	
	// Test different document types
	testCases := []struct {
		name     string
		content  string
		filename string
		expected int // expected number of chunks
	}{
		{
			name:     "Simple text",
			content:  "This is a simple test document. It has multiple sentences. Each sentence should be processed correctly.",
			filename: "test.txt",
			expected: 1,
		},
		{
			name:     "JSON document",
			content:  `{"name": "John Doe", "age": 30, "city": "New York", "description": "A software engineer with 5 years of experience."}`,
			filename: "test.json",
			expected: 1,
		},
		{
			name:     "CSV document",
			content:  "Name,Age,City\nJohn Doe,30,New York\nJane Smith,25,Los Angeles\nBob Johnson,35,Chicago",
			filename: "test.csv",
			expected: 1,
		},
		{
			name:     "Markdown document",
			content:  "# Test Document\n\nThis is a **markdown** document with:\n\n- Lists\n- Links: [Example](http://example.com)\n- Code: `inline code`\n\n## Section 2\n\nMore content here.",
			filename: "test.md",
			expected: 1,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := strings.NewReader(tc.content)
			
			// Process the document
			result, err := service.ProcessDocument(context.Background(), reader, tc.filename)
			if err != nil {
				t.Fatalf("Failed to process document: %v", err)
			}
			
			// Verify results
			if result == nil {
				t.Fatal("Expected non-nil result")
			}
			
			if result.TotalChunks == 0 {
				t.Error("Expected at least one chunk")
			}
			
			if len(result.Chunks) == 0 {
				t.Error("Expected non-empty chunks array")
			}
			
			// Verify chunks have embeddings
			for i, chunk := range result.Chunks {
				if len(chunk.Embedding) == 0 {
					t.Errorf("Chunk %d missing embedding", i)
				}
				
				if chunk.Text == "" {
					t.Errorf("Chunk %d has empty text", i)
				}
				
				if chunk.TokenCount <= 0 {
					t.Errorf("Chunk %d has invalid token count: %d", i, chunk.TokenCount)
				}
			}
			
			t.Logf("Successfully processed %s: %d chunks, parser: %s", tc.filename, result.TotalChunks, result.ParserUsed)
		})
	}
}

func TestParserFactory(t *testing.T) {
	factory := NewParserFactory()
	
	// Test supported formats
	supportedTypes := factory.GetSupportedTypes()
	if len(supportedTypes) == 0 {
		t.Error("Expected at least one supported type")
	}
	
	supportedExtensions := factory.GetSupportedExtensions()
	if len(supportedExtensions) == 0 {
		t.Error("Expected at least one supported extension")
	}
	
	// Test parser creation by extension
	testExtensions := []string{".txt", ".json", ".csv", ".md", ".html", ".pdf"}
	for _, ext := range testExtensions {
		parser, err := factory.GetParserByExtension(ext)
		if err != nil {
			t.Errorf("Failed to get parser for extension %s: %v", ext, err)
			continue
		}
		
		if parser == nil {
			t.Errorf("Got nil parser for extension %s", ext)
		}
	}
}

func TestSemanticChunker(t *testing.T) {
	config := chunkers.DefaultChunkerConfig()
	config.ChunkSize = 100
	config.ChunkOverlap = 20
	
	chunker, err := chunkers.NewSemanticChunker(config)
	if err != nil {
		t.Fatalf("Failed to create semantic chunker: %v", err)
	}
	
	text := `Artificial intelligence is transforming the world. Machine learning algorithms are becoming more sophisticated each day. 
	Neural networks can now process complex patterns in data. Deep learning has revolutionized computer vision and natural language processing.
	However, there are still challenges to overcome. Ethical considerations in AI development are crucial. 
	We must ensure that AI systems are fair and unbiased. The future of AI depends on responsible development practices.`
	
	chunks, err := chunker.Chunk(context.Background(), text)
	if err != nil {
		t.Fatalf("Failed to chunk text: %v", err)
	}
	
	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}
	
	for i, chunk := range chunks {
		if chunk.Text == "" {
			t.Errorf("Chunk %d has empty text", i)
		}
		
		if chunk.TokenCount <= 0 {
			t.Errorf("Chunk %d has invalid token count: %d", i, chunk.TokenCount)
		}
		
		// Check semantic metadata
		if coherence, exists := chunk.Metadata["semantic_coherence"]; exists {
			if score, ok := coherence.(float64); ok {
				if score < 0 || score > 1 {
					t.Errorf("Chunk %d has invalid coherence score: %f", i, score)
				}
			}
		}
		
		t.Logf("Chunk %d: %d tokens, text: %.50s...", i, chunk.TokenCount, chunk.Text)
	}
}

func TestTextProcessingConfig(t *testing.T) {
	// Test default config
	config := DefaultTextProcessingConfig()
	
	err := ValidateConfig(config)
	if err != nil {
		t.Errorf("Default config should be valid: %v", err)
	}
	
	// Test optimized configs
	useCases := []string{"research", "social_media", "documentation", "legal", "web_scraping", "multilingual", "high_performance", "memory_constrained"}
	
	for _, useCase := range useCases {
		optimizedConfig := CreateOptimizedConfig(useCase)
		
		err := ValidateConfig(optimizedConfig)
		if err != nil {
			t.Errorf("Optimized config for %s should be valid: %v", useCase, err)
		}
		
		t.Logf("Successfully validated %s config", useCase)
	}
}

func TestJSONParser(t *testing.T) {
	parser := NewJSONParser()
	
	jsonContent := `{
		"title": "Test Document",
		"author": "John Doe",
		"content": {
			"sections": [
				{
					"heading": "Introduction",
					"text": "This is the introduction section with detailed information about the topic."
				},
				{
					"heading": "Methods",
					"text": "This section describes the methodology used in the research."
				}
			]
		},
		"metadata": {
			"created": "2024-01-01",
			"tags": ["test", "example", "json"]
		}
	}`
	
	reader := strings.NewReader(jsonContent)
	config := DefaultParserConfig()
	
	result, err := parser.Parse(context.Background(), reader, config)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	
	if result.Content == "" {
		t.Error("Expected non-empty content")
	}
	
	if result.Metadata.MimeType != "application/json" {
		t.Errorf("Expected MIME type application/json, got %s", result.Metadata.MimeType)
	}
	
	if result.ParserType != string(ParserTypeJSON) {
		t.Errorf("Expected parser type %s, got %s", ParserTypeJSON, result.ParserType)
	}
	
	t.Logf("Successfully parsed JSON document: %d characters", len(result.Content))
}

func TestCSVParser(t *testing.T) {
	parser := NewCSVParser()
	
	csvContent := `Name,Age,City,Occupation
John Doe,30,New York,Software Engineer
Jane Smith,25,Los Angeles,Data Scientist
Bob Johnson,35,Chicago,Product Manager
Alice Brown,28,Seattle,UX Designer`
	
	reader := strings.NewReader(csvContent)
	config := DefaultParserConfig()
	
	result, err := parser.Parse(context.Background(), reader, config)
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}
	
	if result.Content == "" {
		t.Error("Expected non-empty content")
	}
	
	if result.Metadata.MimeType != "text/csv" {
		t.Errorf("Expected MIME type text/csv, got %s", result.Metadata.MimeType)
	}
	
	// Check structured content
	if result.StructuredContent == nil {
		t.Error("Expected structured content for CSV")
	} else {
		if len(result.StructuredContent.Tables) == 0 {
			t.Error("Expected at least one table in structured content")
		} else {
			table := result.StructuredContent.Tables[0]
			if len(table.Headers) == 0 {
				t.Error("Expected headers in CSV table")
			}
			if len(table.Rows) == 0 {
				t.Error("Expected rows in CSV table")
			}
			
			t.Logf("Successfully parsed CSV: %d headers, %d rows", len(table.Headers), len(table.Rows))
		}
	}
}

func BenchmarkTextProcessing(b *testing.B) {
	// Create service
	logger := logger.NewLogger()
	embedder := &MockEmbedder{}
	config := DefaultTextProcessingConfig()
	
	service, err := NewTextProcessingService(config, embedder, logger)
	if err != nil {
		b.Fatalf("Failed to create service: %v", err)
	}
	
	// Test content
	content := strings.Repeat("This is a test sentence for benchmarking text processing performance. ", 100)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(content)
		_, err := service.ProcessDocument(context.Background(), reader, "test.txt")
		if err != nil {
			b.Fatalf("Failed to process document: %v", err)
		}
	}
}