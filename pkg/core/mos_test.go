package core

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/errors"
	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/memory"
	"github.com/memtensor/memgos/pkg/types"
)

// Mock implementations for testing
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(msg string, fields ...map[string]interface{}) {
	m.Called(msg, fields)
}

func (m *MockLogger) Info(msg string, fields ...map[string]interface{}) {
	m.Called(msg, fields)
}

func (m *MockLogger) Warn(msg string, fields ...map[string]interface{}) {
	m.Called(msg, fields)
}

func (m *MockLogger) Error(msg string, err error, fields ...map[string]interface{}) {
	m.Called(msg, err, fields)
}

func (m *MockLogger) Fatal(msg string, err error, fields ...map[string]interface{}) {
	m.Called(msg, err, fields)
}

func (m *MockLogger) WithFields(fields map[string]interface{}) interfaces.Logger {
	args := m.Called(fields)
	return args.Get(0).(interfaces.Logger)
}

type MockMetrics struct {
	mock.Mock
}

func (m *MockMetrics) Counter(name string, value float64, labels map[string]string) {
	m.Called(name, value, labels)
}

func (m *MockMetrics) Gauge(name string, value float64, labels map[string]string) {
	m.Called(name, value, labels)
}

func (m *MockMetrics) Histogram(name string, value float64, labels map[string]string) {
	m.Called(name, value, labels)
}

func (m *MockMetrics) Timer(name string, duration float64, labels map[string]string) {
	m.Called(name, duration, labels)
}

type MockUserManager struct {
	mock.Mock
}

func (m *MockUserManager) ValidateUser(ctx context.Context, userID string) (bool, error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserManager) CreateUser(ctx context.Context, userName string, role types.UserRole, userID string) (string, error) {
	args := m.Called(ctx, userName, role, userID)
	return args.String(0), args.Error(1)
}

func (m *MockUserManager) GetUser(ctx context.Context, userID string) (*types.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.User), args.Error(1)
}

func (m *MockUserManager) GetUserCubes(ctx context.Context, userID string) ([]*types.MemCube, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*types.MemCube), args.Error(1)
}

type MockChatManager struct {
	mock.Mock
}

func (m *MockChatManager) GetChatHistory(ctx context.Context, userID, sessionID string) (*types.ChatHistory, error) {
	args := m.Called(ctx, userID, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.ChatHistory), args.Error(1)
}

func (m *MockChatManager) SaveChatHistory(ctx context.Context, history *types.ChatHistory) error {
	args := m.Called(ctx, history)
	return args.Error(0)
}

type MockLLM struct {
	mock.Mock
}

func (m *MockLLM) Generate(ctx context.Context, messages types.MessageList) (string, error) {
	args := m.Called(ctx, messages)
	return args.String(0), args.Error(1)
}

func (m *MockLLM) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockScheduler struct {
	mock.Mock
}

func (m *MockScheduler) Schedule(ctx context.Context, task *types.ScheduledTask) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *MockScheduler) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestNewMOSCore(t *testing.T) {
	t.Run("NewMOSCore Success", func(t *testing.T) {
		cfg := &config.MOSConfig{
			UserID:    "test-user",
			SessionID: "test-session",
		}
		logger := &MockLogger{}
		metrics := &MockMetrics{}

		core, err := NewMOSCore(cfg, logger, metrics)
		
		require.NoError(t, err)
		assert.NotNil(t, core)
		assert.Equal(t, cfg, core.config)
		assert.Equal(t, "test-user", core.userID)
		assert.Equal(t, "test-session", core.sessionID)
		assert.Equal(t, logger, core.logger)
		assert.Equal(t, metrics, core.metrics)
		assert.NotNil(t, core.cubeRegistry)
		assert.False(t, core.initialized)
		assert.False(t, core.closed)
	})

	t.Run("NewMOSCore Nil Config", func(t *testing.T) {
		logger := &MockLogger{}
		metrics := &MockMetrics{}

		core, err := NewMOSCore(nil, logger, metrics)
		
		assert.Error(t, err)
		assert.Nil(t, core)
		assert.Contains(t, err.Error(), "configuration is required")
	})
}

func TestMOSCoreInitialize(t *testing.T) {
	t.Run("Initialize Success", func(t *testing.T) {
		cfg := &config.MOSConfig{
			UserID:                 "test-user",
			SessionID:              "test-session",
			EnableTextualMemory:    true,
			EnableActivationMemory: false,
			EnableParametricMemory: false,
			EnableMemScheduler:     false,
			HealthCheckEnabled:     false,
		}
		logger := &MockLogger{}
		metrics := &MockMetrics{}

		// Setup logger expectations
		logger.On("Info", "LLM initialization placeholder", mock.Anything).Return()
		logger.On("Info", "User manager initialization placeholder", mock.Anything).Return()
		logger.On("Info", "Chat manager initialization placeholder", mock.Anything).Return()
		logger.On("Info", "MOS Core initialized", mock.Anything).Return()

		// Setup metrics expectations
		metrics.On("Timer", "mos_core_initialize_duration", mock.AnythingOfType("float64"), mock.Anything).Return()
		metrics.On("Counter", "mos_core_initialize_count", float64(1), mock.Anything).Return()

		core, err := NewMOSCore(cfg, logger, metrics)
		require.NoError(t, err)

		ctx := context.Background()
		err = core.Initialize(ctx)
		
		assert.NoError(t, err)
		assert.True(t, core.initialized)
		
		logger.AssertExpectations(t)
		metrics.AssertExpectations(t)
	})

	t.Run("Initialize Already Initialized", func(t *testing.T) {
		cfg := &config.MOSConfig{
			UserID:    "test-user",
			SessionID: "test-session",
		}
		logger := &MockLogger{}
		metrics := &MockMetrics{}

		core, err := NewMOSCore(cfg, logger, metrics)
		require.NoError(t, err)
		
		core.initialized = true
		
		ctx := context.Background()
		err = core.Initialize(ctx)
		
		assert.NoError(t, err) // Should return nil if already initialized
	})

	t.Run("Initialize Closed Core", func(t *testing.T) {
		cfg := &config.MOSConfig{
			UserID:    "test-user",
			SessionID: "test-session",
		}
		logger := &MockLogger{}
		metrics := &MockMetrics{}

		core, err := NewMOSCore(cfg, logger, metrics)
		require.NoError(t, err)
		
		core.closed = true
		
		ctx := context.Background()
		err = core.Initialize(ctx)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "MOS core is closed")
	})
}

func TestMOSCoreSearch(t *testing.T) {
	t.Run("Search Not Initialized", func(t *testing.T) {
		cfg := &config.MOSConfig{
			UserID:    "test-user",
			SessionID: "test-session",
		}
		logger := &MockLogger{}
		metrics := &MockMetrics{}

		core, err := NewMOSCore(cfg, logger, metrics)
		require.NoError(t, err)

		ctx := context.Background()
		query := &types.SearchQuery{
			Query: "test query",
		}

		result, err := core.Search(ctx, query)
		
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "MOS core not initialized")
	})

	t.Run("Search Success", func(t *testing.T) {
		cfg := &config.MOSConfig{
			UserID:                 "test-user",
			SessionID:              "test-session",
			EnableTextualMemory:    true,
			EnableActivationMemory: false,
			EnableParametricMemory: false,
			TopK:                   5,
		}
		logger := &MockLogger{}
		metrics := &MockMetrics{}

		// Setup logger expectations
		logger.On("Info", mock.AnythingOfType("string"), mock.Anything).Return()
		logger.On("Warn", mock.AnythingOfType("string"), mock.Anything).Return().Maybe()

		// Setup metrics expectations
		metrics.On("Timer", "mos_search_duration", mock.AnythingOfType("float64"), mock.Anything).Return()
		metrics.On("Counter", "mos_search_count", float64(1), mock.Anything).Return()
		metrics.On("Gauge", mock.AnythingOfType("string"), mock.AnythingOfType("float64"), mock.Anything).Return().Maybe()

		core, err := NewMOSCore(cfg, logger, metrics)
		require.NoError(t, err)
		
		core.initialized = true

		ctx := context.Background()
		query := &types.SearchQuery{
			Query:  "test query",
			UserID: "test-user",
			TopK:   3,
		}

		result, err := core.Search(ctx, query)
		
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.TextMem)
		assert.NotNil(t, result.ActMem)
		assert.NotNil(t, result.ParaMem)
	})
}

func TestMOSCoreAdd(t *testing.T) {
	t.Run("Add Not Initialized", func(t *testing.T) {
		cfg := &config.MOSConfig{
			UserID:    "test-user",
			SessionID: "test-session",
		}
		logger := &MockLogger{}
		metrics := &MockMetrics{}

		core, err := NewMOSCore(cfg, logger, metrics)
		require.NoError(t, err)

		ctx := context.Background()
		content := "test content"
		request := &types.AddMemoryRequest{
			MemoryContent: &content,
		}

		err = core.Add(ctx, request)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "MOS core not initialized")
	})

	t.Run("Add Invalid Request", func(t *testing.T) {
		cfg := &config.MOSConfig{
			UserID:    "test-user",
			SessionID: "test-session",
		}
		logger := &MockLogger{}
		metrics := &MockMetrics{}

		core, err := NewMOSCore(cfg, logger, metrics)
		require.NoError(t, err)
		
		core.initialized = true

		ctx := context.Background()
		request := &types.AddMemoryRequest{} // No content

		err = core.Add(ctx, request)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one of messages, memory_content, or doc_path must be provided")
	})

	t.Run("Add No Available Cubes", func(t *testing.T) {
		cfg := &config.MOSConfig{
			UserID:    "test-user",
			SessionID: "test-session",
		}
		logger := &MockLogger{}
		metrics := &MockMetrics{}

		core, err := NewMOSCore(cfg, logger, metrics)
		require.NoError(t, err)
		
		core.initialized = true

		ctx := context.Background()
		content := "test content"
		request := &types.AddMemoryRequest{
			MemoryContent: &content,
		}

		err = core.Add(ctx, request)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no memory cubes available")
	})
}

func TestMOSCoreChat(t *testing.T) {
	t.Run("Chat Not Initialized", func(t *testing.T) {
		cfg := &config.MOSConfig{
			UserID:    "test-user",
			SessionID: "test-session",
		}
		logger := &MockLogger{}
		metrics := &MockMetrics{}

		core, err := NewMOSCore(cfg, logger, metrics)
		require.NoError(t, err)

		ctx := context.Background()
		request := &types.ChatRequest{
			Query: "Hello",
		}

		response, err := core.Chat(ctx, request)
		
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "MOS core not initialized")
	})

	t.Run("Chat Success", func(t *testing.T) {
		cfg := &config.MOSConfig{
			UserID:                 "test-user",
			SessionID:              "test-session",
			EnableTextualMemory:    true,
			EnableActivationMemory: false,
			EnableParametricMemory: false,
			EnableMemScheduler:     false,
			TopK:                   5,
		}
		logger := &MockLogger{}
		metrics := &MockMetrics{}

		// Setup logger expectations
		logger.On("Info", mock.AnythingOfType("string"), mock.Anything).Return()
		logger.On("Warn", mock.AnythingOfType("string"), mock.Anything).Return().Maybe()

		// Setup metrics expectations
		metrics.On("Timer", "mos_chat_duration", mock.AnythingOfType("float64"), mock.Anything).Return()
		metrics.On("Counter", "mos_chat_count", float64(1), mock.Anything).Return()

		core, err := NewMOSCore(cfg, logger, metrics)
		require.NoError(t, err)
		
		core.initialized = true

		ctx := context.Background()
		userID := "test-user"
		request := &types.ChatRequest{
			Query:  "Hello, how are you?",
			UserID: &userID,
		}

		response, err := core.Chat(ctx, request)
		
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, "Chat LLM not available", response.Response)
		assert.Equal(t, cfg.SessionID, response.SessionID)
		assert.Equal(t, "test-user", response.UserID)
		assert.NotZero(t, response.Timestamp)
	})
}

func TestMOSCoreUserManagement(t *testing.T) {
	t.Run("CreateUser Not Initialized", func(t *testing.T) {
		cfg := &config.MOSConfig{
			UserID:    "test-user",
			SessionID: "test-session",
		}
		logger := &MockLogger{}
		metrics := &MockMetrics{}

		core, err := NewMOSCore(cfg, logger, metrics)
		require.NoError(t, err)

		ctx := context.Background()
		userID, err := core.CreateUser(ctx, "Test User", types.UserRoleUser, "new-user")
		
		assert.Error(t, err)
		assert.Empty(t, userID)
		assert.Contains(t, err.Error(), "MOS core not initialized")
	})

	t.Run("CreateUser No User Manager", func(t *testing.T) {
		cfg := &config.MOSConfig{
			UserID:    "test-user",
			SessionID: "test-session",
		}
		logger := &MockLogger{}
		metrics := &MockMetrics{}

		core, err := NewMOSCore(cfg, logger, metrics)
		require.NoError(t, err)
		
		core.initialized = true

		ctx := context.Background()
		userID, err := core.CreateUser(ctx, "Test User", types.UserRoleUser, "new-user")
		
		assert.Error(t, err)
		assert.Empty(t, userID)
		assert.Contains(t, err.Error(), "user manager not available")
	})

	t.Run("GetUserInfo Not Initialized", func(t *testing.T) {
		cfg := &config.MOSConfig{
			UserID:    "test-user",
			SessionID: "test-session",
		}
		logger := &MockLogger{}
		metrics := &MockMetrics{}

		core, err := NewMOSCore(cfg, logger, metrics)
		require.NoError(t, err)

		ctx := context.Background()
		info, err := core.GetUserInfo(ctx, "test-user")
		
		assert.Error(t, err)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "MOS core not initialized")
	})

	t.Run("GetUserInfo No User Manager", func(t *testing.T) {
		cfg := &config.MOSConfig{
			UserID:    "test-user",
			SessionID: "test-session",
		}
		logger := &MockLogger{}
		metrics := &MockMetrics{}

		core, err := NewMOSCore(cfg, logger, metrics)
		require.NoError(t, err)
		
		core.initialized = true

		ctx := context.Background()
		info, err := core.GetUserInfo(ctx, "test-user")
		
		assert.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, "test-user", info["user_id"])
		assert.Equal(t, "user manager not available", info["message"])
	})
}

func TestMOSCoreClose(t *testing.T) {
	t.Run("Close Success", func(t *testing.T) {
		cfg := &config.MOSConfig{
			UserID:    "test-user",
			SessionID: "test-session",
		}
		logger := &MockLogger{}
		metrics := &MockMetrics{}

		// Setup logger expectations
		logger.On("Info", "MOS Core closed", mock.Anything).Return()

		core, err := NewMOSCore(cfg, logger, metrics)
		require.NoError(t, err)

		err = core.Close()
		
		assert.NoError(t, err)
		assert.True(t, core.closed)
		
		logger.AssertExpectations(t)
	})

	t.Run("Close Already Closed", func(t *testing.T) {
		cfg := &config.MOSConfig{
			UserID:    "test-user",
			SessionID: "test-session",
		}
		logger := &MockLogger{}
		metrics := &MockMetrics{}

		core, err := NewMOSCore(cfg, logger, metrics)
		require.NoError(t, err)
		
		core.closed = true

		err = core.Close()
		
		assert.NoError(t, err) // Should not error if already closed
	})
}

func TestMOSCoreHelperMethods(t *testing.T) {
	t.Run("isLocalPath", func(t *testing.T) {
		cfg := &config.MOSConfig{
			UserID:    "test-user",
			SessionID: "test-session",
		}
		logger := &MockLogger{}
		metrics := &MockMetrics{}

		core, err := NewMOSCore(cfg, logger, metrics)
		require.NoError(t, err)

		// Test local paths
		assert.True(t, core.isLocalPath("/path/to/cube"))
		assert.True(t, core.isLocalPath("./relative/path"))
		assert.True(t, core.isLocalPath("../parent/path"))
		assert.True(t, core.isLocalPath("file://local/file"))

		// Test remote paths
		assert.False(t, core.isLocalPath("http://example.com/cube"))
		assert.False(t, core.isLocalPath("https://github.com/user/repo"))
		assert.False(t, core.isLocalPath("HTTP://EXAMPLE.COM/CUBE")) // case insensitive
	})

	t.Run("buildSystemPrompt", func(t *testing.T) {
		cfg := &config.MOSConfig{
			UserID:    "test-user",
			SessionID: "test-session",
			TopK:      3,
		}
		logger := &MockLogger{}
		metrics := &MockMetrics{}

		core, err := NewMOSCore(cfg, logger, metrics)
		require.NoError(t, err)

		// Test with no memories
		prompt := core.buildSystemPrompt([]types.MemoryItem{})
		assert.Contains(t, prompt, "knowledgeable and helpful AI assistant")
		assert.NotContains(t, prompt, "## Memories:")

		// Test with memories
		memories := []types.MemoryItem{
			&types.TextualMemoryItem{
				ID:     "mem1",
				Memory: "Test memory 1",
			},
			&types.TextualMemoryItem{
				ID:     "mem2",
				Memory: "Test memory 2",
			},
		}

		prompt = core.buildSystemPrompt(memories)
		assert.Contains(t, prompt, "knowledgeable and helpful AI assistant")
		assert.Contains(t, prompt, "## Memories:")
		assert.Contains(t, prompt, "1. Test memory 1")
		assert.Contains(t, prompt, "2. Test memory 2")
	})
}

func TestMOSCoreErrorConditions(t *testing.T) {
	t.Run("Initialize with Validation Error", func(t *testing.T) {
		cfg := &config.MOSConfig{
			UserID:                 "test-user",
			SessionID:              "test-session",
			EnableTextualMemory:    true,
			EnableActivationMemory: false,
			EnableParametricMemory: false,
			EnableMemScheduler:     false,
			HealthCheckEnabled:     false,
		}
		logger := &MockLogger{}
		metrics := &MockMetrics{}
		userManager := &MockUserManager{}

		// Setup logger expectations
		logger.On("Info", mock.AnythingOfType("string"), mock.Anything).Return()

		// Setup metrics expectations
		metrics.On("Timer", "mos_core_initialize_duration", mock.AnythingOfType("float64"), mock.Anything).Return()

		// Setup user manager to return validation error
		userManager.On("ValidateUser", mock.Anything, "test-user").Return(false, errors.NewValidationError("user not found"))

		core, err := NewMOSCore(cfg, logger, metrics)
		require.NoError(t, err)
		
		core.userManager = userManager

		ctx := context.Background()
		err = core.Initialize(ctx)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to validate user")
		assert.False(t, core.initialized)
	})

	t.Run("Close with Scheduler Error", func(t *testing.T) {
		cfg := &config.MOSConfig{
			UserID:    "test-user",
			SessionID: "test-session",
		}
		logger := &MockLogger{}
		metrics := &MockMetrics{}
		scheduler := &MockScheduler{}

		// Setup logger expectations
		logger.On("Info", "MOS Core closed", mock.Anything).Return()

		// Setup scheduler to return error on stop
		scheduler.On("Stop", mock.Anything).Return(errors.NewInternalError("scheduler stop failed"))

		core, err := NewMOSCore(cfg, logger, metrics)
		require.NoError(t, err)
		
		core.scheduler = scheduler

		err = core.Close()
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "errors occurred while closing MOS core")
		assert.Contains(t, err.Error(), "scheduler stop failed")
		assert.True(t, core.closed) // Should still be marked as closed
	})
}

// Benchmark tests
func BenchmarkNewMOSCore(b *testing.B) {
	cfg := &config.MOSConfig{
		UserID:    "test-user",
		SessionID: "test-session",
	}
	logger := &MockLogger{}
	metrics := &MockMetrics{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		core, _ := NewMOSCore(cfg, logger, metrics)
		_ = core
	}
}

func BenchmarkMOSCoreBuildSystemPrompt(b *testing.B) {
	cfg := &config.MOSConfig{
		UserID:    "test-user",
		SessionID: "test-session",
		TopK:      5,
	}
	logger := &MockLogger{}
	metrics := &MockMetrics{}

	core, _ := NewMOSCore(cfg, logger, metrics)

	memories := make([]types.MemoryItem, 10)
	for i := 0; i < 10; i++ {
		memories[i] = &types.TextualMemoryItem{
			ID:     string(rune(i)),
			Memory: "Test memory content for benchmarking",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = core.buildSystemPrompt(memories)
	}
}

// Integration tests
func TestMOSCoreIntegration(t *testing.T) {
	t.Run("Complete Workflow", func(t *testing.T) {
		cfg := &config.MOSConfig{
			UserID:                 "test-user",
			SessionID:              "test-session",
			EnableTextualMemory:    true,
			EnableActivationMemory: false,
			EnableParametricMemory: false,
			EnableMemScheduler:     false,
			HealthCheckEnabled:     false,
			TopK:                   5,
		}
		logger := &MockLogger{}
		metrics := &MockMetrics{}

		// Setup expectations for complete workflow
		logger.On("Info", mock.AnythingOfType("string"), mock.Anything).Return()
		logger.On("Warn", mock.AnythingOfType("string"), mock.Anything).Return().Maybe()
		metrics.On("Timer", mock.AnythingOfType("string"), mock.AnythingOfType("float64"), mock.Anything).Return()
		metrics.On("Counter", mock.AnythingOfType("string"), mock.AnythingOfType("float64"), mock.Anything).Return()
		metrics.On("Gauge", mock.AnythingOfType("string"), mock.AnythingOfType("float64"), mock.Anything).Return().Maybe()

		// Create and initialize core
		core, err := NewMOSCore(cfg, logger, metrics)
		require.NoError(t, err)

		ctx := context.Background()
		err = core.Initialize(ctx)
		require.NoError(t, err)
		assert.True(t, core.initialized)

		// Test search (should work even with no cubes)
		searchQuery := &types.SearchQuery{
			Query:  "test search",
			UserID: cfg.UserID,
			TopK:   3,
		}

		searchResult, err := core.Search(ctx, searchQuery)
		assert.NoError(t, err)
		assert.NotNil(t, searchResult)

		// Test chat
		chatRequest := &types.ChatRequest{
			Query:  "Hello!",
			UserID: &cfg.UserID,
		}

		chatResponse, err := core.Chat(ctx, chatRequest)
		assert.NoError(t, err)
		assert.NotNil(t, chatResponse)
		assert.Equal(t, cfg.UserID, chatResponse.UserID)
		assert.Equal(t, cfg.SessionID, chatResponse.SessionID)

		// Test user info
		userInfo, err := core.GetUserInfo(ctx, cfg.UserID)
		assert.NoError(t, err)
		assert.NotNil(t, userInfo)

		// Close core
		err = core.Close()
		assert.NoError(t, err)
		assert.True(t, core.closed)

		// Verify expectations
		logger.AssertExpectations(t)
		metrics.AssertExpectations(t)
	})
}

// Example tests for documentation
func ExampleNewMOSCore() {
	cfg := &config.MOSConfig{
		UserID:                 "user123",
		SessionID:              "session456",
		EnableTextualMemory:    true,
		EnableActivationMemory: false,
		EnableParametricMemory: false,
		TopK:                   5,
	}

	logger := &MockLogger{}
	metrics := &MockMetrics{}

	core, err := NewMOSCore(cfg, logger, metrics)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	err = core.Initialize(ctx)
	if err != nil {
		panic(err)
	}

	defer core.Close()

	// Now the core is ready to use
	println("MOS Core initialized successfully")
}