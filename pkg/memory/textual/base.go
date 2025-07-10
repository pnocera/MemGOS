// Package textual provides textual memory implementations for MemGOS
package textual

import (
	"context"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// SearchMode represents the search mode for textual memories
type SearchMode string

const (
	SearchModeFast     SearchMode = "fast"     // Fast search with basic similarity
	SearchModeFine     SearchMode = "fine"     // Fine-grained search with advanced reasoning
	SearchModeSemantic SearchMode = "semantic" // Semantic search using embeddings
)

// MemoryType represents different memory scopes
type MemoryType string

const (
	MemoryTypeAll          MemoryType = "All"
	MemoryTypeWorking      MemoryType = "WorkingMemory"
	MemoryTypeLongTerm     MemoryType = "LongTermMemory"
	MemoryTypeUser         MemoryType = "UserMemory"
	MemoryTypeInternet     MemoryType = "InternetMemory"
)

// TextualMemoryInterface defines the interface for textual memory implementations
type TextualMemoryInterface interface {
	// Core operations
	Add(ctx context.Context, memories []*types.TextualMemoryItem) error
	Get(ctx context.Context, id string) (*types.TextualMemoryItem, error)
	GetByIDs(ctx context.Context, ids []string) ([]*types.TextualMemoryItem, error)
	GetAll(ctx context.Context) ([]*types.TextualMemoryItem, error)
	Update(ctx context.Context, id string, memory *types.TextualMemoryItem) error
	Delete(ctx context.Context, ids []string) error
	DeleteAll(ctx context.Context) error

	// Search operations
	Search(ctx context.Context, query string, topK int, mode SearchMode, memoryType MemoryType) ([]*types.TextualMemoryItem, error)
	SearchSemantic(ctx context.Context, query string, topK int, filters map[string]string) ([]*types.TextualMemoryItem, error)
	
	// Advanced operations
	GetRelevantSubgraph(ctx context.Context, query string, topK int, depth int) (map[string]interface{}, error)
	GetWorkingMemory(ctx context.Context) ([]*types.TextualMemoryItem, error)
	ReplaceWorkingMemory(ctx context.Context, memories []*types.TextualMemoryItem) error
	GetMemorySize(ctx context.Context) (map[string]int, error)

	// Persistence
	Load(ctx context.Context, dir string) error
	Dump(ctx context.Context, dir string) error
	Drop(ctx context.Context, keepLastN int) error

	// Lifecycle
	Close() error
}

// BaseTextualMemory provides common functionality for textual memory implementations
type BaseTextualMemory struct {
	embedder    interfaces.Embedder
	vectorDB    interfaces.VectorDB
	graphDB     interfaces.GraphDB
	llm         interfaces.LLM
	logger      interfaces.Logger
	metrics     interfaces.Metrics
	searchIndex SearchIndex
	closed      bool
}

// SearchIndex provides advanced search capabilities
type SearchIndex struct {
	invertedIndex map[string][]string           // Word -> memory IDs
	tfIndex       map[string]map[string]float64 // Memory ID -> word -> TF score
	idfIndex      map[string]float64            // Word -> IDF score
	embeddings    map[string][]float32          // Memory ID -> embedding vector
	metadata      map[string]map[string]interface{} // Memory ID -> metadata
}

// NewBaseTextualMemory creates a new base textual memory
func NewBaseTextualMemory(embedder interfaces.Embedder, vectorDB interfaces.VectorDB, graphDB interfaces.GraphDB, llm interfaces.LLM, logger interfaces.Logger, metrics interfaces.Metrics) *BaseTextualMemory {
	return &BaseTextualMemory{
		embedder:    embedder,
		vectorDB:    vectorDB,
		graphDB:     graphDB,
		llm:         llm,
		logger:      logger,
		metrics:     metrics,
		searchIndex: SearchIndex{
			invertedIndex: make(map[string][]string),
			tfIndex:       make(map[string]map[string]float64),
			idfIndex:      make(map[string]float64),
			embeddings:    make(map[string][]float32),
			metadata:      make(map[string]map[string]interface{}),
		},
	}
}

// Close marks the memory as closed
func (btm *BaseTextualMemory) Close() error {
	if btm.closed {
		return nil
	}
	btm.closed = true
	if btm.logger != nil {
		btm.logger.Info("Textual memory closed")
	}
	return nil
}

// IsClosed returns true if the memory is closed
func (btm *BaseTextualMemory) IsClosed() bool {
	return btm.closed
}

// GetEmbedder returns the embedder
func (btm *BaseTextualMemory) GetEmbedder() interfaces.Embedder {
	return btm.embedder
}

// GetVectorDB returns the vector database
func (btm *BaseTextualMemory) GetVectorDB() interfaces.VectorDB {
	return btm.vectorDB
}

// GetGraphDB returns the graph database
func (btm *BaseTextualMemory) GetGraphDB() interfaces.GraphDB {
	return btm.graphDB
}

// GetLLM returns the LLM
func (btm *BaseTextualMemory) GetLLM() interfaces.LLM {
	return btm.llm
}

// SearchResult represents a search result with scoring
type SearchResult struct {
	Memory *types.TextualMemoryItem `json:"memory"`
	Score  float64                  `json:"score"`
	Source string                   `json:"source"` // "index", "semantic", "graph"
}

// SearchRequest represents a search request
type SearchRequest struct {
	Query           string            `json:"query"`
	TopK            int               `json:"top_k"`
	Mode            SearchMode        `json:"mode"`
	MemoryType      MemoryType        `json:"memory_type"`
	Filters         map[string]string `json:"filters"`
	UseInternet     bool              `json:"use_internet"`
	GraphDepth      int               `json:"graph_depth"`
	MinScore        float64           `json:"min_score"`
	IncludeMetadata bool              `json:"include_metadata"`
}

// SearchResponse represents a search response
type SearchResponse struct {
	Results     []*SearchResult       `json:"results"`
	TotalFound  int                   `json:"total_found"`
	SearchTime  time.Duration         `json:"search_time"`
	SearchMode  SearchMode            `json:"search_mode"`
	UsedSources []string              `json:"used_sources"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// MemoryStats represents memory statistics
type MemoryStats struct {
	TotalMemories    int                    `json:"total_memories"`
	MemoryTypes      map[string]int         `json:"memory_types"`
	IndexSize        int                    `json:"index_size"`
	EmbeddingSize    int                    `json:"embedding_size"`
	AverageMemoryAge time.Duration          `json:"average_memory_age"`
	TopWords         []string               `json:"top_words"`
	LastUpdated      time.Time              `json:"last_updated"`
	StorageSize      int64                  `json:"storage_size"`
	SearchMetrics    map[string]interface{} `json:"search_metrics"`
}

// InternetRetrievalResult represents internet retrieval result
type InternetRetrievalResult struct {
	URL         string                 `json:"url"`
	Title       string                 `json:"title"`
	Content     string                 `json:"content"`
	Score       float64                `json:"score"`
	Source      string                 `json:"source"`
	RetrievedAt time.Time              `json:"retrieved_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// GraphNode represents a node in the memory graph
type GraphNode struct {
	ID          string                 `json:"id"`
	Memory      *types.TextualMemoryItem `json:"memory"`
	Connections []string               `json:"connections"`
	Weight      float64                `json:"weight"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// GraphEdge represents an edge in the memory graph
type GraphEdge struct {
	Source string  `json:"source"`
	Target string  `json:"target"`
	Type   string  `json:"type"`
	Weight float64 `json:"weight"`
}

// SubgraphResult represents a subgraph result
type SubgraphResult struct {
	CoreID string       `json:"core_id"`
	Nodes  []*GraphNode `json:"nodes"`
	Edges  []*GraphEdge `json:"edges"`
	Score  float64      `json:"score"`
}