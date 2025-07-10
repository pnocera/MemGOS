// Package textual provides naive textual memory implementation
package textual

import (
	"context"
	"encoding/json"
	"fmt"
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

// NaiveTextMemory implements a simple in-memory textual memory storage
type NaiveTextMemory struct {
	*BaseTextualMemory
	config    *config.MemoryConfig
	memories  map[string]*types.TextualMemoryItem
	memMu     sync.RWMutex
	wordIndex map[string][]string // word -> memory IDs
}

// NewNaiveTextMemory creates a new naive textual memory
func NewNaiveTextMemory(cfg *config.MemoryConfig, logger interfaces.Logger, metrics interfaces.Metrics) (*NaiveTextMemory, error) {
	base := NewBaseTextualMemory(nil, nil, nil, nil, logger, metrics)
	
	ntm := &NaiveTextMemory{
		BaseTextualMemory: base,
		config:            cfg,
		memories:          make(map[string]*types.TextualMemoryItem),
		wordIndex:         make(map[string][]string),
	}
	
	return ntm, nil
}

// Add adds memories to the storage
func (ntm *NaiveTextMemory) Add(ctx context.Context, memories []*types.TextualMemoryItem) error {
	if ntm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	start := time.Now()
	defer func() {
		if ntm.metrics != nil {
			ntm.metrics.Timer("naive_memory_add_duration", float64(time.Since(start).Milliseconds()), nil)
		}
	}()
	
	ntm.memMu.Lock()
	defer ntm.memMu.Unlock()
	
	for _, memory := range memories {
		if memory.ID == "" {
			memory.ID = ntm.generateID()
		}
		
		if memory.CreatedAt.IsZero() {
			memory.CreatedAt = time.Now()
		}
		memory.UpdatedAt = time.Now()
		
		// Store memory
		ntm.memories[memory.ID] = memory
		
		// Update word index
		ntm.updateWordIndex(memory)
	}
	
	ntm.logger.Info("Added naive memories", map[string]interface{}{
		"count": len(memories),
	})
	
	if ntm.metrics != nil {
		ntm.metrics.Counter("naive_memory_add_count", float64(len(memories)), nil)
	}
	
	return nil
}

// Get retrieves a memory by ID
func (ntm *NaiveTextMemory) Get(ctx context.Context, id string) (*types.TextualMemoryItem, error) {
	if ntm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	ntm.memMu.RLock()
	defer ntm.memMu.RUnlock()
	
	memory, exists := ntm.memories[id]
	if !exists {
		return nil, errors.NewMemoryNotFoundError(id)
	}
	
	return memory, nil
}

// GetByIDs retrieves multiple memories by IDs
func (ntm *NaiveTextMemory) GetByIDs(ctx context.Context, ids []string) ([]*types.TextualMemoryItem, error) {
	if ntm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	ntm.memMu.RLock()
	defer ntm.memMu.RUnlock()
	
	result := make([]*types.TextualMemoryItem, 0, len(ids))
	for _, id := range ids {
		if memory, exists := ntm.memories[id]; exists {
			result = append(result, memory)
		}
	}
	
	return result, nil
}

// GetAll retrieves all memories
func (ntm *NaiveTextMemory) GetAll(ctx context.Context) ([]*types.TextualMemoryItem, error) {
	if ntm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	ntm.memMu.RLock()
	defer ntm.memMu.RUnlock()
	
	result := make([]*types.TextualMemoryItem, 0, len(ntm.memories))
	for _, memory := range ntm.memories {
		result = append(result, memory)
	}
	
	return result, nil
}

// Update updates an existing memory
func (ntm *NaiveTextMemory) Update(ctx context.Context, id string, memory *types.TextualMemoryItem) error {
	if ntm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	ntm.memMu.Lock()
	defer ntm.memMu.Unlock()
	
	if _, exists := ntm.memories[id]; !exists {
		return errors.NewMemoryNotFoundError(id)
	}
	
	// Remove old word index entries
	if oldMemory, exists := ntm.memories[id]; exists {
		ntm.removeFromWordIndex(oldMemory)
	}
	
	memory.ID = id
	memory.UpdatedAt = time.Now()
	ntm.memories[id] = memory
	
	// Update word index
	ntm.updateWordIndex(memory)
	
	ntm.logger.Info("Updated naive memory", map[string]interface{}{
		"memory_id": id,
	})
	
	return nil
}

// Delete deletes memories by IDs
func (ntm *NaiveTextMemory) Delete(ctx context.Context, ids []string) error {
	if ntm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	ntm.memMu.Lock()
	defer ntm.memMu.Unlock()
	
	deletedCount := 0
	for _, id := range ids {
		if memory, exists := ntm.memories[id]; exists {
			// Remove from word index
			ntm.removeFromWordIndex(memory)
			
			delete(ntm.memories, id)
			deletedCount++
		}
	}
	
	ntm.logger.Info("Deleted naive memories", map[string]interface{}{
		"count": deletedCount,
	})
	
	return nil
}

// DeleteAll deletes all memories
func (ntm *NaiveTextMemory) DeleteAll(ctx context.Context) error {
	if ntm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	ntm.memMu.Lock()
	defer ntm.memMu.Unlock()
	
	count := len(ntm.memories)
	ntm.memories = make(map[string]*types.TextualMemoryItem)
	ntm.wordIndex = make(map[string][]string)
	
	ntm.logger.Info("Deleted all naive memories", map[string]interface{}{
		"count": count,
	})
	
	return nil
}

// Search performs simple keyword-based search
func (ntm *NaiveTextMemory) Search(ctx context.Context, query string, topK int, mode SearchMode, memoryType MemoryType) ([]*types.TextualMemoryItem, error) {
	if ntm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	start := time.Now()
	defer func() {
		if ntm.metrics != nil {
			ntm.metrics.Timer("naive_memory_search_duration", float64(time.Since(start).Milliseconds()), nil)
		}
	}()
	
	ntm.memMu.RLock()
	defer ntm.memMu.RUnlock()
	
	// Simple keyword search
	results := ntm.keywordSearch(query, topK)
	
	ntm.logger.Info("Naive search completed", map[string]interface{}{
		"query":   query,
		"results": len(results),
		"mode":    mode,
	})
	
	return results, nil
}

// SearchSemantic performs semantic search (falls back to keyword search)
func (ntm *NaiveTextMemory) SearchSemantic(ctx context.Context, query string, topK int, filters map[string]string) ([]*types.TextualMemoryItem, error) {
	// Naive implementation falls back to keyword search
	return ntm.Search(ctx, query, topK, SearchModeFast, MemoryTypeAll)
}

// GetRelevantSubgraph returns empty result (not supported)
func (ntm *NaiveTextMemory) GetRelevantSubgraph(ctx context.Context, query string, topK int, depth int) (map[string]interface{}, error) {
	return map[string]interface{}{
		"core_id": "",
		"nodes":   []interface{}{},
		"edges":   []interface{}{},
	}, nil
}

// GetWorkingMemory returns all memories (naive implementation)
func (ntm *NaiveTextMemory) GetWorkingMemory(ctx context.Context) ([]*types.TextualMemoryItem, error) {
	return ntm.GetAll(ctx)
}

// ReplaceWorkingMemory replaces all memories (naive implementation)
func (ntm *NaiveTextMemory) ReplaceWorkingMemory(ctx context.Context, memories []*types.TextualMemoryItem) error {
	if err := ntm.DeleteAll(ctx); err != nil {
		return err
	}
	return ntm.Add(ctx, memories)
}

// GetMemorySize returns memory size statistics
func (ntm *NaiveTextMemory) GetMemorySize(ctx context.Context) (map[string]int, error) {
	if ntm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	ntm.memMu.RLock()
	defer ntm.memMu.RUnlock()
	
	return map[string]int{
		"total":      len(ntm.memories),
		"word_index": len(ntm.wordIndex),
	}, nil
}

// Load loads memories from directory
func (ntm *NaiveTextMemory) Load(ctx context.Context, dir string) error {
	if ntm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	filePath := filepath.Join(dir, ntm.config.MemoryFilename)
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		ntm.logger.Info("Naive memory file not found, starting with empty memories", map[string]interface{}{
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
	var memories []*types.TextualMemoryItem
	if err := json.Unmarshal(data, &memories); err != nil {
		return fmt.Errorf("failed to parse memory file: %w", err)
	}
	
	// Add memories
	if err := ntm.Add(ctx, memories); err != nil {
		return fmt.Errorf("failed to add loaded memories: %w", err)
	}
	
	ntm.logger.Info("Loaded naive memories", map[string]interface{}{
		"count":     len(memories),
		"file_path": filePath,
	})
	
	return nil
}

// Dump saves memories to directory
func (ntm *NaiveTextMemory) Dump(ctx context.Context, dir string) error {
	if ntm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Get all memories
	memories, err := ntm.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to get memories: %w", err)
	}
	
	// Marshal to JSON
	data, err := json.MarshalIndent(memories, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal memories: %w", err)
	}
	
	// Write to file
	filePath := filepath.Join(dir, ntm.config.MemoryFilename)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write memory file: %w", err)
	}
	
	ntm.logger.Info("Dumped naive memories", map[string]interface{}{
		"count":     len(memories),
		"file_path": filePath,
	})
	
	return nil
}

// Drop deletes all memories (simple implementation)
func (ntm *NaiveTextMemory) Drop(ctx context.Context, keepLastN int) error {
	return ntm.DeleteAll(ctx)
}

// Helper methods

func (ntm *NaiveTextMemory) generateID() string {
	return fmt.Sprintf("naive_%d", time.Now().UnixNano())
}

func (ntm *NaiveTextMemory) updateWordIndex(memory *types.TextualMemoryItem) {
	words := ntm.extractWords(memory.Memory)
	
	for _, word := range words {
		word = strings.ToLower(word)
		if ntm.wordIndex[word] == nil {
			ntm.wordIndex[word] = make([]string, 0)
		}
		
		// Check if memory ID already exists for this word
		exists := false
		for _, id := range ntm.wordIndex[word] {
			if id == memory.ID {
				exists = true
				break
			}
		}
		
		if !exists {
			ntm.wordIndex[word] = append(ntm.wordIndex[word], memory.ID)
		}
	}
}

func (ntm *NaiveTextMemory) removeFromWordIndex(memory *types.TextualMemoryItem) {
	words := ntm.extractWords(memory.Memory)
	
	for _, word := range words {
		word = strings.ToLower(word)
		if memoryIDs, exists := ntm.wordIndex[word]; exists {
			// Remove memory ID from the list
			newIDs := make([]string, 0)
			for _, id := range memoryIDs {
				if id != memory.ID {
					newIDs = append(newIDs, id)
				}
			}
			
			if len(newIDs) == 0 {
				delete(ntm.wordIndex, word)
			} else {
				ntm.wordIndex[word] = newIDs
			}
		}
	}
}

func (ntm *NaiveTextMemory) extractWords(text string) []string {
	// Simple word extraction
	text = strings.ToLower(text)
	words := strings.Fields(text)
	
	// Clean words (remove punctuation)
	cleanWords := make([]string, 0)
	for _, word := range words {
		cleaned := ntm.cleanWord(word)
		if len(cleaned) > 0 {
			cleanWords = append(cleanWords, cleaned)
		}
	}
	
	return cleanWords
}

func (ntm *NaiveTextMemory) cleanWord(word string) string {
	// Remove common punctuation
	punctuation := ".,!?;:()[]{}\"'-"
	cleaned := word
	
	for _, char := range punctuation {
		cleaned = strings.ReplaceAll(cleaned, string(char), "")
	}
	
	return cleaned
}

func (ntm *NaiveTextMemory) keywordSearch(query string, topK int) []*types.TextualMemoryItem {
	queryWords := ntm.extractWords(query)
	if len(queryWords) == 0 {
		return []*types.TextualMemoryItem{}
	}
	
	// Count matches for each memory
	memoryScores := make(map[string]int)
	
	for _, word := range queryWords {
		if memoryIDs, exists := ntm.wordIndex[word]; exists {
			for _, memoryID := range memoryIDs {
				memoryScores[memoryID]++
			}
		}
	}
	
	// Sort by score
	type memoryScore struct {
		id    string
		score int
	}
	
	scores := make([]*memoryScore, 0)
	for memoryID, score := range memoryScores {
		scores = append(scores, &memoryScore{
			id:    memoryID,
			score: score,
		})
	}
	
	// Sort by score (descending)
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].score == scores[j].score {
			// If scores are equal, sort by memory ID for consistency
			return scores[i].id < scores[j].id
		}
		return scores[i].score > scores[j].score
	})
	
	// Limit to topK
	if topK > 0 && len(scores) > topK {
		scores = scores[:topK]
	}
	
	// Build result
	result := make([]*types.TextualMemoryItem, 0, len(scores))
	for _, score := range scores {
		if memory, exists := ntm.memories[score.id]; exists {
			result = append(result, memory)
		}
	}
	
	return result
}