// Package chat - MemChat Factory implementation
package chat

import (
	"fmt"
	"strings"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/logger"
)

// MemChatBackend represents the type of MemChat backend
type MemChatBackend string

const (
	// Available MemChat backends
	MemChatBackendSimple     MemChatBackend = "simple"
	MemChatBackendAdvanced   MemChatBackend = "advanced"
	MemChatBackendMultiModal MemChatBackend = "multimodal"
	MemChatBackendCustom     MemChatBackend = "custom"
)

// MemChatFactory creates MemChat instances based on configuration
type MemChatFactory struct {
	logger        interfaces.Logger
	registeredBackends map[MemChatBackend]MemChatCreator
}

// MemChatCreator defines the function signature for creating MemChat instances
type MemChatCreator func(config *MemChatConfig) (BaseMemChat, error)

// NewMemChatFactory creates a new MemChat factory
func NewMemChatFactory() *MemChatFactory {
	factory := &MemChatFactory{
		logger:             logger.NewLogger("MemChatFactory"),
		registeredBackends: make(map[MemChatBackend]MemChatCreator),
	}
	
	// Register default backends
	factory.registerDefaultBackends()
	
	return factory
}

// registerDefaultBackends registers the built-in MemChat backends
func (f *MemChatFactory) registerDefaultBackends() {
	// Register Simple MemChat
	f.registeredBackends[MemChatBackendSimple] = func(config *MemChatConfig) (BaseMemChat, error) {
		return NewSimpleMemChat(config)
	}
	
	// Register Advanced MemChat (placeholder for future implementation)
	f.registeredBackends[MemChatBackendAdvanced] = func(config *MemChatConfig) (BaseMemChat, error) {
		return nil, fmt.Errorf("advanced MemChat backend not yet implemented")
	}
	
	// Register MultiModal MemChat (placeholder for future implementation)
	f.registeredBackends[MemChatBackendMultiModal] = func(config *MemChatConfig) (BaseMemChat, error) {
		return nil, fmt.Errorf("multimodal MemChat backend not yet implemented")
	}
	
	f.logger.Info("Registered default MemChat backends", map[string]interface{}{
		"backends": []string{string(MemChatBackendSimple), string(MemChatBackendAdvanced), string(MemChatBackendMultiModal)},
	})
}

// RegisterBackend registers a custom MemChat backend
func (f *MemChatFactory) RegisterBackend(backend MemChatBackend, creator MemChatCreator) error {
	if creator == nil {
		return fmt.Errorf("creator function cannot be nil")
	}
	
	f.registeredBackends[backend] = creator
	
	f.logger.Info("Registered custom MemChat backend", map[string]interface{}{
		"backend": string(backend),
	})
	
	return nil
}

// UnregisterBackend removes a MemChat backend
func (f *MemChatFactory) UnregisterBackend(backend MemChatBackend) {
	delete(f.registeredBackends, backend)
	
	f.logger.Info("Unregistered MemChat backend", map[string]interface{}{
		"backend": string(backend),
	})
}

// GetAvailableBackends returns a list of available MemChat backends
func (f *MemChatFactory) GetAvailableBackends() []MemChatBackend {
	backends := make([]MemChatBackend, 0, len(f.registeredBackends))
	for backend := range f.registeredBackends {
		backends = append(backends, backend)
	}
	return backends
}

// IsBackendAvailable checks if a backend is available
func (f *MemChatFactory) IsBackendAvailable(backend MemChatBackend) bool {
	_, exists := f.registeredBackends[backend]
	return exists
}

// CreateFromBackend creates a MemChat instance from a specific backend
func (f *MemChatFactory) CreateFromBackend(backend MemChatBackend, config *MemChatConfig) (BaseMemChat, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	
	if err := ValidateMemChatConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	
	creator, exists := f.registeredBackends[backend]
	if !exists {
		return nil, fmt.Errorf("backend '%s' is not registered", backend)
	}
	
	chat, err := creator(config)
	if err != nil {
		f.logger.Error("Failed to create MemChat instance", err, map[string]interface{}{
			"backend": string(backend),
		})
		return nil, fmt.Errorf("failed to create MemChat with backend '%s': %w", backend, err)
	}
	
	f.logger.Info("Created MemChat instance", map[string]interface{}{
		"backend":    string(backend),
		"user_id":    config.UserID,
		"session_id": config.SessionID,
	})
	
	return chat, nil
}

// CreateFromConfig creates a MemChat instance based on configuration
func (f *MemChatFactory) CreateFromConfig(config *MemChatConfig) (BaseMemChat, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	
	// Determine backend from config
	backend := f.determineBackendFromConfig(config)
	
	return f.CreateFromBackend(backend, config)
}

// CreateSimple creates a simple MemChat instance with minimal configuration
func (f *MemChatFactory) CreateSimple(userID, sessionID string) (BaseMemChat, error) {
	config := DefaultMemChatConfig()
	config.UserID = userID
	config.SessionID = sessionID
	
	return f.CreateFromBackend(MemChatBackendSimple, config)
}

// CreateWithLLM creates a MemChat instance with specific LLM configuration
func (f *MemChatFactory) CreateWithLLM(userID, sessionID string, llmConfig LLMConfig) (BaseMemChat, error) {
	config := DefaultMemChatConfig()
	config.UserID = userID
	config.SessionID = sessionID
	config.ChatLLM = llmConfig
	
	return f.CreateFromBackend(MemChatBackendSimple, config)
}

// CreateWithMemory creates a MemChat instance with memory configuration
func (f *MemChatFactory) CreateWithMemory(userID, sessionID string, memCube interfaces.MemCube, topK int) (BaseMemChat, error) {
	config := DefaultMemChatConfig()
	config.UserID = userID
	config.SessionID = sessionID
	config.EnableTextualMemory = true
	config.TopK = topK
	
	chat, err := f.CreateFromBackend(MemChatBackendSimple, config)
	if err != nil {
		return nil, err
	}
	
	if memCube != nil {
		err = chat.SetMemCube(memCube)
		if err != nil {
			return nil, fmt.Errorf("failed to set memory cube: %w", err)
		}
	}
	
	return chat, nil
}

// CreateAdvanced creates an advanced MemChat instance (when implemented)
func (f *MemChatFactory) CreateAdvanced(userID, sessionID string, advancedConfig map[string]interface{}) (BaseMemChat, error) {
	config := DefaultMemChatConfig()
	config.UserID = userID
	config.SessionID = sessionID
	config.CustomSettings = advancedConfig
	
	return f.CreateFromBackend(MemChatBackendAdvanced, config)
}

// CreateMultiModal creates a multi-modal MemChat instance (when implemented)
func (f *MemChatFactory) CreateMultiModal(userID, sessionID string, multiModalConfig map[string]interface{}) (BaseMemChat, error) {
	config := DefaultMemChatConfig()
	config.UserID = userID
	config.SessionID = sessionID
	config.CustomSettings = multiModalConfig
	
	return f.CreateFromBackend(MemChatBackendMultiModal, config)
}

// determineBackendFromConfig determines the best backend based on configuration
func (f *MemChatFactory) determineBackendFromConfig(config *MemChatConfig) MemChatBackend {
	// Check if custom backend is specified
	if backendName, exists := config.CustomSettings["backend"]; exists {
		if backendStr, ok := backendName.(string); ok {
			backend := MemChatBackend(backendStr)
			if f.IsBackendAvailable(backend) {
				return backend
			}
		}
	}
	
	// Auto-select based on features
	if f.requiresMultiModal(config) && f.IsBackendAvailable(MemChatBackendMultiModal) {
		return MemChatBackendMultiModal
	}
	
	if f.requiresAdvanced(config) && f.IsBackendAvailable(MemChatBackendAdvanced) {
		return MemChatBackendAdvanced
	}
	
	// Default to simple
	return MemChatBackendSimple
}

// requiresMultiModal checks if config requires multi-modal capabilities
func (f *MemChatFactory) requiresMultiModal(config *MemChatConfig) bool {
	if enableMultiModal, exists := config.CustomSettings["enable_multimodal"]; exists {
		if enabled, ok := enableMultiModal.(bool); ok && enabled {
			return true
		}
	}
	
	if features, exists := config.CustomSettings["features"]; exists {
		if featureList, ok := features.([]string); ok {
			for _, feature := range featureList {
				if strings.Contains(strings.ToLower(feature), "image") ||
				   strings.Contains(strings.ToLower(feature), "document") ||
				   strings.Contains(strings.ToLower(feature), "file") {
					return true
				}
			}
		}
	}
	
	return false
}

// requiresAdvanced checks if config requires advanced capabilities
func (f *MemChatFactory) requiresAdvanced(config *MemChatConfig) bool {
	if enableAdvanced, exists := config.CustomSettings["enable_advanced"]; exists {
		if enabled, ok := enableAdvanced.(bool); ok && enabled {
			return true
		}
	}
	
	if features, exists := config.CustomSettings["features"]; exists {
		if featureList, ok := features.([]string); ok {
			for _, feature := range featureList {
				if strings.Contains(strings.ToLower(feature), "semantic") ||
				   strings.Contains(strings.ToLower(feature), "cluster") ||
				   strings.Contains(strings.ToLower(feature), "analysis") ||
				   strings.Contains(strings.ToLower(feature), "learning") {
					return true
				}
			}
		}
	}
	
	// Check if advanced memory features are needed
	if config.EnableActivationMemory {
		return true
	}
	
	return false
}

// GetBackendCapabilities returns the capabilities of a specific backend
func (f *MemChatFactory) GetBackendCapabilities(backend MemChatBackend) map[string]interface{} {
	capabilities := make(map[string]interface{})
	
	switch backend {
	case MemChatBackendSimple:
		capabilities["description"] = "Basic memory-augmented chat with textual memory support"
		capabilities["features"] = []string{
			"textual_memory",
			"conversation_history",
			"memory_extraction",
			"conversation_modes",
			"command_handling",
			"metrics_tracking",
		}
		capabilities["max_complexity"] = "medium"
		capabilities["performance"] = "high"
		capabilities["memory_types"] = []string{"textual"}
		
	case MemChatBackendAdvanced:
		capabilities["description"] = "Advanced chat with semantic search, clustering, and learning"
		capabilities["features"] = []string{
			"semantic_search",
			"memory_clustering",
			"conversation_analysis",
			"adaptive_learning",
			"response_validation",
			"citation_generation",
			"topic_detection",
			"personalization",
		}
		capabilities["max_complexity"] = "high"
		capabilities["performance"] = "medium"
		capabilities["memory_types"] = []string{"textual", "activation", "parametric"}
		
	case MemChatBackendMultiModal:
		capabilities["description"] = "Multi-modal chat supporting text, images, and documents"
		capabilities["features"] = []string{
			"image_processing",
			"document_analysis",
			"file_support",
			"content_summarization",
			"multi_modal_memory",
			"visual_understanding",
		}
		capabilities["max_complexity"] = "very_high"
		capabilities["performance"] = "medium"
		capabilities["memory_types"] = []string{"textual", "visual", "document"}
		capabilities["supported_formats"] = []string{
			"text", "markdown", "pdf", "docx", "jpg", "png", "gif",
		}
		
	default:
		capabilities["description"] = "Unknown backend"
		capabilities["features"] = []string{}
	}
	
	capabilities["available"] = f.IsBackendAvailable(backend)
	
	return capabilities
}

// GetAllCapabilities returns capabilities for all registered backends
func (f *MemChatFactory) GetAllCapabilities() map[string]interface{} {
	allCapabilities := make(map[string]interface{})
	
	for backend := range f.registeredBackends {
		allCapabilities[string(backend)] = f.GetBackendCapabilities(backend)
	}
	
	return allCapabilities
}

// ValidateBackendConfig validates configuration for a specific backend
func (f *MemChatFactory) ValidateBackendConfig(backend MemChatBackend, config *MemChatConfig) error {
	if err := ValidateMemChatConfig(config); err != nil {
		return err
	}
	
	switch backend {
	case MemChatBackendSimple:
		// Simple backend is flexible and accepts most configurations
		return nil
		
	case MemChatBackendAdvanced:
		// Advanced backend might require specific features
		if !config.EnableTextualMemory {
			return fmt.Errorf("advanced backend requires textual memory to be enabled")
		}
		
	case MemChatBackendMultiModal:
		// Multi-modal backend requirements
		if features, exists := config.CustomSettings["supported_formats"]; exists {
			if formatList, ok := features.([]string); ok && len(formatList) == 0 {
				return fmt.Errorf("multi-modal backend requires at least one supported format")
			}
		}
		
	default:
		// For custom backends, we can't validate much
		return nil
	}
	
	return nil
}

// GetRecommendedBackend suggests the best backend for given requirements
func (f *MemChatFactory) GetRecommendedBackend(requirements map[string]interface{}) MemChatBackend {
	// Check for multi-modal requirements
	if needsMultiModal, exists := requirements["multimodal"]; exists {
		if needs, ok := needsMultiModal.(bool); ok && needs {
			if f.IsBackendAvailable(MemChatBackendMultiModal) {
				return MemChatBackendMultiModal
			}
		}
	}
	
	// Check for advanced features
	if needsAdvanced, exists := requirements["advanced"]; exists {
		if needs, ok := needsAdvanced.(bool); ok && needs {
			if f.IsBackendAvailable(MemChatBackendAdvanced) {
				return MemChatBackendAdvanced
			}
		}
	}
	
	// Check performance requirements
	if perfReq, exists := requirements["performance"]; exists {
		if perf, ok := perfReq.(string); ok && perf == "high" {
			return MemChatBackendSimple
		}
	}
	
	// Default recommendation
	return MemChatBackendSimple
}

// CreateFromFactory creates a MemChat instance using factory-pattern configuration
type FactoryConfig struct {
	Backend      MemChatBackend         `json:"backend"`
	UserID       string                 `json:"user_id"`
	SessionID    string                 `json:"session_id"`
	LLMProvider  string                 `json:"llm_provider"`
	LLMModel     string                 `json:"llm_model"`
	Features     []string               `json:"features"`
	Settings     map[string]interface{} `json:"settings"`
}

// CreateFromFactoryConfig creates a MemChat instance from factory configuration
func (f *MemChatFactory) CreateFromFactoryConfig(factoryConfig *FactoryConfig) (BaseMemChat, error) {
	if factoryConfig == nil {
		return nil, fmt.Errorf("factory config cannot be nil")
	}
	
	// Build MemChat config from factory config
	config := DefaultMemChatConfig()
	config.UserID = factoryConfig.UserID
	config.SessionID = factoryConfig.SessionID
	
	if factoryConfig.LLMProvider != "" {
		config.ChatLLM.Provider = factoryConfig.LLMProvider
	}
	if factoryConfig.LLMModel != "" {
		config.ChatLLM.Model = factoryConfig.LLMModel
	}
	
	// Apply feature settings
	for _, feature := range factoryConfig.Features {
		switch strings.ToLower(feature) {
		case "textual_memory":
			config.EnableTextualMemory = true
		case "activation_memory":
			config.EnableActivationMemory = true
		case "streaming":
			config.StreamingEnabled = true
		}
	}
	
	// Apply custom settings
	if factoryConfig.Settings != nil {
		config.CustomSettings = factoryConfig.Settings
	}
	
	// Validate backend or auto-select
	backend := factoryConfig.Backend
	if backend == "" {
		backend = f.determineBackendFromConfig(config)
	}
	
	return f.CreateFromBackend(backend, config)
}

// Global factory instance
var globalFactory *MemChatFactory

// GetGlobalFactory returns the global MemChat factory instance
func GetGlobalFactory() *MemChatFactory {
	if globalFactory == nil {
		globalFactory = NewMemChatFactory()
	}
	return globalFactory
}

// Convenience functions using global factory
func CreateSimpleChat(userID, sessionID string) (BaseMemChat, error) {
	return GetGlobalFactory().CreateSimple(userID, sessionID)
}

func CreateChatWithLLM(userID, sessionID string, llmConfig LLMConfig) (BaseMemChat, error) {
	return GetGlobalFactory().CreateWithLLM(userID, sessionID, llmConfig)
}

func CreateChatWithMemory(userID, sessionID string, memCube interfaces.MemCube, topK int) (BaseMemChat, error) {
	return GetGlobalFactory().CreateWithMemory(userID, sessionID, memCube, topK)
}

func CreateFromConfig(config *MemChatConfig) (BaseMemChat, error) {
	return GetGlobalFactory().CreateFromConfig(config)
}

func GetAvailableBackends() []MemChatBackend {
	return GetGlobalFactory().GetAvailableBackends()
}

func GetBackendCapabilities(backend MemChatBackend) map[string]interface{} {
	return GetGlobalFactory().GetBackendCapabilities(backend)
}