package users

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService provides authentication functionality
type AuthService struct {
	config     *Config
	repository *Repository
}

// NewAuthService creates a new authentication service
func NewAuthService(config *Config, repository *Repository) *AuthService {
	return &AuthService{
		config:     config,
		repository: repository,
	}
}

// LoginCredentials represents user login credentials
type LoginCredentials struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	User         *User  `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresAt    int64  `json:"expires_at"`
	TokenType    string `json:"token_type"`
}

// TokenClaims represents JWT token claims
type TokenClaims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Role     UserRole `json:"role"`
	jwt.RegisteredClaims
}

// RefreshTokenClaims represents refresh token claims
type RefreshTokenClaims struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}

// AuthenticateUser authenticates a user with username and password
func (as *AuthService) AuthenticateUser(credentials LoginCredentials) (*AuthResponse, error) {
	// Get user by username
	user, err := as.repository.GetUserByName(credentials.Username)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	if user == nil {
		return nil, NewAuthenticationError("invalid username or password")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, NewAuthenticationError("user account is deactivated")
	}

	// Verify password
	if !as.VerifyPassword(credentials.Password, user.Password) {
		return nil, NewAuthenticationError("invalid username or password")
	}

	// Generate tokens
	accessToken, err := as.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := as.GenerateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create session
	session := &Session{
		UserID:    user.UserID,
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(as.config.RefreshTokenExpiry),
		IsActive:  true,
	}

	_, err = as.repository.CreateSession(session)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Remove password from response
	userResponse := *user
	userResponse.Password = ""

	return &AuthResponse{
		User:         &userResponse,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(as.config.JWTExpirationTime).Unix(),
		TokenType:    "Bearer",
	}, nil
}

// ValidateToken validates a JWT access token and returns the user
func (as *AuthService) ValidateToken(tokenString string) (*User, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(as.config.JWTSecret), nil
	})

	if err != nil {
		return nil, NewAuthenticationError("invalid token")
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, NewAuthenticationError("invalid token claims")
	}

	// Get user from database
	user, err := as.repository.GetUser(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil || !user.IsActive {
		return nil, NewAuthenticationError("user not found or inactive")
	}

	return user, nil
}

// RefreshToken generates a new access token using a refresh token
func (as *AuthService) RefreshToken(refreshTokenString string) (*AuthResponse, error) {
	token, err := jwt.ParseWithClaims(refreshTokenString, &RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(as.config.JWTSecret), nil
	})

	if err != nil {
		return nil, NewAuthenticationError("invalid refresh token")
	}

	claims, ok := token.Claims.(*RefreshTokenClaims)
	if !ok || !token.Valid {
		return nil, NewAuthenticationError("invalid refresh token claims")
	}

	// Validate session
	session, err := as.repository.GetSession(claims.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session == nil || !session.IsActive || session.IsExpired() {
		return nil, NewAuthenticationError("invalid or expired session")
	}

	// Get user
	user, err := as.repository.GetUser(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil || !user.IsActive {
		return nil, NewAuthenticationError("user not found or inactive")
	}

	// Generate new access token
	accessToken, err := as.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Remove password from response
	userResponse := *user
	userResponse.Password = ""

	return &AuthResponse{
		User:        &userResponse,
		AccessToken: accessToken,
		ExpiresAt:   time.Now().Add(as.config.JWTExpirationTime).Unix(),
		TokenType:   "Bearer",
	}, nil
}

// Logout invalidates a user session
func (as *AuthService) Logout(userID, sessionID string) error {
	return as.repository.InvalidateSession(sessionID)
}

// LogoutAll invalidates all sessions for a user
func (as *AuthService) LogoutAll(userID string) error {
	return as.repository.InvalidateAllUserSessions(userID)
}

// GenerateAccessToken generates a JWT access token for a user
func (as *AuthService) GenerateAccessToken(user *User) (string, error) {
	now := time.Now()
	claims := &TokenClaims{
		UserID:   user.UserID,
		Username: user.UserName,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(as.config.JWTExpirationTime)),
			Subject:   user.UserID,
			Issuer:    "memgos",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(as.config.JWTSecret))
}

// GenerateRefreshToken generates a refresh token for a user
func (as *AuthService) GenerateRefreshToken(user *User) (string, error) {
	sessionID := generateSecureRandomString(32)
	now := time.Now()

	claims := &RefreshTokenClaims{
		UserID:    user.UserID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(as.config.RefreshTokenExpiry)),
			Subject:   user.UserID,
			Issuer:    "memgos",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(as.config.JWTSecret))
}

// HashPassword hashes a password using bcrypt
func (as *AuthService) HashPassword(password string) (string, error) {
	// Validate password against policy
	if err := as.ValidatePassword(password); err != nil {
		return "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword verifies a password against its hash
func (as *AuthService) VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidatePassword validates a password against the configured policy
func (as *AuthService) ValidatePassword(password string) error {
	policy := as.config.PasswordPolicy

	if len(password) < policy.MinLength {
		return NewValidationError(fmt.Sprintf("password must be at least %d characters long", policy.MinLength))
	}

	if policy.RequireUppercase && !containsUppercase(password) {
		return NewValidationError("password must contain at least one uppercase letter")
	}

	if policy.RequireLowercase && !containsLowercase(password) {
		return NewValidationError("password must contain at least one lowercase letter")
	}

	if policy.RequireNumbers && !containsNumber(password) {
		return NewValidationError("password must contain at least one number")
	}

	if policy.RequireSymbols && !containsSymbol(password) {
		return NewValidationError("password must contain at least one special character")
	}

	return nil
}

// ChangePassword changes a user's password
func (as *AuthService) ChangePassword(userID, oldPassword, newPassword string) error {
	user, err := as.repository.GetUser(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return NewValidationError("user not found")
	}

	// Verify old password
	if !as.VerifyPassword(oldPassword, user.Password) {
		return NewAuthenticationError("invalid current password")
	}

	// Hash new password
	hashedPassword, err := as.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update password
	user.Password = hashedPassword
	return as.repository.UpdateUser(user)
}

// ResetPassword resets a user's password (admin function)
func (as *AuthService) ResetPassword(userID, newPassword string) error {
	user, err := as.repository.GetUser(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return NewValidationError("user not found")
	}

	// Hash new password
	hashedPassword, err := as.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update password
	user.Password = hashedPassword
	return as.repository.UpdateUser(user)
}

// generateSecureRandomString generates a cryptographically secure random string
func generateSecureRandomString(length int) string {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		// Fallback to timestamp-based string if crypto rand fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length]
}

// Helper functions for password validation
func containsUppercase(s string) bool {
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}
	return false
}

func containsLowercase(s string) bool {
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			return true
		}
	}
	return false
}

func containsNumber(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func containsSymbol(s string) bool {
	symbols := "!@#$%^&*()_+-=[]{}|;:,.<>?"
	for _, r := range s {
		for _, symbol := range symbols {
			if r == symbol {
				return true
			}
		}
	}
	return false
}

// AuthenticationError represents an authentication error
type AuthenticationError struct {
	Message string
}

func (e AuthenticationError) Error() string {
	return e.Message
}

// NewAuthenticationError creates a new authentication error
func NewAuthenticationError(message string) error {
	return AuthenticationError{Message: message}
}