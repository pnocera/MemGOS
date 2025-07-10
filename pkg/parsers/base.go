// Package parsers provides document parsing functionality for MemGOS
package parsers

import (
	"context"
	"io"
	"time"
)

// DocumentMetadata contains metadata extracted from parsed documents
type DocumentMetadata struct {
	// Title of the document
	Title string `json:"title,omitempty"`
	
	// Author of the document
	Author string `json:"author,omitempty"`
	
	// Subject/topic of the document
	Subject string `json:"subject,omitempty"`
	
	// Language of the document
	Language string `json:"language,omitempty"`
	
	// CreatedAt is when the document was created
	CreatedAt *time.Time `json:"created_at,omitempty"`
	
	// ModifiedAt is when the document was last modified
	ModifiedAt *time.Time `json:"modified_at,omitempty"`
	
	// FileSize is the size of the original file in bytes
	FileSize int64 `json:"file_size,omitempty"`
	
	// MimeType is the MIME type of the document
	MimeType string `json:"mime_type,omitempty"`
	
	// FileExtension is the file extension
	FileExtension string `json:"file_extension,omitempty"`
	
	// PageCount for multi-page documents
	PageCount int `json:"page_count,omitempty"`
	
	// WordCount approximate word count
	WordCount int `json:"word_count,omitempty"`
	
	// CharacterCount total character count
	CharacterCount int `json:"character_count,omitempty"`
	
	// Keywords extracted from the document
	Keywords []string `json:"keywords,omitempty"`
	
	// Custom metadata fields
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// ParsedDocument represents a parsed document with content and metadata
type ParsedDocument struct {
	// Content is the extracted text content
	Content string `json:"content"`
	
	// Metadata contains document metadata
	Metadata *DocumentMetadata `json:"metadata"`
	
	// StructuredContent contains parsed structured elements
	StructuredContent *StructuredContent `json:"structured_content,omitempty"`
	
	// ParsedAt is when the document was parsed
	ParsedAt time.Time `json:"parsed_at"`
	
	// ParserType indicates which parser was used
	ParserType string `json:"parser_type"`
	
	// ParsingDuration is how long parsing took
	ParsingDuration time.Duration `json:"parsing_duration"`
}

// StructuredContent contains structured elements extracted from documents
type StructuredContent struct {
	// Headings contains document headings with hierarchy levels
	Headings []Heading `json:"headings,omitempty"`
	
	// Paragraphs contains text paragraphs
	Paragraphs []string `json:"paragraphs,omitempty"`
	
	// Lists contains bulleted or numbered lists
	Lists []List `json:"lists,omitempty"`
	
	// Tables contains tabular data
	Tables []Table `json:"tables,omitempty"`
	
	// Images contains image references and alt text
	Images []Image `json:"images,omitempty"`
	
	// Links contains hyperlinks
	Links []Link `json:"links,omitempty"`
	
	// CodeBlocks contains code snippets
	CodeBlocks []CodeBlock `json:"code_blocks,omitempty"`
	
	// Footnotes contains footnote text
	Footnotes []Footnote `json:"footnotes,omitempty"`
}

// Heading represents a document heading
type Heading struct {
	Level int    `json:"level"`  // 1-6 for HTML-style headings
	Text  string `json:"text"`
	ID    string `json:"id,omitempty"`
}

// List represents a bulleted or numbered list
type List struct {
	Type  string   `json:"type"`  // "ordered" or "unordered"
	Items []string `json:"items"`
}

// Table represents tabular data
type Table struct {
	Headers []string   `json:"headers,omitempty"`
	Rows    [][]string `json:"rows"`
	Caption string     `json:"caption,omitempty"`
}

// Image represents an image reference
type Image struct {
	URL     string `json:"url,omitempty"`
	AltText string `json:"alt_text,omitempty"`
	Caption string `json:"caption,omitempty"`
	Width   int    `json:"width,omitempty"`
	Height  int    `json:"height,omitempty"`
}

// Link represents a hyperlink
type Link struct {
	URL  string `json:"url"`
	Text string `json:"text"`
}

// CodeBlock represents a code snippet
type CodeBlock struct {
	Language string `json:"language,omitempty"`
	Code     string `json:"code"`
}

// Footnote represents a footnote
type Footnote struct {
	Number int    `json:"number"`
	Text   string `json:"text"`
}

// ParserConfig represents base configuration for document parsers
type ParserConfig struct {
	// ExtractMetadata indicates whether to extract document metadata
	ExtractMetadata bool `json:"extract_metadata"`
	
	// ExtractStructure indicates whether to extract structural elements
	ExtractStructure bool `json:"extract_structure"`
	
	// PreserveFormatting indicates whether to preserve text formatting
	PreserveFormatting bool `json:"preserve_formatting"`
	
	// MaxFileSize is the maximum file size to process (in bytes)
	MaxFileSize int64 `json:"max_file_size"`
	
	// Timeout for parsing operations
	Timeout time.Duration `json:"timeout"`
	
	// Language hint for better parsing
	Language string `json:"language,omitempty"`
	
	// ExtractImages indicates whether to process images
	ExtractImages bool `json:"extract_images"`
	
	// ExtractTables indicates whether to extract table data
	ExtractTables bool `json:"extract_tables"`
	
	// OCREnabled indicates whether to use OCR for image text
	OCREnabled bool `json:"ocr_enabled"`
	
	// Custom configuration options
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// DefaultParserConfig returns a sensible default configuration
func DefaultParserConfig() *ParserConfig {
	return &ParserConfig{
		ExtractMetadata:    true,
		ExtractStructure:   true,
		PreserveFormatting: true,
		MaxFileSize:        100 * 1024 * 1024, // 100MB
		Timeout:            30 * time.Second,
		Language:           "auto",
		ExtractImages:      true,
		ExtractTables:      true,
		OCREnabled:         false,
		Custom:             make(map[string]interface{}),
	}
}

// Parser defines the interface for all document parsing implementations
type Parser interface {
	// Parse parses a document from a reader and returns structured content
	Parse(ctx context.Context, reader io.Reader, config *ParserConfig) (*ParsedDocument, error)
	
	// ParseFile parses a document from a file path
	ParseFile(ctx context.Context, filePath string, config *ParserConfig) (*ParsedDocument, error)
	
	// ParseBytes parses a document from byte data
	ParseBytes(ctx context.Context, data []byte, filename string, config *ParserConfig) (*ParsedDocument, error)
	
	// SupportedTypes returns the MIME types supported by this parser
	SupportedTypes() []string
	
	// SupportedExtensions returns the file extensions supported by this parser
	SupportedExtensions() []string
	
	// GetParserType returns the type identifier for this parser
	GetParserType() string
	
	// ValidateInput validates if the input can be parsed by this parser
	ValidateInput(ctx context.Context, reader io.Reader) error
	
	// EstimateComplexity estimates parsing complexity (0-100 scale)
	EstimateComplexity(ctx context.Context, reader io.Reader) (int, error)
}

// StreamingParser defines the interface for parsers that support streaming
type StreamingParser interface {
	Parser
	
	// ParseStream parses a document in streaming mode, sending chunks to a channel
	ParseStream(ctx context.Context, reader io.Reader, config *ParserConfig, chunkChan chan<- string) error
}

// MultiFormatParser defines the interface for parsers that can handle multiple formats
type MultiFormatParser interface {
	Parser
	
	// DetectFormat detects the format of the input data
	DetectFormat(ctx context.Context, reader io.Reader) (string, error)
	
	// ParseWithFormat parses using a specific format hint
	ParseWithFormat(ctx context.Context, reader io.Reader, format string, config *ParserConfig) (*ParsedDocument, error)
}

// ParserType represents different parser implementations
type ParserType string

const (
	// ParserTypeText for plain text files
	ParserTypeText ParserType = "text"
	
	// ParserTypePDF for PDF documents
	ParserTypePDF ParserType = "pdf"
	
	// ParserTypeHTML for HTML documents
	ParserTypeHTML ParserType = "html"
	
	// ParserTypeMarkdown for Markdown documents
	ParserTypeMarkdown ParserType = "markdown"
	
	// ParserTypeDOCX for Microsoft Word documents
	ParserTypeDOCX ParserType = "docx"
	
	// ParserTypeRTF for Rich Text Format documents
	ParserTypeRTF ParserType = "rtf"
	
	// ParserTypeCSV for CSV files
	ParserTypeCSV ParserType = "csv"
	
	// ParserTypeJSON for JSON files
	ParserTypeJSON ParserType = "json"
	
	// ParserTypeXML for XML files
	ParserTypeXML ParserType = "xml"
	
	// ParserTypeEPUB for EPUB ebooks
	ParserTypeEPUB ParserType = "epub"
	
	// ParserTypeMarkItDown for universal document parsing
	ParserTypeMarkItDown ParserType = "markitdown"
)

// SupportedParserTypes returns all supported parser types
func SupportedParserTypes() []ParserType {
	return []ParserType{
		ParserTypeText,
		ParserTypePDF,
		ParserTypeHTML,
		ParserTypeMarkdown,
		ParserTypeDOCX,
		ParserTypeRTF,
		ParserTypeCSV,
		ParserTypeJSON,
		ParserTypeXML,
		ParserTypeEPUB,
		ParserTypeMarkItDown,
	}
}

// IsValidParserType checks if a parser type is supported
func IsValidParserType(parserType ParserType) bool {
	for _, supported := range SupportedParserTypes() {
		if supported == parserType {
			return true
		}
	}
	return false
}

// ParseResult represents the result of a parsing operation
type ParseResult struct {
	// Success indicates if parsing was successful
	Success bool `json:"success"`
	
	// Document contains the parsed document (if successful)
	Document *ParsedDocument `json:"document,omitempty"`
	
	// Error contains error information (if failed)
	Error error `json:"error,omitempty"`
	
	// Warnings contains non-fatal issues encountered during parsing
	Warnings []string `json:"warnings,omitempty"`
	
	// Stats contains parsing statistics
	Stats *ParsingStats `json:"stats,omitempty"`
}

// ParsingStats provides statistics about the parsing process
type ParsingStats struct {
	// InputSize is the size of the input data in bytes
	InputSize int64 `json:"input_size"`
	
	// OutputSize is the size of the extracted content in characters
	OutputSize int `json:"output_size"`
	
	// ProcessingTime is the total time taken for parsing
	ProcessingTime time.Duration `json:"processing_time"`
	
	// MemoryUsage is the peak memory usage during parsing (if available)
	MemoryUsage int64 `json:"memory_usage,omitempty"`
	
	// ElementsCounts contains counts of different structural elements
	ElementCounts map[string]int `json:"element_counts,omitempty"`
	
	// ErrorCount is the number of errors encountered
	ErrorCount int `json:"error_count"`
	
	// WarningCount is the number of warnings encountered
	WarningCount int `json:"warning_count"`
}

// CalculateParsingStats computes statistics for a parsed document
func CalculateParsingStats(doc *ParsedDocument, inputSize int64, processingTime time.Duration) *ParsingStats {
	stats := &ParsingStats{
		InputSize:      inputSize,
		OutputSize:     len(doc.Content),
		ProcessingTime: processingTime,
		ElementCounts:  make(map[string]int),
	}
	
	if doc.StructuredContent != nil {
		stats.ElementCounts["headings"] = len(doc.StructuredContent.Headings)
		stats.ElementCounts["paragraphs"] = len(doc.StructuredContent.Paragraphs)
		stats.ElementCounts["lists"] = len(doc.StructuredContent.Lists)
		stats.ElementCounts["tables"] = len(doc.StructuredContent.Tables)
		stats.ElementCounts["images"] = len(doc.StructuredContent.Images)
		stats.ElementCounts["links"] = len(doc.StructuredContent.Links)
		stats.ElementCounts["code_blocks"] = len(doc.StructuredContent.CodeBlocks)
		stats.ElementCounts["footnotes"] = len(doc.StructuredContent.Footnotes)
	}
	
	return stats
}