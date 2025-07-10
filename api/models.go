package api

import "github.com/memtensor/memgos/pkg/types"

// BaseRequest represents the base structure for all API requests
type BaseRequest struct {
	UserID *string `json:"user_id,omitempty" example:"user123"`
}

// BaseResponse represents the base structure for all API responses
type BaseResponse[T any] struct {
	Code    int    `json:"code" example:"200"`
	Message string `json:"message" example:"Operation successful"`
	Data    *T     `json:"data,omitempty"`
}

// SimpleResponse for operations without data return
type SimpleResponse = BaseResponse[interface{}]

// Message represents a chat message
type Message struct {
	Role    string `json:"role" binding:"required" example:"user"`
	Content string `json:"content" binding:"required" example:"Hello, how can I help you?"`
}

// MemoryCreate represents a request to create memories
type MemoryCreate struct {
	BaseRequest
	Messages      []Message `json:"messages,omitempty"`
	MemCubeID     *string   `json:"mem_cube_id,omitempty" example:"cube123"`
	MemoryContent *string   `json:"memory_content,omitempty" example:"This is a memory content"`
	DocPath       *string   `json:"doc_path,omitempty" example:"/path/to/document.txt"`
}

// SearchRequest represents a search request
type SearchRequest struct {
	BaseRequest
	Query          string   `json:"query" binding:"required" example:"How to implement a feature?"`
	InstallCubeIDs []string `json:"install_cube_ids,omitempty" example:"cube123,cube456"`
	TopK           *int     `json:"top_k,omitempty" example:"5"`
}

// MemCubeRegister represents a request to register a memory cube
type MemCubeRegister struct {
	BaseRequest
	MemCubeNameOrPath string  `json:"mem_cube_name_or_path" binding:"required" example:"/path/to/cube"`
	MemCubeID         *string `json:"mem_cube_id,omitempty" example:"cube123"`
}

// ChatRequest represents a chat request
type ChatRequest struct {
	BaseRequest
	Query     string            `json:"query" binding:"required" example:"What is the latest update?"`
	Context   map[string]string `json:"context,omitempty"`
	MaxTokens *int              `json:"max_tokens,omitempty" example:"1000"`
	TopK      *int              `json:"top_k,omitempty" example:"5"`
}

// UserCreate represents a request to create a user
type UserCreate struct {
	BaseRequest
	UserName *string          `json:"user_name,omitempty" example:"john_doe"`
	Role     types.UserRole   `json:"role" example:"user"`
	UserID   string           `json:"user_id" binding:"required" example:"user123"`
}

// CubeShare represents a request to share a cube
type CubeShare struct {
	BaseRequest
	TargetUserID string `json:"target_user_id" binding:"required" example:"user456"`
}

// ConfigRequest represents a configuration request
type ConfigRequest struct {
	UserID                   *string                    `json:"user_id,omitempty"`
	SessionID                *string                    `json:"session_id,omitempty"`
	EnableTextualMemory      *bool                      `json:"enable_textual_memory,omitempty"`
	EnableActivationMemory   *bool                      `json:"enable_activation_memory,omitempty"`
	EnableParametricMemory   *bool                      `json:"enable_parametric_memory,omitempty"`
	EnableMemScheduler       *bool                      `json:"enable_mem_scheduler,omitempty"`
	MaxTurnsWindow           *int                       `json:"max_turns_window,omitempty"`
	TopK                     *int                       `json:"top_k,omitempty"`
	PROMode                  *bool                      `json:"pro_mode,omitempty"`
	ChatModel                *LLMConfigRequest          `json:"chat_model,omitempty"`
	MemReader                *MemReaderConfigRequest    `json:"mem_reader,omitempty"`
	MemScheduler             *SchedulerConfigRequest    `json:"mem_scheduler,omitempty"`
}

// LLMConfigRequest represents LLM configuration
type LLMConfigRequest struct {
	Backend string                 `json:"backend" binding:"required"`
	Config  map[string]interface{} `json:"config" binding:"required"`
}

// MemReaderConfigRequest represents MemReader configuration
type MemReaderConfigRequest struct {
	Backend string                 `json:"backend" binding:"required"`
	Config  map[string]interface{} `json:"config" binding:"required"`
}

// SchedulerConfigRequest represents Scheduler configuration
type SchedulerConfigRequest struct {
	Backend string                 `json:"backend" binding:"required"`
	Config  map[string]interface{} `json:"config" binding:"required"`
}

// Response types
type ConfigResponse = BaseResponse[interface{}]
type MemoryResponse = BaseResponse[map[string]interface{}]
type SearchResponse = BaseResponse[map[string]interface{}]
type ChatResponse = BaseResponse[string]
type UserResponse = BaseResponse[map[string]interface{}]
type UserListResponse = BaseResponse[[]map[string]interface{}]

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Version   string            `json:"version"`
	Uptime    string            `json:"uptime"`
	Checks    map[string]string `json:"checks"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
	Details string `json:"details,omitempty"`
}

// MetricsResponse represents metrics response
type MetricsResponse struct {
	Timestamp string                 `json:"timestamp"`
	Uptime    string                 `json:"uptime"`
	Metrics   map[string]interface{} `json:"metrics"`
}