package core

import (
	"context"
	"fmt"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/schedulers"
	"github.com/memtensor/memgos/pkg/types"
)

// SchedulerAdapter adapts the schedulers.Scheduler to implement interfaces.Scheduler
type SchedulerAdapter struct {
	scheduler schedulers.Scheduler
	status    string
}

// NewSchedulerAdapter creates a new SchedulerAdapter
func NewSchedulerAdapter(scheduler schedulers.Scheduler) interfaces.Scheduler {
	return &SchedulerAdapter{
		scheduler: scheduler,
		status:    "stopped",
	}
}

// Start starts the scheduler
func (s *SchedulerAdapter) Start(ctx context.Context) error {
	err := s.scheduler.Start()
	if err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}
	
	s.status = "running"
	return nil
}

// Stop stops the scheduler
func (s *SchedulerAdapter) Stop(ctx context.Context) error {
	err := s.scheduler.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop scheduler: %w", err)
	}
	
	s.status = "stopped"
	return nil
}

// Schedule schedules a task
func (s *SchedulerAdapter) Schedule(ctx context.Context, task *types.ScheduledTask) error {
	// For now, we'll just log the task or submit it as a message
	// This would need to be implemented based on the actual scheduler requirements
	
	// Convert types.ScheduledTask to something the scheduler can handle
	// This is a simplified implementation
	if s.scheduler.IsRunning() {
		s.status = "processing"
		// TODO: Implement actual task submission based on scheduler interface
		s.status = "running"
		return nil
	}
	
	return fmt.Errorf("scheduler is not running")
}

// GetStatus returns the scheduler status
func (s *SchedulerAdapter) GetStatus() string {
	if s.scheduler.IsRunning() {
		return "running"
	}
	return s.status
}

// ListTasks lists all scheduled tasks
func (s *SchedulerAdapter) ListTasks(ctx context.Context) ([]*types.ScheduledTask, error) {
	// This would need to be implemented based on the actual scheduler interface
	// For now, return an empty list
	return []*types.ScheduledTask{}, nil
}