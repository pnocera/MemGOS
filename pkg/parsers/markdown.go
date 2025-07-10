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

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// MarkdownParser implements parsing for Markdown files
type MarkdownParser struct{}

// NewMarkdownParser creates a new markdown parser
func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{}
}

// Parse parses a Markdown document from a reader
func (mp *MarkdownParser) Parse(ctx context.Context, reader io.Reader, config *ParserConfig) (*ParsedDocument, error) {
	if config == nil {
		config = DefaultParserConfig()
	}
	
	startTime := time.Now()
	
	// Read all content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read markdown content: %w", err)
	}
	
	// Check file size limit
	if config.MaxFileSize > 0 && int64(len(content)) > config.MaxFileSize {
		return nil, fmt.Errorf("file size %d bytes exceeds limit %d bytes", len(content), config.MaxFileSize)
	}
	
	markdownContent := string(content)
	
	// Create document metadata
	metadata := &DocumentMetadata{
		MimeType:       "text/markdown",
		FileSize:       int64(len(content)),
		WordCount:      mp.countWords(markdownContent),
		CharacterCount: len(markdownContent),
		Language:       "en", // Default, could be improved with detection
	}
	
	// Extract front matter metadata if present
	if config.ExtractMetadata {
		mp.extractFrontMatter(markdownContent, metadata)
	}
	
	// Extract structured content
	var structuredContent *StructuredContent
	if config.ExtractStructure {
		structuredContent = mp.extractStructure(markdownContent)
	}
	
	// Process content based on configuration
	processedContent := mp.processContent(markdownContent, config)
	
	// Create parsed document
	doc := &ParsedDocument{
		Content:           processedContent,
		Metadata:          metadata,
		StructuredContent: structuredContent,
		ParsedAt:          time.Now(),
		ParserType:        string(ParserTypeMarkdown),
		ParsingDuration:   time.Since(startTime),
	}
	
	return doc, nil
}

// ParseFile parses a Markdown document from a file path
func (mp *MarkdownParser) ParseFile(ctx context.Context, filePath string, config *ParserConfig) (*ParsedDocument, error) {
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
	doc, err := mp.Parse(ctx, file, config)
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

// ParseBytes parses a Markdown document from byte data
func (mp *MarkdownParser) ParseBytes(ctx context.Context, data []byte, filename string, config *ParserConfig) (*ParsedDocument, error) {
	if config == nil {
		config = DefaultParserConfig()
	}
	
	reader := strings.NewReader(string(data))
	doc, err := mp.Parse(ctx, reader, config)
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
func (mp *MarkdownParser) SupportedTypes() []string {
	return []string{
		"text/markdown",
		"text/x-markdown",
		"application/markdown",
	}
}

// SupportedExtensions returns the file extensions supported by this parser
func (mp *MarkdownParser) SupportedExtensions() []string {
	return []string{
		".md", ".markdown", ".mdown", ".mkdn", ".mkd", ".mdwn", ".mdtxt", ".mdtext",
	}
}

// GetParserType returns the type identifier for this parser
func (mp *MarkdownParser) GetParserType() string {
	return string(ParserTypeMarkdown)
}

// ValidateInput validates if the input can be parsed by this parser
func (mp *MarkdownParser) ValidateInput(ctx context.Context, reader io.Reader) error {
	// Read a sample to validate
	buffer := make([]byte, 1024)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read input: %w", err)
	}
	
	if n == 0 {
		return nil // Empty file is valid
	}
	
	content := string(buffer[:n])
	
	// Check for common Markdown patterns
	if mp.hasMarkdownPatterns(content) {
		return nil
	}
	
	// If no clear Markdown patterns, treat as potentially valid plain text
	return nil
}

// EstimateComplexity estimates parsing complexity (0-100 scale)
func (mp *MarkdownParser) EstimateComplexity(ctx context.Context, reader io.Reader) (int, error) {
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
	
	// Base complexity for Markdown
	complexity := 15
	
	// Increase complexity based on Markdown features
	if mp.hasHeadings(content) {
		complexity += 10
	}
	
	if mp.hasLists(content) {
		complexity += 10
	}
	
	if mp.hasLinks(content) {
		complexity += 15
	}
	
	if mp.hasImages(content) {
		complexity += 15
	}
	
	if mp.hasCodeBlocks(content) {
		complexity += 10
	}
	
	if mp.hasTables(content) {
		complexity += 20
	}
	
	if mp.hasFrontMatter(content) {
		complexity += 10
	}
	
	// Cap at reasonable maximum
	if complexity > 80 {
		complexity = 80
	}
	
	return complexity, nil
}

// extractFrontMatter extracts YAML/TOML front matter from Markdown
func (mp *MarkdownParser) extractFrontMatter(content string, metadata *DocumentMetadata) {
	// Check for YAML front matter (---...---)
	yamlPattern := regexp.MustCompile(`^---\s*\n(.*?)\n---\s*\n`)
	if matches := yamlPattern.FindStringSubmatch(content); len(matches) > 1 {
		mp.parseFrontMatterFields(matches[1], metadata)
		return
	}
	
	// Check for TOML front matter (+++...+++)
	tomlPattern := regexp.MustCompile(`^\+\+\+\s*\n(.*?)\n\+\+\+\s*\n`)
	if matches := tomlPattern.FindStringSubmatch(content); len(matches) > 1 {
		mp.parseFrontMatterFields(matches[1], metadata)
		return
	}
}

// parseFrontMatterFields parses front matter fields into metadata
func (mp *MarkdownParser) parseFrontMatterFields(frontMatter string, metadata *DocumentMetadata) {
	lines := strings.Split(frontMatter, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Simple key-value parsing
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		
		switch strings.ToLower(key) {
		case "title":
			metadata.Title = value
		case "author":
			metadata.Author = value
		case "subject", "description":
			metadata.Subject = value
		case "language", "lang":
			metadata.Language = value
		case "keywords", "tags":
			// Simple comma-separated keywords
			keywords := strings.Split(value, ",")
			for i, keyword := range keywords {
				keywords[i] = strings.TrimSpace(keyword)
			}
			metadata.Keywords = keywords
		}
	}
}

// extractStructure extracts structural elements from Markdown using goldmark
func (mp *MarkdownParser) extractStructure(content string) *StructuredContent {
	structure := &StructuredContent{}
	
	// Remove front matter for processing
	cleanContent := mp.removeFrontMatter(content)
	
	// Try goldmark parsing first
	if goldmarkStructure := mp.extractStructureWithGoldmark(cleanContent); goldmarkStructure != nil {
		return goldmarkStructure
	}
	
	// Fallback to regex-based parsing
	return mp.extractStructureFallback(cleanContent)
}

// extractStructureWithGoldmark uses goldmark to parse Markdown structure
func (mp *MarkdownParser) extractStructureWithGoldmark(content string) *StructuredContent {
	md := goldmark.New()
	
	source := []byte(content)
	reader := text.NewReader(source)
	
	doc := md.Parser().Parse(reader)
	if doc == nil {
		return nil
	}
	
	structure := &StructuredContent{}
	
	// Walk the AST to extract structure
	ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		
		switch n := node.(type) {
		case *ast.Heading:
			heading := Heading{
				Level: n.Level,
				Text:  mp.extractTextFromNode(n, source),
			}
			// Extract ID if present
			if id := n.AttributeValue([]byte("id")); id != nil {
				heading.ID = string(id.([]byte))
			}
			structure.Headings = append(structure.Headings, heading)
			
		case *ast.Paragraph:
			text := mp.extractTextFromNode(n, source)
			if text != "" {
				structure.Paragraphs = append(structure.Paragraphs, text)
			}
			
		case *ast.List:
			list := List{
				Type: "unordered",
			}
			if n.IsOrdered() {
				list.Type = "ordered"
			}
			
			// Extract list items
			for child := n.FirstChild(); child != nil; child = child.NextSibling() {
				if listItem, ok := child.(*ast.ListItem); ok {
					itemText := mp.extractTextFromNode(listItem, source)
					if itemText != "" {
						list.Items = append(list.Items, itemText)
					}
				}
			}
			
			if len(list.Items) > 0 {
				structure.Lists = append(structure.Lists, list)
			}
			
		case *ast.Table:
			table := mp.extractTableFromNode(n, source)
			if table.Rows != nil || table.Headers != nil {
				structure.Tables = append(structure.Tables, table)
			}
			
		case *ast.Image:
			image := Image{
				URL:     string(n.Destination),
				AltText: mp.extractTextFromNode(n, source),
			}
			if title := n.Title; title != nil {
				image.Caption = string(title)
			}
			structure.Images = append(structure.Images, image)
			
		case *ast.Link:
			link := Link{
				URL:  string(n.Destination),
				Text: mp.extractTextFromNode(n, source),
			}
			structure.Links = append(structure.Links, link)
			
		case *ast.FencedCodeBlock:
			codeBlock := CodeBlock{
				Code: mp.extractTextFromNode(n, source),
			}
			if lang := n.Language(source); lang != nil {
				codeBlock.Language = string(lang)
			}
			structure.CodeBlocks = append(structure.CodeBlocks, codeBlock)
			
		case *ast.CodeBlock:
			codeBlock := CodeBlock{
				Code: mp.extractTextFromNode(n, source),
			}
			structure.CodeBlocks = append(structure.CodeBlocks, codeBlock)
		}
		
		return ast.WalkContinue, nil
	})
	
	return structure
}

// extractTextFromNode extracts text content from an AST node
func (mp *MarkdownParser) extractTextFromNode(node ast.Node, source []byte) string {
	var buf bytes.Buffer
	
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if textNode, ok := child.(*ast.Text); ok {
			segment := textNode.Segment
			buf.Write(segment.Value(source))
		} else if child.HasChildren() {
			buf.WriteString(mp.extractTextFromNode(child, source))
		}
	}
	
	return strings.TrimSpace(buf.String())
}

// extractTableFromNode extracts table data from an AST table node
func (mp *MarkdownParser) extractTableFromNode(tableNode *ast.Table, source []byte) Table {
	table := Table{}
	
	var currentRow []string
	var isHeader bool = true
	
	for child := tableNode.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.TableRow:
			currentRow = []string{}
			
			for cell := n.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if cellNode, ok := cell.(*ast.TableCell); ok {
					cellText := mp.extractTextFromNode(cellNode, source)
					currentRow = append(currentRow, cellText)
				}
			}
			
			if len(currentRow) > 0 {
				if isHeader && len(table.Headers) == 0 {
					table.Headers = currentRow
					isHeader = false
				} else {
					table.Rows = append(table.Rows, currentRow)
				}
			}
		}
	}
	
	return table
}

// extractStructureFallback provides fallback extraction using regex
func (mp *MarkdownParser) extractStructureFallback(content string) *StructuredContent {
	structure := &StructuredContent{}
	
	lines := strings.Split(content, "\n")
	var paragraphs []string
	var currentParagraph strings.Builder
	var currentList *List
	
	for i, line := range lines {
		originalLine := line
		line = strings.TrimSpace(line)
		
		if line == "" {
			// Empty line - end current paragraph or list
			if currentParagraph.Len() > 0 {
				paragraphs = append(paragraphs, currentParagraph.String())
				currentParagraph.Reset()
			}
			if currentList != nil {
				structure.Lists = append(structure.Lists, *currentList)
				currentList = nil
			}
			continue
		}
		
		// Check for headings
		if heading := mp.parseHeading(line); heading != nil {
			// Finish current paragraph/list
			if currentParagraph.Len() > 0 {
				paragraphs = append(paragraphs, currentParagraph.String())
				currentParagraph.Reset()
			}
			if currentList != nil {
				structure.Lists = append(structure.Lists, *currentList)
				currentList = nil
			}
			
			structure.Headings = append(structure.Headings, *heading)
			continue
		}
		
		// Check for list items
		if listItem := mp.parseListItem(line); listItem != nil {
			// Finish current paragraph
			if currentParagraph.Len() > 0 {
				paragraphs = append(paragraphs, currentParagraph.String())
				currentParagraph.Reset()
			}
			
			// Start new list or continue existing one
			if currentList == nil || currentList.Type != listItem.Type {
				if currentList != nil {
					structure.Lists = append(structure.Lists, *currentList)
				}
				currentList = &List{Type: listItem.Type, Items: []string{}}
			}
			currentList.Items = append(currentList.Items, listItem.Text)
			continue
		}
		
		// Check for code blocks
		if mp.isCodeBlockStart(line) {
			// Finish current paragraph/list
			if currentParagraph.Len() > 0 {
				paragraphs = append(paragraphs, currentParagraph.String())
				currentParagraph.Reset()
			}
			if currentList != nil {
				structure.Lists = append(structure.Lists, *currentList)
				currentList = nil
			}
			
			// Parse code block
			if codeBlock := mp.parseCodeBlock(lines, &i); codeBlock != nil {
				structure.CodeBlocks = append(structure.CodeBlocks, *codeBlock)
			}
			continue
		}
		
		// Check for tables
		if mp.isTableRow(line) {
			// Finish current paragraph/list
			if currentParagraph.Len() > 0 {
				paragraphs = append(paragraphs, currentParagraph.String())
				currentParagraph.Reset()
			}
			if currentList != nil {
				structure.Lists = append(structure.Lists, *currentList)
				currentList = nil
			}
			
			// Parse table
			if table := mp.parseTable(lines, &i); table != nil {
				structure.Tables = append(structure.Tables, *table)
			}
			continue
		}
		
		// Extract links and images from regular content
		mp.extractLinksAndImages(line, structure)
		
		// Add to current paragraph
		if currentParagraph.Len() > 0 {
			currentParagraph.WriteString(" ")
		}
		currentParagraph.WriteString(mp.stripMarkdown(line))
	}
	
	// Add final paragraph/list
	if currentParagraph.Len() > 0 {
		paragraphs = append(paragraphs, currentParagraph.String())
	}
	if currentList != nil {
		structure.Lists = append(structure.Lists, *currentList)
	}
	
	structure.Paragraphs = paragraphs
	
	return structure
}

// processContent processes Markdown content based on configuration
func (mp *MarkdownParser) processContent(content string, config *ParserConfig) string {
	if !config.PreserveFormatting {
		// Convert Markdown to plain text
		content = mp.markdownToPlainText(content)
	}
	
	// Remove front matter unless preserving formatting
	if !config.PreserveFormatting {
		content = mp.removeFrontMatter(content)
	}
	
	return content
}

// markdownToPlainText converts Markdown to plain text
func (mp *MarkdownParser) markdownToPlainText(content string) string {
	// Remove front matter
	content = mp.removeFrontMatter(content)
	
	// Convert headings
	headingRegex := regexp.MustCompile(`^#{1,6}\s+(.+)$`)
	content = headingRegex.ReplaceAllString(content, "$1")
	
	// Convert bold and italic
	content = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(content, "$1")
	content = regexp.MustCompile(`\*([^*]+)\*`).ReplaceAllString(content, "$1")
	content = regexp.MustCompile(`__([^_]+)__`).ReplaceAllString(content, "$1")
	content = regexp.MustCompile(`_([^_]+)_`).ReplaceAllString(content, "$1")
	
	// Convert links
	content = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`).ReplaceAllString(content, "$1")
	
	// Convert images
	content = regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`).ReplaceAllString(content, "$1")
	
	// Convert inline code
	content = regexp.MustCompile("`([^`]+)`").ReplaceAllString(content, "$1")
	
	// Convert code blocks
	content = regexp.MustCompile("(?s)```[^`]*```").ReplaceAllString(content, "")
	
	// Clean up list markers
	content = regexp.MustCompile(`^[-*+]\s+`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`^\d+\.\s+`).ReplaceAllString(content, "")
	
	return content
}

// Helper methods for pattern recognition

func (mp *MarkdownParser) hasMarkdownPatterns(content string) bool {
	patterns := []string{
		`#+ `,      // Headings
		`\*\*`,     // Bold
		`__`,       // Bold alternative
		`\[.*\]\(`, // Links
		`!\[.*\]\(`, // Images
		"```",      // Code blocks
		`^[-*+] `,  // Lists
	}
	
	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, content); matched {
			return true
		}
	}
	
	return false
}

func (mp *MarkdownParser) hasHeadings(content string) bool {
	matched, _ := regexp.MatchString(`^#{1,6}\s+`, content)
	return matched
}

func (mp *MarkdownParser) hasLists(content string) bool {
	matched, _ := regexp.MatchString(`^[-*+]\s+|^\d+\.\s+`, content)
	return matched
}

func (mp *MarkdownParser) hasLinks(content string) bool {
	matched, _ := regexp.MatchString(`\[.*\]\(.*\)`, content)
	return matched
}

func (mp *MarkdownParser) hasImages(content string) bool {
	matched, _ := regexp.MatchString(`!\[.*\]\(.*\)`, content)
	return matched
}

func (mp *MarkdownParser) hasCodeBlocks(content string) bool {
	return strings.Contains(content, "```") || strings.Contains(content, "    ") // Indented code
}

func (mp *MarkdownParser) hasTables(content string) bool {
	return strings.Contains(content, "|")
}

func (mp *MarkdownParser) hasFrontMatter(content string) bool {
	return strings.HasPrefix(content, "---") || strings.HasPrefix(content, "+++")
}

// Parsing helper methods

func (mp *MarkdownParser) parseHeading(line string) *Heading {
	headingRegex := regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	matches := headingRegex.FindStringSubmatch(line)
	if len(matches) < 3 {
		return nil
	}
	
	return &Heading{
		Level: len(matches[1]),
		Text:  matches[2],
	}
}

type listItemResult struct {
	Type string
	Text string
}

func (mp *MarkdownParser) parseListItem(line string) *listItemResult {
	// Unordered list
	unorderedRegex := regexp.MustCompile(`^[-*+]\s+(.+)$`)
	if matches := unorderedRegex.FindStringSubmatch(line); len(matches) > 1 {
		return &listItemResult{Type: "unordered", Text: matches[1]}
	}
	
	// Ordered list
	orderedRegex := regexp.MustCompile(`^\d+\.\s+(.+)$`)
	if matches := orderedRegex.FindStringSubmatch(line); len(matches) > 1 {
		return &listItemResult{Type: "ordered", Text: matches[1]}
	}
	
	return nil
}

func (mp *MarkdownParser) isCodeBlockStart(line string) bool {
	return strings.HasPrefix(line, "```")
}

func (mp *MarkdownParser) parseCodeBlock(lines []string, index *int) *CodeBlock {
	if *index >= len(lines) {
		return nil
	}
	
	startLine := lines[*index]
	language := strings.TrimPrefix(startLine, "```")
	language = strings.TrimSpace(language)
	
	var codeLines []string
	*index++
	
	for *index < len(lines) {
		line := lines[*index]
		if strings.HasPrefix(line, "```") {
			break
		}
		codeLines = append(codeLines, line)
		*index++
	}
	
	return &CodeBlock{
		Language: language,
		Code:     strings.Join(codeLines, "\n"),
	}
}

func (mp *MarkdownParser) isTableRow(line string) bool {
	return strings.Contains(line, "|")
}

func (mp *MarkdownParser) parseTable(lines []string, index *int) *Table {
	if *index >= len(lines) {
		return nil
	}
	
	table := &Table{}
	
	// Parse header row
	headerLine := lines[*index]
	headers := mp.parseTableRow(headerLine)
	table.Headers = headers
	*index++
	
	// Skip separator row if present
	if *index < len(lines) && mp.isTableSeparator(lines[*index]) {
		*index++
	}
	
	// Parse data rows
	for *index < len(lines) {
		line := lines[*index]
		if !mp.isTableRow(line) {
			break
		}
		
		row := mp.parseTableRow(line)
		table.Rows = append(table.Rows, row)
		*index++
	}
	
	*index-- // Back up one since the loop will increment
	
	return table
}

func (mp *MarkdownParser) parseTableRow(line string) []string {
	cells := strings.Split(line, "|")
	var result []string
	
	for _, cell := range cells {
		cell = strings.TrimSpace(cell)
		if cell != "" {
			result = append(result, cell)
		}
	}
	
	return result
}

func (mp *MarkdownParser) isTableSeparator(line string) bool {
	// Check if line contains only |, -, :, and spaces
	cleaned := strings.ReplaceAll(line, "|", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, ":", "")
	cleaned = strings.TrimSpace(cleaned)
	return cleaned == ""
}

func (mp *MarkdownParser) extractLinksAndImages(line string, structure *StructuredContent) {
	// Extract links
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	linkMatches := linkRegex.FindAllStringSubmatch(line, -1)
	for _, match := range linkMatches {
		if len(match) >= 3 {
			structure.Links = append(structure.Links, Link{
				Text: match[1],
				URL:  match[2],
			})
		}
	}
	
	// Extract images
	imageRegex := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	imageMatches := imageRegex.FindAllStringSubmatch(line, -1)
	for _, match := range imageMatches {
		if len(match) >= 3 {
			structure.Images = append(structure.Images, Image{
				AltText: match[1],
				URL:     match[2],
			})
		}
	}
}

func (mp *MarkdownParser) stripMarkdown(line string) string {
	// Remove basic Markdown formatting for plain text extraction
	line = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(line, "$1")
	line = regexp.MustCompile(`\*([^*]+)\*`).ReplaceAllString(line, "$1")
	line = regexp.MustCompile(`__([^_]+)__`).ReplaceAllString(line, "$1")
	line = regexp.MustCompile(`_([^_]+)_`).ReplaceAllString(line, "$1")
	line = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`).ReplaceAllString(line, "$1")
	line = regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`).ReplaceAllString(line, "$1")
	line = regexp.MustCompile("`([^`]+)`").ReplaceAllString(line, "$1")
	
	return line
}

func (mp *MarkdownParser) removeFrontMatter(content string) string {
	// Remove YAML front matter
	yamlPattern := regexp.MustCompile(`^---\s*\n.*?\n---\s*\n`)
	content = yamlPattern.ReplaceAllString(content, "")
	
	// Remove TOML front matter
	tomlPattern := regexp.MustCompile(`^\+\+\+\s*\n.*?\n\+\+\+\s*\n`)
	content = tomlPattern.ReplaceAllString(content, "")
	
	return content
}

func (mp *MarkdownParser) countWords(text string) int {
	// Count words in markdown content (excluding markup)
	plainText := mp.markdownToPlainText(text)
	return len(strings.Fields(plainText))
}