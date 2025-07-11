package chunkers

import (
	"math"
	"testing"
	"time"
)

func TestCosineSimilarityCalculator(t *testing.T) {
	calc := NewCosineSimilarityCalculator()
	
	// Test identical vectors
	vec1 := []float64{1.0, 2.0, 3.0}
	vec2 := []float64{1.0, 2.0, 3.0}
	
	similarity, err := calc.CosineSimilarity(vec1, vec2)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if math.Abs(similarity-1.0) > 1e-10 {
		t.Errorf("Expected similarity to be 1.0, got %f", similarity)
	}
	
	// Test orthogonal vectors
	vec1 = []float64{1.0, 0.0, 0.0}
	vec2 = []float64{0.0, 1.0, 0.0}
	
	similarity, err = calc.CosineSimilarity(vec1, vec2)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if math.Abs(similarity-0.0) > 1e-10 {
		t.Errorf("Expected similarity to be 0.0, got %f", similarity)
	}
	
	// Test opposite vectors
	vec1 = []float64{1.0, 2.0, 3.0}
	vec2 = []float64{-1.0, -2.0, -3.0}
	
	similarity, err = calc.CosineSimilarity(vec1, vec2)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if math.Abs(similarity-(-1.0)) > 1e-10 {
		t.Errorf("Expected similarity to be -1.0, got %f", similarity)
	}
}

func TestCosineSimilarityCalculatorErrors(t *testing.T) {
	calc := NewCosineSimilarityCalculator()
	
	// Test different length vectors
	vec1 := []float64{1.0, 2.0}
	vec2 := []float64{1.0, 2.0, 3.0}
	
	_, err := calc.CosineSimilarity(vec1, vec2)
	if err == nil {
		t.Error("Expected error for different length vectors")
	}
	
	// Test empty vectors
	vec1 = []float64{}
	vec2 = []float64{}
	
	_, err = calc.CosineSimilarity(vec1, vec2)
	if err == nil {
		t.Error("Expected error for empty vectors")
	}
	
	// Test zero vectors
	vec1 = []float64{0.0, 0.0, 0.0}
	vec2 = []float64{1.0, 2.0, 3.0}
	
	similarity, err := calc.CosineSimilarity(vec1, vec2)
	if err != nil {
		t.Errorf("Expected no error for zero vector, got %v", err)
	}
	
	if similarity != 0.0 {
		t.Errorf("Expected similarity to be 0.0 for zero vector, got %f", similarity)
	}
}

func TestBatchCosineSimilarity(t *testing.T) {
	calc := NewCosineSimilarityCalculator()
	
	vectors1 := [][]float64{
		{1.0, 0.0, 0.0},
		{0.0, 1.0, 0.0},
		{1.0, 1.0, 0.0},
	}
	
	vectors2 := [][]float64{
		{1.0, 0.0, 0.0},
		{0.0, 1.0, 0.0},
		{1.0, 0.0, 0.0},
	}
	
	similarities, err := calc.BatchCosineSimilarity(vectors1, vectors2)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if len(similarities) != 3 {
		t.Errorf("Expected 3 similarities, got %d", len(similarities))
	}
	
	// First pair: identical vectors
	if math.Abs(similarities[0]-1.0) > 1e-10 {
		t.Errorf("Expected first similarity to be 1.0, got %f", similarities[0])
	}
	
	// Second pair: identical vectors
	if math.Abs(similarities[1]-1.0) > 1e-10 {
		t.Errorf("Expected second similarity to be 1.0, got %f", similarities[1])
	}
	
	// Third pair: different vectors
	expected := 1.0 / math.Sqrt(2) // cos(45°)
	if math.Abs(similarities[2]-expected) > 1e-10 {
		t.Errorf("Expected third similarity to be %f, got %f", expected, similarities[2])
	}
}

func TestSimilarityMatrix(t *testing.T) {
	calc := NewCosineSimilarityCalculator()
	
	vectors := [][]float64{
		{1.0, 0.0, 0.0},
		{0.0, 1.0, 0.0},
		{1.0, 1.0, 0.0},
	}
	
	matrix, err := calc.SimilarityMatrix(vectors)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if len(matrix) != 3 {
		t.Errorf("Expected 3x3 matrix, got %d rows", len(matrix))
		return
	}
	
	for i, row := range matrix {
		if len(row) != 3 {
			t.Errorf("Expected row %d to have 3 columns, got %d", i, len(row))
		}
	}
	
	// Check diagonal elements (should be 1.0)
	for i := 0; i < 3; i++ {
		if math.Abs(matrix[i][i]-1.0) > 1e-10 {
			t.Errorf("Expected diagonal element [%d][%d] to be 1.0, got %f", i, i, matrix[i][i])
		}
	}
	
	// Check symmetry
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if math.Abs(matrix[i][j]-matrix[j][i]) > 1e-10 {
				t.Errorf("Expected symmetric matrix, but [%d][%d] = %f, [%d][%d] = %f", 
					i, j, matrix[i][j], j, i, matrix[j][i])
			}
		}
	}
	
	// Check specific values
	// vectors[0] and vectors[1] are orthogonal
	if math.Abs(matrix[0][1]-0.0) > 1e-10 {
		t.Errorf("Expected orthogonal vectors to have similarity 0.0, got %f", matrix[0][1])
	}
}

func TestFindSimilarVectors(t *testing.T) {
	calc := NewCosineSimilarityCalculator()
	
	query := []float64{1.0, 0.0, 0.0}
	vectors := [][]float64{
		{1.0, 0.0, 0.0},     // identical
		{0.0, 1.0, 0.0},     // orthogonal
		{0.5, 0.5, 0.0},     // similar
		{-1.0, 0.0, 0.0},    // opposite
	}
	
	// Find vectors with similarity > 0.5
	matches, err := calc.FindSimilarVectors(query, vectors, 0.5)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if len(matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(matches))
	}
	
	// Check that results are sorted by score (descending)
	for i := 0; i < len(matches)-1; i++ {
		if matches[i].Score < matches[i+1].Score {
			t.Errorf("Expected results to be sorted by score descending")
		}
	}
	
	// First match should be identical vector
	if matches[0].Index != 0 {
		t.Errorf("Expected first match to be index 0, got %d", matches[0].Index)
	}
	
	if math.Abs(matches[0].Score-1.0) > 1e-10 {
		t.Errorf("Expected first match score to be 1.0, got %f", matches[0].Score)
	}
}

func TestSemanticBoundaryDetector(t *testing.T) {
	detector := NewCosineSemanticBoundaryDetector()
	
	// Create test embeddings with clear boundaries
	embeddings := [][]float64{
		{1.0, 0.0, 0.0}, // Group 1
		{0.9, 0.1, 0.0}, // Group 1
		{0.8, 0.2, 0.0}, // Group 1
		{0.0, 1.0, 0.0}, // Group 2 (boundary here)
		{0.1, 0.9, 0.0}, // Group 2
		{0.2, 0.8, 0.0}, // Group 2
	}
	
	// Test boundary score calculation
	scores, err := detector.GetBoundaryScores(embeddings)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if len(scores) != len(embeddings)-1 {
		t.Errorf("Expected %d boundary scores, got %d", len(embeddings)-1, len(scores))
	}
	
	// The boundary between groups should have the lowest score
	minScore := scores[0]
	minIndex := 0
	for i, score := range scores {
		if score < minScore {
			minScore = score
			minIndex = i
		}
	}
	
	// The boundary should be between index 2 and 3
	if minIndex != 2 {
		t.Errorf("Expected boundary at index 2, got %d", minIndex)
	}
}

func TestBoundaryDetectionMethods(t *testing.T) {
	detector := NewCosineSemanticBoundaryDetector()
	
	// Create test embeddings
	embeddings := [][]float64{
		{1.0, 0.0, 0.0},
		{0.9, 0.1, 0.0},
		{0.0, 1.0, 0.0}, // Clear boundary
		{0.1, 0.9, 0.0},
		{0.0, 0.0, 1.0}, // Another boundary
		{0.0, 0.1, 0.9},
	}
	
	methods := []BoundaryDetectionMethod{
		BoundaryMethodPercentile,
		BoundaryMethodInterquartile,
		BoundaryMethodGradient,
		BoundaryMethodAdaptive,
	}
	
	for _, method := range methods {
		boundaries, err := detector.DetectBoundaries(embeddings, method)
		if err != nil {
			t.Errorf("Expected no error for method %s, got %v", method, err)
		}
		
		// Some methods may not detect boundaries depending on the data
		// This is acceptable behavior, so just log it
		if len(boundaries) == 0 {
			t.Logf("No boundaries detected for method %s (this may be expected)", method)
		}
		
		// Boundaries should be in valid range
		for _, boundary := range boundaries {
			if boundary < 1 || boundary >= len(embeddings) {
				t.Errorf("Invalid boundary position %d for method %s", boundary, method)
			}
		}
	}
}

func TestThresholdMethods(t *testing.T) {
	detector := NewCosineSemanticBoundaryDetector()
	
	embeddings := [][]float64{
		{1.0, 0.0, 0.0},
		{0.9, 0.1, 0.0},
		{0.0, 1.0, 0.0},
		{0.1, 0.9, 0.0},
	}
	
	methods := []ThresholdMethod{
		ThresholdMethodMean,
		ThresholdMethodMedian,
		ThresholdMethodPercentile,
		ThresholdMethodStdDev,
	}
	
	for _, method := range methods {
		threshold, err := detector.CalculateOptimalThreshold(embeddings, method)
		if err != nil {
			t.Errorf("Expected no error for method %s, got %v", method, err)
		}
		
		// Threshold should be reasonable (between -1 and 1 for cosine similarity)
		if threshold < -1.0 || threshold > 1.0 {
			t.Errorf("Invalid threshold %f for method %s", threshold, method)
		}
	}
}

func TestSemanticAnalyzer(t *testing.T) {
	analyzer := NewSemanticAnalyzer()
	
	// Create test embeddings with two distinct groups
	embeddings := [][]float64{
		{1.0, 0.0, 0.0}, // Group 1
		{0.9, 0.1, 0.0}, // Group 1
		{0.8, 0.2, 0.0}, // Group 1
		{0.0, 1.0, 0.0}, // Group 2
		{0.1, 0.9, 0.0}, // Group 2
		{0.2, 0.8, 0.0}, // Group 2
	}
	
	analysis, err := analyzer.AnalyzeSemanticCoherence(embeddings)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Check basic properties
	if analysis.SegmentCount != len(embeddings) {
		t.Errorf("Expected segment count %d, got %d", len(embeddings), analysis.SegmentCount)
	}
	
	if analysis.AverageSimilarity < 0 || analysis.AverageSimilarity > 1 {
		t.Errorf("Average similarity should be between 0 and 1, got %f", analysis.AverageSimilarity)
	}
	
	if analysis.SequentialSimilarity < 0 || analysis.SequentialSimilarity > 1 {
		t.Errorf("Sequential similarity should be between 0 and 1, got %f", analysis.SequentialSimilarity)
	}
	
	if analysis.CoherenceScore < 0 || analysis.CoherenceScore > 1 {
		t.Errorf("Coherence score should be between 0 and 1, got %f", analysis.CoherenceScore)
	}
	
	// Should detect at least one boundary
	if len(analysis.BoundaryPositions) == 0 {
		t.Error("Expected at least one boundary to be detected")
	}
	
	// Check similarity matrix
	if len(analysis.SimilarityMatrix) != len(embeddings) {
		t.Errorf("Expected similarity matrix size %d, got %d", 
			len(embeddings), len(analysis.SimilarityMatrix))
	}
}

func TestSemanticAnalyzerEdgeCases(t *testing.T) {
	analyzer := NewSemanticAnalyzer()
	
	// Test empty embeddings
	_, err := analyzer.AnalyzeSemanticCoherence([][]float64{})
	if err == nil {
		t.Error("Expected error for empty embeddings")
	}
	
	// Test single embedding
	embeddings := [][]float64{
		{1.0, 0.0, 0.0},
	}
	
	analysis, err := analyzer.AnalyzeSemanticCoherence(embeddings)
	if err != nil {
		t.Errorf("Expected no error for single embedding, got %v", err)
	}
	
	if analysis.SegmentCount != 1 {
		t.Errorf("Expected segment count 1, got %d", analysis.SegmentCount)
	}
	
	if analysis.SequentialSimilarity != 0 {
		t.Errorf("Expected sequential similarity 0 for single embedding, got %f", analysis.SequentialSimilarity)
	}
}

func TestCalculateStats(t *testing.T) {
	// Helper function to create test chunks
	createChunk := func(text string, tokenCount int) *Chunk {
		return &Chunk{
			Text:       text,
			TokenCount: tokenCount,
		}
	}
	
	chunks := []*Chunk{
		createChunk("chunk1", 10),
		createChunk("chunk2", 20),
		createChunk("chunk3", 30),
	}
	
	stats := CalculateStats(chunks, 100, 10*time.Millisecond)
	
	if stats.TotalChunks != 3 {
		t.Errorf("Expected 3 chunks, got %d", stats.TotalChunks)
	}
	
	if stats.TotalTokens != 60 {
		t.Errorf("Expected 60 total tokens, got %d", stats.TotalTokens)
	}
	
	if stats.MinChunkSize != 10 {
		t.Errorf("Expected min chunk size 10, got %d", stats.MinChunkSize)
	}
	
	if stats.MaxChunkSize != 30 {
		t.Errorf("Expected max chunk size 30, got %d", stats.MaxChunkSize)
	}
	
	if stats.AverageChunkSize != 20.0 {
		t.Errorf("Expected average chunk size 20.0, got %f", stats.AverageChunkSize)
	}
	
	if stats.OriginalTextLength != 100 {
		t.Errorf("Expected original text length 100, got %d", stats.OriginalTextLength)
	}
}