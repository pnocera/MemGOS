// Package memory provides memory cube implementations for MemGOS
package memory

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/errors"
	"github.com/memtensor/memgos/pkg/interfaces"
)

// GeneralMemCube implements a general memory cube containing all memory types
type GeneralMemCube struct {
	id              string
	name            string
	config          *config.MemCubeConfig
	textualMemory   interfaces.TextualMemory
	activationMemory interfaces.ActivationMemory
	parametricMemory interfaces.ParametricMemory
	logger          interfaces.Logger
	metrics         interfaces.Metrics
	mu              sync.RWMutex
	closed          bool
}

// NewGeneralMemCube creates a new general memory cube
func NewGeneralMemCube(id, name string, cfg *config.MemCubeConfig, logger interfaces.Logger, metrics interfaces.Metrics) (*GeneralMemCube, error) {
	cube := &GeneralMemCube{
		id:      id,
		name:    name,
		config:  cfg,
		logger:  logger,
		metrics: metrics,
	}
	
	// Initialize memory components based on configuration
	if err := cube.initializeMemories(); err != nil {
		return nil, fmt.Errorf("failed to initialize memories: %w", err)
	}
	
	return cube, nil
}

// initializeMemories initializes the memory components
func (gc *GeneralMemCube) initializeMemories() error {
	var err error
	
	// Initialize textual memory
	if gc.config.TextMem != nil {
		gc.textualMemory, err = NewTextualMemory(gc.config.TextMem, gc.logger, gc.metrics)
		if err != nil {
			return fmt.Errorf("failed to create textual memory: %w", err)
		}
	}
	
	// Initialize activation memory
	if gc.config.ActMem != nil {
		gc.activationMemory, err = NewActivationMemory(gc.config.ActMem, gc.logger, gc.metrics)
		if err != nil {
			return fmt.Errorf("failed to create activation memory: %w", err)
		}
	}
	
	// Initialize parametric memory
	if gc.config.ParaMem != nil {
		gc.parametricMemory, err = NewParametricMemory(gc.config.ParaMem, gc.logger, gc.metrics)
		if err != nil {
			return fmt.Errorf("failed to create parametric memory: %w", err)
		}
	}
	
	return nil
}

// GetID returns the cube ID
func (gc *GeneralMemCube) GetID() string {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	return gc.id
}

// GetName returns the cube name
func (gc *GeneralMemCube) GetName() string {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	return gc.name
}

// GetTextualMemory returns the textual memory interface
func (gc *GeneralMemCube) GetTextualMemory() interfaces.TextualMemory {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	return gc.textualMemory
}

// GetActivationMemory returns the activation memory interface
func (gc *GeneralMemCube) GetActivationMemory() interfaces.ActivationMemory {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	return gc.activationMemory
}

// GetParametricMemory returns the parametric memory interface
func (gc *GeneralMemCube) GetParametricMemory() interfaces.ParametricMemory {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	return gc.parametricMemory
}

// Load loads the cube from a directory
func (gc *GeneralMemCube) Load(ctx context.Context, dir string) error {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	
	if gc.closed {
		return errors.NewMemoryError("memory cube is closed")
	}
	
	// Load textual memory
	if gc.textualMemory != nil {
		if err := gc.textualMemory.Load(ctx, dir); err != nil {
			return fmt.Errorf("failed to load textual memory: %w", err)
		}
	}
	
	// Load activation memory
	if gc.activationMemory != nil {
		if err := gc.activationMemory.Load(ctx, dir); err != nil {
			return fmt.Errorf("failed to load activation memory: %w", err)
		}
	}
	
	// Load parametric memory
	if gc.parametricMemory != nil {
		if err := gc.parametricMemory.Load(ctx, dir); err != nil {
			return fmt.Errorf("failed to load parametric memory: %w", err)
		}
	}
	
	gc.logger.Info("Loaded memory cube", map[string]interface{}{
		"cube_id": gc.id,
		"cube_name": gc.name,
		"directory": dir,
	})
	
	if gc.metrics != nil {
		gc.metrics.Counter("memory_cube_load_count", 1, map[string]string{
			"cube_id": gc.id,
		})
	}
	
	return nil
}

// Dump saves the cube to a directory
func (gc *GeneralMemCube) Dump(ctx context.Context, dir string) error {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	
	if gc.closed {
		return errors.NewMemoryError("memory cube is closed")
	}
	
	// Dump textual memory
	if gc.textualMemory != nil {
		if err := gc.textualMemory.Dump(ctx, dir); err != nil {
			return fmt.Errorf("failed to dump textual memory: %w", err)
		}
	}
	
	// Dump activation memory
	if gc.activationMemory != nil {
		if err := gc.activationMemory.Dump(ctx, dir); err != nil {
			return fmt.Errorf("failed to dump activation memory: %w", err)
		}
	}
	
	// Dump parametric memory
	if gc.parametricMemory != nil {
		if err := gc.parametricMemory.Dump(ctx, dir); err != nil {
			return fmt.Errorf("failed to dump parametric memory: %w", err)
		}
	}
	
	// Save cube configuration
	configPath := filepath.Join(dir, "config.json")
	if err := gc.config.ToJSONFile(configPath); err != nil {
		return fmt.Errorf("failed to save cube configuration: %w", err)
	}
	
	gc.logger.Info("Dumped memory cube", map[string]interface{}{
		"cube_id": gc.id,
		"cube_name": gc.name,
		"directory": dir,
	})
	
	if gc.metrics != nil {
		gc.metrics.Counter("memory_cube_dump_count", 1, map[string]string{
			"cube_id": gc.id,
		})
	}
	
	return nil
}

// Close closes the cube and all its memories
func (gc *GeneralMemCube) Close() error {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	
	if gc.closed {
		return nil
	}
	
	var errs []error
	
	// Close textual memory
	if gc.textualMemory != nil {
		if err := gc.textualMemory.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close textual memory: %w", err))
		}
	}
	
	// Close activation memory
	if gc.activationMemory != nil {
		if err := gc.activationMemory.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close activation memory: %w", err))
		}
	}
	
	// Close parametric memory
	if gc.parametricMemory != nil {
		if err := gc.parametricMemory.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close parametric memory: %w", err))
		}
	}
	
	gc.closed = true
	
	gc.logger.Info("Closed memory cube", map[string]interface{}{
		"cube_id": gc.id,
		"cube_name": gc.name,
	})
	
	if len(errs) > 0 {
		return fmt.Errorf("errors occurred while closing memory cube: %v", errs)
	}
	
	return nil
}

// Factory functions for creating memory cubes

// MemCubeFactory provides factory methods for creating memory cubes
type MemCubeFactory struct {
	logger  interfaces.Logger
	metrics interfaces.Metrics
}

// NewMemCubeFactory creates a new memory cube factory
func NewMemCubeFactory(logger interfaces.Logger, metrics interfaces.Metrics) *MemCubeFactory {
	return &MemCubeFactory{
		logger:  logger,
		metrics: metrics,
	}
}

// CreateFromConfig creates a memory cube from configuration
func (f *MemCubeFactory) CreateFromConfig(id, name string, cfg *config.MemCubeConfig) (interfaces.MemCube, error) {
	return NewGeneralMemCube(id, name, cfg, f.logger, f.metrics)
}

// CreateFromDirectory creates a memory cube from a directory
func (f *MemCubeFactory) CreateFromDirectory(ctx context.Context, id, name, dir string) (interfaces.MemCube, error) {
	// Load configuration from directory
	configPath := filepath.Join(dir, "config.json")
	cfg := config.NewMemCubeConfig()
	
	if err := cfg.FromJSONFile(configPath); err != nil {
		// If config doesn't exist, use default configuration
		f.logger.Info("Configuration file not found, using default configuration", map[string]interface{}{
			"config_path": configPath,
		})
	}
	
	// Create cube
	cube, err := f.CreateFromConfig(id, name, cfg)
	if err != nil {
		return nil, err
	}
	
	// Load memories from directory
	if err := cube.Load(ctx, dir); err != nil {
		cube.Close()
		return nil, err
	}
	
	return cube, nil
}

// CreateFromRemoteRepository creates a memory cube from a remote repository
func (f *MemCubeFactory) CreateFromRemoteRepository(ctx context.Context, id, name, repoURL string) (interfaces.MemCube, error) {
	// For now, this is a placeholder for remote repository loading
	// In a full implementation, this would:
	// 1. Download/clone the repository to a temporary directory
	// 2. Load the configuration and memories from that directory
	// 3. Create and return the cube
	
	f.logger.Info("Remote repository loading requested", map[string]interface{}{
		"repo_url": repoURL,
		"cube_id":  id,
	})
	
	// For now, create an empty cube with default configuration
	return f.CreateEmpty(id, name)
}

// CreateEmpty creates an empty memory cube with default configuration
func (f *MemCubeFactory) CreateEmpty(id, name string) (interfaces.MemCube, error) {
	cfg := config.NewMemCubeConfig()
	return f.CreateFromConfig(id, name, cfg)
}

// CubeRegistry manages registered memory cubes
type CubeRegistry struct {
	cubes   map[string]interfaces.MemCube
	factory *MemCubeFactory
	mu      sync.RWMutex
	logger  interfaces.Logger
}

// NewCubeRegistry creates a new cube registry
func NewCubeRegistry(logger interfaces.Logger, metrics interfaces.Metrics) *CubeRegistry {
	return &CubeRegistry{
		cubes:   make(map[string]interfaces.MemCube),
		factory: NewMemCubeFactory(logger, metrics),
		logger:  logger,
	}
}

// Register registers a memory cube
func (cr *CubeRegistry) Register(cube interfaces.MemCube) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	
	id := cube.GetID()
	if _, exists := cr.cubes[id]; exists {
		return errors.NewAlreadyExistsError(fmt.Sprintf("memory cube with ID %s", id))
	}
	
	cr.cubes[id] = cube
	
	cr.logger.Info("Registered memory cube", map[string]interface{}{
		"cube_id": id,
		"cube_name": cube.GetName(),
	})
	
	return nil
}

// Unregister unregisters a memory cube
func (cr *CubeRegistry) Unregister(id string) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	
	cube, exists := cr.cubes[id]
	if !exists {
		return errors.NewNotFoundError(fmt.Sprintf("memory cube with ID %s", id))
	}
	
	// Close the cube before unregistering
	if err := cube.Close(); err != nil {
		cr.logger.Warn("Error closing memory cube during unregistration", map[string]interface{}{
			"cube_id": id,
			"error":   err.Error(),
		})
	}
	
	delete(cr.cubes, id)
	
	cr.logger.Info("Unregistered memory cube", map[string]interface{}{
		"cube_id": id,
	})
	
	return nil
}

// Get retrieves a registered memory cube
func (cr *CubeRegistry) Get(id string) (interfaces.MemCube, error) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	
	cube, exists := cr.cubes[id]
	if !exists {
		return nil, errors.NewNotFoundError(fmt.Sprintf("memory cube with ID %s", id))
	}
	
	return cube, nil
}

// List returns all registered memory cubes
func (cr *CubeRegistry) List() []interfaces.MemCube {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	
	cubes := make([]interfaces.MemCube, 0, len(cr.cubes))
	for _, cube := range cr.cubes {
		cubes = append(cubes, cube)
	}
	
	return cubes
}

// LoadFromDirectory loads a memory cube from a directory and registers it
func (cr *CubeRegistry) LoadFromDirectory(ctx context.Context, id, name, dir string) error {
	cube, err := cr.factory.CreateFromDirectory(ctx, id, name, dir)
	if err != nil {
		return err
	}
	
	return cr.Register(cube)
}

// LoadFromRemoteRepository loads a memory cube from a remote repository and registers it
func (cr *CubeRegistry) LoadFromRemoteRepository(ctx context.Context, id, name, repoURL string) error {
	cube, err := cr.factory.CreateFromRemoteRepository(ctx, id, name, repoURL)
	if err != nil {
		return fmt.Errorf("failed to create cube from remote repository: %w", err)
	}
	
	if err := cr.Register(cube); err != nil {
		return fmt.Errorf("failed to register cube from remote repository: %w", err)
	}
	
	cr.logger.Info("Successfully loaded cube from remote repository", map[string]interface{}{
		"cube_id":  id,
		"repo_url": repoURL,
	})
	
	return nil
}

// Close closes all registered memory cubes
func (cr *CubeRegistry) Close() error {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	
	var errs []error
	for id, cube := range cr.cubes {
		if err := cube.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close cube %s: %w", id, err))
		}
	}
	
	cr.cubes = make(map[string]interfaces.MemCube)
	
	if len(errs) > 0 {
		return fmt.Errorf("errors occurred while closing cubes: %v", errs)
	}
	
	return nil
}