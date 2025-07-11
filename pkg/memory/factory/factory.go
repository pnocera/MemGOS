// Package factory provides memory factory for dynamic memory creation
package factory

import (
	"fmt"

	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/memory/activation"
	"github.com/memtensor/memgos/pkg/memory/parametric"
	"github.com/memtensor/memgos/pkg/memory/textual"
	"github.com/memtensor/memgos/pkg/types"
)

// MemoryType represents the type of memory to create
type MemoryType string

const (
	// Textual memory types
	MemoryTypeNaiveTextual   MemoryType = "naive_textual"
	MemoryTypeGeneralTextual MemoryType = "general_textual"
	MemoryTypeTreeTextual    MemoryType = "tree_textual"
	
	// Activation memory types
	MemoryTypeKVCache        MemoryType = "kv_cache"
	MemoryTypeActivation     MemoryType = "activation"
	
	// Parametric memory types
	MemoryTypeLoRA           MemoryType = "lora"
	MemoryTypeLoHa           MemoryType = "loha"
	MemoryTypeLoKr           MemoryType = "lokr"
	MemoryTypeIA3            MemoryType = "ia3"
	MemoryTypeParametric     MemoryType = "parametric"
)

// MemoryFactory provides factory methods for creating different types of memories
type MemoryFactory struct {
	config           *config.MOSConfig
	logger           interfaces.Logger
	metrics          interfaces.Metrics
	embedderRegistry map[string]interfaces.Embedder
	llmRegistry      map[string]interfaces.LLM
	vectorDBRegistry map[string]interfaces.VectorDB
	graphDBRegistry  map[string]interfaces.GraphDB
}

// MemoryConfig represents configuration for memory creation
type MemoryConfig struct {
	Type            MemoryType             `json:"type"`
	Name            string                 `json:"name"`
	Backend         types.MemoryBackend    `json:"backend"`
	Config          *config.MemoryConfig   `json:"config"`
	EmbedderType    string                 `json:"embedder_type"`
	LLMType         string                 `json:"llm_type"`
	VectorDBType    string                 `json:"vector_db_type"`
	GraphDBType     string                 `json:"graph_db_type"`
	Parameters      map[string]interface{} `json:"parameters"`
	EnableCaching   bool                   `json:"enable_caching"`
	EnableMetrics   bool                   `json:"enable_metrics"`
	CompressionMode string                 `json:"compression_mode"`
	OptimizeOnInit  bool                   `json:"optimize_on_init"`
}

// MemoryInstance represents a created memory instance
type MemoryInstance struct {
	ID          string      `json:"id"`
	Type        MemoryType  `json:"type"`
	Name        string      `json:"name"`
	Memory      interface{} `json:"memory"`
	Config      *MemoryConfig `json:"config"`
	CreatedAt   string      `json:"created_at"`
	Status      string      `json:"status"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NewMemoryFactory creates a new memory factory
func NewMemoryFactory(cfg *config.MOSConfig, logger interfaces.Logger, metrics interfaces.Metrics) *MemoryFactory {
	return &MemoryFactory{
		config:           cfg,
		logger:           logger,
		metrics:          metrics,
		embedderRegistry: make(map[string]interfaces.Embedder),
		llmRegistry:      make(map[string]interfaces.LLM),
		vectorDBRegistry: make(map[string]interfaces.VectorDB),
		graphDBRegistry:  make(map[string]interfaces.GraphDB),
	}
}

// RegisterEmbedder registers an embedder with the factory
func (mf *MemoryFactory) RegisterEmbedder(name string, embedder interfaces.Embedder) {
	mf.embedderRegistry[name] = embedder
}

// RegisterLLM registers an LLM with the factory
func (mf *MemoryFactory) RegisterLLM(name string, llm interfaces.LLM) {
	mf.llmRegistry[name] = llm
}

// RegisterVectorDB registers a vector database with the factory
func (mf *MemoryFactory) RegisterVectorDB(name string, vectorDB interfaces.VectorDB) {
	mf.vectorDBRegistry[name] = vectorDB
}

// RegisterGraphDB registers a graph database with the factory
func (mf *MemoryFactory) RegisterGraphDB(name string, graphDB interfaces.GraphDB) {
	mf.graphDBRegistry[name] = graphDB
}

// CreateMemory creates a memory instance based on the provided configuration
func (mf *MemoryFactory) CreateMemory(memoryConfig *MemoryConfig) (*MemoryInstance, error) {
	if memoryConfig == nil {
		return nil, fmt.Errorf("memory configuration is required")
	}
	
	// Get required components
	embedder := mf.getEmbedder(memoryConfig.EmbedderType)
	llm := mf.getLLM(memoryConfig.LLMType)
	vectorDB := mf.getVectorDB(memoryConfig.VectorDBType)
	graphDB := mf.getGraphDB(memoryConfig.GraphDBType)
	
	// Create memory based on type
	var memory interface{}
	var err error
	
	switch memoryConfig.Type {
	case MemoryTypeNaiveTextual:
		memory, err = mf.createNaiveTextualMemory(memoryConfig, embedder, vectorDB, llm)
	case MemoryTypeGeneralTextual:
		memory, err = mf.createGeneralTextualMemory(memoryConfig, embedder, vectorDB, llm)
	case MemoryTypeTreeTextual:
		memory, err = mf.createTreeTextualMemory(memoryConfig, embedder, vectorDB, graphDB, llm)
	case MemoryTypeKVCache:
		memory, err = mf.createKVCacheMemory(memoryConfig, llm)
	case MemoryTypeActivation:
		memory, err = mf.createActivationMemory(memoryConfig, llm)
	case MemoryTypeLoRA:
		memory, err = mf.createLoRAMemory(memoryConfig)
	case MemoryTypeParametric:
		memory, err = mf.createParametricMemory(memoryConfig)
	default:
		return nil, fmt.Errorf("unsupported memory type: %s", memoryConfig.Type)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to create memory: %w", err)
	}
	
	// Create memory instance
	instance := &MemoryInstance{
		ID:        mf.generateID(),
		Type:      memoryConfig.Type,
		Name:      memoryConfig.Name,
		Memory:    memory,
		Config:    memoryConfig,
		CreatedAt: mf.getCurrentTime(),
		Status:    "active",
		Metadata:  make(map[string]interface{}),
	}
	
	// Set metadata
	instance.Metadata["embedder_type"] = memoryConfig.EmbedderType
	instance.Metadata["llm_type"] = memoryConfig.LLMType
	instance.Metadata["vector_db_type"] = memoryConfig.VectorDBType
	instance.Metadata["graph_db_type"] = memoryConfig.GraphDBType
	instance.Metadata["enable_caching"] = memoryConfig.EnableCaching
	instance.Metadata["enable_metrics"] = memoryConfig.EnableMetrics
	
	// Optimize if requested
	if memoryConfig.OptimizeOnInit {
		if err := mf.optimizeMemory(memory); err != nil {
			mf.logger.Error("Failed to optimize memory on initialization", err, map[string]interface{}{
				"memory_id": instance.ID,
			})
		}
	}
	
	mf.logger.Info("Created memory instance", map[string]interface{}{
		"memory_id":   instance.ID,
		"memory_type": string(memoryConfig.Type),
		"memory_name": memoryConfig.Name,
	})
	
	return instance, nil
}

// CreateTextualMemory creates a textual memory with specific backend
func (mf *MemoryFactory) CreateTextualMemory(backend types.MemoryBackend, name string, config *config.MemoryConfig) (*MemoryInstance, error) {
	var memoryType MemoryType
	
	switch backend {
	case types.MemoryBackendNaive:
		memoryType = MemoryTypeNaiveTextual
	case types.MemoryBackendGeneral:
		memoryType = MemoryTypeGeneralTextual
	case types.MemoryBackendTree, types.MemoryBackendTreeText:
		memoryType = MemoryTypeTreeTextual
	default:
		return nil, fmt.Errorf("unsupported textual memory backend: %s", backend)
	}
	
	memoryConfig := &MemoryConfig{
		Type:            memoryType,
		Name:            name,
		Backend:         backend,
		Config:          config,
		EmbedderType:    "default",
		LLMType:         "default",
		VectorDBType:    "default",
		GraphDBType:     "default",
		EnableCaching:   true,
		EnableMetrics:   true,
		CompressionMode: "lz4",
		OptimizeOnInit:  false,
	}
	
	return mf.CreateMemory(memoryConfig)
}

// CreateActivationMemory creates an activation memory with specific backend
func (mf *MemoryFactory) CreateActivationMemory(backend types.MemoryBackend, name string, config *config.MemoryConfig) (*MemoryInstance, error) {
	var memoryType MemoryType
	
	switch backend {
	case types.MemoryBackendKVCache:
		memoryType = MemoryTypeKVCache
	default:
		memoryType = MemoryTypeActivation
	}
	
	memoryConfig := &MemoryConfig{
		Type:            memoryType,
		Name:            name,
		Backend:         backend,
		Config:          config,
		LLMType:         "default",
		EnableCaching:   true,
		EnableMetrics:   true,
		CompressionMode: "zstd",
		OptimizeOnInit:  true,
	}
	
	return mf.CreateMemory(memoryConfig)
}

// CreateParametricMemory creates a parametric memory with specific backend
func (mf *MemoryFactory) CreateParametricMemory(backend types.MemoryBackend, name string, config *config.MemoryConfig) (*MemoryInstance, error) {
	var memoryType MemoryType
	
	switch backend {
	case types.MemoryBackendLoRA:
		memoryType = MemoryTypeLoRA
	default:
		memoryType = MemoryTypeParametric
	}
	
	memoryConfig := &MemoryConfig{
		Type:            memoryType,
		Name:            name,
		Backend:         backend,
		Config:          config,
		EnableCaching:   true,
		EnableMetrics:   true,
		CompressionMode: "custom",
		OptimizeOnInit:  false,
	}
	
	return mf.CreateMemory(memoryConfig)
}

// CreateMemoryFromTemplate creates memory from a predefined template
func (mf *MemoryFactory) CreateMemoryFromTemplate(templateName string, parameters map[string]interface{}) (*MemoryInstance, error) {
	template, err := mf.getTemplate(templateName)
	if err != nil {
		return nil, err
	}
	
	// Apply parameters to template
	memoryConfig := mf.applyParametersToTemplate(template, parameters)
	
	return mf.CreateMemory(memoryConfig)
}

// ListAvailableTypes returns available memory types
func (mf *MemoryFactory) ListAvailableTypes() []MemoryType {
	return []MemoryType{
		MemoryTypeNaiveTextual,
		MemoryTypeGeneralTextual,
		MemoryTypeTreeTextual,
		MemoryTypeKVCache,
		MemoryTypeActivation,
		MemoryTypeLoRA,
		MemoryTypeParametric,
	}
}

// GetMemoryCapabilities returns capabilities for a memory type
func (mf *MemoryFactory) GetMemoryCapabilities(memoryType MemoryType) (map[string]interface{}, error) {
	capabilities := make(map[string]interface{})
	
	switch memoryType {
	case MemoryTypeNaiveTextual:
		capabilities["search_modes"] = []string{"simple", "keyword"}
		capabilities["supports_semantic_search"] = false
		capabilities["supports_internet_retrieval"] = false
		capabilities["max_memory_size"] = 100000
		capabilities["compression_supported"] = true
	case MemoryTypeGeneralTextual:
		capabilities["search_modes"] = []string{"simple", "keyword", "semantic"}
		capabilities["supports_semantic_search"] = true
		capabilities["supports_internet_retrieval"] = true
		capabilities["max_memory_size"] = 1000000
		capabilities["compression_supported"] = true
	case MemoryTypeTreeTextual:
		capabilities["search_modes"] = []string{"fast", "fine", "semantic", "graph"}
		capabilities["supports_semantic_search"] = true
		capabilities["supports_internet_retrieval"] = true
		capabilities["supports_graph_operations"] = true
		capabilities["supports_working_memory"] = true
		capabilities["max_memory_size"] = 10000000
		capabilities["compression_supported"] = true
	case MemoryTypeKVCache:
		capabilities["supports_cache_merging"] = true
		capabilities["supports_compression"] = true
		capabilities["supports_device_management"] = true
		capabilities["max_cache_size"] = 8 * 1024 * 1024 * 1024 // 8GB
		capabilities["eviction_policies"] = []string{"lru", "lfu", "fifo"}
	case MemoryTypeActivation:
		capabilities["supports_tensor_storage"] = true
		capabilities["supports_compression"] = true
		capabilities["supports_device_management"] = true
		capabilities["max_activation_size"] = 16 * 1024 * 1024 * 1024 // 16GB
	case MemoryTypeLoRA:
		capabilities["supports_adapter_composition"] = true
		capabilities["supports_fine_tuning"] = true
		capabilities["supports_version_control"] = true
		capabilities["supports_model_checkpoints"] = true
		capabilities["adapter_types"] = []string{"lora", "loha", "lokr", "ia3"}
		capabilities["max_adapters"] = 1000
	case MemoryTypeParametric:
		capabilities["supports_parameter_storage"] = true
		capabilities["supports_optimization"] = true
		capabilities["supports_compression"] = true
		capabilities["max_parameter_size"] = 100 * 1024 * 1024 * 1024 // 100GB
	default:
		return nil, fmt.Errorf("unknown memory type: %s", memoryType)
	}
	
	return capabilities, nil
}

// ValidateConfiguration validates memory configuration
func (mf *MemoryFactory) ValidateConfiguration(memoryConfig *MemoryConfig) error {
	if memoryConfig == nil {
		return fmt.Errorf("memory configuration is required")
	}
	
	if memoryConfig.Type == "" {
		return fmt.Errorf("memory type is required")
	}
	
	if memoryConfig.Name == "" {
		return fmt.Errorf("memory name is required")
	}
	
	// Validate memory type
	validTypes := mf.ListAvailableTypes()
	valid := false
	for _, validType := range validTypes {
		if memoryConfig.Type == validType {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid memory type: %s", memoryConfig.Type)
	}
	
	// Validate dependencies
	if mf.requiresEmbedder(memoryConfig.Type) && memoryConfig.EmbedderType == "" {
		return fmt.Errorf("embedder is required for memory type: %s", memoryConfig.Type)
	}
	
	if mf.requiresLLM(memoryConfig.Type) && memoryConfig.LLMType == "" {
		return fmt.Errorf("LLM is required for memory type: %s", memoryConfig.Type)
	}
	
	if mf.requiresVectorDB(memoryConfig.Type) && memoryConfig.VectorDBType == "" {
		return fmt.Errorf("VectorDB is required for memory type: %s", memoryConfig.Type)
	}
	
	if mf.requiresGraphDB(memoryConfig.Type) && memoryConfig.GraphDBType == "" {
		return fmt.Errorf("GraphDB is required for memory type: %s", memoryConfig.Type)
	}
	
	return nil
}

// Private helper methods

func (mf *MemoryFactory) createNaiveTextualMemory(memoryConfig *MemoryConfig, embedder interfaces.Embedder, vectorDB interfaces.VectorDB, llm interfaces.LLM) (textual.TextualMemoryInterface, error) {
	// For now, return tree textual memory as naive implementation
	return mf.createTreeTextualMemory(memoryConfig, embedder, vectorDB, nil, llm)
}

func (mf *MemoryFactory) createGeneralTextualMemory(memoryConfig *MemoryConfig, embedder interfaces.Embedder, vectorDB interfaces.VectorDB, llm interfaces.LLM) (textual.TextualMemoryInterface, error) {
	// For now, return tree textual memory as general implementation
	return mf.createTreeTextualMemory(memoryConfig, embedder, vectorDB, nil, llm)
}

func (mf *MemoryFactory) createTreeTextualMemory(memoryConfig *MemoryConfig, embedder interfaces.Embedder, vectorDB interfaces.VectorDB, graphDB interfaces.GraphDB, llm interfaces.LLM) (textual.TextualMemoryInterface, error) {
	if memoryConfig.Config == nil {
		memoryConfig.Config = &config.MemoryConfig{
			MemoryFilename: "textual_memory.json",
		}
	}
	
	return textual.NewTreeTextMemory(memoryConfig.Config, embedder, vectorDB, graphDB, llm, mf.logger, mf.metrics)
}

func (mf *MemoryFactory) createKVCacheMemory(memoryConfig *MemoryConfig, llm interfaces.LLM) (activation.ActivationMemoryInterface, error) {
	if memoryConfig.Config == nil {
		memoryConfig.Config = &config.MemoryConfig{
			MemoryFilename: "kv_cache_memory.json",
		}
	}
	
	return activation.NewKVCacheMemory(memoryConfig.Config, llm, mf.logger, mf.metrics)
}

func (mf *MemoryFactory) createActivationMemory(memoryConfig *MemoryConfig, llm interfaces.LLM) (activation.ActivationMemoryInterface, error) {
	// For now, return KV cache memory as activation implementation
	return mf.createKVCacheMemory(memoryConfig, llm)
}

func (mf *MemoryFactory) createLoRAMemory(memoryConfig *MemoryConfig) (parametric.ParametricMemoryInterface, error) {
	if memoryConfig.Config == nil {
		memoryConfig.Config = &config.MemoryConfig{
			MemoryFilename: "lora_memory.json",
		}
	}
	
	return parametric.NewLoRAMemory(memoryConfig.Config, mf.logger, mf.metrics)
}

func (mf *MemoryFactory) createParametricMemory(memoryConfig *MemoryConfig) (parametric.ParametricMemoryInterface, error) {
	// For now, return LoRA memory as parametric implementation
	return mf.createLoRAMemory(memoryConfig)
}

func (mf *MemoryFactory) getEmbedder(embedderType string) interfaces.Embedder {
	if embedderType == "" || embedderType == "default" {
		// Return first available embedder
		for _, embedder := range mf.embedderRegistry {
			return embedder
		}
		return nil
	}
	
	return mf.embedderRegistry[embedderType]
}

func (mf *MemoryFactory) getLLM(llmType string) interfaces.LLM {
	if llmType == "" || llmType == "default" {
		// Return first available LLM
		for _, llm := range mf.llmRegistry {
			return llm
		}
		return nil
	}
	
	return mf.llmRegistry[llmType]
}

func (mf *MemoryFactory) getVectorDB(vectorDBType string) interfaces.VectorDB {
	if vectorDBType == "" || vectorDBType == "default" {
		// Return first available vector DB
		for _, vectorDB := range mf.vectorDBRegistry {
			return vectorDB
		}
		return nil
	}
	
	return mf.vectorDBRegistry[vectorDBType]
}

func (mf *MemoryFactory) getGraphDB(graphDBType string) interfaces.GraphDB {
	if graphDBType == "" || graphDBType == "default" {
		// Return first available graph DB
		for _, graphDB := range mf.graphDBRegistry {
			return graphDB
		}
		return nil
	}
	
	return mf.graphDBRegistry[graphDBType]
}

func (mf *MemoryFactory) requiresEmbedder(memoryType MemoryType) bool {
	return memoryType == MemoryTypeGeneralTextual || memoryType == MemoryTypeTreeTextual
}

func (mf *MemoryFactory) requiresLLM(memoryType MemoryType) bool {
	return memoryType != MemoryTypeNaiveTextual
}

func (mf *MemoryFactory) requiresVectorDB(memoryType MemoryType) bool {
	return memoryType == MemoryTypeGeneralTextual || memoryType == MemoryTypeTreeTextual
}

func (mf *MemoryFactory) requiresGraphDB(memoryType MemoryType) bool {
	return memoryType == MemoryTypeTreeTextual
}

func (mf *MemoryFactory) optimizeMemory(memory interface{}) error {
	// Try to optimize memory if it supports optimization
	if optimizer, ok := memory.(interface{ Optimize(interface{}) error }); ok {
		return optimizer.Optimize(nil)
	}
	return nil
}

func (mf *MemoryFactory) generateID() string {
	return fmt.Sprintf("mem_%d", mf.getCurrentTimestamp())
}

func (mf *MemoryFactory) getCurrentTime() string {
	// In a real implementation, this would use time.Now()
	return "2024-01-01T00:00:00Z"
}

func (mf *MemoryFactory) getCurrentTimestamp() int64 {
	// In a real implementation, this would use time.Now().Unix()
	return 1704067200 // 2024-01-01 00:00:00 UTC
}

func (mf *MemoryFactory) getTemplate(templateName string) (*MemoryConfig, error) {
	// Predefined templates
	templates := map[string]*MemoryConfig{
		"fast_textual": {
			Type:            MemoryTypeNaiveTextual,
			Backend:         types.MemoryBackendNaive,
			EmbedderType:    "",
			LLMType:         "",
			VectorDBType:    "",
			GraphDBType:     "",
			EnableCaching:   true,
			EnableMetrics:   false,
			CompressionMode: "none",
			OptimizeOnInit:  false,
		},
		"semantic_textual": {
			Type:            MemoryTypeGeneralTextual,
			Backend:         types.MemoryBackendGeneral,
			EmbedderType:    "default",
			LLMType:         "default",
			VectorDBType:    "default",
			GraphDBType:     "",
			EnableCaching:   true,
			EnableMetrics:   true,
			CompressionMode: "lz4",
			OptimizeOnInit:  false,
		},
		"advanced_textual": {
			Type:            MemoryTypeTreeTextual,
			Backend:         types.MemoryBackendTree,
			EmbedderType:    "default",
			LLMType:         "default",
			VectorDBType:    "default",
			GraphDBType:     "default",
			EnableCaching:   true,
			EnableMetrics:   true,
			CompressionMode: "zstd",
			OptimizeOnInit:  true,
		},
		"kv_cache": {
			Type:            MemoryTypeKVCache,
			Backend:         types.MemoryBackendKVCache,
			LLMType:         "default",
			EnableCaching:   true,
			EnableMetrics:   true,
			CompressionMode: "zstd",
			OptimizeOnInit:  true,
		},
		"lora_adapter": {
			Type:            MemoryTypeLoRA,
			Backend:         types.MemoryBackendLoRA,
			EnableCaching:   true,
			EnableMetrics:   true,
			CompressionMode: "custom",
			OptimizeOnInit:  false,
		},
	}
	
	template, exists := templates[templateName]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateName)
	}
	
	return template, nil
}

func (mf *MemoryFactory) applyParametersToTemplate(template *MemoryConfig, parameters map[string]interface{}) *MemoryConfig {
	// Copy template
	config := *template
	
	// Apply parameters
	if name, exists := parameters["name"]; exists {
		if nameStr, ok := name.(string); ok {
			config.Name = nameStr
		}
	}
	
	if embedderType, exists := parameters["embedder_type"]; exists {
		if embedderStr, ok := embedderType.(string); ok {
			config.EmbedderType = embedderStr
		}
	}
	
	if llmType, exists := parameters["llm_type"]; exists {
		if llmStr, ok := llmType.(string); ok {
			config.LLMType = llmStr
		}
	}
	
	if vectorDBType, exists := parameters["vector_db_type"]; exists {
		if vectorDBStr, ok := vectorDBType.(string); ok {
			config.VectorDBType = vectorDBStr
		}
	}
	
	if graphDBType, exists := parameters["graph_db_type"]; exists {
		if graphDBStr, ok := graphDBType.(string); ok {
			config.GraphDBType = graphDBStr
		}
	}
	
	if enableCaching, exists := parameters["enable_caching"]; exists {
		if cachingBool, ok := enableCaching.(bool); ok {
			config.EnableCaching = cachingBool
		}
	}
	
	if enableMetrics, exists := parameters["enable_metrics"]; exists {
		if metricsBool, ok := enableMetrics.(bool); ok {
			config.EnableMetrics = metricsBool
		}
	}
	
	if compressionMode, exists := parameters["compression_mode"]; exists {
		if compressionStr, ok := compressionMode.(string); ok {
			config.CompressionMode = compressionStr
		}
	}
	
	if optimizeOnInit, exists := parameters["optimize_on_init"]; exists {
		if optimizeBool, ok := optimizeOnInit.(bool); ok {
			config.OptimizeOnInit = optimizeBool
		}
	}
	
	if config.Parameters == nil {
		config.Parameters = make(map[string]interface{})
	}
	
	// Copy additional parameters
	for key, value := range parameters {
		if key != "name" && key != "embedder_type" && key != "llm_type" && 
		   key != "vector_db_type" && key != "graph_db_type" && 
		   key != "enable_caching" && key != "enable_metrics" && 
		   key != "compression_mode" && key != "optimize_on_init" {
			config.Parameters[key] = value
		}
	}
	
	return &config
}