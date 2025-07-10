package modules

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
)

// SchedulerMonitor monitors scheduler performance and manages activation memory
type SchedulerMonitor struct {
	*BaseSchedulerModule
	mu                        sync.RWMutex
	statistics               map[string]interface{}
	intentHistory            []string
	activationMemSize        int
	activationMemoryFreqList []*ActivationMemoryItem
	chatLLM                  interfaces.LLM
	logger                   interfaces.Logger
	startTime                time.Time
	processedTasks           int64
	errorCount               int64
}

// NewSchedulerMonitor creates a new scheduler monitor
func NewSchedulerMonitor(chatLLM interfaces.LLM, activationMemSize int) *SchedulerMonitor {
	if activationMemSize <= 0 {
		activationMemSize = DefaultActivationMemSize
	}
	
	monitor := &SchedulerMonitor{
		BaseSchedulerModule:      NewBaseSchedulerModule(),
		statistics:               make(map[string]interface{}),
		intentHistory:            make([]string, 0),
		activationMemSize:        activationMemSize,
		activationMemoryFreqList: make([]*ActivationMemoryItem, activationMemSize),
		chatLLM:                  chatLLM,
		logger:                   nil, // Will be set by parent
		startTime:                time.Now(),
		processedTasks:           0,
		errorCount:               0,
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
		monitor.logger.Error("Failed to initialize templates", "error", err)
	}
	
	return monitor
}

// UpdateStats updates monitor statistics with memory cube information
func (m *SchedulerMonitor) UpdateStats(memCube interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.statistics["activation_mem_size"] = m.activationMemSize
	m.statistics["processed_tasks"] = m.processedTasks
	m.statistics["error_count"] = m.errorCount
	m.statistics["uptime"] = time.Since(m.startTime)
	
	// Add memory cube specific information if available
	if memCubeInfo := m.getMemCubeInfo(memCube); memCubeInfo != nil {
		m.statistics["mem_cube"] = memCubeInfo
	}
	
	m.logger.Debug("Statistics updated", "stats", m.statistics)
}

// getMemCubeInfo extracts information from memory cube
func (m *SchedulerMonitor) getMemCubeInfo(memCube interface{}) map[string]interface{} {
	if memCube == nil {
		return nil
	}
	
	// This would need to be adapted based on the actual MemCube interface
	// For now, return basic information
	return map[string]interface{}{
		"type": fmt.Sprintf("%T", memCube),
		"initialized": true,
	}
}

// DetectIntent analyzes user queries and working memory to determine if retrieval is needed
func (m *SchedulerMonitor) DetectIntent(qList []string, textWorkingMemory []string) (*IntentResult, error) {
	if m.chatLLM == nil {
		return nil, fmt.Errorf("chat LLM not initialized")
	}
	
	params := map[string]interface{}{
		"q_list":               qList,
		"working_memory_list":   textWorkingMemory,
	}
	
	prompt, err := m.BuildPrompt("intent_recognizing", params)
	if err != nil {
		return nil, fmt.Errorf("failed to build intent recognition prompt: %w", err)
	}
	
	m.logger.Debug("Detecting intent", "query_count", len(qList), "memory_count", len(textWorkingMemory))
	
	// Generate response from LLM
	messages := []map[string]string{
		{"role": "user", "content": prompt},
	}
	
	response, err := m.chatLLM.Generate(messages)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}
	
	// Parse JSON response
	intentResult, err := m.extractJSONDict(response)
	if err != nil {
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
	
	m.logger.Debug("Intent detected", 
		"trigger_retrieval", result.TriggerRetrieval,
		"missing_evidence_count", len(result.MissingEvidence),
		"confidence", result.Confidence)
	
	return result, nil
}

// UpdateFreq uses LLM to detect which memories appear in the answer and updates their frequency
func (m *SchedulerMonitor) UpdateFreq(answer string) ([]*ActivationMemoryItem, error) {
	if m.chatLLM == nil {
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
		return freqList, fmt.Errorf("failed to build frequency detection prompt: %w", err)
	}
	
	// Generate response from LLM
	messages := []map[string]string{
		{"role": "user", "content": prompt},
	}
	
	response, err := m.chatLLM.Generate(messages)
	if err != nil {
		m.logger.Error("LLM generation failed for frequency update", "error", err)
		return freqList, err
	}
	
	// Parse JSON response
	result, err := m.extractJSONDict(response)
	if err != nil {
		m.logger.Error("Failed to parse frequency response", "error", err)
		return freqList, err
	}
	
	// Extract updated frequency list
	if updatedList, ok := result["activation_memory_freq_list"]; ok {
		if updatedItems, err := m.parseActivationMemoryItems(updatedList); err == nil {
			m.mu.Lock()
			m.activationMemoryFreqList = updatedItems
			m.mu.Unlock()
			return updatedItems, nil
		} else {
			m.logger.Error("Failed to parse updated activation memory items", "error", err)
		}
	}
	
	return freqList, nil
}

// parseActivationMemoryItems converts interface{} to []*ActivationMemoryItem
func (m *SchedulerMonitor) parseActivationMemoryItems(data interface{}) ([]*ActivationMemoryItem, error) {
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
func (m *SchedulerMonitor) GetActivationMemoryFreqList() []*ActivationMemoryItem {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make([]*ActivationMemoryItem, len(m.activationMemoryFreqList))
	copy(result, m.activationMemoryFreqList)
	return result
}

// GetStatistics returns a copy of current statistics
func (m *SchedulerMonitor) GetStatistics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := make(map[string]interface{})
	for k, v := range m.statistics {
		stats[k] = v
	}
	return stats
}

// IncrementProcessedTasks increments the processed tasks counter
func (m *SchedulerMonitor) IncrementProcessedTasks() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processedTasks++
}

// IncrementErrorCount increments the error counter
func (m *SchedulerMonitor) IncrementErrorCount() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorCount++
}

// GetIntentHistory returns recent intent detection history
func (m *SchedulerMonitor) GetIntentHistory() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	history := make([]string, len(m.intentHistory))
	copy(history, m.intentHistory)
	return history
}

// extractJSONDict extracts JSON dictionary from LLM response
func (m *SchedulerMonitor) extractJSONDict(response string) (map[string]interface{}, error) {
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
func getBoolValue(data map[string]interface{}, key string) bool {
	if val, ok := data[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func getFloat64Value(data map[string]interface{}, key string) float64 {
	if val, ok := data[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		case int64:
			return float64(v)
		}
	}
	return 0.0
}

func getStringSliceValue(data map[string]interface{}, key string) []string {
	if val, ok := data[key]; ok {
		if slice, ok := val.([]interface{}); ok {
			result := make([]string, 0, len(slice))
			for _, item := range slice {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return []string{}
}

func getStringValue(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
