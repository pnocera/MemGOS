package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/memtensor/memgos/pkg/types"
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
func (am *AuthManager) CreateSession(ctx context.Context, userID string) (string, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}

	now := time.Now()
	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		CreatedAt: now,
		LastSeen:  now,
		ExpiresAt: now.Add(am.expiry),
		Active:    true,
		Metadata:  make(map[string]interface{}),
	}

	am.mu.Lock()
	am.sessions[sessionID] = session
	am.mu.Unlock()

	return sessionID, nil
}

// GetSession retrieves a session by ID
func (am *AuthManager) GetSession(ctx context.Context, sessionID string) (*types.UserSession, error) {
	am.mu.RLock()
	session, exists := am.sessions[sessionID]
	am.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	if !session.Active || time.Now().After(session.ExpiresAt) {
		am.InvalidateSession(ctx, sessionID)
		return nil, fmt.Errorf("session expired or inactive")
	}

	// Update last seen
	am.mu.Lock()
	session.LastSeen = time.Now()
	am.mu.Unlock()

	// Convert to types.UserSession
	userSession := &types.UserSession{
		SessionID: session.ID,
		UserID:    session.UserID,
		CreatedAt: session.CreatedAt,
		ExpiresAt: session.ExpiresAt,
		IsActive:  session.Active,
	}

	return userSession, nil
}

// InvalidateSession invalidates a session
func (am *AuthManager) InvalidateSession(ctx context.Context, sessionID string) error {
	am.mu.Lock()
	if session, exists := am.sessions[sessionID]; exists {
		session.Active = false
	}
	am.mu.Unlock()
	return nil
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

// GenerateJWT generates a JWT token for a user
func (am *AuthManager) GenerateJWT(ctx context.Context, userID string, expirationMinutes int) (string, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(expirationMinutes) * time.Minute)
	
	// Generate a session ID for the JWT
	sessionID, err := generateSessionID()
	if err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}

	claims := &Claims{
		UserID:    userID,
		SessionID: sessionID,
		Role:      "user", // Default role
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "memgos-api",
			Subject:   userID,
			ID:        sessionID,
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
func (am *AuthManager) ValidateJWT(ctx context.Context, tokenString string) (*types.JWTClaims, error) {
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
		_, err := am.GetSession(ctx, claims.SessionID)
		if err != nil {
			return nil, fmt.Errorf("session invalid: %w", err)
		}
		
		// Convert to types.JWTClaims
		jwtClaims := &types.JWTClaims{
			UserID:    claims.UserID,
			Role:      types.UserRole(claims.Role),
			SessionID: claims.SessionID,
			IssuedAt:  claims.IssuedAt.Time,
			ExpiresAt: claims.ExpiresAt.Time,
		}
		
		return jwtClaims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// Login handles user login and creates a session
// @Summary User Login
// @Description Authenticate user and create a session with JWT token
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body AuthRequest true "Login credentials"
// @Success 200 {object} BaseResponse[AuthResponse]
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/login [post]
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
	sessionID, err := s.authManager.CreateSession(c.Request.Context(), req.UserID)
	if err != nil {
		s.handleError(c, "Failed to create session", err)
		return
	}

	// Generate JWT
	token, err := s.authManager.GenerateJWT(c.Request.Context(), req.UserID, 60) // 60 minutes expiration
	if err != nil {
		s.handleError(c, "Failed to generate token", err)
		return
	}

	response := AuthResponse{
		Token:     token,
		SessionID: sessionID,
		UserID:    req.UserID,
		ExpiresAt: time.Now().Add(60 * time.Minute),
	}

	c.JSON(http.StatusOK, BaseResponse[AuthResponse]{
		Code:    http.StatusOK,
		Message: "Login successful",
		Data:    &response,
	})
}

// Logout handles user logout
// @Summary User Logout
// @Description Logout user and invalidate session
// @Tags authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SimpleResponse
// @Router /auth/logout [post]
func (s *Server) logout(c *gin.Context) {
	sessionID := c.GetString("session_id")
	if sessionID != "" {
		s.authManager.InvalidateSession(c.Request.Context(), sessionID)
	}

	c.JSON(http.StatusOK, SimpleResponse{
		Code:    http.StatusOK,
		Message: "Logout successful",
	})
}

// RefreshToken handles token refresh
// @Summary Refresh JWT Token
// @Description Refresh an existing JWT token to extend session
// @Tags authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} BaseResponse[AuthResponse]
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/refresh [post]
func (s *Server) refreshToken(c *gin.Context) {
	tokenString := extractTokenFromHeader(c.GetHeader("Authorization"))
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "No token provided",
		})
		return
	}

	claims, err := s.authManager.ValidateJWT(c.Request.Context(), tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "Invalid token",
			Error:   err.Error(),
		})
		return
	}

	// Get current session
	session, err := s.authManager.GetSession(c.Request.Context(), claims.SessionID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "Session invalid",
			Error:   err.Error(),
		})
		return
	}

	// Generate new token
	newToken, err := s.authManager.GenerateJWT(c.Request.Context(), session.UserID, 60) // 60 minutes expiration
	if err != nil {
		s.handleError(c, "Failed to generate new token", err)
		return
	}

	response := AuthResponse{
		Token:     newToken,
		SessionID: session.SessionID,
		UserID:    session.UserID,
		ExpiresAt: time.Now().Add(60 * time.Minute),
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
			claims, err := s.authManager.ValidateJWT(c.Request.Context(), tokenString)
			if err == nil {
				c.Set("user_id", claims.UserID)
				c.Set("session_id", claims.SessionID)
				c.Set("role", string(claims.Role))
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
		"/swagger",
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

// GenerateToken generates a simple token for the user (used by enhanced server)
func (am *AuthManager) GenerateToken(username string) string {
	// Use the existing JWT generation but with a simple interface
	ctx := context.Background()
	token, err := am.GenerateJWT(ctx, username, 60) // 60 minutes expiration
	if err != nil {
		// Fall back to a simple placeholder token
		return "placeholder_token_" + username
	}
	return token
}