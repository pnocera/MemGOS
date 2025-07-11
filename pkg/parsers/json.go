package parsers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

// JSONParser implements parsing for JSON documents
type JSONParser struct{}

// NewJSONParser creates a new JSON parser
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// Parse parses a JSON document from a reader
func (jp *JSONParser) Parse(ctx context.Context, reader io.Reader, config *ParserConfig) (*ParsedDocument, error) {
	if config == nil {
		config = DefaultParserConfig()
	}
	
	startTime := time.Now()
	
	// Read all content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON content: %w", err)
	}
	
	// Check file size limit
	if config.MaxFileSize > 0 && int64(len(content)) > config.MaxFileSize {
		return nil, fmt.Errorf("file size %d bytes exceeds limit %d bytes", len(content), config.MaxFileSize)
	}
	
	// Validate JSON
	var jsonData interface{}
	if err := json.Unmarshal(content, &jsonData); err != nil {
		return nil, fmt.Errorf("invalid JSON content: %w", err)
	}
	
	// Create document metadata
	metadata := &DocumentMetadata{
		MimeType:       "application/json",
		FileSize:       int64(len(content)),
		CharacterCount: len(content),
		Language:       "json",
	}
	
	// Extract structured content
	var structuredContent *StructuredContent
	if config.ExtractStructure {
		structuredContent = jp.extractStructure(jsonData)
	}
	
	// Convert JSON to readable text
	textContent := jp.jsonToText(jsonData, config)
	metadata.WordCount = jp.countWords(textContent)
	
	// Create parsed document
	doc := &ParsedDocument{
		Content:           textContent,
		Metadata:          metadata,
		StructuredContent: structuredContent,
		ParsedAt:          time.Now(),
		ParserType:        string(ParserTypeJSON),
		ParsingDuration:   time.Since(startTime),
	}
	
	return doc, nil
}

// ParseFile parses a JSON document from a file path
func (jp *JSONParser) ParseFile(ctx context.Context, filePath string, config *ParserConfig) (*ParsedDocument, error) {
	if config == nil {
		config = DefaultParserConfig()
	}
	
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}
	
	// Check file size limit
	if config.MaxFileSize > 0 && fileInfo.Size() > config.MaxFileSize {
		return nil, fmt.Errorf("file size %d bytes exceeds limit %d bytes", fileInfo.Size(), config.MaxFileSize)
	}
	
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	// Parse content
	doc, err := jp.Parse(ctx, file, config)
	if err != nil {
		return nil, err
	}
	
	// Add file-specific metadata
	if doc.Metadata != nil {
		doc.Metadata.FileExtension = strings.ToLower(filepath.Ext(filePath))
		modTime := fileInfo.ModTime()
		doc.Metadata.CreatedAt = &modTime
		doc.Metadata.ModifiedAt = &modTime
		
		// Extract title from filename
		filename := filepath.Base(filePath)
		if ext := filepath.Ext(filename); ext != "" {
			filename = filename[:len(filename)-len(ext)]
		}
		doc.Metadata.Title = filename
	}
	
	return doc, nil
}

// ParseBytes parses a JSON document from byte data
func (jp *JSONParser) ParseBytes(ctx context.Context, data []byte, filename string, config *ParserConfig) (*ParsedDocument, error) {
	if config == nil {
		config = DefaultParserConfig()
	}
	
	reader := strings.NewReader(string(data))
	doc, err := jp.Parse(ctx, reader, config)
	if err != nil {
		return nil, err
	}
	
	// Add filename-based metadata
	if doc.Metadata != nil && filename != "" {
		doc.Metadata.FileExtension = strings.ToLower(filepath.Ext(filename))
		
		// Extract title from filename
		name := filepath.Base(filename)
		if ext := filepath.Ext(name); ext != "" {
			name = name[:len(name)-len(ext)]
		}
		doc.Metadata.Title = name
	}
	
	return doc, nil
}

// SupportedTypes returns the MIME types supported by this parser
func (jp *JSONParser) SupportedTypes() []string {
	return []string{
		"application/json",
		"text/json",
	}
}

// SupportedExtensions returns the file extensions supported by this parser
func (jp *JSONParser) SupportedExtensions() []string {
	return []string{
		".json", ".jsonl", ".ndjson",
	}
}

// GetParserType returns the type identifier for this parser
func (jp *JSONParser) GetParserType() string {
	return string(ParserTypeJSON)
}

// ValidateInput validates if the input can be parsed by this parser
func (jp *JSONParser) ValidateInput(ctx context.Context, reader io.Reader) error {
	// Read a sample to validate
	buffer := make([]byte, 1024)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read input: %w", err)
	}
	
	if n == 0 {
		return fmt.Errorf("empty file")
	}
	
	// Try to parse as JSON
	var jsonData interface{}
	if err := json.Unmarshal(buffer[:n], &jsonData); err != nil {
		return fmt.Errorf("content does not appear to be valid JSON: %w", err)
	}
	
	return nil
}

// EstimateComplexity estimates parsing complexity (0-100 scale)
func (jp *JSONParser) EstimateComplexity(ctx context.Context, reader io.Reader) (int, error) {
	// Read a sample to estimate complexity
	buffer := make([]byte, 4096)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return 0, fmt.Errorf("failed to read input: %w", err)
	}
	
	if n == 0 {
		return 0, nil // Empty file has zero complexity
	}
	
	// Try to parse JSON to understand structure
	var jsonData interface{}
	if err := json.Unmarshal(buffer[:n], &jsonData); err != nil {
		return 90, nil // Invalid JSON is complex to handle
	}
	
	// Base complexity for JSON
	complexity := 20
	
	// Analyze JSON structure complexity
	complexity += jp.analyzeJSONComplexity(jsonData)
	
	// Analyze content size
	if n > 2048 {
		complexity += 10
	}
	if n > 8192 {
		complexity += 10
	}
	
	// Cap at reasonable maximum
	if complexity > 85 {
		complexity = 85
	}
	
	return complexity, nil
}

// analyzeJSONComplexity analyzes the complexity of JSON data structure
func (jp *JSONParser) analyzeJSONComplexity(data interface{}) int {
	complexity := 0
	
	switch v := data.(type) {
	case map[string]interface{}:
		complexity += 10 // Object adds complexity
		complexity += len(v) / 2 // More fields = more complexity
		
		// Analyze nested structures
		for _, value := range v {
			complexity += jp.analyzeJSONComplexity(value) / 4 // Reduce recursive impact
		}
		
	case []interface{}:
		complexity += 5 // Array adds some complexity
		complexity += len(v) / 10 // Many items add complexity
		
		// Analyze array elements
		if len(v) > 0 {
			complexity += jp.analyzeJSONComplexity(v[0]) / 2 // Sample first element
		}
		
	case string:
		if len(v) > 100 {
			complexity += 2 // Long strings add slight complexity
		}
		
	case float64, int, bool, nil:
		// Simple types don't add much complexity
		complexity += 1
	}
	
	return complexity
}

// extractStructure extracts structural elements from JSON data
func (jp *JSONParser) extractStructure(data interface{}) *StructuredContent {
	structure := &StructuredContent{}
	
	// Convert JSON structure to text paragraphs and extract information
	jp.extractJSONStructure(data, "", 0, structure)
	
	return structure
}

// extractJSONStructure recursively extracts structure from JSON
func (jp *JSONParser) extractJSONStructure(data interface{}, path string, depth int, structure *StructuredContent) {
	// Prevent infinite recursion
	if depth > 10 {
		return
	}
	
	switch v := data.(type) {
	case map[string]interface{}:
		// Treat objects as sections with headings
		if path != "" {
			structure.Headings = append(structure.Headings, Heading{
				Level: depth + 1,
				Text:  jp.formatPath(path),
			})
		}
		
		// Extract key-value pairs as paragraphs
		for key, value := range v {
			newPath := jp.buildPath(path, key)
			
			switch valueType := value.(type) {
			case string:
				if len(valueType) > 20 {
					structure.Paragraphs = append(structure.Paragraphs, 
						fmt.Sprintf("%s: %s", key, valueType))
				}
			case map[string]interface{}, []interface{}:
				// Recurse into nested structures
				jp.extractJSONStructure(value, newPath, depth+1, structure)
			default:
				// Simple values
				structure.Paragraphs = append(structure.Paragraphs, 
					fmt.Sprintf("%s: %v", key, value))
			}
		}
		
	case []interface{}:
		// Treat arrays as lists
		if len(v) > 0 {
			var items []string
			
			for i, item := range v {
				if i >= 10 { // Limit list items to prevent huge lists
					items = append(items, fmt.Sprintf("... and %d more items", len(v)-i))
					break
				}
				
				switch itemType := item.(type) {
				case string:
					items = append(items, itemType)
				case map[string]interface{}, []interface{}:
					// For complex items, just show the type
					items = append(items, fmt.Sprintf("[%s]", reflect.TypeOf(item).String()))
				default:
					items = append(items, fmt.Sprintf("%v", item))
				}
			}
			
			if len(items) > 0 {
				structure.Lists = append(structure.Lists, List{
					Type:  "unordered",
					Items: items,
				})
			}
		}
	}
}

// buildPath builds a JSON path string
func (jp *JSONParser) buildPath(parent, key string) string {
	if parent == "" {
		return key
	}
	return parent + "." + key
}

// formatPath formats a JSON path for display
func (jp *JSONParser) formatPath(path string) string {
	parts := strings.Split(path, ".")
	return strings.Join(parts, " > ")
}

// jsonToText converts JSON data to readable text
func (jp *JSONParser) jsonToText(data interface{}, config *ParserConfig) string {
	if config.PreserveFormatting {
		// Pretty-print JSON
		formatted, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return jp.jsonToTextRecursive(data, 0)
		}
		return string(formatted)
	}
	
	// Convert to readable plain text
	return jp.jsonToTextRecursive(data, 0)
}

// jsonToTextRecursive recursively converts JSON to readable text
func (jp *JSONParser) jsonToTextRecursive(data interface{}, depth int) string {
	indent := strings.Repeat("  ", depth)
	
	switch v := data.(type) {
	case map[string]interface{}:
		var parts []string
		for key, value := range v {
			valueText := jp.jsonToTextRecursive(value, depth+1)
			if strings.Contains(valueText, "\n") {
				parts = append(parts, fmt.Sprintf("%s%s:\n%s", indent, key, valueText))
			} else {
				parts = append(parts, fmt.Sprintf("%s%s: %s", indent, key, valueText))
			}
		}
		return strings.Join(parts, "\n")
		
	case []interface{}:
		var parts []string
		for i, item := range v {
			if i >= 20 { // Limit to prevent huge outputs
				parts = append(parts, fmt.Sprintf("%s... and %d more items", indent, len(v)-i))
				break
			}
			itemText := jp.jsonToTextRecursive(item, depth+1)
			if strings.Contains(itemText, "\n") {
				parts = append(parts, fmt.Sprintf("%s- %s", indent, itemText))
			} else {
				parts = append(parts, fmt.Sprintf("%s- %s", indent, itemText))
			}
		}
		return strings.Join(parts, "\n")
		
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.2f", v)
	case int:
		return fmt.Sprintf("%d", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// countWords counts words in the text representation
func (jp *JSONParser) countWords(text string) int {
	// For JSON, count meaningful words (not just structure)
	words := strings.Fields(text)
	meaningfulWords := 0
	
	for _, word := range words {
		// Skip JSON structure tokens
		if word == "{" || word == "}" || word == "[" || word == "]" || 
		   word == "," || word == ":" || word == "null" || word == "true" || word == "false" {
			continue
		}
		
		// Skip pure numbers and quotes
		if strings.HasPrefix(word, "\"") && strings.HasSuffix(word, "\"") {
			meaningfulWords++
		} else if len(word) > 1 {
			meaningfulWords++
		}
	}
	
	return meaningfulWords
}