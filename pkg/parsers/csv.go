package parsers

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CSVParser implements parsing for CSV documents
type CSVParser struct{}

// NewCSVParser creates a new CSV parser
func NewCSVParser() *CSVParser {
	return &CSVParser{}
}

// Parse parses a CSV document from a reader
func (cp *CSVParser) Parse(ctx context.Context, reader io.Reader, config *ParserConfig) (*ParsedDocument, error) {
	if config == nil {
		config = DefaultParserConfig()
	}
	
	startTime := time.Now()
	
	// Read all content first to check size
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV content: %w", err)
	}
	
	// Check file size limit
	if config.MaxFileSize > 0 && int64(len(content)) > config.MaxFileSize {
		return nil, fmt.Errorf("file size %d bytes exceeds limit %d bytes", len(content), config.MaxFileSize)
	}
	
	// Parse CSV
	csvReader := csv.NewReader(strings.NewReader(string(content)))
	csvReader.FieldsPerRecord = -1 // Allow variable number of fields
	
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}
	
	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}
	
	// Create document metadata
	metadata := &DocumentMetadata{
		MimeType:       "text/csv",
		FileSize:       int64(len(content)),
		CharacterCount: len(content),
		Language:       "csv",
	}
	
	// Extract structured content
	var structuredContent *StructuredContent
	if config.ExtractStructure {
		structuredContent = cp.extractStructure(records)
	}
	
	// Convert CSV to readable text
	textContent := cp.csvToText(records, config)
	metadata.WordCount = cp.countWords(textContent)
	
	// Add CSV-specific metadata
	cp.extractCSVMetadata(records, metadata)
	
	// Create parsed document
	doc := &ParsedDocument{
		Content:           textContent,
		Metadata:          metadata,
		StructuredContent: structuredContent,
		ParsedAt:          time.Now(),
		ParserType:        string(ParserTypeCSV),
		ParsingDuration:   time.Since(startTime),
	}
	
	return doc, nil
}

// ParseFile parses a CSV document from a file path
func (cp *CSVParser) ParseFile(ctx context.Context, filePath string, config *ParserConfig) (*ParsedDocument, error) {
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
	doc, err := cp.Parse(ctx, file, config)
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

// ParseBytes parses a CSV document from byte data
func (cp *CSVParser) ParseBytes(ctx context.Context, data []byte, filename string, config *ParserConfig) (*ParsedDocument, error) {
	if config == nil {
		config = DefaultParserConfig()
	}
	
	reader := strings.NewReader(string(data))
	doc, err := cp.Parse(ctx, reader, config)
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
func (cp *CSVParser) SupportedTypes() []string {
	return []string{
		"text/csv",
		"application/csv",
		"text/comma-separated-values",
	}
}

// SupportedExtensions returns the file extensions supported by this parser
func (cp *CSVParser) SupportedExtensions() []string {
	return []string{
		".csv", ".tsv", ".tab",
	}
}

// GetParserType returns the type identifier for this parser
func (cp *CSVParser) GetParserType() string {
	return string(ParserTypeCSV)
}

// ValidateInput validates if the input can be parsed by this parser
func (cp *CSVParser) ValidateInput(ctx context.Context, reader io.Reader) error {
	// Read a sample to validate
	buffer := make([]byte, 1024)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read input: %w", err)
	}
	
	if n == 0 {
		return fmt.Errorf("empty file")
	}
	
	// Try to parse as CSV
	csvReader := csv.NewReader(strings.NewReader(string(buffer[:n])))
	csvReader.FieldsPerRecord = -1
	
	_, err = csvReader.ReadAll()
	if err != nil {
		return fmt.Errorf("content does not appear to be valid CSV: %w", err)
	}
	
	return nil
}

// EstimateComplexity estimates parsing complexity (0-100 scale)
func (cp *CSVParser) EstimateComplexity(ctx context.Context, reader io.Reader) (int, error) {
	// Read a sample to estimate complexity
	buffer := make([]byte, 4096)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return 0, fmt.Errorf("failed to read input: %w", err)
	}
	
	if n == 0 {
		return 0, nil // Empty file has zero complexity
	}
	
	// Try to parse CSV to understand structure
	csvReader := csv.NewReader(strings.NewReader(string(buffer[:n])))
	csvReader.FieldsPerRecord = -1
	
	records, err := csvReader.ReadAll()
	if err != nil {
		return 60, nil // Invalid CSV is moderately complex
	}
	
	// Base complexity for CSV
	complexity := 10
	
	// Analyze CSV structure complexity
	if len(records) > 0 {
		// Number of columns affects complexity
		cols := len(records[0])
		if cols > 10 {
			complexity += 15
		} else if cols > 5 {
			complexity += 10
		} else if cols > 2 {
			complexity += 5
		}
		
		// Number of rows affects complexity
		rows := len(records)
		if rows > 1000 {
			complexity += 20
		} else if rows > 100 {
			complexity += 15
		} else if rows > 10 {
			complexity += 10
		}
		
		// Check for embedded commas, quotes, or newlines
		hasComplexFields := cp.hasComplexFields(records)
		if hasComplexFields {
			complexity += 15
		}
	}
	
	// Cap at reasonable maximum
	if complexity > 70 {
		complexity = 70
	}
	
	return complexity, nil
}

// hasComplexFields checks if CSV has fields with embedded commas, quotes, or newlines
func (cp *CSVParser) hasComplexFields(records [][]string) bool {
	for _, record := range records {
		for _, field := range record {
			if strings.Contains(field, ",") || strings.Contains(field, "\"") || strings.Contains(field, "\n") {
				return true
			}
		}
	}
	return false
}

// extractCSVMetadata extracts CSV-specific metadata
func (cp *CSVParser) extractCSVMetadata(records [][]string, metadata *DocumentMetadata) {
	if len(records) == 0 {
		return
	}
	
	// Store CSV info in custom metadata
	if metadata.Custom == nil {
		metadata.Custom = make(map[string]interface{})
	}
	
	metadata.Custom["csv_rows"] = len(records)
	metadata.Custom["csv_columns"] = len(records[0])
	
	// Check if first row looks like headers
	if len(records) > 1 {
		firstRow := records[0]
		hasHeaders := cp.detectHeaders(firstRow, records[1])
		metadata.Custom["csv_has_headers"] = hasHeaders
		
		if hasHeaders {
			metadata.Custom["csv_headers"] = firstRow
		}
	}
	
	// Analyze data types in columns
	if len(records) > 1 {
		columnTypes := cp.analyzeColumnTypes(records)
		metadata.Custom["csv_column_types"] = columnTypes
	}
}

// detectHeaders attempts to detect if the first row contains headers
func (cp *CSVParser) detectHeaders(firstRow, secondRow []string) bool {
	if len(firstRow) != len(secondRow) {
		return false
	}
	
	headerScore := 0
	for i, header := range firstRow {
		if i >= len(secondRow) {
			break
		}
		
		// Headers are typically text, data might be numbers
		headerIsText := cp.isLikelyText(header)
		dataIsNumber := cp.isLikelyNumber(secondRow[i])
		
		if headerIsText && dataIsNumber {
			headerScore++
		}
		
		// Headers often don't have special characters that data might have
		if !strings.Contains(header, "@") && !strings.Contains(header, "/") && !strings.Contains(header, ":") {
			headerScore++
		}
	}
	
	// If more than half the columns suggest headers, assume headers exist
	return headerScore > len(firstRow)/2
}

// isLikelyText checks if a string is likely descriptive text
func (cp *CSVParser) isLikelyText(s string) bool {
	if len(s) == 0 {
		return false
	}
	
	// Text headers often contain spaces or underscores
	if strings.Contains(s, " ") || strings.Contains(s, "_") {
		return true
	}
	
	// Check if it's not a pure number
	return !cp.isLikelyNumber(s)
}

// isLikelyNumber checks if a string represents a number
func (cp *CSVParser) isLikelyNumber(s string) bool {
	if len(s) == 0 {
		return false
	}
	
	// Simple check for numeric data
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	
	// Check if it contains only digits, decimal points, and signs
	for _, r := range s {
		if !((r >= '0' && r <= '9') || r == '.' || r == '-' || r == '+' || r == ',' || r == '$' || r == '%') {
			return false
		}
	}
	
	return true
}

// analyzeColumnTypes analyzes the data types in each column
func (cp *CSVParser) analyzeColumnTypes(records [][]string) []string {
	if len(records) == 0 {
		return nil
	}
	
	numCols := len(records[0])
	columnTypes := make([]string, numCols)
	
	for col := 0; col < numCols; col++ {
		columnTypes[col] = cp.detectColumnType(records, col)
	}
	
	return columnTypes
}

// detectColumnType detects the data type of a specific column
func (cp *CSVParser) detectColumnType(records [][]string, colIndex int) string {
	if len(records) <= 1 {
		return "unknown"
	}
	
	// Skip header row if present
	startRow := 1
	
	numberCount := 0
	textCount := 0
	emailCount := 0
	urlCount := 0
	dateCount := 0
	totalRows := 0
	
	for i := startRow; i < len(records) && i < startRow+20; i++ { // Sample up to 20 rows
		if colIndex >= len(records[i]) {
			continue
		}
		
		value := strings.TrimSpace(records[i][colIndex])
		if value == "" {
			continue
		}
		
		totalRows++
		
		if cp.isLikelyNumber(value) {
			numberCount++
		} else if strings.Contains(value, "@") && strings.Contains(value, ".") {
			emailCount++
		} else if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
			urlCount++
		} else if cp.isLikelyDate(value) {
			dateCount++
		} else {
			textCount++
		}
	}
	
	if totalRows == 0 {
		return "empty"
	}
	
	// Determine the most likely type
	threshold := totalRows / 2
	
	if numberCount > threshold {
		return "number"
	} else if emailCount > threshold {
		return "email"
	} else if urlCount > threshold {
		return "url"
	} else if dateCount > threshold {
		return "date"
	} else {
		return "text"
	}
}

// isLikelyDate checks if a string looks like a date
func (cp *CSVParser) isLikelyDate(s string) bool {
	if len(s) < 8 {
		return false
	}
	
	// Simple date pattern detection
	datePatterns := []string{
		"/", "-", ".", " ",
	}
	
	for _, pattern := range datePatterns {
		if strings.Contains(s, pattern) {
			parts := strings.Split(s, pattern)
			if len(parts) >= 3 {
				// Check if parts could be month/day/year
				for _, part := range parts {
					if cp.isLikelyNumber(part) {
						return true
					}
				}
			}
		}
	}
	
	return false
}

// extractStructure extracts structural elements from CSV data
func (cp *CSVParser) extractStructure(records [][]string) *StructuredContent {
	structure := &StructuredContent{}
	
	if len(records) == 0 {
		return structure
	}
	
	// Convert CSV to table structure
	table := Table{}
	
	// Check if first row is headers
	if len(records) > 1 && cp.detectHeaders(records[0], records[1]) {
		table.Headers = records[0]
		table.Rows = records[1:]
	} else {
		// Generate generic headers
		if len(records) > 0 {
			for i := 0; i < len(records[0]); i++ {
				table.Headers = append(table.Headers, fmt.Sprintf("Column %d", i+1))
			}
			table.Rows = records
		}
	}
	
	structure.Tables = []Table{table}
	
	// Create summary paragraphs
	if len(records) > 0 {
		summary := fmt.Sprintf("CSV data with %d rows and %d columns.", len(records), len(records[0]))
		structure.Paragraphs = append(structure.Paragraphs, summary)
		
		// Add column summary if we have headers
		if len(table.Headers) > 0 {
			columnSummary := fmt.Sprintf("Columns: %s", strings.Join(table.Headers, ", "))
			structure.Paragraphs = append(structure.Paragraphs, columnSummary)
		}
	}
	
	return structure
}

// csvToText converts CSV data to readable text
func (cp *CSVParser) csvToText(records [][]string, config *ParserConfig) string {
	if len(records) == 0 {
		return ""
	}
	
	if config.PreserveFormatting {
		// Format as table-like text
		return cp.formatAsTable(records)
	}
	
	// Convert to paragraph format
	return cp.formatAsParagraphs(records)
}

// formatAsTable formats CSV data as a table-like text
func (cp *CSVParser) formatAsTable(records [][]string) string {
	if len(records) == 0 {
		return ""
	}
	
	var result strings.Builder
	
	// Calculate column widths
	colWidths := make([]int, len(records[0]))
	for _, record := range records {
		for i, field := range record {
			if i < len(colWidths) && len(field) > colWidths[i] {
				colWidths[i] = len(field)
			}
		}
	}
	
	// Format each row
	for i, record := range records {
		var row strings.Builder
		for j, field := range record {
			if j > 0 {
				row.WriteString(" | ")
			}
			
			if j < len(colWidths) {
				row.WriteString(fmt.Sprintf("%-*s", colWidths[j], field))
			} else {
				row.WriteString(field)
			}
		}
		
		result.WriteString(row.String())
		result.WriteString("\n")
		
		// Add separator after header row
		if i == 0 && len(records) > 1 {
			var separator strings.Builder
			for j := 0; j < len(record); j++ {
				if j > 0 {
					separator.WriteString("-+-")
				}
				if j < len(colWidths) {
					separator.WriteString(strings.Repeat("-", colWidths[j]))
				}
			}
			result.WriteString(separator.String())
			result.WriteString("\n")
		}
	}
	
	return result.String()
}

// formatAsParagraphs formats CSV data as readable paragraphs
func (cp *CSVParser) formatAsParagraphs(records [][]string) string {
	if len(records) == 0 {
		return ""
	}
	
	var result strings.Builder
	
	// Check if first row is headers
	hasHeaders := len(records) > 1 && cp.detectHeaders(records[0], records[1])
	
	if hasHeaders {
		headers := records[0]
		result.WriteString(fmt.Sprintf("Data with columns: %s\n\n", strings.Join(headers, ", ")))
		
		// Format each data row
		for i := 1; i < len(records) && i <= 10; i++ { // Limit to first 10 rows
			record := records[i]
			var rowParts []string
			
			for j, field := range record {
				if j < len(headers) && field != "" {
					rowParts = append(rowParts, fmt.Sprintf("%s: %s", headers[j], field))
				}
			}
			
			if len(rowParts) > 0 {
				result.WriteString(strings.Join(rowParts, ", "))
				result.WriteString(".\n")
			}
		}
		
		if len(records) > 11 {
			result.WriteString(fmt.Sprintf("\n... and %d more records.\n", len(records)-11))
		}
	} else {
		// No clear headers, format as simple rows
		for i, record := range records {
			if i >= 10 { // Limit output
				result.WriteString(fmt.Sprintf("\n... and %d more rows.\n", len(records)-i))
				break
			}
			
			result.WriteString(fmt.Sprintf("Row %d: %s\n", i+1, strings.Join(record, ", ")))
		}
	}
	
	return result.String()
}

// countWords counts meaningful words in CSV text
func (cp *CSVParser) countWords(text string) int {
	words := strings.Fields(text)
	meaningfulWords := 0
	
	for _, word := range words {
		// Skip CSV structure tokens and separators
		if word == "|" || word == ":" || word == "," || word == "..." {
			continue
		}
		
		// Skip row indicators
		if strings.HasPrefix(word, "Row") || strings.HasPrefix(word, "Column") {
			continue
		}
		
		meaningfulWords++
	}
	
	return meaningfulWords
}