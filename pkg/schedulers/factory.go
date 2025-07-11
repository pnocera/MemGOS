package schedulers

import (
	"fmt"
	"strings"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/logger"
	"github.com/memtensor/memgos/pkg/schedulers/modules"
)

// SchedulerType represents different types of schedulers
type SchedulerType string

const (
	GeneralSchedulerType SchedulerType = "general"
	BaseSchedulerType    SchedulerType = "base"
)

// Scheduler interface defines the common operations for all schedulers
type Scheduler interface {
	// Core operations
	InitializeModules(chatLLM interfaces.LLM) error
	Start() error
	Stop() error
	IsRunning() bool
	
	// Message operations
	SubmitMessages(messages ...*modules.ScheduleMessageItem) error
	SubmitWebLogs(messages ...*modules.ScheduleLogForWebItem) error
	GetWebLogMessages() []*modules.ScheduleLogForWebItem
	
	// Session management
	SetCurrentSession(userID, memCubeID string, memCube interface{})
	GetCurrentSession() (string, string, interface{})
	
	// Component access
	GetRetriever() *modules.SchedulerRetriever
	GetMonitor() *modules.SchedulerMonitor
	GetDispatcher() *modules.SchedulerDispatcher
	
	// Statistics and monitoring
	GetStats() map[string]interface{}
	
	// Utility
	CreateAutofilledLogItem(logTitle, logContent, label string) *modules.ScheduleLogForWebItem
}

// SchedulerFactory creates and configures different types of schedulers
type SchedulerFactory struct {
	logger interfaces.Logger
}

// NewSchedulerFactory creates a new scheduler factory
func NewSchedulerFactory() *SchedulerFactory {
	return &SchedulerFactory{
		logger: logger.NewConsoleLogger("info"),
	}
}

// CreateScheduler creates a scheduler of the specified type with the given configuration
func (f *SchedulerFactory) CreateScheduler(schedulerType SchedulerType, config interface{}) (Scheduler, error) {
	f.logger.Info("Creating scheduler", map[string]interface{}{"type": string(schedulerType)})
	
	switch schedulerType {
	case GeneralSchedulerType:
		return f.createGeneralScheduler(config)
	case BaseSchedulerType:
		return f.createBaseScheduler(config)
	default:
		return nil, fmt.Errorf("unsupported scheduler type: %s", schedulerType)
	}
}

// createGeneralScheduler creates a GeneralScheduler with the provided configuration
func (f *SchedulerFactory) createGeneralScheduler(config interface{}) (Scheduler, error) {
	var generalConfig *GeneralSchedulerConfig
	
	switch c := config.(type) {
	case *GeneralSchedulerConfig:
		generalConfig = c
	case map[string]interface{}:
		// Convert map to GeneralSchedulerConfig
		var err error
		generalConfig, err = f.mapToGeneralSchedulerConfig(c)
		if err != nil {
			return nil, fmt.Errorf("failed to convert config map: %w", err)
		}
	case nil:
		generalConfig = DefaultGeneralSchedulerConfig()
	default:
		return nil, fmt.Errorf("invalid config type for GeneralScheduler: %T", config)
	}
	
	scheduler := NewGeneralScheduler(generalConfig)
	f.logger.Info("GeneralScheduler created successfully", map[string]interface{}{
		"top_k": generalConfig.TopK,
		"top_n": generalConfig.TopN,
		"activation_mem_size": generalConfig.ActivationMemSize})
	
	return scheduler, nil
}

// createBaseScheduler creates a BaseScheduler with the provided configuration
func (f *SchedulerFactory) createBaseScheduler(config interface{}) (Scheduler, error) {
	var baseConfig *BaseSchedulerConfig
	
	switch c := config.(type) {
	case *BaseSchedulerConfig:
		baseConfig = c
	case map[string]interface{}:
		// Convert map to BaseSchedulerConfig
		var err error
		baseConfig, err = f.mapToBaseSchedulerConfig(c)
		if err != nil {
			return nil, fmt.Errorf("failed to convert config map: %w", err)
		}
	case nil:
		baseConfig = DefaultBaseSchedulerConfig()
	default:
		return nil, fmt.Errorf("invalid config type for BaseScheduler: %T", config)
	}
	
	scheduler := NewBaseScheduler(baseConfig)
	f.logger.Info("BaseScheduler created successfully", map[string]interface{}{
		"max_workers": baseConfig.ThreadPoolMaxWorkers,
		"parallel_dispatch": baseConfig.EnableParallelDispatch})
	
	return scheduler, nil
}

// CreateDefaultScheduler creates a GeneralScheduler with default configuration
func (f *SchedulerFactory) CreateDefaultScheduler() (Scheduler, error) {
	return f.CreateScheduler(GeneralSchedulerType, nil)
}

// CreateSchedulerFromType creates a scheduler from a string type identifier
func (f *SchedulerFactory) CreateSchedulerFromType(typeStr string, config interface{}) (Scheduler, error) {
	schedulerType := SchedulerType(strings.ToLower(typeStr))
	return f.CreateScheduler(schedulerType, config)
}

// ValidateSchedulerType checks if the provided scheduler type is supported
func (f *SchedulerFactory) ValidateSchedulerType(schedulerType SchedulerType) error {
	switch schedulerType {
	case GeneralSchedulerType, BaseSchedulerType:
		return nil
	default:
		return fmt.Errorf("unsupported scheduler type: %s", schedulerType)
	}
}

// GetSupportedTypes returns a list of supported scheduler types
func (f *SchedulerFactory) GetSupportedTypes() []SchedulerType {
	return []SchedulerType{
		GeneralSchedulerType,
		BaseSchedulerType,
	}
}

// mapToGeneralSchedulerConfig converts a map to GeneralSchedulerConfig
func (f *SchedulerFactory) mapToGeneralSchedulerConfig(configMap map[string]interface{}) (*GeneralSchedulerConfig, error) {
	config := DefaultGeneralSchedulerConfig()
	
	// Base configuration
	if baseConfig, err := f.extractBaseConfigFromMap(configMap); err == nil {
		config.BaseSchedulerConfig = baseConfig
	}
	
	// General scheduler specific configuration
	if topK, ok := configMap["top_k"]; ok {
		if k, ok := topK.(int); ok {
			config.TopK = k
		} else if k, ok := topK.(float64); ok {
			config.TopK = int(k)
		}
	}
	
	if topN, ok := configMap["top_n"]; ok {
		if n, ok := topN.(int); ok {
			config.TopN = n
		} else if n, ok := topN.(float64); ok {
			config.TopN = int(n)
		}
	}
	
	if activationMemSize, ok := configMap["activation_mem_size"]; ok {
		if size, ok := activationMemSize.(int); ok {
			config.ActivationMemSize = size
		} else if size, ok := activationMemSize.(float64); ok {
			config.ActivationMemSize = int(size)
		}
	}
	
	if searchMethod, ok := configMap["search_method"]; ok {
		if method, ok := searchMethod.(string); ok {
			config.SearchMethod = method
		}
	}
	
	if actMemDumpPath, ok := configMap["act_mem_dump_path"]; ok {
		if path, ok := actMemDumpPath.(string); ok {
			config.ActMemDumpPath = path
		}
	}
	
	return config, nil
}

// mapToBaseSchedulerConfig converts a map to BaseSchedulerConfig
func (f *SchedulerFactory) mapToBaseSchedulerConfig(configMap map[string]interface{}) (*BaseSchedulerConfig, error) {
	return f.extractBaseConfigFromMap(configMap)
}

// extractBaseConfigFromMap extracts base scheduler configuration from a map
func (f *SchedulerFactory) extractBaseConfigFromMap(configMap map[string]interface{}) (*BaseSchedulerConfig, error) {
	config := DefaultBaseSchedulerConfig()
	
	if maxWorkers, ok := configMap["thread_pool_max_workers"]; ok {
		if workers, ok := maxWorkers.(int); ok {
			config.ThreadPoolMaxWorkers = workers
		} else if workers, ok := maxWorkers.(float64); ok {
			config.ThreadPoolMaxWorkers = int(workers)
		}
	}
	
	if enableParallel, ok := configMap["enable_parallel_dispatch"]; ok {
		if parallel, ok := enableParallel.(bool); ok {
			config.EnableParallelDispatch = parallel
		}
	}
	
	if natsConfig, ok := configMap["nats_config"]; ok {
		if natsMap, ok := natsConfig.(map[string]interface{}); ok {
			if nats, err := f.extractNATSConfigFromMap(natsMap); err == nil {
				config.NATSConfig = nats
			}
		}
	}
	
	// Extract NATS KV configuration
	if natsKVConfig, ok := configMap["nats_kv_config"]; ok {
		if natsKVMap, ok := natsKVConfig.(map[string]interface{}); ok {
			if natsKV, err := f.extractNATSKVConfigFromMap(natsKVMap); err == nil {
				config.NATSKVConfig = natsKV
			}
		}
	}
	
	// Extract UseNATSKV flag
	if useNATSKV, ok := configMap["use_nats_kv"]; ok {
		if use, ok := useNATSKV.(bool); ok {
			config.UseNATSKV = use
		}
	}
	
	return config, nil
}

// extractNATSConfigFromMap extracts NATS configuration from a map
func (f *SchedulerFactory) extractNATSConfigFromMap(configMap map[string]interface{}) (*NATSConfig, error) {
	config := DefaultNATSConfig()
	
	if urls, ok := configMap["urls"]; ok {
		if urlSlice, ok := urls.([]interface{}); ok {
			urlStrings := make([]string, 0, len(urlSlice))
			for _, url := range urlSlice {
				if urlStr, ok := url.(string); ok {
					urlStrings = append(urlStrings, urlStr)
				}
			}
			if len(urlStrings) > 0 {
				config.URLs = urlStrings
			}
		}
	}
	
	if username, ok := configMap["username"]; ok {
		if u, ok := username.(string); ok {
			config.Username = u
		}
	}
	
	if password, ok := configMap["password"]; ok {
		if p, ok := password.(string); ok {
			config.Password = p
		}
	}
	
	if token, ok := configMap["token"]; ok {
		if t, ok := token.(string); ok {
			config.Token = t
		}
	}
	
	if maxReconnect, ok := configMap["max_reconnect"]; ok {
		if mr, ok := maxReconnect.(int); ok {
			config.MaxReconnect = mr
		} else if mr, ok := maxReconnect.(float64); ok {
			config.MaxReconnect = int(mr)
		}
	}
	
	// Extract JetStream configuration if present
	if jsConfig, ok := configMap["jetstream_config"]; ok {
		if jsMap, ok := jsConfig.(map[string]interface{}); ok {
			if js, err := f.extractJetStreamConfigFromMap(jsMap); err == nil {
				config.JetStreamConfig = js
			}
		}
	}
	
	return config, nil
}

// extractJetStreamConfigFromMap extracts JetStream configuration from a map
func (f *SchedulerFactory) extractJetStreamConfigFromMap(configMap map[string]interface{}) (*JetStreamConfig, error) {
	config := DefaultNATSConfig().JetStreamConfig
	
	if streamName, ok := configMap["stream_name"]; ok {
		if sn, ok := streamName.(string); ok {
			config.StreamName = sn
		}
	}
	
	if subjects, ok := configMap["stream_subjects"]; ok {
		if subjectSlice, ok := subjects.([]interface{}); ok {
			subjectStrings := make([]string, 0, len(subjectSlice))
			for _, subject := range subjectSlice {
				if subjectStr, ok := subject.(string); ok {
					subjectStrings = append(subjectStrings, subjectStr)
				}
			}
			if len(subjectStrings) > 0 {
				config.StreamSubjects = subjectStrings
			}
		}
	}
	
	if consumerName, ok := configMap["consumer_name"]; ok {
		if cn, ok := consumerName.(string); ok {
			config.ConsumerName = cn
		}
	}
	
	if durable, ok := configMap["consumer_durable"]; ok {
		if d, ok := durable.(bool); ok {
			config.ConsumerDurable = d
		}
	}
	
	if maxDeliver, ok := configMap["max_deliver"]; ok {
		if md, ok := maxDeliver.(int); ok {
			config.MaxDeliver = md
		} else if md, ok := maxDeliver.(float64); ok {
			config.MaxDeliver = int(md)
		}
	}
	
	if replicas, ok := configMap["replicas"]; ok {
		if r, ok := replicas.(int); ok {
			config.Replicas = r
		} else if r, ok := replicas.(float64); ok {
			config.Replicas = int(r)
		}
	}
	
	return config, nil
}

// extractNATSKVConfigFromMap extracts NATS KV configuration from a map
func (f *SchedulerFactory) extractNATSKVConfigFromMap(configMap map[string]interface{}) (*modules.NATSKVConfig, error) {
	config := modules.DefaultNATSKVConfig()
	
	if urls, ok := configMap["urls"]; ok {
		if urlSlice, ok := urls.([]interface{}); ok {
			urlStrings := make([]string, 0, len(urlSlice))
			for _, url := range urlSlice {
				if urlStr, ok := url.(string); ok {
					urlStrings = append(urlStrings, urlStr)
				}
			}
			if len(urlStrings) > 0 {
				config.URLs = urlStrings
			}
		}
	}
	
	if bucketName, ok := configMap["bucket_name"]; ok {
		if bn, ok := bucketName.(string); ok {
			config.BucketName = bn
		}
	}
	
	if description, ok := configMap["description"]; ok {
		if desc, ok := description.(string); ok {
			config.Description = desc
		}
	}
	
	if maxValueSize, ok := configMap["max_value_size"]; ok {
		if mvs, ok := maxValueSize.(int); ok {
			config.MaxValueSize = int32(mvs)
		} else if mvs, ok := maxValueSize.(float64); ok {
			config.MaxValueSize = int32(mvs)
		}
	}
	
	if history, ok := configMap["history"]; ok {
		if h, ok := history.(int); ok {
			config.History = uint8(h)
		} else if h, ok := history.(float64); ok {
			config.History = uint8(h)
		}
	}
	
	if maxBytes, ok := configMap["max_bytes"]; ok {
		if mb, ok := maxBytes.(int); ok {
			config.MaxBytes = int64(mb)
		} else if mb, ok := maxBytes.(float64); ok {
			config.MaxBytes = int64(mb)
		}
	}
	
	if storage, ok := configMap["storage"]; ok {
		if s, ok := storage.(string); ok {
			config.Storage = s
		}
	}
	
	if replicas, ok := configMap["replicas"]; ok {
		if r, ok := replicas.(int); ok {
			config.Replicas = r
		} else if r, ok := replicas.(float64); ok {
			config.Replicas = int(r)
		}
	}
	
	if streamName, ok := configMap["stream_name"]; ok {
		if sn, ok := streamName.(string); ok {
			config.StreamName = sn
		}
	}
	
	if streamSubjects, ok := configMap["stream_subjects"]; ok {
		if subjectSlice, ok := streamSubjects.([]interface{}); ok {
			subjectStrings := make([]string, 0, len(subjectSlice))
			for _, subject := range subjectSlice {
				if subjectStr, ok := subject.(string); ok {
					subjectStrings = append(subjectStrings, subjectStr)
				}
			}
			if len(subjectStrings) > 0 {
				config.StreamSubjects = subjectStrings
			}
		}
	}
	
	if consumerName, ok := configMap["consumer_name"]; ok {
		if cn, ok := consumerName.(string); ok {
			config.ConsumerName = cn
		}
	}
	
	if consumerDurable, ok := configMap["consumer_durable"]; ok {
		if cd, ok := consumerDurable.(bool); ok {
			config.ConsumerDurable = cd
		}
	}
	
	if maxDeliver, ok := configMap["max_deliver"]; ok {
		if md, ok := maxDeliver.(int); ok {
			config.MaxDeliver = md
		} else if md, ok := maxDeliver.(float64); ok {
			config.MaxDeliver = int(md)
		}
	}
	
	if maxAckPending, ok := configMap["max_ack_pending"]; ok {
		if map_, ok := maxAckPending.(int); ok {
			config.MaxAckPending = map_
		} else if map_, ok := maxAckPending.(float64); ok {
			config.MaxAckPending = int(map_)
		}
	}
	
	if ackWait, ok := configMap["ack_wait"]; ok {
		if aw, ok := ackWait.(string); ok {
			// Parse duration string (e.g., "10s", "1m", "30s")
			if duration, err := time.ParseDuration(aw); err == nil {
				config.AckWait = duration
			}
		} else if aw, ok := ackWait.(int); ok {
			// Assume seconds if integer
			config.AckWait = time.Duration(aw) * time.Second
		} else if aw, ok := ackWait.(float64); ok {
			// Assume seconds if float
			config.AckWait = time.Duration(aw) * time.Second
		}
	}
	
	return config, nil
}

// CreateSchedulerWithLLM creates and initializes a scheduler with the provided LLM
func (f *SchedulerFactory) CreateSchedulerWithLLM(schedulerType SchedulerType, config interface{}, chatLLM interfaces.LLM) (Scheduler, error) {
	scheduler, err := f.CreateScheduler(schedulerType, config)
	if err != nil {
		return nil, err
	}
	
	if err := scheduler.InitializeModules(chatLLM); err != nil {
		return nil, fmt.Errorf("failed to initialize scheduler modules: %w", err)
	}
	
	f.logger.Info("Scheduler created and initialized with LLM", map[string]interface{}{"type": schedulerType})
	return scheduler, nil
}

// CreateAutoStartScheduler creates, initializes, and starts a scheduler
func (f *SchedulerFactory) CreateAutoStartScheduler(schedulerType SchedulerType, config interface{}, chatLLM interfaces.LLM) (Scheduler, error) {
	scheduler, err := f.CreateSchedulerWithLLM(schedulerType, config, chatLLM)
	if err != nil {
		return nil, err
	}
	
	if err := scheduler.Start(); err != nil {
		return nil, fmt.Errorf("failed to start scheduler: %w", err)
	}
	
	f.logger.Info("Scheduler created, initialized, and started", map[string]interface{}{"type": schedulerType})
	return scheduler, nil
}

// GetFactoryStats returns factory statistics
func (f *SchedulerFactory) GetFactoryStats() map[string]interface{} {
	return map[string]interface{}{
		"supported_types": f.GetSupportedTypes(),
		"factory_version": "1.0.0",
	}
}

// Global factory instance
var defaultFactory = NewSchedulerFactory()

// Global convenience functions

// CreateScheduler creates a scheduler using the default factory
func CreateScheduler(schedulerType SchedulerType, config interface{}) (Scheduler, error) {
	return defaultFactory.CreateScheduler(schedulerType, config)
}

// CreateDefaultScheduler creates a default GeneralScheduler
func CreateDefaultScheduler() (Scheduler, error) {
	return defaultFactory.CreateDefaultScheduler()
}

// CreateSchedulerFromType creates a scheduler from a string type
func CreateSchedulerFromType(typeStr string, config interface{}) (Scheduler, error) {
	return defaultFactory.CreateSchedulerFromType(typeStr, config)
}

// CreateSchedulerWithLLM creates and initializes a scheduler with LLM
func CreateSchedulerWithLLM(schedulerType SchedulerType, config interface{}, chatLLM interfaces.LLM) (Scheduler, error) {
	return defaultFactory.CreateSchedulerWithLLM(schedulerType, config, chatLLM)
}

// CreateAutoStartScheduler creates, initializes, and starts a scheduler
func CreateAutoStartScheduler(schedulerType SchedulerType, config interface{}, chatLLM interfaces.LLM) (Scheduler, error) {
	return defaultFactory.CreateAutoStartScheduler(schedulerType, config, chatLLM)
}
