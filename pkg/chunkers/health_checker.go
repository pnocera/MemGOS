package chunkers

import (
	"sync"
	"time"
)

// HealthChecker monitors the health of chunking strategies
type HealthChecker struct {
	checks        map[string]*HealthCheck
	checkInterval time.Duration
	mutex         sync.RWMutex
	isRunning     bool
	stopChan      chan struct{}
}

// HealthCheck represents health monitoring data for a strategy
type HealthCheck struct {
	StrategyName     string        `json:"strategy_name"`
	LastCheck        time.Time     `json:"last_check"`
	HealthScore      float64       `json:"health_score"`
	ResponseTime     time.Duration `json:"response_time"`
	SuccessRate      float64       `json:"success_rate"`
	ErrorRate        float64       `json:"error_rate"`
	IsHealthy        bool          `json:"is_healthy"`
	ConsecutiveFails int           `json:"consecutive_fails"`
	LastError        string        `json:"last_error,omitempty"`
}

// HealthStatus represents overall health status
type HealthStatus struct {
	OverallHealth     float64                 `json:"overall_health"`
	HealthyStrategies int                     `json:"healthy_strategies"`
	TotalStrategies   int                     `json:"total_strategies"`
	LastUpdated       time.Time               `json:"last_updated"`
	StrategyHealth    map[string]*HealthCheck `json:"strategy_health"`
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(checkInterval time.Duration) *HealthChecker {
	return &HealthChecker{
		checks:        make(map[string]*HealthCheck),
		checkInterval: checkInterval,
		stopChan:      make(chan struct{}),
	}
}

// Start begins health monitoring
func (hc *HealthChecker) Start() {
	hc.mutex.Lock()
	if hc.isRunning {
		hc.mutex.Unlock()
		return
	}
	hc.isRunning = true
	hc.mutex.Unlock()

	go hc.monitorHealth()
}

// Stop stops health monitoring
func (hc *HealthChecker) Stop() {
	hc.mutex.Lock()
	if !hc.isRunning {
		hc.mutex.Unlock()
		return
	}
	hc.isRunning = false
	hc.mutex.Unlock()

	close(hc.stopChan)
}

// RegisterStrategy registers a strategy for health monitoring
func (hc *HealthChecker) RegisterStrategy(strategyName string) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	hc.checks[strategyName] = &HealthCheck{
		StrategyName:     strategyName,
		LastCheck:        time.Now(),
		HealthScore:      1.0,
		SuccessRate:      1.0,
		ErrorRate:        0.0,
		IsHealthy:        true,
		ConsecutiveFails: 0,
	}
}

// RecordSuccess records a successful operation for a strategy
func (hc *HealthChecker) RecordSuccess(strategyName string, responseTime time.Duration) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	check, exists := hc.checks[strategyName]
	if !exists {
		hc.RegisterStrategy(strategyName)
		check = hc.checks[strategyName]
	}

	check.LastCheck = time.Now()
	check.ResponseTime = responseTime
	check.ConsecutiveFails = 0

	// Update success rate (exponential moving average)
	alpha := 0.1
	check.SuccessRate = alpha*1.0 + (1-alpha)*check.SuccessRate
	check.ErrorRate = alpha*0.0 + (1-alpha)*check.ErrorRate

	// Update health score
	hc.updateHealthScore(check)
}

// RecordFailure records a failed operation for a strategy
func (hc *HealthChecker) RecordFailure(strategyName string, err error) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	check, exists := hc.checks[strategyName]
	if !exists {
		hc.RegisterStrategy(strategyName)
		check = hc.checks[strategyName]
	}

	check.LastCheck = time.Now()
	check.ConsecutiveFails++
	if err != nil {
		check.LastError = err.Error()
	}

	// Update error rate (exponential moving average)
	alpha := 0.1
	check.SuccessRate = alpha*0.0 + (1-alpha)*check.SuccessRate
	check.ErrorRate = alpha*1.0 + (1-alpha)*check.ErrorRate

	// Update health score
	hc.updateHealthScore(check)
}

// GetHealthStatus returns the current health status of all strategies
func (hc *HealthChecker) GetHealthStatus() *HealthStatus {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	status := &HealthStatus{
		LastUpdated:    time.Now(),
		StrategyHealth: make(map[string]*HealthCheck),
	}

	totalHealth := 0.0
	healthyCount := 0

	for name, check := range hc.checks {
		// Create a copy of the health check
		checkCopy := *check
		status.StrategyHealth[name] = &checkCopy

		totalHealth += check.HealthScore
		if check.IsHealthy {
			healthyCount++
		}
	}

	status.TotalStrategies = len(hc.checks)
	status.HealthyStrategies = healthyCount

	if status.TotalStrategies > 0 {
		status.OverallHealth = totalHealth / float64(status.TotalStrategies)
	}

	return status
}

// GetStrategyHealth returns health information for a specific strategy
func (hc *HealthChecker) GetStrategyHealth(strategyName string) (*HealthCheck, bool) {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	check, exists := hc.checks[strategyName]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	checkCopy := *check
	return &checkCopy, true
}

// IsStrategyHealthy checks if a strategy is currently healthy
func (hc *HealthChecker) IsStrategyHealthy(strategyName string) bool {
	check, exists := hc.GetStrategyHealth(strategyName)
	if !exists {
		return false
	}

	return check.IsHealthy
}

// monitorHealth runs the background health monitoring
func (hc *HealthChecker) monitorHealth() {
	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.performHealthChecks()
		case <-hc.stopChan:
			return
		}
	}
}

// performHealthChecks performs periodic health assessments
func (hc *HealthChecker) performHealthChecks() {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	now := time.Now()

	for _, check := range hc.checks {
		// Check if strategy has been inactive for too long
		inactiveThreshold := 5 * time.Minute
		if now.Sub(check.LastCheck) > inactiveThreshold {
			// Gradually reduce health score for inactive strategies
			check.HealthScore *= 0.95
		}

		// Update health status based on consecutive failures
		if check.ConsecutiveFails > 5 {
			check.HealthScore *= 0.8
		}

		// Update overall health status
		hc.updateHealthScore(check)
	}
}

// updateHealthScore updates the health score and health status for a check
func (hc *HealthChecker) updateHealthScore(check *HealthCheck) {
	// Calculate health score based on multiple factors
	healthScore := 1.0

	// Factor 1: Success rate (50% weight)
	healthScore *= (0.5 * check.SuccessRate)

	// Factor 2: Error rate penalty (30% weight)
	healthScore *= (0.3 * (1.0 - check.ErrorRate))

	// Factor 3: Consecutive failures penalty (20% weight)
	consecutiveFailurePenalty := 1.0
	if check.ConsecutiveFails > 0 {
		consecutiveFailurePenalty = 1.0 / (1.0 + float64(check.ConsecutiveFails)*0.1)
	}
	healthScore *= (0.2 * consecutiveFailurePenalty)

	// Ensure health score is between 0 and 1
	if healthScore > 1.0 {
		healthScore = 1.0
	}
	if healthScore < 0.0 {
		healthScore = 0.0
	}

	check.HealthScore = healthScore

	// Determine if strategy is healthy (threshold of 0.5)
	check.IsHealthy = healthScore >= 0.5 && check.ConsecutiveFails < 3
}

// Reset resets health monitoring for a strategy
func (hc *HealthChecker) Reset(strategyName string) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	if check, exists := hc.checks[strategyName]; exists {
		check.HealthScore = 1.0
		check.SuccessRate = 1.0
		check.ErrorRate = 0.0
		check.IsHealthy = true
		check.ConsecutiveFails = 0
		check.LastError = ""
		check.LastCheck = time.Now()
	}
}

// RemoveStrategy removes a strategy from health monitoring
func (hc *HealthChecker) RemoveStrategy(strategyName string) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	delete(hc.checks, strategyName)
}
