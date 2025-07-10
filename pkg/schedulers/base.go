package schedulers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/logger"
	"github.com/memtensor/memgos/pkg/schedulers/modules"
)

// BaseSchedulerConfig holds configuration for the base scheduler
type BaseSchedulerConfig struct {
	ThreadPoolMaxWorkers     int           `json:"thread_pool_max_workers" yaml:"thread_pool_max_workers"`
	EnableParallelDispatch   bool          `json:"enable_parallel_dispatch" yaml:"enable_parallel_dispatch"`
	ConsumeIntervalSeconds   time.Duration `json:"consume_interval_seconds" yaml:"consume_interval_seconds"`
	NATSConfig               *NATSConfig   `json:"nats_config" yaml:"nats_config"`
}

// DefaultBaseSchedulerConfig returns default base scheduler configuration
func DefaultBaseSchedulerConfig() *BaseSchedulerConfig {
	return &BaseSchedulerConfig{
		ThreadPoolMaxWorkers:   modules.DefaultThreadPoolMaxWorkers,
		EnableParallelDispatch: false,
		ConsumeIntervalSeconds: modules.DefaultConsumeIntervalSeconds,
		NATSConfig:             DefaultNATSConfig(),
	}
}

// BaseScheduler provides the foundation for all memory schedulers
type BaseScheduler struct {
	*NATSSchedulerModule
	mu                    sync.RWMutex
	config                *BaseSchedulerConfig
	maxWorkers            int
	retriever             *modules.SchedulerRetriever
	monitor               *modules.SchedulerMonitor
	enableParallelDispatch bool
	dispatcher            *modules.SchedulerDispatcher
	
	// Message queues
	memosMessageQueue    chan *modules.ScheduleMessageItem
	webLogMessageQueue   chan *modules.ScheduleLogForWebItem
	
	// Consumer management
	consumerCtx          context.Context
	consumerCancel       context.CancelFunc
	consumerRunning      bool
	consumeInterval      time.Duration
	
	// State management
	currentUserID        string
	currentMemCubeID     string
	currentMemCube       interface{} // Will be GeneralMemCube interface
	
	logger               *logger.Logger
	startTime            time.Time
}

// NewBaseScheduler creates a new base scheduler
func NewBaseScheduler(config *BaseSchedulerConfig) *BaseScheduler {
	if config == nil {
		config = DefaultBaseSchedulerConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	scheduler := &BaseScheduler{
		NATSSchedulerModule:    NewNATSSchedulerModule(config.NATSConfig),
		config:                 config,
		maxWorkers:             config.ThreadPoolMaxWorkers,
		enableParallelDispatch: config.EnableParallelDispatch,
		memosMessageQueue:      make(chan *modules.ScheduleMessageItem, 1000),
		webLogMessageQueue:     make(chan *modules.ScheduleLogForWebItem, 1000),
		consumerCtx:            ctx,
		consumerCancel:         cancel,
		consumerRunning:        false,
		consumeInterval:        config.ConsumeIntervalSeconds,
		logger:                 logger.GetLogger("base-scheduler"),
		startTime:              time.Now(),
	}
	
	// Initialize dispatcher
	scheduler.dispatcher = modules.NewSchedulerDispatcher(
		scheduler.maxWorkers,
		scheduler.enableParallelDispatch,
	)
	
	return scheduler
}

// InitializeModules initializes all necessary modules for the scheduler
func (b *BaseScheduler) InitializeModules(chatLLM interfaces.LLM) error {
	if chatLLM == nil {
		return fmt.Errorf("chat LLM cannot be nil")
	}
	
	b.mu.Lock()
	defer b.mu.Unlock()
	
	// Initialize monitor
	b.monitor = modules.NewSchedulerMonitor(chatLLM, modules.DefaultActivationMemSize)
	
	// Initialize retriever
	b.retriever = modules.NewSchedulerRetriever(chatLLM)
	
	// Initialize NATS connection
	if err := b.InitializeNATS(); err != nil {
		b.logger.Warn("Failed to initialize NATS", "error", err)
		// Continue without NATS - scheduler can work in local mode
	}
	
	b.logger.Info("Base scheduler modules initialized",
		"max_workers", b.maxWorkers,
		"parallel_dispatch", b.enableParallelDispatch,
		"consume_interval", b.consumeInterval)
	
	return nil
}

// SubmitMessages submits one or more messages to the message queue
func (b *BaseScheduler) SubmitMessages(messages ...*modules.ScheduleMessageItem) error {
	if len(messages) == 0 {
		return nil
	}
	
	for _, message := range messages {
		if message == nil {
			continue
		}
		
		select {
		case b.memosMessageQueue <- message:
			b.logger.Debug("Message submitted", 
				"label", message.Label, 
				"content_length", len(message.Content))
		case <-time.After(5 * time.Second):
			return fmt.Errorf("timeout submitting message with label: %s", message.Label)
		}
	}
	
	return nil
}

// SubmitWebLogs submits web log messages to the web log queue
func (b *BaseScheduler) SubmitWebLogs(messages ...*modules.ScheduleLogForWebItem) error {
	if len(messages) == 0 {
		return nil
	}
	
	for _, message := range messages {
		if message == nil {
			continue
		}
		
		select {
		case b.webLogMessageQueue <- message:
			b.logger.Debug("Web log submitted", 
				"title", message.LogTitle, 
				"content_length", len(message.LogContent))
		case <-time.After(5 * time.Second):
			return fmt.Errorf("timeout submitting web log: %s", message.LogTitle)
		}
	}
	
	return nil
}

// GetWebLogMessages retrieves all web log messages from the queue
func (b *BaseScheduler) GetWebLogMessages() []*modules.ScheduleLogForWebItem {
	messages := make([]*modules.ScheduleLogForWebItem, 0)
	
	// Drain the queue
	for {
		select {
		case msg := <-b.webLogMessageQueue:
			messages = append(messages, msg)
		default:
			return messages
		}
	}
}

// messageConsumer continuously processes messages from the queue
func (b *BaseScheduler) messageConsumer() {
	b.logger.Info("Message consumer started")
	defer b.logger.Info("Message consumer stopped")
	
	for {
		select {
		case <-b.consumerCtx.Done():
			return
		case <-time.After(b.consumeInterval):
			// Collect available messages
			messages := b.collectMessages()
			if len(messages) > 0 {
				if err := b.dispatchMessages(messages); err != nil {
					b.logger.Error("Failed to dispatch messages", "error", err)
					if b.monitor != nil {
						b.monitor.IncrementErrorCount()
					}
				} else if b.monitor != nil {
					b.monitor.IncrementProcessedTasks()
				}
			}
		}
	}
}

// collectMessages collects available messages from the queue
func (b *BaseScheduler) collectMessages() []*modules.ScheduleMessageItem {
	messages := make([]*modules.ScheduleMessageItem, 0)
	
	// Collect all immediately available messages
	for {
		select {
		case msg := <-b.memosMessageQueue:
			messages = append(messages, msg)
		default:
			return messages
		}
	}
}

// dispatchMessages dispatches messages to the dispatcher
func (b *BaseScheduler) dispatchMessages(messages []*modules.ScheduleMessageItem) error {
	if b.dispatcher == nil {
		return fmt.Errorf("dispatcher not initialized")
	}
	
	return b.dispatcher.Dispatch(messages)
}

// Start begins the scheduler operation
func (b *BaseScheduler) Start() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if b.consumerRunning {
		return fmt.Errorf("scheduler is already running")
	}
	
	// Start dispatcher
	if err := b.dispatcher.Start(); err != nil {
		return fmt.Errorf("failed to start dispatcher: %w", err)
	}
	
	// Start message consumer
	b.consumerRunning = true
	go b.messageConsumer()
	
	b.logger.Info("Base scheduler started")
	return nil
}

// Stop gracefully stops the scheduler
func (b *BaseScheduler) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if !b.consumerRunning {
		return fmt.Errorf("scheduler is not running")
	}
	
	// Stop consumer
	b.consumerCancel()
	b.consumerRunning = false
	
	// Stop dispatcher
	if err := b.dispatcher.Stop(); err != nil {
		b.logger.Error("Failed to stop dispatcher", "error", err)
	}
	
	// Stop NATS listener if running
	if err := b.StopListening(); err != nil {
		b.logger.Error("Failed to stop NATS listener", "error", err)
	}
	
	// Close NATS connection
	if err := b.Close(); err != nil {
		b.logger.Error("Failed to close NATS connection", "error", err)
	}
	
	b.logger.Info("Base scheduler stopped")
	return nil
}

// IsRunning returns whether the scheduler is currently running
func (b *BaseScheduler) IsRunning() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.consumerRunning
}

// GetStats returns scheduler statistics
func (b *BaseScheduler) GetStats() map[string]interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	stats := map[string]interface{}{
		"running":                b.consumerRunning,
		"uptime":                 time.Since(b.startTime),
		"max_workers":            b.maxWorkers,
		"parallel_dispatch":      b.enableParallelDispatch,
		"consume_interval":       b.consumeInterval,
		"message_queue_size":     len(b.memosMessageQueue),
		"web_log_queue_size":     len(b.webLogMessageQueue),
		"nats_connected":         b.IsConnected(),
	}
	
	// Add dispatcher stats
	if b.dispatcher != nil {
		stats["dispatcher"] = b.dispatcher.GetStats()
	}
	
	// Add monitor stats
	if b.monitor != nil {
		stats["monitor"] = b.monitor.GetStatistics()
	}
	
	// Add retriever cache stats
	if b.retriever != nil {
		stats["retriever_cache"] = b.retriever.GetCacheStats()
	}
	
	return stats
}

// SetCurrentSession sets the current user and memory cube session
func (b *BaseScheduler) SetCurrentSession(userID, memCubeID string, memCube interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.currentUserID = userID
	b.currentMemCubeID = memCubeID
	b.currentMemCube = memCube
	
	// Update retriever with new memory cube
	if b.retriever != nil {
		b.retriever.SetMemCube(memCube)
	}
	
	b.logger.Debug("Session updated", 
		"user_id", userID, 
		"mem_cube_id", memCubeID)
}

// GetCurrentSession returns the current session information
func (b *BaseScheduler) GetCurrentSession() (string, string, interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.currentUserID, b.currentMemCubeID, b.currentMemCube
}

// GetRetriever returns the scheduler retriever
func (b *BaseScheduler) GetRetriever() *modules.SchedulerRetriever {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.retriever
}

// GetMonitor returns the scheduler monitor
func (b *BaseScheduler) GetMonitor() *modules.SchedulerMonitor {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.monitor
}

// GetDispatcher returns the scheduler dispatcher
func (b *BaseScheduler) GetDispatcher() *modules.SchedulerDispatcher {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.dispatcher
}

// CreateAutofilledLogItem creates a log item with current session context
func (b *BaseScheduler) CreateAutofilledLogItem(logTitle, logContent, label string) *modules.ScheduleLogForWebItem {
	userID, memCubeID, _ := b.GetCurrentSession()
	
	logItem := modules.NewScheduleLogForWebItem(userID, memCubeID, label, logTitle, logContent)
	
	// Add current memory sizes and capacities if monitor is available
	if b.monitor != nil {
		stats := b.monitor.GetStatistics()
		// Update log item with actual stats
		if memCubeStats, ok := stats["mem_cube"]; ok {
			logItem.LogContent += fmt.Sprintf("\n\nMemory Stats: %+v", memCubeStats)
		}
	}
	
	return logItem
}
