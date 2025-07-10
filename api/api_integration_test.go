package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/errors"
	"github.com/memtensor/memgos/pkg/types"
)

// TestAPIIntegration provides comprehensive integration tests for the MemGOS API
func TestAPIIntegration(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	t.Run("ComprehensiveAPIWorkflow", func(t *testing.T) {
		server, mockCore, mockLogger := setupTestServer()
		
		// Setup mock expectations for full workflow
		setupMockExpectationsForWorkflow(mockCore, mockLogger)

		// 1. Health Check
		w := performRequest(server.router, "GET", "/health", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		// 2. Create User
		userReq := UserCreate{
			UserName: stringPtr("john_doe"),
			Role:     types.UserRoleUser,
			UserID:   "user123",
		}
		w = performRequest(server.router, "POST", "/users", userReq)
		assert.Equal(t, http.StatusOK, w.Code)

		// 3. Register Memory Cube
		cubeReq := MemCubeRegister{
			MemCubeNameOrPath: "/path/to/test-cube",
			MemCubeID:         stringPtr("test-cube-123"),
		}
		w = performRequest(server.router, "POST", "/mem_cubes", cubeReq)
		assert.Equal(t, http.StatusOK, w.Code)

		// 4. Add Memory
		memoryReq := MemoryCreate{
			MemoryContent: stringPtr("This is a test memory for integration testing"),
			MemCubeID:     stringPtr("test-cube-123"),
		}
		w = performRequest(server.router, "POST", "/memories", memoryReq)
		assert.Equal(t, http.StatusOK, w.Code)

		// 5. Search Memories
		searchReq := SearchRequest{
			Query: "integration testing",
			TopK:  intPtr(5),
		}
		w = performRequest(server.router, "POST", "/search", searchReq)
		assert.Equal(t, http.StatusOK, w.Code)

		// 6. Chat
		chatReq := ChatRequest{
			Query: "Tell me about integration testing",
		}
		w = performRequest(server.router, "POST", "/chat", chatReq)
		assert.Equal(t, http.StatusOK, w.Code)

		// 7. Get User Info
		w = performRequest(server.router, "GET", "/users/me", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		// 8. List Users
		w = performRequest(server.router, "GET", "/users", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		// 9. Get Metrics
		w = performRequest(server.router, "GET", "/metrics", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		// 10. OpenAPI Spec
		w = performRequest(server.router, "GET", "/openapi.json", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify all expectations
		mockCore.AssertExpectations(t)
	})
}

// TestAuthenticationWorkflows tests authentication and session management
func TestAuthenticationWorkflows(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("LoginLogoutWorkflow", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// 1. Login
		loginReq := AuthRequest{
			UserID:   "test-user",
			Password: "test-password",
		}
		w := performRequest(server.router, "POST", "/auth/login", loginReq)
		assert.Equal(t, http.StatusOK, w.Code)

		var loginResponse BaseResponse[AuthResponse]
		err := json.Unmarshal(w.Body.Bytes(), &loginResponse)
		require.NoError(t, err)
		
		assert.Equal(t, http.StatusOK, loginResponse.Code)
		assert.Equal(t, "Login successful", loginResponse.Message)
		assert.NotNil(t, loginResponse.Data)
		assert.NotEmpty(t, loginResponse.Data.Token)
		assert.NotEmpty(t, loginResponse.Data.SessionID)

		token := loginResponse.Data.Token
		sessionID := loginResponse.Data.SessionID

		// 2. Access protected endpoint with token
		req, _ := http.NewRequest("GET", "/users/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 3. Refresh token
		req, _ = http.NewRequest("POST", "/auth/refresh", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var refreshResponse BaseResponse[AuthResponse]
		err = json.Unmarshal(w.Body.Bytes(), &refreshResponse)
		require.NoError(t, err)
		assert.NotEmpty(t, refreshResponse.Data.Token)

		// 4. Logout
		req, _ = http.NewRequest("POST", "/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify session was invalidated
		session, err := server.authManager.GetSession(sessionID)
		assert.Error(t, err)
		assert.Nil(t, session)
	})

	t.Run("UnauthorizedAccess", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// Try to access protected endpoint without token
		w := performRequest(server.router, "POST", "/memories", MemoryCreate{
			MemoryContent: stringPtr("unauthorized test"),
		})

		// Should still work with fallback auth (using default user_id from config)
		assert.Equal(t, http.StatusBadRequest, w.Code) // Bad request due to missing fields, not auth

		// Try with invalid token
		req, _ := http.NewRequest("POST", "/memories", bytes.NewBuffer([]byte(`{"memory_content":"test"}`)))
		req.Header.Set("Authorization", "Bearer invalid-token")
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		// Should fallback to header auth
		assert.Equal(t, http.StatusBadRequest, w.Code) // Bad request due to missing fields, not auth
	})
}

// TestErrorHandling tests comprehensive error scenarios
func TestErrorHandling(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("ValidationErrors", func(t *testing.T) {
		server, _, _ := setupTestServer()

		testCases := []struct {
			name           string
			method         string
			path           string
			body           interface{}
			expectedStatus int
			expectedError  string
		}{
			{
				name:           "EmptySearchQuery",
				method:         "POST",
				path:           "/search",
				body:           SearchRequest{},
				expectedStatus: http.StatusBadRequest,
				expectedError:  "query is required",
			},
			{
				name:           "EmptyChatQuery",
				method:         "POST",
				path:           "/chat",
				body:           ChatRequest{},
				expectedStatus: http.StatusBadRequest,
				expectedError:  "query is required",
			},
			{
				name:           "EmptyUserCreation",
				method:         "POST",
				path:           "/users",
				body:           UserCreate{},
				expectedStatus: http.StatusBadRequest,
				expectedError:  "user_id is required",
			},
			{
				name:           "EmptyMemCubeRegistration",
				method:         "POST",
				path:           "/mem_cubes",
				body:           MemCubeRegister{},
				expectedStatus: http.StatusBadRequest,
				expectedError:  "mem_cube_name_or_path is required",
			},
			{
				name:           "InvalidMemoryContent",
				method:         "POST",
				path:           "/memories",
				body:           MemoryCreate{},
				expectedStatus: http.StatusBadRequest,
				expectedError:  "at least one of",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				w := performRequest(server.router, tc.method, tc.path, tc.body)
				assert.Equal(t, tc.expectedStatus, w.Code)

				var errorResponse ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
				require.NoError(t, err)
				assert.Contains(t, strings.ToLower(errorResponse.Message), strings.ToLower(tc.expectedError))
			})
		}
	})

	t.Run("CoreServiceErrors", func(t *testing.T) {
		server, mockCore, _ := setupTestServer()

		// Setup mock to return errors
		mockCore.On("Add", mock.Anything, mock.AnythingOfType("*types.AddMemoryRequest")).Return(
			errors.NewValidationError("cube not found"))
		mockCore.On("Search", mock.Anything, mock.AnythingOfType("*types.SearchQuery")).Return(
			nil, errors.NewInternalError("search service unavailable"))
		mockCore.On("Chat", mock.Anything, mock.AnythingOfType("*types.ChatRequest")).Return(
			nil, errors.NewResourceNotFoundError("LLM service not available"))

		// Test memory addition error
		memoryReq := MemoryCreate{
			MemoryContent: stringPtr("test content"),
			MemCubeID:     stringPtr("nonexistent-cube"),
		}
		w := performRequest(server.router, "POST", "/memories", memoryReq)
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		// Test search error
		searchReq := SearchRequest{
			Query: "test query",
		}
		w = performRequest(server.router, "POST", "/search", searchReq)
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		// Test chat error
		chatReq := ChatRequest{
			Query: "test query",
		}
		w = performRequest(server.router, "POST", "/chat", chatReq)
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		mockCore.AssertExpectations(t)
	})
}

// TestMiddleware tests middleware functionality
func TestMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("CORSMiddleware", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// Test CORS preflight request
		req, _ := http.NewRequest("OPTIONS", "/health", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type")
		
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Origin"), "*")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
	})

	t.Run("RequestIDMiddleware", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		w := performRequest(server.router, "GET", "/health", nil)
		assert.Equal(t, http.StatusOK, w.Code)
		
		// Request ID should be added to response headers
		requestID := w.Header().Get("X-Request-ID")
		assert.NotEmpty(t, requestID)
	})

	t.Run("MetricsMiddleware", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// Make several requests
		for i := 0; i < 5; i++ {
			w := performRequest(server.router, "GET", "/health", nil)
			assert.Equal(t, http.StatusOK, w.Code)
		}

		// Check metrics endpoint
		w := performRequest(server.router, "GET", "/metrics", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		var metricsResponse MetricsResponse
		err := json.Unmarshal(w.Body.Bytes(), &metricsResponse)
		require.NoError(t, err)
		
		assert.NotNil(t, metricsResponse.Metrics)
		assert.Contains(t, metricsResponse.Metrics, "requests_total")
	})
}

// TestMemoryOperations tests all memory-related operations
func TestMemoryOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("CompleteMemoryLifecycle", func(t *testing.T) {
		server, mockCore, _ := setupTestServer()

		// Setup mock expectations
		mockCore.On("Add", mock.Anything, mock.AnythingOfType("*types.AddMemoryRequest")).Return(nil)
		mockCore.On("GetAll", mock.Anything, mock.AnythingOfType("*types.GetAllMemoriesRequest")).Return(
			map[string]interface{}{
				"memories": []map[string]interface{}{
					{"id": "mem1", "content": "test memory 1"},
					{"id": "mem2", "content": "test memory 2"},
				},
			}, nil)
		mockCore.On("Get", mock.Anything, mock.AnythingOfType("*types.GetMemoryRequest")).Return(
			map[string]interface{}{
				"id":      "mem1",
				"content": "test memory 1",
				"cube_id": "cube123",
			}, nil)
		mockCore.On("Update", mock.Anything, mock.AnythingOfType("*types.UpdateMemoryRequest")).Return(nil)
		mockCore.On("Delete", mock.Anything, mock.AnythingOfType("*types.DeleteMemoryRequest")).Return(nil)
		mockCore.On("DeleteAll", mock.Anything, mock.AnythingOfType("*types.DeleteAllMemoriesRequest")).Return(nil)

		// 1. Add memory
		memoryReq := MemoryCreate{
			MemoryContent: stringPtr("test memory content"),
			MemCubeID:     stringPtr("cube123"),
		}
		w := performRequest(server.router, "POST", "/memories", memoryReq)
		assert.Equal(t, http.StatusOK, w.Code)

		// 2. Get all memories
		w = performRequest(server.router, "GET", "/memories", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		// 3. Get specific memory
		w = performRequest(server.router, "GET", "/memories/cube123/mem1", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		// 4. Update memory
		updateReq := map[string]interface{}{
			"content": "updated memory content",
		}
		w = performRequest(server.router, "PUT", "/memories/cube123/mem1", updateReq)
		assert.Equal(t, http.StatusOK, w.Code)

		// 5. Delete specific memory
		w = performRequest(server.router, "DELETE", "/memories/cube123/mem1", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		// 6. Delete all memories in cube
		w = performRequest(server.router, "DELETE", "/memories/cube123", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		mockCore.AssertExpectations(t)
	})
}

// TestCubeManagement tests memory cube operations
func TestCubeManagement(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("CubeOperations", func(t *testing.T) {
		server, mockCore, _ := setupTestServer()

		// Setup mock expectations
		mockCore.On("RegisterMemCube", mock.Anything, "/path/to/cube", "cube123", "test-user").Return(nil)
		mockCore.On("UnregisterMemCube", mock.Anything, "cube123", "test-user").Return(nil)
		mockCore.On("ShareCubeWithUser", mock.Anything, "cube123", "target-user").Return(true, nil)

		// 1. Register cube
		cubeReq := MemCubeRegister{
			MemCubeNameOrPath: "/path/to/cube",
			MemCubeID:         stringPtr("cube123"),
		}
		w := performRequest(server.router, "POST", "/mem_cubes", cubeReq)
		assert.Equal(t, http.StatusOK, w.Code)

		// 2. Share cube
		shareReq := CubeShare{
			TargetUserID: "target-user",
		}
		w = performRequest(server.router, "POST", "/mem_cubes/cube123/share", shareReq)
		assert.Equal(t, http.StatusOK, w.Code)

		// 3. Unregister cube
		w = performRequest(server.router, "DELETE", "/mem_cubes/cube123", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		mockCore.AssertExpectations(t)
	})
}

// TestConfigurationEndpoint tests configuration management
func TestConfigurationEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("SetConfiguration", func(t *testing.T) {
		server, _, _ := setupTestServer()

		configReq := ConfigRequest{
			UserID:                 stringPtr("config-user"),
			EnableTextualMemory:    boolPtr(true),
			EnableActivationMemory: boolPtr(false),
			TopK:                   intPtr(10),
			ChatModel: &LLMConfigRequest{
				Backend: "openai",
				Config: map[string]interface{}{
					"model":       "gpt-4",
					"temperature": 0.7,
				},
			},
		}

		w := performRequest(server.router, "POST", "/configure", configReq)
		assert.Equal(t, http.StatusOK, w.Code)

		var response SimpleResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Configuration updated successfully", response.Message)
	})
}

// TestPerformanceAndLoad tests API performance
func TestPerformanceAndLoad(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("ConcurrentRequests", func(t *testing.T) {
		server, mockCore, _ := setupTestServer()

		// Setup mock expectations for concurrent requests
		mockCore.On("Search", mock.Anything, mock.AnythingOfType("*types.SearchQuery")).Return(
			&types.SearchResult{
				TextMem:  []map[string]interface{}{},
				ActMem:   []map[string]interface{}{},
				ParaMem:  []map[string]interface{}{},
			}, nil).Maybe()

		// Simulate concurrent requests
		numRequests := 10
		done := make(chan bool, numRequests)

		for i := 0; i < numRequests; i++ {
			go func(id int) {
				searchReq := SearchRequest{
					Query: fmt.Sprintf("concurrent query %d", id),
					TopK:  intPtr(5),
				}
				w := performRequest(server.router, "POST", "/search", searchReq)
				assert.Equal(t, http.StatusOK, w.Code)
				done <- true
			}(i)
		}

		// Wait for all requests to complete
		for i := 0; i < numRequests; i++ {
			<-done
		}
	})
}

// Helper functions for enhanced testing

func setupEnhancedTestServer() (*EnhancedServer, *MockMOSCore, *MockLogger) {
	mockCore := &MockMOSCore{}
	mockLogger := &MockLogger{}

	cfg := &config.MOSConfig{
		UserID:      "test-user",
		SessionID:   "test-session",
		APIPort:     8080,
		Environment: "test",
	}

	server := NewEnhancedServer(mockCore, cfg, mockLogger)
	return server, mockCore, mockLogger
}

func setupMockExpectationsForWorkflow(mockCore *MockMOSCore, mockLogger *MockLogger) {
	// Logger expectations
	mockLogger.On("Info", mock.AnythingOfType("string"), mock.Anything).Maybe()
	mockLogger.On("Debug", mock.AnythingOfType("string"), mock.Anything).Maybe()
	mockLogger.On("Error", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Maybe()

	// Core service expectations
	mockCore.On("CreateUser", mock.Anything, "john_doe", types.UserRoleUser, "user123").Return("user123", nil)
	mockCore.On("RegisterMemCube", mock.Anything, "/path/to/test-cube", "test-cube-123", "test-user").Return(nil)
	mockCore.On("Add", mock.Anything, mock.AnythingOfType("*types.AddMemoryRequest")).Return(nil)
	mockCore.On("Search", mock.Anything, mock.AnythingOfType("*types.SearchQuery")).Return(
		&types.SearchResult{
			TextMem: []map[string]interface{}{
				{"id": "mem1", "content": "integration testing memory"},
			},
			ActMem:  []map[string]interface{}{},
			ParaMem: []map[string]interface{}{},
		}, nil)
	mockCore.On("Chat", mock.Anything, mock.AnythingOfType("*types.ChatRequest")).Return(
		&types.ChatResponse{
			Response: "Integration testing is important for ensuring system components work together correctly.",
		}, nil)
	mockCore.On("GetUserInfo", mock.Anything, "test-user").Return(
		map[string]interface{}{
			"user_id": "test-user",
			"name":    "Test User",
			"role":    "user",
		}, nil)
	mockCore.On("ListUsers", mock.Anything).Return([]types.User{
		{
			ID:        "test-user",
			Name:      "Test User",
			Role:      types.UserRoleUser,
			CreatedAt: time.Now(),
		},
	}, nil)
}

func setupMockLogger(mockLogger *MockLogger) {
	mockLogger.On("Info", mock.AnythingOfType("string"), mock.Anything).Maybe()
	mockLogger.On("Debug", mock.AnythingOfType("string"), mock.Anything).Maybe()
	mockLogger.On("Error", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Warn", mock.AnythingOfType("string"), mock.Anything).Maybe()
}

func boolPtr(b bool) *bool {
	return &b
}

// Benchmark tests for API performance

func BenchmarkHealthCheckEndpoint(b *testing.B) {
	gin.SetMode(gin.TestMode)
	server, _, _ := setupTestServer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := performRequest(server.router, "GET", "/health", nil)
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

func BenchmarkSearchEndpoint(b *testing.B) {
	gin.SetMode(gin.TestMode)
	server, mockCore, _ := setupTestServer()
	
	mockCore.On("Search", mock.Anything, mock.AnythingOfType("*types.SearchQuery")).Return(
		&types.SearchResult{
			TextMem:  []map[string]interface{}{{"id": "mem1", "content": "test"}},
			ActMem:   []map[string]interface{}{},
			ParaMem:  []map[string]interface{}{},
		}, nil)

	searchReq := SearchRequest{
		Query: "benchmark test query",
		TopK:  intPtr(5),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := performRequest(server.router, "POST", "/search", searchReq)
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

func BenchmarkAuthenticationWorkflow(b *testing.B) {
	gin.SetMode(gin.TestMode)
	server, _, mockLogger := setupEnhancedTestServer()
	setupMockLogger(mockLogger)

	loginReq := AuthRequest{
		UserID: "benchmark-user",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := performRequest(server.router, "POST", "/auth/login", loginReq)
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}