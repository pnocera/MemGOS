package schedulers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/logger"
	"github.com/memtensor/memgos/pkg/schedulers/modules"
)

// NATSConfig holds NATS connection configuration
type NATSConfig struct {
	URLs         []string      `json:"urls" yaml:"urls"`
	Username     string        `json:"username,omitempty" yaml:"username,omitempty"`
	Password     string        `json:"password,omitempty" yaml:"password,omitempty"`
	Token        string        `json:"token,omitempty" yaml:"token,omitempty"`
	TLSConfig    *NATSTLSConfig `json:"tls_config,omitempty" yaml:"tls_config,omitempty"`
	MaxReconnect int           `json:"max_reconnect" yaml:"max_reconnect"`
	ReconnectWait time.Duration `json:"reconnect_wait" yaml:"reconnect_wait"`
	Timeout      time.Duration `json:"timeout" yaml:"timeout"`
	
	// JetStream configuration
	JetStreamConfig *JetStreamConfig `json:"jetstream_config,omitempty" yaml:"jetstream_config,omitempty"`
	
	// Clustering configuration
	ClusterConfig *NATSClusterConfig `json:"cluster_config,omitempty" yaml:"cluster_config,omitempty"`
}

// NATSTLSConfig holds TLS configuration for NATS
type NATSTLSConfig struct {
	CertFile string `json:"cert_file,omitempty" yaml:"cert_file,omitempty"`
	KeyFile  string `json:"key_file,omitempty" yaml:"key_file,omitempty"`
	CAFile   string `json:"ca_file,omitempty" yaml:"ca_file,omitempty"`
	Insecure bool   `json:"insecure" yaml:"insecure"`
}

// JetStreamConfig holds JetStream configuration
type JetStreamConfig struct {
	StreamName         string        `json:"stream_name" yaml:"stream_name"`
	StreamSubjects     []string      `json:"stream_subjects" yaml:"stream_subjects"`
	ConsumerName       string        `json:"consumer_name" yaml:"consumer_name"`
	ConsumerDurable    bool          `json:"consumer_durable" yaml:"consumer_durable"`
	MaxDeliver         int           `json:"max_deliver" yaml:"max_deliver"`
	AckWait           time.Duration `json:"ack_wait" yaml:"ack_wait"`
	MaxAckPending     int           `json:"max_ack_pending" yaml:"max_ack_pending"`
	Replicas          int           `json:"replicas" yaml:"replicas"`
	MaxAge            time.Duration `json:"max_age" yaml:"max_age"`
	MaxBytes          int64         `json:"max_bytes" yaml:"max_bytes"`
	MaxMsgs           int64         `json:"max_msgs" yaml:"max_msgs"`
	Retention         string        `json:"retention" yaml:"retention"` // WorkQueue, Limits, Interest
	Discard           string        `json:"discard" yaml:"discard"`     // Old, New
}

// NATSClusterConfig holds NATS clustering configuration
type NATSClusterConfig struct {
	Name       string   `json:"name" yaml:"name"`
	Host       string   `json:"host" yaml:"host"`
	Port       int      `json:"port" yaml:"port"`
	Routes     []string `json:"routes" yaml:"routes"`
	NoAdvertise bool    `json:"no_advertise" yaml:"no_advertise"`
}

// DefaultNATSConfig returns default NATS configuration
func DefaultNATSConfig() *NATSConfig {
	return &NATSConfig{
		URLs:          []string{nats.DefaultURL},
		MaxReconnect:  -1, // Infinite reconnect attempts
		ReconnectWait: 2 * time.Second,
		Timeout:       5 * time.Second,
		JetStreamConfig: &JetStreamConfig{
			StreamName:      "SCHEDULER_STREAM",
			StreamSubjects:  []string{"scheduler.>"},
			ConsumerName:    "scheduler_consumer",
			ConsumerDurable: true,
			MaxDeliver:      3,
			AckWait:         30 * time.Second,
			MaxAckPending:   256,
			Replicas:        1,
			MaxAge:          24 * time.Hour,
			MaxBytes:        1024 * 1024 * 1024, // 1GB
			MaxMsgs:         1000000,
			Retention:       "WorkQueue",
			Discard:         "Old",
		},
		ClusterConfig: &NATSClusterConfig{
			Name: "scheduler-cluster",
			Host: "localhost",
			Port: 4248,
		},
	}
}

// NATSSchedulerModule provides NATS integration for scheduler with JetStream support
type NATSSchedulerModule struct {
	*modules.BaseSchedulerModule
	mu                  sync.RWMutex
	conn                *nats.Conn
	js                  jetstream.JetStream
	config              *NATSConfig
	stream              jetstream.Stream
	consumer            jetstream.Consumer
	
	// Message handling
	listenerRunning     bool
	listenerCtx         context.Context
	listenerCancel      context.CancelFunc
	messageHandlers     map[string]modules.HandlerFunc
	
	// Subjects for different message types
	querySubject        string
	answerSubject       string
	logSubject          string
	monitorSubject      string
	
	logger              *logger.Logger
	startTime           time.Time
}

// NewNATSSchedulerModule creates a new NATS scheduler module
func NewNATSSchedulerModule(config *NATSConfig) *NATSSchedulerModule {
	if config == nil {
		config = DefaultNATSConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	module := &NATSSchedulerModule{
		BaseSchedulerModule: modules.NewBaseSchedulerModule(),
		config:              config,
		listenerRunning:     false,
		listenerCtx:         ctx,
		listenerCancel:      cancel,
		messageHandlers:     make(map[string]modules.HandlerFunc),
		querySubject:        "scheduler.query",
		answerSubject:       "scheduler.answer",
		logSubject:          "scheduler.log",
		monitorSubject:      "scheduler.monitor",
		logger:              logger.GetLogger("nats-scheduler"),
		startTime:           time.Now(),
	}
	
	return module
}

// InitializeNATS establishes connection to NATS and sets up JetStream
func (n *NATSSchedulerModule) InitializeNATS() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	n.logger.Info("Connecting to NATS", "urls", n.config.URLs)
	
	// Build connection options
	opts := []nats.Option{
		nats.MaxReconnects(n.config.MaxReconnect),
		nats.ReconnectWait(n.config.ReconnectWait),
		nats.Timeout(n.config.Timeout),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				n.logger.Warn("NATS disconnected", "error", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			n.logger.Info("NATS reconnected", "url", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			n.logger.Info("NATS connection closed")
		}),
	}
	
	// Add authentication if configured
	if n.config.Username != "" && n.config.Password != "" {
		opts = append(opts, nats.UserInfo(n.config.Username, n.config.Password))
	}
	if n.config.Token != "" {
		opts = append(opts, nats.Token(n.config.Token))
	}
	
	// Add TLS configuration if specified
	if n.config.TLSConfig != nil {
		if n.config.TLSConfig.CertFile != "" && n.config.TLSConfig.KeyFile != "" {
			opts = append(opts, nats.ClientCert(n.config.TLSConfig.CertFile, n.config.TLSConfig.KeyFile))
		}
		if n.config.TLSConfig.CAFile != "" {
			opts = append(opts, nats.RootCAs(n.config.TLSConfig.CAFile))
		}
		if n.config.TLSConfig.Insecure {
			opts = append(opts, nats.Secure())
		}
	}
	
	// Connect to NATS
	var err error
	n.conn, err = nats.Connect(strings.Join(n.config.URLs, ","), opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	
	n.logger.Info("NATS connection established", 
		"url", n.conn.ConnectedUrl(),
		"server_id", n.conn.ConnectedServerId(),
		"server_name", n.conn.ConnectedServerName())
	
	// Initialize JetStream
	if err := n.initializeJetStream(); err != nil {
		n.conn.Close()
		n.conn = nil
		return fmt.Errorf("failed to initialize JetStream: %w", err)
	}
	
	return nil
}

// initializeJetStream sets up JetStream stream and consumer
func (n *NATSSchedulerModule) initializeJetStream() error {
	var err error
	n.js, err = jetstream.New(n.conn)
	if err != nil {
		return fmt.Errorf("failed to create JetStream context: %w", err)
	}
	
	jsCfg := n.config.JetStreamConfig
	
	// Create or get stream
	streamConfig := jetstream.StreamConfig{
		Name:        jsCfg.StreamName,
		Subjects:    jsCfg.StreamSubjects,
		Retention:   n.parseRetentionPolicy(jsCfg.Retention),
		Discard:     n.parseDiscardPolicy(jsCfg.Discard),
		MaxAge:      jsCfg.MaxAge,
		MaxBytes:    jsCfg.MaxBytes,
		MaxMsgs:     jsCfg.MaxMsgs,
		Replicas:    jsCfg.Replicas,
	}
	
	n.stream, err = n.js.CreateOrUpdateStream(n.listenerCtx, streamConfig)
	if err != nil {
		return fmt.Errorf("failed to create or update stream: %w", err)
	}
	
	n.logger.Info("JetStream stream initialized", 
		"name", jsCfg.StreamName,
		"subjects", jsCfg.StreamSubjects)
	
	// Create consumer
	consumerConfig := jetstream.ConsumerConfig{
		Name:          jsCfg.ConsumerName,
		Durable:       jsCfg.ConsumerName, // Durable consumer
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    jsCfg.MaxDeliver,
		AckWait:       jsCfg.AckWait,
		MaxAckPending: jsCfg.MaxAckPending,
		ReplayPolicy:  jetstream.ReplayInstantPolicy,
	}
	
	n.consumer, err = n.stream.CreateOrUpdateConsumer(n.listenerCtx, consumerConfig)
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}
	
	n.logger.Info("JetStream consumer initialized", "name", jsCfg.ConsumerName)
	
	return nil
}

// parseRetentionPolicy converts string to JetStream retention policy
func (n *NATSSchedulerModule) parseRetentionPolicy(policy string) jetstream.RetentionPolicy {
	switch strings.ToLower(policy) {
	case "workqueue":
		return jetstream.WorkQueuePolicy
	case "limits":
		return jetstream.LimitsPolicy
	case "interest":
		return jetstream.InterestPolicy
	default:
		return jetstream.WorkQueuePolicy
	}
}

// parseDiscardPolicy converts string to JetStream discard policy
func (n *NATSSchedulerModule) parseDiscardPolicy(policy string) jetstream.DiscardPolicy {
	switch strings.ToLower(policy) {
	case "old":
		return jetstream.DiscardOld
	case "new":
		return jetstream.DiscardNew
	default:
		return jetstream.DiscardOld
	}
}

// GetConnection returns the NATS connection
func (n *NATSSchedulerModule) GetConnection() *nats.Conn {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.conn
}

// GetJetStream returns the JetStream context
func (n *NATSSchedulerModule) GetJetStream() jetstream.JetStream {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.js
}

// IsConnected checks if NATS connection is active
func (n *NATSSchedulerModule) IsConnected() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	if n.conn == nil {
		return false
	}
	
	return n.conn.IsConnected()
}

// PublishMessage publishes a message to JetStream
func (n *NATSSchedulerModule) PublishMessage(subject string, message *modules.ScheduleMessageItem) (string, error) {
	if n.js == nil {
		return "", fmt.Errorf("JetStream not initialized")
	}
	
	// Serialize message
	data, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}
	
	// Publish to JetStream
	ack, err := n.js.Publish(n.listenerCtx, subject, data)
	if err != nil {
		return "", fmt.Errorf("failed to publish message: %w", err)
	}
	
	messageID := fmt.Sprintf("%s-%d", ack.Stream, ack.Sequence)
	
	n.logger.Debug("Message published to JetStream",
		"subject", subject,
		"stream", ack.Stream,
		"sequence", ack.Sequence,
		"message_id", messageID,
		"label", message.Label)
	
	return messageID, nil
}

// PublishQueryMessage publishes a query message
func (n *NATSSchedulerModule) PublishQueryMessage(message *modules.ScheduleMessageItem) (string, error) {
	return n.PublishMessage(n.querySubject, message)
}

// PublishAnswerMessage publishes an answer message
func (n *NATSSchedulerModule) PublishAnswerMessage(message *modules.ScheduleMessageItem) (string, error) {
	return n.PublishMessage(n.answerSubject, message)
}

// PublishLogMessage publishes a log message
func (n *NATSSchedulerModule) PublishLogMessage(logItem *modules.ScheduleLogForWebItem) error {
	if n.js == nil {
		return fmt.Errorf("JetStream not initialized")
	}
	
	// Serialize log item
	data, err := json.Marshal(logItem)
	if err != nil {
		return fmt.Errorf("failed to marshal log item: %w", err)
	}
	
	// Publish to JetStream
	_, err = n.js.Publish(n.listenerCtx, n.logSubject, data)
	if err != nil {
		return fmt.Errorf("failed to publish log message: %w", err)
	}
	
	n.logger.Debug("Log message published", "subject", n.logSubject, "title", logItem.LogTitle)
	return nil
}

// RegisterHandler registers a message handler for a specific label
func (n *NATSSchedulerModule) RegisterHandler(label string, handler modules.HandlerFunc) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.messageHandlers[label] = handler
}

// StartListening starts consuming messages from JetStream
func (n *NATSSchedulerModule) StartListening() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	if n.listenerRunning {
		return fmt.Errorf("listener is already running")
	}
	
	if n.consumer == nil {
		return fmt.Errorf("consumer not initialized")
	}
	
	n.listenerRunning = true
	
	// Start message consumer goroutine
	go func() {
		defer func() {
			n.mu.Lock()
			n.listenerRunning = false
			n.mu.Unlock()
		}()
		
		if err := n.consumeMessages(); err != nil {
			n.logger.Error("Message consumer error", "error", err)
		}
	}()
	
	n.logger.Info("NATS JetStream listener started",
		"stream", n.config.JetStreamConfig.StreamName,
		"consumer", n.config.JetStreamConfig.ConsumerName)
	
	return nil
}

// consumeMessages continuously consumes messages from JetStream
func (n *NATSSchedulerModule) consumeMessages() error {
	consumeCtx, err := n.consumer.Consume(func(msg jetstream.Msg) {
		if err := n.processJetStreamMessage(msg); err != nil {
			n.logger.Error("Failed to process message", "error", err)
			// Negative acknowledgment - message will be redelivered
			msg.Nak()
		} else {
			// Positive acknowledgment
			msg.Ack()
		}
	})
	
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}
	
	// Wait for context cancellation
	select {
	case <-n.listenerCtx.Done():
		consumeCtx.Stop()
		return nil
	}
}

// processJetStreamMessage processes a message from JetStream
func (n *NATSSchedulerModule) processJetStreamMessage(msg jetstream.Msg) error {
	// Deserialize message
	var scheduleMsg modules.ScheduleMessageItem
	if err := json.Unmarshal(msg.Data(), &scheduleMsg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}
	
	// Get handler for message label
	n.mu.RLock()
	handler, exists := n.messageHandlers[scheduleMsg.Label]
	n.mu.RUnlock()
	
	if !exists {
		n.logger.Warn("No handler registered for message label", "label", scheduleMsg.Label)
		return nil // Skip message
	}
	
	// Process message with handler
	return handler([]*modules.ScheduleMessageItem{&scheduleMsg})
}

// StopListening stops the message listener
func (n *NATSSchedulerModule) StopListening() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	if !n.listenerRunning {
		return fmt.Errorf("listener is not running")
	}
	
	n.listenerCancel()
	
	// Wait for listener to stop (with timeout)
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-timeout:
			n.logger.Warn("Timeout waiting for listener to stop")
			return fmt.Errorf("timeout waiting for listener to stop")
		case <-ticker.C:
			if !n.listenerRunning {
				n.logger.Info("NATS JetStream listener stopped")
				return nil
			}
		}
	}
}

// Close closes the NATS connection
func (n *NATSSchedulerModule) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	if n.listenerRunning {
		n.listenerCancel()
	}
	
	if n.conn != nil {
		n.conn.Close()
		n.conn = nil
	}
	
	n.logger.Info("NATS connection closed")
	return nil
}

// GetStreamInfo returns information about the JetStream stream
func (n *NATSSchedulerModule) GetStreamInfo() (map[string]interface{}, error) {
	if n.stream == nil {
		return nil, fmt.Errorf("stream not initialized")
	}
	
	info, err := n.stream.Info(n.listenerCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}
	
	return map[string]interface{}{
		"name":             info.Config.Name,
		"subjects":         info.Config.Subjects,
		"messages":         info.State.Msgs,
		"bytes":            info.State.Bytes,
		"first_sequence":   info.State.FirstSeq,
		"last_sequence":    info.State.LastSeq,
		"consumer_count":   info.State.NumConsumers,
		"created":          info.Created,
		"retention":        info.Config.Retention.String(),
		"replicas":         info.Config.Replicas,
	}, nil
}

// GetConsumerInfo returns information about the JetStream consumer
func (n *NATSSchedulerModule) GetConsumerInfo() (map[string]interface{}, error) {
	if n.consumer == nil {
		return nil, fmt.Errorf("consumer not initialized")
	}
	
	info, err := n.consumer.Info(n.listenerCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get consumer info: %w", err)
	}
	
	return map[string]interface{}{
		"name":             info.Name,
		"stream_name":      info.Stream,
		"delivered":        info.Delivered,
		"ack_pending":      info.AckFloor,
		"num_pending":      info.NumPending,
		"num_redelivered":  info.NumRedelivered,
		"num_waiting":      info.NumWaiting,
		"created":          info.Created,
		"config":           info.Config,
	}, nil
}

// PublishEvent publishes an event to a NATS subject using core NATS
func (n *NATSSchedulerModule) PublishEvent(subject string, data interface{}) error {
	if n.conn == nil {
		return fmt.Errorf("NATS connection not initialized")
	}
	
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}
	
	err = n.conn.Publish(subject, payload)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}
	
	n.logger.Debug("Event published", "subject", subject)
	return nil
}

// SubscribeToEvents subscribes to NATS subject events using core NATS
func (n *NATSSchedulerModule) SubscribeToEvents(subject string, handler func(string, []byte) error) (*nats.Subscription, error) {
	if n.conn == nil {
		return nil, fmt.Errorf("NATS connection not initialized")
	}
	
	sub, err := n.conn.Subscribe(subject, func(msg *nats.Msg) {
		if err := handler(msg.Subject, msg.Data); err != nil {
			n.logger.Error("Event handler error",
				"subject", msg.Subject,
				"error", err)
		}
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to events: %w", err)
	}
	
	n.logger.Info("Subscribed to events", "subject", subject)
	return sub, nil
}

// RequestReply performs a request-reply operation using NATS
func (n *NATSSchedulerModule) RequestReply(subject string, data interface{}, timeout time.Duration) ([]byte, error) {
	if n.conn == nil {
		return nil, fmt.Errorf("NATS connection not initialized")
	}
	
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}
	
	msg, err := n.conn.Request(subject, payload, timeout)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	
	return msg.Data, nil
}

// GetStats returns NATS module statistics
func (n *NATSSchedulerModule) GetStats() map[string]interface{} {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	stats := map[string]interface{}{
		"connected":        n.IsConnected(),
		"listener_running": n.listenerRunning,
		"uptime":          time.Since(n.startTime),
		"handlers_count":  len(n.messageHandlers),
	}
	
	if n.conn != nil {
		natsStats := n.conn.Stats()
		stats["nats_stats"] = map[string]interface{}{
			"in_msgs":     natsStats.InMsgs,
			"out_msgs":    natsStats.OutMsgs,
			"in_bytes":    natsStats.InBytes,
			"out_bytes":   natsStats.OutBytes,
			"reconnects":  natsStats.Reconnects,
		}
		
		stats["connected_url"] = n.conn.ConnectedUrl()
		stats["server_id"] = n.conn.ConnectedServerId()
		stats["server_name"] = n.conn.ConnectedServerName()
	}
	
	// Add stream and consumer info if available
	if streamInfo, err := n.GetStreamInfo(); err == nil {
		stats["stream_info"] = streamInfo
	}
	
	if consumerInfo, err := n.GetConsumerInfo(); err == nil {
		stats["consumer_info"] = consumerInfo
	}
	
	return stats
}