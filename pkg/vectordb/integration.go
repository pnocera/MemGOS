// Package vectordb integration example
// This file demonstrates how to integrate the vector database package with the MemGOS system

package vectordb

import (
	"context"
	"fmt"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// MemGOSVectorDBAdapter adapts our vector database implementation to the MemGOS VectorDB interface
type MemGOSVectorDBAdapter struct {
	vectorDB   BaseVectorDB
	collection string
}

// NewMemGOSVectorDBAdapter creates a new adapter
func NewMemGOSVectorDBAdapter(vectorDB BaseVectorDB, collection string) *MemGOSVectorDBAdapter {
	return &MemGOSVectorDBAdapter{
		vectorDB:   vectorDB,
		collection: collection,
	}
}

// Insert implements the MemGOS VectorDB interface
func (a *MemGOSVectorDBAdapter) Insert(ctx context.Context, vectors []types.EmbeddingVector, metadata []map[string]interface{}) error {
	if len(vectors) != len(metadata) {
		return fmt.Errorf("vectors and metadata length mismatch: %d vs %d", len(vectors), len(metadata))
	}

	items := make(VectorDBItemList, len(vectors))
	for i, vector := range vectors {
		item, err := NewVectorDBItem("", vector, metadata[i])
		if err != nil {
			return fmt.Errorf("failed to create vector item %d: %w", i, err)
		}
		items[i] = item
	}

	return a.vectorDB.Add(ctx, a.collection, items)
}

// Search implements the MemGOS VectorDB interface
func (a *MemGOSVectorDBAdapter) Search(ctx context.Context, query types.EmbeddingVector, topK int, filters map[string]string) ([]types.VectorSearchResult, error) {
	// Convert filters to the expected format
	filterMap := make(map[string]interface{})
	for k, v := range filters {
		filterMap[k] = v
	}

	items, err := a.vectorDB.Search(ctx, a.collection, query, topK, filterMap)
	if err != nil {
		return nil, err
	}

	results := make([]types.VectorSearchResult, len(items))
	for i, item := range items {
		score := float32(0)
		if item.Score != nil {
			score = *item.Score
		}

		results[i] = types.VectorSearchResult{
			ID:       item.ID,
			Score:    score,
			Metadata: item.Payload,
		}
	}

	return results, nil
}

// Update implements the MemGOS VectorDB interface
func (a *MemGOSVectorDBAdapter) Update(ctx context.Context, id string, vector types.EmbeddingVector, metadata map[string]interface{}) error {
	item, err := NewVectorDBItem(id, vector, metadata)
	if err != nil {
		return fmt.Errorf("failed to create vector item: %w", err)
	}

	return a.vectorDB.Update(ctx, a.collection, VectorDBItemList{item})
}

// Delete implements the MemGOS VectorDB interface
func (a *MemGOSVectorDBAdapter) Delete(ctx context.Context, ids []string) error {
	return a.vectorDB.Delete(ctx, a.collection, ids)
}

// GetStats implements the MemGOS VectorDB interface
func (a *MemGOSVectorDBAdapter) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats, err := a.vectorDB.GetStats(ctx, a.collection)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"collection_name":  stats.CollectionName,
		"points_count":     stats.PointsCount,
		"indexed_vectors":  stats.IndexedVectors,
		"memory_usage":     stats.MemoryUsage,
		"disk_usage":       stats.DiskUsage,
		"segments_count":   stats.SegmentsCount,
		"payload_schema":   stats.PayloadSchema,
	}, nil
}

// Close implements the MemGOS VectorDB interface
func (a *MemGOSVectorDBAdapter) Close() error {
	return a.vectorDB.Disconnect(context.Background())
}

// Verify that our adapter implements the interface
var _ interfaces.VectorDB = (*MemGOSVectorDBAdapter)(nil)

// IntegrationExample demonstrates how to integrate vector database with MemGOS
func IntegrationExample() error {
	// Create vector database configuration
	config := DefaultVectorDBConfig()
	config.Qdrant.Host = "localhost"
	config.Qdrant.Port = 6333
	config.Qdrant.CollectionName = "memgos_integration"

	// Create vector database
	vectorDB, err := CreateVectorDB(config)
	if err != nil {
		return fmt.Errorf("failed to create vector database: %w", err)
	}

	ctx := context.Background()

	// Connect to database
	if err := vectorDB.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer vectorDB.Disconnect(ctx)

	// Create collection
	collectionConfig := &CollectionConfig{
		Name:       "memgos_integration",
		VectorSize: 768,
		Distance:   "cosine",
	}

	if err := vectorDB.CreateCollection(ctx, collectionConfig); err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	// Create adapter for MemGOS interface
	adapter := NewMemGOSVectorDBAdapter(vectorDB, "memgos_integration")

	// Now the adapter can be used anywhere the MemGOS VectorDB interface is expected
	_ = adapter

	return nil
}

// TextualMemoryIntegration shows how to integrate with textual memory
func TextualMemoryIntegration(textualMem interfaces.TextualMemory, vectorDB BaseVectorDB) error {
	// This function would typically be called when adding new textual memories
	// to ensure they are also indexed in the vector database

	ctx := context.Background()

	// Get all textual memories
	memories, err := textualMem.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to get textual memories: %w", err)
	}

	// Convert to vector database items (assuming embeddings are generated elsewhere)
	var items VectorDBItemList
	for _, memory := range memories {
		if textMem, ok := memory.(*types.TextualMemoryItem); ok {
			// In practice, you would generate embeddings here using an embedder
			// For this example, we'll create a dummy embedding
			embedding := make([]float32, 768)
			for i := range embedding {
				embedding[i] = 0.1 // Dummy values
			}

			item, err := NewVectorDBItem(textMem.ID, embedding, map[string]interface{}{
				"memory":     textMem.Memory,
				"created_at": textMem.CreatedAt,
				"updated_at": textMem.UpdatedAt,
			})
			if err != nil {
				continue
			}

			items = append(items, item)
		}
	}

	// Add to vector database
	if len(items) > 0 {
		return vectorDB.BatchAdd(ctx, "textual_memories", items, 100)
	}

	return nil
}

// VectorSearchWithTextualMemory demonstrates semantic search over textual memories
func VectorSearchWithTextualMemory(vectorDB BaseVectorDB, queryEmbedding []float32, topK int) ([]map[string]interface{}, error) {
	ctx := context.Background()

	// Search for similar vectors
	results, err := vectorDB.Search(ctx, "textual_memories", queryEmbedding, topK, nil)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Convert to structured results
	searchResults := make([]map[string]interface{}, len(results))
	for i, item := range results {
		searchResults[i] = map[string]interface{}{
			"id":       item.ID,
			"score":    item.Score,
			"memory":   item.Payload["memory"],
			"metadata": item.Payload,
		}
	}

	return searchResults, nil
}

// BulkOperationsExample demonstrates efficient bulk operations
func BulkOperationsExample(vectorDB BaseVectorDB) error {
	ctx := context.Background()

	// Create large number of items for bulk insert
	items := make(VectorDBItemList, 1000)
	for i := range items {
		embedding := make([]float32, 768)
		for j := range embedding {
			embedding[j] = float32(i*j) / 1000.0
		}

		item, _ := NewVectorDBItem("", embedding, map[string]interface{}{
			"index":     i,
			"category":  fmt.Sprintf("cat_%d", i%10),
			"timestamp": "2023-01-01",
		})
		items[i] = item
	}

	// Perform bulk insert with batching
	batchSize := 100
	return vectorDB.BatchAdd(ctx, "bulk_test", items, batchSize)
}

// PerformanceMonitoringExample shows how to monitor vector database performance
func PerformanceMonitoringExample(vectorDB BaseVectorDB) error {
	monitor := NewPerformanceMonitor()
	ctx := context.Background()

	// Mock some operations for demonstration
	queryVector := make([]float32, 768)
	for i := range queryVector {
		queryVector[i] = 0.5
	}

	// Time the search operation
	_, err := vectorDB.Search(ctx, "test_collection", queryVector, 10, nil)
	
	// Record the operation (in practice, this would be done automatically)
	monitor.RecordOperation("search", 100e6, err == nil) // 100ms

	// Get performance metrics
	metrics := monitor.GetMetrics("search")
	if metrics != nil {
		fmt.Printf("Search metrics: %+v\n", metrics)
	}

	return nil
}