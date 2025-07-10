package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuthManager tests the authentication manager functionality
func TestAuthManager(t *testing.T) {
	t.Run("NewAuthManager", func(t *testing.T) {
		authManager := NewAuthManager()
		
		assert.NotNil(t, authManager)
		assert.NotNil(t, authManager.sessions)
		assert.NotNil(t, authManager.jwtSecret)
		assert.Equal(t, 24*time.Hour, authManager.expiry)
		assert.Len(t, authManager.sessions, 0)
	})

	t.Run("SetJWTSecret", func(t *testing.T) {
		authManager := NewAuthManager()
		customSecret := "my-custom-jwt-secret-key"
		
		authManager.SetJWTSecret(customSecret)
		assert.Equal(t, []byte(customSecret), authManager.jwtSecret)
	})

	t.Run("SetExpiry", func(t *testing.T) {
		authManager := NewAuthManager()
		customExpiry := 12 * time.Hour
		
		authManager.SetExpiry(customExpiry)
		assert.Equal(t, customExpiry, authManager.expiry)
	})
}

// TestSessionManagement tests session creation, retrieval, and invalidation
func TestSessionManagement(t *testing.T) {
	t.Run("CreateSession", func(t *testing.T) {
		authManager := NewAuthManager()
		userID := "test-user-123"
		metadata := map[string]interface{}{
			"ip_address": "192.168.1.1",
			"user_agent": "test-agent",
		}

		session, err := authManager.CreateSession(userID, metadata)
		
		require.NoError(t, err)
		assert.NotNil(t, session)
		assert.NotEmpty(t, session.ID)
		assert.Equal(t, userID, session.UserID)
		assert.True(t, session.Active)
		assert.Equal(t, metadata, session.Metadata)
		assert.True(t, time.Now().Before(session.ExpiresAt))
		assert.True(t, time.Now().After(session.CreatedAt.Add(-time.Second)))

		// Verify session is stored in manager
		assert.Len(t, authManager.sessions, 1)
		assert.Contains(t, authManager.sessions, session.ID)
	})

	t.Run("GetSession", func(t *testing.T) {
		authManager := NewAuthManager()
		userID := "test-user-123"

		// Create session
		originalSession, err := authManager.CreateSession(userID, nil)
		require.NoError(t, err)

		// Retrieve session
		retrievedSession, err := authManager.GetSession(originalSession.ID)
		
		require.NoError(t, err)
		assert.Equal(t, originalSession.ID, retrievedSession.ID)
		assert.Equal(t, originalSession.UserID, retrievedSession.UserID)
		assert.True(t, retrievedSession.LastSeen.After(originalSession.LastSeen))
	})

	t.Run("GetSessionNotFound", func(t *testing.T) {
		authManager := NewAuthManager()
		
		session, err := authManager.GetSession("non-existent-session")
		
		assert.Error(t, err)
		assert.Nil(t, session)
		assert.Contains(t, err.Error(), "session not found")
	})

	t.Run("GetExpiredSession", func(t *testing.T) {
		authManager := NewAuthManager()
		authManager.SetExpiry(-time.Hour) // Set expiry to past
		userID := "test-user-123"

		// Create expired session
		originalSession, err := authManager.CreateSession(userID, nil)
		require.NoError(t, err)

		// Try to retrieve expired session
		retrievedSession, err := authManager.GetSession(originalSession.ID)
		
		assert.Error(t, err)
		assert.Nil(t, retrievedSession)
		assert.Contains(t, err.Error(), "session expired or inactive")
	})

	t.Run("InvalidateSession", func(t *testing.T) {
		authManager := NewAuthManager()
		userID := "test-user-123"

		// Create session
		session, err := authManager.CreateSession(userID, nil)
		require.NoError(t, err)
		assert.True(t, session.Active)

		// Invalidate session
		authManager.InvalidateSession(session.ID)

		// Verify session is inactive
		invalidatedSession := authManager.sessions[session.ID]
		assert.False(t, invalidatedSession.Active)

		// Try to retrieve invalidated session
		retrievedSession, err := authManager.GetSession(session.ID)
		assert.Error(t, err)
		assert.Nil(t, retrievedSession)
	})

	t.Run("CleanupExpiredSessions", func(t *testing.T) {
		authManager := NewAuthManager()

		// Create active session
		activeSession, err := authManager.CreateSession("active-user", nil)
		require.NoError(t, err)

		// Create expired session by manipulating expiry
		expiredSession, err := authManager.CreateSession("expired-user", nil)
		require.NoError(t, err)
		expiredSession.ExpiresAt = time.Now().Add(-time.Hour)
		authManager.sessions[expiredSession.ID] = expiredSession

		// Create inactive session
		inactiveSession, err := authManager.CreateSession("inactive-user", nil)
		require.NoError(t, err)
		inactiveSession.Active = false
		authManager.sessions[inactiveSession.ID] = inactiveSession

		assert.Len(t, authManager.sessions, 3)

		// Cleanup expired sessions
		authManager.CleanupExpiredSessions()

		// Only active session should remain
		assert.Len(t, authManager.sessions, 1)
		assert.Contains(t, authManager.sessions, activeSession.ID)
		assert.NotContains(t, authManager.sessions, expiredSession.ID)
		assert.NotContains(t, authManager.sessions, inactiveSession.ID)
	})
}

// TestJWTOperations tests JWT generation and validation
func TestJWTOperations(t *testing.T) {
	t.Run("GenerateJWT", func(t *testing.T) {
		authManager := NewAuthManager()
		authManager.SetJWTSecret("test-secret-key")
		
		userID := "test-user-123"
		role := "admin"
		session, err := authManager.CreateSession(userID, nil)
		require.NoError(t, err)

		token, err := authManager.GenerateJWT(session, role)
		
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		
		// JWT should have 3 parts separated by dots
		parts := len(strings.Split(token, "."))
		assert.Equal(t, 3, parts)
	})

	t.Run("ValidateJWT", func(t *testing.T) {
		authManager := NewAuthManager()
		authManager.SetJWTSecret("test-secret-key")
		
		userID := "test-user-123"
		role := "admin"
		session, err := authManager.CreateSession(userID, nil)
		require.NoError(t, err)

		// Generate token
		token, err := authManager.GenerateJWT(session, role)
		require.NoError(t, err)

		// Validate token
		claims, err := authManager.ValidateJWT(token)
		
		require.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, session.ID, claims.SessionID)
		assert.Equal(t, role, claims.Role)
		assert.Equal(t, "memgos-api", claims.Issuer)
		assert.Equal(t, userID, claims.Subject)
	})

	t.Run("ValidateInvalidJWT", func(t *testing.T) {
		authManager := NewAuthManager()
		authManager.SetJWTSecret("test-secret-key")

		testCases := []struct {
			name  string
			token string
		}{
			{"EmptyToken", ""},
			{"InvalidFormat", "invalid.token"},
			{"WrongSecret", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"},
			{"MalformedToken", "not.a.jwt.token"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				claims, err := authManager.ValidateJWT(tc.token)
				assert.Error(t, err)
				assert.Nil(t, claims)
			})
		}
	})

	t.Run("ValidateJWTWithInvalidSession", func(t *testing.T) {
		authManager := NewAuthManager()
		authManager.SetJWTSecret("test-secret-key")
		
		userID := "test-user-123"
		session, err := authManager.CreateSession(userID, nil)
		require.NoError(t, err)

		// Generate token
		token, err := authManager.GenerateJWT(session, "user")
		require.NoError(t, err)

		// Invalidate session
		authManager.InvalidateSession(session.ID)

		// Try to validate token with invalid session
		claims, err := authManager.ValidateJWT(token)
		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.Contains(t, err.Error(), "session invalid")
	})
}

// TestAuthenticationEndpoints tests the authentication HTTP endpoints
func TestAuthenticationEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("LoginEndpoint", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		loginReq := AuthRequest{
			UserID:   "test-user",
			Password: "test-password",
		}

		w := performRequest(server.router, "POST", "/auth/login", loginReq)
		
		assert.Equal(t, http.StatusOK, w.Code)

		var response BaseResponse[AuthResponse]
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, http.StatusOK, response.Code)
		assert.Equal(t, "Login successful", response.Message)
		assert.NotNil(t, response.Data)
		assert.NotEmpty(t, response.Data.Token)
		assert.NotEmpty(t, response.Data.SessionID)
		assert.Equal(t, "test-user", response.Data.UserID)
		assert.NotNil(t, response.Data.Metadata)
	})

	t.Run("LoginInvalidRequest", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// Missing required user_id field
		invalidReq := map[string]interface{}{
			"password": "test-password",
		}

		w := performRequest(server.router, "POST", "/auth/login", invalidReq)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Contains(t, response.Message, "Invalid request format")
	})

	t.Run("LogoutEndpoint", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// First login to get a session
		loginReq := AuthRequest{
			UserID: "test-user",
		}
		w := performRequest(server.router, "POST", "/auth/login", loginReq)
		require.Equal(t, http.StatusOK, w.Code)

		var loginResponse BaseResponse[AuthResponse]
		err := json.Unmarshal(w.Body.Bytes(), &loginResponse)
		require.NoError(t, err)

		// Now logout
		req, _ := http.NewRequest("POST", "/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer "+loginResponse.Data.Token)
		req.Header.Set("Content-Type", "application/json")
		
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)

		var logoutResponse SimpleResponse
		err = json.Unmarshal(w.Body.Bytes(), &logoutResponse)
		require.NoError(t, err)
		assert.Equal(t, "Logout successful", logoutResponse.Message)

		// Verify session is invalidated
		session, err := server.authManager.GetSession(loginResponse.Data.SessionID)
		assert.Error(t, err)
		assert.Nil(t, session)
	})

	t.Run("RefreshTokenEndpoint", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// First login to get a token
		loginReq := AuthRequest{
			UserID: "test-user",
		}
		w := performRequest(server.router, "POST", "/auth/login", loginReq)
		require.Equal(t, http.StatusOK, w.Code)

		var loginResponse BaseResponse[AuthResponse]
		err := json.Unmarshal(w.Body.Bytes(), &loginResponse)
		require.NoError(t, err)

		originalToken := loginResponse.Data.Token

		// Refresh token
		req, _ := http.NewRequest("POST", "/auth/refresh", nil)
		req.Header.Set("Authorization", "Bearer "+originalToken)
		req.Header.Set("Content-Type", "application/json")
		
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)

		var refreshResponse BaseResponse[AuthResponse]
		err = json.Unmarshal(w.Body.Bytes(), &refreshResponse)
		require.NoError(t, err)
		
		assert.Equal(t, "Token refreshed successfully", refreshResponse.Message)
		assert.NotEmpty(t, refreshResponse.Data.Token)
		assert.NotEqual(t, originalToken, refreshResponse.Data.Token) // New token should be different
		assert.Equal(t, loginResponse.Data.SessionID, refreshResponse.Data.SessionID) // Same session
	})

	t.Run("RefreshTokenInvalid", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// Try to refresh with invalid token
		req, _ := http.NewRequest("POST", "/auth/refresh", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Invalid token", response.Message)
	})

	t.Run("RefreshTokenMissing", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// Try to refresh without token
		req, _ := http.NewRequest("POST", "/auth/refresh", nil)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "No token provided", response.Message)
	})
}

// TestJWTAuthMiddleware tests the JWT authentication middleware
func TestJWTAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("ValidJWTToken", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// First login to get a valid token
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

	t.Run("PublicEndpointAccess", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		publicEndpoints := []string{
			"/health",
			"/",
			"/openapi.json",
			"/auth/login",
		}

		for _, endpoint := range publicEndpoints {
			t.Run("PublicEndpoint_"+endpoint, func(t *testing.T) {
				w := performRequest(server.router, "GET", endpoint, nil)
				// Should not return 401 Unauthorized
				assert.NotEqual(t, http.StatusUnauthorized, w.Code)
			})
		}
	})

	t.Run("FallbackToHeaderAuth", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// Access endpoint with X-User-ID header (fallback auth)
		req, _ := http.NewRequest("GET", "/users/me", nil)
		req.Header.Set("X-User-ID", "header-user")
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("FallbackToQueryAuth", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// Access endpoint with user_id query parameter
		req, _ := http.NewRequest("GET", "/users/me?user_id=query-user", nil)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("DefaultConfigAuth", func(t *testing.T) {
		server, _, mockLogger := setupEnhancedTestServer()
		setupMockLogger(mockLogger)

		// Access endpoint without any auth (should use default from config)
		req, _ := http.NewRequest("GET", "/users/me", nil)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestHelperFunctions tests the helper functions
func TestHelperFunctions(t *testing.T) {
	t.Run("generateSessionID", func(t *testing.T) {
		sessionID1, err := generateSessionID()
		require.NoError(t, err)
		assert.NotEmpty(t, sessionID1)
		assert.Len(t, sessionID1, 32) // 16 bytes = 32 hex characters

		sessionID2, err := generateSessionID()
		require.NoError(t, err)
		assert.NotEqual(t, sessionID1, sessionID2) // Should be unique
	})

	t.Run("extractTokenFromHeader", func(t *testing.T) {
		testCases := []struct {
			name           string
			authHeader     string
			expectedToken  string
		}{
			{
				name:           "ValidBearerToken",
				authHeader:     "Bearer abc123def456",
				expectedToken:  "abc123def456",
			},
			{
				name:           "ValidBearerTokenLowercase",
				authHeader:     "bearer xyz789",
				expectedToken:  "xyz789",
			},
			{
				name:           "EmptyHeader",
				authHeader:     "",
				expectedToken:  "",
			},
			{
				name:           "InvalidFormat",
				authHeader:     "Invalid format",
				expectedToken:  "",
			},
			{
				name:           "MissingToken",
				authHeader:     "Bearer",
				expectedToken:  "",
			},
			{
				name:           "DifferentAuthType",
				authHeader:     "Basic user:pass",
				expectedToken:  "",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				token := extractTokenFromHeader(tc.authHeader)
				assert.Equal(t, tc.expectedToken, token)
			})
		}
	})

	t.Run("isPublicEndpoint", func(t *testing.T) {
		server, _, _ := setupEnhancedTestServer()

		testCases := []struct {
			path     string
			expected bool
		}{
			{"/health", true},
			{"/", true},
			{"/docs", true},
			{"/docs/swagger", true},
			{"/openapi.json", true},
			{"/auth/login", true},
			{"/auth/refresh", true},
			{"/auth/logout", false}, // Not in public paths
			{"/users", false},
			{"/memories", false},
			{"/search", false},
			{"/chat", false},
		}

		for _, tc := range testCases {
			t.Run("Path_"+tc.path, func(t *testing.T) {
				result := server.isPublicEndpoint(tc.path)
				assert.Equal(t, tc.expected, result)
			})
		}
	})
}

// TestSessionCleanupGoroutine tests the session cleanup functionality
func TestSessionCleanupGoroutine(t *testing.T) {
	t.Run("SessionCleanupProcess", func(t *testing.T) {
		authManager := NewAuthManager()
		authManager.SetExpiry(100 * time.Millisecond) // Very short expiry for testing

		// Create session that will expire quickly
		session, err := authManager.CreateSession("test-user", nil)
		require.NoError(t, err)
		assert.Len(t, authManager.sessions, 1)

		// Wait for expiry
		time.Sleep(150 * time.Millisecond)

		// Manual cleanup (simulating the cleanup goroutine)
		authManager.CleanupExpiredSessions()

		// Session should be removed
		assert.Len(t, authManager.sessions, 0)
		
		// Verify session cannot be retrieved
		retrievedSession, err := authManager.GetSession(session.ID)
		assert.Error(t, err)
		assert.Nil(t, retrievedSession)
	})
}

// Benchmark tests for authentication performance

func BenchmarkCreateSession(b *testing.B) {
	authManager := NewAuthManager()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userID := fmt.Sprintf("user-%d", i)
		_, err := authManager.CreateSession(userID, nil)
		if err != nil {
			b.Fatalf("Failed to create session: %v", err)
		}
	}
}

func BenchmarkGenerateJWT(b *testing.B) {
	authManager := NewAuthManager()
	authManager.SetJWTSecret("benchmark-secret-key")
	
	session, err := authManager.CreateSession("benchmark-user", nil)
	if err != nil {
		b.Fatalf("Failed to create session: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := authManager.GenerateJWT(session, "user")
		if err != nil {
			b.Fatalf("Failed to generate JWT: %v", err)
		}
	}
}

func BenchmarkValidateJWT(b *testing.B) {
	authManager := NewAuthManager()
	authManager.SetJWTSecret("benchmark-secret-key")
	
	session, err := authManager.CreateSession("benchmark-user", nil)
	if err != nil {
		b.Fatalf("Failed to create session: %v", err)
	}
	
	token, err := authManager.GenerateJWT(session, "user")
	if err != nil {
		b.Fatalf("Failed to generate JWT: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := authManager.ValidateJWT(token)
		if err != nil {
			b.Fatalf("Failed to validate JWT: %v", err)
		}
	}
}

func BenchmarkLoginEndpoint(b *testing.B) {
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