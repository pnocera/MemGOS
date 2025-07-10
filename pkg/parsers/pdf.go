package parsers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
)

// PDFParser implements parsing for PDF documents
// Note: This is a basic implementation. For production use, you would integrate
// with a proper PDF parsing library like github.com/ledongthuc/pdf or similar
type PDFParser struct{}

// NewPDFParser creates a new PDF parser
func NewPDFParser() *PDFParser {
	return &PDFParser{}
}

// Parse parses a PDF document from a reader
func (pp *PDFParser) Parse(ctx context.Context, reader io.Reader, config *ParserConfig) (*ParsedDocument, error) {
	if config == nil {
		config = DefaultParserConfig()
	}
	
	startTime := time.Now()
	
	// Read all content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF content: %w", err)
	}
	
	// Check file size limit
	if config.MaxFileSize > 0 && int64(len(content)) > config.MaxFileSize {
		return nil, fmt.Errorf("file size %d bytes exceeds limit %d bytes", len(content), config.MaxFileSize)
	}
	
	// Validate PDF header
	if !pp.isPDFContent(content) {
		return nil, fmt.Errorf("content does not appear to be a valid PDF")
	}
	
	// Extract text content (simplified implementation)
	textContent, metadata := pp.extractTextAndMetadata(content)
	
	// Update metadata with file information
	metadata.MimeType = "application/pdf"
	metadata.FileSize = int64(len(content))
	metadata.CharacterCount = len(textContent)
	metadata.WordCount = pp.countWords(textContent)
	
	// Extract structured content if requested
	var structuredContent *StructuredContent
	if config.ExtractStructure {
		structuredContent = pp.extractStructure(textContent)
	}
	
	// Create parsed document
	doc := &ParsedDocument{
		Content:           textContent,
		Metadata:          metadata,
		StructuredContent: structuredContent,
		ParsedAt:          time.Now(),
		ParserType:        string(ParserTypePDF),
		ParsingDuration:   time.Since(startTime),
	}
	
	return doc, nil
}

// ParseFile parses a PDF document from a file path
func (pp *PDFParser) ParseFile(ctx context.Context, filePath string, config *ParserConfig) (*ParsedDocument, error) {
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
	doc, err := pp.Parse(ctx, file, config)
	if err != nil {
		return nil, err
	}
	
	// Add file-specific metadata
	if doc.Metadata != nil {
		doc.Metadata.FileExtension = strings.ToLower(filepath.Ext(filePath))
		doc.Metadata.CreatedAt = &fileInfo.ModTime()
		doc.Metadata.ModifiedAt = &fileInfo.ModTime()
		
		// Extract title from filename if not already set
		if doc.Metadata.Title == "" {
			filename := filepath.Base(filePath)
			if ext := filepath.Ext(filename); ext != "" {
				filename = filename[:len(filename)-len(ext)]
			}
			doc.Metadata.Title = filename
		}
	}
	
	return doc, nil
}

// ParseBytes parses a PDF document from byte data
func (pp *PDFParser) ParseBytes(ctx context.Context, data []byte, filename string, config *ParserConfig) (*ParsedDocument, error) {
	if config == nil {
		config = DefaultParserConfig()
	}
	
	reader := strings.NewReader(string(data))
	doc, err := pp.Parse(ctx, reader, config)
	if err != nil {
		return nil, err
	}
	
	// Add filename-based metadata
	if doc.Metadata != nil && filename != "" {
		doc.Metadata.FileExtension = strings.ToLower(filepath.Ext(filename))
		
		// Extract title from filename if not already set
		if doc.Metadata.Title == "" {
			name := filepath.Base(filename)
			if ext := filepath.Ext(name); ext != "" {
				name = name[:len(name)-len(ext)]
			}
			doc.Metadata.Title = name
		}
	}
	
	return doc, nil
}

// SupportedTypes returns the MIME types supported by this parser
func (pp *PDFParser) SupportedTypes() []string {
	return []string{
		"application/pdf",
	}
}

// SupportedExtensions returns the file extensions supported by this parser
func (pp *PDFParser) SupportedExtensions() []string {
	return []string{
		".pdf",
	}
}

// GetParserType returns the type identifier for this parser
func (pp *PDFParser) GetParserType() string {
	return string(ParserTypePDF)
}

// ValidateInput validates if the input can be parsed by this parser
func (pp *PDFParser) ValidateInput(ctx context.Context, reader io.Reader) error {
	// Read PDF header to validate
	buffer := make([]byte, 1024)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read input: %w", err)
	}
	
	if n == 0 {
		return fmt.Errorf("empty file")
	}
	
	if !pp.isPDFContent(buffer[:n]) {
		return fmt.Errorf("content does not appear to be a valid PDF")
	}
	
	return nil
}

// EstimateComplexity estimates parsing complexity (0-100 scale)
func (pp *PDFParser) EstimateComplexity(ctx context.Context, reader io.Reader) (int, error) {
	// Read a sample to estimate complexity
	buffer := make([]byte, 8192)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return 0, fmt.Errorf("failed to read input: %w", err)
	}
	
	if n == 0 {
		return 0, nil // Empty file has zero complexity
	}
	
	content := buffer[:n]
	
	// Base complexity for PDF
	complexity := 60 // PDFs are inherently complex
	
	// Increase complexity based on PDF features
	if pp.hasImages(content) {
		complexity += 15
	}
	
	if pp.hasForms(content) {
		complexity += 10
	}
	
	if pp.hasAnnotations(content) {
		complexity += 10
	}
	
	if pp.hasEmbeddedFonts(content) {
		complexity += 5
	}
	
	// Estimate based on PDF version
	if pp.isModernPDF(content) {
		complexity += 10
	}
	
	// Cap at maximum
	if complexity > 95 {
		complexity = 95
	}
	
	return complexity, nil
}

// isPDFContent checks if content appears to be a PDF
func (pp *PDFParser) isPDFContent(content []byte) bool {
	if len(content) < 5 {
		return false
	}
	
	// Check for PDF magic bytes
	return string(content[:5]) == "%PDF-"
}

// extractTextAndMetadata extracts text content and metadata from PDF bytes
func (pp *PDFParser) extractTextAndMetadata(content []byte) (string, *DocumentMetadata) {
	metadata := &DocumentMetadata{}
	
	// Use the pdf library to parse the PDF
	reader := bytes.NewReader(content)
	
	// Open the PDF
	pdfReader, err := pdf.NewReader(reader, int64(len(content)))
	if err != nil {
		// Fallback to basic extraction if PDF parsing fails
		return pp.extractTextFromPDFFallback(string(content)), metadata
	}
	
	// Extract metadata from PDF info
	if info := pdfReader.GetDocumentInfo(); info != nil {
		if title := info.Get("Title"); title != nil {
			if titleStr, ok := title.(string); ok {
				metadata.Title = titleStr
			}
		}
		if author := info.Get("Author"); author != nil {
			if authorStr, ok := author.(string); ok {
				metadata.Author = authorStr
			}
		}
		if subject := info.Get("Subject"); subject != nil {
			if subjectStr, ok := subject.(string); ok {
				metadata.Subject = subjectStr
			}
		}
		if keywords := info.Get("Keywords"); keywords != nil {
			if keywordsStr, ok := keywords.(string); ok {
				metadata.Keywords = strings.Split(keywordsStr, ",")
				for i, keyword := range metadata.Keywords {
					metadata.Keywords[i] = strings.TrimSpace(keyword)
				}
			}
		}
	}
	
	// Extract page count
	metadata.PageCount = pdfReader.NumPage()
	
	// Extract text content from all pages
	var textContent strings.Builder
	
	for pageNum := 1; pageNum <= pdfReader.NumPage(); pageNum++ {
		page := pdfReader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}
		
		// Extract text from the page
		pageText, err := page.GetPlainText()
		if err != nil {
			// Try to get content through other means if plain text fails
			if content := page.Content(); content != nil {
				// Extract readable text from content streams
				textContent.WriteString(pp.extractTextFromPageContent(content.Text))
				textContent.WriteString("\n")
			}
			continue
		}
		
		textContent.WriteString(pageText)
		textContent.WriteString("\n")
	}
	
	return textContent.String(), metadata
}

// extractPDFMetadata extracts metadata from PDF info dictionary
func (pp *PDFParser) extractPDFMetadata(content string, metadata *DocumentMetadata) {
	// Look for common PDF metadata fields
	// This is a very simplified approach - proper implementation would parse PDF structure
	
	patterns := map[string]string{
		"Title":    `/Title\s*\(\s*([^)]+)\s*\)`,
		"Author":   `/Author\s*\(\s*([^)]+)\s*\)`,
		"Subject":  `/Subject\s*\(\s*([^)]+)\s*\)`,
		"Keywords": `/Keywords\s*\(\s*([^)]+)\s*\)`,
	}
	
	for field, pattern := range patterns {
		// This is a very basic regex-based extraction
		// In practice, you'd properly parse the PDF structure
		// For now, we'll just set some default values
		switch field {
		case "Title":
			// Would extract title from PDF metadata
		case "Author":
			// Would extract author from PDF metadata
		case "Subject":
			// Would extract subject from PDF metadata
		case "Keywords":
			// Would extract keywords from PDF metadata
		}
	}
	
	// Set some basic defaults since this is a simplified implementation
	metadata.Language = "en"
}

// extractTextFromPDFFallback provides fallback text extraction for PDFs that can't be parsed
func (pp *PDFParser) extractTextFromPDFFallback(content string) string {
	// Basic fallback that looks for readable text patterns in PDF
	if strings.Contains(content, "%PDF-") {
		return "[PDF content detected - text extraction failed, consider using OCR for scanned documents]"
	}
	return ""
}

// extractTextFromPageContent extracts readable text from PDF page content streams
func (pp *PDFParser) extractTextFromPageContent(content string) string {
	// Look for text objects in PDF content streams
	// This is a simplified approach that looks for text between BT and ET operators
	btEtRegex := regexp.MustCompile(`BT\s+(.*?)\s+ET`)
	matches := btEtRegex.FindAllStringSubmatch(content, -1)
	
	var textParts []string
	for _, match := range matches {
		if len(match) > 1 {
			// Extract text from Tj and TJ operators
			textContent := match[1]
			
			// Look for Tj operators (show text)
			tjRegex := regexp.MustCompile(`\((.*?)\)\s*Tj`)
			tjMatches := tjRegex.FindAllStringSubmatch(textContent, -1)
			
			for _, tjMatch := range tjMatches {
				if len(tjMatch) > 1 {
					text := pp.decodePDFString(tjMatch[1])
					if text != "" {
						textParts = append(textParts, text)
					}
				}
			}
			
			// Look for TJ operators (show text with positioning)
			tjArrayRegex := regexp.MustCompile(`\[(.*?)\]\s*TJ`)
			tjArrayMatches := tjArrayRegex.FindAllStringSubmatch(textContent, -1)
			
			for _, tjArrayMatch := range tjArrayMatches {
				if len(tjArrayMatch) > 1 {
					// Parse array content
					arrayContent := tjArrayMatch[1]
					stringRegex := regexp.MustCompile(`\(([^)]*)\)`)
					stringMatches := stringRegex.FindAllStringSubmatch(arrayContent, -1)
					
					for _, stringMatch := range stringMatches {
						if len(stringMatch) > 1 {
							text := pp.decodePDFString(stringMatch[1])
							if text != "" {
								textParts = append(textParts, text)
							}
						}
					}
				}
			}
		}
	}
	
	return strings.Join(textParts, " ")
}

// decodePDFString decodes PDF string literals
func (pp *PDFParser) decodePDFString(encoded string) string {
	// Handle basic PDF string encoding
	// This is a simplified implementation - full PDF string decoding is complex
	
	// Remove escape sequences
	encoded = strings.ReplaceAll(encoded, `\(`, "(")
	encoded = strings.ReplaceAll(encoded, `\)`, ")")
	encoded = strings.ReplaceAll(encoded, `\\`, "\\")
	encoded = strings.ReplaceAll(encoded, `\n`, "\n")
	encoded = strings.ReplaceAll(encoded, `\r`, "\r")
	encoded = strings.ReplaceAll(encoded, `\t`, "\t")
	
	// Handle octal sequences (\nnn)
	octalRegex := regexp.MustCompile(`\\([0-7]{1,3})`)
	encoded = octalRegex.ReplaceAllStringFunc(encoded, func(match string) string {
		// Simple octal to character conversion
		// In practice, you'd need proper octal parsing
		return " " // Placeholder
	})
	
	return encoded
}

// extractStructure extracts structural elements from PDF text
func (pp *PDFParser) extractStructure(textContent string) *StructuredContent {
	// Since this is a simplified implementation, we'll do basic structure extraction
	// In practice, PDF structure extraction would involve:
	// 1. Analyzing PDF bookmarks/outline
	// 2. Detecting heading fonts and styles
	// 3. Identifying columns and reading order
	// 4. Extracting tables and forms
	// 5. Processing annotations and links
	
	structure := &StructuredContent{}
	
	// Basic paragraph splitting
	if textContent != "" {
		paragraphs := strings.Split(textContent, "\n\n")
		for _, para := range paragraphs {
			para = strings.TrimSpace(para)
			if para != "" {
				structure.Paragraphs = append(structure.Paragraphs, para)
			}
		}
	}
	
	return structure
}

// PDF feature detection methods (simplified)

func (pp *PDFParser) hasImages(content []byte) bool {
	contentStr := string(content)
	return strings.Contains(contentStr, "/Image") || strings.Contains(contentStr, "/XObject")
}

func (pp *PDFParser) hasForms(content []byte) bool {
	contentStr := string(content)
	return strings.Contains(contentStr, "/AcroForm") || strings.Contains(contentStr, "/Field")
}

func (pp *PDFParser) hasAnnotations(content []byte) bool {
	contentStr := string(content)
	return strings.Contains(contentStr, "/Annot")
}

func (pp *PDFParser) hasEmbeddedFonts(content []byte) bool {
	contentStr := string(content)
	return strings.Contains(contentStr, "/FontDescriptor")
}

func (pp *PDFParser) isModernPDF(content []byte) bool {
	contentStr := string(content)
	// Check for PDF version 1.4 and above (which support more complex features)
	return strings.Contains(contentStr, "%PDF-1.4") ||
		strings.Contains(contentStr, "%PDF-1.5") ||
		strings.Contains(contentStr, "%PDF-1.6") ||
		strings.Contains(contentStr, "%PDF-1.7") ||
		strings.Contains(contentStr, "%PDF-2.0")
}

func (pp *PDFParser) countWords(text string) int {
	if text == "" {
		return 0
	}
	return len(strings.Fields(text))
}

// Note: For production use, you should integrate with a proper PDF parsing library
// such as:
// - github.com/ledongthuc/pdf (pure Go)
// - github.com/unidoc/unipdf (commercial)
// - Calling external tools like pdftotext
// - Using cloud services like Google Document AI or AWS Textract
//
// This implementation provides the interface structure but requires
// proper PDF parsing implementation for full functionality.