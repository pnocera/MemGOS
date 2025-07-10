package vectordb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/memtensor/memgos/pkg/types"
)

func TestVectorDBItem(t *testing.T) {
	t.Run("NewVectorDBItem", func(t *testing.T) {
		vector := []float32{0.1, 0.2, 0.3}
		payload := map[string]interface{}{
			"text": "test content",
			"id":   123,
		}

		item, err := NewVectorDBItem("", vector, payload)
		require.NoError(t, err)
		assert.NotEmpty(t, item.ID)
		assert.Equal(t, vector, item.Vector)
		assert.Equal(t, payload, item.Payload)
		assert.Nil(t, item.Score)
	})

	t.Run("NewVectorDBItemWithScore", func(t *testing.T) {
		vector := []float32{0.1, 0.2, 0.3}
		payload := map[string]interface{}{"text": "test"}
		score := float32(0.95)

		item, err := NewVectorDBItemWithScore("550e8400-e29b-41d4-a716-446655440000", vector, payload, score)
		require.NoError(t, err)
		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", item.ID)
		assert.Equal(t, &score, item.Score)
	})

	t.Run("Validate", func(t *testing.T) {
		item := &VectorDBItem{ID: "invalid-id"}
		err := item.Validate()
		assert.Error(t, err)

		item.ID = "550e8400-e29b-41d4-a716-446655440000"
		err = item.Validate()
		assert.NoError(t, err)
	})

	t.Run("ToMap", func(t *testing.T) {
		score := float32(0.8)
		item := &VectorDBItem{
			ID:      "test-id",
			Vector:  []float32{1, 2, 3},
			Payload: map[string]interface{}{"key": "value"},
			Score:   &score,
		}

		m := item.ToMap()
		assert.Equal(t, "test-id", m["id"])
		assert.Equal(t, []float32{1, 2, 3}, m["vector"])
		assert.Equal(t, map[string]interface{}{"key": "value"}, m["payload"])
		assert.Equal(t, float32(0.8), m["score"])
	})

	t.Run("FromMap", func(t *testing.T) {
		data := map[string]interface{}{
			"id":      "550e8400-e29b-41d4-a716-446655440000",
			"vector":  []float32{1, 2, 3},
			"payload": map[string]interface{}{"key": "value"},
			"score":   0.8,
		}

		item, err := FromMap(data)
		require.NoError(t, err)
		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", item.ID)
		assert.Equal(t, []float32{1, 2, 3}, item.Vector)
		assert.Equal(t, map[string]interface{}{"key": "value"}, item.Payload)
		assert.Equal(t, float32(0.8), *item.Score)
	})
}

func TestQdrantConfig(t *testing.T) {
	t.Run("DefaultQdrantConfig", func(t *testing.T) {
		config := DefaultQdrantConfig()
		assert.Equal(t, "localhost", config.Host)
		assert.Equal(t, 6333, config.Port)
		assert.Equal(t, "memgos_vectors", config.CollectionName)
		assert.Equal(t, 768, config.VectorSize)
		assert.Equal(t, "cosine", config.Distance)
	})

	t.Run("Validate", func(t *testing.T) {
		config := &QdrantConfig{
			Host:           "localhost",
			Port:           6333,
			CollectionName: "test",
			VectorSize:     768,
			Distance:       "cosine",
			BatchSize:      100,
			SearchLimit:    1000,
			DefaultTopK:    10,
		}
		err := config.Validate()
		assert.NoError(t, err)

		// Test invalid distance
		config.Distance = "invalid"
		err = config.Validate()
		assert.Error(t, err)
	})

	t.Run("GetConnectionURL", func(t *testing.T) {
		config := &QdrantConfig{
			Host:     "localhost",
			Port:     6333,
			UseHTTPS: false,
		}
		assert.Equal(t, "http://localhost:6333", config.GetConnectionURL())

		config.UseHTTPS = true
		assert.Equal(t, "https://localhost:6333", config.GetConnectionURL())
	})
}

func TestVectorDBFactory(t *testing.T) {
	t.Run("RegisterProvider", func(t *testing.T) {
		factory := NewVectorDBFactory()
		provider := &QdrantProvider{}

		err := factory.RegisterProvider(provider)
		assert.NoError(t, err)

		// Test duplicate registration
		err = factory.RegisterProvider(provider)
		assert.Error(t, err)
	})

	t.Run("GetProvider", func(t *testing.T) {
		factory := NewVectorDBFactory()
		provider := &QdrantProvider{}
		factory.RegisterProvider(provider)

		retrieved, err := factory.GetProvider(types.BackendQdrant)
		assert.NoError(t, err)
		assert.Equal(t, provider, retrieved)

		// Test non-existent provider
		_, err = factory.GetProvider("non-existent")
		assert.Error(t, err)
	})

	t.Run("ListProviders", func(t *testing.T) {
		factory := NewVectorDBFactory()
		provider := &QdrantProvider{}
		factory.RegisterProvider(provider)

		providers := factory.ListProviders()
		assert.Len(t, providers, 1)
		assert.Contains(t, providers, types.BackendQdrant)
	})
}

func TestQdrantProvider(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		provider := &QdrantProvider{}
		config := DefaultVectorDBConfig()

		db, err := provider.Create(config)
		assert.NoError(t, err)
		assert.NotNil(t, db)

		qdrantDB, ok := db.(*QdrantVectorDB)
		assert.True(t, ok)
		assert.Equal(t, config.Qdrant, qdrantDB.config)
	})

	t.Run("Validate", func(t *testing.T) {
		provider := &QdrantProvider{}
		config := DefaultVectorDBConfig()

		err := provider.Validate(config)
		assert.NoError(t, err)

		// Test invalid backend
		config.Backend = "invalid"
		err = provider.Validate(config)
		assert.Error(t, err)
	})

	t.Run("GetBackendType", func(t *testing.T) {
		provider := &QdrantProvider{}
		assert.Equal(t, types.BackendQdrant, provider.GetBackendType())
	})
}

func TestSimilarityCalculator(t *testing.T) {
	calc := NewSimilarityCalculator()

	t.Run("CosineSimilarity", func(t *testing.T) {
		a := []float32{1, 0, 0}
		b := []float32{1, 0, 0}
		
		similarity, err := calc.CosineSimilarity(a, b)
		assert.NoError(t, err)
		assert.InDelta(t, 1.0, similarity, 0.001)

		// Test orthogonal vectors
		a = []float32{1, 0}
		b = []float32{0, 1}
		similarity, err = calc.CosineSimilarity(a, b)
		assert.NoError(t, err)
		assert.InDelta(t, 0.0, similarity, 0.001)
	})

	t.Run("EuclideanDistance", func(t *testing.T) {
		a := []float32{0, 0}
		b := []float32{3, 4}
		
		distance, err := calc.EuclideanDistance(a, b)
		assert.NoError(t, err)
		assert.InDelta(t, 5.0, distance, 0.001)
	})

	t.Run("DotProduct", func(t *testing.T) {
		a := []float32{1, 2, 3}
		b := []float32{4, 5, 6}
		
		product, err := calc.DotProduct(a, b)
		assert.NoError(t, err)
		assert.InDelta(t, 32.0, product, 0.001) // 1*4 + 2*5 + 3*6 = 32
	})

	t.Run("VectorNorm", func(t *testing.T) {
		vector := []float32{3, 4}
		norm := calc.VectorNorm(vector)
		assert.InDelta(t, 5.0, norm, 0.001)
	})

	t.Run("NormalizeVector", func(t *testing.T) {
		vector := []float32{3, 4}
		normalized := calc.NormalizeVector(vector)
		
		expectedNorm := float32(1.0)
		actualNorm := calc.VectorNorm(normalized)
		assert.InDelta(t, expectedNorm, actualNorm, 0.001)
	})
}

func TestVectorAnalyzer(t *testing.T) {
	analyzer := NewVectorAnalyzer()

	t.Run("AnalyzeVector", func(t *testing.T) {
		vector := []float32{1, 2, 3, 4, 5}
		stats := analyzer.AnalyzeVector(vector)

		assert.Equal(t, 5, stats.Dimensions)
		assert.InDelta(t, 3.0, stats.Mean, 0.001)
		assert.Equal(t, float32(1), stats.Min)
		assert.Equal(t, float32(5), stats.Max)
		assert.True(t, stats.Std > 0)
		assert.True(t, stats.Norm > 0)
	})

	t.Run("FindSimilarVectors", func(t *testing.T) {
		query := []float32{1, 0}
		vectors := [][]float32{
			{1, 0},     // identical
			{0.9, 0.1}, // similar
			{0, 1},     // orthogonal
		}

		indices, err := analyzer.FindSimilarVectors(query, vectors, 0.8, "cosine")
		assert.NoError(t, err)
		assert.Len(t, indices, 2) // First two vectors should be similar enough
		assert.Contains(t, indices, 0)
		assert.Contains(t, indices, 1)
	})
}

func TestVectorIndex(t *testing.T) {
	index := NewVectorIndex()

	t.Run("Add and Search", func(t *testing.T) {
		// Add some vectors
		index.Add([]float32{1, 0}, map[string]interface{}{"id": 1})
		index.Add([]float32{0, 1}, map[string]interface{}{"id": 2})
		index.Add([]float32{0.9, 0.1}, map[string]interface{}{"id": 3})

		assert.Equal(t, 3, index.Size())

		// Search for similar vectors
		query := []float32{1, 0}
		results, err := index.Search(query, 2, "cosine")
		assert.NoError(t, err)
		assert.Len(t, results, 2)

		// First result should be the most similar
		assert.True(t, *results[0].Score > *results[1].Score)
	})

	t.Run("Clear", func(t *testing.T) {
		index := NewVectorIndex() // Create fresh index for this test
		index.Add([]float32{1, 0}, map[string]interface{}{"id": 1})
		assert.Equal(t, 1, index.Size())

		index.Clear()
		assert.Equal(t, 0, index.Size())
	})
}

func TestPerformanceMonitor(t *testing.T) {
	monitor := NewPerformanceMonitor()

	t.Run("RecordOperation", func(t *testing.T) {
		operation := "search"
		duration := 100 * time.Millisecond

		monitor.RecordOperation(operation, duration, true)
		monitor.RecordOperation(operation, 200*time.Millisecond, true)
		monitor.RecordOperation(operation, 50*time.Millisecond, false)

		metrics := monitor.GetMetrics(operation)
		assert.NotNil(t, metrics)
		assert.Equal(t, int64(3), metrics.TotalOperations)
		assert.Equal(t, int64(2), metrics.SuccessCount)
		assert.Equal(t, int64(1), metrics.ErrorCount)
		assert.Equal(t, 50*time.Millisecond, metrics.MinDuration)
		assert.Equal(t, 200*time.Millisecond, metrics.MaxDuration)
	})

	t.Run("GetAllMetrics", func(t *testing.T) {
		monitor.RecordOperation("add", 50*time.Millisecond, true)
		monitor.RecordOperation("delete", 25*time.Millisecond, true)

		allMetrics := monitor.GetAllMetrics()
		assert.Len(t, allMetrics, 3) // search + add + delete
		assert.Contains(t, allMetrics, "add")
		assert.Contains(t, allMetrics, "delete")
	})

	t.Run("Reset", func(t *testing.T) {
		monitor.Reset()
		allMetrics := monitor.GetAllMetrics()
		assert.Len(t, allMetrics, 0)
	})
}

func TestConnectionManager(t *testing.T) {
	t.Run("GetConnection", func(t *testing.T) {
		manager := NewConnectionManager()
		config := DefaultVectorDBConfig()

		// This would normally require a running Qdrant instance
		// For testing, we'll just verify the connection logic without actual connection
		_, err := manager.GetConnection(context.Background(), "test", config)
		// We expect an error since we don't have a running Qdrant instance
		assert.Error(t, err)
	})

	t.Run("ListConnections", func(t *testing.T) {
		manager := NewConnectionManager()
		connections := manager.ListConnections()
		assert.Len(t, connections, 0)
	})
}

func TestVectorDBRegistry(t *testing.T) {
	registry := NewVectorDBRegistry()

	t.Run("Register", func(t *testing.T) {
		config := DefaultVectorDBConfig()
		err := registry.Register("test-db", config)
		assert.NoError(t, err)

		// Test duplicate registration
		err = registry.Register("test-db", config)
		assert.Error(t, err)
	})

	t.Run("List", func(t *testing.T) {
		names := registry.List()
		assert.Contains(t, names, "test-db")
	})

	t.Run("GetInfo", func(t *testing.T) {
		info, err := registry.GetInfo("test-db")
		assert.NoError(t, err)
		assert.Equal(t, "test-db", info.Name)
		assert.Equal(t, types.BackendQdrant, info.Backend)

		// Test non-existent database
		_, err = registry.GetInfo("non-existent")
		assert.Error(t, err)
	})

	t.Run("Unregister", func(t *testing.T) {
		err := registry.Unregister(context.Background(), "test-db")
		assert.NoError(t, err)

		names := registry.List()
		assert.NotContains(t, names, "test-db")
	})
}

// Benchmark tests
func BenchmarkCosineSimilarity(b *testing.B) {
	calc := NewSimilarityCalculator()
	a := make([]float32, 768)
	vec := make([]float32, 768)
	
	for i := range a {
		a[i] = float32(i) / 768.0
		vec[i] = float32(768-i) / 768.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = calc.CosineSimilarity(a, vec)
	}
}

func BenchmarkVectorIndex(b *testing.B) {
	index := NewVectorIndex()
	
	// Add 1000 vectors
	for i := 0; i < 1000; i++ {
		vector := make([]float32, 128)
		for j := range vector {
			vector[j] = float32(i*j) / 1000.0
		}
		index.Add(vector, map[string]interface{}{"id": i})
	}

	query := make([]float32, 128)
	for i := range query {
		query[i] = 0.5
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = index.Search(query, 10, "cosine")
	}
}