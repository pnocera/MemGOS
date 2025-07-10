package modules

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
)

// SchedulerRetriever handles memory retrieval and reranking operations
type SchedulerRetriever struct {
	*BaseSchedulerModule
	mu       sync.RWMutex
	chatLLM  interfaces.LLM
	memCube  interface{} // Will be GeneralMemCube interface when available
	logger   interfaces.Logger
	cacheMap map[string]*RetrievalCache
}

// RetrievalCache represents cached retrieval results
type RetrievalCache struct {
	Query     string         `json:"query"`
	Results   []*SearchResult `json:"results"`
	Timestamp time.Time      `json:"timestamp"`
	TTL       time.Duration  `json:"ttl"`
}

// IsExpired checks if the cache entry has expired
func (rc *RetrievalCache) IsExpired() bool {
	return time.Since(rc.Timestamp) > rc.TTL
}

// NewSchedulerRetriever creates a new scheduler retriever
func NewSchedulerRetriever(chatLLM interfaces.LLM) *SchedulerRetriever {
	retriever := &SchedulerRetriever{
		BaseSchedulerModule: NewBaseSchedulerModule(),
		chatLLM:             chatLLM,
		logger:              nil, // Will be set by parent
		cacheMap:            make(map[string]*RetrievalCache),
	}
	
	// Initialize default templates
	if err := retriever.InitializeDefaultTemplates(); err != nil {
		retriever.logger.Error("Failed to initialize templates", "error", err)
	}
	
	return retriever
}

// SetMemCube sets the memory cube for retrieval operations
func (r *SchedulerRetriever) SetMemCube(memCube interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.memCube = memCube
}

// GetMemCube returns the current memory cube
func (r *SchedulerRetriever) GetMemCube() interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.memCube
}

// Search performs a search operation using the specified method
func (r *SchedulerRetriever) Search(query string, topK int, method string) ([]*SearchResult, error) {
	if r.memCube == nil {
		return nil, fmt.Errorf("memory cube not initialized")
	}
	
	// Check cache first
	if cached := r.getCachedResults(query); cached != nil && !cached.IsExpired() {
		r.logger.Debug("Using cached search results", "query", query)
		return cached.Results, nil
	}
	
	r.logger.Debug("Performing search", "query", query, "top_k", topK, "method", method)
	
	var results []*SearchResult
	var err error
	
	switch method {
	case TextMemorySearchMethod:
		results, err = r.performTextMemorySearch(query, topK)
	case TreeTextMemorySearchMethod:
		results, err = r.performTreeTextMemorySearch(query, topK)
	default:
		return nil, fmt.Errorf("unsupported search method: %s", method)
	}
	
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	
	// Cache results
	r.cacheResults(query, results)
	
	return results, nil
}

// performTextMemorySearch performs search using text memory method
func (r *SchedulerRetriever) performTextMemorySearch(query string, topK int) ([]*SearchResult, error) {
	// This would integrate with the actual text memory implementation
	// For now, return a placeholder implementation
	r.logger.Debug("Performing text memory search", "query", query, "top_k", topK)
	
	// Placeholder: In real implementation, this would call the memory cube's search method
	results := make([]*SearchResult, 0)
	
	// Example placeholder results
	for i := 0; i < min(topK, 3); i++ {
		result := &SearchResult{
			ID:      fmt.Sprintf("text_mem_%d", i),
			Content: fmt.Sprintf("Text memory result %d for query: %s", i+1, query),
			Score:   0.9 - float64(i)*0.1,
			Metadata: map[string]interface{}{
				"method": "text_memory",
				"type":   "LongTermMemory",
			},
		}
		results = append(results, result)
	}
	
	return results, nil
}

// performTreeTextMemorySearch performs search using tree text memory method
func (r *SchedulerRetriever) performTreeTextMemorySearch(query string, topK int) ([]*SearchResult, error) {
	// This would integrate with the actual tree text memory implementation
	r.logger.Debug("Performing tree text memory search", "query", query, "top_k", topK)
	
	// Placeholder: In real implementation, this would:
	// 1. Search LongTermMemory
	// 2. Search UserMemory  
	// 3. Combine and rank results
	
	results := make([]*SearchResult, 0)
	
	// Simulate LongTermMemory results
	for i := 0; i < min(topK/2, 2); i++ {
		result := &SearchResult{
			ID:      fmt.Sprintf("ltm_%d", i),
			Content: fmt.Sprintf("Long-term memory result %d for: %s", i+1, query),
			Score:   0.95 - float64(i)*0.05,
			Metadata: map[string]interface{}{
				"method": "tree_text_memory",
				"type":   "LongTermMemory",
			},
		}
		results = append(results, result)
	}
	
	// Simulate UserMemory results
	for i := 0; i < min(topK/2, 2); i++ {
		result := &SearchResult{
			ID:      fmt.Sprintf("um_%d", i),
			Content: fmt.Sprintf("User memory result %d for: %s", i+1, query),
			Score:   0.85 - float64(i)*0.05,
			Metadata: map[string]interface{}{
				"method": "tree_text_memory", 
				"type":   "UserMemory",
			},
		}
		results = append(results, result)
	}
	
	return results, nil
}

// ReplaceWorkingMemory reranks and replaces working memory with new candidates
func (r *SchedulerRetriever) ReplaceWorkingMemory(
	originalMemory []string,
	newMemory []*SearchResult,
	topK, topN int,
	query string) ([]string, error) {
	
	if r.chatLLM == nil {
		return nil, fmt.Errorf("chat LLM not initialized")
	}
	
	// Combine original memory and new search results
	combinedMemory := make([]string, 0, len(originalMemory)+len(newMemory))
	combinedMemory = append(combinedMemory, originalMemory...)
	
	for _, result := range newMemory {
		combinedMemory = append(combinedMemory, result.Content)
	}
	
	// Remove duplicates while preserving order
	uniqueMemory := removeDuplicates(combinedMemory)
	
	// Rerank using LLM
	rerankedMemory, err := r.rerankMemory(query, uniqueMemory, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to rerank memory: %w", err)
	}
	
	// Return top N + top K items
	maxItems := min(len(rerankedMemory), topN+topK)
	result := rerankedMemory[:maxItems]
	
	r.logger.Debug("Working memory replaced", 
		"original_count", len(originalMemory),
		"new_count", len(newMemory),
		"result_count", len(result))
	
	return result, nil
}

// rerankMemory uses LLM to rerank memory items based on relevance
func (r *SchedulerRetriever) rerankMemory(query string, currentOrder []string, stagingBuffer []string) ([]string, error) {
	params := map[string]interface{}{
		"query":          query,
		"current_order":  currentOrder,
		"staging_buffer": stagingBuffer,
	}
	
	prompt, err := r.BuildPrompt("memory_reranking", params)
	if err != nil {
		return currentOrder, fmt.Errorf("failed to build reranking prompt: %w", err)
	}
	
	// Generate response from LLM
	messages := []map[string]string{
		{"role": "user", "content": prompt},
	}
	
	response, err := r.chatLLM.Generate(messages)
	if err != nil {
		return currentOrder, fmt.Errorf("LLM generation failed: %w", err)
	}
	
	// Parse JSON response
	result, err := r.extractJSONDict(response)
	if err != nil {
		return currentOrder, fmt.Errorf("failed to parse reranking response: %w", err)
	}
	
	// Extract new order
	if newOrder, ok := result["new_order"]; ok {
		if orderSlice, ok := newOrder.([]interface{}); ok {
			reranked := make([]string, 0, len(orderSlice))
			for _, item := range orderSlice {
				if str, ok := item.(string); ok {
					reranked = append(reranked, str)
				}
			}
			return reranked, nil
		}
	}
	
	return currentOrder, fmt.Errorf("failed to extract new order from response")
}

// getCachedResults retrieves cached search results
func (r *SchedulerRetriever) getCachedResults(query string) *RetrievalCache {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if cached, exists := r.cacheMap[query]; exists {
		return cached
	}
	return nil
}

// cacheResults stores search results in cache
func (r *SchedulerRetriever) cacheResults(query string, results []*SearchResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	cache := &RetrievalCache{
		Query:     query,
		Results:   results,
		Timestamp: time.Now(),
		TTL:       5 * time.Minute, // Cache for 5 minutes
	}
	
	r.cacheMap[query] = cache
	
	// Clean up expired entries (simple cleanup)
	if len(r.cacheMap) > 100 {
		r.cleanupCache()
	}
}

// cleanupCache removes expired cache entries
func (r *SchedulerRetriever) cleanupCache() {
	for query, cache := range r.cacheMap {
		if cache.IsExpired() {
			delete(r.cacheMap, query)
		}
	}
}

// ClearCache clears all cached results
func (r *SchedulerRetriever) ClearCache() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cacheMap = make(map[string]*RetrievalCache)
}

// GetCacheStats returns cache statistics
func (r *SchedulerRetriever) GetCacheStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	expiredCount := 0
	for _, cache := range r.cacheMap {
		if cache.IsExpired() {
			expiredCount++
		}
	}
	
	return map[string]interface{}{
		"total_entries":   len(r.cacheMap),
		"expired_entries": expiredCount,
		"active_entries":  len(r.cacheMap) - expiredCount,
	}
}

// extractJSONDict extracts JSON dictionary from LLM response
func (r *SchedulerRetriever) extractJSONDict(response string) (map[string]interface{}, error) {
	// Try to parse as JSON directly
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err == nil {
		return result, nil
	}
	
	// If direct parsing fails, try to extract JSON from the response
	start := -1
	end := -1
	
	for i, char := range response {
		if char == '{' && start == -1 {
			start = i
		}
		if char == '}' {
			end = i + 1
		}
	}
	
	if start != -1 && end != -1 && end > start {
		jsonStr := response[start:end]
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
			return result, nil
		}
	}
	
	return nil, fmt.Errorf("failed to extract JSON from response: %s", response)
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(slice))
	
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	
	return result
}
