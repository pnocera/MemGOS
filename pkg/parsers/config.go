package parsers

import (
	"fmt"
	"time"

	"github.com/memtensor/memgos/pkg/chunkers"
)

// TextProcessingConfig represents comprehensive configuration for text processing
type TextProcessingConfig struct {
	// Parser configurations
	ParserConfig *ParserConfig `json:"parser_config"`
	
	// Chunker configurations
	ChunkerType   chunkers.ChunkerType   `json:"chunker_type"`
	ChunkerConfig *chunkers.ChunkerConfig `json:"chunker_config"`
	
	// Processing pipeline options
	PipelineConfig *PipelineConfig `json:"pipeline_config"`
	
	// Language processing options
	LanguageProcessing *LanguageProcessingConfig `json:"language_processing"`
	
	// Performance tuning
	PerformanceConfig *PerformanceConfig `json:"performance_config"`
	
	// Content filtering and validation
	ContentFiltering *ContentFilteringConfig `json:"content_filtering"`
}

// LanguageProcessingConfig configures language-specific processing
type LanguageProcessingConfig struct {
	// Default language for processing
	DefaultLanguage string `json:"default_language"`
	
	// Enable automatic language detection
	EnableLanguageDetection bool `json:"enable_language_detection"`
	
	// Language-specific processing rules
	LanguageRules map[string]*LanguageRule `json:"language_rules"`
	
	// Text normalization options
	NormalizeUnicode     bool `json:"normalize_unicode"`
	NormalizeWhitespace  bool `json:"normalize_whitespace"`
	NormalizePunctuation bool `json:"normalize_punctuation"`
	
	// Encoding handling
	DefaultEncoding string   `json:"default_encoding"`
	SupportedEncodings []string `json:"supported_encodings"`
}

// LanguageRule defines processing rules for specific languages
type LanguageRule struct {
	// Language code (e.g., "en", "es", "fr")
	Language string `json:"language"`
	
	// Sentence boundary detection patterns
	SentencePatterns []string `json:"sentence_patterns"`
	
	// Paragraph detection rules
	ParagraphRules []string `json:"paragraph_rules"`
	
	// Stop words for content analysis
	StopWords []string `json:"stop_words"`
	
	// Chunking preferences
	PreferredChunkSize int `json:"preferred_chunk_size"`
	
	// Text direction (ltr, rtl)
	TextDirection string `json:"text_direction"`
}

// PerformanceConfig configures performance-related settings
type PerformanceConfig struct {
	// Maximum number of concurrent processing tasks
	MaxConcurrentTasks int `json:"max_concurrent_tasks"`
	
	// Memory usage limits
	MaxMemoryUsageBytes int64 `json:"max_memory_usage_bytes"`
	
	// Processing timeouts
	DefaultTimeout    time.Duration `json:"default_timeout"`
	ParsingTimeout    time.Duration `json:"parsing_timeout"`
	ChunkingTimeout   time.Duration `json:"chunking_timeout"`
	EmbeddingTimeout  time.Duration `json:"embedding_timeout"`
	
	// Caching options
	EnableCaching         bool          `json:"enable_caching"`
	CacheSize             int           `json:"cache_size"`
	CacheTTL              time.Duration `json:"cache_ttl"`
	
	// Batch processing
	BatchSize           int  `json:"batch_size"`
	EnableBatchMode     bool `json:"enable_batch_mode"`
	
	// Resource optimization
	OptimizeForMemory   bool `json:"optimize_for_memory"`
	OptimizeForSpeed    bool `json:"optimize_for_speed"`
	EnableCompression   bool `json:"enable_compression"`
}

// ContentFilteringConfig configures content filtering and validation
type ContentFilteringConfig struct {
	// File type restrictions
	AllowedFileTypes    []string `json:"allowed_file_types"`
	BlockedFileTypes    []string `json:"blocked_file_types"`
	
	// Content size limits
	MinContentLength    int   `json:"min_content_length"`
	MaxContentLength    int   `json:"max_content_length"`
	MaxFileSize         int64 `json:"max_file_size"`
	
	// Content quality filters
	MinReadabilityScore float64 `json:"min_readability_score"`
	MinContentDensity   float64 `json:"min_content_density"`
	
	// Content type detection
	EnableContentTypeDetection bool     `json:"enable_content_type_detection"`
	RequiredContentTypes      []string `json:"required_content_types"`
	
	// Text cleaning options
	RemoveUrls           bool `json:"remove_urls"`
	RemoveEmails         bool `json:"remove_emails"`
	RemovePhoneNumbers   bool `json:"remove_phone_numbers"`
	RemoveSpecialChars   bool `json:"remove_special_chars"`
	RemoveExtraWhitespace bool `json:"remove_extra_whitespace"`
	
	// Privacy and security
	EnablePIIDetection   bool     `json:"enable_pii_detection"`
	PIIPatterns         []string `json:"pii_patterns"`
	RedactPII           bool     `json:"redact_pii"`
	
	// Spam and quality detection
	EnableSpamDetection  bool     `json:"enable_spam_detection"`
	SpamKeywords        []string `json:"spam_keywords"`
	MinWordCount        int      `json:"min_word_count"`
}

// DefaultTextProcessingConfig returns a comprehensive default configuration
func DefaultTextProcessingConfig() *TextProcessingConfig {
	return &TextProcessingConfig{
		ParserConfig:   DefaultParserConfig(),
		ChunkerType:    chunkers.ChunkerTypeSentence,
		ChunkerConfig:  chunkers.DefaultChunkerConfig(),
		PipelineConfig: DefaultPipelineConfig(),
		LanguageProcessing: &LanguageProcessingConfig{
			DefaultLanguage:         "en",
			EnableLanguageDetection: true,
			LanguageRules:          make(map[string]*LanguageRule),
			NormalizeUnicode:       true,
			NormalizeWhitespace:    true,
			NormalizePunctuation:   false,
			DefaultEncoding:        "utf-8",
			SupportedEncodings:     []string{"utf-8", "ascii", "iso-8859-1"},
		},
		PerformanceConfig: &PerformanceConfig{
			MaxConcurrentTasks:    4,
			MaxMemoryUsageBytes:   500 * 1024 * 1024, // 500MB
			DefaultTimeout:        30 * time.Second,
			ParsingTimeout:        60 * time.Second,
			ChunkingTimeout:       30 * time.Second,
			EmbeddingTimeout:      120 * time.Second,
			EnableCaching:         true,
			CacheSize:            1000,
			CacheTTL:             24 * time.Hour,
			BatchSize:            10,
			EnableBatchMode:      true,
			OptimizeForMemory:    true,
			OptimizeForSpeed:     false,
			EnableCompression:    false,
		},
		ContentFiltering: &ContentFilteringConfig{
			AllowedFileTypes:           []string{".txt", ".md", ".html", ".pdf", ".json", ".csv"},
			BlockedFileTypes:           []string{".exe", ".bin", ".zip"},
			MinContentLength:           10,
			MaxContentLength:           10 * 1024 * 1024, // 10MB
			MaxFileSize:               100 * 1024 * 1024, // 100MB
			MinReadabilityScore:        0.0,
			MinContentDensity:          0.1,
			EnableContentTypeDetection: true,
			RequiredContentTypes:       []string{"text"},
			RemoveUrls:                false,
			RemoveEmails:              false,
			RemovePhoneNumbers:        false,
			RemoveSpecialChars:        false,
			RemoveExtraWhitespace:     true,
			EnablePIIDetection:        false,
			PIIPatterns:              []string{},
			RedactPII:                false,
			EnableSpamDetection:       false,
			SpamKeywords:             []string{},
			MinWordCount:             5,
		},
	}
}

// CreateOptimizedConfig creates a configuration optimized for specific use cases
func CreateOptimizedConfig(useCase string) *TextProcessingConfig {
	config := DefaultTextProcessingConfig()
	
	switch useCase {
	case "research":
		// Optimize for academic/research content
		config.ChunkerType = chunkers.ChunkerTypeSemantic
		config.ChunkerConfig.ChunkSize = 1024
		config.ChunkerConfig.ChunkOverlap = 256
		config.ParserConfig.ExtractStructure = true
		config.ParserConfig.ExtractMetadata = true
		config.ContentFiltering.MinReadabilityScore = 30.0
		config.ContentFiltering.MinWordCount = 50
		
	case "social_media":
		// Optimize for short-form content
		config.ChunkerType = chunkers.ChunkerTypeParagraph
		config.ChunkerConfig.ChunkSize = 256
		config.ChunkerConfig.ChunkOverlap = 64
		config.ContentFiltering.MinContentLength = 5
		config.ContentFiltering.MaxContentLength = 5000
		config.ContentFiltering.EnableSpamDetection = true
		
	case "documentation":
		// Optimize for technical documentation
		config.ChunkerType = chunkers.ChunkerTypeRecursive
		config.ChunkerConfig.ChunkSize = 2048
		config.ChunkerConfig.ChunkOverlap = 512
		config.ParserConfig.ExtractStructure = true
		config.ParserConfig.ExtractTables = true
		config.ParserConfig.ExtractImages = true
		
	case "legal":
		// Optimize for legal documents
		config.ChunkerType = chunkers.ChunkerTypeSentence
		config.ChunkerConfig.ChunkSize = 512
		config.ChunkerConfig.ChunkOverlap = 128
		config.ContentFiltering.EnablePIIDetection = true
		config.ContentFiltering.RedactPII = false // Preserve for legal analysis
		config.ParserConfig.PreserveFormatting = true
		
	case "web_scraping":
		// Optimize for web content
		config.ParserConfig.ExtractTables = true
		config.ParserConfig.ExtractImages = false // Usually not needed
		config.ContentFiltering.RemoveUrls = false
		config.ContentFiltering.RemoveExtraWhitespace = true
		config.ContentFiltering.EnableSpamDetection = true
		
	case "multilingual":
		// Optimize for multiple languages
		config.LanguageProcessing.EnableLanguageDetection = true
		config.LanguageProcessing.NormalizeUnicode = true
		config.LanguageProcessing.SupportedEncodings = []string{"utf-8", "utf-16", "iso-8859-1", "windows-1252"}
		config.ChunkerType = chunkers.ChunkerTypeSemantic
		
	case "high_performance":
		// Optimize for speed and throughput
		config.PerformanceConfig.MaxConcurrentTasks = 8
		config.PerformanceConfig.OptimizeForSpeed = true
		config.PerformanceConfig.OptimizeForMemory = false
		config.PerformanceConfig.EnableCaching = true
		config.PerformanceConfig.BatchSize = 50
		config.ParserConfig.ExtractStructure = false
		config.ContentFiltering.EnableContentTypeDetection = false
		
	case "memory_constrained":
		// Optimize for low memory usage
		config.PerformanceConfig.MaxConcurrentTasks = 2
		config.PerformanceConfig.OptimizeForMemory = true
		config.PerformanceConfig.MaxMemoryUsageBytes = 100 * 1024 * 1024 // 100MB
		config.PerformanceConfig.EnableCompression = true
		config.ContentFiltering.MaxFileSize = 10 * 1024 * 1024 // 10MB
		config.ChunkerConfig.ChunkSize = 256
	}
	
	return config
}

// ValidateConfig validates a text processing configuration
func ValidateConfig(config *TextProcessingConfig) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}
	
	// Validate parser config
	if config.ParserConfig == nil {
		return fmt.Errorf("parser configuration is required")
	}
	
	if config.ParserConfig.MaxFileSize < 0 {
		return fmt.Errorf("max file size cannot be negative")
	}
	
	if config.ParserConfig.Timeout < 0 {
		return fmt.Errorf("parser timeout cannot be negative")
	}
	
	// Validate chunker config
	if config.ChunkerConfig == nil {
		return fmt.Errorf("chunker configuration is required")
	}
	
	if config.ChunkerConfig.ChunkSize <= 0 {
		return fmt.Errorf("chunk size must be positive")
	}
	
	if config.ChunkerConfig.ChunkOverlap < 0 {
		return fmt.Errorf("chunk overlap cannot be negative")
	}
	
	if config.ChunkerConfig.ChunkOverlap >= config.ChunkerConfig.ChunkSize {
		return fmt.Errorf("chunk overlap must be less than chunk size")
	}
	
	// Validate pipeline config
	if config.PipelineConfig == nil {
		return fmt.Errorf("pipeline configuration is required")
	}
	
	if config.PipelineConfig.MaxConcurrency <= 0 {
		return fmt.Errorf("max concurrency must be positive")
	}
	
	// Validate performance config
	if config.PerformanceConfig != nil {
		if config.PerformanceConfig.MaxConcurrentTasks <= 0 {
			return fmt.Errorf("max concurrent tasks must be positive")
		}
		
		if config.PerformanceConfig.MaxMemoryUsageBytes < 0 {
			return fmt.Errorf("max memory usage cannot be negative")
		}
		
		if config.PerformanceConfig.CacheSize < 0 {
			return fmt.Errorf("cache size cannot be negative")
		}
		
		if config.PerformanceConfig.BatchSize <= 0 {
			return fmt.Errorf("batch size must be positive")
		}
	}
	
	// Validate content filtering config
	if config.ContentFiltering != nil {
		if config.ContentFiltering.MinContentLength < 0 {
			return fmt.Errorf("min content length cannot be negative")
		}
		
		if config.ContentFiltering.MaxContentLength < config.ContentFiltering.MinContentLength {
			return fmt.Errorf("max content length must be greater than min content length")
		}
		
		if config.ContentFiltering.MaxFileSize < 0 {
			return fmt.Errorf("max file size cannot be negative")
		}
		
		if config.ContentFiltering.MinReadabilityScore < 0 || config.ContentFiltering.MinReadabilityScore > 100 {
			return fmt.Errorf("readability score must be between 0 and 100")
		}
		
		if config.ContentFiltering.MinContentDensity < 0 || config.ContentFiltering.MinContentDensity > 1 {
			return fmt.Errorf("content density must be between 0 and 1")
		}
	}
	
	return nil
}

// MergeConfigs merges multiple configurations with priority order
func MergeConfigs(base *TextProcessingConfig, overrides ...*TextProcessingConfig) (*TextProcessingConfig, error) {
	if base == nil {
		return nil, fmt.Errorf("base configuration cannot be nil")
	}
	
	// Create a copy of the base config
	result := *base
	
	for _, override := range overrides {
		if override == nil {
			continue
		}
		
		// Merge parser config
		if override.ParserConfig != nil {
			if result.ParserConfig == nil {
				result.ParserConfig = override.ParserConfig
			} else {
				merged := *result.ParserConfig
				if override.ParserConfig.ExtractMetadata {
					merged.ExtractMetadata = true
				}
				if override.ParserConfig.ExtractStructure {
					merged.ExtractStructure = true
				}
				if override.ParserConfig.PreserveFormatting {
					merged.PreserveFormatting = true
				}
				if override.ParserConfig.MaxFileSize > 0 {
					merged.MaxFileSize = override.ParserConfig.MaxFileSize
				}
				if override.ParserConfig.Timeout > 0 {
					merged.Timeout = override.ParserConfig.Timeout
				}
				result.ParserConfig = &merged
			}
		}
		
		// Merge chunker config
		if override.ChunkerConfig != nil {
			if result.ChunkerConfig == nil {
				result.ChunkerConfig = override.ChunkerConfig
			} else {
				merged := *result.ChunkerConfig
				if override.ChunkerConfig.ChunkSize > 0 {
					merged.ChunkSize = override.ChunkerConfig.ChunkSize
				}
				if override.ChunkerConfig.ChunkOverlap >= 0 {
					merged.ChunkOverlap = override.ChunkerConfig.ChunkOverlap
				}
				result.ChunkerConfig = &merged
			}
		}
		
		// Override chunker type if specified
		if override.ChunkerType != "" {
			result.ChunkerType = override.ChunkerType
		}
	}
	
	return &result, nil
}

// GetLanguageRule returns the language rule for a specific language
func (lpc *LanguageProcessingConfig) GetLanguageRule(language string) *LanguageRule {
	if lpc.LanguageRules == nil {
		return nil
	}
	
	rule, exists := lpc.LanguageRules[language]
	if !exists {
		return nil
	}
	
	return rule
}

// AddLanguageRule adds or updates a language rule
func (lpc *LanguageProcessingConfig) AddLanguageRule(rule *LanguageRule) {
	if lpc.LanguageRules == nil {
		lpc.LanguageRules = make(map[string]*LanguageRule)
	}
	
	lpc.LanguageRules[rule.Language] = rule
}

// DefaultLanguageRules returns default language rules for common languages
func DefaultLanguageRules() map[string]*LanguageRule {
	return map[string]*LanguageRule{
		"en": {
			Language:           "en",
			SentencePatterns:   []string{`\.`, `\!`, `\?`, `\.\.\.`},
			ParagraphRules:     []string{`\n\n`, `\r\n\r\n`},
			StopWords:         []string{"the", "a", "an", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with", "by"},
			PreferredChunkSize: 512,
			TextDirection:      "ltr",
		},
		"es": {
			Language:           "es",
			SentencePatterns:   []string{`\.`, `\!`, `\?`, `\.\.\.`},
			ParagraphRules:     []string{`\n\n`, `\r\n\r\n`},
			StopWords:         []string{"el", "la", "de", "que", "y", "a", "en", "un", "es", "se", "no", "te", "lo", "le"},
			PreferredChunkSize: 512,
			TextDirection:      "ltr",
		},
		"fr": {
			Language:           "fr",
			SentencePatterns:   []string{`\.`, `\!`, `\?`, `\.\.\.`},
			ParagraphRules:     []string{`\n\n`, `\r\n\r\n`},
			StopWords:         []string{"le", "de", "et", "à", "un", "il", "être", "et", "en", "avoir", "que", "pour"},
			PreferredChunkSize: 512,
			TextDirection:      "ltr",
		},
		"de": {
			Language:           "de",
			SentencePatterns:   []string{`\.`, `\!`, `\?`, `\.\.\.`},
			ParagraphRules:     []string{`\n\n`, `\r\n\r\n`},
			StopWords:         []string{"der", "die", "und", "in", "den", "von", "zu", "das", "mit", "sich", "des", "auf"},
			PreferredChunkSize: 600, // German words tend to be longer
			TextDirection:      "ltr",
		},
		"zh": {
			Language:           "zh",
			SentencePatterns:   []string{`。`, `！`, `？`, `...`},
			ParagraphRules:     []string{`\n\n`, `\r\n\r\n`},
			StopWords:         []string{"的", "一", "是", "在", "不", "了", "有", "和", "人", "这", "中", "大"},
			PreferredChunkSize: 400, // Chinese characters are denser
			TextDirection:      "ltr",
		},
	}
}