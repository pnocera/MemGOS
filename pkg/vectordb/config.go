package vectordb

import (
	"errors"
	"fmt"
	"time"

	"github.com/memtensor/memgos/pkg/types"
)

// QdrantConfig represents Qdrant-specific configuration
type QdrantConfig struct {
	// Connection settings
	Host string `json:"host" validate:"required"`
	Port int    `json:"port" validate:"required,min=1,max=65535"`

	// Collection settings
	CollectionName string `json:"collection_name" validate:"required"`
	VectorSize     int    `json:"vector_size" validate:"required,min=1"`

	// Authentication
	APIKey   string `json:"api_key,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	// Connection options
	UseHTTPS         bool          `json:"use_https"`
	ConnectionTimeout time.Duration `json:"connection_timeout"`
	RequestTimeout   time.Duration `json:"request_timeout"`

	// Performance settings
	MaxRetries      int           `json:"max_retries"`
	RetryInterval   time.Duration `json:"retry_interval"`
	MaxConnections  int           `json:"max_connections"`
	BatchSize       int           `json:"batch_size"`

	// Vector search settings
	SearchLimit     int     `json:"search_limit"`
	DefaultTopK     int     `json:"default_top_k"`
	ScoreThreshold  float32 `json:"score_threshold"`

	// Collection configuration
	Distance         string `json:"distance"` // "cosine", "euclidean", "dot"
	OnDiskPayload    bool   `json:"on_disk_payload"`
	QuantizationConfig *QuantizationConfig `json:"quantization_config,omitempty"`
}

// QuantizationConfig represents quantization settings for Qdrant
type QuantizationConfig struct {
	Scalar *ScalarQuantization `json:"scalar,omitempty"`
	Binary *BinaryQuantization `json:"binary,omitempty"`
}

// ScalarQuantization represents scalar quantization settings
type ScalarQuantization struct {
	Type      string  `json:"type"`      // "int8"
	Quantile  float32 `json:"quantile"`  // 0.99
	AlwaysRam bool    `json:"always_ram"`
}

// BinaryQuantization represents binary quantization settings
type BinaryQuantization struct {
	AlwaysRam bool `json:"always_ram"`
}

// VectorDBConfig represents the general vector database configuration
type VectorDBConfig struct {
	Backend types.BackendType `json:"backend" validate:"required"`
	Qdrant  *QdrantConfig     `json:"qdrant,omitempty"`
	
	// General settings
	EnableMetrics     bool `json:"enable_metrics"`
	EnableHealthCheck bool `json:"enable_health_check"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	
	// Batch operation settings
	DefaultBatchSize    int           `json:"default_batch_size"`
	MaxBatchSize        int           `json:"max_batch_size"`
	BatchTimeout        time.Duration `json:"batch_timeout"`
	
	// Circuit breaker settings
	EnableCircuitBreaker bool          `json:"enable_circuit_breaker"`
	FailureThreshold     int           `json:"failure_threshold"`
	RecoveryTimeout      time.Duration `json:"recovery_timeout"`
}

// DefaultQdrantConfig returns a default Qdrant configuration
func DefaultQdrantConfig() *QdrantConfig {
	return &QdrantConfig{
		Host:             "localhost",
		Port:             6333,
		CollectionName:   "memgos_vectors",
		VectorSize:       768,
		UseHTTPS:         false,
		ConnectionTimeout: 30 * time.Second,
		RequestTimeout:   60 * time.Second,
		MaxRetries:       3,
		RetryInterval:    time.Second,
		MaxConnections:   10,
		BatchSize:        100,
		SearchLimit:      1000,
		DefaultTopK:      10,
		ScoreThreshold:   0.0,
		Distance:         "cosine",
		OnDiskPayload:    false,
	}
}

// DefaultVectorDBConfig returns a default vector database configuration
func DefaultVectorDBConfig() *VectorDBConfig {
	return &VectorDBConfig{
		Backend:              types.BackendQdrant,
		Qdrant:              DefaultQdrantConfig(),
		EnableMetrics:       true,
		EnableHealthCheck:   true,
		HealthCheckInterval: 30 * time.Second,
		DefaultBatchSize:    50,
		MaxBatchSize:        1000,
		BatchTimeout:        30 * time.Second,
		EnableCircuitBreaker: true,
		FailureThreshold:    5,
		RecoveryTimeout:     60 * time.Second,
	}
}

// Validate validates the vector database configuration
func (c *VectorDBConfig) Validate() error {
	if c.Backend == "" {
		return errors.New("backend type is required")
	}

	switch c.Backend {
	case types.BackendQdrant:
		if c.Qdrant == nil {
			return errors.New("qdrant configuration is required when backend is qdrant")
		}
		return c.Qdrant.Validate()
	default:
		return fmt.Errorf("unsupported backend type: %s", c.Backend)
	}
}

// Validate validates the Qdrant configuration
func (c *QdrantConfig) Validate() error {
	if c.Host == "" {
		return errors.New("host is required")
	}

	if c.Port <= 0 || c.Port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}

	if c.CollectionName == "" {
		return errors.New("collection_name is required")
	}

	if c.VectorSize <= 0 {
		return errors.New("vector_size must be greater than 0")
	}

	if c.Distance != "" {
		validDistances := []string{"cosine", "euclidean", "dot"}
		valid := false
		for _, d := range validDistances {
			if c.Distance == d {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid distance metric: %s, must be one of: %v", c.Distance, validDistances)
		}
	}

	if c.MaxRetries < 0 {
		return errors.New("max_retries cannot be negative")
	}

	if c.BatchSize <= 0 {
		return errors.New("batch_size must be greater than 0")
	}

	if c.SearchLimit <= 0 {
		return errors.New("search_limit must be greater than 0")
	}

	if c.DefaultTopK <= 0 {
		return errors.New("default_top_k must be greater than 0")
	}

	if c.ScoreThreshold < 0 {
		return errors.New("score_threshold cannot be negative")
	}

	return nil
}

// GetConnectionURL returns the connection URL for Qdrant
func (c *QdrantConfig) GetConnectionURL() string {
	scheme := "http"
	if c.UseHTTPS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, c.Host, c.Port)
}

// GetDistance returns the distance metric for Qdrant collection
func (c *QdrantConfig) GetDistance() string {
	if c.Distance == "" {
		return "cosine"
	}
	return c.Distance
}

// Clone creates a deep copy of the configuration
func (c *QdrantConfig) Clone() *QdrantConfig {
	clone := *c
	
	// Clone quantization config if present
	if c.QuantizationConfig != nil {
		clone.QuantizationConfig = &QuantizationConfig{}
		if c.QuantizationConfig.Scalar != nil {
			scalar := *c.QuantizationConfig.Scalar
			clone.QuantizationConfig.Scalar = &scalar
		}
		if c.QuantizationConfig.Binary != nil {
			binary := *c.QuantizationConfig.Binary
			clone.QuantizationConfig.Binary = &binary
		}
	}
	
	return &clone
}

// Clone creates a deep copy of the vector database configuration
func (c *VectorDBConfig) Clone() *VectorDBConfig {
	clone := *c
	
	// Clone Qdrant config if present
	if c.Qdrant != nil {
		clone.Qdrant = c.Qdrant.Clone()
	}
	
	return &clone
}

// MergeFromEnv merges configuration from environment variables
func (c *QdrantConfig) MergeFromEnv() {
	// This would typically use os.Getenv to read from environment
	// For now, we'll leave it as a placeholder for future implementation
}

// ToConnectionParams converts config to connection parameters for Qdrant client
func (c *QdrantConfig) ToConnectionParams() map[string]interface{} {
	params := map[string]interface{}{
		"host":    c.Host,
		"port":    c.Port,
		"https":   c.UseHTTPS,
		"timeout": c.ConnectionTimeout,
	}

	if c.APIKey != "" {
		params["api_key"] = c.APIKey
	}

	if c.Username != "" && c.Password != "" {
		params["username"] = c.Username
		params["password"] = c.Password
	}

	return params
}