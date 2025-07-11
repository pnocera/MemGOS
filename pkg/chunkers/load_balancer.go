package chunkers

import (
	"math"
	"sync"
	"time"
)

// LoadBalancer manages distribution of requests across chunking strategies
type LoadBalancer struct {
	strategies        map[string]*StrategyNode
	algorithm         LoadBalancingAlgorithm
	mutex             sync.RWMutex
	requestCounter    uint64
	healthThreshold   float64
}

// StrategyNode represents a strategy in the load balancer
type StrategyNode struct {
	Name              string        `json:"name"`
	Weight            float64       `json:"weight"`
	CurrentLoad       int           `json:"current_load"`
	MaxLoad           int           `json:"max_load"`
	ResponseTime      time.Duration `json:"response_time"`
	HealthScore       float64       `json:"health_score"`
	IsAvailable       bool          `json:"is_available"`
	LastUsed          time.Time     `json:"last_used"`
	RequestCount      uint64        `json:"request_count"`
	mutex             sync.RWMutex  `json:"-"`
}

// LoadBalancingAlgorithm defines the load balancing strategy
type LoadBalancingAlgorithm int

const (
	AlgorithmRoundRobin LoadBalancingAlgorithm = iota
	AlgorithmWeightedRoundRobin
	AlgorithmLeastConnections
	AlgorithmWeightedResponse
	AlgorithmHealthBased
	AlgorithmAdaptive
)

// LoadBalancingStats provides statistics about load balancing
type LoadBalancingStats struct {
	TotalRequests        uint64                      `json:"total_requests"`
	StrategyDistribution map[string]uint64           `json:"strategy_distribution"`
	AverageResponseTime  time.Duration               `json:"average_response_time"`
	StrategyStats        map[string]*StrategyStats   `json:"strategy_stats"`
	LastUpdated          time.Time                   `json:"last_updated"`
}

// StrategyStats provides detailed statistics for a strategy
type StrategyStats struct {
	RequestCount      uint64        `json:"request_count"`
	AverageResponse   time.Duration `json:"average_response"`
	CurrentLoad       int           `json:"current_load"`
	HealthScore       float64       `json:"health_score"`
	SuccessRate       float64       `json:"success_rate"`
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		strategies:      make(map[string]*StrategyNode),
		algorithm:       AlgorithmAdaptive,
		healthThreshold: 0.5,
	}
}

// AddStrategy adds a strategy to the load balancer
func (lb *LoadBalancer) AddStrategy(name string, weight float64, maxLoad int) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	
	lb.strategies[name] = &StrategyNode{
		Name:        name,
		Weight:      weight,
		MaxLoad:     maxLoad,
		HealthScore: 1.0,
		IsAvailable: true,
		LastUsed:    time.Now(),
	}
}

// RemoveStrategy removes a strategy from the load balancer
func (lb *LoadBalancer) RemoveStrategy(name string) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	
	delete(lb.strategies, name)
}

// SelectStrategy selects the best strategy based on the load balancing algorithm
func (lb *LoadBalancer) SelectStrategy() *StrategyNode {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()
	
	availableStrategies := lb.getAvailableStrategies()
	if len(availableStrategies) == 0 {
		return nil
	}
	
	switch lb.algorithm {
	case AlgorithmRoundRobin:
		return lb.selectRoundRobin(availableStrategies)
	case AlgorithmWeightedRoundRobin:
		return lb.selectWeightedRoundRobin(availableStrategies)
	case AlgorithmLeastConnections:
		return lb.selectLeastConnections(availableStrategies)
	case AlgorithmWeightedResponse:
		return lb.selectWeightedResponse(availableStrategies)
	case AlgorithmHealthBased:
		return lb.selectHealthBased(availableStrategies)
	case AlgorithmAdaptive:
		return lb.selectAdaptive(availableStrategies)
	default:
		return availableStrategies[0]
	}
}

// UpdateStrategyHealth updates the health score for a strategy
func (lb *LoadBalancer) UpdateStrategyHealth(name string, healthScore float64) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	
	if strategy, exists := lb.strategies[name]; exists {
		strategy.mutex.Lock()
		strategy.HealthScore = healthScore
		strategy.IsAvailable = healthScore >= lb.healthThreshold
		strategy.mutex.Unlock()
	}
}

// UpdateStrategyLoad updates the current load for a strategy
func (lb *LoadBalancer) UpdateStrategyLoad(name string, load int) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	
	if strategy, exists := lb.strategies[name]; exists {
		strategy.mutex.Lock()
		strategy.CurrentLoad = load
		strategy.IsAvailable = load < strategy.MaxLoad && strategy.HealthScore >= lb.healthThreshold
		strategy.mutex.Unlock()
	}
}

// UpdateStrategyResponseTime updates the response time for a strategy
func (lb *LoadBalancer) UpdateStrategyResponseTime(name string, responseTime time.Duration) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	
	if strategy, exists := lb.strategies[name]; exists {
		strategy.mutex.Lock()
		// Use exponential moving average for response time
		alpha := 0.1
		if strategy.ResponseTime == 0 {
			strategy.ResponseTime = responseTime
		} else {
			strategy.ResponseTime = time.Duration(alpha*float64(responseTime) + (1-alpha)*float64(strategy.ResponseTime))
		}
		strategy.LastUsed = time.Now()
		strategy.RequestCount++
		strategy.mutex.Unlock()
	}
}

// getAvailableStrategies returns strategies that are available for load balancing
func (lb *LoadBalancer) getAvailableStrategies() []*StrategyNode {
	var available []*StrategyNode
	
	for _, strategy := range lb.strategies {
		strategy.mutex.RLock()
		if strategy.IsAvailable {
			available = append(available, strategy)
		}
		strategy.mutex.RUnlock()
	}
	
	return available
}

// selectRoundRobin implements round-robin selection
func (lb *LoadBalancer) selectRoundRobin(strategies []*StrategyNode) *StrategyNode {
	lb.requestCounter++
	index := lb.requestCounter % uint64(len(strategies))
	return strategies[index]
}

// selectWeightedRoundRobin implements weighted round-robin selection
func (lb *LoadBalancer) selectWeightedRoundRobin(strategies []*StrategyNode) *StrategyNode {
	totalWeight := 0.0
	for _, strategy := range strategies {
		strategy.mutex.RLock()
		totalWeight += strategy.Weight
		strategy.mutex.RUnlock()
	}
	
	if totalWeight == 0 {
		return lb.selectRoundRobin(strategies)
	}
	
	// Generate weighted random selection
	target := float64(lb.requestCounter%100) / 100.0 * totalWeight
	cumulative := 0.0
	
	for _, strategy := range strategies {
		strategy.mutex.RLock()
		cumulative += strategy.Weight
		strategy.mutex.RUnlock()
		
		if cumulative >= target {
			return strategy
		}
	}
	
	return strategies[len(strategies)-1]
}

// selectLeastConnections implements least connections selection
func (lb *LoadBalancer) selectLeastConnections(strategies []*StrategyNode) *StrategyNode {
	var best *StrategyNode
	minLoad := int(^uint(0) >> 1) // Max int
	
	for _, strategy := range strategies {
		strategy.mutex.RLock()
		if strategy.CurrentLoad < minLoad {
			minLoad = strategy.CurrentLoad
			best = strategy
		}
		strategy.mutex.RUnlock()
	}
	
	return best
}

// selectWeightedResponse implements weighted response time selection
func (lb *LoadBalancer) selectWeightedResponse(strategies []*StrategyNode) *StrategyNode {
	var best *StrategyNode
	bestScore := 0.0
	
	for _, strategy := range strategies {
		strategy.mutex.RLock()
		
		// Calculate score based on weight and response time
		responseSeconds := float64(strategy.ResponseTime.Nanoseconds()) / float64(time.Second.Nanoseconds())
		if responseSeconds == 0 {
			responseSeconds = 0.001 // Avoid division by zero
		}
		
		score := strategy.Weight / responseSeconds
		
		if score > bestScore {
			bestScore = score
			best = strategy
		}
		
		strategy.mutex.RUnlock()
	}
	
	if best == nil {
		return strategies[0]
	}
	
	return best
}

// selectHealthBased implements health-based selection
func (lb *LoadBalancer) selectHealthBased(strategies []*StrategyNode) *StrategyNode {
	var best *StrategyNode
	bestHealth := 0.0
	
	for _, strategy := range strategies {
		strategy.mutex.RLock()
		if strategy.HealthScore > bestHealth {
			bestHealth = strategy.HealthScore
			best = strategy
		}
		strategy.mutex.RUnlock()
	}
	
	return best
}

// selectAdaptive implements adaptive selection based on multiple factors
func (lb *LoadBalancer) selectAdaptive(strategies []*StrategyNode) *StrategyNode {
	var best *StrategyNode
	bestScore := 0.0
	
	for _, strategy := range strategies {
		strategy.mutex.RLock()
		
		// Calculate composite score based on multiple factors
		score := lb.calculateAdaptiveScore(strategy)
		
		if score > bestScore {
			bestScore = score
			best = strategy
		}
		
		strategy.mutex.RUnlock()
	}
	
	if best == nil {
		return strategies[0]
	}
	
	return best
}

// calculateAdaptiveScore calculates a composite score for adaptive selection
func (lb *LoadBalancer) calculateAdaptiveScore(strategy *StrategyNode) float64 {
	// Weight factors
	healthWeight := 0.4
	loadWeight := 0.3
	responseWeight := 0.2
	recentUsageWeight := 0.1
	
	// Health score (0-1, higher is better)
	healthScore := strategy.HealthScore
	
	// Load score (0-1, lower load is better)
	loadScore := 1.0
	if strategy.MaxLoad > 0 {
		loadScore = 1.0 - (float64(strategy.CurrentLoad) / float64(strategy.MaxLoad))
	}
	
	// Response time score (0-1, lower response time is better)
	responseScore := 1.0
	if strategy.ResponseTime > 0 {
		// Normalize response time (assume 1 second as reference)
		responseSeconds := float64(strategy.ResponseTime.Nanoseconds()) / float64(time.Second.Nanoseconds())
		responseScore = 1.0 / (1.0 + responseSeconds)
	}
	
	// Recent usage score (favor less recently used strategies)
	recentUsageScore := 1.0
	timeSinceLastUsed := time.Since(strategy.LastUsed)
	if timeSinceLastUsed > 0 {
		// Favor strategies not used in the last minute
		minutesSinceUsed := float64(timeSinceLastUsed.Nanoseconds()) / float64(time.Minute.Nanoseconds())
		recentUsageScore = 1.0 / (1.0 + math.Exp(-minutesSinceUsed))
	}
	
	// Calculate weighted composite score
	compositeScore := healthWeight*healthScore +
		loadWeight*loadScore +
		responseWeight*responseScore +
		recentUsageWeight*recentUsageScore
	
	return compositeScore
}

// GetStats returns load balancing statistics
func (lb *LoadBalancer) GetStats() *LoadBalancingStats {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()
	
	stats := &LoadBalancingStats{
		TotalRequests:        lb.requestCounter,
		StrategyDistribution: make(map[string]uint64),
		StrategyStats:        make(map[string]*StrategyStats),
		LastUpdated:          time.Now(),
	}
	
	totalResponseTime := time.Duration(0)
	activeStrategies := 0
	
	for name, strategy := range lb.strategies {
		strategy.mutex.RLock()
		
		stats.StrategyDistribution[name] = strategy.RequestCount
		stats.StrategyStats[name] = &StrategyStats{
			RequestCount:    strategy.RequestCount,
			AverageResponse: strategy.ResponseTime,
			CurrentLoad:     strategy.CurrentLoad,
			HealthScore:     strategy.HealthScore,
			SuccessRate:     1.0, // This would be updated by external health monitoring
		}
		
		if strategy.RequestCount > 0 {
			totalResponseTime += strategy.ResponseTime
			activeStrategies++
		}
		
		strategy.mutex.RUnlock()
	}
	
	if activeStrategies > 0 {
		stats.AverageResponseTime = totalResponseTime / time.Duration(activeStrategies)
	}
	
	return stats
}

// SetAlgorithm sets the load balancing algorithm
func (lb *LoadBalancer) SetAlgorithm(algorithm LoadBalancingAlgorithm) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	
	lb.algorithm = algorithm
}

// GetAlgorithm returns the current load balancing algorithm
func (lb *LoadBalancer) GetAlgorithm() LoadBalancingAlgorithm {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()
	
	return lb.algorithm
}

// SetHealthThreshold sets the health threshold for strategy availability
func (lb *LoadBalancer) SetHealthThreshold(threshold float64) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	
	lb.healthThreshold = threshold
	
	// Update availability of existing strategies
	for _, strategy := range lb.strategies {
		strategy.mutex.Lock()
		strategy.IsAvailable = strategy.HealthScore >= threshold && strategy.CurrentLoad < strategy.MaxLoad
		strategy.mutex.Unlock()
	}
}