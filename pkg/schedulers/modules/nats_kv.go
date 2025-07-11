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
)

// NATSKVConfig holds NATS Key-Value store configuration
type NATSKVConfig struct {
	// Connection settings
	URLs          []string       `json:"urls" yaml:"urls"`
	Username      string         `json:"username,omitempty" yaml:"username,omitempty"`
	Password      string         `json:"password,omitempty" yaml:"password,omitempty"`
	Token         string         `json:"token,omitempty" yaml:"token,omitempty"`
	MaxReconnect  int            `json:"max_reconnect" yaml:"max_reconnect"`
	ReconnectWait time.Duration  `json:"reconnect_wait" yaml:"reconnect_wait"`
	Timeout       time.Duration  `json:"timeout" yaml:"timeout"`

	// KV Bucket configuration
	BucketName    string        `json:"bucket_name" yaml:"bucket_name"`
	Description   string        `json:"description,omitempty" yaml:"description,omitempty"`
	MaxValueSize  int32         `json:"max_value_size" yaml:"max_value_size"`
	History       uint8         `json:"history" yaml:"history"`
	TTL           time.Duration `json:"ttl,omitempty" yaml:"ttl,omitempty"`
	MaxBytes      int64         `json:"max_bytes" yaml:"max_bytes"`
	Storage       string        `json:"storage" yaml:"storage"` // File or Memory
	Replicas      int           `json:"replicas" yaml:"replicas"`

	// Stream configuration for messaging
	StreamName     string   `json:"stream_name" yaml:"stream_name"`
	StreamSubjects []string `json:"stream_subjects" yaml:"stream_subjects"`
	
	// Consumer configuration
	ConsumerName    string        `json:"consumer_name" yaml:"consumer_name"`
	ConsumerDurable bool          `json:"consumer_durable" yaml:"consumer_durable"`
	MaxDeliver      int           `json:"max_deliver" yaml:"max_deliver"`
	AckWait         time.Duration `json:"ack_wait" yaml:"ack_wait"`
	MaxAckPending   int           `json:"max_ack_pending" yaml:"max_ack_pending"`
}

// DefaultNATSKVConfig returns default NATS KV configuration
func DefaultNATSKVConfig() *NATSKVConfig {
	return &NATSKVConfig{
		URLs:          []string{nats.DefaultURL},
		MaxReconnect:  -1, // Infinite reconnect attempts
		ReconnectWait: 2 * time.Second,
		Timeout:       5 * time.Second,
		BucketName:    "memgos-scheduler",
		Description:   "MemGOS Scheduler Key-Value Store",
		MaxValueSize:  1024 * 1024, // 1MB
		History:       10,
		MaxBytes:      1024 * 1024 * 1024, // 1GB
		Storage:       "File",
		Replicas:      1,
		StreamName:    "SCHEDULER_STREAM",
		StreamSubjects: []string{"scheduler.>"},
		ConsumerName:   "scheduler_consumer",
		ConsumerDurable: true,
		MaxDeliver:     3,
		AckWait:        30 * time.Second,
		MaxAckPending:  256,
	}
}

// NATSKVSchedulerModule provides NATS Key-Value store integration for scheduler
type NATSKVSchedulerModule struct {
	*BaseSchedulerModule
	mu     sync.RWMutex
	conn   *nats.Conn
	js     jetstream.JetStream
	kv     jetstream.KeyValue
	config *NATSKVConfig

	// Stream and consumer for messaging
	stream   jetstream.Stream
	consumer jetstream.Consumer

	// Message handling
	listenerRunning bool
	listenerCtx     context.Context
	listenerCancel  context.CancelFunc
	messageHandlers map[string]HandlerFunc

	// Subjects for different message types
	querySubject   string
	answerSubject  string
	logSubject     string
	monitorSubject string

	logger    interfaces.Logger
	startTime time.Time
}

// NewNATSKVSchedulerModule creates a new NATS KV scheduler module
func NewNATSKVSchedulerModule(config *NATSKVConfig) *NATSKVSchedulerModule {
	if config == nil {
		config = DefaultNATSKVConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &NATSKVSchedulerModule{
		BaseSchedulerModule: NewBaseSchedulerModule(),
		config:              config,
		listenerRunning:     false,
		listenerCtx:         ctx,
		listenerCancel:      cancel,
		messageHandlers:     make(map[string]HandlerFunc),
		querySubject:        "scheduler.query",
		answerSubject:       "scheduler.answer",
		logSubject:          "scheduler.log",
		monitorSubject:      "scheduler.monitor",
		logger:              nil, // Will be set by parent
		startTime:           time.Now(),
	}
}

// InitializeNATSKV establishes connection to NATS and sets up KV store
func (n *NATSKVSchedulerModule) InitializeNATSKV() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.logger.Info("Connecting to NATS for KV store", map[string]interface{}{
		"urls":   n.config.URLs,
		"bucket": n.config.BucketName,
	})

	// Build connection options
	opts := []nats.Option{
		nats.MaxReconnects(n.config.MaxReconnect),
		nats.ReconnectWait(n.config.ReconnectWait),
		nats.Timeout(n.config.Timeout),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				n.logger.Warn("NATS disconnected", map[string]interface{}{"error": err})
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			n.logger.Info("NATS reconnected", map[string]interface{}{"url": nc.ConnectedUrl()})
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

	// Connect to NATS
	var err error
	n.conn, err = nats.Connect(fmt.Sprintf("%s", n.config.URLs[0]), opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	n.logger.Info("NATS connection established", map[string]interface{}{
		"url":         n.conn.ConnectedUrl(),
		"server_id":   n.conn.ConnectedServerId(),
		"server_name": n.conn.ConnectedServerName(),
	})

	// Initialize JetStream
	if err := n.initializeJetStream(); err != nil {
		n.conn.Close()
		n.conn = nil
		return fmt.Errorf("failed to initialize JetStream: %w", err)
	}

	// Initialize KV store
	if err := n.initializeKVStore(); err != nil {
		n.conn.Close()
		n.conn = nil
		return fmt.Errorf("failed to initialize KV store: %w", err)
	}

	return nil
}

// initializeJetStream sets up JetStream context
func (n *NATSKVSchedulerModule) initializeJetStream() error {
	var err error
	n.js, err = jetstream.New(n.conn)
	if err != nil {
		return fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Create stream for messaging
	streamConfig := jetstream.StreamConfig{
		Name:      n.config.StreamName,
		Subjects:  n.config.StreamSubjects,
		Retention: jetstream.WorkQueuePolicy,
		Discard:   jetstream.DiscardOld,
		MaxAge:    24 * time.Hour,
		MaxBytes:  n.config.MaxBytes,
		MaxMsgs:   1000000,
		Replicas:  n.config.Replicas,
	}

	n.stream, err = n.js.CreateOrUpdateStream(n.listenerCtx, streamConfig)
	if err != nil {
		return fmt.Errorf("failed to create or update stream: %w", err)
	}

	n.logger.Info("JetStream stream initialized", map[string]interface{}{
		"name":     n.config.StreamName,
		"subjects": n.config.StreamSubjects,
	})

	// Create consumer
	consumerConfig := jetstream.ConsumerConfig{
		Name:          n.config.ConsumerName,
		Durable:       n.config.ConsumerName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    n.config.MaxDeliver,
		AckWait:       n.config.AckWait,
		MaxAckPending: n.config.MaxAckPending,
		ReplayPolicy:  jetstream.ReplayInstantPolicy,
	}

	n.consumer, err = n.stream.CreateOrUpdateConsumer(n.listenerCtx, consumerConfig)
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	n.logger.Info("JetStream consumer initialized", map[string]interface{}{
		"name": n.config.ConsumerName,
	})

	return nil
}

// initializeKVStore sets up the Key-Value store
func (n *NATSKVSchedulerModule) initializeKVStore() error {
	kvConfig := jetstream.KeyValueConfig{
		Bucket:       n.config.BucketName,
		Description:  n.config.Description,
		MaxValueSize: n.config.MaxValueSize,
		History:      n.config.History,
		TTL:          n.config.TTL,
		MaxBytes:     n.config.MaxBytes,
		Storage:      n.parseStorageType(n.config.Storage),
		Replicas:     n.config.Replicas,
	}

	var err error
	n.kv, err = n.js.CreateOrUpdateKeyValue(n.listenerCtx, kvConfig)
	if err != nil {
		return fmt.Errorf("failed to create or update KV store: %w", err)
	}

	n.logger.Info("NATS KV store initialized", map[string]interface{}{
		"bucket":     n.config.BucketName,
		"history":    n.config.History,
		"max_bytes":  n.config.MaxBytes,
		"storage":    n.config.Storage,
		"replicas":   n.config.Replicas,
	})

	return nil
}

// parseStorageType converts string to JetStream storage type
func (n *NATSKVSchedulerModule) parseStorageType(storage string) jetstream.StorageType {
	switch storage {
	case "Memory":
		return jetstream.MemoryStorage
	case "File":
		return jetstream.FileStorage
	default:
		return jetstream.FileStorage
	}
}

// GetConnection returns the NATS connection
func (n *NATSKVSchedulerModule) GetConnection() *nats.Conn {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.conn
}

// GetJetStream returns the JetStream context
func (n *NATSKVSchedulerModule) GetJetStream() jetstream.JetStream {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.js
}

// GetKVStore returns the Key-Value store
func (n *NATSKVSchedulerModule) GetKVStore() jetstream.KeyValue {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.kv
}

// IsConnected checks if NATS connection is active
func (n *NATSKVSchedulerModule) IsConnected() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.conn == nil {
		return false
	}

	return n.conn.IsConnected()
}

// KV Store Operations

// Put stores a key-value pair in the KV store
func (n *NATSKVSchedulerModule) Put(key string, value interface{}) (uint64, error) {
	if n.kv == nil {
		return 0, fmt.Errorf("KV store not initialized")
	}

	data, err := json.Marshal(value)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal value: %w", err)
	}

	revision, err := n.kv.Put(n.listenerCtx, key, data)
	if err != nil {
		return 0, fmt.Errorf("failed to put key-value: %w", err)
	}

	n.logger.Debug("Key-value stored", map[string]interface{}{
		"key":      key,
		"revision": revision,
	})

	return revision, nil
}

// Get retrieves a value from the KV store
func (n *NATSKVSchedulerModule) Get(key string) ([]byte, uint64, error) {
	if n.kv == nil {
		return nil, 0, fmt.Errorf("KV store not initialized")
	}

	entry, err := n.kv.Get(n.listenerCtx, key)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get key-value: %w", err)
	}

	return entry.Value(), entry.Revision(), nil
}

// GetJSON retrieves and unmarshals a JSON value from the KV store
func (n *NATSKVSchedulerModule) GetJSON(key string, target interface{}) (uint64, error) {
	data, revision, err := n.Get(key)
	if err != nil {
		return 0, err
	}

	if err := json.Unmarshal(data, target); err != nil {
		return 0, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return revision, nil
}

// Delete removes a key from the KV store
func (n *NATSKVSchedulerModule) Delete(key string) error {
	if n.kv == nil {
		return fmt.Errorf("KV store not initialized")
	}

	err := n.kv.Delete(n.listenerCtx, key)
	if err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}

	n.logger.Debug("Key deleted", map[string]interface{}{"key": key})
	return nil
}

// Update performs atomic update with revision check
func (n *NATSKVSchedulerModule) Update(key string, value interface{}, revision uint64) (uint64, error) {
	if n.kv == nil {
		return 0, fmt.Errorf("KV store not initialized")
	}

	data, err := json.Marshal(value)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal value: %w", err)
	}

	newRevision, err := n.kv.Update(n.listenerCtx, key, data, revision)
	if err != nil {
		return 0, fmt.Errorf("failed to update key-value: %w", err)
	}

	n.logger.Debug("Key-value updated", map[string]interface{}{
		"key":          key,
		"old_revision": revision,
		"new_revision": newRevision,
	})

	return newRevision, nil
}

// Create creates a key-value pair only if the key doesn't exist
func (n *NATSKVSchedulerModule) Create(key string, value interface{}) (uint64, error) {
	if n.kv == nil {
		return 0, fmt.Errorf("KV store not initialized")
	}

	data, err := json.Marshal(value)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal value: %w", err)
	}

	revision, err := n.kv.Create(n.listenerCtx, key, data)
	if err != nil {
		return 0, fmt.Errorf("failed to create key-value: %w", err)
	}

	n.logger.Debug("Key-value created", map[string]interface{}{
		"key":      key,
		"revision": revision,
	})

	return revision, nil
}

// Watch watches for changes to a specific key
func (n *NATSKVSchedulerModule) Watch(key string, handler func(jetstream.KeyValueEntry)) (jetstream.KeyWatcher, error) {
	if n.kv == nil {
		return nil, fmt.Errorf("KV store not initialized")
	}

	watcher, err := n.kv.Watch(n.listenerCtx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to watch key: %w", err)
	}

	// Start watching in a goroutine
	go func() {
		for entry := range watcher.Updates() {
			if entry != nil {
				handler(entry)
			}
		}
	}()

	n.logger.Debug("Watching key", map[string]interface{}{"key": key})
	return watcher, nil
}

// WatchAll watches for changes to all keys in the bucket
func (n *NATSKVSchedulerModule) WatchAll(handler func(jetstream.KeyValueEntry)) (jetstream.KeyWatcher, error) {
	if n.kv == nil {
		return nil, fmt.Errorf("KV store not initialized")
	}

	watcher, err := n.kv.WatchAll(n.listenerCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to watch all keys: %w", err)
	}

	// Start watching in a goroutine
	go func() {
		for entry := range watcher.Updates() {
			if entry != nil {
				handler(entry)
			}
		}
	}()

	n.logger.Debug("Watching all keys in bucket", map[string]interface{}{"bucket": n.config.BucketName})
	return watcher, nil
}

// Keys returns all keys in the KV store
func (n *NATSKVSchedulerModule) Keys() ([]string, error) {
	if n.kv == nil {
		return nil, fmt.Errorf("KV store not initialized")
	}

	keys, err := n.kv.Keys(n.listenerCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}

	return keys, nil
}

// Redis-Compatible Interface Methods

// AddMessageToStream adds a message to the stream (Redis streams compatibility)
func (n *NATSKVSchedulerModule) AddMessageToStream(message *ScheduleMessageItem) (string, error) {
	if n.js == nil {
		return "", fmt.Errorf("JetStream not initialized")
	}

	// Store in KV store for persistence
	key := fmt.Sprintf("message:%s:%d", message.UserID, time.Now().UnixNano())
	revision, err := n.Put(key, message)
	if err != nil {
		n.logger.Warn("Failed to store message in KV", map[string]interface{}{"error": err})
	}

	// Publish to stream for real-time processing
	data, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}

	subject := fmt.Sprintf("%s.%s", n.querySubject, message.Label)
	ack, err := n.js.Publish(n.listenerCtx, subject, data)
	if err != nil {
		return "", fmt.Errorf("failed to publish message: %w", err)
	}

	messageID := fmt.Sprintf("%s-%d", ack.Stream, ack.Sequence)
	if revision > 0 {
		messageID = fmt.Sprintf("%s-kv-%d", messageID, revision)
	}

	n.logger.Debug("Message added to stream and KV store", map[string]interface{}{
		"message_id": messageID,
		"kv_key":     key,
		"kv_revision": revision,
		"stream":     ack.Stream,
		"sequence":   ack.Sequence,
		"label":      message.Label,
	})

	return messageID, nil
}

// StartListening starts consuming messages from the stream
func (n *NATSKVSchedulerModule) StartListening(handler func(*ScheduleMessageItem) error) error {
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

		if err := n.consumeMessages(handler); err != nil {
			n.logger.Error("Message consumer error", err, map[string]interface{}{})
		}
	}()

	n.logger.Info("NATS KV scheduler listener started", map[string]interface{}{
		"stream":   n.config.StreamName,
		"consumer": n.config.ConsumerName,
		"bucket":   n.config.BucketName,
	})

	return nil
}

// consumeMessages continuously consumes messages from JetStream
func (n *NATSKVSchedulerModule) consumeMessages(handler func(*ScheduleMessageItem) error) error {
	consumeCtx, err := n.consumer.Consume(func(msg jetstream.Msg) {
		if err := n.processJetStreamMessage(msg, handler); err != nil {
			n.logger.Error("Failed to process message", err, map[string]interface{}{})
			msg.Nak()
		} else {
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
func (n *NATSKVSchedulerModule) processJetStreamMessage(msg jetstream.Msg, handler func(*ScheduleMessageItem) error) error {
	var scheduleMsg ScheduleMessageItem
	if err := json.Unmarshal(msg.Data(), &scheduleMsg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return handler(&scheduleMsg)
}

// StopListening stops the message listener
func (n *NATSKVSchedulerModule) StopListening() error {
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
				n.logger.Info("NATS KV scheduler listener stopped")
				return nil
			}
		}
	}
}

// Close closes the NATS connection and cleanup resources
func (n *NATSKVSchedulerModule) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.listenerRunning {
		n.listenerCancel()
	}

	if n.conn != nil {
		n.conn.Close()
		n.conn = nil
	}

	n.logger.Info("NATS KV scheduler module closed")
	return nil
}

// GetKVInfo returns information about the KV store
func (n *NATSKVSchedulerModule) GetKVInfo() (map[string]interface{}, error) {
	if n.kv == nil {
		return nil, fmt.Errorf("KV store not initialized")
	}

	status, err := n.kv.Status(n.listenerCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get KV status: %w", err)
	}

	return map[string]interface{}{
		"bucket":         status.Bucket(),
		"values":         status.Values(),
		"history":        status.History(),
		"ttl":            status.TTL(),
		"bucket_location": status.BackingStore(),
	}, nil
}

// GetStats returns comprehensive statistics
func (n *NATSKVSchedulerModule) GetStats() map[string]interface{} {
	n.mu.RLock()
	defer n.mu.RUnlock()

	stats := map[string]interface{}{
		"connected":        n.IsConnected(),
		"listener_running": n.listenerRunning,
		"uptime":           time.Since(n.startTime),
		"bucket_name":      n.config.BucketName,
		"stream_name":      n.config.StreamName,
	}

	if n.conn != nil {
		natsStats := n.conn.Stats()
		stats["nats_stats"] = map[string]interface{}{
			"in_msgs":    natsStats.InMsgs,
			"out_msgs":   natsStats.OutMsgs,
			"in_bytes":   natsStats.InBytes,
			"out_bytes":  natsStats.OutBytes,
			"reconnects": natsStats.Reconnects,
		}

		stats["connected_url"] = n.conn.ConnectedUrl()
		stats["server_id"] = n.conn.ConnectedServerId()
		stats["server_name"] = n.conn.ConnectedServerName()
	}

	// Add KV info if available
	if kvInfo, err := n.GetKVInfo(); err == nil {
		stats["kv_info"] = kvInfo
	}

	return stats
}

// PublishEvent publishes an event to a NATS subject
func (n *NATSKVSchedulerModule) PublishEvent(subject string, data interface{}) error {
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

	n.logger.Debug("Event published", map[string]interface{}{"subject": subject})
	return nil
}

// SubscribeToEvents subscribes to NATS subject events
func (n *NATSKVSchedulerModule) SubscribeToEvents(subject string, handler func(string, []byte) error) (*nats.Subscription, error) {
	if n.conn == nil {
		return nil, fmt.Errorf("NATS connection not initialized")
	}

	sub, err := n.conn.Subscribe(subject, func(msg *nats.Msg) {
		if err := handler(msg.Subject, msg.Data); err != nil {
			n.logger.Error("Event handler error", err, map[string]interface{}{
				"subject": msg.Subject,
			})
		}
	})

	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to events: %w", err)
	}

	n.logger.Info("Subscribed to events", map[string]interface{}{"subject": subject})
	return sub, nil
}