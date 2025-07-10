// Package users provides comprehensive user management, authentication, and authorization
// functionality for MemGOS (Go port of MemOS).
//
// This package is a complete Go implementation based on the Python SQLAlchemy models
// found in the original MemOS system. It provides:
//
// # Core Features
//
//   - User lifecycle management (create, read, update, delete)
//   - Role-based access control (RBAC) with predefined roles: root, admin, user, guest
//   - JWT-based authentication with refresh tokens
//   - Session management and tracking
//   - Memory cube access control and sharing
//   - Comprehensive audit logging
//   - Password policies and security features
//   - Multi-factor authentication support (configurable)
//   - LDAP and OAuth integration (configurable)
//
// # Architecture
//
// The package follows a layered architecture:
//
//	┌─────────────────┐
//	│    Manager      │  ← Main orchestration layer
//	├─────────────────┤
//	│ Auth │   Authz  │  ← Authentication & Authorization services
//	├─────────────────┤
//	│   Repository    │  ← Data access layer
//	├─────────────────┤
//	│   GORM/SQLite   │  ← Database layer
//	└─────────────────┘
//
// # Models
//
// The package defines several core models:
//
//   - User: Represents a user in the system with roles and permissions
//   - Cube: Represents memory cubes that can be owned and shared
//   - Session: Represents active user sessions for authentication tracking
//   - AuditLog: Tracks all user actions for security and compliance
//   - UserProfile: Extended user information and preferences
//   - Permission: Granular permissions for fine-grained access control
//
// # User Roles
//
// The system supports a hierarchical role system:
//
//   - Root: System administrator with all permissions
//   - Admin: Administrative user with user and system management capabilities
//   - User: Regular user with standard cube and API access
//   - Guest: Limited access user with read-only permissions
//
// # Quick Start
//
//	// Initialize user manager with default configuration
//	config := users.DefaultConfig()
//	config.DatabasePath = "./data/users.db"
//	config.JWTSecret = "your-secret-key"
//
//	manager, err := users.NewManager(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer manager.Close()
//
//	// Create a new user
//	params := users.CreateUserParams{
//	    Username: "alice",
//	    Email:    "alice@example.com",
//	    Password: "SecurePass123!",
//	    Role:     users.RoleUser,
//	}
//
//	user, err := manager.CreateUser(params)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Authenticate user
//	credentials := users.LoginCredentials{
//	    Username: "alice",
//	    Password: "SecurePass123!",
//	}
//
//	authResponse, err := manager.Authenticate(credentials)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Use the access token for API calls
//	fmt.Printf("Access Token: %s\n", authResponse.AccessToken)
//
//	// Create a memory cube
//	cubeParams := users.CreateCubeParams{
//	    CubeName: "Alice's Knowledge Base",
//	    OwnerID:  user.UserID,
//	}
//
//	cube, err := manager.CreateCube(cubeParams)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Check permissions
//	result, err := manager.CheckCubePermission(user.UserID, cube.CubeID, "read")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	if result.Allowed {
//	    fmt.Println("User has access to the cube")
//	}
//
// # Advanced Configuration
//
// The package supports extensive configuration options:
//
//	config := &users.Config{
//	    DatabaseType: "sqlite",
//	    DatabasePath: "./users.db",
//
//	    AuthMethod:         "jwt",
//	    JWTSecret:          "your-jwt-secret",
//	    JWTExpirationTime:  24 * time.Hour,
//	    SessionTimeout:     8 * time.Hour,
//	    RefreshTokenExpiry: 7 * 24 * time.Hour,
//
//	    PasswordPolicy: users.PasswordPolicy{
//	        MinLength:        12,
//	        RequireUppercase: true,
//	        RequireLowercase: true,
//	        RequireNumbers:   true,
//	        RequireSymbols:   true,
//	        MaxAge:           90 * 24 * time.Hour,
//	        PreventReuse:     10,
//	    },
//
//	    MaxLoginAttempts:   5,
//	    LockoutDuration:    30 * time.Minute,
//	    RequireMFA:         true,
//	    EnableAuditLogging: true,
//
//	    DefaultRole:      users.RoleUser,
//	    AllowSelfSignup:  false,
//
//	    LDAP: users.LDAPConfig{
//	        Enabled:      true,
//	        Server:       "ldap.example.com",
//	        Port:         389,
//	        BaseDN:       "dc=example,dc=com",
//	        UserFilter:   "(&(objectClass=user)(sAMAccountName=%s))",
//	        GroupFilter:  "(&(objectClass=group)(member=%s))",
//	    },
//	}
//
// # Security Features
//
// The package implements several security best practices:
//
//   - Password hashing using bcrypt with configurable cost
//   - JWT tokens with configurable expiration
//   - Session management with automatic cleanup
//   - Rate limiting support (configurable)
//   - Audit logging for all user actions
//   - Role-based access control (RBAC)
//   - Password policies with complexity requirements
//   - Account lockout after failed login attempts
//   - Multi-factor authentication support
//   - Secure token generation using crypto/rand
//
// # Database Support
//
// Currently supports SQLite with GORM as the ORM. The architecture allows
// for easy extension to other databases supported by GORM:
//
//   - PostgreSQL
//   - MySQL
//   - SQL Server
//
// # Thread Safety
//
// The Manager and its services are designed to be thread-safe and can be used
// concurrently from multiple goroutines. The underlying GORM database connection
// pool handles concurrent access safely.
//
// # Integration with MemGOS
//
// This package integrates seamlessly with other MemGOS components:
//
//   - API layer uses the Manager for authentication middleware
//   - Memory cubes use the access control system
//   - Chat system uses user profiles for personalization
//   - All components can use the audit logging for compliance
//
// # Error Handling
//
// The package defines custom error types for different scenarios:
//
//   - ValidationError: For input validation failures
//   - AuthenticationError: For authentication failures
//   - AuthorizationError: For authorization failures
//
// All errors are wrapped with context to provide meaningful error messages.
//
// # Performance Considerations
//
// The package is designed for performance:
//
//   - Database connection pooling via GORM
//   - Efficient queries with proper indexing
//   - JWT tokens for stateless authentication
//   - Session cleanup background processes
//   - Audit log rotation (configurable)
//
// # Monitoring and Observability
//
// The package provides health check endpoints and supports:
//
//   - Database health monitoring
//   - Session metrics
//   - Authentication failure tracking
//   - Performance metrics via audit logs
//
// For more detailed examples and advanced usage, see the test files and
// the MemGOS documentation.
package users