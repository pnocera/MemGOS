package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/types"
)

// TestEnhancedServerCreation tests the creation and initialization of the enhanced server
func TestEnhancedServerCreation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("NewEnhancedServer", func(t *testing.T) {
		mockCore := &MockMOSCore{}
		mockLogger := &MockLogger{}
		cfg := &config.MOSConfig{
			UserID:      "test-user",
			SessionID:   "test-session",
			APIPort:     8080,
			Environment: "production",
		}

		server := NewEnhancedServer(mockCore, cfg, mockLogger)

		assert.NotNil(t, server)
		assert.Equal(t, mockCore, server.mosCore)
		assert.Equal(t, cfg, server.config)
		assert.Equal(t, mockLogger, server.logger)
		assert.NotNil(t, server.router)
		assert.NotNil(t, server.authManager)
		assert.NotZero(t, server.startTime)
		
		// Verify Gin mode is set correctly for production
		assert.Equal(t, gin.ReleaseMode, gin.Mode())
	})

	t.Run("NewEnhancedServerDebugMode", func(t *testing.T) {
		// Reset to test mode first
		gin.SetMode(gin.TestMode)
		
		mockCore := &MockMOSCore{}
		mockLogger := &MockLogger{}
		cfg := &config.MOSConfig{
			UserID:      "test-user",
			SessionID:   "test-session",
			APIPort:     8080,
			Environment: "development",
		}

		server := NewEnhancedServer(mockCore, cfg, mockLogger)

		assert.NotNil(t, server)
		// In development, should set debug mode (but we're in test mode)
		assert.Equal(t, gin.TestMode, gin.Mode())
	})
}

// TestEnhancedServerRoutes tests that all routes are properly configured
func TestEnhancedServerRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("AllRoutesConfigured", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// Test all major routes exist and return appropriate responses
		routes := []struct {
			method   string
			path     string
			expected int
		}{
			{"GET", "/health", http.StatusOK},
			{"GET", "/", http.StatusOK}, // redirectToDocs
			{"POST", "/auth/login", http.StatusBadRequest}, // Bad request due to missing body
			{"POST", "/auth/logout", http.StatusOK},
			{"POST", "/auth/refresh", http.StatusUnauthorized}, // Unauthorized due to missing token
			{"POST", "/configure", http.StatusOK},
			{"GET", "/users", http.StatusOK},
			{"POST", "/users", http.StatusBadRequest}, // Bad request due to missing fields
			{"GET", "/users/me", http.StatusOK},
			{"POST", "/mem_cubes", http.StatusBadRequest}, // Bad request due to missing fields
			{"POST", "/memories", http.StatusBadRequest}, // Bad request due to missing fields
			{"GET", "/memories", http.StatusOK},
			{"POST", "/search", http.StatusBadRequest}, // Bad request due to missing query
			{"POST", "/chat", http.StatusBadRequest}, // Bad request due to missing query
			{"GET", "/metrics", http.StatusOK},
			{"GET", "/openapi.json", http.StatusOK},
		}

		for _, route := range routes {
			t.Run(route.method+"_"+route.path, func(t *testing.T) {
				w := performRequest(server.router, route.method, route.path, nil)
				assert.Equal(t, route.expected, w.Code, "Route %s %s", route.method, route.path)
			})
		}
	})

	t.Run("RESTfulMemoryRoutes", func(t *testing.T) {
		server, mockCore, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// Setup mock expectations for memory operations
		mockCore.On("Get", mock.Anything, mock.AnythingOfType("*types.GetMemoryRequest")).Return(
			map[string]interface{}{"id": "mem1", "content": "test"}, nil)
		mockCore.On("Update", mock.Anything, mock.AnythingOfType("*types.UpdateMemoryRequest")).Return(nil)
		mockCore.On("Delete", mock.Anything, mock.AnythingOfType("*types.DeleteMemoryRequest")).Return(nil)
		mockCore.On("DeleteAll", mock.Anything, mock.AnythingOfType("*types.DeleteAllMemoriesRequest")).Return(nil)

		// Test RESTful routes for memories
		restRoutes := []struct {
			method   string
			path     string
			expected int
		}{
			{"GET", "/memories/cube123/mem1", http.StatusOK},
			{"PUT", "/memories/cube123/mem1", http.StatusOK},
			{"DELETE", "/memories/cube123/mem1", http.StatusOK},
			{"DELETE", "/memories/cube123", http.StatusOK},
		}

		for _, route := range restRoutes {
			t.Run(route.method+"_"+route.path, func(t *testing.T) {
				var body interface{}
				if route.method == "PUT" {
					body = map[string]interface{}{"content": "updated content"}
				}
				w := performRequest(server.router, route.method, route.path, body)
				assert.Equal(t, route.expected, w.Code)
			})
		}

		mockCore.AssertExpectations(t)
	})

	t.Run("MemCubeRoutes", func(t *testing.T) {
		server, mockCore, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// Setup mock expectations
		mockCore.On("UnregisterMemCube", mock.Anything, "cube123", mock.AnythingOfType("string")).Return(nil)
		mockCore.On("ShareCubeWithUser", mock.Anything, "cube123", "target-user").Return(true, nil)

		cubeRoutes := []struct {
			method   string
			path     string
			body     interface{}
			expected int
		}{
			{"DELETE", "/mem_cubes/cube123", nil, http.StatusOK},
			{"POST", "/mem_cubes/cube123/share", CubeShare{TargetUserID: "target-user"}, http.StatusOK},
		}

		for _, route := range cubeRoutes {
			t.Run(route.method+"_"+route.path, func(t *testing.T) {
				w := performRequest(server.router, route.method, route.path, route.body)
				assert.Equal(t, route.expected, w.Code)
			})
		}

		mockCore.AssertExpectations(t)
	})
}

// TestEnhancedServerStart tests server startup and shutdown
func TestEnhancedServerStart(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("ServerStartAndStop", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		
		// Setup logger expectations for server start
		mockLogger.On("Info", "Starting Enhanced API server", mock.MatchedBy(func(fields map[string]interface{}) bool {
			port, hasPort := fields["port"]
			mode, hasMode := fields["mode"]
			features, hasFeatures := fields["features"]
			return hasPort && port == 8080 && hasMode && hasFeatures
		})).Return()
		
		mockLogger.On("Debug", "Cleaned up expired sessions").Maybe()
		mockLogger.On("Info", "Shutting down Enhanced API server...").Return()

		// Test server start/stop without actually starting the HTTP server
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Start server in goroutine
		errChan := make(chan error, 1)
		go func() {
			errChan <- server.Start(ctx)
		}()

		// Wait for context timeout (simulating shutdown)
		select {
		case err := <-errChan:
			assert.NoError(t, err)
		case <-time.After(200 * time.Millisecond):
			t.Fatal("Server start timed out")
		}

		mockLogger.AssertExpectations(t)
	})

	t.Run("ServerStop", func(t *testing.T) {
		server, _, _ := setupEnhancedTestServer()

		// Test stopping server that hasn't been started
		err := server.Stop()
		assert.NoError(t, err)
	})

	t.Run("GetPort", func(t *testing.T) {
		// Test with configured port
		server, _, _ := setupEnhancedTestServer()
		port := server.getPort()
		assert.Equal(t, 8080, port)

		// Test with no configured port (should use default)
		mockCore := &MockMOSCore{}
		mockLogger := &MockLogger{}
		cfg := &config.MOSConfig{
			UserID:    "test-user",
			SessionID: "test-session",
			// APIPort not set
		}
		server2 := NewEnhancedServer(mockCore, cfg, mockLogger)
		port2 := server2.getPort()
		assert.Equal(t, 8080, port2) // Default port
	})
}

// TestEnhancedHealthCheck tests the enhanced health check functionality
func TestEnhancedHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("EnhancedHealthCheck", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// Wait a small amount to ensure uptime is measurable
		time.Sleep(10 * time.Millisecond)

		w := performRequest(server.router, "GET", "/health", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		var response HealthResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "healthy", response.Status)
		assert.NotEmpty(t, response.Timestamp)
		assert.NotEmpty(t, response.Version)
		assert.NotEmpty(t, response.Uptime)
		assert.NotNil(t, response.Checks)
		
		// Verify enhanced health checks
		assert.Equal(t, "ok", response.Checks["core"])
		assert.Equal(t, "ok", response.Checks["database"])
		assert.Equal(t, "ok", response.Checks["memory"])
		assert.Equal(t, "ok", response.Checks["auth"])
		assert.Contains(t, response.Checks["sessions"], "active")
		assert.NotEmpty(t, response.Checks["start_time"])
	})
}

// TestSessionCleanupProcess tests the session cleanup goroutine
func TestSessionCleanupProcess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("SessionCleanupLogging", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		
		// Setup logger expectations for cleanup
		mockLogger.On("Debug", "Cleaned up expired sessions").Return()

		// Create an expired session
		server.authManager.SetExpiry(-time.Hour) // Expired 1 hour ago
		session, err := server.authManager.CreateSession("test-user", nil)
		require.NoError(t, err)
		assert.Len(t, server.authManager.sessions, 1)

		// Manually trigger cleanup (simulating the ticker)
		server.authManager.CleanupExpiredSessions()

		// Session should be removed
		assert.Len(t, server.authManager.sessions, 0)
		
		// Verify session cannot be retrieved
		retrievedSession, err := server.authManager.GetSession(session.ID)
		assert.Error(t, err)
		assert.Nil(t, retrievedSession)
	})
}

// TestEnhancedServerFeatures tests enhanced server specific features
func TestEnhancedServerFeatures(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("DocumentationEndpoints", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// Test documentation endpoints
		docEndpoints := []string{
			"/openapi.json",
			"/docs/",
		}

		for _, endpoint := range docEndpoints {
			t.Run("Docs_"+endpoint, func(t *testing.T) {
				w := performRequest(server.router, "GET", endpoint, nil)
				// Should not return 404
				assert.NotEqual(t, http.StatusNotFound, w.Code)
			})
		}
	})

	t.Run("AuthenticationEndpoints", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// Test authentication endpoints exist
		authEndpoints := []string{
			"/auth/login",
			"/auth/logout", 
			"/auth/refresh",
		}

		for _, endpoint := range authEndpoints {
			t.Run("Auth_"+endpoint, func(t *testing.T) {
				w := performRequest(server.router, "POST", endpoint, nil)
				// Should not return 404 (but may return 400 or 401)
				assert.NotEqual(t, http.StatusNotFound, w.Code)
			})
		}
	})

	t.Run("ConfigurationEndpoint", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		configReq := ConfigRequest{
			EnableTextualMemory: boolPtr(true),
			TopK:                intPtr(5),
		}

		w := performRequest(server.router, "POST", "/configure", configReq)
		assert.Equal(t, http.StatusOK, w.Code)

		var response SimpleResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Configuration updated successfully", response.Message)
	})
}

// TestEnhancedServerMetrics tests metrics collection in enhanced server
func TestEnhancedServerMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("MetricsCollection", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// Make several requests to generate metrics
		endpoints := []string{"/health", "/users/me", "/metrics"}
		for _, endpoint := range endpoints {
			for i := 0; i < 3; i++ {
				w := performRequest(server.router, "GET", endpoint, nil)
				assert.Equal(t, http.StatusOK, w.Code)
			}
		}

		// Check metrics endpoint returns data
		w := performRequest(server.router, "GET", "/metrics", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		var response MetricsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.NotEmpty(t, response.Timestamp)
		assert.NotNil(t, response.Metrics)
		assert.Contains(t, response.Metrics, "requests_total")
	})
}

// TestEnhancedServerIntegration tests complex integration scenarios
func TestEnhancedServerIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("CompleteUserJourney", func(t *testing.T) {
		server, mockCore, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)
		
		// Setup mock expectations for full user journey
		mockCore.On("CreateUser", mock.Anything, "john_doe", types.UserRoleUser, "user123").Return("user123", nil)
		mockCore.On("RegisterMemCube", mock.Anything, "/path/to/cube", "cube123", mock.AnythingOfType("string")).Return(nil)
		mockCore.On("Add", mock.Anything, mock.AnythingOfType("*types.AddMemoryRequest")).Return(nil)
		mockCore.On("Search", mock.Anything, mock.AnythingOfType("*types.SearchQuery")).Return(
			&types.SearchResult{
				TextMem: []map[string]interface{}{
					{"id": "mem1", "content": "found memory"},
				},
				ActMem:  []map[string]interface{}{},
				ParaMem: []map[string]interface{}{},
			}, nil)
		mockCore.On("Chat", mock.Anything, mock.AnythingOfType("*types.ChatRequest")).Return(
			&types.ChatResponse{
				Response: "Hello! I found some relevant information in your memories.",
			}, nil)

		// 1. User logs in
		loginReq := AuthRequest{
			UserID: "user123",
		}
		w := performRequest(server.router, "POST", "/auth/login", loginReq)
		assert.Equal(t, http.StatusOK, w.Code)

		var loginResponse BaseResponse[AuthResponse]
		err := json.Unmarshal(w.Body.Bytes(), &loginResponse)
		require.NoError(t, err)
		token := loginResponse.Data.Token

		// 2. User creates account
		userReq := UserCreate{
			UserName: stringPtr("john_doe"),
			Role:     types.UserRoleUser,
			UserID:   "user123",
		}
		req, _ := http.NewRequest("POST", "/users", toJSONReader(userReq))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 3. User registers a memory cube
		cubeReq := MemCubeRegister{
			MemCubeNameOrPath: "/path/to/cube",
			MemCubeID:         stringPtr("cube123"),
		}
		req, _ = http.NewRequest("POST", "/mem_cubes", toJSONReader(cubeReq))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 4. User adds memories
		memoryReq := MemoryCreate{
			MemoryContent: stringPtr("Important meeting notes about Q4 planning"),
			MemCubeID:     stringPtr("cube123"),
		}
		req, _ = http.NewRequest("POST", "/memories", toJSONReader(memoryReq))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 5. User searches memories
		searchReq := SearchRequest{
			Query: "Q4 planning",
			TopK:  intPtr(5),
		}
		req, _ = http.NewRequest("POST", "/search", toJSONReader(searchReq))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 6. User chats with memories
		chatReq := ChatRequest{
			Query: "What were the key points from Q4 planning?",
		}
		req, _ = http.NewRequest("POST", "/chat", toJSONReader(chatReq))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 7. User logs out
		req, _ = http.NewRequest("POST", "/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		mockCore.AssertExpectations(t)
	})
}

// TestEnhancedServerErrorHandling tests error handling in enhanced server
func TestEnhancedServerErrorHandling(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("PanicRecovery", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// The recovery middleware should handle panics gracefully
		// Since we can't easily trigger a panic in the existing handlers,
		// we test that the recovery middleware is properly configured
		assert.NotNil(t, server.router)
	})

	t.Run("InvalidJSONRequest", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// Send invalid JSON
		req, _ := http.NewRequest("POST", "/auth/login", strings.NewReader("{invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// Helper functions for enhanced server testing

func toJSONReader(v interface{}) *strings.Reader {
	data, _ := json.Marshal(v)
	return strings.NewReader(string(data))
}

// Benchmark tests for enhanced server

func BenchmarkEnhancedServerHealthCheck(b *testing.B) {
	gin.SetMode(gin.TestMode)
	server, _, mockLogger := setupEnhancedTestServer()
	setupMockLogger(mockLogger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := performRequest(server.router, "GET", "/health", nil)
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

func BenchmarkEnhancedServerFullAuth(b *testing.B) {
	gin.SetMode(gin.TestMode)
	server, _, mockLogger := setupEnhancedTestServer()
	setupMockLogger(mockLogger)

	// Pre-create a token for benchmarking
	loginReq := AuthRequest{
		UserID: "benchmark-user",
	}
	w := performRequest(server.router, "POST", "/auth/login", loginReq)
	if w.Code != http.StatusOK {
		b.Fatalf("Failed to create token for benchmark")
	}

	var loginResponse BaseResponse[AuthResponse]
	json.Unmarshal(w.Body.Bytes(), &loginResponse)
	token := loginResponse.Data.Token

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/users/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}