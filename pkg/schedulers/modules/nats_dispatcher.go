package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/logger"
)

// NATSDispatcherConfig holds configuration for NATS-based dispatcher
type NATSDispatcherConfig struct {
	MaxWorkers              int           `json:"max_workers" yaml:"max_workers"`
	EnableParallelDispatch  bool          `json:"enable_parallel_dispatch" yaml:"enable_parallel_dispatch"`
	BatchSize               int           `json:"batch_size" yaml:"batch_size"`
	ProcessingTimeout       time.Duration `json:"processing_timeout" yaml:"processing_timeout"`
	WorkQueueSubject        string        `json:"work_queue_subject" yaml:"work_queue_subject"`
	ResultSubject          string        `json:"result_subject" yaml:"result_subject"`
	ErrorSubject           string        `json:"error_subject" yaml:"error_subject"`
	AckWait                time.Duration `json:"ack_wait" yaml:"ack_wait"`
	MaxDeliver             int           `json:"max_deliver" yaml:"max_deliver"`
}

// DefaultNATSDispatcherConfig returns default NATS dispatcher configuration
func DefaultNATSDispatcherConfig() *NATSDispatcherConfig {
	return &NATSDispatcherConfig{
		MaxWorkers:             DefaultThreadPoolMaxWorkers,
		EnableParallelDispatch: false,
		BatchSize:              10,
		ProcessingTimeout:      30 * time.Second,
		WorkQueueSubject:       "scheduler.work",
		ResultSubject:         "scheduler.result",
		ErrorSubject:          "scheduler.error",
		AckWait:               30 * time.Second,
		MaxDeliver:            3,
	}
}

// NATSSchedulerDispatcher handles NATS-based message dispatch with JetStream work queues
type NATSSchedulerDispatcher struct {
	*BaseSchedulerModule
	mu                      sync.RWMutex
	config                  *NATSDispatcherConfig
	conn                    *nats.Conn
	js                      jetstream.JetStream
	consumer                jetstream.Consumer
	stream                  jetstream.Stream
	
	// Worker pool management
	workerPool              chan struct{}   // Semaphore for worker limit
	handlers                map[string]HandlerFunc
	running                 bool
	ctx                     context.Context
	cancel                  context.CancelFunc
	
	// Statistics
	processedMessages       int64
	failedMessages          int64
	activeWorkers           int64
	
	logger                  *logger.Logger
	startTime               time.Time
}

// NewNATSSchedulerDispatcher creates a new NATS-based scheduler dispatcher
func NewNATSSchedulerDispatcher(conn *nats.Conn, js jetstream.JetStream, config *NATSDispatcherConfig) *NATSSchedulerDispatcher {
	if config == nil {
		config = DefaultNATSDispatcherConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	dispatcher := &NATSSchedulerDispatcher{
		BaseSchedulerModule:    NewBaseSchedulerModule(),
		config:                 config,
		conn:                   conn,
		js:                     js,
		workerPool:             make(chan struct{}, config.MaxWorkers),
		handlers:               make(map[string]HandlerFunc),
		running:                false,
		ctx:                    ctx,
		cancel:                 cancel,
		processedMessages:      0,
		failedMessages:         0,
		activeWorkers:          0,
		logger:                 logger.GetLogger("nats-dispatcher"),
		startTime:              time.Now(),
	}

	// Initialize worker pool semaphore
	for i := 0; i < config.MaxWorkers; i++ {
		dispatcher.workerPool <- struct{}{}
	}

	return dispatcher
}

// InitializeWorkQueue sets up the NATS JetStream work queue
func (d *NATSSchedulerDispatcher) InitializeWorkQueue() error {
	if d.js == nil {
		return fmt.Errorf("JetStream not initialized")
	}
	
	// Create or get work queue stream
	streamConfig := jetstream.StreamConfig{
		Name:        "SCHEDULER_WORK_QUEUE",
		Subjects:    []string{d.config.WorkQueueSubject, d.config.ResultSubject, d.config.ErrorSubject},
		Retention:   jetstream.WorkQueuePolicy,
		Discard:     jetstream.DiscardOld,
		MaxAge:      24 * time.Hour,
		MaxBytes:    512 * 1024 * 1024, // 512MB
		MaxMsgs:     100000,
		Replicas:    1,
	}
	
	var err error
	d.stream, err = d.js.CreateOrUpdateStream(d.ctx, streamConfig)
	if err != nil {
		return fmt.Errorf("failed to create work queue stream: %w", err)
	}
	
	// Create consumer for work queue
	consumerConfig := jetstream.ConsumerConfig{
		Name:          "scheduler_work_consumer",
		Durable:       "scheduler_work_consumer",
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    d.config.MaxDeliver,
		AckWait:       d.config.AckWait,
		MaxAckPending: d.config.MaxWorkers * 2, // Allow some buffering
		ReplayPolicy:  jetstream.ReplayInstantPolicy,
		FilterSubject: d.config.WorkQueueSubject,
	}
	
	d.consumer, err = d.stream.CreateOrUpdateConsumer(d.ctx, consumerConfig)
	if err != nil {
		return fmt.Errorf("failed to create work queue consumer: %w", err)
	}
	
	d.logger.Info("NATS work queue initialized",
		"stream", streamConfig.Name,
		"subject", d.config.WorkQueueSubject,
		"max_workers", d.config.MaxWorkers)
	
	return nil
}

// RegisterHandler registers a handler function for a specific message label
func (d *NATSSchedulerDispatcher) RegisterHandler(label string, handler HandlerFunc) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	d.handlers[label] = handler
	d.logger.Debug("Handler registered", "label", label)
}

// RegisterHandlers bulk registers multiple handlers from a map
func (d *NATSSchedulerDispatcher) RegisterHandlers(handlers map[string]HandlerFunc) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	registeredCount := 0
	for label, handler := range handlers {
		if handler == nil {
			d.logger.Error("Handler is nil", "label", label)
			continue
		}
		
		d.handlers[label] = handler
		registeredCount++
	}
	
	d.logger.Info("Handlers registered in bulk", "count", registeredCount)
	return nil
}

// DispatchToNATS publishes messages to the NATS work queue for distributed processing
func (d *NATSSchedulerDispatcher) DispatchToNATS(msgList []*ScheduleMessageItem) error {
	if len(msgList) == 0 {
		return nil
	}
	
	if d.js == nil {
		return fmt.Errorf("JetStream not initialized")
	}
	
	// Group messages by label
	labelGroups := make(map[string][]*ScheduleMessageItem)
	for _, message := range msgList {
		labelGroups[message.Label] = append(labelGroups[message.Label], message)
	}
	
	// Publish each group as a work item
	errors := make([]error, 0)
	for label, msgs := range labelGroups {
		workItem := &WorkItem{
			ID:       fmt.Sprintf("work_%d_%s", time.Now().UnixNano(), label),
			Label:    label,
			Messages: msgs,
			Created:  time.Now(),
			Retries:  0,
		}
		
		if err := d.publishWorkItem(workItem); err != nil {
			errors = append(errors, fmt.Errorf("failed to publish work item for label '%s': %w", label, err))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("dispatch to NATS failed with %d errors: %v", len(errors), errors)
	}
	
	return nil
}

// publishWorkItem publishes a work item to the NATS work queue
func (d *NATSSchedulerDispatcher) publishWorkItem(workItem *WorkItem) error {
	data, err := json.Marshal(workItem)
	if err != nil {
		return fmt.Errorf("failed to marshal work item: %w", err)
	}
	
	// Publish to work queue
	_, err = d.js.Publish(d.ctx, d.config.WorkQueueSubject, data)
	if err != nil {
		return fmt.Errorf("failed to publish work item: %w", err)
	}
	
	d.logger.Debug("Work item published",
		"id", workItem.ID,
		"label", workItem.Label,
		"message_count", len(workItem.Messages))
	
	return nil
}

// Dispatch processes messages locally (legacy mode)
func (d *NATSSchedulerDispatcher) Dispatch(msgList []*ScheduleMessageItem) error {
	if len(msgList) == 0 {
		return nil
	}

	// Group messages by label
	labelGroups := make(map[string][]*ScheduleMessageItem)
	for _, message := range msgList {
		labelGroups[message.Label] = append(labelGroups[message.Label], message)
	}

	// Process each label group
	errors := make([]error, 0)
	for label, msgs := range labelGroups {
		if err := d.dispatchGroup(label, msgs); err != nil {
			errors = append(errors, fmt.Errorf("failed to dispatch group '%s': %w", label, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("dispatch failed with %d errors: %v", len(errors), errors)
	}

	return nil
}

// dispatchGroup handles a group of messages with the same label
func (d *NATSSchedulerDispatcher) dispatchGroup(label string, msgs []*ScheduleMessageItem) error {
	d.mu.RLock()
	handler, exists := d.handlers[label]
	d.mu.RUnlock()

	if !exists {
		d.logger.Warn("No handler registered for label, using default", "label", label)
		handler = d.defaultMessageHandler
	}

	d.logger.Debug("Dispatching message group", 
		"label", label, 
		"message_count", len(msgs))

	if d.config.EnableParallelDispatch {
		return d.dispatchParallel(handler, msgs)
	} else {
		return d.dispatchSerial(handler, msgs)
	}
}

// dispatchParallel handles messages in parallel using worker pool
func (d *NATSSchedulerDispatcher) dispatchParallel(handler HandlerFunc, msgs []*ScheduleMessageItem) error {
	select {
	case <-d.workerPool: // Acquire worker
		d.mu.Lock()
		d.activeWorkers++
		d.mu.Unlock()
		
		go func() {
			defer func() {
				d.workerPool <- struct{}{} // Release worker
				d.mu.Lock()
				d.activeWorkers--
				d.mu.Unlock()
			}()
			
			if err := handler(msgs); err != nil {
				d.logger.Error("Parallel handler execution failed", "error", err)
				d.mu.Lock()
				d.failedMessages += int64(len(msgs))
				d.mu.Unlock()
			} else {
				d.mu.Lock()
				d.processedMessages += int64(len(msgs))
				d.mu.Unlock()
			}
		}()
		return nil
	case <-d.ctx.Done():
		return fmt.Errorf("dispatcher context cancelled")
	case <-time.After(d.config.ProcessingTimeout):
		return fmt.Errorf("timeout waiting for available worker")
	}
}

// dispatchSerial handles messages serially
func (d *NATSSchedulerDispatcher) dispatchSerial(handler HandlerFunc, msgs []*ScheduleMessageItem) error {
	err := handler(msgs)
	
	d.mu.Lock()
	if err != nil {
		d.failedMessages += int64(len(msgs))
	} else {
		d.processedMessages += int64(len(msgs))
	}
	d.mu.Unlock()
	
	return err
}

// defaultMessageHandler provides a default handler for unregistered message types
func (d *NATSSchedulerDispatcher) defaultMessageHandler(messages []*ScheduleMessageItem) error {
	d.logger.Debug("Using default message handler", "message_count", len(messages))
	for _, msg := range messages {
		d.logger.Debug("Processing with default handler", 
			"label", msg.Label, 
			"content_length", len(msg.Content))
	}
	return nil
}

// StartWorkers starts the NATS work queue consumer workers
func (d *NATSSchedulerDispatcher) StartWorkers() error {
	if d.consumer == nil {
		return fmt.Errorf("consumer not initialized")
	}
	
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return fmt.Errorf("workers are already running")
	}
	d.running = true
	d.mu.Unlock()
	
	// Start consuming work items
	go func() {
		defer func() {
			d.mu.Lock()
			d.running = false
			d.mu.Unlock()
		}()
		
		if err := d.consumeWorkItems(); err != nil {
			d.logger.Error("Work item consumer error", "error", err)
		}
	}()
	
	d.logger.Info("NATS dispatcher workers started", "max_workers", d.config.MaxWorkers)
	return nil
}

// consumeWorkItems continuously consumes work items from the NATS work queue
func (d *NATSSchedulerDispatcher) consumeWorkItems() error {
	consumeCtx, err := d.consumer.Consume(func(msg jetstream.Msg) {
		if err := d.processWorkItem(msg); err != nil {
			d.logger.Error("Failed to process work item", "error", err)
			// Negative acknowledgment - item will be redelivered
			msg.Nak()
		} else {
			// Positive acknowledgment
			msg.Ack()
		}
	})
	
	if err != nil {
		return fmt.Errorf("failed to start consuming work items: %w", err)
	}
	
	// Wait for context cancellation
	select {
	case <-d.ctx.Done():
		consumeCtx.Stop()
		return nil
	}
}

// processWorkItem processes a work item from the NATS work queue
func (d *NATSSchedulerDispatcher) processWorkItem(msg jetstream.Msg) error {
	// Deserialize work item
	var workItem WorkItem
	if err := json.Unmarshal(msg.Data(), &workItem); err != nil {
		return fmt.Errorf("failed to unmarshal work item: %w", err)
	}
	
	d.logger.Debug("Processing work item",
		"id", workItem.ID,
		"label", workItem.Label,
		"message_count", len(workItem.Messages),
		"retries", workItem.Retries)
	
	// Get handler for work item label
	d.mu.RLock()
	handler, exists := d.handlers[workItem.Label]
	d.mu.RUnlock()
	
	if !exists {
		d.logger.Warn("No handler registered for work item label", "label", workItem.Label)
		return nil // Skip work item
	}
	
	// Process work item with timeout
	ctx, cancel := context.WithTimeout(d.ctx, d.config.ProcessingTimeout)
	defer cancel()
	
	// Execute handler in a separate goroutine with timeout
	done := make(chan error, 1)
	go func() {
		done <- handler(workItem.Messages)
	}()
	
	select {
	case err := <-done:
		if err != nil {
			d.mu.Lock()
			d.failedMessages += int64(len(workItem.Messages))
			d.mu.Unlock()
			
			// Publish error result
			d.publishErrorResult(&workItem, err)
			return err
		} else {
			d.mu.Lock()
			d.processedMessages += int64(len(workItem.Messages))
			d.mu.Unlock()
			
			// Publish success result
			d.publishSuccessResult(&workItem)
			return nil
		}
	case <-ctx.Done():
		d.mu.Lock()
		d.failedMessages += int64(len(workItem.Messages))
		d.mu.Unlock()
		
		timeoutErr := fmt.Errorf("work item processing timeout")
		d.publishErrorResult(&workItem, timeoutErr)
		return timeoutErr
	}
}

// publishSuccessResult publishes a success result to NATS
func (d *NATSSchedulerDispatcher) publishSuccessResult(workItem *WorkItem) {
	result := &WorkResult{
		WorkItemID:  workItem.ID,
		Label:       workItem.Label,
		Success:     true,
		ProcessedAt: time.Now(),
		MessageCount: len(workItem.Messages),
	}
	
	data, err := json.Marshal(result)
	if err != nil {
		d.logger.Error("Failed to marshal success result", "error", err)
		return
	}
	
	if _, err := d.js.Publish(d.ctx, d.config.ResultSubject, data); err != nil {
		d.logger.Error("Failed to publish success result", "error", err)
	}
}

// publishErrorResult publishes an error result to NATS
func (d *NATSSchedulerDispatcher) publishErrorResult(workItem *WorkItem, processingError error) {
	result := &WorkResult{
		WorkItemID:   workItem.ID,
		Label:        workItem.Label,
		Success:      false,
		Error:        processingError.Error(),
		ProcessedAt:  time.Now(),
		MessageCount: len(workItem.Messages),
		Retries:      workItem.Retries,
	}
	
	data, err := json.Marshal(result)
	if err != nil {
		d.logger.Error("Failed to marshal error result", "error", err)
		return
	}
	
	if _, err := d.js.Publish(d.ctx, d.config.ErrorSubject, data); err != nil {
		d.logger.Error("Failed to publish error result", "error", err)
	}
}

// Start begins the dispatcher service (legacy mode)
func (d *NATSSchedulerDispatcher) Start() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	if d.running {
		return fmt.Errorf("dispatcher is already running")
	}
	
	d.running = true
	d.logger.Info("NATS scheduler dispatcher started")
	return nil
}

// Stop gracefully shuts down the dispatcher
func (d *NATSSchedulerDispatcher) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	if !d.running {
		return fmt.Errorf("dispatcher is not running")
	}
	
	d.cancel() // Cancel context to stop workers
	d.running = false
	
	// Wait for active workers to complete (with timeout)
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-timeout:
			d.logger.Warn("Timeout waiting for workers to complete")
			return fmt.Errorf("timeout waiting for workers to complete")
		case <-ticker.C:
			if d.activeWorkers == 0 {
				d.logger.Info("NATS scheduler dispatcher stopped")
				return nil
			}
		}
	}
}

// GetStats returns dispatcher statistics
func (d *NATSSchedulerDispatcher) GetStats() map[string]interface{} {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	availableWorkers := len(d.workerPool)
	
	stats := map[string]interface{}{
		"max_workers":        d.config.MaxWorkers,
		"active_workers":     d.activeWorkers,
		"available_workers":  availableWorkers,
		"parallel_enabled":   d.config.EnableParallelDispatch,
		"running":           d.running,
		"handlers_count":    len(d.handlers),
		"processed_messages": d.processedMessages,
		"failed_messages":    d.failedMessages,
		"uptime":            time.Since(d.startTime),
		"batch_size":        d.config.BatchSize,
		"processing_timeout": d.config.ProcessingTimeout,
		"work_queue_subject": d.config.WorkQueueSubject,
	}
	
	// Add consumer info if available
	if d.consumer != nil {
		if info, err := d.consumer.Info(d.ctx); err == nil {
			stats["consumer_info"] = map[string]interface{}{
				"delivered":        info.Delivered,
				"ack_pending":      info.AckFloor,
				"num_pending":      info.NumPending,
				"num_redelivered":  info.NumRedelivered,
				"num_waiting":      info.NumWaiting,
			}
		}
	}
	
	return stats
}

// IsRunning returns whether the dispatcher is currently running
func (d *NATSSchedulerDispatcher) IsRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}

// WorkItem represents a work item in the NATS work queue
type WorkItem struct {
	ID       string                 `json:"id"`
	Label    string                 `json:"label"`
	Messages []*ScheduleMessageItem `json:"messages"`
	Created  time.Time              `json:"created"`
	Retries  int                    `json:"retries"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// WorkResult represents the result of processing a work item
type WorkResult struct {
	WorkItemID   string    `json:"work_item_id"`
	Label        string    `json:"label"`
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
	ProcessedAt  time.Time `json:"processed_at"`
	MessageCount int       `json:"message_count"`
	Retries      int       `json:"retries"`
	ProcessingTime time.Duration `json:"processing_time,omitempty"`
}