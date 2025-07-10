// Package textual provides general textual memory implementation
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

// GeneralTextMemory implements a general-purpose textual memory with semantic search
type GeneralTextMemory struct {
	*BaseTextualMemory
	config             *config.MemoryConfig
	memories           map[string]*types.TextualMemoryItem
	memMu              sync.RWMutex
	embeddings         map[string][]float32
	tfIdfIndex         *TfIdfIndex
	internetRetriever  interfaces.InternetRetriever
	compressionEnabled bool
}

// TfIdfIndex provides TF-IDF based search capabilities
type TfIdfIndex struct {
	termFreq      map[string]map[string]int     // memoryID -> term -> frequency
	docFreq       map[string]int                // term -> document frequency
	totalDocs     int
	termWeights   map[string]map[string]float64 // memoryID -> term -> tf-idf weight
	mu            sync.RWMutex
}

// NewGeneralTextMemory creates a new general textual memory
func NewGeneralTextMemory(cfg *config.MemoryConfig, embedder interfaces.Embedder, vectorDB interfaces.VectorDB, llm interfaces.LLM, logger interfaces.Logger, metrics interfaces.Metrics) (*GeneralTextMemory, error) {
	base := NewBaseTextualMemory(embedder, vectorDB, nil, llm, logger, metrics)
	
	tfIdfIndex := &TfIdfIndex{
		termFreq:    make(map[string]map[string]int),
		docFreq:     make(map[string]int),
		totalDocs:   0,
		termWeights: make(map[string]map[string]float64),
	}
	
	gtm := &GeneralTextMemory{
		BaseTextualMemory:  base,
		config:             cfg,
		memories:           make(map[string]*types.TextualMemoryItem),
		embeddings:         make(map[string][]float32),
		tfIdfIndex:         tfIdfIndex,
		compressionEnabled: true,
	}
	
	return gtm, nil
}

// SetInternetRetriever sets the internet retriever
func (gtm *GeneralTextMemory) SetInternetRetriever(retriever interfaces.InternetRetriever) {
	gtm.internetRetriever = retriever
}

// Add adds memories to the storage
func (gtm *GeneralTextMemory) Add(ctx context.Context, memories []*types.TextualMemoryItem) error {
	if gtm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	start := time.Now()
	defer func() {
		if gtm.metrics != nil {
			gtm.metrics.Timer("general_memory_add_duration", float64(time.Since(start).Milliseconds()), nil)
		}
	}()
	
	gtm.memMu.Lock()
	defer gtm.memMu.Unlock()
	
	for _, memory := range memories {
		if memory.ID == "" {
			memory.ID = gtm.generateID()
		}
		
		if memory.CreatedAt.IsZero() {
			memory.CreatedAt = time.Now()
		}
		memory.UpdatedAt = time.Now()
		
		// Store memory
		gtm.memories[memory.ID] = memory
		
		// Generate embedding if embedder is available
		if gtm.embedder != nil {
			if err := gtm.generateEmbedding(ctx, memory); err != nil {
				gtm.logger.Error("Failed to generate embedding", map[string]interface{}{
					"memory_id": memory.ID,
					"error":     err.Error(),
				})
			}
		}
		
		// Update TF-IDF index
		gtm.tfIdfIndex.addDocument(memory.ID, memory.Memory)
		
		// Store in vector database if available
		if gtm.vectorDB != nil && gtm.embeddings[memory.ID] != nil {
			if err := gtm.storeInVectorDB(memory); err != nil {
				gtm.logger.Error("Failed to store in vector database", map[string]interface{}{
					"memory_id": memory.ID,
					"error":     err.Error(),
				})
			}
		}
	}
	
	gtm.logger.Info("Added general memories", map[string]interface{}{
		"count": len(memories),
	})
	
	if gtm.metrics != nil {
		gtm.metrics.Counter("general_memory_add_count", float64(len(memories)), nil)
	}
	
	return nil
}

// Get retrieves a memory by ID
func (gtm *GeneralTextMemory) Get(ctx context.Context, id string) (*types.TextualMemoryItem, error) {
	if gtm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	gtm.memMu.RLock()
	defer gtm.memMu.RUnlock()
	
	memory, exists := gtm.memories[id]
	if !exists {
		return nil, errors.NewMemoryNotFoundError(id)
	}
	
	return memory, nil
}

// GetByIDs retrieves multiple memories by IDs
func (gtm *GeneralTextMemory) GetByIDs(ctx context.Context, ids []string) ([]*types.TextualMemoryItem, error) {
	if gtm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	gtm.memMu.RLock()
	defer gtm.memMu.RUnlock()
	
	result := make([]*types.TextualMemoryItem, 0, len(ids))
	for _, id := range ids {
		if memory, exists := gtm.memories[id]; exists {
			result = append(result, memory)
		}
	}
	
	return result, nil
}

// GetAll retrieves all memories
func (gtm *GeneralTextMemory) GetAll(ctx context.Context) ([]*types.TextualMemoryItem, error) {
	if gtm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	gtm.memMu.RLock()
	defer gtm.memMu.RUnlock()
	
	result := make([]*types.TextualMemoryItem, 0, len(gtm.memories))
	for _, memory := range gtm.memories {
		result = append(result, memory)
	}
	
	return result, nil
}

// Update updates an existing memory
func (gtm *GeneralTextMemory) Update(ctx context.Context, id string, memory *types.TextualMemoryItem) error {
	if gtm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	gtm.memMu.Lock()
	defer gtm.memMu.Unlock()
	
	if _, exists := gtm.memories[id]; !exists {
		return errors.NewMemoryNotFoundError(id)
	}
	
	// Remove old index entries
	gtm.tfIdfIndex.removeDocument(id)
	delete(gtm.embeddings, id)
	
	memory.ID = id
	memory.UpdatedAt = time.Now()
	gtm.memories[id] = memory
	
	// Regenerate embedding
	if gtm.embedder != nil {
		if err := gtm.generateEmbedding(ctx, memory); err != nil {
			gtm.logger.Error("Failed to regenerate embedding", map[string]interface{}{
				"memory_id": id,
				"error":     err.Error(),
			})
		}
	}
	
	// Update TF-IDF index
	gtm.tfIdfIndex.addDocument(id, memory.Memory)
	
	gtm.logger.Info("Updated general memory", map[string]interface{}{
		"memory_id": id,
	})
	
	return nil
}

// Delete deletes memories by IDs
func (gtm *GeneralTextMemory) Delete(ctx context.Context, ids []string) error {
	if gtm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	gtm.memMu.Lock()
	defer gtm.memMu.Unlock()
	
	deletedCount := 0
	for _, id := range ids {
		if _, exists := gtm.memories[id]; exists {
			// Remove from indices
			gtm.tfIdfIndex.removeDocument(id)
			delete(gtm.embeddings, id)
			
			// Remove from vector database
			if gtm.vectorDB != nil {
				if err := gtm.removeFromVectorDB(id); err != nil {
					gtm.logger.Error("Failed to remove from vector database", map[string]interface{}{
						"memory_id": id,
						"error":     err.Error(),
					})
				}
			}
			
			delete(gtm.memories, id)
			deletedCount++
		}
	}
	
	gtm.logger.Info("Deleted general memories", map[string]interface{}{
		"count": deletedCount,
	})
	
	return nil
}

// DeleteAll deletes all memories
func (gtm *GeneralTextMemory) DeleteAll(ctx context.Context) error {
	if gtm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	gtm.memMu.Lock()
	defer gtm.memMu.Unlock()
	
	count := len(gtm.memories)
	gtm.memories = make(map[string]*types.TextualMemoryItem)
	gtm.embeddings = make(map[string][]float32)
	
	// Reset TF-IDF index
	gtm.tfIdfIndex = &TfIdfIndex{
		termFreq:    make(map[string]map[string]int),
		docFreq:     make(map[string]int),
		totalDocs:   0,
		termWeights: make(map[string]map[string]float64),
	}
	
	// Clear vector database
	if gtm.vectorDB != nil {
		if err := gtm.clearVectorDB(); err != nil {
			gtm.logger.Error("Failed to clear vector database", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
	
	gtm.logger.Info("Deleted all general memories", map[string]interface{}{
		"count": count,
	})
	
	return nil
}

// Search performs multi-modal search (TF-IDF + semantic + internet)
func (gtm *GeneralTextMemory) Search(ctx context.Context, query string, topK int, mode SearchMode, memoryType MemoryType) ([]*types.TextualMemoryItem, error) {
	if gtm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	start := time.Now()
	defer func() {
		if gtm.metrics != nil {
			gtm.metrics.Timer("general_memory_search_duration", float64(time.Since(start).Milliseconds()), nil)
		}
	}()
	
	var results []*types.TextualMemoryItem
	var err error
	
	switch mode {
	case SearchModeFast:
		results, err = gtm.fastSearch(ctx, query, topK)
	case SearchModeSemantic:
		results, err = gtm.semanticSearch(ctx, query, topK)
	case SearchModeFine:
		results, err = gtm.hybridSearch(ctx, query, topK)
	default:
		results, err = gtm.fastSearch(ctx, query, topK)
	}
	
	if err != nil {
		return nil, err
	}
	
	// Add internet results if enabled and not enough local results
	if gtm.internetRetriever != nil && len(results) < topK {
		internetResults, err := gtm.searchInternet(ctx, query, topK-len(results))
		if err != nil {
			gtm.logger.Error("Failed to search internet", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			results = append(results, internetResults...)
		}
	}
	
	gtm.logger.Info("General search completed", map[string]interface{}{
		"query":   query,
		"results": len(results),
		"mode":    mode,
	})
	
	return results, nil
}

// SearchSemantic performs semantic search using embeddings
func (gtm *GeneralTextMemory) SearchSemantic(ctx context.Context, query string, topK int, filters map[string]string) ([]*types.TextualMemoryItem, error) {
	return gtm.semanticSearch(ctx, query, topK)
}

// GetRelevantSubgraph returns empty result (not supported in general memory)
func (gtm *GeneralTextMemory) GetRelevantSubgraph(ctx context.Context, query string, topK int, depth int) (map[string]interface{}, error) {
	return map[string]interface{}{
		"core_id": "",
		"nodes":   []interface{}{},
		"edges":   []interface{}{},
	}, nil
}

// GetWorkingMemory returns recent memories
func (gtm *GeneralTextMemory) GetWorkingMemory(ctx context.Context) ([]*types.TextualMemoryItem, error) {
	memories, err := gtm.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	
	// Sort by update time (most recent first)
	sort.Slice(memories, func(i, j int) bool {
		return memories[i].UpdatedAt.After(memories[j].UpdatedAt)
	})
	
	// Return last 100 memories
	if len(memories) > 100 {
		memories = memories[:100]
	}
	
	return memories, nil
}

// ReplaceWorkingMemory replaces recent memories
func (gtm *GeneralTextMemory) ReplaceWorkingMemory(ctx context.Context, memories []*types.TextualMemoryItem) error {
	// Get current working memory
	workingMemory, err := gtm.GetWorkingMemory(ctx)
	if err != nil {
		return err
	}
	
	// Delete working memory
	workingIDs := make([]string, len(workingMemory))
	for i, memory := range workingMemory {
		workingIDs[i] = memory.ID
	}
	
	if err := gtm.Delete(ctx, workingIDs); err != nil {
		return err
	}
	
	// Add new memories
	return gtm.Add(ctx, memories)
}

// GetMemorySize returns memory size statistics
func (gtm *GeneralTextMemory) GetMemorySize(ctx context.Context) (map[string]int, error) {
	if gtm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	gtm.memMu.RLock()
	defer gtm.memMu.RUnlock()
	
	return map[string]int{
		"total":       len(gtm.memories),
		"embeddings":  len(gtm.embeddings),
		"tf_idf_docs": gtm.tfIdfIndex.totalDocs,
		"tf_idf_terms": len(gtm.tfIdfIndex.docFreq),
	}, nil
}

// Load loads memories from directory
func (gtm *GeneralTextMemory) Load(ctx context.Context, dir string) error {
	if gtm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	filePath := filepath.Join(dir, gtm.config.MemoryFilename)
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		gtm.logger.Info("General memory file not found, starting with empty memories", map[string]interface{}{
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
	var saveData map[string]interface{}
	if err := json.Unmarshal(data, &saveData); err != nil {
		return fmt.Errorf("failed to parse memory file: %w", err)
	}
	
	// Load memories
	if err := gtm.loadFromSaveData(ctx, saveData); err != nil {
		return fmt.Errorf("failed to load memories: %w", err)
	}
	
	gtm.logger.Info("Loaded general memories", map[string]interface{}{
		"count":     len(gtm.memories),
		"file_path": filePath,
	})
	
	return nil
}

// Dump saves memories to directory
func (gtm *GeneralTextMemory) Dump(ctx context.Context, dir string) error {
	if gtm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Prepare save data
	saveData := gtm.prepareForSave()
	
	// Marshal to JSON
	data, err := json.MarshalIndent(saveData, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal memories: %w", err)
	}
	
	// Write to file
	filePath := filepath.Join(dir, gtm.config.MemoryFilename)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write memory file: %w", err)
	}
	
	gtm.logger.Info("Dumped general memories", map[string]interface{}{
		"count":     len(gtm.memories),
		"file_path": filePath,
	})
	
	return nil
}

// Drop deletes all memories with backup
func (gtm *GeneralTextMemory) Drop(ctx context.Context, keepLastN int) error {
	// Create backup
	backupDir := fmt.Sprintf("/tmp/general_memory_backup_%s", time.Now().Format("20060102_150405"))
	if err := gtm.Dump(ctx, backupDir); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	
	// Delete all memories
	if err := gtm.DeleteAll(ctx); err != nil {
		return fmt.Errorf("failed to delete all memories: %w", err)
	}
	
	gtm.logger.Info("Dropped general memories with backup", map[string]interface{}{
		"backup_dir": backupDir,
	})
	
	return nil
}

// Helper methods

func (gtm *GeneralTextMemory) generateID() string {
	return fmt.Sprintf("general_%d", time.Now().UnixNano())
}

func (gtm *GeneralTextMemory) generateEmbedding(ctx context.Context, memory *types.TextualMemoryItem) error {
	embeddings, err := gtm.embedder.Embed(ctx, []string{memory.Memory})
	if err != nil {
		return err
	}
	
	if len(embeddings) > 0 {
		gtm.embeddings[memory.ID] = embeddings[0]
	}
	
	return nil
}

func (gtm *GeneralTextMemory) storeInVectorDB(memory *types.TextualMemoryItem) error {
	if embedding, exists := gtm.embeddings[memory.ID]; exists {
		// Store in vector database
		return gtm.vectorDB.Add(ctx, []types.VectorSearchResult{
			{
				ID:       memory.ID,
				Score:    1.0,
				Metadata: memory.Metadata,
			},
		})
	}
	return nil
}

func (gtm *GeneralTextMemory) removeFromVectorDB(id string) error {
	return gtm.vectorDB.Delete(ctx, []string{id})
}

func (gtm *GeneralTextMemory) clearVectorDB() error {
	return gtm.vectorDB.Clear(ctx)
}

func (gtm *GeneralTextMemory) fastSearch(ctx context.Context, query string, topK int) ([]*types.TextualMemoryItem, error) {
	// Use TF-IDF search
	results := gtm.tfIdfIndex.search(query, topK)
	
	memories := make([]*types.TextualMemoryItem, 0, len(results))
	for _, result := range results {
		if memory, exists := gtm.memories[result.id]; exists {
			memories = append(memories, memory)
		}
	}
	
	return memories, nil
}

func (gtm *GeneralTextMemory) semanticSearch(ctx context.Context, query string, topK int) ([]*types.TextualMemoryItem, error) {
	if gtm.embedder == nil {
		// Fall back to TF-IDF search
		return gtm.fastSearch(ctx, query, topK)
	}
	
	// Generate query embedding
	queryEmbeddings, err := gtm.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	
	if len(queryEmbeddings) == 0 {
		return gtm.fastSearch(ctx, query, topK)
	}
	
	queryEmbedding := queryEmbeddings[0]
	
	// Calculate similarities
	type memoryScore struct {
		id    string
		score float32
	}
	
	scores := make([]*memoryScore, 0)
	gtm.memMu.RLock()
	for id, embedding := range gtm.embeddings {
		similarity := gtm.cosineSimilarity(queryEmbedding, embedding)
		if similarity > 0.1 { // Threshold
			scores = append(scores, &memoryScore{
				id:    id,
				score: similarity,
			})
		}
	}
	gtm.memMu.RUnlock()
	
	// Sort by similarity
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})
	
	// Limit to topK
	if len(scores) > topK {
		scores = scores[:topK]
	}
	
	// Build result
	memories := make([]*types.TextualMemoryItem, 0, len(scores))
	for _, score := range scores {
		if memory, exists := gtm.memories[score.id]; exists {
			memories = append(memories, memory)
		}
	}
	
	return memories, nil
}

func (gtm *GeneralTextMemory) hybridSearch(ctx context.Context, query string, topK int) ([]*types.TextualMemoryItem, error) {
	// Combine TF-IDF and semantic search
	tfIdfResults, err := gtm.fastSearch(ctx, query, topK*2)
	if err != nil {
		return nil, err
	}
	
	semanticResults, err := gtm.semanticSearch(ctx, query, topK*2)
	if err != nil {
		return nil, err
	}
	
	// Merge and deduplicate results
	seen := make(map[string]bool)
	combined := make([]*types.TextualMemoryItem, 0)
	
	// Add TF-IDF results first (with weight)
	for i, memory := range tfIdfResults {
		if !seen[memory.ID] {
			seen[memory.ID] = true
			combined = append(combined, memory)
		}
		if len(combined) >= topK {
			break
		}
		// Give higher weight to earlier TF-IDF results
		if i >= topK/2 {
			break
		}
	}
	
	// Add semantic results
	for _, memory := range semanticResults {
		if !seen[memory.ID] {
			seen[memory.ID] = true
			combined = append(combined, memory)
		}
		if len(combined) >= topK {
			break
		}
	}
	
	return combined, nil
}

func (gtm *GeneralTextMemory) searchInternet(ctx context.Context, query string, topK int) ([]*types.TextualMemoryItem, error) {
	if gtm.internetRetriever == nil {
		return []*types.TextualMemoryItem{}, nil
	}
	
	// Retrieve from internet
	internetResults, err := gtm.internetRetriever.Retrieve(ctx, query, topK)
	if err != nil {
		return nil, err
	}
	
	// Convert to textual memory items
	memories := make([]*types.TextualMemoryItem, 0, len(internetResults))
	for _, result := range internetResults {
		memory := &types.TextualMemoryItem{
			ID:       gtm.generateID(),
			Memory:   result.Content,
			Metadata: map[string]interface{}{
				"url":           result.URL,
				"title":         result.Title,
				"source":        result.Source,
				"retrieved_at":  result.RetrievedAt,
				"internet_score": result.Score,
				"memory_type":   "internet",
			},
			CreatedAt: result.RetrievedAt,
			UpdatedAt: result.RetrievedAt,
		}
		memories = append(memories, memory)
	}
	
	return memories, nil
}

func (gtm *GeneralTextMemory) cosineSimilarity(a, b []float32) float32 {
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

func (gtm *GeneralTextMemory) loadFromSaveData(ctx context.Context, saveData map[string]interface{}) error {
	// Load memories
	if memoriesData, exists := saveData["memories"]; exists {
		if memoriesSlice, ok := memoriesData.([]interface{}); ok {
			memories := make([]*types.TextualMemoryItem, 0)
			for _, memoryData := range memoriesSlice {
				if memoryMap, ok := memoryData.(map[string]interface{}); ok {
					memory := &types.TextualMemoryItem{}
					
					if id, exists := memoryMap["id"]; exists {
						memory.ID = id.(string)
					}
					if memoryContent, exists := memoryMap["memory"]; exists {
						memory.Memory = memoryContent.(string)
					}
					if metadata, exists := memoryMap["metadata"]; exists {
						memory.Metadata = metadata.(map[string]interface{})
					}
					
					// Parse timestamps
					if createdAt, exists := memoryMap["created_at"]; exists {
						if timestamp, err := time.Parse(time.RFC3339, createdAt.(string)); err == nil {
							memory.CreatedAt = timestamp
						}
					}
					if updatedAt, exists := memoryMap["updated_at"]; exists {
						if timestamp, err := time.Parse(time.RFC3339, updatedAt.(string)); err == nil {
							memory.UpdatedAt = timestamp
						}
					}
					
					memories = append(memories, memory)
				}
			}
			
			// Add memories (this will rebuild indices)
			if err := gtm.Add(ctx, memories); err != nil {
				return err
			}
		}
	}
	
	// Load embeddings if available
	if embeddingsData, exists := saveData["embeddings"]; exists {
		if embeddingsMap, ok := embeddingsData.(map[string]interface{}); ok {
			for id, embeddingData := range embeddingsMap {
				if embeddingSlice, ok := embeddingData.([]interface{}); ok {
					embedding := make([]float32, len(embeddingSlice))
					for i, val := range embeddingSlice {
						if floatVal, ok := val.(float64); ok {
							embedding[i] = float32(floatVal)
						}
					}
					gtm.embeddings[id] = embedding
				}
			}
		}
	}
	
	return nil
}

func (gtm *GeneralTextMemory) prepareForSave() map[string]interface{} {
	gtm.memMu.RLock()
	defer gtm.memMu.RUnlock()
	
	// Prepare memories
	memories := make([]map[string]interface{}, 0)
	for _, memory := range gtm.memories {
		memoryData := map[string]interface{}{
			"id":         memory.ID,
			"memory":     memory.Memory,
			"metadata":   memory.Metadata,
			"created_at": memory.CreatedAt.Format(time.RFC3339),
			"updated_at": memory.UpdatedAt.Format(time.RFC3339),
		}
		memories = append(memories, memoryData)
	}
	
	// Prepare embeddings (optionally compress)
	embeddings := make(map[string]interface{})
	if !gtm.compressionEnabled {
		for id, embedding := range gtm.embeddings {
			embeddings[id] = embedding
		}
	}
	
	return map[string]interface{}{
		"memories":   memories,
		"embeddings": embeddings,
		"metadata": map[string]interface{}{
			"saved_at":         time.Now().Format(time.RFC3339),
			"version":          "1.0",
			"type":             "general_textual_memory",
			"compression":      gtm.compressionEnabled,
			"total_memories":   len(gtm.memories),
			"total_embeddings": len(gtm.embeddings),
		},
	}
}

// TF-IDF Index methods

func (tfidf *TfIdfIndex) addDocument(docID, text string) {
	tfidf.mu.Lock()
	defer tfidf.mu.Unlock()
	
	// Tokenize text
	terms := tfidf.tokenize(text)
	termCount := make(map[string]int)
	
	// Count term frequencies
	for _, term := range terms {
		termCount[term]++
	}
	
	// Update document term frequencies
	tfidf.termFreq[docID] = termCount
	
	// Update document frequencies
	for term := range termCount {
		tfidf.docFreq[term]++
	}
	
	tfidf.totalDocs++
	
	// Calculate TF-IDF weights for this document
	tfidf.calculateWeights(docID, termCount)
}

func (tfidf *TfIdfIndex) removeDocument(docID string) {
	tfidf.mu.Lock()
	defer tfidf.mu.Unlock()
	
	if termCount, exists := tfidf.termFreq[docID]; exists {
		// Update document frequencies
		for term := range termCount {
			tfidf.docFreq[term]--
			if tfidf.docFreq[term] <= 0 {
				delete(tfidf.docFreq, term)
			}
		}
		
		delete(tfidf.termFreq, docID)
		delete(tfidf.termWeights, docID)
		tfidf.totalDocs--
	}
}

func (tfidf *TfIdfIndex) search(query string, topK int) []*tfIdfResult {
	tfidf.mu.RLock()
	defer tfidf.mu.RUnlock()
	
	queryTerms := tfidf.tokenize(query)
	if len(queryTerms) == 0 {
		return []*tfIdfResult{}
	}
	
	// Calculate query TF-IDF
	queryTF := make(map[string]int)
	for _, term := range queryTerms {
		queryTF[term]++
	}
	
	queryWeights := make(map[string]float64)
	for term, tf := range queryTF {
		if df, exists := tfidf.docFreq[term]; exists && df > 0 {
			idf := math.Log(float64(tfidf.totalDocs) / float64(df))
			queryWeights[term] = float64(tf) * idf
		}
	}
	
	// Calculate similarities
	type tfIdfResult struct {
		id    string
		score float64
	}
	
	results := make([]*tfIdfResult, 0)
	
	for docID, docWeights := range tfidf.termWeights {
		similarity := tfidf.cosineSimilarity(queryWeights, docWeights)
		if similarity > 0 {
			results = append(results, &tfIdfResult{
				id:    docID,
				score: similarity,
			})
		}
	}
	
	// Sort by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})
	
	// Limit to topK
	if len(results) > topK {
		results = results[:topK]
	}
	
	return results
}

func (tfidf *TfIdfIndex) tokenize(text string) []string {
	// Simple tokenization
	text = strings.ToLower(text)
	words := strings.Fields(text)
	
	// Clean words
	cleanWords := make([]string, 0)
	for _, word := range words {
		cleaned := tfidf.cleanWord(word)
		if len(cleaned) > 0 && len(cleaned) > 2 { // Filter short words
			cleanWords = append(cleanWords, cleaned)
		}
	}
	
	return cleanWords
}

func (tfidf *TfIdfIndex) cleanWord(word string) string {
	// Remove punctuation
	punctuation := ".,!?;:()[]{}\"'-"
	cleaned := word
	
	for _, char := range punctuation {
		cleaned = strings.ReplaceAll(cleaned, string(char), "")
	}
	
	return cleaned
}

func (tfidf *TfIdfIndex) calculateWeights(docID string, termCount map[string]int) {
	weights := make(map[string]float64)
	totalTerms := 0
	
	for _, count := range termCount {
		totalTerms += count
	}
	
	for term, count := range termCount {
		tf := float64(count) / float64(totalTerms)
		if df, exists := tfidf.docFreq[term]; exists && df > 0 {
			idf := math.Log(float64(tfidf.totalDocs) / float64(df))
			weights[term] = tf * idf
		}
	}
	
	tfidf.termWeights[docID] = weights
}

func (tfidf *TfIdfIndex) cosineSimilarity(a, b map[string]float64) float64 {
	var dotProduct, normA, normB float64
	
	// Calculate dot product and norm of a
	for term, weightA := range a {
		if weightB, exists := b[term]; exists {
			dotProduct += weightA * weightB
		}
		normA += weightA * weightA
	}
	
	// Calculate norm of b
	for _, weightB := range b {
		normB += weightB * weightB
	}
	
	if normA == 0 || normB == 0 {
		return 0
	}
	
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

type tfIdfResult struct {
	id    string
	score float64
}