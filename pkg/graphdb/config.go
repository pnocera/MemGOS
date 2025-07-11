package graphdb

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// ConfigManager manages graph database configurations
type ConfigManager struct {
	configs map[string]*GraphDBConfig
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		configs: make(map[string]*GraphDBConfig),
	}
}

// LoadFromFile loads configuration from a file
func (cm *ConfigManager) LoadFromFile(filePath string) error {
	v := viper.New()
	v.SetConfigFile(filePath)
	
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Load multiple database configurations
	graphDBs := v.GetStringMap("graph_databases")
	for name, configData := range graphDBs {
		v.Set("current", configData)
		
		config := &GraphDBConfig{}
		if err := v.UnmarshalKey("current", config); err != nil {
			return fmt.Errorf("failed to unmarshal config for %s: %w", name, err)
		}
		
		// Set defaults if not specified
		cm.setDefaults(config)
		
		cm.configs[name] = config
	}

	return nil
}

// LoadFromEnv loads configuration from environment variables
func (cm *ConfigManager) LoadFromEnv(prefix string) *GraphDBConfig {
	v := viper.New()
	v.SetEnvPrefix(prefix)
	v.AutomaticEnv()

	config := &GraphDBConfig{
		Provider:      GraphDBProvider(v.GetString("PROVIDER")),
		URI:          v.GetString("URI"),
		Username:     v.GetString("USERNAME"),
		Password:     v.GetString("PASSWORD"),
		Database:     v.GetString("DATABASE"),
		MaxConnPool:  v.GetInt("MAX_CONN_POOL"),
		ConnTimeout:  v.GetDuration("CONN_TIMEOUT"),
		ReadTimeout:  v.GetDuration("READ_TIMEOUT"),
		WriteTimeout: v.GetDuration("WRITE_TIMEOUT"),
		RetryAttempts: v.GetInt("RETRY_ATTEMPTS"),
		RetryDelay:   v.GetDuration("RETRY_DELAY"),
		Metrics:      v.GetBool("METRICS"),
		Logging:      v.GetBool("LOGGING"),
		SSLMode:      v.GetString("SSL_MODE"),
	}

	cm.setDefaults(config)
	return config
}

// setDefaults sets default values for configuration
func (cm *ConfigManager) setDefaults(config *GraphDBConfig) {
	if config.Provider == "" {
		config.Provider = ProviderNeo4j
	}
	
	// Provider-specific defaults
	switch config.Provider {
	case ProviderNeo4j:
		if config.URI == "" {
			config.URI = "bolt://localhost:7687"
		}
		if config.Username == "" {
			config.Username = "neo4j"
		}
		if config.Password == "" {
			config.Password = "password"
		}
		if config.Database == "" {
			config.Database = "neo4j"
		}
		if config.MaxConnPool == 0 {
			config.MaxConnPool = 50
		}
		if config.SSLMode == "" {
			config.SSLMode = "disable"
		}
	case ProviderKuzu:
		if config.KuzuConfig == nil {
			config.KuzuConfig = &KuzuDBConfig{
				DatabasePath:      "./kuzu_data",
				ReadOnly:          false,
				BufferPoolSize:    1024 * 1024 * 1024, // 1GB
				MaxNumThreads:     4,
				EnableCompression: true,
				TimeoutSeconds:    30,
			}
		} else {
			// Set individual defaults for KuzuDB config
			if config.KuzuConfig.DatabasePath == "" {
				config.KuzuConfig.DatabasePath = "./kuzu_data"
			}
			if config.KuzuConfig.BufferPoolSize == 0 {
				config.KuzuConfig.BufferPoolSize = 1024 * 1024 * 1024 // 1GB
			}
			if config.KuzuConfig.MaxNumThreads == 0 {
				config.KuzuConfig.MaxNumThreads = 4
			}
			if config.KuzuConfig.TimeoutSeconds == 0 {
				config.KuzuConfig.TimeoutSeconds = 30
			}
		}
		if config.MaxConnPool == 0 {
			config.MaxConnPool = 1 // KuzuDB is embedded, single connection
		}
	}
	
	// Common defaults
	if config.ConnTimeout == 0 {
		config.ConnTimeout = 30 * time.Second
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 15 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 15 * time.Second
	}
	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = time.Second
	}
}

// GetConfig retrieves a configuration by name
func (cm *ConfigManager) GetConfig(name string) (*GraphDBConfig, bool) {
	config, exists := cm.configs[name]
	return config, exists
}

// AddConfig adds a new configuration
func (cm *ConfigManager) AddConfig(name string, config *GraphDBConfig) {
	cm.setDefaults(config)
	cm.configs[name] = config
}

// ListConfigs returns all configuration names
func (cm *ConfigManager) ListConfigs() []string {
	names := make([]string, 0, len(cm.configs))
	for name := range cm.configs {
		names = append(names, name)
	}
	return names
}

// ValidateConfig validates a graph database configuration
func (cm *ConfigManager) ValidateConfig(config *GraphDBConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is nil")
	}

	if config.Provider == "" {
		return fmt.Errorf("provider is required")
	}

	// Provider-specific validation
	switch config.Provider {
	case ProviderNeo4j:
		if config.URI == "" {
			return fmt.Errorf("URI is required for Neo4j")
		}
		if config.Username == "" {
			return fmt.Errorf("username is required for Neo4j")
		}
		if config.Password == "" {
			return fmt.Errorf("password is required for Neo4j")
		}
	case ProviderKuzu:
		if config.KuzuConfig == nil {
			return fmt.Errorf("KuzuDB configuration is required")
		}
		if config.KuzuConfig.DatabasePath == "" {
			return fmt.Errorf("database path is required for KuzuDB")
		}
		if config.KuzuConfig.BufferPoolSize <= 0 {
			return fmt.Errorf("buffer pool size must be positive for KuzuDB")
		}
		if config.KuzuConfig.MaxNumThreads <= 0 {
			return fmt.Errorf("max number of threads must be positive for KuzuDB")
		}
		if config.KuzuConfig.TimeoutSeconds <= 0 {
			return fmt.Errorf("timeout seconds must be positive for KuzuDB")
		}
	default:
		return fmt.Errorf("unsupported provider: %s", config.Provider)
	}

	if config.MaxConnPool <= 0 {
		return fmt.Errorf("max connection pool must be positive")
	}

	if config.ConnTimeout <= 0 {
		return fmt.Errorf("connection timeout must be positive")
	}

	if config.ReadTimeout <= 0 {
		return fmt.Errorf("read timeout must be positive")
	}

	if config.WriteTimeout <= 0 {
		return fmt.Errorf("write timeout must be positive")
	}

	if config.RetryAttempts < 0 {
		return fmt.Errorf("retry attempts cannot be negative")
	}

	if config.RetryDelay < 0 {
		return fmt.Errorf("retry delay cannot be negative")
	}

	return nil
}

// Neo4jSpecificConfig holds Neo4j-specific configuration options
type Neo4jSpecificConfig struct {
	// Connection settings
	UserAgent              string        `json:"user_agent" yaml:"user_agent"`
	Resolver               string        `json:"resolver" yaml:"resolver"`
	AddressResolver        string        `json:"address_resolver" yaml:"address_resolver"`
	MaxConnectionLifetime  time.Duration `json:"max_connection_lifetime" yaml:"max_connection_lifetime"`
	MaxConnectionPoolSize  int           `json:"max_connection_pool_size" yaml:"max_connection_pool_size"`
	ConnectionLivenessCheckTimeout time.Duration `json:"connection_liveness_check_timeout" yaml:"connection_liveness_check_timeout"`
	
	// Security settings
	RootCAs                []string      `json:"root_cas" yaml:"root_cas"`
	InsecureSkipVerify     bool          `json:"insecure_skip_verify" yaml:"insecure_skip_verify"`
	ClientCertificate      string        `json:"client_certificate" yaml:"client_certificate"`
	ClientKey              string        `json:"client_key" yaml:"client_key"`
	
	// Routing settings
	RoutingContext         map[string]string `json:"routing_context" yaml:"routing_context"`
	FetchSize              int           `json:"fetch_size" yaml:"fetch_size"`
	
	// Transaction settings
	DefaultAccessMode      string        `json:"default_access_mode" yaml:"default_access_mode"`
	Bookmarks              []string      `json:"bookmarks" yaml:"bookmarks"`
	TransactionTimeout     time.Duration `json:"transaction_timeout" yaml:"transaction_timeout"`
	
	// Logging and debugging
	LogLevel               string        `json:"log_level" yaml:"log_level"`
	EnableTracing          bool          `json:"enable_tracing" yaml:"enable_tracing"`
	MetricsEnabled         bool          `json:"metrics_enabled" yaml:"metrics_enabled"`
}

// DefaultNeo4jSpecificConfig returns default Neo4j-specific configuration
func DefaultNeo4jSpecificConfig() *Neo4jSpecificConfig {
	return &Neo4jSpecificConfig{
		UserAgent:              "MemGOS-GraphDB/1.0",
		MaxConnectionLifetime:  time.Hour,
		MaxConnectionPoolSize:  50,
		ConnectionLivenessCheckTimeout: time.Minute,
		InsecureSkipVerify:     false,
		FetchSize:              1000,
		DefaultAccessMode:      "READ",
		TransactionTimeout:     30 * time.Second,
		LogLevel:               "INFO",
		EnableTracing:          false,
		MetricsEnabled:         true,
	}
}

// PerformanceConfig holds performance-related configuration
type PerformanceConfig struct {
	// Query optimization
	QueryCacheSize         int           `json:"query_cache_size" yaml:"query_cache_size"`
	QueryTimeout           time.Duration `json:"query_timeout" yaml:"query_timeout"`
	ParallelQueries        int           `json:"parallel_queries" yaml:"parallel_queries"`
	
	// Batch operation settings
	BatchSize              int           `json:"batch_size" yaml:"batch_size"`
	MaxBatchSize           int           `json:"max_batch_size" yaml:"max_batch_size"`
	BatchTimeout           time.Duration `json:"batch_timeout" yaml:"batch_timeout"`
	
	// Memory management
	MaxMemoryUsage         int64         `json:"max_memory_usage" yaml:"max_memory_usage"`
	GCInterval             time.Duration `json:"gc_interval" yaml:"gc_interval"`
	
	// Caching
	EnableResultCache      bool          `json:"enable_result_cache" yaml:"enable_result_cache"`
	ResultCacheSize        int           `json:"result_cache_size" yaml:"result_cache_size"`
	ResultCacheTTL         time.Duration `json:"result_cache_ttl" yaml:"result_cache_ttl"`
	
	// Indexing
	AutoCreateIndexes      bool          `json:"auto_create_indexes" yaml:"auto_create_indexes"`
	IndexCreationTimeout   time.Duration `json:"index_creation_timeout" yaml:"index_creation_timeout"`
}

// DefaultPerformanceConfig returns default performance configuration
func DefaultPerformanceConfig() *PerformanceConfig {
	return &PerformanceConfig{
		QueryCacheSize:        1000,
		QueryTimeout:          30 * time.Second,
		ParallelQueries:       4,
		BatchSize:             1000,
		MaxBatchSize:          10000,
		BatchTimeout:          5 * time.Minute,
		MaxMemoryUsage:        1024 * 1024 * 1024, // 1GB
		GCInterval:            5 * time.Minute,
		EnableResultCache:     true,
		ResultCacheSize:       500,
		ResultCacheTTL:        10 * time.Minute,
		AutoCreateIndexes:     false,
		IndexCreationTimeout:  time.Minute,
	}
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	// Authentication
	AuthType               string            `json:"auth_type" yaml:"auth_type"`
	KerberosServiceName    string            `json:"kerberos_service_name" yaml:"kerberos_service_name"`
	KerberosRealm          string            `json:"kerberos_realm" yaml:"kerberos_realm"`
	
	// Authorization
	Roles                  []string          `json:"roles" yaml:"roles"`
	Permissions            map[string]string `json:"permissions" yaml:"permissions"`
	
	// Encryption
	EncryptionLevel        string            `json:"encryption_level" yaml:"encryption_level"`
	TrustStrategy          string            `json:"trust_strategy" yaml:"trust_strategy"`
	
	// Audit
	EnableAuditLog         bool              `json:"enable_audit_log" yaml:"enable_audit_log"`
	AuditLogPath           string            `json:"audit_log_path" yaml:"audit_log_path"`
	AuditLogLevel          string            `json:"audit_log_level" yaml:"audit_log_level"`
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		AuthType:        "basic",
		EncryptionLevel: "optional",
		TrustStrategy:   "trust_system_ca_signed_certificates",
		EnableAuditLog:  false,
		AuditLogLevel:   "INFO",
	}
}

// ExtendedGraphDBConfig combines all configuration types
type ExtendedGraphDBConfig struct {
	*GraphDBConfig
	Neo4jConfig      *Neo4jSpecificConfig `json:"neo4j_config,omitempty" yaml:"neo4j_config,omitempty"`
	PerformanceConfig *PerformanceConfig   `json:"performance_config,omitempty" yaml:"performance_config,omitempty"`
	SecurityConfig   *SecurityConfig      `json:"security_config,omitempty" yaml:"security_config,omitempty"`
}

// NewExtendedGraphDBConfig creates a new extended configuration with defaults
func NewExtendedGraphDBConfig() *ExtendedGraphDBConfig {
	return &ExtendedGraphDBConfig{
		GraphDBConfig:     DefaultGraphDBConfig(),
		Neo4jConfig:       DefaultNeo4jSpecificConfig(),
		PerformanceConfig: DefaultPerformanceConfig(),
		SecurityConfig:    DefaultSecurityConfig(),
	}
}

// Validate validates the extended configuration
func (config *ExtendedGraphDBConfig) Validate() error {
	cm := NewConfigManager()
	
	if err := cm.ValidateConfig(config.GraphDBConfig); err != nil {
		return fmt.Errorf("invalid base config: %w", err)
	}
	
	// Validate Neo4j-specific config
	if config.Neo4jConfig != nil {
		if config.Neo4jConfig.MaxConnectionPoolSize <= 0 {
			return fmt.Errorf("max connection pool size must be positive")
		}
		if config.Neo4jConfig.MaxConnectionLifetime <= 0 {
			return fmt.Errorf("max connection lifetime must be positive")
		}
	}
	
	// Validate performance config
	if config.PerformanceConfig != nil {
		if config.PerformanceConfig.BatchSize <= 0 {
			return fmt.Errorf("batch size must be positive")
		}
		if config.PerformanceConfig.MaxBatchSize < config.PerformanceConfig.BatchSize {
			return fmt.Errorf("max batch size must be >= batch size")
		}
	}
	
	return nil
}

// ConfigBuilder helps build configurations fluently
type ConfigBuilder struct {
	config *ExtendedGraphDBConfig
}

// NewConfigBuilder creates a new configuration builder
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: NewExtendedGraphDBConfig(),
	}
}

// Provider sets the database provider
func (cb *ConfigBuilder) Provider(provider GraphDBProvider) *ConfigBuilder {
	cb.config.Provider = provider
	return cb
}

// URI sets the database URI
func (cb *ConfigBuilder) URI(uri string) *ConfigBuilder {
	cb.config.URI = uri
	return cb
}

// Credentials sets the database credentials
func (cb *ConfigBuilder) Credentials(username, password string) *ConfigBuilder {
	cb.config.Username = username
	cb.config.Password = password
	return cb
}

// Database sets the database name
func (cb *ConfigBuilder) Database(database string) *ConfigBuilder {
	cb.config.Database = database
	return cb
}

// ConnectionPool sets connection pool configuration
func (cb *ConfigBuilder) ConnectionPool(maxSize int, timeout time.Duration) *ConfigBuilder {
	cb.config.MaxConnPool = maxSize
	cb.config.ConnTimeout = timeout
	return cb
}

// Timeouts sets various timeout configurations
func (cb *ConfigBuilder) Timeouts(read, write time.Duration) *ConfigBuilder {
	cb.config.ReadTimeout = read
	cb.config.WriteTimeout = write
	return cb
}

// Retry sets retry configuration
func (cb *ConfigBuilder) Retry(attempts int, delay time.Duration) *ConfigBuilder {
	cb.config.RetryAttempts = attempts
	cb.config.RetryDelay = delay
	return cb
}

// EnableMetrics enables metrics collection
func (cb *ConfigBuilder) EnableMetrics() *ConfigBuilder {
	cb.config.Metrics = true
	return cb
}

// EnableLogging enables logging
func (cb *ConfigBuilder) EnableLogging() *ConfigBuilder {
	cb.config.Logging = true
	return cb
}

// SSL sets SSL configuration
func (cb *ConfigBuilder) SSL(mode string) *ConfigBuilder {
	cb.config.SSLMode = mode
	return cb
}

// Build returns the built configuration
func (cb *ConfigBuilder) Build() *ExtendedGraphDBConfig {
	return cb.config
}

// BuildAndValidate builds and validates the configuration
func (cb *ConfigBuilder) BuildAndValidate() (*ExtendedGraphDBConfig, error) {
	config := cb.Build()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return config, nil
}
