package vectordb

import (
	"context"
	"fmt"
	"sync"

	"github.com/memtensor/memgos/pkg/types"
)

// VectorDBFactory manages the creation and registration of vector database implementations
type VectorDBFactory struct {
	providers map[types.BackendType]VectorDBProvider
	mu        sync.RWMutex
}

// VectorDBProvider defines the interface for vector database providers
type VectorDBProvider interface {
	// Create creates a new vector database instance with the given configuration
	Create(config *VectorDBConfig) (BaseVectorDB, error)
	
	// Validate validates the configuration for this provider
	Validate(config *VectorDBConfig) error
	
	// GetBackendType returns the backend type this provider supports
	GetBackendType() types.BackendType
	
	// GetDefaultConfig returns a default configuration for this provider
	GetDefaultConfig() *VectorDBConfig
}

// QdrantProvider implements VectorDBProvider for Qdrant
type QdrantProvider struct{}

// Create creates a new Qdrant vector database instance
func (p *QdrantProvider) Create(config *VectorDBConfig) (BaseVectorDB, error) {
	if config.Backend != types.BackendQdrant {
		return nil, fmt.Errorf("invalid backend type: %s, expected: %s", config.Backend, types.BackendQdrant)
	}
	
	if config.Qdrant == nil {
		return nil, fmt.Errorf("qdrant configuration is required")
	}
	
	return NewQdrantVectorDB(config.Qdrant)
}

// Validate validates the Qdrant configuration
func (p *QdrantProvider) Validate(config *VectorDBConfig) error {
	if config.Backend != types.BackendQdrant {
		return fmt.Errorf("invalid backend type: %s, expected: %s", config.Backend, types.BackendQdrant)
	}
	
	if config.Qdrant == nil {
		return fmt.Errorf("qdrant configuration is required")
	}
	
	return config.Qdrant.Validate()
}

// GetBackendType returns the backend type
func (p *QdrantProvider) GetBackendType() types.BackendType {
	return types.BackendQdrant
}

// GetDefaultConfig returns default Qdrant configuration
func (p *QdrantProvider) GetDefaultConfig() *VectorDBConfig {
	return DefaultVectorDBConfig()
}

// Global factory instance
var (
	defaultFactory *VectorDBFactory
	factoryOnce    sync.Once
)

// GetFactory returns the default vector database factory
func GetFactory() *VectorDBFactory {
	factoryOnce.Do(func() {
		defaultFactory = NewVectorDBFactory()
		// Register default providers
		defaultFactory.RegisterProvider(&QdrantProvider{})
	})
	return defaultFactory
}

// NewVectorDBFactory creates a new vector database factory
func NewVectorDBFactory() *VectorDBFactory {
	return &VectorDBFactory{
		providers: make(map[types.BackendType]VectorDBProvider),
	}
}

// RegisterProvider registers a vector database provider
func (f *VectorDBFactory) RegisterProvider(provider VectorDBProvider) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	backendType := provider.GetBackendType()
	if _, exists := f.providers[backendType]; exists {
		return fmt.Errorf("provider for backend %s is already registered", backendType)
	}
	
	f.providers[backendType] = provider
	return nil
}

// UnregisterProvider unregisters a vector database provider
func (f *VectorDBFactory) UnregisterProvider(backendType types.BackendType) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if _, exists := f.providers[backendType]; !exists {
		return fmt.Errorf("provider for backend %s is not registered", backendType)
	}
	
	delete(f.providers, backendType)
	return nil
}

// GetProvider returns a provider for the given backend type
func (f *VectorDBFactory) GetProvider(backendType types.BackendType) (VectorDBProvider, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	provider, exists := f.providers[backendType]
	if !exists {
		return nil, fmt.Errorf("provider for backend %s is not registered", backendType)
	}
	
	return provider, nil
}

// ListProviders returns all registered provider backend types
func (f *VectorDBFactory) ListProviders() []types.BackendType {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	backends := make([]types.BackendType, 0, len(f.providers))
	for backend := range f.providers {
		backends = append(backends, backend)
	}
	
	return backends
}

// Create creates a new vector database instance
func (f *VectorDBFactory) Create(config *VectorDBConfig) (BaseVectorDB, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	provider, err := f.GetProvider(config.Backend)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}
	
	return provider.Create(config)
}

// CreateWithDefaults creates a vector database instance with default configuration for the backend
func (f *VectorDBFactory) CreateWithDefaults(backendType types.BackendType) (BaseVectorDB, error) {
	provider, err := f.GetProvider(backendType)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}
	
	config := provider.GetDefaultConfig()
	return provider.Create(config)
}

// Validate validates a configuration
func (f *VectorDBFactory) Validate(config *VectorDBConfig) error {
	provider, err := f.GetProvider(config.Backend)
	if err != nil {
		return fmt.Errorf("failed to get provider: %w", err)
	}
	
	return provider.Validate(config)
}

// ConnectionManager manages vector database connections with connection pooling
type ConnectionManager struct {
	connections map[string]BaseVectorDB
	configs     map[string]*VectorDBConfig
	factory     *VectorDBFactory
	mu          sync.RWMutex
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]BaseVectorDB),
		configs:     make(map[string]*VectorDBConfig),
		factory:     GetFactory(),
	}
}

// GetConnection gets or creates a connection with the given name and configuration
func (cm *ConnectionManager) GetConnection(ctx context.Context, name string, config *VectorDBConfig) (BaseVectorDB, error) {
	cm.mu.RLock()
	if db, exists := cm.connections[name]; exists {
		cm.mu.RUnlock()
		
		// Check if connection is still valid
		if db.IsConnected() {
			return db, nil
		}
		
		// Connection is stale, remove it and create a new one
		cm.mu.Lock()
		delete(cm.connections, name)
		delete(cm.configs, name)
		cm.mu.Unlock()
	} else {
		cm.mu.RUnlock()
	}
	
	// Create new connection
	db, err := cm.factory.Create(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vector database: %w", err)
	}
	
	// Connect to the database
	if err := db.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to vector database: %w", err)
	}
	
	// Store the connection
	cm.mu.Lock()
	cm.connections[name] = db
	cm.configs[name] = config.Clone()
	cm.mu.Unlock()
	
	return db, nil
}

// RemoveConnection removes a connection
func (cm *ConnectionManager) RemoveConnection(ctx context.Context, name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if db, exists := cm.connections[name]; exists {
		if err := db.Disconnect(ctx); err != nil {
			return fmt.Errorf("failed to disconnect: %w", err)
		}
		delete(cm.connections, name)
		delete(cm.configs, name)
	}
	
	return nil
}

// ListConnections returns all active connection names
func (cm *ConnectionManager) ListConnections() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	names := make([]string, 0, len(cm.connections))
	for name := range cm.connections {
		names = append(names, name)
	}
	
	return names
}

// CloseAll closes all connections
func (cm *ConnectionManager) CloseAll(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	var lastErr error
	for name, db := range cm.connections {
		if err := db.Disconnect(ctx); err != nil {
			lastErr = err
		}
		delete(cm.connections, name)
		delete(cm.configs, name)
	}
	
	return lastErr
}

// HealthCheck performs health checks on all connections
func (cm *ConnectionManager) HealthCheck(ctx context.Context) map[string]error {
	cm.mu.RLock()
	connections := make(map[string]BaseVectorDB, len(cm.connections))
	for name, db := range cm.connections {
		connections[name] = db
	}
	cm.mu.RUnlock()
	
	results := make(map[string]error)
	for name, db := range connections {
		results[name] = db.HealthCheck(ctx)
	}
	
	return results
}

// ConfiguredVectorDB represents a pre-configured vector database
type ConfiguredVectorDB struct {
	Name     string         `json:"name"`
	Backend  types.BackendType `json:"backend"`
	Config   *VectorDBConfig   `json:"config"`
	Database BaseVectorDB     `json:"-"` // Not serialized
}

// VectorDBRegistry manages pre-configured vector databases
type VectorDBRegistry struct {
	databases map[string]*ConfiguredVectorDB
	manager   *ConnectionManager
	mu        sync.RWMutex
}

// NewVectorDBRegistry creates a new vector database registry
func NewVectorDBRegistry() *VectorDBRegistry {
	return &VectorDBRegistry{
		databases: make(map[string]*ConfiguredVectorDB),
		manager:   NewConnectionManager(),
	}
}

// Register registers a vector database configuration
func (r *VectorDBRegistry) Register(name string, config *VectorDBConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.databases[name]; exists {
		return fmt.Errorf("database %s is already registered", name)
	}
	
	r.databases[name] = &ConfiguredVectorDB{
		Name:    name,
		Backend: config.Backend,
		Config:  config.Clone(),
	}
	
	return nil
}

// Unregister removes a vector database configuration
func (r *VectorDBRegistry) Unregister(ctx context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.databases[name]; !exists {
		return fmt.Errorf("database %s is not registered", name)
	}
	
	// Close connection if exists
	_ = r.manager.RemoveConnection(ctx, name)
	
	delete(r.databases, name)
	return nil
}

// Get gets a vector database instance
func (r *VectorDBRegistry) Get(ctx context.Context, name string) (BaseVectorDB, error) {
	r.mu.RLock()
	configured, exists := r.databases[name]
	r.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("database %s is not registered", name)
	}
	
	return r.manager.GetConnection(ctx, name, configured.Config)
}

// List returns all registered database names
func (r *VectorDBRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	names := make([]string, 0, len(r.databases))
	for name := range r.databases {
		names = append(names, name)
	}
	
	return names
}

// GetInfo returns information about a registered database
func (r *VectorDBRegistry) GetInfo(name string) (*ConfiguredVectorDB, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	configured, exists := r.databases[name]
	if !exists {
		return nil, fmt.Errorf("database %s is not registered", name)
	}
	
	// Return a copy to prevent external modification
	return &ConfiguredVectorDB{
		Name:    configured.Name,
		Backend: configured.Backend,
		Config:  configured.Config.Clone(),
	}, nil
}

// Close closes all connections in the registry
func (r *VectorDBRegistry) Close(ctx context.Context) error {
	return r.manager.CloseAll(ctx)
}

// Global registry instance
var (
	defaultRegistry *VectorDBRegistry
	registryOnce    sync.Once
)

// GetRegistry returns the default vector database registry
func GetRegistry() *VectorDBRegistry {
	registryOnce.Do(func() {
		defaultRegistry = NewVectorDBRegistry()
	})
	return defaultRegistry
}

// Convenience functions for the default factory and registry

// CreateVectorDB creates a vector database using the default factory
func CreateVectorDB(config *VectorDBConfig) (BaseVectorDB, error) {
	return GetFactory().Create(config)
}

// CreateVectorDBWithDefaults creates a vector database with default configuration
func CreateVectorDBWithDefaults(backendType types.BackendType) (BaseVectorDB, error) {
	return GetFactory().CreateWithDefaults(backendType)
}

// RegisterVectorDB registers a vector database in the default registry
func RegisterVectorDB(name string, config *VectorDBConfig) error {
	return GetRegistry().Register(name, config)
}

// GetVectorDB gets a vector database from the default registry
func GetVectorDB(ctx context.Context, name string) (BaseVectorDB, error) {
	return GetRegistry().Get(ctx, name)
}