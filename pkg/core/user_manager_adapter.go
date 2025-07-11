package core

import (
	"context"
	"fmt"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
	"github.com/memtensor/memgos/pkg/users"
)

// UserManagerAdapter adapts the users.Manager to implement interfaces.UserManager
type UserManagerAdapter struct {
	manager *users.Manager
}

// NewUserManagerAdapter creates a new UserManagerAdapter
func NewUserManagerAdapter(manager *users.Manager) interfaces.UserManager {
	return &UserManagerAdapter{
		manager: manager,
	}
}

// CreateUser creates a new user
func (u *UserManagerAdapter) CreateUser(ctx context.Context, userName string, role types.UserRole, userID string) (string, error) {
	// Create user using the users.Manager
	params := users.CreateUserParams{
		UserID:   userID,
		Username: userName,
		Role:     users.UserRole(role), // Convert types.UserRole to users.UserRole
		Password: "defaultpassword",    // Default password, should be changed by user
	}
	
	createdUser, err := u.manager.CreateUser(params)
	if err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}
	
	return createdUser.UserID, nil
}

// GetUser retrieves a user by ID
func (u *UserManagerAdapter) GetUser(ctx context.Context, userID string) (*types.User, error) {
	user, err := u.manager.GetUser(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	
	// Convert users.User to types.User
	return &types.User{
		ID:        user.UserID,
		Name:      user.UserName,
		Role:      types.UserRole(user.Role),
		CreatedAt: user.CreatedAt,
	}, nil
}

// ListUsers lists all users
func (u *UserManagerAdapter) ListUsers(ctx context.Context) ([]*types.User, error) {
	// Use the ListUsers method which returns paginated results
	paginatedResponse, err := u.manager.ListUsers(100, 0) // Get first 100 users
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	
	result := make([]*types.User, len(paginatedResponse.Users))
	for i, user := range paginatedResponse.Users {
		result[i] = &types.User{
			ID:        user.UserID,
			Name:      user.UserName,
			Role:      types.UserRole(user.Role),
			CreatedAt: user.CreatedAt,
		}
	}
	
	return result, nil
}

// ValidateUser validates if a user exists and is active
func (u *UserManagerAdapter) ValidateUser(ctx context.Context, userID string) (bool, error) {
	user, err := u.manager.GetUser(userID)
	if err != nil {
		return false, err
	}
	
	return user != nil && user.IsActive, nil
}

// CreateCube creates a new memory cube
func (u *UserManagerAdapter) CreateCube(ctx context.Context, cubeName, ownerID, cubePath, cubeID string) (string, error) {
	params := users.CreateCubeParams{
		CubeID:   cubeID,
		CubeName: cubeName,
		CubePath: cubePath,
		OwnerID:  ownerID,
	}
	
	cube, err := u.manager.CreateCube(params)
	if err != nil {
		return "", fmt.Errorf("failed to create cube: %w", err)
	}
	
	return cube.CubeID, nil
}

// GetCube retrieves a cube by ID
func (u *UserManagerAdapter) GetCube(ctx context.Context, cubeID string) (*types.MemCube, error) {
	cube, err := u.manager.GetCube(cubeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cube: %w", err)
	}
	
	if cube == nil {
		return nil, fmt.Errorf("cube not found")
	}
	
	// Convert users.Cube to types.MemCube
	return &types.MemCube{
		ID:      cube.CubeID,
		Name:    cube.CubeName,
		Path:    cube.CubePath,
		OwnerID: cube.OwnerID,
	}, nil
}

// GetUserCubes retrieves all cubes accessible by a user
func (u *UserManagerAdapter) GetUserCubes(ctx context.Context, userID string) ([]*types.MemCube, error) {
	cubes, err := u.manager.GetUserCubes(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user cubes: %w", err)
	}
	
	result := make([]*types.MemCube, len(cubes))
	for i, cube := range cubes {
		result[i] = &types.MemCube{
			ID:      cube.CubeID,
			Name:    cube.CubeName,
			Path:    cube.CubePath,
			OwnerID: cube.OwnerID,
		}
	}
	
	return result, nil
}

// ValidateUserCubeAccess validates if a user has access to a cube
func (u *UserManagerAdapter) ValidateUserCubeAccess(ctx context.Context, userID, cubeID string) (bool, error) {
	// Check if cube exists and user has access
	cube, err := u.manager.GetCube(cubeID)
	if err != nil || cube == nil {
		return false, err
	}
	
	// Check if user is owner
	if cube.OwnerID == userID {
		return true, nil
	}
	
	// Check if cube is shared with user
	cubes, err := u.manager.GetUserCubes(userID)
	if err != nil {
		return false, err
	}
	
	for _, userCube := range cubes {
		if userCube.CubeID == cubeID {
			return true, nil
		}
	}
	
	return false, nil
}

// ShareCube shares a cube with another user
func (u *UserManagerAdapter) ShareCube(ctx context.Context, cubeID, fromUserID, toUserID string) error {
	// Use AddUserToCube method as an alternative to ShareCube
	err := u.manager.AddUserToCube(toUserID, cubeID)
	if err != nil {
		return fmt.Errorf("failed to share cube: %w", err)
	}
	
	return nil
}

// CreateSession creates a new user session
func (u *UserManagerAdapter) CreateSession(ctx context.Context, userID string) (string, error) {
	// TODO: Implement session creation once available in users.Manager
	// For now, return a placeholder session ID
	return fmt.Sprintf("session-%s-%d", userID, time.Now().Unix()), nil
}

// InvalidateSession invalidates a user session
func (u *UserManagerAdapter) InvalidateSession(ctx context.Context, sessionID string) error {
	// TODO: Implement session invalidation once available in users.Manager
	// For now, this is a no-op
	return nil
}

// GetSession retrieves session information
func (u *UserManagerAdapter) GetSession(ctx context.Context, sessionID string) (*types.UserSession, error) {
	// TODO: Implement session retrieval once available in users.Manager
	// For now, return a placeholder session
	return &types.UserSession{
		SessionID: sessionID,
		UserID:    "placeholder-user",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IsActive:  true,
	}, nil
}

// ValidateJWT validates a JWT token and returns claims
func (u *UserManagerAdapter) ValidateJWT(ctx context.Context, token string) (*types.JWTClaims, error) {
	// TODO: Implement JWT validation once available in users.Manager
	// For now, return placeholder claims
	return &types.JWTClaims{
		UserID:    "placeholder-user",
		Role:      types.UserRole("user"),
		SessionID: "placeholder-session",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil
}

// GenerateJWT generates a JWT token for a user
func (u *UserManagerAdapter) GenerateJWT(ctx context.Context, userID string, expirationMinutes int) (string, error) {
	// TODO: Implement JWT generation once available in users.Manager
	// For now, return a placeholder token
	return fmt.Sprintf("jwt-token-%s-%d", userID, time.Now().Unix()), nil
}

// API Token management methods

// CreateAPIToken creates a new API token for a user
func (u *UserManagerAdapter) CreateAPIToken(userID string, params interface{}) (interface{}, error) {
	if p, ok := params.(users.CreateAPITokenParams); ok {
		return u.manager.CreateAPIToken(userID, p)
	}
	return nil, fmt.Errorf("invalid parameters type for CreateAPIToken")
}

// GetAPITokens returns all API tokens for a user
func (u *UserManagerAdapter) GetAPITokens(userID string) (interface{}, error) {
	return u.manager.GetAPITokens(userID)
}

// GetAPIToken retrieves a specific API token by ID
func (u *UserManagerAdapter) GetAPIToken(userID, tokenID string) (interface{}, error) {
	return u.manager.GetAPIToken(userID, tokenID)
}

// UpdateAPIToken updates an API token's metadata
func (u *UserManagerAdapter) UpdateAPIToken(userID, tokenID string, params interface{}) (interface{}, error) {
	if p, ok := params.(users.UpdateAPITokenParams); ok {
		return u.manager.UpdateAPIToken(userID, tokenID, p)
	}
	return nil, fmt.Errorf("invalid parameters type for UpdateAPIToken")
}

// RevokeAPIToken revokes an API token
func (u *UserManagerAdapter) RevokeAPIToken(userID, tokenID string) error {
	return u.manager.RevokeAPIToken(userID, tokenID)
}

// ValidateAPIToken validates an API token and returns the associated user
func (u *UserManagerAdapter) ValidateAPIToken(token string) (interface{}, interface{}, error) {
	return u.manager.ValidateAPIToken(token)
}