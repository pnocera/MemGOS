package chunkers

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// CodeBlockProcessor handles processing of code blocks
type CodeBlockProcessor struct{}

// NewCodeBlockProcessor creates a new code block processor
func NewCodeBlockProcessor() *CodeBlockProcessor {
	return &CodeBlockProcessor{}
}

// ProcessCodeBlock processes a code block, optionally splitting it
func (cbp *CodeBlockProcessor) ProcessCodeBlock(block *ContentBlock, chunkSize int, tokenEstimator func(string) int) ([]*Chunk, error) {
	totalTokens := tokenEstimator(block.Content)
	
	// If code block fits in one chunk, return as-is
	if totalTokens <= chunkSize {
		chunk := &Chunk{
			Text:       block.Content,
			TokenCount: totalTokens,
			Sentences:  []string{block.Content},
			StartIndex: block.StartIndex,
			EndIndex:   block.EndIndex,
			Metadata: map[string]interface{}{
				"content_type":    string(block.Type),
				"language":        block.Language,
				"block_metadata":  block.Metadata,
				"code_processed":  true,
			},
			CreatedAt: time.Now(),
		}
		return []*Chunk{chunk}, nil
	}
	
	// Split large code blocks by logical boundaries
	return cbp.splitCodeBlock(block, chunkSize, tokenEstimator)
}

// splitCodeBlock splits a large code block into smaller chunks
func (cbp *CodeBlockProcessor) splitCodeBlock(block *ContentBlock, chunkSize int, tokenEstimator func(string) int) ([]*Chunk, error) {
	var chunks []*Chunk
	
	// Get raw code content
	rawCode, ok := block.Metadata["raw_code"].(string)
	if !ok {
		rawCode = block.Content
	}
	
	lines := strings.Split(rawCode, "\n")
	if len(lines) == 0 {
		return chunks, nil
	}
	
	// Try to split by functions/classes first
	functionBoundaries := cbp.detectLogicalBoundaries(lines, block.Language)
	
	if len(functionBoundaries) > 1 {
		chunks = cbp.splitByBoundaries(lines, functionBoundaries, block, chunkSize, tokenEstimator)
	} else {
		// Fallback to simple line-based splitting
		chunks = cbp.splitByLines(lines, block, chunkSize, tokenEstimator)
	}
	
	return chunks, nil
}

// detectLogicalBoundaries detects function/class boundaries in code
func (cbp *CodeBlockProcessor) detectLogicalBoundaries(lines []string, language string) []int {
	var boundaries []int
	boundaries = append(boundaries, 0) // Start
	
	switch language {
	case "python":
		boundaries = append(boundaries, cbp.detectPythonBoundaries(lines)...)
	case "javascript", "js":
		boundaries = append(boundaries, cbp.detectJavaScriptBoundaries(lines)...)
	case "go":
		boundaries = append(boundaries, cbp.detectGoBoundaries(lines)...)
	case "java":
		boundaries = append(boundaries, cbp.detectJavaBoundaries(lines)...)
	default:
		// Generic boundary detection
		boundaries = append(boundaries, cbp.detectGenericBoundaries(lines)...)
	}
	
	boundaries = append(boundaries, len(lines)) // End
	return cbp.removeDuplicateBoundaries(boundaries)
}

// detectPythonBoundaries detects Python function/class boundaries
func (cbp *CodeBlockProcessor) detectPythonBoundaries(lines []string) []int {
	var boundaries []int
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "class ") {
			boundaries = append(boundaries, i)
		}
	}
	
	return boundaries
}

// detectJavaScriptBoundaries detects JavaScript function boundaries
func (cbp *CodeBlockProcessor) detectJavaScriptBoundaries(lines []string) []int {
	var boundaries []int
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "function ") || 
		   strings.Contains(trimmed, "const ") && strings.Contains(trimmed, "=>") ||
		   strings.Contains(trimmed, "let ") && strings.Contains(trimmed, "=>") ||
		   strings.Contains(trimmed, "var ") && strings.Contains(trimmed, "=>") {
			boundaries = append(boundaries, i)
		}
	}
	
	return boundaries
}

// detectGoBoundaries detects Go function boundaries
func (cbp *CodeBlockProcessor) detectGoBoundaries(lines []string) []int {
	var boundaries []int
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "func ") || strings.HasPrefix(trimmed, "type ") {
			boundaries = append(boundaries, i)
		}
	}
	
	return boundaries
}

// detectJavaBoundaries detects Java method/class boundaries
func (cbp *CodeBlockProcessor) detectJavaBoundaries(lines []string) []int {
	var boundaries []int
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if (strings.Contains(trimmed, "public ") || strings.Contains(trimmed, "private ") || 
		    strings.Contains(trimmed, "protected ")) && 
		   (strings.Contains(trimmed, "class ") || strings.Contains(trimmed, "interface ") ||
		    strings.Contains(trimmed, "void ") || strings.Contains(trimmed, "int ") ||
		    strings.Contains(trimmed, "String ")) {
			boundaries = append(boundaries, i)
		}
	}
	
	return boundaries
}

// detectGenericBoundaries detects generic logical boundaries
func (cbp *CodeBlockProcessor) detectGenericBoundaries(lines []string) []int {
	var boundaries []int
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Look for lines that might start new logical blocks
		if strings.HasSuffix(trimmed, "{") || 
		   strings.Contains(trimmed, "//") && (strings.Contains(trimmed, "Function") || strings.Contains(trimmed, "Method")) {
			boundaries = append(boundaries, i)
		}
	}
	
	return boundaries
}

// removeDuplicateBoundaries removes duplicate boundary positions
func (cbp *CodeBlockProcessor) removeDuplicateBoundaries(boundaries []int) []int {
	if len(boundaries) <= 1 {
		return boundaries
	}
	
	// Sort boundaries
	for i := 0; i < len(boundaries)-1; i++ {
		for j := i + 1; j < len(boundaries); j++ {
			if boundaries[i] > boundaries[j] {
				boundaries[i], boundaries[j] = boundaries[j], boundaries[i]
			}
		}
	}
	
	// Remove duplicates
	var unique []int
	for i, boundary := range boundaries {
		if i == 0 || boundary != boundaries[i-1] {
			unique = append(unique, boundary)
		}
	}
	
	return unique
}

// splitByBoundaries splits code by detected logical boundaries
func (cbp *CodeBlockProcessor) splitByBoundaries(lines []string, boundaries []int, block *ContentBlock, chunkSize int, tokenEstimator func(string) int) []*Chunk {
	var chunks []*Chunk
	
	for i := 0; i < len(boundaries)-1; i++ {
		start := boundaries[i]
		end := boundaries[i+1]
		
		if start >= end || start >= len(lines) {
			continue
		}
		
		sectionLines := lines[start:end]
		sectionContent := strings.Join(sectionLines, "\n")
		
		// If section is too large, split further
		if tokenEstimator(sectionContent) > chunkSize {
			subChunks := cbp.splitByLines(sectionLines, block, chunkSize, tokenEstimator)
			chunks = append(chunks, subChunks...)
		} else {
			chunk := &Chunk{
				Text:       sectionContent,
				TokenCount: tokenEstimator(sectionContent),
				Sentences:  []string{sectionContent},
				StartIndex: block.StartIndex,
				EndIndex:   block.StartIndex + len(sectionContent),
				Metadata: map[string]interface{}{
					"content_type":     string(block.Type),
					"language":         block.Language,
					"block_metadata":   block.Metadata,
					"code_processed":   true,
					"split_method":     "logical",
					"section_start":    start,
					"section_end":      end,
				},
				CreatedAt: time.Now(),
			}
			chunks = append(chunks, chunk)
		}
	}
	
	return chunks
}

// splitByLines splits code by lines when logical splitting isn't possible
func (cbp *CodeBlockProcessor) splitByLines(lines []string, block *ContentBlock, chunkSize int, tokenEstimator func(string) int) []*Chunk {
	var chunks []*Chunk
	var currentLines []string
	currentTokens := 0
	
	for i, line := range lines {
		lineTokens := tokenEstimator(line)
		
		if currentTokens+lineTokens > chunkSize && len(currentLines) > 0 {
			// Create chunk from current lines
			chunkContent := strings.Join(currentLines, "\n")
			chunk := &Chunk{
				Text:       chunkContent,
				TokenCount: currentTokens,
				Sentences:  []string{chunkContent},
				StartIndex: block.StartIndex,
				EndIndex:   block.StartIndex + len(chunkContent),
				Metadata: map[string]interface{}{
					"content_type":   string(block.Type),
					"language":       block.Language,
					"block_metadata": block.Metadata,
					"code_processed": true,
					"split_method":   "lines",
					"line_range":     fmt.Sprintf("%d-%d", i-len(currentLines), i-1),
				},
				CreatedAt: time.Now(),
			}
			chunks = append(chunks, chunk)
			
			// Start new chunk
			currentLines = []string{line}
			currentTokens = lineTokens
		} else {
			currentLines = append(currentLines, line)
			currentTokens += lineTokens
		}
	}
	
	// Add final chunk
	if len(currentLines) > 0 {
		chunkContent := strings.Join(currentLines, "\n")
		chunk := &Chunk{
			Text:       chunkContent,
			TokenCount: currentTokens,
			Sentences:  []string{chunkContent},
			StartIndex: block.StartIndex,
			EndIndex:   block.StartIndex + len(chunkContent),
			Metadata: map[string]interface{}{
				"content_type":   string(block.Type),
				"language":       block.Language,
				"block_metadata": block.Metadata,
				"code_processed": true,
				"split_method":   "lines",
			},
			CreatedAt: time.Now(),
		}
		chunks = append(chunks, chunk)
	}
	
	return chunks
}

// TableProcessor handles processing of table structures
type TableProcessor struct{}

// NewTableProcessor creates a new table processor
func NewTableProcessor() *TableProcessor {
	return &TableProcessor{}
}

// ProcessTable processes a table, optionally splitting it
func (tp *TableProcessor) ProcessTable(block *ContentBlock, chunkSize int, tokenEstimator func(string) int) ([]*Chunk, error) {
	format, _ := block.Metadata["format"].(string)
	
	switch format {
	case "markdown":
		return tp.processMarkdownTable(block, chunkSize, tokenEstimator)
	case "html":
		return tp.processHTMLTable(block, chunkSize, tokenEstimator)
	default:
		// Treat as generic table
		return tp.processGenericTable(block, chunkSize, tokenEstimator)
	}
}

// processMarkdownTable processes a Markdown table
func (tp *TableProcessor) processMarkdownTable(block *ContentBlock, chunkSize int, tokenEstimator func(string) int) ([]*Chunk, error) {
	lines := strings.Split(block.Content, "\n")
	if len(lines) < 3 { // Need at least header, separator, and one data row
		return tp.createSingleTableChunk(block, tokenEstimator), nil
	}
	
	// Identify header and separator
	headerLine := lines[0]
	separatorLine := lines[1]
	dataLines := lines[2:]
	
	// Calculate header size
	headerContent := headerLine + "\n" + separatorLine
	headerTokens := tokenEstimator(headerContent)
	
	var chunks []*Chunk
	var currentRows []string
	currentTokens := headerTokens
	
	for i, row := range dataLines {
		rowTokens := tokenEstimator(row)
		
		if currentTokens+rowTokens > chunkSize && len(currentRows) > 0 {
			// Create chunk with current rows
			chunkContent := headerContent + "\n" + strings.Join(currentRows, "\n")
			chunk := tp.createTableChunk(block, chunkContent, tokenEstimator, i-len(currentRows), i-1)
			chunks = append(chunks, chunk)
			
			// Start new chunk
			currentRows = []string{row}
			currentTokens = headerTokens + rowTokens
		} else {
			currentRows = append(currentRows, row)
			currentTokens += rowTokens
		}
	}
	
	// Add final chunk
	if len(currentRows) > 0 {
		chunkContent := headerContent + "\n" + strings.Join(currentRows, "\n")
		chunk := tp.createTableChunk(block, chunkContent, tokenEstimator, len(dataLines)-len(currentRows), len(dataLines)-1)
		chunks = append(chunks, chunk)
	}
	
	return chunks, nil
}

// processHTMLTable processes an HTML table
func (tp *TableProcessor) processHTMLTable(block *ContentBlock, chunkSize int, tokenEstimator func(string) int) ([]*Chunk, error) {
	// For HTML tables, we'll use a simpler approach and split by <tr> tags
	content := block.Content
	
	// Extract table header if present
	var header string
	if strings.Contains(content, "<thead>") {
		headerPattern := `(?si)<thead>.*?</thead>`
		if match := regexp.MustCompile(headerPattern).FindString(content); match != "" {
			header = match
		}
	}
	
	// Split by table rows
	rowPattern := `(?si)<tr[^>]*>.*?</tr>`
	rows := regexp.MustCompile(rowPattern).FindAllString(content, -1)
	
	if len(rows) == 0 {
		return tp.createSingleTableChunk(block, tokenEstimator), nil
	}
	
	// Process rows similar to Markdown
	headerTokens := tokenEstimator(header)
	var chunks []*Chunk
	var currentRows []string
	currentTokens := headerTokens
	
	for i, row := range rows {
		rowTokens := tokenEstimator(row)
		
		if currentTokens+rowTokens > chunkSize && len(currentRows) > 0 {
			// Create chunk
			chunkContent := tp.buildHTMLTableChunk(header, currentRows)
			chunk := tp.createTableChunk(block, chunkContent, tokenEstimator, i-len(currentRows), i-1)
			chunks = append(chunks, chunk)
			
			// Start new chunk
			currentRows = []string{row}
			currentTokens = headerTokens + rowTokens
		} else {
			currentRows = append(currentRows, row)
			currentTokens += rowTokens
		}
	}
	
	// Add final chunk
	if len(currentRows) > 0 {
		chunkContent := tp.buildHTMLTableChunk(header, currentRows)
		chunk := tp.createTableChunk(block, chunkContent, tokenEstimator, len(rows)-len(currentRows), len(rows)-1)
		chunks = append(chunks, chunk)
	}
	
	return chunks, nil
}

// processGenericTable processes a generic table format
func (tp *TableProcessor) processGenericTable(block *ContentBlock, chunkSize int, tokenEstimator func(string) int) ([]*Chunk, error) {
	// For generic tables, just return as single chunk
	return tp.createSingleTableChunk(block, tokenEstimator), nil
}

// buildHTMLTableChunk builds an HTML table chunk from header and rows
func (tp *TableProcessor) buildHTMLTableChunk(header string, rows []string) string {
	var content strings.Builder
	content.WriteString("<table>")
	
	if header != "" {
		content.WriteString(header)
	}
	
	content.WriteString("<tbody>")
	for _, row := range rows {
		content.WriteString(row)
	}
	content.WriteString("</tbody>")
	content.WriteString("</table>")
	
	return content.String()
}

// createTableChunk creates a table chunk with metadata
func (tp *TableProcessor) createTableChunk(block *ContentBlock, content string, tokenEstimator func(string) int, startRow, endRow int) *Chunk {
	return &Chunk{
		Text:       content,
		TokenCount: tokenEstimator(content),
		Sentences:  []string{content},
		StartIndex: block.StartIndex,
		EndIndex:   block.StartIndex + len(content),
		Metadata: map[string]interface{}{
			"content_type":    string(block.Type),
			"block_metadata":  block.Metadata,
			"table_processed": true,
			"split_method":    "rows",
			"row_range":       fmt.Sprintf("%d-%d", startRow, endRow),
		},
		CreatedAt: time.Now(),
	}
}

// createSingleTableChunk creates a single chunk for the entire table
func (tp *TableProcessor) createSingleTableChunk(block *ContentBlock, tokenEstimator func(string) int) []*Chunk {
	chunk := &Chunk{
		Text:       block.Content,
		TokenCount: tokenEstimator(block.Content),
		Sentences:  []string{block.Content},
		StartIndex: block.StartIndex,
		EndIndex:   block.EndIndex,
		Metadata: map[string]interface{}{
			"content_type":    string(block.Type),
			"block_metadata":  block.Metadata,
			"table_processed": true,
			"preserved":       true,
		},
		CreatedAt: time.Now(),
	}
	return []*Chunk{chunk}
}

// ImageProcessor handles processing of image references
type ImageProcessor struct{}

// NewImageProcessor creates a new image processor
func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{}
}

// ProcessImage processes an image reference (always returns single chunk)
func (ip *ImageProcessor) ProcessImage(block *ContentBlock, tokenEstimator func(string) int) []*Chunk {
	chunk := &Chunk{
		Text:       block.Content,
		TokenCount: tokenEstimator(block.Content),
		Sentences:  []string{block.Content},
		StartIndex: block.StartIndex,
		EndIndex:   block.EndIndex,
		Metadata: map[string]interface{}{
			"content_type":     string(block.Type),
			"block_metadata":   block.Metadata,
			"image_processed":  true,
			"is_media":         true,
		},
		CreatedAt: time.Now(),
	}
	
	return []*Chunk{chunk}
}