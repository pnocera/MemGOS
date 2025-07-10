package modules

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Constants for scheduler configuration
const (
	QueryLabel                     = "query"
	AnswerLabel                    = "answer"
	TreeTextMemorySearchMethod     = "tree_text_memory_search"
	TextMemorySearchMethod         = "text_memory_search"
	DefaultActivationMemSize       = 5
	DefaultThreadPoolMaxWorkers    = 5
	DefaultConsumeIntervalSeconds  = 3 * time.Second
	NotInitialized                 = -1
)

// DictConversionInterface provides methods for dictionary conversion
type DictConversionInterface interface {
	ToDict() map[string]interface{}
}

// ScheduleMessageItem represents a message item for scheduling
type ScheduleMessageItem struct {
	ItemID     string      `json:"item_id"`
	UserID     string      `json:"user_id"`
	MemCubeID  string      `json:"mem_cube_id"`
	Label      string      `json:"label"`
	MemCube    interface{} `json:"mem_cube,omitempty"` // Will be GeneralMemCube interface
	Content    string      `json:"content"`
	Timestamp  time.Time   `json:"timestamp"`
}

// NewScheduleMessageItem creates a new schedule message item
func NewScheduleMessageItem(userID, memCubeID, label, content string, memCube interface{}) *ScheduleMessageItem {
	return &ScheduleMessageItem{
		ItemID:    uuid.New().String(),
		UserID:    userID,
		MemCubeID: memCubeID,
		Label:     label,
		MemCube:   memCube,
		Content:   content,
		Timestamp: time.Now(),
	}
}

// ToDict converts ScheduleMessageItem to dictionary
func (s *ScheduleMessageItem) ToDict() map[string]interface{} {
	return map[string]interface{}{
		"item_id":     s.ItemID,
		"user_id":     s.UserID,
		"mem_cube_id": s.MemCubeID,
		"label":       s.Label,
		"content":     s.Content,
		"timestamp":   s.Timestamp.Format(time.RFC3339),
	}
}

// FromDict creates ScheduleMessageItem from dictionary
func ScheduleMessageItemFromDict(data map[string]interface{}) (*ScheduleMessageItem, error) {
	timestamp, err := time.Parse(time.RFC3339, data["timestamp"].(string))
	if err != nil {
		timestamp = time.Now()
	}

	return &ScheduleMessageItem{
		ItemID:    data["item_id"].(string),
		UserID:    data["user_id"].(string),
		MemCubeID: data["mem_cube_id"].(string),
		Label:     data["label"].(string),
		Content:   data["content"].(string),
		Timestamp: timestamp,
	}, nil
}

// MemorySizes represents current memory utilization
type MemorySizes struct {
	LongTermMemorySize        int `json:"long_term_memory_size"`
	UserMemorySize            int `json:"user_memory_size"`
	WorkingMemorySize         int `json:"working_memory_size"`
	TransformedActMemorySize  int `json:"transformed_act_memory_size"`
	ParameterMemorySize       int `json:"parameter_memory_size"`
}

// MemoryCapacities represents maximum memory capacities
type MemoryCapacities struct {
	LongTermMemoryCapacity       int `json:"long_term_memory_capacity"`
	UserMemoryCapacity           int `json:"user_memory_capacity"`
	WorkingMemoryCapacity        int `json:"working_memory_capacity"`
	TransformedActMemoryCapacity int `json:"transformed_act_memory_capacity"`
	ParameterMemoryCapacity      int `json:"parameter_memory_capacity"`
}

// DefaultMemorySizes returns default memory sizes
func DefaultMemorySizes() MemorySizes {
	return MemorySizes{
		LongTermMemorySize:       NotInitialized,
		UserMemorySize:           NotInitialized,
		WorkingMemorySize:        NotInitialized,
		TransformedActMemorySize: NotInitialized,
		ParameterMemorySize:      NotInitialized,
	}
}

// DefaultMemoryCapacities returns default memory capacities
func DefaultMemoryCapacities() MemoryCapacities {
	return MemoryCapacities{
		LongTermMemoryCapacity:       10000,
		UserMemoryCapacity:           10000,
		WorkingMemoryCapacity:        20,
		TransformedActMemoryCapacity: NotInitialized,
		ParameterMemoryCapacity:      NotInitialized,
	}
}

// ScheduleLogForWebItem represents a log entry for web display
type ScheduleLogForWebItem struct {
	ItemID              string            `json:"item_id"`
	UserID              string            `json:"user_id"`
	MemCubeID           string            `json:"mem_cube_id"`
	Label               string            `json:"label"`
	LogTitle            string            `json:"log_title"`
	LogContent          string            `json:"log_content"`
	CurrentMemorySizes  MemorySizes       `json:"current_memory_sizes"`
	MemoryCapacities    MemoryCapacities  `json:"memory_capacities"`
	Timestamp           time.Time         `json:"timestamp"`
}

// NewScheduleLogForWebItem creates a new log item for web display
func NewScheduleLogForWebItem(userID, memCubeID, label, logTitle, logContent string) *ScheduleLogForWebItem {
	return &ScheduleLogForWebItem{
		ItemID:             uuid.New().String(),
		UserID:             userID,
		MemCubeID:          memCubeID,
		Label:              label,
		LogTitle:           logTitle,
		LogContent:         logContent,
		CurrentMemorySizes: DefaultMemorySizes(),
		MemoryCapacities:   DefaultMemoryCapacities(),
		Timestamp:          time.Now(),
	}
}

// ToDict converts ScheduleLogForWebItem to dictionary
func (s *ScheduleLogForWebItem) ToDict() map[string]interface{} {
	return map[string]interface{}{
		"item_id":               s.ItemID,
		"user_id":               s.UserID,
		"mem_cube_id":           s.MemCubeID,
		"label":                 s.Label,
		"log_title":             s.LogTitle,
		"log_content":           s.LogContent,
		"current_memory_sizes":  s.CurrentMemorySizes,
		"memory_capacities":     s.MemoryCapacities,
		"timestamp":             s.Timestamp.Format(time.RFC3339),
	}
}

// TaskResult represents the result of a scheduled task
type TaskResult struct {
	TaskID    string                 `json:"task_id"`
	Result    interface{}            `json:"result"`
	Error     error                  `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewTaskResult creates a new task result
func NewTaskResult(taskID string, result interface{}, err error) *TaskResult {
	return &TaskResult{
		TaskID:    taskID,
		Result:    result,
		Error:     err,
		Metadata:  make(map[string]interface{}),
		Timestamp: time.Now(),
	}
}

// ToJSON converts task result to JSON
func (tr *TaskResult) ToJSON() ([]byte, error) {
	return json.Marshal(tr)
}

// HandlerFunc represents a message handler function
type HandlerFunc func([]*ScheduleMessageItem) error

// ActivationMemoryItem represents an item in activation memory
type ActivationMemoryItem struct {
	Memory string `json:"memory"`
	Count  int    `json:"count"`
}

// IntentResult represents the result of intent detection
type IntentResult struct {
	TriggerRetrieval bool     `json:"trigger_retrieval"`
	MissingEvidence  []string `json:"missing_evidence"`
	Confidence       float64  `json:"confidence"`
}

// SearchResult represents a search result
type SearchResult struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Score    float64                `json:"score"`
	Metadata map[string]interface{} `json:"metadata"`
}

// NATSMessage represents a message envelope for NATS communication
type NATSMessage struct {
	Type      string                 `json:"type"`      // MESSAGE, WORK_ITEM, RESPONSE, etc.
	Source    string                 `json:"source"`    // Node ID of sender
	Target    string                 `json:"target"`    // Node ID of recipient (optional)
	MessageID string                 `json:"message_id"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   interface{}            `json:"payload"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// DistributedTaskRequest represents a distributed task request via NATS
type DistributedTaskRequest struct {
	TaskID       string                 `json:"task_id"`
	TaskType     string                 `json:"task_type"`     // SEARCH, RERANK, MONITOR, etc.
	Priority     string                 `json:"priority"`      // HIGH, MEDIUM, LOW
	RequestorID  string                 `json:"requestor_id"`
	Parameters   map[string]interface{} `json:"parameters"`
	ReplySubject string                 `json:"reply_subject"`
	Timeout      time.Duration          `json:"timeout"`
	Created      time.Time              `json:"created"`
	Deadline     time.Time              `json:"deadline"`
}

// DistributedTaskResponse represents a response to a distributed task
type DistributedTaskResponse struct {
	TaskID        string                 `json:"task_id"`
	ResponderID   string                 `json:"responder_id"`
	Success       bool                   `json:"success"`
	Result        interface{}            `json:"result,omitempty"`
	Error         string                 `json:"error,omitempty"`
	ProcessingTime time.Duration         `json:"processing_time"`
	Timestamp     time.Time              `json:"timestamp"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ClusterNodeInfo represents information about a node in the cluster
type ClusterNodeInfo struct {
	NodeID            string                 `json:"node_id"`
	NodeType          string                 `json:"node_type"`     // SCHEDULER, RETRIEVER, MONITOR
	Version           string                 `json:"version"`
	Capabilities      []string               `json:"capabilities"`
	LastHeartbeat     time.Time              `json:"last_heartbeat"`
	Status            string                 `json:"status"`        // ACTIVE, INACTIVE, DEGRADED
	LoadPercent       float64                `json:"load_percent"`
	MemoryUsageMB     int64                  `json:"memory_usage_mb"`
	TasksProcessed    int64                  `json:"tasks_processed"`
	TasksInProgress   int                    `json:"tasks_in_progress"`
	AvgResponseTime   time.Duration          `json:"avg_response_time"`
	ErrorRate         float64                `json:"error_rate"`
	CustomMetrics     map[string]interface{} `json:"custom_metrics,omitempty"`
}

// SchedulerClusterState represents the state of the entire scheduler cluster
type SchedulerClusterState struct {
	ClusterID         string                    `json:"cluster_id"`
	Timestamp         time.Time                 `json:"timestamp"`
	TotalNodes        int                       `json:"total_nodes"`
	ActiveNodes       int                       `json:"active_nodes"`
	DegradedNodes     int                       `json:"degraded_nodes"`
	InactiveNodes     int                       `json:"inactive_nodes"`
	Nodes             map[string]*ClusterNodeInfo `json:"nodes"`
	OverallHealth     string                    `json:"overall_health"`   // HEALTHY, DEGRADED, CRITICAL
	TotalCapacity     int                       `json:"total_capacity"`
	UtilizedCapacity  int                       `json:"utilized_capacity"`
	PendingTasks      int64                     `json:"pending_tasks"`
	ProcessingTasks   int64                     `json:"processing_tasks"`
	CompletedTasks    int64                     `json:"completed_tasks"`
	FailedTasks       int64                     `json:"failed_tasks"`
	ClusterMetrics    map[string]interface{}    `json:"cluster_metrics,omitempty"`
}

// NATSStreamInfo represents NATS JetStream stream information
type NATSStreamInfo struct {
	Name          string    `json:"name"`
	Subjects      []string  `json:"subjects"`
	Messages      uint64    `json:"messages"`
	Bytes         uint64    `json:"bytes"`
	FirstSeq      uint64    `json:"first_seq"`
	LastSeq       uint64    `json:"last_seq"`
	ConsumerCount int       `json:"consumer_count"`
	Created       time.Time `json:"created"`
	Config        interface{} `json:"config"`
}

// NATSConsumerInfo represents NATS JetStream consumer information
type NATSConsumerInfo struct {
	Name           string    `json:"name"`
	StreamName     string    `json:"stream_name"`
	Delivered      uint64    `json:"delivered"`
	AckPending     uint64    `json:"ack_pending"`
	NumPending     uint64    `json:"num_pending"`
	NumRedelivered uint64    `json:"num_redelivered"`
	NumWaiting     int       `json:"num_waiting"`
	Created        time.Time `json:"created"`
	Config         interface{} `json:"config"`
}

// LoadBalancingStrategy represents different strategies for distributing work
type LoadBalancingStrategy string

const (
	RoundRobin          LoadBalancingStrategy = "round_robin"
	LeastConnections    LoadBalancingStrategy = "least_connections"
	WeightedRoundRobin  LoadBalancingStrategy = "weighted_round_robin"
	CapacityBased       LoadBalancingStrategy = "capacity_based"
	ResponseTimeBased   LoadBalancingStrategy = "response_time_based"
	GeographicAffinity  LoadBalancingStrategy = "geographic_affinity"
)

// WorkloadDistribution represents how work is distributed across the cluster
type WorkloadDistribution struct {
	Strategy           LoadBalancingStrategy      `json:"strategy"`
	NodeWeights        map[string]float64         `json:"node_weights,omitempty"`
	AffinityRules      map[string][]string        `json:"affinity_rules,omitempty"`
	CapacityThresholds map[string]float64         `json:"capacity_thresholds,omitempty"`
	LastRebalance      time.Time                  `json:"last_rebalance"`
	RebalanceInterval  time.Duration              `json:"rebalance_interval"`
	Metrics            map[string]interface{}     `json:"metrics,omitempty"`
}

// QueueDepthMetrics tracks queue depth across different subjects/topics
type QueueDepthMetrics struct {
	Subject       string            `json:"subject"`
	PendingCount  int64             `json:"pending_count"`
	ProcessingCount int64           `json:"processing_count"`
	CompletedCount int64            `json:"completed_count"`
	FailedCount   int64             `json:"failed_count"`
	AvgWaitTime   time.Duration     `json:"avg_wait_time"`
	AvgProcessingTime time.Duration `json:"avg_processing_time"`
	LastUpdate    time.Time         `json:"last_update"`
	Histogram     map[string]int64  `json:"histogram,omitempty"` // Time buckets
}

// SchedulerStats represents scheduler statistics
type SchedulerStats struct {
	ActivationMemSize int                    `json:"activation_mem_size"`
	TextMem          map[string]interface{} `json:"text_mem"`
	ProcessedTasks   int64                  `json:"processed_tasks"`
	ErrorCount       int64                  `json:"error_count"`
	Uptime           time.Duration          `json:"uptime"`
}
