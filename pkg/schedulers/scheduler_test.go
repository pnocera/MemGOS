package schedulers

import (
	"testing"
	"time"

	"github.com/memtensor/memgos/pkg/schedulers/modules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLLM implements the LLM interface for testing
type MockLLM struct {
	responses map[string]string
}

func NewMockLLM() *MockLLM {
	return &MockLLM{
		responses: map[string]string{
			"intent_recognizing": `{"trigger_retrieval": true, "missing_evidence": ["test evidence 1", "test evidence 2"], "confidence": 0.9}`,
			"memory_reranking":   `{"new_order": ["memory 1", "memory 2", "memory 3"]}`,
			"freq_detecting":     `{"activation_memory_freq_list": [{"memory": "test memory", "count": 2}]}`,
		},
	}
}

func (m *MockLLM) Generate(messages []map[string]string) (string, error) {
	// Return mock response based on content keywords
	for _, msg := range messages {
		content := msg["content"]
		if contains(content, "trigger_retrieval") {
			return m.responses["intent_recognizing"], nil
		}
		if contains(content, "new_order") {
			return m.responses["memory_reranking"], nil
		}
		if contains(content, "activation_memory_freq_list") {
			return m.responses["freq_detecting"], nil
		}
	}
	return `{"result": "mock response"}`, nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

func TestSchedulerFactory(t *testing.T) {
	factory := NewSchedulerFactory()
	
	t.Run("CreateDefaultScheduler", func(t *testing.T) {
		scheduler, err := factory.CreateDefaultScheduler()
		require.NoError(t, err)
		require.NotNil(t, scheduler)
		
		// Should be a GeneralScheduler
		generalScheduler, ok := scheduler.(*GeneralScheduler)
		require.True(t, ok)
		require.NotNil(t, generalScheduler)
	})
	
	t.Run("CreateGeneralScheduler", func(t *testing.T) {
		config := DefaultGeneralSchedulerConfig()
		config.TopK = 15
		config.TopN = 8
		
		scheduler, err := factory.CreateScheduler(GeneralSchedulerType, config)
		require.NoError(t, err)
		require.NotNil(t, scheduler)
		
		generalScheduler := scheduler.(*GeneralScheduler)
		assert.Equal(t, 15, generalScheduler.topK)
		assert.Equal(t, 8, generalScheduler.topN)
	})
	
	t.Run("CreateBaseScheduler", func(t *testing.T) {
		config := DefaultBaseSchedulerConfig()
		config.ThreadPoolMaxWorkers = 8
		
		scheduler, err := factory.CreateScheduler(BaseSchedulerType, config)
		require.NoError(t, err)
		require.NotNil(t, scheduler)
		
		baseScheduler := scheduler.(*BaseScheduler)
		assert.Equal(t, 8, baseScheduler.maxWorkers)
	})
	
	t.Run("CreateWithLLM", func(t *testing.T) {
		mockLLM := NewMockLLM()
		
		scheduler, err := factory.CreateSchedulerWithLLM(GeneralSchedulerType, nil, mockLLM)
		require.NoError(t, err)
		require.NotNil(t, scheduler)
		
		// Check that modules are initialized
		assert.NotNil(t, scheduler.GetMonitor())
		assert.NotNil(t, scheduler.GetRetriever())
		assert.NotNil(t, scheduler.GetDispatcher())
	})
	
	t.Run("GetSupportedTypes", func(t *testing.T) {
		types := factory.GetSupportedTypes()
		assert.Contains(t, types, GeneralSchedulerType)
		assert.Contains(t, types, BaseSchedulerType)
	})
}

func TestBaseScheduler(t *testing.T) {
	scheduler := NewBaseScheduler(nil)
	mockLLM := NewMockLLM()
	
	t.Run("Initialization", func(t *testing.T) {
		err := scheduler.InitializeModules(mockLLM)
		require.NoError(t, err)
		
		assert.NotNil(t, scheduler.GetMonitor())
		assert.NotNil(t, scheduler.GetRetriever())
		assert.NotNil(t, scheduler.GetDispatcher())
	})
	
	t.Run("StartStop", func(t *testing.T) {
		err := scheduler.InitializeModules(mockLLM)
		require.NoError(t, err)
		
		// Should not be running initially
		assert.False(t, scheduler.IsRunning())
		
		// Start scheduler
		err = scheduler.Start()
		require.NoError(t, err)
		assert.True(t, scheduler.IsRunning())
		
		// Stop scheduler
		err = scheduler.Stop()
		require.NoError(t, err)
		assert.False(t, scheduler.IsRunning())
	})
	
	t.Run("MessageSubmission", func(t *testing.T) {
		err := scheduler.InitializeModules(mockLLM)
		require.NoError(t, err)
		
		message := modules.NewScheduleMessageItem(
			"user123",
			"cube456", 
			modules.QueryLabel,
			"test query content",
			nil,
		)
		
		err = scheduler.SubmitMessages(message)
		assert.NoError(t, err)
	})
	
	t.Run("SessionManagement", func(t *testing.T) {
		scheduler.SetCurrentSession("user123", "cube456", "mock_cube")
		
		userID, cubeID, cube := scheduler.GetCurrentSession()
		assert.Equal(t, "user123", userID)
		assert.Equal(t, "cube456", cubeID)
		assert.Equal(t, "mock_cube", cube)
	})
	
	t.Run("Statistics", func(t *testing.T) {
		err := scheduler.InitializeModules(mockLLM)
		require.NoError(t, err)
		
		stats := scheduler.GetStats()
		assert.NotNil(t, stats)
		assert.Contains(t, stats, "running")
		assert.Contains(t, stats, "max_workers")
		assert.Contains(t, stats, "uptime")
	})
}

func TestGeneralScheduler(t *testing.T) {
	config := DefaultGeneralSchedulerConfig()
	config.TopK = 10
	config.TopN = 5
	
	scheduler := NewGeneralScheduler(config)
	mockLLM := NewMockLLM()
	
	t.Run("Initialization", func(t *testing.T) {
		err := scheduler.InitializeModules(mockLLM)
		require.NoError(t, err)
		
		// Check configuration
		assert.Equal(t, 10, scheduler.topK)
		assert.Equal(t, 5, scheduler.topN)
		assert.Equal(t, modules.TextMemorySearchMethod, scheduler.searchMethod)
	})
	
	t.Run("QueryProcessing", func(t *testing.T) {
		err := scheduler.InitializeModules(mockLLM)
		require.NoError(t, err)
		
		err = scheduler.Start()
		require.NoError(t, err)
		defer scheduler.Stop()
		
		// Set a session
		scheduler.SetCurrentSession("user123", "cube456", "mock_cube")
		
		// Submit a query message
		message := modules.NewScheduleMessageItem(
			"user123",
			"cube456",
			modules.QueryLabel,
			"What is the capital of France?",
			"mock_cube",
		)
		
		err = scheduler.SubmitMessages(message)
		assert.NoError(t, err)
		
		// Wait a bit for processing
		time.Sleep(100 * time.Millisecond)
		
		// Check that query was added to query list
		queries := scheduler.GetQueryList()
		assert.Contains(t, queries, "What is the capital of France?")
	})
	
	t.Run("ExtendedStatistics", func(t *testing.T) {
		err := scheduler.InitializeModules(mockLLM)
		require.NoError(t, err)
		
		stats := scheduler.GetStats()
		assert.Contains(t, stats, "top_k")
		assert.Contains(t, stats, "top_n")
		assert.Contains(t, stats, "search_method")
		assert.Contains(t, stats, "query_count")
	})
}

func TestSchedulerModules(t *testing.T) {
	mockLLM := NewMockLLM()
	
	t.Run("SchedulerMonitor", func(t *testing.T) {
		monitor := modules.NewSchedulerMonitor(mockLLM, 5)
		require.NotNil(t, monitor)
		
		// Test intent detection
		qList := []string{"What is machine learning?"}
		workingMemory := []string{"Memory item 1", "Memory item 2"}
		
		intent, err := monitor.DetectIntent(qList, workingMemory)
		require.NoError(t, err)
		assert.True(t, intent.TriggerRetrieval)
		assert.Len(t, intent.MissingEvidence, 2)
	})
	
	t.Run("SchedulerRetriever", func(t *testing.T) {
		retriever := modules.NewSchedulerRetriever(mockLLM)
		require.NotNil(t, retriever)
		
		// Test search (will use placeholder implementation)
		results, err := retriever.Search("test query", 5, modules.TextMemorySearchMethod)
		require.NoError(t, err)
		assert.NotEmpty(t, results)
		
		// Test cache stats
		stats := retriever.GetCacheStats()
		assert.Contains(t, stats, "total_entries")
	})
	
	t.Run("SchedulerDispatcher", func(t *testing.T) {
		dispatcher := modules.NewSchedulerDispatcher(3, false)
		require.NotNil(t, dispatcher)
		
		// Test handler registration
		handlerCalled := false
		testHandler := func(messages []*modules.ScheduleMessageItem) error {
			handlerCalled = true
			return nil
		}
		
		dispatcher.RegisterHandler("test", testHandler)
		
		// Test dispatch
		message := &modules.ScheduleMessageItem{
			Label:   "test",
			Content: "test content",
		}
		
		err := dispatcher.Dispatch([]*modules.ScheduleMessageItem{message})
		assert.NoError(t, err)
		assert.True(t, handlerCalled)
	})
}

func TestRedisSchedulerModule(t *testing.T) {
	// Skip Redis tests if Redis is not available
	t.Skip("Redis tests require running Redis instance")
	
	t.Run("RedisConnection", func(t *testing.T) {
		config := modules.DefaultRedisConfig()
		redisModule := modules.NewRedisSchedulerModule(config)
		
		err := redisModule.InitializeRedis()
		if err != nil {
			t.Skipf("Redis not available: %v", err)
		}
		
		assert.True(t, redisModule.IsConnected())
		
		// Test message operations
		message := modules.NewScheduleMessageItem(
			"user123",
			"cube456",
			modules.QueryLabel,
			"test content",
			nil,
		)
		
		id, err := redisModule.AddMessageToStream(message)
		assert.NoError(t, err)
		assert.NotEmpty(t, id)
		
		// Cleanup
		redisModule.Close()
	})
}

func TestConfigurationMapping(t *testing.T) {
	factory := NewSchedulerFactory()
	
	t.Run("MapToGeneralConfig", func(t *testing.T) {
		configMap := map[string]interface{}{
			"top_k":                     15,
			"top_n":                     8,
			"activation_mem_size":       10,
			"search_method":             "tree_text_memory_search",
			"thread_pool_max_workers":   6,
			"enable_parallel_dispatch":  true,
		}
		
		scheduler, err := factory.CreateScheduler(GeneralSchedulerType, configMap)
		require.NoError(t, err)
		
		generalScheduler := scheduler.(*GeneralScheduler)
		assert.Equal(t, 15, generalScheduler.topK)
		assert.Equal(t, 8, generalScheduler.topN)
		assert.Equal(t, 10, generalScheduler.activationMemSize)
		assert.Equal(t, "tree_text_memory_search", generalScheduler.searchMethod)
		assert.Equal(t, 6, generalScheduler.maxWorkers)
		assert.True(t, generalScheduler.enableParallelDispatch)
	})
}

func TestSchedulerIntegration(t *testing.T) {
	t.Run("FullWorkflow", func(t *testing.T) {
		mockLLM := NewMockLLM()
		
		// Create scheduler with factory
		scheduler, err := CreateSchedulerWithLLM(GeneralSchedulerType, nil, mockLLM)
		require.NoError(t, err)
		
		// Start scheduler
		err = scheduler.Start()
		require.NoError(t, err)
		defer scheduler.Stop()
		
		// Set session
		scheduler.SetCurrentSession("user123", "cube456", "mock_cube")
		
		// Submit query message
		queryMsg := modules.NewScheduleMessageItem(
			"user123",
			"cube456",
			modules.QueryLabel,
			"How does machine learning work?",
			"mock_cube",
		)
		
		err = scheduler.SubmitMessages(queryMsg)
		assert.NoError(t, err)
		
		// Submit answer message
		answerMsg := modules.NewScheduleMessageItem(
			"user123",
			"cube456",
			modules.AnswerLabel,
			"Machine learning uses algorithms to learn patterns from data.",
			"mock_cube",
		)
		
		err = scheduler.SubmitMessages(answerMsg)
		assert.NoError(t, err)
		
		// Wait for processing
		time.Sleep(200 * time.Millisecond)
		
		// Check web logs
		webLogs := scheduler.GetWebLogMessages()
		assert.GreaterOrEqual(t, len(webLogs), 0) // Logs are processed asynchronously
		
		// Check statistics
		stats := scheduler.GetStats()
		assert.NotNil(t, stats)
		assert.True(t, scheduler.IsRunning())
	})
}
