package chat

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/memtensor/memgos/pkg/types"
)

func TestDefaultMemChatConfig(t *testing.T) {
	config := DefaultMemChatConfig()
	
	assert.NotEmpty(t, config.UserID)
	assert.NotEmpty(t, config.SessionID)
	assert.Equal(t, "openai", config.ChatLLM.Provider)
	assert.Equal(t, "gpt-3.5-turbo", config.ChatLLM.Model)
	assert.True(t, config.EnableTextualMemory)
	assert.False(t, config.EnableActivationMemory)
	assert.Equal(t, 5, config.TopK)
	assert.Equal(t, 10, config.MaxTurnsWindow)
}

func TestValidateMemChatConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *MemChatConfig
		wantErr bool
	}{
		{
			name:   "valid config",
			config: DefaultMemChatConfig(),
			wantErr: false,
		},
		{
			name: "missing user_id",
			config: &MemChatConfig{
				SessionID: "test-session",
				ChatLLM: LLMConfig{
					Provider: "openai",
					Model:    "gpt-3.5-turbo",
				},
			},
			wantErr: true,
		},
		{
			name: "missing session_id",
			config: &MemChatConfig{
				UserID: "test-user",
				ChatLLM: LLMConfig{
					Provider: "openai",
					Model:    "gpt-3.5-turbo",
				},
			},
			wantErr: true,
		},
		{
			name: "missing llm provider",
			config: &MemChatConfig{
				UserID:    "test-user",
				SessionID: "test-session",
				ChatLLM: LLMConfig{
					Model: "gpt-3.5-turbo",
				},
			},
			wantErr: true,
		},
		{
			name: "negative top_k",
			config: &MemChatConfig{
				UserID:    "test-user",
				SessionID: "test-session",
				TopK:      -1,
				ChatLLM: LLMConfig{
					Provider: "openai",
					Model:    "gpt-3.5-turbo",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMemChatConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewChatContext(t *testing.T) {
	userID := "test-user"
	sessionID := "test-session"
	mode := ConversationModeAnalytical
	
	context := NewChatContext(userID, sessionID, mode)
	
	assert.Equal(t, userID, context.UserID)
	assert.Equal(t, sessionID, context.SessionID)
	assert.Equal(t, mode, context.Mode)
	assert.NotNil(t, context.Messages)
	assert.NotNil(t, context.Memories)
	assert.NotNil(t, context.Metadata)
	assert.Equal(t, 0, context.TokensUsed)
	assert.Equal(t, 0, context.TurnCount)
}

func TestNewChatMetrics(t *testing.T) {
	metrics := NewChatMetrics()
	
	assert.Equal(t, 0, metrics.TotalMessages)
	assert.Equal(t, 0, metrics.TotalTokens)
	assert.Equal(t, time.Duration(0), metrics.AverageResponseTime)
	assert.Equal(t, 0, metrics.MemoriesRetrieved)
	assert.Equal(t, 0, metrics.MemoriesStored)
	assert.Equal(t, 0, metrics.ErrorCount)
	assert.Equal(t, time.Duration(0), metrics.SessionDuration)
}

func TestChatError(t *testing.T) {
	// Test basic error
	err := NewChatError("test_code", "test message")
	assert.Equal(t, "test_code", err.Code)
	assert.Equal(t, "test message", err.Message)
	assert.Equal(t, "chat_error", err.Type)
	assert.Equal(t, "test message", err.Error())
	
	// Test error with cause
	cause := assert.AnError
	err = NewChatErrorWithCause("test_code", "test message", cause)
	assert.Equal(t, cause, err.Cause)
	assert.Contains(t, err.Error(), "test message")
	assert.Contains(t, err.Error(), cause.Error())
}

func TestConversationModes(t *testing.T) {
	modes := []ConversationMode{
		ConversationModeDefault,
		ConversationModeAnalytical,
		ConversationModeCreative,
		ConversationModeHelpful,
		ConversationModeExplainer,
		ConversationModeSummarizer,
	}
	
	for _, mode := range modes {
		assert.NotEmpty(t, string(mode))
	}
}

func TestChatCommands(t *testing.T) {
	commands := []ChatCommand{
		ChatCommandBye,
		ChatCommandClear,
		ChatCommandMemory,
		ChatCommandExport,
		ChatCommandHelp,
		ChatCommandStats,
		ChatCommandMode,
		ChatCommandSettings,
	}
	
	for _, cmd := range commands {
		assert.NotEmpty(t, string(cmd))
	}
}

func TestMemoryExtractionConfig(t *testing.T) {
	config := &MemoryExtractionConfig{
		ExtractionMode:       "automatic",
		KeywordFilters:      []string{"important", "remember"},
		MinimumLength:       10,
		MaximumLength:       500,
		ImportanceScore:     0.5,
		EnableSummarization: true,
		EnableCategorization: true,
	}
	
	assert.Equal(t, "automatic", config.ExtractionMode)
	assert.Contains(t, config.KeywordFilters, "important")
	assert.Equal(t, 10, config.MinimumLength)
	assert.Equal(t, 500, config.MaximumLength)
	assert.Equal(t, 0.5, config.ImportanceScore)
	assert.True(t, config.EnableSummarization)
	assert.True(t, config.EnableCategorization)
}

func TestChatResult(t *testing.T) {
	result := &ChatResult{
		Response:          "Test response",
		Memories:          []*types.TextualMemoryItem{},
		NewMemories:       []*types.TextualMemoryItem{},
		IsCommand:         false,
		FollowUpQuestions: []string{"Question 1", "Question 2"},
		TokensUsed:        25,
		Duration:          time.Second,
		Timestamp:         time.Now(),
	}
	
	assert.Equal(t, "Test response", result.Response)
	assert.False(t, result.IsCommand)
	assert.Len(t, result.FollowUpQuestions, 2)
	assert.Equal(t, 25, result.TokensUsed)
	assert.Equal(t, time.Second, result.Duration)
}

func TestLLMConfig(t *testing.T) {
	config := LLMConfig{
		Provider:    "openai",
		Model:       "gpt-3.5-turbo",
		APIKey:      "test-key",
		BaseURL:     "https://api.openai.com/v1",
		MaxTokens:   1024,
		Temperature: 0.7,
		TopP:        0.9,
		Timeout:     30 * time.Second,
		Extra:       map[string]interface{}{"custom": "value"},
	}
	
	assert.Equal(t, "openai", config.Provider)
	assert.Equal(t, "gpt-3.5-turbo", config.Model)
	assert.Equal(t, "test-key", config.APIKey)
	assert.Equal(t, 1024, config.MaxTokens)
	assert.Equal(t, 0.7, config.Temperature)
	assert.Contains(t, config.Extra, "custom")
}

func TestStreamingCallback(t *testing.T) {
	var chunks []string
	var complete bool
	var metadata map[string]interface{}
	
	callback := func(chunk string, isComplete bool, meta map[string]interface{}) error {
		chunks = append(chunks, chunk)
		complete = isComplete
		metadata = meta
		return nil
	}
	
	// Test callback
	err := callback("test chunk", false, map[string]interface{}{"tokens": 10})
	assert.NoError(t, err)
	assert.Contains(t, chunks, "test chunk")
	assert.False(t, complete)
	assert.Equal(t, 10, metadata["tokens"])
	
	err = callback("final chunk", true, map[string]interface{}{"tokens": 25})
	assert.NoError(t, err)
	assert.True(t, complete)
	assert.Equal(t, 25, metadata["tokens"])
}

// Benchmark tests
func BenchmarkNewChatContext(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewChatContext("user", "session", ConversationModeDefault)
	}
}

func BenchmarkNewChatMetrics(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewChatMetrics()
	}
}

func BenchmarkValidateMemChatConfig(b *testing.B) {
	config := DefaultMemChatConfig()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateMemChatConfig(config)
	}
}

// Test helper functions
func createTestConfig() *MemChatConfig {
	config := DefaultMemChatConfig()
	config.UserID = "test-user"
	config.SessionID = "test-session"
	return config
}

func createTestContext() *ChatContext {
	return NewChatContext("test-user", "test-session", ConversationModeDefault)
}

func createTestMetrics() *ChatMetrics {
	metrics := NewChatMetrics()
	metrics.TotalMessages = 5
	metrics.TotalTokens = 150
	metrics.AverageResponseTime = time.Millisecond * 500
	return metrics
}

// Integration test helper
func setupTestChat(t *testing.T) BaseMemChat {
	config := createTestConfig()
	
	// Create a simple chat for testing
	chat, err := NewSimpleMemChat(config)
	require.NoError(t, err)
	
	return chat
}

func TestChatIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	chat := setupTestChat(t)
	defer chat.Close()
	
	ctx := context.Background()
	
	// Test basic functionality
	assert.Equal(t, "test-user", chat.GetUserID())
	assert.Equal(t, "test-session", chat.GetSessionID())
	assert.Equal(t, ConversationModeDefault, chat.GetMode())
	
	// Test health check
	err := chat.HealthCheck(ctx)
	assert.NoError(t, err)
	
	// Test configuration
	config := chat.GetConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "test-user", config.UserID)
	
	// Test metrics
	metrics, err := chat.GetMetrics(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, metrics)
}

func TestFactoryIntegration(t *testing.T) {
	factory := GetGlobalFactory()
	
	// Test backend listing
	backends := factory.GetAvailableBackends()
	assert.Contains(t, backends, MemChatBackendSimple)
	
	// Test backend capabilities
	capabilities := factory.GetBackendCapabilities(MemChatBackendSimple)
	assert.NotNil(t, capabilities)
	assert.Contains(t, capabilities, "features")
	
	// Test configuration validation
	config := createTestConfig()
	err := factory.ValidateBackendConfig(MemChatBackendSimple, config)
	assert.NoError(t, err)
}