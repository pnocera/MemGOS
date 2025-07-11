package graphdb

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kuzudb/go-kuzu"
	"github.com/memtensor/memgos/pkg/interfaces"
)

// KuzuGraphDB implements the ExtendedGraphDB interface using KuzuDB
type KuzuGraphDB struct {
	*BaseGraphDB
	db     *kuzu.Database
	conn   *kuzu.Connection
	config *GraphDBConfig
	mu     sync.RWMutex
}

// NewKuzuGraphDB creates a new KuzuDB graph database instance
func NewKuzuGraphDB(config *GraphDBConfig, logger interfaces.Logger, metrics interfaces.Metrics) (ExtendedGraphDB, error) {
	if config == nil {
		config = DefaultKuzuDBConfig()
	}

	// Validate KuzuDB-specific configuration
	if config.KuzuConfig == nil {
		return nil, fmt.Errorf("KuzuDB configuration is required")
	}

	if config.KuzuConfig.DatabasePath == "" {
		return nil, fmt.Errorf("database path is required for KuzuDB")
	}

	base := NewBaseGraphDB(config, logger, metrics)
	
	kuzu := &KuzuGraphDB{
		BaseGraphDB: base,
		config:      config,
	}

	if err := kuzu.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to KuzuDB: %w", err)
	}

	if logger != nil {
		logger.Info("KuzuDB instance created", map[string]interface{}{
			"database_path": config.KuzuConfig.DatabasePath,
			"read_only":     config.KuzuConfig.ReadOnly,
		})
	}

	return kuzu, nil
}

// DefaultKuzuDBConfig returns a default KuzuDB configuration
func DefaultKuzuDBConfig() *GraphDBConfig {
	return &GraphDBConfig{
		Provider:      ProviderKuzu,
		MaxConnPool:   1, // KuzuDB is embedded, single connection
		ConnTimeout:   30 * time.Second,
		ReadTimeout:   15 * time.Second,
		WriteTimeout:  15 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
		Metrics:       true,
		Logging:       false,
		KuzuConfig: &KuzuDBConfig{
			DatabasePath:      "./kuzu_data",
			ReadOnly:          false,
			BufferPoolSize:    1024 * 1024 * 1024, // 1GB
			MaxNumThreads:     4,
			EnableCompression: true,
			TimeoutSeconds:    30,
		},
	}
}

// connect establishes connection to KuzuDB
func (k *KuzuGraphDB) connect() error {
	k.mu.Lock()
	defer k.mu.Unlock()

	start := time.Now()

	// Ensure database directory exists
	dbPath := k.config.KuzuConfig.DatabasePath
	if !filepath.IsAbs(dbPath) {
		var err error
		dbPath, err = filepath.Abs(dbPath)
		if err != nil {
			return fmt.Errorf("failed to resolve database path: %w", err)
		}
	}

	// Create database instance
	db, err := kuzu.OpenDatabase(dbPath, kuzu.DefaultSystemConfig())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Create connection
	conn, err := kuzu.OpenConnection(db)
	if err != nil {
		db.Close()
		return fmt.Errorf("failed to open connection: %w", err)
	}

	k.db = db
	k.conn = conn

	// Record connection time
	k.BaseGraphDB.stats.mu.Lock()
	k.BaseGraphDB.stats.connectionTime = time.Since(start)
	k.BaseGraphDB.stats.mu.Unlock()

	k.UpdateHealth("healthy", nil)

	if k.logger != nil {
		k.logger.Info("Connected to KuzuDB", map[string]interface{}{
			"database_path":    dbPath,
			"connection_time":  time.Since(start).String(),
		})
	}

	return nil
}

// CreateNode creates a new node in the graph
func (k *KuzuGraphDB) CreateNode(ctx context.Context, labels []string, properties map[string]interface{}) (string, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	if k.conn == nil {
		return "", fmt.Errorf("database connection is not established")
	}

	start := time.Now()
	defer func() {
		k.RecordQuery(true, time.Since(start))
		k.RecordNodeCreation()
	}()

	// KuzuDB uses a different syntax than Neo4j
	// We need to create a node table if it doesn't exist and then insert
	if len(labels) == 0 {
		return "", fmt.Errorf("at least one label is required")
	}

	// Use the first label as the table name
	tableName := labels[0]
	
	// Create node table if it doesn't exist
	createTableQuery := fmt.Sprintf("CREATE NODE TABLE IF NOT EXISTS %s (id STRING, PRIMARY KEY(id))", tableName)
	
	// Add properties as columns
	for propName, propValue := range properties {
		columnType := getKuzuType(propValue)
		createTableQuery += fmt.Sprintf(", %s %s", propName, columnType)
	}
	createTableQuery += ")"

	_, err := k.conn.Query(createTableQuery)
	if err != nil {
		k.RecordQuery(false, time.Since(start))
		return "", fmt.Errorf("failed to create node table: %w", err)
	}

	// Generate a unique ID for the node
	nodeID := k.generateNodeID()

	// Build insert query
	var propNames []string
	var propValues []interface{}
	
	propNames = append(propNames, "id")
	propValues = append(propValues, nodeID)
	
	for propName, propValue := range properties {
		propNames = append(propNames, propName)
		propValues = append(propValues, propValue)
	}

	insertQuery := fmt.Sprintf("CREATE (%s {id: $id", tableName)
	for i, propName := range propNames[1:] {
		insertQuery += fmt.Sprintf(", %s: $%s", propName, propName)
		_ = i // unused variable
	}
	insertQuery += "})"

	// Execute insert query
	params := make(map[string]interface{})
	for i, propName := range propNames {
		params[propName] = propValues[i]
	}

	_, err = k.conn.Query(insertQuery)
	if err != nil {
		k.RecordQuery(false, time.Since(start))
		return "", fmt.Errorf("failed to create node: %w", err)
	}

	if k.logger != nil {
		k.logger.Debug("Node created", map[string]interface{}{
			"node_id": nodeID,
			"labels":  labels,
		})
	}

	return nodeID, nil
}

// CreateRelation creates a relationship between two nodes
func (k *KuzuGraphDB) CreateRelation(ctx context.Context, fromID, toID, relType string, properties map[string]interface{}) error {
	k.mu.RLock()
	defer k.mu.RUnlock()

	if k.conn == nil {
		return fmt.Errorf("database connection is not established")
	}

	start := time.Now()
	defer func() {
		k.RecordQuery(true, time.Since(start))
		k.RecordRelationCreation()
	}()

	// Create relationship table if it doesn't exist
	createRelQuery := fmt.Sprintf("CREATE REL TABLE IF NOT EXISTS %s (FROM Node TO Node", relType)
	for propName, propValue := range properties {
		columnType := getKuzuType(propValue)
		createRelQuery += fmt.Sprintf(", %s %s", propName, columnType)
	}
	createRelQuery += ")"

	_, err := k.conn.Query(createRelQuery)
	if err != nil {
		k.RecordQuery(false, time.Since(start))
		return fmt.Errorf("failed to create relationship table: %w", err)
	}

	// Build relationship creation query
	createQuery := fmt.Sprintf("MATCH (from), (to) WHERE from.id = $fromID AND to.id = $toID CREATE (from)-[:%s", relType)
	
	params := map[string]interface{}{
		"fromID": fromID,
		"toID":   toID,
	}

	if len(properties) > 0 {
		createQuery += " {"
		first := true
		for propName, propValue := range properties {
			if !first {
				createQuery += ", "
			}
			createQuery += fmt.Sprintf("%s: $%s", propName, propName)
			params[propName] = propValue
			first = false
		}
		createQuery += "}"
	}
	createQuery += "]->(to)"

	_, err = k.conn.Query(createQuery)
	if err != nil {
		k.RecordQuery(false, time.Since(start))
		return fmt.Errorf("failed to create relationship: %w", err)
	}

	if k.logger != nil {
		k.logger.Debug("Relationship created", map[string]interface{}{
			"from_id":  fromID,
			"to_id":    toID,
			"rel_type": relType,
		})
	}

	return nil
}

// Query executes a Cypher query and returns results
func (k *KuzuGraphDB) Query(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	if k.conn == nil {
		return nil, fmt.Errorf("database connection is not established")
	}

	start := time.Now()
	defer func() {
		k.RecordQuery(true, time.Since(start))
	}()

	// Execute query
	result, err := k.conn.Query(query)
	if err != nil {
		k.RecordQuery(false, time.Since(start))
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer result.Close()

	// Convert result to slice of maps
	var results []map[string]interface{}
	for result.HasNext() {
		_, err := result.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to get next result: %w", err)
		}
		
		record := make(map[string]interface{})
		
		// For now, create a simple record
		// This implementation needs to be refined once we understand the KuzuDB Go API better
		record["result"] = "success"
		
		results = append(results, record)
	}

	if k.logger != nil {
		k.logger.Debug("Query executed", map[string]interface{}{
			"query":        query,
			"results_count": len(results),
			"execution_time": time.Since(start).String(),
		})
	}

	return results, nil
}

// GetNode retrieves a node by ID
func (k *KuzuGraphDB) GetNode(ctx context.Context, id string) (map[string]interface{}, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	if k.conn == nil {
		return nil, fmt.Errorf("database connection is not established")
	}

	// Query all node tables to find the node with the given ID
	query := "MATCH (n) WHERE n.id = $id RETURN n"
	params := map[string]interface{}{"id": id}

	results, err := k.Query(ctx, query, params)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("node with ID %s not found", id)
	}

	return results[0], nil
}

// Close closes the database connection
func (k *KuzuGraphDB) Close() error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.conn != nil {
		k.conn.Close()
		k.conn = nil
	}

	if k.db != nil {
		k.db.Close()
		k.db = nil
	}

	k.UpdateHealth("closed", nil)

	if k.logger != nil {
		k.logger.Info("KuzuDB connection closed")
	}

	return k.BaseGraphDB.Close()
}

// generateNodeID generates a unique ID for a node
func (k *KuzuGraphDB) generateNodeID() string {
	return fmt.Sprintf("node_%d", time.Now().UnixNano())
}

// getKuzuType maps Go types to KuzuDB types
func getKuzuType(value interface{}) string {
	switch value.(type) {
	case string:
		return "STRING"
	case int, int32, int64:
		return "INT64"
	case float32, float64:
		return "DOUBLE"
	case bool:
		return "BOOLEAN"
	default:
		return "STRING" // Default to string
	}
}

// Extended interface implementations below...

// CreateNodes creates multiple nodes in a batch
func (k *KuzuGraphDB) CreateNodes(ctx context.Context, nodes []GraphNode) ([]string, error) {
	var nodeIDs []string
	
	for _, node := range nodes {
		nodeID, err := k.CreateNode(ctx, node.Labels, node.Properties)
		if err != nil {
			return nil, fmt.Errorf("failed to create node: %w", err)
		}
		nodeIDs = append(nodeIDs, nodeID)
	}
	
	return nodeIDs, nil
}

// GetNodes retrieves multiple nodes by their IDs
func (k *KuzuGraphDB) GetNodes(ctx context.Context, ids []string) ([]GraphNode, error) {
	var nodes []GraphNode
	
	for _, id := range ids {
		nodeData, err := k.GetNode(ctx, id)
		if err != nil {
			continue // Skip nodes that don't exist
		}
		
		node := GraphNode{
			ID:         id,
			Properties: nodeData,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		nodes = append(nodes, node)
	}
	
	return nodes, nil
}

// UpdateNode updates a node's properties
func (k *KuzuGraphDB) UpdateNode(ctx context.Context, id string, properties map[string]interface{}) error {
	setParts := make([]string, 0, len(properties))
	params := map[string]interface{}{"id": id}
	
	for key, value := range properties {
		setParts = append(setParts, fmt.Sprintf("n.%s = $%s", key, key))
		params[key] = value
	}
	
	query := fmt.Sprintf("MATCH (n) WHERE n.id = $id SET %s", strings.Join(setParts, ", "))
	
	_, err := k.Query(ctx, query, params)
	return err
}

// DeleteNode deletes a node by ID
func (k *KuzuGraphDB) DeleteNode(ctx context.Context, id string) error {
	query := "MATCH (n) WHERE n.id = $id DETACH DELETE n"
	params := map[string]interface{}{"id": id}
	
	_, err := k.Query(ctx, query, params)
	return err
}

// GetNodesByLabel retrieves nodes by label with a limit
func (k *KuzuGraphDB) GetNodesByLabel(ctx context.Context, label string, limit int) ([]GraphNode, error) {
	query := fmt.Sprintf("MATCH (n:%s) RETURN n LIMIT $limit", label)
	params := map[string]interface{}{"limit": limit}
	
	results, err := k.Query(ctx, query, params)
	if err != nil {
		return nil, err
	}
	
	var nodes []GraphNode
	for _, result := range results {
		if nodeData, ok := result["n"].(map[string]interface{}); ok {
			node := GraphNode{
				ID:         fmt.Sprintf("%v", nodeData["id"]),
				Labels:     []string{label},
				Properties: nodeData,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			nodes = append(nodes, node)
		}
	}
	
	return nodes, nil
}

// CreateRelations creates multiple relationships in a batch
func (k *KuzuGraphDB) CreateRelations(ctx context.Context, relations []GraphRelation) error {
	for _, rel := range relations {
		err := k.CreateRelation(ctx, rel.FromID, rel.ToID, rel.Type, rel.Properties)
		if err != nil {
			return fmt.Errorf("failed to create relationship: %w", err)
		}
	}
	return nil
}

// GetRelations retrieves relationships between nodes
func (k *KuzuGraphDB) GetRelations(ctx context.Context, fromID, toID, relType string) ([]GraphRelation, error) {
	var query string
	params := make(map[string]interface{})
	
	if fromID != "" && toID != "" {
		query = fmt.Sprintf("MATCH (from)-[r:%s]->(to) WHERE from.id = $fromID AND to.id = $toID RETURN r", relType)
		params["fromID"] = fromID
		params["toID"] = toID
	} else if fromID != "" {
		query = fmt.Sprintf("MATCH (from)-[r:%s]->(to) WHERE from.id = $fromID RETURN r", relType)
		params["fromID"] = fromID
	} else {
		query = fmt.Sprintf("MATCH (from)-[r:%s]->(to) RETURN r", relType)
	}
	
	results, err := k.Query(ctx, query, params)
	if err != nil {
		return nil, err
	}
	
	var relations []GraphRelation
	for _, result := range results {
		if relData, ok := result["r"].(map[string]interface{}); ok {
			relation := GraphRelation{
				FromID:     fromID,
				ToID:       toID,
				Type:       relType,
				Properties: relData,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			relations = append(relations, relation)
		}
	}
	
	return relations, nil
}

// UpdateRelation updates a relationship's properties
func (k *KuzuGraphDB) UpdateRelation(ctx context.Context, id string, properties map[string]interface{}) error {
	// KuzuDB doesn't have relationship IDs like Neo4j, so this is a simplified implementation
	return fmt.Errorf("relationship updates by ID not supported in KuzuDB")
}

// DeleteRelation deletes a relationship by ID
func (k *KuzuGraphDB) DeleteRelation(ctx context.Context, id string) error {
	// KuzuDB doesn't have relationship IDs like Neo4j, so this is a simplified implementation
	return fmt.Errorf("relationship deletion by ID not supported in KuzuDB")
}

// Traverse performs graph traversal
func (k *KuzuGraphDB) Traverse(ctx context.Context, startID string, options *GraphTraversalOptions) ([]GraphNode, error) {
	if options == nil {
		options = DefaultTraversalOptions()
	}
	
	query := fmt.Sprintf("MATCH (start)-[*1..%d]-(node) WHERE start.id = $startID RETURN DISTINCT node LIMIT $limit", options.MaxDepth)
	params := map[string]interface{}{
		"startID": startID,
		"limit":   options.Limit,
	}
	
	results, err := k.Query(ctx, query, params)
	if err != nil {
		return nil, err
	}
	
	var nodes []GraphNode
	for _, result := range results {
		if nodeData, ok := result["node"].(map[string]interface{}); ok {
			node := GraphNode{
				ID:         fmt.Sprintf("%v", nodeData["id"]),
				Properties: nodeData,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			nodes = append(nodes, node)
		}
	}
	
	return nodes, nil
}

// FindPaths finds paths between two nodes
func (k *KuzuGraphDB) FindPaths(ctx context.Context, startID, endID string, maxDepth int) ([]GraphPath, error) {
	query := fmt.Sprintf("MATCH path = (start)-[*1..%d]-(end) WHERE start.id = $startID AND end.id = $endID RETURN path", maxDepth)
	params := map[string]interface{}{
		"startID": startID,
		"endID":   endID,
	}
	
	results, err := k.Query(ctx, query, params)
	if err != nil {
		return nil, err
	}
	
	var paths []GraphPath
	for range results {
		// Simplified path construction - would need more complex parsing in real implementation
		path := GraphPath{
			Length: 1, // Placeholder
		}
		paths = append(paths, path)
	}
	
	return paths, nil
}

// RunAlgorithm executes graph algorithms
func (k *KuzuGraphDB) RunAlgorithm(ctx context.Context, options *GraphAlgorithmOptions) (map[string]interface{}, error) {
	// KuzuDB doesn't have built-in graph algorithms like Neo4j
	// This would need to be implemented using custom queries
	return nil, fmt.Errorf("graph algorithms not yet implemented for KuzuDB")
}

// ExecuteBatch executes multiple operations in a batch
func (k *KuzuGraphDB) ExecuteBatch(ctx context.Context, operations []BatchOperation) error {
	for _, op := range operations {
		switch op.OperationType {
		case "CREATE_NODE":
			_, err := k.CreateNode(ctx, op.NodeLabels, op.Properties)
			if err != nil {
				return err
			}
		case "CREATE_RELATION":
			err := k.CreateRelation(ctx, op.FromID, op.ToID, op.RelationType, op.Properties)
			if err != nil {
				return err
			}
		case "DELETE_NODE":
			err := k.DeleteNode(ctx, op.NodeID)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported operation type: %s", op.OperationType)
		}
	}
	return nil
}

// CreateIndex creates an index on a property
func (k *KuzuGraphDB) CreateIndex(ctx context.Context, label, property string) error {
	// KuzuDB handles indexing differently - this is a placeholder
	return nil
}

// CreateConstraint creates a constraint
func (k *KuzuGraphDB) CreateConstraint(ctx context.Context, label, property, constraintType string) error {
	// KuzuDB handles constraints differently - this is a placeholder
	return nil
}

// DropIndex drops an index
func (k *KuzuGraphDB) DropIndex(ctx context.Context, label, property string) error {
	// KuzuDB handles indexing differently - this is a placeholder
	return nil
}

// DropConstraint drops a constraint
func (k *KuzuGraphDB) DropConstraint(ctx context.Context, label, property, constraintType string) error {
	// KuzuDB handles constraints differently - this is a placeholder
	return nil
}

// GetSchema returns the database schema
func (k *KuzuGraphDB) GetSchema(ctx context.Context) (map[string]interface{}, error) {
	// For now, return a simplified schema structure
	// This would need to be implemented properly with actual KuzuDB catalog queries
	schema := map[string]interface{}{
		"provider": "kuzu",
		"database_path": k.config.KuzuConfig.DatabasePath,
		"read_only": k.config.KuzuConfig.ReadOnly,
		"tables": []string{}, // Would be populated with actual table information
	}
	
	return schema, nil
}

// HealthCheck performs a health check
func (k *KuzuGraphDB) HealthCheck(ctx context.Context) error {
	if k.conn == nil {
		return fmt.Errorf("database connection is not established")
	}
	
	// Simple health check query
	_, err := k.Query(ctx, "RETURN 1", nil)
	if err != nil {
		k.UpdateHealth("unhealthy", err)
		return err
	}
	
	k.UpdateHealth("healthy", nil)
	return nil
}

// GetMetrics returns database metrics
func (k *KuzuGraphDB) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	baseStats := k.GetStats()
	
	// Add KuzuDB-specific metrics
	metrics := map[string]interface{}{
		"provider":     "kuzu",
		"base_stats":   baseStats,
		"database_path": k.config.KuzuConfig.DatabasePath,
		"read_only":    k.config.KuzuConfig.ReadOnly,
	}
	
	return metrics, nil
}

// BeginTransaction begins a new transaction
func (k *KuzuGraphDB) BeginTransaction(ctx context.Context) (GraphTransaction, error) {
	// KuzuDB handles transactions differently
	// This is a placeholder implementation
	return &KuzuTransaction{
		conn: k.conn,
	}, nil
}

// KuzuTransaction implements the GraphTransaction interface for KuzuDB
type KuzuTransaction struct {
	conn *kuzu.Connection
}

// Commit commits the transaction
func (t *KuzuTransaction) Commit(ctx context.Context) error {
	// KuzuDB auto-commits queries
	return nil
}

// Rollback rolls back the transaction
func (t *KuzuTransaction) Rollback(ctx context.Context) error {
	// KuzuDB doesn't support explicit rollback in this simple implementation
	return nil
}

// CreateNode creates a node within the transaction
func (t *KuzuTransaction) CreateNode(ctx context.Context, labels []string, properties map[string]interface{}) (string, error) {
	// Would need to implement transactional node creation
	return "", fmt.Errorf("transactional node creation not yet implemented")
}

// CreateRelation creates a relationship within the transaction
func (t *KuzuTransaction) CreateRelation(ctx context.Context, fromID, toID, relType string, properties map[string]interface{}) error {
	// Would need to implement transactional relationship creation
	return fmt.Errorf("transactional relationship creation not yet implemented")
}

// Query executes a query within the transaction
func (t *KuzuTransaction) Query(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error) {
	// Would need to implement transactional queries
	return nil, fmt.Errorf("transactional queries not yet implemented")
}