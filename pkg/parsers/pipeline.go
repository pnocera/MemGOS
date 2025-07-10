package parsers

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/memtensor/memgos/pkg/chunkers"
	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/logger"
	"github.com/memtensor/memgos/pkg/types"
)

// ProcessingPipeline handles the complete document processing workflow
type ProcessingPipeline struct {
	parserFactory  *ParserFactory
	chunkerFactory *chunkers.ChunkerFactory
	embedder       interfaces.Embedder
	logger         logger.Logger
	config         *PipelineConfig
	
	// Processing statistics
	stats      *PipelineStats
	statsMutex sync.RWMutex
}

// PipelineConfig configures the document processing pipeline
type PipelineConfig struct {
	// Parser configuration
	ParserConfig *ParserConfig `json:"parser_config"`
	
	// Chunker configuration
	ChunkerType   chunkers.ChunkerType   `json:"chunker_type"`
	ChunkerConfig *chunkers.ChunkerConfig `json:"chunker_config"`
	
	// Processing options
	MaxConcurrency     int           `json:"max_concurrency"`
	ProcessingTimeout  time.Duration `json:"processing_timeout"`
	EnableEmbedding    bool          `json:"enable_embedding"`
	EnablePreprocessing bool         `json:"enable_preprocessing"`
	
	// Content filtering
	MinContentLength int      `json:"min_content_length"`
	MaxContentLength int      `json:"max_content_length"`
	ExcludePatterns  []string `json:"exclude_patterns"`
	
	// Output options
	PreserveParsedDocument bool `json:"preserve_parsed_document"`
	PreserveChunkMetadata  bool `json:"preserve_chunk_metadata"`
	
	// Error handling
	ContinueOnError    bool `json:"continue_on_error"`
	MaxRetries         int  `json:"max_retries"`
	RetryDelay         time.Duration `json:"retry_delay"`
}

// DefaultPipelineConfig returns a default pipeline configuration
func DefaultPipelineConfig() *PipelineConfig {
	return &PipelineConfig{
		ParserConfig:           DefaultParserConfig(),
		ChunkerType:           chunkers.ChunkerTypeSentence,
		ChunkerConfig:         chunkers.DefaultChunkerConfig(),
		MaxConcurrency:        4,
		ProcessingTimeout:     300 * time.Second, // 5 minutes
		EnableEmbedding:       true,
		EnablePreprocessing:   true,
		MinContentLength:      10,
		MaxContentLength:      1000000, // 1MB of text
		ExcludePatterns:       []string{},
		PreserveParsedDocument: false,
		PreserveChunkMetadata: true,
		ContinueOnError:       true,
		MaxRetries:            3,
		RetryDelay:            time.Second,
	}
}

// PipelineStats tracks processing statistics
type PipelineStats struct {
	TotalDocuments     int64         `json:"total_documents"`
	SuccessfulDocuments int64        `json:"successful_documents"`
	FailedDocuments    int64         `json:"failed_documents"`
	TotalChunks        int64         `json:"total_chunks"`
	TotalProcessingTime time.Duration `json:"total_processing_time"`
	AverageChunksPerDoc float64      `json:"average_chunks_per_doc"`
	BytesProcessed     int64         `json:"bytes_processed"`
	ErrorsByType       map[string]int64 `json:"errors_by_type"`
	ParserUsage        map[string]int64 `json:"parser_usage"`
}

// NewProcessingPipeline creates a new document processing pipeline
func NewProcessingPipeline(embedder interfaces.Embedder, logger logger.Logger, config *PipelineConfig) (*ProcessingPipeline, error) {
	if config == nil {
		config = DefaultPipelineConfig()
	}
	
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	
	// Create factories
	parserFactory := NewParserFactory()
	chunkerFactory := chunkers.NewChunkerFactory()
	
	// Validate chunker configuration
	if err := chunkerFactory.ValidateConfig(config.ChunkerConfig); err != nil {
		return nil, fmt.Errorf("invalid chunker config: %w", err)
	}
	
	// Validate parser configuration
	if err := parserFactory.ValidateParserConfiguration(config.ParserConfig); err != nil {
		return nil, fmt.Errorf("invalid parser config: %w", err)
	}
	
	pipeline := &ProcessingPipeline{
		parserFactory:  parserFactory,
		chunkerFactory: chunkerFactory,
		embedder:       embedder,
		logger:         logger,
		config:         config,
		stats: &PipelineStats{
			ErrorsByType: make(map[string]int64),
			ParserUsage:  make(map[string]int64),
		},
	}
	
	return pipeline, nil
}

// ProcessedDocument represents the result of processing a document through the pipeline
type ProcessedDocument struct {
	// Original document information
	OriginalPath   string `json:"original_path"`
	OriginalSize   int64  `json:"original_size"`
	
	// Parsed document (optional, based on config)
	ParsedDocument *ParsedDocument `json:"parsed_document,omitempty"`
	
	// Processing results
	Chunks           []*ProcessedChunk `json:"chunks"`
	TotalChunks      int               `json:"total_chunks"`
	ProcessingTime   time.Duration     `json:"processing_time"`
	ParserUsed       string            `json:"parser_used"`
	ChunkerUsed      string            `json:"chunker_used"`
	
	// Error information
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	
	// Metadata
	ProcessedAt time.Time `json:"processed_at"`
	PipelineVersion string `json:"pipeline_version"`
}

// ProcessedChunk represents a processed text chunk with embeddings
type ProcessedChunk struct {
	// Chunk content and metadata
	*chunkers.Chunk
	
	// Generated embedding
	Embedding types.EmbeddingVector `json:"embedding,omitempty"`
	
	// Processing metadata
	ProcessingTime time.Duration `json:"processing_time"`
	EmbeddingTime  time.Duration `json:"embedding_time"`
	
	// Content analysis
	Language     string   `json:"language,omitempty"`
	Keywords     []string `json:"keywords,omitempty"`
	ContentType  string   `json:"content_type,omitempty"`
	
	// Quality metrics
	ReadabilityScore float64 `json:"readability_score,omitempty"`
	ContentDensity   float64 `json:"content_density,omitempty"`
}

// ProcessFile processes a single file through the complete pipeline
func (pp *ProcessingPipeline) ProcessFile(ctx context.Context, filePath string) (*ProcessedDocument, error) {
	startTime := time.Now()
	
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}
	
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()
	
	// Process with metadata
	result, err := pp.ProcessReader(ctx, file, filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	
	// Add file-specific information
	result.OriginalPath = filePath
	result.OriginalSize = fileInfo.Size()
	
	// Update statistics
	pp.updateStats(result, time.Since(startTime))
	
	return result, nil
}

// ProcessReader processes content from an io.Reader
func (pp *ProcessingPipeline) ProcessReader(ctx context.Context, reader io.Reader, filename string) (*ProcessedDocument, error) {
	startTime := time.Now()
	
	// Create processing context with timeout
	processingCtx, cancel := context.WithTimeout(ctx, pp.config.ProcessingTimeout)
	defer cancel()
	
	// Parse the document
	parsedDoc, err := pp.parseDocument(processingCtx, reader, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to parse document: %w", err)
	}
	
	// Preprocess content if enabled
	content := parsedDoc.Content
	if pp.config.EnablePreprocessing {
		content = pp.preprocessContent(content)
	}
	
	// Validate content length
	if err := pp.validateContent(content); err != nil {
		return nil, err
	}
	
	// Chunk the content
	chunks, err := pp.chunkContent(processingCtx, content, parsedDoc.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to chunk content: %w", err)
	}
	
	// Process chunks (embeddings, analysis, etc.)
	processedChunks, err := pp.processChunks(processingCtx, chunks)
	if err != nil {
		return nil, fmt.Errorf("failed to process chunks: %w", err)
	}
	
	// Create result document
	result := &ProcessedDocument{
		Chunks:          processedChunks,
		TotalChunks:     len(processedChunks),
		ProcessingTime:  time.Since(startTime),
		ParserUsed:      parsedDoc.ParserType,
		ChunkerUsed:     string(pp.config.ChunkerType),
		ProcessedAt:     time.Now(),
		PipelineVersion: "1.0.0", // Version of the pipeline
	}
	
	// Include parsed document if requested
	if pp.config.PreserveParsedDocument {
		result.ParsedDocument = parsedDoc
	}
	
	return result, nil
}

// ProcessBatch processes multiple documents in parallel
func (pp *ProcessingPipeline) ProcessBatch(ctx context.Context, files []string) (map[string]*ProcessedDocument, error) {
	if len(files) == 0 {
		return make(map[string]*ProcessedDocument), nil
	}
	
	// Create worker pool
	maxWorkers := pp.config.MaxConcurrency
	if maxWorkers <= 0 {
		maxWorkers = 4
	}
	
	type fileJob struct {
		path   string
		result *ProcessedDocument
		err    error
	}
	
	// Create channels
	jobs := make(chan string, len(files))
	results := make(chan fileJob, len(files))
	
	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range jobs {
				result, err := pp.ProcessFile(ctx, filePath)
				results <- fileJob{
					path:   filePath,
					result: result,
					err:    err,
				}
			}
		}()
	}
	
	// Send jobs
	go func() {
		defer close(jobs)
		for _, file := range files {
			jobs <- file
		}
	}()
	
	// Close results when workers are done
	go func() {
		wg.Wait()
		close(results)
	}()
	
	// Collect results
	processedDocs := make(map[string]*ProcessedDocument)
	var errors []string
	
	for job := range results {
		if job.err != nil {
			errorMsg := fmt.Sprintf("failed to process %s: %v", job.path, job.err)
			errors = append(errors, errorMsg)
			
			if !pp.config.ContinueOnError {
				return nil, fmt.Errorf("batch processing failed: %s", errorMsg)
			}
			
			pp.logger.Error("batch processing error", job.err, map[string]interface{}{
				"file": job.path,
			})
		} else {
			processedDocs[job.path] = job.result
		}
	}
	
	if len(errors) > 0 && !pp.config.ContinueOnError {
		return nil, fmt.Errorf("batch processing failed with %d errors", len(errors))
	}
	
	return processedDocs, nil
}

// parseDocument parses a document using the appropriate parser
func (pp *ProcessingPipeline) parseDocument(ctx context.Context, reader io.Reader, filename string) (*ParsedDocument, error) {
	// Parse with best parser
	parsedDoc, err := pp.parserFactory.ParseWithBestParser(ctx, reader, filename, pp.config.ParserConfig)
	if err != nil {
		return nil, err
	}
	
	// Update parser usage statistics
	pp.statsMutex.Lock()
	pp.stats.ParserUsage[parsedDoc.ParserType]++
	pp.statsMutex.Unlock()
	
	return parsedDoc, nil
}

// preprocessContent preprocesses the content before chunking
func (pp *ProcessingPipeline) preprocessContent(content string) string {
	// Normalize whitespace
	content = strings.Join(strings.Fields(content), " ")
	
	// Remove excessive newlines
	content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	
	// Trim whitespace
	content = strings.TrimSpace(content)
	
	return content
}

// validateContent validates that content meets requirements
func (pp *ProcessingPipeline) validateContent(content string) error {
	contentLength := len(content)
	
	if contentLength < pp.config.MinContentLength {
		return fmt.Errorf("content length %d is below minimum %d", contentLength, pp.config.MinContentLength)
	}
	
	if contentLength > pp.config.MaxContentLength {
		return fmt.Errorf("content length %d exceeds maximum %d", contentLength, pp.config.MaxContentLength)
	}
	
	// Check exclude patterns
	for _, pattern := range pp.config.ExcludePatterns {
		if strings.Contains(strings.ToLower(content), strings.ToLower(pattern)) {
			return fmt.Errorf("content matches exclude pattern: %s", pattern)
		}
	}
	
	return nil
}

// chunkContent chunks the content using the configured chunker
func (pp *ProcessingPipeline) chunkContent(ctx context.Context, content string, metadata *DocumentMetadata) ([]*chunkers.Chunk, error) {
	// Create chunker
	chunker, err := pp.chunkerFactory.CreateChunker(pp.config.ChunkerType, pp.config.ChunkerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunker: %w", err)
	}
	
	// Prepare chunk metadata
	chunkMetadata := make(map[string]interface{})
	if pp.config.PreserveChunkMetadata && metadata != nil {
		chunkMetadata["document_title"] = metadata.Title
		chunkMetadata["document_author"] = metadata.Author
		chunkMetadata["document_language"] = metadata.Language
		chunkMetadata["document_mime_type"] = metadata.MimeType
	}
	
	// Chunk content
	chunks, err := chunker.ChunkWithMetadata(ctx, content, chunkMetadata)
	if err != nil {
		return nil, err
	}
	
	return chunks, nil
}

// processChunks processes chunks to add embeddings and analysis
func (pp *ProcessingPipeline) processChunks(ctx context.Context, chunks []*chunkers.Chunk) ([]*ProcessedChunk, error) {
	processedChunks := make([]*ProcessedChunk, len(chunks))
	
	for i, chunk := range chunks {
		startTime := time.Now()
		
		processedChunk := &ProcessedChunk{
			Chunk: chunk,
		}
		
		// Generate embedding if enabled and embedder is available
		if pp.config.EnableEmbedding && pp.embedder != nil {
			embeddingStart := time.Now()
			embedding, err := pp.embedder.Embed(ctx, chunk.Text)
			if err != nil {
				if !pp.config.ContinueOnError {
					return nil, fmt.Errorf("failed to generate embedding for chunk %d: %w", i, err)
				}
				pp.logger.Warn("failed to generate embedding", map[string]interface{}{
					"chunk_index": i,
					"error":       err.Error(),
				})
			} else {
				processedChunk.Embedding = embedding
				processedChunk.EmbeddingTime = time.Since(embeddingStart)
			}
		}
		
		// Analyze content
		pp.analyzeChunkContent(processedChunk)
		
		processedChunk.ProcessingTime = time.Since(startTime)
		processedChunks[i] = processedChunk
	}
	
	return processedChunks, nil
}

// analyzeChunkContent performs content analysis on a chunk
func (pp *ProcessingPipeline) analyzeChunkContent(chunk *ProcessedChunk) {
	// Simple content analysis
	text := chunk.Text
	
	// Estimate language (simplified)
	chunk.Language = pp.estimateLanguage(text)
	
	// Extract simple keywords (top frequent words)
	chunk.Keywords = pp.extractKeywords(text, 5)
	
	// Determine content type
	chunk.ContentType = pp.determineContentType(text)
	
	// Calculate content density (text to total characters ratio)
	if len(text) > 0 {
		nonSpaceChars := len(strings.ReplaceAll(text, " ", ""))
		chunk.ContentDensity = float64(nonSpaceChars) / float64(len(text))
	}
	
	// Simple readability score (based on sentence and word length)
	chunk.ReadabilityScore = pp.calculateReadabilityScore(text)
}

// estimateLanguage provides simple language estimation
func (pp *ProcessingPipeline) estimateLanguage(text string) string {
	// Simple heuristic - in practice, use a proper language detection library
	commonEnglishWords := []string{"the", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with", "by"}
	englishCount := 0
	words := strings.Fields(strings.ToLower(text))
	
	for _, word := range words {
		for _, englishWord := range commonEnglishWords {
			if word == englishWord {
				englishCount++
				break
			}
		}
	}
	
	if len(words) > 0 && float64(englishCount)/float64(len(words)) > 0.05 {
		return "en"
	}
	
	return "unknown"
}

// extractKeywords extracts simple keywords from text
func (pp *ProcessingPipeline) extractKeywords(text string, maxKeywords int) []string {
	// Simple keyword extraction based on word frequency
	words := strings.Fields(strings.ToLower(text))
	wordCount := make(map[string]int)
	
	// Count words (excluding common stop words)
	stopWords := map[string]bool{
		"the": true, "and": true, "or": true, "but": true, "in": true,
		"on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "a": true, "an": true, "is": true,
		"are": true, "was": true, "were": true, "be": true, "been": true,
		"have": true, "has": true, "had": true, "will": true, "would": true,
		"could": true, "should": true, "may": true, "might": true, "can": true,
	}
	
	for _, word := range words {
		word = strings.Trim(word, ".,!?;:\"'()[]{}...")
		if len(word) > 3 && !stopWords[word] {
			wordCount[word]++
		}
	}
	
	// Sort by frequency and return top keywords
	type wordFreq struct {
		word  string
		count int
	}
	
	var sortedWords []wordFreq
	for word, count := range wordCount {
		sortedWords = append(sortedWords, wordFreq{word, count})
	}
	
	// Simple sorting (bubble sort for small arrays)
	for i := 0; i < len(sortedWords)-1; i++ {
		for j := 0; j < len(sortedWords)-i-1; j++ {
			if sortedWords[j].count < sortedWords[j+1].count {
				sortedWords[j], sortedWords[j+1] = sortedWords[j+1], sortedWords[j]
			}
		}
	}
	
	// Extract top keywords
	var keywords []string
	for i := 0; i < len(sortedWords) && i < maxKeywords; i++ {
		keywords = append(keywords, sortedWords[i].word)
	}
	
	return keywords
}

// determineContentType determines the type of content
func (pp *ProcessingPipeline) determineContentType(text string) string {
	text = strings.ToLower(text)
	
	// Simple content type detection
	if strings.Contains(text, "function") || strings.Contains(text, "class") || 
		strings.Contains(text, "import") || strings.Contains(text, "def ") {
		return "code"
	}
	
	if strings.Contains(text, "http://") || strings.Contains(text, "https://") {
		return "web_content"
	}
	
	if strings.Contains(text, "@") && strings.Contains(text, ".") {
		return "contact_info"
	}
	
	return "text"
}

// calculateReadabilityScore calculates a simple readability score
func (pp *ProcessingPipeline) calculateReadabilityScore(text string) float64 {
	if len(text) == 0 {
		return 0
	}
	
	sentences := strings.Split(text, ".")
	words := strings.Fields(text)
	
	if len(sentences) == 0 || len(words) == 0 {
		return 0
	}
	
	avgWordsPerSentence := float64(len(words)) / float64(len(sentences))
	avgCharsPerWord := float64(len(text)) / float64(len(words))
	
	// Simple readability formula (lower is better)
	// Based on average sentence length and word complexity
	score := 100 - (avgWordsPerSentence * 1.015) - (avgCharsPerWord * 84.6)
	
	// Normalize to 0-100 scale
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	
	return score
}

// updateStats updates pipeline statistics
func (pp *ProcessingPipeline) updateStats(result *ProcessedDocument, processingTime time.Duration) {
	pp.statsMutex.Lock()
	defer pp.statsMutex.Unlock()
	
	pp.stats.TotalDocuments++
	if len(result.Errors) == 0 {
		pp.stats.SuccessfulDocuments++
	} else {
		pp.stats.FailedDocuments++
		for _, err := range result.Errors {
			pp.stats.ErrorsByType[err]++
		}
	}
	
	pp.stats.TotalChunks += int64(result.TotalChunks)
	pp.stats.TotalProcessingTime += processingTime
	pp.stats.BytesProcessed += result.OriginalSize
	
	// Update average chunks per document
	if pp.stats.TotalDocuments > 0 {
		pp.stats.AverageChunksPerDoc = float64(pp.stats.TotalChunks) / float64(pp.stats.TotalDocuments)
	}
}

// GetStats returns current pipeline statistics
func (pp *ProcessingPipeline) GetStats() *PipelineStats {
	pp.statsMutex.RLock()
	defer pp.statsMutex.RUnlock()
	
	// Return a copy of the stats
	statsCopy := *pp.stats
	statsCopy.ErrorsByType = make(map[string]int64)
	statsCopy.ParserUsage = make(map[string]int64)
	
	for k, v := range pp.stats.ErrorsByType {
		statsCopy.ErrorsByType[k] = v
	}
	
	for k, v := range pp.stats.ParserUsage {
		statsCopy.ParserUsage[k] = v
	}
	
	return &statsCopy
}

// ResetStats resets pipeline statistics
func (pp *ProcessingPipeline) ResetStats() {
	pp.statsMutex.Lock()
	defer pp.statsMutex.Unlock()
	
	pp.stats = &PipelineStats{
		ErrorsByType: make(map[string]int64),
		ParserUsage:  make(map[string]int64),
	}
}

// GetConfig returns the current pipeline configuration
func (pp *ProcessingPipeline) GetConfig() *PipelineConfig {
	// Return a copy to prevent external modification
	configCopy := *pp.config
	return &configCopy
}

// UpdateConfig updates the pipeline configuration
func (pp *ProcessingPipeline) UpdateConfig(config *PipelineConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	
	// Validate configurations
	if err := pp.parserFactory.ValidateParserConfiguration(config.ParserConfig); err != nil {
		return fmt.Errorf("invalid parser config: %w", err)
	}
	
	if err := pp.chunkerFactory.ValidateConfig(config.ChunkerConfig); err != nil {
		return fmt.Errorf("invalid chunker config: %w", err)
	}
	
	pp.config = config
	return nil
}