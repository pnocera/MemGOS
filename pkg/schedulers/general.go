package schedulers

import (
	"fmt"
	"sync"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/logger"
	"github.com/memtensor/memgos/pkg/schedulers/modules"
)

// GeneralSchedulerConfig extends base configuration with general scheduler specific options
type GeneralSchedulerConfig struct {
	*BaseSchedulerConfig
	TopK                    int           `json:"top_k" yaml:"top_k"`
	TopN                    int           `json:"top_n" yaml:"top_n"`
	ActMemUpdateInterval    time.Duration `json:"act_mem_update_interval" yaml:"act_mem_update_interval"`
	ContextWindowSize       int           `json:"context_window_size" yaml:"context_window_size"`
	ActivationMemSize       int           `json:"activation_mem_size" yaml:"activation_mem_size"`
	ActMemDumpPath          string        `json:"act_mem_dump_path" yaml:"act_mem_dump_path"`
	SearchMethod            string        `json:"search_method" yaml:"search_method"`
}

// DefaultGeneralSchedulerConfig returns default general scheduler configuration
func DefaultGeneralSchedulerConfig() *GeneralSchedulerConfig {
	return &GeneralSchedulerConfig{
		BaseSchedulerConfig:  DefaultBaseSchedulerConfig(),
		TopK:                 10,
		TopN:                 5,
		ActMemUpdateInterval: 5 * time.Minute, // 300 seconds
		ContextWindowSize:    5,
		ActivationMemSize:    modules.DefaultActivationMemSize,
		ActMemDumpPath:       "/tmp/mem_scheduler/activation_memory.cache",
		SearchMethod:         modules.TextMemorySearchMethod,
	}
}

// GeneralScheduler implements a comprehensive memory scheduler with full functionality
type GeneralScheduler struct {
	*BaseScheduler
	mu                            sync.RWMutex
	config                        *GeneralSchedulerConfig
	topK                          int
	topN                          int
	actMemUpdateInterval          time.Duration
	contextWindowSize             int
	activationMemSize             int
	actMemDumpPath                string
	searchMethod                  string
	lastActivationMemUpdateTime   time.Time
	queryList                     []string
	logger                        interfaces.Logger
}

// NewGeneralScheduler creates a new general scheduler
func NewGeneralScheduler(config *GeneralSchedulerConfig) *GeneralScheduler {
	if config == nil {
		config = DefaultGeneralSchedulerConfig()
	}
	
	scheduler := &GeneralScheduler{
		BaseScheduler:               NewBaseScheduler(config.BaseSchedulerConfig),
		config:                      config,
		topK:                        config.TopK,
		topN:                        config.TopN,
		actMemUpdateInterval:        config.ActMemUpdateInterval,
		contextWindowSize:           config.ContextWindowSize,
		activationMemSize:           config.ActivationMemSize,
		actMemDumpPath:              config.ActMemDumpPath,
		searchMethod:                config.SearchMethod,
		lastActivationMemUpdateTime: time.Time{},
		queryList:                   make([]string, 0),
		logger:                      logger.NewConsoleLogger("info"),
	}
	
	return scheduler
}

// InitializeModules initializes all necessary modules for the general scheduler
func (g *GeneralScheduler) InitializeModules(chatLLM interfaces.LLM) error {
	// Initialize base modules first
	if err := g.BaseScheduler.InitializeModules(chatLLM); err != nil {
		return fmt.Errorf("failed to initialize base modules: %w", err)
	}
	
	// Register message handlers specific to general scheduler
	handlers := map[string]modules.HandlerFunc{
		modules.QueryLabel:  g.queryMessageHandler,
		modules.AnswerLabel: g.answerMessageHandler,
	}
	
	if err := g.GetDispatcher().RegisterHandlers(handlers); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}
	
	g.logger.Info("General scheduler initialized", map[string]interface{}{
		"top_k": g.topK,
		"top_n": g.topN,
		"activation_mem_size": g.activationMemSize,
		"search_method": g.searchMethod,
	})
	
	return nil
}

// queryMessageHandler processes query messages
func (g *GeneralScheduler) queryMessageHandler(messages []*modules.ScheduleMessageItem) error {
	g.logger.Debug("Processing query messages", map[string]interface{}{"count": len(messages)})
	
	for _, msg := range messages {
		if msg.Label != modules.QueryLabel {
			g.logger.Error("Query handler received wrong message type", nil, map[string]interface{}{"label": msg.Label})
			continue
		}
		
		// Set current session context
		g.SetCurrentSession(msg.UserID, msg.MemCubeID, msg.MemCube)
		
		// Process the query
		if err := g.processSessionTurn(msg.Content, g.topK, g.topN); err != nil {
			g.logger.Error("Failed to process session turn", err, map[string]interface{}{})
			return err
		}
		
		// Create and submit log
		logItem := g.CreateAutofilledLogItem(
			"User query triggers scheduling...",
			fmt.Sprintf("Processed query: %s", msg.Content),
			modules.QueryLabel,
		)
		
		if err := g.SubmitWebLogs(logItem); err != nil {
			g.logger.Error("Failed to submit web log", err, map[string]interface{}{})
		}
	}
	
	return nil
}

// answerMessageHandler processes answer messages
func (g *GeneralScheduler) answerMessageHandler(messages []*modules.ScheduleMessageItem) error {
	g.logger.Debug("Processing answer messages", map[string]interface{}{"count": len(messages)})
	
	for _, msg := range messages {
		if msg.Label != modules.AnswerLabel {
			g.logger.Error("Answer handler received wrong message type", nil, map[string]interface{}{"label": msg.Label})
			continue
		}
		
		// Set current session context
		g.SetCurrentSession(msg.UserID, msg.MemCubeID, msg.MemCube)
		
		// Process the answer
		if err := g.processAnswer(msg.Content); err != nil {
			g.logger.Error("Failed to process answer", err, map[string]interface{}{})
			return err
		}
		
		// Create and submit log
		logItem := g.CreateAutofilledLogItem(
			"Answer triggers memory update...",
			fmt.Sprintf("Processed answer and updated activation memory"),
			modules.AnswerLabel,
		)
		
		if err := g.SubmitWebLogs(logItem); err != nil {
			g.logger.Error("Failed to submit web log", err, map[string]interface{}{})
		}
	}
	
	return nil
}

// processSessionTurn handles a complete dialog turn with intent detection and retrieval
func (g *GeneralScheduler) processSessionTurn(query string, topK, topN int) error {
	g.mu.Lock()
	g.queryList = append(g.queryList, query)
	g.mu.Unlock()
	
	qList := []string{query}
	
	// Get current working memory from memory cube
	workingMemory, err := g.getCurrentWorkingMemory()
	if err != nil {
		return fmt.Errorf("failed to get working memory: %w", err)
	}
	
	// Detect intent using monitor
	monitor := g.GetMonitor()
	if monitor == nil {
		return fmt.Errorf("monitor not initialized")
	}
	
	intentResult, err := monitor.DetectIntent(qList, workingMemory)
	if err != nil {
		return fmt.Errorf("intent detection failed: %w", err)
	}
	
	g.logger.Debug("Intent detection result", map[string]interface{}{
		"trigger_retrieval": intentResult.TriggerRetrieval,
		"missing_evidence_count": len(intentResult.MissingEvidence),
		"confidence": intentResult.Confidence,
	})
	
	// If retrieval is triggered, perform search and update working memory
	if intentResult.TriggerRetrieval {
		if err := g.performRetrieval(intentResult.MissingEvidence, topK, topN, workingMemory); err != nil {
			return fmt.Errorf("retrieval failed: %w", err)
		}
	}
	
	return nil
}

// performRetrieval performs search and updates working memory
func (g *GeneralScheduler) performRetrieval(missingEvidence []string, topK, topN int, originalMemory []string) error {
	retriever := g.GetRetriever()
	if retriever == nil {
		return fmt.Errorf("retriever not initialized")
	}
	
	numEvidence := len(missingEvidence)
	if numEvidence == 0 {
		return nil
	}
	
	kPerEvidence := max(1, topK/max(1, numEvidence))
	newCandidates := make([]*modules.SearchResult, 0)
	
	// Search for each piece of missing evidence
	for _, evidence := range missingEvidence {
		g.logger.Debug("Searching for missing evidence", map[string]interface{}{"evidence": evidence})
		
		results, err := retriever.Search(evidence, kPerEvidence, g.searchMethod)
		if err != nil {
			g.logger.Error("Search failed", err, map[string]interface{}{"evidence": evidence})
			continue
		}
		
		g.logger.Debug("Search results", map[string]interface{}{"evidence": evidence, "result_count": len(results)})
		newCandidates = append(newCandidates, results...)
	}
	
	if len(newCandidates) > 0 {
		// Replace working memory with reranked candidates
		newOrderMemory, err := retriever.ReplaceWorkingMemory(
			originalMemory,
			newCandidates,
			topK,
			topN,
			g.queryList[len(g.queryList)-1], // Use latest query for reranking
		)
		
		if err != nil {
			return fmt.Errorf("failed to replace working memory: %w", err)
		}
		
		// Update activation memory
		if err := g.updateActivationMemory(newOrderMemory); err != nil {
			g.logger.Error("Failed to update activation memory", err, map[string]interface{}{})
		}
		
		g.logger.Info("Working memory updated", map[string]interface{}{
			"original_count": len(originalMemory),
			"new_candidates": len(newCandidates),
			"final_count": len(newOrderMemory),
		})
	}
	
	return nil
}

// processAnswer handles answer processing and frequency updates
func (g *GeneralScheduler) processAnswer(answer string) error {
	monitor := g.GetMonitor()
	if monitor == nil {
		return fmt.Errorf("monitor not initialized")
	}
	
	// Update activation memory frequencies based on answer
	updatedFreqList, err := monitor.UpdateFreq(answer)
	if err != nil {
		g.logger.Error("Failed to update frequencies", err, map[string]interface{}{})
		// Continue with existing frequencies
	}
	
	// Check if it's time to update activation memory
	now := time.Now()
	if now.Sub(g.lastActivationMemUpdateTime) >= g.actMemUpdateInterval {
		// Extract current activation memories
		currentMemories := make([]string, 0)
		for _, item := range updatedFreqList {
			if item != nil && item.Memory != "" {
				currentMemories = append(currentMemories, item.Memory)
			}
		}
		
		if err := g.updateActivationMemory(currentMemories); err != nil {
			g.logger.Error("Failed to update activation memory", err, map[string]interface{}{})
		} else {
			g.lastActivationMemUpdateTime = now
		}
	}
	
	return nil
}

// getCurrentWorkingMemory retrieves current working memory from the memory cube
func (g *GeneralScheduler) getCurrentWorkingMemory() ([]string, error) {
	_, _, memCube := g.GetCurrentSession()
	if memCube == nil {
		return []string{}, fmt.Errorf("memory cube not set")
	}
	
	// This would integrate with the actual memory cube interface
	// For now, return placeholder working memory
	// In real implementation, this would call: memCube.GetWorkingMemory()
	workingMemory := []string{
		"Example working memory item 1",
		"Example working memory item 2",
		"Example working memory item 3",
	}
	
	g.logger.Debug("Retrieved working memory", map[string]interface{}{"count": len(workingMemory)})
	return workingMemory, nil
}

// updateActivationMemory updates the activation memory with new memory items
func (g *GeneralScheduler) updateActivationMemory(newMemory []string) error {
	if len(newMemory) == 0 {
		g.logger.Warn("No new memory to update activation memory")
		return nil
	}
	
	// Assemble memory text using template
	memoryText := g.assembleMemoryText(newMemory)
	
	// This would integrate with the actual memory cube's activation memory
	// For now, simulate the activation memory update
	g.logger.Debug("Updating activation memory", map[string]interface{}{
		"memory_count": len(newMemory),
		"text_length": len(memoryText),
	})
	
	// In real implementation, this would:
	// 1. Get activation memory from memory cube
	// 2. Delete all existing items
	// 3. Extract new cache items from memory text
	// 4. Add new cache items
	// 5. Dump to specified path
	
	// Simulate saving to dump path
	if err := g.saveActivationMemoryToDisk(memoryText); err != nil {
		return fmt.Errorf("failed to save activation memory: %w", err)
	}
	
	g.logger.Info("Activation memory updated successfully", map[string]interface{}{"items": len(newMemory)})
	return nil
}

// assembleMemoryText assembles memory items into formatted text
func (g *GeneralScheduler) assembleMemoryText(memoryItems []string) string {
	assembledText := ""
	for i, item := range memoryItems {
		if item != "" {
			assembledText += fmt.Sprintf("%d. %s\n", i+1, item)
		}
	}
	return fmt.Sprintf("Assembled Memory Context:\n\n%s\n\nThis represents the current active memory context for processing.", assembledText)
}

// saveActivationMemoryToDisk saves activation memory to disk (placeholder implementation)
func (g *GeneralScheduler) saveActivationMemoryToDisk(memoryText string) error {
	// In real implementation, this would save to g.actMemDumpPath
	// For now, just log the operation
	g.logger.Debug("Saving activation memory to disk", map[string]interface{}{"path": g.actMemDumpPath})
	
	// Placeholder: Create directory if it doesn't exist and save file
	// os.MkdirAll(filepath.Dir(g.actMemDumpPath), 0755)
	// return os.WriteFile(g.actMemDumpPath, []byte(memoryText), 0644)
	
	return nil
}

// GetConfig returns the general scheduler configuration
func (g *GeneralScheduler) GetConfig() *GeneralSchedulerConfig {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.config
}

// GetQueryList returns a copy of the current query list
func (g *GeneralScheduler) GetQueryList() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	queries := make([]string, len(g.queryList))
	copy(queries, g.queryList)
	return queries
}

// ClearQueryList clears the query list
func (g *GeneralScheduler) ClearQueryList() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.queryList = make([]string, 0)
}

// GetStats returns extended statistics including general scheduler metrics
func (g *GeneralScheduler) GetStats() map[string]interface{} {
	stats := g.BaseScheduler.GetStats()
	
	g.mu.RLock()
	stats["query_count"] = len(g.queryList)
	stats["last_activation_update"] = g.lastActivationMemUpdateTime
	stats["top_k"] = g.topK
	stats["top_n"] = g.topN
	stats["search_method"] = g.searchMethod
	stats["activation_mem_size"] = g.activationMemSize
	stats["context_window_size"] = g.contextWindowSize
	g.mu.RUnlock()
	
	return stats
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
