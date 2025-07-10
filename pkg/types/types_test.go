package types

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageRole(t *testing.T) {
	t.Run("MessageRole Constants", func(t *testing.T) {
		assert.Equal(t, MessageRole("user"), MessageRoleUser)
		assert.Equal(t, MessageRole("assistant"), MessageRoleAssistant)
		assert.Equal(t, MessageRole("system"), MessageRoleSystem)
	})
}

func TestMessageDict(t *testing.T) {
	t.Run("MessageDict Creation", func(t *testing.T) {
		msg := MessageDict{
			Role:    MessageRoleUser,
			Content: "Hello, world!",
		}
		
		assert.Equal(t, MessageRoleUser, msg.Role)
		assert.Equal(t, "Hello, world!", msg.Content)
	})

	t.Run("MessageDict JSON Serialization", func(t *testing.T) {
		msg := MessageDict{
			Role:    MessageRoleAssistant,
			Content: "Hello! How can I help you today?",
		}
		
		data, err := json.Marshal(msg)
		require.NoError(t, err)
		
		var decoded MessageDict
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		
		assert.Equal(t, msg.Role, decoded.Role)
		assert.Equal(t, msg.Content, decoded.Content)
	})
}

func TestMessageList(t *testing.T) {
	t.Run("MessageList Operations", func(t *testing.T) {
		messages := MessageList{
			{Role: MessageRoleUser, Content: "Hello"},
			{Role: MessageRoleAssistant, Content: "Hi there!"},
			{Role: MessageRoleUser, Content: "How are you?"},
		}
		
		assert.Len(t, messages, 3)
		assert.Equal(t, MessageRoleUser, messages[0].Role)
		assert.Equal(t, MessageRoleAssistant, messages[1].Role)
		assert.Equal(t, "How are you?", messages[2].Content)
	})

	t.Run("MessageList JSON Serialization", func(t *testing.T) {
		messages := MessageList{
			{Role: MessageRoleSystem, Content: "System prompt"},
			{Role: MessageRoleUser, Content: "User message"},
		}
		
		data, err := json.Marshal(messages)
		require.NoError(t, err)
		
		var decoded MessageList
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		
		assert.Len(t, decoded, 2)
		assert.Equal(t, messages[0].Role, decoded[0].Role)
		assert.Equal(t, messages[1].Content, decoded[1].Content)
	})
}

func TestChatHistory(t *testing.T) {
	t.Run("ChatHistory Creation", func(t *testing.T) {
		now := time.Now()
		history := ChatHistory{
			UserID:        "user123",
			SessionID:     "session456",
			CreatedAt:     now,
			TotalMessages: 2,
			ChatHistory: MessageList{
				{Role: MessageRoleUser, Content: "Hello"},
				{Role: MessageRoleAssistant, Content: "Hi!"},
			},
		}
		
		assert.Equal(t, "user123", history.UserID)
		assert.Equal(t, "session456", history.SessionID)
		assert.Equal(t, 2, history.TotalMessages)
		assert.Len(t, history.ChatHistory, 2)
		assert.Equal(t, now, history.CreatedAt)
	})

	t.Run("ChatHistory JSON Serialization", func(t *testing.T) {
		history := ChatHistory{
			UserID:        "user123",
			SessionID:     "session456",
			CreatedAt:     time.Now(),
			TotalMessages: 1,
			ChatHistory: MessageList{
				{Role: MessageRoleUser, Content: "Test message"},
			},
		}
		
		data, err := json.Marshal(history)
		require.NoError(t, err)
		
		var decoded ChatHistory
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		
		assert.Equal(t, history.UserID, decoded.UserID)
		assert.Equal(t, history.SessionID, decoded.SessionID)
		assert.Equal(t, history.TotalMessages, decoded.TotalMessages)
		assert.Len(t, decoded.ChatHistory, 1)
	})
}

func TestTextualMemoryItem(t *testing.T) {
	now := time.Now()
	metadata := map[string]interface{}{
		"source": "test",
		"tags":   []string{"important", "conversation"},
	}
	
	item := &TextualMemoryItem{
		ID:        "mem123",
		Memory:    "This is a test memory",
		Metadata:  metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	t.Run("MemoryItem Interface Implementation", func(t *testing.T) {
		var memItem MemoryItem = item
		
		assert.Equal(t, "mem123", memItem.GetID())
		assert.Equal(t, "This is a test memory", memItem.GetContent())
		assert.Equal(t, metadata, memItem.GetMetadata())
		assert.Equal(t, now, memItem.GetCreatedAt())
		assert.Equal(t, now, memItem.GetUpdatedAt())
	})

	t.Run("Direct Method Access", func(t *testing.T) {
		assert.Equal(t, "mem123", item.GetID())
		assert.Equal(t, "This is a test memory", item.GetContent())
		assert.Equal(t, metadata, item.GetMetadata())
		assert.Equal(t, now, item.GetCreatedAt())
		assert.Equal(t, now, item.GetUpdatedAt())
	})

	t.Run("JSON Serialization", func(t *testing.T) {
		data, err := json.Marshal(item)
		require.NoError(t, err)
		
		var decoded TextualMemoryItem
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		
		assert.Equal(t, item.ID, decoded.ID)
		assert.Equal(t, item.Memory, decoded.Memory)
		assert.Equal(t, item.Metadata["source"], decoded.Metadata["source"])
	})
}

func TestActivationMemoryItem(t *testing.T) {
	now := time.Now()
	activationData := map[string]interface{}{
		"layer_1": []float32{0.1, 0.2, 0.3},
		"layer_2": []float32{0.4, 0.5, 0.6},
	}
	metadata := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"layer": 12,
	}
	
	item := &ActivationMemoryItem{
		ID:        "act123",
		Memory:    activationData,
		Metadata:  metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	t.Run("MemoryItem Interface Implementation", func(t *testing.T) {
		var memItem MemoryItem = item
		
		assert.Equal(t, "act123", memItem.GetID())
		assert.Equal(t, "", memItem.GetContent()) // Activation memory has no string content
		assert.Equal(t, metadata, memItem.GetMetadata())
		assert.Equal(t, now, memItem.GetCreatedAt())
		assert.Equal(t, now, memItem.GetUpdatedAt())
	})

	t.Run("Activation Data Access", func(t *testing.T) {
		assert.Equal(t, activationData, item.Memory)
		assert.Equal(t, "gpt-3.5-turbo", item.Metadata["model"])
		assert.Equal(t, 12, item.Metadata["layer"])
	})
}

func TestParametricMemoryItem(t *testing.T) {
	now := time.Now()
	parametricData := map[string]interface{}{
		"weights": [][]float32{{0.1, 0.2}, {0.3, 0.4}},
		"bias":    []float32{0.5, 0.6},
	}
	metadata := map[string]interface{}{
		"adapter_name": "lora_adapter",
		"rank":         16,
	}
	
	item := &ParametricMemoryItem{
		ID:        "param123",
		Memory:    parametricData,
		Metadata:  metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	t.Run("MemoryItem Interface Implementation", func(t *testing.T) {
		var memItem MemoryItem = item
		
		assert.Equal(t, "param123", memItem.GetID())
		assert.Equal(t, "", memItem.GetContent()) // Parametric memory has no string content
		assert.Equal(t, metadata, memItem.GetMetadata())
		assert.Equal(t, now, memItem.GetCreatedAt())
		assert.Equal(t, now, memItem.GetUpdatedAt())
	})

	t.Run("Parametric Data Access", func(t *testing.T) {
		assert.Equal(t, parametricData, item.Memory)
		assert.Equal(t, "lora_adapter", item.Metadata["adapter_name"])
		assert.Equal(t, 16, item.Metadata["rank"])
	})
}

func TestMOSSearchResult(t *testing.T) {
	t.Run("MOSSearchResult Structure", func(t *testing.T) {
		result := MOSSearchResult{
			TextMem: []CubeMemoryResult{
				{CubeID: "cube1", Memories: []MemoryItem{}},
			},
			ActMem: []CubeMemoryResult{
				{CubeID: "cube2", Memories: []MemoryItem{}},
			},
			ParaMem: []CubeMemoryResult{},
		}
		
		assert.Len(t, result.TextMem, 1)
		assert.Len(t, result.ActMem, 1)
		assert.Len(t, result.ParaMem, 0)
		assert.Equal(t, "cube1", result.TextMem[0].CubeID)
		assert.Equal(t, "cube2", result.ActMem[0].CubeID)
	})

	t.Run("MOSSearchResult JSON Serialization", func(t *testing.T) {
		textItem := &TextualMemoryItem{
			ID:        "text1",
			Memory:    "test content",
			Metadata:  map[string]interface{}{"type": "text"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		result := MOSSearchResult{
			TextMem: []CubeMemoryResult{
				{CubeID: "cube1", Memories: []MemoryItem{textItem}},
			},
			ActMem:  []CubeMemoryResult{},
			ParaMem: []CubeMemoryResult{},
		}
		
		data, err := json.Marshal(result)
		require.NoError(t, err)
		assert.Contains(t, string(data), "text_mem")
		assert.Contains(t, string(data), "cube1")
	})
}

func TestUserRole(t *testing.T) {
	t.Run("UserRole Constants", func(t *testing.T) {
		assert.Equal(t, UserRole("user"), UserRoleUser)
		assert.Equal(t, UserRole("admin"), UserRoleAdmin)
		assert.Equal(t, UserRole("guest"), UserRoleGuest)
	})
}

func TestUser(t *testing.T) {
	t.Run("User Creation", func(t *testing.T) {
		now := time.Now()
		user := User{
			UserID:    "user123",
			UserName:  "John Doe",
			Role:      UserRoleUser,
			CreatedAt: now,
			UpdatedAt: now,
			IsActive:  true,
		}
		
		assert.Equal(t, "user123", user.UserID)
		assert.Equal(t, "John Doe", user.UserName)
		assert.Equal(t, UserRoleUser, user.Role)
		assert.True(t, user.IsActive)
		assert.Equal(t, now, user.CreatedAt)
	})

	t.Run("User JSON Serialization", func(t *testing.T) {
		user := User{
			UserID:    "user456",
			UserName:  "Jane Smith",
			Role:      UserRoleAdmin,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			IsActive:  true,
		}
		
		data, err := json.Marshal(user)
		require.NoError(t, err)
		
		var decoded User
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		
		assert.Equal(t, user.UserID, decoded.UserID)
		assert.Equal(t, user.UserName, decoded.UserName)
		assert.Equal(t, user.Role, decoded.Role)
		assert.Equal(t, user.IsActive, decoded.IsActive)
	})
}

func TestMemCube(t *testing.T) {
	t.Run("MemCube Creation", func(t *testing.T) {
		now := time.Now()
		cube := MemCube{
			ID:        "cube123",
			Name:      "Test Cube",
			Path:      "/path/to/cube",
			OwnerID:   "user123",
			CreatedAt: now,
			UpdatedAt: now,
			IsLoaded:  true,
		}
		
		assert.Equal(t, "cube123", cube.ID)
		assert.Equal(t, "Test Cube", cube.Name)
		assert.Equal(t, "/path/to/cube", cube.Path)
		assert.Equal(t, "user123", cube.OwnerID)
		assert.True(t, cube.IsLoaded)
	})
}

func TestSearchQuery(t *testing.T) {
	t.Run("SearchQuery Creation", func(t *testing.T) {
		query := SearchQuery{
			Query:   "search term",
			TopK:    10,
			Filters: map[string]string{"type": "conversation"},
			CubeIDs: []string{"cube1", "cube2"},
			UserID:  "user123",
			SessionID: "session456",
		}
		
		assert.Equal(t, "search term", query.Query)
		assert.Equal(t, 10, query.TopK)
		assert.Equal(t, "conversation", query.Filters["type"])
		assert.Len(t, query.CubeIDs, 2)
		assert.Equal(t, "user123", query.UserID)
	})

	t.Run("SearchQuery JSON Serialization", func(t *testing.T) {
		query := SearchQuery{
			Query:   "test query",
			TopK:    5,
			UserID:  "user789",
		}
		
		data, err := json.Marshal(query)
		require.NoError(t, err)
		
		var decoded SearchQuery
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		
		assert.Equal(t, query.Query, decoded.Query)
		assert.Equal(t, query.TopK, decoded.TopK)
		assert.Equal(t, query.UserID, decoded.UserID)
	})
}

func TestAddMemoryRequest(t *testing.T) {
	t.Run("AddMemoryRequest with Messages", func(t *testing.T) {
		messages := &MessageList{
			{Role: MessageRoleUser, Content: "Hello"},
			{Role: MessageRoleAssistant, Content: "Hi!"},
		}
		cubeID := "cube123"
		userID := "user456"
		
		request := AddMemoryRequest{
			Messages: messages,
			CubeID:   &cubeID,
			UserID:   &userID,
		}
		
		assert.NotNil(t, request.Messages)
		assert.Nil(t, request.MemoryContent)
		assert.Nil(t, request.DocPath)
		assert.Equal(t, "cube123", *request.CubeID)
		assert.Equal(t, "user456", *request.UserID)
	})

	t.Run("AddMemoryRequest with Content", func(t *testing.T) {
		content := "This is memory content"
		
		request := AddMemoryRequest{
			MemoryContent: &content,
		}
		
		assert.Nil(t, request.Messages)
		assert.NotNil(t, request.MemoryContent)
		assert.Equal(t, "This is memory content", *request.MemoryContent)
	})

	t.Run("AddMemoryRequest with Document", func(t *testing.T) {
		docPath := "/path/to/document.pdf"
		
		request := AddMemoryRequest{
			DocPath: &docPath,
		}
		
		assert.Nil(t, request.Messages)
		assert.Nil(t, request.MemoryContent)
		assert.NotNil(t, request.DocPath)
		assert.Equal(t, "/path/to/document.pdf", *request.DocPath)
	})
}

func TestChatRequest(t *testing.T) {
	t.Run("ChatRequest Creation", func(t *testing.T) {
		userID := "user123"
		request := ChatRequest{
			Query:  "What is the weather today?",
			UserID: &userID,
		}
		
		assert.Equal(t, "What is the weather today?", request.Query)
		assert.Equal(t, "user123", *request.UserID)
	})

	t.Run("ChatRequest without UserID", func(t *testing.T) {
		request := ChatRequest{
			Query: "How are you?",
		}
		
		assert.Equal(t, "How are you?", request.Query)
		assert.Nil(t, request.UserID)
	})
}

func TestChatResponse(t *testing.T) {
	t.Run("ChatResponse Creation", func(t *testing.T) {
		now := time.Now()
		response := ChatResponse{
			Response:  "I'm doing well, thank you!",
			SessionID: "session123",
			UserID:    "user456",
			Timestamp: now,
		}
		
		assert.Equal(t, "I'm doing well, thank you!", response.Response)
		assert.Equal(t, "session123", response.SessionID)
		assert.Equal(t, "user456", response.UserID)
		assert.Equal(t, now, response.Timestamp)
	})
}

func TestBackendType(t *testing.T) {
	t.Run("BackendType Constants", func(t *testing.T) {
		assert.Equal(t, BackendType("openai"), BackendOpenAI)
		assert.Equal(t, BackendType("ollama"), BackendOllama)
		assert.Equal(t, BackendType("huggingface"), BackendHuggingFace)
		assert.Equal(t, BackendType("qdrant"), BackendQdrant)
		assert.Equal(t, BackendType("neo4j"), BackendNeo4j)
		assert.Equal(t, BackendType("redis"), BackendRedis)
	})
}

func TestMemoryBackend(t *testing.T) {
	t.Run("MemoryBackend Constants", func(t *testing.T) {
		assert.Equal(t, MemoryBackend("naive"), MemoryBackendNaive)
		assert.Equal(t, MemoryBackend("tree"), MemoryBackendTree)
		assert.Equal(t, MemoryBackend("tree_text"), MemoryBackendTreeText)
		assert.Equal(t, MemoryBackend("general"), MemoryBackendGeneral)
		assert.Equal(t, MemoryBackend("kv_cache"), MemoryBackendKVCache)
		assert.Equal(t, MemoryBackend("lora"), MemoryBackendLoRA)
	})
}

func TestEmbeddingVector(t *testing.T) {
	t.Run("EmbeddingVector Operations", func(t *testing.T) {
		vector := EmbeddingVector{0.1, 0.2, 0.3, 0.4}
		
		assert.Len(t, vector, 4)
		assert.Equal(t, float32(0.1), vector[0])
		assert.Equal(t, float32(0.4), vector[3])
	})

	t.Run("EmbeddingVector JSON Serialization", func(t *testing.T) {
		vector := EmbeddingVector{1.0, 2.0, 3.0}
		
		data, err := json.Marshal(vector)
		require.NoError(t, err)
		
		var decoded EmbeddingVector
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		
		assert.Equal(t, vector, decoded)
	})
}

func TestVectorSearchResult(t *testing.T) {
	t.Run("VectorSearchResult Creation", func(t *testing.T) {
		result := VectorSearchResult{
			ID:    "result123",
			Score: 0.95,
			Metadata: map[string]interface{}{
				"source": "document1",
				"page":   42,
			},
		}
		
		assert.Equal(t, "result123", result.ID)
		assert.Equal(t, float32(0.95), result.Score)
		assert.Equal(t, "document1", result.Metadata["source"])
		assert.Equal(t, 42, result.Metadata["page"])
	})
}

func TestTaskPriority(t *testing.T) {
	t.Run("TaskPriority Constants", func(t *testing.T) {
		assert.Equal(t, TaskPriority("low"), TaskPriorityLow)
		assert.Equal(t, TaskPriority("medium"), TaskPriorityMedium)
		assert.Equal(t, TaskPriority("high"), TaskPriorityHigh)
		assert.Equal(t, TaskPriority("critical"), TaskPriorityCritical)
	})
}

func TestScheduledTask(t *testing.T) {
	t.Run("ScheduledTask Creation", func(t *testing.T) {
		now := time.Now()
		payload := map[string]interface{}{
			"operation": "process_memory",
			"data":      "sample data",
		}
		
		task := ScheduledTask{
			ID:        "task123",
			UserID:    "user456",
			CubeID:    "cube789",
			TaskType:  "memory_processing",
			Priority:  TaskPriorityHigh,
			Payload:   payload,
			CreatedAt: now,
			UpdatedAt: now,
			Status:    "pending",
		}
		
		assert.Equal(t, "task123", task.ID)
		assert.Equal(t, "user456", task.UserID)
		assert.Equal(t, "cube789", task.CubeID)
		assert.Equal(t, "memory_processing", task.TaskType)
		assert.Equal(t, TaskPriorityHigh, task.Priority)
		assert.Equal(t, payload, task.Payload)
		assert.Equal(t, "pending", task.Status)
	})
}

func TestErrorType(t *testing.T) {
	t.Run("ErrorType Constants", func(t *testing.T) {
		assert.Equal(t, ErrorType("validation"), ErrorTypeValidation)
		assert.Equal(t, ErrorType("not_found"), ErrorTypeNotFound)
		assert.Equal(t, ErrorType("unauthorized"), ErrorTypeUnauthorized)
		assert.Equal(t, ErrorType("internal"), ErrorTypeInternal)
		assert.Equal(t, ErrorType("external"), ErrorTypeExternal)
	})
}

func TestMemGOSError(t *testing.T) {
	t.Run("NewMemGOSError", func(t *testing.T) {
		details := map[string]interface{}{
			"field": "username",
			"value": "",
		}
		
		err := NewMemGOSError(ErrorTypeValidation, "username is required", "MISSING_FIELD", details)
		
		assert.Equal(t, ErrorTypeValidation, err.Type)
		assert.Equal(t, "username is required", err.Message)
		assert.Equal(t, "MISSING_FIELD", err.Code)
		assert.Equal(t, details, err.Details)
	})

	t.Run("MemGOSError Error Method", func(t *testing.T) {
		err := NewMemGOSError(ErrorTypeInternal, "something went wrong", "INTERNAL_ERROR", nil)
		
		assert.Equal(t, "something went wrong", err.Error())
	})

	t.Run("MemGOSError JSON Serialization", func(t *testing.T) {
		err := NewMemGOSError(ErrorTypeNotFound, "resource not found", "NOT_FOUND", map[string]interface{}{
			"resource_id": "123",
		})
		
		data, errJson := json.Marshal(err)
		require.NoError(t, errJson)
		
		var decoded MemGOSError
		errJson = json.Unmarshal(data, &decoded)
		require.NoError(t, errJson)
		
		assert.Equal(t, err.Type, decoded.Type)
		assert.Equal(t, err.Message, decoded.Message)
		assert.Equal(t, err.Code, decoded.Code)
		assert.Equal(t, err.Details["resource_id"], decoded.Details["resource_id"])
	})
}

func TestContextKey(t *testing.T) {
	t.Run("ContextKey Constants", func(t *testing.T) {
		assert.Equal(t, ContextKey("user_id"), ContextKeyUserID)
		assert.Equal(t, ContextKey("session_id"), ContextKeySessionID)
		assert.Equal(t, ContextKey("request_id"), ContextKeyRequestID)
		assert.Equal(t, ContextKey("trace_id"), ContextKeyTraceID)
	})
}

func TestRequestContext(t *testing.T) {
	t.Run("NewRequestContext", func(t *testing.T) {
		reqCtx := NewRequestContext("user123", "session456")
		
		assert.Equal(t, "user123", reqCtx.UserID)
		assert.Equal(t, "session456", reqCtx.SessionID)
		assert.NotEmpty(t, reqCtx.RequestID)
		assert.NotEmpty(t, reqCtx.TraceID)
		
		// IDs should be different
		assert.NotEqual(t, reqCtx.RequestID, reqCtx.TraceID)
	})

	t.Run("GetRequestContext with values", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, ContextKeyUserID, "user789")
		ctx = context.WithValue(ctx, ContextKeySessionID, "session101")
		ctx = context.WithValue(ctx, ContextKeyRequestID, "req456")
		ctx = context.WithValue(ctx, ContextKeyTraceID, "trace789")
		
		reqCtx := GetRequestContext(ctx)
		
		assert.Equal(t, "user789", reqCtx.UserID)
		assert.Equal(t, "session101", reqCtx.SessionID)
		assert.Equal(t, "req456", reqCtx.RequestID)
		assert.Equal(t, "trace789", reqCtx.TraceID)
	})

	t.Run("GetRequestContext with empty context", func(t *testing.T) {
		ctx := context.Background()
		
		reqCtx := GetRequestContext(ctx)
		
		assert.Empty(t, reqCtx.UserID)
		assert.Empty(t, reqCtx.SessionID)
		assert.Empty(t, reqCtx.RequestID)
		assert.Empty(t, reqCtx.TraceID)
	})

	t.Run("GetRequestContext with wrong type values", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, ContextKeyUserID, 123) // Wrong type
		ctx = context.WithValue(ctx, ContextKeySessionID, "session456")
		
		reqCtx := GetRequestContext(ctx)
		
		assert.Empty(t, reqCtx.UserID) // Should be empty due to wrong type
		assert.Equal(t, "session456", reqCtx.SessionID)
	})
}

func TestgetStringFromContext(t *testing.T) {
	t.Run("Valid string value", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), ContextKeyUserID, "test_user")
		
		result := getStringFromContext(ctx, ContextKeyUserID)
		assert.Equal(t, "test_user", result)
	})

	t.Run("Non-string value", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), ContextKeyUserID, 123)
		
		result := getStringFromContext(ctx, ContextKeyUserID)
		assert.Empty(t, result)
	})

	t.Run("Missing value", func(t *testing.T) {
		ctx := context.Background()
		
		result := getStringFromContext(ctx, ContextKeyUserID)
		assert.Empty(t, result)
	})

	t.Run("Nil value", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), ContextKeyUserID, nil)
		
		result := getStringFromContext(ctx, ContextKeyUserID)
		assert.Empty(t, result)
	})
}

// Benchmark tests
func BenchmarkMessageDictMarshal(b *testing.B) {
	msg := MessageDict{
		Role:    MessageRoleUser,
		Content: "This is a benchmark test message",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(msg)
	}
}

func BenchmarkMessageListMarshal(b *testing.B) {
	messages := MessageList{
		{Role: MessageRoleUser, Content: "Message 1"},
		{Role: MessageRoleAssistant, Content: "Message 2"},
		{Role: MessageRoleUser, Content: "Message 3"},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(messages)
	}
}

func BenchmarkNewRequestContext(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewRequestContext("user123", "session456")
	}
}

func BenchmarkGetRequestContext(b *testing.B) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, ContextKeyUserID, "user123")
	ctx = context.WithValue(ctx, ContextKeySessionID, "session456")
	ctx = context.WithValue(ctx, ContextKeyRequestID, "req789")
	ctx = context.WithValue(ctx, ContextKeyTraceID, "trace101")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetRequestContext(ctx)
	}
}

// Integration tests
func TestTypeIntegration(t *testing.T) {
	t.Run("CompleteConversationFlow", func(t *testing.T) {
		// Create a complete conversation flow using all related types
		
		// 1. Create user
		user := User{
			UserID:    "user123",
			UserName:  "Test User",
			Role:      UserRoleUser,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			IsActive:  true,
		}
		
		// 2. Create memory cube
		cube := MemCube{
			ID:        "cube456",
			Name:      "Test Cube",
			OwnerID:   user.UserID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			IsLoaded:  true,
		}
		
		// 3. Create chat request
		chatReq := ChatRequest{
			Query:  "What is artificial intelligence?",
			UserID: &user.UserID,
		}
		
		// 4. Create chat history
		history := ChatHistory{
			UserID:        user.UserID,
			SessionID:     "session789",
			CreatedAt:     time.Now(),
			TotalMessages: 2,
			ChatHistory: MessageList{
				{Role: MessageRoleUser, Content: chatReq.Query},
				{Role: MessageRoleAssistant, Content: "AI is a field of computer science..."},
			},
		}
		
		// 5. Create memories
		textMem := &TextualMemoryItem{
			ID:        "mem001",
			Memory:    "AI is artificial intelligence",
			Metadata:  map[string]interface{}{"source": "conversation", "user_id": user.UserID},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		actMem := &ActivationMemoryItem{
			ID:        "act001",
			Memory:    map[string]interface{}{"hidden_states": []float32{0.1, 0.2, 0.3}},
			Metadata:  map[string]interface{}{"model": "transformer", "layer": 12},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		// 6. Create search result
		searchResult := MOSSearchResult{
			TextMem: []CubeMemoryResult{
				{CubeID: cube.ID, Memories: []MemoryItem{textMem}},
			},
			ActMem: []CubeMemoryResult{
				{CubeID: cube.ID, Memories: []MemoryItem{actMem}},
			},
			ParaMem: []CubeMemoryResult{},
		}
		
		// 7. Create request context
		reqCtx := NewRequestContext(user.UserID, history.SessionID)
		
		// Verify the complete flow
		assert.Equal(t, user.UserID, cube.OwnerID)
		assert.Equal(t, user.UserID, *chatReq.UserID)
		assert.Equal(t, user.UserID, history.UserID)
		assert.Equal(t, user.UserID, textMem.Metadata["user_id"])
		assert.Equal(t, cube.ID, searchResult.TextMem[0].CubeID)
		assert.Equal(t, user.UserID, reqCtx.UserID)
		assert.Equal(t, history.SessionID, reqCtx.SessionID)
		
		// Verify interface implementations
		var memItem MemoryItem = textMem
		assert.Equal(t, "mem001", memItem.GetID())
		assert.Equal(t, "AI is artificial intelligence", memItem.GetContent())
		
		var actMemItem MemoryItem = actMem
		assert.Equal(t, "act001", actMemItem.GetID())
		assert.Equal(t, "", actMemItem.GetContent()) // Activation memory has no string content
	})
}

// Example tests for documentation
func ExampleMessageDict() {
	msg := MessageDict{
		Role:    MessageRoleUser,
		Content: "Hello, how are you?",
	}
	
	data, _ := json.Marshal(msg)
	println(string(data))
	// Output will be JSON representation of the message
}

func ExampleNewRequestContext() {
	reqCtx := NewRequestContext("user123", "session456")
	
	println("User ID:", reqCtx.UserID)
	println("Session ID:", reqCtx.SessionID)
	println("Request ID:", reqCtx.RequestID)
	println("Trace ID:", reqCtx.TraceID)
}

func ExampleTextualMemoryItem_GetContent() {
	item := &TextualMemoryItem{
		ID:        "mem123",
		Memory:    "This is a sample memory",
		Metadata:  map[string]interface{}{"source": "chat"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	content := item.GetContent()
	println("Memory content:", content)
	// Output: Memory content: This is a sample memory
}