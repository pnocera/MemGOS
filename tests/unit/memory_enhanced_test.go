package unit

import (
	"context"
	"testing"
	"time"

	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/memory/activation"
	"github.com/memtensor/memgos/pkg/memory/factory"
	"github.com/memtensor/memgos/pkg/memory/parametric"
	"github.com/memtensor/memgos/pkg/memory/textual"
	"github.com/memtensor/memgos/pkg/types"
)

// Mock implementations for testing
type MockLogger struct{}

func (ml *MockLogger) Info(msg string, fields ...map[string]interface{}) {}
func (ml *MockLogger) Error(msg string, fields ...map[string]interface{}) {}
func (ml *MockLogger) Debug(msg string, fields ...map[string]interface{}) {}
func (ml *MockLogger) Warn(msg string, fields ...map[string]interface{}) {}

type MockMetrics struct{}

func (mm *MockMetrics) Counter(name string, value float64, tags []string) {}
func (mm *MockMetrics) Timer(name string, value float64, tags []string) {}
func (mm *MockMetrics) Gauge(name string, value float64, tags []string) {}

type MockEmbedder struct{}

func (me *MockEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i := range texts {
		// Create mock embedding with dimension 128
		embedding := make([]float32, 128)
		for j := range embedding {
			embedding[j] = float32(i*j) / 100.0
		}
		embeddings[i] = embedding
	}
	return embeddings, nil
}

type MockLLM struct{}

func (ml *MockLLM) Generate(ctx context.Context, prompt string) (string, error) {
	return "Mock LLM response to: " + prompt, nil
}

func (ml *MockLLM) BuildKVCache(text string) (interface{}, error) {
	// Mock KV cache data
	return map[string]interface{}{
		"keys":   [][]float32{{1.0, 2.0, 3.0}},
		"values": [][]float32{{4.0, 5.0, 6.0}},
		"layers": 1,
	}, nil
}

// Test Naive Textual Memory
func TestNaiveTextualMemory(t *testing.T) {
	logger := &MockLogger{}
	metrics := &MockMetrics{}
	config := &config.MemoryConfig{
		MemoryFilename: "test_naive_memory.json",
	}
	
	memory, err := textual.NewNaiveTextMemory(config, logger, metrics)
	if err != nil {
		t.Fatalf("Failed to create naive textual memory: %v", err)
	}
	defer memory.Close()
	
	ctx := context.Background()
	
	// Test adding memories
	testMemories := []*types.TextualMemoryItem{
		{
			ID:        "test1",
			Memory:    "This is a test memory about artificial intelligence",
			Metadata:  map[string]interface{}{"category": "ai"},
			CreatedAt: time.Now(),
		},
		{
			ID:        "test2",
			Memory:    "Another test memory about machine learning",
			Metadata:  map[string]interface{}{"category": "ml"},
			CreatedAt: time.Now(),
		},
	}
	
	err = memory.Add(ctx, testMemories)
	if err != nil {
		t.Fatalf("Failed to add memories: %v", err)
	}
	
	// Test getting memory
	retrieved, err := memory.Get(ctx, "test1")
	if err != nil {
		t.Fatalf("Failed to get memory: %v", err)
	}
	
	if retrieved.Memory != testMemories[0].Memory {
		t.Errorf("Expected memory content %s, got %s", testMemories[0].Memory, retrieved.Memory)
	}
	
	// Test search
	results, err := memory.Search(ctx, "artificial intelligence", 10, textual.SearchModeFast, textual.MemoryTypeAll)
	if err != nil {
		t.Fatalf("Failed to search memories: %v", err)
	}
	
	if len(results) == 0 {
		t.Error("Expected search results, got none")
	}
	
	// Test getting all memories
	allMemories, err := memory.GetAll(ctx)
	if err != nil {
		t.Fatalf("Failed to get all memories: %v", err)
	}
	
	if len(allMemories) != 2 {
		t.Errorf("Expected 2 memories, got %d", len(allMemories))
	}
	
	// Test memory size
	size, err := memory.GetMemorySize(ctx)
	if err != nil {
		t.Fatalf("Failed to get memory size: %v", err)
	}
	
	if size["total"] != 2 {
		t.Errorf("Expected total size 2, got %d", size["total"])
	}
	
	// Test delete
	err = memory.Delete(ctx, []string{"test1"})
	if err != nil {
		t.Fatalf("Failed to delete memory: %v", err)
	}
	
	// Verify deletion
	_, err = memory.Get(ctx, "test1")
	if err == nil {
		t.Error("Expected error when getting deleted memory")
	}
}

// Test General Textual Memory
func TestGeneralTextualMemory(t *testing.T) {
	logger := &MockLogger{}
	metrics := &MockMetrics{}
	embedder := &MockEmbedder{}
	llm := &MockLLM{}
	config := &config.MemoryConfig{
		MemoryFilename: "test_general_memory.json",
	}
	
	memory, err := textual.NewGeneralTextMemory(config, embedder, nil, llm, logger, metrics)
	if err != nil {
		t.Fatalf("Failed to create general textual memory: %v", err)
	}
	defer memory.Close()
	
	ctx := context.Background()
	
	// Test adding memories
	testMemories := []*types.TextualMemoryItem{
		{
			ID:        "test1",
			Memory:    "Deep learning is a subset of machine learning",
			Metadata:  map[string]interface{}{"topic": "ai"},
			CreatedAt: time.Now(),
		},
		{
			ID:        "test2",
			Memory:    "Neural networks are inspired by biological neurons",
			Metadata:  map[string]interface{}{"topic": "neuroscience"},
			CreatedAt: time.Now(),
		},
	}
	
	err = memory.Add(ctx, testMemories)
	if err != nil {
		t.Fatalf("Failed to add memories: %v", err)
	}
	
	// Test semantic search
	results, err := memory.SearchSemantic(ctx, "machine learning", 10, nil)
	if err != nil {
		t.Fatalf("Failed to perform semantic search: %v", err)
	}
	
	if len(results) == 0 {
		t.Error("Expected semantic search results, got none")
	}
	
	// Test hybrid search
	results, err = memory.Search(ctx, "neural networks", 10, textual.SearchModeFine, textual.MemoryTypeAll)
	if err != nil {
		t.Fatalf("Failed to perform hybrid search: %v", err)
	}
	
	if len(results) == 0 {
		t.Error("Expected hybrid search results, got none")
	}
	
	// Test working memory operations
	workingMemory, err := memory.GetWorkingMemory(ctx)
	if err != nil {
		t.Fatalf("Failed to get working memory: %v", err)
	}
	
	if len(workingMemory) != 2 {
		t.Errorf("Expected 2 working memories, got %d", len(workingMemory))
	}
	
	// Test replace working memory
	newMemories := []*types.TextualMemoryItem{
		{
			ID:        "new1",
			Memory:    "Replacement memory content",
			Metadata:  map[string]interface{}{"type": "replacement"},
			CreatedAt: time.Now(),
		},
	}
	
	err = memory.ReplaceWorkingMemory(ctx, newMemories)
	if err != nil {
		t.Fatalf("Failed to replace working memory: %v", err)
	}
}

// Test Tree Textual Memory
func TestTreeTextualMemory(t *testing.T) {
	logger := &MockLogger{}
	metrics := &MockMetrics{}
	embedder := &MockEmbedder{}
	llm := &MockLLM{}
	config := &config.MemoryConfig{
		MemoryFilename: "test_tree_memory.json",
	}
	
	memory, err := textual.NewTreeTextMemory(config, embedder, nil, nil, llm, logger, metrics)
	if err != nil {
		t.Fatalf("Failed to create tree textual memory: %v", err)
	}
	defer memory.Close()
	
	ctx := context.Background()
	
	// Test adding memories with different types
	testMemories := []*types.TextualMemoryItem{
		{
			ID:     "work1",
			Memory: "Working memory content about current task",
			Metadata: map[string]interface{}{
				"memory_type": "WorkingMemory",
				"priority":    "high",
			},
			CreatedAt: time.Now(),
		},
		{
			ID:     "long1",
			Memory: "Long-term memory about historical events",
			Metadata: map[string]interface{}{
				"memory_type": "LongTermMemory",
				"category":    "history",
			},
			CreatedAt: time.Now(),
		},
	}
	
	err = memory.Add(ctx, testMemories)
	if err != nil {
		t.Fatalf("Failed to add memories: %v", err)
	}
	
	// Test different search modes
	fastResults, err := memory.Search(ctx, "current task", 5, textual.SearchModeFast, textual.MemoryTypeWorking)
	if err != nil {
		t.Fatalf("Failed to perform fast search: %v", err)
	}
	
	fineResults, err := memory.Search(ctx, "historical events", 5, textual.SearchModeFine, textual.MemoryTypeLongTerm)
	if err != nil {
		t.Fatalf("Failed to perform fine search: %v", err)
	}
	
	// Test subgraph operations
	subgraph, err := memory.GetRelevantSubgraph(ctx, "task", 3, 2)
	if err != nil {
		t.Fatalf("Failed to get relevant subgraph: %v", err)
	}
	
	if subgraph == nil {
		t.Error("Expected subgraph result, got nil")
	}
	
	// Test working memory specific operations
	workingMemories, err := memory.GetWorkingMemory(ctx)
	if err != nil {
		t.Fatalf("Failed to get working memories: %v", err)
	}
	
	// Should contain the working memory item
	foundWorking := false
	for _, mem := range workingMemories {
		if mem.ID == "work1" {
			foundWorking = true
			break
		}
	}
	
	if !foundWorking {
		t.Error("Working memory item not found in working memory results")
	}
	
	// Test memory size statistics
	size, err := memory.GetMemorySize(ctx)
	if err != nil {
		t.Fatalf("Failed to get memory size: %v", err)
	}
	
	if size["total"] != 2 {
		t.Errorf("Expected total size 2, got %d", size["total"])
	}
	
	if size["working"] != 1 {
		t.Errorf("Expected working memory size 1, got %d", size["working"])
	}
}

// Test KV Cache Memory
func TestKVCacheMemory(t *testing.T) {
	logger := &MockLogger{}
	metrics := &MockMetrics{}
	llm := &MockLLM{}
	config := &config.MemoryConfig{
		MemoryFilename: "test_kv_cache.json",
	}
	
	memory, err := activation.NewKVCacheMemory(config, llm, logger, metrics)
	if err != nil {
		t.Fatalf("Failed to create KV cache memory: %v", err)
	}
	defer memory.Close()
	
	ctx := context.Background()
	
	// Test extracting KV cache from text
	extractedCache, err := memory.Extract(ctx, "This is test text for KV cache extraction")
	if err != nil {
		t.Fatalf("Failed to extract KV cache: %v", err)
	}
	
	if extractedCache == nil {
		t.Error("Expected extracted cache, got nil")
	}
	
	// Test adding KV cache memories
	testMemories := []*types.ActivationMemoryItem{
		{
			ID:     "cache1",
			Memory: map[string]interface{}{
				"keys":   [][]float32{{1.0, 2.0}, {3.0, 4.0}},
				"values": [][]float32{{5.0, 6.0}, {7.0, 8.0}},
				"layers": 2,
			},
			Metadata: map[string]interface{}{
				"model_id": "test_model",
				"type":     "kv_cache",
			},
			CreatedAt: time.Now(),
		},
	}
	
	err = memory.Add(ctx, testMemories)
	if err != nil {
		t.Fatalf("Failed to add KV cache memories: %v", err)
	}
	
	// Test getting cache
	retrieved, err := memory.GetCache(ctx, "cache1")
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}
	
	if retrieved == nil {
		t.Error("Expected retrieved cache, got nil")
	}
	
	// Test merging caches
	cache2 := &types.ActivationMemoryItem{
		ID:     "cache2",
		Memory: map[string]interface{}{
			"keys":   [][]float32{{9.0, 10.0}, {11.0, 12.0}},
			"values": [][]float32{{13.0, 14.0}, {15.0, 16.0}},
			"layers": 2,
		},
		Metadata: map[string]interface{}{
			"model_id": "test_model",
			"type":     "kv_cache",
		},
		CreatedAt: time.Now(),
	}
	
	err = memory.Add(ctx, []*types.ActivationMemoryItem{cache2})
	if err != nil {
		t.Fatalf("Failed to add second cache: %v", err)
	}
	
	// Test cache merging
	mergedCache, err := memory.MergeCache(ctx, []string{"cache1", "cache2"})
	if err != nil {
		t.Fatalf("Failed to merge caches: %v", err)
	}
	
	if mergedCache == nil {
		t.Error("Expected merged cache, got nil")
	}
	
	// Test compression
	err = memory.Compress(ctx, 0.5)
	if err != nil {
		t.Fatalf("Failed to compress memory: %v", err)
	}
	
	// Test optimization
	err = memory.Optimize(ctx)
	if err != nil {
		t.Fatalf("Failed to optimize memory: %v", err)
	}
}

// Test LoRA Memory
func TestLoRAMemory(t *testing.T) {
	logger := &MockLogger{}
	metrics := &MockMetrics{}
	config := &config.MemoryConfig{
		MemoryFilename: "test_lora_memory.json",
	}
	
	memory, err := parametric.NewLoRAMemory(config, logger, metrics)
	if err != nil {
		t.Fatalf("Failed to create LoRA memory: %v", err)
	}
	defer memory.Close()
	
	ctx := context.Background()
	
	// Test adding LoRA adapter
	adapterData := map[string]interface{}{
		"config": map[string]interface{}{
			"rank":          16,
			"alpha":         32.0,
			"target_layers": []string{"q_proj", "v_proj"},
		},
		"weights": map[string]interface{}{
			"weight_a": map[string]interface{}{
				"q_proj": []float32{1.0, 2.0, 3.0, 4.0},
				"v_proj": []float32{5.0, 6.0, 7.0, 8.0},
			},
			"weight_b": map[string]interface{}{
				"q_proj": []float32{9.0, 10.0, 11.0, 12.0},
				"v_proj": []float32{13.0, 14.0, 15.0, 16.0},
			},
		},
	}
	
	testMemories := []*types.ParametricMemoryItem{
		{
			ID:     "adapter1",
			Memory: adapterData,
			Metadata: map[string]interface{}{
				"adapter_type": "lora",
				"model_id":     "test_model",
				"name":         "test_adapter",
			},
			CreatedAt: time.Now(),
		},
	}
	
	err = memory.Add(ctx, testMemories)
	if err != nil {
		t.Fatalf("Failed to add LoRA adapter: %v", err)
	}
	
	// Test applying adapter
	err = memory.ApplyAdapter(ctx, "adapter1", "gpt2")
	if err != nil {
		t.Fatalf("Failed to apply adapter: %v", err)
	}
	
	// Test saving model parameters
	modelParams := map[string]interface{}{
		"weights": []float32{1.0, 2.0, 3.0},
		"biases":  []float32{0.1, 0.2, 0.3},
	}
	
	err = memory.SaveModelParameters(ctx, "test_model", modelParams)
	if err != nil {
		t.Fatalf("Failed to save model parameters: %v", err)
	}
	
	// Test loading model parameters
	loadedParams, err := memory.LoadModelParameters(ctx, "test_model")
	if err != nil {
		t.Fatalf("Failed to load model parameters: %v", err)
	}
	
	if loadedParams == nil {
		t.Error("Expected loaded parameters, got nil")
	}
	
	// Test fine-tuning job
	jobConfig := map[string]interface{}{
		"learning_rate": 0.001,
		"epochs":        10,
		"batch_size":    32,
	}
	
	jobID, err := memory.StartFineTuning(ctx, "gpt2", jobConfig)
	if err != nil {
		t.Fatalf("Failed to start fine-tuning: %v", err)
	}
	
	if jobID == "" {
		t.Error("Expected job ID, got empty string")
	}
	
	// Test monitoring fine-tuning
	jobStatus, err := memory.MonitorFineTuning(ctx, jobID)
	if err != nil {
		t.Fatalf("Failed to monitor fine-tuning: %v", err)
	}
	
	if jobStatus == nil {
		t.Error("Expected job status, got nil")
	}
	
	// Test creating adapter version
	err = memory.CreateVersion(ctx, "adapter1", "v1.0")
	if err != nil {
		t.Fatalf("Failed to create adapter version: %v", err)
	}
	
	// Test listing versions
	versions, err := memory.ListVersions(ctx, "adapter1")
	if err != nil {
		t.Fatalf("Failed to list versions: %v", err)
	}
	
	if len(versions) == 0 {
		t.Error("Expected versions, got none")
	}
	
	// Test creating checkpoint
	err = memory.CreateCheckpoint(ctx, "test_model", "checkpoint1")
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}
	
	// Test compression
	err = memory.Compress(ctx, 0.3)
	if err != nil {
		t.Fatalf("Failed to compress memory: %v", err)
	}
	
	// Test optimization
	err = memory.Optimize(ctx)
	if err != nil {
		t.Fatalf("Failed to optimize memory: %v", err)
	}
}

// Test Memory Factory
func TestMemoryFactory(t *testing.T) {
	logger := &MockLogger{}
	metrics := &MockMetrics{}
	config := &config.Config{}
	
	memoryFactory := factory.NewMemoryFactory(config, logger, metrics)
	
	// Register mock components
	memoryFactory.RegisterEmbedder("default", &MockEmbedder{})
	memoryFactory.RegisterLLM("default", &MockLLM{})
	
	// Test creating naive textual memory
	naiveConfig := &factory.MemoryConfig{
		Type:         factory.MemoryTypeNaiveTextual,
		Name:         "test_naive",
		Backend:      types.MemoryBackendNaive,
		Config:       &config.MemoryConfig{MemoryFilename: "naive.json"},
		EmbedderType: "",
		LLMType:      "",
	}
	
	naiveInstance, err := memoryFactory.CreateMemory(naiveConfig)
	if err != nil {
		t.Fatalf("Failed to create naive memory: %v", err)
	}
	
	if naiveInstance.Type != factory.MemoryTypeNaiveTextual {
		t.Errorf("Expected naive textual type, got %s", naiveInstance.Type)
	}
	
	// Test creating general textual memory
	generalConfig := &factory.MemoryConfig{
		Type:         factory.MemoryTypeGeneralTextual,
		Name:         "test_general",
		Backend:      types.MemoryBackendGeneral,
		Config:       &config.MemoryConfig{MemoryFilename: "general.json"},
		EmbedderType: "default",
		LLMType:      "default",
	}
	
	generalInstance, err := memoryFactory.CreateMemory(generalConfig)
	if err != nil {
		t.Fatalf("Failed to create general memory: %v", err)
	}
	
	if generalInstance.Type != factory.MemoryTypeGeneralTextual {
		t.Errorf("Expected general textual type, got %s", generalInstance.Type)
	}
	
	// Test creating KV cache memory
	kvConfig := &factory.MemoryConfig{
		Type:    factory.MemoryTypeKVCache,
		Name:    "test_kv",
		Backend: types.MemoryBackendKVCache,
		Config:  &config.MemoryConfig{MemoryFilename: "kv_cache.json"},
		LLMType: "default",
	}
	
	kvInstance, err := memoryFactory.CreateMemory(kvConfig)
	if err != nil {
		t.Fatalf("Failed to create KV cache memory: %v", err)
	}
	
	if kvInstance.Type != factory.MemoryTypeKVCache {
		t.Errorf("Expected KV cache type, got %s", kvInstance.Type)
	}
	
	// Test creating LoRA memory
	loraConfig := &factory.MemoryConfig{
		Type:    factory.MemoryTypeLoRA,
		Name:    "test_lora",
		Backend: types.MemoryBackendLoRA,
		Config:  &config.MemoryConfig{MemoryFilename: "lora.json"},
	}
	
	loraInstance, err := memoryFactory.CreateMemory(loraConfig)
	if err != nil {
		t.Fatalf("Failed to create LoRA memory: %v", err)
	}
	
	if loraInstance.Type != factory.MemoryTypeLoRA {
		t.Errorf("Expected LoRA type, got %s", loraInstance.Type)
	}
	
	// Test creating memory from template
	templateInstance, err := memoryFactory.CreateMemoryFromTemplate("semantic_textual", map[string]interface{}{
		"name":         "test_template",
		"embedder_type": "default",
		"llm_type":     "default",
	})
	if err != nil {
		t.Fatalf("Failed to create memory from template: %v", err)
	}
	
	if templateInstance.Name != "test_template" {
		t.Errorf("Expected name 'test_template', got %s", templateInstance.Name)
	}
	
	// Test listing available types
	availableTypes := memoryFactory.ListAvailableTypes()
	if len(availableTypes) == 0 {
		t.Error("Expected available types, got none")
	}
	
	// Test getting capabilities
	capabilities, err := memoryFactory.GetMemoryCapabilities(factory.MemoryTypeTreeTextual)
	if err != nil {
		t.Fatalf("Failed to get capabilities: %v", err)
	}
	
	if capabilities == nil {
		t.Error("Expected capabilities, got nil")
	}
	
	// Test configuration validation
	invalidConfig := &factory.MemoryConfig{
		Type: "invalid_type",
		Name: "test",
	}
	
	err = memoryFactory.ValidateConfiguration(invalidConfig)
	if err == nil {
		t.Error("Expected validation error for invalid config")
	}
}

// Test persistence operations
func TestMemoryPersistence(t *testing.T) {
	logger := &MockLogger{}
	metrics := &MockMetrics{}
	config := &config.MemoryConfig{
		MemoryFilename: "test_persistence.json",
	}
	
	// Create temporary directory
	tempDir := "/tmp/memgos_test_" + time.Now().Format("20060102150405")
	
	memory, err := textual.NewNaiveTextMemory(config, logger, metrics)
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}
	defer memory.Close()
	
	ctx := context.Background()
	
	// Add test data
	testMemories := []*types.TextualMemoryItem{
		{
			ID:        "persist1",
			Memory:    "Persistent memory test data",
			Metadata:  map[string]interface{}{"test": "persistence"},
			CreatedAt: time.Now(),
		},
	}
	
	err = memory.Add(ctx, testMemories)
	if err != nil {
		t.Fatalf("Failed to add memories: %v", err)
	}
	
	// Test dump
	err = memory.Dump(ctx, tempDir)
	if err != nil {
		t.Fatalf("Failed to dump memories: %v", err)
	}
	
	// Create new memory instance and load
	memory2, err := textual.NewNaiveTextMemory(config, logger, metrics)
	if err != nil {
		t.Fatalf("Failed to create second memory: %v", err)
	}
	defer memory2.Close()
	
	err = memory2.Load(ctx, tempDir)
	if err != nil {
		t.Fatalf("Failed to load memories: %v", err)
	}
	
	// Verify loaded data
	retrieved, err := memory2.Get(ctx, "persist1")
	if err != nil {
		t.Fatalf("Failed to get loaded memory: %v", err)
	}
	
	if retrieved.Memory != testMemories[0].Memory {
		t.Errorf("Expected loaded memory %s, got %s", testMemories[0].Memory, retrieved.Memory)
	}
}

// Benchmark tests
func BenchmarkNaiveMemoryAdd(b *testing.B) {
	logger := &MockLogger{}
	metrics := &MockMetrics{}
	config := &config.MemoryConfig{MemoryFilename: "bench.json"}
	
	memory, err := textual.NewNaiveTextMemory(config, logger, metrics)
	if err != nil {
		b.Fatalf("Failed to create memory: %v", err)
	}
	defer memory.Close()
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testMemory := &types.TextualMemoryItem{
			ID:        "bench" + string(rune(i)),
			Memory:    "Benchmark memory content for testing performance",
			Metadata:  map[string]interface{}{"benchmark": true},
			CreatedAt: time.Now(),
		}
		
		err := memory.Add(ctx, []*types.TextualMemoryItem{testMemory})
		if err != nil {
			b.Fatalf("Failed to add memory: %v", err)
		}
	}
}

func BenchmarkNaiveMemorySearch(b *testing.B) {
	logger := &MockLogger{}
	metrics := &MockMetrics{}
	config := &config.MemoryConfig{MemoryFilename: "bench.json"}
	
	memory, err := textual.NewNaiveTextMemory(config, logger, metrics)
	if err != nil {
		b.Fatalf("Failed to create memory: %v", err)
	}
	defer memory.Close()
	
	ctx := context.Background()
	
	// Add test data
	for i := 0; i < 1000; i++ {
		testMemory := &types.TextualMemoryItem{
			ID:        "bench" + string(rune(i)),
			Memory:    "This is benchmark memory content for testing search performance",
			Metadata:  map[string]interface{}{"index": i},
			CreatedAt: time.Now(),
		}
		
		err := memory.Add(ctx, []*types.TextualMemoryItem{testMemory})
		if err != nil {
			b.Fatalf("Failed to add memory: %v", err)
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := memory.Search(ctx, "benchmark content", 10, textual.SearchModeFast, textual.MemoryTypeAll)
		if err != nil {
			b.Fatalf("Failed to search: %v", err)
		}
	}
}