package users

import (
	"fmt"
	"strings"
)

// AuthorizationService provides authorization functionality
type AuthorizationService struct {
	config     *Config
	repository *Repository
}

// NewAuthorizationService creates a new authorization service
func NewAuthorizationService(config *Config, repository *Repository) *AuthorizationService {
	return &AuthorizationService{
		config:     config,
		repository: repository,
	}
}

// PermissionResult represents the result of a permission check
type PermissionResult struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// ResourcePermission represents a permission request for a resource
type ResourcePermission struct {
	UserID     string `json:"user_id"`
	Resource   string `json:"resource"`
	Action     string `json:"action"`
	ResourceID string `json:"resource_id,omitempty"`
}

// CheckPermission checks if a user has permission to perform an action on a resource
func (azs *AuthorizationService) CheckPermission(userID, resource, action string) (*PermissionResult, error) {
	user, err := azs.repository.GetUser(userID)
	if err != nil {
		return &PermissionResult{
			Allowed: false,
			Reason:  "failed to get user information",
		}, err
	}

	if user == nil {
		return &PermissionResult{
			Allowed: false,
			Reason:  "user not found",
		}, nil
	}

	if !user.IsActive {
		return &PermissionResult{
			Allowed: false,
			Reason:  "user account is inactive",
		}, nil
	}

	// Check role-based permissions
	if user.Role.HasPermission(resource, action) {
		return &PermissionResult{
			Allowed: true,
		}, nil
	}

	return &PermissionResult{
		Allowed: false,
		Reason:  fmt.Sprintf("insufficient permissions: %s role cannot %s %s", user.Role, action, resource),
	}, nil
}

// CheckCubePermission checks if a user has permission to access a specific cube
func (azs *AuthorizationService) CheckCubePermission(userID, cubeID, action string) (*PermissionResult, error) {
	// First check general cube permissions
	result, err := azs.CheckPermission(userID, "cubes", action)
	if err != nil {
		return result, err
	}

	if !result.Allowed {
		return result, nil
	}

	// Check specific cube access
	hasAccess, err := azs.repository.ValidateUserCubeAccess(userID, cubeID)
	if err != nil {
		return &PermissionResult{
			Allowed: false,
			Reason:  "failed to validate cube access",
		}, err
	}

	if !hasAccess {
		return &PermissionResult{
			Allowed: false,
			Reason:  "user does not have access to this cube",
		}, nil
	}

	return &PermissionResult{
		Allowed: true,
	}, nil
}

// CheckUserManagementPermission checks if a user can manage other users
func (azs *AuthorizationService) CheckUserManagementPermission(adminUserID, targetUserID, action string) (*PermissionResult, error) {
	// Check if admin has user management permissions
	result, err := azs.CheckPermission(adminUserID, "users", action)
	if err != nil {
		return result, err
	}

	if !result.Allowed {
		return result, nil
	}

	// Get both users
	adminUser, err := azs.repository.GetUser(adminUserID)
	if err != nil {
		return &PermissionResult{
			Allowed: false,
			Reason:  "failed to get admin user information",
		}, err
	}

	targetUser, err := azs.repository.GetUser(targetUserID)
	if err != nil {
		return &PermissionResult{
			Allowed: false,
			Reason:  "failed to get target user information",
		}, err
	}

	if adminUser == nil || targetUser == nil {
		return &PermissionResult{
			Allowed: false,
			Reason:  "user not found",
		}, nil
	}

	// Root users can manage anyone
	if adminUser.Role == RoleRoot {
		return &PermissionResult{
			Allowed: true,
		}, nil
	}

	// Prevent managing root users (unless you are root)
	if targetUser.Role == RoleRoot {
		return &PermissionResult{
			Allowed: false,
			Reason:  "cannot manage root users",
		}, nil
	}

	// Admins can manage users and guests, but not other admins (unless specified)
	if adminUser.Role == RoleAdmin {
		if targetUser.Role == RoleAdmin && action == "delete" {
			return &PermissionResult{
				Allowed: false,
				Reason:  "admins cannot delete other admins",
			}, nil
		}
		return &PermissionResult{
			Allowed: true,
		}, nil
	}

	return &PermissionResult{
		Allowed: false,
		Reason:  "insufficient privileges for user management",
	}, nil
}

// CheckAPIPermission checks if a user has permission to access an API endpoint
func (azs *AuthorizationService) CheckAPIPermission(userID, endpoint, method string) (*PermissionResult, error) {
	user, err := azs.repository.GetUser(userID)
	if err != nil {
		return &PermissionResult{
			Allowed: false,
			Reason:  "failed to get user information",
		}, err
	}

	if user == nil {
		return &PermissionResult{
			Allowed: false,
			Reason:  "user not found",
		}, nil
	}

	if !user.IsActive {
		return &PermissionResult{
			Allowed: false,
			Reason:  "user account is inactive",
		}, nil
	}

	// Map HTTP methods to actions
	action := strings.ToLower(method)
	switch action {
	case "get", "head", "options":
		action = "read"
	case "post":
		action = "create"
	case "put", "patch":
		action = "update"
	case "delete":
		action = "delete"
	}

	// Determine resource from endpoint
	resource := extractResourceFromEndpoint(endpoint)

	// Check role-based permissions
	if user.Role.HasPermission(resource, action) {
		return &PermissionResult{
			Allowed: true,
		}, nil
	}

	return &PermissionResult{
		Allowed: false,
		Reason:  fmt.Sprintf("insufficient permissions: %s role cannot %s %s", user.Role, action, resource),
	}, nil
}

// RequireRole checks if a user has a specific role or higher
func (azs *AuthorizationService) RequireRole(userID string, requiredRole UserRole) (*PermissionResult, error) {
	user, err := azs.repository.GetUser(userID)
	if err != nil {
		return &PermissionResult{
			Allowed: false,
			Reason:  "failed to get user information",
		}, err
	}

	if user == nil {
		return &PermissionResult{
			Allowed: false,
			Reason:  "user not found",
		}, nil
	}

	if !user.IsActive {
		return &PermissionResult{
			Allowed: false,
			Reason:  "user account is inactive",
		}, nil
	}

	// Check role hierarchy: root > admin > user > guest
	roleHierarchy := map[UserRole]int{
		RoleGuest: 1,
		RoleUser:  2,
		RoleAdmin: 3,
		RoleRoot:  4,
	}

	userLevel := roleHierarchy[user.Role]
	requiredLevel := roleHierarchy[requiredRole]

	if userLevel >= requiredLevel {
		return &PermissionResult{
			Allowed: true,
		}, nil
	}

	return &PermissionResult{
		Allowed: false,
		Reason:  fmt.Sprintf("insufficient role: requires %s or higher, user has %s", requiredRole, user.Role),
	}, nil
}

// RequireOwnership checks if a user owns a specific resource
func (azs *AuthorizationService) RequireOwnership(userID, resourceType, resourceID string) (*PermissionResult, error) {
	switch resourceType {
	case "cube":
		cube, err := azs.repository.GetCube(resourceID)
		if err != nil {
			return &PermissionResult{
				Allowed: false,
				Reason:  "failed to get cube information",
			}, err
		}

		if cube == nil {
			return &PermissionResult{
				Allowed: false,
				Reason:  "cube not found",
			}, nil
		}

		if cube.OwnerID == userID {
			return &PermissionResult{
				Allowed: true,
			}, nil
		}

		return &PermissionResult{
			Allowed: false,
			Reason:  "user is not the owner of this cube",
		}, nil

	default:
		return &PermissionResult{
			Allowed: false,
			Reason:  "unknown resource type for ownership check",
		}, nil
	}
}

// CanAccessUserData checks if a user can access another user's data
func (azs *AuthorizationService) CanAccessUserData(requestingUserID, targetUserID string) (*PermissionResult, error) {
	// Users can always access their own data
	if requestingUserID == targetUserID {
		return &PermissionResult{
			Allowed: true,
		}, nil
	}

	// Check if requesting user has user management permissions
	return azs.CheckUserManagementPermission(requestingUserID, targetUserID, "read")
}

// GetUserPermissions returns all permissions for a user
func (azs *AuthorizationService) GetUserPermissions(userID string) (map[string][]string, error) {
	user, err := azs.repository.GetUser(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Define all possible resources and actions
	resources := []string{"users", "cubes", "api", "system"}
	actions := []string{"create", "read", "update", "delete"}

	permissions := make(map[string][]string)

	for _, resource := range resources {
		allowedActions := []string{}
		for _, action := range actions {
			if user.Role.HasPermission(resource, action) {
				allowedActions = append(allowedActions, action)
			}
		}
		if len(allowedActions) > 0 {
			permissions[resource] = allowedActions
		}
	}

	return permissions, nil
}

// IsRoleElevated checks if a role change represents an elevation in privileges
func (azs *AuthorizationService) IsRoleElevated(fromRole, toRole UserRole) bool {
	roleHierarchy := map[UserRole]int{
		RoleGuest: 1,
		RoleUser:  2,
		RoleAdmin: 3,
		RoleRoot:  4,
	}

	return roleHierarchy[toRole] > roleHierarchy[fromRole]
}

// extractResourceFromEndpoint extracts the resource type from an API endpoint
func extractResourceFromEndpoint(endpoint string) string {
	// Remove leading slash and query parameters
	endpoint = strings.TrimPrefix(endpoint, "/")
	if idx := strings.Index(endpoint, "?"); idx != -1 {
		endpoint = endpoint[:idx]
	}

	// Split by slash and get the first segment
	parts := strings.Split(endpoint, "/")
	if len(parts) == 0 {
		return "api"
	}

	// Map endpoint prefixes to resources
	resourceMap := map[string]string{
		"users":   "users",
		"cubes":   "cubes",
		"auth":    "api",
		"admin":   "system",
		"system":  "system",
		"health":  "api",
		"metrics": "system",
	}

	if resource, exists := resourceMap[parts[0]]; exists {
		return resource
	}

	// Default to api for unknown endpoints
	return "api"
}

// AuthorizationError represents an authorization error
type AuthorizationError struct {
	Message string
	UserID  string
	Action  string
	Resource string
}

func (e AuthorizationError) Error() string {
	return e.Message
}

// NewAuthorizationError creates a new authorization error
func NewAuthorizationError(userID, action, resource, message string) error {
	return AuthorizationError{
		Message:  message,
		UserID:   userID,
		Action:   action,
		Resource: resource,
	}
}