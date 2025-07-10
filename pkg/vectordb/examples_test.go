package vectordb_test

import (
	"context"
	"fmt"
	"log"

	"github.com/memtensor/memgos/pkg/types"
	"github.com/memtensor/memgos/pkg/vectordb"
)

// Example_basicUsage demonstrates basic vector database usage
func Example_basicUsage() {
	// Create a Qdrant configuration
	config := vectordb.DefaultVectorDBConfig()
	config.Qdrant.Host = "localhost"
	config.Qdrant.Port = 6333
	config.Qdrant.CollectionName = "example_collection"
	config.Qdrant.VectorSize = 3

	// Create the vector database
	db, err := vectordb.CreateVectorDB(config)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Connect to the database (Note: requires running Qdrant instance)
	if err := db.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}
	defer db.Disconnect(ctx)

	// Create a collection
	collectionConfig := &vectordb.CollectionConfig{
		Name:       "example_collection",
		VectorSize: 3,
		Distance:   "cosine",
	}

	err = db.CreateCollection(ctx, collectionConfig)
	if err != nil {
		log.Printf("Failed to create collection: %v", err)
		return
	}

	// Create some vector items
	items := vectordb.VectorDBItemList{
		{
			ID:      "item1",
			Vector:  []float32{0.1, 0.2, 0.3},
			Payload: map[string]interface{}{"text": "first item"},
		},
		{
			ID:      "item2",
			Vector:  []float32{0.4, 0.5, 0.6},
			Payload: map[string]interface{}{"text": "second item"},
		},
	}

	// Add items to the collection
	err = db.Add(ctx, "example_collection", items)
	if err != nil {
		log.Printf("Failed to add items: %v", err)
		return
	}

	// Search for similar vectors
	queryVector := []float32{0.1, 0.2, 0.3}
	results, err := db.Search(ctx, "example_collection", queryVector, 10, nil)
	if err != nil {
		log.Printf("Failed to search: %v", err)
		return
	}

	fmt.Printf("Found %d results\n", len(results))
}

// Example_factory demonstrates using the factory pattern
func Example_factory() {
	// Get the default factory
	factory := vectordb.GetFactory()

	// Create a vector database with default configuration
	db, err := factory.CreateWithDefaults(types.BackendQdrant)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Created database with backend: %s\n", db.GetConfig().Backend)
	// Output: Created database with backend: qdrant
}

// Example_registry demonstrates using the registry for managing multiple databases
func Example_registry() {
	// Get the default registry
	registry := vectordb.GetRegistry()

	// Register a database configuration
	config := vectordb.DefaultVectorDBConfig()
	config.Qdrant.CollectionName = "registry_example"

	err := registry.Register("my-database", config)
	if err != nil {
		log.Fatal(err)
	}

	// List registered databases
	_ = registry.List()
	// fmt.Printf("Registered databases: %v\n", databases)

	// Get database info
	info, err := registry.GetInfo("my-database")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Database backend: %s\n", info.Backend)
	// Output: Database backend: qdrant
}

// Example_operations demonstrates advanced vector operations
func Example_operations() {
	registry := vectordb.NewVectorDBRegistry()
	operations := vectordb.NewVectorOperations(registry)

	// Configure operations
	operations.SetBatchSize(50)
	operations.SetMaxWorkers(5)

	fmt.Printf("Operations configured with batch size and worker limits\n")
	// Output: Operations configured with batch size and worker limits
}

// Example_similarity demonstrates similarity calculations
func Example_similarity() {
	calc := vectordb.NewSimilarityCalculator()

	vectorA := []float32{1, 0, 0}
	vectorB := []float32{0, 1, 0}

	// Calculate cosine similarity
	similarity, err := calc.CosineSimilarity(vectorA, vectorB)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Cosine similarity: %.2f\n", similarity)

	// Calculate Euclidean distance
	distance, err := calc.EuclideanDistance(vectorA, vectorB)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Euclidean distance: %.2f\n", distance)

	// Output:
	// Cosine similarity: 0.00
	// Euclidean distance: 1.41
}

// Example_analyzer demonstrates vector analysis
func Example_analyzer() {
	analyzer := vectordb.NewVectorAnalyzer()

	vector := []float32{1, 2, 3, 4, 5}
	stats := analyzer.AnalyzeVector(vector)

	fmt.Printf("Vector dimensions: %d\n", stats.Dimensions)
	fmt.Printf("Vector mean: %.2f\n", stats.Mean)
	fmt.Printf("Vector std: %.2f\n", stats.Std)
	fmt.Printf("Vector norm: %.2f\n", stats.Norm)

	// Output:
	// Vector dimensions: 5
	// Vector mean: 3.00
	// Vector std: 1.41
	// Vector norm: 7.42
}

// Example_index demonstrates in-memory vector indexing
func Example_index() {
	index := vectordb.NewVectorIndex()

	// Add vectors to the index
	index.Add([]float32{1, 0}, map[string]interface{}{"id": 1, "text": "first"})
	index.Add([]float32{0, 1}, map[string]interface{}{"id": 2, "text": "second"})
	index.Add([]float32{0.7, 0.7}, map[string]interface{}{"id": 3, "text": "third"})

	fmt.Printf("Index size: %d\n", index.Size())

	// Search for similar vectors
	query := []float32{1, 0}
	results, err := index.Search(query, 2, "cosine")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d similar vectors\n", len(results))

	// Output:
	// Index size: 3
	// Found 2 similar vectors
}

// Example_performanceMonitoring demonstrates performance monitoring
func Example_performanceMonitoring() {
	monitor := vectordb.NewPerformanceMonitor()

	// Simulate some operations
	monitor.RecordOperation("search", 100e6, true) // 100ms
	monitor.RecordOperation("search", 150e6, true) // 150ms
	monitor.RecordOperation("add", 50e6, true)     // 50ms

	// Get metrics
	searchMetrics := monitor.GetMetrics("search")
	if searchMetrics != nil {
		fmt.Printf("Search operations: %d\n", searchMetrics.TotalOperations)
		fmt.Printf("Success rate: %.2f%%\n", float64(searchMetrics.SuccessCount)/float64(searchMetrics.TotalOperations)*100)
	}

	// Output:
	// Search operations: 2
	// Success rate: 100.00%
}

// Example_batchOperations demonstrates batch operations
func Example_batchOperations() {
	// Create sample items for batch operations
	items := make(vectordb.VectorDBItemList, 100)
	for i := range items {
		item, _ := vectordb.NewVectorDBItem(
			"",
			[]float32{float32(i), float32(i + 1), float32(i + 2)},
			map[string]interface{}{"index": i},
		)
		items[i] = item
	}

	fmt.Printf("Created %d items for batch operation\n", len(items))

	// In a real scenario, you would use these with a vector database:
	// err := db.BatchAdd(ctx, "collection", items, 50)

	// Output: Created 100 items for batch operation
}

// Example_configuration demonstrates configuration management
func Example_configuration() {
	// Create custom configuration
	config := &vectordb.QdrantConfig{
		Host:           "custom-host",
		Port:           6333,
		CollectionName: "custom_collection",
		VectorSize:     512,
		Distance:       "euclidean",
		BatchSize:      200,
		SearchLimit:    500,
		DefaultTopK:    20,
	}

	// Validate configuration
	err := config.Validate()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Configuration is valid\n")
	fmt.Printf("Connection URL: %s\n", config.GetConnectionURL())

	// Output:
	// Configuration is valid
	// Connection URL: http://custom-host:6333
}

// Example_errorHandling demonstrates error handling patterns
func Example_errorHandling() {
	// Create an invalid configuration
	config := &vectordb.QdrantConfig{
		Host:           "", // Invalid: empty host
		Port:           -1, // Invalid: negative port
		CollectionName: "", // Invalid: empty collection name
		VectorSize:     0,  // Invalid: zero vector size
	}

	err := config.Validate()
	if err != nil {
		fmt.Printf("Configuration error: %v\n", err)
	}

	// Create an invalid vector item
	item := &vectordb.VectorDBItem{
		ID: "invalid-uuid", // Invalid: not a proper UUID
	}

	err = item.Validate()
	if err != nil {
		fmt.Printf("Item validation error: %v\n", err)
	}

	// Output:
	// Configuration error: host is required
	// Item validation error: invalid ID format: invalid-uuid, must be a valid UUID
}