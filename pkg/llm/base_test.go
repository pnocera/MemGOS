package llm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/memtensor/memgos/pkg/types"
)

func TestBaseLLM(t *testing.T) {
	t.Run("InitialValues", func(t *testing.T) {
		base := NewBaseLLM("test-model")
		assert.Equal(t, "test-model", base.GetModelName())
		assert.Equal(t, 1024, base.GetMaxTokens())
		assert.Equal(t, 0.7, base.GetTemperature())
		assert.Equal(t, 0.9, base.GetTopP())
		assert.Equal(t, 30*time.Second, base.GetTimeout())
	})
	
	t.Run("SettersAndGetters", func(t *testing.T) {
		base := NewBaseLLM("test-model")
		base.SetMaxTokens(2048)
		assert.Equal(t, 2048, base.GetMaxTokens())
		
		base.SetTemperature(0.5)
		assert.Equal(t, 0.5, base.GetTemperature())
		
		base.SetTopP(0.8)
		assert.Equal(t, 0.8, base.GetTopP())
		
		timeout := 60 * time.Second
		base.SetTimeout(timeout)
		assert.Equal(t, timeout, base.GetTimeout())
	})
	
	t.Run("GetModelInfo", func(t *testing.T) {
		base := NewBaseLLM("test-model")
		info := base.GetModelInfo()
		assert.Equal(t, "test-model", info["model"])
		assert.Equal(t, 1024, info["max_tokens"])
		assert.Equal(t, 0.7, info["temperature"])
		assert.Equal(t, 0.9, info["top_p"])
		assert.NotNil(t, info["metrics"])
	})
	
	t.Run("ValidateMessages", func(t *testing.T) {
		// Valid messages
		messages := types.MessageList{
			{Role: types.MessageRoleUser, Content: "Hello"},
			{Role: types.MessageRoleAssistant, Content: "Hi there"},
		}
		err := base.ValidateMessages(messages)
		assert.NoError(t, err)
		
		// Empty message list
		err = base.ValidateMessages(types.MessageList{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty message list")
		
		// Missing role
		messages = types.MessageList{
			{Content: "Hello"},
		}
		err = base.ValidateMessages(messages)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role is required")
		
		// Missing content
		messages = types.MessageList{
			{Role: types.MessageRoleUser},
		}
		err = base.ValidateMessages(messages)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "content is required")
		
		// Invalid role
		messages = types.MessageList{
			{Role: "invalid", Content: "Hello"},
		}
		err = base.ValidateMessages(messages)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid role")
	})
	
	t.Run("FormatMessages", func(t *testing.T) {
		messages := types.MessageList{
			{Role: types.MessageRoleUser, Content: "Hello"},
			{Role: types.MessageRoleAssistant, Content: "Hi there"},
		}
		
		formatted := base.FormatMessages(messages)
		assert.Len(t, formatted, 2)
		assert.Equal(t, "user", formatted[0]["role"])
		assert.Equal(t, "Hello", formatted[0]["content"])
		assert.Equal(t, "assistant", formatted[1]["role"])
		assert.Equal(t, "Hi there", formatted[1]["content"])
	})
	
	t.Run("ExtractContent", func(t *testing.T) {
		// String response
		content, err := base.ExtractContent("Hello world")
		assert.NoError(t, err)
		assert.Equal(t, "Hello world", content)
		
		// Map with content field
		response := map[string]interface{}{
			"content": "Hello world",
		}
		content, err = base.ExtractContent(response)
		assert.NoError(t, err)
		assert.Equal(t, "Hello world", content)
		
		// Map with message.content field
		response = map[string]interface{}{
			"message": map[string]interface{}{
				"content": "Hello world",
			},
		}
		content, err = base.ExtractContent(response)
		assert.NoError(t, err)
		assert.Equal(t, "Hello world", content)
		
		// OpenAI-style choices response
		response = map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{
						"content": "Hello world",
					},
				},
			},
		}
		content, err = base.ExtractContent(response)
		assert.NoError(t, err)
		assert.Equal(t, "Hello world", content)
		
		// No content found
		response = map[string]interface{}{
			"other": "data",
		}
		content, err = base.ExtractContent(response)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no content found")
		
		// Unsupported type
		content, err = base.ExtractContent(123)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported response type")
	})
	
	t.Run("RecordMetrics", func(t *testing.T) {
		base.RecordMetrics("test_metric", 123)
		metrics := base.GetMetrics()
		assert.Equal(t, 123, metrics["test_metric"])
		
		base.RecordMetrics("another_metric", "test_value")
		metrics = base.GetMetrics()
		assert.Equal(t, "test_value", metrics["another_metric"])
	})
	
	t.Run("TokenCount", func(t *testing.T) {
		text := "Hello world"
		count := base.TokenCount(text)
		expected := len(text) / 4 // Simple heuristic
		assert.Equal(t, expected, count)
		
		// Empty text
		count = base.TokenCount("")
		assert.Equal(t, 0, count)
	})
	
	t.Run("TruncateToTokens", func(t *testing.T) {
		text := "Hello world this is a test"
		
		// Truncate to 5 tokens (20 chars)
		truncated := base.TruncateToTokens(text, 5)
		assert.Equal(t, "Hello world this is ", truncated)
		
		// Text shorter than limit
		truncated = base.TruncateToTokens("Hi", 10)
		assert.Equal(t, "Hi", truncated)
		
		// Zero tokens
		truncated = base.TruncateToTokens(text, 0)
		assert.Equal(t, "", truncated)
		
		// Negative tokens
		truncated = base.TruncateToTokens(text, -1)
		assert.Equal(t, "", truncated)
	})
	
	t.Run("GenerateRequestID", func(t *testing.T) {
		id1 := base.GenerateRequestID()
		id2 := base.GenerateRequestID()
		
		assert.NotEmpty(t, id1)
		assert.NotEmpty(t, id2)
		assert.NotEqual(t, id1, id2)
		assert.Contains(t, id1, "req_")
		assert.Contains(t, id2, "req_")
	})
	
	t.Run("ParseJSONResponse", func(t *testing.T) {
		jsonData := []byte(`{"key": "value", "number": 123}`)
		
		response, err := base.ParseJSONResponse(jsonData)
		assert.NoError(t, err)
		assert.Equal(t, "value", response["key"])
		assert.Equal(t, float64(123), response["number"])
		
		// Invalid JSON
		invalidJSON := []byte(`{"invalid": json}`)
		response, err = base.ParseJSONResponse(invalidJSON)
		assert.Error(t, err)
		assert.Nil(t, response)
	})
	
	t.Run("BuildPrompt", func(t *testing.T) {
		messages := types.MessageList{
			{Role: types.MessageRoleSystem, Content: "You are a helpful assistant"},
			{Role: types.MessageRoleUser, Content: "Hello"},
			{Role: types.MessageRoleAssistant, Content: "Hi there"},
		}
		
		prompt := base.BuildPrompt(messages)
		assert.Contains(t, prompt, "System: You are a helpful assistant")
		assert.Contains(t, prompt, "User: Hello")
		assert.Contains(t, prompt, "Assistant: Hi there")
	})
	
	t.Run("Close", func(t *testing.T) {
		err := base.Close()
		assert.NoError(t, err)
	})
}

func TestStreamingLLM(t *testing.T) {
	streaming := NewStreamingLLM("test-model")
	
	t.Run("InitialState", func(t *testing.T) {
		assert.False(t, streaming.IsStreaming())
		assert.NotNil(t, streaming.streamBuffer)
	})
	
	t.Run("StartStopStreaming", func(t *testing.T) {
		streaming.StartStreaming()
		assert.True(t, streaming.IsStreaming())
		
		streaming.StopStreaming()
		assert.False(t, streaming.IsStreaming())
	})
	
	t.Run("WriteReadStream", func(t *testing.T) {
		streaming.StartStreaming()
		
		streaming.WriteToStream("Hello")
		streaming.WriteToStream("World")
		
		streamChan := streaming.ReadFromStream()
		
		content1 := <-streamChan
		assert.Equal(t, "Hello", content1)
		
		content2 := <-streamChan
		assert.Equal(t, "World", content2)
		
		streaming.StopStreaming()
	})
	
	t.Run("WriteToInactiveStream", func(t *testing.T) {
		// Should not panic when writing to inactive stream
		streaming.WriteToStream("Test")
	})
	
	t.Run("Close", func(t *testing.T) {
		err := streaming.Close()
		assert.NoError(t, err)
		assert.False(t, streaming.IsStreaming())
	})
}

func TestLLMError(t *testing.T) {
	err := NewLLMError("test_code", "test message", "test_type")
	
	assert.Equal(t, "test_code", err.Code)
	assert.Equal(t, "test message", err.Message)
	assert.Equal(t, "test_type", err.Type)
	assert.Equal(t, "LLM Error [test_code]: test message", err.Error())
}

func TestLLMResponse(t *testing.T) {
	response := &LLMResponse{
		Content:      "Hello world",
		TokensUsed:   10,
		Model:        "test-model",
		FinishReason: "stop",
		Metadata:     map[string]interface{}{"key": "value"},
		RequestID:    "req_123",
	}
	
	assert.Equal(t, "Hello world", response.Content)
	assert.Equal(t, 10, response.TokensUsed)
	assert.Equal(t, "test-model", response.Model)
	assert.Equal(t, "stop", response.FinishReason)
	assert.Equal(t, "value", response.Metadata["key"])
	assert.Equal(t, "req_123", response.RequestID)
}

func TestLLMConfig(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		config := DefaultLLMConfig()
		assert.Equal(t, "openai", config.Provider)
		assert.Equal(t, "gpt-3.5-turbo", config.Model)
		assert.Equal(t, 1024, config.MaxTokens)
		assert.Equal(t, 0.7, config.Temperature)
		assert.Equal(t, 0.9, config.TopP)
		assert.Equal(t, 30*time.Second, config.Timeout)
		assert.NotNil(t, config.Extra)
	})
	
	t.Run("ConfigValidation", func(t *testing.T) {
		config := &LLMConfig{
			Provider:    "openai",
			Model:       "gpt-3.5-turbo",
			MaxTokens:   1024,
			Temperature: 0.7,
			TopP:        0.9,
		}
		
		err := config.Validate()
		assert.NoError(t, err)
		
		// Test validation errors
		config.Provider = ""
		err = config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "provider is required")
		
		config.Provider = "openai"
		config.Model = ""
		err = config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "model is required")
		
		config.Model = "gpt-3.5-turbo"
		config.MaxTokens = -1
		err = config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max_tokens must be non-negative")
		
		config.MaxTokens = 1024
		config.Temperature = -1
		err = config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "temperature must be between")
		
		config.Temperature = 3
		err = config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "temperature must be between")
		
		config.Temperature = 0.7
		config.TopP = -1
		err = config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "top_p must be between")
		
		config.TopP = 1.5
		err = config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "top_p must be between")
	})
}