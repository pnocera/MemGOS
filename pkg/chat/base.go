// Package chat provides memory-augmented conversation implementations for MemGOS
package chat

import (
	"context"
	"io"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// MemChatConfig represents configuration for MemChat implementations
type MemChatConfig struct {
	// User and session information
	UserID    string `json:"user_id" validate:"required"`
	SessionID string `json:"session_id" validate:"required"`
	CreatedAt time.Time `json:"created_at"`

	// LLM configuration
	ChatLLM LLMConfig `json:"chat_llm" validate:"required"`

	// Memory settings
	EnableTextualMemory    bool `json:"enable_textual_memory"`
	EnableActivationMemory bool `json:"enable_activation_memory"`
	TopK                   int  `json:"top_k"`
	MaxTurnsWindow         int  `json:"max_turns_window"`

	// Chat behavior
	SystemPrompt     string                 `json:"system_prompt"`
	Temperature      float64                `json:"temperature"`
	MaxTokens        int                    `json:"max_tokens"`
	StreamingEnabled bool                   `json:"streaming_enabled"`
	ResponseFormat   string                 `json:"response_format"`
	CustomSettings   map[string]interface{} `json:"custom_settings"`
}

// LLMConfig represents LLM configuration for chat
type LLMConfig struct {
	Provider    string            `json:"provider" validate:"required"`
	Model       string            `json:"model" validate:"required"`
	APIKey      string            `json:"api_key"`
	BaseURL     string            `json:"base_url"`
	MaxTokens   int               `json:"max_tokens"`
	Temperature float64           `json:"temperature"`
	TopP        float64           `json:"top_p"`
	Timeout     time.Duration     `json:"timeout"`
	Extra       map[string]interface{} `json:"extra"`
}

// ConversationMode represents different chat personalities/modes
type ConversationMode string

const (
	ConversationModeDefault     ConversationMode = "default"
	ConversationModeAnalytical  ConversationMode = "analytical"
	ConversationModeCreative    ConversationMode = "creative"
	ConversationModeHelpful     ConversationMode = "helpful"
	ConversationModeExplainer   ConversationMode = "explainer"
	ConversationModeSummarizer  ConversationMode = "summarizer"
)

// ChatContext represents the current conversation context
type ChatContext struct {
	Messages      types.MessageList          `json:"messages"`
	Memories      []*types.TextualMemoryItem `json:"memories"`
	SessionID     string                     `json:"session_id"`
	UserID        string                     `json:"user_id"`
	Mode          ConversationMode           `json:"mode"`
	LastUpdated   time.Time                  `json:"last_updated"`
	TokensUsed    int                        `json:"tokens_used"`
	TurnCount     int                        `json:"turn_count"`
	Metadata      map[string]interface{}     `json:"metadata"`
}

// ChatMetrics represents metrics for chat performance
type ChatMetrics struct {
	TotalMessages      int           `json:"total_messages"`
	TotalTokens        int           `json:"total_tokens"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	MemoriesRetrieved  int           `json:"memories_retrieved"`
	MemoriesStored     int           `json:"memories_stored"`
	ErrorCount         int           `json:"error_count"`
	SessionDuration    time.Duration `json:"session_duration"`
	LastActive         time.Time     `json:"last_active"`
}

// ChatCommand represents special chat commands
type ChatCommand string

const (
	ChatCommandBye       ChatCommand = "bye"
	ChatCommandClear     ChatCommand = "clear"
	ChatCommandMemory    ChatCommand = "mem"
	ChatCommandExport    ChatCommand = "export"
	ChatCommandHelp      ChatCommand = "help"
	ChatCommandStats     ChatCommand = "stats"
	ChatCommandMode      ChatCommand = "mode"
	ChatCommandSettings  ChatCommand = "settings"
)

// ChatResult represents the result of a chat interaction
type ChatResult struct {
	Response          string                     `json:"response"`
	Memories          []*types.TextualMemoryItem `json:"memories"`
	NewMemories       []*types.TextualMemoryItem `json:"new_memories"`
	Context           *ChatContext               `json:"context"`
	Metrics           *ChatMetrics               `json:"metrics"`
	IsCommand         bool                       `json:"is_command"`
	CommandResult     interface{}                `json:"command_result,omitempty"`
	Citations         []string                   `json:"citations,omitempty"`
	FollowUpQuestions []string                   `json:"follow_up_questions,omitempty"`
	TokensUsed        int                        `json:"tokens_used"`
	Duration          time.Duration              `json:"duration"`
	Timestamp         time.Time                  `json:"timestamp"`
}

// MemoryExtractionConfig represents configuration for memory extraction
type MemoryExtractionConfig struct {
	ExtractionMode    string   `json:"extraction_mode"`    // "automatic", "manual", "hybrid"
	KeywordFilters    []string `json:"keyword_filters"`
	MinimumLength     int      `json:"minimum_length"`
	MaximumLength     int      `json:"maximum_length"`
	ImportanceScore   float64  `json:"importance_score"`
	EnableSummarization bool   `json:"enable_summarization"`
	EnableCategorization bool  `json:"enable_categorization"`
}

// StreamingCallback represents a callback for streaming responses
type StreamingCallback func(chunk string, isComplete bool, metadata map[string]interface{}) error

// BaseMemChat defines the interface for all MemChat implementations
type BaseMemChat interface {
	// Core chat functionality
	Chat(ctx context.Context, query string) (*ChatResult, error)
	ChatStream(ctx context.Context, query string, callback StreamingCallback) (*ChatResult, error)
	
	// Memory integration
	GetMemCube() interfaces.MemCube
	SetMemCube(cube interfaces.MemCube) error
	
	// Session management
	GetSessionID() string
	GetUserID() string
	GetChatHistory(ctx context.Context) (*types.ChatHistory, error)
	ClearChatHistory(ctx context.Context) error
	SaveChatHistory(ctx context.Context) error
	ExportChatHistory(ctx context.Context, outputPath string) (string, error)
	
	// Context management
	GetContext(ctx context.Context) (*ChatContext, error)
	UpdateContext(ctx context.Context, context *ChatContext) error
	GetContextWindow() int
	SetContextWindow(window int)
	
	// Memory operations
	RetrieveMemories(ctx context.Context, query string, topK int) ([]*types.TextualMemoryItem, error)
	StoreMemories(ctx context.Context, memories []*types.TextualMemoryItem) error
	ExtractMemories(ctx context.Context, messages types.MessageList) ([]*types.TextualMemoryItem, error)
	
	// Mode and configuration
	GetMode() ConversationMode
	SetMode(mode ConversationMode) error
	GetConfig() *MemChatConfig
	UpdateConfig(config *MemChatConfig) error
	
	// Metrics and monitoring
	GetMetrics(ctx context.Context) (*ChatMetrics, error)
	ResetMetrics(ctx context.Context) error
	
	// Control and lifecycle
	Run(ctx context.Context) error  // Interactive chat mode
	Stop(ctx context.Context) error
	IsRunning() bool
	HealthCheck(ctx context.Context) error
	Close() error
}

// MultiModalMemChat extends BaseMemChat with multi-modal capabilities
type MultiModalMemChat interface {
	BaseMemChat
	
	// Multi-modal support
	ChatWithImage(ctx context.Context, query string, imageData []byte, imageType string) (*ChatResult, error)
	ChatWithDocument(ctx context.Context, query string, document io.Reader, docType string) (*ChatResult, error)
	ChatWithFile(ctx context.Context, query string, filePath string) (*ChatResult, error)
	
	// File processing
	ProcessDocument(ctx context.Context, document io.Reader, docType string) ([]*types.TextualMemoryItem, error)
	ProcessImage(ctx context.Context, imageData []byte, imageType string) ([]*types.TextualMemoryItem, error)
	
	// Content analysis
	AnalyzeContent(ctx context.Context, content interface{}) (map[string]interface{}, error)
	SummarizeContent(ctx context.Context, content string, maxLength int) (string, error)
}

// AdvancedMemChat extends BaseMemChat with advanced features
type AdvancedMemChat interface {
	BaseMemChat
	
	// Advanced memory management
	SearchMemoriesSemantic(ctx context.Context, query string, topK int, filters map[string]string) ([]*types.TextualMemoryItem, error)
	ClusterMemories(ctx context.Context) (map[string][]*types.TextualMemoryItem, error)
	AnalyzeMemoryPatterns(ctx context.Context) (map[string]interface{}, error)
	
	// Conversation analysis
	AnalyzeConversation(ctx context.Context) (map[string]interface{}, error)
	GenerateFollowUpQuestions(ctx context.Context, response string) ([]string, error)
	DetectTopicShift(ctx context.Context, newQuery string) (bool, string, error)
	
	// Response enhancement
	AddCitations(ctx context.Context, response string, memories []*types.TextualMemoryItem) (string, error)
	ValidateResponse(ctx context.Context, response string, originalQuery string) (bool, string, error)
	EnhanceResponse(ctx context.Context, response string, context map[string]interface{}) (string, error)
	
	// Learning and adaptation
	LearnFromFeedback(ctx context.Context, query string, response string, feedback map[string]interface{}) error
	AdaptToUser(ctx context.Context, userPreferences map[string]interface{}) error
	UpdatePersonalization(ctx context.Context, interactions []map[string]interface{}) error
}

// ChatManagerConfig represents configuration for the chat manager
type ChatManagerConfig struct {
	DefaultChatType   string                 `json:"default_chat_type"`
	MaxConcurrentChats int                   `json:"max_concurrent_chats"`
	SessionTimeout     time.Duration          `json:"session_timeout"`
	MemoryConfig       *MemoryExtractionConfig `json:"memory_config"`
	GlobalSettings     map[string]interface{} `json:"global_settings"`
}

// ChatManager manages multiple chat sessions and provides chat orchestration
type ChatManager interface {
	// Session management
	CreateSession(ctx context.Context, userID string, config *MemChatConfig) (string, error)
	GetSession(ctx context.Context, sessionID string) (BaseMemChat, error)
	CloseSession(ctx context.Context, sessionID string) error
	ListSessions(ctx context.Context, userID string) ([]string, error)
	
	// Chat routing
	RouteChat(ctx context.Context, sessionID string, query string) (*ChatResult, error)
	BroadcastMessage(ctx context.Context, userID string, message string) error
	
	// Global operations
	GetGlobalMetrics(ctx context.Context) (map[string]*ChatMetrics, error)
	CleanupSessions(ctx context.Context) error
	BackupSessions(ctx context.Context, outputPath string) error
	RestoreSessions(ctx context.Context, inputPath string) error
	
	// Configuration
	GetConfig() *ChatManagerConfig
	UpdateConfig(config *ChatManagerConfig) error
	
	// Lifecycle
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	HealthCheck(ctx context.Context) error
}

// DefaultMemChatConfig returns a default configuration for MemChat
func DefaultMemChatConfig() *MemChatConfig {
	return &MemChatConfig{
		UserID:    "default_user",
		SessionID: "default_session",
		CreatedAt: time.Now(),
		ChatLLM: LLMConfig{
			Provider:    "openai",
			Model:       "gpt-3.5-turbo",
			MaxTokens:   1024,
			Temperature: 0.7,
			TopP:        0.9,
			Timeout:     30 * time.Second,
			Extra:       make(map[string]interface{}),
		},
		EnableTextualMemory:    true,
		EnableActivationMemory: false,
		TopK:                   5,
		MaxTurnsWindow:         10,
		SystemPrompt: "You are a knowledgeable and helpful AI assistant with access to conversation memories. " +
			"Use the memories to provide more personalized and contextual responses.",
		Temperature:      0.7,
		MaxTokens:        1024,
		StreamingEnabled: false,
		ResponseFormat:   "text",
		CustomSettings:   make(map[string]interface{}),
	}
}

// ValidateMemChatConfig validates a MemChat configuration
func ValidateMemChatConfig(config *MemChatConfig) error {
	if config.UserID == "" {
		return &ChatError{Code: "invalid_config", Message: "user_id is required"}
	}
	if config.SessionID == "" {
		return &ChatError{Code: "invalid_config", Message: "session_id is required"}
	}
	if config.ChatLLM.Provider == "" {
		return &ChatError{Code: "invalid_config", Message: "chat_llm.provider is required"}
	}
	if config.ChatLLM.Model == "" {
		return &ChatError{Code: "invalid_config", Message: "chat_llm.model is required"}
	}
	if config.TopK < 0 {
		return &ChatError{Code: "invalid_config", Message: "top_k must be non-negative"}
	}
	if config.MaxTurnsWindow < 0 {
		return &ChatError{Code: "invalid_config", Message: "max_turns_window must be non-negative"}
	}
	return nil
}

// ChatError represents a chat-specific error
type ChatError struct {
	Code     string                 `json:"code"`
	Message  string                 `json:"message"`
	Type     string                 `json:"type"`
	Details  map[string]interface{} `json:"details,omitempty"`
	Cause    error                  `json:"-"`
}

// Error implements the error interface
func (e *ChatError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// NewChatError creates a new chat error
func NewChatError(code, message string) *ChatError {
	return &ChatError{
		Code:    code,
		Message: message,
		Type:    "chat_error",
		Details: make(map[string]interface{}),
	}
}

// NewChatErrorWithCause creates a new chat error with a cause
func NewChatErrorWithCause(code, message string, cause error) *ChatError {
	return &ChatError{
		Code:    code,
		Message: message,
		Type:    "chat_error",
		Details: make(map[string]interface{}),
		Cause:   cause,
	}
}

// Utility functions for working with chat contexts and configurations
func NewChatContext(userID, sessionID string, mode ConversationMode) *ChatContext {
	return &ChatContext{
		Messages:    make(types.MessageList, 0),
		Memories:    make([]*types.TextualMemoryItem, 0),
		SessionID:   sessionID,
		UserID:      userID,
		Mode:        mode,
		LastUpdated: time.Now(),
		TokensUsed:  0,
		TurnCount:   0,
		Metadata:    make(map[string]interface{}),
	}
}

func NewChatMetrics() *ChatMetrics {
	return &ChatMetrics{
		TotalMessages:       0,
		TotalTokens:         0,
		AverageResponseTime: 0,
		MemoriesRetrieved:   0,
		MemoriesStored:      0,
		ErrorCount:          0,
		SessionDuration:     0,
		LastActive:          time.Now(),
	}
}