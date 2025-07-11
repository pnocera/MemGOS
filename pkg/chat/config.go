// Package chat - Configuration management for MemChat
package chat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/logger"
)

// ChatConfigManager handles configuration for MemChat instances
type ChatConfigManager struct {
	logger       interfaces.Logger
	configDir    string
	profiles     map[string]*MemChatConfig
	globalConfig *GlobalChatConfig
}

// GlobalChatConfig represents global chat system configuration
type GlobalChatConfig struct {
	// Default settings
	DefaultLLMProvider    string            `json:"default_llm_provider"`
	DefaultLLMModel       string            `json:"default_llm_model"`
	DefaultMaxTokens      int               `json:"default_max_tokens"`
	DefaultTemperature    float64           `json:"default_temperature"`
	DefaultTopK           int               `json:"default_top_k"`
	DefaultMaxTurnsWindow int               `json:"default_max_turns_window"`
	
	// System settings
	MaxConcurrentSessions int           `json:"max_concurrent_sessions"`
	SessionTimeout        time.Duration `json:"session_timeout"`
	LogLevel              string        `json:"log_level"`
	MetricsEnabled        bool          `json:"metrics_enabled"`
	
	// Memory settings
	DefaultMemoryEnabled  bool   `json:"default_memory_enabled"`
	MemoryRetentionPeriod string `json:"memory_retention_period"`
	
	// Security settings
	APIKeyEncryption     bool     `json:"api_key_encryption"`
	AllowedLLMProviders  []string `json:"allowed_llm_providers"`
	RateLimitEnabled     bool     `json:"rate_limit_enabled"`
	MaxRequestsPerMinute int      `json:"max_requests_per_minute"`
	
	// Features
	EnabledFeatures      []string               `json:"enabled_features"`
	ExperimentalFeatures []string               `json:"experimental_features"`
	PluginDirectories    []string               `json:"plugin_directories"`
	
	// Custom settings
	CustomDefaults map[string]interface{} `json:"custom_defaults"`
}

// ConfigProfile represents a named configuration profile
type ConfigProfile struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Config      *MemChatConfig `json:"config"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Tags        []string       `json:"tags"`
	IsDefault   bool           `json:"is_default"`
}

// NewChatConfigManager creates a new configuration manager
func NewChatConfigManager(configDir string) (*ChatConfigManager, error) {
	if configDir == "" {
		configDir = getDefaultConfigDir()
	}
	
	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}
	
	manager := &ChatConfigManager{
		logger:       logger.NewLogger(),
		configDir:    configDir,
		profiles:     make(map[string]*MemChatConfig),
		globalConfig: DefaultGlobalChatConfig(),
	}
	
	// Load existing configurations
	if err := manager.loadConfigurations(); err != nil {
		manager.logger.Warn("Failed to load existing configurations", map[string]interface{}{
			"error": err.Error(),
		})
	}
	
	return manager, nil
}

// getDefaultConfigDir returns the default configuration directory
func getDefaultConfigDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".memgos", "chat", "config")
	}
	return filepath.Join(".", "config", "chat")
}

// LoadGlobalConfig loads global configuration from file
func (cm *ChatConfigManager) LoadGlobalConfig() error {
	configPath := filepath.Join(cm.configDir, "global.json")
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config file
			return cm.SaveGlobalConfig()
		}
		return fmt.Errorf("failed to read global config: %w", err)
	}
	
	config := &GlobalChatConfig{}
	if err := json.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse global config: %w", err)
	}
	
	cm.globalConfig = config
	cm.logger.Info("Loaded global configuration", map[string]interface{}{
		"config_path": configPath,
	})
	
	return nil
}

// SaveGlobalConfig saves global configuration to file
func (cm *ChatConfigManager) SaveGlobalConfig() error {
	configPath := filepath.Join(cm.configDir, "global.json")
	
	data, err := json.MarshalIndent(cm.globalConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal global config: %w", err)
	}
	
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write global config: %w", err)
	}
	
	cm.logger.Info("Saved global configuration", map[string]interface{}{
		"config_path": configPath,
	})
	
	return nil
}

// GetGlobalConfig returns the global configuration
func (cm *ChatConfigManager) GetGlobalConfig() *GlobalChatConfig {
	return cm.globalConfig
}

// UpdateGlobalConfig updates the global configuration
func (cm *ChatConfigManager) UpdateGlobalConfig(config *GlobalChatConfig) error {
	if err := validateGlobalConfig(config); err != nil {
		return fmt.Errorf("invalid global config: %w", err)
	}
	
	cm.globalConfig = config
	return cm.SaveGlobalConfig()
}

// CreateProfile creates a new configuration profile
func (cm *ChatConfigManager) CreateProfile(name, description string, config *MemChatConfig) error {
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}
	
	if err := ValidateMemChatConfig(config); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	
	// Apply global defaults to config
	cm.applyGlobalDefaults(config)
	
	// Store profile
	cm.profiles[name] = config
	
	// Save to file
	profile := &ConfigProfile{
		Name:        name,
		Description: description,
		Config:      config,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Tags:        []string{},
		IsDefault:   len(cm.profiles) == 1, // First profile is default
	}
	
	if err := cm.saveProfile(profile); err != nil {
		delete(cm.profiles, name)
		return fmt.Errorf("failed to save profile: %w", err)
	}
	
	cm.logger.Info("Created configuration profile", map[string]interface{}{
		"profile_name": name,
		"description":  description,
	})
	
	return nil
}

// GetProfile retrieves a configuration profile
func (cm *ChatConfigManager) GetProfile(name string) (*MemChatConfig, error) {
	config, exists := cm.profiles[name]
	if !exists {
		return nil, fmt.Errorf("profile '%s' not found", name)
	}
	
	// Return a copy to prevent modifications
	configCopy := *config
	return &configCopy, nil
}

// UpdateProfile updates an existing configuration profile
func (cm *ChatConfigManager) UpdateProfile(name string, config *MemChatConfig) error {
	if _, exists := cm.profiles[name]; !exists {
		return fmt.Errorf("profile '%s' not found", name)
	}
	
	if err := ValidateMemChatConfig(config); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	
	// Apply global defaults
	cm.applyGlobalDefaults(config)
	
	// Update profile
	cm.profiles[name] = config
	
	// Load existing profile metadata
	profile, err := cm.loadProfile(name)
	if err != nil {
		// Create new profile metadata if loading fails
		profile = &ConfigProfile{
			Name:      name,
			CreatedAt: time.Now(),
		}
	}
	
	profile.Config = config
	profile.UpdatedAt = time.Now()
	
	// Save to file
	if err := cm.saveProfile(profile); err != nil {
		return fmt.Errorf("failed to save updated profile: %w", err)
	}
	
	cm.logger.Info("Updated configuration profile", map[string]interface{}{
		"profile_name": name,
	})
	
	return nil
}

// DeleteProfile deletes a configuration profile
func (cm *ChatConfigManager) DeleteProfile(name string) error {
	if _, exists := cm.profiles[name]; !exists {
		return fmt.Errorf("profile '%s' not found", name)
	}
	
	// Remove from memory
	delete(cm.profiles, name)
	
	// Remove file
	profilePath := filepath.Join(cm.configDir, "profiles", name+".json")
	if err := os.Remove(profilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete profile file: %w", err)
	}
	
	cm.logger.Info("Deleted configuration profile", map[string]interface{}{
		"profile_name": name,
	})
	
	return nil
}

// ListProfiles returns a list of all available profiles
func (cm *ChatConfigManager) ListProfiles() []string {
	profiles := make([]string, 0, len(cm.profiles))
	for name := range cm.profiles {
		profiles = append(profiles, name)
	}
	return profiles
}

// GetDefaultProfile returns the default configuration profile
func (cm *ChatConfigManager) GetDefaultProfile() (*MemChatConfig, error) {
	// Try to find a profile marked as default
	for name := range cm.profiles {
		profile, err := cm.loadProfile(name)
		if err == nil && profile.IsDefault {
			return cm.GetProfile(name)
		}
	}
	
	// If no default found, return the first profile
	if len(cm.profiles) > 0 {
		for name := range cm.profiles {
			return cm.GetProfile(name)
		}
	}
	
	// If no profiles exist, create and return a default one
	defaultConfig := cm.CreateDefaultConfig("default", "default")
	if err := cm.CreateProfile("default", "Default configuration", defaultConfig); err != nil {
		return nil, fmt.Errorf("failed to create default profile: %w", err)
	}
	
	return cm.GetProfile("default")
}

// SetDefaultProfile sets a profile as the default
func (cm *ChatConfigManager) SetDefaultProfile(name string) error {
	if _, exists := cm.profiles[name]; !exists {
		return fmt.Errorf("profile '%s' not found", name)
	}
	
	// Update all profiles to not be default
	for profileName := range cm.profiles {
		profile, err := cm.loadProfile(profileName)
		if err != nil {
			continue
		}
		profile.IsDefault = (profileName == name)
		cm.saveProfile(profile)
	}
	
	cm.logger.Info("Set default configuration profile", map[string]interface{}{
		"profile_name": name,
	})
	
	return nil
}

// CreateDefaultConfig creates a default configuration with global settings applied
func (cm *ChatConfigManager) CreateDefaultConfig(userID, sessionID string) *MemChatConfig {
	config := DefaultMemChatConfig()
	config.UserID = userID
	config.SessionID = sessionID
	
	// Apply global defaults
	cm.applyGlobalDefaults(config)
	
	return config
}

// CreateConfigFromTemplate creates a configuration from a template
func (cm *ChatConfigManager) CreateConfigFromTemplate(template string, userID, sessionID string) (*MemChatConfig, error) {
	var config *MemChatConfig
	
	switch strings.ToLower(template) {
	case "minimal":
		config = &MemChatConfig{
			UserID:                 userID,
			SessionID:              sessionID,
			CreatedAt:              time.Now(),
			EnableTextualMemory:    false,
			EnableActivationMemory: false,
			TopK:                   3,
			MaxTurnsWindow:         5,
			Temperature:            0.7,
			MaxTokens:              512,
			StreamingEnabled:       false,
			CustomSettings:         make(map[string]interface{}),
		}
		
	case "standard":
		config = DefaultMemChatConfig()
		config.UserID = userID
		config.SessionID = sessionID
		
	case "advanced":
		config = DefaultMemChatConfig()
		config.UserID = userID
		config.SessionID = sessionID
		config.EnableActivationMemory = true
		config.TopK = 10
		config.MaxTurnsWindow = 20
		config.StreamingEnabled = true
		config.CustomSettings["enable_advanced"] = true
		
	case "research":
		config = DefaultMemChatConfig()
		config.UserID = userID
		config.SessionID = sessionID
		config.TopK = 15
		config.MaxTurnsWindow = 30
		config.SystemPrompt = "You are a research assistant AI with access to extensive memories. " +
			"Provide detailed, well-researched responses with citations and references."
		config.CustomSettings["features"] = []string{"citation", "analysis", "research"}
		
	case "creative":
		config = DefaultMemChatConfig()
		config.UserID = userID
		config.SessionID = sessionID
		config.Temperature = 0.9
		config.TopK = 8
		config.SystemPrompt = "You are a creative AI assistant. Think outside the box and provide " +
			"innovative, imaginative responses. Use memories to inspire creative solutions."
		
	default:
		return nil, fmt.Errorf("unknown template: %s", template)
	}
	
	// Apply global defaults
	cm.applyGlobalDefaults(config)
	
	return config, nil
}

// ExportProfile exports a profile to a file
func (cm *ChatConfigManager) ExportProfile(name, filePath string) error {
	config, err := cm.GetProfile(name)
	if err != nil {
		return err
	}
	
	profile := &ConfigProfile{
		Name:        name,
		Description: fmt.Sprintf("Exported profile: %s", name),
		Config:      config,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Tags:        []string{"exported"},
	}
	
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profile: %w", err)
	}
	
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write profile file: %w", err)
	}
	
	cm.logger.Info("Exported configuration profile", map[string]interface{}{
		"profile_name": name,
		"file_path":    filePath,
	})
	
	return nil
}

// ImportProfile imports a profile from a file
func (cm *ChatConfigManager) ImportProfile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read profile file: %w", err)
	}
	
	profile := &ConfigProfile{}
	if err := json.Unmarshal(data, profile); err != nil {
		return fmt.Errorf("failed to parse profile file: %w", err)
	}
	
	// Validate the configuration
	if err := ValidateMemChatConfig(profile.Config); err != nil {
		return fmt.Errorf("invalid imported config: %w", err)
	}
	
	// Create the profile
	return cm.CreateProfile(profile.Name, profile.Description, profile.Config)
}

// Helper functions
func (cm *ChatConfigManager) applyGlobalDefaults(config *MemChatConfig) {
	if cm.globalConfig == nil {
		return
	}
	
	// Apply LLM defaults
	if config.ChatLLM.Provider == "" && cm.globalConfig.DefaultLLMProvider != "" {
		config.ChatLLM.Provider = cm.globalConfig.DefaultLLMProvider
	}
	if config.ChatLLM.Model == "" && cm.globalConfig.DefaultLLMModel != "" {
		config.ChatLLM.Model = cm.globalConfig.DefaultLLMModel
	}
	if config.ChatLLM.MaxTokens == 0 && cm.globalConfig.DefaultMaxTokens > 0 {
		config.ChatLLM.MaxTokens = cm.globalConfig.DefaultMaxTokens
	}
	if config.ChatLLM.Temperature == 0 && cm.globalConfig.DefaultTemperature > 0 {
		config.ChatLLM.Temperature = cm.globalConfig.DefaultTemperature
	}
	
	// Apply chat defaults
	if config.TopK == 0 && cm.globalConfig.DefaultTopK > 0 {
		config.TopK = cm.globalConfig.DefaultTopK
	}
	if config.MaxTurnsWindow == 0 && cm.globalConfig.DefaultMaxTurnsWindow > 0 {
		config.MaxTurnsWindow = cm.globalConfig.DefaultMaxTurnsWindow
	}
	
	// Apply memory defaults
	if !config.EnableTextualMemory && cm.globalConfig.DefaultMemoryEnabled {
		config.EnableTextualMemory = cm.globalConfig.DefaultMemoryEnabled
	}
	
	// Apply custom defaults
	if cm.globalConfig.CustomDefaults != nil {
		if config.CustomSettings == nil {
			config.CustomSettings = make(map[string]interface{})
		}
		for key, value := range cm.globalConfig.CustomDefaults {
			if _, exists := config.CustomSettings[key]; !exists {
				config.CustomSettings[key] = value
			}
		}
	}
}

func (cm *ChatConfigManager) loadConfigurations() error {
	// Load global config
	if err := cm.LoadGlobalConfig(); err != nil {
		cm.logger.Warn("Failed to load global config", map[string]interface{}{
			"error": err.Error(),
		})
	}
	
	// Load profiles
	profilesDir := filepath.Join(cm.configDir, "profiles")
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		return fmt.Errorf("failed to create profiles directory: %w", err)
	}
	
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		return fmt.Errorf("failed to read profiles directory: %w", err)
	}
	
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			name := strings.TrimSuffix(entry.Name(), ".json")
			profile, err := cm.loadProfile(name)
			if err != nil {
				cm.logger.Warn("Failed to load profile", map[string]interface{}{
					"profile_name": name,
					"error":        err.Error(),
				})
				continue
			}
			cm.profiles[name] = profile.Config
		}
	}
	
	cm.logger.Info("Loaded configurations", map[string]interface{}{
		"profiles_count": len(cm.profiles),
	})
	
	return nil
}

func (cm *ChatConfigManager) loadProfile(name string) (*ConfigProfile, error) {
	profilePath := filepath.Join(cm.configDir, "profiles", name+".json")
	
	data, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile file: %w", err)
	}
	
	profile := &ConfigProfile{}
	if err := json.Unmarshal(data, profile); err != nil {
		return nil, fmt.Errorf("failed to parse profile file: %w", err)
	}
	
	return profile, nil
}

func (cm *ChatConfigManager) saveProfile(profile *ConfigProfile) error {
	profilesDir := filepath.Join(cm.configDir, "profiles")
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		return fmt.Errorf("failed to create profiles directory: %w", err)
	}
	
	profilePath := filepath.Join(profilesDir, profile.Name+".json")
	
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profile: %w", err)
	}
	
	if err := os.WriteFile(profilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write profile file: %w", err)
	}
	
	return nil
}

// DefaultGlobalChatConfig returns default global configuration
func DefaultGlobalChatConfig() *GlobalChatConfig {
	return &GlobalChatConfig{
		DefaultLLMProvider:    "openai",
		DefaultLLMModel:       "gpt-3.5-turbo",
		DefaultMaxTokens:      1024,
		DefaultTemperature:    0.7,
		DefaultTopK:           5,
		DefaultMaxTurnsWindow: 10,
		
		MaxConcurrentSessions: 100,
		SessionTimeout:        time.Hour * 24,
		LogLevel:              "info",
		MetricsEnabled:        true,
		
		DefaultMemoryEnabled:  true,
		MemoryRetentionPeriod: "30d",
		
		APIKeyEncryption:     true,
		AllowedLLMProviders:  []string{"openai", "ollama", "huggingface"},
		RateLimitEnabled:     false,
		MaxRequestsPerMinute: 60,
		
		EnabledFeatures:      []string{"textual_memory", "conversation_history", "metrics"},
		ExperimentalFeatures: []string{},
		PluginDirectories:    []string{},
		
		CustomDefaults: make(map[string]interface{}),
	}
}

// validateGlobalConfig validates global configuration
func validateGlobalConfig(config *GlobalChatConfig) error {
	if config.DefaultTemperature < 0 || config.DefaultTemperature > 2 {
		return fmt.Errorf("default_temperature must be between 0 and 2")
	}
	
	if config.DefaultTopK < 0 {
		return fmt.Errorf("default_top_k must be non-negative")
	}
	
	if config.DefaultMaxTurnsWindow < 0 {
		return fmt.Errorf("default_max_turns_window must be non-negative")
	}
	
	if config.MaxConcurrentSessions < 1 {
		return fmt.Errorf("max_concurrent_sessions must be at least 1")
	}
	
	if config.MaxRequestsPerMinute < 1 && config.RateLimitEnabled {
		return fmt.Errorf("max_requests_per_minute must be at least 1 when rate limiting is enabled")
	}
	
	return nil
}

// GetAvailableTemplates returns a list of available configuration templates
func GetAvailableTemplates() []string {
	return []string{
		"minimal",
		"standard",
		"advanced",
		"research",
		"creative",
	}
}

// GetTemplateDescription returns a description of a configuration template
func GetTemplateDescription(template string) string {
	descriptions := map[string]string{
		"minimal":  "Minimal configuration with basic features and low resource usage",
		"standard": "Standard configuration with balanced features and performance",
		"advanced": "Advanced configuration with all features enabled and high resource usage",
		"research": "Optimized for research tasks with extensive memory and citation features",
		"creative": "Optimized for creative tasks with high temperature and imaginative prompts",
	}
	
	if desc, exists := descriptions[strings.ToLower(template)]; exists {
		return desc
	}
	
	return "Unknown template"
}