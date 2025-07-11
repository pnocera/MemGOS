package parsers

import (
	"context"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"
	"time"
)

// ParserFactory provides a factory for creating different types of parsers
type ParserFactory struct {
	// parsers maps parser types to parser instances
	parsers map[ParserType]Parser
	
	// mimeTypeMap maps MIME types to parser types
	mimeTypeMap map[string]ParserType
	
	// extensionMap maps file extensions to parser types
	extensionMap map[string]ParserType
}

// NewParserFactory creates a new parser factory with all available parsers
func NewParserFactory() *ParserFactory {
	factory := &ParserFactory{
		parsers:      make(map[ParserType]Parser),
		mimeTypeMap:  make(map[string]ParserType),
		extensionMap: make(map[string]ParserType),
	}
	
	// Register all available parsers
	factory.registerDefaultParsers()
	
	return factory
}

// registerDefaultParsers registers all default parser implementations
func (pf *ParserFactory) registerDefaultParsers() {
	// Register text parser
	textParser := NewTextParser()
	pf.RegisterParser(ParserTypeText, textParser)
	
	// Register HTML parser
	htmlParser := NewHTMLParser()
	pf.RegisterParser(ParserTypeHTML, htmlParser)
	
	// Register Markdown parser
	markdownParser := NewMarkdownParser()
	pf.RegisterParser(ParserTypeMarkdown, markdownParser)
	
	// Register PDF parser
	pdfParser := NewPDFParser()
	pf.RegisterParser(ParserTypePDF, pdfParser)
	
	// Register JSON parser
	jsonParser := NewJSONParser()
	pf.RegisterParser(ParserTypeJSON, jsonParser)
	
	// Register CSV parser
	csvParser := NewCSVParser()
	pf.RegisterParser(ParserTypeCSV, csvParser)
	
	// Note: Additional parsers can be registered here
	// For example: DOCX, RTF, XML, EPUB parsers
}

// RegisterParser registers a parser for a specific type
func (pf *ParserFactory) RegisterParser(parserType ParserType, parser Parser) error {
	if !IsValidParserType(parserType) {
		return fmt.Errorf("invalid parser type: %s", parserType)
	}
	
	if parser == nil {
		return fmt.Errorf("parser cannot be nil")
	}
	
	// Register the parser
	pf.parsers[parserType] = parser
	
	// Register MIME types
	for _, mimeType := range parser.SupportedTypes() {
		pf.mimeTypeMap[strings.ToLower(mimeType)] = parserType
	}
	
	// Register file extensions
	for _, ext := range parser.SupportedExtensions() {
		pf.extensionMap[strings.ToLower(ext)] = parserType
	}
	
	return nil
}

// GetParser retrieves a parser by type
func (pf *ParserFactory) GetParser(parserType ParserType) (Parser, error) {
	parser, exists := pf.parsers[parserType]
	if !exists {
		return nil, fmt.Errorf("parser not found for type: %s", parserType)
	}
	return parser, nil
}

// GetParserByMimeType retrieves a parser by MIME type
func (pf *ParserFactory) GetParserByMimeType(mimeType string) (Parser, error) {
	parserType, exists := pf.mimeTypeMap[strings.ToLower(mimeType)]
	if !exists {
		return nil, fmt.Errorf("no parser found for MIME type: %s", mimeType)
	}
	return pf.GetParser(parserType)
}

// GetParserByExtension retrieves a parser by file extension
func (pf *ParserFactory) GetParserByExtension(extension string) (Parser, error) {
	if !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}
	
	parserType, exists := pf.extensionMap[strings.ToLower(extension)]
	if !exists {
		return nil, fmt.Errorf("no parser found for extension: %s", extension)
	}
	return pf.GetParser(parserType)
}

// DetectParserByContent detects the appropriate parser by analyzing content
func (pf *ParserFactory) DetectParserByContent(ctx context.Context, reader io.Reader) (Parser, error) {
	// Read a sample of content for detection
	buffer := make([]byte, 2048)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read content for detection: %w", err)
	}
	
	if n == 0 {
		return nil, fmt.Errorf("empty content")
	}
	
	content := buffer[:n]
	
	// Try to detect content type
	if parserType := pf.detectContentType(content); parserType != "" {
		return pf.GetParser(parserType)
	}
	
	// Fallback to text parser for unknown content
	return pf.GetParser(ParserTypeText)
}

// detectContentType detects content type from byte content
func (pf *ParserFactory) detectContentType(content []byte) ParserType {
	contentStr := string(content)
	contentLower := strings.ToLower(contentStr)
	
	// Check for PDF magic bytes
	if len(content) >= 5 && string(content[:5]) == "%PDF-" {
		return ParserTypePDF
	}
	
	// Check for HTML patterns
	htmlPatterns := []string{
		"<!doctype html",
		"<html",
		"<head>",
		"<body>",
		"<div",
		"<span",
		"<p>",
	}
	for _, pattern := range htmlPatterns {
		if strings.Contains(contentLower, pattern) {
			return ParserTypeHTML
		}
	}
	
	// Check for Markdown patterns
	markdownPatterns := []string{
		"# ",      // Heading
		"## ",     // Subheading
		"```",     // Code block
		"---",     // Horizontal rule or front matter
		"[",       // Link start
		"![",      // Image
		"* ",      // List item
		"- ",      // List item
		"+ ",      // List item
	}
	markdownScore := 0
	for _, pattern := range markdownPatterns {
		if strings.Contains(contentStr, pattern) {
			markdownScore++
		}
	}
	if markdownScore >= 2 {
		return ParserTypeMarkdown
	}
	
	// Default to text for unknown content
	return ParserTypeText
}

// CreateParserFromFilename creates appropriate parser based on filename
func (pf *ParserFactory) CreateParserFromFilename(filename string) (Parser, error) {
	if filename == "" {
		return nil, fmt.Errorf("filename cannot be empty")
	}
	
	// Try by extension first
	ext := filepath.Ext(filename)
	if ext != "" {
		if parser, err := pf.GetParserByExtension(ext); err == nil {
			return parser, nil
		}
	}
	
	// Try by MIME type
	mimeType := mime.TypeByExtension(ext)
	if mimeType != "" {
		// Remove charset information
		if idx := strings.Index(mimeType, ";"); idx != -1 {
			mimeType = mimeType[:idx]
		}
		if parser, err := pf.GetParserByMimeType(mimeType); err == nil {
			return parser, nil
		}
	}
	
	// Fallback to text parser
	return pf.GetParser(ParserTypeText)
}

// ParseWithBestParser automatically selects and uses the best parser for the content
func (pf *ParserFactory) ParseWithBestParser(ctx context.Context, reader io.Reader, filename string, config *ParserConfig) (*ParsedDocument, error) {
	if config == nil {
		config = DefaultParserConfig()
	}
	
	var parser Parser
	var err error
	
	// Try to determine parser by filename if provided
	if filename != "" {
		parser, err = pf.CreateParserFromFilename(filename)
		if err == nil {
			// Validate that the parser can handle the content
			if err := parser.ValidateInput(ctx, reader); err == nil {
				// Parse with the filename-based parser
				if filename != "" {
					return parser.ParseBytes(ctx, pf.readerToBytes(reader), filename, config)
				}
				return parser.Parse(ctx, reader, config)
			}
		}
	}
	
	// Fallback: detect by content
	parser, err = pf.DetectParserByContent(ctx, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to detect appropriate parser: %w", err)
	}
	
	// Parse with the detected parser
	if filename != "" {
		return parser.ParseBytes(ctx, pf.readerToBytes(reader), filename, config)
	}
	return parser.Parse(ctx, reader, config)
}

// GetSupportedTypes returns all supported MIME types
func (pf *ParserFactory) GetSupportedTypes() []string {
	var types []string
	for mimeType := range pf.mimeTypeMap {
		types = append(types, mimeType)
	}
	return types
}

// GetSupportedExtensions returns all supported file extensions
func (pf *ParserFactory) GetSupportedExtensions() []string {
	var extensions []string
	for ext := range pf.extensionMap {
		extensions = append(extensions, ext)
	}
	return extensions
}

// GetRegisteredParsers returns all registered parser types
func (pf *ParserFactory) GetRegisteredParsers() []ParserType {
	var parserTypes []ParserType
	for parserType := range pf.parsers {
		parserTypes = append(parserTypes, parserType)
	}
	return parserTypes
}

// IsTypeSupported checks if a MIME type is supported
func (pf *ParserFactory) IsTypeSupported(mimeType string) bool {
	_, exists := pf.mimeTypeMap[strings.ToLower(mimeType)]
	return exists
}

// IsExtensionSupported checks if a file extension is supported
func (pf *ParserFactory) IsExtensionSupported(extension string) bool {
	if !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}
	_, exists := pf.extensionMap[strings.ToLower(extension)]
	return exists
}

// GetParserInfo returns information about a parser
func (pf *ParserFactory) GetParserInfo(parserType ParserType) (*ParserInfo, error) {
	parser, err := pf.GetParser(parserType)
	if err != nil {
		return nil, err
	}
	
	return &ParserInfo{
		Type:                parserType,
		SupportedTypes:      parser.SupportedTypes(),
		SupportedExtensions: parser.SupportedExtensions(),
		Description:         pf.getParserDescription(parserType),
	}, nil
}

// ParserInfo contains information about a parser
type ParserInfo struct {
	Type                ParserType `json:"type"`
	SupportedTypes      []string   `json:"supported_types"`
	SupportedExtensions []string   `json:"supported_extensions"`
	Description         string     `json:"description"`
}

// getParserDescription returns a description for a parser type
func (pf *ParserFactory) getParserDescription(parserType ParserType) string {
	descriptions := map[ParserType]string{
		ParserTypeText:     "Plain text parser for .txt, .log, and other text files",
		ParserTypeHTML:     "HTML parser for web pages and HTML documents",
		ParserTypeMarkdown: "Markdown parser for .md and .markdown files",
		ParserTypePDF:      "PDF parser for Adobe PDF documents",
		ParserTypeDOCX:     "Microsoft Word document parser",
		ParserTypeRTF:      "Rich Text Format parser",
		ParserTypeCSV:      "Comma-separated values parser",
		ParserTypeJSON:     "JSON document parser",
		ParserTypeXML:      "XML document parser",
		ParserTypeEPUB:     "EPUB ebook parser",
	}
	
	if desc, exists := descriptions[parserType]; exists {
		return desc
	}
	return "Parser for " + string(parserType) + " format"
}

// ValidateParserConfiguration validates a parser configuration
func (pf *ParserFactory) ValidateParserConfiguration(config *ParserConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	
	if config.MaxFileSize < 0 {
		return fmt.Errorf("max file size cannot be negative")
	}
	
	if config.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}
	
	return nil
}

// CreateDefaultConfig creates a default configuration for a parser type
func (pf *ParserFactory) CreateDefaultConfig(parserType ParserType) *ParserConfig {
	config := DefaultParserConfig()
	
	// Customize configuration based on parser type
	switch parserType {
	case ParserTypePDF:
		config.ExtractImages = true
		config.ExtractTables = true
		config.OCREnabled = false // Can be enabled if OCR library is available
		config.Timeout = 60 * time.Second // PDFs can take longer
		
	case ParserTypeHTML:
		config.ExtractImages = true
		config.ExtractTables = true
		config.PreserveFormatting = false // Usually want plain text from HTML
		
	case ParserTypeMarkdown:
		config.ExtractStructure = true
		config.PreserveFormatting = true
		
	case ParserTypeText:
		config.ExtractStructure = false // Plain text has minimal structure
		config.PreserveFormatting = true
	}
	
	return config
}

// Helper method to convert reader to bytes
func (pf *ParserFactory) readerToBytes(reader io.Reader) []byte {
	// This is a simplified implementation
	// In practice, you might want to handle this more efficiently
	if data, err := io.ReadAll(reader); err == nil {
		return data
	}
	return []byte{}
}

// BatchParseInfo contains information for batch parsing
type BatchParseInfo struct {
	Filename   string        `json:"filename"`
	Parser     ParserType    `json:"parser"`
	Success    bool          `json:"success"`
	Error      string        `json:"error,omitempty"`
	Duration   time.Duration `json:"duration"`
	OutputSize int           `json:"output_size"`
}

// BatchParse parses multiple documents in a batch
func (pf *ParserFactory) BatchParse(ctx context.Context, files map[string]io.Reader, config *ParserConfig) (map[string]*ParsedDocument, []BatchParseInfo) {
	if config == nil {
		config = DefaultParserConfig()
	}
	
	results := make(map[string]*ParsedDocument)
	var infos []BatchParseInfo
	
	for filename, reader := range files {
		startTime := time.Now()
		info := BatchParseInfo{
			Filename: filename,
		}
		
		doc, err := pf.ParseWithBestParser(ctx, reader, filename, config)
		info.Duration = time.Since(startTime)
		
		if err != nil {
			info.Success = false
			info.Error = err.Error()
		} else {
			info.Success = true
			info.OutputSize = len(doc.Content)
			results[filename] = doc
			
			// Determine which parser was used
			info.Parser = ParserType(doc.ParserType)
		}
		
		infos = append(infos, info)
	}
	
	return results, infos
}