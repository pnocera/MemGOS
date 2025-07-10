package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/logger"
)

// NATSRetrieverConfig holds configuration for NATS-based retrieval
type NATSRetrieverConfig struct {
	RetrievalRequestSubject   string        `json:"retrieval_request_subject" yaml:"retrieval_request_subject"`
	RetrievalResponseSubject  string        `json:"retrieval_response_subject" yaml:"retrieval_response_subject"`
	CacheSubject             string        `json:"cache_subject" yaml:"cache_subject"`
	DistributedCacheEnabled  bool          `json:"distributed_cache_enabled" yaml:"distributed_cache_enabled"`
	RetrievalTimeout         time.Duration `json:"retrieval_timeout" yaml:"retrieval_timeout"`
	CacheTTL                 time.Duration `json:"cache_ttl" yaml:"cache_ttl"`
	MaxCacheSize             int           `json:"max_cache_size" yaml:"max_cache_size"`
	EnableReplicationFactor  int           `json:"enable_replication_factor" yaml:"enable_replication_factor"`
	ConsistencyLevel         string        `json:"consistency_level" yaml:"consistency_level"` // STRONG, EVENTUAL
}

// DefaultNATSRetrieverConfig returns default NATS retriever configuration
func DefaultNATSRetrieverConfig() *NATSRetrieverConfig {
	return &NATSRetrieverConfig{
		RetrievalRequestSubject:  "scheduler.retrieval.request",
		RetrievalResponseSubject: "scheduler.retrieval.response",
		CacheSubject:            "scheduler.cache",
		DistributedCacheEnabled: true,
		RetrievalTimeout:        30 * time.Second,
		CacheTTL:                5 * time.Minute,
		MaxCacheSize:            1000,
		EnableReplicationFactor: 1,
		ConsistencyLevel:        "EVENTUAL",
	}
}

// NATSSchedulerRetriever handles NATS-based distributed memory retrieval and caching
type NATSSchedulerRetriever struct {
	*BaseSchedulerModule
	mu                      sync.RWMutex
	config                  *NATSRetrieverConfig
	conn                    *nats.Conn
	js                      jetstream.JetStream
	
	// Core retrieval components
	chatLLM                 interfaces.LLM
	memCube                 interface{} // Will be GeneralMemCube interface when available
	
	// Distributed caching
	localCache              map[string]*DistributedRetrievalCache
	cacheRequests           chan *CacheRequest
	cacheResponses          chan *CacheResponse
	
	// Request/Response tracking
	pendingRequests         map[string]*RetrievalRequest
	requestTimeout          map[string]*time.Timer
	
	// Statistics
	cacheHits               int64
	cacheMisses             int64
	distributedHits         int64
	totalRetrievals         int64
	avgRetrievalTime        time.Duration
	
	logger                  *logger.Logger
	startTime               time.Time
	nodeID                  string
}

// DistributedRetrievalCache represents a distributed cache entry
type DistributedRetrievalCache struct {
	Query       string         `json:"query"`
	Results     []*SearchResult `json:"results"`
	Timestamp   time.Time      `json:"timestamp"`
	TTL         time.Duration  `json:"ttl"`
	NodeID      string         `json:"node_id"`
	ReplicatedTo []string      `json:"replicated_to"`
	Version     int64          `json:"version"`
}

// IsExpired checks if the cache entry has expired
func (drc *DistributedRetrievalCache) IsExpired() bool {
	return time.Since(drc.Timestamp) > drc.TTL
}

// RetrievalRequest represents a distributed retrieval request
type RetrievalRequest struct {
	ID            string    `json:"id"`
	Query         string    `json:"query"`
	TopK          int       `json:"top_k"`
	Method        string    `json:"method"`
	RequestorID   string    `json:"requestor_id"`
	Timestamp     time.Time `json:"timestamp"`
	ReplySubject  string    `json:"reply_subject"`
}

// RetrievalResponse represents a distributed retrieval response
type RetrievalResponse struct {
	RequestID     string         `json:"request_id"`
	Results       []*SearchResult `json:"results"`
	Success       bool           `json:"success"`
	Error         string         `json:"error,omitempty"`
	ResponderID   string         `json:"responder_id"`
	ProcessingTime time.Duration `json:"processing_time"`
	Timestamp     time.Time      `json:"timestamp"`
}

// CacheRequest represents a cache lookup request
type CacheRequest struct {
	Query      string `json:"query"`
	ResponseCh chan *CacheResponse
}

// CacheResponse represents a cache lookup response
type CacheResponse struct {
	Results []*SearchResult `json:"results"`
	Hit     bool            `json:"hit"`
	Source  string          `json:"source"` // LOCAL, DISTRIBUTED, MISS
}

// RerankingRequest represents a distributed reranking request
type RerankingRequest struct {
	ID              string    `json:"id"`
	Query           string    `json:"query"`
	OriginalMemory  []string  `json:"original_memory"`
	NewCandidates   []*SearchResult `json:"new_candidates"`
	TopK            int       `json:"top_k"`
	TopN            int       `json:"top_n"`
	RequestorID     string    `json:"requestor_id"`
	Timestamp       time.Time `json:"timestamp"`
	ReplySubject    string    `json:"reply_subject"`
}

// RerankingResponse represents a distributed reranking response
type RerankingResponse struct {
	RequestID       string        `json:"request_id"`
	RerankedMemory  []string      `json:"reranked_memory"`
	Success         bool          `json:"success"`
	Error           string        `json:"error,omitempty"`
	ResponderID     string        `json:"responder_id"`
	ProcessingTime  time.Duration `json:"processing_time"`
	Timestamp       time.Time     `json:"timestamp"`
}

// NewNATSSchedulerRetriever creates a new NATS-based scheduler retriever
func NewNATSSchedulerRetriever(conn *nats.Conn, js jetstream.JetStream, chatLLM interfaces.LLM, config *NATSRetrieverConfig) *NATSSchedulerRetriever {
	if config == nil {
		config = DefaultNATSRetrieverConfig()
	}
	
	// Generate unique node ID
	nodeID := fmt.Sprintf("retriever-node-%d", time.Now().UnixNano())
	
	retriever := &NATSSchedulerRetriever{
		BaseSchedulerModule: NewBaseSchedulerModule(),
		config:              config,
		conn:                conn,
		js:                  js,
		chatLLM:             chatLLM,
		localCache:          make(map[string]*DistributedRetrievalCache),
		cacheRequests:       make(chan *CacheRequest, 100),
		cacheResponses:      make(chan *CacheResponse, 100),
		pendingRequests:     make(map[string]*RetrievalRequest),
		requestTimeout:      make(map[string]*time.Timer),
		cacheHits:           0,
		cacheMisses:         0,
		distributedHits:     0,
		totalRetrievals:     0,
		avgRetrievalTime:    0,
		logger:              logger.GetLogger("nats-retriever"),
		startTime:           time.Now(),
		nodeID:              nodeID,
	}
	
	// Initialize default templates
	if err := retriever.InitializeDefaultTemplates(); err != nil {
		retriever.logger.Error("Failed to initialize templates", "error", err)
	}
	
	return retriever
}

// InitializeRetrievalServices sets up NATS services for distributed retrieval
func (r *NATSSchedulerRetriever) InitializeRetrievalServices() error {
	if r.conn == nil {
		return fmt.Errorf("NATS connection not initialized")
	}
	
	// Subscribe to retrieval requests
	if err := r.subscribeToRetrievalRequests(); err != nil {
		return fmt.Errorf("failed to subscribe to retrieval requests: %w", err)
	}
	
	// Subscribe to cache requests if distributed cache is enabled
	if r.config.DistributedCacheEnabled {
		if err := r.subscribeToDistributedCache(); err != nil {
			r.logger.Warn("Failed to subscribe to distributed cache", "error", err)
		}
	}
	
	// Start cache management goroutines
	go r.cacheManager()
	go r.cacheCleanup()
	
	r.logger.Info("NATS retrieval services initialized",
		"node_id", r.nodeID,
		"distributed_cache", r.config.DistributedCacheEnabled,
		"cache_ttl", r.config.CacheTTL)
	
	return nil
}

// subscribeToRetrievalRequests subscribes to distributed retrieval requests
func (r *NATSSchedulerRetriever) subscribeToRetrievalRequests() error {
	_, err := r.conn.Subscribe(r.config.RetrievalRequestSubject, func(msg *nats.Msg) {
		var request RetrievalRequest
		if err := json.Unmarshal(msg.Data, &request); err != nil {
			r.logger.Error("Failed to unmarshal retrieval request", "error", err)
			return
		}
		
		// Don't process our own requests
		if request.RequestorID == r.nodeID {
			return
		}
		
		r.logger.Debug("Processing retrieval request",
			"id", request.ID,
			"query", request.Query,
			"requestor", request.RequestorID)
		
		// Process request asynchronously
		go r.processRetrievalRequest(&request, msg.Reply)
	})
	
	if err != nil {
		return fmt.Errorf("failed to subscribe to retrieval requests: %w", err)
	}
	
	return nil
}

// subscribeToDistributedCache subscribes to distributed cache operations
func (r *NATSSchedulerRetriever) subscribeToDistributedCache() error {
	// Subscribe to cache invalidation events
	_, err := r.conn.Subscribe(r.config.CacheSubject+".invalidate", func(msg *nats.Msg) {
		var invalidation struct {
			Query  string `json:"query"`
			NodeID string `json:"node_id"`
		}
		
		if err := json.Unmarshal(msg.Data, &invalidation); err != nil {
			r.logger.Error("Failed to unmarshal cache invalidation", "error", err)
			return
		}
		
		// Don't invalidate based on our own actions
		if invalidation.NodeID == r.nodeID {
			return
		}
		
		r.mu.Lock()
		delete(r.localCache, invalidation.Query)
		r.mu.Unlock()
		
		r.logger.Debug("Cache invalidated", "query", invalidation.Query, "by_node", invalidation.NodeID)
	})
	
	if err != nil {
		return fmt.Errorf("failed to subscribe to cache invalidation: %w", err)
	}
	
	// Subscribe to cache replication
	_, err = r.conn.Subscribe(r.config.CacheSubject+".replicate", func(msg *nats.Msg) {
		var cacheEntry DistributedRetrievalCache
		if err := json.Unmarshal(msg.Data, &cacheEntry); err != nil {
			r.logger.Error("Failed to unmarshal cache replication", "error", err)
			return
		}
		
		// Don't replicate our own cache entries
		if cacheEntry.NodeID == r.nodeID {
			return
		}
		
		// Store replicated cache entry if we have space
		r.mu.Lock()
		if len(r.localCache) < r.config.MaxCacheSize {
			r.localCache[cacheEntry.Query] = &cacheEntry
		}
		r.mu.Unlock()
		
		r.logger.Debug("Cache replicated", "query", cacheEntry.Query, "from_node", cacheEntry.NodeID)
	})
	
	return err
}

// processRetrievalRequest processes a distributed retrieval request
func (r *NATSSchedulerRetriever) processRetrievalRequest(request *RetrievalRequest, replySubject string) {
	startTime := time.Now()
	
	// Perform local search
	results, err := r.performLocalSearch(request.Query, request.TopK, request.Method)
	
	response := &RetrievalResponse{
		RequestID:      request.ID,
		ResponderID:    r.nodeID,
		ProcessingTime: time.Since(startTime),
		Timestamp:      time.Now(),
	}
	
	if err != nil {
		response.Success = false
		response.Error = err.Error()
	} else {
		response.Success = true
		response.Results = results
	}
	
	// Send response
	data, err := json.Marshal(response)
	if err != nil {
		r.logger.Error("Failed to marshal retrieval response", "error", err)
		return
	}
	
	if err := r.conn.Publish(replySubject, data); err != nil {
		r.logger.Error("Failed to publish retrieval response", "error", err)
		return
	}
	
	r.logger.Debug("Retrieval response sent",
		"request_id", request.ID,
		"results_count", len(response.Results),
		"processing_time", response.ProcessingTime)
}

// performLocalSearch performs search using local resources
func (r *NATSSchedulerRetriever) performLocalSearch(query string, topK int, method string) ([]*SearchResult, error) {
	if r.memCube == nil {
		return nil, fmt.Errorf("memory cube not initialized")
	}
	
	// Check local cache first
	if cached := r.getCachedResults(query); cached != nil && !cached.IsExpired() {
		r.mu.Lock()
		r.cacheHits++
		r.mu.Unlock()
		return cached.Results, nil
	}
	
	r.mu.Lock()
	r.cacheMisses++
	r.mu.Unlock()
	
	r.logger.Debug("Performing local search", "query", query, "top_k", topK, "method", method)
	
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
		return nil, fmt.Errorf("local search failed: %w", err)
	}
	
	// Cache results locally and potentially distribute
	r.cacheResults(query, results)
	
	return results, nil
}

// Search performs distributed search operation
func (r *NATSSchedulerRetriever) Search(query string, topK int, method string) ([]*SearchResult, error) {
	startTime := time.Now()
	defer func() {
		r.mu.Lock()
		r.totalRetrievals++
		r.avgRetrievalTime = (r.avgRetrievalTime*time.Duration(r.totalRetrievals-1) + time.Since(startTime)) / time.Duration(r.totalRetrievals)
		r.mu.Unlock()
	}()
	
	// First try local search (includes local cache)
	results, err := r.performLocalSearch(query, topK, method)
	if err == nil && len(results) > 0 {
		return results, nil
	}
	
	// If local search fails or returns no results, try distributed search
	distributedResults, err := r.performDistributedSearch(query, topK, method)
	if err != nil {
		r.logger.Warn("Distributed search failed, using local results", "error", err)
		return results, nil // Return local results even if not ideal
	}
	
	r.mu.Lock()
	r.distributedHits++
	r.mu.Unlock()
	
	// Merge and deduplicate results
	mergedResults := r.mergeSearchResults(results, distributedResults, topK)
	
	// Cache merged results
	r.cacheResults(query, mergedResults)
	
	return mergedResults, nil
}

// performDistributedSearch performs search across the distributed cluster
func (r *NATSSchedulerRetriever) performDistributedSearch(query string, topK int, method string) ([]*SearchResult, error) {
	requestID := fmt.Sprintf("req_%d_%s", time.Now().UnixNano(), r.nodeID)
	replySubject := fmt.Sprintf("%s.reply.%s", r.config.RetrievalResponseSubject, requestID)
	
	// Create retrieval request
	request := &RetrievalRequest{
		ID:           requestID,
		Query:        query,
		TopK:         topK,
		Method:       method,
		RequestorID:  r.nodeID,
		Timestamp:    time.Now(),
		ReplySubject: replySubject,
	}
	
	// Subscribe to responses
	responses := make(chan *RetrievalResponse, 10)
	sub, err := r.conn.Subscribe(replySubject, func(msg *nats.Msg) {
		var response RetrievalResponse
		if err := json.Unmarshal(msg.Data, &response); err != nil {
			r.logger.Error("Failed to unmarshal retrieval response", "error", err)
			return
		}
		
		select {
		case responses <- &response:
		default:
			r.logger.Warn("Response channel full, dropping response")
		}
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to responses: %w", err)
	}
	defer sub.Unsubscribe()
	
	// Publish request
	data, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	if err := r.conn.Publish(r.config.RetrievalRequestSubject, data); err != nil {
		return nil, fmt.Errorf("failed to publish request: %w", err)
	}
	
	// Collect responses with timeout
	ctx, cancel := context.WithTimeout(context.Background(), r.config.RetrievalTimeout)
	defer cancel()
	
	var allResults []*SearchResult
	responseCount := 0
	
	for {
		select {
		case <-ctx.Done():
			r.logger.Debug("Distributed search timeout", "responses_received", responseCount)
			return allResults, nil
		case response := <-responses:
			responseCount++
			if response.Success {
				allResults = append(allResults, response.Results...)
			}
			
			// If we have enough responses or results, return early
			if responseCount >= 3 || len(allResults) >= topK*2 {
				return allResults, nil
			}
		}
	}
}

// mergeSearchResults merges and deduplicates search results from multiple sources
func (r *NATSSchedulerRetriever) mergeSearchResults(local, distributed []*SearchResult, topK int) []*SearchResult {
	seen := make(map[string]bool)
	merged := make([]*SearchResult, 0, len(local)+len(distributed))
	
	// Add local results first (higher priority)
	for _, result := range local {
		if !seen[result.ID] {
			seen[result.ID] = true
			merged = append(merged, result)
		}
	}
	
	// Add distributed results
	for _, result := range distributed {
		if !seen[result.ID] {
			seen[result.ID] = true
			merged = append(merged, result)
		}
	}
	
	// Sort by score and limit to topK
	if len(merged) > topK {
		// Simple sort by score (descending)
		for i := 0; i < len(merged)-1; i++ {
			for j := i + 1; j < len(merged); j++ {
				if merged[i].Score < merged[j].Score {
					merged[i], merged[j] = merged[j], merged[i]
				}
			}
		}
		merged = merged[:topK]
	}
	
	return merged
}

// ReplaceWorkingMemory performs distributed reranking and replaces working memory
func (r *NATSSchedulerRetriever) ReplaceWorkingMemory(
	originalMemory []string,
	newMemory []*SearchResult,
	topK, topN int,
	query string) ([]string, error) {
	
	if r.chatLLM == nil {
		return nil, fmt.Errorf("chat LLM not initialized")
	}
	
	// Try local reranking first
	localResult, err := r.performLocalReranking(originalMemory, newMemory, topK, topN, query)
	if err == nil {
		return localResult, nil
	}
	
	// If local reranking fails, try distributed reranking
	distributedResult, err := r.performDistributedReranking(originalMemory, newMemory, topK, topN, query)
	if err != nil {
		r.logger.Warn("Distributed reranking failed, using simple merge", "error", err)
		// Fall back to simple merge
		return r.simpleMergeMemory(originalMemory, newMemory, topK, topN), nil
	}
	
	return distributedResult, nil
}

// performLocalReranking performs reranking using local LLM
func (r *NATSSchedulerRetriever) performLocalReranking(
	originalMemory []string,
	newMemory []*SearchResult,
	topK, topN int,
	query string) ([]string, error) {
	
	// Combine original memory and new search results
	combinedMemory := make([]string, 0, len(originalMemory)+len(newMemory))
	combinedMemory = append(combinedMemory, originalMemory...)
	
	for _, result := range newMemory {
		combinedMemory = append(combinedMemory, result.Content)
	}
	
	// Remove duplicates while preserving order
	uniqueMemory := r.removeDuplicates(combinedMemory)
	
	// Rerank using LLM
	rerankedMemory, err := r.rerankMemory(query, uniqueMemory, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to rerank memory: %w", err)
	}
	
	// Return top N + top K items
	maxItems := r.min(len(rerankedMemory), topN+topK)
	result := rerankedMemory[:maxItems]
	
	r.logger.Debug("Local reranking completed",
		"original_count", len(originalMemory),
		"new_count", len(newMemory),
		"result_count", len(result))
	
	return result, nil
}

// performDistributedReranking performs reranking using distributed cluster
func (r *NATSSchedulerRetriever) performDistributedReranking(
	originalMemory []string,
	newMemory []*SearchResult,
	topK, topN int,
	query string) ([]string, error) {
	
	requestID := fmt.Sprintf("rerank_%d_%s", time.Now().UnixNano(), r.nodeID)
	replySubject := fmt.Sprintf("%s.rerank.reply.%s", r.config.RetrievalResponseSubject, requestID)
	
	// Create reranking request
	request := &RerankingRequest{
		ID:             requestID,
		Query:          query,
		OriginalMemory: originalMemory,
		NewCandidates:  newMemory,
		TopK:           topK,
		TopN:           topN,
		RequestorID:    r.nodeID,
		Timestamp:      time.Now(),
		ReplySubject:   replySubject,
	}
	
	// Subscribe to responses
	responses := make(chan *RerankingResponse, 5)
	sub, err := r.conn.Subscribe(replySubject, func(msg *nats.Msg) {
		var response RerankingResponse
		if err := json.Unmarshal(msg.Data, &response); err != nil {
			r.logger.Error("Failed to unmarshal reranking response", "error", err)
			return
		}
		
		select {
		case responses <- &response:
		default:
			r.logger.Warn("Reranking response channel full")
		}
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to reranking responses: %w", err)
	}
	defer sub.Unsubscribe()
	
	// Publish request
	data, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal reranking request: %w", err)
	}
	
	if err := r.conn.Publish(r.config.RetrievalRequestSubject+".rerank", data); err != nil {
		return nil, fmt.Errorf("failed to publish reranking request: %w", err)
	}
	
	// Wait for first successful response
	ctx, cancel := context.WithTimeout(context.Background(), r.config.RetrievalTimeout)
	defer cancel()
	
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("distributed reranking timeout")
	case response := <-responses:
		if response.Success {
			return response.RerankedMemory, nil
		}
		return nil, fmt.Errorf("reranking failed: %s", response.Error)
	}
}

// simpleMergeMemory performs simple memory merging without LLM reranking
func (r *NATSSchedulerRetriever) simpleMergeMemory(
	originalMemory []string,
	newMemory []*SearchResult,
	topK, topN int) []string {
	
	combined := make([]string, 0, len(originalMemory)+len(newMemory))
	combined = append(combined, originalMemory...)
	
	for _, result := range newMemory {
		combined = append(combined, result.Content)
	}
	
	// Remove duplicates and limit size
	unique := r.removeDuplicates(combined)
	maxItems := r.min(len(unique), topK+topN)
	
	return unique[:maxItems]
}

// cacheManager manages the local cache
func (r *NATSSchedulerRetriever) cacheManager() {
	for request := range r.cacheRequests {
		r.mu.RLock()
		cached, exists := r.localCache[request.Query]
		r.mu.RUnlock()
		
		response := &CacheResponse{
			Hit: false,
			Source: "MISS",
		}
		
		if exists && !cached.IsExpired() {
			response.Hit = true
			response.Results = cached.Results
			if cached.NodeID == r.nodeID {
				response.Source = "LOCAL"
			} else {
				response.Source = "DISTRIBUTED"
			}
		}
		
		select {
		case request.ResponseCh <- response:
		default:
			r.logger.Warn("Cache response channel full")
		}
	}
}

// cacheCleanup periodically cleans up expired cache entries
func (r *NATSSchedulerRetriever) cacheCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		r.mu.Lock()
		for query, cached := range r.localCache {
			if cached.IsExpired() {
				delete(r.localCache, query)
			}
		}
		r.mu.Unlock()
	}
}

// getCachedResults retrieves cached search results
func (r *NATSSchedulerRetriever) getCachedResults(query string) *DistributedRetrievalCache {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if cached, exists := r.localCache[query]; exists {
		return cached
	}
	return nil
}

// cacheResults stores search results in cache and optionally distributes them
func (r *NATSSchedulerRetriever) cacheResults(query string, results []*SearchResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Check cache size limit
	if len(r.localCache) >= r.config.MaxCacheSize {
		// Remove oldest entry
		var oldestQuery string
		var oldestTime time.Time
		for q, cache := range r.localCache {
			if oldestQuery == "" || cache.Timestamp.Before(oldestTime) {
				oldestQuery = q
				oldestTime = cache.Timestamp
			}
		}
		if oldestQuery != "" {
			delete(r.localCache, oldestQuery)
		}
	}
	
	cache := &DistributedRetrievalCache{
		Query:     query,
		Results:   results,
		Timestamp: time.Now(),
		TTL:       r.config.CacheTTL,
		NodeID:    r.nodeID,
		Version:   time.Now().UnixNano(),
	}
	
	r.localCache[query] = cache
	
	// Distribute cache entry if enabled
	if r.config.DistributedCacheEnabled && r.config.EnableReplicationFactor > 1 {
		go r.distributeCacheEntry(cache)
	}
}

// distributeCacheEntry distributes cache entry to other nodes
func (r *NATSSchedulerRetriever) distributeCacheEntry(cache *DistributedRetrievalCache) {
	data, err := json.Marshal(cache)
	if err != nil {
		r.logger.Error("Failed to marshal cache entry for distribution", "error", err)
		return
	}
	
	if err := r.conn.Publish(r.config.CacheSubject+".replicate", data); err != nil {
		r.logger.Error("Failed to distribute cache entry", "error", err)
	}
}

// SetMemCube sets the memory cube for retrieval operations
func (r *NATSSchedulerRetriever) SetMemCube(memCube interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.memCube = memCube
}

// GetMemCube returns the current memory cube
func (r *NATSSchedulerRetriever) GetMemCube() interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.memCube
}

// ClearCache clears all cached results
func (r *NATSSchedulerRetriever) ClearCache() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.localCache = make(map[string]*DistributedRetrievalCache)
}

// GetCacheStats returns cache statistics
func (r *NATSSchedulerRetriever) GetCacheStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	expiredCount := 0
	distributedCount := 0
	for _, cache := range r.localCache {
		if cache.IsExpired() {
			expiredCount++
		}
		if cache.NodeID != r.nodeID {
			distributedCount++
		}
	}
	
	hitRate := 0.0
	if r.cacheHits+r.cacheMisses > 0 {
		hitRate = float64(r.cacheHits) / float64(r.cacheHits+r.cacheMisses) * 100
	}
	
	return map[string]interface{}{
		"total_entries":      len(r.localCache),
		"expired_entries":    expiredCount,
		"distributed_entries": distributedCount,
		"local_entries":      len(r.localCache) - distributedCount,
		"cache_hits":         r.cacheHits,
		"cache_misses":       r.cacheMisses,
		"distributed_hits":   r.distributedHits,
		"hit_rate_percent":   hitRate,
		"total_retrievals":   r.totalRetrievals,
		"avg_retrieval_time": r.avgRetrievalTime,
		"max_cache_size":     r.config.MaxCacheSize,
	}
}

// Helper methods from original retriever

// performTextMemorySearch performs search using text memory method
func (r *NATSSchedulerRetriever) performTextMemorySearch(query string, topK int) ([]*SearchResult, error) {
	r.logger.Debug("Performing text memory search", "query", query, "top_k", topK)
	
	results := make([]*SearchResult, 0)
	
	// Example placeholder results
	for i := 0; i < r.min(topK, 3); i++ {
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
func (r *NATSSchedulerRetriever) performTreeTextMemorySearch(query string, topK int) ([]*SearchResult, error) {
	r.logger.Debug("Performing tree text memory search", "query", query, "top_k", topK)
	
	results := make([]*SearchResult, 0)
	
	// Simulate LongTermMemory results
	for i := 0; i < r.min(topK/2, 2); i++ {
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
	for i := 0; i < r.min(topK/2, 2); i++ {
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

// rerankMemory uses LLM to rerank memory items based on relevance
func (r *NATSSchedulerRetriever) rerankMemory(query string, currentOrder []string, stagingBuffer []string) ([]string, error) {
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

// extractJSONDict extracts JSON dictionary from LLM response
func (r *NATSSchedulerRetriever) extractJSONDict(response string) (map[string]interface{}, error) {
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
func (r *NATSSchedulerRetriever) min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (r *NATSSchedulerRetriever) removeDuplicates(slice []string) []string {
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