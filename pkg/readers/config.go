package readers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ConfigManager manages reader configurations
type ConfigManager struct {
	configs map[string]*ReaderConfig
	defaultConfig *ReaderConfig
	configPath string
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(configPath string) *ConfigManager {
	return &ConfigManager{
		configs:       make(map[string]*ReaderConfig),
		defaultConfig: DefaultReaderConfig(),
		configPath:    configPath,
	}
}

// LoadConfig loads a reader configuration from file
func (cm *ConfigManager) LoadConfig(name string, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}
	
	var config ReaderConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file %s: %w", filePath, err)
	}
	
	// Validate configuration
	if err := cm.validateConfig(&config); err != nil {
		return fmt.Errorf("invalid configuration in %s: %w", filePath, err)
	}
	
	cm.configs[name] = &config
	return nil
}

// SaveConfig saves a reader configuration to file
func (cm *ConfigManager) SaveConfig(name string, filePath string) error {
	config, exists := cm.configs[name]
	if !exists {
		return fmt.Errorf("configuration %s not found", name)
	}
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", filePath, err)
	}
	
	return nil
}

// GetConfig retrieves a configuration by name
func (cm *ConfigManager) GetConfig(name string) (*ReaderConfig, error) {
	if config, exists := cm.configs[name]; exists {
		// Return a copy to prevent modification
		return cm.copyConfig(config), nil
	}
	
	return nil, fmt.Errorf("configuration %s not found", name)
}

// SetConfig sets a configuration
func (cm *ConfigManager) SetConfig(name string, config *ReaderConfig) error {
	if err := cm.validateConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	
	cm.configs[name] = cm.copyConfig(config)
	return nil
}

// ListConfigs returns all available configuration names
func (cm *ConfigManager) ListConfigs() []string {
	names := make([]string, 0, len(cm.configs))
	for name := range cm.configs {
		names = append(names, name)
	}
	return names
}

// DeleteConfig removes a configuration
func (cm *ConfigManager) DeleteConfig(name string) error {
	if _, exists := cm.configs[name]; !exists {
		return fmt.Errorf("configuration %s not found", name)
	}
	
	delete(cm.configs, name)
	return nil
}

// GetDefaultConfig returns the default configuration
func (cm *ConfigManager) GetDefaultConfig() *ReaderConfig {
	return cm.copyConfig(cm.defaultConfig)
}

// SetDefaultConfig sets the default configuration
func (cm *ConfigManager) SetDefaultConfig(config *ReaderConfig) error {
	if err := cm.validateConfig(config); err != nil {
		return fmt.Errorf("invalid default configuration: %w", err)
	}
	
	cm.defaultConfig = cm.copyConfig(config)
	return nil
}

// LoadConfigDirectory loads all configurations from a directory
func (cm *ConfigManager) LoadConfigDirectory(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return fmt.Errorf("configuration directory %s does not exist", dirPath)
	}
	
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read configuration directory %s: %w", dirPath, err)
	}
	
	var lastErr error
	loaded := 0
	
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		
		configName := strings.TrimSuffix(file.Name(), ".json")
		filePath := filepath.Join(dirPath, file.Name())
		
		if err := cm.LoadConfig(configName, filePath); err != nil {
			lastErr = fmt.Errorf("failed to load config %s: %w", configName, err)
			continue
		}
		
		loaded++
	}
	
	if loaded == 0 && lastErr != nil {
		return fmt.Errorf("no configurations loaded: %w", lastErr)
	}
	
	return nil
}

// SaveAllConfigs saves all configurations to the specified directory
func (cm *ConfigManager) SaveAllConfigs(dirPath string) error {
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}
	
	var lastErr error
	saved := 0
	
	for name, config := range cm.configs {
		filePath := filepath.Join(dirPath, name+".json")
		
		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			lastErr = fmt.Errorf("failed to marshal config %s: %w", name, err)
			continue
		}
		
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			lastErr = fmt.Errorf("failed to write config %s: %w", name, err)
			continue
		}
		
		saved++
	}
	
	if saved == 0 && lastErr != nil {
		return fmt.Errorf("no configurations saved: %w", lastErr)
	}
	
	return nil
}

// CreateProfiledConfig creates a configuration optimized for a specific use case
func (cm *ConfigManager) CreateProfiledConfig(profile ConfigProfile) *ReaderConfig {
	base := cm.copyConfig(cm.defaultConfig)
	
	switch profile {
	case ProfileHighPerformance:
		return cm.applyHighPerformanceProfile(base)
	case ProfileHighAccuracy:
		return cm.applyHighAccuracyProfile(base)
	case ProfileBalanced:
		return cm.applyBalancedProfile(base)
	case ProfileLowResource:
		return cm.applyLowResourceProfile(base)
	case ProfileAnalyticsHeavy:
		return cm.applyAnalyticsHeavyProfile(base)
	case ProfileMinimal:
		return cm.applyMinimalProfile(base)
	default:
		return base
	}
}

// ConfigProfile defines different configuration profiles
type ConfigProfile string

const (
	ProfileHighPerformance ConfigProfile = "high_performance"
	ProfileHighAccuracy    ConfigProfile = "high_accuracy"
	ProfileBalanced        ConfigProfile = "balanced"
	ProfileLowResource     ConfigProfile = "low_resource"
	ProfileAnalyticsHeavy  ConfigProfile = "analytics_heavy"
	ProfileMinimal         ConfigProfile = "minimal"
)

// GetAvailableProfiles returns all available configuration profiles
func (cm *ConfigManager) GetAvailableProfiles() []ConfigProfile {
	return []ConfigProfile{
		ProfileHighPerformance,
		ProfileHighAccuracy,
		ProfileBalanced,
		ProfileLowResource,
		ProfileAnalyticsHeavy,
		ProfileMinimal,
	}
}

// MergeConfigs merges multiple configurations, with later configs taking precedence
func (cm *ConfigManager) MergeConfigs(configs ...*ReaderConfig) *ReaderConfig {
	if len(configs) == 0 {
		return cm.copyConfig(cm.defaultConfig)
	}
	
	merged := cm.copyConfig(configs[0])
	
	for i := 1; i < len(configs); i++ {
		cm.mergeInto(merged, configs[i])
	}
	
	return merged
}

// CreateCustomConfig creates a custom configuration with specific options
func (cm *ConfigManager) CreateCustomConfig(options map[string]interface{}) (*ReaderConfig, error) {
	config := cm.copyConfig(cm.defaultConfig)
	
	for key, value := range options {
		if err := cm.setConfigOption(config, key, value); err != nil {
			return nil, fmt.Errorf("failed to set option %s: %w", key, err)
		}
	}
	
	if err := cm.validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid custom configuration: %w", err)
	}
	
	return config, nil
}

// ConfigTemplate provides a template for creating configurations
type ConfigTemplate struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	BaseProfile ConfigProfile          `json:"base_profile"`
	Options     map[string]interface{} `json:"options"`
	Tags        []string               `json:"tags"`
	Version     string                 `json:"version"`
	CreatedAt   time.Time              `json:"created_at"`
}

// CreateFromTemplate creates a configuration from a template
func (cm *ConfigManager) CreateFromTemplate(template *ConfigTemplate) (*ReaderConfig, error) {
	// Start with the base profile
	config := cm.CreateProfiledConfig(template.BaseProfile)
	
	// Apply template options
	for key, value := range template.Options {
		if err := cm.setConfigOption(config, key, value); err != nil {
			return nil, fmt.Errorf("failed to apply template option %s: %w", key, err)
		}
	}
	
	if err := cm.validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid template configuration: %w", err)
	}
	
	return config, nil
}

// Private helper methods

func (cm *ConfigManager) validateConfig(config *ReaderConfig) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}
	
	// Validate strategy
	validStrategies := []ReadStrategy{
		ReadStrategySimple,
		ReadStrategyAdvanced,
		ReadStrategySemantic,
		ReadStrategyStructural,
	}
	
	valid := false
	for _, strategy := range validStrategies {
		if config.Strategy == strategy {
			valid = true
			break
		}
	}
	
	if !valid {
		return fmt.Errorf("invalid strategy: %s", config.Strategy)
	}
	
	// Validate analysis depth
	validDepths := []string{"basic", "medium", "deep", "comprehensive"}
	valid = false
	for _, depth := range validDepths {
		if config.AnalysisDepth == depth {
			valid = true
			break
		}
	}
	
	if !valid {
		return fmt.Errorf("invalid analysis depth: %s", config.AnalysisDepth)
	}
	
	// Validate pattern config if present
	if config.PatternConfig != nil {
		if err := cm.validatePatternConfig(config.PatternConfig); err != nil {
			return fmt.Errorf("invalid pattern config: %w", err)
		}
	}
	
	// Validate summarization config if present
	if config.SummarizationConfig != nil {
		if err := cm.validateSummarizationConfig(config.SummarizationConfig); err != nil {
			return fmt.Errorf("invalid summarization config: %w", err)
		}
	}
	
	return nil
}

func (cm *ConfigManager) validatePatternConfig(config *PatternConfig) error {
	if config.MinSupport < 0 || config.MinSupport > 1 {
		return fmt.Errorf("min_support must be between 0 and 1, got %f", config.MinSupport)
	}
	
	if config.MinConfidence < 0 || config.MinConfidence > 1 {
		return fmt.Errorf("min_confidence must be between 0 and 1, got %f", config.MinConfidence)
	}
	
	if config.MaxPatterns < 1 {
		return fmt.Errorf("max_patterns must be at least 1, got %d", config.MaxPatterns)
	}
	
	validTypes := []string{"sequential", "frequent", "anomaly", "structural", "temporal"}
	for _, patternType := range config.PatternTypes {
		valid := false
		for _, validType := range validTypes {
			if patternType == validType {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid pattern type: %s", patternType)
		}
	}
	
	return nil
}

func (cm *ConfigManager) validateSummarizationConfig(config *SummarizationConfig) error {
	if config.MaxLength < config.MinLength {
		return fmt.Errorf("max_length (%d) must be greater than min_length (%d)", config.MaxLength, config.MinLength)
	}
	
	if config.MinLength < 1 {
		return fmt.Errorf("min_length must be at least 1, got %d", config.MinLength)
	}
	
	validStrategies := []string{"extractive", "abstractive", "hybrid"}
	valid := false
	for _, strategy := range validStrategies {
		if config.Strategy == strategy {
			valid = true
			break
		}
	}
	
	if !valid {
		return fmt.Errorf("invalid summarization strategy: %s", config.Strategy)
	}
	
	if config.KeyPoints < 1 {
		return fmt.Errorf("key_points must be at least 1, got %d", config.KeyPoints)
	}
	
	return nil
}

func (cm *ConfigManager) copyConfig(config *ReaderConfig) *ReaderConfig {
	if config == nil {
		return nil
	}
	
	copy := &ReaderConfig{
		Strategy:           config.Strategy,
		AnalysisDepth:      config.AnalysisDepth,
		PatternDetection:   config.PatternDetection,
		QualityAssessment:  config.QualityAssessment,
		DuplicateDetection: config.DuplicateDetection,
		SentimentAnalysis:  config.SentimentAnalysis,
		Summarization:      config.Summarization,
		Options:            make(map[string]interface{}),
	}
	
	// Deep copy pattern config
	if config.PatternConfig != nil {
		copy.PatternConfig = &PatternConfig{
			MinSupport:    config.PatternConfig.MinSupport,
			MinConfidence: config.PatternConfig.MinConfidence,
			MaxPatterns:   config.PatternConfig.MaxPatterns,
			PatternTypes:  make([]string, len(config.PatternConfig.PatternTypes)),
		}
		// Copy pattern types slice
		for i, patternType := range config.PatternConfig.PatternTypes {
			copy.PatternConfig.PatternTypes[i] = patternType
		}
	}
	
	// Deep copy summarization config
	if config.SummarizationConfig != nil {
		copy.SummarizationConfig = &SummarizationConfig{
			MaxLength:        config.SummarizationConfig.MaxLength,
			MinLength:        config.SummarizationConfig.MinLength,
			Strategy:         config.SummarizationConfig.Strategy,
			KeyPoints:        config.SummarizationConfig.KeyPoints,
			IncludeThemes:    config.SummarizationConfig.IncludeThemes,
			IncludeEntities:  config.SummarizationConfig.IncludeEntities,
			IncludeSentiment: config.SummarizationConfig.IncludeSentiment,
		}
	}
	
	// Deep copy options
	for key, value := range config.Options {
		copy.Options[key] = value
	}
	
	return copy
}

func (cm *ConfigManager) mergeInto(target, source *ReaderConfig) {
	if source == nil {
		return
	}
	
	// Only update non-zero values
	if source.Strategy != "" {
		target.Strategy = source.Strategy
	}
	
	if source.AnalysisDepth != "" {
		target.AnalysisDepth = source.AnalysisDepth
	}
	
	// Boolean fields - only merge if explicitly set in source
	target.PatternDetection = source.PatternDetection
	target.QualityAssessment = source.QualityAssessment
	target.DuplicateDetection = source.DuplicateDetection
	target.SentimentAnalysis = source.SentimentAnalysis
	target.Summarization = source.Summarization
	
	// Merge sub-configs
	if source.PatternConfig != nil {
		if target.PatternConfig == nil {
			target.PatternConfig = &PatternConfig{}
		}
		cm.mergePatternConfig(target.PatternConfig, source.PatternConfig)
	}
	
	if source.SummarizationConfig != nil {
		if target.SummarizationConfig == nil {
			target.SummarizationConfig = &SummarizationConfig{}
		}
		cm.mergeSummarizationConfig(target.SummarizationConfig, source.SummarizationConfig)
	}
	
	// Merge options
	for key, value := range source.Options {
		target.Options[key] = value
	}
}

func (cm *ConfigManager) mergePatternConfig(target, source *PatternConfig) {
	if source.MinSupport > 0 {
		target.MinSupport = source.MinSupport
	}
	if source.MinConfidence > 0 {
		target.MinConfidence = source.MinConfidence
	}
	if source.MaxPatterns > 0 {
		target.MaxPatterns = source.MaxPatterns
	}
	if len(source.PatternTypes) > 0 {
		target.PatternTypes = make([]string, len(source.PatternTypes))
		copy(target.PatternTypes, source.PatternTypes)
	}
}

func (cm *ConfigManager) mergeSummarizationConfig(target, source *SummarizationConfig) {
	if source.MaxLength > 0 {
		target.MaxLength = source.MaxLength
	}
	if source.MinLength > 0 {
		target.MinLength = source.MinLength
	}
	if source.Strategy != "" {
		target.Strategy = source.Strategy
	}
	if source.KeyPoints > 0 {
		target.KeyPoints = source.KeyPoints
	}
	
	target.IncludeThemes = source.IncludeThemes
	target.IncludeEntities = source.IncludeEntities
	target.IncludeSentiment = source.IncludeSentiment
}

func (cm *ConfigManager) setConfigOption(config *ReaderConfig, key string, value interface{}) error {
	switch key {
	case "strategy":
		if strVal, ok := value.(string); ok {
			config.Strategy = ReadStrategy(strVal)
		} else {
			return fmt.Errorf("strategy must be a string")
		}
	case "analysis_depth":
		if strVal, ok := value.(string); ok {
			config.AnalysisDepth = strVal
		} else {
			return fmt.Errorf("analysis_depth must be a string")
		}
	case "pattern_detection":
		if boolVal, ok := value.(bool); ok {
			config.PatternDetection = boolVal
		} else {
			return fmt.Errorf("pattern_detection must be a boolean")
		}
	case "quality_assessment":
		if boolVal, ok := value.(bool); ok {
			config.QualityAssessment = boolVal
		} else {
			return fmt.Errorf("quality_assessment must be a boolean")
		}
	case "duplicate_detection":
		if boolVal, ok := value.(bool); ok {
			config.DuplicateDetection = boolVal
		} else {
			return fmt.Errorf("duplicate_detection must be a boolean")
		}
	case "sentiment_analysis":
		if boolVal, ok := value.(bool); ok {
			config.SentimentAnalysis = boolVal
		} else {
			return fmt.Errorf("sentiment_analysis must be a boolean")
		}
	case "summarization":
		if boolVal, ok := value.(bool); ok {
			config.Summarization = boolVal
		} else {
			return fmt.Errorf("summarization must be a boolean")
		}
	default:
		// Store in options
		config.Options[key] = value
	}
	
	return nil
}

// Profile application methods

func (cm *ConfigManager) applyHighPerformanceProfile(config *ReaderConfig) *ReaderConfig {
	config.Strategy = ReadStrategySimple
	config.AnalysisDepth = "basic"
	config.PatternDetection = false
	config.QualityAssessment = false
	config.DuplicateDetection = false
	config.SentimentAnalysis = false
	config.Summarization = false
	
	config.Options["max_concurrent"] = 10
	config.Options["cache_results"] = true
	config.Options["skip_heavy_analysis"] = true
	
	return config
}

func (cm *ConfigManager) applyHighAccuracyProfile(config *ReaderConfig) *ReaderConfig {
	config.Strategy = ReadStrategyAdvanced
	config.AnalysisDepth = "comprehensive"
	config.PatternDetection = true
	config.QualityAssessment = true
	config.DuplicateDetection = true
	config.SentimentAnalysis = true
	config.Summarization = true
	
	if config.PatternConfig != nil {
		config.PatternConfig.MinConfidence = 0.8
		config.PatternConfig.MaxPatterns = 200
	}
	
	config.Options["deep_analysis"] = true
	config.Options["multiple_passes"] = true
	config.Options["validation_enabled"] = true
	
	return config
}

func (cm *ConfigManager) applyBalancedProfile(config *ReaderConfig) *ReaderConfig {
	config.Strategy = ReadStrategyAdvanced
	config.AnalysisDepth = "medium"
	config.PatternDetection = true
	config.QualityAssessment = true
	config.DuplicateDetection = true
	config.SentimentAnalysis = true
	config.Summarization = true
	
	config.Options["balanced_mode"] = true
	config.Options["adaptive_depth"] = true
	
	return config
}

func (cm *ConfigManager) applyLowResourceProfile(config *ReaderConfig) *ReaderConfig {
	config.Strategy = ReadStrategySimple
	config.AnalysisDepth = "basic"
	config.PatternDetection = false
	config.QualityAssessment = false
	config.DuplicateDetection = false
	config.SentimentAnalysis = false
	config.Summarization = false
	
	config.Options["memory_efficient"] = true
	config.Options["minimal_processing"] = true
	config.Options["max_concurrent"] = 1
	
	return config
}

func (cm *ConfigManager) applyAnalyticsHeavyProfile(config *ReaderConfig) *ReaderConfig {
	config.Strategy = ReadStrategyAdvanced
	config.AnalysisDepth = "comprehensive"
	config.PatternDetection = true
	config.QualityAssessment = true
	config.DuplicateDetection = true
	config.SentimentAnalysis = true
	config.Summarization = true
	
	if config.PatternConfig != nil {
		config.PatternConfig.MaxPatterns = 500
		config.PatternConfig.PatternTypes = []string{"sequential", "frequent", "anomaly", "structural", "temporal"}
	}
	
	config.Options["analytics_mode"] = true
	config.Options["extensive_patterns"] = true
	config.Options["detailed_statistics"] = true
	
	return config
}

func (cm *ConfigManager) applyMinimalProfile(config *ReaderConfig) *ReaderConfig {
	config.Strategy = ReadStrategySimple
	config.AnalysisDepth = "basic"
	config.PatternDetection = false
	config.QualityAssessment = false
	config.DuplicateDetection = false
	config.SentimentAnalysis = false
	config.Summarization = false
	
	config.Options["minimal_mode"] = true
	config.Options["basic_only"] = true
	
	return config
}

// Utility functions for configuration management

// GetDefaultConfigManager returns a default configuration manager
func GetDefaultConfigManager() *ConfigManager {
	return NewConfigManager("")
}

// CreateConfigFromProfile creates a configuration from a profile name
func CreateConfigFromProfile(profile string) *ReaderConfig {
	cm := GetDefaultConfigManager()
	return cm.CreateProfiledConfig(ConfigProfile(profile))
}

// ValidateConfigFile validates a configuration file without loading it
func ValidateConfigFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	
	var config ReaderConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}
	
	cm := GetDefaultConfigManager()
	return cm.validateConfig(&config)
}