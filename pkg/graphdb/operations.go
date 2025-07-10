package graphdb

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
)

// GraphOperations provides high-level graph operations and query builders
type GraphOperations struct {
	db      ExtendedGraphDB
	logger  interfaces.Logger
	metrics interfaces.Metrics
	mu      sync.RWMutex
}

// NewGraphOperations creates a new graph operations instance
func NewGraphOperations(db ExtendedGraphDB, logger interfaces.Logger, metrics interfaces.Metrics) *GraphOperations {
	return &GraphOperations{
		db:      db,
		logger:  logger,
		metrics: metrics,
	}
}

// QueryBuilder helps build complex Cypher queries
type QueryBuilder struct {
	parts   []string
	params  map[string]interface{}
	clauses map[string][]string
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		parts:   make([]string, 0),
		params:  make(map[string]interface{}),
		clauses: make(map[string][]string),
	}
}

// Match adds a MATCH clause
func (qb *QueryBuilder) Match(pattern string) *QueryBuilder {
	qb.addClause("MATCH", pattern)
	return qb
}

// OptionalMatch adds an OPTIONAL MATCH clause
func (qb *QueryBuilder) OptionalMatch(pattern string) *QueryBuilder {
	qb.addClause("OPTIONAL MATCH", pattern)
	return qb
}

// Where adds a WHERE clause
func (qb *QueryBuilder) Where(condition string) *QueryBuilder {
	qb.addClause("WHERE", condition)
	return qb
}

// Create adds a CREATE clause
func (qb *QueryBuilder) Create(pattern string) *QueryBuilder {
	qb.addClause("CREATE", pattern)
	return qb
}

// Set adds a SET clause
func (qb *QueryBuilder) Set(assignment string) *QueryBuilder {
	qb.addClause("SET", assignment)
	return qb
}

// Return adds a RETURN clause
func (qb *QueryBuilder) Return(expression string) *QueryBuilder {
	qb.addClause("RETURN", expression)
	return qb
}

// OrderBy adds an ORDER BY clause
func (qb *QueryBuilder) OrderBy(expression string) *QueryBuilder {
	qb.addClause("ORDER BY", expression)
	return qb
}

// Limit adds a LIMIT clause
func (qb *QueryBuilder) Limit(count int) *QueryBuilder {
	qb.addClause("LIMIT", fmt.Sprintf("%d", count))
	return qb
}

// Skip adds a SKIP clause
func (qb *QueryBuilder) Skip(count int) *QueryBuilder {
	qb.addClause("SKIP", fmt.Sprintf("%d", count))
	return qb
}

// With adds a WITH clause
func (qb *QueryBuilder) With(expression string) *QueryBuilder {
	qb.addClause("WITH", expression)
	return qb
}

// Union adds a UNION clause
func (qb *QueryBuilder) Union() *QueryBuilder {
	qb.addClause("UNION", "")
	return qb
}

// UnionAll adds a UNION ALL clause
func (qb *QueryBuilder) UnionAll() *QueryBuilder {
	qb.addClause("UNION ALL", "")
	return qb
}

// AddParameter adds a parameter to the query
func (qb *QueryBuilder) AddParameter(key string, value interface{}) *QueryBuilder {
	qb.params[key] = value
	return qb
}

// AddParameters adds multiple parameters to the query
func (qb *QueryBuilder) AddParameters(params map[string]interface{}) *QueryBuilder {
	for k, v := range params {
		qb.params[k] = v
	}
	return qb
}

// addClause adds a clause to the query
func (qb *QueryBuilder) addClause(clauseType, content string) {
	if _, exists := qb.clauses[clauseType]; !exists {
		qb.clauses[clauseType] = make([]string, 0)
	}
	qb.clauses[clauseType] = append(qb.clauses[clauseType], content)
}

// Build builds the final query string
func (qb *QueryBuilder) Build() (string, map[string]interface{}) {
	var query strings.Builder
	
	// Define clause order
	clauseOrder := []string{"MATCH", "OPTIONAL MATCH", "WHERE", "CREATE", "SET", "DELETE", "REMOVE", "WITH", "RETURN", "ORDER BY", "SKIP", "LIMIT", "UNION", "UNION ALL"}
	
	for _, clauseType := range clauseOrder {
		if clauses, exists := qb.clauses[clauseType]; exists {
			for i, clause := range clauses {
				if i == 0 {
					if query.Len() > 0 {
						query.WriteString(" ")
					}
					query.WriteString(clauseType)
					if clause != "" {
						query.WriteString(" ")
						query.WriteString(clause)
					}
				} else {
					if clauseType == "WHERE" {
						query.WriteString(" AND ")
					} else if clauseType == "UNION" || clauseType == "UNION ALL" {
						query.WriteString(" ")
						query.WriteString(clauseType)
						if clause != "" {
							query.WriteString(" ")
						}
					} else {
						query.WriteString(", ")
					}
					if clause != "" {
						query.WriteString(clause)
					}
				}
			}
		}
	}
	
	return query.String(), qb.params
}

// Execute executes the built query
func (qb *QueryBuilder) Execute(ctx context.Context, db ExtendedGraphDB) ([]map[string]interface{}, error) {
	query, params := qb.Build()
	return db.Query(ctx, query, params)
}

// Graph algorithms and analytics

// FindShortestPath finds the shortest path between two nodes
func (gops *GraphOperations) FindShortestPath(ctx context.Context, startID, endID string, relationTypes []string) ([]GraphPath, error) {
	gops.mu.RLock()
	defer gops.mu.RUnlock()

	qb := NewQueryBuilder()
	qb.Match("(start), (end)")
	qb.Where("elementId(start) = $startId AND elementId(end) = $endId")
	
	if len(relationTypes) > 0 {
		relTypeStr := strings.Join(relationTypes, "|")
		qb.Match(fmt.Sprintf("path = shortestPath((start)-[:%s*]-(end))", relTypeStr))
	} else {
		qb.Match("path = shortestPath((start)-[*]-(end))")
	}
	
	qb.Return("path, length(path) as pathLength")
	qb.AddParameter("startId", startID)
	qb.AddParameter("endId", endID)

	results, err := qb.Execute(ctx, gops.db)
	if err != nil {
		return nil, fmt.Errorf("shortest path query failed: %w", err)
	}

	// Convert results to GraphPath objects
	paths := make([]GraphPath, len(results))
	for i, result := range results {
		paths[i] = GraphPath{
			Length: int(result["pathLength"].(int64)),
			// In real implementation, would parse the path object
			Nodes:         []GraphNode{},
			Relationships: []GraphRelation{},
		}
	}

	return paths, nil
}

// FindAllPaths finds all paths between two nodes within max depth
func (gops *GraphOperations) FindAllPaths(ctx context.Context, startID, endID string, maxDepth int, relationTypes []string) ([]GraphPath, error) {
	gops.mu.RLock()
	defer gops.mu.RUnlock()

	qb := NewQueryBuilder()
	qb.Match("(start), (end)")
	qb.Where("elementId(start) = $startId AND elementId(end) = $endId")
	
	if len(relationTypes) > 0 {
		relTypeStr := strings.Join(relationTypes, "|")
		qb.Match(fmt.Sprintf("path = (start)-[:%s*1..%d]-(end)", relTypeStr, maxDepth))
	} else {
		qb.Match(fmt.Sprintf("path = (start)-[*1..%d]-(end)", maxDepth))
	}
	
	qb.Return("path, length(path) as pathLength")
	qb.Limit(100) // Limit to prevent excessive results
	qb.AddParameter("startId", startID)
	qb.AddParameter("endId", endID)

	results, err := qb.Execute(ctx, gops.db)
	if err != nil {
		return nil, fmt.Errorf("all paths query failed: %w", err)
	}

	paths := make([]GraphPath, len(results))
	for i, result := range results {
		paths[i] = GraphPath{
			Length: int(result["pathLength"].(int64)),
			Nodes:  []GraphNode{},
			Relationships: []GraphRelation{},
		}
	}

	return paths, nil
}

// GetNeighbors gets all neighbors of a node
func (gops *GraphOperations) GetNeighbors(ctx context.Context, nodeID string, direction string, relationTypes []string, depth int) ([]GraphNode, error) {
	gops.mu.RLock()
	defer gops.mu.RUnlock()

	qb := NewQueryBuilder()
	qb.Match("(start) WHERE elementId(start) = $nodeId")
	
	var pattern string
	switch direction {
	case "IN":
		if len(relationTypes) > 0 {
			relTypeStr := strings.Join(relationTypes, "|")
			pattern = fmt.Sprintf("(n)<-[:%s*1..%d]-(start)", relTypeStr, depth)
		} else {
			pattern = fmt.Sprintf("(n)<-[*1..%d]-(start)", depth)
		}
	case "OUT":
		if len(relationTypes) > 0 {
			relTypeStr := strings.Join(relationTypes, "|")
			pattern = fmt.Sprintf("(start)-[:%s*1..%d]->(n)", relTypeStr, depth)
		} else {
			pattern = fmt.Sprintf("(start)-[*1..%d]->(n)", depth)
		}
	default: // BOTH
		if len(relationTypes) > 0 {
			relTypeStr := strings.Join(relationTypes, "|")
			pattern = fmt.Sprintf("(n)-[:%s*1..%d]-(start)", relTypeStr, depth)
		} else {
			pattern = fmt.Sprintf("(n)-[*1..%d]-(start)", depth)
		}
	}
	
	qb.Match(pattern)
	qb.Return("DISTINCT elementId(n) as id, n, labels(n) as labels")
	qb.Limit(1000)
	qb.AddParameter("nodeId", nodeID)

	results, err := qb.Execute(ctx, gops.db)
	if err != nil {
		return nil, fmt.Errorf("neighbors query failed: %w", err)
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

// CalculateDegree calculates the degree (number of connections) of a node
func (gops *GraphOperations) CalculateDegree(ctx context.Context, nodeID string, direction string) (int, error) {
	gops.mu.RLock()
	defer gops.mu.RUnlock()

	qb := NewQueryBuilder()
	qb.Match("(n) WHERE elementId(n) = $nodeId")
	
	switch direction {
	case "IN":
		qb.OptionalMatch("(n)<-[r]-(m)")
	case "OUT":
		qb.OptionalMatch("(n)-[r]->(m)")
	default: // BOTH
		qb.OptionalMatch("(n)-[r]-(m)")
	}
	
	qb.Return("count(r) as degree")
	qb.AddParameter("nodeId", nodeID)

	results, err := qb.Execute(ctx, gops.db)
	if err != nil {
		return 0, fmt.Errorf("degree calculation failed: %w", err)
	}

	if len(results) > 0 {
		return int(results[0]["degree"].(int64)), nil
	}

	return 0, nil
}

// FindCommunities detects communities in the graph (simplified version)
func (gops *GraphOperations) FindCommunities(ctx context.Context, algorithm string) (map[string]interface{}, error) {
	gops.mu.RLock()
	defer gops.mu.RUnlock()

	// This would typically use Neo4j GDS library
	// Simplified implementation for demonstration
	qb := NewQueryBuilder()
	
	switch algorithm {
	case "louvain":
		// Would use: CALL gds.louvain.stream('graph')
		qb.Match("(n)")
		qb.Return("elementId(n) as nodeId, labels(n) as labels")
		qb.Limit(100)
		
	case "label_propagation":
		// Would use: CALL gds.labelPropagation.stream('graph')
		qb.Match("(n)")
		qb.Return("elementId(n) as nodeId, labels(n) as labels")
		qb.Limit(100)
		
	default:
		return nil, fmt.Errorf("unsupported community detection algorithm: %s", algorithm)
	}

	results, err := qb.Execute(ctx, gops.db)
	if err != nil {
		return nil, fmt.Errorf("community detection failed: %w", err)
	}

	return map[string]interface{}{
		"algorithm": algorithm,
		"communities": results,
		"count": len(results),
	}, nil
}

// AnalyzeGraphStructure provides basic graph structure analysis
func (gops *GraphOperations) AnalyzeGraphStructure(ctx context.Context) (map[string]interface{}, error) {
	gops.mu.RLock()
	defer gops.mu.RUnlock()

	analysis := make(map[string]interface{})

	// Count nodes
	nodeCountQuery := NewQueryBuilder()
	nodeCountQuery.Match("(n)")
	nodeCountQuery.Return("count(n) as nodeCount")
	
	nodeResults, err := nodeCountQuery.Execute(ctx, gops.db)
	if err == nil && len(nodeResults) > 0 {
		analysis["node_count"] = nodeResults[0]["nodeCount"]
	}

	// Count relationships
	relCountQuery := NewQueryBuilder()
	relCountQuery.Match("()-[r]-()")
	relCountQuery.Return("count(r) as relCount")
	
	relResults, err := relCountQuery.Execute(ctx, gops.db)
	if err == nil && len(relResults) > 0 {
		analysis["relationship_count"] = relResults[0]["relCount"]
	}

	// Get label distribution
	labelQuery := NewQueryBuilder()
	labelQuery.Match("(n)")
	labelQuery.Return("labels(n) as nodeLabels, count(*) as count")
	labelQuery.OrderBy("count DESC")
	labelQuery.Limit(20)
	
	labelResults, err := labelQuery.Execute(ctx, gops.db)
	if err == nil {
		analysis["label_distribution"] = labelResults
	}

	// Get relationship type distribution
	relTypeQuery := NewQueryBuilder()
	relTypeQuery.Match("()-[r]-()")
	relTypeQuery.Return("type(r) as relType, count(*) as count")
	relTypeQuery.OrderBy("count DESC")
	relTypeQuery.Limit(20)
	
	relTypeResults, err := relTypeQuery.Execute(ctx, gops.db)
	if err == nil {
		analysis["relationship_type_distribution"] = relTypeResults
	}

	// Calculate average degree
	degreeQuery := NewQueryBuilder()
	degreeQuery.Match("(n)")
	degreeQuery.OptionalMatch("(n)-[r]-(m)")
	degreeQuery.Return("n, count(r) as degree")
	degreeQuery.With("avg(degree) as avgDegree, max(degree) as maxDegree, min(degree) as minDegree")
	degreeQuery.Return("avgDegree, maxDegree, minDegree")
	
	degreeResults, err := degreeQuery.Execute(ctx, gops.db)
	if err == nil && len(degreeResults) > 0 {
		analysis["degree_statistics"] = degreeResults[0]
	}

	analysis["timestamp"] = time.Now()
	return analysis, nil
}

// BulkNodeCreation creates multiple nodes efficiently
func (gops *GraphOperations) BulkNodeCreation(ctx context.Context, nodes []GraphNode, batchSize int) error {
	gops.mu.RLock()
	defer gops.mu.RUnlock()

	if batchSize <= 0 {
		batchSize = 1000
	}

	for i := 0; i < len(nodes); i += batchSize {
		end := i + batchSize
		if end > len(nodes) {
			end = len(nodes)
		}
		
		batch := nodes[i:end]
		if _, err := gops.db.CreateNodes(ctx, batch); err != nil {
			return fmt.Errorf("bulk node creation failed at batch %d: %w", i/batchSize, err)
		}
		
		if gops.metrics != nil {
			gops.metrics.Counter("graphdb_bulk_nodes_created", float64(len(batch)), nil)
		}
	}

	return nil
}

// BulkRelationshipCreation creates multiple relationships efficiently
func (gops *GraphOperations) BulkRelationshipCreation(ctx context.Context, relations []GraphRelation, batchSize int) error {
	gops.mu.RLock()
	defer gops.mu.RUnlock()

	if batchSize <= 0 {
		batchSize = 1000
	}

	for i := 0; i < len(relations); i += batchSize {
		end := i + batchSize
		if end > len(relations) {
			end = len(relations)
		}
		
		batch := relations[i:end]
		if err := gops.db.CreateRelations(ctx, batch); err != nil {
			return fmt.Errorf("bulk relationship creation failed at batch %d: %w", i/batchSize, err)
		}
		
		if gops.metrics != nil {
			gops.metrics.Counter("graphdb_bulk_relations_created", float64(len(batch)), nil)
		}
	}

	return nil
}

// ImportFromCSV imports graph data from CSV format (simplified)
func (gops *GraphOperations) ImportFromCSV(ctx context.Context, csvData map[string]interface{}) error {
	gops.mu.RLock()
	defer gops.mu.RUnlock()

	// This is a simplified implementation
	// In practice, would use LOAD CSV or APOC procedures
	
	if nodes, ok := csvData["nodes"].([]GraphNode); ok {
		if err := gops.BulkNodeCreation(ctx, nodes, 1000); err != nil {
			return fmt.Errorf("failed to import nodes: %w", err)
		}
	}

	if relations, ok := csvData["relationships"].([]GraphRelation); ok {
		if err := gops.BulkRelationshipCreation(ctx, relations, 1000); err != nil {
			return fmt.Errorf("failed to import relationships: %w", err)
		}
	}

	return nil
}

// ExportToFormat exports graph data to various formats
func (gops *GraphOperations) ExportToFormat(ctx context.Context, format string, options map[string]interface{}) (map[string]interface{}, error) {
	gops.mu.RLock()
	defer gops.mu.RUnlock()

	switch format {
	case "json":
		return gops.exportToJSON(ctx, options)
	case "gexf":
		return gops.exportToGEXF(ctx, options)
	case "graphml":
		return gops.exportToGraphML(ctx, options)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportToJSON exports graph data to JSON format
func (gops *GraphOperations) exportToJSON(ctx context.Context, options map[string]interface{}) (map[string]interface{}, error) {
	// Get all nodes
	nodeQuery := NewQueryBuilder()
	nodeQuery.Match("(n)")
	nodeQuery.Return("elementId(n) as id, labels(n) as labels, properties(n) as properties")
	
	if limit, ok := options["limit"].(int); ok {
		nodeQuery.Limit(limit)
	} else {
		nodeQuery.Limit(10000)
	}

	nodes, err := nodeQuery.Execute(ctx, gops.db)
	if err != nil {
		return nil, fmt.Errorf("failed to export nodes: %w", err)
	}

	// Get all relationships
	relQuery := NewQueryBuilder()
	relQuery.Match("(a)-[r]->(b)")
	relQuery.Return("elementId(r) as id, elementId(a) as source, elementId(b) as target, type(r) as type, properties(r) as properties")
	
	if limit, ok := options["limit"].(int); ok {
		relQuery.Limit(limit)
	} else {
		relQuery.Limit(10000)
	}

	relationships, err := relQuery.Execute(ctx, gops.db)
	if err != nil {
		return nil, fmt.Errorf("failed to export relationships: %w", err)
	}

	return map[string]interface{}{
		"nodes":         nodes,
		"relationships": relationships,
		"format":        "json",
		"timestamp":     time.Now(),
	}, nil
}

// exportToGEXF exports graph data to GEXF format (simplified)
func (gops *GraphOperations) exportToGEXF(ctx context.Context, options map[string]interface{}) (map[string]interface{}, error) {
	// Simplified GEXF export - in practice would generate actual GEXF XML
	return map[string]interface{}{
		"format": "gexf",
		"message": "GEXF export not fully implemented",
	}, nil
}

// exportToGraphML exports graph data to GraphML format (simplified)
func (gops *GraphOperations) exportToGraphML(ctx context.Context, options map[string]interface{}) (map[string]interface{}, error) {
	// Simplified GraphML export - in practice would generate actual GraphML XML
	return map[string]interface{}{
		"format": "graphml",
		"message": "GraphML export not fully implemented",
	}, nil
}
