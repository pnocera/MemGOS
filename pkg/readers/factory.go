package readers

import (
	"fmt"
	"strings"

	"github.com/memtensor/memgos/pkg/interfaces"
)

// ReaderType defines the type of memory reader
type ReaderType string

const (
	ReaderTypeSimple   ReaderType = "simple"
	ReaderTypeAdvanced ReaderType = "advanced"
	ReaderTypeSemantic ReaderType = "semantic"
	ReaderTypeHybrid   ReaderType = "hybrid"
)

// ReaderFactory creates memory readers based on configuration
type ReaderFactory struct {
	config   *ReaderConfig
	memCube  interfaces.MemCube
	embedder interfaces.Embedder
	vectorDB interfaces.VectorDB
}

// NewReaderFactory creates a new reader factory
func NewReaderFactory(config *ReaderConfig) *ReaderFactory {
	if config == nil {
		config = DefaultReaderConfig()
	}
	
	return &ReaderFactory{
		config: config,
	}
}

// WithMemCube sets the memory cube for the factory
func (f *ReaderFactory) WithMemCube(memCube interfaces.MemCube) *ReaderFactory {
	f.memCube = memCube
	return f
}

// WithEmbedder sets the embedder for the factory
func (f *ReaderFactory) WithEmbedder(embedder interfaces.Embedder) *ReaderFactory {
	f.embedder = embedder
	return f
}

// WithVectorDB sets the vector database for the factory
func (f *ReaderFactory) WithVectorDB(vectorDB interfaces.VectorDB) *ReaderFactory {
	f.vectorDB = vectorDB
	return f
}

// CreateReader creates a memory reader based on the specified type
func (f *ReaderFactory) CreateReader(readerType ReaderType) (MemReader, error) {
	switch readerType {
	case ReaderTypeSimple:
		return f.createSimpleReader()
	case ReaderTypeAdvanced:
		return f.createAdvancedReader()
	case ReaderTypeSemantic:
		return f.createSemanticReader()
	case ReaderTypeHybrid:
		return f.createHybridReader()
	default:
		return nil, fmt.Errorf("unsupported reader type: %s", readerType)
	}
}

// CreateReaderFromConfig creates a reader based on the factory configuration
func (f *ReaderFactory) CreateReaderFromConfig() (MemReader, error) {
	switch f.config.Strategy {
	case ReadStrategySimple:
		return f.createSimpleReader()
	case ReadStrategyAdvanced:
		return f.createAdvancedReader()
	case ReadStrategySemantic:
		return f.createSemanticReader()
	case ReadStrategyStructural:
		return f.createHybridReader()
	default:
		// Default to advanced reader
		return f.createAdvancedReader()
	}
}

// CreateReaderFromString creates a reader from a string specification
func (f *ReaderFactory) CreateReaderFromString(spec string) (MemReader, error) {
	spec = strings.ToLower(strings.TrimSpace(spec))
	
	switch spec {
	case "simple", "basic":
		return f.createSimpleReader()
	case "advanced", "deep", "full":
		return f.createAdvancedReader()
	case "semantic", "embedding", "vector":
		return f.createSemanticReader()
	case "hybrid", "mixed", "combined":
		return f.createHybridReader()
	default:
		return nil, fmt.Errorf("unsupported reader specification: %s", spec)
	}
}

// GetAvailableReaderTypes returns the available reader types
func (f *ReaderFactory) GetAvailableReaderTypes() []ReaderType {
	types := []ReaderType{ReaderTypeSimple}
	
	// Advanced reader is always available
	types = append(types, ReaderTypeAdvanced)
	
	// Semantic reader requires embedder and vectorDB
	if f.embedder != nil && f.vectorDB != nil {
		types = append(types, ReaderTypeSemantic)
	}
	
	// Hybrid reader requires advanced capabilities
	if f.memCube != nil {
		types = append(types, ReaderTypeHybrid)
	}
	
	return types
}

// ValidateReaderType checks if a reader type is supported
func (f *ReaderFactory) ValidateReaderType(readerType ReaderType) error {
	availableTypes := f.GetAvailableReaderTypes()
	
	for _, availableType := range availableTypes {
		if availableType == readerType {
			return nil
		}
	}
	
	return fmt.Errorf("reader type %s not available, supported types: %v", readerType, availableTypes)
}

// GetReaderCapabilities returns the capabilities of a reader type
func (f *ReaderFactory) GetReaderCapabilities(readerType ReaderType) (*ReaderCapabilities, error) {
	switch readerType {
	case ReaderTypeSimple:
		return &ReaderCapabilities{
			SupportsPatternDetection:   true,
			SupportsQualityAssessment:  true,
			SupportsDuplicateDetection: true,
			SupportsSentimentAnalysis:  true,
			SupportsSummarization:      true,
			SupportsSemanticSearch:     false,
			SupportsAdvancedAnalytics:  false,
			SupportsClustering:         false,
			SupportsAnomalyDetection:   false,
			MaxMemoriesPerQuery:        1000,
			RecommendedMemoryLimit:     100,
			PerformanceLevel:           "fast",
			AccuracyLevel:              "basic",
		}, nil
		
	case ReaderTypeAdvanced:
		return &ReaderCapabilities{
			SupportsPatternDetection:   true,
			SupportsQualityAssessment:  true,
			SupportsDuplicateDetection: true,
			SupportsSentimentAnalysis:  true,
			SupportsSummarization:      true,
			SupportsSemanticSearch:     f.embedder != nil,
			SupportsAdvancedAnalytics:  true,
			SupportsClustering:         true,
			SupportsAnomalyDetection:   true,
			MaxMemoriesPerQuery:        10000,
			RecommendedMemoryLimit:     1000,
			PerformanceLevel:           "medium",
			AccuracyLevel:              "high",
		}, nil
		
	case ReaderTypeSemantic:
		return &ReaderCapabilities{
			SupportsPatternDetection:   true,
			SupportsQualityAssessment:  true,
			SupportsDuplicateDetection: true,
			SupportsSentimentAnalysis:  true,
			SupportsSummarization:      true,
			SupportsSemanticSearch:     true,
			SupportsAdvancedAnalytics:  true,
			SupportsClustering:         true,
			SupportsAnomalyDetection:   true,
			MaxMemoriesPerQuery:        50000,
			RecommendedMemoryLimit:     5000,
			PerformanceLevel:           "slow",
			AccuracyLevel:              "very_high",
		}, nil
		
	case ReaderTypeHybrid:
		return &ReaderCapabilities{
			SupportsPatternDetection:   true,
			SupportsQualityAssessment:  true,
			SupportsDuplicateDetection: true,
			SupportsSentimentAnalysis:  true,
			SupportsSummarization:      true,
			SupportsSemanticSearch:     f.embedder != nil,
			SupportsAdvancedAnalytics:  true,
			SupportsClustering:         true,
			SupportsAnomalyDetection:   true,
			MaxMemoriesPerQuery:        25000,
			RecommendedMemoryLimit:     2500,
			PerformanceLevel:           "medium",
			AccuracyLevel:              "very_high",
		}, nil
		
	default:
		return nil, fmt.Errorf("unknown reader type: %s", readerType)
	}
}

// ReaderCapabilities describes what a reader can do
type ReaderCapabilities struct {
	SupportsPatternDetection   bool   `json:"supports_pattern_detection"`
	SupportsQualityAssessment  bool   `json:"supports_quality_assessment"`
	SupportsDuplicateDetection bool   `json:"supports_duplicate_detection"`
	SupportsSentimentAnalysis  bool   `json:"supports_sentiment_analysis"`
	SupportsSummarization      bool   `json:"supports_summarization"`
	SupportsSemanticSearch     bool   `json:"supports_semantic_search"`
	SupportsAdvancedAnalytics  bool   `json:"supports_advanced_analytics"`
	SupportsClustering         bool   `json:"supports_clustering"`
	SupportsAnomalyDetection   bool   `json:"supports_anomaly_detection"`
	MaxMemoriesPerQuery        int    `json:"max_memories_per_query"`
	RecommendedMemoryLimit     int    `json:"recommended_memory_limit"`
	PerformanceLevel           string `json:"performance_level"`
	AccuracyLevel              string `json:"accuracy_level"`
}

// RecommendReaderType recommends the best reader type based on requirements
func (f *ReaderFactory) RecommendReaderType(requirements *ReaderRequirements) (ReaderType, error) {
	if requirements == nil {
		return ReaderTypeAdvanced, nil // Default recommendation
	}
	
	// Score each available reader type
	availableTypes := f.GetAvailableReaderTypes()
	bestType := ReaderTypeSimple
	bestScore := 0.0
	
	for _, readerType := range availableTypes {
		capabilities, err := f.GetReaderCapabilities(readerType)
		if err != nil {
			continue
		}
		
		score := f.calculateReaderScore(requirements, capabilities)
		if score > bestScore {
			bestScore = score
			bestType = readerType
		}
	}
	
	if bestScore == 0.0 {
		return "", fmt.Errorf("no suitable reader type found for requirements")
	}
	
	return bestType, nil
}

// ReaderRequirements specifies what capabilities are needed
type ReaderRequirements struct {
	NeedsSemanticSearch     bool `json:"needs_semantic_search"`
	NeedsAdvancedAnalytics  bool `json:"needs_advanced_analytics"`
	NeedsClustering         bool `json:"needs_clustering"`
	NeedsAnomalyDetection   bool `json:"needs_anomaly_detection"`
	MaxMemoryCount          int  `json:"max_memory_count"`
	PerformancePriority     int  `json:"performance_priority"`     // 1-10, higher = more important
	AccuracyPriority        int  `json:"accuracy_priority"`        // 1-10, higher = more important
	ResourceConstraints     bool `json:"resource_constraints"`
}

// CreateAutoReader creates the best reader for the given requirements
func (f *ReaderFactory) CreateAutoReader(requirements *ReaderRequirements) (MemReader, error) {
	readerType, err := f.RecommendReaderType(requirements)
	if err != nil {
		return nil, fmt.Errorf("failed to recommend reader type: %w", err)
	}
	
	return f.CreateReader(readerType)
}

// Private factory methods

func (f *ReaderFactory) createSimpleReader() (MemReader, error) {
	config := f.createReaderConfig(ReadStrategySimple)
	config.AnalysisDepth = "basic"
	
	return NewSimpleMemReader(config), nil
}

func (f *ReaderFactory) createAdvancedReader() (MemReader, error) {
	config := f.createReaderConfig(ReadStrategyAdvanced)
	config.AnalysisDepth = "deep"
	
	return NewAdvancedMemReader(config, f.memCube, f.embedder, f.vectorDB), nil
}

func (f *ReaderFactory) createSemanticReader() (MemReader, error) {
	if f.embedder == nil {
		return nil, fmt.Errorf("embedder required for semantic reader")
	}
	
	if f.vectorDB == nil {
		return nil, fmt.Errorf("vector database required for semantic reader")
	}
	
	config := f.createReaderConfig(ReadStrategySemantic)
	config.AnalysisDepth = "deep"
	
	// Create advanced reader with semantic capabilities
	return NewAdvancedMemReader(config, f.memCube, f.embedder, f.vectorDB), nil
}

func (f *ReaderFactory) createHybridReader() (MemReader, error) {
	config := f.createReaderConfig(ReadStrategyStructural)
	config.AnalysisDepth = "deep"
	
	// Create advanced reader with all available capabilities
	return NewAdvancedMemReader(config, f.memCube, f.embedder, f.vectorDB), nil
}

func (f *ReaderFactory) createReaderConfig(strategy ReadStrategy) *ReaderConfig {
	config := &ReaderConfig{
		Strategy:           strategy,
		PatternDetection:   f.config.PatternDetection,
		QualityAssessment:  f.config.QualityAssessment,
		DuplicateDetection: f.config.DuplicateDetection,
		SentimentAnalysis:  f.config.SentimentAnalysis,
		Summarization:      f.config.Summarization,
		PatternConfig:      f.config.PatternConfig,
		SummarizationConfig: f.config.SummarizationConfig,
		Options:            make(map[string]interface{}),
	}
	
	// Copy options
	for key, value := range f.config.Options {
		config.Options[key] = value
	}
	
	return config
}

func (f *ReaderFactory) calculateReaderScore(requirements *ReaderRequirements, capabilities *ReaderCapabilities) float64 {
	score := 1.0 // Base score
	
	// Check required capabilities
	if requirements.NeedsSemanticSearch && !capabilities.SupportsSemanticSearch {
		return 0.0 // Hard requirement not met
	}
	
	if requirements.NeedsAdvancedAnalytics && !capabilities.SupportsAdvancedAnalytics {
		return 0.0 // Hard requirement not met
	}
	
	if requirements.NeedsClustering && !capabilities.SupportsClustering {
		return 0.0 // Hard requirement not met
	}
	
	if requirements.NeedsAnomalyDetection && !capabilities.SupportsAnomalyDetection {
		return 0.0 // Hard requirement not met
	}
	
	// Check memory count constraints
	if requirements.MaxMemoryCount > capabilities.MaxMemoriesPerQuery {
		return 0.0 // Cannot handle the memory count
	}
	
	// Score performance vs accuracy trade-off
	performanceScore := f.getPerformanceScore(capabilities.PerformanceLevel)
	accuracyScore := f.getAccuracyScore(capabilities.AccuracyLevel)
	
	performanceWeight := float64(requirements.PerformancePriority) / 10.0
	accuracyWeight := float64(requirements.AccuracyPriority) / 10.0
	
	if performanceWeight+accuracyWeight == 0 {
		performanceWeight = 0.5
		accuracyWeight = 0.5
	} else {
		total := performanceWeight + accuracyWeight
		performanceWeight /= total
		accuracyWeight /= total
	}
	
	score *= (performanceScore*performanceWeight + accuracyScore*accuracyWeight)
	
	// Bonus for memory efficiency
	if requirements.MaxMemoryCount <= capabilities.RecommendedMemoryLimit {
		score *= 1.2
	}
	
	// Resource constraints penalty
	if requirements.ResourceConstraints {
		if capabilities.PerformanceLevel == "slow" {
			score *= 0.8
		}
	}
	
	return score
}

func (f *ReaderFactory) getPerformanceScore(level string) float64 {
	switch level {
	case "fast":
		return 1.0
	case "medium":
		return 0.7
	case "slow":
		return 0.4
	default:
		return 0.5
	}
}

func (f *ReaderFactory) getAccuracyScore(level string) float64 {
	switch level {
	case "basic":
		return 0.4
	case "high":
		return 0.7
	case "very_high":
		return 1.0
	default:
		return 0.5
	}
}

// Factory utility functions

// CreateDefaultFactory creates a factory with default configuration
func CreateDefaultFactory() *ReaderFactory {
	return NewReaderFactory(DefaultReaderConfig())
}

// CreateFactoryWithComponents creates a factory with all components
func CreateFactoryWithComponents(memCube interfaces.MemCube, embedder interfaces.Embedder, vectorDB interfaces.VectorDB) *ReaderFactory {
	return NewReaderFactory(DefaultReaderConfig()).
		WithMemCube(memCube).
		WithEmbedder(embedder).
		WithVectorDB(vectorDB)
}

// CreateFactoryFromConfig creates a factory from a specific configuration
func CreateFactoryFromConfig(config *ReaderConfig) *ReaderFactory {
	return NewReaderFactory(config)
}

// MultiReaderStrategy allows using multiple readers in parallel
type MultiReaderStrategy struct {
	readers map[string]MemReader
	weights map[string]float64
	factory *ReaderFactory
}

// NewMultiReaderStrategy creates a new multi-reader strategy
func NewMultiReaderStrategy(factory *ReaderFactory) *MultiReaderStrategy {
	return &MultiReaderStrategy{
		readers: make(map[string]MemReader),
		weights: make(map[string]float64),
		factory: factory,
	}
}

// AddReader adds a reader to the strategy
func (m *MultiReaderStrategy) AddReader(name string, readerType ReaderType, weight float64) error {
	reader, err := m.factory.CreateReader(readerType)
	if err != nil {
		return fmt.Errorf("failed to create reader %s: %w", name, err)
	}
	
	m.readers[name] = reader
	m.weights[name] = weight
	
	return nil
}

// GetReaders returns all readers in the strategy
func (m *MultiReaderStrategy) GetReaders() map[string]MemReader {
	return m.readers
}

// GetWeights returns the weights for all readers
func (m *MultiReaderStrategy) GetWeights() map[string]float64 {
	return m.weights
}

// Close closes all readers in the strategy
func (m *MultiReaderStrategy) Close() error {
	var lastErr error
	
	for _, reader := range m.readers {
		if err := reader.Close(); err != nil {
			lastErr = err
		}
	}
	
	return lastErr
}

// ReaderPool manages a pool of readers for concurrent access
type ReaderPool struct {
	factory     *ReaderFactory
	readerType  ReaderType
	poolSize    int
	readers     chan MemReader
	created     int
}

// NewReaderPool creates a new reader pool
func NewReaderPool(factory *ReaderFactory, readerType ReaderType, poolSize int) *ReaderPool {
	return &ReaderPool{
		factory:    factory,
		readerType: readerType,
		poolSize:   poolSize,
		readers:    make(chan MemReader, poolSize),
		created:    0,
	}
}

// GetReader gets a reader from the pool
func (p *ReaderPool) GetReader() (MemReader, error) {
	select {
	case reader := <-p.readers:
		return reader, nil
	default:
		if p.created < p.poolSize {
			reader, err := p.factory.CreateReader(p.readerType)
			if err != nil {
				return nil, fmt.Errorf("failed to create reader: %w", err)
			}
			p.created++
			return reader, nil
		}
		
		// Wait for an available reader
		reader := <-p.readers
		return reader, nil
	}
}

// ReturnReader returns a reader to the pool
func (p *ReaderPool) ReturnReader(reader MemReader) {
	select {
	case p.readers <- reader:
	default:
		// Pool is full, close the reader
		reader.Close()
		p.created--
	}
}

// Close closes all readers in the pool
func (p *ReaderPool) Close() error {
	close(p.readers)
	
	var lastErr error
	for reader := range p.readers {
		if err := reader.Close(); err != nil {
			lastErr = err
		}
	}
	
	return lastErr
}