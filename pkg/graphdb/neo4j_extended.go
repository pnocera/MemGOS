package graphdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Extended Neo4j implementation for advanced graph operations

// CreateRelations creates multiple relationships in a single transaction
func (ng *Neo4jGraphDB) CreateRelations(ctx context.Context, relations []GraphRelation) error {
	if ng.IsClosed() {
		return fmt.Errorf("graph database is closed")
	}

	start := time.Now()
	var success bool
	defer func() {
		ng.RecordQuery(success, time.Since(start))
	}()

	session := ng.getSession(ctx, neo4j.AccessModeWrite)
	defer session.Close(ctx)

	tx, err := session.BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Close(ctx)

	for _, rel := range relations {
		query := fmt.Sprintf(`
			MATCH (a), (b)
			WHERE elementId(a) = $fromId AND elementId(b) = $toId
			CREATE (a)-[r:%s]->(b)
			SET r = $props
			RETURN r
		`, rel.Type)
		
		_, err := tx.Run(ctx, query, map[string]interface{}{
			"fromId": rel.FromID,
			"toId":   rel.ToID,
			"props":  rel.Properties,
		})
		
		if err != nil {
			return fmt.Errorf("failed to create relation: %w", err)
		}
		
		ng.RecordRelationCreation()
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	success = true
	return nil
}

// GetRelations retrieves relationships between nodes
func (ng *Neo4jGraphDB) GetRelations(ctx context.Context, fromID, toID, relType string) ([]GraphRelation, error) {
	if ng.IsClosed() {
		return nil, fmt.Errorf("graph database is closed")
	}

	var query string
	params := make(map[string]interface{})

	if fromID != "" && toID != "" {
		if relType != "" {
			query = fmt.Sprintf("MATCH (a)-[r:%s]->(b) WHERE elementId(a) = $fromId AND elementId(b) = $toId RETURN elementId(r) as id, r, type(r) as type", relType)
		} else {
			query = "MATCH (a)-[r]->(b) WHERE elementId(a) = $fromId AND elementId(b) = $toId RETURN elementId(r) as id, r, type(r) as type"
		}
		params["fromId"] = fromID
		params["toId"] = toID
	} else if fromID != "" {
		if relType != "" {
			query = fmt.Sprintf("MATCH (a)-[r:%s]->() WHERE elementId(a) = $fromId RETURN elementId(r) as id, r, type(r) as type", relType)
		} else {
			query = "MATCH (a)-[r]->() WHERE elementId(a) = $fromId RETURN elementId(r) as id, r, type(r) as type"
		}
		params["fromId"] = fromID
	} else if relType != "" {
		query = fmt.Sprintf("MATCH ()-[r:%s]->() RETURN elementId(r) as id, r, type(r) as type LIMIT 100", relType)
	} else {
		query = "MATCH ()-[r]->() RETURN elementId(r) as id, r, type(r) as type LIMIT 100"
	}

	results, err := ng.Query(ctx, query, params)
	if err != nil {
		return nil, err
	}

	relations := make([]GraphRelation, len(results))
	for i, result := range results {
		relations[i] = GraphRelation{
			ID:         result["id"].(string),
			Type:       result["type"].(string),
			Properties: result["r"].(map[string]interface{}),
			CreatedAt:  time.Now(), // Would extract from properties in real implementation
			UpdatedAt:  time.Now(),
		}
	}

	return relations, nil
}

// UpdateRelation updates a relationship's properties
func (ng *Neo4jGraphDB) UpdateRelation(ctx context.Context, id string, properties map[string]interface{}) error {
	if ng.IsClosed() {
		return fmt.Errorf("graph database is closed")
	}

	query := "MATCH ()-[r]->() WHERE elementId(r) = $id SET r += $props RETURN r"
	params := map[string]interface{}{
		"id":    id,
		"props": properties,
	}

	_, err := ng.Query(ctx, query, params)
	return err
}

// DeleteRelation deletes a specific relationship
func (ng *Neo4jGraphDB) DeleteRelation(ctx context.Context, id string) error {
	if ng.IsClosed() {
		return fmt.Errorf("graph database is closed")
	}

	query := "MATCH ()-[r]->() WHERE elementId(r) = $id DELETE r"
	params := map[string]interface{}{"id": id}

	_, err := ng.Query(ctx, query, params)
	return err
}

// GetNodesByLabel retrieves nodes by label with limit
func (ng *Neo4jGraphDB) GetNodesByLabel(ctx context.Context, label string, limit int) ([]GraphNode, error) {
	if ng.IsClosed() {
		return nil, fmt.Errorf("graph database is closed")
	}

	query := fmt.Sprintf("MATCH (n:%s) RETURN elementId(n) as id, n, labels(n) as labels LIMIT $limit", label)
	params := map[string]interface{}{"limit": limit}

	results, err := ng.Query(ctx, query, params)
	if err != nil {
		return nil, err
	}

	nodes := make([]GraphNode, len(results))
	for i, result := range results {
		nodes[i] = GraphNode{
			ID:         result["id"].(string),
			Labels:     result["labels"].([]string),
			Properties: result["n"].(map[string]interface{}),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
	}

	return nodes, nil
}

// Traverse performs graph traversal starting from a node
func (ng *Neo4jGraphDB) Traverse(ctx context.Context, startID string, options *GraphTraversalOptions) ([]GraphNode, error) {
	if ng.IsClosed() {
		return nil, fmt.Errorf("graph database is closed")
	}

	if options == nil {
		options = DefaultTraversalOptions()
	}

	// Build dynamic Cypher query based on options
	var query strings.Builder
	query.WriteString("MATCH (start) WHERE elementId(start) = $startId ")
	
	// Add traversal pattern
	switch options.Direction {
	case "IN":
		query.WriteString("MATCH (n)<-")
	case "OUT":
		query.WriteString("MATCH (n)-")
	default: // BOTH
		query.WriteString("MATCH (n)-")
	}

	// Add relationship type filters
	if len(options.RelationTypes) > 0 {
		relTypes := strings.Join(options.RelationTypes, "|")
		query.WriteString(fmt.Sprintf("[:%s*1..%d]-", relTypes, options.MaxDepth))
	} else {
		query.WriteString(fmt.Sprintf("[*1..%d]-", options.MaxDepth))
	}

	if options.Direction == "IN" {
		query.WriteString("(start) ")
	} else {
		query.WriteString(">(start) ")
	}

	// Add node label filters
	if len(options.NodeLabels) > 0 {
		labels := strings.Join(options.NodeLabels, ":")
		query.WriteString(fmt.Sprintf("WHERE n:%s ", labels))
	}

	query.WriteString("RETURN DISTINCT elementId(n) as id, n, labels(n) as labels ")

	// Add limit and skip
	if options.Skip > 0 {
		query.WriteString(fmt.Sprintf("SKIP %d ", options.Skip))
	}
	if options.Limit > 0 {
		query.WriteString(fmt.Sprintf("LIMIT %d", options.Limit))
	}

	params := map[string]interface{}{"startId": startID}

	results, err := ng.Query(ctx, query.String(), params)
	if err != nil {
		return nil, fmt.Errorf("traversal failed: %w", err)
	}

	nodes := make([]GraphNode, len(results))
	for i, result := range results {
		nodes[i] = GraphNode{
			ID:         result["id"].(string),
			Labels:     result["labels"].([]string),
			Properties: result["n"].(map[string]interface{}),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
	}

	return nodes, nil
}

// FindPaths finds paths between two nodes
func (ng *Neo4jGraphDB) FindPaths(ctx context.Context, startID, endID string, maxDepth int) ([]GraphPath, error) {
	if ng.IsClosed() {
		return nil, fmt.Errorf("graph database is closed")
	}

	query := fmt.Sprintf(`
		MATCH (start), (end)
		WHERE elementId(start) = $startId AND elementId(end) = $endId
		MATCH path = (start)-[*1..%d]-(end)
		RETURN path
		LIMIT 10
	`, maxDepth)

	params := map[string]interface{}{
		"startId": startID,
		"endId":   endID,
	}

	results, err := ng.Query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("path finding failed: %w", err)
	}

	// In a real implementation, would parse the path objects from Neo4j
	// This is a simplified version
	paths := make([]GraphPath, len(results))
	for i := range results {
		paths[i] = GraphPath{
			Nodes:         []GraphNode{}, // Would parse nodes from path
			Relationships: []GraphRelation{}, // Would parse relationships from path
			Length:        1, // Would calculate actual length
		}
	}

	return paths, nil
}

// RunAlgorithm executes graph algorithms
func (ng *Neo4jGraphDB) RunAlgorithm(ctx context.Context, options *GraphAlgorithmOptions) (map[string]interface{}, error) {
	if ng.IsClosed() {
		return nil, fmt.Errorf("graph database is closed")
	}

	var query string
	params := make(map[string]interface{})

	switch options.Algorithm {
	case AlgorithmShortestPath:
		query = `
			MATCH (start), (end)
			WHERE elementId(start) = $startNode AND elementId(end) = $endNode
			MATCH path = shortestPath((start)-[*]-(end))
			RETURN length(path) as distance, path
		`
		params["startNode"] = options.StartNode
		params["endNode"] = options.EndNode

	case AlgorithmPageRank:
		// Note: This requires APOC or GDS library in Neo4j
		query = "CALL gds.pageRank.stream('graph') YIELD nodeId, score RETURN nodeId, score ORDER BY score DESC LIMIT 10"

	case AlgorithmCentrality:
		// Betweenness centrality example
		query = "CALL gds.betweenness.stream('graph') YIELD nodeId, score RETURN nodeId, score ORDER BY score DESC LIMIT 10"

	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", options.Algorithm)
	}

	results, err := ng.Query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("algorithm execution failed: %w", err)
	}

	return map[string]interface{}{
		"algorithm": string(options.Algorithm),
		"results":   results,
		"count":     len(results),
	}, nil
}

// ExecuteBatch executes multiple operations in a single transaction
func (ng *Neo4jGraphDB) ExecuteBatch(ctx context.Context, operations []BatchOperation) error {
	if ng.IsClosed() {
		return fmt.Errorf("graph database is closed")
	}

	start := time.Now()
	var success bool
	defer func() {
		ng.RecordQuery(success, time.Since(start))
	}()

	session := ng.getSession(ctx, neo4j.AccessModeWrite)
	defer session.Close(ctx)

	tx, err := session.BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Close(ctx)

	for _, op := range operations {
		var query string
		var params map[string]interface{}

		switch op.OperationType {
		case "CREATE_NODE":
			labelStr := strings.Join(op.NodeLabels, ":")
			if labelStr != "" {
				labelStr = ":" + labelStr
			}
			query = fmt.Sprintf("CREATE (n%s) SET n = $props RETURN elementId(n) as id", labelStr)
			params = map[string]interface{}{"props": op.Properties}

		case "CREATE_RELATION":
			query = fmt.Sprintf(`
				MATCH (a), (b)
				WHERE elementId(a) = $fromId AND elementId(b) = $toId
				CREATE (a)-[r:%s]->(b)
				SET r = $props
			`, op.RelationType)
			params = map[string]interface{}{
				"fromId": op.FromID,
				"toId":   op.ToID,
				"props":  op.Properties,
			}

		case "UPDATE_NODE":
			query = "MATCH (n) WHERE elementId(n) = $id SET n += $props RETURN n"
			params = map[string]interface{}{
				"id":    op.NodeID,
				"props": op.Properties,
			}

		case "DELETE_NODE":
			query = "MATCH (n) WHERE elementId(n) = $id DETACH DELETE n"
			params = map[string]interface{}{"id": op.NodeID}

		default:
			return fmt.Errorf("unsupported operation type: %s", op.OperationType)
		}

		_, err := tx.Run(ctx, query, params)
		if err != nil {
			return fmt.Errorf("batch operation failed: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit batch transaction: %w", err)
	}

	success = true
	return nil
}

// Schema management methods

// CreateIndex creates an index on a node property
func (ng *Neo4jGraphDB) CreateIndex(ctx context.Context, label, property string) error {
	if ng.IsClosed() {
		return fmt.Errorf("graph database is closed")
	}

	query := fmt.Sprintf("CREATE INDEX IF NOT EXISTS FOR (n:%s) ON (n.%s)", label, property)
	_, err := ng.Query(ctx, query, nil)
	return err
}

// CreateConstraint creates a constraint on a node property
func (ng *Neo4jGraphDB) CreateConstraint(ctx context.Context, label, property, constraintType string) error {
	if ng.IsClosed() {
		return fmt.Errorf("graph database is closed")
	}

	var query string
	switch constraintType {
	case "UNIQUE":
		query = fmt.Sprintf("CREATE CONSTRAINT IF NOT EXISTS FOR (n:%s) REQUIRE n.%s IS UNIQUE", label, property)
	case "EXISTS":
		query = fmt.Sprintf("CREATE CONSTRAINT IF NOT EXISTS FOR (n:%s) REQUIRE n.%s IS NOT NULL", label, property)
	default:
		return fmt.Errorf("unsupported constraint type: %s", constraintType)
	}

	_, err := ng.Query(ctx, query, nil)
	return err
}

// DropIndex drops an index
func (ng *Neo4jGraphDB) DropIndex(ctx context.Context, label, property string) error {
	if ng.IsClosed() {
		return fmt.Errorf("graph database is closed")
	}

	// Note: In Neo4j 4.x+, you need to use the index name
	query := fmt.Sprintf("DROP INDEX ON :%s(%s)", label, property)
	_, err := ng.Query(ctx, query, nil)
	return err
}

// DropConstraint drops a constraint
func (ng *Neo4jGraphDB) DropConstraint(ctx context.Context, label, property, constraintType string) error {
	if ng.IsClosed() {
		return fmt.Errorf("graph database is closed")
	}

	// Note: Actual implementation would need to find constraint by name
	query := fmt.Sprintf("DROP CONSTRAINT ON (n:%s) ASSERT n.%s IS UNIQUE", label, property)
	_, err := ng.Query(ctx, query, nil)
	return err
}

// GetSchema returns database schema information
func (ng *Neo4jGraphDB) GetSchema(ctx context.Context) (map[string]interface{}, error) {
	if ng.IsClosed() {
		return nil, fmt.Errorf("graph database is closed")
	}

	schema := make(map[string]interface{})

	// Get all labels
	labelsResult, err := ng.Query(ctx, "CALL db.labels()", nil)
	if err == nil {
		schema["labels"] = labelsResult
	}

	// Get all relationship types
	relTypesResult, err := ng.Query(ctx, "CALL db.relationshipTypes()", nil)
	if err == nil {
		schema["relationship_types"] = relTypesResult
	}

	// Get all indexes
	indexesResult, err := ng.Query(ctx, "SHOW INDEXES", nil)
	if err == nil {
		schema["indexes"] = indexesResult
	}

	// Get all constraints
	constraintsResult, err := ng.Query(ctx, "SHOW CONSTRAINTS", nil)
	if err == nil {
		schema["constraints"] = constraintsResult
	}

	return schema, nil
}

// BeginTransaction starts a new transaction
func (ng *Neo4jGraphDB) BeginTransaction(ctx context.Context) (GraphTransaction, error) {
	if ng.IsClosed() {
		return nil, fmt.Errorf("graph database is closed")
	}

	session := ng.getSession(ctx, neo4j.AccessModeWrite)
	tx, err := session.BeginTransaction(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &Neo4jTransaction{
		tx:      tx,
		session: session,
		ng:      ng,
	}, nil
}

// Neo4jTransaction implements GraphTransaction interface
type Neo4jTransaction struct {
	tx      neo4j.ExplicitTransaction
	session neo4j.SessionWithContext
	ng      *Neo4jGraphDB
}

// Commit commits the transaction
func (nt *Neo4jTransaction) Commit(ctx context.Context) error {
	defer nt.session.Close(ctx)
	return nt.tx.Commit(ctx)
}

// Rollback rolls back the transaction
func (nt *Neo4jTransaction) Rollback(ctx context.Context) error {
	defer nt.session.Close(ctx)
	return nt.tx.Rollback(ctx)
}

// CreateNode creates a node within the transaction
func (nt *Neo4jTransaction) CreateNode(ctx context.Context, labels []string, properties map[string]interface{}) (string, error) {
	labelStr := strings.Join(labels, ":")
	if labelStr != "" {
		labelStr = ":" + labelStr
	}
	
	query := fmt.Sprintf("CREATE (n%s) SET n = $props RETURN elementId(n) as id", labelStr)
	
	result, err := nt.tx.Run(ctx, query, map[string]interface{}{
		"props": properties,
	})
	if err != nil {
		return "", err
	}

	if result.Next(ctx) {
		record := result.Record()
		if id, ok := record.Get("id"); ok {
			nt.ng.RecordNodeCreation()
			return id.(string), nil
		}
	}
	
	return "", fmt.Errorf("failed to get node ID")
}

// CreateRelation creates a relationship within the transaction
func (nt *Neo4jTransaction) CreateRelation(ctx context.Context, fromID, toID, relType string, properties map[string]interface{}) error {
	query := fmt.Sprintf(`
		MATCH (a), (b)
		WHERE elementId(a) = $fromId AND elementId(b) = $toId
		CREATE (a)-[r:%s]->(b)
		SET r = $props
		RETURN r
	`, relType)
	
	_, err := nt.tx.Run(ctx, query, map[string]interface{}{
		"fromId": fromID,
		"toId":   toID,
		"props":  properties,
	})
	
	if err == nil {
		nt.ng.RecordRelationCreation()
	}
	
	return err
}

// Query executes a query within the transaction
func (nt *Neo4jTransaction) Query(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error) {
	result, err := nt.tx.Run(ctx, query, params)
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for result.Next(ctx) {
		record := result.Record()
		resultMap := make(map[string]interface{})
		
		for _, key := range record.Keys {
			if value, ok := record.Get(key); ok {
				resultMap[key] = value
			}
		}
		
		results = append(results, resultMap)
	}
	
	return results, result.Err()
}
