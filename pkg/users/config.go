package users

import (
	"time"
)

// Config holds the configuration for the user management system
type Config struct {
	// Database configuration
	DatabaseType string `json:"database_type" yaml:"database_type"` // "sqlite", "postgres", "mysql"
	DatabaseURL  string `json:"database_url" yaml:"database_url"`   // Database connection URL
	DatabasePath string `json:"database_path" yaml:"database_path"` // SQLite database file path

	// Authentication configuration
	AuthMethod         string        `json:"auth_method" yaml:"auth_method"`                   // "local", "jwt", "oauth", "ldap"
	JWTSecret          string        `json:"jwt_secret" yaml:"jwt_secret"`                     // Secret for JWT signing
	JWTExpirationTime  time.Duration `json:"jwt_expiration_time" yaml:"jwt_expiration_time"`   // JWT token expiration
	SessionTimeout     time.Duration `json:"session_timeout" yaml:"session_timeout"`           // Session timeout duration
	RefreshTokenExpiry time.Duration `json:"refresh_token_expiry" yaml:"refresh_token_expiry"` // Refresh token expiration

	// Password policy configuration
	PasswordPolicy PasswordPolicy `json:"password_policy" yaml:"password_policy"`

	// Security configuration
	MaxLoginAttempts    int           `json:"max_login_attempts" yaml:"max_login_attempts"`       // Maximum failed login attempts
	LockoutDuration     time.Duration `json:"lockout_duration" yaml:"lockout_duration"`           // Account lockout duration
	RequireMFA          bool          `json:"require_mfa" yaml:"require_mfa"`                     // Require multi-factor authentication
	AllowGuestAccess    bool          `json:"allow_guest_access" yaml:"allow_guest_access"`       // Allow guest user access
	EnableAuditLogging  bool          `json:"enable_audit_logging" yaml:"enable_audit_logging"`   // Enable audit logging
	EnableRateLimiting  bool          `json:"enable_rate_limiting" yaml:"enable_rate_limiting"`   // Enable API rate limiting

	// Role and permission configuration
	DefaultRole      UserRole `json:"default_role" yaml:"default_role"`           // Default role for new users
	AllowSelfSignup  bool     `json:"allow_self_signup" yaml:"allow_self_signup"` // Allow user self-registration
	RequireEmailVerification bool `json:"require_email_verification" yaml:"require_email_verification"` // Require email verification

	// LDAP configuration (if using LDAP authentication)
	LDAP LDAPConfig `json:"ldap" yaml:"ldap"`

	// OAuth configuration (if using OAuth authentication)
	OAuth OAuthConfig `json:"oauth" yaml:"oauth"`
}

// PasswordPolicy defines password requirements
type PasswordPolicy struct {
	MinLength        int  `json:"min_length" yaml:"min_length"`               // Minimum password length
	RequireUppercase bool `json:"require_uppercase" yaml:"require_uppercase"` // Require uppercase letters
	RequireLowercase bool `json:"require_lowercase" yaml:"require_lowercase"` // Require lowercase letters
	RequireNumbers   bool `json:"require_numbers" yaml:"require_numbers"`     // Require numbers
	RequireSymbols   bool `json:"require_symbols" yaml:"require_symbols"`     // Require special symbols
	MaxAge           time.Duration `json:"max_age" yaml:"max_age"`             // Maximum password age
	PreventReuse     int  `json:"prevent_reuse" yaml:"prevent_reuse"`         // Number of previous passwords to prevent reuse
}

// LDAPConfig holds LDAP configuration settings
type LDAPConfig struct {
	Enabled        bool   `json:"enabled" yaml:"enabled"`
	Server         string `json:"server" yaml:"server"`
	Port           int    `json:"port" yaml:"port"`
	BaseDN         string `json:"base_dn" yaml:"base_dn"`
	BindDN         string `json:"bind_dn" yaml:"bind_dn"`
	BindPassword   string `json:"bind_password" yaml:"bind_password"`
	UserFilter     string `json:"user_filter" yaml:"user_filter"`
	GroupFilter    string `json:"group_filter" yaml:"group_filter"`
	EmailAttribute string `json:"email_attribute" yaml:"email_attribute"`
	NameAttribute  string `json:"name_attribute" yaml:"name_attribute"`
	UseSSL         bool   `json:"use_ssl" yaml:"use_ssl"`
	SkipTLSVerify  bool   `json:"skip_tls_verify" yaml:"skip_tls_verify"`
}

// OAuthConfig holds OAuth configuration settings
type OAuthConfig struct {
	Enabled      bool              `json:"enabled" yaml:"enabled"`
	Providers    map[string]OAuth2Provider `json:"providers" yaml:"providers"`
	RedirectURL  string            `json:"redirect_url" yaml:"redirect_url"`
}

// OAuth2Provider represents an OAuth2 provider configuration
type OAuth2Provider struct {
	ClientID     string   `json:"client_id" yaml:"client_id"`
	ClientSecret string   `json:"client_secret" yaml:"client_secret"`
	AuthURL      string   `json:"auth_url" yaml:"auth_url"`
	TokenURL     string   `json:"token_url" yaml:"token_url"`
	UserInfoURL  string   `json:"user_info_url" yaml:"user_info_url"`
	Scopes       []string `json:"scopes" yaml:"scopes"`
}

// DefaultConfig returns a default configuration for the user management system
func DefaultConfig() *Config {
	return &Config{
		DatabaseType: "sqlite",
		DatabasePath: "./data/memos_users.db",
		
		AuthMethod:         "jwt",
		JWTExpirationTime:  24 * time.Hour,
		SessionTimeout:     8 * time.Hour,
		RefreshTokenExpiry: 7 * 24 * time.Hour,

		PasswordPolicy: PasswordPolicy{
			MinLength:        8,
			RequireUppercase: true,
			RequireLowercase: true,
			RequireNumbers:   true,
			RequireSymbols:   false,
			MaxAge:           90 * 24 * time.Hour, // 90 days
			PreventReuse:     5,
		},

		MaxLoginAttempts:    5,
		LockoutDuration:     15 * time.Minute,
		RequireMFA:          false,
		AllowGuestAccess:    true,
		EnableAuditLogging:  true,
		EnableRateLimiting:  true,

		DefaultRole:      RoleUser,
		AllowSelfSignup:  false,
		RequireEmailVerification: false,

		LDAP: LDAPConfig{
			Enabled: false,
			Port:    389,
		},

		OAuth: OAuthConfig{
			Enabled:   false,
			Providers: make(map[string]OAuth2Provider),
		},
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.DatabaseType == "" {
		return NewValidationError("database_type is required")
	}

	if c.DatabaseType == "sqlite" && c.DatabasePath == "" {
		return NewValidationError("database_path is required for SQLite")
	}

	if c.DatabaseType != "sqlite" && c.DatabaseURL == "" {
		return NewValidationError("database_url is required for non-SQLite databases")
	}

	if c.AuthMethod == "" {
		return NewValidationError("auth_method is required")
	}

	if c.AuthMethod == "jwt" && c.JWTSecret == "" {
		return NewValidationError("jwt_secret is required for JWT authentication")
	}

	if !c.DefaultRole.IsValid() {
		return NewValidationError("invalid default_role")
	}

	if c.PasswordPolicy.MinLength < 4 {
		return NewValidationError("password minimum length must be at least 4")
	}

	if c.MaxLoginAttempts < 1 {
		return NewValidationError("max_login_attempts must be at least 1")
	}

	if c.LockoutDuration < time.Minute {
		return NewValidationError("lockout_duration must be at least 1 minute")
	}

	return nil
}

// IsLocalAuth returns true if using local authentication
func (c *Config) IsLocalAuth() bool {
	return c.AuthMethod == "local" || c.AuthMethod == "jwt"
}

// IsLDAPAuth returns true if using LDAP authentication
func (c *Config) IsLDAPAuth() bool {
	return c.AuthMethod == "ldap" && c.LDAP.Enabled
}

// IsOAuthAuth returns true if using OAuth authentication
func (c *Config) IsOAuthAuth() bool {
	return c.AuthMethod == "oauth" && c.OAuth.Enabled
}

// GetEnabledOAuthProviders returns a list of enabled OAuth providers
func (c *Config) GetEnabledOAuthProviders() []string {
	if !c.OAuth.Enabled {
		return nil
	}

	providers := make([]string, 0, len(c.OAuth.Providers))
	for name := range c.OAuth.Providers {
		providers = append(providers, name)
	}
	return providers
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

// NewValidationError creates a new validation error
func NewValidationError(message string) error {
	return ValidationError{Message: message}
}