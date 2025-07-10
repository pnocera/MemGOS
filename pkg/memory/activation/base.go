// Package activation provides activation memory implementations for MemGOS
package activation

import (
	"context"
	"encoding/json"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// ActivationMemoryInterface defines the interface for activation memory implementations
type ActivationMemoryInterface interface {
	// Core operations
	Add(ctx context.Context, memories []*types.ActivationMemoryItem) error
	Get(ctx context.Context, id string) (*types.ActivationMemoryItem, error)
	GetByIDs(ctx context.Context, ids []string) ([]*types.ActivationMemoryItem, error)
	GetAll(ctx context.Context) ([]*types.ActivationMemoryItem, error)
	Update(ctx context.Context, id string, memory *types.ActivationMemoryItem) error
	Delete(ctx context.Context, ids []string) error
	DeleteAll(ctx context.Context) error

	// Activation-specific operations
	Extract(ctx context.Context, text string) (*types.ActivationMemoryItem, error)
	Store(ctx context.Context, item *types.ActivationMemoryItem) error
	GetCache(ctx context.Context, key string) (*types.ActivationMemoryItem, error)
	MergeCache(ctx context.Context, cacheIDs []string) (*types.ActivationMemoryItem, error)
	
	// Model state operations
	SaveModelState(ctx context.Context, modelID string, state interface{}) error
	LoadModelState(ctx context.Context, modelID string) (interface{}, error)
	
	// Performance operations
	Compress(ctx context.Context, compressionRatio float64) error
	Optimize(ctx context.Context) error
	
	// Persistence
	Load(ctx context.Context, dir string) error
	Dump(ctx context.Context, dir string) error
	
	// Lifecycle
	Close() error
}

// KVCacheData represents key-value cache data
type KVCacheData struct {
	Keys   [][]float32 `json:"keys"`   // Key tensors for each layer
	Values [][]float32 `json:"values"` // Value tensors for each layer
	Layers int         `json:"layers"` // Number of layers
	SeqLen int         `json:"seq_len"` // Sequence length
	HeadDim int        `json:"head_dim"` // Head dimension
	NumHeads int       `json:"num_heads"` // Number of attention heads
}

// ActivationItem represents a neural activation item
type ActivationItem struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // "kv_cache", "hidden_state", "attention"
	Data        interface{}            `json:"data"`
	ModelID     string                 `json:"model_id"`
	LayerID     string                 `json:"layer_id"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Size        int64                  `json:"size"`        // Size in bytes
	Compressed  bool                   `json:"compressed"`  // Whether data is compressed
	Device      string                 `json:"device"`      // CPU, GPU, etc.
}

// CacheStats represents cache statistics
type CacheStats struct {
	TotalItems      int           `json:"total_items"`
	TotalSize       int64         `json:"total_size"`
	CachedModels    []string      `json:"cached_models"`
	AverageSize     int64         `json:"average_size"`
	CompressionRatio float64      `json:"compression_ratio"`
	HitRate         float64       `json:"hit_rate"`
	LastAccess      time.Time     `json:"last_access"`
	CacheTypes      map[string]int `json:"cache_types"`
}

// BaseActivationMemory provides common functionality for activation memory implementations
type BaseActivationMemory struct {
	llm         interfaces.LLM
	logger      interfaces.Logger
	metrics     interfaces.Metrics
	cacheStats  *CacheStats
	closed      bool
	deviceManager *DeviceManager
	compressor  *ActivationCompressor
}

// DeviceManager handles device allocation and data movement
type DeviceManager struct {
	availableDevices []string
	deviceMemory     map[string]int64
	allocations      map[string]string // itemID -> device
}

// ActivationCompressor handles compression and decompression
type ActivationCompressor struct {
	compressionLevel int
	algorithm        string
}

// NewBaseActivationMemory creates a new base activation memory
func NewBaseActivationMemory(llm interfaces.LLM, logger interfaces.Logger, metrics interfaces.Metrics) *BaseActivationMemory {
	deviceManager := &DeviceManager{
		availableDevices: []string{"cpu"},
		deviceMemory:     map[string]int64{"cpu": 8 * 1024 * 1024 * 1024}, // 8GB default
		allocations:      make(map[string]string),
	}
	
	compressor := &ActivationCompressor{
		compressionLevel: 6,
		algorithm:        "zstd",
	}
	
	cacheStats := &CacheStats{
		CacheTypes: make(map[string]int),
	}
	
	return &BaseActivationMemory{
		llm:           llm,
		logger:        logger,
		metrics:       metrics,
		cacheStats:    cacheStats,
		deviceManager: deviceManager,
		compressor:    compressor,
	}
}

// Close marks the memory as closed
func (bam *BaseActivationMemory) Close() error {
	if bam.closed {
		return nil
	}
	bam.closed = true
	if bam.logger != nil {
		bam.logger.Info("Activation memory closed")
	}
	return nil
}

// IsClosed returns true if the memory is closed
func (bam *BaseActivationMemory) IsClosed() bool {
	return bam.closed
}

// GetLLM returns the LLM
func (bam *BaseActivationMemory) GetLLM() interfaces.LLM {
	return bam.llm
}

// GetStats returns cache statistics
func (bam *BaseActivationMemory) GetStats() *CacheStats {
	return bam.cacheStats
}

// UpdateStats updates cache statistics
func (bam *BaseActivationMemory) UpdateStats(itemType string, size int64, operation string) {
	if bam.cacheStats == nil {
		return
	}
	
	switch operation {
	case "add":
		bam.cacheStats.TotalItems++
		bam.cacheStats.TotalSize += size
		bam.cacheStats.CacheTypes[itemType]++
		if bam.cacheStats.TotalItems > 0 {
			bam.cacheStats.AverageSize = bam.cacheStats.TotalSize / int64(bam.cacheStats.TotalItems)
		}
	case "remove":
		bam.cacheStats.TotalItems--
		bam.cacheStats.TotalSize -= size
		bam.cacheStats.CacheTypes[itemType]--
		if bam.cacheStats.CacheTypes[itemType] <= 0 {
			delete(bam.cacheStats.CacheTypes, itemType)
		}
		if bam.cacheStats.TotalItems > 0 {
			bam.cacheStats.AverageSize = bam.cacheStats.TotalSize / int64(bam.cacheStats.TotalItems)
		}
	case "access":
		bam.cacheStats.LastAccess = time.Now()
		bam.cacheStats.HitRate = (bam.cacheStats.HitRate*0.9 + 1.0*0.1) // Exponential moving average
	}
}

// AllocateDevice allocates an item to a device
func (dm *DeviceManager) AllocateDevice(itemID string, size int64) string {
	// Simple allocation strategy - choose device with most available memory
	bestDevice := "cpu"
	maxAvailable := int64(0)
	
	for device, totalMemory := range dm.deviceMemory {
		used := dm.getUsedMemory(device)
		available := totalMemory - used
		if available >= size && available > maxAvailable {
			maxAvailable = available
			bestDevice = device
		}
	}
	
	dm.allocations[itemID] = bestDevice
	return bestDevice
}

// GetDeviceForItem returns the device for an item
func (dm *DeviceManager) GetDeviceForItem(itemID string) string {
	if device, exists := dm.allocations[itemID]; exists {
		return device
	}
	return "cpu"
}

// FreeDevice frees device allocation for an item
func (dm *DeviceManager) FreeDevice(itemID string) {
	delete(dm.allocations, itemID)
}

// getUsedMemory calculates used memory for a device
func (dm *DeviceManager) getUsedMemory(device string) int64 {
	// Simplified calculation
	used := int64(0)
	for _, allocatedDevice := range dm.allocations {
		if allocatedDevice == device {
			used += 1024 * 1024 // Assume 1MB per allocation for simplicity
		}
	}
	return used
}

// CompressData compresses activation data
func (ac *ActivationCompressor) CompressData(data interface{}) ([]byte, error) {
	// Simple JSON compression for now
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	
	// In a real implementation, this would use zstd or similar
	return jsonData, nil
}

// DecompressData decompresses activation data
func (ac *ActivationCompressor) DecompressData(compressedData []byte, target interface{}) error {
	// Simple JSON decompression for now
	return json.Unmarshal(compressedData, target)
}

// GetCompressionRatio estimates compression ratio
func (ac *ActivationCompressor) GetCompressionRatio() float64 {
	// Simple estimation - in practice this would be calculated from actual compression
	switch ac.algorithm {
	case "zstd":
		return 0.3 // 30% of original size
	case "lz4":
		return 0.5 // 50% of original size
	default:
		return 1.0 // No compression
	}
}

// MoveToDevice simulates moving data to a different device
func (bam *BaseActivationMemory) MoveToDevice(item *types.ActivationMemoryItem, targetDevice string) error {
	// Update device in metadata
	if item.Metadata == nil {
		item.Metadata = make(map[string]interface{})
	}
	
	sourceDevice := "cpu"
	if device, exists := item.Metadata["device"]; exists {
		sourceDevice = device.(string)
	}
	
	// Simulate device transfer
	if sourceDevice != targetDevice {
		bam.logger.Info("Moving activation memory to device", map[string]interface{}{
			"item_id":       item.ID,
			"source_device": sourceDevice,
			"target_device": targetDevice,
		})
		
		item.Metadata["device"] = targetDevice
		item.UpdatedAt = time.Now()
		
		// Update device allocations
		bam.deviceManager.FreeDevice(item.ID)
		bam.deviceManager.allocations[item.ID] = targetDevice
	}
	
	return nil
}

// EstimateSize estimates the size of activation data
func (bam *BaseActivationMemory) EstimateSize(data interface{}) int64 {
	// Simple size estimation
	jsonData, err := json.Marshal(data)
	if err != nil {
		return 0
	}
	return int64(len(jsonData))
}

// OptimizeMemoryLayout optimizes memory layout for better performance
func (bam *BaseActivationMemory) OptimizeMemoryLayout() error {
	// Placeholder for memory layout optimization
	if bam.logger != nil {
		bam.logger.Info("Optimizing activation memory layout")
	}
	return nil
}

// GenerateID generates a unique ID for activation items
func (bam *BaseActivationMemory) GenerateID() string {
	return "act_" + time.Now().Format("20060102150405") + "_" + generateRandomString(8)
}

// generateRandomString generates a random string of specified length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}