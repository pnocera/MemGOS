// Package textual provides tree-based textual memory implementation
package textual

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/errors"
	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// TreeTextMemory implements tree-based textual memory with graph organization
type TreeTextMemory struct {
	*BaseTextualMemory
	config           *config.MemoryConfig
	memories         map[string]*types.TextualMemoryItem
	graph            *MemoryGraph
	workingMemory    map[string]*types.TextualMemoryItem
	longTermMemory   map[string]*types.TextualMemoryItem
	userMemory       map[string]*types.TextualMemoryItem
	internetMemory   map[string]*types.TextualMemoryItem
	memMu            sync.RWMutex
	internetRetriever interfaces.InternetRetriever
	memoryManager    *MemoryManager
	searcher         *TreeSearcher
}

// MemoryGraph represents the memory graph structure
type MemoryGraph struct {
	nodes map[string]*GraphNode
	edges map[string][]*GraphEdge
	mu    sync.RWMutex
}

// MemoryManager handles memory organization and lifecycle
type MemoryManager struct {
	graph      *MemoryGraph
	embedder   interfaces.Embedder
	llm        interfaces.LLM
	logger     interfaces.Logger
	extractLLM interfaces.LLM
}

// TreeSearcher provides advanced search capabilities
type TreeSearcher struct {
	graph               *MemoryGraph
	embedder            interfaces.Embedder
	dispatcherLLM       interfaces.LLM
	internetRetriever   interfaces.InternetRetriever
	logger              interfaces.Logger
	reasoner            *MemoryReasoner
	reranker            *MemoryReranker
	taskGoalParser      *TaskGoalParser
}

// MemoryReasoner provides reasoning capabilities
type MemoryReasoner struct {
	llm    interfaces.LLM
	logger interfaces.Logger
}

// MemoryReranker provides reranking capabilities
type MemoryReranker struct {
	llm    interfaces.LLM
	logger interfaces.Logger
}

// TaskGoalParser parses user queries into structured goals
type TaskGoalParser struct {
	llm    interfaces.LLM
	logger interfaces.Logger
}

// NewTreeTextMemory creates a new tree-based textual memory
func NewTreeTextMemory(cfg *config.MemoryConfig, embedder interfaces.Embedder, vectorDB interfaces.VectorDB, graphDB interfaces.GraphDB, llm interfaces.LLM, logger interfaces.Logger, metrics interfaces.Metrics) (*TreeTextMemory, error) {
	base := NewBaseTextualMemory(embedder, vectorDB, graphDB, llm, logger, metrics)
	
	graph := &MemoryGraph{
		nodes: make(map[string]*GraphNode),
		edges: make(map[string][]*GraphEdge),
	}
	
	memoryManager := &MemoryManager{
		graph:    graph,
		embedder: embedder,
		llm:      llm,
		logger:   logger,
	}
	
	searcher := &TreeSearcher{
		graph:         graph,
		embedder:      embedder,
		dispatcherLLM: llm,
		logger:        logger,
		reasoner:      &MemoryReasoner{llm: llm, logger: logger},
		reranker:      &MemoryReranker{llm: llm, logger: logger},
		taskGoalParser: &TaskGoalParser{llm: llm, logger: logger},
	}
	
	ttm := &TreeTextMemory{
		BaseTextualMemory: base,
		config:           cfg,
		memories:         make(map[string]*types.TextualMemoryItem),
		graph:            graph,
		workingMemory:    make(map[string]*types.TextualMemoryItem),
		longTermMemory:   make(map[string]*types.TextualMemoryItem),
		userMemory:       make(map[string]*types.TextualMemoryItem),
		internetMemory:   make(map[string]*types.TextualMemoryItem),
		memoryManager:    memoryManager,
		searcher:         searcher,
	}
	
	return ttm, nil
}

// SetInternetRetriever sets the internet retriever
func (ttm *TreeTextMemory) SetInternetRetriever(retriever interfaces.InternetRetriever) {
	ttm.internetRetriever = retriever
	ttm.searcher.internetRetriever = retriever
}

// Add adds memories to the tree structure
func (ttm *TreeTextMemory) Add(ctx context.Context, memories []*types.TextualMemoryItem) error {
	if ttm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	start := time.Now()
	defer func() {
		if ttm.metrics != nil {
			ttm.metrics.Timer("tree_memory_add_duration", float64(time.Since(start).Milliseconds()), nil)
		}
	}()
	
	ttm.memMu.Lock()
	defer ttm.memMu.Unlock()
	
	// Add memories to the tree structure
	for _, memory := range memories {
		if memory.ID == "" {
			memory.ID = ttm.generateID()
		}
		
		if memory.CreatedAt.IsZero() {
			memory.CreatedAt = time.Now()
		}
		memory.UpdatedAt = time.Now()
		
		// Store in main memory map
		ttm.memories[memory.ID] = memory
		
		// Categorize memory based on metadata
		memoryType := ttm.categorizeMemory(memory)
		switch memoryType {
		case MemoryTypeWorking:
			ttm.workingMemory[memory.ID] = memory
		case MemoryTypeLongTerm:
			ttm.longTermMemory[memory.ID] = memory
		case MemoryTypeUser:
			ttm.userMemory[memory.ID] = memory
		case MemoryTypeInternet:
			ttm.internetMemory[memory.ID] = memory
		}
		
		// Add to graph
		if err := ttm.addToGraph(ctx, memory); err != nil {
			ttm.logger.Error("Failed to add memory to graph", err, map[string]interface{}{
				"memory_id": memory.ID,
			})
		}
	}
	
	// Update search index
	if err := ttm.updateSearchIndex(memories); err != nil {
		ttm.logger.Error("Failed to update search index", err, map[string]interface{}{})
	}
	
	ttm.logger.Info("Added tree memories", map[string]interface{}{
		"count": len(memories),
	})
	
	if ttm.metrics != nil {
		ttm.metrics.Counter("tree_memory_add_count", float64(len(memories)), nil)
	}
	
	return nil
}

// Search performs advanced search with multiple modes
func (ttm *TreeTextMemory) Search(ctx context.Context, query string, topK int, mode SearchMode, memoryType MemoryType) ([]*types.TextualMemoryItem, error) {
	if ttm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	start := time.Now()
	defer func() {
		if ttm.metrics != nil {
			ttm.metrics.Timer("tree_memory_search_duration", float64(time.Since(start).Milliseconds()), nil)
		}
	}()
	
	searchRequest := &SearchRequest{
		Query:      query,
		TopK:       topK,
		Mode:       mode,
		MemoryType: memoryType,
	}
	
	switch mode {
	case SearchModeFast:
		return ttm.fastSearch(ctx, searchRequest)
	case SearchModeFine:
		return ttm.fineSearch(ctx, searchRequest)
	case SearchModeSemantic:
		return ttm.semanticSearch(ctx, searchRequest)
	default:
		return ttm.fastSearch(ctx, searchRequest)
	}
}

// fastSearch performs fast search using indexed approaches
func (ttm *TreeTextMemory) fastSearch(ctx context.Context, req *SearchRequest) ([]*types.TextualMemoryItem, error) {
	ttm.memMu.RLock()
	defer ttm.memMu.RUnlock()
	
	// Get candidate memories based on memory type
	candidates := ttm.getCandidateMemories(req.MemoryType)
	
	// Simple TF-IDF based search
	results := ttm.tfIdfSearch(req.Query, candidates, req.TopK)
	
	// Convert to memory items
	memories := make([]*types.TextualMemoryItem, len(results))
	for i, result := range results {
		memories[i] = result.Memory
	}
	
	ttm.logger.Info("Fast search completed", map[string]interface{}{
		"query":   req.Query,
		"results": len(memories),
		"mode":    req.Mode,
	})
	
	return memories, nil
}

// fineSearch performs fine-grained search with LLM reasoning
func (ttm *TreeTextMemory) fineSearch(ctx context.Context, req *SearchRequest) ([]*types.TextualMemoryItem, error) {
	ttm.memMu.RLock()
	defer ttm.memMu.RUnlock()
	
	// Parse query into structured goals
	goals, err := ttm.searcher.taskGoalParser.ParseQuery(ctx, req.Query)
	if err != nil {
		ttm.logger.Error("Failed to parse query", err, map[string]interface{}{})
		// Fall back to fast search
		return ttm.fastSearch(ctx, req)
	}
	
	// Get initial candidates
	candidates := ttm.getCandidateMemories(req.MemoryType)
	
	// Perform multi-stage search
	stage1Results := ttm.tfIdfSearch(req.Query, candidates, req.TopK*3) // Get more candidates
	
	// Use LLM for reranking
	rerankedResults, err := ttm.searcher.reranker.Rerank(ctx, stage1Results, goals)
	if err != nil {
		ttm.logger.Error("Failed to rerank results", err, map[string]interface{}{})
		// Use original results
		rerankedResults = stage1Results
	}
	
	// Limit to topK
	if len(rerankedResults) > req.TopK {
		rerankedResults = rerankedResults[:req.TopK]
	}
	
	// Apply reasoning
	finalResults, err := ttm.searcher.reasoner.Reason(ctx, rerankedResults, goals)
	if err != nil {
		ttm.logger.Error("Failed to reason about results", err, map[string]interface{}{})
		finalResults = rerankedResults
	}
	
	// Convert to memory items
	memories := make([]*types.TextualMemoryItem, len(finalResults))
	for i, result := range finalResults {
		memories[i] = result.Memory
	}
	
	// Include internet retrieval if enabled
	if ttm.internetRetriever != nil {
		internetResults, err := ttm.retrieveFromInternet(ctx, req.Query, req.TopK/2)
		if err != nil {
			ttm.logger.Error("Failed to retrieve from internet", err, map[string]interface{}{})
		} else {
			memories = append(memories, internetResults...)
		}
	}
	
	ttm.logger.Info("Fine search completed", map[string]interface{}{
		"query":   req.Query,
		"results": len(memories),
		"mode":    req.Mode,
	})
	
	return memories, nil
}

// semanticSearch performs semantic search using embeddings
func (ttm *TreeTextMemory) semanticSearch(ctx context.Context, req *SearchRequest) ([]*types.TextualMemoryItem, error) {
	if ttm.embedder == nil {
		return ttm.fastSearch(ctx, req)
	}
	
	// Generate query embedding
	queryEmbedding, err := ttm.embedder.Embed(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}
	
	ttm.memMu.RLock()
	defer ttm.memMu.RUnlock()
	
	// Get candidate memories
	candidates := ttm.getCandidateMemories(req.MemoryType)
	
	// Calculate semantic similarity
	results := ttm.semanticSimilaritySearch([]float32(queryEmbedding), candidates, req.TopK)
	
	// Convert to memory items
	memories := make([]*types.TextualMemoryItem, len(results))
	for i, result := range results {
		memories[i] = result.Memory
	}
	
	ttm.logger.Info("Semantic search completed", map[string]interface{}{
		"query":   req.Query,
		"results": len(memories),
		"mode":    req.Mode,
	})
	
	return memories, nil
}

// GetRelevantSubgraph finds relevant subgraphs based on query
func (ttm *TreeTextMemory) GetRelevantSubgraph(ctx context.Context, query string, topK int, depth int) (map[string]interface{}, error) {
	if ttm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	// Generate query embedding
	queryEmbedding, err := ttm.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}
	
	ttm.graph.mu.RLock()
	defer ttm.graph.mu.RUnlock()
	
	// Find most similar nodes
	similarNodes := ttm.findSimilarNodes([]float32(queryEmbedding), topK)
	
	// Build subgraph
	subgraph := ttm.buildSubgraph(similarNodes, depth)
	
	result := map[string]interface{}{
		"core_id": "",
		"nodes":   subgraph.Nodes,
		"edges":   subgraph.Edges,
	}
	
	if len(subgraph.Nodes) > 0 {
		result["core_id"] = subgraph.Nodes[0].ID
	}
	
	return result, nil
}

// GetWorkingMemory returns current working memory
func (ttm *TreeTextMemory) GetWorkingMemory(ctx context.Context) ([]*types.TextualMemoryItem, error) {
	if ttm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	ttm.memMu.RLock()
	defer ttm.memMu.RUnlock()
	
	memories := make([]*types.TextualMemoryItem, 0, len(ttm.workingMemory))
	for _, memory := range ttm.workingMemory {
		memories = append(memories, memory)
	}
	
	// Sort by updated time (most recent first)
	sort.Slice(memories, func(i, j int) bool {
		return memories[i].UpdatedAt.After(memories[j].UpdatedAt)
	})
	
	return memories, nil
}

// ReplaceWorkingMemory replaces working memory with new memories
func (ttm *TreeTextMemory) ReplaceWorkingMemory(ctx context.Context, memories []*types.TextualMemoryItem) error {
	if ttm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	ttm.memMu.Lock()
	defer ttm.memMu.Unlock()
	
	// Clear current working memory
	ttm.workingMemory = make(map[string]*types.TextualMemoryItem)
	
	// Add new memories
	for _, memory := range memories {
		if memory.ID == "" {
			memory.ID = ttm.generateID()
		}
		memory.UpdatedAt = time.Now()
		ttm.workingMemory[memory.ID] = memory
		ttm.memories[memory.ID] = memory
	}
	
	ttm.logger.Info("Replaced working memory", map[string]interface{}{
		"count": len(memories),
	})
	
	return nil
}

// GetMemorySize returns memory size statistics
func (ttm *TreeTextMemory) GetMemorySize(ctx context.Context) (map[string]int, error) {
	if ttm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	ttm.memMu.RLock()
	defer ttm.memMu.RUnlock()
	
	return map[string]int{
		"total":         len(ttm.memories),
		"working":       len(ttm.workingMemory),
		"long_term":     len(ttm.longTermMemory),
		"user":          len(ttm.userMemory),
		"internet":      len(ttm.internetMemory),
		"graph_nodes":   len(ttm.graph.nodes),
		"graph_edges":   ttm.countGraphEdges(),
		"index_size":    len(ttm.searchIndex.invertedIndex),
		"embeddings":    len(ttm.searchIndex.embeddings),
	}, nil
}

// Get retrieves a memory by ID
func (ttm *TreeTextMemory) Get(ctx context.Context, id string) (*types.TextualMemoryItem, error) {
	if ttm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	ttm.memMu.RLock()
	defer ttm.memMu.RUnlock()
	
	memory, exists := ttm.memories[id]
	if !exists {
		return nil, errors.NewMemoryNotFoundError(id)
	}
	
	return memory, nil
}

// GetByIDs retrieves multiple memories by IDs
func (ttm *TreeTextMemory) GetByIDs(ctx context.Context, ids []string) ([]*types.TextualMemoryItem, error) {
	if ttm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	ttm.memMu.RLock()
	defer ttm.memMu.RUnlock()
	
	result := make([]*types.TextualMemoryItem, 0, len(ids))
	for _, id := range ids {
		if memory, exists := ttm.memories[id]; exists {
			result = append(result, memory)
		}
	}
	
	return result, nil
}

// GetAll retrieves all memories
func (ttm *TreeTextMemory) GetAll(ctx context.Context) ([]*types.TextualMemoryItem, error) {
	if ttm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	ttm.memMu.RLock()
	defer ttm.memMu.RUnlock()
	
	result := make([]*types.TextualMemoryItem, 0, len(ttm.memories))
	for _, memory := range ttm.memories {
		result = append(result, memory)
	}
	
	return result, nil
}

// Update updates an existing memory
func (ttm *TreeTextMemory) Update(ctx context.Context, id string, memory *types.TextualMemoryItem) error {
	if ttm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	ttm.memMu.Lock()
	defer ttm.memMu.Unlock()
	
	if _, exists := ttm.memories[id]; !exists {
		return errors.NewMemoryNotFoundError(id)
	}
	
	memory.ID = id
	memory.UpdatedAt = time.Now()
	ttm.memories[id] = memory
	
	// Update in appropriate category
	memoryType := ttm.categorizeMemory(memory)
	switch memoryType {
	case MemoryTypeWorking:
		ttm.workingMemory[id] = memory
	case MemoryTypeLongTerm:
		ttm.longTermMemory[id] = memory
	case MemoryTypeUser:
		ttm.userMemory[id] = memory
	case MemoryTypeInternet:
		ttm.internetMemory[id] = memory
	}
	
	// Update graph
	if err := ttm.updateGraphNode(ctx, memory); err != nil {
		ttm.logger.Error("Failed to update graph node", err, map[string]interface{}{
			"memory_id": id,
		})
	}
	
	ttm.logger.Info("Updated tree memory", map[string]interface{}{
		"memory_id": id,
	})
	
	return nil
}

// Delete deletes memories by IDs
func (ttm *TreeTextMemory) Delete(ctx context.Context, ids []string) error {
	if ttm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	ttm.memMu.Lock()
	defer ttm.memMu.Unlock()
	
	deletedCount := 0
	for _, id := range ids {
		if _, exists := ttm.memories[id]; exists {
			delete(ttm.memories, id)
			delete(ttm.workingMemory, id)
			delete(ttm.longTermMemory, id)
			delete(ttm.userMemory, id)
			delete(ttm.internetMemory, id)
			
			// Remove from graph
			ttm.removeFromGraph(id)
			
			deletedCount++
		}
	}
	
	ttm.logger.Info("Deleted tree memories", map[string]interface{}{
		"count": deletedCount,
	})
	
	return nil
}

// DeleteAll deletes all memories
func (ttm *TreeTextMemory) DeleteAll(ctx context.Context) error {
	if ttm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	ttm.memMu.Lock()
	defer ttm.memMu.Unlock()
	
	count := len(ttm.memories)
	ttm.memories = make(map[string]*types.TextualMemoryItem)
	ttm.workingMemory = make(map[string]*types.TextualMemoryItem)
	ttm.longTermMemory = make(map[string]*types.TextualMemoryItem)
	ttm.userMemory = make(map[string]*types.TextualMemoryItem)
	ttm.internetMemory = make(map[string]*types.TextualMemoryItem)
	
	// Clear graph
	ttm.graph.nodes = make(map[string]*GraphNode)
	ttm.graph.edges = make(map[string][]*GraphEdge)
	
	// Clear search index
	ttm.searchIndex = SearchIndex{
		invertedIndex: make(map[string][]string),
		tfIndex:       make(map[string]map[string]float64),
		idfIndex:      make(map[string]float64),
		embeddings:    make(map[string][]float32),
		metadata:      make(map[string]map[string]interface{}),
	}
	
	ttm.logger.Info("Deleted all tree memories", map[string]interface{}{
		"count": count,
	})
	
	return nil
}

// Load loads memories from directory
func (ttm *TreeTextMemory) Load(ctx context.Context, dir string) error {
	if ttm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	filePath := filepath.Join(dir, ttm.config.MemoryFilename)
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		ttm.logger.Info("Memory file not found, starting with empty memories", map[string]interface{}{
			"file_path": filePath,
		})
		return nil
	}
	
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read memory file: %w", err)
	}
	
	// Parse JSON
	var exportData map[string]interface{}
	if err := json.Unmarshal(data, &exportData); err != nil {
		return fmt.Errorf("failed to parse memory file: %w", err)
	}
	
	// Load memories
	if err := ttm.loadFromExportData(exportData); err != nil {
		return fmt.Errorf("failed to load memories: %w", err)
	}
	
	ttm.logger.Info("Loaded tree memories", map[string]interface{}{
		"count":     len(ttm.memories),
		"file_path": filePath,
	})
	
	return nil
}

// Dump saves memories to directory
func (ttm *TreeTextMemory) Dump(ctx context.Context, dir string) error {
	if ttm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Export data
	exportData := ttm.exportToData()
	
	// Marshal to JSON
	data, err := json.MarshalIndent(exportData, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	
	// Write to file
	filePath := filepath.Join(dir, ttm.config.MemoryFilename)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write memory file: %w", err)
	}
	
	ttm.logger.Info("Dumped tree memories", map[string]interface{}{
		"count":     len(ttm.memories),
		"file_path": filePath,
	})
	
	return nil
}

// Drop deletes all memories and creates backup
func (ttm *TreeTextMemory) Drop(ctx context.Context, keepLastN int) error {
	if ttm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	// Create backup
	backupDir := fmt.Sprintf("/tmp/memos_backup_%s", time.Now().Format("20060102_150405"))
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}
	
	// Dump to backup
	if err := ttm.Dump(ctx, backupDir); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	
	// Clean up old backups
	if err := ttm.cleanupOldBackups(keepLastN); err != nil {
		ttm.logger.Error("Failed to clean up old backups", err, map[string]interface{}{})
	}
	
	// Delete all memories
	if err := ttm.DeleteAll(ctx); err != nil {
		return fmt.Errorf("failed to delete all memories: %w", err)
	}
	
	ttm.logger.Info("Dropped tree memories with backup", map[string]interface{}{
		"backup_dir": backupDir,
		"keep_last":  keepLastN,
	})
	
	return nil
}

// Helper methods

func (ttm *TreeTextMemory) generateID() string {
	return fmt.Sprintf("tree_%d", time.Now().UnixNano())
}

func (ttm *TreeTextMemory) categorizeMemory(memory *types.TextualMemoryItem) MemoryType {
	if memory.Metadata != nil {
		if memType, exists := memory.Metadata["memory_type"]; exists {
			if typeStr, ok := memType.(string); ok {
				return MemoryType(typeStr)
			}
		}
		
		// Check for internet retrieval indicators
		if _, exists := memory.Metadata["url"]; exists {
			return MemoryTypeInternet
		}
		
		// Check for user-specific indicators
		if _, exists := memory.Metadata["user_id"]; exists {
			return MemoryTypeUser
		}
	}
	
	// Default to working memory
	return MemoryTypeWorking
}

func (ttm *TreeTextMemory) getCandidateMemories(memoryType MemoryType) map[string]*types.TextualMemoryItem {
	switch memoryType {
	case MemoryTypeWorking:
		return ttm.workingMemory
	case MemoryTypeLongTerm:
		return ttm.longTermMemory
	case MemoryTypeUser:
		return ttm.userMemory
	case MemoryTypeInternet:
		return ttm.internetMemory
	default:
		return ttm.memories
	}
}

func (ttm *TreeTextMemory) tfIdfSearch(query string, candidates map[string]*types.TextualMemoryItem, topK int) []*SearchResult {
	// Simple TF-IDF implementation
	queryWords := strings.Fields(strings.ToLower(query))
	results := make([]*SearchResult, 0)
	
	for _, memory := range candidates {
		score := ttm.calculateTfIdfScore(memory.Memory, queryWords)
		if score > 0 {
			results = append(results, &SearchResult{
				Memory: memory,
				Score:  score,
				Source: "index",
			})
		}
	}
	
	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	
	// Limit to topK
	if len(results) > topK {
		results = results[:topK]
	}
	
	return results
}

func (ttm *TreeTextMemory) calculateTfIdfScore(text string, queryWords []string) float64 {
	textWords := strings.Fields(strings.ToLower(text))
	if len(textWords) == 0 {
		return 0
	}
	
	// Calculate TF for each query word
	wordCount := make(map[string]int)
	for _, word := range textWords {
		wordCount[word]++
	}
	
	score := 0.0
	for _, queryWord := range queryWords {
		tf := float64(wordCount[queryWord]) / float64(len(textWords))
		if tf > 0 {
			// Simple IDF approximation
			idf := math.Log(float64(len(ttm.memories)) / (1 + float64(len(ttm.searchIndex.invertedIndex[queryWord]))))
			score += tf * idf
		}
	}
	
	return score
}

func (ttm *TreeTextMemory) semanticSimilaritySearch(queryEmbedding []float32, candidates map[string]*types.TextualMemoryItem, topK int) []*SearchResult {
	results := make([]*SearchResult, 0)
	
	for id, memory := range candidates {
		if embedding, exists := ttm.searchIndex.embeddings[id]; exists {
			similarity := ttm.cosineSimilarity(queryEmbedding, embedding)
			if similarity > 0.1 { // Threshold
				results = append(results, &SearchResult{
					Memory: memory,
					Score:  float64(similarity),
					Source: "semantic",
				})
			}
		}
	}
	
	// Sort by similarity (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	
	// Limit to topK
	if len(results) > topK {
		results = results[:topK]
	}
	
	return results
}

func (ttm *TreeTextMemory) cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	
	var dotProduct, normA, normB float32
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	
	if normA == 0 || normB == 0 {
		return 0
	}
	
	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

func (ttm *TreeTextMemory) updateSearchIndex(memories []*types.TextualMemoryItem) error {
	for _, memory := range memories {
		// Update inverted index
		words := strings.Fields(strings.ToLower(memory.Memory))
		for _, word := range words {
			if ttm.searchIndex.invertedIndex[word] == nil {
				ttm.searchIndex.invertedIndex[word] = make([]string, 0)
			}
			ttm.searchIndex.invertedIndex[word] = append(ttm.searchIndex.invertedIndex[word], memory.ID)
		}
		
		// Generate and store embedding if embedder is available
		if ttm.embedder != nil {
			embedding, err := ttm.embedder.Embed(context.Background(), memory.Memory)
			if err == nil {
				ttm.searchIndex.embeddings[memory.ID] = embedding
			}
		}
		
		// Store metadata
		ttm.searchIndex.metadata[memory.ID] = memory.Metadata
	}
	
	return nil
}

func (ttm *TreeTextMemory) addToGraph(ctx context.Context, memory *types.TextualMemoryItem) error {
	ttm.graph.mu.Lock()
	defer ttm.graph.mu.Unlock()
	
	node := &GraphNode{
		ID:          memory.ID,
		Memory:      memory,
		Connections: make([]string, 0),
		Weight:      1.0,
		Metadata:    memory.Metadata,
	}
	
	ttm.graph.nodes[memory.ID] = node
	
	// Find connections to existing nodes
	if err := ttm.findAndCreateConnections(ctx, node); err != nil {
		return err
	}
	
	return nil
}

func (ttm *TreeTextMemory) findAndCreateConnections(ctx context.Context, node *GraphNode) error {
	// Simple connection logic based on semantic similarity
	if ttm.embedder == nil {
		return nil
	}
	
	nodeEmbedding, exists := ttm.searchIndex.embeddings[node.ID]
	if !exists {
		return nil
	}
	
	threshold := float32(0.7) // Similarity threshold for connections
	
	for id, embedding := range ttm.searchIndex.embeddings {
		if id == node.ID {
			continue
		}
		
		similarity := ttm.cosineSimilarity(nodeEmbedding, embedding)
		if similarity > threshold {
			edge := &GraphEdge{
				Source: node.ID,
				Target: id,
				Type:   "semantic",
				Weight: float64(similarity),
			}
			
			ttm.graph.edges[node.ID] = append(ttm.graph.edges[node.ID], edge)
			node.Connections = append(node.Connections, id)
			
			// Add reverse connection
			if targetNode, exists := ttm.graph.nodes[id]; exists {
				reverseEdge := &GraphEdge{
					Source: id,
					Target: node.ID,
					Type:   "semantic",
					Weight: float64(similarity),
				}
				ttm.graph.edges[id] = append(ttm.graph.edges[id], reverseEdge)
				targetNode.Connections = append(targetNode.Connections, node.ID)
			}
		}
	}
	
	return nil
}

func (ttm *TreeTextMemory) updateGraphNode(ctx context.Context, memory *types.TextualMemoryItem) error {
	ttm.graph.mu.Lock()
	defer ttm.graph.mu.Unlock()
	
	if node, exists := ttm.graph.nodes[memory.ID]; exists {
		node.Memory = memory
		node.Metadata = memory.Metadata
	}
	
	return nil
}

func (ttm *TreeTextMemory) removeFromGraph(id string) {
	ttm.graph.mu.Lock()
	defer ttm.graph.mu.Unlock()
	
	// Remove node
	delete(ttm.graph.nodes, id)
	
	// Remove edges
	delete(ttm.graph.edges, id)
	
	// Remove connections from other nodes
	for nodeID, edges := range ttm.graph.edges {
		newEdges := make([]*GraphEdge, 0)
		for _, edge := range edges {
			if edge.Target != id {
				newEdges = append(newEdges, edge)
			}
		}
		ttm.graph.edges[nodeID] = newEdges
	}
}

func (ttm *TreeTextMemory) findSimilarNodes(queryEmbedding []float32, topK int) []*GraphNode {
	type nodeScore struct {
		node  *GraphNode
		score float32
	}
	
	scores := make([]*nodeScore, 0)
	
	for id, embedding := range ttm.searchIndex.embeddings {
		if node, exists := ttm.graph.nodes[id]; exists {
			similarity := ttm.cosineSimilarity(queryEmbedding, embedding)
			scores = append(scores, &nodeScore{
				node:  node,
				score: similarity,
			})
		}
	}
	
	// Sort by similarity (descending)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})
	
	// Limit to topK
	if len(scores) > topK {
		scores = scores[:topK]
	}
	
	// Extract nodes
	nodes := make([]*GraphNode, len(scores))
	for i, score := range scores {
		nodes[i] = score.node
	}
	
	return nodes
}

func (ttm *TreeTextMemory) buildSubgraph(centerNodes []*GraphNode, depth int) *SubgraphResult {
	visitedNodes := make(map[string]*GraphNode)
	visitedEdges := make(map[string]*GraphEdge)
	
	// BFS to build subgraph
	queue := make([]*GraphNode, 0)
	nodeDepth := make(map[string]int)
	
	// Add center nodes
	for _, node := range centerNodes {
		visitedNodes[node.ID] = node
		queue = append(queue, node)
		nodeDepth[node.ID] = 0
	}
	
	// BFS traversal
	for len(queue) > 0 {
		currentNode := queue[0]
		queue = queue[1:]
		
		currentDepth := nodeDepth[currentNode.ID]
		if currentDepth >= depth {
			continue
		}
		
		// Add connected nodes
		for _, edge := range ttm.graph.edges[currentNode.ID] {
			edgeKey := fmt.Sprintf("%s->%s", edge.Source, edge.Target)
			visitedEdges[edgeKey] = edge
			
			if _, visited := visitedNodes[edge.Target]; !visited {
				if targetNode, exists := ttm.graph.nodes[edge.Target]; exists {
					visitedNodes[edge.Target] = targetNode
					queue = append(queue, targetNode)
					nodeDepth[edge.Target] = currentDepth + 1
				}
			}
		}
	}
	
	// Convert to slices
	nodes := make([]*GraphNode, 0, len(visitedNodes))
	for _, node := range visitedNodes {
		nodes = append(nodes, node)
	}
	
	edges := make([]*GraphEdge, 0, len(visitedEdges))
	for _, edge := range visitedEdges {
		edges = append(edges, edge)
	}
	
	return &SubgraphResult{
		CoreID: "",
		Nodes:  nodes,
		Edges:  edges,
		Score:  0.0,
	}
}

func (ttm *TreeTextMemory) countGraphEdges() int {
	count := 0
	for _, edges := range ttm.graph.edges {
		count += len(edges)
	}
	return count
}

func (ttm *TreeTextMemory) retrieveFromInternet(ctx context.Context, query string, topK int) ([]*types.TextualMemoryItem, error) {
	if ttm.internetRetriever == nil {
		return nil, nil
	}
	
	// Retrieve from internet using Search method
	internetResults, err := ttm.internetRetriever.Search(ctx, query, topK)
	if err != nil {
		return nil, err
	}
	
	// Convert to textual memory items
	memories := make([]*types.TextualMemoryItem, 0, len(internetResults))
	for _, result := range internetResults {
		memory := &types.TextualMemoryItem{
			ID:       ttm.generateID(),
			Memory:   result.Content,
			Metadata: map[string]interface{}{
				"url":           result.URL,
				"title":         result.Title,
				"retrieved_at":  result.RetrievedAt,
				"memory_type":   string(MemoryTypeInternet),
				"internet_score": result.Score,
			},
			CreatedAt: result.RetrievedAt,
			UpdatedAt: result.RetrievedAt,
		}
		memories = append(memories, memory)
		
		// Store in internet memory
		ttm.internetMemory[memory.ID] = memory
		ttm.memories[memory.ID] = memory
	}
	
	return memories, nil
}

func (ttm *TreeTextMemory) loadFromExportData(data map[string]interface{}) error {
	// Load nodes
	if nodesData, exists := data["nodes"]; exists {
		if nodesList, ok := nodesData.([]interface{}); ok {
			for _, nodeData := range nodesList {
				if nodeMap, ok := nodeData.(map[string]interface{}); ok {
					memory := &types.TextualMemoryItem{}
					
					// Parse node data
					if id, exists := nodeMap["id"]; exists {
						memory.ID = id.(string)
					}
					if memoryContent, exists := nodeMap["memory"]; exists {
						memory.Memory = memoryContent.(string)
					}
					if metadata, exists := nodeMap["metadata"]; exists {
						memory.Metadata = metadata.(map[string]interface{})
					}
					
					// Parse timestamps
					if createdAt, exists := nodeMap["created_at"]; exists {
						if timestamp, err := time.Parse(time.RFC3339, createdAt.(string)); err == nil {
							memory.CreatedAt = timestamp
						}
					}
					if updatedAt, exists := nodeMap["updated_at"]; exists {
						if timestamp, err := time.Parse(time.RFC3339, updatedAt.(string)); err == nil {
							memory.UpdatedAt = timestamp
						}
					}
					
					// Store memory
					ttm.memories[memory.ID] = memory
					
					// Categorize memory
					memoryType := ttm.categorizeMemory(memory)
					switch memoryType {
					case MemoryTypeWorking:
						ttm.workingMemory[memory.ID] = memory
					case MemoryTypeLongTerm:
						ttm.longTermMemory[memory.ID] = memory
					case MemoryTypeUser:
						ttm.userMemory[memory.ID] = memory
					case MemoryTypeInternet:
						ttm.internetMemory[memory.ID] = memory
					}
				}
			}
		}
	}
	
	// Rebuild search index
	memories := make([]*types.TextualMemoryItem, 0)
	for _, memory := range ttm.memories {
		memories = append(memories, memory)
	}
	return ttm.updateSearchIndex(memories)
}

func (ttm *TreeTextMemory) exportToData() map[string]interface{} {
	ttm.memMu.RLock()
	defer ttm.memMu.RUnlock()
	
	// Export nodes
	nodes := make([]map[string]interface{}, 0)
	for _, memory := range ttm.memories {
		node := map[string]interface{}{
			"id":         memory.ID,
			"memory":     memory.Memory,
			"metadata":   memory.Metadata,
			"created_at": memory.CreatedAt.Format(time.RFC3339),
			"updated_at": memory.UpdatedAt.Format(time.RFC3339),
		}
		nodes = append(nodes, node)
	}
	
	// Export edges
	edges := make([]map[string]interface{}, 0)
	for _, edgeList := range ttm.graph.edges {
		for _, edge := range edgeList {
			edgeMap := map[string]interface{}{
				"source": edge.Source,
				"target": edge.Target,
				"type":   edge.Type,
				"weight": edge.Weight,
			}
			edges = append(edges, edgeMap)
		}
	}
	
	return map[string]interface{}{
		"nodes":    nodes,
		"edges":    edges,
		"metadata": map[string]interface{}{
			"exported_at": time.Now().Format(time.RFC3339),
			"version":     "1.0",
			"type":        "tree_text_memory",
		},
	}
}

func (ttm *TreeTextMemory) cleanupOldBackups(keepLastN int) error {
	// Simple cleanup - in a real implementation, this would be more sophisticated
	return nil
}

// SearchSemantic performs semantic search with filters
func (ttm *TreeTextMemory) SearchSemantic(ctx context.Context, query string, topK int, filters map[string]string) ([]*types.TextualMemoryItem, error) {
	req := &SearchRequest{
		Query:   query,
		TopK:    topK,
		Mode:    SearchModeSemantic,
		Filters: filters,
	}
	
	return ttm.semanticSearch(ctx, req)
}

// Placeholder implementations for searcher components

func (tgp *TaskGoalParser) ParseQuery(ctx context.Context, query string) (map[string]interface{}, error) {
	// Simple goal parsing - in a real implementation, this would use LLM
	return map[string]interface{}{
		"intent": "search",
		"query":  query,
		"goals":  []string{query},
	}, nil
}

func (mr *MemoryReranker) Rerank(ctx context.Context, results []*SearchResult, goals map[string]interface{}) ([]*SearchResult, error) {
	// Simple reranking - in a real implementation, this would use LLM
	return results, nil
}

func (mr *MemoryReasoner) Reason(ctx context.Context, results []*SearchResult, goals map[string]interface{}) ([]*SearchResult, error) {
	// Simple reasoning - in a real implementation, this would use LLM
	return results, nil
}