// Package memory provides memory implementations for MemGOS
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/errors"
	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// BaseMemory provides common functionality for all memory implementations
type BaseMemory struct {
	config    *config.MemoryConfig
	mu        sync.RWMutex
	closed    bool
	logger    interfaces.Logger
	metrics   interfaces.Metrics
}

// NewBaseMemory creates a new base memory
func NewBaseMemory(cfg *config.MemoryConfig, logger interfaces.Logger, metrics interfaces.Metrics) *BaseMemory {
	return &BaseMemory{
		config:  cfg,
		logger:  logger,
		metrics: metrics,
	}
}

// GetConfig returns the memory configuration
func (bm *BaseMemory) GetConfig() *config.MemoryConfig {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.config
}

// IsClosed returns true if the memory is closed
func (bm *BaseMemory) IsClosed() bool {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.closed
}

// Close marks the memory as closed
func (bm *BaseMemory) Close() error {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	
	if bm.closed {
		return nil
	}
	
	bm.closed = true
	if bm.logger != nil {
		bm.logger.Info("Memory closed")
	}
	return nil
}

// ensureDir ensures that a directory exists
func (bm *BaseMemory) ensureDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.NewFileError(fmt.Sprintf("failed to create directory %s: %v", dir, err))
	}
	return nil
}

// saveToFile saves data to a file as JSON
func (bm *BaseMemory) saveToFile(data interface{}, filePath string) error {
	if err := bm.ensureDir(filepath.Dir(filePath)); err != nil {
		return err
	}
	
	file, err := os.Create(filePath)
	if err != nil {
		return errors.NewFileError(fmt.Sprintf("failed to create file %s: %v", filePath, err))
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	
	if err := encoder.Encode(data); err != nil {
		return errors.NewFileError(fmt.Sprintf("failed to encode data to file %s: %v", filePath, err))
	}
	
	return nil
}

// loadFromFile loads data from a JSON file
func (bm *BaseMemory) loadFromFile(filePath string, target interface{}) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return errors.NewFileNotFoundError(filePath)
	}
	
	file, err := os.Open(filePath)
	if err != nil {
		return errors.NewFileError(fmt.Sprintf("failed to open file %s: %v", filePath, err))
	}
	defer file.Close()
	
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(target); err != nil {
		return errors.NewFileCorruptedError(filePath)
	}
	
	return nil
}

// generateID generates a new UUID
func (bm *BaseMemory) generateID() string {
	return uuid.New().String()
}

// Enhanced memory implementations are now in separate packages
// TextualMemory remains for backward compatibility but delegates to new implementations
type TextualMemory struct {
	*BaseMemory
	memories  map[string]*types.TextualMemoryItem
	embedder  interfaces.Embedder
	vectorDB  interfaces.VectorDB
	memMu     sync.RWMutex
	
	// Enhanced features
	implementation interfaces.TextualMemory // Delegates to actual implementation
}

// NewTextualMemory creates a new textual memory
func NewTextualMemory(cfg *config.MemoryConfig, logger interfaces.Logger, metrics interfaces.Metrics) (*TextualMemory, error) {
	tm := &TextualMemory{
		BaseMemory: NewBaseMemory(cfg, logger, metrics),
		memories:   make(map[string]*types.TextualMemoryItem),
	}
	
	return tm, nil
}

// Load loads memories from a directory
func (tm *TextualMemory) Load(ctx context.Context, dir string) error {
	if tm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	filePath := filepath.Join(dir, tm.config.MemoryFilename)
	
	var memories []*types.TextualMemoryItem
	if err := tm.loadFromFile(filePath, &memories); err != nil {
		if errors.GetMemGOSError(err) != nil && errors.GetMemGOSError(err).Code == errors.ErrCodeFileNotFound {
			// File doesn't exist, start with empty memories
			tm.logger.Info("Memory file not found, starting with empty memories", map[string]interface{}{
				"file_path": filePath,
			})
			return nil
		}
		return err
	}
	
	tm.memMu.Lock()
	defer tm.memMu.Unlock()
	
	tm.memories = make(map[string]*types.TextualMemoryItem)
	for _, memory := range memories {
		tm.memories[memory.ID] = memory
	}
	
	tm.logger.Info("Loaded textual memories", map[string]interface{}{
		"count":     len(memories),
		"file_path": filePath,
	})
	
	if tm.metrics != nil {
		tm.metrics.Gauge("textual_memory_loaded_count", float64(len(memories)), nil)
	}
	
	return nil
}

// Dump saves memories to a directory
func (tm *TextualMemory) Dump(ctx context.Context, dir string) error {
	if tm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	tm.memMu.RLock()
	memories := make([]*types.TextualMemoryItem, 0, len(tm.memories))
	for _, memory := range tm.memories {
		memories = append(memories, memory)
	}
	tm.memMu.RUnlock()
	
	filePath := filepath.Join(dir, tm.config.MemoryFilename)
	if err := tm.saveToFile(memories, filePath); err != nil {
		return err
	}
	
	tm.logger.Info("Dumped textual memories", map[string]interface{}{
		"count":     len(memories),
		"file_path": filePath,
	})
	
	if tm.metrics != nil {
		tm.metrics.Gauge("textual_memory_dumped_count", float64(len(memories)), nil)
	}
	
	return nil
}

// Add adds new memories
func (tm *TextualMemory) Add(ctx context.Context, memories []types.MemoryItem) error {
	if tm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	start := time.Now()
	defer func() {
		if tm.metrics != nil {
			tm.metrics.Timer("textual_memory_add_duration", float64(time.Since(start).Milliseconds()), nil)
		}
	}()
	
	tm.memMu.Lock()
	defer tm.memMu.Unlock()
	
	for _, item := range memories {
		textItem, ok := item.(*types.TextualMemoryItem)
		if !ok {
			return errors.NewValidationError("invalid memory item type for textual memory")
		}
		
		if textItem.ID == "" {
			textItem.ID = tm.generateID()
		}
		
		if textItem.CreatedAt.IsZero() {
			textItem.CreatedAt = time.Now()
		}
		textItem.UpdatedAt = time.Now()
		
		tm.memories[textItem.ID] = textItem
	}
	
	tm.logger.Info("Added textual memories", map[string]interface{}{
		"count": len(memories),
	})
	
	if tm.metrics != nil {
		tm.metrics.Counter("textual_memory_add_count", float64(len(memories)), nil)
	}
	
	return nil
}

// AddTextual adds textual memory items
func (tm *TextualMemory) AddTextual(ctx context.Context, items []*types.TextualMemoryItem) error {
	memoryItems := make([]types.MemoryItem, len(items))
	for i, item := range items {
		memoryItems[i] = item
	}
	return tm.Add(ctx, memoryItems)
}

// Get retrieves a specific memory by ID
func (tm *TextualMemory) Get(ctx context.Context, id string) (types.MemoryItem, error) {
	if tm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	tm.memMu.RLock()
	defer tm.memMu.RUnlock()
	
	memory, exists := tm.memories[id]
	if !exists {
		return nil, errors.NewMemoryNotFoundError(id)
	}
	
	return memory, nil
}

// GetTextual retrieves textual memories
func (tm *TextualMemory) GetTextual(ctx context.Context, ids []string) ([]*types.TextualMemoryItem, error) {
	if tm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	tm.memMu.RLock()
	defer tm.memMu.RUnlock()
	
	result := make([]*types.TextualMemoryItem, 0, len(ids))
	for _, id := range ids {
		if memory, exists := tm.memories[id]; exists {
			result = append(result, memory)
		}
	}
	
	return result, nil
}

// GetAll retrieves all memories
func (tm *TextualMemory) GetAll(ctx context.Context) ([]types.MemoryItem, error) {
	if tm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	tm.memMu.RLock()
	defer tm.memMu.RUnlock()
	
	result := make([]types.MemoryItem, 0, len(tm.memories))
	for _, memory := range tm.memories {
		result = append(result, memory)
	}
	
	return result, nil
}

// Update updates an existing memory
func (tm *TextualMemory) Update(ctx context.Context, id string, memory types.MemoryItem) error {
	if tm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	textItem, ok := memory.(*types.TextualMemoryItem)
	if !ok {
		return errors.NewValidationError("invalid memory item type for textual memory")
	}
	
	tm.memMu.Lock()
	defer tm.memMu.Unlock()
	
	if _, exists := tm.memories[id]; !exists {
		return errors.NewMemoryNotFoundError(id)
	}
	
	textItem.ID = id
	textItem.UpdatedAt = time.Now()
	tm.memories[id] = textItem
	
	tm.logger.Info("Updated textual memory", map[string]interface{}{
		"memory_id": id,
	})
	
	if tm.metrics != nil {
		tm.metrics.Counter("textual_memory_update_count", 1, nil)
	}
	
	return nil
}

// Delete deletes a memory by ID
func (tm *TextualMemory) Delete(ctx context.Context, id string) error {
	if tm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	tm.memMu.Lock()
	defer tm.memMu.Unlock()
	
	if _, exists := tm.memories[id]; !exists {
		return errors.NewMemoryNotFoundError(id)
	}
	
	delete(tm.memories, id)
	
	tm.logger.Info("Deleted textual memory", map[string]interface{}{
		"memory_id": id,
	})
	
	if tm.metrics != nil {
		tm.metrics.Counter("textual_memory_delete_count", 1, nil)
	}
	
	return nil
}

// DeleteAll deletes all memories
func (tm *TextualMemory) DeleteAll(ctx context.Context) error {
	if tm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	tm.memMu.Lock()
	defer tm.memMu.Unlock()
	
	count := len(tm.memories)
	tm.memories = make(map[string]*types.TextualMemoryItem)
	
	tm.logger.Info("Deleted all textual memories", map[string]interface{}{
		"count": count,
	})
	
	if tm.metrics != nil {
		tm.metrics.Counter("textual_memory_delete_all_count", float64(count), nil)
	}
	
	return nil
}

// Search searches for memories based on query
func (tm *TextualMemory) Search(ctx context.Context, query string, topK int) ([]types.MemoryItem, error) {
	if tm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	start := time.Now()
	defer func() {
		if tm.metrics != nil {
			tm.metrics.Timer("textual_memory_search_duration", float64(time.Since(start).Milliseconds()), nil)
		}
	}()
	
	// For now, implement simple string matching
	// TODO: Implement semantic search with embeddings
	return tm.simpleSearch(ctx, query, topK)
}

// SearchSemantic performs semantic search on textual memories
func (tm *TextualMemory) SearchSemantic(ctx context.Context, query string, topK int, filters map[string]string) ([]*types.TextualMemoryItem, error) {
	if tm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	// TODO: Implement semantic search using embedder and vector DB
	// For now, fall back to simple search
	items, err := tm.simpleSearch(ctx, query, topK)
	if err != nil {
		return nil, err
	}
	
	result := make([]*types.TextualMemoryItem, len(items))
	for i, item := range items {
		result[i] = item.(*types.TextualMemoryItem)
	}
	
	return result, nil
}

// simpleSearch implements simple string-based search
func (tm *TextualMemory) simpleSearch(ctx context.Context, query string, topK int) ([]types.MemoryItem, error) {
	tm.memMu.RLock()
	defer tm.memMu.RUnlock()
	
	var matches []types.MemoryItem
	var scores []float64
	
	for _, memory := range tm.memories {
		// Simple substring matching with basic scoring
		if contains(memory.Memory, query) {
			matches = append(matches, memory)
			// Simple scoring based on query position and length
			score := calculateSimpleScore(memory.Memory, query)
			scores = append(scores, score)
		}
	}
	
	// Sort by score (highest first)
	if len(matches) > 1 {
		sortMemoriesByScore(matches, scores)
	}
	
	// Limit results to topK
	if topK > 0 && len(matches) > topK {
		matches = matches[:topK]
	}
	
	tm.logger.Info("Searched textual memories", map[string]interface{}{
		"query":   query,
		"matches": len(matches),
		"top_k":   topK,
	})
	
	if tm.metrics != nil {
		tm.metrics.Counter("textual_memory_search_count", 1, nil)
		tm.metrics.Gauge("textual_memory_search_results", float64(len(matches)), nil)
	}
	
	return matches, nil
}

// calculateSimpleScore calculates a simple relevance score
func calculateSimpleScore(text, query string) float64 {
	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)
	
	// Exact match gets highest score
	if lowerText == lowerQuery {
		return 1.0
	}
	
	// Check if query appears at the beginning
	if strings.HasPrefix(lowerText, lowerQuery) {
		return 0.9
	}
	
	// Check frequency of query words
	queryWords := strings.Fields(lowerQuery)
	matchCount := 0
	for _, word := range queryWords {
		if strings.Contains(lowerText, word) {
			matchCount++
		}
	}
	
	if len(queryWords) > 0 {
		return float64(matchCount) / float64(len(queryWords)) * 0.8
	}
	
	return 0.1
}

// sortMemoriesByScore sorts memories by their scores in descending order
func sortMemoriesByScore(memories []types.MemoryItem, scores []float64) {
	// Simple bubble sort for small arrays
	for i := 0; i < len(memories)-1; i++ {
		for j := 0; j < len(memories)-i-1; j++ {
			if scores[j] < scores[j+1] {
				// Swap memories
				memories[j], memories[j+1] = memories[j+1], memories[j]
				// Swap scores
				scores[j], scores[j+1] = scores[j+1], scores[j]
			}
		}
	}
}

// contains performs case-insensitive substring matching
func contains(text, query string) bool {
	// Simple case-insensitive search
	return len(query) <= len(text) && 
		   strings.Contains(strings.ToLower(text), strings.ToLower(query))
}

// SetEmbedder sets the embedder for semantic search
func (tm *TextualMemory) SetEmbedder(embedder interfaces.Embedder) {
	tm.embedder = embedder
}

// SetVectorDB sets the vector database for semantic search
func (tm *TextualMemory) SetVectorDB(vectorDB interfaces.VectorDB) {
	tm.vectorDB = vectorDB
}

// ActivationMemory implements activation memory storage
type ActivationMemory struct {
	*BaseMemory
	memories map[string]*types.ActivationMemoryItem
	memMu    sync.RWMutex
}

// NewActivationMemory creates a new activation memory
func NewActivationMemory(cfg *config.MemoryConfig, logger interfaces.Logger, metrics interfaces.Metrics) (*ActivationMemory, error) {
	am := &ActivationMemory{
		BaseMemory: NewBaseMemory(cfg, logger, metrics),
		memories:   make(map[string]*types.ActivationMemoryItem),
	}
	
	return am, nil
}

// Load loads memories from a directory
func (am *ActivationMemory) Load(ctx context.Context, dir string) error {
	if am.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	// TODO: Implement activation memory loading
	// This might involve loading binary files or pickle equivalents
	am.logger.Info("Loading activation memories", map[string]interface{}{
		"dir": dir,
	})
	
	return nil
}

// Dump saves memories to a directory
func (am *ActivationMemory) Dump(ctx context.Context, dir string) error {
	if am.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	// TODO: Implement activation memory dumping
	am.logger.Info("Dumping activation memories", map[string]interface{}{
		"dir": dir,
	})
	
	return nil
}

// Add adds new memories
func (am *ActivationMemory) Add(ctx context.Context, memories []types.MemoryItem) error {
	if am.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	am.memMu.Lock()
	defer am.memMu.Unlock()
	
	for _, item := range memories {
		actItem, ok := item.(*types.ActivationMemoryItem)
		if !ok {
			return errors.NewValidationError("invalid memory item type for activation memory")
		}
		
		if actItem.ID == "" {
			actItem.ID = am.generateID()
		}
		
		if actItem.CreatedAt.IsZero() {
			actItem.CreatedAt = time.Now()
		}
		actItem.UpdatedAt = time.Now()
		
		am.memories[actItem.ID] = actItem
	}
	
	am.logger.Info("Added activation memories", map[string]interface{}{
		"count": len(memories),
	})
	
	return nil
}

// Extract extracts activation memory based on query
func (am *ActivationMemory) Extract(ctx context.Context, query string) (*types.ActivationMemoryItem, error) {
	if am.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	// TODO: Implement activation memory extraction logic
	am.logger.Info("Extracting activation memory", map[string]interface{}{
		"query": query,
	})
	
	return nil, nil
}

// Store stores activation memory
func (am *ActivationMemory) Store(ctx context.Context, item *types.ActivationMemoryItem) error {
	if am.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	return am.Add(ctx, []types.MemoryItem{item})
}

// GetCache retrieves cached activations
func (am *ActivationMemory) GetCache(ctx context.Context, key string) (*types.ActivationMemoryItem, error) {
	if am.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	am.memMu.RLock()
	defer am.memMu.RUnlock()
	
	memory, exists := am.memories[key]
	if !exists {
		return nil, errors.NewMemoryNotFoundError(key)
	}
	
	return memory, nil
}

// Get retrieves a specific memory by ID
func (am *ActivationMemory) Get(ctx context.Context, id string) (types.MemoryItem, error) {
	if am.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	am.memMu.RLock()
	defer am.memMu.RUnlock()
	
	memory, exists := am.memories[id]
	if !exists {
		return nil, errors.NewMemoryNotFoundError(id)
	}
	
	return memory, nil
}

// GetAll retrieves all memories
func (am *ActivationMemory) GetAll(ctx context.Context) ([]types.MemoryItem, error) {
	if am.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	am.memMu.RLock()
	defer am.memMu.RUnlock()
	
	result := make([]types.MemoryItem, 0, len(am.memories))
	for _, memory := range am.memories {
		result = append(result, memory)
	}
	
	return result, nil
}

// Update updates an existing memory
func (am *ActivationMemory) Update(ctx context.Context, id string, memory types.MemoryItem) error {
	if am.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	actItem, ok := memory.(*types.ActivationMemoryItem)
	if !ok {
		return errors.NewValidationError("invalid memory item type for activation memory")
	}
	
	am.memMu.Lock()
	defer am.memMu.Unlock()
	
	if _, exists := am.memories[id]; !exists {
		return errors.NewMemoryNotFoundError(id)
	}
	
	actItem.ID = id
	actItem.UpdatedAt = time.Now()
	am.memories[id] = actItem
	
	return nil
}

// Delete deletes a memory by ID
func (am *ActivationMemory) Delete(ctx context.Context, id string) error {
	if am.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	am.memMu.Lock()
	defer am.memMu.Unlock()
	
	if _, exists := am.memories[id]; !exists {
		return errors.NewMemoryNotFoundError(id)
	}
	
	delete(am.memories, id)
	return nil
}

// DeleteAll deletes all memories
func (am *ActivationMemory) DeleteAll(ctx context.Context) error {
	if am.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	am.memMu.Lock()
	defer am.memMu.Unlock()
	
	am.memories = make(map[string]*types.ActivationMemoryItem)
	return nil
}

// Search searches for memories based on query
func (am *ActivationMemory) Search(ctx context.Context, query string, topK int) ([]types.MemoryItem, error) {
	if am.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	// TODO: Implement activation memory search
	return nil, nil
}

// ParametricMemory implements parametric memory storage
type ParametricMemory struct {
	*BaseMemory
	memories map[string]*types.ParametricMemoryItem
	memMu    sync.RWMutex
}

// NewParametricMemory creates a new parametric memory
func NewParametricMemory(cfg *config.MemoryConfig, logger interfaces.Logger, metrics interfaces.Metrics) (*ParametricMemory, error) {
	pm := &ParametricMemory{
		BaseMemory: NewBaseMemory(cfg, logger, metrics),
		memories:   make(map[string]*types.ParametricMemoryItem),
	}
	
	return pm, nil
}

// Implementation methods for ParametricMemory would follow similar patterns
// to TextualMemory and ActivationMemory...

// Load loads memories from a directory
func (pm *ParametricMemory) Load(ctx context.Context, dir string) error {
	if pm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	// TODO: Implement parametric memory loading
	pm.logger.Info("Loading parametric memories", map[string]interface{}{
		"dir": dir,
	})
	
	return nil
}

// Dump saves memories to a directory
func (pm *ParametricMemory) Dump(ctx context.Context, dir string) error {
	if pm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	// TODO: Implement parametric memory dumping
	pm.logger.Info("Dumping parametric memories", map[string]interface{}{
		"dir": dir,
	})
	
	return nil
}

// LoadAdapter loads a parametric adapter
func (pm *ParametricMemory) LoadAdapter(ctx context.Context, path string) (*types.ParametricMemoryItem, error) {
	if pm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	// TODO: Implement adapter loading
	return nil, nil
}

// SaveAdapter saves a parametric adapter
func (pm *ParametricMemory) SaveAdapter(ctx context.Context, item *types.ParametricMemoryItem, path string) error {
	if pm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	// TODO: Implement adapter saving
	return nil
}

// ApplyAdapter applies an adapter to a model
func (pm *ParametricMemory) ApplyAdapter(ctx context.Context, adapterID string, modelID string) error {
	if pm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	// TODO: Implement adapter application
	return nil
}

// Add adds new memories
func (pm *ParametricMemory) Add(ctx context.Context, memories []types.MemoryItem) error {
	if pm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	start := time.Now()
	defer func() {
		if pm.metrics != nil {
			pm.metrics.Timer("parametric_memory_add_duration", float64(time.Since(start).Milliseconds()), nil)
		}
	}()
	
	pm.memMu.Lock()
	defer pm.memMu.Unlock()
	
	for _, item := range memories {
		paraItem, ok := item.(*types.ParametricMemoryItem)
		if !ok {
			return errors.NewValidationError("invalid memory item type for parametric memory")
		}
		
		if paraItem.ID == "" {
			paraItem.ID = pm.generateID()
		}
		
		if paraItem.CreatedAt.IsZero() {
			paraItem.CreatedAt = time.Now()
		}
		paraItem.UpdatedAt = time.Now()
		
		pm.memories[paraItem.ID] = paraItem
	}
	
	pm.logger.Info("Added parametric memories", map[string]interface{}{
		"count": len(memories),
	})
	
	return nil
}

// Get retrieves a specific memory by ID
func (pm *ParametricMemory) Get(ctx context.Context, id string) (types.MemoryItem, error) {
	if pm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	pm.memMu.RLock()
	defer pm.memMu.RUnlock()
	
	memory, exists := pm.memories[id]
	if !exists {
		return nil, errors.NewMemoryNotFoundError(id)
	}
	
	return memory, nil
}

// GetAll retrieves all memories
func (pm *ParametricMemory) GetAll(ctx context.Context) ([]types.MemoryItem, error) {
	if pm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	pm.memMu.RLock()
	defer pm.memMu.RUnlock()
	
	result := make([]types.MemoryItem, 0, len(pm.memories))
	for _, memory := range pm.memories {
		result = append(result, memory)
	}
	
	return result, nil
}

// Update updates an existing memory
func (pm *ParametricMemory) Update(ctx context.Context, id string, memory types.MemoryItem) error {
	if pm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	paraItem, ok := memory.(*types.ParametricMemoryItem)
	if !ok {
		return errors.NewValidationError("invalid memory item type for parametric memory")
	}
	
	pm.memMu.Lock()
	defer pm.memMu.Unlock()
	
	if _, exists := pm.memories[id]; !exists {
		return errors.NewMemoryNotFoundError(id)
	}
	
	paraItem.ID = id
	paraItem.UpdatedAt = time.Now()
	pm.memories[id] = paraItem
	
	return nil
}

// Delete deletes a memory by ID
func (pm *ParametricMemory) Delete(ctx context.Context, id string) error {
	if pm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	pm.memMu.Lock()
	defer pm.memMu.Unlock()
	
	if _, exists := pm.memories[id]; !exists {
		return errors.NewMemoryNotFoundError(id)
	}
	
	delete(pm.memories, id)
	return nil
}

// DeleteAll deletes all memories
func (pm *ParametricMemory) DeleteAll(ctx context.Context) error {
	if pm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	pm.memMu.Lock()
	defer pm.memMu.Unlock()
	
	pm.memories = make(map[string]*types.ParametricMemoryItem)
	return nil
}

// Search searches for memories based on query
func (pm *ParametricMemory) Search(ctx context.Context, query string, topK int) ([]types.MemoryItem, error) {
	if pm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	// TODO: Implement parametric memory search
	// This might involve searching by adapter names, model types, etc.
	return nil, nil
}