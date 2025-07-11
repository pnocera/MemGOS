package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/memtensor/memgos/pkg/interfaces"
)

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	DB       int    `json:"db" yaml:"db"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	PoolSize int    `json:"pool_size" yaml:"pool_size"`
}

// DefaultRedisConfig returns default Redis configuration
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Host:     "localhost",
		Port:     6379,
		DB:       0,
		Password: "",
		PoolSize: 10,
	}
}

// RedisSchedulerModule provides Redis integration for scheduler
type RedisSchedulerModule struct {
	*BaseSchedulerModule
	mu                  sync.RWMutex
	client              *redis.Client
	config              *RedisConfig
	queryListCapacity   int
	listenerRunning     bool
	listenerCtx         context.Context
	listenerCancel      context.CancelFunc
	streamName          string
	consumerGroup       string
	consumerName        string
	logger              interfaces.Logger
}

// NewRedisSchedulerModule creates a new Redis scheduler module
func NewRedisSchedulerModule(config *RedisConfig) *RedisSchedulerModule {
	if config == nil {
		config = DefaultRedisConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &RedisSchedulerModule{
		BaseSchedulerModule: NewBaseSchedulerModule(),
		config:              config,
		queryListCapacity:   1000,
		listenerRunning:     false,
		listenerCtx:         ctx,
		listenerCancel:      cancel,
		streamName:          "user:queries:stream",
		consumerGroup:       "scheduler-group",
		consumerName:        "scheduler-consumer",
		logger:              nil, // Will be set by parent
	}
}

// InitializeRedis establishes connection to Redis
func (r *RedisSchedulerModule) InitializeRedis() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	addr := fmt.Sprintf("%s:%d", r.config.Host, r.config.Port)
	r.logger.Info("Connecting to Redis", map[string]interface{}{"address": addr, "db": r.config.DB})
	
	r.client = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: r.config.Password,
		DB:       r.config.DB,
		PoolSize: r.config.PoolSize,
	})
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	pong, err := r.client.Ping(ctx).Result()
	if err != nil {
		r.client = nil
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	
	r.logger.Info("Redis connection established", map[string]interface{}{"response": pong})
	
	// Initialize stream and consumer group
	if err := r.initializeStream(); err != nil {
		r.logger.Warn("Failed to initialize stream", map[string]interface{}{"error": err.Error()})
		// Non-critical error, continue
	}
	
	return nil
}

// initializeStream creates the stream and consumer group if they don't exist
func (r *RedisSchedulerModule) initializeStream() error {
	ctx := context.Background()
	
	// Try to create consumer group (will fail if it already exists)
	err := r.client.XGroupCreate(ctx, r.streamName, r.consumerGroup, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}
	
	// Trim stream to capacity
	err = r.client.XTrimMaxLen(ctx, r.streamName, int64(r.queryListCapacity)).Err()
	if err != nil {
		r.logger.Warn("Failed to trim stream", map[string]interface{}{"error": err.Error()})
	}
	
	return nil
}

// GetClient returns the Redis client
func (r *RedisSchedulerModule) GetClient() *redis.Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.client
}

// IsConnected checks if Redis connection is active
func (r *RedisSchedulerModule) IsConnected() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if r.client == nil {
		return false
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	err := r.client.Ping(ctx).Err()
	return err == nil
}

// AddMessageToStream adds a message to the Redis stream
func (r *RedisSchedulerModule) AddMessageToStream(message *ScheduleMessageItem) (string, error) {
	if r.client == nil {
		return "", fmt.Errorf("Redis client not initialized")
	}
	
	messageData := message.ToDict()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	id, err := r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: r.streamName,
		Values: messageData,
	}).Result()
	
	if err != nil {
		return "", fmt.Errorf("failed to add message to stream: %w", err)
	}
	
	r.logger.Debug("Message added to stream", map[string]interface{}{
		"stream": r.streamName, 
		"id": id, 
		"label": message.Label,
	})
	
	return id, nil
}

// ConsumeMessageStream consumes messages from Redis stream
func (r *RedisSchedulerModule) ConsumeMessageStream(handler func(*ScheduleMessageItem) error) error {
	if r.client == nil {
		return fmt.Errorf("Redis client not initialized")
	}
	
	for r.listenerRunning {
		select {
		case <-r.listenerCtx.Done():
			return nil
		default:
			if err := r.consumeBatch(handler); err != nil {
				r.logger.Error("Error consuming messages", err, map[string]interface{}{})
				time.Sleep(1 * time.Second) // Backoff on error
			}
		}
	}
	
	return nil
}

// consumeBatch processes a batch of messages from the stream
func (r *RedisSchedulerModule) consumeBatch(handler func(*ScheduleMessageItem) error) error {
	ctx, cancel := context.WithTimeout(r.listenerCtx, 5*time.Second)
	defer cancel()
	
	streams, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    r.consumerGroup,
		Consumer: r.consumerName,
		Streams:  []string{r.streamName, ">"},
		Count:    10,
		Block:    2 * time.Second,
	}).Result()
	
	if err != nil {
		if err == redis.Nil {
			// No messages available
			return nil
		}
		return fmt.Errorf("failed to read from stream: %w", err)
	}
	
	for _, stream := range streams {
		for _, message := range stream.Messages {
			if err := r.processMessage(message, handler); err != nil {
				r.logger.Error("Error processing message", err, map[string]interface{}{
					"message_id": message.ID,
				})
				continue
			}
			
			// Acknowledge message
			if err := r.client.XAck(ctx, r.streamName, r.consumerGroup, message.ID).Err(); err != nil {
				r.logger.Error("Failed to acknowledge message", err, map[string]interface{}{
					"message_id": message.ID,
				})
			}
		}
	}
	
	return nil
}

// processMessage converts Redis message to ScheduleMessageItem and calls handler
func (r *RedisSchedulerModule) processMessage(msg redis.XMessage, handler func(*ScheduleMessageItem) error) error {
	// Convert Redis message values to ScheduleMessageItem
	scheduleMsg, err := r.redisMessageToScheduleItem(msg)
	if err != nil {
		return fmt.Errorf("failed to convert message: %w", err)
	}
	
	return handler(scheduleMsg)
}

// redisMessageToScheduleItem converts Redis XMessage to ScheduleMessageItem
func (r *RedisSchedulerModule) redisMessageToScheduleItem(msg redis.XMessage) (*ScheduleMessageItem, error) {
	values := msg.Values
	
	// Parse timestamp
	timestamp := time.Now()
	if timestampStr, ok := values["timestamp"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, timestampStr); err == nil {
			timestamp = parsed
		}
	}
	
	return &ScheduleMessageItem{
		ItemID:    getRedisStringValue(values, "item_id"),
		UserID:    getRedisStringValue(values, "user_id"),
		MemCubeID: getRedisStringValue(values, "mem_cube_id"),
		Label:     getRedisStringValue(values, "label"),
		Content:   getRedisStringValue(values, "content"),
		Timestamp: timestamp,
		// Note: MemCube is not serialized in Redis
	}, nil
}

// getRedisStringValue safely extracts string value from Redis message values  
func getRedisStringValue(values map[string]interface{}, key string) string {
	if val, ok := values[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// StartListening starts the Redis stream listener
func (r *RedisSchedulerModule) StartListening(handler func(*ScheduleMessageItem) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.listenerRunning {
		return fmt.Errorf("listener is already running")
	}
	
	if r.client == nil {
		return fmt.Errorf("Redis client not initialized")
	}
	
	r.listenerRunning = true
	
	go func() {
		defer func() {
			r.mu.Lock()
			r.listenerRunning = false
			r.mu.Unlock()
		}()
		
		if err := r.ConsumeMessageStream(handler); err != nil {
			r.logger.Error("Stream listener error", err, map[string]interface{}{})
		}
	}()
	
	r.logger.Info("Redis stream listener started", map[string]interface{}{
		"stream": r.streamName, 
		"group": r.consumerGroup,
	})
	
	return nil
}

// StopListening stops the Redis stream listener
func (r *RedisSchedulerModule) StopListening() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if !r.listenerRunning {
		return fmt.Errorf("listener is not running")
	}
	
	r.listenerCancel()
	
	// Wait for listener to stop (with timeout)
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-timeout:
			r.logger.Warn("Timeout waiting for listener to stop", map[string]interface{}{})
			return fmt.Errorf("timeout waiting for listener to stop")
		case <-ticker.C:
			if !r.listenerRunning {
				r.logger.Info("Redis stream listener stopped", map[string]interface{}{})
				return nil
			}
		}
	}
}

// Close closes the Redis connection
func (r *RedisSchedulerModule) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.listenerRunning {
		r.listenerCancel()
	}
	
	if r.client != nil {
		err := r.client.Close()
		r.client = nil
		return err
	}
	
	return nil
}

// GetStreamInfo returns information about the Redis stream
func (r *RedisSchedulerModule) GetStreamInfo() (map[string]interface{}, error) {
	if r.client == nil {
		return nil, fmt.Errorf("Redis client not initialized")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	streamInfo, err := r.client.XInfoStream(ctx, r.streamName).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}
	
	return map[string]interface{}{
		"length":           streamInfo.Length,
		"radix_tree_keys":  streamInfo.RadixTreeKeys,
		"radix_tree_nodes": streamInfo.RadixTreeNodes,
		"groups":           streamInfo.Groups,
		"last_generated_id": streamInfo.LastGeneratedID,
		"first_entry":      streamInfo.FirstEntry,
		"last_entry":       streamInfo.LastEntry,
	}, nil
}

// PublishEvent publishes an event to a Redis channel
func (r *RedisSchedulerModule) PublishEvent(channel string, data interface{}) error {
	if r.client == nil {
		return fmt.Errorf("Redis client not initialized")
	}
	
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err = r.client.Publish(ctx, channel, payload).Err()
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}
	
	r.logger.Debug("Event published", map[string]interface{}{"channel": channel})
	return nil
}

// SubscribeToEvents subscribes to Redis channel events
func (r *RedisSchedulerModule) SubscribeToEvents(channels []string, handler func(string, []byte) error) error {
	if r.client == nil {
		return fmt.Errorf("Redis client not initialized")
	}
	
	pubsub := r.client.Subscribe(r.listenerCtx, channels...)
	defer pubsub.Close()
	
	ch := pubsub.Channel()
	
	for {
		select {
		case <-r.listenerCtx.Done():
			return nil
		case msg := <-ch:
			if err := handler(msg.Channel, []byte(msg.Payload)); err != nil {
				r.logger.Error("Event handler error", err, map[string]interface{}{
					"channel": msg.Channel,
				})
			}
		}
	}
}
