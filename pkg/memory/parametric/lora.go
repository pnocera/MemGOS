// Package parametric provides LoRA adapter memory implementation
package parametric

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/errors"
	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// LoRAMemory implements LoRA adapter memory storage and management
type LoRAMemory struct {
	*BaseParametricMemory
	config           *config.MemoryConfig
	memories         map[string]*types.ParametricMemoryItem
	memMu            sync.RWMutex
	adapterManager   *AdapterManager
	modelRegistry    *ModelRegistry
	fineTuningEngine *FineTuningEngine
}

// AdapterManager handles adapter lifecycle and operations
type AdapterManager struct {
	activeAdapters   map[string]*LoRAAdapter
	adapterChains    map[string][]*LoRAAdapter // Model ID -> chain of adapters
	adaptorComposer  *AdapterComposer
	mu               sync.RWMutex
}

// ModelRegistry manages model information and compatibility
type ModelRegistry struct {
	models           map[string]*ModelInfo
	compatibility    map[string][]string // Model ID -> compatible adapter types
	mu               sync.RWMutex
}

// ModelInfo represents information about a model
type ModelInfo struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Architecture    string            `json:"architecture"`
	Layers          []string          `json:"layers"`
	ParameterCount  int64             `json:"parameter_count"`
	SupportedAdapters []AdapterType   `json:"supported_adapters"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// FineTuningEngine handles fine-tuning operations
type FineTuningEngine struct {
	activeJobs    map[string]*FineTuningJob
	jobQueue      []*FineTuningJob
	maxConcurrent int
	mu            sync.RWMutex
}

// AdapterComposer handles adapter composition and merging
type AdapterComposer struct {
	compositionRules map[string]interface{}
	mergingStrategy  string
}

// NewLoRAMemory creates a new LoRA memory instance
func NewLoRAMemory(cfg *config.MemoryConfig, logger interfaces.Logger, metrics interfaces.Metrics) (*LoRAMemory, error) {
	base := NewBaseParametricMemory(logger, metrics)
	
	adapterManager := &AdapterManager{
		activeAdapters:  make(map[string]*LoRAAdapter),
		adapterChains:   make(map[string][]*LoRAAdapter),
		adaptorComposer: &AdapterComposer{
			compositionRules: make(map[string]interface{}),
			mergingStrategy:  "weighted_sum",
		},
	}
	
	modelRegistry := &ModelRegistry{
		models:        make(map[string]*ModelInfo),
		compatibility: make(map[string][]string),
	}
	
	fineTuningEngine := &FineTuningEngine{
		activeJobs:    make(map[string]*FineTuningJob),
		jobQueue:      make([]*FineTuningJob, 0),
		maxConcurrent: 2,
	}
	
	lm := &LoRAMemory{
		BaseParametricMemory: base,
		config:               cfg,
		memories:             make(map[string]*types.ParametricMemoryItem),
		adapterManager:       adapterManager,
		modelRegistry:        modelRegistry,
		fineTuningEngine:     fineTuningEngine,
	}
	
	// Initialize with common model types
	lm.initializeCommonModels()
	
	return lm, nil
}

// initializeCommonModels initializes registry with common model types
func (lm *LoRAMemory) initializeCommonModels() {
	// Add common transformer models
	models := []*ModelInfo{
		{
			ID:           "gpt2",
			Name:         "GPT-2",
			Architecture: "transformer",
			Layers:       []string{"attention", "mlp"},
			ParameterCount: 117000000,
			SupportedAdapters: []AdapterType{AdapterTypeLoRA, AdapterTypeIA3, AdapterTypePrefix},
		},
		{
			ID:           "llama2",
			Name:         "LLaMA 2",
			Architecture: "transformer",
			Layers:       []string{"self_attn", "mlp"},
			ParameterCount: 7000000000,
			SupportedAdapters: []AdapterType{AdapterTypeLoRA, AdapterTypeLoHa, AdapterTypeLoKr},
		},
		{
			ID:           "bert",
			Name:         "BERT",
			Architecture: "transformer",
			Layers:       []string{"attention", "intermediate", "output"},
			ParameterCount: 110000000,
			SupportedAdapters: []AdapterType{AdapterTypeLoRA, AdapterTypeIA3},
		},
	}
	
	for _, model := range models {
		lm.modelRegistry.models[model.ID] = model
		
		// Set compatibility
		compatibility := make([]string, len(model.SupportedAdapters))
		for i, adapterType := range model.SupportedAdapters {
			compatibility[i] = string(adapterType)
		}
		lm.modelRegistry.compatibility[model.ID] = compatibility
	}
}

// Add adds parametric memory items
func (lm *LoRAMemory) Add(ctx context.Context, memories []*types.ParametricMemoryItem) error {
	if lm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	start := time.Now()
	defer func() {
		if lm.metrics != nil {
			lm.metrics.Timer("lora_memory_add_duration", float64(time.Since(start).Milliseconds()), nil)
		}
	}()
	
	lm.memMu.Lock()
	defer lm.memMu.Unlock()
	
	for _, memory := range memories {
		if memory.ID == "" {
			memory.ID = lm.generateID()
		}
		
		if memory.CreatedAt.IsZero() {
			memory.CreatedAt = time.Now()
		}
		memory.UpdatedAt = time.Now()
		
		// Store memory
		lm.memories[memory.ID] = memory
		
		// Handle adapter data
		if err := lm.processAdapterMemory(memory); err != nil {
			lm.logger.Error("Failed to process adapter memory", err, map[string]interface{}{
				"memory_id": memory.ID,
			})
		}
	}
	
	lm.logger.Info("Added LoRA memories", map[string]interface{}{
		"count": len(memories),
	})
	
	if lm.metrics != nil {
		lm.metrics.Counter("lora_memory_add_count", float64(len(memories)), nil)
	}
	
	return nil
}

// processAdapterMemory processes adapter memory items
func (lm *LoRAMemory) processAdapterMemory(memory *types.ParametricMemoryItem) error {
	// Check if this is adapter data
	if adapterType, exists := memory.Metadata["adapter_type"]; exists {
		switch adapterType {
		case string(AdapterTypeLoRA):
			return lm.processLoRAAdapter(memory)
		case string(AdapterTypeLoHa):
			return lm.processLoHaAdapter(memory)
		case string(AdapterTypeLoKr):
			return lm.processLoKrAdapter(memory)
		default:
			return lm.processGenericAdapter(memory)
		}
	}
	
	return nil
}

// processLoRAAdapter processes LoRA adapter data
func (lm *LoRAMemory) processLoRAAdapter(memory *types.ParametricMemoryItem) error {
	// Convert memory data to LoRA adapter
	adapter, err := lm.convertToLoRAAdapter(memory)
	if err != nil {
		return err
	}
	
	// Store in adapter manager
	lm.adapterManager.mu.Lock()
	lm.adapterManager.activeAdapters[adapter.ID] = adapter
	lm.adapterManager.mu.Unlock()
	
	// Update model chains if specified
	if modelID, exists := memory.Metadata["model_id"]; exists {
		if modelStr, ok := modelID.(string); ok {
			lm.addToAdapterChain(modelStr, adapter)
		}
	}
	
	return nil
}

// convertToLoRAAdapter converts memory item to LoRA adapter
func (lm *LoRAMemory) convertToLoRAAdapter(memory *types.ParametricMemoryItem) (*LoRAAdapter, error) {
	adapter := &LoRAAdapter{
		ID:           memory.ID,
		Name:         fmt.Sprintf("adapter_%s", memory.ID),
		WeightA:      make(map[string][]float32),
		WeightB:      make(map[string][]float32),
		TargetLayers: make([]string, 0),
		Metadata:     memory.Metadata,
		CreatedAt:    memory.CreatedAt,
		UpdatedAt:    memory.UpdatedAt,
		Compressed:   false,
	}
	
	// Parse adapter data from memory
	if adapterData, ok := memory.Memory.(map[string]interface{}); ok {
		// Parse configuration
		if configData, exists := adapterData["config"]; exists {
			if configMap, ok := configData.(map[string]interface{}); ok {
				adapter.Config = &AdapterConfig{
					Type:         AdapterTypeLoRA,
					Rank:         16, // Default
					Alpha:        32, // Default
					DropoutRate:  0.1,
					TargetLayers: []string{"q_proj", "v_proj"},
					InitMethod:   "normal",
					Parameters:   make(map[string]interface{}),
				}
				
				// Parse specific config values
				if rank, exists := configMap["rank"]; exists {
					if rankFloat, ok := rank.(float64); ok {
						adapter.Config.Rank = int(rankFloat)
					}
				}
				
				if alpha, exists := configMap["alpha"]; exists {
					if alphaFloat, ok := alpha.(float64); ok {
						adapter.Config.Alpha = alphaFloat
					}
				}
				
				if targetLayers, exists := configMap["target_layers"]; exists {
					if layerSlice, ok := targetLayers.([]interface{}); ok {
						adapter.Config.TargetLayers = make([]string, len(layerSlice))
						for i, layer := range layerSlice {
							adapter.Config.TargetLayers[i] = layer.(string)
						}
					}
				}
			}
		}
		
		// Parse weight data
		if weightsData, exists := adapterData["weights"]; exists {
			if weightsMap, ok := weightsData.(map[string]interface{}); ok {
				lm.parseWeights(weightsMap, adapter)
			}
		}
	}
	
	// Calculate scaling factor
	if adapter.Config != nil {
		adapter.Scaling = adapter.Config.Alpha / float64(adapter.Config.Rank)
		adapter.TargetLayers = adapter.Config.TargetLayers
	}
	
	// Calculate size
	adapter.Size = lm.calculateAdapterSize(adapter)
	
	return adapter, nil
}

// parseWeights parses weight data for adapter
func (lm *LoRAMemory) parseWeights(weightsMap map[string]interface{}, adapter *LoRAAdapter) {
	// Parse weight A
	if weightAData, exists := weightsMap["weight_a"]; exists {
		if weightAMap, ok := weightAData.(map[string]interface{}); ok {
			for layer, weights := range weightAMap {
				if weightSlice, ok := weights.([]interface{}); ok {
					adapter.WeightA[layer] = make([]float32, len(weightSlice))
					for i, weight := range weightSlice {
						if weightFloat, ok := weight.(float64); ok {
							adapter.WeightA[layer][i] = float32(weightFloat)
						}
					}
				}
			}
		}
	}
	
	// Parse weight B
	if weightBData, exists := weightsMap["weight_b"]; exists {
		if weightBMap, ok := weightBData.(map[string]interface{}); ok {
			for layer, weights := range weightBMap {
				if weightSlice, ok := weights.([]interface{}); ok {
					adapter.WeightB[layer] = make([]float32, len(weightSlice))
					for i, weight := range weightSlice {
						if weightFloat, ok := weight.(float64); ok {
							adapter.WeightB[layer][i] = float32(weightFloat)
						}
					}
				}
			}
		}
	}
}

// LoadAdapter loads an adapter from a file path
func (lm *LoRAMemory) LoadAdapter(ctx context.Context, path string) (*types.ParametricMemoryItem, error) {
	if lm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	// Read adapter file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read adapter file: %w", err)
	}
	
	// Parse adapter data
	var adapterData map[string]interface{}
	if err := json.Unmarshal(data, &adapterData); err != nil {
		return nil, fmt.Errorf("failed to parse adapter data: %w", err)
	}
	
	// Create memory item
	memory := &types.ParametricMemoryItem{
		ID:     lm.generateID(),
		Memory: adapterData,
		Metadata: map[string]interface{}{
			"adapter_type": string(AdapterTypeLoRA),
			"loaded_from":  path,
			"file_size":    len(data),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	// Process the adapter
	if err := lm.processAdapterMemory(memory); err != nil {
		return nil, fmt.Errorf("failed to process loaded adapter: %w", err)
	}
	
	// Store the memory
	lm.memMu.Lock()
	lm.memories[memory.ID] = memory
	lm.memMu.Unlock()
	
	lm.logger.Info("Loaded adapter from file", map[string]interface{}{
		"adapter_id": memory.ID,
		"path":       path,
		"size":       len(data),
	})
	
	return memory, nil
}

// SaveAdapter saves an adapter to a file path
func (lm *LoRAMemory) SaveAdapter(ctx context.Context, item *types.ParametricMemoryItem, path string) error {
	if lm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	// Get adapter data
	adapterData, ok := item.Memory.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid adapter data format")
	}
	
	// Marshal to JSON
	data, err := json.MarshalIndent(adapterData, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal adapter data: %w", err)
	}
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write adapter file: %w", err)
	}
	
	lm.logger.Info("Saved adapter to file", map[string]interface{}{
		"adapter_id": item.ID,
		"path":       path,
		"size":       len(data),
	})
	
	return nil
}

// ApplyAdapter applies an adapter to a model
func (lm *LoRAMemory) ApplyAdapter(ctx context.Context, adapterID string, modelID string) error {
	if lm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	// Check model compatibility
	if !lm.isAdapterCompatible(adapterID, modelID) {
		return fmt.Errorf("adapter %s is not compatible with model %s", adapterID, modelID)
	}
	
	// Get adapter
	lm.adapterManager.mu.RLock()
	adapter, exists := lm.adapterManager.activeAdapters[adapterID]
	lm.adapterManager.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("adapter %s not found", adapterID)
	}
	
	// Apply adapter to model (simulation)
	lm.logger.Info("Applying adapter to model", map[string]interface{}{
		"adapter_id": adapterID,
		"model_id":   modelID,
		"scaling":    adapter.Scaling,
	})
	
	// Add to adapter chain
	lm.addToAdapterChain(modelID, adapter)
	
	return nil
}

// ComposeAdapters composes multiple adapters into a single adapter
func (lm *LoRAMemory) ComposeAdapters(ctx context.Context, adapterIDs []string) (*types.ParametricMemoryItem, error) {
	if lm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	if len(adapterIDs) < 2 {
		return nil, fmt.Errorf("at least 2 adapters required for composition")
	}
	
	// Get adapters
	adapters := make([]*LoRAAdapter, 0, len(adapterIDs))
	lm.adapterManager.mu.RLock()
	for _, id := range adapterIDs {
		if adapter, exists := lm.adapterManager.activeAdapters[id]; exists {
			adapters = append(adapters, adapter)
		}
	}
	lm.adapterManager.mu.RUnlock()
	
	if len(adapters) != len(adapterIDs) {
		return nil, fmt.Errorf("some adapters not found")
	}
	
	// Compose adapters
	composedAdapter, err := lm.adapterManager.adaptorComposer.composeAdapters(adapters)
	if err != nil {
		return nil, fmt.Errorf("failed to compose adapters: %w", err)
	}
	
	// Convert to memory item
	adapterData := lm.adapterToMap(composedAdapter)
	memory := &types.ParametricMemoryItem{
		ID:     composedAdapter.ID,
		Memory: adapterData,
		Metadata: map[string]interface{}{
			"adapter_type":    string(AdapterTypeLoRA),
			"composed_from":   adapterIDs,
			"composition_strategy": lm.adapterManager.adaptorComposer.mergingStrategy,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	// Store composed adapter
	lm.memMu.Lock()
	lm.memories[memory.ID] = memory
	lm.memMu.Unlock()
	
	lm.adapterManager.mu.Lock()
	lm.adapterManager.activeAdapters[composedAdapter.ID] = composedAdapter
	lm.adapterManager.mu.Unlock()
	
	lm.logger.Info("Composed adapters", map[string]interface{}{
		"composed_id":    composedAdapter.ID,
		"source_adapters": adapterIDs,
		"strategy":       lm.adapterManager.adaptorComposer.mergingStrategy,
	})
	
	return memory, nil
}

// SaveModelParameters saves model parameters
func (lm *LoRAMemory) SaveModelParameters(ctx context.Context, modelID string, parameters interface{}) error {
	if lm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	modelParams := lm.CreateModelParameters(modelID, parameters)
	
	lm.modelParameters[modelParams.ID] = modelParams
	
	lm.logger.Info("Saved model parameters", map[string]interface{}{
		"model_id":    modelID,
		"params_id":   modelParams.ID,
		"size":        modelParams.Size,
		"checksum":    modelParams.Checksum,
	})
	
	return nil
}

// LoadModelParameters loads model parameters
func (lm *LoRAMemory) LoadModelParameters(ctx context.Context, modelID string) (interface{}, error) {
	if lm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	// Find parameters for model
	for _, params := range lm.modelParameters {
		if params.ModelID == modelID {
			// Validate checksum
			if !lm.parameterManager.ValidateChecksum(params) {
				return nil, fmt.Errorf("checksum validation failed for model %s", modelID)
			}
			
			return params.Parameters, nil
		}
	}
	
	return nil, fmt.Errorf("parameters not found for model %s", modelID)
}

// StartFineTuning starts a fine-tuning job
func (lm *LoRAMemory) StartFineTuning(ctx context.Context, baseModelID string, config interface{}) (string, error) {
	if lm.IsClosed() {
		return "", errors.NewMemoryError("memory is closed")
	}
	
	// Check if model exists
	if _, exists := lm.modelRegistry.models[baseModelID]; !exists {
		return "", fmt.Errorf("base model %s not found", baseModelID)
	}
	
	// Create fine-tuning job
	configMap, ok := config.(map[string]interface{})
	if !ok {
		configMap = map[string]interface{}{"config": config}
	}
	
	job := lm.CreateFineTuningJob(baseModelID, configMap)
	
	// Store job
	lm.fineTuningEngine.mu.Lock()
	lm.fineTuningEngine.jobQueue = append(lm.fineTuningEngine.jobQueue, job)
	lm.fineTuningEngine.mu.Unlock()
	
	// Start job processing
	go lm.processFineTuningJob(job)
	
	lm.logger.Info("Started fine-tuning job", map[string]interface{}{
		"job_id":        job.ID,
		"base_model_id": baseModelID,
	})
	
	return job.ID, nil
}

// MonitorFineTuning monitors a fine-tuning job
func (lm *LoRAMemory) MonitorFineTuning(ctx context.Context, jobID string) (interface{}, error) {
	if lm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	lm.fineTuningEngine.mu.RLock()
	job, exists := lm.fineTuningEngine.activeJobs[jobID]
	lm.fineTuningEngine.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("fine-tuning job %s not found", jobID)
	}
	
	return map[string]interface{}{
		"id":           job.ID,
		"status":       job.Status,
		"progress":     job.Progress,
		"metrics":      job.Metrics,
		"created_at":   job.CreatedAt,
		"started_at":   job.StartedAt,
		"completed_at": job.CompletedAt,
		"error":        job.ErrorMessage,
	}, nil
}

// StopFineTuning stops a fine-tuning job
func (lm *LoRAMemory) StopFineTuning(ctx context.Context, jobID string) error {
	if lm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	lm.fineTuningEngine.mu.Lock()
	defer lm.fineTuningEngine.mu.Unlock()
	
	if job, exists := lm.fineTuningEngine.activeJobs[jobID]; exists {
		job.Status = "stopped"
		now := time.Now()
		job.CompletedAt = &now
		
		lm.logger.Info("Stopped fine-tuning job", map[string]interface{}{
			"job_id": jobID,
		})
		
		return nil
	}
	
	return fmt.Errorf("fine-tuning job %s not found", jobID)
}

// CreateCheckpoint creates a model checkpoint
func (lm *LoRAMemory) CreateCheckpoint(ctx context.Context, modelID string, checkpointName string) error {
	if lm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	// Get current model parameters
	params, err := lm.LoadModelParameters(ctx, modelID)
	if err != nil {
		return fmt.Errorf("failed to load model parameters: %w", err)
	}
	
	// Create checkpoint
	checkpoint := lm.CreateModelParameters(modelID+"_"+checkpointName, params)
	checkpoint.Metadata["checkpoint_name"] = checkpointName
	checkpoint.Metadata["checkpoint_created"] = time.Now().Format(time.RFC3339)
	
	lm.modelParameters[checkpoint.ID] = checkpoint
	
	lm.logger.Info("Created model checkpoint", map[string]interface{}{
		"model_id":        modelID,
		"checkpoint_name": checkpointName,
		"checkpoint_id":   checkpoint.ID,
	})
	
	return nil
}

// RestoreCheckpoint restores a model from checkpoint
func (lm *LoRAMemory) RestoreCheckpoint(ctx context.Context, modelID string, checkpointName string) error {
	if lm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	// Find checkpoint
	checkpointID := modelID + "_" + checkpointName
	var checkpoint *ModelParameters
	
	for _, params := range lm.modelParameters {
		if params.ModelID == checkpointID {
			checkpoint = params
			break
		}
	}
	
	if checkpoint == nil {
		return fmt.Errorf("checkpoint %s not found for model %s", checkpointName, modelID)
	}
	
	// Restore parameters
	if err := lm.SaveModelParameters(ctx, modelID, checkpoint.Parameters); err != nil {
		return fmt.Errorf("failed to restore checkpoint: %w", err)
	}
	
	lm.logger.Info("Restored model checkpoint", map[string]interface{}{
		"model_id":        modelID,
		"checkpoint_name": checkpointName,
	})
	
	return nil
}

// CreateVersion creates a new version of an adapter
func (lm *LoRAMemory) CreateVersion(ctx context.Context, adapterID string, version string) error {
	if lm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	// Check if adapter exists
	if _, exists := lm.adapterManager.activeAdapters[adapterID]; !exists {
		return fmt.Errorf("adapter %s not found", adapterID)
	}
	
	// Create version
	adapterVersion := lm.CreateAdapterVersion(adapterID, version, "Version "+version)
	
	// Add to versions list
	if versions, exists := lm.adapterVersions[adapterID]; exists {
		lm.adapterVersions[adapterID] = append(versions, adapterVersion)
	} else {
		lm.adapterVersions[adapterID] = []*AdapterVersion{adapterVersion}
	}
	
	// Prune old versions
	lm.adapterVersions[adapterID] = lm.versionManager.PruneVersions(lm.adapterVersions[adapterID])
	
	lm.logger.Info("Created adapter version", map[string]interface{}{
		"adapter_id": adapterID,
		"version":    version,
	})
	
	return nil
}

// ListVersions lists all versions of an adapter
func (lm *LoRAMemory) ListVersions(ctx context.Context, adapterID string) ([]string, error) {
	if lm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	versions, exists := lm.adapterVersions[adapterID]
	if !exists {
		return []string{}, nil
	}
	
	versionList := make([]string, len(versions))
	for i, version := range versions {
		versionList[i] = version.Version
	}
	
	return versionList, nil
}

// GetVersion gets a specific version of an adapter
func (lm *LoRAMemory) GetVersion(ctx context.Context, adapterID string, version string) (*types.ParametricMemoryItem, error) {
	if lm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	versions, exists := lm.adapterVersions[adapterID]
	if !exists {
		return nil, fmt.Errorf("no versions found for adapter %s", adapterID)
	}
	
	for _, ver := range versions {
		if ver.Version == version {
			// Get the adapter at this version (simplified - would need actual version storage)
			if adapter, exists := lm.adapterManager.activeAdapters[adapterID]; exists {
				adapterData := lm.adapterToMap(adapter)
				memory := &types.ParametricMemoryItem{
					ID:     adapter.ID + "_v" + version,
					Memory: adapterData,
					Metadata: map[string]interface{}{
						"adapter_type": string(AdapterTypeLoRA),
						"version":      version,
						"original_id":  adapterID,
					},
					CreatedAt: ver.CreatedAt,
					UpdatedAt: ver.CreatedAt,
				}
				return memory, nil
			}
		}
	}
	
	return nil, fmt.Errorf("version %s not found for adapter %s", version, adapterID)
}

// Get retrieves a memory by ID
func (lm *LoRAMemory) Get(ctx context.Context, id string) (*types.ParametricMemoryItem, error) {
	if lm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	lm.memMu.RLock()
	defer lm.memMu.RUnlock()
	
	memory, exists := lm.memories[id]
	if !exists {
		return nil, errors.NewMemoryNotFoundError(id)
	}
	
	return memory, nil
}

// GetByIDs retrieves multiple memories by IDs
func (lm *LoRAMemory) GetByIDs(ctx context.Context, ids []string) ([]*types.ParametricMemoryItem, error) {
	if lm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	lm.memMu.RLock()
	defer lm.memMu.RUnlock()
	
	result := make([]*types.ParametricMemoryItem, 0, len(ids))
	for _, id := range ids {
		if memory, exists := lm.memories[id]; exists {
			result = append(result, memory)
		}
	}
	
	return result, nil
}

// GetAll retrieves all memories
func (lm *LoRAMemory) GetAll(ctx context.Context) ([]*types.ParametricMemoryItem, error) {
	if lm.IsClosed() {
		return nil, errors.NewMemoryError("memory is closed")
	}
	
	lm.memMu.RLock()
	defer lm.memMu.RUnlock()
	
	result := make([]*types.ParametricMemoryItem, 0, len(lm.memories))
	for _, memory := range lm.memories {
		result = append(result, memory)
	}
	
	return result, nil
}

// Update updates an existing memory
func (lm *LoRAMemory) Update(ctx context.Context, id string, memory *types.ParametricMemoryItem) error {
	if lm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	lm.memMu.Lock()
	defer lm.memMu.Unlock()
	
	if _, exists := lm.memories[id]; !exists {
		return errors.NewMemoryNotFoundError(id)
	}
	
	memory.ID = id
	memory.UpdatedAt = time.Now()
	lm.memories[id] = memory
	
	// Reprocess adapter if needed
	if err := lm.processAdapterMemory(memory); err != nil {
		lm.logger.Error("Failed to reprocess adapter memory", err, map[string]interface{}{
			"memory_id": id,
		})
	}
	
	lm.logger.Info("Updated LoRA memory", map[string]interface{}{
		"memory_id": id,
	})
	
	return nil
}

// Delete deletes memories by IDs
func (lm *LoRAMemory) Delete(ctx context.Context, ids []string) error {
	if lm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	lm.memMu.Lock()
	defer lm.memMu.Unlock()
	
	deletedCount := 0
	for _, id := range ids {
		if _, exists := lm.memories[id]; exists {
			delete(lm.memories, id)
			
			// Remove from adapter manager
			lm.adapterManager.mu.Lock()
			delete(lm.adapterManager.activeAdapters, id)
			lm.adapterManager.mu.Unlock()
			
			// Remove from versions
			delete(lm.adapterVersions, id)
			
			deletedCount++
		}
	}
	
	lm.logger.Info("Deleted LoRA memories", map[string]interface{}{
		"count": deletedCount,
	})
	
	return nil
}

// DeleteAll deletes all memories
func (lm *LoRAMemory) DeleteAll(ctx context.Context) error {
	if lm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	lm.memMu.Lock()
	defer lm.memMu.Unlock()
	
	count := len(lm.memories)
	lm.memories = make(map[string]*types.ParametricMemoryItem)
	
	// Clear all manager data
	lm.adapterManager.mu.Lock()
	lm.adapterManager.activeAdapters = make(map[string]*LoRAAdapter)
	lm.adapterManager.adapterChains = make(map[string][]*LoRAAdapter)
	lm.adapterManager.mu.Unlock()
	
	lm.modelParameters = make(map[string]*ModelParameters)
	lm.adapterVersions = make(map[string][]*AdapterVersion)
	
	lm.fineTuningEngine.mu.Lock()
	lm.fineTuningEngine.activeJobs = make(map[string]*FineTuningJob)
	lm.fineTuningEngine.jobQueue = make([]*FineTuningJob, 0)
	lm.fineTuningEngine.mu.Unlock()
	
	lm.logger.Info("Deleted all LoRA memories", map[string]interface{}{
		"count": count,
	})
	
	return nil
}

// Compress compresses memory data
func (lm *LoRAMemory) Compress(ctx context.Context, compressionRatio float64) error {
	if lm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	compressedCount := 0
	
	// Compress adapters
	lm.adapterManager.mu.Lock()
	for _, adapter := range lm.adapterManager.activeAdapters {
		if err := lm.parameterManager.CompressAdapter(adapter); err != nil {
			lm.logger.Error("Failed to compress adapter", err, map[string]interface{}{
				"adapter_id": adapter.ID,
			})
		} else {
			compressedCount++
		}
	}
	lm.adapterManager.mu.Unlock()
	
	lm.logger.Info("Compressed LoRA memory", map[string]interface{}{
		"compressed_count":   compressedCount,
		"compression_ratio":  compressionRatio,
	})
	
	return nil
}

// Optimize optimizes memory performance
func (lm *LoRAMemory) Optimize(ctx context.Context) error {
	if lm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	// Optimize adapters
	lm.adapterManager.mu.Lock()
	for _, adapter := range lm.adapterManager.activeAdapters {
		if err := lm.optimizationManager.OptimizeAdapter(adapter); err != nil {
			lm.logger.Error("Failed to optimize adapter", err, map[string]interface{}{
				"adapter_id": adapter.ID,
			})
		}
	}
	lm.adapterManager.mu.Unlock()
	
	// Clean up old versions
	for adapterID, versions := range lm.adapterVersions {
		lm.adapterVersions[adapterID] = lm.versionManager.PruneVersions(versions)
	}
	
	// Clean up completed fine-tuning jobs
	lm.cleanupFineTuningJobs()
	
	lm.logger.Info("Optimized LoRA memory")
	
	return nil
}

// Load loads memories from directory
func (lm *LoRAMemory) Load(ctx context.Context, dir string) error {
	if lm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	filePath := filepath.Join(dir, lm.config.MemoryFilename)
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		lm.logger.Info("LoRA memory file not found, starting with empty memories", map[string]interface{}{
			"file_path": filePath,
		})
		return nil
	}
	
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read LoRA memory file: %w", err)
	}
	
	// Parse JSON
	var saveData map[string]interface{}
	if err := json.Unmarshal(data, &saveData); err != nil {
		return fmt.Errorf("failed to parse LoRA memory file: %w", err)
	}
	
	// Load memories
	if err := lm.loadFromSaveData(saveData); err != nil {
		return fmt.Errorf("failed to load LoRA memories: %w", err)
	}
	
	lm.logger.Info("Loaded LoRA memories", map[string]interface{}{
		"count":     len(lm.memories),
		"file_path": filePath,
	})
	
	return nil
}

// Dump saves memories to directory
func (lm *LoRAMemory) Dump(ctx context.Context, dir string) error {
	if lm.IsClosed() {
		return errors.NewMemoryError("memory is closed")
	}
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Prepare save data
	saveData := lm.prepareForSave()
	
	// Marshal to JSON
	data, err := json.MarshalIndent(saveData, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal LoRA data: %w", err)
	}
	
	// Write to file
	filePath := filepath.Join(dir, lm.config.MemoryFilename)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write LoRA memory file: %w", err)
	}
	
	lm.logger.Info("Dumped LoRA memories", map[string]interface{}{
		"count":     len(lm.memories),
		"file_path": filePath,
	})
	
	return nil
}

// Helper methods

func (lm *LoRAMemory) processLoHaAdapter(memory *types.ParametricMemoryItem) error {
	// Process LoHa (Low-rank Hadamard) adapter
	// This is a placeholder for LoHa-specific processing
	return lm.processGenericAdapter(memory)
}

func (lm *LoRAMemory) processLoKrAdapter(memory *types.ParametricMemoryItem) error {
	// Process LoKr (Low-rank Kronecker) adapter
	// This is a placeholder for LoKr-specific processing
	return lm.processGenericAdapter(memory)
}

func (lm *LoRAMemory) processGenericAdapter(memory *types.ParametricMemoryItem) error {
	// Generic adapter processing
	lm.logger.Info("Processing generic adapter", map[string]interface{}{
		"memory_id": memory.ID,
		"type":      memory.Metadata["adapter_type"],
	})
	return nil
}

func (lm *LoRAMemory) isAdapterCompatible(adapterID, modelID string) bool {
	// Check if adapter is compatible with model
	if compatibility, exists := lm.modelRegistry.compatibility[modelID]; exists {
		for _, adapterType := range compatibility {
			if adapterType == string(AdapterTypeLoRA) {
				return true
			}
		}
	}
	return false
}

func (lm *LoRAMemory) addToAdapterChain(modelID string, adapter *LoRAAdapter) {
	lm.adapterManager.mu.Lock()
	defer lm.adapterManager.mu.Unlock()
	
	if chain, exists := lm.adapterManager.adapterChains[modelID]; exists {
		lm.adapterManager.adapterChains[modelID] = append(chain, adapter)
	} else {
		lm.adapterManager.adapterChains[modelID] = []*LoRAAdapter{adapter}
	}
}

func (lm *LoRAMemory) adapterToMap(adapter *LoRAAdapter) map[string]interface{} {
	return map[string]interface{}{
		"id":            adapter.ID,
		"name":          adapter.Name,
		"config":        adapter.Config,
		"weight_a":      adapter.WeightA,
		"weight_b":      adapter.WeightB,
		"scaling":       adapter.Scaling,
		"target_layers": adapter.TargetLayers,
		"metadata":      adapter.Metadata,
		"created_at":    adapter.CreatedAt.Format(time.RFC3339),
		"updated_at":    adapter.UpdatedAt.Format(time.RFC3339),
		"size":          adapter.Size,
		"compressed":    adapter.Compressed,
	}
}

func (lm *LoRAMemory) processFineTuningJob(job *FineTuningJob) {
	// Move job to active jobs
	lm.fineTuningEngine.mu.Lock()
	lm.fineTuningEngine.activeJobs[job.ID] = job
	lm.fineTuningEngine.mu.Unlock()
	
	// Start job
	job.Status = "running"
	now := time.Now()
	job.StartedAt = &now
	
	// Simulate training progress
	for progress := 0.0; progress <= 1.0; progress += 0.1 {
		if job.Status == "stopped" {
			break
		}
		
		job.Progress = progress
		job.Metrics["current_loss"] = 1.0 - progress + 0.1*math.Sin(progress*10)
		
		time.Sleep(time.Second) // Simulate training time
	}
	
	// Complete job
	if job.Status != "stopped" {
		job.Status = "completed"
		job.Progress = 1.0
		completedAt := time.Now()
		job.CompletedAt = &completedAt
	}
	
	lm.logger.Info("Fine-tuning job completed", map[string]interface{}{
		"job_id":   job.ID,
		"status":   job.Status,
		"progress": job.Progress,
	})
}

func (lm *LoRAMemory) cleanupFineTuningJobs() {
	lm.fineTuningEngine.mu.Lock()
	defer lm.fineTuningEngine.mu.Unlock()
	
	// Remove completed jobs older than 24 hours
	cutoff := time.Now().Add(-24 * time.Hour)
	
	for jobID, job := range lm.fineTuningEngine.activeJobs {
		if job.Status == "completed" && job.CompletedAt != nil && job.CompletedAt.Before(cutoff) {
			delete(lm.fineTuningEngine.activeJobs, jobID)
		}
	}
}

func (lm *LoRAMemory) loadFromSaveData(saveData map[string]interface{}) error {
	lm.memMu.Lock()
	defer lm.memMu.Unlock()
	
	// Load memories
	if memoriesData, exists := saveData["memories"]; exists {
		if memoriesMap, ok := memoriesData.(map[string]interface{}); ok {
			for id, memoryData := range memoriesMap {
				if memoryMap, ok := memoryData.(map[string]interface{}); ok {
					memory := &types.ParametricMemoryItem{
						ID: id,
					}
					
					// Parse memory data
					if memoryContent, exists := memoryMap["memory"]; exists {
						memory.Memory = memoryContent
					}
					if metadata, exists := memoryMap["metadata"]; exists {
						memory.Metadata = metadata.(map[string]interface{})
					}
					
					// Parse timestamps
					if createdAt, exists := memoryMap["created_at"]; exists {
						if timestamp, err := time.Parse(time.RFC3339, createdAt.(string)); err == nil {
							memory.CreatedAt = timestamp
						}
					}
					if updatedAt, exists := memoryMap["updated_at"]; exists {
						if timestamp, err := time.Parse(time.RFC3339, updatedAt.(string)); err == nil {
							memory.UpdatedAt = timestamp
						}
					}
					
					lm.memories[id] = memory
					
					// Process adapter memory
					if err := lm.processAdapterMemory(memory); err != nil {
						lm.logger.Error("Failed to process loaded adapter", err, map[string]interface{}{
							"memory_id": id,
						})
					}
				}
			}
		}
	}
	
	return nil
}

func (lm *LoRAMemory) prepareForSave() map[string]interface{} {
	lm.memMu.RLock()
	defer lm.memMu.RUnlock()
	
	// Prepare memories for saving
	memoriesData := make(map[string]interface{})
	for id, memory := range lm.memories {
		memoryData := map[string]interface{}{
			"memory":     memory.Memory,
			"metadata":   memory.Metadata,
			"created_at": memory.CreatedAt.Format(time.RFC3339),
			"updated_at": memory.UpdatedAt.Format(time.RFC3339),
		}
		memoriesData[id] = memoryData
	}
	
	// Prepare adapters
	adaptersData := make(map[string]interface{})
	lm.adapterManager.mu.RLock()
	for id, adapter := range lm.adapterManager.activeAdapters {
		adaptersData[id] = lm.adapterToMap(adapter)
	}
	lm.adapterManager.mu.RUnlock()
	
	return map[string]interface{}{
		"memories":         memoriesData,
		"adapters":         adaptersData,
		"model_parameters": lm.modelParameters,
		"adapter_versions": lm.adapterVersions,
		"saved_at":         time.Now().Format(time.RFC3339),
	}
}

// composeAdapters composes multiple adapters into one
func (ac *AdapterComposer) composeAdapters(adapters []*LoRAAdapter) (*LoRAAdapter, error) {
	if len(adapters) == 0 {
		return nil, fmt.Errorf("no adapters to compose")
	}
	
	if len(adapters) == 1 {
		return adapters[0], nil
	}
	
	// Create composed adapter
	composed := &LoRAAdapter{
		ID:           "composed_" + time.Now().Format("20060102150405"),
		Name:         "Composed Adapter",
		WeightA:      make(map[string][]float32),
		WeightB:      make(map[string][]float32),
		TargetLayers: adapters[0].TargetLayers,
		Metadata:     make(map[string]interface{}),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Compressed:   false,
	}
	
	// Merge weights based on strategy
	switch ac.mergingStrategy {
	case "weighted_sum":
		return ac.weightedSumComposition(adapters, composed)
	case "concatenation":
		return ac.concatenationComposition(adapters, composed)
	default:
		return ac.weightedSumComposition(adapters, composed)
	}
}

func (ac *AdapterComposer) weightedSumComposition(adapters []*LoRAAdapter, composed *LoRAAdapter) (*LoRAAdapter, error) {
	weight := 1.0 / float32(len(adapters))
	
	// Initialize composed weights
	for _, layer := range composed.TargetLayers {
		if len(adapters[0].WeightA[layer]) > 0 {
			composed.WeightA[layer] = make([]float32, len(adapters[0].WeightA[layer]))
			composed.WeightB[layer] = make([]float32, len(adapters[0].WeightB[layer]))
		}
	}
	
	// Sum weighted contributions
	for _, adapter := range adapters {
		for layer, weights := range adapter.WeightA {
			for i, w := range weights {
				composed.WeightA[layer][i] += w * weight
			}
		}
		
		for layer, weights := range adapter.WeightB {
			for i, w := range weights {
				composed.WeightB[layer][i] += w * weight
			}
		}
	}
	
	// Set average scaling
	totalScaling := 0.0
	for _, adapter := range adapters {
		totalScaling += adapter.Scaling
	}
	composed.Scaling = totalScaling / float64(len(adapters))
	
	return composed, nil
}

func (ac *AdapterComposer) concatenationComposition(adapters []*LoRAAdapter, composed *LoRAAdapter) (*LoRAAdapter, error) {
	// Concatenate adapter weights
	for _, layer := range composed.TargetLayers {
		totalSizeA := 0
		totalSizeB := 0
		
		// Calculate total size
		for _, adapter := range adapters {
			totalSizeA += len(adapter.WeightA[layer])
			totalSizeB += len(adapter.WeightB[layer])
		}
		
		// Allocate concatenated arrays
		composed.WeightA[layer] = make([]float32, totalSizeA)
		composed.WeightB[layer] = make([]float32, totalSizeB)
		
		// Copy weights
		offsetA := 0
		offsetB := 0
		for _, adapter := range adapters {
			copy(composed.WeightA[layer][offsetA:], adapter.WeightA[layer])
			copy(composed.WeightB[layer][offsetB:], adapter.WeightB[layer])
			offsetA += len(adapter.WeightA[layer])
			offsetB += len(adapter.WeightB[layer])
		}
	}
	
	// Use maximum scaling
	maxScaling := 0.0
	for _, adapter := range adapters {
		if adapter.Scaling > maxScaling {
			maxScaling = adapter.Scaling
		}
	}
	composed.Scaling = maxScaling
	
	return composed, nil
}