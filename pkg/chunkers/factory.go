package chunkers

import (
	"fmt"
	"strings"
)

// ChunkerFactory provides a factory for creating different types of chunkers
type ChunkerFactory struct{}

// NewChunkerFactory creates a new chunker factory
func NewChunkerFactory() *ChunkerFactory {
	return &ChunkerFactory{}
}

// CreateChunker creates a chunker instance based on the specified type and configuration
func (cf *ChunkerFactory) CreateChunker(chunkerType ChunkerType, config *ChunkerConfig) (Chunker, error) {
	if config == nil {
		config = DefaultChunkerConfig()
	}
	
	// Validate chunker type
	if !IsValidChunkerType(chunkerType) {
		return nil, fmt.Errorf("unsupported chunker type: %s. Supported types: %v", 
			chunkerType, SupportedChunkerTypes())
	}
	
	switch chunkerType {
	case ChunkerTypeSentence:
		return NewSentenceChunker(config)
	
	case ChunkerTypeParagraph:
		return NewParagraphChunker(config)
	
	case ChunkerTypeSemantic:
		return NewSemanticChunker(config)
	
	case ChunkerTypeFixed:
		return NewFixedChunker(config)
	
	case ChunkerTypeRecursive:
		return NewRecursiveChunker(config)
	
	default:
		return nil, fmt.Errorf("chunker type %s is not implemented yet", chunkerType)
	}
}

// CreateChunkerFromString creates a chunker from a string type name
func (cf *ChunkerFactory) CreateChunkerFromString(chunkerTypeName string, config *ChunkerConfig) (Chunker, error) {
	chunkerTypeName = strings.ToLower(strings.TrimSpace(chunkerTypeName))
	
	var chunkerType ChunkerType
	switch chunkerTypeName {
	case "sentence":
		chunkerType = ChunkerTypeSentence
	case "paragraph":
		chunkerType = ChunkerTypeParagraph
	case "semantic":
		chunkerType = ChunkerTypeSemantic
	case "fixed":
		chunkerType = ChunkerTypeFixed
	case "recursive":
		chunkerType = ChunkerTypeRecursive
	default:
		return nil, fmt.Errorf("unknown chunker type: %s", chunkerTypeName)
	}
	
	return cf.CreateChunker(chunkerType, config)
}

// GetDefaultChunker returns a default sentence-based chunker with standard configuration
func (cf *ChunkerFactory) GetDefaultChunker() (Chunker, error) {
	return cf.CreateChunker(ChunkerTypeSentence, DefaultChunkerConfig())
}

// GetSupportedTypes returns a list of all supported chunker types
func (cf *ChunkerFactory) GetSupportedTypes() []ChunkerType {
	return SupportedChunkerTypes()
}

// ValidateConfig validates a chunker configuration
func (cf *ChunkerFactory) ValidateConfig(config *ChunkerConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	
	if config.ChunkSize <= 0 {
		return fmt.Errorf("chunk size must be positive, got: %d", config.ChunkSize)
	}
	
	if config.ChunkOverlap < 0 {
		return fmt.Errorf("chunk overlap cannot be negative, got: %d", config.ChunkOverlap)
	}
	
	if config.ChunkOverlap >= config.ChunkSize {
		return fmt.Errorf("chunk overlap (%d) must be less than chunk size (%d)", 
			config.ChunkOverlap, config.ChunkSize)
	}
	
	if config.MinSentencesPerChunk <= 0 {
		return fmt.Errorf("minimum sentences per chunk must be positive, got: %d", 
			config.MinSentencesPerChunk)
	}
	
	if config.MaxSentencesPerChunk <= 0 {
		return fmt.Errorf("maximum sentences per chunk must be positive, got: %d", 
			config.MaxSentencesPerChunk)
	}
	
	if config.MinSentencesPerChunk > config.MaxSentencesPerChunk {
		return fmt.Errorf("minimum sentences (%d) cannot be greater than maximum sentences (%d)", 
			config.MinSentencesPerChunk, config.MaxSentencesPerChunk)
	}
	
	if config.Language == "" {
		return fmt.Errorf("language cannot be empty")
	}
	
	return nil
}

// BuildConfigFromMap creates a chunker configuration from a map of parameters
func (cf *ChunkerFactory) BuildConfigFromMap(params map[string]interface{}) (*ChunkerConfig, error) {
	config := DefaultChunkerConfig()
	
	if val, ok := params["chunk_size"]; ok {
		if size, ok := val.(int); ok {
			config.ChunkSize = size
		} else {
			return nil, fmt.Errorf("chunk_size must be an integer")
		}
	}
	
	if val, ok := params["chunk_overlap"]; ok {
		if overlap, ok := val.(int); ok {
			config.ChunkOverlap = overlap
		} else {
			return nil, fmt.Errorf("chunk_overlap must be an integer")
		}
	}
	
	if val, ok := params["min_sentences_per_chunk"]; ok {
		if minSent, ok := val.(int); ok {
			config.MinSentencesPerChunk = minSent
		} else {
			return nil, fmt.Errorf("min_sentences_per_chunk must be an integer")
		}
	}
	
	if val, ok := params["max_sentences_per_chunk"]; ok {
		if maxSent, ok := val.(int); ok {
			config.MaxSentencesPerChunk = maxSent
		} else {
			return nil, fmt.Errorf("max_sentences_per_chunk must be an integer")
		}
	}
	
	if val, ok := params["preserve_formatting"]; ok {
		if preserve, ok := val.(bool); ok {
			config.PreserveFormatting = preserve
		} else {
			return nil, fmt.Errorf("preserve_formatting must be a boolean")
		}
	}
	
	if val, ok := params["language"]; ok {
		if lang, ok := val.(string); ok {
			config.Language = lang
		} else {
			return nil, fmt.Errorf("language must be a string")
		}
	}
	
	// Validate the built configuration
	if err := cf.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	return config, nil
}

// ChunkerDescriptor provides information about a chunker type
type ChunkerDescriptor struct {
	Type        ChunkerType `json:"type"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Features    []string    `json:"features"`
	UseCases    []string    `json:"use_cases"`
}

// GetChunkerDescriptors returns descriptive information about all chunker types
func (cf *ChunkerFactory) GetChunkerDescriptors() []ChunkerDescriptor {
	return []ChunkerDescriptor{
		{
			Type:        ChunkerTypeSentence,
			Name:        "Sentence Chunker",
			Description: "Splits text into chunks based on sentence boundaries while respecting token limits",
			Features: []string{
				"Sentence boundary detection",
				"Token-aware chunking",
				"Configurable overlap",
				"Multi-language support",
			},
			UseCases: []string{
				"General document processing",
				"Q&A systems",
				"Semantic search",
				"Content summarization",
			},
		},
		{
			Type:        ChunkerTypeParagraph,
			Name:        "Paragraph Chunker",
			Description: "Splits text into chunks based on paragraph boundaries",
			Features: []string{
				"Paragraph boundary detection",
				"Preserves document structure",
				"Configurable size limits",
			},
			UseCases: []string{
				"Document analysis",
				"Content organization",
				"Topic modeling",
			},
		},
		{
			Type:        ChunkerTypeSemantic,
			Name:        "Semantic Chunker",
			Description: "Splits text based on semantic coherence and meaning",
			Features: []string{
				"Semantic boundary detection",
				"Context preservation",
				"AI-powered segmentation",
			},
			UseCases: []string{
				"Advanced NLP tasks",
				"High-quality embedding generation",
				"Context-aware processing",
			},
		},
		{
			Type:        ChunkerTypeFixed,
			Name:        "Fixed Chunker",
			Description: "Splits text into fixed-size chunks by token or character count",
			Features: []string{
				"Predictable chunk sizes",
				"Simple and fast",
				"Token or character based",
			},
			UseCases: []string{
				"Batch processing",
				"Memory-constrained environments",
				"Simple text splitting",
			},
		},
		{
			Type:        ChunkerTypeRecursive,
			Name:        "Recursive Chunker",
			Description: "Recursively splits text preserving hierarchical structure",
			Features: []string{
				"Hierarchical splitting",
				"Structure preservation",
				"Adaptive sizing",
			},
			UseCases: []string{
				"Structured documents",
				"Code documentation",
				"Technical manuals",
			},
		},
	}
}