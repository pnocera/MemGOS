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
	"github.com/memtensor/memgos/pkg/types"
)

// NATSMonitorConfig holds configuration for NATS-based monitoring
type NATSMonitorConfig struct {
	MetricsSubject        string        `json:"metrics_subject" yaml:"metrics_subject"`
	AlertsSubject         string        `json:"alerts_subject" yaml:"alerts_subject"`
	HealthCheckSubject    string        `json:"health_check_subject" yaml:"health_check_subject"`
	MetricsInterval       time.Duration `json:"metrics_interval" yaml:"metrics_interval"`
	HealthCheckInterval   time.Duration `json:"health_check_interval" yaml:"health_check_interval"`
	RetentionPeriod       time.Duration `json:"retention_period" yaml:"retention_period"`
	AlertThresholds       *AlertThresholds `json:"alert_thresholds" yaml:"alert_thresholds"`
}

// AlertThresholds defines thresholds for various alerts
type AlertThresholds struct {
	ErrorRatePercent          float64       `json:"error_rate_percent" yaml:"error_rate_percent"`
	ProcessingTimeMs          int64         `json:"processing_time_ms" yaml:"processing_time_ms"`
	QueueDepthThreshold       int64         `json:"queue_depth_threshold" yaml:"queue_depth_threshold"`
	MemoryUsageThresholdMB    int64         `json:"memory_usage_threshold_mb" yaml:"memory_usage_threshold_mb"`
	UnhealthyWorkersThreshold int           `json:"unhealthy_workers_threshold" yaml:"unhealthy_workers_threshold"`
}

// DefaultNATSMonitorConfig returns default NATS monitor configuration
func DefaultNATSMonitorConfig() *NATSMonitorConfig {
	return &NATSMonitorConfig{
		MetricsSubject:      "scheduler.metrics",
		AlertsSubject:       "scheduler.alerts",
		HealthCheckSubject:  "scheduler.health",
		MetricsInterval:     30 * time.Second,
		HealthCheckInterval: 60 * time.Second,
		RetentionPeriod:     24 * time.Hour,
		AlertThresholds: &AlertThresholds{
			ErrorRatePercent:          10.0, // 10% error rate
			ProcessingTimeMs:          5000, // 5 seconds
			QueueDepthThreshold:       1000,
			MemoryUsageThresholdMB:    512,
			UnhealthyWorkersThreshold: 2,
		},
	}
}

// NATSSchedulerMonitor provides NATS-based distributed monitoring for scheduler
type NATSSchedulerMonitor struct {
	*BaseSchedulerModule
	mu                        sync.RWMutex
	config                    *NATSMonitorConfig
	conn                      *nats.Conn
	js                        jetstream.JetStream
	
	// Core monitoring components
	statistics               map[string]interface{}
	intentHistory            []string
	activationMemSize        int
	activationMemoryFreqList []*ActivationMemoryItem
	chatLLM                  interfaces.LLM
	
	// NATS-specific monitoring
	distributedMetrics       map[string]*NodeMetrics
	clusterHealth           *ClusterHealth
	
	// Monitoring control
	monitoringActive         bool
	monitoringCtx            context.Context
	monitoringCancel         context.CancelFunc
	
	// Statistics tracking
	processedTasks           int64
	errorCount               int64
	totalProcessingTime      time.Duration
	lastMetricsPublish       time.Time
	lastHealthCheck          time.Time
	
	logger                   interfaces.Logger
	startTime                time.Time
	nodeID                   string
}

// NodeMetrics represents metrics for a single node
type NodeMetrics struct {
	NodeID              string            `json:"node_id"`
	Timestamp           time.Time         `json:"timestamp"`
	ProcessedTasks      int64             `json:"processed_tasks"`
	ErrorCount          int64             `json:"error_count"`
	ActiveWorkers       int               `json:"active_workers"`
	QueueDepth          int64             `json:"queue_depth"`
	MemoryUsageMB       int64             `json:"memory_usage_mb"`
	CPUUsagePercent     float64           `json:"cpu_usage_percent"`
	AvgProcessingTimeMs int64             `json:"avg_processing_time_ms"`
	CustomMetrics       map[string]interface{} `json:"custom_metrics,omitempty"`
}

// ClusterHealth represents overall cluster health
type ClusterHealth struct {
	Timestamp           time.Time              `json:"timestamp"`
	TotalNodes          int                    `json:"total_nodes"`
	HealthyNodes        int                    `json:"healthy_nodes"`
	UnhealthyNodes      []string               `json:"unhealthy_nodes"`
	TotalProcessedTasks int64                  `json:"total_processed_tasks"`
	TotalErrors         int64                  `json:"total_errors"`
	OverallErrorRate    float64                `json:"overall_error_rate"`
	AverageLoadPercent  float64                `json:"average_load_percent"`
	ClusterStats        map[string]interface{} `json:"cluster_stats"`
}

// Alert represents a monitoring alert
type Alert struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	NodeID      string                 `json:"node_id"`
	Level       string                 `json:"level"` // INFO, WARN, ERROR, CRITICAL
	Type        string                 `json:"type"`  // ERROR_RATE, PROCESSING_TIME, QUEUE_DEPTH, etc.
	Message     string                 `json:"message"`
	Value       interface{}            `json:"value"`
	Threshold   interface{}            `json:"threshold"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NewNATSSchedulerMonitor creates a new NATS-based scheduler monitor
func NewNATSSchedulerMonitor(conn *nats.Conn, js jetstream.JetStream, chatLLM interfaces.LLM, activationMemSize int, config *NATSMonitorConfig) *NATSSchedulerMonitor {
	if activationMemSize <= 0 {
		activationMemSize = DefaultActivationMemSize
	}
	
	if config == nil {
		config = DefaultNATSMonitorConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	// Generate unique node ID
	nodeID := fmt.Sprintf("scheduler-node-%d", time.Now().UnixNano())
	
	monitor := &NATSSchedulerMonitor{
		BaseSchedulerModule:      NewBaseSchedulerModule(),
		config:                   config,
		conn:                     conn,
		js:                       js,
		statistics:               make(map[string]interface{}),
		intentHistory:            make([]string, 0),
		activationMemSize:        activationMemSize,
		activationMemoryFreqList: make([]*ActivationMemoryItem, activationMemSize),
		chatLLM:                  chatLLM,
		distributedMetrics:       make(map[string]*NodeMetrics),
		clusterHealth:            &ClusterHealth{},
		monitoringActive:         false,
		monitoringCtx:            ctx,
		monitoringCancel:         cancel,
		processedTasks:           0,
		errorCount:               0,
		totalProcessingTime:      0,
		lastMetricsPublish:       time.Time{},
		lastHealthCheck:          time.Time{},
		logger:                   logger.NewLogger(),
		startTime:                time.Now(),
		nodeID:                   nodeID,
	}
	
	// Initialize activation memory with empty items
	for i := 0; i < activationMemSize; i++ {
		monitor.activationMemoryFreqList[i] = &ActivationMemoryItem{
			Memory: "",
			Count:  0,
		}
	}
	
	// Initialize default templates
	if err := monitor.InitializeDefaultTemplates(); err != nil {
		monitor.logger.Error("Failed to initialize templates", err, map[string]interface{}{})
	}
	
	return monitor
}

// InitializeMonitoringStreams sets up NATS streams for monitoring
func (m *NATSSchedulerMonitor) InitializeMonitoringStreams() error {
	if m.js == nil {
		return fmt.Errorf("JetStream not initialized")
	}
	
	// Create monitoring streams
	streamConfig := jetstream.StreamConfig{
		Name:        "SCHEDULER_MONITORING",
		Subjects:    []string{m.config.MetricsSubject, m.config.AlertsSubject, m.config.HealthCheckSubject},
		Retention:   jetstream.LimitsPolicy,
		Discard:     jetstream.DiscardOld,
		MaxAge:      m.config.RetentionPeriod,
		MaxBytes:    256 * 1024 * 1024, // 256MB
		MaxMsgs:     50000,
		Replicas:    1,
	}
	
	_, err := m.js.CreateOrUpdateStream(m.monitoringCtx, streamConfig)
	if err != nil {
		return fmt.Errorf("failed to create monitoring stream: %w", err)
	}
	
	// Subscribe to metrics from other nodes
	if err := m.subscribeToClusterMetrics(); err != nil {
		m.logger.Warn("Failed to subscribe to cluster metrics", map[string]interface{}{"error": err.Error()})
	}
	
	m.logger.Info("NATS monitoring streams initialized", map[string]interface{}{
		"metrics_subject": m.config.MetricsSubject,
		"alerts_subject": m.config.AlertsSubject,
		"health_subject": m.config.HealthCheckSubject,
		"node_id": m.nodeID,
	})
	
	return nil
}

// subscribeToClusterMetrics subscribes to metrics from other nodes
func (m *NATSSchedulerMonitor) subscribeToClusterMetrics() error {
	// Subscribe to metrics from all nodes
	_, err := m.conn.Subscribe(m.config.MetricsSubject, func(msg *nats.Msg) {
		var nodeMetrics NodeMetrics
		if err := json.Unmarshal(msg.Data, &nodeMetrics); err != nil {
			m.logger.Error("Failed to unmarshal node metrics", err, map[string]interface{}{})
			return
		}
		
		// Only process metrics from other nodes
		if nodeMetrics.NodeID != m.nodeID {
			m.mu.Lock()
			m.distributedMetrics[nodeMetrics.NodeID] = &nodeMetrics
			m.mu.Unlock()
			
			m.logger.Debug("Received metrics from node", map[string]interface{}{"node_id": nodeMetrics.NodeID})
		}
	})
	
	if err != nil {
		return fmt.Errorf("failed to subscribe to metrics: %w", err)
	}
	
	// Subscribe to alerts
	_, err = m.conn.Subscribe(m.config.AlertsSubject, func(msg *nats.Msg) {
		var alert Alert
		if err := json.Unmarshal(msg.Data, &alert); err != nil {
			m.logger.Error("Failed to unmarshal alert", err, map[string]interface{}{})
			return
		}
		
		m.logger.Info("Received alert", map[string]interface{}{
			"id": alert.ID,
			"level": alert.Level,
			"type": alert.Type,
			"message": alert.Message,
			"node_id": alert.NodeID,
		})
	})
	
	if err != nil {
		return fmt.Errorf("failed to subscribe to alerts: %w", err)
	}
	
	return nil
}

// StartMonitoring begins the monitoring loop
func (m *NATSSchedulerMonitor) StartMonitoring() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.monitoringActive {
		return fmt.Errorf("monitoring is already active")
	}
	
	m.monitoringActive = true
	
	// Start metrics publishing goroutine
	go m.metricsPublishLoop()
	
	// Start health check goroutine
	go m.healthCheckLoop()
	
	// Start cluster monitoring goroutine
	go m.clusterMonitoringLoop()
	
	m.logger.Info("NATS monitoring started", map[string]interface{}{
		"node_id": m.nodeID,
	})
	return nil
}

// metricsPublishLoop continuously publishes node metrics
func (m *NATSSchedulerMonitor) metricsPublishLoop() {
	ticker := time.NewTicker(m.config.MetricsInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.monitoringCtx.Done():
			return
		case <-ticker.C:
			if err := m.publishNodeMetrics(); err != nil {
				m.logger.Error("Failed to publish node metrics", err, map[string]interface{}{})
			}
		}
	}
}

// healthCheckLoop continuously performs health checks
func (m *NATSSchedulerMonitor) healthCheckLoop() {
	ticker := time.NewTicker(m.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.monitoringCtx.Done():
			return
		case <-ticker.C:
			if err := m.performHealthCheck(); err != nil {
				m.logger.Error("Health check failed", err, map[string]interface{}{})
			}
		}
	}
}

// clusterMonitoringLoop continuously monitors cluster health
func (m *NATSSchedulerMonitor) clusterMonitoringLoop() {
	ticker := time.NewTicker(m.config.HealthCheckInterval * 2) // Less frequent
	defer ticker.Stop()
	
	for {
		select {
		case <-m.monitoringCtx.Done():
			return
		case <-ticker.C:
			m.updateClusterHealth()
			m.checkAlertThresholds()
		}
	}
}

// publishNodeMetrics publishes current node metrics to NATS
func (m *NATSSchedulerMonitor) publishNodeMetrics() error {
	m.mu.RLock()
	
	avgProcessingTime := int64(0)
	if m.processedTasks > 0 {
		avgProcessingTime = m.totalProcessingTime.Milliseconds() / m.processedTasks
	}
	
	metrics := &NodeMetrics{
		NodeID:              m.nodeID,
		Timestamp:           time.Now(),
		ProcessedTasks:      m.processedTasks,
		ErrorCount:          m.errorCount,
		ActiveWorkers:       0, // Would need to get from dispatcher
		QueueDepth:          0, // Would need to get from NATS consumer
		MemoryUsageMB:       0, // Would use runtime.MemStats
		CPUUsagePercent:     0, // Would use system monitoring
		AvgProcessingTimeMs: avgProcessingTime,
		CustomMetrics: map[string]interface{}{
			"activation_mem_size":  m.activationMemSize,
			"intent_history_count": len(m.intentHistory),
			"uptime_seconds":       time.Since(m.startTime).Seconds(),
		},
	}
	
	m.mu.RUnlock()
	
	// Publish metrics
	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}
	
	if err := m.conn.Publish(m.config.MetricsSubject, data); err != nil {
		return fmt.Errorf("failed to publish metrics: %w", err)
	}
	
	m.mu.Lock()
	m.lastMetricsPublish = time.Now()
	m.mu.Unlock()
	
	m.logger.Debug("Node metrics published", map[string]interface{}{"node_id": m.nodeID})
	return nil
}

// performHealthCheck performs and publishes health check
func (m *NATSSchedulerMonitor) performHealthCheck() error {
	healthStatus := map[string]interface{}{
		"node_id":          m.nodeID,
		"timestamp":        time.Now(),
		"healthy":          true,
		"nats_connected":   m.conn.IsConnected(),
		"uptime_seconds":   time.Since(m.startTime).Seconds(),
		"monitoring_active": m.monitoringActive,
	}
	
	// Add LLM health check
	if m.chatLLM != nil {
		// Could add a simple LLM ping test here
		healthStatus["llm_available"] = true
	} else {
		healthStatus["llm_available"] = false
		healthStatus["healthy"] = false
	}
	
	// Publish health status
	data, err := json.Marshal(healthStatus)
	if err != nil {
		return fmt.Errorf("failed to marshal health status: %w", err)
	}
	
	if err := m.conn.Publish(m.config.HealthCheckSubject, data); err != nil {
		return fmt.Errorf("failed to publish health status: %w", err)
	}
	
	m.mu.Lock()
	m.lastHealthCheck = time.Now()
	m.mu.Unlock()
	
	return nil
}

// updateClusterHealth calculates and updates cluster health metrics
func (m *NATSSchedulerMonitor) updateClusterHealth() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	totalNodes := 1 + len(m.distributedMetrics) // Include this node
	healthyNodes := 1 // Assume this node is healthy
	var unhealthyNodes []string
	var totalProcessedTasks int64 = m.processedTasks
	var totalErrors int64 = m.errorCount
	
	// Check health of other nodes
	cutoffTime := time.Now().Add(-m.config.HealthCheckInterval * 3) // 3x interval for unhealthy
	
	for nodeID, metrics := range m.distributedMetrics {
		if metrics.Timestamp.Before(cutoffTime) {
			unhealthyNodes = append(unhealthyNodes, nodeID)
		} else {
			healthyNodes++
		}
		
		totalProcessedTasks += metrics.ProcessedTasks
		totalErrors += metrics.ErrorCount
	}
	
	// Calculate error rate
	errorRate := 0.0
	if totalProcessedTasks > 0 {
		errorRate = (float64(totalErrors) / float64(totalProcessedTasks)) * 100
	}
	
	// Update cluster health
	m.clusterHealth = &ClusterHealth{
		Timestamp:           time.Now(),
		TotalNodes:          totalNodes,
		HealthyNodes:        healthyNodes,
		UnhealthyNodes:      unhealthyNodes,
		TotalProcessedTasks: totalProcessedTasks,
		TotalErrors:         totalErrors,
		OverallErrorRate:    errorRate,
		AverageLoadPercent:  0, // Would calculate based on worker utilization
		ClusterStats: map[string]interface{}{
			"monitoring_nodes": len(m.distributedMetrics),
			"cluster_uptime":   time.Since(m.startTime).Seconds(),
		},
	}
}

// checkAlertThresholds checks for alert conditions and publishes alerts
func (m *NATSSchedulerMonitor) checkAlertThresholds() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	thresholds := m.config.AlertThresholds
	
	// Check error rate
	if m.processedTasks > 0 {
		errorRate := (float64(m.errorCount) / float64(m.processedTasks)) * 100
		if errorRate > thresholds.ErrorRatePercent {
			m.publishAlert("ERROR_RATE", "ERROR", 
				fmt.Sprintf("Error rate %.2f%% exceeds threshold %.2f%%", errorRate, thresholds.ErrorRatePercent),
				errorRate, thresholds.ErrorRatePercent)
		}
	}
	
	// Check unhealthy workers
	if len(m.clusterHealth.UnhealthyNodes) > thresholds.UnhealthyWorkersThreshold {
		m.publishAlert("UNHEALTHY_WORKERS", "WARN",
			fmt.Sprintf("%d unhealthy nodes exceed threshold %d", len(m.clusterHealth.UnhealthyNodes), thresholds.UnhealthyWorkersThreshold),
			len(m.clusterHealth.UnhealthyNodes), thresholds.UnhealthyWorkersThreshold)
	}
	
	// Check cluster error rate
	if m.clusterHealth.OverallErrorRate > thresholds.ErrorRatePercent {
		m.publishAlert("CLUSTER_ERROR_RATE", "CRITICAL",
			fmt.Sprintf("Cluster error rate %.2f%% exceeds threshold %.2f%%", m.clusterHealth.OverallErrorRate, thresholds.ErrorRatePercent),
			m.clusterHealth.OverallErrorRate, thresholds.ErrorRatePercent)
	}
}

// publishAlert publishes an alert to NATS
func (m *NATSSchedulerMonitor) publishAlert(alertType, level, message string, value, threshold interface{}) {
	alert := &Alert{
		ID:        fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		NodeID:    m.nodeID,
		Level:     level,
		Type:      alertType,
		Message:   message,
		Value:     value,
		Threshold: threshold,
		Metadata: map[string]interface{}{
			"cluster_health": m.clusterHealth,
		},
	}
	
	data, err := json.Marshal(alert)
	if err != nil {
		m.logger.Error("Failed to marshal alert", err, map[string]interface{}{})
		return
	}
	
	if err := m.conn.Publish(m.config.AlertsSubject, data); err != nil {
		m.logger.Error("Failed to publish alert", err, map[string]interface{}{})
		return
	}
	
	m.logger.Warn("Alert published", map[string]interface{}{
		"id":      alert.ID,
		"type":    alertType,
		"level":   level,
		"message": message,
	})
}

// StopMonitoring stops the monitoring loop
func (m *NATSSchedulerMonitor) StopMonitoring() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.monitoringActive {
		return fmt.Errorf("monitoring is not active")
	}
	
	m.monitoringCancel()
	m.monitoringActive = false
	
	m.logger.Info("NATS monitoring stopped", map[string]interface{}{
		"node_id": m.nodeID,
	})
	return nil
}

// UpdateStats updates monitor statistics with memory cube information
func (m *NATSSchedulerMonitor) UpdateStats(memCube interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.statistics["activation_mem_size"] = m.activationMemSize
	m.statistics["processed_tasks"] = m.processedTasks
	m.statistics["error_count"] = m.errorCount
	m.statistics["uptime"] = time.Since(m.startTime)
	m.statistics["node_id"] = m.nodeID
	m.statistics["monitoring_active"] = m.monitoringActive
	m.statistics["cluster_health"] = m.clusterHealth
	
	// Add memory cube specific information if available
	if memCubeInfo := m.getMemCubeInfo(memCube); memCubeInfo != nil {
		m.statistics["mem_cube"] = memCubeInfo
	}
}

// getMemCubeInfo extracts information from memory cube
func (m *NATSSchedulerMonitor) getMemCubeInfo(memCube interface{}) map[string]interface{} {
	if memCube == nil {
		return nil
	}
	
	return map[string]interface{}{
		"type":        fmt.Sprintf("%T", memCube),
		"initialized": true,
	}
}

// DetectIntent analyzes user queries and working memory to determine if retrieval is needed
func (m *NATSSchedulerMonitor) DetectIntent(qList []string, textWorkingMemory []string) (*IntentResult, error) {
	startTime := time.Now()
	defer func() {
		processingTime := time.Since(startTime)
		m.mu.Lock()
		m.totalProcessingTime += processingTime
		m.mu.Unlock()
	}()
	
	if m.chatLLM == nil {
		m.IncrementErrorCount()
		return nil, fmt.Errorf("chat LLM not initialized")
	}
	
	params := map[string]interface{}{
		"q_list":             qList,
		"working_memory_list": textWorkingMemory,
	}
	
	prompt, err := m.BuildPrompt("intent_recognizing", params)
	if err != nil {
		m.IncrementErrorCount()
		return nil, fmt.Errorf("failed to build intent recognition prompt: %w", err)
	}
	
	m.logger.Debug("Detecting intent", map[string]interface{}{
		"query_count":  len(qList),
		"memory_count": len(textWorkingMemory),
	})
	
	// Generate response from LLM
	messages := types.MessageList{
		{Role: "user", Content: prompt},
	}
	
	response, err := m.chatLLM.Generate(context.Background(), messages)
	if err != nil {
		m.IncrementErrorCount()
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}
	
	// Parse JSON response
	intentResult, err := m.extractJSONDict(response)
	if err != nil {
		m.IncrementErrorCount()
		return nil, fmt.Errorf("failed to parse intent response: %w", err)
	}
	
	// Convert to IntentResult struct
	result := &IntentResult{
		TriggerRetrieval: getBoolValue(intentResult, "trigger_retrieval"),
		MissingEvidence:  getStringSliceValue(intentResult, "missing_evidence"),
		Confidence:       getFloat64Value(intentResult, "confidence"),
	}
	
	// Add to intent history
	m.mu.Lock()
	m.intentHistory = append(m.intentHistory, fmt.Sprintf("trigger:%t, evidence:%d", 
		result.TriggerRetrieval, len(result.MissingEvidence)))
	
	// Keep only recent history
	if len(m.intentHistory) > 100 {
		m.intentHistory = m.intentHistory[len(m.intentHistory)-100:]
	}
	m.mu.Unlock()
	
	m.IncrementProcessedTasks()
	
	m.logger.Debug("Intent detected", map[string]interface{}{
		"trigger_retrieval":       result.TriggerRetrieval,
		"missing_evidence_count": len(result.MissingEvidence),
		"confidence":             result.Confidence,
	})
	
	return result, nil
}

// UpdateFreq uses LLM to detect which memories appear in the answer and updates their frequency
func (m *NATSSchedulerMonitor) UpdateFreq(answer string) ([]*ActivationMemoryItem, error) {
	startTime := time.Now()
	defer func() {
		processingTime := time.Since(startTime)
		m.mu.Lock()
		m.totalProcessingTime += processingTime
		m.mu.Unlock()
	}()
	
	if m.chatLLM == nil {
		m.IncrementErrorCount()
		return m.activationMemoryFreqList, fmt.Errorf("chat LLM not initialized")
	}
	
	m.mu.RLock()
	freqList := make([]*ActivationMemoryItem, len(m.activationMemoryFreqList))
	copy(freqList, m.activationMemoryFreqList)
	m.mu.RUnlock()
	
	params := map[string]interface{}{
		"answer":                        answer,
		"activation_memory_freq_list":   freqList,
	}
	
	prompt, err := m.BuildPrompt("freq_detecting", params)
	if err != nil {
		m.IncrementErrorCount()
		return freqList, fmt.Errorf("failed to build frequency detection prompt: %w", err)
	}
	
	// Generate response from LLM
	messages := types.MessageList{
		{Role: "user", Content: prompt},
	}
	
	response, err := m.chatLLM.Generate(context.Background(), messages)
	if err != nil {
		m.IncrementErrorCount()
		m.logger.Error("LLM generation failed for frequency update", err, map[string]interface{}{})
		return freqList, err
	}
	
	// Parse JSON response
	result, err := m.extractJSONDict(response)
	if err != nil {
		m.IncrementErrorCount()
		m.logger.Error("Failed to parse frequency response", err, map[string]interface{}{})
		return freqList, err
	}
	
	// Extract updated frequency list
	if updatedList, ok := result["activation_memory_freq_list"]; ok {
		if updatedItems, err := m.parseActivationMemoryItems(updatedList); err == nil {
			m.mu.Lock()
			m.activationMemoryFreqList = updatedItems
			m.mu.Unlock()
			m.IncrementProcessedTasks()
			return updatedItems, nil
		} else {
			m.IncrementErrorCount()
			m.logger.Error("Failed to parse updated activation memory items", err, map[string]interface{}{})
		}
	}
	
	return freqList, nil
}

// parseActivationMemoryItems converts interface{} to []*ActivationMemoryItem
func (m *NATSSchedulerMonitor) parseActivationMemoryItems(data interface{}) ([]*ActivationMemoryItem, error) {
	switch items := data.(type) {
	case []interface{}:
		result := make([]*ActivationMemoryItem, 0, len(items))
		for _, item := range items {
			if itemMap, ok := item.(map[string]interface{}); ok {
				memItem := &ActivationMemoryItem{
					Memory: getStringValue(itemMap, "memory"),
					Count:  int(getFloat64Value(itemMap, "count")),
				}
				result = append(result, memItem)
			}
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unexpected data type for activation memory items: %T", data)
	}
}

// GetActivationMemoryFreqList returns a copy of the current activation memory frequency list
func (m *NATSSchedulerMonitor) GetActivationMemoryFreqList() []*ActivationMemoryItem {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make([]*ActivationMemoryItem, len(m.activationMemoryFreqList))
	copy(result, m.activationMemoryFreqList)
	return result
}

// GetStatistics returns a copy of current statistics including distributed metrics
func (m *NATSSchedulerMonitor) GetStatistics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := make(map[string]interface{})
	for k, v := range m.statistics {
		stats[k] = v
	}
	
	// Add distributed metrics
	distributedStats := make(map[string]interface{})
	for nodeID, metrics := range m.distributedMetrics {
		distributedStats[nodeID] = metrics
	}
	stats["distributed_metrics"] = distributedStats
	
	return stats
}

// IncrementProcessedTasks increments the processed tasks counter
func (m *NATSSchedulerMonitor) IncrementProcessedTasks() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processedTasks++
}

// IncrementErrorCount increments the error counter
func (m *NATSSchedulerMonitor) IncrementErrorCount() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorCount++
}

// GetIntentHistory returns recent intent detection history
func (m *NATSSchedulerMonitor) GetIntentHistory() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	history := make([]string, len(m.intentHistory))
	copy(history, m.intentHistory)
	return history
}

// GetClusterHealth returns current cluster health
func (m *NATSSchedulerMonitor) GetClusterHealth() *ClusterHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Return a copy
	health := *m.clusterHealth
	return &health
}

// GetNodeID returns the current node ID
func (m *NATSSchedulerMonitor) GetNodeID() string {
	return m.nodeID
}

// extractJSONDict extracts JSON dictionary from LLM response
func (m *NATSSchedulerMonitor) extractJSONDict(response string) (map[string]interface{}, error) {
	// Try to parse as JSON directly
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err == nil {
		return result, nil
	}
	
	// If direct parsing fails, try to extract JSON from markdown or other formats
	// Look for JSON blocks in the response
	start := -1
	end := -1
	
	for i, char := range response {
		if char == '{' && start == -1 {
			start = i
		}
		if char == '}' {
			end = i + 1
		}
	}
	
	if start != -1 && end != -1 && end > start {
		jsonStr := response[start:end]
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
			return result, nil
		}
	}
	
	return nil, fmt.Errorf("failed to extract JSON from response: %s", response)
}

// Helper functions for type conversion
func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getFloat64Value function removed - using the one from monitor.go