// Package users provides user management, authentication, and authorization for MemGOS.
// This is the Go implementation based on the Python SQLAlchemy models.
package users

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRole represents the different roles a user can have in the system
type UserRole string

const (
	RoleRoot  UserRole = "root"  // Root user with all permissions
	RoleAdmin UserRole = "admin" // Admin user with management permissions
	RoleUser  UserRole = "user"  // Regular user with standard permissions
	RoleGuest UserRole = "guest" // Guest user with limited permissions
)

// User represents a user in the system
type User struct {
	UserID    string    `gorm:"primaryKey;type:varchar(36)" json:"user_id"`
	UserName  string    `gorm:"uniqueIndex;not null" json:"user_name"`
	Email     string    `gorm:"uniqueIndex" json:"email,omitempty"`
	Role      UserRole  `gorm:"not null;default:'user'" json:"role"`
	Password  string    `gorm:"not null" json:"-"` // Password hash, never returned in JSON
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
	IsActive  bool      `gorm:"not null;default:true" json:"is_active"`

	// Relationships
	OwnedCubes []Cube `gorm:"foreignKey:OwnerID;constraint:OnDelete:CASCADE" json:"owned_cubes,omitempty"`
	Cubes      []Cube `gorm:"many2many:user_cube_associations" json:"cubes,omitempty"`
	Sessions   []Session `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	AuditLogs  []AuditLog `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
}

// BeforeCreate hook for User model
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.UserID == "" {
		u.UserID = uuid.New().String()
	}
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate hook for User model
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	u.UpdatedAt = time.Now()
	return nil
}

// Cube represents a memory cube that can be owned and shared
type Cube struct {
	CubeID    string    `gorm:"primaryKey;type:varchar(36)" json:"cube_id"`
	CubeName  string    `gorm:"not null" json:"cube_name"`
	CubePath  string    `json:"cube_path,omitempty"` // Local path or remote repo
	OwnerID   string    `gorm:"not null;type:varchar(36)" json:"owner_id"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
	IsActive  bool      `gorm:"not null;default:true" json:"is_active"`

	// Relationships
	Owner User   `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Users []User `gorm:"many2many:user_cube_associations" json:"users,omitempty"`
}

// BeforeCreate hook for Cube model
func (c *Cube) BeforeCreate(tx *gorm.DB) error {
	if c.CubeID == "" {
		c.CubeID = uuid.New().String()
	}
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate hook for Cube model
func (c *Cube) BeforeUpdate(tx *gorm.DB) error {
	c.UpdatedAt = time.Now()
	return nil
}

// UserCubeAssociation represents the many-to-many relationship between users and cubes
type UserCubeAssociation struct {
	UserID    string    `gorm:"primaryKey;type:varchar(36)"`
	CubeID    string    `gorm:"primaryKey;type:varchar(36)"`
	CreatedAt time.Time `gorm:"not null"`

	// Relationships
	User User `gorm:"foreignKey:UserID"`
	Cube Cube `gorm:"foreignKey:CubeID"`
}

// BeforeCreate hook for UserCubeAssociation
func (uca *UserCubeAssociation) BeforeCreate(tx *gorm.DB) error {
	uca.CreatedAt = time.Now()
	return nil
}

// Session represents a user authentication session
type Session struct {
	SessionID string    `gorm:"primaryKey;type:varchar(36)" json:"session_id"`
	UserID    string    `gorm:"not null;type:varchar(36)" json:"user_id"`
	Token     string    `gorm:"not null;uniqueIndex" json:"-"` // JWT token or session token
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
	IsActive  bool      `gorm:"not null;default:true" json:"is_active"`
	UserAgent string    `json:"user_agent,omitempty"`
	IPAddress string    `json:"ip_address,omitempty"`

	// Relationships
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// BeforeCreate hook for Session model
func (s *Session) BeforeCreate(tx *gorm.DB) error {
	if s.SessionID == "" {
		s.SessionID = uuid.New().String()
	}
	s.CreatedAt = time.Now()
	return nil
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// Permission represents a specific permission in the system
type Permission struct {
	PermissionID   string `gorm:"primaryKey;type:varchar(36)" json:"permission_id"`
	PermissionName string `gorm:"not null;uniqueIndex" json:"permission_name"`
	Description    string `json:"description,omitempty"`
	Resource       string `gorm:"not null" json:"resource"` // e.g., "cubes", "users", "api"
	Action         string `gorm:"not null" json:"action"`   // e.g., "create", "read", "update", "delete"
	CreatedAt      time.Time `gorm:"not null" json:"created_at"`
}

// BeforeCreate hook for Permission model
func (p *Permission) BeforeCreate(tx *gorm.DB) error {
	if p.PermissionID == "" {
		p.PermissionID = uuid.New().String()
	}
	p.CreatedAt = time.Now()
	return nil
}

// RolePermission represents the many-to-many relationship between roles and permissions
type RolePermission struct {
	Role         UserRole `gorm:"primaryKey"`
	PermissionID string   `gorm:"primaryKey;type:varchar(36)"`
	CreatedAt    time.Time `gorm:"not null"`

	// Relationships
	Permission Permission `gorm:"foreignKey:PermissionID"`
}

// BeforeCreate hook for RolePermission
func (rp *RolePermission) BeforeCreate(tx *gorm.DB) error {
	rp.CreatedAt = time.Now()
	return nil
}

// AuditLog represents an audit log entry for tracking user actions
type AuditLog struct {
	LogID      string    `gorm:"primaryKey;type:varchar(36)" json:"log_id"`
	UserID     string    `gorm:"type:varchar(36)" json:"user_id,omitempty"`
	Action     string    `gorm:"not null" json:"action"`
	Resource   string    `gorm:"not null" json:"resource"`
	ResourceID string    `gorm:"type:varchar(36)" json:"resource_id,omitempty"`
	Details    string    `gorm:"type:text" json:"details,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
	UserAgent  string    `json:"user_agent,omitempty"`
	CreatedAt  time.Time `gorm:"not null" json:"created_at"`
	Success    bool      `gorm:"not null" json:"success"`

	// Relationships
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// BeforeCreate hook for AuditLog model
func (al *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if al.LogID == "" {
		al.LogID = uuid.New().String()
	}
	al.CreatedAt = time.Now()
	return nil
}

// UserProfile represents extended user profile information
type UserProfile struct {
	UserID      string    `gorm:"primaryKey;type:varchar(36)" json:"user_id"`
	FirstName   string    `json:"first_name,omitempty"`
	LastName    string    `json:"last_name,omitempty"`
	DisplayName string    `json:"display_name,omitempty"`
	Avatar      string    `json:"avatar,omitempty"` // URL or base64 encoded image
	Timezone    string    `json:"timezone,omitempty"`
	Language    string    `json:"language,omitempty"`
	Preferences map[string]interface{} `gorm:"type:json" json:"preferences,omitempty"`
	CreatedAt   time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null" json:"updated_at"`

	// Relationships
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

// BeforeCreate hook for UserProfile model
func (up *UserProfile) BeforeCreate(tx *gorm.DB) error {
	up.CreatedAt = time.Now()
	up.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate hook for UserProfile model
func (up *UserProfile) BeforeUpdate(tx *gorm.DB) error {
	up.UpdatedAt = time.Now()
	return nil
}

// HasPermission checks if a user role has a specific permission
func (r UserRole) HasPermission(resource, action string) bool {
	// Root has all permissions
	if r == RoleRoot {
		return true
	}

	// Define role-based permissions
	rolePermissions := map[UserRole]map[string][]string{
		RoleAdmin: {
			"users":  {"create", "read", "update", "delete"},
			"cubes":  {"create", "read", "update", "delete"},
			"api":    {"read", "update"},
			"system": {"read", "update"},
		},
		RoleUser: {
			"cubes": {"create", "read", "update"},
			"api":   {"read"},
		},
		RoleGuest: {
			"cubes": {"read"},
			"api":   {"read"},
		},
	}

	if permissions, exists := rolePermissions[r]; exists {
		if actions, resourceExists := permissions[resource]; resourceExists {
			for _, allowedAction := range actions {
				if allowedAction == action {
					return true
				}
			}
		}
	}

	return false
}

// String returns the string representation of UserRole
func (r UserRole) String() string {
	return string(r)
}

// IsValid checks if the UserRole is a valid role
func (r UserRole) IsValid() bool {
	switch r {
	case RoleRoot, RoleAdmin, RoleUser, RoleGuest:
		return true
	default:
		return false
	}
}