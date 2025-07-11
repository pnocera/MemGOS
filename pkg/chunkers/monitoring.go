package chunkers

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// RealTimeMonitor provides comprehensive real-time monitoring for chunking systems
type RealTimeMonitor struct {
	metrics         *MonitoringMetrics
	alerts          chan Alert
	thresholds      map[string]float64
	collectors      []MetricCollector
	alertHandlers   []AlertHandler
	isRunning       bool
	stopChan        chan struct{}
	config          *MonitoringConfig
	storage         MetricStorage
	mutex           sync.RWMutex
}

// MonitoringConfig configures real-time monitoring
type MonitoringConfig struct {
	SampleInterval    time.Duration         `json:"sample_interval"`
	AlertBuffer       int                   `json:"alert_buffer"`
	Thresholds        map[string]float64    `json:"thresholds"`
	EnableAlerts      bool                  `json:"enable_alerts"`
	EnablePersistence bool                  `json:"enable_persistence"`
	RetentionPeriod   time.Duration         `json:"retention_period"`
	AlertCooldown     time.Duration         `json:"alert_cooldown"`
	MetricAggregation map[string]string     `json:"metric_aggregation"`
}

// MonitoringMetrics contains comprehensive monitoring data
type MonitoringMetrics struct {
	// System metrics
	Timestamp         time.Time             `json:"timestamp"`
	CPUUsage          float64               `json:"cpu_usage"`
	MemoryUsage       int64                 `json:"memory_usage"`
	MemoryTotal       int64                 `json:"memory_total"`
	GoroutineCount    int                   `json:"goroutine_count"`
	GCPauseTime       time.Duration         `json:"gc_pause_time"`
	
	// Request metrics
	RequestCount      int64                 `json:"request_count"`
	ResponseTimes     []time.Duration       `json:"response_times"`
	ErrorRate         float64               `json:"error_rate"`
	Throughput        float64               `json:"throughput"`
	ActiveRequests    int64                 `json:"active_requests"`
	
	// Chunking specific metrics
	ChunksGenerated   int64                 `json:"chunks_generated"`
	AverageChunkSize  float64               `json:"average_chunk_size"`
	QualityScore      float64               `json:"quality_score"`
	ProcessingLatency time.Duration         `json:"processing_latency"`
	
	// Strategy metrics
	StrategyUsage     map[string]int64      `json:"strategy_usage"`
	StrategyErrors    map[string]int64      `json:"strategy_errors"`
	StrategyLatency   map[string]time.Duration `json:"strategy_latency"`
	
	// Cache metrics
	CacheHitRate      float64               `json:"cache_hit_rate"`
	CacheSize         int                   `json:"cache_size"`
	CacheEvictions    int64                 `json:"cache_evictions"`
	
	mutex             sync.RWMutex          `json:"-"`
}

// Alert represents a monitoring alert with severity and context
type Alert struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    AlertSeverity          `json:"severity"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Timestamp   time.Time              `json:"timestamp"`
	Metrics     map[string]interface{} `json:"metrics"`
	Source      string                 `json:"source"`
	Tags        []string               `json:"tags"`
	Resolution  string                 `json:"resolution,omitempty"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
}

// AlertSeverity defines alert severity levels
type AlertSeverity int

const (
	AlertSeverityInfo AlertSeverity = iota
	AlertSeverityWarning
	AlertSeverityError
	AlertSeverityCritical
)

func (s AlertSeverity) String() string {
	switch s {
	case AlertSeverityInfo:
		return "info"
	case AlertSeverityWarning:
		return "warning"
	case AlertSeverityError:
		return "error"
	case AlertSeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// MetricCollector defines interface for collecting specific metrics
type MetricCollector interface {
	Name() string
	Collect() (map[string]interface{}, error)
	Interval() time.Duration
}

// AlertHandler defines interface for handling alerts
type AlertHandler interface {
	HandleAlert(alert Alert) error
	CanHandle(alertType string) bool
}

// MetricStorage defines interface for storing metrics
type MetricStorage interface {
	Store(metrics *MonitoringMetrics) error
	Query(start, end time.Time, filters map[string]string) ([]*MonitoringMetrics, error)
	Cleanup(before time.Time) error
}

// MonitoringHealthStatus represents system health information for monitoring
type MonitoringHealthStatus struct {
	Status        string                 `json:"status"`
	Score         float64                `json:"score"`
	Checks        map[string]CheckResult `json:"checks"`
	Timestamp     time.Time              `json:"timestamp"`
	Uptime        time.Duration          `json:"uptime"`
	Version       string                 `json:"version"`
}

// CheckResult represents individual health check result
type CheckResult struct {
	Status    string                 `json:"status"`
	Message   string                 `json:"message"`
	Duration  time.Duration          `json:"duration"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// PerformanceProfile represents performance profiling data
type MonitoringPerformanceProfile struct {
	CPUProfile    []byte                 `json:"cpu_profile,omitempty"`
	MemProfile    []byte                 `json:"mem_profile,omitempty"`
	Traces        []TraceData            `json:"traces"`
	Timestamp     time.Time              `json:"timestamp"`
	Duration      time.Duration          `json:"duration"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// TraceData represents execution trace information
type TraceData struct {
	ID          string                 `json:"id"`
	Operation   string                 `json:"operation"`
	StartTime   time.Time              `json:"start_time"`
	Duration    time.Duration          `json:"duration"`
	Success     bool                   `json:"success"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NewRealTimeMonitor creates a new real-time monitor
func NewRealTimeMonitor(config *MonitoringConfig) *RealTimeMonitor {
	if config == nil {
		config = &MonitoringConfig{
			SampleInterval:  10 * time.Second,
			AlertBuffer:     100,
			EnableAlerts:    true,
			AlertCooldown:   5 * time.Minute,
			RetentionPeriod: 24 * time.Hour,
		}
	}
	
	monitor := &RealTimeMonitor{
		metrics:       &MonitoringMetrics{},
		alerts:        make(chan Alert, config.AlertBuffer),
		thresholds:    make(map[string]float64),
		collectors:    make([]MetricCollector, 0),
		alertHandlers: make([]AlertHandler, 0),
		stopChan:      make(chan struct{}),
		config:        config,
		storage:       &InMemoryMetricStorage{},
	}
	
	// Add default collectors
	monitor.AddCollector(&SystemMetricCollector{})
	monitor.AddCollector(&RuntimeMetricCollector{})
	monitor.AddCollector(&ChunkingMetricCollector{})
	
	// Add default alert handlers
	monitor.AddAlertHandler(&LogAlertHandler{})
	
	return monitor
}

// Start begins monitoring
func (rtm *RealTimeMonitor) Start() {
	rtm.mutex.Lock()
	defer rtm.mutex.Unlock()
	
	if rtm.isRunning {
		return
	}
	
	rtm.isRunning = true
	
	// Start metric collection
	go rtm.runMetricCollection()
	
	// Start alert processing
	if rtm.config.EnableAlerts {
		go rtm.runAlertProcessing()
	}
	
	// Start metric storage cleanup
	if rtm.config.EnablePersistence {
		go rtm.runStorageCleanup()
	}
}

// Stop stops monitoring
func (rtm *RealTimeMonitor) Stop() {
	rtm.mutex.Lock()
	defer rtm.mutex.Unlock()
	
	if !rtm.isRunning {
		return
	}
	
	rtm.isRunning = false
	close(rtm.stopChan)
}

// AddCollector adds a metric collector
func (rtm *RealTimeMonitor) AddCollector(collector MetricCollector) {
	rtm.mutex.Lock()
	defer rtm.mutex.Unlock()
	
	rtm.collectors = append(rtm.collectors, collector)
}

// AddAlertHandler adds an alert handler
func (rtm *RealTimeMonitor) AddAlertHandler(handler AlertHandler) {
	rtm.mutex.Lock()
	defer rtm.mutex.Unlock()
	
	rtm.alertHandlers = append(rtm.alertHandlers, handler)
}

// SetThreshold sets an alert threshold
func (rtm *RealTimeMonitor) SetThreshold(metric string, threshold float64) {
	rtm.mutex.Lock()
	defer rtm.mutex.Unlock()
	
	rtm.thresholds[metric] = threshold
}

// GetMetrics returns current metrics
func (rtm *RealTimeMonitor) GetMetrics() *MonitoringMetrics {
	rtm.metrics.mutex.RLock()
	defer rtm.metrics.mutex.RUnlock()
	
	// Create a deep copy
	metrics := &MonitoringMetrics{
		Timestamp:         rtm.metrics.Timestamp,
		CPUUsage:          rtm.metrics.CPUUsage,
		MemoryUsage:       rtm.metrics.MemoryUsage,
		MemoryTotal:       rtm.metrics.MemoryTotal,
		GoroutineCount:    rtm.metrics.GoroutineCount,
		GCPauseTime:       rtm.metrics.GCPauseTime,
		RequestCount:      rtm.metrics.RequestCount,
		ErrorRate:         rtm.metrics.ErrorRate,
		Throughput:        rtm.metrics.Throughput,
		ActiveRequests:    rtm.metrics.ActiveRequests,
		ChunksGenerated:   rtm.metrics.ChunksGenerated,
		AverageChunkSize:  rtm.metrics.AverageChunkSize,
		QualityScore:      rtm.metrics.QualityScore,
		ProcessingLatency: rtm.metrics.ProcessingLatency,
		CacheHitRate:      rtm.metrics.CacheHitRate,
		CacheSize:         rtm.metrics.CacheSize,
		CacheEvictions:    rtm.metrics.CacheEvictions,
		StrategyUsage:     make(map[string]int64),
		StrategyErrors:    make(map[string]int64),
		StrategyLatency:   make(map[string]time.Duration),
	}
	
	// Copy maps
	for k, v := range rtm.metrics.StrategyUsage {
		metrics.StrategyUsage[k] = v
	}
	for k, v := range rtm.metrics.StrategyErrors {
		metrics.StrategyErrors[k] = v
	}
	for k, v := range rtm.metrics.StrategyLatency {
		metrics.StrategyLatency[k] = v
	}
	
	return metrics
}

// SendAlert sends an alert through the monitoring system
func (rtm *RealTimeMonitor) SendAlert(alert Alert) {
	if !rtm.config.EnableAlerts {
		return
	}
	
	select {
	case rtm.alerts <- alert:
	default:
		// Alert buffer full, could log this
	}
}

// GetHealthStatus returns current health status
func (rtm *RealTimeMonitor) GetHealthStatus() *MonitoringHealthStatus {
	checks := make(map[string]CheckResult)
	overallScore := 0.0
	checkCount := 0
	
	// Memory check
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryUsage := float64(m.Alloc) / float64(1<<30) // GB
	
	memoryStatus := "healthy"
	if memoryUsage > 0.8 {
		memoryStatus = "warning"
	}
	if memoryUsage > 0.9 {
		memoryStatus = "critical"
	}
	
	checks["memory"] = CheckResult{
		Status:   memoryStatus,
		Message:  fmt.Sprintf("Memory usage: %.2f GB", memoryUsage),
		Duration: time.Millisecond,
		Metadata: map[string]interface{}{"usage_gb": memoryUsage},
	}
	overallScore += rtm.getCheckScore(memoryStatus)
	checkCount++
	
	// Goroutine check
	goroutineCount := runtime.NumGoroutine()
	goroutineStatus := "healthy"
	if goroutineCount > 1000 {
		goroutineStatus = "warning"
	}
	if goroutineCount > 10000 {
		goroutineStatus = "critical"
	}
	
	checks["goroutines"] = CheckResult{
		Status:   goroutineStatus,
		Message:  fmt.Sprintf("Goroutine count: %d", goroutineCount),
		Duration: time.Millisecond,
		Metadata: map[string]interface{}{"count": goroutineCount},
	}
	overallScore += rtm.getCheckScore(goroutineStatus)
	checkCount++
	
	// Error rate check
	metrics := rtm.GetMetrics()
	errorStatus := "healthy"
	if metrics.ErrorRate > 0.05 {
		errorStatus = "warning"
	}
	if metrics.ErrorRate > 0.1 {
		errorStatus = "critical"
	}
	
	checks["error_rate"] = CheckResult{
		Status:   errorStatus,
		Message:  fmt.Sprintf("Error rate: %.2f%%", metrics.ErrorRate*100),
		Duration: time.Millisecond,
		Metadata: map[string]interface{}{"rate": metrics.ErrorRate},
	}
	overallScore += rtm.getCheckScore(errorStatus)
	checkCount++
	
	// Calculate overall status
	if checkCount > 0 {
		overallScore /= float64(checkCount)
	}
	
	status := "healthy"
	if overallScore < 0.8 {
		status = "degraded"
	}
	if overallScore < 0.5 {
		status = "unhealthy"
	}
	
	return &MonitoringHealthStatus{
		Status:    status,
		Score:     overallScore,
		Checks:    checks,
		Timestamp: time.Now(),
		Uptime:    time.Since(time.Now()), // This would be tracked from start time
		Version:   "1.0.0",
	}
}

// RecordRequest records a request metric
func (rtm *RealTimeMonitor) RecordRequest(duration time.Duration, success bool) {
	rtm.metrics.mutex.Lock()
	defer rtm.metrics.mutex.Unlock()
	
	atomic.AddInt64(&rtm.metrics.RequestCount, 1)
	rtm.metrics.ResponseTimes = append(rtm.metrics.ResponseTimes, duration)
	
	// Keep only last 1000 response times
	if len(rtm.metrics.ResponseTimes) > 1000 {
		rtm.metrics.ResponseTimes = rtm.metrics.ResponseTimes[len(rtm.metrics.ResponseTimes)-1000:]
	}
	
	if !success {
		// Update error rate (exponential moving average)
		alpha := 0.1
		errorIncrement := 1.0
		rtm.metrics.ErrorRate = alpha*errorIncrement + (1-alpha)*rtm.metrics.ErrorRate
	} else {
		// Decay error rate
		alpha := 0.1
		rtm.metrics.ErrorRate = (1 - alpha) * rtm.metrics.ErrorRate
	}
}

// RecordChunkingMetrics records chunking-specific metrics
func (rtm *RealTimeMonitor) RecordChunkingMetrics(chunkCount int, averageSize float64, quality float64, strategy string) {
	rtm.metrics.mutex.Lock()
	defer rtm.metrics.mutex.Unlock()
	
	atomic.AddInt64(&rtm.metrics.ChunksGenerated, int64(chunkCount))
	rtm.metrics.AverageChunkSize = averageSize
	rtm.metrics.QualityScore = quality
	
	if rtm.metrics.StrategyUsage == nil {
		rtm.metrics.StrategyUsage = make(map[string]int64)
	}
	rtm.metrics.StrategyUsage[strategy]++
}

// Internal methods

func (rtm *RealTimeMonitor) runMetricCollection() {
	ticker := time.NewTicker(rtm.config.SampleInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			rtm.collectMetrics()
		case <-rtm.stopChan:
			return
		}
	}
}

func (rtm *RealTimeMonitor) collectMetrics() {
	rtm.metrics.mutex.Lock()
	defer rtm.metrics.mutex.Unlock()
	
	rtm.metrics.Timestamp = time.Now()
	
	// Collect from all registered collectors
	for _, collector := range rtm.collectors {
		metrics, err := collector.Collect()
		if err != nil {
			continue // Skip failed collections
		}
		
		// Merge metrics
		rtm.mergeMetrics(metrics)
	}
	
	// Store metrics if persistence is enabled
	if rtm.config.EnablePersistence {
		rtm.storage.Store(rtm.metrics)
	}
	
	// Check thresholds and generate alerts
	rtm.checkThresholds()
}

func (rtm *RealTimeMonitor) mergeMetrics(metrics map[string]interface{}) {
	for key, value := range metrics {
		switch key {
		case "cpu_usage":
			if v, ok := value.(float64); ok {
				rtm.metrics.CPUUsage = v
			}
		case "memory_usage":
			if v, ok := value.(int64); ok {
				rtm.metrics.MemoryUsage = v
			}
		case "goroutine_count":
			if v, ok := value.(int); ok {
				rtm.metrics.GoroutineCount = v
			}
		// Add more metric mappings as needed
		}
	}
}

func (rtm *RealTimeMonitor) checkThresholds() {
	metrics := rtm.metrics
	
	// Check memory threshold
	if threshold, exists := rtm.thresholds["memory_usage"]; exists {
		if float64(metrics.MemoryUsage) > threshold {
			rtm.SendAlert(Alert{
				ID:        fmt.Sprintf("memory_%d", time.Now().Unix()),
				Type:      "memory",
				Severity:  AlertSeverityWarning,
				Title:     "High Memory Usage",
				Message:   fmt.Sprintf("Memory usage %.2f GB exceeds threshold %.2f GB", float64(metrics.MemoryUsage)/float64(1<<30), threshold/float64(1<<30)),
				Timestamp: time.Now(),
				Source:    "monitoring",
				Metrics: map[string]interface{}{
					"current": metrics.MemoryUsage,
					"threshold": threshold,
				},
			})
		}
	}
	
	// Check error rate threshold
	if threshold, exists := rtm.thresholds["error_rate"]; exists {
		if metrics.ErrorRate > threshold {
			rtm.SendAlert(Alert{
				ID:        fmt.Sprintf("error_rate_%d", time.Now().Unix()),
				Type:      "error_rate",
				Severity:  AlertSeverityError,
				Title:     "High Error Rate",
				Message:   fmt.Sprintf("Error rate %.2f%% exceeds threshold %.2f%%", metrics.ErrorRate*100, threshold*100),
				Timestamp: time.Now(),
				Source:    "monitoring",
				Metrics: map[string]interface{}{
					"current": metrics.ErrorRate,
					"threshold": threshold,
				},
			})
		}
	}
}

func (rtm *RealTimeMonitor) runAlertProcessing() {
	for {
		select {
		case alert := <-rtm.alerts:
			rtm.processAlert(alert)
		case <-rtm.stopChan:
			return
		}
	}
}

func (rtm *RealTimeMonitor) processAlert(alert Alert) {
	for _, handler := range rtm.alertHandlers {
		if handler.CanHandle(alert.Type) {
			handler.HandleAlert(alert)
		}
	}
}

func (rtm *RealTimeMonitor) runStorageCleanup() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			cutoff := time.Now().Add(-rtm.config.RetentionPeriod)
			rtm.storage.Cleanup(cutoff)
		case <-rtm.stopChan:
			return
		}
	}
}

func (rtm *RealTimeMonitor) getCheckScore(status string) float64 {
	switch status {
	case "healthy":
		return 1.0
	case "warning":
		return 0.7
	case "critical":
		return 0.3
	default:
		return 0.0
	}
}

// Metric Collectors

// SystemMetricCollector collects system-level metrics
type SystemMetricCollector struct{}

func (smc *SystemMetricCollector) Name() string {
	return "system"
}

func (smc *SystemMetricCollector) Interval() time.Duration {
	return 10 * time.Second
}

func (smc *SystemMetricCollector) Collect() (map[string]interface{}, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return map[string]interface{}{
		"memory_usage":    int64(m.Alloc),
		"memory_total":    int64(m.Sys),
		"goroutine_count": runtime.NumGoroutine(),
		"gc_pause_time":   time.Duration(m.PauseNs[(m.NumGC+255)%256]),
	}, nil
}

// RuntimeMetricCollector collects Go runtime metrics
type RuntimeMetricCollector struct{}

func (rmc *RuntimeMetricCollector) Name() string {
	return "runtime"
}

func (rmc *RuntimeMetricCollector) Interval() time.Duration {
	return 10 * time.Second
}

func (rmc *RuntimeMetricCollector) Collect() (map[string]interface{}, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return map[string]interface{}{
		"heap_alloc":     m.HeapAlloc,
		"heap_sys":       m.HeapSys,
		"heap_inuse":     m.HeapInuse,
		"heap_released":  m.HeapReleased,
		"gc_num":         m.NumGC,
		"gc_cpu_percent": m.GCCPUFraction,
	}, nil
}

// ChunkingMetricCollector collects chunking-specific metrics
type ChunkingMetricCollector struct{}

func (cmc *ChunkingMetricCollector) Name() string {
	return "chunking"
}

func (cmc *ChunkingMetricCollector) Interval() time.Duration {
	return 30 * time.Second
}

func (cmc *ChunkingMetricCollector) Collect() (map[string]interface{}, error) {
	// This would collect metrics from the chunking pipeline
	return map[string]interface{}{
		"chunks_processed": 0, // Would be tracked elsewhere
		"average_quality":  0.85,
		"processing_time":  time.Millisecond * 100,
	}, nil
}

// Alert Handlers

// LogAlertHandler logs alerts to stdout/stderr
type LogAlertHandler struct{}

func (lah *LogAlertHandler) CanHandle(alertType string) bool {
	return true // Handle all alert types
}

func (lah *LogAlertHandler) HandleAlert(alert Alert) error {
	alertJSON, _ := json.MarshalIndent(alert, "", "  ")
	fmt.Printf("ALERT: %s\n", string(alertJSON))
	return nil
}

// InMemoryMetricStorage provides in-memory metric storage
type InMemoryMetricStorage struct {
	metrics []MonitoringMetrics
	mutex   sync.RWMutex
}

func (imms *InMemoryMetricStorage) Store(metrics *MonitoringMetrics) error {
	imms.mutex.Lock()
	defer imms.mutex.Unlock()
	
	imms.metrics = append(imms.metrics, *metrics)
	return nil
}

func (imms *InMemoryMetricStorage) Query(start, end time.Time, filters map[string]string) ([]*MonitoringMetrics, error) {
	imms.mutex.RLock()
	defer imms.mutex.RUnlock()
	
	var result []*MonitoringMetrics
	for _, metric := range imms.metrics {
		if metric.Timestamp.After(start) && metric.Timestamp.Before(end) {
			result = append(result, &metric)
		}
	}
	
	return result, nil
}

func (imms *InMemoryMetricStorage) Cleanup(before time.Time) error {
	imms.mutex.Lock()
	defer imms.mutex.Unlock()
	
	var filtered []MonitoringMetrics
	for _, metric := range imms.metrics {
		if metric.Timestamp.After(before) {
			filtered = append(filtered, metric)
		}
	}
	
	imms.metrics = filtered
	return nil
}