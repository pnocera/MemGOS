// Package interfaces defines the core interfaces for MemGOS components
package interfaces

import (
	"context"
	"io"

	"github.com/memtensor/memgos/pkg/types"
)

// Memory defines the interface for all memory implementations
type Memory interface {
	// Load loads memories from a directory
	Load(ctx context.Context, dir string) error
	
	// Dump saves memories to a directory
	Dump(ctx context.Context, dir string) error
	
	// Add adds new memories
	Add(ctx context.Context, memories []types.MemoryItem) error
	
	// Get retrieves a specific memory by ID
	Get(ctx context.Context, id string) (types.MemoryItem, error)
	
	// GetAll retrieves all memories
	GetAll(ctx context.Context) ([]types.MemoryItem, error)
	
	// Update updates an existing memory
	Update(ctx context.Context, id string, memory types.MemoryItem) error
	
	// Delete deletes a memory by ID
	Delete(ctx context.Context, id string) error
	
	// DeleteAll deletes all memories
	DeleteAll(ctx context.Context) error
	
	// Search searches for memories based on query
	Search(ctx context.Context, query string, topK int) ([]types.MemoryItem, error)
	
	// Close closes the memory backend
	Close() error
}

// TextualMemory defines the interface for textual memory implementations
type TextualMemory interface {
	Memory
	
	// SearchSemantic performs semantic search on textual memories
	SearchSemantic(ctx context.Context, query string, topK int, filters map[string]string) ([]*types.TextualMemoryItem, error)
	
	// AddTextual adds textual memory items
	AddTextual(ctx context.Context, items []*types.TextualMemoryItem) error
	
	// GetTextual retrieves textual memories
	GetTextual(ctx context.Context, ids []string) ([]*types.TextualMemoryItem, error)
}

// ActivationMemory defines the interface for activation memory implementations
type ActivationMemory interface {
	Memory
	
	// Extract extracts activation memory based on query
	Extract(ctx context.Context, query string) (*types.ActivationMemoryItem, error)
	
	// Store stores activation memory
	Store(ctx context.Context, item *types.ActivationMemoryItem) error
	
	// GetCache retrieves cached activations
	GetCache(ctx context.Context, key string) (*types.ActivationMemoryItem, error)
}

// ParametricMemory defines the interface for parametric memory implementations
type ParametricMemory interface {
	Memory
	
	// LoadAdapter loads a parametric adapter
	LoadAdapter(ctx context.Context, path string) (*types.ParametricMemoryItem, error)
	
	// SaveAdapter saves a parametric adapter
	SaveAdapter(ctx context.Context, item *types.ParametricMemoryItem, path string) error
	
	// ApplyAdapter applies an adapter to a model
	ApplyAdapter(ctx context.Context, adapterID string, modelID string) error
}

// MemCube defines the interface for memory cube implementations
type MemCube interface {
	// GetID returns the cube ID
	GetID() string
	
	// GetName returns the cube name
	GetName() string
	
	// GetTextualMemory returns the textual memory interface
	GetTextualMemory() TextualMemory
	
	// GetActivationMemory returns the activation memory interface
	GetActivationMemory() ActivationMemory
	
	// GetParametricMemory returns the parametric memory interface
	GetParametricMemory() ParametricMemory
	
	// Load loads the cube from a directory
	Load(ctx context.Context, dir string) error
	
	// Dump saves the cube to a directory
	Dump(ctx context.Context, dir string) error
	
	// Close closes the cube and all its memories
	Close() error
}

// LLM defines the interface for Large Language Model implementations
type LLM interface {
	// Generate generates text based on messages
	Generate(ctx context.Context, messages types.MessageList) (string, error)
	
	// GenerateWithCache generates text with KV cache support
	GenerateWithCache(ctx context.Context, messages types.MessageList, cache interface{}) (string, error)
	
	// GenerateStream generates text with streaming support
	GenerateStream(ctx context.Context, messages types.MessageList, stream chan<- string) error
	
	// Embed generates embeddings for text
	Embed(ctx context.Context, text string) (types.EmbeddingVector, error)
	
	// GetModelInfo returns model information
	GetModelInfo() map[string]interface{}
	
	// Close closes the LLM connection
	Close() error
}

// Embedder defines the interface for embedding implementations
type Embedder interface {
	// Embed generates embeddings for text
	Embed(ctx context.Context, text string) (types.EmbeddingVector, error)
	
	// EmbedBatch generates embeddings for multiple texts
	EmbedBatch(ctx context.Context, texts []string) ([]types.EmbeddingVector, error)
	
	// GetDimension returns the embedding dimension
	GetDimension() int
	
	// Close closes the embedder
	Close() error
}

// VectorDB defines the interface for vector database implementations
type VectorDB interface {
	// Insert inserts vectors into the database
	Insert(ctx context.Context, vectors []types.EmbeddingVector, metadata []map[string]interface{}) error
	
	// Search searches for similar vectors
	Search(ctx context.Context, query types.EmbeddingVector, topK int, filters map[string]string) ([]types.VectorSearchResult, error)
	
	// Update updates existing vectors
	Update(ctx context.Context, id string, vector types.EmbeddingVector, metadata map[string]interface{}) error
	
	// Delete deletes vectors by ID
	Delete(ctx context.Context, ids []string) error
	
	// GetStats returns database statistics
	GetStats(ctx context.Context) (map[string]interface{}, error)
	
	// Close closes the database connection
	Close() error
}

// GraphDB defines the interface for graph database implementations
type GraphDB interface {
	// CreateNode creates a new node
	CreateNode(ctx context.Context, labels []string, properties map[string]interface{}) (string, error)
	
	// CreateRelation creates a relationship between nodes
	CreateRelation(ctx context.Context, fromID, toID, relType string, properties map[string]interface{}) error
	
	// Query executes a query and returns results
	Query(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error)
	
	// GetNode retrieves a node by ID
	GetNode(ctx context.Context, id string) (map[string]interface{}, error)
	
	// Close closes the database connection
	Close() error
}

// Parser defines the interface for document parsing implementations
type Parser interface {
	// Parse parses a document and returns structured content
	Parse(ctx context.Context, reader io.Reader, contentType string) ([]string, error)
	
	// ParseFile parses a file and returns structured content
	ParseFile(ctx context.Context, filePath string) ([]string, error)
	
	// SupportedTypes returns supported content types
	SupportedTypes() []string
}

// Chunker defines the interface for text chunking implementations
type Chunker interface {
	// Chunk splits text into chunks
	Chunk(ctx context.Context, text string) ([]string, error)
	
	// ChunkWithOverlap splits text into chunks with overlap
	ChunkWithOverlap(ctx context.Context, text string, overlap int) ([]string, error)
	
	// GetChunkSize returns the configured chunk size
	GetChunkSize() int
}

// Scheduler defines the interface for memory scheduling implementations
type Scheduler interface {
	// Start starts the scheduler
	Start(ctx context.Context) error
	
	// Stop stops the scheduler
	Stop(ctx context.Context) error
	
	// Schedule schedules a task
	Schedule(ctx context.Context, task *types.ScheduledTask) error
	
	// GetStatus returns the scheduler status
	GetStatus() string
	
	// ListTasks lists all scheduled tasks
	ListTasks(ctx context.Context) ([]*types.ScheduledTask, error)
}

// UserManager defines the interface for user management implementations
type UserManager interface {
	// CreateUser creates a new user
	CreateUser(ctx context.Context, userName string, role types.UserRole, userID string) (string, error)
	
	// GetUser retrieves a user by ID
	GetUser(ctx context.Context, userID string) (*types.User, error)
	
	// ListUsers lists all users
	ListUsers(ctx context.Context) ([]*types.User, error)
	
	// ValidateUser validates if a user exists and is active
	ValidateUser(ctx context.Context, userID string) (bool, error)
	
	// CreateCube creates a new memory cube
	CreateCube(ctx context.Context, cubeName, ownerID, cubePath, cubeID string) (string, error)
	
	// GetCube retrieves a cube by ID
	GetCube(ctx context.Context, cubeID string) (*types.MemCube, error)
	
	// GetUserCubes retrieves all cubes accessible by a user
	GetUserCubes(ctx context.Context, userID string) ([]*types.MemCube, error)
	
	// ValidateUserCubeAccess validates if a user has access to a cube
	ValidateUserCubeAccess(ctx context.Context, userID, cubeID string) (bool, error)
	
	// ShareCube shares a cube with another user
	ShareCube(ctx context.Context, cubeID, fromUserID, toUserID string) error
}

// ChatManager defines the interface for chat management implementations
type ChatManager interface {
	// Chat processes a chat request and returns a response
	Chat(ctx context.Context, request *types.ChatRequest) (*types.ChatResponse, error)
	
	// GetChatHistory retrieves chat history for a user
	GetChatHistory(ctx context.Context, userID, sessionID string) (*types.ChatHistory, error)
	
	// ClearChatHistory clears chat history for a user
	ClearChatHistory(ctx context.Context, userID, sessionID string) error
	
	// SaveChatHistory saves chat history
	SaveChatHistory(ctx context.Context, history *types.ChatHistory) error
}

// MOSCore defines the interface for Memory Operating System Core
type MOSCore interface {
	// Initialize initializes the MOS core
	Initialize(ctx context.Context) error
	
	// RegisterMemCube registers a memory cube
	RegisterMemCube(ctx context.Context, cubePath, cubeID, userID string) error
	
	// UnregisterMemCube unregisters a memory cube
	UnregisterMemCube(ctx context.Context, cubeID, userID string) error
	
	// Search searches across all registered memory cubes
	Search(ctx context.Context, query *types.SearchQuery) (*types.MOSSearchResult, error)
	
	// Add adds memory to a specific cube
	Add(ctx context.Context, request *types.AddMemoryRequest) error
	
	// Get retrieves a specific memory item
	Get(ctx context.Context, cubeID, memoryID, userID string) (types.MemoryItem, error)
	
	// GetAll retrieves all memories from a cube
	GetAll(ctx context.Context, cubeID, userID string) (*types.MOSSearchResult, error)
	
	// Update updates a memory item
	Update(ctx context.Context, cubeID, memoryID, userID string, memory types.MemoryItem) error
	
	// Delete deletes a memory item
	Delete(ctx context.Context, cubeID, memoryID, userID string) error
	
	// DeleteAll deletes all memories from a cube
	DeleteAll(ctx context.Context, cubeID, userID string) error
	
	// Chat processes a chat request
	Chat(ctx context.Context, request *types.ChatRequest) (*types.ChatResponse, error)
	
	// CreateUser creates a new user
	CreateUser(ctx context.Context, userName string, role types.UserRole, userID string) (string, error)
	
	// GetUserInfo retrieves user information
	GetUserInfo(ctx context.Context, userID string) (map[string]interface{}, error)
	
	// Close closes the MOS core
	Close() error
}

// ConfigManager defines the interface for configuration management
type ConfigManager interface {
	// Load loads configuration from a file
	Load(ctx context.Context, path string) error
	
	// Get retrieves a configuration value
	Get(key string) interface{}
	
	// Set sets a configuration value
	Set(key string, value interface{}) error
	
	// Save saves configuration to a file
	Save(ctx context.Context, path string) error
	
	// Watch watches for configuration changes
	Watch(ctx context.Context, callback func(key string, value interface{})) error
}

// Logger defines the interface for logging implementations
type Logger interface {
	// Debug logs debug level messages
	Debug(msg string, fields ...map[string]interface{})
	
	// Info logs info level messages
	Info(msg string, fields ...map[string]interface{})
	
	// Warn logs warning level messages
	Warn(msg string, fields ...map[string]interface{})
	
	// Error logs error level messages
	Error(msg string, err error, fields ...map[string]interface{})
	
	// Fatal logs fatal level messages and exits
	Fatal(msg string, err error, fields ...map[string]interface{})
	
	// WithFields returns a logger with additional fields
	WithFields(fields map[string]interface{}) Logger
}

// Metrics defines the interface for metrics collection
type Metrics interface {
	// Counter increments a counter metric
	Counter(name string, value float64, labels map[string]string)
	
	// Gauge sets a gauge metric
	Gauge(name string, value float64, labels map[string]string)
	
	// Histogram records a histogram metric
	Histogram(name string, value float64, labels map[string]string)
	
	// Timer records timing metrics
	Timer(name string, duration float64, labels map[string]string)
}

// HealthChecker defines the interface for health checking
type HealthChecker interface {
	// Check performs a health check
	Check(ctx context.Context) error
	
	// GetStatus returns the current health status
	GetStatus() string
	
	// RegisterCheck registers a health check
	RegisterCheck(name string, check func(ctx context.Context) error)
}