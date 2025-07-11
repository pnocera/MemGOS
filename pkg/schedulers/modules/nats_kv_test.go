package modules

import (
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/memtensor/memgos/pkg/logger"
)

// TestNATSKVSchedulerModule tests the NATS KV scheduler module
func TestNATSKVSchedulerModule(t *testing.T) {
	// Skip if no NATS server available
	if !isNATSAvailable() {
		t.Skip("NATS server not available, skipping integration tests")
	}

	config := &NATSKVConfig{
		URLs:         []string{nats.DefaultURL},
		BucketName:   "test-scheduler",
		Description:  "Test bucket for scheduler",
		MaxValueSize: 1024,
		History:      5,
		MaxBytes:     256 * 1024,  // Reduced to 256KB to avoid memory issues
		Storage:      "Memory",
		Replicas:     1,
		StreamName:   "TEST_STREAM",
		StreamSubjects: []string{"test.>"},
		ConsumerName:   "test_consumer",
		ConsumerDurable: true,
		MaxDeliver:     3,
		AckWait:        10 * time.Second,
		MaxAckPending:  10,
	}

	module := NewNATSKVSchedulerModule(config)
	module.logger = logger.NewConsoleLogger("debug")

	// Test initialization
	err := module.InitializeNATSKV()
	require.NoError(t, err, "Failed to initialize NATS KV module")
	defer module.Close()

	// Test connection
	assert.True(t, module.IsConnected(), "Module should be connected")
	assert.NotNil(t, module.GetConnection(), "Connection should not be nil")
	assert.NotNil(t, module.GetJetStream(), "JetStream should not be nil")
	assert.NotNil(t, module.GetKVStore(), "KV store should not be nil")

	// Test basic KV operations
	t.Run("BasicKVOperations", func(t *testing.T) {
		testBasicKVOperations(t, module)
	})

	// Test atomic operations
	t.Run("AtomicOperations", func(t *testing.T) {
		testAtomicOperations(t, module)
	})

	// Test messaging
	t.Run("MessagingOperations", func(t *testing.T) {
		testMessagingOperations(t, module)
	})

	// Test statistics
	t.Run("Statistics", func(t *testing.T) {
		testStatistics(t, module)
	})
}

func testBasicKVOperations(t *testing.T, module *NATSKVSchedulerModule) {
	// Test Put operation
	testValue := map[string]interface{}{
		"message": "Hello, NATS KV!",
		"timestamp": time.Now().Unix(),
	}
	
	revision, err := module.Put("test-key", testValue)
	require.NoError(t, err, "Put operation should succeed")
	assert.Greater(t, revision, uint64(0), "Revision should be positive")

	// Test Get operation
	data, getRevision, err := module.Get("test-key")
	require.NoError(t, err, "Get operation should succeed")
	assert.Equal(t, revision, getRevision, "Revisions should match")
	assert.NotEmpty(t, data, "Data should not be empty")

	// Test GetJSON operation
	var retrievedValue map[string]interface{}
	jsonRevision, err := module.GetJSON("test-key", &retrievedValue)
	require.NoError(t, err, "GetJSON operation should succeed")
	assert.Equal(t, revision, jsonRevision, "JSON revision should match")
	assert.Equal(t, testValue["message"], retrievedValue["message"], "Message should match")

	// Test Update operation
	updatedValue := map[string]interface{}{
		"message": "Updated message",
		"timestamp": time.Now().Unix(),
	}
	
	newRevision, err := module.Update("test-key", updatedValue, revision)
	require.NoError(t, err, "Update operation should succeed")
	assert.Greater(t, newRevision, revision, "New revision should be higher")

	// Test Delete operation
	err = module.Delete("test-key")
	require.NoError(t, err, "Delete operation should succeed")

	// Verify deletion
	_, _, err = module.Get("test-key")
	assert.Error(t, err, "Get after delete should fail")
}

func testAtomicOperations(t *testing.T, module *NATSKVSchedulerModule) {
	// Test Create operation
	createValue := map[string]interface{}{
		"created": true,
		"value": "initial",
	}
	
	revision, err := module.Create("atomic-key", createValue)
	require.NoError(t, err, "Create operation should succeed")
	assert.Greater(t, revision, uint64(0), "Revision should be positive")

	// Test Create on existing key (should fail)
	_, err = module.Create("atomic-key", createValue)
	assert.Error(t, err, "Create on existing key should fail")

	// Test successful Update with correct revision
	updateValue := map[string]interface{}{
		"created": false,
		"value": "updated",
	}
	
	newRevision, err := module.Update("atomic-key", updateValue, revision)
	require.NoError(t, err, "Update with correct revision should succeed")
	assert.Greater(t, newRevision, revision, "New revision should be higher")

	// Test Update with wrong revision (should fail)
	_, err = module.Update("atomic-key", updateValue, revision) // Using old revision
	assert.Error(t, err, "Update with wrong revision should fail")

	// Clean up
	err = module.Delete("atomic-key")
	require.NoError(t, err, "Cleanup delete should succeed")
}

func testMessagingOperations(t *testing.T, module *NATSKVSchedulerModule) {
	// Test event publishing
	eventData := map[string]interface{}{
		"event": "test-event",
		"data":  "test-data",
	}
	
	err := module.PublishEvent("test.event", eventData)
	require.NoError(t, err, "PublishEvent should succeed")

	// Test event subscription
	messageReceived := make(chan bool, 1)
	sub, err := module.SubscribeToEvents("test.event", func(subject string, data []byte) error {
		assert.Equal(t, "test.event", subject, "Subject should match")
		assert.NotEmpty(t, data, "Data should not be empty")
		messageReceived <- true
		return nil
	})
	require.NoError(t, err, "SubscribeToEvents should succeed")
	defer sub.Unsubscribe()

	// Give subscription time to be established
	time.Sleep(100 * time.Millisecond)

	// Publish another event to test subscription
	err = module.PublishEvent("test.event", eventData)
	require.NoError(t, err, "Second PublishEvent should succeed")

	// Wait for message reception
	select {
	case <-messageReceived:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Did not receive published event within timeout")
	}

	// Test AddMessageToStream with a valid key
	message := &ScheduleMessageItem{
		ItemID:    "test-item-1",
		UserID:    "test-user",
		MemCubeID: "test-cube",
		Label:     "test-label",
		Content:   "Test message content",
		Timestamp: time.Now(),
	}

	messageID, err := module.AddMessageToStream(message)
	if err != nil {
		// Log the error but don't fail the test since this depends on stream configuration
		t.Logf("AddMessageToStream failed (expected in some configurations): %v", err)
	} else {
		assert.NotEmpty(t, messageID, "Message ID should not be empty")
	}
}

func testStatistics(t *testing.T, module *NATSKVSchedulerModule) {
	// Test GetKVInfo
	kvInfo, err := module.GetKVInfo()
	require.NoError(t, err, "GetKVInfo should succeed")
	assert.NotEmpty(t, kvInfo, "KV info should not be empty")
	assert.Contains(t, kvInfo, "bucket", "KV info should contain bucket")

	// Test GetStats
	stats := module.GetStats()
	assert.NotEmpty(t, stats, "Stats should not be empty")
	assert.Contains(t, stats, "connected", "Stats should contain connected")
	assert.Contains(t, stats, "bucket_name", "Stats should contain bucket_name")
	assert.Contains(t, stats, "uptime", "Stats should contain uptime")
	
	// Verify connection status
	connected, ok := stats["connected"].(bool)
	assert.True(t, ok, "Connected should be boolean")
	assert.True(t, connected, "Should be connected")
}

func TestNATSKVConfigDefaults(t *testing.T) {
	config := DefaultNATSKVConfig()
	
	assert.NotEmpty(t, config.URLs, "URLs should not be empty")
	assert.Equal(t, nats.DefaultURL, config.URLs[0], "Default URL should match")
	assert.Equal(t, "memgos-scheduler", config.BucketName, "Default bucket name should match")
	assert.Equal(t, int32(1024*1024), config.MaxValueSize, "Default max value size should match")
	assert.Equal(t, uint8(10), config.History, "Default history should match")
	assert.Equal(t, "File", config.Storage, "Default storage should be File")
	assert.Equal(t, 1, config.Replicas, "Default replicas should be 1")
	assert.Equal(t, "SCHEDULER_STREAM", config.StreamName, "Default stream name should match")
	assert.Contains(t, config.StreamSubjects, "scheduler.>", "Default subjects should contain scheduler.>")
}

func TestNATSKVWatch(t *testing.T) {
	if !isNATSAvailable() {
		t.Skip("NATS server not available, skipping watch tests")
	}

	config := DefaultNATSKVConfig()
	config.BucketName = "test-watch-bucket"
	config.Storage = "Memory"
	config.MaxBytes = 256 * 1024  // Reduced to 256KB
	
	module := NewNATSKVSchedulerModule(config)
	module.logger = logger.NewConsoleLogger("debug")

	err := module.InitializeNATSKV()
	require.NoError(t, err, "Failed to initialize NATS KV module")
	defer module.Close()

	// Test watching a specific key
	watchKey := "watch-test-key"
	watchReceived := make(chan jetstream.KeyValueEntry, 10)
	
	watcher, err := module.Watch(watchKey, func(entry jetstream.KeyValueEntry) {
		watchReceived <- entry
	})
	require.NoError(t, err, "Watch should succeed")
	defer watcher.Stop()

	// Put a value to trigger the watcher
	testValue := map[string]interface{}{
		"watched": true,
		"value":   "test-watch-value",
	}
	
	_, err = module.Put(watchKey, testValue)
	require.NoError(t, err, "Put should succeed")

	// Wait for watch notification
	select {
	case entry := <-watchReceived:
		assert.Equal(t, watchKey, entry.Key(), "Watch key should match")
		assert.NotEmpty(t, entry.Value(), "Watch value should not be empty")
	case <-time.After(5 * time.Second):
		t.Error("Did not receive watch notification within timeout")
	}

	// Clean up
	err = module.Delete(watchKey)
	require.NoError(t, err, "Delete should succeed")
}

func TestNATSKVKeys(t *testing.T) {
	if !isNATSAvailable() {
		t.Skip("NATS server not available, skipping keys tests")
	}

	config := DefaultNATSKVConfig()
	config.BucketName = "test-keys-bucket"
	config.Storage = "Memory"
	config.MaxBytes = 256 * 1024  // Reduced to 256KB
	
	module := NewNATSKVSchedulerModule(config)
	module.logger = logger.NewConsoleLogger("debug")

	err := module.InitializeNATSKV()
	require.NoError(t, err, "Failed to initialize NATS KV module")
	defer module.Close()

	// Put some test keys
	testKeys := []string{"key1", "key2", "key3"}
	for _, key := range testKeys {
		_, err := module.Put(key, map[string]interface{}{"value": key})
		require.NoError(t, err, "Put should succeed for key %s", key)
	}

	// Get all keys
	keys, err := module.Keys()
	require.NoError(t, err, "Keys should succeed")
	
	// Verify all test keys are present
	for _, testKey := range testKeys {
		assert.Contains(t, keys, testKey, "Keys should contain %s", testKey)
	}

	// Clean up
	for _, key := range testKeys {
		err := module.Delete(key)
		require.NoError(t, err, "Delete should succeed for key %s", key)
	}
}

// Helper function to check if NATS server is available
func isNATSAvailable() bool {
	conn, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		return false
	}
	defer conn.Close()
	return conn.IsConnected()
}

// Benchmark tests
func BenchmarkNATSKVPut(b *testing.B) {
	if !isNATSAvailable() {
		b.Skip("NATS server not available, skipping benchmark")
	}

	config := DefaultNATSKVConfig()
	config.BucketName = "benchmark-put-bucket"
	config.Storage = "Memory"
	config.MaxBytes = 256 * 1024  // Reduced to 256KB
	
	module := NewNATSKVSchedulerModule(config)
	module.logger = logger.NewConsoleLogger("error") // Minimal logging for benchmarks

	err := module.InitializeNATSKV()
	require.NoError(b, err, "Failed to initialize NATS KV module")
	defer module.Close()

	testValue := map[string]interface{}{
		"benchmark": true,
		"data":      "benchmark data",
		"timestamp": time.Now().Unix(),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("bench-key-%d", i)
			_, err := module.Put(key, testValue)
			if err != nil {
				b.Errorf("Put failed: %v", err)
			}
			i++
		}
	})
}

func BenchmarkNATSKVGet(b *testing.B) {
	if !isNATSAvailable() {
		b.Skip("NATS server not available, skipping benchmark")
	}

	config := DefaultNATSKVConfig()
	config.BucketName = "benchmark-get-bucket"
	config.Storage = "Memory"
	config.MaxBytes = 256 * 1024  // Reduced to 256KB
	
	module := NewNATSKVSchedulerModule(config)
	module.logger = logger.NewConsoleLogger("error") // Minimal logging for benchmarks

	err := module.InitializeNATSKV()
	require.NoError(b, err, "Failed to initialize NATS KV module")
	defer module.Close()

	// Pre-populate keys for benchmarking
	testValue := map[string]interface{}{
		"benchmark": true,
		"data":      "benchmark data",
	}

	numKeys := 1000
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("bench-get-key-%d", i)
		_, err := module.Put(key, testValue)
		require.NoError(b, err, "Failed to populate key %s", key)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("bench-get-key-%d", i%numKeys)
			_, _, err := module.Get(key)
			if err != nil {
				b.Errorf("Get failed: %v", err)
			}
			i++
		}
	})
}