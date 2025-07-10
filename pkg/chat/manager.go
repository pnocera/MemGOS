// Package chat - Chat Manager implementation for session orchestration
package chat

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/logger"
)

// SimpleChatManager implements the ChatManager interface
type SimpleChatManager struct {
	// Configuration
	config *ChatManagerConfig
	logger interfaces.Logger
	
	// Session management
	sessions    map[string]BaseMemChat
	sessionsMux sync.RWMutex
	
	// Factory for creating new chats
	factory *MemChatFactory
	
	// Global metrics
	globalMetrics map[string]*ChatMetrics
	metricsMux    sync.RWMutex
	
	// Control
	isRunning bool
	stopChan  chan struct{}
	
	// Background tasks
	cleanupTicker *time.Ticker
}

// NewChatManager creates a new chat manager
func NewChatManager(config *ChatManagerConfig) (*SimpleChatManager, error) {
	if config == nil {
		config = DefaultChatManagerConfig()
	}
	
	if err := validateChatManagerConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	
	manager := &SimpleChatManager{
		config:        config,
		logger:        logger.NewLogger("ChatManager"),
		sessions:      make(map[string]BaseMemChat),
		factory:       NewMemChatFactory(),
		globalMetrics: make(map[string]*ChatMetrics),
		isRunning:     false,
		stopChan:      make(chan struct{}),
	}
	
	return manager, nil
}

// Start starts the chat manager
func (cm *SimpleChatManager) Start(ctx context.Context) error {
	cm.sessionsMux.Lock()
	defer cm.sessionsMux.Unlock()
	
	if cm.isRunning {
		return fmt.Errorf("chat manager is already running")
	}
	
	cm.isRunning = true
	
	// Start background cleanup task
	cm.cleanupTicker = time.NewTicker(time.Minute * 10) // Cleanup every 10 minutes
	go cm.backgroundCleanup(ctx)
	
	cm.logger.Info("Chat manager started", map[string]interface{}{
		"max_concurrent_sessions": cm.config.MaxConcurrentChats,
		"session_timeout":         cm.config.SessionTimeout,
	})
	
	return nil
}

// Stop stops the chat manager
func (cm *SimpleChatManager) Stop(ctx context.Context) error {
	cm.sessionsMux.Lock()
	defer cm.sessionsMux.Unlock()
	
	if !cm.isRunning {
		return nil
	}
	
	cm.isRunning = false
	
	// Stop background tasks
	if cm.cleanupTicker != nil {
		cm.cleanupTicker.Stop()
	}
	close(cm.stopChan)
	
	// Close all active sessions
	for sessionID, chat := range cm.sessions {
		if err := chat.Close(); err != nil {
			cm.logger.Warn("Failed to close session", map[string]interface{}{
				"session_id": sessionID,
				"error":      err.Error(),
			})
		}
	}
	cm.sessions = make(map[string]BaseMemChat)
	
	cm.logger.Info("Chat manager stopped")
	
	return nil
}

// CreateSession creates a new chat session
func (cm *SimpleChatManager) CreateSession(ctx context.Context, userID string, config *MemChatConfig) (string, error) {
	cm.sessionsMux.Lock()
	defer cm.sessionsMux.Unlock()
	
	if !cm.isRunning {
		return "", fmt.Errorf("chat manager is not running")
	}
	
	// Check session limit
	if len(cm.sessions) >= cm.config.MaxConcurrentChats {
		return "", fmt.Errorf("maximum concurrent sessions reached (%d)", cm.config.MaxConcurrentChats)
	}
	
	// Generate session ID
	sessionID := generateSessionID()
	
	// Apply default chat type if not specified
	if config == nil {
		// Create default config
		config = DefaultMemChatConfig()
		config.UserID = userID
		config.SessionID = sessionID
		
		// Apply global settings
		cm.applyGlobalSettings(config)
	} else {
		// Ensure session ID is set
		config.SessionID = sessionID
		if config.UserID == "" {
			config.UserID = userID
		}
	}
	
	// Create chat instance
	chat, err := cm.factory.CreateFromConfig(config)
	if err != nil {
		return "", fmt.Errorf("failed to create chat session: %w", err)
	}
	
	// Store session
	cm.sessions[sessionID] = chat
	
	// Initialize metrics
	cm.metricsMux.Lock()
	cm.globalMetrics[sessionID] = NewChatMetrics()
	cm.metricsMux.Unlock()
	
	cm.logger.Info("Created chat session", map[string]interface{}{
		"session_id": sessionID,
		"user_id":    userID,
		"chat_type":  cm.config.DefaultChatType,
	})
	
	return sessionID, nil
}

// GetSession retrieves a chat session
func (cm *SimpleChatManager) GetSession(ctx context.Context, sessionID string) (BaseMemChat, error) {
	cm.sessionsMux.RLock()
	defer cm.sessionsMux.RUnlock()
	
	chat, exists := cm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session '%s' not found", sessionID)
	}
	
	return chat, nil
}

// CloseSession closes a specific chat session
func (cm *SimpleChatManager) CloseSession(ctx context.Context, sessionID string) error {
	cm.sessionsMux.Lock()
	defer cm.sessionsMux.Unlock()
	
	chat, exists := cm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session '%s' not found", sessionID)
	}
	
	// Close the chat
	if err := chat.Close(); err != nil {
		cm.logger.Warn("Failed to close chat", map[string]interface{}{
			"session_id": sessionID,
			"error":      err.Error(),
		})
	}
	
	// Remove from sessions
	delete(cm.sessions, sessionID)
	
	// Remove metrics
	cm.metricsMux.Lock()
	delete(cm.globalMetrics, sessionID)
	cm.metricsMux.Unlock()
	
	cm.logger.Info("Closed chat session", map[string]interface{}{
		"session_id": sessionID,
	})
	
	return nil
}

// ListSessions returns a list of active sessions for a user
func (cm *SimpleChatManager) ListSessions(ctx context.Context, userID string) ([]string, error) {
	cm.sessionsMux.RLock()
	defer cm.sessionsMux.RUnlock()
	
	var sessions []string
	for sessionID, chat := range cm.sessions {
		if chat.GetUserID() == userID {
			sessions = append(sessions, sessionID)
		}
	}
	
	return sessions, nil
}

// RouteChat routes a chat message to the appropriate session
func (cm *SimpleChatManager) RouteChat(ctx context.Context, sessionID string, query string) (*ChatResult, error) {
	chat, err := cm.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	
	// Process the chat
	result, err := chat.Chat(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("chat processing failed: %w", err)
	}
	
	// Update global metrics
	cm.updateGlobalMetrics(sessionID, result)
	
	return result, nil
}

// BroadcastMessage sends a message to all sessions of a user
func (cm *SimpleChatManager) BroadcastMessage(ctx context.Context, userID string, message string) error {
	cm.sessionsMux.RLock()
	sessions := make([]BaseMemChat, 0)
	for _, chat := range cm.sessions {
		if chat.GetUserID() == userID {
			sessions = append(sessions, chat)
		}
	}
	cm.sessionsMux.RUnlock()
	
	if len(sessions) == 0 {
		return fmt.Errorf("no active sessions found for user '%s'", userID)
	}
	
	// Send message to all user's sessions
	for _, chat := range sessions {
		// For simplicity, we'll just add a system message to the context
		// In a real implementation, you might want to handle this differently
		_, err := chat.Chat(ctx, fmt.Sprintf("[SYSTEM BROADCAST] %s", message))
		if err != nil {
			cm.logger.Warn("Failed to broadcast message to session", map[string]interface{}{
				"user_id":    userID,
				"session_id": chat.GetSessionID(),
				"error":      err.Error(),
			})
		}
	}
	
	cm.logger.Info("Broadcasted message", map[string]interface{}{
		"user_id":  userID,
		"sessions": len(sessions),
		"message":  message,
	})
	
	return nil
}

// GetGlobalMetrics returns metrics for all sessions
func (cm *SimpleChatManager) GetGlobalMetrics(ctx context.Context) (map[string]*ChatMetrics, error) {
	cm.metricsMux.RLock()
	defer cm.metricsMux.RUnlock()
	
	// Return a copy of metrics to prevent modifications
	metrics := make(map[string]*ChatMetrics)
	for sessionID, metric := range cm.globalMetrics {
		metricCopy := *metric
		metrics[sessionID] = &metricCopy
	}
	
	return metrics, nil
}

// CleanupSessions removes expired or inactive sessions
func (cm *SimpleChatManager) CleanupSessions(ctx context.Context) error {
	cm.sessionsMux.Lock()
	defer cm.sessionsMux.Unlock()
	
	now := time.Now()
	expiredSessions := make([]string, 0)
	
	// Find expired sessions
	for sessionID, chat := range cm.sessions {
		metrics, err := chat.GetMetrics(ctx)
		if err != nil {
			continue
		}
		
		// Check if session has been inactive for too long
		if now.Sub(metrics.LastActive) > cm.config.SessionTimeout {
			expiredSessions = append(expiredSessions, sessionID)
		}
	}
	
	// Close expired sessions
	for _, sessionID := range expiredSessions {
		chat := cm.sessions[sessionID]
		if err := chat.Close(); err != nil {
			cm.logger.Warn("Failed to close expired session", map[string]interface{}{
				"session_id": sessionID,
				"error":      err.Error(),
			})
		}
		
		delete(cm.sessions, sessionID)
		
		cm.metricsMux.Lock()
		delete(cm.globalMetrics, sessionID)
		cm.metricsMux.Unlock()
	}
	
	if len(expiredSessions) > 0 {
		cm.logger.Info("Cleaned up expired sessions", map[string]interface{}{
			"expired_count": len(expiredSessions),
			"active_count":  len(cm.sessions),
		})
	}
	
	return nil
}

// BackupSessions saves session data to a backup location
func (cm *SimpleChatManager) BackupSessions(ctx context.Context, outputPath string) error {
	cm.sessionsMux.RLock()
	defer cm.sessionsMux.RUnlock()
	
	backupData := make(map[string]interface{})
	
	// Collect session data
	for sessionID, chat := range cm.sessions {
		history, err := chat.GetChatHistory(ctx)
		if err != nil {
			cm.logger.Warn("Failed to get chat history for backup", map[string]interface{}{
				"session_id": sessionID,
				"error":      err.Error(),
			})
			continue
		}
		
		metrics, _ := chat.GetMetrics(ctx)
		config := chat.GetConfig()
		
		backupData[sessionID] = map[string]interface{}{
			"history": history,
			"metrics": metrics,
			"config":  config,
		}
	}
	
	// TODO: Implement actual backup to file/database
	cm.logger.Info("Sessions backup prepared", map[string]interface{}{
		"session_count": len(backupData),
		"output_path":   outputPath,
	})
	
	return nil
}

// RestoreSessions restores session data from a backup
func (cm *SimpleChatManager) RestoreSessions(ctx context.Context, inputPath string) error {
	// TODO: Implement session restoration from backup
	cm.logger.Info("Sessions restore initiated", map[string]interface{}{
		"input_path": inputPath,
	})
	
	return nil
}

// GetConfig returns the chat manager configuration
func (cm *SimpleChatManager) GetConfig() *ChatManagerConfig {
	return cm.config
}

// UpdateConfig updates the chat manager configuration
func (cm *SimpleChatManager) UpdateConfig(config *ChatManagerConfig) error {
	if err := validateChatManagerConfig(config); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	
	cm.config = config
	
	cm.logger.Info("Updated chat manager configuration")
	
	return nil
}

// HealthCheck performs a health check on the chat manager
func (cm *SimpleChatManager) HealthCheck(ctx context.Context) error {
	if !cm.isRunning {
		return fmt.Errorf("chat manager is not running")
	}
	
	// Check if we can create new sessions
	if len(cm.sessions) >= cm.config.MaxConcurrentChats {
		cm.logger.Warn("Chat manager at capacity", map[string]interface{}{
			"active_sessions":     len(cm.sessions),
			"max_sessions":        cm.config.MaxConcurrentChats,
		})
	}
	
	// Check active sessions health
	unhealthySessions := 0
	cm.sessionsMux.RLock()
	for sessionID, chat := range cm.sessions {
		if err := chat.HealthCheck(ctx); err != nil {
			unhealthySessions++
			cm.logger.Warn("Unhealthy session detected", map[string]interface{}{
				"session_id": sessionID,
				"error":      err.Error(),
			})
		}
	}
	cm.sessionsMux.RUnlock()
	
	if unhealthySessions > 0 {
		return fmt.Errorf("found %d unhealthy sessions", unhealthySessions)
	}
	
	return nil
}

// Helper methods
func (cm *SimpleChatManager) backgroundCleanup(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-cm.stopChan:
			return
		case <-cm.cleanupTicker.C:
			if err := cm.CleanupSessions(ctx); err != nil {
				cm.logger.Error("Failed to cleanup sessions", err, nil)
			}
		}
	}
}

func (cm *SimpleChatManager) applyGlobalSettings(config *MemChatConfig) {
	if cm.config.GlobalSettings == nil {
		return
	}
	
	// Apply global settings to config
	for key, value := range cm.config.GlobalSettings {
		if config.CustomSettings == nil {
			config.CustomSettings = make(map[string]interface{})
		}
		config.CustomSettings[key] = value
	}
}

func (cm *SimpleChatManager) updateGlobalMetrics(sessionID string, result *ChatResult) {
	cm.metricsMux.Lock()
	defer cm.metricsMux.Unlock()
	
	metrics, exists := cm.globalMetrics[sessionID]
	if !exists {
		metrics = NewChatMetrics()
		cm.globalMetrics[sessionID] = metrics
	}
	
	// Update metrics based on chat result
	metrics.TotalMessages++
	metrics.TotalTokens += result.TokensUsed
	metrics.MemoriesRetrieved += len(result.Memories)
	metrics.MemoriesStored += len(result.NewMemories)
	metrics.LastActive = time.Now()
	
	// Update average response time
	if metrics.TotalMessages == 1 {
		metrics.AverageResponseTime = result.Duration
	} else {
		total := int64(metrics.AverageResponseTime) * (int64(metrics.TotalMessages) - 1)
		total += int64(result.Duration)
		metrics.AverageResponseTime = time.Duration(total / int64(metrics.TotalMessages))
	}
}

// Utility functions
func generateSessionID() string {
	return uuid.New().String()
}

func validateChatManagerConfig(config *ChatManagerConfig) error {
	if config.MaxConcurrentChats < 1 {
		return fmt.Errorf("max_concurrent_chats must be at least 1")
	}
	
	if config.SessionTimeout < time.Minute {
		return fmt.Errorf("session_timeout must be at least 1 minute")
	}
	
	return nil
}

// DefaultChatManagerConfig returns default chat manager configuration
func DefaultChatManagerConfig() *ChatManagerConfig {
	return &ChatManagerConfig{
		DefaultChatType:    "simple",
		MaxConcurrentChats: 100,
		SessionTimeout:     time.Hour * 24,
		MemoryConfig: &MemoryExtractionConfig{
			ExtractionMode:       "automatic",
			KeywordFilters:      []string{"important", "remember", "note"},
			MinimumLength:       10,
			MaximumLength:       500,
			ImportanceScore:     0.5,
			EnableSummarization: true,
			EnableCategorization: true,
		},
		GlobalSettings: make(map[string]interface{}),
	}
}

// SessionInfo represents information about a chat session
type SessionInfo struct {
	SessionID     string                 `json:"session_id"`
	UserID        string                 `json:"user_id"`
	CreatedAt     time.Time              `json:"created_at"`
	LastActive    time.Time              `json:"last_active"`
	MessageCount  int                    `json:"message_count"`
	IsActive      bool                   `json:"is_active"`
	ChatType      string                 `json:"chat_type"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// GetSessionInfo returns detailed information about active sessions
func (cm *SimpleChatManager) GetSessionInfo(ctx context.Context) ([]SessionInfo, error) {
	cm.sessionsMux.RLock()
	defer cm.sessionsMux.RUnlock()
	
	sessionInfos := make([]SessionInfo, 0, len(cm.sessions))
	
	for sessionID, chat := range cm.sessions {
		metrics, err := chat.GetMetrics(ctx)
		if err != nil {
			continue
		}
		
		config := chat.GetConfig()
		
		info := SessionInfo{
			SessionID:    sessionID,
			UserID:       chat.GetUserID(),
			CreatedAt:    config.CreatedAt,
			LastActive:   metrics.LastActive,
			MessageCount: metrics.TotalMessages,
			IsActive:     chat.IsRunning(),
			ChatType:     cm.determineChatType(config),
			Metadata: map[string]interface{}{
				"tokens_used":        metrics.TotalTokens,
				"memories_retrieved": metrics.MemoriesRetrieved,
				"memories_stored":    metrics.MemoriesStored,
				"avg_response_time":  metrics.AverageResponseTime,
			},
		}
		
		sessionInfos = append(sessionInfos, info)
	}
	
	return sessionInfos, nil
}

func (cm *SimpleChatManager) determineChatType(config *MemChatConfig) string {
	if chatType, exists := config.CustomSettings["backend"]; exists {
		if typeStr, ok := chatType.(string); ok {
			return typeStr
		}
	}
	return cm.config.DefaultChatType
}

// GetSessionCount returns the number of active sessions
func (cm *SimpleChatManager) GetSessionCount() int {
	cm.sessionsMux.RLock()
	defer cm.sessionsMux.RUnlock()
	return len(cm.sessions)
}

// GetUserSessionCount returns the number of active sessions for a specific user
func (cm *SimpleChatManager) GetUserSessionCount(userID string) int {
	cm.sessionsMux.RLock()
	defer cm.sessionsMux.RUnlock()
	
	count := 0
	for _, chat := range cm.sessions {
		if chat.GetUserID() == userID {
			count++
		}
	}
	return count
}

// IsRunning returns whether the chat manager is running
func (cm *SimpleChatManager) IsRunning() bool {
	cm.sessionsMux.RLock()
	defer cm.sessionsMux.RUnlock()
	return cm.isRunning
}