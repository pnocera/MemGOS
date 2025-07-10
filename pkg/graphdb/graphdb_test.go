package graphdb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/memtensor/memgos/pkg/interfaces"
)

type GraphDBTestSuite struct {
	suite.Suite
	ctx     context.Context
	cancel  context.CancelFunc
	logger  interfaces.Logger
	metrics interfaces.Metrics
	config  *GraphDBConfig
}

func (suite *GraphDBTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithTimeout(context.Background(), 30*time.Second)
	
	// Use nil for logger and metrics in tests (or implement mock)
	suite.logger = nil
	suite.metrics = nil
	
	// Test configuration
	suite.config = &GraphDBConfig{
		Provider:      ProviderNeo4j,
		URI:          "bolt://localhost:7687",
		Username:     "neo4j",
		Password:     "testpassword",
		Database:     "neo4j",
		MaxConnPool:  10,
		ConnTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		RetryAttempts: 2,
		RetryDelay:   500 * time.Millisecond,
		Metrics:      true,
		Logging:      true,
		SSLMode:      "disable",
	}
}

func (suite *GraphDBTestSuite) TearDownSuite() {
	suite.cancel()
}

func (suite *GraphDBTestSuite) TestConfigValidation() {
	cm := NewConfigManager()
	
	// Test valid config
	err := cm.ValidateConfig(suite.config)
	assert.NoError(suite.T(), err)
	
	// Test invalid configs
	invalidConfigs := []*GraphDBConfig{
		nil,
		{}, // empty config
		{Provider: ProviderNeo4j}, // missing URI
		{Provider: ProviderNeo4j, URI: "bolt://localhost:7687"}, // missing username
		{Provider: ProviderNeo4j, URI: "bolt://localhost:7687", Username: "neo4j"}, // missing password
		{Provider: ProviderNeo4j, URI: "bolt://localhost:7687", Username: "neo4j", Password: "pass", MaxConnPool: -1}, // invalid pool size
	}
	
	for _, config := range invalidConfigs {
		err := cm.ValidateConfig(config)
		assert.Error(suite.T(), err)
	}
}

func (suite *GraphDBTestSuite) TestConfigBuilder() {
	config, err := NewConfigBuilder().
		Provider(ProviderNeo4j).
		URI("bolt://localhost:7687").
		Credentials("neo4j", "password").
		Database("test").
		ConnectionPool(20, 10*time.Second).
		Timeouts(5*time.Second, 5*time.Second).
		Retry(3, time.Second).
		EnableMetrics().
		EnableLogging().
		SSL("disable").
		BuildAndValidate()
	
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), ProviderNeo4j, config.Provider)
	assert.Equal(suite.T(), "bolt://localhost:7687", config.URI)
	assert.Equal(suite.T(), "neo4j", config.Username)
	assert.Equal(suite.T(), "password", config.Password)
	assert.Equal(suite.T(), "test", config.Database)
	assert.Equal(suite.T(), 20, config.MaxConnPool)
	assert.Equal(suite.T(), 10*time.Second, config.ConnTimeout)
	assert.True(suite.T(), config.Metrics)
	assert.True(suite.T(), config.Logging)
}

func (suite *GraphDBTestSuite) TestQueryBuilder() {
	qb := NewQueryBuilder()
	
	// Build a complex query
	qb.Match("(n:Person)")
	qb.Where("n.age > $minAge")
	qb.OptionalMatch("(n)-[:KNOWS]->(friend)")
	qb.Return("n.name as name, count(friend) as friendCount")
	qb.OrderBy("friendCount DESC")
	qb.Limit(10)
	qb.AddParameter("minAge", 25)
	
	query, params := qb.Build()
	
	expectedQuery := "MATCH (n:Person) OPTIONAL MATCH (n)-[:KNOWS]->(friend) WHERE n.age > $minAge RETURN n.name as name, count(friend) as friendCount ORDER BY friendCount DESC LIMIT 10"
	assert.Equal(suite.T(), expectedQuery, query)
	assert.Equal(suite.T(), 25, params["minAge"])
}

func (suite *GraphDBTestSuite) TestFactory() {
	factory := NewGraphDBFactory(suite.logger, suite.metrics)
	
	// Test provider registration
	providers := factory.ListProviders()
	assert.Contains(suite.T(), providers, ProviderNeo4j)
	
	// Test instance creation (will fail without actual Neo4j instance)
	_, err := factory.Create(suite.config)
	assert.Error(suite.T(), err) // Expected to fail without real Neo4j
	
	// Test get non-existent instance
	_, exists := factory.Get(ProviderNeo4j, "bolt://localhost:7687", "neo4j")
	assert.False(suite.T(), exists)
}

func (suite *GraphDBTestSuite) TestGraphOperations() {
	// Create a mock graph database for testing operations
	mockDB := &MockGraphDB{}
	graphOps := NewGraphOperations(mockDB, suite.logger, suite.metrics)
	
	// Test query builder usage
	qb := NewQueryBuilder()
	qb.Match("(n:Test)")
	qb.Return("n")
	qb.Limit(1)
	
	query, params := qb.Build()
	assert.Equal(suite.T(), "MATCH (n:Test) RETURN n LIMIT 1", query)
	assert.Empty(suite.T(), params)
	
	// Test bulk operations (would normally interact with real DB)
	nodes := []GraphNode{
		{
			ID:         "test-1",
			Labels:     []string{"Test"},
			Properties: map[string]interface{}{"name": "test1"},
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
	}
	
	err := graphOps.BulkNodeCreation(suite.ctx, nodes, 1000)
	assert.NoError(suite.T(), err)
}

func (suite *GraphDBTestSuite) TestConnectionPool() {
	poolManager := NewConnectionPoolManager(suite.logger, suite.metrics)
	
	pool, err := poolManager.GetPool(suite.config)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), pool)
	
	// Test that we get the same pool for same config
	pool2, err := poolManager.GetPool(suite.config)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), pool, pool2)
}

func (suite *GraphDBTestSuite) TestDefaultConfigs() {
	// Test default GraphDB config
	defaultConfig := DefaultGraphDBConfig()
	assert.Equal(suite.T(), ProviderNeo4j, defaultConfig.Provider)
	assert.Equal(suite.T(), "bolt://localhost:7687", defaultConfig.URI)
	assert.Equal(suite.T(), 50, defaultConfig.MaxConnPool)
	
	// Test default Neo4j config
	neo4jConfig := DefaultNeo4jSpecificConfig()
	assert.Equal(suite.T(), "MemGOS-GraphDB/1.0", neo4jConfig.UserAgent)
	assert.Equal(suite.T(), 50, neo4jConfig.MaxConnectionPoolSize)
	
	// Test default performance config
	perfConfig := DefaultPerformanceConfig()
	assert.Equal(suite.T(), 1000, perfConfig.QueryCacheSize)
	assert.Equal(suite.T(), 1000, perfConfig.BatchSize)
	
	// Test default security config
	secConfig := DefaultSecurityConfig()
	assert.Equal(suite.T(), "basic", secConfig.AuthType)
	assert.Equal(suite.T(), "optional", secConfig.EncryptionLevel)
}

func (suite *GraphDBTestSuite) TestExtendedConfig() {
	extendedConfig := NewExtendedGraphDBConfig()
	assert.NotNil(suite.T(), extendedConfig.GraphDBConfig)
	assert.NotNil(suite.T(), extendedConfig.Neo4jConfig)
	assert.NotNil(suite.T(), extendedConfig.PerformanceConfig)
	assert.NotNil(suite.T(), extendedConfig.SecurityConfig)
	
	err := extendedConfig.Validate()
	assert.NoError(suite.T(), err)
}

// MockGraphDB implements ExtendedGraphDB for testing
type MockGraphDB struct {
	closed bool
}

func (m *MockGraphDB) CreateNode(ctx context.Context, labels []string, properties map[string]interface{}) (string, error) {
	return "mock-node-id", nil
}

func (m *MockGraphDB) CreateRelation(ctx context.Context, fromID, toID, relType string, properties map[string]interface{}) error {
	return nil
}

func (m *MockGraphDB) Query(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

func (m *MockGraphDB) GetNode(ctx context.Context, id string) (map[string]interface{}, error) {
	return map[string]interface{}{"id": id}, nil
}

func (m *MockGraphDB) Close() error {
	m.closed = true
	return nil
}

func (m *MockGraphDB) CreateNodes(ctx context.Context, nodes []GraphNode) ([]string, error) {
	ids := make([]string, len(nodes))
	for i := range nodes {
		ids[i] = "mock-id-" + string(rune(i))
	}
	return ids, nil
}

func (m *MockGraphDB) GetNodes(ctx context.Context, ids []string) ([]GraphNode, error) {
	nodes := make([]GraphNode, len(ids))
	for i, id := range ids {
		nodes[i] = GraphNode{ID: id}
	}
	return nodes, nil
}

func (m *MockGraphDB) UpdateNode(ctx context.Context, id string, properties map[string]interface{}) error {
	return nil
}

func (m *MockGraphDB) DeleteNode(ctx context.Context, id string) error {
	return nil
}

func (m *MockGraphDB) GetNodesByLabel(ctx context.Context, label string, limit int) ([]GraphNode, error) {
	return []GraphNode{}, nil
}

func (m *MockGraphDB) CreateRelations(ctx context.Context, relations []GraphRelation) error {
	return nil
}

func (m *MockGraphDB) GetRelations(ctx context.Context, fromID, toID, relType string) ([]GraphRelation, error) {
	return []GraphRelation{}, nil
}

func (m *MockGraphDB) UpdateRelation(ctx context.Context, id string, properties map[string]interface{}) error {
	return nil
}

func (m *MockGraphDB) DeleteRelation(ctx context.Context, id string) error {
	return nil
}

func (m *MockGraphDB) Traverse(ctx context.Context, startID string, options *GraphTraversalOptions) ([]GraphNode, error) {
	return []GraphNode{}, nil
}

func (m *MockGraphDB) FindPaths(ctx context.Context, startID, endID string, maxDepth int) ([]GraphPath, error) {
	return []GraphPath{}, nil
}

func (m *MockGraphDB) RunAlgorithm(ctx context.Context, options *GraphAlgorithmOptions) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (m *MockGraphDB) ExecuteBatch(ctx context.Context, operations []BatchOperation) error {
	return nil
}

func (m *MockGraphDB) CreateIndex(ctx context.Context, label, property string) error {
	return nil
}

func (m *MockGraphDB) CreateConstraint(ctx context.Context, label, property, constraintType string) error {
	return nil
}

func (m *MockGraphDB) DropIndex(ctx context.Context, label, property string) error {
	return nil
}

func (m *MockGraphDB) DropConstraint(ctx context.Context, label, property, constraintType string) error {
	return nil
}

func (m *MockGraphDB) GetSchema(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (m *MockGraphDB) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *MockGraphDB) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (m *MockGraphDB) BeginTransaction(ctx context.Context) (GraphTransaction, error) {
	return &MockTransaction{}, nil
}

// MockTransaction implements GraphTransaction for testing
type MockTransaction struct{}

func (mt *MockTransaction) Commit(ctx context.Context) error {
	return nil
}

func (mt *MockTransaction) Rollback(ctx context.Context) error {
	return nil
}

func (mt *MockTransaction) CreateNode(ctx context.Context, labels []string, properties map[string]interface{}) (string, error) {
	return "mock-tx-node-id", nil
}

func (mt *MockTransaction) CreateRelation(ctx context.Context, fromID, toID, relType string, properties map[string]interface{}) error {
	return nil
}

func (mt *MockTransaction) Query(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

func TestGraphDBTestSuite(t *testing.T) {
	suite.Run(t, new(GraphDBTestSuite))
}

// Benchmark tests
func BenchmarkQueryBuilder(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qb := NewQueryBuilder()
		qb.Match("(n:Person)")
		qb.Where("n.age > $minAge")
		qb.Return("n.name")
		qb.AddParameter("minAge", 25)
		_, _ = qb.Build()
	}
}

func BenchmarkConfigValidation(b *testing.B) {
	cm := NewConfigManager()
	config := DefaultGraphDBConfig()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cm.ValidateConfig(config)
	}
}

// Integration tests (require running Neo4j instance)
func TestNeo4jIntegration(t *testing.T) {
	t.Skip("Skipping integration tests - requires running Neo4j instance")
	
	config := &GraphDBConfig{
		Provider:      ProviderNeo4j,
		URI:          "bolt://localhost:7687",
		Username:     "neo4j",
		Password:     "password",
		Database:     "neo4j",
		MaxConnPool:  10,
		ConnTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		RetryAttempts: 2,
		RetryDelay:   500 * time.Millisecond,
		Metrics:      true,
		Logging:      false,
		SSLMode:      "disable",
	}
	
	db, err := NewNeo4jGraphDB(config, nil, nil)
	if err != nil {
		t.Skipf("Cannot connect to Neo4j: %v", err)
	}
	defer db.Close()
	
	ctx := context.Background()
	
	// Test health check
	err = db.HealthCheck(ctx)
	assert.NoError(t, err)
	
	// Test basic query
	results, err := db.Query(ctx, "RETURN 1 as test", nil)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, int64(1), results[0]["test"])
	
	// Test node creation
	nodeID, err := db.CreateNode(ctx, []string{"TestNode"}, map[string]interface{}{
		"name":      "test",
		"timestamp": time.Now().Unix(),
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, nodeID)
	
	// Test node retrieval
	node, err := db.GetNode(ctx, nodeID)
	assert.NoError(t, err)
	assert.NotNil(t, node)
	
	// Clean up
	err = db.DeleteNode(ctx, nodeID)
	assert.NoError(t, err)
}
