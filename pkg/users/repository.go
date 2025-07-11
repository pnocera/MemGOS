package users

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Repository provides data access for user management
type Repository struct {
	db     *gorm.DB
	config *Config
}

// NewRepository creates a new user repository
func NewRepository(config *Config) (*Repository, error) {
	var db *gorm.DB
	var err error

	switch config.DatabaseType {
	case "sqlite":
		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(config.DatabasePath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}

		db, err = gorm.Open(sqlite.Open(config.DatabasePath), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent), // Reduce log noise
		})
	default:
		return nil, fmt.Errorf("unsupported database type: %s", config.DatabaseType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	repo := &Repository{
		db:     db,
		config: config,
	}

	// Auto-migrate database schema
	if err := repo.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Initialize default data
	if err := repo.initializeDefaultData(); err != nil {
		return nil, fmt.Errorf("failed to initialize default data: %w", err)
	}

	return repo, nil
}

// migrate runs database migrations
func (r *Repository) migrate() error {
	return r.db.AutoMigrate(
		&User{},
		&Cube{},
		&UserCubeAssociation{},
		&Session{},
		&APIToken{},
		&Permission{},
		&RolePermission{},
		&AuditLog{},
		&UserProfile{},
	)
}

// initializeDefaultData creates default permissions and root user
func (r *Repository) initializeDefaultData() error {
	// Create default permissions
	defaultPermissions := []Permission{
		{PermissionName: "users.create", Resource: "users", Action: "create", Description: "Create new users"},
		{PermissionName: "users.read", Resource: "users", Action: "read", Description: "View user information"},
		{PermissionName: "users.update", Resource: "users", Action: "update", Description: "Update user information"},
		{PermissionName: "users.delete", Resource: "users", Action: "delete", Description: "Delete users"},
		{PermissionName: "cubes.create", Resource: "cubes", Action: "create", Description: "Create new cubes"},
		{PermissionName: "cubes.read", Resource: "cubes", Action: "read", Description: "View cube information"},
		{PermissionName: "cubes.update", Resource: "cubes", Action: "update", Description: "Update cube information"},
		{PermissionName: "cubes.delete", Resource: "cubes", Action: "delete", Description: "Delete cubes"},
		{PermissionName: "api.read", Resource: "api", Action: "read", Description: "Access read-only API endpoints"},
		{PermissionName: "api.update", Resource: "api", Action: "update", Description: "Access API update endpoints"},
		{PermissionName: "system.read", Resource: "system", Action: "read", Description: "View system information"},
		{PermissionName: "system.update", Resource: "system", Action: "update", Description: "Update system settings"},
	}

	for _, perm := range defaultPermissions {
		var existing Permission
		result := r.db.Where("permission_name = ?", perm.PermissionName).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			if err := r.db.Create(&perm).Error; err != nil {
				return fmt.Errorf("failed to create permission %s: %w", perm.PermissionName, err)
			}
		}
	}

	// Check if root user exists
	var rootUser User
	result := r.db.Where("role = ?", RoleRoot).First(&rootUser)
	if result.Error == gorm.ErrRecordNotFound {
		// Create root user with default credentials
		rootUser = User{
			UserID:   "root",
			UserName: "root",
			Email:    "root@memgos.local",
			Role:     RoleRoot,
			Password: "$2a$10$8K1p/a0dhrxC9H4H4X6YV.BsKvpXUTQ5PjVG8XQ4nGhf5l7wX2.2m", // Default: "admin123"
			IsActive: true,
		}

		if err := r.db.Create(&rootUser).Error; err != nil {
			return fmt.Errorf("failed to create root user: %w", err)
		}
	}

	return nil
}

// User operations

// CreateUser creates a new user
func (r *Repository) CreateUser(user *User) (*User, error) {
	if err := r.db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return user, nil
}

// GetUser retrieves a user by ID
func (r *Repository) GetUser(userID string) (*User, error) {
	var user User
	if err := r.db.Where("user_id = ? AND is_active = ?", userID, true).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetUserByName retrieves a user by username
func (r *Repository) GetUserByName(username string) (*User, error) {
	var user User
	if err := r.db.Where("user_name = ? AND is_active = ?", username, true).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by name: %w", err)
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (r *Repository) GetUserByEmail(email string) (*User, error) {
	var user User
	if err := r.db.Where("email = ? AND is_active = ?", email, true).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &user, nil
}

// UpdateUser updates a user
func (r *Repository) UpdateUser(user *User) error {
	if err := r.db.Save(user).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// DeleteUser soft deletes a user
func (r *Repository) DeleteUser(userID string) error {
	result := r.db.Model(&User{}).Where("user_id = ? AND role != ?", userID, RoleRoot).Update("is_active", false)
	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found or cannot delete root user")
	}
	return nil
}

// ListUsers returns all active users with pagination
func (r *Repository) ListUsers(limit, offset int) ([]User, int64, error) {
	var users []User
	var total int64

	// Get total count
	if err := r.db.Model(&User{}).Where("is_active = ?", true).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Get users with pagination
	if err := r.db.Where("is_active = ?", true).Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	return users, total, nil
}

// ValidateUser checks if a user exists and is active
func (r *Repository) ValidateUser(userID string) (bool, error) {
	var count int64
	if err := r.db.Model(&User{}).Where("user_id = ? AND is_active = ?", userID, true).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to validate user: %w", err)
	}
	return count > 0, nil
}

// Cube operations

// CreateCube creates a new cube
func (r *Repository) CreateCube(cube *Cube) (*Cube, error) {
	// Start transaction
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create cube
	if err := tx.Create(cube).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create cube: %w", err)
	}

	// Add owner to cube users
	association := UserCubeAssociation{
		UserID: cube.OwnerID,
		CubeID: cube.CubeID,
	}
	if err := tx.Create(&association).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create cube association: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return cube, nil
}

// GetCube retrieves a cube by ID
func (r *Repository) GetCube(cubeID string) (*Cube, error) {
	var cube Cube
	if err := r.db.Where("cube_id = ? AND is_active = ?", cubeID, true).First(&cube).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get cube: %w", err)
	}
	return &cube, nil
}

// UpdateCube updates a cube
func (r *Repository) UpdateCube(cube *Cube) error {
	if err := r.db.Save(cube).Error; err != nil {
		return fmt.Errorf("failed to update cube: %w", err)
	}
	return nil
}

// DeleteCube soft deletes a cube
func (r *Repository) DeleteCube(cubeID string) error {
	result := r.db.Model(&Cube{}).Where("cube_id = ?", cubeID).Update("is_active", false)
	if result.Error != nil {
		return fmt.Errorf("failed to delete cube: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("cube not found")
	}
	return nil
}

// GetUserCubes returns all cubes accessible by a user
func (r *Repository) GetUserCubes(userID string) ([]Cube, error) {
	var cubes []Cube
	
	// Get cubes through many-to-many relationship
	if err := r.db.Joins("JOIN user_cube_associations ON cubes.cube_id = user_cube_associations.cube_id").
		Where("user_cube_associations.user_id = ? AND cubes.is_active = ?", userID, true).
		Order("cubes.created_at DESC").
		Find(&cubes).Error; err != nil {
		return nil, fmt.Errorf("failed to get user cubes: %w", err)
	}

	return cubes, nil
}

// ValidateUserCubeAccess checks if a user has access to a cube
func (r *Repository) ValidateUserCubeAccess(userID, cubeID string) (bool, error) {
	// Check if user exists and is active
	userValid, err := r.ValidateUser(userID)
	if err != nil {
		return false, err
	}
	if !userValid {
		return false, nil
	}

	// Check if cube exists and is active
	cube, err := r.GetCube(cubeID)
	if err != nil {
		return false, err
	}
	if cube == nil {
		return false, nil
	}

	// Check if user is owner
	if cube.OwnerID == userID {
		return true, nil
	}

	// Check many-to-many relationship
	var count int64
	if err := r.db.Model(&UserCubeAssociation{}).
		Where("user_id = ? AND cube_id = ?", userID, cubeID).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to validate cube access: %w", err)
	}

	return count > 0, nil
}

// AddUserToCube adds a user to a cube's access list
func (r *Repository) AddUserToCube(userID, cubeID string) error {
	// Check if association already exists
	var existing UserCubeAssociation
	result := r.db.Where("user_id = ? AND cube_id = ?", userID, cubeID).First(&existing)
	
	if result.Error == gorm.ErrRecordNotFound {
		// Create new association
		association := UserCubeAssociation{
			UserID: userID,
			CubeID: cubeID,
		}
		if err := r.db.Create(&association).Error; err != nil {
			return fmt.Errorf("failed to add user to cube: %w", err)
		}
	} else if result.Error != nil {
		return fmt.Errorf("failed to check existing association: %w", result.Error)
	}

	return nil
}

// RemoveUserFromCube removes a user from a cube's access list
func (r *Repository) RemoveUserFromCube(userID, cubeID string) error {
	// Check if user is owner
	cube, err := r.GetCube(cubeID)
	if err != nil {
		return err
	}
	if cube == nil {
		return fmt.Errorf("cube not found")
	}
	if cube.OwnerID == userID {
		return fmt.Errorf("cannot remove owner from cube")
	}

	// Remove association
	result := r.db.Where("user_id = ? AND cube_id = ?", userID, cubeID).Delete(&UserCubeAssociation{})
	if result.Error != nil {
		return fmt.Errorf("failed to remove user from cube: %w", result.Error)
	}

	return nil
}

// Session operations

// CreateSession creates a new session
func (r *Repository) CreateSession(session *Session) (*Session, error) {
	if err := r.db.Create(session).Error; err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	return session, nil
}

// GetSession retrieves a session by ID
func (r *Repository) GetSession(sessionID string) (*Session, error) {
	var session Session
	if err := r.db.Where("session_id = ? AND is_active = ?", sessionID, true).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &session, nil
}

// InvalidateSession invalidates a session
func (r *Repository) InvalidateSession(sessionID string) error {
	result := r.db.Model(&Session{}).Where("session_id = ?", sessionID).Update("is_active", false)
	if result.Error != nil {
		return fmt.Errorf("failed to invalidate session: %w", result.Error)
	}
	return nil
}

// InvalidateAllUserSessions invalidates all sessions for a user
func (r *Repository) InvalidateAllUserSessions(userID string) error {
	result := r.db.Model(&Session{}).Where("user_id = ?", userID).Update("is_active", false)
	if result.Error != nil {
		return fmt.Errorf("failed to invalidate user sessions: %w", result.Error)
	}
	return nil
}

// CleanupExpiredSessions removes expired sessions
func (r *Repository) CleanupExpiredSessions() error {
	result := r.db.Where("expires_at < ?", time.Now()).Delete(&Session{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", result.Error)
	}
	return nil
}

// Audit log operations

// CreateAuditLog creates a new audit log entry
func (r *Repository) CreateAuditLog(log *AuditLog) error {
	if err := r.db.Create(log).Error; err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}
	return nil
}

// GetAuditLogs retrieves audit logs with pagination and filtering
func (r *Repository) GetAuditLogs(limit, offset int, userID, action, resource string) ([]AuditLog, int64, error) {
	var logs []AuditLog
	var total int64

	query := r.db.Model(&AuditLog{})

	// Apply filters
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if resource != "" {
		query = query.Where("resource = ?", resource)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Get logs with pagination
	if err := query.Preload("User").Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get audit logs: %w", err)
	}

	return logs, total, nil
}

// API Token operations

// CreateAPIToken creates a new API token
func (r *Repository) CreateAPIToken(token *APIToken) (*APIToken, error) {
	if err := r.db.Create(token).Error; err != nil {
		return nil, fmt.Errorf("failed to create API token: %w", err)
	}
	return token, nil
}

// GetAPIToken retrieves an API token by ID
func (r *Repository) GetAPIToken(tokenID string) (*APIToken, error) {
	var token APIToken
	result := r.db.Where("token_id = ? AND is_active = ?", tokenID, true).First(&token)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get API token: %w", result.Error)
	}
	return &token, nil
}

// GetUserAPITokens retrieves all API tokens for a user
func (r *Repository) GetUserAPITokens(userID string) ([]APIToken, error) {
	var tokens []APIToken
	if err := r.db.Where("user_id = ? AND is_active = ?", userID, true).Find(&tokens).Error; err != nil {
		return nil, fmt.Errorf("failed to get user API tokens: %w", err)
	}
	return tokens, nil
}

// GetAllActiveAPITokens retrieves all active API tokens (for validation)
func (r *Repository) GetAllActiveAPITokens() ([]APIToken, error) {
	var tokens []APIToken
	if err := r.db.Where("is_active = ?", true).Find(&tokens).Error; err != nil {
		return nil, fmt.Errorf("failed to get active API tokens: %w", err)
	}
	return tokens, nil
}

// UpdateAPIToken updates an API token
func (r *Repository) UpdateAPIToken(token *APIToken) error {
	if err := r.db.Save(token).Error; err != nil {
		return fmt.Errorf("failed to update API token: %w", err)
	}
	return nil
}

// DeleteAPIToken deletes (revokes) an API token
func (r *Repository) DeleteAPIToken(tokenID string) error {
	result := r.db.Where("token_id = ?", tokenID).Delete(&APIToken{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete API token: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("API token not found")
	}
	return nil
}

// CleanupExpiredAPITokens removes expired API tokens
func (r *Repository) CleanupExpiredAPITokens() error {
	result := r.db.Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).Delete(&APIToken{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup expired API tokens: %w", result.Error)
	}
	return nil
}

// Health check operation

// HealthCheck performs a database health check
func (r *Repository) HealthCheck() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// Close closes the database connection
func (r *Repository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	return sqlDB.Close()
}