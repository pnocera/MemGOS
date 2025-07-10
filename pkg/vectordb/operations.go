package vectordb

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// VectorOperations provides high-level vector operations across multiple databases
type VectorOperations struct {
	registry    *VectorDBRegistry
	batchSize   int
	maxWorkers  int
	timeout     time.Duration
}

// NewVectorOperations creates a new vector operations instance
func NewVectorOperations(registry *VectorDBRegistry) *VectorOperations {
	return &VectorOperations{
		registry:   registry,
		batchSize:  100,
		maxWorkers: 10,
		timeout:    30 * time.Second,
	}
}

// SetBatchSize sets the default batch size for operations
func (vo *VectorOperations) SetBatchSize(size int) {
	if size > 0 {
		vo.batchSize = size
	}
}

// SetMaxWorkers sets the maximum number of worker goroutines
func (vo *VectorOperations) SetMaxWorkers(workers int) {
	if workers > 0 {
		vo.maxWorkers = workers
	}
}

// SetTimeout sets the operation timeout
func (vo *VectorOperations) SetTimeout(timeout time.Duration) {
	vo.timeout = timeout
}

// ParallelSearchResult represents results from parallel search across multiple databases
type ParallelSearchResult struct {
	DatabaseName string           `json:"database_name"`
	Results      VectorDBItemList `json:"results"`
	Error        error            `json:"error,omitempty"`
	Duration     time.Duration    `json:"duration"`
}

// ParallelSearch performs parallel search across multiple vector databases
func (vo *VectorOperations) ParallelSearch(ctx context.Context, databases []string, vector []float32, limit int, filters map[string]interface{}) ([]*ParallelSearchResult, error) {
	if len(databases) == 0 {
		return nil, fmt.Errorf("no databases specified")
	}

	ctx, cancel := context.WithTimeout(ctx, vo.timeout)
	defer cancel()

	results := make([]*ParallelSearchResult, len(databases))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, vo.maxWorkers)

	for i, dbName := range databases {
		wg.Add(1)
		go func(index int, name string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			start := time.Now()
			result := &ParallelSearchResult{
				DatabaseName: name,
				Duration:     0,
			}

			db, err := vo.registry.Get(ctx, name)
			if err != nil {
				result.Error = fmt.Errorf("failed to get database %s: %w", name, err)
				results[index] = result
				return
			}

			// Assume single collection for now - this could be made configurable
			collections, err := db.ListCollections(ctx)
			if err != nil {
				result.Error = fmt.Errorf("failed to list collections: %w", err)
				results[index] = result
				return
			}

			if len(collections) == 0 {
				result.Error = fmt.Errorf("no collections found in database %s", name)
				results[index] = result
				return
			}

			// Search in the first collection
			items, err := db.Search(ctx, collections[0], vector, limit, filters)
			if err != nil {
				result.Error = fmt.Errorf("search failed: %w", err)
				results[index] = result
				return
			}

			result.Results = items
			result.Duration = time.Since(start)
			results[index] = result
		}(i, dbName)
	}

	wg.Wait()
	return results, nil
}

// MergeSearchResults merges and ranks results from multiple databases
func (vo *VectorOperations) MergeSearchResults(results []*ParallelSearchResult, limit int) VectorDBItemList {
	var allItems VectorDBItemList

	for _, result := range results {
		if result.Error == nil {
			allItems = append(allItems, result.Results...)
		}
	}

	// Sort by score (descending)
	sort.Slice(allItems, func(i, j int) bool {
		scoreI := float32(0)
		scoreJ := float32(0)
		
		if allItems[i].Score != nil {
			scoreI = *allItems[i].Score
		}
		if allItems[j].Score != nil {
			scoreJ = *allItems[j].Score
		}
		
		return scoreI > scoreJ
	})

	// Apply limit
	if limit > 0 && len(allItems) > limit {
		allItems = allItems[:limit]
	}

	return allItems
}

// BulkOperation represents a bulk operation across multiple databases
type BulkOperation struct {
	Database   string           `json:"database"`
	Collection string           `json:"collection"`
	Operation  VectorOperation  `json:"operation"`
	Items      VectorDBItemList `json:"items,omitempty"`
	IDs        []string         `json:"ids,omitempty"`
}

// BulkOperationResult represents the result of a bulk operation
type BulkOperationResult struct {
	Database     string        `json:"database"`
	Collection   string        `json:"collection"`
	Operation    VectorOperation `json:"operation"`
	Success      bool          `json:"success"`
	ProcessedCount int         `json:"processed_count"`
	Error        error         `json:"error,omitempty"`
	Duration     time.Duration `json:"duration"`
}

// ExecuteBulkOperations executes multiple operations in parallel
func (vo *VectorOperations) ExecuteBulkOperations(ctx context.Context, operations []*BulkOperation) ([]*BulkOperationResult, error) {
	if len(operations) == 0 {
		return nil, fmt.Errorf("no operations specified")
	}

	ctx, cancel := context.WithTimeout(ctx, vo.timeout)
	defer cancel()

	results := make([]*BulkOperationResult, len(operations))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, vo.maxWorkers)

	for i, op := range operations {
		wg.Add(1)
		go func(index int, operation *BulkOperation) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			start := time.Now()
			result := &BulkOperationResult{
				Database:   operation.Database,
				Collection: operation.Collection,
				Operation:  operation.Operation,
				Success:    false,
				Duration:   0,
			}

			db, err := vo.registry.Get(ctx, operation.Database)
			if err != nil {
				result.Error = fmt.Errorf("failed to get database: %w", err)
				results[index] = result
				return
			}

			switch operation.Operation {
			case OperationAdd:
				err = db.BatchAdd(ctx, operation.Collection, operation.Items, vo.batchSize)
				result.ProcessedCount = len(operation.Items)
			case OperationUpdate:
				err = db.BatchUpdate(ctx, operation.Collection, operation.Items, vo.batchSize)
				result.ProcessedCount = len(operation.Items)
			case OperationUpsert:
				err = db.Upsert(ctx, operation.Collection, operation.Items)
				result.ProcessedCount = len(operation.Items)
			case OperationDelete:
				err = db.BatchDelete(ctx, operation.Collection, operation.IDs, vo.batchSize)
				result.ProcessedCount = len(operation.IDs)
			default:
				err = fmt.Errorf("unsupported operation: %s", operation.Operation)
			}

			if err != nil {
				result.Error = err
			} else {
				result.Success = true
			}

			result.Duration = time.Since(start)
			results[index] = result
		}(i, op)
	}

	wg.Wait()
	return results, nil
}

// SimilarityCalculator provides various similarity calculation methods
type SimilarityCalculator struct{}

// NewSimilarityCalculator creates a new similarity calculator
func NewSimilarityCalculator() *SimilarityCalculator {
	return &SimilarityCalculator{}
}

// CosineSimilarity calculates cosine similarity between two vectors
func (sc *SimilarityCalculator) CosineSimilarity(a, b []float32) (float32, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors must have the same length")
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0, nil
	}

	return float32(dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))), nil
}

// EuclideanDistance calculates Euclidean distance between two vectors
func (sc *SimilarityCalculator) EuclideanDistance(a, b []float32) (float32, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors must have the same length")
	}

	var sum float64
	for i := range a {
		diff := float64(a[i]) - float64(b[i])
		sum += diff * diff
	}

	return float32(math.Sqrt(sum)), nil
}

// DotProduct calculates dot product between two vectors
func (sc *SimilarityCalculator) DotProduct(a, b []float32) (float32, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors must have the same length")
	}

	var product float64
	for i := range a {
		product += float64(a[i]) * float64(b[i])
	}

	return float32(product), nil
}

// VectorNorm calculates the L2 norm of a vector
func (sc *SimilarityCalculator) VectorNorm(vector []float32) float32 {
	var sum float64
	for _, v := range vector {
		sum += float64(v) * float64(v)
	}
	return float32(math.Sqrt(sum))
}

// NormalizeVector normalizes a vector to unit length
func (sc *SimilarityCalculator) NormalizeVector(vector []float32) []float32 {
	norm := sc.VectorNorm(vector)
	if norm == 0 {
		return vector
	}

	normalized := make([]float32, len(vector))
	for i, v := range vector {
		normalized[i] = v / norm
	}
	return normalized
}

// VectorAnalyzer provides vector analysis and statistics
type VectorAnalyzer struct {
	calculator *SimilarityCalculator
}

// NewVectorAnalyzer creates a new vector analyzer
func NewVectorAnalyzer() *VectorAnalyzer {
	return &VectorAnalyzer{
		calculator: NewSimilarityCalculator(),
	}
}

// VectorStats represents statistics about a vector
type VectorStats struct {
	Dimensions int     `json:"dimensions"`
	Mean       float32 `json:"mean"`
	Std        float32 `json:"std"`
	Min        float32 `json:"min"`
	Max        float32 `json:"max"`
	Norm       float32 `json:"norm"`
}

// AnalyzeVector analyzes a vector and returns statistics
func (va *VectorAnalyzer) AnalyzeVector(vector []float32) *VectorStats {
	if len(vector) == 0 {
		return &VectorStats{}
	}

	stats := &VectorStats{
		Dimensions: len(vector),
		Min:        vector[0],
		Max:        vector[0],
	}

	var sum float64
	for _, v := range vector {
		sum += float64(v)
		if v < stats.Min {
			stats.Min = v
		}
		if v > stats.Max {
			stats.Max = v
		}
	}

	stats.Mean = float32(sum / float64(len(vector)))

	// Calculate standard deviation
	var variance float64
	for _, v := range vector {
		diff := float64(v) - float64(stats.Mean)
		variance += diff * diff
	}
	stats.Std = float32(math.Sqrt(variance / float64(len(vector))))

	// Calculate norm
	stats.Norm = va.calculator.VectorNorm(vector)

	return stats
}

// FindSimilarVectors finds vectors similar to a query vector within a threshold
func (va *VectorAnalyzer) FindSimilarVectors(query []float32, vectors [][]float32, threshold float32, metric string) ([]int, error) {
	var indices []int

	for i, vector := range vectors {
		var similarity float32
		var err error

		switch metric {
		case "cosine":
			similarity, err = va.calculator.CosineSimilarity(query, vector)
		case "euclidean":
			distance, err2 := va.calculator.EuclideanDistance(query, vector)
			if err2 != nil {
				err = err2
			} else {
				// Convert distance to similarity (1 / (1 + distance))
				similarity = 1.0 / (1.0 + distance)
			}
		case "dot":
			similarity, err = va.calculator.DotProduct(query, vector)
		default:
			return nil, fmt.Errorf("unsupported metric: %s", metric)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to calculate similarity: %w", err)
		}

		if similarity >= threshold {
			indices = append(indices, i)
		}
	}

	return indices, nil
}

// VectorIndex provides in-memory vector indexing for fast similarity search
type VectorIndex struct {
	vectors    [][]float32
	metadata   []map[string]interface{}
	calculator *SimilarityCalculator
	mu         sync.RWMutex
}

// NewVectorIndex creates a new vector index
func NewVectorIndex() *VectorIndex {
	return &VectorIndex{
		calculator: NewSimilarityCalculator(),
	}
}

// Add adds a vector to the index
func (vi *VectorIndex) Add(vector []float32, metadata map[string]interface{}) {
	vi.mu.Lock()
	defer vi.mu.Unlock()

	vi.vectors = append(vi.vectors, vector)
	vi.metadata = append(vi.metadata, metadata)
}

// Search searches for similar vectors in the index
func (vi *VectorIndex) Search(query []float32, topK int, metric string) ([]*VectorDBItem, error) {
	vi.mu.RLock()
	defer vi.mu.RUnlock()

	if len(vi.vectors) == 0 {
		return []*VectorDBItem{}, nil
	}

	type result struct {
		index      int
		similarity float32
	}

	results := make([]result, len(vi.vectors))
	for i, vector := range vi.vectors {
		var similarity float32
		var err error

		switch metric {
		case "cosine":
			similarity, err = vi.calculator.CosineSimilarity(query, vector)
		case "euclidean":
			distance, err2 := vi.calculator.EuclideanDistance(query, vector)
			if err2 != nil {
				err = err2
			} else {
				similarity = 1.0 / (1.0 + distance)
			}
		case "dot":
			similarity, err = vi.calculator.DotProduct(query, vector)
		default:
			return nil, fmt.Errorf("unsupported metric: %s", metric)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to calculate similarity: %w", err)
		}

		results[i] = result{index: i, similarity: similarity}
	}

	// Sort by similarity (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].similarity > results[j].similarity
	})

	// Apply topK limit
	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}

	// Convert to VectorDBItems
	items := make([]*VectorDBItem, len(results))
	for i, result := range results {
		item := &VectorDBItem{
			ID:      fmt.Sprintf("index_%d", result.index),
			Vector:  vi.vectors[result.index],
			Payload: vi.metadata[result.index],
			Score:   &result.similarity,
		}
		items[i] = item
	}

	return items, nil
}

// Size returns the number of vectors in the index
func (vi *VectorIndex) Size() int {
	vi.mu.RLock()
	defer vi.mu.RUnlock()
	return len(vi.vectors)
}

// Clear removes all vectors from the index
func (vi *VectorIndex) Clear() {
	vi.mu.Lock()
	defer vi.mu.Unlock()
	vi.vectors = nil
	vi.metadata = nil
}

// PerformanceMonitor tracks vector database operation performance
type PerformanceMonitor struct {
	metrics map[string]*OperationMetrics
	mu      sync.RWMutex
}

// OperationMetrics tracks metrics for a specific operation
type OperationMetrics struct {
	TotalOperations int64         `json:"total_operations"`
	TotalDuration   time.Duration `json:"total_duration"`
	SuccessCount    int64         `json:"success_count"`
	ErrorCount      int64         `json:"error_count"`
	AvgDuration     time.Duration `json:"avg_duration"`
	MinDuration     time.Duration `json:"min_duration"`
	MaxDuration     time.Duration `json:"max_duration"`
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics: make(map[string]*OperationMetrics),
	}
}

// RecordOperation records an operation's performance metrics
func (pm *PerformanceMonitor) RecordOperation(operation string, duration time.Duration, success bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.metrics[operation] == nil {
		pm.metrics[operation] = &OperationMetrics{
			MinDuration: duration,
			MaxDuration: duration,
		}
	}

	metrics := pm.metrics[operation]
	metrics.TotalOperations++
	metrics.TotalDuration += duration

	if success {
		metrics.SuccessCount++
	} else {
		metrics.ErrorCount++
	}

	metrics.AvgDuration = time.Duration(int64(metrics.TotalDuration) / metrics.TotalOperations)

	if duration < metrics.MinDuration {
		metrics.MinDuration = duration
	}
	if duration > metrics.MaxDuration {
		metrics.MaxDuration = duration
	}
}

// GetMetrics returns metrics for a specific operation
func (pm *PerformanceMonitor) GetMetrics(operation string) *OperationMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if metrics, exists := pm.metrics[operation]; exists {
		// Return a copy to prevent external modification
		copy := *metrics
		return &copy
	}
	return nil
}

// GetAllMetrics returns all operation metrics
func (pm *PerformanceMonitor) GetAllMetrics() map[string]*OperationMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[string]*OperationMetrics)
	for op, metrics := range pm.metrics {
		copy := *metrics
		result[op] = &copy
	}
	return result
}

// Reset resets all metrics
func (pm *PerformanceMonitor) Reset() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.metrics = make(map[string]*OperationMetrics)
}