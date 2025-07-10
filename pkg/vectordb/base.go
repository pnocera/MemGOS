package vectordb

import (
	"context"
	"time"

	"github.com/memtensor/memgos/pkg/types"
)

// BaseVectorDB defines the common interface for all vector database implementations
type BaseVectorDB interface {
	// Collection management
	CreateCollection(ctx context.Context, config *CollectionConfig) error
	ListCollections(ctx context.Context) ([]string, error)
	DeleteCollection(ctx context.Context, collectionName string) error
	CollectionExists(ctx context.Context, collectionName string) (bool, error)
	GetCollectionInfo(ctx context.Context, collectionName string) (*CollectionInfo, error)

	// Vector operations
	Search(ctx context.Context, collectionName string, vector []float32, limit int, filters map[string]interface{}) (VectorDBItemList, error)
	Add(ctx context.Context, collectionName string, items VectorDBItemList) error
	Update(ctx context.Context, collectionName string, items VectorDBItemList) error
	Upsert(ctx context.Context, collectionName string, items VectorDBItemList) error
	Delete(ctx context.Context, collectionName string, ids []string) error

	// Retrieval operations
	GetByID(ctx context.Context, collectionName string, id string) (*VectorDBItem, error)
	GetByIDs(ctx context.Context, collectionName string, ids []string) (VectorDBItemList, error)
	GetByFilter(ctx context.Context, collectionName string, filters map[string]interface{}) (VectorDBItemList, error)
	GetAll(ctx context.Context, collectionName string, limit int, offset int) (VectorDBItemList, error)
	Count(ctx context.Context, collectionName string, filters map[string]interface{}) (int64, error)

	// Batch operations
	BatchAdd(ctx context.Context, collectionName string, items VectorDBItemList, batchSize int) error
	BatchUpdate(ctx context.Context, collectionName string, items VectorDBItemList, batchSize int) error
	BatchDelete(ctx context.Context, collectionName string, ids []string, batchSize int) error

	// Health and stats
	HealthCheck(ctx context.Context) error
	GetStats(ctx context.Context, collectionName string) (*DatabaseStats, error)
	
	// Connection management
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	IsConnected() bool
	
	// Configuration
	GetConfig() *VectorDBConfig
}

// CollectionConfig represents configuration for creating a collection
type CollectionConfig struct {
	Name          string `json:"name"`
	VectorSize    int    `json:"vector_size"`
	Distance      string `json:"distance"`      // "cosine", "euclidean", "dot"
	OnDiskPayload bool   `json:"on_disk_payload"`
	Quantization  *QuantizationConfig `json:"quantization,omitempty"`
}

// CollectionInfo represents information about a collection
type CollectionInfo struct {
	Name            string    `json:"name"`
	VectorSize      int       `json:"vector_size"`
	Distance        string    `json:"distance"`
	PointsCount     int64     `json:"points_count"`
	IndexedVectors  int64     `json:"indexed_vectors"`
	PayloadSchema   map[string]interface{} `json:"payload_schema"`
	Status          string    `json:"status"`
	OptimizerStatus string    `json:"optimizer_status"`
	CreatedAt       time.Time `json:"created_at"`
}

// DatabaseStats represents database statistics
type DatabaseStats struct {
	CollectionName    string            `json:"collection_name"`
	PointsCount       int64             `json:"points_count"`
	IndexedVectors    int64             `json:"indexed_vectors"`
	MemoryUsage       int64             `json:"memory_usage_bytes"`
	DiskUsage         int64             `json:"disk_usage_bytes"`
	SegmentsCount     int               `json:"segments_count"`
	PayloadSchema     map[string]string `json:"payload_schema"`
	OptimizationStats map[string]interface{} `json:"optimization_stats"`
}

// SearchOptions represents options for vector search
type SearchOptions struct {
	TopK           int                    `json:"top_k"`
	ScoreThreshold *float32               `json:"score_threshold,omitempty"`
	Filters        map[string]interface{} `json:"filters,omitempty"`
	WithPayload    bool                   `json:"with_payload"`
	WithVector     bool                   `json:"with_vector"`
	Offset         int                    `json:"offset"`
}

// DefaultSearchOptions returns default search options
func DefaultSearchOptions() *SearchOptions {
	return &SearchOptions{
		TopK:        10,
		WithPayload: true,
		WithVector:  false,
		Offset:      0,
	}
}

// BatchOperation represents a batch operation
type BatchOperation struct {
	Type        string        `json:"type"` // "add", "update", "upsert", "delete"
	Items       VectorDBItemList `json:"items,omitempty"`
	IDs         []string      `json:"ids,omitempty"`
	Collection  string        `json:"collection"`
}

// BatchResult represents the result of a batch operation
type BatchResult struct {
	SuccessCount int      `json:"success_count"`
	FailureCount int      `json:"failure_count"`
	Errors       []string `json:"errors,omitempty"`
	ProcessedIDs []string `json:"processed_ids,omitempty"`
}

// VectorSearchFilter represents advanced search filters
type VectorSearchFilter struct {
	Must    []FilterCondition `json:"must,omitempty"`
	MustNot []FilterCondition `json:"must_not,omitempty"`
	Should  []FilterCondition `json:"should,omitempty"`
}

// FilterCondition represents a single filter condition
type FilterCondition struct {
	Key      string      `json:"key"`
	Operator string      `json:"operator"` // "eq", "ne", "gt", "gte", "lt", "lte", "in", "nin", "exists"
	Value    interface{} `json:"value"`
}

// DistanceMetric represents different distance metrics
type DistanceMetric string

const (
	DistanceCosine    DistanceMetric = "cosine"
	DistanceEuclidean DistanceMetric = "euclidean"
	DistanceDot       DistanceMetric = "dot"
)

// ConnectionStatus represents the connection status
type ConnectionStatus string

const (
	ConnectionStatusConnected    ConnectionStatus = "connected"
	ConnectionStatusDisconnected ConnectionStatus = "disconnected"
	ConnectionStatusConnecting   ConnectionStatus = "connecting"
	ConnectionStatusError        ConnectionStatus = "error"
)

// VectorOperation represents different vector operations
type VectorOperation string

const (
	OperationAdd    VectorOperation = "add"
	OperationUpdate VectorOperation = "update"
	OperationUpsert VectorOperation = "upsert"
	OperationDelete VectorOperation = "delete"
	OperationSearch VectorOperation = "search"
)

// BaseVectorDBImpl provides a base implementation with common functionality
type BaseVectorDBImpl struct {
	config           *VectorDBConfig
	connectionStatus ConnectionStatus
	lastHealthCheck  time.Time
	metrics          map[string]interface{}
}

// NewBaseVectorDBImpl creates a new base vector database implementation
func NewBaseVectorDBImpl(config *VectorDBConfig) *BaseVectorDBImpl {
	return &BaseVectorDBImpl{
		config:           config,
		connectionStatus: ConnectionStatusDisconnected,
		metrics:          make(map[string]interface{}),
	}
}

// GetConfig returns the configuration
func (b *BaseVectorDBImpl) GetConfig() *VectorDBConfig {
	return b.config
}

// IsConnected returns the connection status
func (b *BaseVectorDBImpl) IsConnected() bool {
	return b.connectionStatus == ConnectionStatusConnected
}

// GetConnectionStatus returns the current connection status
func (b *BaseVectorDBImpl) GetConnectionStatus() ConnectionStatus {
	return b.connectionStatus
}

// SetConnectionStatus sets the connection status
func (b *BaseVectorDBImpl) SetConnectionStatus(status ConnectionStatus) {
	b.connectionStatus = status
}

// UpdateMetrics updates the metrics
func (b *BaseVectorDBImpl) UpdateMetrics(key string, value interface{}) {
	b.metrics[key] = value
}

// GetMetrics returns the current metrics
func (b *BaseVectorDBImpl) GetMetrics() map[string]interface{} {
	return b.metrics
}

// SetLastHealthCheck sets the last health check time
func (b *BaseVectorDBImpl) SetLastHealthCheck(t time.Time) {
	b.lastHealthCheck = t
}

// GetLastHealthCheck returns the last health check time
func (b *BaseVectorDBImpl) GetLastHealthCheck() time.Time {
	return b.lastHealthCheck
}

// ValidateCollection validates collection name and configuration
func (b *BaseVectorDBImpl) ValidateCollection(config *CollectionConfig) error {
	if config.Name == "" {
		return &types.MemGOSError{
			Type:    types.ErrorTypeValidation,
			Message: "collection name cannot be empty",
			Code:    "INVALID_COLLECTION_NAME",
		}
	}
	
	if config.VectorSize <= 0 {
		return &types.MemGOSError{
			Type:    types.ErrorTypeValidation,
			Message: "vector size must be greater than 0",
			Code:    "INVALID_VECTOR_SIZE",
		}
	}
	
	validDistances := []string{"cosine", "euclidean", "dot"}
	validDistance := false
	for _, d := range validDistances {
		if config.Distance == d {
			validDistance = true
			break
		}
	}
	if !validDistance {
		return &types.MemGOSError{
			Type:    types.ErrorTypeValidation,
			Message: "invalid distance metric",
			Code:    "INVALID_DISTANCE_METRIC",
			Details: map[string]interface{}{
				"provided": config.Distance,
				"valid":    validDistances,
			},
		}
	}
	
	return nil
}

// ValidateItems validates a list of vector database items
func (b *BaseVectorDBImpl) ValidateItems(items VectorDBItemList) error {
	if len(items) == 0 {
		return &types.MemGOSError{
			Type:    types.ErrorTypeValidation,
			Message: "items list cannot be empty",
			Code:    "EMPTY_ITEMS_LIST",
		}
	}
	
	for i, item := range items {
		if err := item.Validate(); err != nil {
			return &types.MemGOSError{
				Type:    types.ErrorTypeValidation,
				Message: "invalid item",
				Code:    "INVALID_ITEM",
				Details: map[string]interface{}{
					"index": i,
					"error": err.Error(),
				},
			}
		}
	}
	
	return nil
}

// ValidateSearchOptions validates search options
func (b *BaseVectorDBImpl) ValidateSearchOptions(options *SearchOptions) error {
	if options.TopK <= 0 {
		return &types.MemGOSError{
			Type:    types.ErrorTypeValidation,
			Message: "top_k must be greater than 0",
			Code:    "INVALID_TOP_K",
		}
	}
	
	if options.Offset < 0 {
		return &types.MemGOSError{
			Type:    types.ErrorTypeValidation,
			Message: "offset cannot be negative",
			Code:    "INVALID_OFFSET",
		}
	}
	
	return nil
}