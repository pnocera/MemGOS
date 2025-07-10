package parsers

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// HTMLParser implements parsing for HTML documents
type HTMLParser struct{}

// NewHTMLParser creates a new HTML parser
func NewHTMLParser() *HTMLParser {
	return &HTMLParser{}
}

// Parse parses an HTML document from a reader
func (hp *HTMLParser) Parse(ctx context.Context, reader io.Reader, config *ParserConfig) (*ParsedDocument, error) {
	if config == nil {
		config = DefaultParserConfig()
	}
	
	startTime := time.Now()
	
	// Read all content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read HTML content: %w", err)
	}
	
	// Check file size limit
	if config.MaxFileSize > 0 && int64(len(content)) > config.MaxFileSize {
		return nil, fmt.Errorf("file size %d bytes exceeds limit %d bytes", len(content), config.MaxFileSize)
	}
	
	htmlContent := string(content)
	
	// Create document metadata
	metadata := &DocumentMetadata{
		MimeType:       "text/html",
		FileSize:       int64(len(content)),
		CharacterCount: len(htmlContent),
		Language:       "en", // Default, will be updated if detected
	}
	
	// Extract HTML metadata if requested
	if config.ExtractMetadata {
		hp.extractHTMLMetadata(htmlContent, metadata)
	}
	
	// Extract structured content
	var structuredContent *StructuredContent
	if config.ExtractStructure {
		structuredContent = hp.extractStructure(htmlContent)
	}
	
	// Process content based on configuration
	processedContent := hp.processContent(htmlContent, config)
	metadata.WordCount = hp.countWords(processedContent)
	
	// Create parsed document
	doc := &ParsedDocument{
		Content:           processedContent,
		Metadata:          metadata,
		StructuredContent: structuredContent,
		ParsedAt:          time.Now(),
		ParserType:        string(ParserTypeHTML),
		ParsingDuration:   time.Since(startTime),
	}
	
	return doc, nil
}

// ParseFile parses an HTML document from a file path
func (hp *HTMLParser) ParseFile(ctx context.Context, filePath string, config *ParserConfig) (*ParsedDocument, error) {
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
	doc, err := hp.Parse(ctx, file, config)
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

// ParseBytes parses an HTML document from byte data
func (hp *HTMLParser) ParseBytes(ctx context.Context, data []byte, filename string, config *ParserConfig) (*ParsedDocument, error) {
	if config == nil {
		config = DefaultParserConfig()
	}
	
	reader := strings.NewReader(string(data))
	doc, err := hp.Parse(ctx, reader, config)
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
func (hp *HTMLParser) SupportedTypes() []string {
	return []string{
		"text/html",
		"application/xhtml+xml",
		"text/xhtml",
	}
}

// SupportedExtensions returns the file extensions supported by this parser
func (hp *HTMLParser) SupportedExtensions() []string {
	return []string{
		".html", ".htm", ".xhtml", ".shtml",
	}
}

// GetParserType returns the type identifier for this parser
func (hp *HTMLParser) GetParserType() string {
	return string(ParserTypeHTML)
}

// ValidateInput validates if the input can be parsed by this parser
func (hp *HTMLParser) ValidateInput(ctx context.Context, reader io.Reader) error {
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
	
	// Check for HTML patterns
	if hp.hasHTMLPatterns(content) {
		return nil
	}
	
	return fmt.Errorf("content does not appear to be valid HTML")
}

// EstimateComplexity estimates parsing complexity (0-100 scale)
func (hp *HTMLParser) EstimateComplexity(ctx context.Context, reader io.Reader) (int, error) {
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
	
	// Base complexity for HTML
	complexity := 25
	
	// Increase complexity based on HTML features
	if hp.hasTables(content) {
		complexity += 20
	}
	
	if hp.hasLists(content) {
		complexity += 10
	}
	
	if hp.hasImages(content) {
		complexity += 15
	}
	
	if hp.hasLinks(content) {
		complexity += 10
	}
	
	if hp.hasForms(content) {
		complexity += 15
	}
	
	if hp.hasScripts(content) {
		complexity += 10
	}
	
	if hp.hasStylesheets(content) {
		complexity += 5
	}
	
	// Count HTML tags for additional complexity
	tagCount := strings.Count(content, "<")
	if tagCount > 50 {
		complexity += 10
	}
	if tagCount > 200 {
		complexity += 10
	}
	
	// Cap at reasonable maximum
	if complexity > 90 {
		complexity = 90
	}
	
	return complexity, nil
}

// extractHTMLMetadata extracts metadata from HTML meta tags and document structure
func (hp *HTMLParser) extractHTMLMetadata(content string, metadata *DocumentMetadata) {
	// Extract title
	if title := hp.extractTitle(content); title != "" {
		metadata.Title = title
	}
	
	// Extract meta tags
	metaTags := hp.extractMetaTags(content)
	
	for name, value := range metaTags {
		switch strings.ToLower(name) {
		case "author":
			metadata.Author = value
		case "description":
			metadata.Subject = value
		case "keywords":
			keywords := strings.Split(value, ",")
			for i, keyword := range keywords {
				keywords[i] = strings.TrimSpace(keyword)
			}
			metadata.Keywords = keywords
		case "language", "lang":
			metadata.Language = value
		}
	}
	
	// Extract language from html tag
	if lang := hp.extractLanguageFromHTML(content); lang != "" && metadata.Language == "" {
		metadata.Language = lang
	}
}

// extractStructure extracts structural elements from HTML using goquery
func (hp *HTMLParser) extractStructure(content string) *StructuredContent {
	structure := &StructuredContent{}
	
	// Parse HTML with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		// Fallback to regex-based extraction
		return hp.extractStructureFallback(content)
	}
	
	// Extract headings using goquery
	structure.Headings = hp.extractHeadingsWithGoquery(doc)
	
	// Extract paragraphs using goquery
	structure.Paragraphs = hp.extractParagraphsWithGoquery(doc)
	
	// Extract lists using goquery
	structure.Lists = hp.extractListsWithGoquery(doc)
	
	// Extract tables using goquery
	structure.Tables = hp.extractTablesWithGoquery(doc)
	
	// Extract images using goquery
	structure.Images = hp.extractImagesWithGoquery(doc)
	
	// Extract links using goquery
	structure.Links = hp.extractLinksWithGoquery(doc)
	
	// Extract code blocks using goquery
	structure.CodeBlocks = hp.extractCodeBlocksWithGoquery(doc)
	
	return structure
}

// processContent processes HTML content based on configuration
func (hp *HTMLParser) processContent(content string, config *ParserConfig) string {
	if !config.PreserveFormatting {
		// Convert HTML to plain text
		content = hp.htmlToPlainText(content)
	}
	
	return content
}

// htmlToPlainText converts HTML to plain text
func (hp *HTMLParser) htmlToPlainText(content string) string {
	// Remove script and style content
	content = hp.removeScriptAndStyle(content)
	
	// Replace common block elements with newlines
	blockElements := []string{"div", "p", "h1", "h2", "h3", "h4", "h5", "h6", "li", "br"}
	for _, element := range blockElements {
		pattern := fmt.Sprintf("(?i)<%s[^>]*>", element)
		re := regexp.MustCompile(pattern)
		content = re.ReplaceAllString(content, "\n")
		
		pattern = fmt.Sprintf("(?i)</%s>", element)
		re = regexp.MustCompile(pattern)
		content = re.ReplaceAllString(content, "\n")
	}
	
	// Remove all remaining HTML tags
	htmlTagRegex := regexp.MustCompile(`<[^>]*>`)
	content = htmlTagRegex.ReplaceAllString(content, "")
	
	// Decode HTML entities
	content = hp.decodeHTMLEntities(content)
	
	// Normalize whitespace
	content = hp.normalizeWhitespace(content)
	
	return content
}

// Helper methods for HTML parsing

func (hp *HTMLParser) hasHTMLPatterns(content string) bool {
	patterns := []string{
		`<!DOCTYPE html>`,
		`<html[^>]*>`,
		`<head[^>]*>`,
		`<body[^>]*>`,
		`<[a-zA-Z][^>]*>`,
	}
	
	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(`(?i)`+pattern, content); matched {
			return true
		}
	}
	
	return false
}

func (hp *HTMLParser) hasTables(content string) bool {
	matched, _ := regexp.MatchString(`(?i)<table[^>]*>`, content)
	return matched
}

func (hp *HTMLParser) hasLists(content string) bool {
	matched, _ := regexp.MatchString(`(?i)<[ou]l[^>]*>`, content)
	return matched
}

func (hp *HTMLParser) hasImages(content string) bool {
	matched, _ := regexp.MatchString(`(?i)<img[^>]*>`, content)
	return matched
}

func (hp *HTMLParser) hasLinks(content string) bool {
	matched, _ := regexp.MatchString(`(?i)<a[^>]*href`, content)
	return matched
}

func (hp *HTMLParser) hasForms(content string) bool {
	matched, _ := regexp.MatchString(`(?i)<form[^>]*>`, content)
	return matched
}

func (hp *HTMLParser) hasScripts(content string) bool {
	matched, _ := regexp.MatchString(`(?i)<script[^>]*>`, content)
	return matched
}

func (hp *HTMLParser) hasStylesheets(content string) bool {
	matched, _ := regexp.MatchString(`(?i)<style[^>]*>|<link[^>]*stylesheet`, content)
	return matched
}

func (hp *HTMLParser) extractTitle(content string) string {
	titleRegex := regexp.MustCompile(`(?i)<title[^>]*>(.*?)</title>`)
	matches := titleRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(hp.decodeHTMLEntities(matches[1]))
	}
	return ""
}

func (hp *HTMLParser) extractMetaTags(content string) map[string]string {
	metaTags := make(map[string]string)
	
	metaRegex := regexp.MustCompile(`(?i)<meta[^>]+>`)
	matches := metaRegex.FindAllString(content, -1)
	
	for _, match := range matches {
		name := hp.extractAttribute(match, "name")
		if name == "" {
			name = hp.extractAttribute(match, "property")
		}
		content := hp.extractAttribute(match, "content")
		
		if name != "" && content != "" {
			metaTags[name] = content
		}
	}
	
	return metaTags
}

func (hp *HTMLParser) extractAttribute(tag, attrName string) string {
	pattern := fmt.Sprintf(`(?i)%s\s*=\s*["']([^"']*)["']`, attrName)
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(tag)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (hp *HTMLParser) extractLanguageFromHTML(content string) string {
	htmlRegex := regexp.MustCompile(`(?i)<html[^>]*lang\s*=\s*["']([^"']*)["']`)
	matches := htmlRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (hp *HTMLParser) extractHeadings(content string) []Heading {
	var headings []Heading
	
	for level := 1; level <= 6; level++ {
		pattern := fmt.Sprintf(`(?i)<h%d[^>]*>(.*?)</h%d>`, level, level)
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(content, -1)
		
		for _, match := range matches {
			if len(match) > 1 {
				text := hp.stripHTMLTags(match[1])
				text = hp.decodeHTMLEntities(text)
				text = strings.TrimSpace(text)
				
				if text != "" {
					headings = append(headings, Heading{
						Level: level,
						Text:  text,
					})
				}
			}
		}
	}
	
	return headings
}

func (hp *HTMLParser) extractParagraphs(content string) []string {
	var paragraphs []string
	
	paragraphRegex := regexp.MustCompile(`(?i)<p[^>]*>(.*?)</p>`)
	matches := paragraphRegex.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			text := hp.stripHTMLTags(match[1])
			text = hp.decodeHTMLEntities(text)
			text = strings.TrimSpace(text)
			
			if text != "" {
				paragraphs = append(paragraphs, text)
			}
		}
	}
	
	return paragraphs
}

func (hp *HTMLParser) extractLists(content string) []List {
	var lists []List
	
	// Extract unordered lists
	ulRegex := regexp.MustCompile(`(?i)<ul[^>]*>(.*?)</ul>`)
	ulMatches := ulRegex.FindAllStringSubmatch(content, -1)
	
	for _, match := range ulMatches {
		if len(match) > 1 {
			items := hp.extractListItems(match[1])
			if len(items) > 0 {
				lists = append(lists, List{
					Type:  "unordered",
					Items: items,
				})
			}
		}
	}
	
	// Extract ordered lists
	olRegex := regexp.MustCompile(`(?i)<ol[^>]*>(.*?)</ol>`)
	olMatches := olRegex.FindAllStringSubmatch(content, -1)
	
	for _, match := range olMatches {
		if len(match) > 1 {
			items := hp.extractListItems(match[1])
			if len(items) > 0 {
				lists = append(lists, List{
					Type:  "ordered",
					Items: items,
				})
			}
		}
	}
	
	return lists
}

func (hp *HTMLParser) extractListItems(listContent string) []string {
	var items []string
	
	liRegex := regexp.MustCompile(`(?i)<li[^>]*>(.*?)</li>`)
	matches := liRegex.FindAllStringSubmatch(listContent, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			text := hp.stripHTMLTags(match[1])
			text = hp.decodeHTMLEntities(text)
			text = strings.TrimSpace(text)
			
			if text != "" {
				items = append(items, text)
			}
		}
	}
	
	return items
}

func (hp *HTMLParser) extractTables(content string) []Table {
	var tables []Table
	
	tableRegex := regexp.MustCompile(`(?i)<table[^>]*>(.*?)</table>`)
	matches := tableRegex.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			table := hp.parseTable(match[1])
			if len(table.Rows) > 0 || len(table.Headers) > 0 {
				tables = append(tables, table)
			}
		}
	}
	
	return tables
}

func (hp *HTMLParser) parseTable(tableContent string) Table {
	table := Table{}
	
	// Extract headers from thead or first tr
	theadRegex := regexp.MustCompile(`(?i)<thead[^>]*>(.*?)</thead>`)
	if theadMatch := theadRegex.FindStringSubmatch(tableContent); len(theadMatch) > 1 {
		table.Headers = hp.extractTableCells(theadMatch[1], "th")
	} else {
		// Try first row as headers
		trRegex := regexp.MustCompile(`(?i)<tr[^>]*>(.*?)</tr>`)
		if trMatch := trRegex.FindStringSubmatch(tableContent); len(trMatch) > 1 {
			if headers := hp.extractTableCells(trMatch[1], "th"); len(headers) > 0 {
				table.Headers = headers
			}
		}
	}
	
	// Extract all rows
	trRegex := regexp.MustCompile(`(?i)<tr[^>]*>(.*?)</tr>`)
	trMatches := trRegex.FindAllStringSubmatch(tableContent, -1)
	
	for i, trMatch := range trMatches {
		if len(trMatch) > 1 {
			// Skip first row if it was used as headers
			if i == 0 && len(table.Headers) > 0 {
				continue
			}
			
			cells := hp.extractTableCells(trMatch[1], "td")
			if len(cells) > 0 {
				table.Rows = append(table.Rows, cells)
			}
		}
	}
	
	return table
}

func (hp *HTMLParser) extractTableCells(rowContent, cellTag string) []string {
	var cells []string
	
	pattern := fmt.Sprintf(`(?i)<%s[^>]*>(.*?)</%s>`, cellTag, cellTag)
	cellRegex := regexp.MustCompile(pattern)
	matches := cellRegex.FindAllStringSubmatch(rowContent, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			text := hp.stripHTMLTags(match[1])
			text = hp.decodeHTMLEntities(text)
			text = strings.TrimSpace(text)
			cells = append(cells, text)
		}
	}
	
	return cells
}

func (hp *HTMLParser) extractImages(content string) []Image {
	var images []Image
	
	imgRegex := regexp.MustCompile(`(?i)<img[^>]*>`)
	matches := imgRegex.FindAllString(content, -1)
	
	for _, match := range matches {
		src := hp.extractAttribute(match, "src")
		alt := hp.extractAttribute(match, "alt")
		title := hp.extractAttribute(match, "title")
		
		if src != "" {
			image := Image{
				URL:     src,
				AltText: alt,
				Caption: title,
			}
			images = append(images, image)
		}
	}
	
	return images
}

func (hp *HTMLParser) extractLinks(content string) []Link {
	var links []Link
	
	linkRegex := regexp.MustCompile(`(?i)<a[^>]*href\s*=\s*["']([^"']*)["'][^>]*>(.*?)</a>`)
	matches := linkRegex.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) > 2 {
			href := match[1]
			text := hp.stripHTMLTags(match[2])
			text = hp.decodeHTMLEntities(text)
			text = strings.TrimSpace(text)
			
			// Validate URL
			if _, err := url.Parse(href); err == nil && text != "" {
				links = append(links, Link{
					URL:  href,
					Text: text,
				})
			}
		}
	}
	
	return links
}

func (hp *HTMLParser) extractCodeBlocks(content string) []CodeBlock {
	var codeBlocks []CodeBlock
	
	// Extract <pre><code> blocks
	preCodeRegex := regexp.MustCompile(`(?i)<pre[^>]*><code[^>]*>(.*?)</code></pre>`)
	matches := preCodeRegex.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			code := hp.decodeHTMLEntities(match[1])
			codeBlocks = append(codeBlocks, CodeBlock{
				Code: code,
			})
		}
	}
	
	// Extract standalone <code> blocks
	codeRegex := regexp.MustCompile(`(?i)<code[^>]*>(.*?)</code>`)
	codeMatches := codeRegex.FindAllStringSubmatch(content, -1)
	
	for _, match := range codeMatches {
		if len(match) > 1 {
			code := hp.decodeHTMLEntities(match[1])
			if !strings.Contains(code, "\n") && len(code) < 100 {
				// Skip inline code snippets
				continue
			}
			codeBlocks = append(codeBlocks, CodeBlock{
				Code: code,
			})
		}
	}
	
	return codeBlocks
}

func (hp *HTMLParser) removeScriptAndStyle(content string) string {
	// Remove script tags and content
	scriptRegex := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	content = scriptRegex.ReplaceAllString(content, "")
	
	// Remove style tags and content
	styleRegex := regexp.MustCompile(`(?i)<style[^>]*>.*?</style>`)
	content = styleRegex.ReplaceAllString(content, "")
	
	return content
}

func (hp *HTMLParser) stripHTMLTags(content string) string {
	htmlTagRegex := regexp.MustCompile(`<[^>]*>`)
	return htmlTagRegex.ReplaceAllString(content, "")
}

func (hp *HTMLParser) decodeHTMLEntities(content string) string {
	// Basic HTML entity decoding
	entities := map[string]string{
		"&amp;":    "&",
		"&lt;":     "<",
		"&gt;":     ">",
		"&quot;":   "\"",
		"&apos;":   "'",
		"&nbsp;":   " ",
		"&copy;":   "©",
		"&reg;":    "®",
		"&trade;":  "™",
		"&hellip;": "...",
		"&mdash;":  "—",
		"&ndash;":  "–",
		"&ldquo;":  """,
		"&rdquo;":  """,
		"&lsquo;":  "'",
		"&rsquo;":  "'",
	}
	
	for entity, replacement := range entities {
		content = strings.ReplaceAll(content, entity, replacement)
	}
	
	// Decode numeric entities (simplified)
	numericRegex := regexp.MustCompile(`&#(\d+);`)
	content = numericRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Simple numeric entity handling - in practice, you'd parse the number
		// and convert to the appropriate Unicode character
		return " " // Placeholder for proper implementation
	})
	
	return content
}

func (hp *HTMLParser) normalizeWhitespace(content string) string {
	// Replace multiple whitespace with single space
	re := regexp.MustCompile(`\s+`)
	content = re.ReplaceAllString(content, " ")
	
	// Clean up line breaks
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	
	return strings.TrimSpace(content)
}

func (hp *HTMLParser) countWords(text string) int {
	return len(strings.Fields(text))
}

// Goquery-based extraction methods (more reliable than regex)

func (hp *HTMLParser) extractHeadingsWithGoquery(doc *goquery.Document) []Heading {
	var headings []Heading
	
	for level := 1; level <= 6; level++ {
		selector := fmt.Sprintf("h%d", level)
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			id, _ := s.Attr("id")
			
			if text != "" {
				headings = append(headings, Heading{
					Level: level,
					Text:  text,
					ID:    id,
				})
			}
		})
	}
	
	return headings
}

func (hp *HTMLParser) extractParagraphsWithGoquery(doc *goquery.Document) []string {
	var paragraphs []string
	
	doc.Find("p").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			paragraphs = append(paragraphs, text)
		}
	})
	
	return paragraphs
}

func (hp *HTMLParser) extractListsWithGoquery(doc *goquery.Document) []List {
	var lists []List
	
	// Extract unordered lists
	doc.Find("ul").Each(func(i int, s *goquery.Selection) {
		var items []string
		s.Find("li").Each(func(j int, li *goquery.Selection) {
			text := strings.TrimSpace(li.Text())
			if text != "" {
				items = append(items, text)
			}
		})
		
		if len(items) > 0 {
			lists = append(lists, List{
				Type:  "unordered",
				Items: items,
			})
		}
	})
	
	// Extract ordered lists
	doc.Find("ol").Each(func(i int, s *goquery.Selection) {
		var items []string
		s.Find("li").Each(func(j int, li *goquery.Selection) {
			text := strings.TrimSpace(li.Text())
			if text != "" {
				items = append(items, text)
			}
		})
		
		if len(items) > 0 {
			lists = append(lists, List{
				Type:  "ordered",
				Items: items,
			})
		}
	})
	
	return lists
}

func (hp *HTMLParser) extractTablesWithGoquery(doc *goquery.Document) []Table {
	var tables []Table
	
	doc.Find("table").Each(func(i int, table *goquery.Selection) {
		tableData := Table{}
		
		// Extract caption if present
		caption := table.Find("caption").First()
		if caption.Length() > 0 {
			tableData.Caption = strings.TrimSpace(caption.Text())
		}
		
		// Extract headers from thead or first tr
		thead := table.Find("thead tr").First()
		if thead.Length() == 0 {
			// Check if first row has th elements
			firstRow := table.Find("tr").First()
			if firstRow.Find("th").Length() > 0 {
				thead = firstRow
			}
		}
		
		if thead.Length() > 0 {
			thead.Find("th, td").Each(func(j int, cell *goquery.Selection) {
				text := strings.TrimSpace(cell.Text())
				tableData.Headers = append(tableData.Headers, text)
			})
		}
		
		// Extract body rows
		var startRow int
		if len(tableData.Headers) > 0 {
			startRow = 1 // Skip header row
		}
		
		table.Find("tr").Each(func(j int, row *goquery.Selection) {
			if j < startRow {
				return // Skip header row
			}
			
			var cells []string
			row.Find("td, th").Each(func(k int, cell *goquery.Selection) {
				text := strings.TrimSpace(cell.Text())
				cells = append(cells, text)
			})
			
			if len(cells) > 0 {
				tableData.Rows = append(tableData.Rows, cells)
			}
		})
		
		if len(tableData.Rows) > 0 || len(tableData.Headers) > 0 {
			tables = append(tables, tableData)
		}
	})
	
	return tables
}

func (hp *HTMLParser) extractImagesWithGoquery(doc *goquery.Document) []Image {
	var images []Image
	
	doc.Find("img").Each(func(i int, img *goquery.Selection) {
		src, exists := img.Attr("src")
		if !exists || src == "" {
			return
		}
		
		alt, _ := img.Attr("alt")
		title, _ := img.Attr("title")
		
		image := Image{
			URL:     src,
			AltText: alt,
			Caption: title,
		}
		
		// Try to extract dimensions
		if width, exists := img.Attr("width"); exists {
			// Convert to int if needed
			image.Width = 0 // Placeholder - would need proper conversion
		}
		
		if height, exists := img.Attr("height"); exists {
			// Convert to int if needed
			image.Height = 0 // Placeholder - would need proper conversion
		}
		
		images = append(images, image)
	})
	
	return images
}

func (hp *HTMLParser) extractLinksWithGoquery(doc *goquery.Document) []Link {
	var links []Link
	
	doc.Find("a[href]").Each(func(i int, link *goquery.Selection) {
		href, exists := link.Attr("href")
		if !exists || href == "" {
			return
		}
		
		text := strings.TrimSpace(link.Text())
		if text == "" {
			return
		}
		
		// Validate URL
		if _, err := url.Parse(href); err == nil {
			links = append(links, Link{
				URL:  href,
				Text: text,
			})
		}
	})
	
	return links
}

func (hp *HTMLParser) extractCodeBlocksWithGoquery(doc *goquery.Document) []CodeBlock {
	var codeBlocks []CodeBlock
	
	// Extract <pre><code> blocks
	doc.Find("pre code").Each(func(i int, code *goquery.Selection) {
		text := code.Text()
		if text != "" {
			language, _ := code.Attr("class")
			// Extract language from class like "language-go" or "lang-python"
			if strings.HasPrefix(language, "language-") {
				language = strings.TrimPrefix(language, "language-")
			} else if strings.HasPrefix(language, "lang-") {
				language = strings.TrimPrefix(language, "lang-")
			}
			
			codeBlocks = append(codeBlocks, CodeBlock{
				Language: language,
				Code:     text,
			})
		}
	})
	
	// Extract standalone <code> blocks (only if they're multiline)
	doc.Find("code").Not("pre code").Each(func(i int, code *goquery.Selection) {
		text := code.Text()
		if text != "" && strings.Contains(text, "\n") && len(text) > 50 {
			codeBlocks = append(codeBlocks, CodeBlock{
				Code: text,
			})
		}
	})
	
	return codeBlocks
}

// extractStructureFallback provides fallback extraction using regex
func (hp *HTMLParser) extractStructureFallback(content string) *StructuredContent {
	structure := &StructuredContent{}
	
	// Use the existing regex-based methods as fallback
	structure.Headings = hp.extractHeadings(content)
	structure.Paragraphs = hp.extractParagraphs(content)
	structure.Lists = hp.extractLists(content)
	structure.Tables = hp.extractTables(content)
	structure.Images = hp.extractImages(content)
	structure.Links = hp.extractLinks(content)
	structure.CodeBlocks = hp.extractCodeBlocks(content)
	
	return structure
}