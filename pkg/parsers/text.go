package parsers

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// TextParser implements parsing for plain text files
type TextParser struct{}

// NewTextParser creates a new text parser
func NewTextParser() *TextParser {
	return &TextParser{}
}

// Parse parses a text document from a reader
func (tp *TextParser) Parse(ctx context.Context, reader io.Reader, config *ParserConfig) (*ParsedDocument, error) {
	if config == nil {
		config = DefaultParserConfig()
	}
	
	startTime := time.Now()
	
	// Read all content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read text content: %w", err)
	}
	
	// Check file size limit
	if config.MaxFileSize > 0 && int64(len(content)) > config.MaxFileSize {
		return nil, fmt.Errorf("file size %d bytes exceeds limit %d bytes", len(content), config.MaxFileSize)
	}
	
	// Validate UTF-8 encoding
	if !utf8.Valid(content) {
		return nil, fmt.Errorf("content is not valid UTF-8")
	}
	
	textContent := string(content)
	
	// Create document metadata
	metadata := &DocumentMetadata{
		MimeType:       "text/plain",
		FileSize:       int64(len(content)),
		WordCount:      tp.countWords(textContent),
		CharacterCount: len(textContent),
		Language:       tp.detectLanguage(textContent),
	}
	
	// Extract structured content if requested
	var structuredContent *StructuredContent
	if config.ExtractStructure {
		structuredContent = tp.extractStructure(textContent)
	}
	
	// Create parsed document
	doc := &ParsedDocument{
		Content:           tp.cleanContent(textContent, config),
		Metadata:          metadata,
		StructuredContent: structuredContent,
		ParsedAt:          time.Now(),
		ParserType:        string(ParserTypeText),
		ParsingDuration:   time.Since(startTime),
	}
	
	return doc, nil
}

// ParseFile parses a text document from a file path
func (tp *TextParser) ParseFile(ctx context.Context, filePath string, config *ParserConfig) (*ParsedDocument, error) {
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
	doc, err := tp.Parse(ctx, file, config)
	if err != nil {
		return nil, err
	}
	
	// Add file-specific metadata
	if doc.Metadata != nil {
		doc.Metadata.FileExtension = strings.ToLower(filepath.Ext(filePath))
		doc.Metadata.CreatedAt = &fileInfo.ModTime()
		doc.Metadata.ModifiedAt = &fileInfo.ModTime()
		
		// Extract title from filename
		filename := filepath.Base(filePath)
		if ext := filepath.Ext(filename); ext != "" {
			filename = filename[:len(filename)-len(ext)]
		}
		doc.Metadata.Title = filename
	}
	
	return doc, nil
}

// ParseBytes parses a text document from byte data
func (tp *TextParser) ParseBytes(ctx context.Context, data []byte, filename string, config *ParserConfig) (*ParsedDocument, error) {
	if config == nil {
		config = DefaultParserConfig()
	}
	
	reader := strings.NewReader(string(data))
	doc, err := tp.Parse(ctx, reader, config)
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
func (tp *TextParser) SupportedTypes() []string {
	return []string{
		"text/plain",
		"text/x-log",
		"application/x-log",
		"text/x-readme",
	}
}

// SupportedExtensions returns the file extensions supported by this parser
func (tp *TextParser) SupportedExtensions() []string {
	return []string{
		".txt", ".text", ".log", ".readme", ".md", ".rst",
		".asc", ".conf", ".cfg", ".ini", ".properties",
		".yml", ".yaml", ".sh", ".bat", ".cmd",
	}
}

// GetParserType returns the type identifier for this parser
func (tp *TextParser) GetParserType() string {
	return string(ParserTypeText)
}

// ValidateInput validates if the input can be parsed by this parser
func (tp *TextParser) ValidateInput(ctx context.Context, reader io.Reader) error {
	// Read a small sample to validate
	buffer := make([]byte, 1024)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read input: %w", err)
	}
	
	if n == 0 {
		return nil // Empty file is valid
	}
	
	// Check if content is valid UTF-8
	if !utf8.Valid(buffer[:n]) {
		return fmt.Errorf("content is not valid UTF-8 text")
	}
	
	// Check if content contains too many binary characters
	printableChars := 0
	for _, b := range buffer[:n] {
		if unicode.IsPrint(rune(b)) || unicode.IsSpace(rune(b)) {
			printableChars++
		}
	}
	
	printableRatio := float64(printableChars) / float64(n)
	if printableRatio < 0.8 {
		return fmt.Errorf("content appears to be binary (%.2f%% printable)", printableRatio*100)
	}
	
	return nil
}

// EstimateComplexity estimates parsing complexity (0-100 scale)
func (tp *TextParser) EstimateComplexity(ctx context.Context, reader io.Reader) (int, error) {
	// Read a sample to estimate complexity
	buffer := make([]byte, 4096)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return 0, fmt.Errorf("failed to read input: %w", err)
	}
	
	if n == 0 {
		return 0, nil // Empty file has zero complexity
	}
	
	content := string(buffer[:n])
	
	// Base complexity is very low for plain text
	complexity := 5
	
	// Increase complexity based on content characteristics
	lines := strings.Split(content, "\n")
	if len(lines) > 100 {
		complexity += 10 // Many lines
	}
	
	// Check for structured content
	if tp.hasStructuredContent(content) {
		complexity += 15
	}
	
	// Check for special characters or formatting
	specialChars := 0
	for _, r := range content {
		if !unicode.IsLetter(r) && !unicode.IsSpace(r) && !unicode.IsDigit(r) {
			specialChars++
		}
	}
	
	if specialChars > len(content)/10 {
		complexity += 10 // Many special characters
	}
	
	// Cap at reasonable maximum for text files
	if complexity > 40 {
		complexity = 40
	}
	
	return complexity, nil
}

// cleanContent cleans and processes the text content
func (tp *TextParser) cleanContent(content string, config *ParserConfig) string {
	if !config.PreserveFormatting {
		// Normalize whitespace
		content = tp.normalizeWhitespace(content)
	}
	
	// Remove BOM if present
	content = strings.TrimPrefix(content, "\ufeff")
	
	return content
}

// normalizeWhitespace normalizes whitespace in text
func (tp *TextParser) normalizeWhitespace(text string) string {
	// Replace multiple spaces with single space
	text = strings.Join(strings.Fields(text), " ")
	
	// Normalize line endings
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	
	return text
}

// countWords counts the number of words in text
func (tp *TextParser) countWords(text string) int {
	return len(strings.Fields(text))
}

// detectLanguage attempts to detect the language of the text
func (tp *TextParser) detectLanguage(text string) string {
	// Simple heuristic-based language detection
	// In a real implementation, you'd use a proper language detection library
	
	sample := text
	if len(sample) > 1000 {
		sample = sample[:1000]
	}
	
	// Count common English words
	englishWords := []string{"the", "and", "or", "is", "are", "was", "were", "a", "an", "this", "that"}
	englishCount := 0
	words := strings.Fields(strings.ToLower(sample))
	
	for _, word := range words {
		for _, englishWord := range englishWords {
			if word == englishWord {
				englishCount++
				break
			}
		}
	}
	
	if len(words) > 0 && float64(englishCount)/float64(len(words)) > 0.1 {
		return "en"
	}
	
	return "unknown"
}

// extractStructure extracts structural elements from plain text
func (tp *TextParser) extractStructure(content string) *StructuredContent {
	structure := &StructuredContent{}
	
	lines := strings.Split(content, "\n")
	var paragraphs []string
	var currentParagraph strings.Builder
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if line == "" {
			// Empty line - end current paragraph
			if currentParagraph.Len() > 0 {
				paragraphs = append(paragraphs, currentParagraph.String())
				currentParagraph.Reset()
			}
			continue
		}
		
		// Check for heading-like patterns
		if tp.isHeadingLike(line) {
			// Finish current paragraph
			if currentParagraph.Len() > 0 {
				paragraphs = append(paragraphs, currentParagraph.String())
				currentParagraph.Reset()
			}
			
			// Add as heading
			structure.Headings = append(structure.Headings, Heading{
				Level: tp.estimateHeadingLevel(line),
				Text:  line,
			})
			continue
		}
		
		// Check for list items
		if tp.isListItem(line) {
			// Finish current paragraph
			if currentParagraph.Len() > 0 {
				paragraphs = append(paragraphs, currentParagraph.String())
				currentParagraph.Reset()
			}
			
			// Add to lists (simplified - would need better list grouping)
			continue
		}
		
		// Add to current paragraph
		if currentParagraph.Len() > 0 {
			currentParagraph.WriteString(" ")
		}
		currentParagraph.WriteString(line)
	}
	
	// Add final paragraph
	if currentParagraph.Len() > 0 {
		paragraphs = append(paragraphs, currentParagraph.String())
	}
	
	structure.Paragraphs = paragraphs
	
	return structure
}

// isHeadingLike checks if a line looks like a heading
func (tp *TextParser) isHeadingLike(line string) bool {
	// Check for all caps (likely heading)
	if strings.ToUpper(line) == line && len(line) > 0 && len(line) < 100 {
		return true
	}
	
	// Check for numbered sections
	if strings.HasPrefix(line, "1.") || strings.HasPrefix(line, "I.") || 
	   strings.HasPrefix(line, "Chapter") || strings.HasPrefix(line, "Section") {
		return true
	}
	
	// Check for underlined headings (next line would be dashes/equals)
	return false
}

// estimateHeadingLevel estimates the level of a heading
func (tp *TextParser) estimateHeadingLevel(line string) int {
	// Simple heuristics
	if strings.HasPrefix(line, "Chapter") {
		return 1
	}
	if strings.HasPrefix(line, "Section") {
		return 2
	}
	if strings.Contains(line, ".") {
		return 3
	}
	return 2 // Default level
}

// isListItem checks if a line looks like a list item
func (tp *TextParser) isListItem(line string) bool {
	// Check for bullet points
	if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || 
	   strings.HasPrefix(line, "+ ") || strings.HasPrefix(line, "• ") {
		return true
	}
	
	// Check for numbered lists
	if len(line) > 3 && line[1] == '.' && line[2] == ' ' {
		return unicode.IsDigit(rune(line[0]))
	}
	
	return false
}

// hasStructuredContent checks if content has structured elements
func (tp *TextParser) hasStructuredContent(content string) bool {
	lines := strings.Split(content, "\n")
	structuredLines := 0
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if tp.isHeadingLike(line) || tp.isListItem(line) {
			structuredLines++
		}
	}
	
	return structuredLines > 0 && float64(structuredLines)/float64(len(lines)) > 0.05
}