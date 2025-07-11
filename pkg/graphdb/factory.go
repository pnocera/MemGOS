package graphdb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
)

// GraphDBFactory manages graph database instances
type GraphDBFactory struct {
	mu        sync.RWMutex
	providers map[GraphDBProvider]GraphDBConstructor
	instances map[string]ExtendedGraphDB
	logger    interfaces.Logger
	metrics   interfaces.Metrics
}

// GraphDBConstructor defines a function that creates graph database instances
type GraphDBConstructor func(config *GraphDBConfig, logger interfaces.Logger, metrics interfaces.Metrics) (ExtendedGraphDB, error)

// NewGraphDBFactory creates a new graph database factory
func NewGraphDBFactory(logger interfaces.Logger, metrics interfaces.Metrics) *GraphDBFactory {
	factory := &GraphDBFactory{
		providers: make(map[GraphDBProvider]GraphDBConstructor),
		instances: make(map[string]ExtendedGraphDB),
		logger:    logger,
		metrics:   metrics,
	}

	// Register default providers
	factory.RegisterProvider(ProviderNeo4j, func(config *GraphDBConfig, logger interfaces.Logger, metrics interfaces.Metrics) (ExtendedGraphDB, error) {
		return NewNeo4jGraphDB(config, logger, metrics)
	})
	
	factory.RegisterProvider(ProviderKuzu, func(config *GraphDBConfig, logger interfaces.Logger, metrics interfaces.Metrics) (ExtendedGraphDB, error) {
		return NewKuzuGraphDB(config, logger, metrics)
	})

	return factory
}

// RegisterProvider registers a new graph database provider
func (f *GraphDBFactory) RegisterProvider(provider GraphDBProvider, constructor GraphDBConstructor) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.providers[provider] = constructor

	if f.logger != nil {
		f.logger.Info("Graph database provider registered", map[string]interface{}{
			"provider": string(provider),
		})
	}
}

// Create creates a new graph database instance
func (f *GraphDBFactory) Create(config *GraphDBConfig) (ExtendedGraphDB, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if config == nil {
		config = DefaultGraphDBConfig()
	}

	constructor, exists := f.providers[config.Provider]
	if !exists {
		return nil, fmt.Errorf("unsupported graph database provider: %s", config.Provider)
	}

	instance, err := constructor(config, f.logger, f.metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to create graph database instance: %w", err)
	}

	// Generate instance key
	instanceKey := fmt.Sprintf("%s_%s_%s", config.Provider, config.URI, config.Database)
	f.instances[instanceKey] = instance

	if f.logger != nil {
		f.logger.Info("Graph database instance created", map[string]interface{}{
			"provider": string(config.Provider),
			"uri":      config.URI,
			"database": config.Database,
		})
	}

	if f.metrics != nil {
		f.metrics.Counter("graphdb_instances_created", 1, map[string]string{
			"provider": string(config.Provider),
		})
	}

	return instance, nil
}

// Get retrieves an existing graph database instance
func (f *GraphDBFactory) Get(provider GraphDBProvider, uri, database string) (ExtendedGraphDB, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	instanceKey := fmt.Sprintf("%s_%s_%s", provider, uri, database)
	instance, exists := f.instances[instanceKey]
	return instance, exists
}

// GetOrCreate gets an existing instance or creates a new one
func (f *GraphDBFactory) GetOrCreate(config *GraphDBConfig) (ExtendedGraphDB, error) {
	if config == nil {
		config = DefaultGraphDBConfig()
	}

	if instance, exists := f.Get(config.Provider, config.URI, config.Database); exists {
		return instance, nil
	}

	return f.Create(config)
}

// CloseAll closes all graph database instances
func (f *GraphDBFactory) CloseAll() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var lastErr error
	for key, instance := range f.instances {
		if err := instance.Close(); err != nil {
			lastErr = err
			if f.logger != nil {
				f.logger.Error("Failed to close graph database instance", err, map[string]interface{}{
					"key": key,
				})
			}
		}
	}

	f.instances = make(map[string]ExtendedGraphDB)

	if f.logger != nil {
		f.logger.Info("All graph database instances closed")
	}

	return lastErr
}

// ListProviders returns all registered providers
func (f *GraphDBFactory) ListProviders() []GraphDBProvider {
	f.mu.RLock()
	defer f.mu.RUnlock()

	providers := make([]GraphDBProvider, 0, len(f.providers))
	for provider := range f.providers {
		providers = append(providers, provider)
	}

	return providers
}

// ListInstances returns all active instances
func (f *GraphDBFactory) ListInstances() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	instances := make([]string, 0, len(f.instances))
	for key := range f.instances {
		instances = append(instances, key)
	}

	return instances
}

// HealthCheck performs health checks on all instances
func (f *GraphDBFactory) HealthCheck(ctx context.Context) map[string]error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	results := make(map[string]error)
	for key, instance := range f.instances {
		if err := instance.HealthCheck(ctx); err != nil {
			results[key] = err
		} else {
			results[key] = nil
		}
	}

	return results
}

// GetMetrics returns metrics for all instances
func (f *GraphDBFactory) GetMetrics(ctx context.Context) map[string]map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	results := make(map[string]map[string]interface{})
	for key, instance := range f.instances {
		if metrics, err := instance.GetMetrics(ctx); err == nil {
			results[key] = metrics
		}
	}

	return results
}

// GraphDBManager provides high-level graph database management
type GraphDBManager struct {
	factory   *GraphDBFactory
	defaultDB ExtendedGraphDB
	logger    interfaces.Logger
	metrics   interfaces.Metrics
	mu        sync.RWMutex
}

// NewGraphDBManager creates a new graph database manager
func NewGraphDBManager(config *GraphDBConfig, logger interfaces.Logger, metrics interfaces.Metrics) (*GraphDBManager, error) {
	factory := NewGraphDBFactory(logger, metrics)
	
	defaultDB, err := factory.Create(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create default graph database: %w", err)
	}

	return &GraphDBManager{
		factory:   factory,
		defaultDB: defaultDB,
		logger:    logger,
		metrics:   metrics,
	}, nil
}

// GetDefault returns the default graph database instance
func (m *GraphDBManager) GetDefault() ExtendedGraphDB {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultDB
}

// GetFactory returns the graph database factory
func (m *GraphDBManager) GetFactory() *GraphDBFactory {
	return m.factory
}

// CreateDatabase creates a new graph database instance
func (m *GraphDBManager) CreateDatabase(config *GraphDBConfig) (ExtendedGraphDB, error) {
	return m.factory.Create(config)
}

// SetDefault sets a new default database
func (m *GraphDBManager) SetDefault(db ExtendedGraphDB) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultDB = db
}

// Close closes the manager and all instances
func (m *GraphDBManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.factory.CloseAll(); err != nil {
		return err
	}

	m.defaultDB = nil
	return nil
}

// HealthCheck performs health checks on all managed databases
func (m *GraphDBManager) HealthCheck(ctx context.Context) map[string]error {
	return m.factory.HealthCheck(ctx)
}

// GetMetrics returns metrics for all managed databases
func (m *GraphDBManager) GetMetrics(ctx context.Context) map[string]map[string]interface{} {
	return m.factory.GetMetrics(ctx)
}

// Global factory instance
var globalFactory *GraphDBFactory
var globalFactoryOnce sync.Once

// GetGlobalFactory returns the global graph database factory
func GetGlobalFactory() *GraphDBFactory {
	globalFactoryOnce.Do(func() {
		globalFactory = NewGraphDBFactory(nil, nil)
	})
	return globalFactory
}

// CreateGraphDB creates a graph database using the global factory
func CreateGraphDB(config *GraphDBConfig) (ExtendedGraphDB, error) {
	return GetGlobalFactory().Create(config)
}

// GetOrCreateGraphDB gets or creates a graph database using the global factory
func GetOrCreateGraphDB(config *GraphDBConfig) (ExtendedGraphDB, error) {
	return GetGlobalFactory().GetOrCreate(config)
}

// Connection pool manager
type ConnectionPoolManager struct {
	pools   map[string]*ConnectionPool
	mu      sync.RWMutex
	logger  interfaces.Logger
	metrics interfaces.Metrics
}

// ConnectionPool manages connections to a graph database
type ConnectionPool struct {
	config      *GraphDBConfig
	connections chan ExtendedGraphDB
	created     int
	maxSize     int
	currentSize int
	mu          sync.RWMutex
	closed      bool
	factory     *GraphDBFactory
}

// NewConnectionPoolManager creates a new connection pool manager
func NewConnectionPoolManager(logger interfaces.Logger, metrics interfaces.Metrics) *ConnectionPoolManager {
	return &ConnectionPoolManager{
		pools:   make(map[string]*ConnectionPool),
		logger:  logger,
		metrics: metrics,
	}
}

// GetPool gets or creates a connection pool
func (cpm *ConnectionPoolManager) GetPool(config *GraphDBConfig) (*ConnectionPool, error) {
	cpm.mu.Lock()
	defer cpm.mu.Unlock()

	key := fmt.Sprintf("%s_%s_%s", config.Provider, config.URI, config.Database)
	if pool, exists := cpm.pools[key]; exists {
		return pool, nil
	}

	pool := &ConnectionPool{
		config:      config,
		connections: make(chan ExtendedGraphDB, config.MaxConnPool),
		maxSize:     config.MaxConnPool,
		factory:     NewGraphDBFactory(cpm.logger, cpm.metrics),
	}

	cpm.pools[key] = pool
	return pool, nil
}

// Get gets a connection from the pool
func (cp *ConnectionPool) Get(ctx context.Context) (ExtendedGraphDB, error) {
	cp.mu.RLock()
	if cp.closed {
		cp.mu.RUnlock()
		return nil, fmt.Errorf("connection pool is closed")
	}
	cp.mu.RUnlock()

	select {
	case conn := <-cp.connections:
		return conn, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Create new connection if pool not at max
		cp.mu.Lock()
		if cp.currentSize < cp.maxSize {
			conn, err := cp.factory.Create(cp.config)
			if err != nil {
				cp.mu.Unlock()
				return nil, err
			}
			cp.currentSize++
			cp.mu.Unlock()
			return conn, nil
		}
		cp.mu.Unlock()

		// Wait for connection with timeout
		select {
		case conn := <-cp.connections:
			return conn, nil
		case <-time.After(cp.config.ConnTimeout):
			return nil, fmt.Errorf("connection pool timeout")
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// Put returns a connection to the pool
func (cp *ConnectionPool) Put(conn ExtendedGraphDB) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	if cp.closed {
		conn.Close()
		return
	}

	select {
	case cp.connections <- conn:
		// Connection returned to pool
	default:
		// Pool is full, close the connection
		conn.Close()
		cp.mu.RUnlock()
		cp.mu.Lock()
		cp.currentSize--
		cp.mu.Unlock()
		cp.mu.RLock()
	}
}

// Close closes the connection pool
func (cp *ConnectionPool) Close() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.closed {
		return nil
	}

	cp.closed = true
	close(cp.connections)

	// Close all connections in pool
	for conn := range cp.connections {
		conn.Close()
	}

	return nil
}
