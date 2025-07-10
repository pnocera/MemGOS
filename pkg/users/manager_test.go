package users

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_CreateUser(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(t, manager)

	params := CreateUserParams{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "TestPass123!",
		Role:     RoleUser,
	}

	user, err := manager.CreateUser(params)
	require.NoError(t, err)
	assert.NotEmpty(t, user.UserID)
	assert.Equal(t, "testuser", user.UserName)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, RoleUser, user.Role)
	assert.True(t, user.IsActive)
	assert.Empty(t, user.Password) // Should be empty in response
}

func TestManager_CreateUser_DuplicateUsername(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(t, manager)

	params := CreateUserParams{
		Username: "testuser",
		Email:    "test1@example.com",
		Password: "TestPass123!",
		Role:     RoleUser,
	}

	// Create first user
	_, err := manager.CreateUser(params)
	require.NoError(t, err)

	// Try to create duplicate
	params.Email = "test2@example.com"
	_, err = manager.CreateUser(params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestManager_CreateUser_DuplicateEmail(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(t, manager)

	params1 := CreateUserParams{
		Username: "testuser1",
		Email:    "test@example.com",
		Password: "TestPass123!",
		Role:     RoleUser,
	}

	params2 := CreateUserParams{
		Username: "testuser2",
		Email:    "test@example.com", // Same email
		Password: "TestPass123!",
		Role:     RoleUser,
	}

	// Create first user
	_, err := manager.CreateUser(params1)
	require.NoError(t, err)

	// Try to create with duplicate email
	_, err = manager.CreateUser(params2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestManager_GetUser(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(t, manager)

	// Create a user first
	params := CreateUserParams{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "TestPass123!",
		Role:     RoleUser,
	}

	createdUser, err := manager.CreateUser(params)
	require.NoError(t, err)

	// Get the user
	user, err := manager.GetUser(createdUser.UserID)
	require.NoError(t, err)
	assert.Equal(t, createdUser.UserID, user.UserID)
	assert.Equal(t, "testuser", user.UserName)
	assert.Empty(t, user.Password) // Should be empty in response
}

func TestManager_GetUserByName(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(t, manager)

	// Create a user first
	params := CreateUserParams{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "TestPass123!",
		Role:     RoleUser,
	}

	createdUser, err := manager.CreateUser(params)
	require.NoError(t, err)

	// Get the user by name
	user, err := manager.GetUserByName("testuser")
	require.NoError(t, err)
	assert.Equal(t, createdUser.UserID, user.UserID)
	assert.Equal(t, "testuser", user.UserName)
}

func TestManager_UpdateUser(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(t, manager)

	// Create a user first
	params := CreateUserParams{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "TestPass123!",
		Role:     RoleUser,
	}

	createdUser, err := manager.CreateUser(params)
	require.NoError(t, err)

	// Update the user
	updateParams := UpdateUserParams{
		Username: "updateduser",
		Email:    "updated@example.com",
		Role:     RoleAdmin,
	}

	updatedUser, err := manager.UpdateUser(createdUser.UserID, updateParams)
	require.NoError(t, err)
	assert.Equal(t, "updateduser", updatedUser.UserName)
	assert.Equal(t, "updated@example.com", updatedUser.Email)
	assert.Equal(t, RoleAdmin, updatedUser.Role)
}

func TestManager_DeleteUser(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(t, manager)

	// Create a user first
	params := CreateUserParams{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "TestPass123!",
		Role:     RoleUser,
	}

	createdUser, err := manager.CreateUser(params)
	require.NoError(t, err)

	// Delete the user
	err = manager.DeleteUser(createdUser.UserID)
	require.NoError(t, err)

	// Verify user is deactivated
	valid, err := manager.ValidateUser(createdUser.UserID)
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestManager_DeleteUser_RootUser(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(t, manager)

	// Try to delete root user
	err := manager.DeleteUser("root")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete root user")

	// Verify root user is still active
	valid, err := manager.ValidateUser("root")
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestManager_ListUsers(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(t, manager)

	// Create multiple users
	for i := 0; i < 3; i++ {
		params := CreateUserParams{
			Username: "testuser" + string(rune('1'+i)),
			Email:    "test" + string(rune('1'+i)) + "@example.com",
			Password: "TestPass123!",
			Role:     RoleUser,
		}
		_, err := manager.CreateUser(params)
		require.NoError(t, err)
	}

	// List users
	response, err := manager.ListUsers(10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(response.Users), 4) // 3 created + 1 root user
	assert.GreaterOrEqual(t, response.Total, int64(4))

	// Check that passwords are not returned
	for _, user := range response.Users {
		assert.Empty(t, user.Password)
	}
}

func TestManager_CreateCube(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(t, manager)

	// Create a user first
	userParams := CreateUserParams{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "TestPass123!",
		Role:     RoleUser,
	}

	user, err := manager.CreateUser(userParams)
	require.NoError(t, err)

	// Create a cube
	cubeParams := CreateCubeParams{
		CubeName: "testcube",
		CubePath: "/path/to/cube",
		OwnerID:  user.UserID,
	}

	cube, err := manager.CreateCube(cubeParams)
	require.NoError(t, err)
	assert.NotEmpty(t, cube.CubeID)
	assert.Equal(t, "testcube", cube.CubeName)
	assert.Equal(t, "/path/to/cube", cube.CubePath)
	assert.Equal(t, user.UserID, cube.OwnerID)
	assert.True(t, cube.IsActive)
}

func TestManager_GetUserCubes(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(t, manager)

	// Create a user
	userParams := CreateUserParams{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "TestPass123!",
		Role:     RoleUser,
	}

	user, err := manager.CreateUser(userParams)
	require.NoError(t, err)

	// Create multiple cubes
	for i := 0; i < 3; i++ {
		cubeParams := CreateCubeParams{
			CubeName: "testcube" + string(rune('1'+i)),
			OwnerID:  user.UserID,
		}
		_, err := manager.CreateCube(cubeParams)
		require.NoError(t, err)
	}

	// Get user cubes
	cubes, err := manager.GetUserCubes(user.UserID)
	require.NoError(t, err)
	assert.Len(t, cubes, 3)
}

func TestManager_ValidateUserCubeAccess(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(t, manager)

	// Create users
	user1Params := CreateUserParams{
		Username: "user1",
		Email:    "user1@example.com",
		Password: "TestPass123!",
		Role:     RoleUser,
	}

	user2Params := CreateUserParams{
		Username: "user2",
		Email:    "user2@example.com",
		Password: "TestPass123!",
		Role:     RoleUser,
	}

	user1, err := manager.CreateUser(user1Params)
	require.NoError(t, err)

	user2, err := manager.CreateUser(user2Params)
	require.NoError(t, err)

	// Create cube owned by user1
	cubeParams := CreateCubeParams{
		CubeName: "testcube",
		OwnerID:  user1.UserID,
	}

	cube, err := manager.CreateCube(cubeParams)
	require.NoError(t, err)

	// Owner should have access
	hasAccess, err := manager.ValidateUserCubeAccess(user1.UserID, cube.CubeID)
	require.NoError(t, err)
	assert.True(t, hasAccess)

	// Other user should not have access initially
	hasAccess, err = manager.ValidateUserCubeAccess(user2.UserID, cube.CubeID)
	require.NoError(t, err)
	assert.False(t, hasAccess)

	// Add user2 to cube
	err = manager.AddUserToCube(user2.UserID, cube.CubeID)
	require.NoError(t, err)

	// Now user2 should have access
	hasAccess, err = manager.ValidateUserCubeAccess(user2.UserID, cube.CubeID)
	require.NoError(t, err)
	assert.True(t, hasAccess)
}

func TestManager_Authentication(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(t, manager)

	// Create a user
	userParams := CreateUserParams{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "TestPass123!",
		Role:     RoleUser,
	}

	user, err := manager.CreateUser(userParams)
	require.NoError(t, err)

	// Authenticate user
	credentials := LoginCredentials{
		Username: "testuser",
		Password: "TestPass123!",
	}

	authResponse, err := manager.Authenticate(credentials)
	require.NoError(t, err)
	assert.Equal(t, user.UserID, authResponse.User.UserID)
	assert.NotEmpty(t, authResponse.AccessToken)
	assert.NotEmpty(t, authResponse.RefreshToken)
	assert.Equal(t, "Bearer", authResponse.TokenType)

	// Validate token
	validatedUser, err := manager.ValidateToken(authResponse.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, user.UserID, validatedUser.UserID)
}

func TestManager_Authentication_InvalidCredentials(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(t, manager)

	// Create a user
	userParams := CreateUserParams{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "TestPass123!",
		Role:     RoleUser,
	}

	_, err := manager.CreateUser(userParams)
	require.NoError(t, err)

	// Try to authenticate with wrong password
	credentials := LoginCredentials{
		Username: "testuser",
		Password: "WrongPassword",
	}

	_, err = manager.Authenticate(credentials)
	assert.Error(t, err)
}

func TestManager_ChangePassword(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(t, manager)

	// Create a user
	userParams := CreateUserParams{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "OldPass123!",
		Role:     RoleUser,
	}

	user, err := manager.CreateUser(userParams)
	require.NoError(t, err)

	// Change password
	err = manager.ChangePassword(user.UserID, "OldPass123!", "NewPass123!")
	require.NoError(t, err)

	// Verify old password doesn't work
	credentials := LoginCredentials{
		Username: "testuser",
		Password: "OldPass123!",
	}

	_, err = manager.Authenticate(credentials)
	assert.Error(t, err)

	// Verify new password works
	credentials.Password = "NewPass123!"
	_, err = manager.Authenticate(credentials)
	require.NoError(t, err)
}

func TestManager_CheckPermission(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(t, manager)

	// Create users with different roles
	adminParams := CreateUserParams{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "TestPass123!",
		Role:     RoleAdmin,
	}

	userParams := CreateUserParams{
		Username: "user",
		Email:    "user@example.com",
		Password: "TestPass123!",
		Role:     RoleUser,
	}

	admin, err := manager.CreateUser(adminParams)
	require.NoError(t, err)

	user, err := manager.CreateUser(userParams)
	require.NoError(t, err)

	// Admin should have user management permissions
	result, err := manager.CheckPermission(admin.UserID, "users", "create")
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	// Regular user should not have user management permissions
	result, err = manager.CheckPermission(user.UserID, "users", "create")
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	// Both should have cube read permissions
	result, err = manager.CheckPermission(admin.UserID, "cubes", "read")
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	result, err = manager.CheckPermission(user.UserID, "cubes", "read")
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestManager_HealthCheck(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(t, manager)

	err := manager.HealthCheck()
	assert.NoError(t, err)
}

// Helper functions for testing

func setupTestManager(t *testing.T) *Manager {
	// Create temporary directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_users.db")

	config := DefaultConfig()
	config.DatabasePath = dbPath
	config.JWTSecret = "test-secret-key-for-testing-only"
	config.EnableAuditLogging = false // Disable for cleaner tests

	manager, err := NewManager(config)
	require.NoError(t, err)

	return manager
}

func teardownTestManager(t *testing.T, manager *Manager) {
	err := manager.Close()
	assert.NoError(t, err)
}