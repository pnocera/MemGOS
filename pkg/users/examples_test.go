package users_test

import (
	"fmt"
	"log"
	"time"

	"github.com/memtensor/memgos/pkg/users"
)

// Example_basicUsage demonstrates basic user management operations
func Example_basicUsage() {
	// Initialize user manager with default configuration
	config := users.DefaultConfig()
	config.DatabasePath = ":memory:" // Use in-memory database for example
	config.JWTSecret = "example-secret-key"

	manager, err := users.NewManager(config)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	// Create a new user
	params := users.CreateUserParams{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "SecurePass123!",
		Role:     users.RoleUser,
	}

	user, err := manager.CreateUser(params)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Created user: %s (ID: %s)\n", user.UserName, user.UserID)

	// Authenticate user
	credentials := users.LoginCredentials{
		Username: "alice",
		Password: "SecurePass123!",
	}

	authResponse, err := manager.Authenticate(credentials)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Authentication successful. Token expires at: %s\n", 
		time.Unix(authResponse.ExpiresAt, 0).Format(time.RFC3339))

	// Output:
	// Created user: alice (ID: [generated-uuid])
	// Authentication successful. Token expires at: [timestamp]
}

// Example_cubeManagement demonstrates memory cube operations
func Example_cubeManagement() {
	// Setup manager
	config := users.DefaultConfig()
	config.DatabasePath = ":memory:"
	config.JWTSecret = "example-secret-key"

	manager, err := users.NewManager(config)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	// Create users
	ownerParams := users.CreateUserParams{
		Username: "owner",
		Email:    "owner@example.com",
		Password: "SecurePass123!",
		Role:     users.RoleUser,
	}

	collaboratorParams := users.CreateUserParams{
		Username: "collaborator",
		Email:    "collaborator@example.com",
		Password: "SecurePass123!",
		Role:     users.RoleUser,
	}

	owner, err := manager.CreateUser(ownerParams)
	if err != nil {
		log.Fatal(err)
	}

	collaborator, err := manager.CreateUser(collaboratorParams)
	if err != nil {
		log.Fatal(err)
	}

	// Create a memory cube
	cubeParams := users.CreateCubeParams{
		CubeName: "Knowledge Base",
		CubePath: "/data/knowledge-base",
		OwnerID:  owner.UserID,
	}

	cube, err := manager.CreateCube(cubeParams)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Created cube: %s (ID: %s)\n", cube.CubeName, cube.CubeID)

	// Check owner access
	hasAccess, err := manager.ValidateUserCubeAccess(owner.UserID, cube.CubeID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Owner has access: %t\n", hasAccess)

	// Check collaborator access (should be false initially)
	hasAccess, err = manager.ValidateUserCubeAccess(collaborator.UserID, cube.CubeID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Collaborator has access (before sharing): %t\n", hasAccess)

	// Share cube with collaborator
	err = manager.AddUserToCube(collaborator.UserID, cube.CubeID)
	if err != nil {
		log.Fatal(err)
	}

	// Check collaborator access again
	hasAccess, err = manager.ValidateUserCubeAccess(collaborator.UserID, cube.CubeID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Collaborator has access (after sharing): %t\n", hasAccess)

	// Get all cubes for collaborator
	userCubes, err := manager.GetUserCubes(collaborator.UserID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Collaborator has access to %d cubes\n", len(userCubes))

	// Output:
	// Created cube: Knowledge Base (ID: [generated-uuid])
	// Owner has access: true
	// Collaborator has access (before sharing): false
	// Collaborator has access (after sharing): true
	// Collaborator has access to 1 cubes
}

// Example_roleBasedAccess demonstrates role-based access control
func Example_roleBasedAccess() {
	// Setup manager
	config := users.DefaultConfig()
	config.DatabasePath = ":memory:"
	config.JWTSecret = "example-secret-key"

	manager, err := users.NewManager(config)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	// Create users with different roles
	adminParams := users.CreateUserParams{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "AdminPass123!",
		Role:     users.RoleAdmin,
	}

	userParams := users.CreateUserParams{
		Username: "regularuser",
		Email:    "user@example.com",
		Password: "UserPass123!",
		Role:     users.RoleUser,
	}

	guestParams := users.CreateUserParams{
		Username: "guest",
		Email:    "guest@example.com",
		Password: "GuestPass123!",
		Role:     users.RoleGuest,
	}

	admin, err := manager.CreateUser(adminParams)
	if err != nil {
		log.Fatal(err)
	}

	regularUser, err := manager.CreateUser(userParams)
	if err != nil {
		log.Fatal(err)
	}

	guest, err := manager.CreateUser(guestParams)
	if err != nil {
		log.Fatal(err)
	}

	// Test permissions for different roles
	testPermission := func(userID, username string, resource, action string) {
		result, err := manager.CheckPermission(userID, resource, action)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s can %s %s: %t\n", username, action, resource, result.Allowed)
	}

	// Test user management permissions
	testPermission(admin.UserID, "admin", "users", "create")
	testPermission(regularUser.UserID, "user", "users", "create")
	testPermission(guest.UserID, "guest", "users", "create")

	// Test cube permissions
	testPermission(admin.UserID, "admin", "cubes", "delete")
	testPermission(regularUser.UserID, "user", "cubes", "create")
	testPermission(guest.UserID, "guest", "cubes", "create")

	// Test API access
	testPermission(admin.UserID, "admin", "api", "read")
	testPermission(regularUser.UserID, "user", "api", "read")
	testPermission(guest.UserID, "guest", "api", "read")

	// Output:
	// admin can create users: true
	// user can create users: false
	// guest can create users: false
	// admin can delete cubes: true
	// user can create cubes: true
	// guest can create cubes: false
	// admin can read api: true
	// user can read api: true
	// guest can read api: true
}

// Example_passwordManagement demonstrates password operations
func Example_passwordManagement() {
	// Setup manager with strict password policy
	config := users.DefaultConfig()
	config.DatabasePath = ":memory:"
	config.JWTSecret = "example-secret-key"
	config.PasswordPolicy = users.PasswordPolicy{
		MinLength:        12,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireNumbers:   true,
		RequireSymbols:   true,
		MaxAge:           90 * 24 * time.Hour,
		PreventReuse:     5,
	}

	manager, err := users.NewManager(config)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	// Create a user
	params := users.CreateUserParams{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "VerySecurePass123!",
		Role:     users.RoleUser,
	}

	user, err := manager.CreateUser(params)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Created user with secure password: %s\n", user.UserName)

	// Authenticate with original password
	credentials := users.LoginCredentials{
		Username: "alice",
		Password: "VerySecurePass123!",
	}

	_, err = manager.Authenticate(credentials)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Authentication with original password: successful")

	// Change password
	err = manager.ChangePassword(user.UserID, "VerySecurePass123!", "NewSecurePass456!")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Password changed successfully")

	// Try to authenticate with old password (should fail)
	_, err = manager.Authenticate(credentials)
	if err != nil {
		fmt.Println("Authentication with old password: failed (expected)")
	}

	// Authenticate with new password
	credentials.Password = "NewSecurePass456!"
	_, err = manager.Authenticate(credentials)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Authentication with new password: successful")

	// Output:
	// Created user with secure password: alice
	// Authentication with original password: successful
	// Password changed successfully
	// Authentication with old password: failed (expected)
	// Authentication with new password: successful
}

// Example_tokenRefresh demonstrates JWT token refresh functionality
func Example_tokenRefresh() {
	// Setup manager
	config := users.DefaultConfig()
	config.DatabasePath = ":memory:"
	config.JWTSecret = "example-secret-key"
	config.JWTExpirationTime = 15 * time.Minute    // Short-lived access tokens
	config.RefreshTokenExpiry = 7 * 24 * time.Hour // Long-lived refresh tokens

	manager, err := users.NewManager(config)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	// Create and authenticate user
	params := users.CreateUserParams{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "SecurePass123!",
		Role:     users.RoleUser,
	}

	_, err := manager.CreateUser(params)
	if err != nil {
		log.Fatal(err)
	}

	credentials := users.LoginCredentials{
		Username: "alice",
		Password: "SecurePass123!",
	}

	authResponse, err := manager.Authenticate(credentials)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Initial authentication successful\n")
	fmt.Printf("Access token expires at: %s\n", 
		time.Unix(authResponse.ExpiresAt, 0).Format(time.RFC3339))

	// Validate the access token
	validatedUser, err := manager.ValidateToken(authResponse.AccessToken)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Token validation successful for user: %s\n", validatedUser.UserName)

	// Refresh the token
	refreshResponse, err := manager.RefreshToken(authResponse.RefreshToken)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Token refresh successful\n")
	fmt.Printf("New access token expires at: %s\n", 
		time.Unix(refreshResponse.ExpiresAt, 0).Format(time.RFC3339))

	// Validate the new access token
	validatedUser, err = manager.ValidateToken(refreshResponse.AccessToken)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("New token validation successful for user: %s\n", validatedUser.UserName)

	// Output:
	// Initial authentication successful
	// Access token expires at: [timestamp]
	// Token validation successful for user: alice
	// Token refresh successful
	// New access token expires at: [timestamp]
	// New token validation successful for user: alice
}

// Example_userManagement demonstrates administrative user operations
func Example_userManagement() {
	// Setup manager
	config := users.DefaultConfig()
	config.DatabasePath = ":memory:"
	config.JWTSecret = "example-secret-key"

	manager, err := users.NewManager(config)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	// Create multiple users
	userParams := []users.CreateUserParams{
		{Username: "alice", Email: "alice@example.com", Password: "Pass123!", Role: users.RoleUser},
		{Username: "bob", Email: "bob@example.com", Password: "Pass123!", Role: users.RoleAdmin},
		{Username: "charlie", Email: "charlie@example.com", Password: "Pass123!", Role: users.RoleGuest},
	}

	for _, params := range userParams {
		_, err := manager.CreateUser(params)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Created user: %s (%s)\n", params.Username, params.Role)
	}

	// List all users
	userList, err := manager.ListUsers(10, 0)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nTotal users in system: %d\n", userList.Total)
	for _, user := range userList.Users {
		fmt.Printf("- %s (%s) - Active: %t\n", user.UserName, user.Role, user.IsActive)
	}

	// Update a user
	alice, err := manager.GetUserByName("alice")
	if err != nil {
		log.Fatal(err)
	}

	updateParams := users.UpdateUserParams{
		Role: users.RoleAdmin,
	}

	updatedAlice, err := manager.UpdateUser(alice.UserID, updateParams)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nUpdated alice's role to: %s\n", updatedAlice.Role)

	// Deactivate a user
	charlie, err := manager.GetUserByName("charlie")
	if err != nil {
		log.Fatal(err)
	}

	err = manager.DeleteUser(charlie.UserID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Deactivated user: charlie\n")

	// Verify charlie is deactivated
	isValid, err := manager.ValidateUser(charlie.UserID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Charlie is still valid: %t\n", isValid)

	// Output:
	// Created user: alice (user)
	// Created user: bob (admin)
	// Created user: charlie (guest)
	//
	// Total users in system: 4
	// - root (root) - Active: true
	// - alice (user) - Active: true
	// - bob (admin) - Active: true
	// - charlie (guest) - Active: true
	//
	// Updated alice's role to: admin
	// Deactivated user: charlie
	// Charlie is still valid: false
}