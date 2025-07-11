package users

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Manager is the main user management service that coordinates all user operations
type Manager struct {
	config            *Config
	repository        *Repository
	authService       *AuthService
	authzService      *AuthorizationService
	auditEnabled      bool
}

// NewManager creates a new user manager instance
func NewManager(config *Config) (*Manager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize repository
	repository, err := NewRepository(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize repository: %w", err)
	}

	// Initialize services
	authService := NewAuthService(config, repository)
	authzService := NewAuthorizationService(config, repository)

	manager := &Manager{
		config:       config,
		repository:   repository,
		authService:  authService,
		authzService: authzService,
		auditEnabled: config.EnableAuditLogging,
	}

	return manager, nil
}

// User Management Operations

// CreateUser creates a new user with the specified parameters
func (m *Manager) CreateUser(params CreateUserParams) (*User, error) {
	// Validate parameters
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Check if user already exists
	existingUser, err := m.repository.GetUserByName(params.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, fmt.Errorf("user with username '%s' already exists", params.Username)
	}

	// Check email if provided
	if params.Email != "" {
		existingUser, err := m.repository.GetUserByEmail(params.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing email: %w", err)
		}
		if existingUser != nil {
			return nil, fmt.Errorf("user with email '%s' already exists", params.Email)
		}
	}

	// Hash password
	hashedPassword, err := m.authService.HashPassword(params.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &User{
		UserID:   params.UserID,
		UserName: params.Username,
		Email:    params.Email,
		Role:     params.Role,
		Password: hashedPassword,
		IsActive: true,
	}

	if user.UserID == "" {
		user.UserID = uuid.New().String()
	}

	createdUser, err := m.repository.CreateUser(user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create user profile if display name is provided
	if params.DisplayName != "" {
		_ = &UserProfile{
			UserID:      createdUser.UserID,
			DisplayName: params.DisplayName,
		}
		// Note: We would implement CreateUserProfile in repository if needed
	}

	// Log audit event
	if m.auditEnabled {
		m.logAuditEvent("", "user_create", "users", createdUser.UserID, 
			fmt.Sprintf("Created user: %s (%s)", createdUser.UserName, createdUser.Role), true)
	}

	// Remove password from response
	createdUser.Password = ""
	return createdUser, nil
}

// GetUser retrieves a user by ID
func (m *Manager) GetUser(userID string) (*User, error) {
	user, err := m.repository.GetUser(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Remove password from response
	user.Password = ""
	return user, nil
}

// GetUserByName retrieves a user by username
func (m *Manager) GetUserByName(username string) (*User, error) {
	user, err := m.repository.GetUserByName(username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Remove password from response
	user.Password = ""
	return user, nil
}

// UpdateUser updates a user's information
func (m *Manager) UpdateUser(userID string, params UpdateUserParams) (*User, error) {
	// Get existing user
	user, err := m.repository.GetUser(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Update fields
	if params.Username != "" && params.Username != user.UserName {
		// Check if username is available
		existingUser, err := m.repository.GetUserByName(params.Username)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing username: %w", err)
		}
		if existingUser != nil {
			return nil, fmt.Errorf("username '%s' is already taken", params.Username)
		}
		user.UserName = params.Username
	}

	if params.Email != "" && params.Email != user.Email {
		// Check if email is available
		existingUser, err := m.repository.GetUserByEmail(params.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing email: %w", err)
		}
		if existingUser != nil {
			return nil, fmt.Errorf("email '%s' is already taken", params.Email)
		}
		user.Email = params.Email
	}

	if params.Role != "" && params.Role != user.Role {
		// Validate role change
		if !params.Role.IsValid() {
			return nil, fmt.Errorf("invalid role: %s", params.Role)
		}
		user.Role = params.Role
	}

	if params.IsActive != nil {
		user.IsActive = *params.IsActive
	}

	// Update user
	if err := m.repository.UpdateUser(user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Log audit event
	if m.auditEnabled {
		m.logAuditEvent("", "user_update", "users", userID, 
			fmt.Sprintf("Updated user: %s", user.UserName), true)
	}

	// Remove password from response
	user.Password = ""
	return user, nil
}

// DeleteUser soft deletes a user
func (m *Manager) DeleteUser(userID string) error {
	// Get user first to check if it exists
	user, err := m.repository.GetUser(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return fmt.Errorf("user not found")
	}

	if user.Role == RoleRoot {
		return fmt.Errorf("cannot delete root user")
	}

	// Soft delete user
	if err := m.repository.DeleteUser(userID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Invalidate all user sessions
	if err := m.repository.InvalidateAllUserSessions(userID); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to invalidate user sessions: %v\n", err)
	}

	// Log audit event
	if m.auditEnabled {
		m.logAuditEvent("", "user_delete", "users", userID, 
			fmt.Sprintf("Deleted user: %s", user.UserName), true)
	}

	return nil
}

// ListUsers returns a paginated list of users
func (m *Manager) ListUsers(limit, offset int) (*PaginatedUsersResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	users, total, err := m.repository.ListUsers(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Remove passwords from response
	for i := range users {
		users[i].Password = ""
	}

	return &PaginatedUsersResponse{
		Users:  users,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// ValidateUser checks if a user exists and is active
func (m *Manager) ValidateUser(userID string) (bool, error) {
	return m.repository.ValidateUser(userID)
}

// Cube Management Operations

// CreateCube creates a new memory cube
func (m *Manager) CreateCube(params CreateCubeParams) (*Cube, error) {
	// Validate parameters
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Validate owner exists
	owner, err := m.repository.GetUser(params.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate owner: %w", err)
	}
	if owner == nil {
		return nil, fmt.Errorf("owner user not found")
	}

	// Create cube
	cube := &Cube{
		CubeID:   params.CubeID,
		CubeName: params.CubeName,
		CubePath: params.CubePath,
		OwnerID:  params.OwnerID,
		IsActive: true,
	}

	if cube.CubeID == "" {
		cube.CubeID = uuid.New().String()
	}

	createdCube, err := m.repository.CreateCube(cube)
	if err != nil {
		return nil, fmt.Errorf("failed to create cube: %w", err)
	}

	// Log audit event
	if m.auditEnabled {
		m.logAuditEvent(params.OwnerID, "cube_create", "cubes", createdCube.CubeID, 
			fmt.Sprintf("Created cube: %s", createdCube.CubeName), true)
	}

	return createdCube, nil
}

// GetCube retrieves a cube by ID
func (m *Manager) GetCube(cubeID string) (*Cube, error) {
	cube, err := m.repository.GetCube(cubeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cube: %w", err)
	}

	if cube == nil {
		return nil, fmt.Errorf("cube not found")
	}

	return cube, nil
}

// GetUserCubes returns all cubes accessible by a user
func (m *Manager) GetUserCubes(userID string) ([]Cube, error) {
	// Validate user exists
	valid, err := m.repository.ValidateUser(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate user: %w", err)
	}
	if !valid {
		return nil, fmt.Errorf("user not found or inactive")
	}

	cubes, err := m.repository.GetUserCubes(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user cubes: %w", err)
	}

	return cubes, nil
}

// ValidateUserCubeAccess checks if a user has access to a cube
func (m *Manager) ValidateUserCubeAccess(userID, cubeID string) (bool, error) {
	return m.repository.ValidateUserCubeAccess(userID, cubeID)
}

// AddUserToCube grants a user access to a cube
func (m *Manager) AddUserToCube(userID, cubeID string) error {
	// Validate user and cube exist
	user, err := m.repository.GetUser(userID)
	if err != nil {
		return fmt.Errorf("failed to validate user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	cube, err := m.repository.GetCube(cubeID)
	if err != nil {
		return fmt.Errorf("failed to validate cube: %w", err)
	}
	if cube == nil {
		return fmt.Errorf("cube not found")
	}

	// Add user to cube
	if err := m.repository.AddUserToCube(userID, cubeID); err != nil {
		return fmt.Errorf("failed to add user to cube: %w", err)
	}

	// Log audit event
	if m.auditEnabled {
		m.logAuditEvent(cube.OwnerID, "cube_share", "cubes", cubeID, 
			fmt.Sprintf("Added user %s to cube %s", user.UserName, cube.CubeName), true)
	}

	return nil
}

// RemoveUserFromCube removes a user's access to a cube
func (m *Manager) RemoveUserFromCube(userID, cubeID string) error {
	// Get user and cube for audit log
	user, err := m.repository.GetUser(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	
	cube, err := m.repository.GetCube(cubeID)
	if err != nil {
		return fmt.Errorf("failed to get cube: %w", err)
	}

	// Remove user from cube
	if err := m.repository.RemoveUserFromCube(userID, cubeID); err != nil {
		return fmt.Errorf("failed to remove user from cube: %w", err)
	}

	// Log audit event
	if m.auditEnabled && user != nil && cube != nil {
		m.logAuditEvent(cube.OwnerID, "cube_unshare", "cubes", cubeID, 
			fmt.Sprintf("Removed user %s from cube %s", user.UserName, cube.CubeName), true)
	}

	return nil
}

// DeleteCube soft deletes a cube
func (m *Manager) DeleteCube(cubeID string) error {
	// Get cube for audit log
	cube, err := m.repository.GetCube(cubeID)
	if err != nil {
		return fmt.Errorf("failed to get cube: %w", err)
	}
	if cube == nil {
		return fmt.Errorf("cube not found")
	}

	// Delete cube
	if err := m.repository.DeleteCube(cubeID); err != nil {
		return fmt.Errorf("failed to delete cube: %w", err)
	}

	// Log audit event
	if m.auditEnabled {
		m.logAuditEvent(cube.OwnerID, "cube_delete", "cubes", cubeID, 
			fmt.Sprintf("Deleted cube: %s", cube.CubeName), true)
	}

	return nil
}

// Authentication Methods

// Authenticate authenticates a user and returns auth response
func (m *Manager) Authenticate(credentials LoginCredentials) (*AuthResponse, error) {
	response, err := m.authService.AuthenticateUser(credentials)
	if err != nil {
		// Log failed authentication
		if m.auditEnabled {
			m.logAuditEvent("", "login_failed", "auth", "", 
				fmt.Sprintf("Failed login attempt for user: %s", credentials.Username), false)
		}
		return nil, err
	}

	// Log successful authentication
	if m.auditEnabled {
		m.logAuditEvent(response.User.UserID, "login_success", "auth", "", 
			fmt.Sprintf("Successful login for user: %s", response.User.UserName), true)
	}

	return response, nil
}

// ValidateToken validates a JWT token and returns the user
func (m *Manager) ValidateToken(token string) (*User, error) {
	return m.authService.ValidateToken(token)
}

// RefreshToken generates a new access token using a refresh token
func (m *Manager) RefreshToken(refreshToken string) (*AuthResponse, error) {
	return m.authService.RefreshToken(refreshToken)
}

// Logout invalidates a user session
func (m *Manager) Logout(userID, sessionID string) error {
	err := m.authService.Logout(userID, sessionID)
	
	// Log logout
	if m.auditEnabled {
		success := err == nil
		m.logAuditEvent(userID, "logout", "auth", "", "User logout", success)
	}

	return err
}

// ChangePassword changes a user's password
func (m *Manager) ChangePassword(userID, oldPassword, newPassword string) error {
	err := m.authService.ChangePassword(userID, oldPassword, newPassword)
	
	// Log password change
	if m.auditEnabled {
		success := err == nil
		m.logAuditEvent(userID, "password_change", "auth", "", "Password changed", success)
	}

	return err
}

// Authorization Methods

// CheckPermission checks if a user has permission to perform an action
func (m *Manager) CheckPermission(userID, resource, action string) (*PermissionResult, error) {
	return m.authzService.CheckPermission(userID, resource, action)
}

// CheckCubePermission checks cube-specific permissions
func (m *Manager) CheckCubePermission(userID, cubeID, action string) (*PermissionResult, error) {
	return m.authzService.CheckCubePermission(userID, cubeID, action)
}

// RequireRole checks if a user has a specific role or higher
func (m *Manager) RequireRole(userID string, requiredRole UserRole) (*PermissionResult, error) {
	return m.authzService.RequireRole(userID, requiredRole)
}

// GetUserPermissions returns all permissions for a user
func (m *Manager) GetUserPermissions(userID string) (map[string][]string, error) {
	return m.authzService.GetUserPermissions(userID)
}

// API Token Management Methods

// CreateAPIToken creates a new API token for a user
func (m *Manager) CreateAPIToken(userID string, params CreateAPITokenParams) (*APITokenResponse, error) {
	// Validate parameters
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Validate user exists
	user, err := m.repository.GetUser(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Generate secure token
	token, err := m.generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Hash the token for storage
	tokenHash, err := m.authService.HashPassword(token)
	if err != nil {
		return nil, fmt.Errorf("failed to hash token: %w", err)
	}

	// Create API token
	apiToken := &APIToken{
		UserID:      userID,
		Name:        params.Name,
		Description: params.Description,
		TokenHash:   tokenHash,
		Prefix:      token[:8], // Store first 8 chars for display
		Scopes:      params.Scopes,
		ExpiresAt:   params.ExpiresAt,
		IsActive:    true,
	}

	// Save to database
	createdToken, err := m.repository.CreateAPIToken(apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create API token: %w", err)
	}

	// Log audit event
	if m.auditEnabled {
		m.logAuditEvent(userID, "api_token_create", "tokens", createdToken.TokenID,
			fmt.Sprintf("Created API token: %s", createdToken.Name), true)
	}

	// Return response with the plain token (only time it's visible)
	return &APITokenResponse{
		Token:       token,
		TokenID:     createdToken.TokenID,
		Name:        createdToken.Name,
		Description: createdToken.Description,
		Prefix:      createdToken.Prefix,
		Scopes:      createdToken.Scopes,
		ExpiresAt:   createdToken.ExpiresAt,
		CreatedAt:   createdToken.CreatedAt,
	}, nil
}

// GetAPITokens returns all API tokens for a user (without the actual token values)
func (m *Manager) GetAPITokens(userID string) ([]APIToken, error) {
	// Validate user exists
	valid, err := m.repository.ValidateUser(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate user: %w", err)
	}
	if !valid {
		return nil, fmt.Errorf("user not found or inactive")
	}

	tokens, err := m.repository.GetUserAPITokens(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API tokens: %w", err)
	}

	return tokens, nil
}

// GetAPIToken retrieves a specific API token by ID
func (m *Manager) GetAPIToken(userID, tokenID string) (*APIToken, error) {
	token, err := m.repository.GetAPIToken(tokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API token: %w", err)
	}

	if token == nil {
		return nil, fmt.Errorf("API token not found")
	}

	// Verify the token belongs to the user
	if token.UserID != userID {
		return nil, fmt.Errorf("API token not found")
	}

	return token, nil
}

// UpdateAPIToken updates an API token's metadata
func (m *Manager) UpdateAPIToken(userID, tokenID string, params UpdateAPITokenParams) (*APIToken, error) {
	// Get existing token
	token, err := m.GetAPIToken(userID, tokenID)
	if err != nil {
		return nil, err
	}

	// Update fields
	if params.Name != "" {
		token.Name = params.Name
	}
	if params.Description != nil {
		token.Description = *params.Description
	}
	if params.Scopes != nil {
		token.Scopes = params.Scopes
	}
	if params.ExpiresAt != nil {
		token.ExpiresAt = params.ExpiresAt
	}
	if params.IsActive != nil {
		token.IsActive = *params.IsActive
	}

	// Update in database
	if err := m.repository.UpdateAPIToken(token); err != nil {
		return nil, fmt.Errorf("failed to update API token: %w", err)
	}

	// Log audit event
	if m.auditEnabled {
		m.logAuditEvent(userID, "api_token_update", "tokens", tokenID,
			fmt.Sprintf("Updated API token: %s", token.Name), true)
	}

	return token, nil
}

// RevokeAPIToken revokes an API token
func (m *Manager) RevokeAPIToken(userID, tokenID string) error {
	// Get token to verify ownership
	token, err := m.GetAPIToken(userID, tokenID)
	if err != nil {
		return err
	}

	// Delete from database
	if err := m.repository.DeleteAPIToken(tokenID); err != nil {
		return fmt.Errorf("failed to revoke API token: %w", err)
	}

	// Log audit event
	if m.auditEnabled {
		m.logAuditEvent(userID, "api_token_revoke", "tokens", tokenID,
			fmt.Sprintf("Revoked API token: %s", token.Name), true)
	}

	return nil
}

// ValidateAPIToken validates an API token and returns the associated user
func (m *Manager) ValidateAPIToken(token string) (*User, *APIToken, error) {
	// Get all active tokens (this is not efficient for large numbers of tokens)
	// In production, you'd want to add an index or use a different approach
	allTokens, err := m.repository.GetAllActiveAPITokens()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get API tokens: %w", err)
	}

	// Check each token hash
	for _, apiToken := range allTokens {
		// Check if token matches hash
		if err := bcrypt.CompareHashAndPassword([]byte(apiToken.TokenHash), []byte(token)); err == nil {
			// Token matches, check if it's still valid
			if !apiToken.IsActive || apiToken.IsExpired() {
				return nil, nil, fmt.Errorf("API token is inactive or expired")
			}

			// Get user
			user, err := m.repository.GetUser(apiToken.UserID)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to get user: %w", err)
			}

			if user == nil || !user.IsActive {
				return nil, nil, fmt.Errorf("user is inactive")
			}

			// Update last used
			apiToken.UpdateLastUsed("") // IP will be set by middleware
			if err := m.repository.UpdateAPIToken(&apiToken); err != nil {
				// Log error but don't fail validation
				fmt.Printf("Warning: failed to update token last used: %v\n", err)
			}

			// Remove password from user response
			user.Password = ""
			return user, &apiToken, nil
		}
	}

	return nil, nil, fmt.Errorf("invalid API token")
}

// generateSecureToken generates a cryptographically secure token
func (m *Manager) generateSecureToken() (string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Encode to base64 and add a prefix
	token := "memgos_" + base64.URLEncoding.EncodeToString(bytes)
	return token, nil
}

// Utility Methods

// HealthCheck performs a health check on the user management system
func (m *Manager) HealthCheck() error {
	return m.repository.HealthCheck()
}

// Close closes the user manager and cleans up resources
func (m *Manager) Close() error {
	return m.repository.Close()
}

// GetConfig returns the current configuration
func (m *Manager) GetConfig() *Config {
	return m.config
}

// logAuditEvent creates an audit log entry
func (m *Manager) logAuditEvent(userID, action, resource, resourceID, details string, success bool) {
	if !m.auditEnabled {
		return
	}

	auditLog := &AuditLog{
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    details,
		Success:    success,
		CreatedAt:  time.Now(),
	}

	// Log error but don't fail the operation
	if err := m.repository.CreateAuditLog(auditLog); err != nil {
		fmt.Printf("Warning: failed to create audit log: %v\n", err)
	}
}

// Parameter structs

// CreateUserParams contains parameters for creating a user
type CreateUserParams struct {
	UserID      string   `json:"user_id,omitempty"`
	Username    string   `json:"username" binding:"required"`
	Email       string   `json:"email,omitempty"`
	Password    string   `json:"password" binding:"required"`
	Role        UserRole `json:"role,omitempty"`
	DisplayName string   `json:"display_name,omitempty"`
}

// Validate validates the create user parameters
func (p CreateUserParams) Validate() error {
	if p.Username == "" {
		return NewValidationError("username is required")
	}
	if p.Password == "" {
		return NewValidationError("password is required")
	}
	if p.Role != "" && !p.Role.IsValid() {
		return NewValidationError("invalid role")
	}
	return nil
}

// UpdateUserParams contains parameters for updating a user
type UpdateUserParams struct {
	Username string    `json:"username,omitempty"`
	Email    string    `json:"email,omitempty"`
	Role     UserRole  `json:"role,omitempty"`
	IsActive *bool     `json:"is_active,omitempty"`
}

// CreateCubeParams contains parameters for creating a cube
type CreateCubeParams struct {
	CubeID   string `json:"cube_id,omitempty"`
	CubeName string `json:"cube_name" binding:"required"`
	CubePath string `json:"cube_path,omitempty"`
	OwnerID  string `json:"owner_id" binding:"required"`
}

// Validate validates the create cube parameters
func (p CreateCubeParams) Validate() error {
	if p.CubeName == "" {
		return NewValidationError("cube_name is required")
	}
	if p.OwnerID == "" {
		return NewValidationError("owner_id is required")
	}
	return nil
}

// PaginatedUsersResponse represents a paginated list of users
type PaginatedUsersResponse struct {
	Users  []User `json:"users"`
	Total  int64  `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// CreateAPITokenParams contains parameters for creating an API token
type CreateAPITokenParams struct {
	Name        string     `json:"name" binding:"required"`
	Description string     `json:"description,omitempty"`
	Scopes      []string   `json:"scopes" binding:"required"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// Validate validates the create API token parameters
func (p CreateAPITokenParams) Validate() error {
	if p.Name == "" {
		return NewValidationError("name is required")
	}
	if len(p.Scopes) == 0 {
		return NewValidationError("at least one scope is required")
	}
	
	// Validate scopes
	validScopes := map[string]bool{
		string(ScopeReadOnly):   true,
		string(ScopeWrite):      true,
		string(ScopeAdmin):      true,
		string(ScopeFullAccess): true,
	}
	
	for _, scope := range p.Scopes {
		if !validScopes[scope] {
			return NewValidationError(fmt.Sprintf("invalid scope: %s", scope))
		}
	}
	
	return nil
}

// UpdateAPITokenParams contains parameters for updating an API token
type UpdateAPITokenParams struct {
	Name        string     `json:"name,omitempty"`
	Description *string    `json:"description,omitempty"`
	Scopes      []string   `json:"scopes,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	IsActive    *bool      `json:"is_active,omitempty"`
}

// APITokenResponse represents the response when creating an API token
type APITokenResponse struct {
	Token       string     `json:"token"`                     // The actual token (only returned on creation)
	TokenID     string     `json:"token_id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Prefix      string     `json:"prefix"`                    // First 8 characters for display
	Scopes      []string   `json:"scopes"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}