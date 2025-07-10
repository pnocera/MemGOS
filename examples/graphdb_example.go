// Example usage of MemGOS Graph Database integration
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/memtensor/memgos/pkg/graphdb"
)

func main() {
	// Create graph database configuration
	config := graphdb.NewConfigBuilder().
		Provider(graphdb.ProviderNeo4j).
		URI("bolt://localhost:7687").
		Credentials("neo4j", "password").
		Database("memgos").
		ConnectionPool(20, 10*time.Second).
		Timeouts(5*time.Second, 5*time.Second).
		Retry(3, time.Second).
		EnableMetrics().
		EnableLogging().
		SSL("disable").
		Build()

	// Validate configuration
	if err := config.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Create graph database manager
	manager, err := graphdb.NewGraphDBManager(config.GraphDBConfig, nil, nil)
	if err != nil {
		log.Fatalf("Failed to create graph database manager: %v", err)
	}
	defer manager.Close()

	// Get default database instance
	db := manager.GetDefault()

	ctx := context.Background()

	// Example 1: Create nodes
	fmt.Println("Creating nodes...")
	
	// Create a person node
	personID, err := db.CreateNode(ctx, []string{"Person"}, map[string]interface{}{
		"name": "Alice",
		"age":  30,
		"city": "New York",
	})
	if err != nil {
		log.Printf("Failed to create person node: %v", err)
		return
	}

	// Create a memory node
	memoryID, err := db.CreateNode(ctx, []string{"Memory"}, map[string]interface{}{
		"content":   "Alice loves machine learning",
		"type":      "textual",
		"timestamp": time.Now().Unix(),
	})
	if err != nil {
		log.Printf("Failed to create memory node: %v", err)
		return
	}

	// Example 2: Create relationships
	fmt.Println("Creating relationships...")
	
	err = db.CreateRelation(ctx, personID, memoryID, "HAS_MEMORY", map[string]interface{}{
		"confidence": 0.95,
		"created_at": time.Now().Unix(),
	})
	if err != nil {
		log.Printf("Failed to create relationship: %v", err)
		return
	}

	// Example 3: Query the graph
	fmt.Println("Querying the graph...")
	
	// Use query builder
	qb := graphdb.NewQueryBuilder()
	qb.Match("(p:Person)-[r:HAS_MEMORY]->(m:Memory)")
	qb.Where("p.name = $name")
	qb.Return("p.name as person, m.content as memory, r.confidence as confidence")
	qb.AddParameter("name", "Alice")

	results, err := qb.Execute(ctx, db)
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}

	fmt.Printf("Query results: %v\n", results)

	// Example 4: Graph operations
	fmt.Println("Performing graph operations...")
	
	graphOps := graphdb.NewGraphOperations(db, nil, nil)

	// Find neighbors
	neighbors, err := graphOps.GetNeighbors(ctx, personID, "OUT", []string{"HAS_MEMORY"}, 1)
	if err != nil {
		log.Printf("Failed to find neighbors: %v", err)
		return
	}
	fmt.Printf("Found %d neighbors\n", len(neighbors))

	// Calculate degree
	degree, err := graphOps.CalculateDegree(ctx, personID, "BOTH")
	if err != nil {
		log.Printf("Failed to calculate degree: %v", err)
		return
	}
	fmt.Printf("Node degree: %d\n", degree)

	// Example 5: Batch operations
	fmt.Println("Performing batch operations...")
	
	batchOps := []graphdb.BatchOperation{
		{
			OperationType: "CREATE_NODE",
			NodeLabels:    []string{"Memory"},
			Properties: map[string]interface{}{
				"content": "Alice enjoys reading",
				"type":    "textual",
			},
		},
		{
			OperationType: "CREATE_RELATION",
			FromID:        personID,
			ToID:          "", // Would be filled by actual implementation
			RelationType:  "HAS_MEMORY",
			Properties: map[string]interface{}{
				"confidence": 0.8,
			},
		},
	}

	err = db.ExecuteBatch(ctx, batchOps)
	if err != nil {
		log.Printf("Batch operation failed: %v", err)
		return
	}

	// Example 6: Graph analytics
	fmt.Println("Analyzing graph structure...")
	
	analysis, err := graphOps.AnalyzeGraphStructure(ctx)
	if err != nil {
		log.Printf("Failed to analyze graph: %v", err)
		return
	}
	fmt.Printf("Graph analysis: %v\n", analysis)

	// Example 7: Health check
	fmt.Println("Checking database health...")
	
	err = db.HealthCheck(ctx)
	if err != nil {
		log.Printf("Health check failed: %v", err)
		return
	}
	fmt.Println("Database is healthy")

	// Example 8: Get metrics
	metrics, err := db.GetMetrics(ctx)
	if err != nil {
		log.Printf("Failed to get metrics: %v", err)
		return
	}
	fmt.Printf("Database metrics: %v\n", metrics)

	fmt.Println("Graph database example completed successfully!")
}

// Example of advanced usage with transactions
func advancedTransactionExample(db graphdb.ExtendedGraphDB) error {
	ctx := context.Background()

	// Begin transaction
	tx, err := db.BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Always rollback in defer, commit will override

	// Create multiple nodes in transaction
	node1ID, err := tx.CreateNode(ctx, []string{"Person"}, map[string]interface{}{
		"name": "Bob",
		"age":  25,
	})
	if err != nil {
		return fmt.Errorf("failed to create node1: %w", err)
	}

	node2ID, err := tx.CreateNode(ctx, []string{"Memory"}, map[string]interface{}{
		"content": "Bob's first memory",
		"type":    "textual",
	})
	if err != nil {
		return fmt.Errorf("failed to create node2: %w", err)
	}

	// Create relationship in transaction
	err = tx.CreateRelation(ctx, node1ID, node2ID, "HAS_MEMORY", map[string]interface{}{
		"confidence": 0.9,
	})
	if err != nil {
		return fmt.Errorf("failed to create relation: %w", err)
	}

	// Query within transaction
	results, err := tx.Query(ctx, "MATCH (p:Person {name: $name}) RETURN count(p) as count", map[string]interface{}{
		"name": "Bob",
	})
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	if len(results) > 0 && results[0]["count"].(int64) > 0 {
		// Commit transaction if everything is successful
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	return nil
}