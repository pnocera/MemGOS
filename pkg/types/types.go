// Package types defines the core types and interfaces for MemGOS
package types

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// MessageRole represents the role of a message in a conversation
type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleSystem    MessageRole = "system"
)

// MessageDict represents a single message in a conversation
type MessageDict struct {
	Role    MessageRole `json:"role" validate:"required,oneof=user assistant system"`
	Content string      `json:"content" validate:"required"`
}

// MessageList represents a list of messages in a conversation
type MessageList []MessageDict

// ChatHistory represents the complete chat history for a session
type ChatHistory struct {
	UserID        string      `json:"user_id"`
	SessionID     string      `json:"session_id"`
	CreatedAt     time.Time   `json:"created_at"`
	TotalMessages int         `json:"total_messages"`
	ChatHistory   MessageList `json:"chat_history"`
}

// MemoryItem represents a generic memory item interface
type MemoryItem interface {
	GetID() string
	GetContent() string
	GetMetadata() map[string]interface{}
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
}

// TextualMemoryItem represents a textual memory item
type TextualMemoryItem struct {
	ID        string                 `json:"id"`
	Memory    string                 `json:"memory"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// GetID returns the memory item ID
func (t *TextualMemoryItem) GetID() string { return t.ID }

// GetContent returns the memory content
func (t *TextualMemoryItem) GetContent() string { return t.Memory }

// GetMetadata returns the memory metadata
func (t *TextualMemoryItem) GetMetadata() map[string]interface{} { return t.Metadata }

// GetCreatedAt returns the creation timestamp
func (t *TextualMemoryItem) GetCreatedAt() time.Time { return t.CreatedAt }

// GetUpdatedAt returns the last update timestamp
func (t *TextualMemoryItem) GetUpdatedAt() time.Time { return t.UpdatedAt }

// ActivationMemoryItem represents activation/KV cache memory
type ActivationMemoryItem struct {
	ID        string                 `json:"id"`
	Memory    interface{}            `json:"memory"` // Can be tensor data, KV cache, etc.
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// GetID returns the memory item ID
func (a *ActivationMemoryItem) GetID() string { return a.ID }

// GetContent returns the memory content as string
func (a *ActivationMemoryItem) GetContent() string { return "" } // Activation memory doesn't have string content

// GetMetadata returns the memory metadata
func (a *ActivationMemoryItem) GetMetadata() map[string]interface{} { return a.Metadata }

// GetCreatedAt returns the creation timestamp
func (a *ActivationMemoryItem) GetCreatedAt() time.Time { return a.CreatedAt }

// GetUpdatedAt returns the last update timestamp
func (a *ActivationMemoryItem) GetUpdatedAt() time.Time { return a.UpdatedAt }

// ParametricMemoryItem represents parametric memory (LoRA, adapters, etc.)
type ParametricMemoryItem struct {
	ID        string                 `json:"id"`
	Memory    interface{}            `json:"memory"` // Model parameters, weights, etc.
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// GetID returns the memory item ID
func (p *ParametricMemoryItem) GetID() string { return p.ID }

// GetContent returns the memory content as string
func (p *ParametricMemoryItem) GetContent() string { return "" } // Parametric memory doesn't have string content

// GetMetadata returns the memory metadata
func (p *ParametricMemoryItem) GetMetadata() map[string]interface{} { return p.Metadata }

// GetCreatedAt returns the creation timestamp
func (p *ParametricMemoryItem) GetCreatedAt() time.Time { return p.CreatedAt }

// GetUpdatedAt returns the last update timestamp
func (p *ParametricMemoryItem) GetUpdatedAt() time.Time { return p.UpdatedAt }

// MOSSearchResult represents the result of a memory search across all memory types
type MOSSearchResult struct {
	TextMem []CubeMemoryResult `json:"text_mem"`
	ActMem  []CubeMemoryResult `json:"act_mem"`
	ParaMem []CubeMemoryResult `json:"para_mem"`
}

// CubeMemoryResult represents memory results from a specific cube
type CubeMemoryResult struct {
	CubeID   string        `json:"cube_id"`
	Memories []MemoryItem  `json:"memories"`
}

// UserRole represents the role of a user in the system
type UserRole string

const (
	UserRoleUser  UserRole = "user"
	UserRoleAdmin UserRole = "admin"
	UserRoleGuest UserRole = "guest"
)

// User represents a user in the system
type User struct {
	ID        string    `json:"user_id"`        // Alias for UserID for compatibility
	UserID    string    `json:"user_id"`
	Name      string    `json:"user_name"`      // Alias for UserName for compatibility  
	UserName  string    `json:"user_name"`
	Role      UserRole  `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	IsActive  bool      `json:"is_active"`
}

// MemCube represents a memory cube containing different types of memories
type MemCube struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path,omitempty"`
	OwnerID   string    `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	IsLoaded  bool      `json:"is_loaded"`
}

// SearchQuery represents a query for memory search
type SearchQuery struct {
	Query          string            `json:"query" validate:"required"`
	TopK           int               `json:"top_k,omitempty"`
	Filters        map[string]string `json:"filters,omitempty"`
	CubeIDs        []string          `json:"cube_ids,omitempty"`
	InstallCubeIDs []string          `json:"install_cube_ids,omitempty"`
	UserID         string            `json:"user_id,omitempty"`
	SessionID      string            `json:"session_id,omitempty"`
}

// AddMemoryRequest represents a request to add memory to a cube
type AddMemoryRequest struct {
	Messages      []map[string]interface{} `json:"messages,omitempty"`
	MemoryContent *string                  `json:"memory_content,omitempty"`
	DocPath       *string                  `json:"doc_path,omitempty"`
	MemCubeID     string                   `json:"mem_cube_id,omitempty"`
	UserID        string                   `json:"user_id,omitempty"`
}

// ChatRequest represents a chat request
type ChatRequest struct {
	Query     string                 `json:"query" validate:"required"`
	UserID    string                 `json:"user_id,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	MaxTokens *int                   `json:"max_tokens,omitempty"`
	TopK      *int                   `json:"top_k,omitempty"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Response  string    `json:"response"`
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
}

// BackendType represents the type of backend (LLM, VectorDB, etc.)
type BackendType string

const (
	BackendOpenAI      BackendType = "openai"
	BackendOllama      BackendType = "ollama"
	BackendHuggingFace BackendType = "huggingface"
	BackendQdrant      BackendType = "qdrant"
	BackendNeo4j       BackendType = "neo4j"
	BackendRedis       BackendType = "redis"
)

// MemoryBackend represents the type of memory backend
type MemoryBackend string

const (
	MemoryBackendNaive      MemoryBackend = "naive"
	MemoryBackendTree       MemoryBackend = "tree"
	MemoryBackendTreeText   MemoryBackend = "tree_text"
	MemoryBackendGeneral    MemoryBackend = "general"
	MemoryBackendKVCache    MemoryBackend = "kv_cache"
	MemoryBackendLoRA       MemoryBackend = "lora"
)

// EmbeddingVector represents an embedding vector
type EmbeddingVector []float32

// VectorSearchResult represents a vector search result
type VectorSearchResult struct {
	ID       string          `json:"id"`
	Score    float32         `json:"score"`
	Metadata map[string]interface{} `json:"metadata"`
}

// RetrievalResult represents a result from internet retrieval
type RetrievalResult struct {
	ID          string                 `json:"id"`
	URL         string                 `json:"url"`
	Title       string                 `json:"title"`
	Content     string                 `json:"content"`
	Summary     string                 `json:"summary,omitempty"`
	Score       float32                `json:"score"`
	Metadata    map[string]interface{} `json:"metadata"`
	RetrievedAt time.Time              `json:"retrieved_at"`
}

// TaskPriority represents the priority of a scheduled task
type TaskPriority string

const (
	TaskPriorityLow      TaskPriority = "low"
	TaskPriorityMedium   TaskPriority = "medium"
	TaskPriorityHigh     TaskPriority = "high"
	TaskPriorityCritical TaskPriority = "critical"
)

// ScheduledTask represents a scheduled task in the memory scheduler
type ScheduledTask struct {
	ID        string       `json:"id"`
	UserID    string       `json:"user_id"`
	CubeID    string       `json:"cube_id"`
	TaskType  string       `json:"task_type"`
	Priority  TaskPriority `json:"priority"`
	Payload   interface{}  `json:"payload"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	Status    string       `json:"status"`
}

// Error types for better error handling
type ErrorType string

const (
	ErrorTypeValidation   ErrorType = "validation"
	ErrorTypeNotFound     ErrorType = "not_found"
	ErrorTypeUnauthorized ErrorType = "unauthorized"
	ErrorTypeInternal     ErrorType = "internal"
	ErrorTypeExternal     ErrorType = "external"
)

// MemGOSError represents a structured error in MemGOS
type MemGOSError struct {
	Type    ErrorType `json:"type"`
	Message string    `json:"message"`
	Code    string    `json:"code,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Error implements the error interface
func (e *MemGOSError) Error() string {
	return e.Message
}

// NewMemGOSError creates a new MemGOS error
func NewMemGOSError(errType ErrorType, message string, code string, details map[string]interface{}) *MemGOSError {
	return &MemGOSError{
		Type:    errType,
		Message: message,
		Code:    code,
		Details: details,
	}
}

// Context keys for request context
type ContextKey string

const (
	ContextKeyUserID    ContextKey = "user_id"
	ContextKeySessionID ContextKey = "session_id"
	ContextKeyRequestID ContextKey = "request_id"
	ContextKeyTraceID   ContextKey = "trace_id"
)

// RequestContext holds request-specific context information
type RequestContext struct {
	UserID    string
	SessionID string
	RequestID string
	TraceID   string
}

// GetRequestContext extracts request context from Go context
func GetRequestContext(ctx context.Context) *RequestContext {
	return &RequestContext{
		UserID:    getStringFromContext(ctx, ContextKeyUserID),
		SessionID: getStringFromContext(ctx, ContextKeySessionID),
		RequestID: getStringFromContext(ctx, ContextKeyRequestID),
		TraceID:   getStringFromContext(ctx, ContextKeyTraceID),
	}
}

// helper function to extract string from context
func getStringFromContext(ctx context.Context, key ContextKey) string {
	if value := ctx.Value(key); value != nil {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// NewRequestContext creates a new request context with generated IDs
func NewRequestContext(userID, sessionID string) *RequestContext {
	return &RequestContext{
		UserID:    userID,
		SessionID: sessionID,
		RequestID: uuid.New().String(),
		TraceID:   uuid.New().String(),
	}
}

// UserSession represents a user session
type UserSession struct {
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IsActive  bool      `json:"is_active"`
	IPAddress string    `json:"ip_address,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
}

// JWTClaims represents JWT token claims
type JWTClaims struct {
	UserID    string    `json:"user_id"`
	Role      UserRole  `json:"role"`
	SessionID string    `json:"session_id"`
	IssuedAt  time.Time `json:"iat"`
	ExpiresAt time.Time `json:"exp"`
}

// AuthRequest represents an authentication request
type AuthRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	Token     string       `json:"token"`
	User      *User        `json:"user"`
	ExpiresAt time.Time    `json:"expires_at"`
	SessionID string       `json:"session_id"`
	UserID    string       `json:"user_id"`
}

// GetAllMemoriesRequest represents a request to get all memories
type GetAllMemoriesRequest struct {
	UserID    string `json:"user_id"`
	MemCubeID string `json:"mem_cube_id"`
}

// GetMemoryRequest represents a request to get a specific memory
type GetMemoryRequest struct {
	UserID    string `json:"user_id"`
	MemCubeID string `json:"mem_cube_id"`
	MemoryID  string `json:"memory_id"`
}

// UpdateMemoryRequest represents a request to update a memory
type UpdateMemoryRequest struct {
	UserID        string                 `json:"user_id"`
	MemCubeID     string                 `json:"mem_cube_id"`
	MemoryID      string                 `json:"memory_id"`
	UpdatedMemory map[string]interface{} `json:"updated_memory"`
}

// DeleteMemoryRequest represents a request to delete a memory
type DeleteMemoryRequest struct {
	UserID    string `json:"user_id"`
	MemCubeID string `json:"mem_cube_id"`
	MemoryID  string `json:"memory_id"`
}

// DeleteAllMemoriesRequest represents a request to delete all memories
type DeleteAllMemoriesRequest struct {
	UserID    string `json:"user_id"`
	MemCubeID string `json:"mem_cube_id"`
}