package graphdb

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/avast/retry-go"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/config"

	"github.com/memtensor/memgos/pkg/errors"
	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// Neo4jGraphDB implements ExtendedGraphDB interface for Neo4j
type Neo4jGraphDB struct {
	*BaseGraphDB
	driver   neo4j.DriverWithContext
	session  neo4j.SessionWithContext
	sessionMu sync.RWMutex
}

// NewNeo4jGraphDB creates a new Neo4j graph database instance
func NewNeo4jGraphDB(config *GraphDBConfig, logger interfaces.Logger, metrics interfaces.Metrics) (*Neo4jGraphDB, error) {
	if config == nil {
		config = DefaultGraphDBConfig()
	}
	
	ng := &Neo4jGraphDB{
		BaseGraphDB: NewBaseGraphDB(config, logger, metrics),
	}
	
	if err := ng.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to Neo4j: %w", err)
	}
	
	return ng, nil
}

// connect establishes connection to Neo4j database
func (ng *Neo4jGraphDB) connect() error {
	start := time.Now()
	defer func() {
		ng.stats.mu.Lock()
		ng.stats.connectionTime = time.Since(start)
		ng.stats.mu.Unlock()
	}()

	// Configure Neo4j driver
	auth := neo4j.BasicAuth(ng.config.Username, ng.config.Password, "")
	
	configFunc := func(conf *config.Config) {
		conf.MaxConnectionPoolSize = ng.config.MaxConnPool
		conf.ConnectionAcquisitionTimeout = ng.config.ConnTimeout
		conf.SocketConnectTimeout = ng.config.ConnTimeout
		conf.SocketKeepalive = true
		
		if ng.config.Logging {
			conf.Log = neo4j.ConsoleLogger(neo4j.DEBUG)
		}
		
		// Configure encryption based on SSL mode
		// Note: Neo4j v5 uses different encryption configuration
		// This would be configured differently in the actual driver
	}

	driver, err := neo4j.NewDriverWithContext(ng.config.URI, auth, configFunc)
	if err != nil {
		ng.UpdateHealth("unhealthy", err)
		return fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	ng.driver = driver
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), ng.config.ConnTimeout)
	defer cancel()
	
	if err := ng.driver.VerifyConnectivity(ctx); err != nil {
		ng.UpdateHealth("unhealthy", err)
		return fmt.Errorf("failed to verify Neo4j connectivity: %w", err)
	}

	ng.UpdateHealth("healthy", nil)
	
	if ng.logger != nil {
		ng.logger.Info("Connected to Neo4j database", map[string]interface{}{
			"uri":      ng.config.URI,
			"database": ng.config.Database,
			"duration": time.Since(start).String(),
		})
	}
	
	return nil
}

// getSession returns a database session
func (ng *Neo4jGraphDB) getSession(ctx context.Context, accessMode neo4j.AccessMode) neo4j.SessionWithContext {
	ng.sessionMu.Lock()
	defer ng.sessionMu.Unlock()
	
	if ng.session != nil {
		ng.session.Close(ctx)
	}
	
	sessionConfig := neo4j.SessionConfig{
		AccessMode:   accessMode,
		DatabaseName: ng.config.Database,
	}
	
	ng.session = ng.driver.NewSession(ctx, sessionConfig)
	return ng.session
}

// CreateNode creates a new node in the graph
func (ng *Neo4jGraphDB) CreateNode(ctx context.Context, labels []string, properties map[string]interface{}) (string, error) {
	if ng.IsClosed() {
		return "", errors.NewMemGOSError(types.ErrorTypeInternal, errors.ErrCodeDatabaseError, "graph database is closed")
	}

	start := time.Now()
	var success bool
	defer func() {
		ng.RecordQuery(success, time.Since(start))
	}()

	err := retry.Do(
		func() error {
			session := ng.getSession(ctx, neo4j.AccessModeWrite)
			defer session.Close(ctx)

			// Build Cypher query
			labelStr := strings.Join(labels, ":")
			if labelStr != "" {
				labelStr = ":" + labelStr
			}
			
			query := fmt.Sprintf("CREATE (n%s) SET n = $props RETURN elementId(n) as id", labelStr)
			
			result, err := session.Run(ctx, query, map[string]interface{}{
				"props": properties,
			})
			if err != nil {
				return err
			}

			if result.Next(ctx) {
				record := result.Record()
				if _, ok := record.Get("id"); ok {
					ng.RecordNodeCreation()
					success = true
					return nil
				}
			}
			
			return fmt.Errorf("failed to get node ID")
		},
		retry.Attempts(uint(ng.config.RetryAttempts)),
		retry.Delay(ng.config.RetryDelay),
		retry.Context(ctx),
	)

	if err != nil {
		if ng.logger != nil {
			ng.logger.Error("Failed to create node", err, map[string]interface{}{
				"labels":     labels,
				"properties": properties,
			})
		}
		return "", fmt.Errorf("failed to create node: %w", err)
	}

	return "node-id", nil // In real implementation, return actual ID
}

// CreateNodes creates multiple nodes in a single transaction
func (ng *Neo4jGraphDB) CreateNodes(ctx context.Context, nodes []GraphNode) ([]string, error) {
	if ng.IsClosed() {
		return nil, errors.NewMemGOSError(types.ErrorTypeInternal, errors.ErrCodeDatabaseError, "graph database is closed")
	}

	start := time.Now()
	var success bool
	defer func() {
		ng.RecordQuery(success, time.Since(start))
	}()

	var nodeIDs []string
	
	err := retry.Do(
		func() error {
			session := ng.getSession(ctx, neo4j.AccessModeWrite)
			defer session.Close(ctx)

			tx, err := session.BeginTransaction(ctx)
			if err != nil {
				return err
			}
			defer tx.Close(ctx)

			nodeIDs = make([]string, len(nodes))
			
			for i, node := range nodes {
				labelStr := strings.Join(node.Labels, ":")
				if labelStr != "" {
					labelStr = ":" + labelStr
				}
				
				query := fmt.Sprintf("CREATE (n%s) SET n = $props RETURN elementId(n) as id", labelStr)
				
				result, err := tx.Run(ctx, query, map[string]interface{}{
					"props": node.Properties,
				})
				if err != nil {
					return err
				}
				
				if result.Next(ctx) {
					record := result.Record()
					if id, ok := record.Get("id"); ok {
						nodeIDs[i] = id.(string)
						ng.RecordNodeCreation()
					}
				}
			}
			
			if err := tx.Commit(ctx); err != nil {
				return err
			}
			
			success = true
			return nil
		},
		retry.Attempts(uint(ng.config.RetryAttempts)),
		retry.Delay(ng.config.RetryDelay),
		retry.Context(ctx),
	)

	if err != nil {
		if ng.logger != nil {
			ng.logger.Error("Failed to create nodes", err, map[string]interface{}{
				"count": len(nodes),
			})
		}
		return nil, fmt.Errorf("failed to create nodes: %w", err)
	}

	return nodeIDs, nil
}

// CreateRelation creates a relationship between two nodes
func (ng *Neo4jGraphDB) CreateRelation(ctx context.Context, fromID, toID, relType string, properties map[string]interface{}) error {
	if ng.IsClosed() {
		return errors.NewMemGOSError(types.ErrorTypeInternal, errors.ErrCodeDatabaseError, "graph database is closed")
	}

	start := time.Now()
	var success bool
	defer func() {
		ng.RecordQuery(success, time.Since(start))
	}()

	err := retry.Do(
		func() error {
			session := ng.getSession(ctx, neo4j.AccessModeWrite)
			defer session.Close(ctx)

			query := fmt.Sprintf(`
				MATCH (a), (b)
				WHERE elementId(a) = $fromId AND elementId(b) = $toId
				CREATE (a)-[r:%s]->(b)
				SET r = $props
				RETURN r
			`, relType)
			
			_, err := session.Run(ctx, query, map[string]interface{}{
				"fromId": fromID,
				"toId":   toID,
				"props":  properties,
			})
			
			if err != nil {
				return err
			}
			
			ng.RecordRelationCreation()
			success = true
			return nil
		},
		retry.Attempts(uint(ng.config.RetryAttempts)),
		retry.Delay(ng.config.RetryDelay),
		retry.Context(ctx),
	)

	if err != nil {
		if ng.logger != nil {
			ng.logger.Error("Failed to create relation", err, map[string]interface{}{
				"from_id":    fromID,
				"to_id":      toID,
				"type":       relType,
				"properties": properties,
			})
		}
		return fmt.Errorf("failed to create relation: %w", err)
	}

	return nil
}

// Query executes a Cypher query and returns results
func (ng *Neo4jGraphDB) Query(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error) {
	if ng.IsClosed() {
		return nil, errors.NewMemGOSError(types.ErrorTypeInternal, errors.ErrCodeDatabaseError, "graph database is closed")
	}

	start := time.Now()
	var success bool
	defer func() {
		ng.RecordQuery(success, time.Since(start))
	}()

	var results []map[string]interface{}
	
	err := retry.Do(
		func() error {
			session := ng.getSession(ctx, neo4j.AccessModeRead)
			defer session.Close(ctx)

			result, err := session.Run(ctx, query, params)
			if err != nil {
				return err
			}

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
			
			if err := result.Err(); err != nil {
				return err
			}
			
			success = true
			return nil
		},
		retry.Attempts(uint(ng.config.RetryAttempts)),
		retry.Delay(ng.config.RetryDelay),
		retry.Context(ctx),
	)

	if err != nil {
		if ng.logger != nil {
			ng.logger.Error("Failed to execute query", err, map[string]interface{}{
				"query":  query,
				"params": params,
			})
		}
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	if ng.logger != nil && ng.config.Logging {
		ng.logger.Info("Query executed successfully", map[string]interface{}{
			"query":        query,
			"result_count": len(results),
			"duration":     time.Since(start).String(),
		})
	}

	return results, nil
}

// GetNode retrieves a node by ID
func (ng *Neo4jGraphDB) GetNode(ctx context.Context, id string) (map[string]interface{}, error) {
	if ng.IsClosed() {
		return nil, errors.NewMemGOSError(types.ErrorTypeInternal, errors.ErrCodeDatabaseError, "graph database is closed")
	}

	query := "MATCH (n) WHERE elementId(n) = $id RETURN n, labels(n) as labels"
	params := map[string]interface{}{"id": id}
	
	results, err := ng.Query(ctx, query, params)
	if err != nil {
		return nil, err
	}
	
	if len(results) == 0 {
		return nil, errors.NewMemGOSError(types.ErrorTypeNotFound, errors.ErrCodeNotFound, "node not found").WithDetail("id", id)
	}
	
	return results[0], nil
}

// GetNodes retrieves multiple nodes by their IDs
func (ng *Neo4jGraphDB) GetNodes(ctx context.Context, ids []string) ([]GraphNode, error) {
	if ng.IsClosed() {
		return nil, errors.NewMemGOSError(types.ErrorTypeInternal, errors.ErrCodeDatabaseError, "graph database is closed")
	}

	query := "MATCH (n) WHERE elementId(n) IN $ids RETURN elementId(n) as id, n, labels(n) as labels"
	params := map[string]interface{}{"ids": ids}
	
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
			CreatedAt:  time.Now(), // Would extract from properties in real implementation
			UpdatedAt:  time.Now(),
		}
	}
	
	return nodes, nil
}

// UpdateNode updates a node's properties
func (ng *Neo4jGraphDB) UpdateNode(ctx context.Context, id string, properties map[string]interface{}) error {
	if ng.IsClosed() {
		return errors.NewMemGOSError(types.ErrorTypeInternal, errors.ErrCodeDatabaseError, "graph database is closed")
	}

	query := "MATCH (n) WHERE elementId(n) = $id SET n += $props RETURN n"
	params := map[string]interface{}{
		"id":    id,
		"props": properties,
	}
	
	_, err := ng.Query(ctx, query, params)
	return err
}

// DeleteNode deletes a node and all its relationships
func (ng *Neo4jGraphDB) DeleteNode(ctx context.Context, id string) error {
	if ng.IsClosed() {
		return errors.NewMemGOSError(types.ErrorTypeInternal, errors.ErrCodeDatabaseError, "graph database is closed")
	}

	query := "MATCH (n) WHERE elementId(n) = $id DETACH DELETE n"
	params := map[string]interface{}{"id": id}
	
	_, err := ng.Query(ctx, query, params)
	return err
}

// Close closes the database connection
func (ng *Neo4jGraphDB) Close() error {
	ng.sessionMu.Lock()
	defer ng.sessionMu.Unlock()
	
	if ng.session != nil {
		ng.session.Close(context.Background())
		ng.session = nil
	}
	
	if ng.driver != nil {
		ng.driver.Close(context.Background())
		ng.driver = nil
	}
	
	return ng.BaseGraphDB.Close()
}

// HealthCheck performs a health check on the database
func (ng *Neo4jGraphDB) HealthCheck(ctx context.Context) error {
	if ng.IsClosed() {
		return errors.NewMemGOSError(types.ErrorTypeInternal, errors.ErrCodeDatabaseError, "graph database is closed")
	}

	ctx, cancel := context.WithTimeout(ctx, ng.config.ReadTimeout)
	defer cancel()
	
	if err := ng.driver.VerifyConnectivity(ctx); err != nil {
		ng.UpdateHealth("unhealthy", err)
		return err
	}
	
	// Test basic query
	_, err := ng.Query(ctx, "RETURN 1 as test", nil)
	if err != nil {
		ng.UpdateHealth("unhealthy", err)
		return err
	}
	
	ng.UpdateHealth("healthy", nil)
	return nil
}

// GetMetrics returns database metrics
func (ng *Neo4jGraphDB) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	baseStats := ng.GetStats()
	
	// Add Neo4j-specific metrics
	metrics := make(map[string]interface{})
	for k, v := range baseStats {
		metrics[k] = v
	}
	
	// Get database info if available
	if !ng.IsClosed() {
		if results, err := ng.Query(ctx, "CALL dbms.components() YIELD name, versions, edition", nil); err == nil {
			metrics["neo4j_info"] = results
		}
	}
	
	health, healthErr, lastCheck := ng.GetHealth()
	metrics["health_status"] = health
	metrics["last_health_check"] = lastCheck
	if healthErr != nil {
		metrics["health_error"] = healthErr.Error()
	}
	
	return metrics, nil
}
