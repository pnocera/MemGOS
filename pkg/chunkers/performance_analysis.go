package chunkers

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"
)

// PerformanceAnalyzer provides comprehensive performance analysis for chunking systems
type PerformanceAnalyzer struct {
	profiles        map[string]*PerformanceProfile
	benchmarks      map[string]*BenchmarkSuite
	regressionTests []*RegressionTest
	config          *AnalysisConfig
	mutex           sync.RWMutex
}

// AnalysisConfig configures performance analysis
type AnalysisConfig struct {
	// Profiling configuration
	EnableCPUProfiling    bool          `json:"enable_cpu_profiling"`
	EnableMemoryProfiling bool          `json:"enable_memory_profiling"`
	ProfilingDuration     time.Duration `json:"profiling_duration"`

	// Benchmark configuration
	BenchmarkIterations int   `json:"benchmark_iterations"`
	WarmupIterations    int   `json:"warmup_iterations"`
	ConcurrencyLevels   []int `json:"concurrency_levels"`

	// Regression testing
	BaselineThreshold   float64 `json:"baseline_threshold"`
	RegressionThreshold float64 `json:"regression_threshold"`

	// Analysis parameters
	PercentilesToTrack  []float64 `json:"percentiles_to_track"`
	EnableTrendAnalysis bool      `json:"enable_trend_analysis"`
	TrendWindow         int       `json:"trend_window"`
}

// DefaultAnalysisConfig returns default analysis configuration
func DefaultAnalysisConfig() *AnalysisConfig {
	return &AnalysisConfig{
		EnableCPUProfiling:    true,
		EnableMemoryProfiling: true,
		ProfilingDuration:     30 * time.Second,
		BenchmarkIterations:   1000,
		WarmupIterations:      100,
		ConcurrencyLevels:     []int{1, 2, 4, 8, 16},
		BaselineThreshold:     0.05, // 5%
		RegressionThreshold:   0.10, // 10%
		PercentilesToTrack:    []float64{50, 90, 95, 99},
		EnableTrendAnalysis:   true,
		TrendWindow:           20,
	}
}

// PerformanceProfile contains detailed performance profiling data
type PerformanceProfile struct {
	Name               string             `json:"name"`
	Timestamp          time.Time          `json:"timestamp"`
	Duration           time.Duration      `json:"duration"`
	CPUUsage           CPUProfile         `json:"cpu_usage"`
	MemoryUsage        MemoryProfile      `json:"memory_usage"`
	GCStats            GCProfile          `json:"gc_stats"`
	ThroughputStats    ThroughputProfile  `json:"throughput_stats"`
	LatencyStats       LatencyProfile     `json:"latency_stats"`
	ConcurrencyStats   ConcurrencyProfile `json:"concurrency_stats"`
	ResourceUsage      ResourceProfile    `json:"resource_usage"`
	BottleneckAnalysis BottleneckAnalysis `json:"bottleneck_analysis"`
	Recommendations    []string           `json:"recommendations"`
}

// CPUProfile contains CPU usage analysis
type CPUProfile struct {
	AverageUsage float64       `json:"average_usage"`
	PeakUsage    float64       `json:"peak_usage"`
	UserTime     time.Duration `json:"user_time"`
	SystemTime   time.Duration `json:"system_time"`
	IdleTime     time.Duration `json:"idle_time"`
	Hotspots     []Hotspot     `json:"hotspots"`
}

// MemoryProfile contains memory usage analysis
type MemoryProfile struct {
	HeapSize    int64             `json:"heap_size"`
	HeapUsed    int64             `json:"heap_used"`
	HeapObjects int64             `json:"heap_objects"`
	StackSize   int64             `json:"stack_size"`
	PeakMemory  int64             `json:"peak_memory"`
	Allocations AllocationProfile `json:"allocations"`
	Leaks       []MemoryLeak      `json:"leaks"`
}

// GCProfile contains garbage collection statistics
type GCProfile struct {
	GCCount        int64         `json:"gc_count"`
	TotalPauseTime time.Duration `json:"total_pause_time"`
	AveragePause   time.Duration `json:"average_pause"`
	MaxPause       time.Duration `json:"max_pause"`
	GCCPUPercent   float64       `json:"gc_cpu_percent"`
	HeapGrowth     int64         `json:"heap_growth"`
}

// ThroughputProfile contains throughput analysis
type ThroughputProfile struct {
	RequestsPerSecond   float64 `json:"requests_per_second"`
	ChunksPerSecond     float64 `json:"chunks_per_second"`
	BytesPerSecond      float64 `json:"bytes_per_second"`
	PeakThroughput      float64 `json:"peak_throughput"`
	SustainedThroughput float64 `json:"sustained_throughput"`
	ThroughputVariance  float64 `json:"throughput_variance"`
}

// LatencyProfile contains latency analysis
type LatencyProfile struct {
	Mean         time.Duration            `json:"mean"`
	Median       time.Duration            `json:"median"`
	Percentiles  map[string]time.Duration `json:"percentiles"`
	StandardDev  time.Duration            `json:"standard_deviation"`
	Min          time.Duration            `json:"min"`
	Max          time.Duration            `json:"max"`
	Distribution []LatencyBucket          `json:"distribution"`
}

// ConcurrencyProfile contains concurrency analysis
type ConcurrencyProfile struct {
	MaxConcurrency     int               `json:"max_concurrency"`
	AverageConcurrency float64           `json:"average_concurrency"`
	GoroutineCount     int               `json:"goroutine_count"`
	ContentionEvents   []ContentionEvent `json:"contention_events"`
	DeadlockRisk       float64           `json:"deadlock_risk"`
	ScalabilityFactor  float64           `json:"scalability_factor"`
}

// ResourceProfile contains system resource usage
type ResourceProfile struct {
	FileDescriptors    int64            `json:"file_descriptors"`
	NetworkConnections int64            `json:"network_connections"`
	DiskIO             DiskIOProfile    `json:"disk_io"`
	NetworkIO          NetworkIOProfile `json:"network_io"`
	CPUCores           int              `json:"cpu_cores"`
	MemoryTotal        int64            `json:"memory_total"`
}

// BottleneckAnalysis identifies performance bottlenecks
type BottleneckAnalysis struct {
	PrimaryBottleneck     string               `json:"primary_bottleneck"`
	BottleneckSeverity    float64              `json:"bottleneck_severity"`
	BottleneckLocation    string               `json:"bottleneck_location"`
	ImpactAssessment      string               `json:"impact_assessment"`
	OptimizationPotential float64              `json:"optimization_potential"`
	RecommendedActions    []OptimizationAction `json:"recommended_actions"`
}

// Supporting types
type Hotspot struct {
	Function  string  `json:"function"`
	CPUTime   float64 `json:"cpu_time"`
	CallCount int64   `json:"call_count"`
}

type AllocationProfile struct {
	AllocationsPerSec    float64             `json:"allocations_per_sec"`
	BytesAllocatedPerSec int64               `json:"bytes_allocated_per_sec"`
	ObjectsAllocated     int64               `json:"objects_allocated"`
	AllocationHotspots   []AllocationHotspot `json:"allocation_hotspots"`
}

type MemoryLeak struct {
	Location   string  `json:"location"`
	LeakRate   float64 `json:"leak_rate"`
	Size       int64   `json:"size"`
	Confidence float64 `json:"confidence"`
}

type LatencyBucket struct {
	Range      string  `json:"range"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

type ContentionEvent struct {
	Location  string        `json:"location"`
	Duration  time.Duration `json:"duration"`
	Frequency int64         `json:"frequency"`
}

type DiskIOProfile struct {
	ReadBytes  int64 `json:"read_bytes"`
	WriteBytes int64 `json:"write_bytes"`
	ReadOps    int64 `json:"read_ops"`
	WriteOps   int64 `json:"write_ops"`
}

type NetworkIOProfile struct {
	BytesSent       int64 `json:"bytes_sent"`
	BytesReceived   int64 `json:"bytes_received"`
	PacketsSent     int64 `json:"packets_sent"`
	PacketsReceived int64 `json:"packets_received"`
}

type AllocationHotspot struct {
	Function    string  `json:"function"`
	Allocations int64   `json:"allocations"`
	Bytes       int64   `json:"bytes"`
	Percentage  float64 `json:"percentage"`
}

type OptimizationAction struct {
	Action        string  `json:"action"`
	Priority      string  `json:"priority"`
	Impact        string  `json:"impact"`
	Difficulty    string  `json:"difficulty"`
	EstimatedGain float64 `json:"estimated_gain"`
}

// BenchmarkSuite contains comprehensive benchmark results
type BenchmarkSuite struct {
	Name          string                      `json:"name"`
	Timestamp     time.Time                   `json:"timestamp"`
	Configuration BenchmarkConfig             `json:"configuration"`
	Results       map[string]*BenchmarkResult `json:"results"`
	Comparison    *BenchmarkComparison        `json:"comparison"`
	TrendAnalysis *TrendAnalysis              `json:"trend_analysis"`
	Summary       BenchmarkSummary            `json:"summary"`
}

// BenchmarkConfig contains benchmark configuration
type BenchmarkConfig struct {
	Iterations        int           `json:"iterations"`
	WarmupIterations  int           `json:"warmup_iterations"`
	ConcurrencyLevels []int         `json:"concurrency_levels"`
	DataSizes         []int         `json:"data_sizes"`
	Strategies        []string      `json:"strategies"`
	TestDuration      time.Duration `json:"test_duration"`
}

// BenchmarkComparison compares different strategies or configurations
type BenchmarkComparison struct {
	BaselineStrategy string                         `json:"baseline_strategy"`
	Comparisons      map[string]*StrategyComparison `json:"comparisons"`
	WinnerAnalysis   WinnerAnalysis                 `json:"winner_analysis"`
}

// StrategyComparison compares a strategy against baseline
type StrategyComparison struct {
	Strategy         string  `json:"strategy"`
	PerformanceRatio float64 `json:"performance_ratio"`
	ThroughputRatio  float64 `json:"throughput_ratio"`
	LatencyRatio     float64 `json:"latency_ratio"`
	MemoryRatio      float64 `json:"memory_ratio"`
	QualityRatio     float64 `json:"quality_ratio"`
	IsSignificant    bool    `json:"is_significant"`
	PValue           float64 `json:"p_value"`
	Recommendation   string  `json:"recommendation"`
}

// WinnerAnalysis determines the best performing strategy
type WinnerAnalysis struct {
	OverallWinner    string  `json:"overall_winner"`
	ThroughputWinner string  `json:"throughput_winner"`
	LatencyWinner    string  `json:"latency_winner"`
	MemoryWinner     string  `json:"memory_winner"`
	QualityWinner    string  `json:"quality_winner"`
	Confidence       float64 `json:"confidence"`
	TradeoffAnalysis string  `json:"tradeoff_analysis"`
}

// TrendAnalysis analyzes performance trends over time
type TrendAnalysis struct {
	ThroughputTrend  TrendData `json:"throughput_trend"`
	LatencyTrend     TrendData `json:"latency_trend"`
	MemoryTrend      TrendData `json:"memory_trend"`
	QualityTrend     TrendData `json:"quality_trend"`
	OverallTrend     string    `json:"overall_trend"`
	PredictedOutlook string    `json:"predicted_outlook"`
}

// TrendData contains trend analysis for a specific metric
type TrendData struct {
	Direction   string  `json:"direction"`
	Slope       float64 `json:"slope"`
	Correlation float64 `json:"correlation"`
	Confidence  float64 `json:"confidence"`
	Prediction  string  `json:"prediction"`
}

// RegressionTest detects performance regressions
type RegressionTest struct {
	Name           string           `json:"name"`
	BaselineData   *PerformanceData `json:"baseline_data"`
	CurrentData    *PerformanceData `json:"current_data"`
	RegressionType string           `json:"regression_type"`
	Severity       string           `json:"severity"`
	Impact         float64          `json:"impact"`
	DetectedAt     time.Time        `json:"detected_at"`
	Details        string           `json:"details"`
}

// PerformanceData contains key performance metrics
type PerformanceData struct {
	Throughput  float64       `json:"throughput"`
	Latency     time.Duration `json:"latency"`
	MemoryUsage int64         `json:"memory_usage"`
	CPUUsage    float64       `json:"cpu_usage"`
	Quality     float64       `json:"quality"`
}

// NewPerformanceAnalyzer creates a new performance analyzer
func NewPerformanceAnalyzer(config *AnalysisConfig) *PerformanceAnalyzer {
	if config == nil {
		config = DefaultAnalysisConfig()
	}

	return &PerformanceAnalyzer{
		profiles:        make(map[string]*PerformanceProfile),
		benchmarks:      make(map[string]*BenchmarkSuite),
		regressionTests: make([]*RegressionTest, 0),
		config:          config,
	}
}

// ProfilePerformance performs comprehensive performance profiling
func (pa *PerformanceAnalyzer) ProfilePerformance(ctx context.Context, name string, pipeline *ChunkingPipeline, testData []string) (*PerformanceProfile, error) {
	profile := &PerformanceProfile{
		Name:      name,
		Timestamp: time.Now(),
	}

	// Start profiling
	start := time.Now()

	// Collect initial metrics
	var initialMem runtime.MemStats
	runtime.ReadMemStats(&initialMem)
	initialGC := initialMem.NumGC

	// Run performance test
	throughputStats, latencyStats, err := pa.runPerformanceTest(ctx, pipeline, testData)
	if err != nil {
		return nil, fmt.Errorf("performance test failed: %w", err)
	}

	// Collect final metrics
	var finalMem runtime.MemStats
	runtime.ReadMemStats(&finalMem)
	finalGC := finalMem.NumGC

	profile.Duration = time.Since(start)

	// Build CPU profile
	profile.CPUUsage = pa.buildCPUProfile()

	// Build memory profile
	profile.MemoryUsage = pa.buildMemoryProfile(&initialMem, &finalMem)

	// Build GC profile
	profile.GCStats = pa.buildGCProfile(&initialMem, &finalMem, initialGC, finalGC)

	// Set throughput and latency stats
	profile.ThroughputStats = *throughputStats
	profile.LatencyStats = *latencyStats

	// Build concurrency profile
	profile.ConcurrencyStats = pa.buildConcurrencyProfile()

	// Build resource profile
	profile.ResourceUsage = pa.buildResourceProfile()

	// Perform bottleneck analysis
	profile.BottleneckAnalysis = pa.analyzeBottlenecks(profile)

	// Generate recommendations
	profile.Recommendations = pa.generateRecommendations(profile)

	// Store profile
	pa.mutex.Lock()
	pa.profiles[name] = profile
	pa.mutex.Unlock()

	return profile, nil
}

// RunBenchmarkSuite runs a comprehensive benchmark suite
func (pa *PerformanceAnalyzer) RunBenchmarkSuite(ctx context.Context, name string, pipeline *ChunkingPipeline, strategies []string, testData []string) (*BenchmarkSuite, error) {
	suite := &BenchmarkSuite{
		Name:      name,
		Timestamp: time.Now(),
		Configuration: BenchmarkConfig{
			Iterations:        pa.config.BenchmarkIterations,
			WarmupIterations:  pa.config.WarmupIterations,
			ConcurrencyLevels: pa.config.ConcurrencyLevels,
			Strategies:        strategies,
			TestDuration:      pa.config.ProfilingDuration,
		},
		Results: make(map[string]*BenchmarkResult),
	}

	// Run benchmarks for each strategy
	for _, strategy := range strategies {
		result, err := pa.runStrategyBenchmark(ctx, strategy, pipeline, testData)
		if err != nil {
			return nil, fmt.Errorf("benchmark for strategy %s failed: %w", strategy, err)
		}
		suite.Results[strategy] = result
	}

	// Perform comparison analysis
	if len(strategies) > 1 {
		suite.Comparison = pa.performBenchmarkComparison(suite.Results, strategies[0])
	}

	// Perform trend analysis if enabled
	if pa.config.EnableTrendAnalysis {
		suite.TrendAnalysis = pa.performTrendAnalysis(name, suite.Results)
	}

	// Generate summary
	suite.Summary = pa.generateBenchmarkSummary(suite.Results)

	// Store benchmark suite
	pa.mutex.Lock()
	pa.benchmarks[name] = suite
	pa.mutex.Unlock()

	return suite, nil
}

// DetectRegressions detects performance regressions compared to baseline
func (pa *PerformanceAnalyzer) DetectRegressions(baseline, current *PerformanceProfile) []*RegressionTest {
	var regressions []*RegressionTest

	// Check throughput regression
	throughputChange := (current.ThroughputStats.RequestsPerSecond - baseline.ThroughputStats.RequestsPerSecond) / baseline.ThroughputStats.RequestsPerSecond
	if throughputChange < -pa.config.RegressionThreshold {
		regressions = append(regressions, &RegressionTest{
			Name:           "Throughput Regression",
			BaselineData:   pa.extractPerformanceData(baseline),
			CurrentData:    pa.extractPerformanceData(current),
			RegressionType: "throughput",
			Severity:       pa.calculateSeverity(throughputChange),
			Impact:         math.Abs(throughputChange),
			DetectedAt:     time.Now(),
			Details:        fmt.Sprintf("Throughput decreased by %.2f%%", math.Abs(throughputChange)*100),
		})
	}

	// Check latency regression
	latencyChange := float64(current.LatencyStats.Mean-baseline.LatencyStats.Mean) / float64(baseline.LatencyStats.Mean)
	if latencyChange > pa.config.RegressionThreshold {
		regressions = append(regressions, &RegressionTest{
			Name:           "Latency Regression",
			BaselineData:   pa.extractPerformanceData(baseline),
			CurrentData:    pa.extractPerformanceData(current),
			RegressionType: "latency",
			Severity:       pa.calculateSeverity(latencyChange),
			Impact:         latencyChange,
			DetectedAt:     time.Now(),
			Details:        fmt.Sprintf("Latency increased by %.2f%%", latencyChange*100),
		})
	}

	// Check memory regression
	memoryChange := float64(current.MemoryUsage.HeapUsed-baseline.MemoryUsage.HeapUsed) / float64(baseline.MemoryUsage.HeapUsed)
	if memoryChange > pa.config.RegressionThreshold {
		regressions = append(regressions, &RegressionTest{
			Name:           "Memory Regression",
			BaselineData:   pa.extractPerformanceData(baseline),
			CurrentData:    pa.extractPerformanceData(current),
			RegressionType: "memory",
			Severity:       pa.calculateSeverity(memoryChange),
			Impact:         memoryChange,
			DetectedAt:     time.Now(),
			Details:        fmt.Sprintf("Memory usage increased by %.2f%%", memoryChange*100),
		})
	}

	// Store regressions
	pa.mutex.Lock()
	pa.regressionTests = append(pa.regressionTests, regressions...)
	pa.mutex.Unlock()

	return regressions
}

// GenerateReport generates a comprehensive performance report
func (pa *PerformanceAnalyzer) GenerateReport() (*PerformanceReport, error) {
	pa.mutex.RLock()
	defer pa.mutex.RUnlock()

	report := &PerformanceReport{
		GeneratedAt:     time.Now(),
		ProfileCount:    len(pa.profiles),
		BenchmarkCount:  len(pa.benchmarks),
		RegressionCount: len(pa.regressionTests),
		Profiles:        make(map[string]*PerformanceProfile),
		Benchmarks:      make(map[string]*BenchmarkSuite),
		Regressions:     make([]*RegressionTest, len(pa.regressionTests)),
	}

	// Copy profiles
	for name, profile := range pa.profiles {
		report.Profiles[name] = profile
	}

	// Copy benchmarks
	for name, benchmark := range pa.benchmarks {
		report.Benchmarks[name] = benchmark
	}

	// Copy regressions
	copy(report.Regressions, pa.regressionTests)

	// Generate overall assessment
	report.OverallAssessment = pa.generateOverallAssessment(report)

	// Generate recommendations
	report.KeyRecommendations = pa.generateKeyRecommendations(report)

	return report, nil
}

// PerformanceReport contains comprehensive performance analysis
type PerformanceReport struct {
	GeneratedAt        time.Time                      `json:"generated_at"`
	ProfileCount       int                            `json:"profile_count"`
	BenchmarkCount     int                            `json:"benchmark_count"`
	RegressionCount    int                            `json:"regression_count"`
	Profiles           map[string]*PerformanceProfile `json:"profiles"`
	Benchmarks         map[string]*BenchmarkSuite     `json:"benchmarks"`
	Regressions        []*RegressionTest              `json:"regressions"`
	OverallAssessment  string                         `json:"overall_assessment"`
	KeyRecommendations []string                       `json:"key_recommendations"`
}

// Helper methods (implementations would be comprehensive but are simplified here for brevity)

func (pa *PerformanceAnalyzer) runPerformanceTest(ctx context.Context, pipeline *ChunkingPipeline, testData []string) (*ThroughputProfile, *LatencyProfile, error) {
	// Implementation would run actual performance tests
	// For now, return placeholder data

	throughput := &ThroughputProfile{
		RequestsPerSecond: 100.0,
		ChunksPerSecond:   1000.0,
		BytesPerSecond:    50000.0,
	}

	latency := &LatencyProfile{
		Mean:   10 * time.Millisecond,
		Median: 8 * time.Millisecond,
		Percentiles: map[string]time.Duration{
			"p50": 8 * time.Millisecond,
			"p90": 15 * time.Millisecond,
			"p95": 20 * time.Millisecond,
			"p99": 30 * time.Millisecond,
		},
	}

	return throughput, latency, nil
}

func (pa *PerformanceAnalyzer) buildCPUProfile() CPUProfile {
	// Implementation would collect actual CPU profiling data
	return CPUProfile{
		AverageUsage: 25.5,
		PeakUsage:    45.2,
		UserTime:     5 * time.Second,
		SystemTime:   1 * time.Second,
	}
}

func (pa *PerformanceAnalyzer) buildMemoryProfile(initial, final *runtime.MemStats) MemoryProfile {
	return MemoryProfile{
		HeapSize:    int64(final.HeapSys),
		HeapUsed:    int64(final.HeapInuse),
		HeapObjects: int64(final.HeapObjects),
		PeakMemory:  int64(final.Alloc),
	}
}

func (pa *PerformanceAnalyzer) buildGCProfile(initial, final *runtime.MemStats, initialGC, finalGC uint32) GCProfile {
	gcCount := int64(finalGC - initialGC)

	return GCProfile{
		GCCount:        gcCount,
		TotalPauseTime: time.Duration(final.PauseTotalNs - initial.PauseTotalNs),
		GCCPUPercent:   final.GCCPUFraction,
	}
}

func (pa *PerformanceAnalyzer) buildConcurrencyProfile() ConcurrencyProfile {
	return ConcurrencyProfile{
		MaxConcurrency:     pa.config.ConcurrencyLevels[len(pa.config.ConcurrencyLevels)-1],
		AverageConcurrency: 4.0,
		GoroutineCount:     runtime.NumGoroutine(),
	}
}

func (pa *PerformanceAnalyzer) buildResourceProfile() ResourceProfile {
	return ResourceProfile{
		CPUCores:    runtime.NumCPU(),
		MemoryTotal: 8 * 1024 * 1024 * 1024, // 8GB placeholder
	}
}

func (pa *PerformanceAnalyzer) analyzeBottlenecks(profile *PerformanceProfile) BottleneckAnalysis {
	// Simplified bottleneck analysis
	analysis := BottleneckAnalysis{
		PrimaryBottleneck:     "memory",
		BottleneckSeverity:    0.3,
		ImpactAssessment:      "Moderate impact on overall performance",
		OptimizationPotential: 0.25,
	}

	// Add optimization actions
	analysis.RecommendedActions = []OptimizationAction{
		{
			Action:        "Optimize memory allocation patterns",
			Priority:      "high",
			Impact:        "significant",
			Difficulty:    "medium",
			EstimatedGain: 0.20,
		},
		{
			Action:        "Implement object pooling",
			Priority:      "medium",
			Impact:        "moderate",
			Difficulty:    "low",
			EstimatedGain: 0.10,
		},
	}

	return analysis
}

func (pa *PerformanceAnalyzer) generateRecommendations(profile *PerformanceProfile) []string {
	recommendations := []string{
		"Consider implementing object pooling to reduce GC pressure",
		"Profile memory allocations to identify optimization opportunities",
		"Monitor goroutine count to prevent resource leaks",
	}

	return recommendations
}

func (pa *PerformanceAnalyzer) runStrategyBenchmark(ctx context.Context, strategy string, pipeline *ChunkingPipeline, testData []string) (*BenchmarkResult, error) {
	// Implementation would run actual benchmarks
	return &BenchmarkResult{
		RunID:          fmt.Sprintf("%s-%d", strategy, time.Now().Unix()),
		Timestamp:      time.Now(),
		ProcessingTime: 10 * time.Millisecond,
		QualityScore:   0.85,
		ChunkCount:     10,
	}, nil
}

func (pa *PerformanceAnalyzer) performBenchmarkComparison(results map[string]*BenchmarkResult, baseline string) *BenchmarkComparison {
	comparison := &BenchmarkComparison{
		BaselineStrategy: baseline,
		Comparisons:      make(map[string]*StrategyComparison),
	}

	baseResult := results[baseline]
	if baseResult == nil {
		return comparison
	}

	for strategy, result := range results {
		if strategy == baseline {
			continue
		}

		comparison.Comparisons[strategy] = &StrategyComparison{
			Strategy:         strategy,
			PerformanceRatio: float64(baseResult.ProcessingTime) / float64(result.ProcessingTime),
			QualityRatio:     result.QualityScore / baseResult.QualityScore,
			IsSignificant:    true,
			PValue:           0.01,
			Recommendation:   "Consider for production use",
		}
	}

	return comparison
}

func (pa *PerformanceAnalyzer) performTrendAnalysis(name string, results map[string]*BenchmarkResult) *TrendAnalysis {
	// Simplified trend analysis
	return &TrendAnalysis{
		ThroughputTrend: TrendData{
			Direction:  "stable",
			Confidence: 0.85,
			Prediction: "Performance will remain stable",
		},
		OverallTrend:     "stable",
		PredictedOutlook: "No significant changes expected",
	}
}

func (pa *PerformanceAnalyzer) generateBenchmarkSummary(results map[string]*BenchmarkResult) BenchmarkSummary {
	return BenchmarkSummary{
		RunCount:         len(results),
		AverageTime:      10 * time.Millisecond,
		ThroughputQPS:    100.0,
		MemoryEfficiency: 0.8,
	}
}

func (pa *PerformanceAnalyzer) extractPerformanceData(profile *PerformanceProfile) *PerformanceData {
	return &PerformanceData{
		Throughput:  profile.ThroughputStats.RequestsPerSecond,
		Latency:     profile.LatencyStats.Mean,
		MemoryUsage: profile.MemoryUsage.HeapUsed,
		CPUUsage:    profile.CPUUsage.AverageUsage,
	}
}

func (pa *PerformanceAnalyzer) calculateSeverity(change float64) string {
	absChange := math.Abs(change)

	if absChange > 0.5 {
		return "critical"
	} else if absChange > 0.2 {
		return "high"
	} else if absChange > 0.1 {
		return "medium"
	}
	return "low"
}

func (pa *PerformanceAnalyzer) generateOverallAssessment(report *PerformanceReport) string {
	if report.RegressionCount > 0 {
		return "Performance regressions detected. Immediate attention required."
	}

	return "Performance is within acceptable parameters. Continue monitoring."
}

func (pa *PerformanceAnalyzer) generateKeyRecommendations(report *PerformanceReport) []string {
	recommendations := []string{
		"Implement continuous performance monitoring",
		"Establish performance baseline for all strategies",
		"Automate regression detection in CI/CD pipeline",
	}

	if report.RegressionCount > 0 {
		recommendations = append(recommendations, "Address identified performance regressions immediately")
	}

	return recommendations
}
