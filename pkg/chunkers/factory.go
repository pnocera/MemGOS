package chunkers

import (
	"fmt"
	"strings"
	
	"github.com/memtensor/memgos/pkg/interfaces"
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
	
	case ChunkerTypeContextual:
		// Contextual chunker requires LLM provider - return error with guidance
		return nil, fmt.Errorf("contextual chunker requires LLM provider, use CreateContextualChunker() instead")
	
	case ChunkerTypePropositionalization:
		// Propositionalization chunker requires LLM provider - return error with guidance
		return nil, fmt.Errorf("propositionalization chunker requires LLM provider, use CreatePropositionalizationChunker() instead")
	
	case ChunkerTypeAgentic:
		// Agentic chunker requires LLM provider - return error with guidance
		return nil, fmt.Errorf("agentic chunker requires LLM provider, use CreateAgenticChunker() instead")
	
	case ChunkerTypeMultiModal:
		return NewMultiModalChunker(config)
	
	case ChunkerTypeHierarchical:
		// Hierarchical chunker optionally uses LLM provider - return error with guidance
		return nil, fmt.Errorf("hierarchical chunker optionally uses LLM provider, use CreateHierarchicalChunker() instead")
	
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
	case "contextual":
		chunkerType = ChunkerTypeContextual
	case "propositionalization":
		chunkerType = ChunkerTypePropositionalization
	case "agentic":
		chunkerType = ChunkerTypeAgentic
	case "multimodal", "multi-modal":
		chunkerType = ChunkerTypeMultiModal
	case "hierarchical":
		chunkerType = ChunkerTypeHierarchical
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
		{
			Type:        ChunkerTypeContextual,
			Name:        "Contextual Chunker",
			Description: "Anthropic-style chunking with LLM-generated contextual descriptions",
			Features: []string{
				"LLM-powered context generation",
				"Document-level context awareness",
				"Semantic metadata enrichment",
				"Enhanced retrieval quality",
			},
			UseCases: []string{
				"Advanced RAG systems",
				"Knowledge base construction",
				"Context-aware search",
				"Document analysis",
			},
		},
		{
			Type:        ChunkerTypePropositionalization,
			Name:        "Propositionalization Chunker",
			Description: "Extracts atomic propositions for complex reasoning tasks",
			Features: []string{
				"Atomic proposition extraction",
				"Self-contained semantic units",
				"LLM-powered validation",
				"Deduplication and filtering",
			},
			UseCases: []string{
				"Fact extraction",
				"Knowledge graphs",
				"Logical reasoning",
				"Information verification",
			},
		},
		{
			Type:        ChunkerTypeAgentic,
			Name:        "Agentic Chunker",
			Description: "LLM-driven chunking with AI decision-making for optimal boundaries",
			Features: []string{
				"LLM-powered boundary detection",
				"Multi-round reasoning",
				"Document type adaptation",
				"Confidence-based decisions",
				"Human-like processing patterns",
			},
			UseCases: []string{
				"High-quality document processing",
				"Adaptive content analysis",
				"Context-aware chunking",
				"Premium RAG systems",
			},
		},
		{
			Type:        ChunkerTypeMultiModal,
			Name:        "Multi-Modal Chunker",
			Description: "Handles documents with mixed content types including text, code, tables, and images",
			Features: []string{
				"Mixed content detection",
				"Code block preservation",
				"Table structure handling",
				"Image reference processing",
				"Format-aware chunking",
			},
			UseCases: []string{
				"Technical documentation",
				"Research papers",
				"Mixed-format documents",
				"Web content processing",
			},
		},
		{
			Type:        ChunkerTypeHierarchical,
			Name:        "Hierarchical Chunker",
			Description: "Creates hierarchical chunk relationships with parent-child structure and summaries",
			Features: []string{
				"Multi-level hierarchy",
				"Parent-child relationships",
				"Automatic summarization",
				"Cross-reference tracking",
				"Structure preservation",
			},
			UseCases: []string{
				"Document navigation",
				"Structured knowledge bases",
				"Multi-level summaries",
				"Complex document analysis",
			},
		},
	}
}

// CreateContextualChunker creates a contextual chunker with LLM provider
func (cf *ChunkerFactory) CreateContextualChunker(config *ChunkerConfig, llmProvider interfaces.LLM) (Chunker, error) {
	if llmProvider == nil {
		return nil, fmt.Errorf("LLM provider is required for contextual chunker")
	}
	
	return NewContextualChunker(config, llmProvider)
}

// CreatePropositionalizationChunker creates a propositionalization chunker with LLM provider
func (cf *ChunkerFactory) CreatePropositionalizationChunker(config *ChunkerConfig, llmProvider interfaces.LLM) (Chunker, error) {
	if llmProvider == nil {
		return nil, fmt.Errorf("LLM provider is required for propositionalization chunker")
	}
	
	return NewPropositionalizationChunker(config, llmProvider)
}

// CreateAgenticChunker creates an agentic chunker with LLM provider
func (cf *ChunkerFactory) CreateAgenticChunker(config *ChunkerConfig, llmProvider interfaces.LLM) (Chunker, error) {
	if llmProvider == nil {
		return nil, fmt.Errorf("LLM provider is required for agentic chunker")
	}
	
	return NewAgenticChunker(config, llmProvider)
}

// CreateMultiModalChunker creates a multi-modal chunker
func (cf *ChunkerFactory) CreateMultiModalChunker(config *ChunkerConfig) (Chunker, error) {
	return NewMultiModalChunker(config)
}

// CreateHierarchicalChunker creates a hierarchical chunker with optional LLM provider
func (cf *ChunkerFactory) CreateHierarchicalChunker(config *ChunkerConfig, llmProvider interfaces.LLM) (Chunker, error) {
	return NewHierarchicalChunker(config, llmProvider)
}

// CreateAdvancedChunker creates chunkers that require additional dependencies
func (cf *ChunkerFactory) CreateAdvancedChunker(chunkerType ChunkerType, config *ChunkerConfig, llmProvider interfaces.LLM) (Chunker, error) {
	if config == nil {
		config = DefaultChunkerConfig()
	}
	
	// Validate chunker type
	if !IsValidChunkerType(chunkerType) {
		return nil, fmt.Errorf("unsupported chunker type: %s. Supported types: %v", 
			chunkerType, SupportedChunkerTypes())
	}
	
	switch chunkerType {
	case ChunkerTypeContextual:
		return cf.CreateContextualChunker(config, llmProvider)
	
	case ChunkerTypePropositionalization:
		return cf.CreatePropositionalizationChunker(config, llmProvider)
	
	case ChunkerTypeAgentic:
		return cf.CreateAgenticChunker(config, llmProvider)
	
	case ChunkerTypeHierarchical:
		return cf.CreateHierarchicalChunker(config, llmProvider)
	
	default:
		// For standard chunkers, use the regular factory method
		return cf.CreateChunker(chunkerType, config)
	}
}