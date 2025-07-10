package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/memtensor/memgos/pkg/types"
)

func TestBaseConfig(t *testing.T) {
	t.Run("NewBaseConfig", func(t *testing.T) {
		config := NewBaseConfig()
		assert.NotNil(t, config)
		assert.NotNil(t, config.validator)
	})

	t.Run("Validate", func(t *testing.T) {
		config := NewBaseConfig()
		config.ModelSchema = "test-schema"
		
		err := config.Validate()
		assert.NoError(t, err)
		
		// Test missing required field
		config.ModelSchema = ""
		err = config.Validate()
		assert.Error(t, err)
	})

	t.Run("Get", func(t *testing.T) {
		config := &BaseConfig{
			ModelSchema: "test-schema",
		}
		
		value := config.Get("model_schema", "default")
		assert.Equal(t, "test-schema", value)
		
		value = config.Get("nonexistent", "default")
		assert.Equal(t, "default", value)
	})
}

func TestJSONConfigOperations(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.json")

	t.Run("ToJSONFile", func(t *testing.T) {
		config := NewLLMConfig()
		config.Backend = types.BackendOpenAI
		config.Model = "gpt-3.5-turbo"
		config.APIKey = "test-key"
		config.MaxTokens = 2048

		err := config.ToJSONFile(configPath)
		assert.NoError(t, err)
		assert.FileExists(t, configPath)
	})

	t.Run("FromJSONFile", func(t *testing.T) {
		config := NewLLMConfig()
		err := config.FromJSONFile(configPath)
		assert.NoError(t, err)
		
		assert.Equal(t, types.BackendOpenAI, config.Backend)
		assert.Equal(t, "gpt-3.5-turbo", config.Model)
		assert.Equal(t, "test-key", config.APIKey)
		assert.Equal(t, 2048, config.MaxTokens)
	})

	t.Run("FromJSONFile_NonExistentFile", func(t *testing.T) {
		config := NewLLMConfig()
		err := config.FromJSONFile("/nonexistent/path.json")
		assert.Error(t, err)
	})
}

func TestYAMLConfigOperations(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yaml")

	t.Run("ToYAMLFile", func(t *testing.T) {
		config := NewEmbedderConfig()
		config.Backend = types.BackendOpenAI
		config.Model = "text-embedding-ada-002"
		config.Dimension = 1536

		err := config.ToYAMLFile(configPath)
		assert.NoError(t, err)
		assert.FileExists(t, configPath)
	})

	t.Run("FromYAMLFile", func(t *testing.T) {
		config := NewEmbedderConfig()
		err := config.FromYAMLFile(configPath)
		assert.NoError(t, err)
		
		assert.Equal(t, types.BackendOpenAI, config.Backend)
		assert.Equal(t, "text-embedding-ada-002", config.Model)
		assert.Equal(t, 1536, config.Dimension)
	})
}

func TestLLMConfig(t *testing.T) {
	t.Run("NewLLMConfig", func(t *testing.T) {
		config := NewLLMConfig()
		assert.NotNil(t, config)
		assert.Equal(t, 1024, config.MaxTokens)
		assert.Equal(t, 0.7, config.Temperature)
		assert.Equal(t, 0.9, config.TopP)
		assert.Equal(t, 30*time.Second, config.Timeout)
	})

	t.Run("Validate", func(t *testing.T) {
		config := NewLLMConfig()
		config.Backend = types.BackendOpenAI
		config.Model = "gpt-3.5-turbo"
		config.ModelSchema = "test-schema"
		
		err := config.Validate()
		assert.NoError(t, err)

		// Test invalid backend
		config.Backend = "invalid"
		err = config.Validate()
		assert.Error(t, err)

		// Test missing model
		config.Backend = types.BackendOpenAI
		config.Model = ""
		err = config.Validate()
		assert.Error(t, err)
	})
}

func TestEmbedderConfig(t *testing.T) {
	t.Run("NewEmbedderConfig", func(t *testing.T) {
		config := NewEmbedderConfig()
		assert.NotNil(t, config)
		assert.Equal(t, 768, config.Dimension)
		assert.Equal(t, 30*time.Second, config.Timeout)
	})

	t.Run("Validate", func(t *testing.T) {
		config := NewEmbedderConfig()
		config.Backend = types.BackendOpenAI
		config.Model = "text-embedding-ada-002"
		config.ModelSchema = "test-schema"
		
		err := config.Validate()
		assert.NoError(t, err)
	})
}

func TestVectorDBConfig(t *testing.T) {
	t.Run("NewVectorDBConfig", func(t *testing.T) {
		config := NewVectorDBConfig()
		assert.NotNil(t, config)
		assert.Equal(t, "localhost", config.Host)
		assert.Equal(t, 6333, config.Port)
		assert.Equal(t, 768, config.Dimension)
		assert.Equal(t, 30*time.Second, config.Timeout)
	})

	t.Run("Validate", func(t *testing.T) {
		config := NewVectorDBConfig()
		config.Backend = types.BackendQdrant
		config.Host = "localhost"
		config.Port = 6333
		config.Collection = "test-collection"
		config.Dimension = 768
		config.ModelSchema = "test-schema"
		
		err := config.Validate()
		assert.NoError(t, err)

		// Test invalid port
		config.Port = 0
		err = config.Validate()
		assert.Error(t, err)

		// Test missing collection
		config.Port = 6333
		config.Collection = ""
		err = config.Validate()
		assert.Error(t, err)
	})
}

func TestGraphDBConfig(t *testing.T) {
	t.Run("NewGraphDBConfig", func(t *testing.T) {
		config := NewGraphDBConfig()
		assert.NotNil(t, config)
		assert.Equal(t, "bolt://localhost:7687", config.URI)
		assert.Equal(t, "neo4j", config.Username)
		assert.Equal(t, "neo4j", config.Database)
		assert.Equal(t, 30*time.Second, config.Timeout)
	})

	t.Run("Validate", func(t *testing.T) {
		config := NewGraphDBConfig()
		config.Backend = types.BackendNeo4j
		config.URI = "bolt://localhost:7687"
		config.Username = "neo4j"
		config.Password = "password"
		config.ModelSchema = "test-schema"
		
		err := config.Validate()
		assert.NoError(t, err)

		// Test missing password
		config.Password = ""
		err = config.Validate()
		assert.Error(t, err)
	})
}

func TestMemoryConfig(t *testing.T) {
	t.Run("NewMemoryConfig", func(t *testing.T) {
		config := NewMemoryConfig()
		assert.NotNil(t, config)
		assert.Equal(t, types.MemoryBackendNaive, config.Backend)
		assert.Equal(t, "memory.json", config.MemoryFilename)
		assert.Equal(t, 5, config.TopK)
		assert.Equal(t, 1000, config.ChunkSize)
		assert.Equal(t, 200, config.ChunkOverlap)
	})

	t.Run("Validate", func(t *testing.T) {
		config := NewMemoryConfig()
		config.Backend = types.MemoryBackendNaive
		config.MemoryFilename = "test.json"
		config.ModelSchema = "test-schema"
		
		err := config.Validate()
		assert.NoError(t, err)
	})
}

func TestMemCubeConfig(t *testing.T) {
	t.Run("NewMemCubeConfig", func(t *testing.T) {
		config := NewMemCubeConfig()
		assert.NotNil(t, config)
		assert.NotNil(t, config.TextMem)
		assert.NotNil(t, config.ActMem)
		assert.NotNil(t, config.ParaMem)
	})

	t.Run("Validate", func(t *testing.T) {
		config := NewMemCubeConfig()
		config.ModelSchema = "test-schema"
		
		err := config.Validate()
		assert.NoError(t, err)
	})
}

func TestSchedulerConfig(t *testing.T) {
	t.Run("NewSchedulerConfig", func(t *testing.T) {
		config := NewSchedulerConfig()
		assert.NotNil(t, config)
		assert.False(t, config.Enabled)
		assert.Equal(t, "localhost", config.RedisHost)
		assert.Equal(t, 6379, config.RedisPort)
		assert.Equal(t, 0, config.RedisDB)
		assert.Equal(t, 4, config.WorkerCount)
		assert.Equal(t, 1000, config.QueueSize)
		assert.Equal(t, 3, config.RetryAttempts)
		assert.Equal(t, 5*time.Second, config.RetryDelay)
	})
}

func TestMOSConfig(t *testing.T) {
	t.Run("NewMOSConfig", func(t *testing.T) {
		config := NewMOSConfig()
		assert.NotNil(t, config)
		assert.NotNil(t, config.ChatModel)
		assert.NotNil(t, config.MemReader)
		assert.NotNil(t, config.MemScheduler)
		assert.True(t, config.EnableTextualMemory)
		assert.False(t, config.EnableActivationMemory)
		assert.False(t, config.EnableParametricMemory)
		assert.False(t, config.EnableMemScheduler)
		assert.Equal(t, 5, config.TopK)
		assert.Equal(t, "info", config.LogLevel)
		assert.True(t, config.MetricsEnabled)
		assert.Equal(t, 9090, config.MetricsPort)
		assert.True(t, config.HealthCheckEnabled)
		assert.Equal(t, 8080, config.HealthCheckPort)
	})

	t.Run("Validate", func(t *testing.T) {
		config := NewMOSConfig()
		config.UserID = "test-user"
		config.SessionID = "test-session"
		config.ModelSchema = "test-schema"
		config.ChatModel.Backend = types.BackendOpenAI
		config.ChatModel.Model = "gpt-3.5-turbo"
		config.ChatModel.ModelSchema = "test-schema"
		
		err := config.Validate()
		assert.NoError(t, err)
	})
}

func TestAPIConfig(t *testing.T) {
	t.Run("NewAPIConfig", func(t *testing.T) {
		config := NewAPIConfig()
		assert.NotNil(t, config)
		assert.Equal(t, "localhost", config.Host)
		assert.Equal(t, 8000, config.Port)
		assert.False(t, config.TLSEnabled)
		assert.True(t, config.CORSEnabled)
		assert.Equal(t, []string{"*"}, config.CORSOrigins)
		assert.Equal(t, 100, config.RateLimit)
		assert.Equal(t, 30*time.Second, config.Timeout)
	})

	t.Run("Validate", func(t *testing.T) {
		config := NewAPIConfig()
		config.Host = "localhost"
		config.Port = 8080
		config.ModelSchema = "test-schema"
		
		err := config.Validate()
		assert.NoError(t, err)

		// Test invalid port
		config.Port = 0
		err = config.Validate()
		assert.Error(t, err)
	})
}

func TestConfigManager(t *testing.T) {
	t.Run("NewConfigManager", func(t *testing.T) {
		cm := NewConfigManager()
		assert.NotNil(t, cm)
	})

	t.Run("SetAndGet", func(t *testing.T) {
		cm := NewConfigManager()
		
		err := cm.Set("test_key", "test_value")
		assert.NoError(t, err)
		
		value := cm.Get("test_key")
		assert.Equal(t, "test_value", value)
		
		// Test non-existent key
		value = cm.Get("nonexistent")
		assert.Nil(t, value)
	})

	t.Run("LoadAndSave", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")
		
		// Create a test config file
		testConfig := `
test_key: test_value
nested:
  key: value
`
		err := os.WriteFile(configPath, []byte(testConfig), 0644)
		require.NoError(t, err)
		
		cm := NewConfigManager()
		ctx := context.Background()
		
		err = cm.Load(ctx, configPath)
		assert.NoError(t, err)
		
		value := cm.Get("test_key")
		assert.Equal(t, "test_value", value)
		
		// Test save
		savePath := filepath.Join(tempDir, "saved_config.yaml")
		err = cm.Save(ctx, savePath)
		assert.NoError(t, err)
		assert.FileExists(t, savePath)
	})

	t.Run("Watch", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "watch_config.yaml")
		
		// Create initial config
		testConfig := `test_key: initial_value`
		err := os.WriteFile(configPath, []byte(testConfig), 0644)
		require.NoError(t, err)
		
		cm := NewConfigManager()
		ctx := context.Background()
		
		err = cm.Load(ctx, configPath)
		assert.NoError(t, err)
		
		// Set up watch with a simple callback
		called := false
		callback := func(key string, value interface{}) {
			called = true
		}
		
		err = cm.Watch(ctx, callback)
		assert.NoError(t, err)
		
		// Note: We can't easily test the file watch functionality in a unit test
		// as it requires actual file system changes and timing
		assert.False(t, called) // No changes yet
	})
}

func TestUtilityFunctions(t *testing.T) {
	t.Run("LoadFromEnv", func(t *testing.T) {
		os.Setenv("MEMGOS_TEST_KEY", "test_value")
		defer os.Unsetenv("MEMGOS_TEST_KEY")
		
		v := LoadFromEnv("MEMGOS")
		assert.NotNil(t, v)
		
		value := v.Get("test.key")
		assert.Equal(t, "test_value", value)
	})

	t.Run("MergeConfigs", func(t *testing.T) {
		config1 := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		}
		
		config2 := map[string]interface{}{
			"key2": "overridden",
			"key3": "value3",
		}
		
		merged := MergeConfigs(config1, config2)
		
		assert.Equal(t, "value1", merged["key1"])
		assert.Equal(t, "overridden", merged["key2"])
		assert.Equal(t, "value3", merged["key3"])
	})
}

// Benchmark tests
func BenchmarkBaseConfigValidate(b *testing.B) {
	config := NewBaseConfig()
	config.ModelSchema = "test-schema"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Validate()
	}
}

func BenchmarkConfigManagerGet(b *testing.B) {
	cm := NewConfigManager()
	cm.Set("benchmark_key", "benchmark_value")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cm.Get("benchmark_key")
	}
}

func BenchmarkMergeConfigs(b *testing.B) {
	config1 := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	
	config2 := map[string]interface{}{
		"key4": "value4",
		"key5": "value5",
		"key6": "value6",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = MergeConfigs(config1, config2)
	}
}

// Error condition tests
func TestConfigErrorConditions(t *testing.T) {
	t.Run("InvalidJSONFile", func(t *testing.T) {
		tempDir := t.TempDir()
		invalidPath := filepath.Join(tempDir, "invalid.json")
		
		// Create invalid JSON
		err := os.WriteFile(invalidPath, []byte(`{invalid json`), 0644)
		require.NoError(t, err)
		
		config := NewLLMConfig()
		err = config.FromJSONFile(invalidPath)
		assert.Error(t, err)
	})

	t.Run("InvalidYAMLFile", func(t *testing.T) {
		tempDir := t.TempDir()
		invalidPath := filepath.Join(tempDir, "invalid.yaml")
		
		// Create invalid YAML
		err := os.WriteFile(invalidPath, []byte(`invalid: yaml: content: [unclosed`), 0644)
		require.NoError(t, err)
		
		config := NewEmbedderConfig()
		err = config.FromYAMLFile(invalidPath)
		assert.Error(t, err)
	})

	t.Run("ReadOnlyDirectory", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Running as root, cannot test read-only directory")
		}
		
		tempDir := t.TempDir()
		readOnlyDir := filepath.Join(tempDir, "readonly")
		err := os.Mkdir(readOnlyDir, 0444)
		require.NoError(t, err)
		
		config := NewAPIConfig()
		config.ModelSchema = "test-schema"
		
		// This should fail due to permission issues
		err = config.ToJSONFile(filepath.Join(readOnlyDir, "config.json"))
		assert.Error(t, err)
	})
}