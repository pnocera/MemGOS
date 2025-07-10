package users

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthService_HashPassword(t *testing.T) {
	config := DefaultConfig()
	config.JWTSecret = "test-secret"
	
	repo := setupTestRepository(t)
	defer teardownTestRepository(t, repo)
	
	authService := NewAuthService(config, repo)

	password := "TestPassword123!"
	hash, err := authService.HashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
}

func TestAuthService_VerifyPassword(t *testing.T) {
	config := DefaultConfig()
	config.JWTSecret = "test-secret"
	
	repo := setupTestRepository(t)
	defer teardownTestRepository(t, repo)
	
	authService := NewAuthService(config, repo)

	password := "TestPassword123!"
	hash, err := authService.HashPassword(password)
	require.NoError(t, err)

	// Correct password should verify
	assert.True(t, authService.VerifyPassword(password, hash))

	// Wrong password should not verify
	assert.False(t, authService.VerifyPassword("WrongPassword", hash))
}

func TestAuthService_ValidatePassword(t *testing.T) {
	config := DefaultConfig()
	config.JWTSecret = "test-secret"
	config.PasswordPolicy = PasswordPolicy{
		MinLength:        8,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireNumbers:   true,
		RequireSymbols:   true,
	}
	
	repo := setupTestRepository(t)
	defer teardownTestRepository(t, repo)
	
	authService := NewAuthService(config, repo)

	tests := []struct {
		password string
		valid    bool
		name     string
	}{
		{"Test123!", true, "valid password"},
		{"test123!", false, "no uppercase"},
		{"TEST123!", false, "no lowercase"},
		{"TestABC!", false, "no numbers"},
		{"Test123", false, "no symbols"},
		{"Test1!", false, "too short"},
		{"ValidPassword123!", true, "valid long password"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := authService.ValidatePassword(tt.password)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestAuthService_GenerateAccessToken(t *testing.T) {
	config := DefaultConfig()
	config.JWTSecret = "test-secret"
	config.JWTExpirationTime = time.Hour
	
	repo := setupTestRepository(t)
	defer teardownTestRepository(t, repo)
	
	authService := NewAuthService(config, repo)

	user := &User{
		UserID:   "test-user-id",
		UserName: "testuser",
		Role:     RoleUser,
	}

	token, err := authService.GenerateAccessToken(user)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify token can be validated
	validatedUser, err := authService.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, user.UserID, validatedUser.UserID)
	assert.Equal(t, user.UserName, validatedUser.UserName)
	assert.Equal(t, user.Role, validatedUser.Role)
}

func TestAuthService_AuthenticateUser(t *testing.T) {
	config := DefaultConfig()
	config.JWTSecret = "test-secret"
	
	repo := setupTestRepository(t)
	defer teardownTestRepository(t, repo)
	
	authService := NewAuthService(config, repo)

	// Create a test user
	password := "TestPassword123!"
	hashedPassword, err := authService.HashPassword(password)
	require.NoError(t, err)

	user := &User{
		UserID:   "test-user-id",
		UserName: "testuser",
		Email:    "test@example.com",
		Password: hashedPassword,
		Role:     RoleUser,
		IsActive: true,
	}

	createdUser, err := repo.CreateUser(user)
	require.NoError(t, err)

	// Test successful authentication
	credentials := LoginCredentials{
		Username: "testuser",
		Password: password,
	}

	authResponse, err := authService.AuthenticateUser(credentials)
	require.NoError(t, err)
	assert.Equal(t, createdUser.UserID, authResponse.User.UserID)
	assert.NotEmpty(t, authResponse.AccessToken)
	assert.NotEmpty(t, authResponse.RefreshToken)
	assert.Equal(t, "Bearer", authResponse.TokenType)
	assert.Empty(t, authResponse.User.Password) // Should be empty in response

	// Test invalid username
	credentials.Username = "nonexistent"
	_, err = authService.AuthenticateUser(credentials)
	assert.Error(t, err)

	// Test invalid password
	credentials.Username = "testuser"
	credentials.Password = "wrongpassword"
	_, err = authService.AuthenticateUser(credentials)
	assert.Error(t, err)
}

func TestAuthService_RefreshToken(t *testing.T) {
	config := DefaultConfig()
	config.JWTSecret = "test-secret"
	config.RefreshTokenExpiry = time.Hour
	
	repo := setupTestRepository(t)
	defer teardownTestRepository(t, repo)
	
	authService := NewAuthService(config, repo)

	// Create a test user
	user := &User{
		UserID:   "test-user-id",
		UserName: "testuser",
		Role:     RoleUser,
		IsActive: true,
	}

	_, err := repo.CreateUser(user)
	require.NoError(t, err)

	// Generate refresh token
	refreshToken, err := authService.GenerateRefreshToken(user)
	require.NoError(t, err)

	// Create session (this would normally be done by AuthenticateUser)
	session := &Session{
		UserID:    user.UserID,
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(config.RefreshTokenExpiry),
		IsActive:  true,
	}

	_, err = repo.CreateSession(session)
	require.NoError(t, err)

	// Test refresh token
	authResponse, err := authService.RefreshToken(refreshToken)
	require.NoError(t, err)
	assert.Equal(t, user.UserID, authResponse.User.UserID)
	assert.NotEmpty(t, authResponse.AccessToken)
	assert.Equal(t, "Bearer", authResponse.TokenType)

	// Test with invalid refresh token
	_, err = authService.RefreshToken("invalid-token")
	assert.Error(t, err)
}

func TestAuthService_ChangePassword(t *testing.T) {
	config := DefaultConfig()
	config.JWTSecret = "test-secret"
	
	repo := setupTestRepository(t)
	defer teardownTestRepository(t, repo)
	
	authService := NewAuthService(config, repo)

	// Create a test user
	oldPassword := "OldPassword123!"
	hashedPassword, err := authService.HashPassword(oldPassword)
	require.NoError(t, err)

	user := &User{
		UserID:   "test-user-id",
		UserName: "testuser",
		Password: hashedPassword,
		Role:     RoleUser,
		IsActive: true,
	}

	_, err = repo.CreateUser(user)
	require.NoError(t, err)

	// Change password
	newPassword := "NewPassword123!"
	err = authService.ChangePassword(user.UserID, oldPassword, newPassword)
	require.NoError(t, err)

	// Verify old password no longer works
	updatedUser, err := repo.GetUser(user.UserID)
	require.NoError(t, err)
	assert.False(t, authService.VerifyPassword(oldPassword, updatedUser.Password))

	// Verify new password works
	assert.True(t, authService.VerifyPassword(newPassword, updatedUser.Password))

	// Test with wrong old password
	err = authService.ChangePassword(user.UserID, "wrongpassword", "AnotherPassword123!")
	assert.Error(t, err)
}

func TestAuthService_ResetPassword(t *testing.T) {
	config := DefaultConfig()
	config.JWTSecret = "test-secret"
	
	repo := setupTestRepository(t)
	defer teardownTestRepository(t, repo)
	
	authService := NewAuthService(config, repo)

	// Create a test user
	oldPassword := "OldPassword123!"
	hashedPassword, err := authService.HashPassword(oldPassword)
	require.NoError(t, err)

	user := &User{
		UserID:   "test-user-id",
		UserName: "testuser",
		Password: hashedPassword,
		Role:     RoleUser,
		IsActive: true,
	}

	_, err = repo.CreateUser(user)
	require.NoError(t, err)

	// Reset password (admin function)
	newPassword := "ResetPassword123!"
	err = authService.ResetPassword(user.UserID, newPassword)
	require.NoError(t, err)

	// Verify old password no longer works
	updatedUser, err := repo.GetUser(user.UserID)
	require.NoError(t, err)
	assert.False(t, authService.VerifyPassword(oldPassword, updatedUser.Password))

	// Verify new password works
	assert.True(t, authService.VerifyPassword(newPassword, updatedUser.Password))

	// Test with non-existent user
	err = authService.ResetPassword("nonexistent", "SomePassword123!")
	assert.Error(t, err)
}

func TestAuthService_Logout(t *testing.T) {
	config := DefaultConfig()
	config.JWTSecret = "test-secret"
	
	repo := setupTestRepository(t)
	defer teardownTestRepository(t, repo)
	
	authService := NewAuthService(config, repo)

	// Create a session
	session := &Session{
		SessionID: "test-session-id",
		UserID:    "test-user-id",
		Token:     "test-token",
		ExpiresAt: time.Now().Add(time.Hour),
		IsActive:  true,
	}

	_, err := repo.CreateSession(session)
	require.NoError(t, err)

	// Logout
	err = authService.Logout("test-user-id", "test-session-id")
	require.NoError(t, err)

	// Verify session is invalidated
	updatedSession, err := repo.GetSession("test-session-id")
	require.NoError(t, err)
	assert.False(t, updatedSession.IsActive)
}

func TestAuthService_LogoutAll(t *testing.T) {
	config := DefaultConfig()
	config.JWTSecret = "test-secret"
	
	repo := setupTestRepository(t)
	defer teardownTestRepository(t, repo)
	
	authService := NewAuthService(config, repo)

	userID := "test-user-id"

	// Create multiple sessions
	for i := 0; i < 3; i++ {
		session := &Session{
			UserID:    userID,
			Token:     "test-token-" + string(rune('1'+i)),
			ExpiresAt: time.Now().Add(time.Hour),
			IsActive:  true,
		}

		_, err := repo.CreateSession(session)
		require.NoError(t, err)
	}

	// Logout all sessions
	err := authService.LogoutAll(userID)
	require.NoError(t, err)

	// Note: We would need a method to get all user sessions to verify this
	// For now, we just verify the method doesn't return an error
}

// Helper functions for auth tests

func setupTestRepository(t *testing.T) *Repository {
	config := DefaultConfig()
	config.DatabasePath = ":memory:" // Use in-memory SQLite for tests
	config.EnableAuditLogging = false

	repo, err := NewRepository(config)
	require.NoError(t, err)

	return repo
}

func teardownTestRepository(t *testing.T, repo *Repository) {
	err := repo.Close()
	assert.NoError(t, err)
}