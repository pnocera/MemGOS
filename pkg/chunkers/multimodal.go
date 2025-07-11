package chunkers

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// MultiModalChunker handles documents with mixed content types including
// text, code blocks, tables, images, and other structured elements
type MultiModalChunker struct {
	config    *ChunkerConfig
	
	// Content processors for different modalities
	textProcessor  Chunker
	codeProcessor  *CodeBlockProcessor
	tableProcessor *TableProcessor
	imageProcessor *ImageProcessor
	
	// Token estimation
	tokenizer      TokenizerProvider
	tokenEstimator func(string) int
	
	// Multi-modal configuration
	multiModalConfig *MultiModalChunkerConfig
}

// MultiModalChunkerConfig contains configuration for multi-modal chunking
type MultiModalChunkerConfig struct {
	// PreserveCodeBlocks keeps code blocks as single units
	PreserveCodeBlocks bool `json:"preserve_code_blocks"`
	
	// PreserveTables keeps tables as single units
	PreserveTables bool `json:"preserve_tables"`
	
	// HandleImages processes image references and metadata
	HandleImages bool `json:"handle_images"`
	
	// SplitLargeTables splits tables that exceed size limits
	SplitLargeTables bool `json:"split_large_tables"`
	
	// MaxTableSize is the maximum size for a table chunk in tokens
	MaxTableSize int `json:"max_table_size"`
	
	// CodeLanguages specifies which code languages to detect
	CodeLanguages []string `json:"code_languages"`
	
	// ImageFormats specifies which image formats to handle
	ImageFormats []string `json:"image_formats"`
	
	// MarkdownSupport enables enhanced Markdown processing
	MarkdownSupport bool `json:"markdown_support"`
	
	// HTMLSupport enables HTML tag processing
	HTMLSupport bool `json:"html_support"`
	
	// ContentTypeDetection enables automatic content type detection
	ContentTypeDetection bool `json:"content_type_detection"`
	
	// ContextualMetadata adds metadata about surrounding content
	ContextualMetadata bool `json:"contextual_metadata"`
}

// DefaultMultiModalConfig returns a default configuration for multi-modal chunking
func DefaultMultiModalConfig() *MultiModalChunkerConfig {
	return &MultiModalChunkerConfig{
		PreserveCodeBlocks:     true,
		PreserveTables:         true,
		HandleImages:           true,
		SplitLargeTables:       true,
		MaxTableSize:           1000,
		CodeLanguages:          []string{"python", "javascript", "go", "java", "cpp", "c", "rust", "sql", "bash", "yaml", "json", "xml"},
		ImageFormats:           []string{"png", "jpg", "jpeg", "gif", "svg", "webp"},
		MarkdownSupport:        true,
		HTMLSupport:            true,
		ContentTypeDetection:   true,
		ContextualMetadata:     true,
	}
}

// NewMultiModalChunker creates a new multi-modal chunker
func NewMultiModalChunker(config *ChunkerConfig) (*MultiModalChunker, error) {
	if config == nil {
		config = DefaultChunkerConfig()
	}
	
	// Create text processor (use semantic chunker as base)
	textProcessor, err := NewSemanticChunker(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create text processor: %w", err)
	}
	
	// Initialize tokenizer
	tokenizerFactory := NewTokenizerFactory()
	tokenizerConfig := DefaultTokenizerConfig()
	tokenizer, err := tokenizerFactory.CreateTokenizer(tokenizerConfig)
	if err != nil {
		fmt.Printf("Warning: advanced tokenizer not available, using fallback: %v\n", err)
		tokenizer = nil
	}
	
	chunker := &MultiModalChunker{
		config:           config,
		textProcessor:    textProcessor,
		codeProcessor:    NewCodeBlockProcessor(),
		tableProcessor:   NewTableProcessor(),
		imageProcessor:   NewImageProcessor(),
		tokenizer:        tokenizer,
		tokenEstimator:   defaultTokenEstimator,
		multiModalConfig: DefaultMultiModalConfig(),
	}
	
	return chunker, nil
}

// SetMultiModalConfig updates the multi-modal configuration
func (mmc *MultiModalChunker) SetMultiModalConfig(config *MultiModalChunkerConfig) {
	if config != nil {
		mmc.multiModalConfig = config
	}
}

// GetMultiModalConfig returns the current multi-modal configuration
func (mmc *MultiModalChunker) GetMultiModalConfig() *MultiModalChunkerConfig {
	return mmc.multiModalConfig
}

// Chunk splits multi-modal content into appropriate chunks
func (mmc *MultiModalChunker) Chunk(ctx context.Context, text string) ([]*Chunk, error) {
	return mmc.ChunkWithMetadata(ctx, text, nil)
}

// ChunkWithMetadata splits multi-modal content with metadata preservation
func (mmc *MultiModalChunker) ChunkWithMetadata(ctx context.Context, text string, metadata map[string]interface{}) ([]*Chunk, error) {
	if text == "" {
		return []*Chunk{}, nil
	}
	
	startTime := time.Now()
	
	// Step 1: Detect and extract different content types
	contentBlocks := mmc.detectContentBlocks(text)
	
	// Step 2: Process each content block according to its type
	var allChunks []*Chunk
	for i, block := range contentBlocks {
		chunks, err := mmc.processContentBlock(ctx, block, i, metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to process content block %d: %w", i, err)
		}
		allChunks = append(allChunks, chunks...)
	}
	
	// Step 3: Apply cross-block optimization
	optimizedChunks := mmc.optimizeChunks(allChunks)
	
	// Step 4: Add multi-modal metadata
	processingTime := time.Since(startTime)
	stats := CalculateStats(optimizedChunks, len(text), processingTime)
	
	for _, chunk := range optimizedChunks {
		if chunk.Metadata == nil {
			chunk.Metadata = make(map[string]interface{})
		}
		chunk.Metadata["chunking_stats"] = stats
		chunk.Metadata["chunker_type"] = "multimodal"
		chunk.Metadata["chunk_config"] = mmc.config
		chunk.Metadata["multimodal_config"] = mmc.multiModalConfig
		chunk.Metadata["processing_time"] = processingTime
	}
	
	return optimizedChunks, nil
}

// ContentBlockType represents different types of content blocks
type ContentBlockType string

const (
	ContentBlockText     ContentBlockType = "text"
	ContentBlockCode     ContentBlockType = "code"
	ContentBlockTable    ContentBlockType = "table"
	ContentBlockImage    ContentBlockType = "image"
	ContentBlockHTML     ContentBlockType = "html"
	ContentBlockMarkdown ContentBlockType = "markdown"
	ContentBlockMixed    ContentBlockType = "mixed"
)

// ContentBlock represents a block of content with a specific type
type ContentBlock struct {
	Type        ContentBlockType       `json:"type"`
	Content     string                 `json:"content"`
	StartIndex  int                    `json:"start_index"`
	EndIndex    int                    `json:"end_index"`
	Language    string                 `json:"language,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Attributes  map[string]string      `json:"attributes,omitempty"`
}

// detectContentBlocks analyzes text and identifies different content types
func (mmc *MultiModalChunker) detectContentBlocks(text string) []*ContentBlock {
	var blocks []*ContentBlock
	
	// If content type detection is disabled, treat everything as text
	if !mmc.multiModalConfig.ContentTypeDetection {
		return []*ContentBlock{{
			Type:       ContentBlockText,
			Content:    text,
			StartIndex: 0,
			EndIndex:   len(text),
		}}
	}
	
	position := 0
	
	// Detect code blocks first (they have highest precedence)
	if mmc.multiModalConfig.PreserveCodeBlocks {
		codeBlocks := mmc.detectCodeBlocks(text)
		position = mmc.integrateBlocks(&blocks, codeBlocks, text, position)
	}
	
	// Detect tables
	if mmc.multiModalConfig.PreserveTables {
		tableBlocks := mmc.detectTables(text)
		position = mmc.integrateBlocks(&blocks, tableBlocks, text, position)
	}
	
	// Detect images
	if mmc.multiModalConfig.HandleImages {
		imageBlocks := mmc.detectImages(text)
		position = mmc.integrateBlocks(&blocks, imageBlocks, text, position)
	}
	
	// Detect HTML if enabled
	if mmc.multiModalConfig.HTMLSupport {
		htmlBlocks := mmc.detectHTML(text)
		position = mmc.integrateBlocks(&blocks, htmlBlocks, text, position)
	}
	
	// Fill remaining gaps with text blocks
	mmc.fillTextBlocks(&blocks, text)
	
	// Sort blocks by position
	mmc.sortBlocksByPosition(blocks)
	
	return blocks
}

// detectCodeBlocks finds code blocks in the text
func (mmc *MultiModalChunker) detectCodeBlocks(text string) []*ContentBlock {
	var blocks []*ContentBlock
	
	// Detect fenced code blocks (```language)
	fencedPattern := regexp.MustCompile("(?s)```(\\w+)?\\s*\\n(.*?)\\n```")
	matches := fencedPattern.FindAllStringSubmatch(text, -1)
	indices := fencedPattern.FindAllStringIndex(text, -1)
	
	for i, match := range matches {
		language := "text"
		if len(match) > 1 && match[1] != "" {
			language = match[1]
		}
		
		blocks = append(blocks, &ContentBlock{
			Type:       ContentBlockCode,
			Content:    match[0],
			StartIndex: indices[i][0],
			EndIndex:   indices[i][1],
			Language:   language,
			Metadata: map[string]interface{}{
				"fenced":    true,
				"raw_code":  match[2],
			},
		})
	}
	
	// Detect indented code blocks (4+ spaces)
	lines := strings.Split(text, "\n")
	inCodeBlock := false
	codeStart := -1
	var codeLines []string
	
	for i, line := range lines {
		isCodeLine := strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t")
		isEmpty := strings.TrimSpace(line) == ""
		
		if isCodeLine && !inCodeBlock {
			inCodeBlock = true
			codeStart = i
			codeLines = []string{line}
		} else if isCodeLine && inCodeBlock {
			codeLines = append(codeLines, line)
		} else if !isCodeLine && !isEmpty && inCodeBlock {
			// End of code block
			mmc.addIndentedCodeBlock(&blocks, codeLines, codeStart, text)
			inCodeBlock = false
			codeLines = nil
		} else if isEmpty && inCodeBlock {
			// Allow empty lines in code blocks
			codeLines = append(codeLines, line)
		}
	}
	
	// Handle code block at end of text
	if inCodeBlock && len(codeLines) > 0 {
		mmc.addIndentedCodeBlock(&blocks, codeLines, codeStart, text)
	}
	
	return blocks
}

// addIndentedCodeBlock adds an indented code block to the blocks list
func (mmc *MultiModalChunker) addIndentedCodeBlock(blocks *[]*ContentBlock, codeLines []string, startLine int, fullText string) {
	if len(codeLines) < 2 { // Require at least 2 lines for indented code
		return
	}
	
	codeContent := strings.Join(codeLines, "\n")
	
	// Find position in original text
	textLines := strings.Split(fullText, "\n")
	startPos := 0
	for i := 0; i < startLine; i++ {
		startPos += len(textLines[i]) + 1 // +1 for newline
	}
	
	*blocks = append(*blocks, &ContentBlock{
		Type:       ContentBlockCode,
		Content:    codeContent,
		StartIndex: startPos,
		EndIndex:   startPos + len(codeContent),
		Language:   "text",
		Metadata: map[string]interface{}{
			"indented": true,
			"raw_code": strings.Join(codeLines, "\n"),
		},
	})
}

// detectTables finds table structures in the text
func (mmc *MultiModalChunker) detectTables(text string) []*ContentBlock {
	var blocks []*ContentBlock
	
	// Detect Markdown tables
	if mmc.multiModalConfig.MarkdownSupport {
		markdownTables := mmc.detectMarkdownTables(text)
		blocks = append(blocks, markdownTables...)
	}
	
	// Detect HTML tables
	if mmc.multiModalConfig.HTMLSupport {
		htmlTables := mmc.detectHTMLTables(text)
		blocks = append(blocks, htmlTables...)
	}
	
	return blocks
}

// detectMarkdownTables finds Markdown table structures
func (mmc *MultiModalChunker) detectMarkdownTables(text string) []*ContentBlock {
	var blocks []*ContentBlock
	
	// Pattern for Markdown tables
	tablePattern := regexp.MustCompile(`(?m)^\|.*\|$\n^\|[-:\s\|]+\|$(?:\n^\|.*\|$)*`)
	matches := tablePattern.FindAllString(text, -1)
	indices := tablePattern.FindAllStringIndex(text, -1)
	
	for i, match := range matches {
		blocks = append(blocks, &ContentBlock{
			Type:       ContentBlockTable,
			Content:    match,
			StartIndex: indices[i][0],
			EndIndex:   indices[i][1],
			Metadata: map[string]interface{}{
				"format":    "markdown",
				"rows":      strings.Count(match, "\n") + 1,
				"columns":   mmc.countTableColumns(match),
			},
		})
	}
	
	return blocks
}

// detectHTMLTables finds HTML table structures
func (mmc *MultiModalChunker) detectHTMLTables(text string) []*ContentBlock {
	var blocks []*ContentBlock
	
	// Pattern for HTML tables
	tablePattern := regexp.MustCompile(`(?si)<table[^>]*>.*?</table>`)
	matches := tablePattern.FindAllString(text, -1)
	indices := tablePattern.FindAllStringIndex(text, -1)
	
	for i, match := range matches {
		blocks = append(blocks, &ContentBlock{
			Type:       ContentBlockTable,
			Content:    match,
			StartIndex: indices[i][0],
			EndIndex:   indices[i][1],
			Metadata: map[string]interface{}{
				"format": "html",
				"rows":   strings.Count(strings.ToLower(match), "<tr"),
				"columns": mmc.countHTMLTableColumns(match),
			},
		})
	}
	
	return blocks
}

// countTableColumns counts columns in a Markdown table
func (mmc *MultiModalChunker) countTableColumns(table string) int {
	lines := strings.Split(table, "\n")
	if len(lines) == 0 {
		return 0
	}
	
	// Count pipes in first line
	firstLine := strings.TrimSpace(lines[0])
	return strings.Count(firstLine, "|") - 1
}

// countHTMLTableColumns counts columns in an HTML table
func (mmc *MultiModalChunker) countHTMLTableColumns(table string) int {
	// Simple count of <td> or <th> tags in first row
	lowerTable := strings.ToLower(table)
	tdCount := strings.Count(lowerTable, "<td")
	thCount := strings.Count(lowerTable, "<th")
	
	if thCount > 0 {
		return thCount
	}
	return tdCount
}

// detectImages finds image references in the text
func (mmc *MultiModalChunker) detectImages(text string) []*ContentBlock {
	var blocks []*ContentBlock
	
	// Detect Markdown images ![alt](src)
	markdownImagePattern := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	matches := markdownImagePattern.FindAllStringSubmatch(text, -1)
	indices := markdownImagePattern.FindAllStringIndex(text, -1)
	
	for i, match := range matches {
		alt := match[1]
		src := match[2]
		
		blocks = append(blocks, &ContentBlock{
			Type:       ContentBlockImage,
			Content:    match[0],
			StartIndex: indices[i][0],
			EndIndex:   indices[i][1],
			Metadata: map[string]interface{}{
				"format":    "markdown",
				"alt_text":  alt,
				"src":       src,
				"extension": strings.ToLower(filepath.Ext(src)),
			},
		})
	}
	
	// Detect HTML images <img>
	htmlImagePattern := regexp.MustCompile(`(?i)<img[^>]*src\s*=\s*["']([^"']+)["'][^>]*>`)
	htmlMatches := htmlImagePattern.FindAllStringSubmatch(text, -1)
	htmlIndices := htmlImagePattern.FindAllStringIndex(text, -1)
	
	for i, match := range htmlMatches {
		src := match[1]
		
		blocks = append(blocks, &ContentBlock{
			Type:       ContentBlockImage,
			Content:    match[0],
			StartIndex: htmlIndices[i][0],
			EndIndex:   htmlIndices[i][1],
			Metadata: map[string]interface{}{
				"format":    "html",
				"src":       src,
				"extension": strings.ToLower(filepath.Ext(src)),
			},
		})
	}
	
	return blocks
}

// detectHTML finds HTML structures in the text
func (mmc *MultiModalChunker) detectHTML(text string) []*ContentBlock {
	var blocks []*ContentBlock
	
	// Detect HTML blocks (div, section, article, etc.)
	// Note: Go regexp doesn't support backreferences, so we'll use a simpler pattern
	htmlBlockPattern := regexp.MustCompile(`(?si)<(div|section|article|aside|header|footer|main)[^>]*>.*?</(?:div|section|article|aside|header|footer|main)>`)
	matches := htmlBlockPattern.FindAllString(text, -1)
	indices := htmlBlockPattern.FindAllStringIndex(text, -1)
	
	for i, match := range matches {
		blocks = append(blocks, &ContentBlock{
			Type:       ContentBlockHTML,
			Content:    match,
			StartIndex: indices[i][0],
			EndIndex:   indices[i][1],
			Metadata: map[string]interface{}{
				"format": "html",
			},
		})
	}
	
	return blocks
}

// integrateBlocks integrates detected blocks, avoiding overlaps
func (mmc *MultiModalChunker) integrateBlocks(existingBlocks *[]*ContentBlock, newBlocks []*ContentBlock, text string, position int) int {
	for _, newBlock := range newBlocks {
		// Check for overlaps with existing blocks
		hasOverlap := false
		for _, existing := range *existingBlocks {
			if mmc.blocksOverlap(existing, newBlock) {
				hasOverlap = true
				break
			}
		}
		
		if !hasOverlap {
			*existingBlocks = append(*existingBlocks, newBlock)
		}
	}
	
	return position
}

// blocksOverlap checks if two content blocks overlap
func (mmc *MultiModalChunker) blocksOverlap(block1, block2 *ContentBlock) bool {
	return !(block1.EndIndex <= block2.StartIndex || block2.EndIndex <= block1.StartIndex)
}

// fillTextBlocks fills gaps between structured blocks with text blocks
func (mmc *MultiModalChunker) fillTextBlocks(blocks *[]*ContentBlock, text string) {
	if len(*blocks) == 0 {
		// No structured blocks, everything is text
		*blocks = append(*blocks, &ContentBlock{
			Type:       ContentBlockText,
			Content:    text,
			StartIndex: 0,
			EndIndex:   len(text),
		})
		return
	}
	
	mmc.sortBlocksByPosition(*blocks)
	
	var newBlocks []*ContentBlock
	lastEnd := 0
	
	for _, block := range *blocks {
		// Add text block before this structured block
		if lastEnd < block.StartIndex {
			textContent := text[lastEnd:block.StartIndex]
			if strings.TrimSpace(textContent) != "" {
				newBlocks = append(newBlocks, &ContentBlock{
					Type:       ContentBlockText,
					Content:    textContent,
					StartIndex: lastEnd,
					EndIndex:   block.StartIndex,
				})
			}
		}
		
		newBlocks = append(newBlocks, block)
		lastEnd = block.EndIndex
	}
	
	// Add final text block if needed
	if lastEnd < len(text) {
		textContent := text[lastEnd:]
		if strings.TrimSpace(textContent) != "" {
			newBlocks = append(newBlocks, &ContentBlock{
				Type:       ContentBlockText,
				Content:    textContent,
				StartIndex: lastEnd,
				EndIndex:   len(text),
			})
		}
	}
	
	*blocks = newBlocks
}

// sortBlocksByPosition sorts blocks by their starting position
func (mmc *MultiModalChunker) sortBlocksByPosition(blocks []*ContentBlock) {
	for i := 0; i < len(blocks)-1; i++ {
		for j := i + 1; j < len(blocks); j++ {
			if blocks[i].StartIndex > blocks[j].StartIndex {
				blocks[i], blocks[j] = blocks[j], blocks[i]
			}
		}
	}
}

// processContentBlock processes a single content block according to its type
func (mmc *MultiModalChunker) processContentBlock(ctx context.Context, block *ContentBlock, blockIndex int, metadata map[string]interface{}) ([]*Chunk, error) {
	switch block.Type {
	case ContentBlockText:
		return mmc.processTextBlock(ctx, block, metadata)
	case ContentBlockCode:
		return mmc.processCodeBlock(ctx, block, metadata)
	case ContentBlockTable:
		return mmc.processTableBlock(ctx, block, metadata)
	case ContentBlockImage:
		return mmc.processImageBlock(ctx, block, metadata)
	case ContentBlockHTML:
		return mmc.processHTMLBlock(ctx, block, metadata)
	default:
		// Fallback to text processing
		return mmc.processTextBlock(ctx, block, metadata)
	}
}

// processTextBlock processes a text content block
func (mmc *MultiModalChunker) processTextBlock(ctx context.Context, block *ContentBlock, metadata map[string]interface{}) ([]*Chunk, error) {
	chunks, err := mmc.textProcessor.ChunkWithMetadata(ctx, block.Content, metadata)
	if err != nil {
		return nil, err
	}
	
	// Adjust indices and add block metadata
	for _, chunk := range chunks {
		chunk.StartIndex += block.StartIndex
		chunk.EndIndex += block.StartIndex
		if chunk.Metadata == nil {
			chunk.Metadata = make(map[string]interface{})
		}
		chunk.Metadata["content_type"] = string(block.Type)
		chunk.Metadata["block_metadata"] = block.Metadata
	}
	
	return chunks, nil
}

// processCodeBlock processes a code content block
func (mmc *MultiModalChunker) processCodeBlock(ctx context.Context, block *ContentBlock, metadata map[string]interface{}) ([]*Chunk, error) {
	// Keep code blocks as single chunks if preserve is enabled
	if mmc.multiModalConfig.PreserveCodeBlocks {
		chunk := &Chunk{
			Text:       block.Content,
			TokenCount: mmc.EstimateTokens(block.Content),
			Sentences:  []string{block.Content},
			StartIndex: block.StartIndex,
			EndIndex:   block.EndIndex,
			Metadata:   make(map[string]interface{}),
			CreatedAt:  time.Now(),
		}
		
		// Copy base metadata
		if metadata != nil {
			for k, v := range metadata {
				chunk.Metadata[k] = v
			}
		}
		
		// Add code-specific metadata
		chunk.Metadata["content_type"] = string(block.Type)
		chunk.Metadata["language"] = block.Language
		chunk.Metadata["block_metadata"] = block.Metadata
		chunk.Metadata["preserved"] = true
		
		return []*Chunk{chunk}, nil
	}
	
	// Split large code blocks if needed
	return mmc.codeProcessor.ProcessCodeBlock(block, mmc.config.ChunkSize, mmc.EstimateTokens)
}

// processTableBlock processes a table content block
func (mmc *MultiModalChunker) processTableBlock(ctx context.Context, block *ContentBlock, metadata map[string]interface{}) ([]*Chunk, error) {
	tokenCount := mmc.EstimateTokens(block.Content)
	
	// Keep small tables as single chunks
	if mmc.multiModalConfig.PreserveTables && tokenCount <= mmc.multiModalConfig.MaxTableSize {
		chunk := &Chunk{
			Text:       block.Content,
			TokenCount: tokenCount,
			Sentences:  []string{block.Content},
			StartIndex: block.StartIndex,
			EndIndex:   block.EndIndex,
			Metadata:   make(map[string]interface{}),
			CreatedAt:  time.Now(),
		}
		
		// Copy base metadata
		if metadata != nil {
			for k, v := range metadata {
				chunk.Metadata[k] = v
			}
		}
		
		// Add table-specific metadata
		chunk.Metadata["content_type"] = string(block.Type)
		chunk.Metadata["block_metadata"] = block.Metadata
		chunk.Metadata["preserved"] = true
		
		return []*Chunk{chunk}, nil
	}
	
	// Split large tables if enabled
	if mmc.multiModalConfig.SplitLargeTables {
		return mmc.tableProcessor.ProcessTable(block, mmc.config.ChunkSize, mmc.EstimateTokens)
	}
	
	// Otherwise, keep as single chunk even if large
	chunk := &Chunk{
		Text:       block.Content,
		TokenCount: tokenCount,
		Sentences:  []string{block.Content},
		StartIndex: block.StartIndex,
		EndIndex:   block.EndIndex,
		Metadata: map[string]interface{}{
			"content_type":    string(block.Type),
			"block_metadata":  block.Metadata,
			"preserved":       true,
			"oversized":       tokenCount > mmc.config.ChunkSize,
		},
		CreatedAt: time.Now(),
	}
	
	return []*Chunk{chunk}, nil
}

// processImageBlock processes an image content block
func (mmc *MultiModalChunker) processImageBlock(ctx context.Context, block *ContentBlock, metadata map[string]interface{}) ([]*Chunk, error) {
	chunk := &Chunk{
		Text:       block.Content,
		TokenCount: mmc.EstimateTokens(block.Content),
		Sentences:  []string{block.Content},
		StartIndex: block.StartIndex,
		EndIndex:   block.EndIndex,
		Metadata:   make(map[string]interface{}),
		CreatedAt:  time.Now(),
	}
	
	// Copy base metadata
	if metadata != nil {
		for k, v := range metadata {
			chunk.Metadata[k] = v
		}
	}
	
	// Add image-specific metadata
	chunk.Metadata["content_type"] = string(block.Type)
	chunk.Metadata["block_metadata"] = block.Metadata
	chunk.Metadata["is_media"] = true
	
	return []*Chunk{chunk}, nil
}

// processHTMLBlock processes an HTML content block
func (mmc *MultiModalChunker) processHTMLBlock(ctx context.Context, block *ContentBlock, metadata map[string]interface{}) ([]*Chunk, error) {
	// Process HTML content similar to text but preserve structure
	chunks, err := mmc.textProcessor.ChunkWithMetadata(ctx, block.Content, metadata)
	if err != nil {
		return nil, err
	}
	
	// Adjust indices and add HTML metadata
	for _, chunk := range chunks {
		chunk.StartIndex += block.StartIndex
		chunk.EndIndex += block.StartIndex
		if chunk.Metadata == nil {
			chunk.Metadata = make(map[string]interface{})
		}
		chunk.Metadata["content_type"] = string(block.Type)
		chunk.Metadata["block_metadata"] = block.Metadata
		chunk.Metadata["html_processed"] = true
	}
	
	return chunks, nil
}

// optimizeChunks performs cross-block optimization on chunks
func (mmc *MultiModalChunker) optimizeChunks(chunks []*Chunk) []*Chunk {
	if len(chunks) <= 1 {
		return chunks
	}
	
	// Merge small adjacent text chunks
	optimized := mmc.mergeSmallTextChunks(chunks)
	
	// Add contextual metadata
	if mmc.multiModalConfig.ContextualMetadata {
		mmc.addContextualMetadata(optimized)
	}
	
	return optimized
}

// mergeSmallTextChunks merges adjacent small text chunks
func (mmc *MultiModalChunker) mergeSmallTextChunks(chunks []*Chunk) []*Chunk {
	if len(chunks) <= 1 {
		return chunks
	}
	
	var optimized []*Chunk
	i := 0
	
	for i < len(chunks) {
		currentChunk := chunks[i]
		
		// Check if this is a small text chunk that can be merged
		if mmc.isSmallTextChunk(currentChunk) && i < len(chunks)-1 {
			nextChunk := chunks[i+1]
			
			// Merge if next chunk is also small text and combined size is reasonable
			if mmc.isSmallTextChunk(nextChunk) && 
			   currentChunk.TokenCount+nextChunk.TokenCount <= mmc.config.ChunkSize {
				
				mergedChunk := mmc.mergeChunks(currentChunk, nextChunk)
				optimized = append(optimized, mergedChunk)
				i += 2 // Skip both chunks
				continue
			}
		}
		
		optimized = append(optimized, currentChunk)
		i++
	}
	
	return optimized
}

// isSmallTextChunk checks if a chunk is a small text chunk suitable for merging
func (mmc *MultiModalChunker) isSmallTextChunk(chunk *Chunk) bool {
	if chunk == nil || chunk.Metadata == nil {
		return false
	}
	
	contentType, ok := chunk.Metadata["content_type"].(string)
	if !ok {
		return false
	}
	
	return contentType == string(ContentBlockText) && 
	       chunk.TokenCount < mmc.config.ChunkSize/2
}

// mergeChunks merges two chunks into one
func (mmc *MultiModalChunker) mergeChunks(chunk1, chunk2 *Chunk) *Chunk {
	mergedText := chunk1.Text + " " + chunk2.Text
	mergedSentences := append(chunk1.Sentences, chunk2.Sentences...)
	
	merged := &Chunk{
		Text:       mergedText,
		TokenCount: chunk1.TokenCount + chunk2.TokenCount,
		Sentences:  mergedSentences,
		StartIndex: chunk1.StartIndex,
		EndIndex:   chunk2.EndIndex,
		Metadata:   make(map[string]interface{}),
		CreatedAt:  time.Now(),
	}
	
	// Merge metadata
	if chunk1.Metadata != nil {
		for k, v := range chunk1.Metadata {
			merged.Metadata[k] = v
		}
	}
	
	merged.Metadata["merged"] = true
	merged.Metadata["original_chunks"] = 2
	
	return merged
}

// addContextualMetadata adds metadata about surrounding content
func (mmc *MultiModalChunker) addContextualMetadata(chunks []*Chunk) {
	for i, chunk := range chunks {
		if chunk.Metadata == nil {
			chunk.Metadata = make(map[string]interface{})
		}
		
		// Add position metadata
		chunk.Metadata["chunk_position"] = i
		chunk.Metadata["total_chunks"] = len(chunks)
		
		// Add context about adjacent chunks
		if i > 0 {
			prevType := chunks[i-1].Metadata["content_type"]
			chunk.Metadata["preceded_by"] = prevType
		}
		
		if i < len(chunks)-1 {
			nextType := chunks[i+1].Metadata["content_type"]
			chunk.Metadata["followed_by"] = nextType
		}
	}
}

// EstimateTokens estimates the number of tokens in text
func (mmc *MultiModalChunker) EstimateTokens(text string) int {
	if mmc.tokenizer != nil {
		count, err := mmc.tokenizer.CountTokens(text)
		if err == nil {
			return count
		}
	}
	return mmc.tokenEstimator(text)
}

// Standard interface implementations
func (mmc *MultiModalChunker) GetConfig() *ChunkerConfig {
	config := *mmc.config
	return &config
}

func (mmc *MultiModalChunker) SetConfig(config *ChunkerConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	mmc.config = config
	return nil
}

func (mmc *MultiModalChunker) GetChunkSize() int {
	return mmc.config.ChunkSize
}

func (mmc *MultiModalChunker) GetChunkOverlap() int {
	return mmc.config.ChunkOverlap
}

func (mmc *MultiModalChunker) GetSupportedLanguages() []string {
	return []string{"en"} // Currently optimized for English
}