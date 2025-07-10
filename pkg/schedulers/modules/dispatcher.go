package modules

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
)

// SchedulerDispatcher handles thread pool-based message dispatch
type SchedulerDispatcher struct {
	*BaseSchedulerModule
	mu                      sync.RWMutex
	maxWorkers              int
	enableParallelDispatch  bool
	workerPool              chan struct{}   // Semaphore for worker limit
	handlers                map[string]HandlerFunc
	running                 bool
	ctx                     context.Context
	cancel                  context.CancelFunc
	logger                  interfaces.Logger
}

// NewSchedulerDispatcher creates a new scheduler dispatcher
func NewSchedulerDispatcher(maxWorkers int, enableParallelDispatch bool) *SchedulerDispatcher {
	ctx, cancel := context.WithCancel(context.Background())
	
	dispatcher := &SchedulerDispatcher{
		BaseSchedulerModule:    NewBaseSchedulerModule(),
		maxWorkers:             maxWorkers,
		enableParallelDispatch: enableParallelDispatch,
		workerPool:             make(chan struct{}, maxWorkers),
		handlers:               make(map[string]HandlerFunc),
		running:                false,
		ctx:                    ctx,
		cancel:                 cancel,
		logger:                 nil, // Will be set by parent
	}

	// Initialize worker pool semaphore
	for i := 0; i < maxWorkers; i++ {
		dispatcher.workerPool <- struct{}{}
	}

	dispatcher.logger.Info("SchedulerDispatcher created", 
		"max_workers", maxWorkers, 
		"parallel_dispatch", enableParallelDispatch)

	return dispatcher
}

// RegisterHandler registers a handler function for a specific message label
func (d *SchedulerDispatcher) RegisterHandler(label string, handler HandlerFunc) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	d.handlers[label] = handler
	d.logger.Debug("Handler registered", "label", label)
}

// RegisterHandlers bulk registers multiple handlers from a map
func (d *SchedulerDispatcher) RegisterHandlers(handlers map[string]HandlerFunc) error {
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

// defaultMessageHandler provides a default handler for unregistered message types
func (d *SchedulerDispatcher) defaultMessageHandler(messages []*ScheduleMessageItem) error {
	d.logger.Debug("Using default message handler", "message_count", len(messages))
	for _, msg := range messages {
		d.logger.Debug("Processing with default handler", 
			"label", msg.Label, 
			"content_length", len(msg.Content))
	}
	return nil
}

// Dispatch processes a list of messages by routing them to appropriate handlers
func (d *SchedulerDispatcher) Dispatch(msgList []*ScheduleMessageItem) error {
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
func (d *SchedulerDispatcher) dispatchGroup(label string, msgs []*ScheduleMessageItem) error {
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

	if d.enableParallelDispatch {
		return d.dispatchParallel(handler, msgs)
	} else {
		return d.dispatchSerial(handler, msgs)
	}
}

// dispatchParallel handles messages in parallel using worker pool
func (d *SchedulerDispatcher) dispatchParallel(handler HandlerFunc, msgs []*ScheduleMessageItem) error {
	select {
	case <-d.workerPool: // Acquire worker
		go func() {
			defer func() {
				d.workerPool <- struct{}{} // Release worker
			}()
			
			if err := handler(msgs); err != nil {
				d.logger.Error("Parallel handler execution failed", "error", err)
			}
		}()
		return nil
	case <-d.ctx.Done():
		return fmt.Errorf("dispatcher context cancelled")
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting for available worker")
	}
}

// dispatchSerial handles messages serially
func (d *SchedulerDispatcher) dispatchSerial(handler HandlerFunc, msgs []*ScheduleMessageItem) error {
	return handler(msgs)
}

// Start begins the dispatcher service
func (d *SchedulerDispatcher) Start() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	if d.running {
		return fmt.Errorf("dispatcher is already running")
	}
	
	d.running = true
	d.logger.Info("Scheduler dispatcher started")
	return nil
}

// Stop gracefully shuts down the dispatcher
func (d *SchedulerDispatcher) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	if !d.running {
		return fmt.Errorf("dispatcher is not running")
	}
	
	d.cancel() // Cancel context to stop workers
	d.running = false
	
	// Wait for active workers to complete (with timeout)
	timeout := time.After(10 * time.Second)
	workerCount := 0
	
drainLoop:
	for {
		select {
		case <-d.workerPool:
			workerCount++
			if workerCount >= d.maxWorkers {
				break drainLoop
			}
		case <-timeout:
			d.logger.Warn("Timeout waiting for workers to complete")
			break drainLoop
		}
	}
	
	d.logger.Info("Scheduler dispatcher stopped")
	return nil
}

// GetStats returns dispatcher statistics
func (d *SchedulerDispatcher) GetStats() map[string]interface{} {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	availableWorkers := len(d.workerPool)
	activeWorkers := d.maxWorkers - availableWorkers
	
	return map[string]interface{}{
		"max_workers":       d.maxWorkers,
		"active_workers":    activeWorkers,
		"available_workers": availableWorkers,
		"parallel_enabled":  d.enableParallelDispatch,
		"running":          d.running,
		"handlers_count":   len(d.handlers),
	}
}

// IsRunning returns whether the dispatcher is currently running
func (d *SchedulerDispatcher) IsRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}
