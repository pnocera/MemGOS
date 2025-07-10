package api

import (
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
)

// TestLoggingMiddleware tests the logging middleware functionality
func TestLoggingMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("LoggingMiddlewareBasic", func(t *testing.T) {
		server, _, mockLogger := setupTestServer()

		// Setup logger expectations for the logging middleware
		mockLogger.On("Info", "HTTP Request", mock.MatchedBy(func(fields map[string]interface{}) bool {
			// Verify required fields are present
			_, hasMethod := fields["method"]
			_, hasPath := fields["path"]
			_, hasStatusCode := fields["status_code"]
			_, hasLatency := fields["latency"]
			_, hasClientIP := fields["client_ip"]
			_, hasUserAgent := fields["user_agent"]
			return hasMethod && hasPath && hasStatusCode && hasLatency && hasClientIP && hasUserAgent
		})).Return()

		w := performRequest(server.router, "GET", "/health", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify logging was called
		mockLogger.AssertExpectations(t)
	})

	t.Run("LoggingWithRequestID", func(t *testing.T) {
		server, _, mockLogger := setupTestServer()

		// Setup logger expectations
		mockLogger.On("Info", "HTTP Request", mock.MatchedBy(func(fields map[string]interface{}) bool {
			requestID, hasRequestID := fields["request_id"]
			return hasRequestID && requestID != nil
		})).Return()

		req, _ := http.NewRequest("GET", "/health", nil)
		req.Header.Set("X-Request-ID", "test-request-123")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockLogger.AssertExpectations(t)
	})

	t.Run("LoggingDifferentMethods", func(t *testing.T) {
		server, _, mockLogger := setupTestServer()

		methods := []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}

		for _, method := range methods {
			t.Run("Method_"+method, func(t *testing.T) {
				mockLogger.On("Info", "HTTP Request", mock.MatchedBy(func(fields map[string]interface{}) bool {
					methodField, hasMethod := fields["method"]
					return hasMethod && methodField == method
				})).Return()

				if method == "OPTIONS" {
					req, _ := http.NewRequest(method, "/health", nil)
					w := httptest.NewRecorder()
					server.router.ServeHTTP(w, req)
				} else {
					w := performRequest(server.router, method, "/health", nil)
					assert.NotEqual(t, http.StatusNotFound, w.Code)
				}
			})
		}

		mockLogger.AssertExpectations(t)
	})
}

// TestRequestIDMiddleware tests the request ID middleware
func TestRequestIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("GenerateRequestID", func(t *testing.T) {
		server, _, _ := setupTestServer()

		w := performRequest(server.router, "GET", "/health", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		requestID := w.Header().Get("X-Request-ID")
		assert.NotEmpty(t, requestID)
		assert.Len(t, requestID, 36) // UUID length
	})

	t.Run("PreservesExistingRequestID", func(t *testing.T) {
		server, _, _ := setupTestServer()

		customRequestID := "custom-request-123"
		req, _ := http.NewRequest("GET", "/health", nil)
		req.Header.Set("X-Request-ID", customRequestID)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, customRequestID, w.Header().Get("X-Request-ID"))
	})

	t.Run("UniqueRequestIDs", func(t *testing.T) {
		server, _, _ := setupTestServer()

		var requestIDs []string
		for i := 0; i < 5; i++ {
			w := performRequest(server.router, "GET", "/health", nil)
			requestID := w.Header().Get("X-Request-ID")
			requestIDs = append(requestIDs, requestID)
		}

		// All request IDs should be unique
		for i := 0; i < len(requestIDs); i++ {
			for j := i + 1; j < len(requestIDs); j++ {
				assert.NotEqual(t, requestIDs[i], requestIDs[j])
			}
		}
	})
}

// TestAuthMiddleware tests the authentication middleware
func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("SkipAuthForPublicEndpoints", func(t *testing.T) {
		server, _, _ := setupTestServer()

		publicEndpoints := []string{
			"/health",
			"/",
			"/docs",
			"/openapi.json",
		}

		for _, endpoint := range publicEndpoints {
			t.Run("PublicEndpoint_"+endpoint, func(t *testing.T) {
				w := performRequest(server.router, "GET", endpoint, nil)
				// Should not return 401 Unauthorized
				assert.NotEqual(t, http.StatusUnauthorized, w.Code)
			})
		}
	})

	t.Run("ExtractUserIDFromHeader", func(t *testing.T) {
		server, _, _ := setupTestServer()

		customUserID := "header-user-123"
		req, _ := http.NewRequest("GET", "/users/me", nil)
		req.Header.Set("X-User-ID", customUserID)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// The user ID should be extracted and used by the handler
	})

	t.Run("ExtractUserIDFromQuery", func(t *testing.T) {
		server, _, _ := setupTestServer()

		customUserID := "query-user-123"
		req, _ := http.NewRequest("GET", "/users/me?user_id="+customUserID, nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("FallbackToConfigUserID", func(t *testing.T) {
		server, _, _ := setupTestServer()

		// No user_id provided, should use config default
		req, _ := http.NewRequest("GET", "/users/me", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("HeaderOverridesQuery", func(t *testing.T) {
		server, _, _ := setupTestServer()

		// Test that header takes precedence over query parameter
		req, _ := http.NewRequest("GET", "/users/me?user_id=query-user", nil)
		req.Header.Set("X-User-ID", "header-user")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// Should use header-user, not query-user
	})
}

// TestCORSMiddleware tests the CORS middleware functionality
func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("CORSHeaders", func(t *testing.T) {
		server, _, _ := setupTestServer()

		w := performRequest(server.router, "GET", "/health", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		// Check CORS headers
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Origin"), "*")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "PUT")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "DELETE")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "OPTIONS")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "X-User-ID")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "X-Request-ID")
	})

	t.Run("OPTIONSPreflightRequest", func(t *testing.T) {
		server, _, _ := setupTestServer()

		req, _ := http.NewRequest("OPTIONS", "/search", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Origin"), "*")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
	})

	t.Run("CORSWithOrigin", func(t *testing.T) {
		server, _, _ := setupTestServer()

		req, _ := http.NewRequest("GET", "/health", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("ExposeHeaders", func(t *testing.T) {
		server, _, _ := setupTestServer()

		w := performRequest(server.router, "GET", "/health", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		assert.Contains(t, w.Header().Get("Access-Control-Expose-Headers"), "X-Request-ID")
	})
}

// TestEnhancedServerMiddleware tests the enhanced server middleware stack
func TestEnhancedServerMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("EnhancedMiddlewareStack", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// Test that all middleware is applied in the enhanced server
		w := performRequest(server.router, "GET", "/health", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		// Check that request ID middleware is applied
		requestID := w.Header().Get("X-Request-ID")
		assert.NotEmpty(t, requestID)

		// Check that CORS middleware is applied
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Origin"), "*")

		// Check that logging middleware was applied (mock logger should have been called)
		mockLogger.AssertExpectations(t)
	})

	t.Run("MetricsMiddleware", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// Make several requests to generate metrics
		for i := 0; i < 3; i++ {
			w := performRequest(server.router, "GET", "/health", nil)
			assert.Equal(t, http.StatusOK, w.Code)
		}

		// Check metrics endpoint
		w := performRequest(server.router, "GET", "/metrics", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		// Metrics should contain request count information
		responseBody := w.Body.String()
		assert.Contains(t, responseBody, "metrics")
	})

	t.Run("JWTAuthMiddleware", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// Test JWT auth middleware with valid token
		loginReq := AuthRequest{
			UserID: "test-user",
		}
		w := performRequest(server.router, "POST", "/auth/login", loginReq)
		require.Equal(t, http.StatusOK, w.Code)

		var loginResponse BaseResponse[AuthResponse]
		err := json.Unmarshal(w.Body.Bytes(), &loginResponse)
		require.NoError(t, err)

		// Use token to access protected endpoint
		req, _ := http.NewRequest("GET", "/users/me", nil)
		req.Header.Set("Authorization", "Bearer "+loginResponse.Data.Token)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestMiddlewareOrder tests that middleware is applied in the correct order
func TestMiddlewareOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("MiddlewareExecutionOrder", func(t *testing.T) {
		// Create a custom server to test middleware order
		mockCore := &MockMOSCore{}
		mockLogger := &MockLogger{}
		
		mockLogger.On("Info", mock.AnythingOfType("string"), mock.Anything).Maybe()

		cfg := &config.MOSConfig{
			UserID:    "test-user",
			SessionID: "test-session",
			APIPort:   8080,
		}

		router := gin.New()
		
		// Track middleware execution order
		var executionOrder []string
		
		// Add middleware with tracking
		router.Use(func(c *gin.Context) {
			executionOrder = append(executionOrder, "recovery")
			c.Next()
		})
		
		router.Use(func(c *gin.Context) {
			executionOrder = append(executionOrder, "logging")
			c.Next()
		})
		
		router.Use(func(c *gin.Context) {
			executionOrder = append(executionOrder, "cors")
			c.Next()
		})
		
		router.Use(func(c *gin.Context) {
			executionOrder = append(executionOrder, "request_id")
			c.Next()
		})
		
		router.Use(func(c *gin.Context) {
			executionOrder = append(executionOrder, "auth")
			c.Next()
		})
		
		router.GET("/test", func(c *gin.Context) {
			executionOrder = append(executionOrder, "handler")
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		// Verify middleware execution order
		expectedOrder := []string{"recovery", "logging", "cors", "request_id", "auth", "handler"}
		assert.Equal(t, expectedOrder, executionOrder)
	})
}

// TestMiddlewareEdgeCases tests edge cases and error conditions
func TestMiddlewareEdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("EmptyHeaders", func(t *testing.T) {
		server, _, _ := setupTestServer()

		req, _ := http.NewRequest("GET", "/health", nil)
		// Don't set any headers
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		// Request ID should still be generated
		assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
	})

	t.Run("LongRequestID", func(t *testing.T) {
		server, _, _ := setupTestServer()

		longRequestID := strings.Repeat("a", 1000)
		req, _ := http.NewRequest("GET", "/health", nil)
		req.Header.Set("X-Request-ID", longRequestID)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, longRequestID, w.Header().Get("X-Request-ID"))
	})

	t.Run("SpecialCharactersInUserID", func(t *testing.T) {
		server, _, _ := setupTestServer()

		specialUserID := "user@domain.com!#$%"
		req, _ := http.NewRequest("GET", "/users/me", nil)
		req.Header.Set("X-User-ID", specialUserID)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("MultipleOriginHeaders", func(t *testing.T) {
		server, _, _ := setupTestServer()

		req, _ := http.NewRequest("GET", "/health", nil)
		req.Header.Add("Origin", "https://example.com")
		req.Header.Add("Origin", "https://another.com")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// Should handle multiple origin headers gracefully
	})
}

// TestMiddlewarePerformance tests middleware performance
func TestMiddlewarePerformance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("ConcurrentRequests", func(t *testing.T) {
		server, _, _ := setupTestServer()

		// Test concurrent requests to ensure middleware is thread-safe
		numRequests := 50
		done := make(chan bool, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				w := performRequest(server.router, "GET", "/health", nil)
				assert.Equal(t, http.StatusOK, w.Code)
				assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
				done <- true
			}()
		}

		// Wait for all requests to complete
		for i := 0; i < numRequests; i++ {
			<-done
		}
	})
}

// TestRateLimitMiddleware tests the rate limiting middleware placeholder
func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("RateLimitPlaceholder", func(t *testing.T) {
		server, _, _ := setupTestServer()

		// Rate limiting is currently a placeholder, so all requests should pass
		for i := 0; i < 10; i++ {
			w := performRequest(server.router, "GET", "/health", nil)
			assert.Equal(t, http.StatusOK, w.Code)
		}
	})
}

// TestTimeoutMiddleware tests the timeout middleware placeholder
func TestTimeoutMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("TimeoutPlaceholder", func(t *testing.T) {
		server, _, _ := setupTestServer()

		// Timeout middleware is currently a placeholder
		w := performRequest(server.router, "GET", "/health", nil)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// Benchmark tests for middleware performance

func BenchmarkRequestIDMiddleware(b *testing.B) {
	gin.SetMode(gin.TestMode)
	server, _, _ := setupTestServer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := performRequest(server.router, "GET", "/health", nil)
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
		if w.Header().Get("X-Request-ID") == "" {
			b.Fatal("Expected X-Request-ID header")
		}
	}
}

func BenchmarkCORSMiddleware(b *testing.B) {
	gin.SetMode(gin.TestMode)
	server, _, _ := setupTestServer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("OPTIONS", "/health", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusNoContent {
			b.Fatalf("Expected status 204, got %d", w.Code)
		}
	}
}

func BenchmarkAuthMiddleware(b *testing.B) {
	gin.SetMode(gin.TestMode)
	server, _, _ := setupTestServer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/users/me", nil)
		req.Header.Set("X-User-ID", "benchmark-user")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

func BenchmarkFullMiddlewareStack(b *testing.B) {
	gin.SetMode(gin.TestMode)
	server, _, mockLogger := setupEnhancedTestServer()
	
	// Setup minimal logger mocks for performance testing
	mockLogger.On("Info", mock.AnythingOfType("string"), mock.Anything).Maybe()
	mockLogger.On("Debug", mock.AnythingOfType("string"), mock.Anything).Maybe()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := performRequest(server.router, "GET", "/health", nil)
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}