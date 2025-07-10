// Package graphdb provides graph database implementations for MemGOS
package graphdb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
)

// GraphDBProvider represents supported graph database providers
type GraphDBProvider string

const (
	ProviderNeo4j GraphDBProvider = "neo4j"
)

// GraphDBConfig holds configuration for graph databases
type GraphDBConfig struct {
	Provider       GraphDBProvider       `json:"provider" yaml:"provider"`
	URI           string                `json:"uri" yaml:"uri"`
	Username      string                `json:"username" yaml:"username"`
	Password      string                `json:"password" yaml:"password"`
	Database      string                `json:"database" yaml:"database"`
	MaxConnPool   int                   `json:"max_conn_pool" yaml:"max_conn_pool"`
	ConnTimeout   time.Duration         `json:"conn_timeout" yaml:"conn_timeout"`
	ReadTimeout   time.Duration         `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout  time.Duration         `json:"write_timeout" yaml:"write_timeout"`
	RetryAttempts int                   `json:"retry_attempts" yaml:"retry_attempts"`
	RetryDelay    time.Duration         `json:"retry_delay" yaml:"retry_delay"`
	Metrics       bool                  `json:"metrics" yaml:"metrics"`
	Logging       bool                  `json:"logging" yaml:"logging"`
	SSLMode       string                `json:"ssl_mode" yaml:"ssl_mode"`
	TLSConfig     map[string]interface{} `json:"tls_config" yaml:"tls_config"`
}

// DefaultGraphDBConfig returns a default configuration
func DefaultGraphDBConfig() *GraphDBConfig {
	return &GraphDBConfig{
		Provider:      ProviderNeo4j,
		URI:          "bolt://localhost:7687",
		Username:     "neo4j",
		Password:     "password",
		Database:     "neo4j",
		MaxConnPool:  50,
		ConnTimeout:  30 * time.Second,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		RetryAttempts: 3,
		RetryDelay:   time.Second,
		Metrics:      true,
		Logging:      false,
		SSLMode:      "disable",
	}
}

// BaseGraphDB provides common functionality for all graph database implementations
type BaseGraphDB struct {
	config  *GraphDBConfig
	logger  interfaces.Logger
	metrics interfaces.Metrics
	mu      sync.RWMutex
	closed  bool

	// Health monitoring
	health struct {
		lastCheck time.Time
		status    string
		error     error
		mu        sync.RWMutex
	}

	// Connection stats
	stats struct {
		totalQueries    int64
		failedQueries   int64
		totalNodes      int64
		totalRelations  int64
		connectionTime  time.Duration
		lastQueryTime   time.Duration
		mu              sync.RWMutex
	}
}

// NewBaseGraphDB creates a new base graph database
func NewBaseGraphDB(config *GraphDBConfig, logger interfaces.Logger, metrics interfaces.Metrics) *BaseGraphDB {
	if config == nil {
		config = DefaultGraphDBConfig()
	}
	
	return &BaseGraphDB{
		config:  config,
		logger:  logger,
		metrics: metrics,
	}
}

// GetConfig returns the database configuration
func (bg *BaseGraphDB) GetConfig() *GraphDBConfig {
	bg.mu.RLock()
	defer bg.mu.RUnlock()
	return bg.config
}

// IsClosed returns true if the database is closed
func (bg *BaseGraphDB) IsClosed() bool {
	bg.mu.RLock()
	defer bg.mu.RUnlock()
	return bg.closed
}

// Close marks the database as closed
func (bg *BaseGraphDB) Close() error {
	bg.mu.Lock()
	defer bg.mu.Unlock()
	
	if bg.closed {
		return nil
	}
	
	bg.closed = true
	if bg.logger != nil {
		bg.logger.Info("Graph database closed")
	}
	return nil
}

// RecordQuery records query statistics
func (bg *BaseGraphDB) RecordQuery(success bool, duration time.Duration) {
	bg.stats.mu.Lock()
	defer bg.stats.mu.Unlock()
	
	bg.stats.totalQueries++
	bg.stats.lastQueryTime = duration
	
	if !success {
		bg.stats.failedQueries++
	}
	
	if bg.metrics != nil {
		bg.metrics.Counter("graphdb_queries_total", 1, map[string]string{"success": fmt.Sprintf("%t", success)})
		bg.metrics.Timer("graphdb_query_duration", float64(duration.Milliseconds()), nil)
	}
}

// RecordNodeCreation records node creation statistics
func (bg *BaseGraphDB) RecordNodeCreation() {
	bg.stats.mu.Lock()
	defer bg.stats.mu.Unlock()
	
	bg.stats.totalNodes++
	
	if bg.metrics != nil {
		bg.metrics.Counter("graphdb_nodes_created", 1, nil)
	}
}

// RecordRelationCreation records relationship creation statistics
func (bg *BaseGraphDB) RecordRelationCreation() {
	bg.stats.mu.Lock()
	defer bg.stats.mu.Unlock()
	
	bg.stats.totalRelations++
	
	if bg.metrics != nil {
		bg.metrics.Counter("graphdb_relations_created", 1, nil)
	}
}

// UpdateHealth updates the health status
func (bg *BaseGraphDB) UpdateHealth(status string, err error) {
	bg.health.mu.Lock()
	defer bg.health.mu.Unlock()
	
	bg.health.lastCheck = time.Now()
	bg.health.status = status
	bg.health.error = err
	
	if bg.metrics != nil {
		bg.metrics.Gauge("graphdb_health_status", map[string]float64{"healthy": 1, "unhealthy": 0}[status], nil)
	}
}

// GetHealth returns current health status
func (bg *BaseGraphDB) GetHealth() (string, error, time.Time) {
	bg.health.mu.RLock()
	defer bg.health.mu.RUnlock()
	
	return bg.health.status, bg.health.error, bg.health.lastCheck
}

// GetStats returns database statistics
func (bg *BaseGraphDB) GetStats() map[string]interface{} {
	bg.stats.mu.RLock()
	defer bg.stats.mu.RUnlock()
	
	return map[string]interface{}{
		"total_queries":     bg.stats.totalQueries,
		"failed_queries":    bg.stats.failedQueries,
		"total_nodes":       bg.stats.totalNodes,
		"total_relations":   bg.stats.totalRelations,
		"connection_time":   bg.stats.connectionTime.String(),
		"last_query_time":   bg.stats.lastQueryTime.String(),
		"success_rate":      float64(bg.stats.totalQueries-bg.stats.failedQueries) / float64(bg.stats.totalQueries),
	}
}

// GraphNode represents a graph node
type GraphNode struct {
	ID         string                 `json:"id"`
	Labels     []string               `json:"labels"`
	Properties map[string]interface{} `json:"properties"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// GraphRelation represents a graph relationship
type GraphRelation struct {
	ID         string                 `json:"id"`
	FromID     string                 `json:"from_id"`
	ToID       string                 `json:"to_id"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// GraphPath represents a path in the graph
type GraphPath struct {
	Nodes         []GraphNode     `json:"nodes"`
	Relationships []GraphRelation `json:"relationships"`
	Length        int             `json:"length"`
}

// GraphTraversalOptions holds options for graph traversal
type GraphTraversalOptions struct {
	MaxDepth        int               `json:"max_depth"`
	Direction       string            `json:"direction"` // "IN", "OUT", "BOTH"
	RelationTypes   []string          `json:"relation_types"`
	NodeLabels      []string          `json:"node_labels"`
	PropertyFilters map[string]interface{} `json:"property_filters"`
	Limit           int               `json:"limit"`
	Skip            int               `json:"skip"`
}

// DefaultTraversalOptions returns default traversal options
func DefaultTraversalOptions() *GraphTraversalOptions {
	return &GraphTraversalOptions{
		MaxDepth:  5,
		Direction: "BOTH",
		Limit:     100,
		Skip:      0,
	}
}

// GraphAlgorithmType represents supported graph algorithms
type GraphAlgorithmType string

const (
	AlgorithmShortestPath    GraphAlgorithmType = "shortest_path"
	AlgorithmPageRank        GraphAlgorithmType = "pagerank"
	AlgorithmCentrality      GraphAlgorithmType = "centrality"
	AlgorithmCommunityDetect GraphAlgorithmType = "community_detection"
	AlgorithmTriangleCount   GraphAlgorithmType = "triangle_count"
	AlgorithmClusteringCoeff GraphAlgorithmType = "clustering_coefficient"
)

// GraphAlgorithmOptions holds options for graph algorithms
type GraphAlgorithmOptions struct {
	Algorithm   GraphAlgorithmType     `json:"algorithm"`
	StartNode   string                 `json:"start_node,omitempty"`
	EndNode     string                 `json:"end_node,omitempty"`
	Iterations  int                    `json:"iterations,omitempty"`
	DampingFactor float64              `json:"damping_factor,omitempty"`
	WeightProperty string              `json:"weight_property,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// BatchOperation represents a batch operation
type BatchOperation struct {
	OperationType string                 `json:"operation_type"` // "CREATE_NODE", "CREATE_RELATION", "UPDATE_NODE", "DELETE_NODE"
	NodeLabels    []string               `json:"node_labels,omitempty"`
	Properties    map[string]interface{} `json:"properties,omitempty"`
	FromID        string                 `json:"from_id,omitempty"`
	ToID          string                 `json:"to_id,omitempty"`
	RelationType  string                 `json:"relation_type,omitempty"`
	NodeID        string                 `json:"node_id,omitempty"`
}

// ExtendedGraphDB extends the base GraphDB interface with advanced features
type ExtendedGraphDB interface {
	interfaces.GraphDB
	
	// Advanced node operations
	CreateNodes(ctx context.Context, nodes []GraphNode) ([]string, error)
	GetNodes(ctx context.Context, ids []string) ([]GraphNode, error)
	UpdateNode(ctx context.Context, id string, properties map[string]interface{}) error
	DeleteNode(ctx context.Context, id string) error
	GetNodesByLabel(ctx context.Context, label string, limit int) ([]GraphNode, error)
	
	// Advanced relationship operations
	CreateRelations(ctx context.Context, relations []GraphRelation) error
	GetRelations(ctx context.Context, fromID, toID, relType string) ([]GraphRelation, error)
	UpdateRelation(ctx context.Context, id string, properties map[string]interface{}) error
	DeleteRelation(ctx context.Context, id string) error
	
	// Graph traversal and analytics
	Traverse(ctx context.Context, startID string, options *GraphTraversalOptions) ([]GraphNode, error)
	FindPaths(ctx context.Context, startID, endID string, maxDepth int) ([]GraphPath, error)
	RunAlgorithm(ctx context.Context, options *GraphAlgorithmOptions) (map[string]interface{}, error)
	
	// Batch operations
	ExecuteBatch(ctx context.Context, operations []BatchOperation) error
	
	// Schema management
	CreateIndex(ctx context.Context, label, property string) error
	CreateConstraint(ctx context.Context, label, property, constraintType string) error
	DropIndex(ctx context.Context, label, property string) error
	DropConstraint(ctx context.Context, label, property, constraintType string) error
	GetSchema(ctx context.Context) (map[string]interface{}, error)
	
	// Health and monitoring
	HealthCheck(ctx context.Context) error
	GetMetrics(ctx context.Context) (map[string]interface{}, error)
	
	// Transaction support
	BeginTransaction(ctx context.Context) (GraphTransaction, error)
}

// GraphTransaction represents a graph database transaction
type GraphTransaction interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
	CreateNode(ctx context.Context, labels []string, properties map[string]interface{}) (string, error)
	CreateRelation(ctx context.Context, fromID, toID, relType string, properties map[string]interface{}) error
	Query(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error)
}
