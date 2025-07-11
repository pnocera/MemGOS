// Package activation provides KV cache memory implementation
package activation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/errors"
	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// KVCacheMemory implements KV cache memory for activation storage
type KVCacheMemory struct {
	*BaseActivationMemory
	config       *config.MemoryConfig
	memories     map[string]*types.ActivationMemoryItem
	kvCaches     map[string]*KVCacheData
	modelStates  map[string]interface{}
	memMu        sync.RWMutex
	cacheManager *CacheManager
}

// CacheManager handles cache lifecycle and optimization
type CacheManager struct {
	maxCacheSize    int64
	evictionPolicy  string
	accessTimes     map[string]time.Time
	cacheHierarchy  map[string][]string // modelID -> cache IDs
	mu              sync.RWMutex
}

// NewKVCacheMemory creates a new KV cache memory
func NewKVCacheMemory(cfg *config.MemoryConfig, llm interfaces.LLM, logger interfaces.Logger, metrics interfaces.Metrics) (*KVCacheMemory, error) {
	base := NewBaseActivationMemory(llm, logger, metrics)
	
	cacheManager := &CacheManager{
		maxCacheSize:   8 * 1024 * 1024 * 1024, // 8GB default
		evictionPolicy: "lru",
		accessTimes:    make(map[string]time.Time),
		cacheHierarchy: make(map[string][]string),
	}
	
	kvcm := &KVCacheMemory{
		BaseActivationMemory: base,
		config:               cfg,
		memories:             make(map[string]*types.ActivationMemoryItem),
		kvCaches:             make(map[string]*KVCacheData),
		modelStates:          make(map[string]interface{}),
		cacheManager:         cacheManager,
	}
	
	return kvcm, nil
}

// Extract extracts KV cache from text using LLM
func (kvcm *KVCacheMemory) Extract(ctx context.Context, text string) (*types.ActivationMemoryItem, error) {
	if kvcm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	start := time.Now()
	defer func() {
		if kvcm.metrics != nil {
			kvcm.metrics.Timer("kv_cache_extract_duration", float64(time.Since(start).Milliseconds()), nil)
		}
	}()
	
	// Build KV cache using LLM
	kvCache, err := kvcm.buildKVCache(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to build KV cache: %w", err)
	}
	
	// Create activation memory item
	item := &types.ActivationMemoryItem{
		ID:     kvcm.GenerateID(),
		Memory: kvCache,
		Metadata: map[string]interface{}{
			"source_text":   text,
			"extracted_at":  time.Now().Format(time.RFC3339),
			"type":          "kv_cache",
			"device":        "cpu",
			"compressed":    false,
			"model_id":      "default",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	// Store the cache data
	kvcm.kvCaches[item.ID] = kvCache
	
	// Estimate size
	size := kvcm.EstimateSize(kvCache)
	kvcm.UpdateStats("kv_cache", size, "add")
	
	kvcm.logger.Info("Extracted KV cache from text", map[string]interface{}{
		"item_id":    item.ID,
		"text_length": len(text),
		"cache_size":  size,
	})
	
	return item, nil
}

// buildKVCache builds KV cache data from text
func (kvcm *KVCacheMemory) buildKVCache(ctx context.Context, text string) (*KVCacheData, error) {
	// Simulate KV cache building with LLM
	// In a real implementation, this would use the actual LLM to generate cache
	
	// Mock KV cache data
	layers := 12 // Typical transformer layers
	seqLen := len(text) / 4 // Rough approximation
	headDim := 64
	numHeads := 12
	
	kvCache := &KVCacheData{
		Keys:     make([][]float32, layers),
		Values:   make([][]float32, layers),
		Layers:   layers,
		SeqLen:   seqLen,
		HeadDim:  headDim,
		NumHeads: numHeads,
	}
	
	// Generate mock key-value tensors
	for layer := 0; layer < layers; layer++ {
		cacheSize := seqLen * headDim * numHeads
		kvCache.Keys[layer] = make([]float32, cacheSize)
		kvCache.Values[layer] = make([]float32, cacheSize)
		
		// Fill with mock data (in real implementation, this would come from LLM)
		for i := 0; i < cacheSize; i++ {
			kvCache.Keys[layer][i] = float32(i%100) / 100.0
			kvCache.Values[layer][i] = float32((i+50)%100) / 100.0
		}
	}
	
	return kvCache, nil
}

// Add adds activation memories
func (kvcm *KVCacheMemory) Add(ctx context.Context, memories []*types.ActivationMemoryItem) error {
	if kvcm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	start := time.Now()
	defer func() {
		if kvcm.metrics != nil {
			kvcm.metrics.Timer("kv_cache_add_duration", float64(time.Since(start).Milliseconds()), nil)
		}
	}()
	
	kvcm.memMu.Lock()
	defer kvcm.memMu.Unlock()
	
	for _, memory := range memories {
		if memory.ID == "" {
			memory.ID = kvcm.GenerateID()
		}
		
		if memory.CreatedAt.IsZero() {
			memory.CreatedAt = time.Now()
		}
		memory.UpdatedAt = time.Now()
		
		// Store memory
		kvcm.memories[memory.ID] = memory
		
		// Handle KV cache data
		if kvData, ok := memory.Memory.(*KVCacheData); ok {
			kvcm.kvCaches[memory.ID] = kvData
		} else if kvMap, ok := memory.Memory.(map[string]interface{}); ok {
			// Convert from map to KVCacheData
			kvData := &KVCacheData{}
			if err := kvcm.convertMapToKVCache(kvMap, kvData); err != nil {
				kvcm.logger.Error("Failed to convert KV cache data", err, map[string]interface{}{
					"memory_id": memory.ID,
				})
			} else {
				kvcm.kvCaches[memory.ID] = kvData
			}
		}
		
		// Update cache manager
		kvcm.cacheManager.accessTimes[memory.ID] = time.Now()
		
		// Update model hierarchy
		if modelID, exists := memory.Metadata["model_id"]; exists {
			if modelStr, ok := modelID.(string); ok {
				kvcm.cacheManager.mu.Lock()
				kvcm.cacheManager.cacheHierarchy[modelStr] = append(
					kvcm.cacheManager.cacheHierarchy[modelStr], 
					memory.ID,
				)
				kvcm.cacheManager.mu.Unlock()
			}
		}
		
		// Update statistics
		size := kvcm.EstimateSize(memory.Memory)
		kvcm.UpdateStats("kv_cache", size, "add")
		
		// Check if eviction is needed
		if err := kvcm.checkEviction(); err != nil {
			kvcm.logger.Error("Failed to check eviction", err, map[string]interface{}{})
		}
	}
	
	kvcm.logger.Info("Added KV cache memories", map[string]interface{}{
		"count": len(memories),
	})
	
	if kvcm.metrics != nil {
		kvcm.metrics.Counter("kv_cache_add_count", float64(len(memories)), nil)
	}
	
	return nil
}

// Get retrieves a memory by ID
func (kvcm *KVCacheMemory) Get(ctx context.Context, id string) (*types.ActivationMemoryItem, error) {
	if kvcm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	kvcm.memMu.RLock()
	defer kvcm.memMu.RUnlock()
	
	memory, exists := kvcm.memories[id]
	if !exists {
		return nil, errors.NewMemoryNotFoundError(id)
	}
	
	// Update access time
	kvcm.cacheManager.accessTimes[id] = time.Now()
	kvcm.UpdateStats("kv_cache", 0, "access")
	
	return memory, nil
}

// GetCache retrieves cached KV data by key
func (kvcm *KVCacheMemory) GetCache(ctx context.Context, key string) (*types.ActivationMemoryItem, error) {
	return kvcm.Get(ctx, key)
}

// MergeCache merges multiple KV caches into a single cache
func (kvcm *KVCacheMemory) MergeCache(ctx context.Context, cacheIDs []string) (*types.ActivationMemoryItem, error) {
	if kvcm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	kvcm.memMu.RLock()
	defer kvcm.memMu.RUnlock()
	
	// Collect caches to merge
	cachesToMerge := make([]*KVCacheData, 0)
	for _, cacheID := range cacheIDs {
		if kvData, exists := kvcm.kvCaches[cacheID]; exists {
			cachesToMerge = append(cachesToMerge, kvData)
		}
	}
	
	if len(cachesToMerge) == 0 {
		return nil, fmt.Errorf("no valid caches found to merge")
	}
	
	// Merge caches
	mergedCache, err := kvcm.mergeCaches(cachesToMerge)
	if err != nil {
		return nil, fmt.Errorf("failed to merge caches: %w", err)
	}
	
	// Create new memory item for merged cache
	mergedItem := &types.ActivationMemoryItem{
		ID:     kvcm.GenerateID(),
		Memory: mergedCache,
		Metadata: map[string]interface{}{
			"type":         "merged_kv_cache",
			"source_caches": cacheIDs,
			"merged_at":    time.Now().Format(time.RFC3339),
			"device":       "cpu",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	// Store merged cache
	kvcm.memories[mergedItem.ID] = mergedItem
	kvcm.kvCaches[mergedItem.ID] = mergedCache
	
	kvcm.logger.Info("Merged KV caches", map[string]interface{}{
		"merged_id":    mergedItem.ID,
		"source_count": len(cacheIDs),
	})
	
	return mergedItem, nil
}

// mergeCaches merges multiple KV cache data structures
func (kvcm *KVCacheMemory) mergeCaches(caches []*KVCacheData) (*KVCacheData, error) {
	if len(caches) == 0 {
		return nil, fmt.Errorf("no caches to merge")
	}
	
	if len(caches) == 1 {
		return caches[0], nil
	}
	
	// Validate cache compatibility
	firstCache := caches[0]
	for i, cache := range caches[1:] {
		if cache.Layers != firstCache.Layers {
			return nil, fmt.Errorf("cache %d has incompatible layer count: %d vs %d", i+1, cache.Layers, firstCache.Layers)
		}
		if cache.HeadDim != firstCache.HeadDim {
			return nil, fmt.Errorf("cache %d has incompatible head dimension: %d vs %d", i+1, cache.HeadDim, firstCache.HeadDim)
		}
		if cache.NumHeads != firstCache.NumHeads {
			return nil, fmt.Errorf("cache %d has incompatible head count: %d vs %d", i+1, cache.NumHeads, firstCache.NumHeads)
		}
	}
	
	// Calculate total sequence length
	totalSeqLen := 0
	for _, cache := range caches {
		totalSeqLen += cache.SeqLen
	}
	
	// Create merged cache
	merged := &KVCacheData{
		Keys:     make([][]float32, firstCache.Layers),
		Values:   make([][]float32, firstCache.Layers),
		Layers:   firstCache.Layers,
		SeqLen:   totalSeqLen,
		HeadDim:  firstCache.HeadDim,
		NumHeads: firstCache.NumHeads,
	}
	
	// Merge each layer
	for layer := 0; layer < firstCache.Layers; layer++ {
		mergedKeySize := totalSeqLen * firstCache.HeadDim * firstCache.NumHeads
		mergedValueSize := totalSeqLen * firstCache.HeadDim * firstCache.NumHeads
		
		merged.Keys[layer] = make([]float32, mergedKeySize)
		merged.Values[layer] = make([]float32, mergedValueSize)
		
		// Concatenate keys and values from all caches
		keyOffset := 0
		valueOffset := 0
		
		for _, cache := range caches {
			keySize := len(cache.Keys[layer])
			valueSize := len(cache.Values[layer])
			
			copy(merged.Keys[layer][keyOffset:keyOffset+keySize], cache.Keys[layer])
			copy(merged.Values[layer][valueOffset:valueOffset+valueSize], cache.Values[layer])
			
			keyOffset += keySize
			valueOffset += valueSize
		}
	}
	
	return merged, nil
}

// Store stores an activation item
func (kvcm *KVCacheMemory) Store(ctx context.Context, item *types.ActivationMemoryItem) error {
	return kvcm.Add(ctx, []*types.ActivationMemoryItem{item})
}

// SaveModelState saves model state
func (kvcm *KVCacheMemory) SaveModelState(ctx context.Context, modelID string, state interface{}) error {
	if kvcm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	kvcm.memMu.Lock()
	defer kvcm.memMu.Unlock()
	
	kvcm.modelStates[modelID] = state
	
	kvcm.logger.Info("Saved model state", map[string]interface{}{
		"model_id": modelID,
	})
	
	return nil
}

// LoadModelState loads model state
func (kvcm *KVCacheMemory) LoadModelState(ctx context.Context, modelID string) (interface{}, error) {
	if kvcm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	kvcm.memMu.RLock()
	defer kvcm.memMu.RUnlock()
	
	state, exists := kvcm.modelStates[modelID]
	if !exists {
		return nil, fmt.Errorf("model state not found for model %s", modelID)
	}
	
	return state, nil
}

// GetByIDs retrieves multiple memories by IDs
func (kvcm *KVCacheMemory) GetByIDs(ctx context.Context, ids []string) ([]*types.ActivationMemoryItem, error) {
	if kvcm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	kvcm.memMu.RLock()
	defer kvcm.memMu.RUnlock()
	
	result := make([]*types.ActivationMemoryItem, 0, len(ids))
	for _, id := range ids {
		if memory, exists := kvcm.memories[id]; exists {
			result = append(result, memory)
			kvcm.cacheManager.accessTimes[id] = time.Now()
		}
	}
	
	kvcm.UpdateStats("kv_cache", 0, "access")
	
	return result, nil
}

// GetAll retrieves all memories
func (kvcm *KVCacheMemory) GetAll(ctx context.Context) ([]*types.ActivationMemoryItem, error) {
	if kvcm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	kvcm.memMu.RLock()
	defer kvcm.memMu.RUnlock()
	
	result := make([]*types.ActivationMemoryItem, 0, len(kvcm.memories))
	for _, memory := range kvcm.memories {
		result = append(result, memory)
	}
	
	return result, nil
}

// Update updates an existing memory
func (kvcm *KVCacheMemory) Update(ctx context.Context, id string, memory *types.ActivationMemoryItem) error {
	if kvcm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	kvcm.memMu.Lock()
	defer kvcm.memMu.Unlock()
	
	if _, exists := kvcm.memories[id]; !exists {
		return errors.NewMemoryNotFoundError(id)
	}
	
	memory.ID = id
	memory.UpdatedAt = time.Now()
	kvcm.memories[id] = memory
	
	// Update KV cache data if present
	if kvData, ok := memory.Memory.(*KVCacheData); ok {
		kvcm.kvCaches[id] = kvData
	}
	
	kvcm.cacheManager.accessTimes[id] = time.Now()
	
	kvcm.logger.Info("Updated KV cache memory", map[string]interface{}{
		"memory_id": id,
	})
	
	return nil
}

// Delete deletes memories by IDs
func (kvcm *KVCacheMemory) Delete(ctx context.Context, ids []string) error {
	if kvcm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	kvcm.memMu.Lock()
	defer kvcm.memMu.Unlock()
	
	deletedCount := 0
	for _, id := range ids {
		if memory, exists := kvcm.memories[id]; exists {
			// Update statistics
			size := kvcm.EstimateSize(memory.Memory)
			kvcm.UpdateStats("kv_cache", size, "remove")
			
			delete(kvcm.memories, id)
			delete(kvcm.kvCaches, id)
			delete(kvcm.cacheManager.accessTimes, id)
			
			// Remove from cache hierarchy
			if modelID, exists := memory.Metadata["model_id"]; exists {
				if modelStr, ok := modelID.(string); ok {
					kvcm.removeFromCacheHierarchy(modelStr, id)
				}
			}
			
			// Free device allocation
			kvcm.deviceManager.FreeDevice(id)
			
			deletedCount++
		}
	}
	
	kvcm.logger.Info("Deleted KV cache memories", map[string]interface{}{
		"count": deletedCount,
	})
	
	return nil
}

// DeleteAll deletes all memories
func (kvcm *KVCacheMemory) DeleteAll(ctx context.Context) error {
	if kvcm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	kvcm.memMu.Lock()
	defer kvcm.memMu.Unlock()
	
	count := len(kvcm.memories)
	kvcm.memories = make(map[string]*types.ActivationMemoryItem)
	kvcm.kvCaches = make(map[string]*KVCacheData)
	kvcm.modelStates = make(map[string]interface{})
	kvcm.cacheManager.accessTimes = make(map[string]time.Time)
	kvcm.cacheManager.cacheHierarchy = make(map[string][]string)
	
	// Reset statistics
	kvcm.cacheStats = &CacheStats{
		CacheTypes: make(map[string]int),
	}
	
	kvcm.logger.Info("Deleted all KV cache memories", map[string]interface{}{
		"count": count,
	})
	
	return nil
}

// Compress compresses cache data to save memory
func (kvcm *KVCacheMemory) Compress(ctx context.Context, compressionRatio float64) error {
	if kvcm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	kvcm.memMu.Lock()
	defer kvcm.memMu.Unlock()
	
	compressedCount := 0
	
	for id, memory := range kvcm.memories {
		// Skip already compressed items
		if compressed, exists := memory.Metadata["compressed"]; exists {
			if isCompressed, ok := compressed.(bool); ok && isCompressed {
				continue
			}
		}
		
		// Compress the cache data
		compressedData, err := kvcm.compressor.CompressData(memory.Memory)
		if err != nil {
			kvcm.logger.Error("Failed to compress memory", err, map[string]interface{}{
				"memory_id": id,
			})
			continue
		}
		
		// Update memory with compressed data
		memory.Memory = compressedData
		memory.Metadata["compressed"] = true
		memory.Metadata["compression_ratio"] = compressionRatio
		memory.UpdatedAt = time.Now()
		
		compressedCount++
	}
	
	// Update compression ratio in stats
	kvcm.cacheStats.CompressionRatio = compressionRatio
	
	kvcm.logger.Info("Compressed KV cache memories", map[string]interface{}{
		"compressed_count":   compressedCount,
		"compression_ratio":  compressionRatio,
	})
	
	return nil
}

// Optimize optimizes cache organization and performance
func (kvcm *KVCacheMemory) Optimize(ctx context.Context) error {
	if kvcm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	// Optimize memory layout
	if err := kvcm.OptimizeMemoryLayout(); err != nil {
		return err
	}
	
	// Run garbage collection on unused caches
	if err := kvcm.garbageCollect(); err != nil {
		return err
	}
	
	// Optimize device allocations
	if err := kvcm.optimizeDeviceAllocations(); err != nil {
		return err
	}
	
	kvcm.logger.Info("Optimized KV cache memory")
	
	return nil
}

// Load loads memories from directory
func (kvcm *KVCacheMemory) Load(ctx context.Context, dir string) error {
	if kvcm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	filePath := filepath.Join(dir, kvcm.config.MemoryFilename)
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		kvcm.logger.Info("KV cache file not found, starting with empty memories", map[string]interface{}{
			"file_path": filePath,
		})
		return nil
	}
	
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read KV cache file: %w", err)
	}
	
	// Parse JSON
	var saveData map[string]interface{}
	if err := json.Unmarshal(data, &saveData); err != nil {
		return fmt.Errorf("failed to parse KV cache file: %w", err)
	}
	
	// Load memories
	if err := kvcm.loadFromSaveData(saveData); err != nil {
		return fmt.Errorf("failed to load KV cache memories: %w", err)
	}
	
	kvcm.logger.Info("Loaded KV cache memories", map[string]interface{}{
		"count":     len(kvcm.memories),
		"file_path": filePath,
	})
	
	return nil
}

// Dump saves memories to directory
func (kvcm *KVCacheMemory) Dump(ctx context.Context, dir string) error {
	if kvcm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Prepare save data
	saveData := kvcm.prepareForSave()
	
	// Marshal to JSON
	data, err := json.MarshalIndent(saveData, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal KV cache data: %w", err)
	}
	
	// Write to file
	filePath := filepath.Join(dir, kvcm.config.MemoryFilename)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write KV cache file: %w", err)
	}
	
	kvcm.logger.Info("Dumped KV cache memories", map[string]interface{}{
		"count":     len(kvcm.memories),
		"file_path": filePath,
	})
	
	return nil
}

// Helper methods

func (kvcm *KVCacheMemory) convertMapToKVCache(kvMap map[string]interface{}, kvData *KVCacheData) error {
	// Convert map to KVCacheData structure
	if layers, exists := kvMap["layers"]; exists {
		if layersFloat, ok := layers.(float64); ok {
			kvData.Layers = int(layersFloat)
		}
	}
	
	if seqLen, exists := kvMap["seq_len"]; exists {
		if seqLenFloat, ok := seqLen.(float64); ok {
			kvData.SeqLen = int(seqLenFloat)
		}
	}
	
	if headDim, exists := kvMap["head_dim"]; exists {
		if headDimFloat, ok := headDim.(float64); ok {
			kvData.HeadDim = int(headDimFloat)
		}
	}
	
	if numHeads, exists := kvMap["num_heads"]; exists {
		if numHeadsFloat, ok := numHeads.(float64); ok {
			kvData.NumHeads = int(numHeadsFloat)
		}
	}
	
	// Convert keys and values
	if keys, exists := kvMap["keys"]; exists {
		if keysSlice, ok := keys.([]interface{}); ok {
			kvData.Keys = make([][]float32, len(keysSlice))
			for i, keyLayer := range keysSlice {
				if keySlice, ok := keyLayer.([]interface{}); ok {
					kvData.Keys[i] = make([]float32, len(keySlice))
					for j, keyVal := range keySlice {
						if keyFloat, ok := keyVal.(float64); ok {
							kvData.Keys[i][j] = float32(keyFloat)
						}
					}
				}
			}
		}
	}
	
	if values, exists := kvMap["values"]; exists {
		if valuesSlice, ok := values.([]interface{}); ok {
			kvData.Values = make([][]float32, len(valuesSlice))
			for i, valueLayer := range valuesSlice {
				if valueSlice, ok := valueLayer.([]interface{}); ok {
					kvData.Values[i] = make([]float32, len(valueSlice))
					for j, valueVal := range valueSlice {
						if valueFloat, ok := valueVal.(float64); ok {
							kvData.Values[i][j] = float32(valueFloat)
						}
					}
				}
			}
		}
	}
	
	return nil
}

func (kvcm *KVCacheMemory) checkEviction() error {
	// Simple LRU eviction policy
	if kvcm.cacheStats.TotalSize <= kvcm.cacheManager.maxCacheSize {
		return nil
	}
	
	// Find oldest accessed items
	type itemAge struct {
		id         string
		accessTime time.Time
		size       int64
	}
	
	items := make([]*itemAge, 0)
	for id, accessTime := range kvcm.cacheManager.accessTimes {
		if memory, exists := kvcm.memories[id]; exists {
			size := kvcm.EstimateSize(memory.Memory)
			items = append(items, &itemAge{
				id:         id,
				accessTime: accessTime,
				size:       size,
			})
		}
	}
	
	// Sort by access time (oldest first)
	for i := 0; i < len(items)-1; i++ {
		for j := 0; j < len(items)-i-1; j++ {
			if items[j].accessTime.After(items[j+1].accessTime) {
				items[j], items[j+1] = items[j+1], items[j]
			}
		}
	}
	
	// Evict oldest items until under limit
	bytesToEvict := kvcm.cacheStats.TotalSize - kvcm.cacheManager.maxCacheSize
	evictedBytes := int64(0)
	evictedCount := 0
	
	for _, item := range items {
		if evictedBytes >= bytesToEvict {
			break
		}
		
		// Delete item
		delete(kvcm.memories, item.id)
		delete(kvcm.kvCaches, item.id)
		delete(kvcm.cacheManager.accessTimes, item.id)
		
		evictedBytes += item.size
		evictedCount++
		
		kvcm.UpdateStats("kv_cache", item.size, "remove")
	}
	
	if evictedCount > 0 {
		kvcm.logger.Info("Evicted old KV cache memories", map[string]interface{}{
			"evicted_count": evictedCount,
			"evicted_bytes": evictedBytes,
		})
	}
	
	return nil
}

func (kvcm *KVCacheMemory) removeFromCacheHierarchy(modelID, cacheID string) {
	kvcm.cacheManager.mu.Lock()
	defer kvcm.cacheManager.mu.Unlock()
	
	if cacheList, exists := kvcm.cacheManager.cacheHierarchy[modelID]; exists {
		newList := make([]string, 0)
		for _, id := range cacheList {
			if id != cacheID {
				newList = append(newList, id)
			}
		}
		kvcm.cacheManager.cacheHierarchy[modelID] = newList
	}
}

func (kvcm *KVCacheMemory) garbageCollect() error {
	// Remove orphaned cache data
	for id := range kvcm.kvCaches {
		if _, exists := kvcm.memories[id]; !exists {
			delete(kvcm.kvCaches, id)
		}
	}
	
	// Remove orphaned access times
	for id := range kvcm.cacheManager.accessTimes {
		if _, exists := kvcm.memories[id]; !exists {
			delete(kvcm.cacheManager.accessTimes, id)
		}
	}
	
	return nil
}

func (kvcm *KVCacheMemory) optimizeDeviceAllocations() error {
	// Optimize device allocations based on usage patterns
	for id, memory := range kvcm.memories {
		if device, exists := memory.Metadata["device"]; exists {
			if deviceStr, ok := device.(string); ok && deviceStr != "cpu" {
				// Move rarely accessed items to CPU
				if accessTime, exists := kvcm.cacheManager.accessTimes[id]; exists {
					if time.Since(accessTime) > time.Hour {
						if err := kvcm.MoveToDevice(memory, "cpu"); err != nil {
							kvcm.logger.Error("Failed to move memory to CPU", err, map[string]interface{}{
								"memory_id": id,
							})
						}
					}
				}
			}
		}
	}
	
	return nil
}

func (kvcm *KVCacheMemory) loadFromSaveData(saveData map[string]interface{}) error {
	kvcm.memMu.Lock()
	defer kvcm.memMu.Unlock()
	
	// Load memories
	if memoriesData, exists := saveData["kv_cache_memories"]; exists {
		if memoriesMap, ok := memoriesData.(map[string]interface{}); ok {
			for id, memoryData := range memoriesMap {
				if memoryMap, ok := memoryData.(map[string]interface{}); ok {
					memory := &types.ActivationMemoryItem{
						ID: id,
					}
					
					// Parse memory data
					if memoryContent, exists := memoryMap["memory"]; exists {
						memory.Memory = memoryContent
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
					
					kvcm.memories[id] = memory
					
					// Convert and store KV cache data
					if kvData, ok := memory.Memory.(map[string]interface{}); ok {
						kvCacheData := &KVCacheData{}
						if err := kvcm.convertMapToKVCache(kvData, kvCacheData); err == nil {
							kvcm.kvCaches[id] = kvCacheData
						}
					}
				}
			}
		}
	}
	
	return nil
}

func (kvcm *KVCacheMemory) prepareForSave() map[string]interface{} {
	kvcm.memMu.RLock()
	defer kvcm.memMu.RUnlock()
	
	// Prepare memories for saving
	memoriesData := make(map[string]interface{})
	for id, memory := range kvcm.memories {
		memoryData := map[string]interface{}{
			"memory":     memory.Memory,
			"metadata":   memory.Metadata,
			"created_at": memory.CreatedAt.Format(time.RFC3339),
			"updated_at": memory.UpdatedAt.Format(time.RFC3339),
		}
		memoriesData[id] = memoryData
	}
	
	return map[string]interface{}{
		"kv_cache_memories": memoriesData,
		"model_states":      kvcm.modelStates,
		"cache_stats":       kvcm.cacheStats,
		"saved_at":          time.Now().Format(time.RFC3339),
	}
}