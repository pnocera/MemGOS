package chunkers

import (
	"context"
	"fmt"
	"strings"
	"time"
	
	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// HierarchicalChunker implements hierarchical chunking with parent-child relationships
// and multi-level document summaries. It creates a tree-like structure of chunks
// where higher-level chunks contain summaries of their children.
type HierarchicalChunker struct {
	config    *ChunkerConfig
	
	// Base chunker for leaf-level chunking
	baseChunker Chunker
	
	// LLM for generating summaries (optional)
	llmProvider interfaces.LLM
	
	// Token estimation
	tokenizer      TokenizerProvider
	tokenEstimator func(string) int
	
	// Hierarchical configuration
	hierarchicalConfig *HierarchicalChunkerConfig
}

// HierarchicalChunkerConfig contains configuration for hierarchical chunking
type HierarchicalChunkerConfig struct {
	// MaxLevels is the maximum number of hierarchy levels
	MaxLevels int `json:"max_levels"`
	
	// SummaryMode controls how summaries are generated
	SummaryMode SummaryMode `json:"summary_mode"`
	
	// BranchingFactor controls how many child chunks per parent
	BranchingFactor int `json:"branching_factor"`
	
	// MinChildrenForSummary minimum children required to create summary
	MinChildrenForSummary int `json:"min_children_for_summary"`
	
	// SummaryRatio controls the target size of summaries relative to content
	SummaryRatio float64 `json:"summary_ratio"`
	
	// PreserveSectionHeaders maintains document section structure
	PreserveSectionHeaders bool `json:"preserve_section_headers"`
	
	// CrossReferenceTracking tracks relationships between chunks
	CrossReferenceTracking bool `json:"cross_reference_tracking"`
	
	// EnableMetadataPropagation propagates metadata up the hierarchy
	EnableMetadataPropagation bool `json:"enable_metadata_propagation"`
	
	// AutoBalance automatically balances the hierarchy tree
	AutoBalance bool `json:"auto_balance"`
	
	// SummaryTemplate for LLM-generated summaries
	SummaryTemplate string `json:"summary_template"`
}

// SummaryMode defines how summaries are generated
type SummaryMode string

const (
	// SummaryModeNone disables summary generation
	SummaryModeNone SummaryMode = "none"
	
	// SummaryModeExtract extracts key sentences
	SummaryModeExtract SummaryMode = "extract"
	
	// SummaryModeLLM uses LLM to generate summaries
	SummaryModeLLM SummaryMode = "llm"
	
	// SummaryModeHybrid combines extraction and LLM
	SummaryModeHybrid SummaryMode = "hybrid"
)

// DefaultHierarchicalConfig returns a default configuration for hierarchical chunking
func DefaultHierarchicalConfig() *HierarchicalChunkerConfig {
	return &HierarchicalChunkerConfig{
		MaxLevels:                 3,
		SummaryMode:               SummaryModeExtract,
		BranchingFactor:           5,
		MinChildrenForSummary:     2,
		SummaryRatio:              0.3,
		PreserveSectionHeaders:    true,
		CrossReferenceTracking:    true,
		EnableMetadataPropagation: true,
		AutoBalance:               true,
		SummaryTemplate: `Summarize the following text chunks into a concise overview that captures the main topics and key information:

{{.Chunks}}

Generate a summary that:
1. Captures the main themes and topics
2. Preserves important details
3. Maintains logical flow
4. Is approximately {{.TargetLength}} words

Summary:`,
	}
}

// NewHierarchicalChunker creates a new hierarchical chunker
func NewHierarchicalChunker(config *ChunkerConfig, llmProvider interfaces.LLM) (*HierarchicalChunker, error) {
	if config == nil {
		config = DefaultChunkerConfig()
	}
	
	// Create base chunker (use semantic chunker)
	baseChunker, err := NewSemanticChunker(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create base chunker: %w", err)
	}
	
	// Initialize tokenizer
	tokenizerFactory := NewTokenizerFactory()
	tokenizerConfig := DefaultTokenizerConfig()
	tokenizer, err := tokenizerFactory.CreateTokenizer(tokenizerConfig)
	if err != nil {
		fmt.Printf("Warning: advanced tokenizer not available, using fallback: %v\n", err)
		tokenizer = nil
	}
	
	chunker := &HierarchicalChunker{
		config:             config,
		baseChunker:        baseChunker,
		llmProvider:        llmProvider,
		tokenizer:          tokenizer,
		tokenEstimator:     defaultTokenEstimator,
		hierarchicalConfig: DefaultHierarchicalConfig(),
	}
	
	return chunker, nil
}

// SetHierarchicalConfig updates the hierarchical configuration
func (hc *HierarchicalChunker) SetHierarchicalConfig(config *HierarchicalChunkerConfig) {
	if config != nil {
		hc.hierarchicalConfig = config
	}
}

// GetHierarchicalConfig returns the current hierarchical configuration
func (hc *HierarchicalChunker) GetHierarchicalConfig() *HierarchicalChunkerConfig {
	return hc.hierarchicalConfig
}

// Chunk splits text into hierarchical chunks
func (hc *HierarchicalChunker) Chunk(ctx context.Context, text string) ([]*Chunk, error) {
	return hc.ChunkWithMetadata(ctx, text, nil)
}

// ChunkWithMetadata splits text into hierarchical chunks with metadata
func (hc *HierarchicalChunker) ChunkWithMetadata(ctx context.Context, text string, metadata map[string]interface{}) ([]*Chunk, error) {
	if text == "" {
		return []*Chunk{}, nil
	}
	
	startTime := time.Now()
	
	// Step 1: Create the hierarchical structure
	hierarchy, err := hc.buildHierarchy(ctx, text, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to build hierarchy: %w", err)
	}
	
	// Step 2: Generate summaries for parent nodes
	err = hc.generateSummaries(ctx, hierarchy)
	if err != nil {
		fmt.Printf("Warning: failed to generate summaries: %v\n", err)
	}
	
	// Step 3: Establish cross-references
	if hc.hierarchicalConfig.CrossReferenceTracking {
		hc.establishCrossReferences(hierarchy)
	}
	
	// Step 4: Balance the hierarchy if enabled
	if hc.hierarchicalConfig.AutoBalance {
		hierarchy = hc.balanceHierarchy(hierarchy)
	}
	
	// Step 5: Convert hierarchy to flat chunk list
	chunks := hc.flattenHierarchy(hierarchy)
	
	// Step 6: Add hierarchical metadata
	processingTime := time.Since(startTime)
	stats := CalculateStats(chunks, len(text), processingTime)
	
	for _, chunk := range chunks {
		if chunk.Metadata == nil {
			chunk.Metadata = make(map[string]interface{})
		}
		chunk.Metadata["chunking_stats"] = stats
		chunk.Metadata["chunker_type"] = "hierarchical"
		chunk.Metadata["chunk_config"] = hc.config
		chunk.Metadata["hierarchical_config"] = hc.hierarchicalConfig
		chunk.Metadata["processing_time"] = processingTime
	}
	
	return chunks, nil
}

// HierarchicalNode represents a node in the hierarchical structure
type HierarchicalNode struct {
	// Core chunk data
	Chunk *Chunk `json:"chunk"`
	
	// Hierarchy structure
	Level       int                  `json:"level"`
	Children    []*HierarchicalNode  `json:"children,omitempty"`
	Parent      *HierarchicalNode    `json:"-"` // Avoid circular reference in JSON
	
	// Identification
	ID          string               `json:"id"`
	ParentID    string               `json:"parent_id,omitempty"`
	
	// Summary information
	Summary     string               `json:"summary,omitempty"`
	SummaryType SummaryMode          `json:"summary_type,omitempty"`
	
	// Cross-references
	References  []string             `json:"references,omitempty"`
	ReferencedBy []string            `json:"referenced_by,omitempty"`
	
	// Section information
	SectionID   string               `json:"section_id,omitempty"`
	SectionTitle string              `json:"section_title,omitempty"`
	
	// Metadata
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// buildHierarchy constructs the initial hierarchical structure
func (hc *HierarchicalChunker) buildHierarchy(ctx context.Context, text string, metadata map[string]interface{}) (*HierarchicalNode, error) {
	// Step 1: Detect document structure
	sections := hc.detectDocumentSections(text)
	
	// Step 2: Create leaf-level chunks
	leafChunks, err := hc.createLeafChunks(ctx, text, sections, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create leaf chunks: %w", err)
	}
	
	// Step 3: Build hierarchy from leaf chunks
	root := hc.buildHierarchyFromLeaves(leafChunks)
	
	return root, nil
}

// DocumentSection represents a section in the document
type DocumentSection struct {
	Title      string `json:"title"`
	Level      int    `json:"level"`
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
	Content    string `json:"content"`
}

// detectDocumentSections identifies sections in the document
func (hc *HierarchicalChunker) detectDocumentSections(text string) []*DocumentSection {
	
	if !hc.hierarchicalConfig.PreserveSectionHeaders {
		// Treat entire text as one section
		return []*DocumentSection{{
			Title:      "Document",
			Level:      1,
			StartIndex: 0,
			EndIndex:   len(text),
			Content:    text,
		}}
	}
	
	// Detect Markdown headers
	markdownSections := hc.detectMarkdownSections(text)
	if len(markdownSections) > 1 {
		return markdownSections
	}
	
	// Detect numbered sections
	numberedSections := hc.detectNumberedSections(text)
	if len(numberedSections) > 1 {
		return numberedSections
	}
	
	// Fallback: detect paragraph-based sections
	return hc.detectParagraphSections(text)
}

// detectMarkdownSections detects Markdown-style headers
func (hc *HierarchicalChunker) detectMarkdownSections(text string) []*DocumentSection {
	var sections []*DocumentSection
	lines := strings.Split(text, "\n")
	
	currentSection := &DocumentSection{
		Title:      "Introduction",
		Level:      1,
		StartIndex: 0,
	}
	linePosition := 0
	
	for i, line := range lines {
		// Check for Markdown headers
		if strings.HasPrefix(line, "#") {
			// Finalize current section
			if i > 0 {
				currentSection.EndIndex = linePosition
				currentSection.Content = text[currentSection.StartIndex:currentSection.EndIndex]
				sections = append(sections, currentSection)
			}
			
			// Start new section
			level := 0
			for _, char := range line {
				if char == '#' {
					level++
				} else {
					break
				}
			}
			
			title := strings.TrimSpace(line[level:])
			currentSection = &DocumentSection{
				Title:      title,
				Level:      level,
				StartIndex: linePosition,
			}
		}
		
		linePosition += len(line) + 1 // +1 for newline
	}
	
	// Finalize last section
	if currentSection != nil {
		currentSection.EndIndex = len(text)
		currentSection.Content = text[currentSection.StartIndex:currentSection.EndIndex]
		sections = append(sections, currentSection)
	}
	
	return sections
}

// detectNumberedSections detects numbered sections (1. 2. etc.)
func (hc *HierarchicalChunker) detectNumberedSections(text string) []*DocumentSection {
	var sections []*DocumentSection
	lines := strings.Split(text, "\n")
	
	currentSection := &DocumentSection{
		Title:      "Introduction",
		Level:      1,
		StartIndex: 0,
	}
	linePosition := 0
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Check for numbered sections (1. 2. etc.)
		if len(trimmed) > 0 && strings.Contains(trimmed, ".") {
			parts := strings.SplitN(trimmed, ".", 2)
			if len(parts) == 2 {
				// Check if first part is a number
				num := strings.TrimSpace(parts[0])
				if hc.isNumber(num) {
					// Finalize current section
					if i > 0 {
						currentSection.EndIndex = linePosition
						currentSection.Content = text[currentSection.StartIndex:currentSection.EndIndex]
						sections = append(sections, currentSection)
					}
					
					// Start new section
					title := strings.TrimSpace(parts[1])
					currentSection = &DocumentSection{
						Title:      title,
						Level:      1,
						StartIndex: linePosition,
					}
				}
			}
		}
		
		linePosition += len(line) + 1
	}
	
	// Finalize last section
	if currentSection != nil {
		currentSection.EndIndex = len(text)
		currentSection.Content = text[currentSection.StartIndex:currentSection.EndIndex]
		sections = append(sections, currentSection)
	}
	
	return sections
}

// detectParagraphSections creates sections based on paragraph breaks
func (hc *HierarchicalChunker) detectParagraphSections(text string) []*DocumentSection {
	paragraphs := strings.Split(text, "\n\n")
	var sections []*DocumentSection
	position := 0
	
	for i, paragraph := range paragraphs {
		if strings.TrimSpace(paragraph) == "" {
			continue
		}
		
		section := &DocumentSection{
			Title:      fmt.Sprintf("Section %d", i+1),
			Level:      1,
			StartIndex: position,
			EndIndex:   position + len(paragraph),
			Content:    paragraph,
		}
		sections = append(sections, section)
		position += len(paragraph) + 2 // +2 for double newline
	}
	
	return sections
}

// isNumber checks if a string represents a number
func (hc *HierarchicalChunker) isNumber(s string) bool {
	for _, char := range s {
		if char < '0' || char > '9' {
			return false
		}
	}
	return len(s) > 0
}

// createLeafChunks creates the leaf-level chunks
func (hc *HierarchicalChunker) createLeafChunks(ctx context.Context, text string, sections []*DocumentSection, metadata map[string]interface{}) ([]*HierarchicalNode, error) {
	var leafNodes []*HierarchicalNode
	
	for i, section := range sections {
		// Chunk each section
		chunks, err := hc.baseChunker.ChunkWithMetadata(ctx, section.Content, metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to chunk section %d: %w", i, err)
		}
		
		// Convert chunks to hierarchical nodes
		for j, chunk := range chunks {
			// Adjust indices to global text
			chunk.StartIndex += section.StartIndex
			chunk.EndIndex += section.StartIndex
			
			node := &HierarchicalNode{
				Chunk:        chunk,
				Level:        0, // Leaf level
				ID:           fmt.Sprintf("leaf_%d_%d", i, j),
				SectionID:    fmt.Sprintf("section_%d", i),
				SectionTitle: section.Title,
				Metadata:     make(map[string]interface{}),
			}
			
			// Add section metadata
			node.Metadata["section_title"] = section.Title
			node.Metadata["section_level"] = section.Level
			node.Metadata["section_index"] = i
			node.Metadata["chunk_in_section"] = j
			
			leafNodes = append(leafNodes, node)
		}
	}
	
	return leafNodes, nil
}

// buildHierarchyFromLeaves constructs the hierarchy tree from leaf nodes
func (hc *HierarchicalChunker) buildHierarchyFromLeaves(leaves []*HierarchicalNode) *HierarchicalNode {
	if len(leaves) == 0 {
		return nil
	}
	
	if len(leaves) == 1 {
		return leaves[0]
	}
	
	// Build hierarchy level by level
	currentLevel := leaves
	level := 1
	
	for len(currentLevel) > 1 && level <= hc.hierarchicalConfig.MaxLevels {
		parentLevel := hc.groupIntoParents(currentLevel, level)
		
		// Set parent-child relationships
		for _, parent := range parentLevel {
			for _, child := range parent.Children {
				child.Parent = parent
				child.ParentID = parent.ID
			}
		}
		
		currentLevel = parentLevel
		level++
	}
	
	// Create root if multiple top-level nodes remain
	if len(currentLevel) > 1 {
		root := &HierarchicalNode{
			Chunk: &Chunk{
				Text:       "Document Root",
				TokenCount: 0,
				Metadata:   make(map[string]interface{}),
				CreatedAt:  time.Now(),
			},
			Level:    level,
			Children: currentLevel,
			ID:       "root",
			Metadata: make(map[string]interface{}),
		}
		
		for _, child := range currentLevel {
			child.Parent = root
			child.ParentID = root.ID
		}
		
		return root
	}
	
	return currentLevel[0]
}

// groupIntoParents groups nodes into parent nodes
func (hc *HierarchicalChunker) groupIntoParents(nodes []*HierarchicalNode, level int) []*HierarchicalNode {
	var parents []*HierarchicalNode
	
	for i := 0; i < len(nodes); i += hc.hierarchicalConfig.BranchingFactor {
		end := i + hc.hierarchicalConfig.BranchingFactor
		if end > len(nodes) {
			end = len(nodes)
		}
		
		children := nodes[i:end]
		
		// Skip grouping if we don't have enough children
		if len(children) < hc.hierarchicalConfig.MinChildrenForSummary {
			parents = append(parents, children...)
			continue
		}
		
		// Create parent node
		parent := &HierarchicalNode{
			Chunk: &Chunk{
				Text:       hc.combineChildrenText(children),
				TokenCount: hc.calculateCombinedTokens(children),
				Metadata:   make(map[string]interface{}),
				CreatedAt:  time.Now(),
			},
			Level:    level,
			Children: children,
			ID:       fmt.Sprintf("parent_%d_%d", level, i/hc.hierarchicalConfig.BranchingFactor),
			Metadata: make(map[string]interface{}),
		}
		
		// Propagate metadata from children if enabled
		if hc.hierarchicalConfig.EnableMetadataPropagation {
			hc.propagateMetadata(parent, children)
		}
		
		parents = append(parents, parent)
	}
	
	return parents
}

// combineChildrenText combines text from child nodes
func (hc *HierarchicalChunker) combineChildrenText(children []*HierarchicalNode) string {
	var combined strings.Builder
	
	for i, child := range children {
		if i > 0 {
			combined.WriteString("\n\n")
		}
		combined.WriteString(child.Chunk.Text)
	}
	
	return combined.String()
}

// calculateCombinedTokens calculates total tokens for combined children
func (hc *HierarchicalChunker) calculateCombinedTokens(children []*HierarchicalNode) int {
	total := 0
	for _, child := range children {
		total += child.Chunk.TokenCount
	}
	return total
}

// propagateMetadata propagates metadata from children to parent
func (hc *HierarchicalChunker) propagateMetadata(parent *HierarchicalNode, children []*HierarchicalNode) {
	// Collect section titles
	sectionTitles := make(map[string]bool)
	for _, child := range children {
		if title, ok := child.Metadata["section_title"].(string); ok {
			sectionTitles[title] = true
		}
	}
	
	// Create list of section titles
	var titles []string
	for title := range sectionTitles {
		titles = append(titles, title)
	}
	
	parent.Metadata["covered_sections"] = titles
	parent.Metadata["child_count"] = len(children)
	parent.Metadata["combined_content"] = true
}

// generateSummaries generates summaries for parent nodes
func (hc *HierarchicalChunker) generateSummaries(ctx context.Context, root *HierarchicalNode) error {
	if root == nil {
		return nil
	}
	
	// Generate summaries in post-order (children first)
	for _, child := range root.Children {
		if err := hc.generateSummaries(ctx, child); err != nil {
			return err
		}
	}
	
	// Generate summary for this node if it has children
	if len(root.Children) > 0 {
		summary, err := hc.generateNodeSummary(ctx, root)
		if err != nil {
			fmt.Printf("Warning: failed to generate summary for node %s: %v\n", root.ID, err)
			// Continue without summary
			return nil
		}
		
		root.Summary = summary
		root.SummaryType = hc.hierarchicalConfig.SummaryMode
		
		// Update chunk text to include summary
		if summary != "" {
			root.Chunk.Text = summary + "\n\n" + root.Chunk.Text
			root.Chunk.TokenCount = hc.EstimateTokens(root.Chunk.Text)
		}
	}
	
	return nil
}

// generateNodeSummary generates a summary for a node based on its children
func (hc *HierarchicalChunker) generateNodeSummary(ctx context.Context, node *HierarchicalNode) (string, error) {
	switch hc.hierarchicalConfig.SummaryMode {
	case SummaryModeNone:
		return "", nil
		
	case SummaryModeExtract:
		return hc.extractiveSummary(node), nil
		
	case SummaryModeLLM:
		if hc.llmProvider != nil {
			return hc.llmSummary(ctx, node)
		}
		// Fallback to extractive
		return hc.extractiveSummary(node), nil
		
	case SummaryModeHybrid:
		extractive := hc.extractiveSummary(node)
		if hc.llmProvider != nil {
			llmSummary, err := hc.llmSummary(ctx, node)
			if err == nil {
				return llmSummary + "\n\nKey Points: " + extractive, nil
			}
		}
		return extractive, nil
		
	default:
		return hc.extractiveSummary(node), nil
	}
}

// extractiveSummary creates an extractive summary from child nodes
func (hc *HierarchicalChunker) extractiveSummary(node *HierarchicalNode) string {
	if len(node.Children) == 0 {
		return ""
	}
	
	// Extract first sentence from each child
	var keyPoints []string
	for _, child := range node.Children {
		if len(child.Chunk.Sentences) > 0 {
			firstSentence := strings.TrimSpace(child.Chunk.Sentences[0])
			if len(firstSentence) > 10 { // Minimum length filter
				keyPoints = append(keyPoints, firstSentence)
			}
		}
	}
	
	// Combine and truncate to target length
	combined := strings.Join(keyPoints, " ")
	targetTokens := int(float64(node.Chunk.TokenCount) * hc.hierarchicalConfig.SummaryRatio)
	
	if hc.EstimateTokens(combined) > targetTokens {
		// Truncate to target length
		words := strings.Fields(combined)
		targetWords := targetTokens * 3 / 4 // Rough word-to-token ratio
		if targetWords > 0 && targetWords < len(words) {
			combined = strings.Join(words[:targetWords], " ") + "..."
		}
	}
	
	return combined
}

// llmSummary creates an LLM-generated summary
func (hc *HierarchicalChunker) llmSummary(ctx context.Context, node *HierarchicalNode) (string, error) {
	if hc.llmProvider == nil {
		return "", fmt.Errorf("LLM provider not available")
	}
	
	// Prepare content for summarization
	var childTexts []string
	for _, child := range node.Children {
		childTexts = append(childTexts, child.Chunk.Text)
	}
	
	chunksText := strings.Join(childTexts, "\n\n---\n\n")
	targetLength := int(float64(len(strings.Fields(chunksText))) * hc.hierarchicalConfig.SummaryRatio)
	
	// Create prompt
	prompt := strings.ReplaceAll(hc.hierarchicalConfig.SummaryTemplate, "{{.Chunks}}", chunksText)
	prompt = strings.ReplaceAll(prompt, "{{.TargetLength}}", fmt.Sprintf("%d", targetLength))
	
	// Generate summary
	messageList := types.MessageList{
		types.MessageDict{
			Role:    types.MessageRoleUser,
			Content: prompt,
		},
	}
	response, err := hc.llmProvider.Generate(ctx, messageList)
	
	if err != nil {
		return "", fmt.Errorf("LLM generation failed: %w", err)
	}
	
	return strings.TrimSpace(response), nil
}

// establishCrossReferences identifies and establishes cross-references between chunks
func (hc *HierarchicalChunker) establishCrossReferences(root *HierarchicalNode) {
	if root == nil {
		return
	}
	
	// Collect all nodes for reference analysis
	allNodes := hc.collectAllNodes(root)
	
	// Find cross-references between nodes
	for _, node := range allNodes {
		references := hc.findReferences(node, allNodes)
		node.References = references
		
		// Update referenced nodes
		for _, refID := range references {
			for _, refNode := range allNodes {
				if refNode.ID == refID {
					refNode.ReferencedBy = append(refNode.ReferencedBy, node.ID)
					break
				}
			}
		}
	}
}

// collectAllNodes collects all nodes in the hierarchy
func (hc *HierarchicalChunker) collectAllNodes(root *HierarchicalNode) []*HierarchicalNode {
	var nodes []*HierarchicalNode
	
	if root == nil {
		return nodes
	}
	
	nodes = append(nodes, root)
	
	for _, child := range root.Children {
		childNodes := hc.collectAllNodes(child)
		nodes = append(nodes, childNodes...)
	}
	
	return nodes
}

// findReferences finds references from one node to others
func (hc *HierarchicalChunker) findReferences(node *HierarchicalNode, allNodes []*HierarchicalNode) []string {
	var references []string
	
	// Simple keyword-based reference detection
	nodeText := strings.ToLower(node.Chunk.Text)
	
	for _, otherNode := range allNodes {
		if otherNode.ID == node.ID {
			continue
		}
		
		// Check for section title references
		if otherNode.SectionTitle != "" {
			titleLower := strings.ToLower(otherNode.SectionTitle)
			if strings.Contains(nodeText, titleLower) {
				references = append(references, otherNode.ID)
			}
		}
		
		// Check for keyword overlap (simplified)
		if hc.hasSignificantOverlap(node.Chunk.Text, otherNode.Chunk.Text) {
			references = append(references, otherNode.ID)
		}
	}
	
	return references
}

// hasSignificantOverlap checks if two texts have significant keyword overlap
func (hc *HierarchicalChunker) hasSignificantOverlap(text1, text2 string) bool {
	words1 := hc.extractKeywords(text1)
	words2 := hc.extractKeywords(text2)
	
	if len(words1) == 0 || len(words2) == 0 {
		return false
	}
	
	overlap := 0
	for word := range words1 {
		if words2[word] {
			overlap++
		}
	}
	
	overlapRatio := float64(overlap) / float64(len(words1))
	return overlapRatio > 0.3 // 30% overlap threshold
}

// extractKeywords extracts keywords from text
func (hc *HierarchicalChunker) extractKeywords(text string) map[string]bool {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "is": true,
		"are": true, "was": true, "were": true, "be": true, "been": true,
	}
	
	words := strings.Fields(strings.ToLower(text))
	keywords := make(map[string]bool)
	
	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,!?;:\"'()[]{}...")
		if len(word) > 3 && !stopWords[word] {
			keywords[word] = true
		}
	}
	
	return keywords
}

// balanceHierarchy balances the hierarchy tree
func (hc *HierarchicalChunker) balanceHierarchy(root *HierarchicalNode) *HierarchicalNode {
	if root == nil || len(root.Children) <= 1 {
		return root
	}
	
	// Recursively balance children
	for _, child := range root.Children {
		hc.balanceHierarchy(child)
	}
	
	// Balance current level if needed
	if len(root.Children) > hc.hierarchicalConfig.BranchingFactor*2 {
		root.Children = hc.redistributeChildren(root.Children)
	}
	
	return root
}

// redistributeChildren redistributes children to balance the tree
func (hc *HierarchicalChunker) redistributeChildren(children []*HierarchicalNode) []*HierarchicalNode {
	if len(children) <= hc.hierarchicalConfig.BranchingFactor {
		return children
	}
	
	// Create intermediate parent nodes
	var newChildren []*HierarchicalNode
	
	for i := 0; i < len(children); i += hc.hierarchicalConfig.BranchingFactor {
		end := i + hc.hierarchicalConfig.BranchingFactor
		if end > len(children) {
			end = len(children)
		}
		
		group := children[i:end]
		
		if len(group) == 1 {
			newChildren = append(newChildren, group[0])
		} else {
			// Create intermediate parent
			parent := &HierarchicalNode{
				Chunk: &Chunk{
					Text:       hc.combineChildrenText(group),
					TokenCount: hc.calculateCombinedTokens(group),
					Metadata:   make(map[string]interface{}),
					CreatedAt:  time.Now(),
				},
				Level:    group[0].Level + 1,
				Children: group,
				ID:       fmt.Sprintf("balanced_%d", i/hc.hierarchicalConfig.BranchingFactor),
				Metadata: make(map[string]interface{}),
			}
			
			// Update parent references
			for _, child := range group {
				child.Parent = parent
				child.ParentID = parent.ID
			}
			
			newChildren = append(newChildren, parent)
		}
	}
	
	return newChildren
}

// flattenHierarchy converts the hierarchy tree to a flat list of chunks
func (hc *HierarchicalChunker) flattenHierarchy(root *HierarchicalNode) []*Chunk {
	var chunks []*Chunk
	
	if root == nil {
		return chunks
	}
	
	// Add current node's chunk with hierarchical metadata
	if root.Chunk != nil {
		chunk := root.Chunk
		if chunk.Metadata == nil {
			chunk.Metadata = make(map[string]interface{})
		}
		
		// Add hierarchical metadata
		chunk.Metadata["hierarchy_level"] = root.Level
		chunk.Metadata["node_id"] = root.ID
		chunk.Metadata["parent_id"] = root.ParentID
		chunk.Metadata["child_count"] = len(root.Children)
		chunk.Metadata["has_summary"] = root.Summary != ""
		chunk.Metadata["summary_type"] = string(root.SummaryType)
		chunk.Metadata["section_id"] = root.SectionID
		chunk.Metadata["section_title"] = root.SectionTitle
		chunk.Metadata["references"] = root.References
		chunk.Metadata["referenced_by"] = root.ReferencedBy
		
		// Copy node metadata
		if root.Metadata != nil {
			for k, v := range root.Metadata {
				chunk.Metadata["node_"+k] = v
			}
		}
		
		chunks = append(chunks, chunk)
	}
	
	// Recursively add children
	for _, child := range root.Children {
		childChunks := hc.flattenHierarchy(child)
		chunks = append(chunks, childChunks...)
	}
	
	return chunks
}

// EstimateTokens estimates the number of tokens in text
func (hc *HierarchicalChunker) EstimateTokens(text string) int {
	if hc.tokenizer != nil {
		count, err := hc.tokenizer.CountTokens(text)
		if err == nil {
			return count
		}
	}
	return hc.tokenEstimator(text)
}

// Standard interface implementations
func (hc *HierarchicalChunker) GetConfig() *ChunkerConfig {
	config := *hc.config
	return &config
}

func (hc *HierarchicalChunker) SetConfig(config *ChunkerConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	hc.config = config
	return nil
}

func (hc *HierarchicalChunker) GetChunkSize() int {
	return hc.config.ChunkSize
}

func (hc *HierarchicalChunker) GetChunkOverlap() int {
	return hc.config.ChunkOverlap
}

func (hc *HierarchicalChunker) GetSupportedLanguages() []string {
	return []string{"en"} // Currently optimized for English
}