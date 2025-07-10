package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthManager handles authentication and session management
type AuthManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	jwtSecret []byte
	expiry   time.Duration
}

// Session represents a user session
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	LastSeen  time.Time `json:"last_seen"`
	ExpiresAt time.Time `json:"expires_at"`
	Active    bool      `json:"active"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Claims represents JWT claims
type Claims struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	Role      string `json:"role,omitempty"`
	jwt.RegisteredClaims
}

// AuthRequest represents an authentication request
type AuthRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	Token     string                 `json:"token"`
	SessionID string                 `json:"session_id"`
	UserID    string                 `json:"user_id"`
	ExpiresAt time.Time              `json:"expires_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewAuthManager creates a new authentication manager
func NewAuthManager() *AuthManager {
	secret := make([]byte, 32)
	rand.Read(secret)

	return &AuthManager{
		sessions:  make(map[string]*Session),
		jwtSecret: secret,
		expiry:    24 * time.Hour, // Default 24 hour expiry
	}
}

// SetJWTSecret sets the JWT signing secret
func (am *AuthManager) SetJWTSecret(secret string) {
	am.jwtSecret = []byte(secret)
}

// SetExpiry sets the session expiry duration
func (am *AuthManager) SetExpiry(duration time.Duration) {
	am.expiry = duration
}

// CreateSession creates a new session for a user
func (am *AuthManager) CreateSession(userID string, metadata map[string]interface{}) (*Session, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	now := time.Now()
	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		CreatedAt: now,
		LastSeen:  now,
		ExpiresAt: now.Add(am.expiry),
		Active:    true,
		Metadata:  metadata,
	}

	am.mu.Lock()
	am.sessions[sessionID] = session
	am.mu.Unlock()

	return session, nil
}

// GetSession retrieves a session by ID
func (am *AuthManager) GetSession(sessionID string) (*Session, error) {
	am.mu.RLock()
	session, exists := am.sessions[sessionID]
	am.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	if !session.Active || time.Now().After(session.ExpiresAt) {
		am.InvalidateSession(sessionID)
		return nil, fmt.Errorf("session expired or inactive")
	}

	// Update last seen
	am.mu.Lock()
	session.LastSeen = time.Now()
	am.mu.Unlock()

	return session, nil
}

// InvalidateSession invalidates a session
func (am *AuthManager) InvalidateSession(sessionID string) {
	am.mu.Lock()
	if session, exists := am.sessions[sessionID]; exists {
		session.Active = false
	}
	am.mu.Unlock()
}

// CleanupExpiredSessions removes expired sessions
func (am *AuthManager) CleanupExpiredSessions() {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()
	for id, session := range am.sessions {
		if !session.Active || now.After(session.ExpiresAt) {
			delete(am.sessions, id)
		}
	}
}

// GenerateJWT generates a JWT token for a session
func (am *AuthManager) GenerateJWT(session *Session, role string) (string, error) {
	claims := &Claims{
		UserID:    session.UserID,
		SessionID: session.ID,
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(session.ExpiresAt),
			IssuedAt:  jwt.NewNumericDate(session.CreatedAt),
			NotBefore: jwt.NewNumericDate(session.CreatedAt),
			Issuer:    "memgos-api",
			Subject:   session.UserID,
			ID:        session.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(am.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return tokenString, nil
}

// ValidateJWT validates a JWT token and returns the claims
func (am *AuthManager) ValidateJWT(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return am.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// Verify session is still active
		_, err := am.GetSession(claims.SessionID)
		if err != nil {
			return nil, fmt.Errorf("session invalid: %w", err)
		}
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// Login handles user login and creates a session
func (s *Server) login(c *gin.Context) {
	var req AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request format",
			Error:   err.Error(),
		})
		return
	}

	// TODO: Validate user credentials
	// For now, accept any user_id

	// Create session
	session, err := s.authManager.CreateSession(req.UserID, map[string]interface{}{
		"login_time": time.Now(),
		"ip_address": c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	})
	if err != nil {
		s.handleError(c, "Failed to create session", err)
		return
	}

	// Generate JWT
	token, err := s.authManager.GenerateJWT(session, "user") // Default role
	if err != nil {
		s.handleError(c, "Failed to generate token", err)
		return
	}

	response := AuthResponse{
		Token:     token,
		SessionID: session.ID,
		UserID:    session.UserID,
		ExpiresAt: session.ExpiresAt,
		Metadata:  session.Metadata,
	}

	c.JSON(http.StatusOK, BaseResponse[AuthResponse]{
		Code:    http.StatusOK,
		Message: "Login successful",
		Data:    &response,
	})
}

// Logout handles user logout
func (s *Server) logout(c *gin.Context) {
	sessionID := c.GetString("session_id")
	if sessionID != "" {
		s.authManager.InvalidateSession(sessionID)
	}

	c.JSON(http.StatusOK, SimpleResponse{
		Code:    http.StatusOK,
		Message: "Logout successful",
	})
}

// RefreshToken handles token refresh
func (s *Server) refreshToken(c *gin.Context) {
	tokenString := extractTokenFromHeader(c.GetHeader("Authorization"))
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "No token provided",
		})
		return
	}

	claims, err := s.authManager.ValidateJWT(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "Invalid token",
			Error:   err.Error(),
		})
		return
	}

	// Get current session
	session, err := s.authManager.GetSession(claims.SessionID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "Session invalid",
			Error:   err.Error(),
		})
		return
	}

	// Extend session
	session.ExpiresAt = time.Now().Add(s.authManager.expiry)

	// Generate new token
	newToken, err := s.authManager.GenerateJWT(session, claims.Role)
	if err != nil {
		s.handleError(c, "Failed to generate new token", err)
		return
	}

	response := AuthResponse{
		Token:     newToken,
		SessionID: session.ID,
		UserID:    session.UserID,
		ExpiresAt: session.ExpiresAt,
	}

	c.JSON(http.StatusOK, BaseResponse[AuthResponse]{
		Code:    http.StatusOK,
		Message: "Token refreshed successfully",
		Data:    &response,
	})
}

// Enhanced auth middleware with JWT support
func (s *Server) jwtAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication for certain endpoints
		if s.isPublicEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Try JWT first
		tokenString := extractTokenFromHeader(c.GetHeader("Authorization"))
		if tokenString != "" {
			claims, err := s.authManager.ValidateJWT(tokenString)
			if err == nil {
				c.Set("user_id", claims.UserID)
				c.Set("session_id", claims.SessionID)
				c.Set("role", claims.Role)
				c.Next()
				return
			}
		}

		// Fall back to header/query auth for backwards compatibility
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			userID = c.Query("user_id")
		}
		if userID == "" {
			userID = s.config.UserID // Use default from config
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

// Helper functions

func generateSessionID() (string, error) {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func extractTokenFromHeader(authHeader string) string {
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}

func (s *Server) isPublicEndpoint(path string) bool {
	publicPaths := []string{
		"/health",
		"/",
		"/docs",
		"/openapi.json",
		"/auth/login",
		"/auth/refresh",
	}

	for _, publicPath := range publicPaths {
		if path == publicPath || strings.HasPrefix(path, publicPath+"/") {
			return true
		}
	}

	return false
}