package chunkers

import (
	"encoding/json"
	"fmt"
	"time"
)

// AdvancedChunkerConfig provides comprehensive configuration for all advanced chunking features
type AdvancedChunkerConfig struct {
	// Base chunker configuration
	BaseConfig *ChunkerConfig `json:"base_config"`
	
	// Agentic chunker configuration
	AgenticConfig *AgenticChunkerConfig `json:"agentic_config,omitempty"`
	
	// Multi-modal chunker configuration
	MultiModalConfig *MultiModalChunkerConfig `json:"multimodal_config,omitempty"`
	
	// Hierarchical chunker configuration
	HierarchicalConfig *HierarchicalChunkerConfig `json:"hierarchical_config,omitempty"`
	
	// Quality metrics configuration
	QualityConfig *QualityMetricsConfig `json:"quality_config,omitempty"`
	
	// Performance configuration
	PerformanceConfig *PerformanceConfig `json:"performance_config,omitempty"`
	
	// Integration configuration
	IntegrationConfig *IntegrationConfig `json:"integration_config,omitempty"`
}

// QualityMetricsConfig configures quality assessment and metrics collection
type QualityMetricsConfig struct {
	// EnableQualityAssessment enables automatic quality assessment
	EnableQualityAssessment bool `json:"enable_quality_assessment"`
	
	// QualityThreshold is the minimum quality score for chunks
	QualityThreshold float64 `json:"quality_threshold"`
	
	// MetricsToCollect specifies which metrics to collect
	MetricsToCollect []QualityMetricType `json:"metrics_to_collect"`
	
	// SamplingRate for quality assessment (0-1)
	SamplingRate float64 `json:"sampling_rate"`
	
	// EnableRealTimeMonitoring enables real-time quality monitoring
	EnableRealTimeMonitoring bool `json:"enable_real_time_monitoring"`
	
	// AlertThresholds for quality degradation alerts
	AlertThresholds map[string]float64 `json:"alert_thresholds"`
	
	// HistoryRetention specifies how long to keep quality history
	HistoryRetention time.Duration `json:"history_retention"`
}

// QualityMetricType represents different quality metrics
type QualityMetricType string

const (
	MetricCoherence         QualityMetricType = "coherence"
	MetricCompleteness      QualityMetricType = "completeness"
	MetricRelevance         QualityMetricType = "relevance"
	MetricInformation       QualityMetricType = "information_density"
	MetricReadability       QualityMetricType = "readability"
	MetricSemanticIntegrity QualityMetricType = "semantic_integrity"
	MetricStructure         QualityMetricType = "structure_preservation"
	MetricOverlap           QualityMetricType = "overlap_optimization"
)

// PerformanceConfig configures performance optimization settings
type PerformanceConfig struct {
	// EnableParallelProcessing enables parallel chunk processing
	EnableParallelProcessing bool `json:"enable_parallel_processing"`
	
	// MaxConcurrency limits concurrent processing
	MaxConcurrency int `json:"max_concurrency"`
	
	// EnableCaching enables result caching
	EnableCaching bool `json:"enable_caching"`
	
	// CacheSize is the maximum number of cached results
	CacheSize int `json:"cache_size"`
	
	// CacheTTL is the time-to-live for cached results
	CacheTTL time.Duration `json:"cache_ttl"`
	
	// EnableProfiling enables performance profiling
	EnableProfiling bool `json:"enable_profiling"`
	
	// MemoryLimits sets memory usage limits
	MemoryLimits *MemoryLimits `json:"memory_limits,omitempty"`
	
	// OptimizationLevel controls optimization aggressiveness
	OptimizationLevel OptimizationLevel `json:"optimization_level"`
}

// MemoryLimits defines memory usage constraints
type MemoryLimits struct {
	MaxHeapSize      int64 `json:"max_heap_size"`      // in bytes
	MaxCacheSize     int64 `json:"max_cache_size"`     // in bytes
	MaxEmbeddingSize int64 `json:"max_embedding_size"` // in bytes
}

// OptimizationLevel defines optimization levels
type OptimizationLevel string

const (
	OptimizationNone         OptimizationLevel = "none"
	OptimizationBasic        OptimizationLevel = "basic"
	OptimizationIntermediate OptimizationLevel = "intermediate"
	OptimizationAggressive   OptimizationLevel = "aggressive"
)

// IntegrationConfig configures integration with external services
type IntegrationConfig struct {
	// LLMProviders configures LLM provider settings
	LLMProviders map[string]*LLMProviderConfig `json:"llm_providers,omitempty"`
	
	// EmbeddingProviders configures embedding provider settings
	EmbeddingProviders map[string]*EmbeddingProviderConfig `json:"embedding_providers,omitempty"`
	
	// VectorDBConfig configures vector database integration
	VectorDBConfig *VectorDBIntegrationConfig `json:"vector_db_config,omitempty"`
	
	// MonitoringConfig configures monitoring and observability
	MonitoringConfig *AdvancedMonitoringConfig `json:"monitoring_config,omitempty"`
}

// LLMProviderConfig configures LLM provider settings
type LLMProviderConfig struct {
	Provider    string                 `json:"provider"`
	APIKey      string                 `json:"api_key,omitempty"`
	APIEndpoint string                 `json:"api_endpoint,omitempty"`
	ModelName   string                 `json:"model_name"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Timeout     time.Duration          `json:"timeout"`
	RetryPolicy *RetryPolicy           `json:"retry_policy,omitempty"`
}

// EmbeddingProviderConfig configures embedding provider settings
type EmbeddingProviderConfig struct {
	Provider    string                 `json:"provider"`
	APIKey      string                 `json:"api_key,omitempty"`
	APIEndpoint string                 `json:"api_endpoint,omitempty"`
	ModelName   string                 `json:"model_name"`
	Dimensions  int                    `json:"dimensions"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	BatchSize   int                    `json:"batch_size"`
	Timeout     time.Duration          `json:"timeout"`
	RetryPolicy *RetryPolicy           `json:"retry_policy,omitempty"`
}

// VectorDBIntegrationConfig configures vector database integration
type VectorDBIntegrationConfig struct {
	Provider       string                 `json:"provider"`
	ConnectionURL  string                 `json:"connection_url"`
	CollectionName string                 `json:"collection_name"`
	IndexSettings  map[string]interface{} `json:"index_settings,omitempty"`
	BatchSize      int                    `json:"batch_size"`
	Timeout        time.Duration          `json:"timeout"`
}

// AdvancedMonitoringConfig configures monitoring and observability
type AdvancedMonitoringConfig struct {
	EnableMetrics    bool                   `json:"enable_metrics"`
	EnableTracing    bool                   `json:"enable_tracing"`
	EnableLogging    bool                   `json:"enable_logging"`
	LogLevel         string                 `json:"log_level"`
	MetricsEndpoint  string                 `json:"metrics_endpoint,omitempty"`
	TracingEndpoint  string                 `json:"tracing_endpoint,omitempty"`
	SamplingRate     float64                `json:"sampling_rate"`
	CustomMetrics    []string               `json:"custom_metrics,omitempty"`
	AlertEndpoints   []string               `json:"alert_endpoints,omitempty"`
	DashboardConfig  map[string]interface{} `json:"dashboard_config,omitempty"`
}

// RetryPolicy defines retry behavior for external services
type RetryPolicy struct {
	MaxRetries    int           `json:"max_retries"`
	InitialDelay  time.Duration `json:"initial_delay"`
	MaxDelay      time.Duration `json:"max_delay"`
	BackoffFactor float64       `json:"backoff_factor"`
	RetryOn       []string      `json:"retry_on"` // HTTP status codes or error types
}

// DefaultAdvancedConfig returns a comprehensive default configuration
func DefaultAdvancedConfig() *AdvancedChunkerConfig {
	return &AdvancedChunkerConfig{
		BaseConfig:         DefaultChunkerConfig(),
		AgenticConfig:      DefaultAgenticConfig(),
		MultiModalConfig:   DefaultMultiModalConfig(),
		HierarchicalConfig: DefaultHierarchicalConfig(),
		QualityConfig:      DefaultQualityConfig(),
		PerformanceConfig:  DefaultPerformanceConfig(),
		IntegrationConfig:  DefaultIntegrationConfig(),
	}
}

// DefaultQualityConfig returns default quality metrics configuration
func DefaultQualityConfig() *QualityMetricsConfig {
	return &QualityMetricsConfig{
		EnableQualityAssessment:  true,
		QualityThreshold:         0.7,
		MetricsToCollect: []QualityMetricType{
			MetricCoherence,
			MetricCompleteness,
			MetricRelevance,
			MetricInformation,
		},
		SamplingRate:             1.0,
		EnableRealTimeMonitoring: true,
		AlertThresholds: map[string]float64{
			"coherence":    0.6,
			"completeness": 0.5,
			"relevance":    0.6,
		},
		HistoryRetention: 24 * time.Hour,
	}
}

// DefaultPerformanceConfig returns default performance configuration
func DefaultPerformanceConfig() *PerformanceConfig {
	return &PerformanceConfig{
		EnableParallelProcessing: true,
		MaxConcurrency:          4,
		EnableCaching:           true,
		CacheSize:               1000,
		CacheTTL:                1 * time.Hour,
		EnableProfiling:         false,
		MemoryLimits: &MemoryLimits{
			MaxHeapSize:      1024 * 1024 * 1024, // 1GB
			MaxCacheSize:     256 * 1024 * 1024,  // 256MB
			MaxEmbeddingSize: 512 * 1024 * 1024,  // 512MB
		},
		OptimizationLevel: OptimizationIntermediate,
	}
}

// DefaultIntegrationConfig returns default integration configuration
func DefaultIntegrationConfig() *IntegrationConfig {
	return &IntegrationConfig{
		LLMProviders: map[string]*LLMProviderConfig{
			"openai": {
				Provider:  "openai",
				ModelName: "gpt-4",
				Timeout:   30 * time.Second,
				RetryPolicy: &RetryPolicy{
					MaxRetries:    3,
					InitialDelay:  1 * time.Second,
					MaxDelay:      10 * time.Second,
					BackoffFactor: 2.0,
					RetryOn:       []string{"429", "500", "502", "503", "504"},
				},
			},
		},
		EmbeddingProviders: map[string]*EmbeddingProviderConfig{
			"openai": {
				Provider:   "openai",
				ModelName:  "text-embedding-3-small",
				Dimensions: 1536,
				BatchSize:  100,
				Timeout:    30 * time.Second,
				RetryPolicy: &RetryPolicy{
					MaxRetries:    3,
					InitialDelay:  1 * time.Second,
					MaxDelay:      10 * time.Second,
					BackoffFactor: 2.0,
					RetryOn:       []string{"429", "500", "502", "503", "504"},
				},
			},
		},
		MonitoringConfig: &AdvancedMonitoringConfig{
			EnableMetrics:   true,
			EnableTracing:   false,
			EnableLogging:   true,
			LogLevel:        "INFO",
			SamplingRate:    0.1,
			CustomMetrics:   []string{"chunk_quality", "processing_time", "cache_hit_rate"},
		},
	}
}

// ConfigurationPreset defines predefined configuration templates
type ConfigurationPreset string

const (
	PresetDefault        ConfigurationPreset = "default"
	PresetHighQuality    ConfigurationPreset = "high_quality"
	PresetHighPerformance ConfigurationPreset = "high_performance"
	PresetBalanced       ConfigurationPreset = "balanced"
	PresetMinimal        ConfigurationPreset = "minimal"
	PresetResearchPaper  ConfigurationPreset = "research_paper"
	PresetTechnicalDoc   ConfigurationPreset = "technical_doc"
	PresetChatbot        ConfigurationPreset = "chatbot"
	PresetRAG            ConfigurationPreset = "rag"
)

// GetPresetConfig returns a predefined configuration
func GetPresetConfig(preset ConfigurationPreset) *AdvancedChunkerConfig {
	switch preset {
	case PresetHighQuality:
		return getHighQualityConfig()
	case PresetHighPerformance:
		return getHighPerformanceConfig()
	case PresetBalanced:
		return getBalancedConfig()
	case PresetMinimal:
		return getMinimalConfig()
	case PresetResearchPaper:
		return getResearchPaperConfig()
	case PresetTechnicalDoc:
		return getTechnicalDocConfig()
	case PresetChatbot:
		return getChatbotConfig()
	case PresetRAG:
		return getRAGConfig()
	default:
		return DefaultAdvancedConfig()
	}
}

// getHighQualityConfig returns configuration optimized for quality
func getHighQualityConfig() *AdvancedChunkerConfig {
	config := DefaultAdvancedConfig()
	
	// Optimize for quality
	config.BaseConfig.ChunkSize = 400
	config.BaseConfig.ChunkOverlap = 100
	
	// Enable advanced features
	config.AgenticConfig.AnalysisDepth = AnalysisDepthExpert
	config.AgenticConfig.ReasoningSteps = 5
	config.AgenticConfig.ConfidenceThreshold = 0.9
	
	config.HierarchicalConfig.SummaryMode = SummaryModeLLM
	config.HierarchicalConfig.MaxLevels = 4
	
	// Higher quality thresholds
	config.QualityConfig.QualityThreshold = 0.8
	config.QualityConfig.MetricsToCollect = []QualityMetricType{
		MetricCoherence,
		MetricCompleteness,
		MetricRelevance,
		MetricInformation,
		MetricReadability,
		MetricSemanticIntegrity,
	}
	
	return config
}

// getHighPerformanceConfig returns configuration optimized for performance
func getHighPerformanceConfig() *AdvancedChunkerConfig {
	config := DefaultAdvancedConfig()
	
	// Optimize for performance
	config.BaseConfig.ChunkSize = 800
	config.BaseConfig.ChunkOverlap = 50
	
	// Minimize expensive operations
	config.AgenticConfig.AnalysisDepth = AnalysisDepthBasic
	config.AgenticConfig.ReasoningSteps = 1
	config.AgenticConfig.MaxLLMCalls = 3
	
	config.HierarchicalConfig.SummaryMode = SummaryModeExtract
	config.HierarchicalConfig.MaxLevels = 2
	
	// Performance optimizations
	config.PerformanceConfig.EnableParallelProcessing = true
	config.PerformanceConfig.MaxConcurrency = 8
	config.PerformanceConfig.OptimizationLevel = OptimizationAggressive
	
	// Reduce quality assessment overhead
	config.QualityConfig.SamplingRate = 0.1
	config.QualityConfig.MetricsToCollect = []QualityMetricType{
		MetricCoherence,
		MetricCompleteness,
	}
	
	return config
}

// getBalancedConfig returns a balanced configuration
func getBalancedConfig() *AdvancedChunkerConfig {
	config := DefaultAdvancedConfig()
	
	// Balanced settings
	config.BaseConfig.ChunkSize = 512
	config.BaseConfig.ChunkOverlap = 80
	
	config.AgenticConfig.AnalysisDepth = AnalysisDepthIntermediate
	config.AgenticConfig.ReasoningSteps = 3
	
	config.HierarchicalConfig.SummaryMode = SummaryModeHybrid
	config.HierarchicalConfig.MaxLevels = 3
	
	config.PerformanceConfig.MaxConcurrency = 4
	config.PerformanceConfig.OptimizationLevel = OptimizationIntermediate
	
	config.QualityConfig.SamplingRate = 0.5
	
	return config
}

// getMinimalConfig returns a minimal configuration
func getMinimalConfig() *AdvancedChunkerConfig {
	return &AdvancedChunkerConfig{
		BaseConfig: DefaultChunkerConfig(),
		QualityConfig: &QualityMetricsConfig{
			EnableQualityAssessment: false,
		},
		PerformanceConfig: &PerformanceConfig{
			EnableParallelProcessing: false,
			MaxConcurrency:          1,
			EnableCaching:           false,
			OptimizationLevel:       OptimizationNone,
		},
	}
}

// getResearchPaperConfig returns configuration optimized for research papers
func getResearchPaperConfig() *AdvancedChunkerConfig {
	config := getHighQualityConfig()
	
	// Research paper specific settings
	config.BaseConfig.ChunkSize = 300
	config.BaseConfig.ChunkOverlap = 50
	
	config.HierarchicalConfig.PreserveSectionHeaders = true
	config.HierarchicalConfig.CrossReferenceTracking = true
	
	config.MultiModalConfig.PreserveTables = true
	config.MultiModalConfig.HandleImages = true
	
	return config
}

// getTechnicalDocConfig returns configuration optimized for technical documentation
func getTechnicalDocConfig() *AdvancedChunkerConfig {
	config := getBalancedConfig()
	
	// Technical documentation specific settings
	config.MultiModalConfig.PreserveCodeBlocks = true
	config.MultiModalConfig.PreserveTables = true
	config.MultiModalConfig.CodeLanguages = []string{
		"python", "javascript", "go", "java", "cpp", "c", "rust", "sql", "bash", "yaml", "json", "xml", "html", "css",
	}
	
	config.HierarchicalConfig.PreserveSectionHeaders = true
	
	return config
}

// getChatbotConfig returns configuration optimized for chatbot applications
func getChatbotConfig() *AdvancedChunkerConfig {
	config := getHighPerformanceConfig()
	
	// Chatbot specific settings
	config.BaseConfig.ChunkSize = 200
	config.BaseConfig.ChunkOverlap = 30
	
	// Focus on conversational coherence
	config.QualityConfig.MetricsToCollect = []QualityMetricType{
		MetricCoherence,
		MetricRelevance,
		MetricReadability,
	}
	
	return config
}

// getRAGConfig returns configuration optimized for RAG applications
func getRAGConfig() *AdvancedChunkerConfig {
	config := getBalancedConfig()
	
	// RAG specific settings
	config.BaseConfig.ChunkSize = 384 // Optimal for many embedding models
	config.BaseConfig.ChunkOverlap = 96
	
	// Focus on semantic integrity
	config.QualityConfig.MetricsToCollect = []QualityMetricType{
		MetricSemanticIntegrity,
		MetricInformation,
		MetricRelevance,
	}
	
	config.HierarchicalConfig.CrossReferenceTracking = true
	
	return config
}

// ConfigValidator validates configuration settings
type ConfigValidator struct{}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// ValidateConfig validates an advanced chunker configuration
func (cv *ConfigValidator) ValidateConfig(config *AdvancedChunkerConfig) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}
	
	// Validate base config
	if config.BaseConfig != nil {
		if err := cv.validateBaseConfig(config.BaseConfig); err != nil {
			return fmt.Errorf("base config validation failed: %w", err)
		}
	}
	
	// Validate agentic config
	if config.AgenticConfig != nil {
		if err := cv.validateAgenticConfig(config.AgenticConfig); err != nil {
			return fmt.Errorf("agentic config validation failed: %w", err)
		}
	}
	
	// Validate quality config
	if config.QualityConfig != nil {
		if err := cv.validateQualityConfig(config.QualityConfig); err != nil {
			return fmt.Errorf("quality config validation failed: %w", err)
		}
	}
	
	// Validate performance config
	if config.PerformanceConfig != nil {
		if err := cv.validatePerformanceConfig(config.PerformanceConfig); err != nil {
			return fmt.Errorf("performance config validation failed: %w", err)
		}
	}
	
	return nil
}

// validateBaseConfig validates base configuration
func (cv *ConfigValidator) validateBaseConfig(config *ChunkerConfig) error {
	if config.ChunkSize <= 0 {
		return fmt.Errorf("chunk size must be positive, got: %d", config.ChunkSize)
	}
	
	if config.ChunkOverlap < 0 {
		return fmt.Errorf("chunk overlap cannot be negative, got: %d", config.ChunkOverlap)
	}
	
	if config.ChunkOverlap >= config.ChunkSize {
		return fmt.Errorf("chunk overlap (%d) must be less than chunk size (%d)", 
			config.ChunkOverlap, config.ChunkSize)
	}
	
	return nil
}

// validateAgenticConfig validates agentic configuration
func (cv *ConfigValidator) validateAgenticConfig(config *AgenticChunkerConfig) error {
	if config.ReasoningSteps <= 0 {
		return fmt.Errorf("reasoning steps must be positive, got: %d", config.ReasoningSteps)
	}
	
	if config.ConfidenceThreshold < 0 || config.ConfidenceThreshold > 1 {
		return fmt.Errorf("confidence threshold must be between 0 and 1, got: %f", config.ConfidenceThreshold)
	}
	
	if config.MaxLLMCalls <= 0 {
		return fmt.Errorf("max LLM calls must be positive, got: %d", config.MaxLLMCalls)
	}
	
	return nil
}

// validateQualityConfig validates quality configuration
func (cv *ConfigValidator) validateQualityConfig(config *QualityMetricsConfig) error {
	if config.QualityThreshold < 0 || config.QualityThreshold > 1 {
		return fmt.Errorf("quality threshold must be between 0 and 1, got: %f", config.QualityThreshold)
	}
	
	if config.SamplingRate < 0 || config.SamplingRate > 1 {
		return fmt.Errorf("sampling rate must be between 0 and 1, got: %f", config.SamplingRate)
	}
	
	return nil
}

// validatePerformanceConfig validates performance configuration
func (cv *ConfigValidator) validatePerformanceConfig(config *PerformanceConfig) error {
	if config.MaxConcurrency <= 0 {
		return fmt.Errorf("max concurrency must be positive, got: %d", config.MaxConcurrency)
	}
	
	if config.CacheSize < 0 {
		return fmt.Errorf("cache size cannot be negative, got: %d", config.CacheSize)
	}
	
	if config.MemoryLimits != nil {
		if config.MemoryLimits.MaxHeapSize <= 0 {
			return fmt.Errorf("max heap size must be positive, got: %d", config.MemoryLimits.MaxHeapSize)
		}
	}
	
	return nil
}

// ConfigSerializer handles configuration serialization
type ConfigSerializer struct{}

// NewConfigSerializer creates a new configuration serializer
func NewConfigSerializer() *ConfigSerializer {
	return &ConfigSerializer{}
}

// ToJSON serializes configuration to JSON
func (cs *ConfigSerializer) ToJSON(config *AdvancedChunkerConfig) ([]byte, error) {
	return json.MarshalIndent(config, "", "  ")
}

// FromJSON deserializes configuration from JSON
func (cs *ConfigSerializer) FromJSON(data []byte) (*AdvancedChunkerConfig, error) {
	var config AdvancedChunkerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}
	
	// Validate deserialized config
	validator := NewConfigValidator()
	if err := validator.ValidateConfig(&config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	
	return &config, nil
}

// ConfigTemplate provides configuration templates for different use cases
type ConfigTemplate struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Config      *AdvancedChunkerConfig   `json:"config"`
	UseCases    []string                 `json:"use_cases"`
	Features    []string                 `json:"features"`
	Metadata    map[string]interface{}   `json:"metadata,omitempty"`
}

// GetAvailableTemplates returns all available configuration templates
func GetAvailableTemplates() []*ConfigTemplate {
	return []*ConfigTemplate{
		{
			Name:        "High Quality",
			Description: "Optimized for maximum chunk quality with advanced AI analysis",
			Config:      GetPresetConfig(PresetHighQuality),
			UseCases:    []string{"Research", "Academic Papers", "Legal Documents"},
			Features:    []string{"LLM-driven analysis", "Multi-round reasoning", "Advanced quality metrics"},
		},
		{
			Name:        "High Performance",
			Description: "Optimized for speed and throughput with minimal overhead",
			Config:      GetPresetConfig(PresetHighPerformance),
			UseCases:    []string{"Real-time Processing", "Large Document Batches", "Production APIs"},
			Features:    []string{"Parallel processing", "Caching", "Simplified analysis"},
		},
		{
			Name:        "Balanced",
			Description: "Balanced approach between quality and performance",
			Config:      GetPresetConfig(PresetBalanced),
			UseCases:    []string{"General Purpose", "Content Management", "Knowledge Bases"},
			Features:    []string{"Hybrid summaries", "Moderate analysis", "Configurable optimization"},
		},
		{
			Name:        "Research Paper",
			Description: "Specialized for academic and research documents",
			Config:      GetPresetConfig(PresetResearchPaper),
			UseCases:    []string{"Academic Research", "Scientific Papers", "Literature Review"},
			Features:    []string{"Section preservation", "Cross-references", "Table handling"},
		},
		{
			Name:        "Technical Documentation",
			Description: "Optimized for technical documents with code and diagrams",
			Config:      GetPresetConfig(PresetTechnicalDoc),
			UseCases:    []string{"API Documentation", "Software Manuals", "Technical Guides"},
			Features:    []string{"Code block preservation", "Multi-modal support", "Structure awareness"},
		},
		{
			Name:        "RAG System",
			Description: "Optimized for Retrieval-Augmented Generation applications",
			Config:      GetPresetConfig(PresetRAG),
			UseCases:    []string{"Question Answering", "Chatbots", "Information Retrieval"},
			Features:    []string{"Semantic integrity", "Optimal embedding size", "Cross-references"},
		},
	}
}