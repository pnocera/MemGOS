// Package config provides configuration management for MemGOS
package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// BaseConfig provides common configuration functionality
type BaseConfig struct {
	ModelSchema string `yaml:"model_schema,omitempty" json:"model_schema,omitempty" validate:"required"`
	mu          sync.RWMutex
	validator   *validator.Validate
}

// NewBaseConfig creates a new base configuration
func NewBaseConfig() *BaseConfig {
	return &BaseConfig{
		validator: validator.New(),
	}
}

// Validate validates the configuration
func (c *BaseConfig) Validate() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return c.validator.Struct(c)
}

// FromJSONFile loads configuration from a JSON file
func (c *BaseConfig) FromJSONFile(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("json")
	
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	
	return v.Unmarshal(c)
}

// FromYAMLFile loads configuration from a YAML file
func (c *BaseConfig) FromYAMLFile(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	
	return v.Unmarshal(c)
}

// ToJSONFile saves configuration to a JSON file
func (c *BaseConfig) ToJSONFile(path string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	v := viper.New()
	v.SetConfigType("json")
	
	// Use reflection to set values
	c.setViperValues(v, c)
	
	return v.WriteConfigAs(path)
}

// ToYAMLFile saves configuration to a YAML file
func (c *BaseConfig) ToYAMLFile(path string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	return os.WriteFile(path, data, 0644)
}

// setViperValues uses reflection to set viper values
func (c *BaseConfig) setViperValues(v *viper.Viper, config interface{}) {
	val := reflect.ValueOf(config)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		
		if !field.CanInterface() {
			continue
		}
		
		tagName := fieldType.Tag.Get("json")
		if tagName == "" {
			tagName = strings.ToLower(fieldType.Name)
		} else {
			tagName = strings.Split(tagName, ",")[0]
		}
		
		if tagName != "" && tagName != "-" {
			v.Set(tagName, field.Interface())
		}
	}
}

// Get retrieves a configuration value
func (c *BaseConfig) Get(key string, defaultValue interface{}) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Use reflection to get field value
	val := reflect.ValueOf(c).Elem()
	typ := val.Type()
	
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		
		tagName := fieldType.Tag.Get("json")
		if tagName == "" {
			tagName = strings.ToLower(fieldType.Name)
		} else {
			tagName = strings.Split(tagName, ",")[0]
		}
		
		if tagName == key {
			if field.CanInterface() {
				return field.Interface()
			}
		}
	}
	
	return defaultValue
}

// LLMConfig represents LLM configuration
type LLMConfig struct {
	BaseConfig `yaml:",inline"`
	Backend    types.BackendType `yaml:"backend" json:"backend" validate:"required,oneof=openai ollama huggingface"`
	Model      string            `yaml:"model" json:"model" validate:"required"`
	APIKey     string            `yaml:"api_key,omitempty" json:"api_key,omitempty"`
	BaseURL    string            `yaml:"base_url,omitempty" json:"base_url,omitempty"`
	MaxTokens  int               `yaml:"max_tokens,omitempty" json:"max_tokens,omitempty"`
	Temperature float64          `yaml:"temperature,omitempty" json:"temperature,omitempty"`
	TopP       float64           `yaml:"top_p,omitempty" json:"top_p,omitempty"`
	Timeout    time.Duration     `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

// NewLLMConfig creates a new LLM configuration
func NewLLMConfig() *LLMConfig {
	return &LLMConfig{
		BaseConfig:  *NewBaseConfig(),
		MaxTokens:   1024,
		Temperature: 0.7,
		TopP:        0.9,
		Timeout:     30 * time.Second,
	}
}

// EmbedderConfig represents embedder configuration
type EmbedderConfig struct {
	BaseConfig `yaml:",inline"`
	Backend    types.BackendType `yaml:"backend" json:"backend" validate:"required,oneof=openai ollama huggingface"`
	Model      string            `yaml:"model" json:"model" validate:"required"`
	APIKey     string            `yaml:"api_key,omitempty" json:"api_key,omitempty"`
	BaseURL    string            `yaml:"base_url,omitempty" json:"base_url,omitempty"`
	Dimension  int               `yaml:"dimension,omitempty" json:"dimension,omitempty"`
	Timeout    time.Duration     `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

// NewEmbedderConfig creates a new embedder configuration
func NewEmbedderConfig() *EmbedderConfig {
	return &EmbedderConfig{
		BaseConfig: *NewBaseConfig(),
		Dimension:  768,
		Timeout:    30 * time.Second,
	}
}

// VectorDBConfig represents vector database configuration
type VectorDBConfig struct {
	BaseConfig `yaml:",inline"`
	Backend    types.BackendType `yaml:"backend" json:"backend" validate:"required,oneof=qdrant"`
	Host       string            `yaml:"host" json:"host" validate:"required"`
	Port       int               `yaml:"port" json:"port" validate:"required,gt=0"`
	APIKey     string            `yaml:"api_key,omitempty" json:"api_key,omitempty"`
	Collection string            `yaml:"collection" json:"collection" validate:"required"`
	Dimension  int               `yaml:"dimension" json:"dimension" validate:"required,gt=0"`
	Timeout    time.Duration     `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

// NewVectorDBConfig creates a new vector database configuration
func NewVectorDBConfig() *VectorDBConfig {
	return &VectorDBConfig{
		BaseConfig: *NewBaseConfig(),
		Host:       "localhost",
		Port:       6333,
		Dimension:  768,
		Timeout:    30 * time.Second,
	}
}

// GraphDBConfig represents graph database configuration
type GraphDBConfig struct {
	BaseConfig `yaml:",inline"`
	Backend    types.BackendType `yaml:"backend" json:"backend" validate:"required,oneof=neo4j"`
	URI        string            `yaml:"uri" json:"uri" validate:"required"`
	Username   string            `yaml:"username" json:"username" validate:"required"`
	Password   string            `yaml:"password" json:"password" validate:"required"`
	Database   string            `yaml:"database,omitempty" json:"database,omitempty"`
	Timeout    time.Duration     `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

// NewGraphDBConfig creates a new graph database configuration
func NewGraphDBConfig() *GraphDBConfig {
	return &GraphDBConfig{
		BaseConfig: *NewBaseConfig(),
		URI:        "bolt://localhost:7687",
		Username:   "neo4j",
		Database:   "neo4j",
		Timeout:    30 * time.Second,
	}
}

// MemoryConfig represents memory configuration
type MemoryConfig struct {
	BaseConfig       `yaml:",inline"`
	Backend          types.MemoryBackend `yaml:"backend" json:"backend" validate:"required"`
	MemoryFilename   string              `yaml:"memory_filename" json:"memory_filename" validate:"required"`
	TopK             int                 `yaml:"top_k,omitempty" json:"top_k,omitempty"`
	ChunkSize        int                 `yaml:"chunk_size,omitempty" json:"chunk_size,omitempty"`
	ChunkOverlap     int                 `yaml:"chunk_overlap,omitempty" json:"chunk_overlap,omitempty"`
	EmbedderConfig   *EmbedderConfig     `yaml:"embedder,omitempty" json:"embedder,omitempty"`
	VectorDBConfig   *VectorDBConfig     `yaml:"vector_db,omitempty" json:"vector_db,omitempty"`
	GraphDBConfig    *GraphDBConfig      `yaml:"graph_db,omitempty" json:"graph_db,omitempty"`
}

// NewMemoryConfig creates a new memory configuration
func NewMemoryConfig() *MemoryConfig {
	return &MemoryConfig{
		BaseConfig:     *NewBaseConfig(),
		Backend:        types.MemoryBackendNaive,
		MemoryFilename: "memory.json",
		TopK:           5,
		ChunkSize:      1000,
		ChunkOverlap:   200,
	}
}

// MemCubeConfig represents memory cube configuration
type MemCubeConfig struct {
	BaseConfig `yaml:",inline"`
	TextMem    *MemoryConfig `yaml:"text_mem,omitempty" json:"text_mem,omitempty"`
	ActMem     *MemoryConfig `yaml:"act_mem,omitempty" json:"act_mem,omitempty"`
	ParaMem    *MemoryConfig `yaml:"para_mem,omitempty" json:"para_mem,omitempty"`
}

// NewMemCubeConfig creates a new memory cube configuration
func NewMemCubeConfig() *MemCubeConfig {
	return &MemCubeConfig{
		BaseConfig: *NewBaseConfig(),
		TextMem:    NewMemoryConfig(),
		ActMem:     NewMemoryConfig(),
		ParaMem:    NewMemoryConfig(),
	}
}

// SchedulerConfig represents scheduler configuration
type SchedulerConfig struct {
	BaseConfig           `yaml:",inline"`
	Enabled              bool              `yaml:"enabled" json:"enabled"`
	
	// NATS Configuration
	NATSURLs             []string          `yaml:"nats_urls,omitempty" json:"nats_urls,omitempty"`
	NATSUsername         string            `yaml:"nats_username,omitempty" json:"nats_username,omitempty"`
	NATSPassword         string            `yaml:"nats_password,omitempty" json:"nats_password,omitempty"`
	NATSToken            string            `yaml:"nats_token,omitempty" json:"nats_token,omitempty"`
	NATSMaxReconnect     int               `yaml:"nats_max_reconnect,omitempty" json:"nats_max_reconnect,omitempty"`
	
	// NATS KV Configuration
	UseNATSKV            bool              `yaml:"use_nats_kv" json:"use_nats_kv"`
	NATSKVBucketName     string            `yaml:"nats_kv_bucket_name,omitempty" json:"nats_kv_bucket_name,omitempty"`
	NATSKVDescription    string            `yaml:"nats_kv_description,omitempty" json:"nats_kv_description,omitempty"`
	NATSKVMaxValueSize   int32             `yaml:"nats_kv_max_value_size,omitempty" json:"nats_kv_max_value_size,omitempty"`
	NATSKVHistory        uint8             `yaml:"nats_kv_history,omitempty" json:"nats_kv_history,omitempty"`
	NATSKVMaxBytes       int64             `yaml:"nats_kv_max_bytes,omitempty" json:"nats_kv_max_bytes,omitempty"`
	NATSKVStorage        string            `yaml:"nats_kv_storage,omitempty" json:"nats_kv_storage,omitempty"`
	NATSKVReplicas       int               `yaml:"nats_kv_replicas,omitempty" json:"nats_kv_replicas,omitempty"`
	
	// JetStream Configuration
	StreamName           string            `yaml:"stream_name,omitempty" json:"stream_name,omitempty"`
	StreamSubjects       []string          `yaml:"stream_subjects,omitempty" json:"stream_subjects,omitempty"`
	ConsumerName         string            `yaml:"consumer_name,omitempty" json:"consumer_name,omitempty"`
	ConsumerDurable      bool              `yaml:"consumer_durable" json:"consumer_durable"`
	MaxDeliver           int               `yaml:"max_deliver,omitempty" json:"max_deliver,omitempty"`
	AckWait              time.Duration     `yaml:"ack_wait,omitempty" json:"ack_wait,omitempty"`
	MaxAckPending        int               `yaml:"max_ack_pending,omitempty" json:"max_ack_pending,omitempty"`
	
	// Scheduler Configuration
	ThreadPoolMaxWorkers int               `yaml:"thread_pool_max_workers,omitempty" json:"thread_pool_max_workers,omitempty"`
	EnableParallelDispatch bool            `yaml:"enable_parallel_dispatch" json:"enable_parallel_dispatch"`
	ConsumeIntervalSeconds time.Duration   `yaml:"consume_interval_seconds,omitempty" json:"consume_interval_seconds,omitempty"`
}

// NewSchedulerConfig creates a new scheduler configuration
func NewSchedulerConfig() *SchedulerConfig {
	return &SchedulerConfig{
		BaseConfig:               *NewBaseConfig(),
		Enabled:                  false,
		
		// NATS Configuration
		NATSURLs:                 []string{"nats://localhost:4222"},
		NATSMaxReconnect:         10,
		
		// NATS KV Configuration
		UseNATSKV:                true,
		NATSKVBucketName:         "memgos-scheduler",
		NATSKVDescription:        "MemGOS Scheduler Key-Value Store",
		NATSKVMaxValueSize:       1024 * 1024, // 1MB
		NATSKVHistory:            10,
		NATSKVMaxBytes:           1024 * 1024 * 1024, // 1GB
		NATSKVStorage:            "File",
		NATSKVReplicas:           1,
		
		// JetStream Configuration
		StreamName:               "SCHEDULER_STREAM",
		StreamSubjects:           []string{"scheduler.>"},
		ConsumerName:             "scheduler_consumer",
		ConsumerDurable:          true,
		MaxDeliver:               3,
		AckWait:                  10 * time.Second,
		MaxAckPending:            10,
		
		// Scheduler Configuration
		ThreadPoolMaxWorkers:     4,
		EnableParallelDispatch:   false,
		ConsumeIntervalSeconds:   5 * time.Second,
	}
}

// ToSchedulerConfigMap converts SchedulerConfig to a map for the scheduler factory
func (sc *SchedulerConfig) ToSchedulerConfigMap() map[string]interface{} {
	configMap := map[string]interface{}{
		"thread_pool_max_workers":   sc.ThreadPoolMaxWorkers,
		"enable_parallel_dispatch":  sc.EnableParallelDispatch,
		"consume_interval_seconds":  sc.ConsumeIntervalSeconds,
		"use_nats_kv":              sc.UseNATSKV,
	}
	
	// NATS configuration
	natsConfig := map[string]interface{}{
		"urls":         sc.NATSURLs,
		"username":     sc.NATSUsername,
		"password":     sc.NATSPassword,
		"token":        sc.NATSToken,
		"max_reconnect": sc.NATSMaxReconnect,
		"jetstream_config": map[string]interface{}{
			"stream_name":        sc.StreamName,
			"stream_subjects":    sc.StreamSubjects,
			"consumer_name":      sc.ConsumerName,
			"consumer_durable":   sc.ConsumerDurable,
			"max_deliver":        sc.MaxDeliver,
			"ack_wait":           sc.AckWait,
			"max_ack_pending":    sc.MaxAckPending,
			"replicas":           sc.NATSKVReplicas,
		},
	}
	configMap["nats_config"] = natsConfig
	
	// NATS KV configuration
	if sc.UseNATSKV {
		natsKVConfig := map[string]interface{}{
			"urls":            sc.NATSURLs,
			"username":        sc.NATSUsername,
			"password":        sc.NATSPassword,
			"token":           sc.NATSToken,
			"bucket_name":     sc.NATSKVBucketName,
			"description":     sc.NATSKVDescription,
			"max_value_size":  sc.NATSKVMaxValueSize,
			"history":         sc.NATSKVHistory,
			"max_bytes":       sc.NATSKVMaxBytes,
			"storage":         sc.NATSKVStorage,
			"replicas":        sc.NATSKVReplicas,
			"stream_name":     sc.StreamName,
			"stream_subjects": sc.StreamSubjects,
			"consumer_name":   sc.ConsumerName,
			"consumer_durable": sc.ConsumerDurable,
			"max_deliver":     sc.MaxDeliver,
			"ack_wait":        sc.AckWait,
			"max_ack_pending": sc.MaxAckPending,
		}
		configMap["nats_kv_config"] = natsKVConfig
	}
	
	return configMap
}

// MOSConfig represents the main MOS configuration
type MOSConfig struct {
	BaseConfig           `yaml:",inline"`
	UserID               string           `yaml:"user_id" json:"user_id" validate:"required"`
	SessionID            string           `yaml:"session_id" json:"session_id" validate:"required"`
	ChatModel            *LLMConfig       `yaml:"chat_model" json:"chat_model" validate:"required"`
	MemReader            *MemoryConfig    `yaml:"mem_reader,omitempty" json:"mem_reader,omitempty"`
	MemScheduler         *SchedulerConfig `yaml:"mem_scheduler,omitempty" json:"mem_scheduler,omitempty"`
	EnableTextualMemory  bool             `yaml:"enable_textual_memory" json:"enable_textual_memory"`
	EnableActivationMemory bool           `yaml:"enable_activation_memory" json:"enable_activation_memory"`
	EnableParametricMemory bool           `yaml:"enable_parametric_memory" json:"enable_parametric_memory"`
	EnableMemScheduler   bool             `yaml:"enable_mem_scheduler" json:"enable_mem_scheduler"`
	TopK                 int              `yaml:"top_k,omitempty" json:"top_k,omitempty"`
	LogLevel             string           `yaml:"log_level,omitempty" json:"log_level,omitempty"`
	LogFile              string           `yaml:"log_file,omitempty" json:"log_file,omitempty"`
	MetricsEnabled       bool             `yaml:"metrics_enabled" json:"metrics_enabled"`
	MetricsPort          int              `yaml:"metrics_port,omitempty" json:"metrics_port,omitempty"`
	HealthCheckEnabled   bool             `yaml:"health_check_enabled" json:"health_check_enabled"`
	HealthCheckPort      int              `yaml:"health_check_port,omitempty" json:"health_check_port,omitempty"`
	
	// API configuration for MCP client mode
	APIToken             string           `yaml:"api_token,omitempty" json:"api_token,omitempty"`
	APIURL               string           `yaml:"api_url,omitempty" json:"api_url,omitempty"`
	MCPMode              bool             `yaml:"mcp_mode" json:"mcp_mode"`
}

// NewMOSConfig creates a new MOS configuration
func NewMOSConfig() *MOSConfig {
	return &MOSConfig{
		BaseConfig:             *NewBaseConfig(),
		ChatModel:              NewLLMConfig(),
		MemReader:              NewMemoryConfig(),
		MemScheduler:           NewSchedulerConfig(),
		EnableTextualMemory:    true,
		EnableActivationMemory: false,
		EnableParametricMemory: false,
		EnableMemScheduler:     false,
		TopK:                   5,
		LogLevel:               "info",
		MetricsEnabled:         true,
		MetricsPort:            9090,
		HealthCheckEnabled:     true,
		HealthCheckPort:        8080,
	}
}

// APIConfig represents API server configuration
type APIConfig struct {
	BaseConfig `yaml:",inline"`
	Host       string        `yaml:"host" json:"host" validate:"required"`
	Port       int           `yaml:"port" json:"port" validate:"required,gt=0"`
	TLSEnabled bool          `yaml:"tls_enabled" json:"tls_enabled"`
	TLSCert    string        `yaml:"tls_cert,omitempty" json:"tls_cert,omitempty"`
	TLSKey     string        `yaml:"tls_key,omitempty" json:"tls_key,omitempty"`
	CORSEnabled bool         `yaml:"cors_enabled" json:"cors_enabled"`
	CORSOrigins []string     `yaml:"cors_origins,omitempty" json:"cors_origins,omitempty"`
	RateLimit   int          `yaml:"rate_limit,omitempty" json:"rate_limit,omitempty"`
	Timeout     time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	JWTSecret   string        `yaml:"jwt_secret,omitempty" json:"jwt_secret,omitempty"`
}

// NewAPIConfig creates a new API configuration
func NewAPIConfig() *APIConfig {
	return &APIConfig{
		BaseConfig:  *NewBaseConfig(),
		Host:        "localhost",
		Port:        8000,
		TLSEnabled:  false,
		CORSEnabled: true,
		CORSOrigins: []string{"*"},
		RateLimit:   100,
		Timeout:     30 * time.Second,
	}
}

// ConfigManager implements the configuration manager interface
type ConfigManager struct {
	config map[string]interface{}
	mu     sync.RWMutex
	viper  *viper.Viper
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() interfaces.ConfigManager {
	return &ConfigManager{
		config: make(map[string]interface{}),
		viper:  viper.New(),
	}
}

// Load loads configuration from a file
func (cm *ConfigManager) Load(ctx context.Context, path string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.viper.SetConfigFile(path)
	
	if err := cm.viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	
	cm.config = cm.viper.AllSettings()
	return nil
}

// Get retrieves a configuration value
func (cm *ConfigManager) Get(key string) interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	return cm.viper.Get(key)
}

// Set sets a configuration value
func (cm *ConfigManager) Set(key string, value interface{}) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.viper.Set(key, value)
	cm.config[key] = value
	return nil
}

// Save saves configuration to a file
func (cm *ConfigManager) Save(ctx context.Context, path string) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	return cm.viper.WriteConfigAs(path)
}

// Watch watches for configuration changes
func (cm *ConfigManager) Watch(ctx context.Context, callback func(key string, value interface{})) error {
	cm.viper.WatchConfig()
	cm.viper.OnConfigChange(func(e fsnotify.Event) {
		cm.mu.Lock()
		defer cm.mu.Unlock()
		
		// Update config map
		cm.config = cm.viper.AllSettings()
		
		// Call callback for each changed key
		for key, value := range cm.config {
			callback(key, value)
		}
	})
	
	return nil
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv(prefix string) *viper.Viper {
	v := viper.New()
	v.SetEnvPrefix(prefix)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	
	return v
}

// MergeConfigs merges multiple configurations
func MergeConfigs(configs ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	for _, config := range configs {
		for key, value := range config {
			result[key] = value
		}
	}
	
	return result
}