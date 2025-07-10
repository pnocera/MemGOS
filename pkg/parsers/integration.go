package parsers

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/memtensor/memgos/pkg/chunkers"
	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/logger"
	"github.com/memtensor/memgos/pkg/types"
)

// TextProcessingService provides a high-level interface for text processing
type TextProcessingService struct {
	config         *TextProcessingConfig
	parserFactory  *ParserFactory
	chunkerFactory *chunkers.ChunkerFactory
	embedder       interfaces.Embedder
	logger         logger.Logger
	
	// Processing statistics
	stats      *ServiceStats
	statsMutex sync.RWMutex
	
	// Cache for parsed documents (if enabled)
	documentCache map[string]*ProcessedDocument
	cacheMutex    sync.RWMutex
}

// ServiceStats tracks service-level statistics
type ServiceStats struct {
	TotalDocumentsProcessed int64         `json:"total_documents_processed"`
	TotalChunksGenerated    int64         `json:"total_chunks_generated"`
	TotalProcessingTime     time.Duration `json:"total_processing_time"`
	AverageProcessingTime   time.Duration `json:"average_processing_time"`
	ErrorCount              int64         `json:"error_count"`
	CacheHits              int64         `json:"cache_hits"`
	CacheMisses            int64         `json:"cache_misses"`
	
	// Parser usage statistics
	ParserUsage map[string]int64 `json:"parser_usage"`
	
	// Performance metrics
	MemoryUsage         int64   `json:"memory_usage"`
	ProcessingThroughput float64 `json:"processing_throughput"` // docs per second
}

// NewTextProcessingService creates a new text processing service
func NewTextProcessingService(config *TextProcessingConfig, embedder interfaces.Embedder, logger logger.Logger) (*TextProcessingService, error) {
	if config == nil {
		config = DefaultTextProcessingConfig()
	}
	
	if err := ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	
	service := &TextProcessingService{
		config:         config,
		parserFactory:  NewParserFactory(),
		chunkerFactory: chunkers.NewChunkerFactory(),
		embedder:       embedder,
		logger:         logger,
		stats: &ServiceStats{
			ParserUsage: make(map[string]int64),
		},
	}
	
	// Initialize cache if enabled
	if config.PerformanceConfig.EnableCaching {
		service.documentCache = make(map[string]*ProcessedDocument)
	}
	
	return service, nil
}

// ProcessDocument processes a document through the complete pipeline
func (tps *TextProcessingService) ProcessDocument(ctx context.Context, reader io.Reader, filename string) (*ProcessedDocument, error) {
	startTime := time.Now()
	
	// Check cache if enabled
	if tps.config.PerformanceConfig.EnableCaching && filename != "" {
		if cached := tps.getCachedDocument(filename); cached != nil {
			tps.updateStats(func(stats *ServiceStats) {
				stats.CacheHits++
			})
			return cached, nil
		}
		tps.updateStats(func(stats *ServiceStats) {
			stats.CacheMisses++
		})
	}
	
	// Create processing pipeline
	pipeline, err := NewProcessingPipeline(tps.embedder, tps.logger, tps.config.PipelineConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create processing pipeline: %w", err)
	}
	
	// Process the document
	result, err := pipeline.ProcessReader(ctx, reader, filename)
	if err != nil {
		tps.updateStats(func(stats *ServiceStats) {
			stats.ErrorCount++
		})
		return nil, err
	}
	
	// Cache the result if caching is enabled
	if tps.config.PerformanceConfig.EnableCaching && filename != "" {
		tps.cacheDocument(filename, result)
	}
	
	// Update statistics
	processingTime := time.Since(startTime)
	tps.updateServiceStats(result, processingTime)
	
	return result, nil
}

// ProcessDocuments processes multiple documents in batch
func (tps *TextProcessingService) ProcessDocuments(ctx context.Context, documents map[string]io.Reader) (map[string]*ProcessedDocument, error) {
	if len(documents) == 0 {
		return make(map[string]*ProcessedDocument), nil
	}
	
	results := make(map[string]*ProcessedDocument)
	errors := make(map[string]error)
	
	// Determine concurrency level
	maxConcurrency := tps.config.PerformanceConfig.MaxConcurrentTasks
	if maxConcurrency <= 0 {
		maxConcurrency = 4
	}
	
	// Create worker pool
	type documentJob struct {
		filename string
		reader   io.Reader
	}
	
	type documentResult struct {
		filename string
		result   *ProcessedDocument
		err      error
	}
	
	jobs := make(chan documentJob, len(documents))
	resultsChan := make(chan documentResult, len(documents))
	
	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				result, err := tps.ProcessDocument(ctx, job.reader, job.filename)
				resultsChan <- documentResult{
					filename: job.filename,
					result:   result,
					err:      err,
				}
			}
		}()
	}
	
	// Send jobs
	go func() {
		defer close(jobs)
		for filename, reader := range documents {
			jobs <- documentJob{filename: filename, reader: reader}
		}
	}()
	
	// Close results when workers are done
	go func() {
		wg.Wait()
		close(resultsChan)
	}()
	
	// Collect results
	for result := range resultsChan {
		if result.err != nil {
			errors[result.filename] = result.err
		} else {
			results[result.filename] = result.result
		}
	}
	
	// Log errors but return successful results
	if len(errors) > 0 {
		for filename, err := range errors {
			tps.logger.Error("failed to process document", err, map[string]interface{}{
				"filename": filename,
			})
		}
	}
	
	return results, nil
}

// ProcessFile processes a single file
func (tps *TextProcessingService) ProcessFile(ctx context.Context, filePath string) (*ProcessedDocument, error) {
	// Create processing pipeline
	pipeline, err := NewProcessingPipeline(tps.embedder, tps.logger, tps.config.PipelineConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create processing pipeline: %w", err)
	}
	
	result, err := pipeline.ProcessFile(ctx, filePath)
	if err != nil {
		tps.updateStats(func(stats *ServiceStats) {
			stats.ErrorCount++
		})
		return nil, err
	}
	
	// Update statistics
	tps.updateServiceStats(result, result.ProcessingTime)
	
	return result, nil
}

// ProcessFiles processes multiple files in batch
func (tps *TextProcessingService) ProcessFiles(ctx context.Context, filePaths []string) (map[string]*ProcessedDocument, error) {
	// Create processing pipeline
	pipeline, err := NewProcessingPipeline(tps.embedder, tps.logger, tps.config.PipelineConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create processing pipeline: %w", err)
	}
	
	results, err := pipeline.ProcessBatch(ctx, filePaths)
	if err != nil {
		return nil, err
	}
	
	// Update statistics for all results
	for _, result := range results {
		tps.updateServiceStats(result, result.ProcessingTime)
	}
	
	return results, nil
}

// ParseOnly parses documents without chunking or embedding
func (tps *TextProcessingService) ParseOnly(ctx context.Context, reader io.Reader, filename string) (*ParsedDocument, error) {
	// Determine parser based on filename
	parser, err := tps.parserFactory.CreateParserFromFilename(filename)
	if err != nil {
		// Fallback to content detection
		parser, err = tps.parserFactory.DetectParserByContent(ctx, reader)
		if err != nil {
			return nil, fmt.Errorf("failed to detect appropriate parser: %w", err)
		}
	}
	
	// Parse the document
	result, err := parser.Parse(ctx, reader, tps.config.ParserConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse document: %w", err)
	}
	
	// Update parser usage statistics
	tps.updateStats(func(stats *ServiceStats) {
		stats.ParserUsage[result.ParserType]++
	})
	
	return result, nil
}

// ChunkOnly chunks text without parsing
func (tps *TextProcessingService) ChunkOnly(ctx context.Context, text string) ([]*chunkers.Chunk, error) {
	// Create chunker
	chunker, err := tps.chunkerFactory.CreateChunker(tps.config.ChunkerType, tps.config.ChunkerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunker: %w", err)
	}
	
	// Chunk the text
	chunks, err := chunker.Chunk(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to chunk text: %w", err)
	}
	
	return chunks, nil
}

// GetSupportedFormats returns all supported file formats
func (tps *TextProcessingService) GetSupportedFormats() []string {
	return tps.parserFactory.GetSupportedExtensions()
}

// GetParserInfo returns information about available parsers
func (tps *TextProcessingService) GetParserInfo() ([]ParserInfo, error) {
	var parsers []ParserInfo
	
	for _, parserType := range tps.parserFactory.GetRegisteredParsers() {
		info, err := tps.parserFactory.GetParserInfo(parserType)
		if err != nil {
			return nil, err
		}
		parsers = append(parsers, *info)
	}
	
	return parsers, nil
}

// DetectFileType detects the file type of content
func (tps *TextProcessingService) DetectFileType(ctx context.Context, reader io.Reader, filename string) (string, error) {
	parser, err := tps.parserFactory.CreateParserFromFilename(filename)
	if err != nil {
		// Try content detection
		parser, err = tps.parserFactory.DetectParserByContent(ctx, reader)
		if err != nil {
			return "", err
		}
	}
	
	return parser.GetParserType(), nil
}

// ValidateDocument validates a document before processing
func (tps *TextProcessingService) ValidateDocument(ctx context.Context, reader io.Reader, filename string) error {
	// Check file extension if filtering is enabled
	if tps.config.ContentFiltering != nil {
		ext := strings.ToLower(filepath.Ext(filename))
		
		// Check blocked file types
		for _, blocked := range tps.config.ContentFiltering.BlockedFileTypes {
			if ext == blocked {
				return fmt.Errorf("file type %s is blocked", ext)
			}
		}
		
		// Check allowed file types (if specified)
		if len(tps.config.ContentFiltering.AllowedFileTypes) > 0 {
			allowed := false
			for _, allowedExt := range tps.config.ContentFiltering.AllowedFileTypes {
				if ext == allowedExt {
					allowed = true
					break
				}
			}
			if !allowed {
				return fmt.Errorf("file type %s is not allowed", ext)
			}
		}
	}
	
	// Try to get a parser for validation
	parser, err := tps.parserFactory.CreateParserFromFilename(filename)
	if err != nil {
		return fmt.Errorf("no parser available for file type: %w", err)
	}
	
	// Validate the content
	return parser.ValidateInput(ctx, reader)
}

// GetStats returns current service statistics
func (tps *TextProcessingService) GetStats() *ServiceStats {
	tps.statsMutex.RLock()
	defer tps.statsMutex.RUnlock()
	
	// Return a copy
	statsCopy := *tps.stats
	statsCopy.ParserUsage = make(map[string]int64)
	for k, v := range tps.stats.ParserUsage {
		statsCopy.ParserUsage[k] = v
	}
	
	return &statsCopy
}

// ResetStats resets service statistics
func (tps *TextProcessingService) ResetStats() {
	tps.statsMutex.Lock()
	defer tps.statsMutex.Unlock()
	
	tps.stats = &ServiceStats{
		ParserUsage: make(map[string]int64),
	}
}

// UpdateConfig updates the service configuration
func (tps *TextProcessingService) UpdateConfig(config *TextProcessingConfig) error {
	if err := ValidateConfig(config); err != nil {
		return err
	}
	
	tps.config = config
	
	// Clear cache if caching is disabled
	if !config.PerformanceConfig.EnableCaching {
		tps.cacheMutex.Lock()
		tps.documentCache = nil
		tps.cacheMutex.Unlock()
	} else if tps.documentCache == nil {
		tps.cacheMutex.Lock()
		tps.documentCache = make(map[string]*ProcessedDocument)
		tps.cacheMutex.Unlock()
	}
	
	return nil
}

// getCachedDocument retrieves a cached document
func (tps *TextProcessingService) getCachedDocument(filename string) *ProcessedDocument {
	if tps.documentCache == nil {
		return nil
	}
	
	tps.cacheMutex.RLock()
	defer tps.cacheMutex.RUnlock()
	
	doc, exists := tps.documentCache[filename]
	if !exists {
		return nil
	}
	
	// Check if cache entry is still valid
	if tps.config.PerformanceConfig.CacheTTL > 0 {
		if time.Since(doc.ProcessedAt) > tps.config.PerformanceConfig.CacheTTL {
			// Cache entry expired
			go func() {
				tps.cacheMutex.Lock()
				delete(tps.documentCache, filename)
				tps.cacheMutex.Unlock()
			}()
			return nil
		}
	}
	
	return doc
}

// cacheDocument stores a document in cache
func (tps *TextProcessingService) cacheDocument(filename string, doc *ProcessedDocument) {
	if tps.documentCache == nil {
		return
	}
	
	tps.cacheMutex.Lock()
	defer tps.cacheMutex.Unlock()
	
	// Check cache size limit
	if tps.config.PerformanceConfig.CacheSize > 0 && len(tps.documentCache) >= tps.config.PerformanceConfig.CacheSize {
		// Remove oldest entry (simplified LRU)
		oldestKey := ""
		oldestTime := time.Now()
		for key, cached := range tps.documentCache {
			if cached.ProcessedAt.Before(oldestTime) {
				oldestTime = cached.ProcessedAt
				oldestKey = key
			}
		}
		if oldestKey != "" {
			delete(tps.documentCache, oldestKey)
		}
	}
	
	tps.documentCache[filename] = doc
}

// updateStats safely updates service statistics
func (tps *TextProcessingService) updateStats(updateFunc func(*ServiceStats)) {
	tps.statsMutex.Lock()
	defer tps.statsMutex.Unlock()
	updateFunc(tps.stats)
}

// updateServiceStats updates statistics based on processing results
func (tps *TextProcessingService) updateServiceStats(result *ProcessedDocument, processingTime time.Duration) {
	tps.updateStats(func(stats *ServiceStats) {
		stats.TotalDocumentsProcessed++
		stats.TotalChunksGenerated += int64(result.TotalChunks)
		stats.TotalProcessingTime += processingTime
		
		if stats.TotalDocumentsProcessed > 0 {
			stats.AverageProcessingTime = stats.TotalProcessingTime / time.Duration(stats.TotalDocumentsProcessed)
			stats.ProcessingThroughput = float64(stats.TotalDocumentsProcessed) / stats.TotalProcessingTime.Seconds()
		}
		
		stats.ParserUsage[result.ParserUsed]++
	})
}

// ClearCache clears the document cache
func (tps *TextProcessingService) ClearCache() {
	tps.cacheMutex.Lock()
	defer tps.cacheMutex.Unlock()
	
	if tps.documentCache != nil {
		tps.documentCache = make(map[string]*ProcessedDocument)
	}
}

// GetCacheStats returns cache statistics
func (tps *TextProcessingService) GetCacheStats() map[string]interface{} {
	tps.cacheMutex.RLock()
	defer tps.cacheMutex.RUnlock()
	
	stats := make(map[string]interface{})
	
	if tps.documentCache != nil {
		stats["cache_enabled"] = true
		stats["cache_size"] = len(tps.documentCache)
		stats["cache_limit"] = tps.config.PerformanceConfig.CacheSize
		
		// Calculate cache age distribution
		now := time.Now()
		ageDistribution := map[string]int{
			"under_1h":  0,
			"1h_to_6h":  0,
			"6h_to_24h": 0,
			"over_24h":  0,
		}
		
		for _, doc := range tps.documentCache {
			age := now.Sub(doc.ProcessedAt)
			if age < time.Hour {
				ageDistribution["under_1h"]++
			} else if age < 6*time.Hour {
				ageDistribution["1h_to_6h"]++
			} else if age < 24*time.Hour {
				ageDistribution["6h_to_24h"]++
			} else {
				ageDistribution["over_24h"]++
			}
		}
		
		stats["age_distribution"] = ageDistribution
	} else {
		stats["cache_enabled"] = false
	}
	
	return stats
}