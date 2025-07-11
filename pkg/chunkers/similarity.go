package chunkers

import (
	"fmt"
	"math"
	"sort"
)

// SimilarityCalculator defines the interface for similarity calculations
type SimilarityCalculator interface {
	// CosineSimilarity calculates cosine similarity between two vectors
	CosineSimilarity(vec1, vec2 []float64) (float64, error)
	
	// BatchCosineSimilarity calculates cosine similarity for multiple vector pairs
	BatchCosineSimilarity(vectors1, vectors2 [][]float64) ([]float64, error)
	
	// SimilarityMatrix calculates pairwise similarity matrix
	SimilarityMatrix(vectors [][]float64) ([][]float64, error)
	
	// FindSimilarVectors finds vectors similar to a query vector above threshold
	FindSimilarVectors(query []float64, vectors [][]float64, threshold float64) ([]SimilarityMatch, error)
	
	// GetSimilarityType returns the type of similarity being calculated
	GetSimilarityType() string
}

// SimilarityMatch represents a similarity match result
type SimilarityMatch struct {
	Index      int     `json:"index"`
	Score      float64 `json:"score"`
	Vector     []float64 `json:"vector,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// CosineSimilarityCalculator implements cosine similarity calculations
type CosineSimilarityCalculator struct{}

// NewCosineSimilarityCalculator creates a new cosine similarity calculator
func NewCosineSimilarityCalculator() *CosineSimilarityCalculator {
	return &CosineSimilarityCalculator{}
}

// CosineSimilarity calculates cosine similarity between two vectors
func (c *CosineSimilarityCalculator) CosineSimilarity(vec1, vec2 []float64) (float64, error) {
	if len(vec1) != len(vec2) {
		return 0, fmt.Errorf("vectors must have the same length: %d vs %d", len(vec1), len(vec2))
	}
	
	if len(vec1) == 0 {
		return 0, fmt.Errorf("vectors cannot be empty")
	}
	
	var dotProduct, norm1, norm2 float64
	
	for i := 0; i < len(vec1); i++ {
		dotProduct += vec1[i] * vec2[i]
		norm1 += vec1[i] * vec1[i]
		norm2 += vec2[i] * vec2[i]
	}
	
	// Handle zero vectors
	if norm1 == 0 || norm2 == 0 {
		return 0, nil
	}
	
	norm1 = math.Sqrt(norm1)
	norm2 = math.Sqrt(norm2)
	
	return dotProduct / (norm1 * norm2), nil
}

// BatchCosineSimilarity calculates cosine similarity for multiple vector pairs
func (c *CosineSimilarityCalculator) BatchCosineSimilarity(vectors1, vectors2 [][]float64) ([]float64, error) {
	if len(vectors1) != len(vectors2) {
		return nil, fmt.Errorf("vector arrays must have the same length")
	}
	
	similarities := make([]float64, len(vectors1))
	
	for i := 0; i < len(vectors1); i++ {
		sim, err := c.CosineSimilarity(vectors1[i], vectors2[i])
		if err != nil {
			return nil, fmt.Errorf("failed to calculate similarity for pair %d: %w", i, err)
		}
		similarities[i] = sim
	}
	
	return similarities, nil
}

// SimilarityMatrix calculates pairwise similarity matrix
func (c *CosineSimilarityCalculator) SimilarityMatrix(vectors [][]float64) ([][]float64, error) {
	if len(vectors) == 0 {
		return [][]float64{}, nil
	}
	
	n := len(vectors)
	matrix := make([][]float64, n)
	
	// Initialize all rows first
	for i := 0; i < n; i++ {
		matrix[i] = make([]float64, n)
	}
	
	// Calculate similarities
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				matrix[i][j] = 1.0 // Self-similarity is always 1
			} else if i < j {
				// Calculate similarity
				sim, err := c.CosineSimilarity(vectors[i], vectors[j])
				if err != nil {
					return nil, fmt.Errorf("failed to calculate similarity between vectors %d and %d: %w", i, j, err)
				}
				matrix[i][j] = sim
				matrix[j][i] = sim // Symmetric matrix
			}
			// If i > j, the value was already set when j < i
		}
	}
	
	return matrix, nil
}

// FindSimilarVectors finds vectors similar to a query vector above threshold
func (c *CosineSimilarityCalculator) FindSimilarVectors(query []float64, vectors [][]float64, threshold float64) ([]SimilarityMatch, error) {
	if len(vectors) == 0 {
		return []SimilarityMatch{}, nil
	}
	
	var matches []SimilarityMatch
	
	for i, vec := range vectors {
		sim, err := c.CosineSimilarity(query, vec)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate similarity for vector %d: %w", i, err)
		}
		
		if sim >= threshold {
			matches = append(matches, SimilarityMatch{
				Index: i,
				Score: sim,
				Vector: vec,
			})
		}
	}
	
	// Sort by score descending
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})
	
	return matches, nil
}

// GetSimilarityType returns the type of similarity being calculated
func (c *CosineSimilarityCalculator) GetSimilarityType() string {
	return "cosine"
}

// SemanticBoundaryDetector detects semantic boundaries in text
type SemanticBoundaryDetector interface {
	// DetectBoundaries detects semantic boundaries in a sequence of embeddings
	DetectBoundaries(embeddings [][]float64, method BoundaryDetectionMethod) ([]int, error)
	
	// DetectBoundariesWithThreshold detects boundaries using a specific threshold
	DetectBoundariesWithThreshold(embeddings [][]float64, threshold float64) ([]int, error)
	
	// CalculateOptimalThreshold calculates optimal threshold for boundary detection
	CalculateOptimalThreshold(embeddings [][]float64, method ThresholdMethod) (float64, error)
	
	// GetBoundaryScores returns boundary scores for each position
	GetBoundaryScores(embeddings [][]float64) ([]float64, error)
}

// BoundaryDetectionMethod defines methods for boundary detection
type BoundaryDetectionMethod string

const (
	// BoundaryMethodPercentile uses percentile-based threshold
	BoundaryMethodPercentile BoundaryDetectionMethod = "percentile"
	
	// BoundaryMethodInterquartile uses interquartile range
	BoundaryMethodInterquartile BoundaryDetectionMethod = "interquartile"
	
	// BoundaryMethodGradient uses gradient-based detection
	BoundaryMethodGradient BoundaryDetectionMethod = "gradient"
	
	// BoundaryMethodAdaptive uses adaptive thresholding
	BoundaryMethodAdaptive BoundaryDetectionMethod = "adaptive"
)

// ThresholdMethod defines methods for threshold calculation
type ThresholdMethod string

const (
	// ThresholdMethodMean uses mean similarity as threshold
	ThresholdMethodMean ThresholdMethod = "mean"
	
	// ThresholdMethodMedian uses median similarity as threshold
	ThresholdMethodMedian ThresholdMethod = "median"
	
	// ThresholdMethodPercentile uses percentile-based threshold
	ThresholdMethodPercentile ThresholdMethod = "percentile"
	
	// ThresholdMethodStdDev uses standard deviation-based threshold
	ThresholdMethodStdDev ThresholdMethod = "stddev"
)

// CosineSemanticBoundaryDetector implements semantic boundary detection using cosine similarity
type CosineSemanticBoundaryDetector struct {
	calculator SimilarityCalculator
}

// NewCosineSemanticBoundaryDetector creates a new cosine-based boundary detector
func NewCosineSemanticBoundaryDetector() *CosineSemanticBoundaryDetector {
	return &CosineSemanticBoundaryDetector{
		calculator: NewCosineSimilarityCalculator(),
	}
}

// DetectBoundaries detects semantic boundaries in a sequence of embeddings
func (d *CosineSemanticBoundaryDetector) DetectBoundaries(embeddings [][]float64, method BoundaryDetectionMethod) ([]int, error) {
	if len(embeddings) <= 1 {
		return []int{}, nil
	}
	
	// Calculate boundary scores
	scores, err := d.GetBoundaryScores(embeddings)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate boundary scores: %w", err)
	}
	
	// Calculate threshold based on method
	var threshold float64
	switch method {
	case BoundaryMethodPercentile:
		threshold, err = d.calculatePercentileThreshold(scores, 0.7) // 70th percentile
	case BoundaryMethodInterquartile:
		threshold, err = d.calculateInterquartileThreshold(scores)
	case BoundaryMethodGradient:
		threshold, err = d.calculateGradientThreshold(scores)
	case BoundaryMethodAdaptive:
		threshold, err = d.calculateAdaptiveThreshold(scores)
	default:
		return nil, fmt.Errorf("unsupported boundary detection method: %s", method)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to calculate threshold: %w", err)
	}
	
	// Find boundaries
	return d.DetectBoundariesWithThreshold(embeddings, threshold)
}

// DetectBoundariesWithThreshold detects boundaries using a specific threshold
func (d *CosineSemanticBoundaryDetector) DetectBoundariesWithThreshold(embeddings [][]float64, threshold float64) ([]int, error) {
	if len(embeddings) <= 1 {
		return []int{}, nil
	}
	
	scores, err := d.GetBoundaryScores(embeddings)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate boundary scores: %w", err)
	}
	
	var boundaries []int
	for i, score := range scores {
		if score <= threshold {
			boundaries = append(boundaries, i+1) // Boundary is after position i
		}
	}
	
	return boundaries, nil
}

// CalculateOptimalThreshold calculates optimal threshold for boundary detection
func (d *CosineSemanticBoundaryDetector) CalculateOptimalThreshold(embeddings [][]float64, method ThresholdMethod) (float64, error) {
	if len(embeddings) <= 1 {
		return 0, fmt.Errorf("need at least 2 embeddings")
	}
	
	scores, err := d.GetBoundaryScores(embeddings)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate boundary scores: %w", err)
	}
	
	switch method {
	case ThresholdMethodMean:
		return d.calculateMean(scores), nil
	case ThresholdMethodMedian:
		return d.calculateMedian(scores), nil
	case ThresholdMethodPercentile:
		return d.calculatePercentileThreshold(scores, 0.5) // 50th percentile
	case ThresholdMethodStdDev:
		return d.calculateStdDevThreshold(scores)
	default:
		return 0, fmt.Errorf("unsupported threshold method: %s", method)
	}
}

// GetBoundaryScores returns boundary scores for each position
func (d *CosineSemanticBoundaryDetector) GetBoundaryScores(embeddings [][]float64) ([]float64, error) {
	if len(embeddings) <= 1 {
		return []float64{}, nil
	}
	
	scores := make([]float64, len(embeddings)-1)
	
	for i := 0; i < len(embeddings)-1; i++ {
		sim, err := d.calculator.CosineSimilarity(embeddings[i], embeddings[i+1])
		if err != nil {
			return nil, fmt.Errorf("failed to calculate similarity between embeddings %d and %d: %w", i, i+1, err)
		}
		scores[i] = sim
	}
	
	return scores, nil
}

// Helper methods for threshold calculation
func (d *CosineSemanticBoundaryDetector) calculatePercentileThreshold(scores []float64, percentile float64) (float64, error) {
	if len(scores) == 0 {
		return 0, fmt.Errorf("no scores provided")
	}
	
	sorted := make([]float64, len(scores))
	copy(sorted, scores)
	sort.Float64s(sorted)
	
	index := int(percentile * float64(len(sorted)))
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	
	return sorted[index], nil
}

func (d *CosineSemanticBoundaryDetector) calculateInterquartileThreshold(scores []float64) (float64, error) {
	if len(scores) == 0 {
		return 0, fmt.Errorf("no scores provided")
	}
	
	q1, err := d.calculatePercentileThreshold(scores, 0.25)
	if err != nil {
		return 0, err
	}
	
	q3, err := d.calculatePercentileThreshold(scores, 0.75)
	if err != nil {
		return 0, err
	}
	
	// Use Q1 - 1.5 * IQR as threshold for outliers
	iqr := q3 - q1
	return q1 - 1.5*iqr, nil
}

func (d *CosineSemanticBoundaryDetector) calculateGradientThreshold(scores []float64) (float64, error) {
	if len(scores) <= 1 {
		return 0, fmt.Errorf("need at least 2 scores")
	}
	
	// Calculate gradients
	gradients := make([]float64, len(scores)-1)
	for i := 0; i < len(scores)-1; i++ {
		gradients[i] = math.Abs(scores[i+1] - scores[i])
	}
	
	// Use mean gradient as threshold
	return d.calculateMean(gradients), nil
}

func (d *CosineSemanticBoundaryDetector) calculateAdaptiveThreshold(scores []float64) (float64, error) {
	if len(scores) == 0 {
		return 0, fmt.Errorf("no scores provided")
	}
	
	// Adaptive threshold based on local statistics
	mean := d.calculateMean(scores)
	stdDev := d.calculateStdDev(scores, mean)
	
	// Threshold is one standard deviation below mean
	return mean - stdDev, nil
}

func (d *CosineSemanticBoundaryDetector) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (d *CosineSemanticBoundaryDetector) calculateMedian(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	
	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

func (d *CosineSemanticBoundaryDetector) calculateStdDev(values []float64, mean float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	
	return math.Sqrt(sumSquares / float64(len(values)-1))
}

func (d *CosineSemanticBoundaryDetector) calculateStdDevThreshold(scores []float64) (float64, error) {
	if len(scores) == 0 {
		return 0, fmt.Errorf("no scores provided")
	}
	
	mean := d.calculateMean(scores)
	stdDev := d.calculateStdDev(scores, mean)
	
	// Threshold is one standard deviation below mean
	return mean - stdDev, nil
}

// SemanticAnalyzer provides high-level semantic analysis utilities
type SemanticAnalyzer struct {
	calculator SimilarityCalculator
	detector   SemanticBoundaryDetector
}

// NewSemanticAnalyzer creates a new semantic analyzer
func NewSemanticAnalyzer() *SemanticAnalyzer {
	return &SemanticAnalyzer{
		calculator: NewCosineSimilarityCalculator(),
		detector:   NewCosineSemanticBoundaryDetector(),
	}
}

// AnalyzeSemanticCoherence analyzes semantic coherence of text segments
func (sa *SemanticAnalyzer) AnalyzeSemanticCoherence(embeddings [][]float64) (*SemanticCoherenceAnalysis, error) {
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings provided")
	}
	
	// Calculate similarity matrix
	matrix, err := sa.calculator.SimilarityMatrix(embeddings)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate similarity matrix: %w", err)
	}
	
	// Calculate coherence metrics
	analysis := &SemanticCoherenceAnalysis{
		SegmentCount:     len(embeddings),
		SimilarityMatrix: matrix,
	}
	
	// Calculate average similarity
	totalSim := 0.0
	count := 0
	for i := 0; i < len(matrix); i++ {
		for j := i + 1; j < len(matrix[i]); j++ {
			totalSim += matrix[i][j]
			count++
		}
	}
	
	if count > 0 {
		analysis.AverageSimilarity = totalSim / float64(count)
	}
	
	// Calculate sequential similarity
	if len(embeddings) > 1 {
		seqSim := 0.0
		for i := 0; i < len(embeddings)-1; i++ {
			seqSim += matrix[i][i+1]
		}
		analysis.SequentialSimilarity = seqSim / float64(len(embeddings)-1)
	}
	
	// Detect boundaries
	boundaries, err := sa.detector.DetectBoundaries(embeddings, BoundaryMethodPercentile)
	if err != nil {
		return nil, fmt.Errorf("failed to detect boundaries: %w", err)
	}
	analysis.BoundaryPositions = boundaries
	
	// Calculate coherence score
	analysis.CoherenceScore = sa.calculateCoherenceScore(analysis)
	
	return analysis, nil
}

// SemanticCoherenceAnalysis contains results of semantic coherence analysis
type SemanticCoherenceAnalysis struct {
	SegmentCount         int         `json:"segment_count"`
	AverageSimilarity    float64     `json:"average_similarity"`
	SequentialSimilarity float64     `json:"sequential_similarity"`
	CoherenceScore       float64     `json:"coherence_score"`
	BoundaryPositions    []int       `json:"boundary_positions"`
	SimilarityMatrix     [][]float64 `json:"similarity_matrix,omitempty"`
}

// calculateCoherenceScore calculates an overall coherence score
func (sa *SemanticAnalyzer) calculateCoherenceScore(analysis *SemanticCoherenceAnalysis) float64 {
	// Weighted combination of different metrics
	sequentialWeight := 0.6
	averageWeight := 0.4
	
	score := sequentialWeight*analysis.SequentialSimilarity + averageWeight*analysis.AverageSimilarity
	
	// Penalty for too many boundaries
	if analysis.SegmentCount > 0 {
		boundaryRatio := float64(len(analysis.BoundaryPositions)) / float64(analysis.SegmentCount)
		if boundaryRatio > 0.5 {
			score *= (1.0 - (boundaryRatio-0.5)*0.5) // Reduce score for high boundary ratio
		}
	}
	
	return math.Max(0, math.Min(1, score))
}