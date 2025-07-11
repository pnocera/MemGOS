// Package core provides the core Memory Operating System implementation
package core

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/errors"
	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/memory"
	"github.com/memtensor/memgos/pkg/types"
)

// MOSCore implements the Memory Operating System Core
type MOSCore struct {
	config        *config.MOSConfig
	userID        string
	sessionID     string
	cubeRegistry  *memory.CubeRegistry
	chatLLM       interfaces.LLM
	userManager   interfaces.UserManager
	chatManager   interfaces.ChatManager
	scheduler     interfaces.Scheduler
	logger        interfaces.Logger
	metrics       interfaces.Metrics
	healthChecker interfaces.HealthChecker
	mu            sync.RWMutex
	initialized   bool
	closed        bool
}

// NewMOSCore creates a new MOS Core instance
func NewMOSCore(cfg *config.MOSConfig, logger interfaces.Logger, metrics interfaces.Metrics) (*MOSCore, error) {
	if cfg == nil {
		return nil, errors.NewValidationError("configuration is required")
	}

	core := &MOSCore{
		config:       cfg,
		userID:       cfg.UserID,
		sessionID:    cfg.SessionID,
		logger:       logger,
		metrics:      metrics,
		cubeRegistry: memory.NewCubeRegistry(logger, metrics),
	}

	return core, nil
}

// Initialize initializes the MOS core
func (mos *MOSCore) Initialize(ctx context.Context) error {
	mos.mu.Lock()
	defer mos.mu.Unlock()

	if mos.initialized {
		return nil
	}

	if mos.closed {
		return errors.NewMemoryError("MOS core is closed")
	}

	start := time.Now()
	defer func() {
		if mos.metrics != nil {
			mos.metrics.Timer("mos_core_initialize_duration", float64(time.Since(start).Milliseconds()), nil)
		}
	}()

	// Initialize components
	if err := mos.initializeComponents(ctx); err != nil {
		return fmt.Errorf("failed to initialize components: %w", err)
	}

	// Validate initial user
	if mos.userManager != nil {
		valid, err := mos.userManager.ValidateUser(ctx, mos.userID)
		if err != nil {
			return fmt.Errorf("failed to validate user: %w", err)
		}
		if !valid {
			return errors.NewUnauthorizedError(fmt.Sprintf("user '%s' does not exist or is inactive", mos.userID))
		}
	}

	mos.initialized = true

	mos.logger.Info("MOS Core initialized", map[string]interface{}{
		"user_id":    mos.userID,
		"session_id": mos.sessionID,
	})

	if mos.metrics != nil {
		mos.metrics.Counter("mos_core_initialize_count", 1, nil)
	}

	return nil
}

// initializeComponents initializes all MOS components
func (mos *MOSCore) initializeComponents(ctx context.Context) error {
	// Initialize LLM
	if err := mos.initializeLLM(); err != nil {
		return fmt.Errorf("failed to initialize LLM: %w", err)
	}

	// Initialize user manager
	if err := mos.initializeUserManager(); err != nil {
		return fmt.Errorf("failed to initialize user manager: %w", err)
	}

	// Initialize chat manager
	if err := mos.initializeChatManager(); err != nil {
		return fmt.Errorf("failed to initialize chat manager: %w", err)
	}

	// Initialize scheduler if enabled
	if mos.config.EnableMemScheduler {
		if err := mos.initializeScheduler(ctx); err != nil {
			return fmt.Errorf("failed to initialize scheduler: %w", err)
		}
	}

	// Initialize health checker if enabled
	if mos.config.HealthCheckEnabled {
		if err := mos.initializeHealthChecker(); err != nil {
			return fmt.Errorf("failed to initialize health checker: %w", err)
		}
	}

	return nil
}

// initializeLLM initializes the LLM component
func (mos *MOSCore) initializeLLM() error {
	// TODO: Implement LLM factory and initialization
	mos.logger.Info("LLM initialization placeholder")
	return nil
}

// initializeUserManager initializes the user manager component
func (mos *MOSCore) initializeUserManager() error {
	// TODO: Implement user manager factory and initialization
	mos.logger.Info("User manager initialization placeholder")
	return nil
}

// initializeChatManager initializes the chat manager component
func (mos *MOSCore) initializeChatManager() error {
	// TODO: Implement chat manager factory and initialization
	mos.logger.Info("Chat manager initialization placeholder")
	return nil
}

// initializeScheduler initializes the scheduler component
func (mos *MOSCore) initializeScheduler(ctx context.Context) error {
	if !mos.config.EnableMemScheduler {
		return nil
	}

	// TODO: Implement scheduler factory and initialization
	mos.logger.Info("Scheduler initialization placeholder")
	return nil
}

// initializeHealthChecker initializes the health checker component
func (mos *MOSCore) initializeHealthChecker() error {
	if !mos.config.HealthCheckEnabled {
		return nil
	}

	// TODO: Implement health checker initialization
	mos.logger.Info("Health checker initialization placeholder")
	return nil
}

// RegisterMemCube registers a memory cube
func (mos *MOSCore) RegisterMemCube(ctx context.Context, cubePath, cubeID, userID string) error {
	if !mos.initialized {
		return errors.NewMemoryError("MOS core not initialized")
	}

	targetUserID := userID
	if targetUserID == "" {
		targetUserID = mos.userID
	}

	// Validate user access
	if mos.userManager != nil {
		valid, err := mos.userManager.ValidateUser(ctx, targetUserID)
		if err != nil {
			return fmt.Errorf("failed to validate user: %w", err)
		}
		if !valid {
			return errors.NewUnauthorizedError(fmt.Sprintf("user '%s' does not exist or is inactive", targetUserID))
		}
	}

	// Set default cube ID if not provided
	if cubeID == "" {
		cubeID = cubePath
	}

	// Check if cube already exists
	if _, err := mos.cubeRegistry.Get(cubeID); err == nil {
		mos.logger.Info("Memory cube already registered", map[string]interface{}{
			"cube_id": cubeID,
			"user_id": targetUserID,
		})
		return nil
	}

	// Register cube from directory or remote repository
	var err error
	if mos.isLocalPath(cubePath) {
		err = mos.cubeRegistry.LoadFromDirectory(ctx, cubeID, cubePath, cubePath)
	} else {
		err = mos.cubeRegistry.LoadFromRemoteRepository(ctx, cubeID, cubePath, cubePath)
	}

	if err != nil {
		return fmt.Errorf("failed to register memory cube: %w", err)
	}

	// TODO: Update user-cube associations in user manager

	mos.logger.Info("Registered memory cube", map[string]interface{}{
		"cube_id":   cubeID,
		"cube_path": cubePath,
		"user_id":   targetUserID,
	})

	if mos.metrics != nil {
		mos.metrics.Counter("memory_cube_register_count", 1, map[string]string{
			"user_id": targetUserID,
		})
	}

	return nil
}

// UnregisterMemCube unregisters a memory cube
func (mos *MOSCore) UnregisterMemCube(ctx context.Context, cubeID, userID string) error {
	if !mos.initialized {
		return errors.NewMemoryError("MOS core not initialized")
	}

	targetUserID := userID
	if targetUserID == "" {
		targetUserID = mos.userID
	}

	// Validate user access to cube
	// TODO: Implement user-cube access validation

	if err := mos.cubeRegistry.Unregister(cubeID); err != nil {
		return fmt.Errorf("failed to unregister memory cube: %w", err)
	}

	mos.logger.Info("Unregistered memory cube", map[string]interface{}{
		"cube_id": cubeID,
		"user_id": targetUserID,
	})

	if mos.metrics != nil {
		mos.metrics.Counter("memory_cube_unregister_count", 1, map[string]string{
			"user_id": targetUserID,
		})
	}

	return nil
}

// Search searches across all registered memory cubes
func (mos *MOSCore) Search(ctx context.Context, query *types.SearchQuery) (*types.MOSSearchResult, error) {
	if !mos.initialized {
		return nil, errors.NewMemoryError("MOS core not initialized")
	}

	start := time.Now()
	defer func() {
		if mos.metrics != nil {
			mos.metrics.Timer("mos_search_duration", float64(time.Since(start).Milliseconds()), nil)
		}
	}()

	targetUserID := query.UserID
	if targetUserID == "" {
		targetUserID = mos.userID
	}

	// Get accessible cubes for user
	cubeIDs := query.CubeIDs
	if len(cubeIDs) == 0 {
		// TODO: Get all cubes accessible by user from user manager
		cubes := mos.cubeRegistry.List()
		for _, cube := range cubes {
			cubeIDs = append(cubeIDs, cube.GetID())
		}
	}

	result := &types.MOSSearchResult{
		TextMem: make([]types.CubeMemoryResult, 0),
		ActMem:  make([]types.CubeMemoryResult, 0),
		ParaMem: make([]types.CubeMemoryResult, 0),
	}

	topK := query.TopK
	if topK <= 0 {
		topK = mos.config.TopK
	}

	// Search across specified cubes
	for _, cubeID := range cubeIDs {
		cube, err := mos.cubeRegistry.Get(cubeID)
		if err != nil {
			mos.logger.Warn("Failed to get cube for search", map[string]interface{}{
				"cube_id": cubeID,
				"error":   err.Error(),
			})
			continue
		}

		// Search textual memory
		if mos.config.EnableTextualMemory && cube.GetTextualMemory() != nil {
			memories, err := cube.GetTextualMemory().Search(ctx, query.Query, topK)
			if err != nil {
				mos.logger.Warn("Failed to search textual memory", map[string]interface{}{
					"cube_id": cubeID,
					"error":   err.Error(),
				})
			} else if len(memories) > 0 {
				result.TextMem = append(result.TextMem, types.CubeMemoryResult{
					CubeID:   cubeID,
					Memories: memories,
				})
			}
		}

		// Search activation memory
		if mos.config.EnableActivationMemory && cube.GetActivationMemory() != nil {
			memories, err := cube.GetActivationMemory().Search(ctx, query.Query, topK)
			if err != nil {
				mos.logger.Warn("Failed to search activation memory", map[string]interface{}{
					"cube_id": cubeID,
					"error":   err.Error(),
				})
			} else if len(memories) > 0 {
				result.ActMem = append(result.ActMem, types.CubeMemoryResult{
					CubeID:   cubeID,
					Memories: memories,
				})
			}
		}

		// Search parametric memory
		if mos.config.EnableParametricMemory && cube.GetParametricMemory() != nil {
			memories, err := cube.GetParametricMemory().Search(ctx, query.Query, topK)
			if err != nil {
				mos.logger.Warn("Failed to search parametric memory", map[string]interface{}{
					"cube_id": cubeID,
					"error":   err.Error(),
				})
			} else if len(memories) > 0 {
				result.ParaMem = append(result.ParaMem, types.CubeMemoryResult{
					CubeID:   cubeID,
					Memories: memories,
				})
			}
		}
	}

	mos.logger.Info("Completed memory search", map[string]interface{}{
		"query":              query.Query,
		"user_id":            targetUserID,
		"cubes_searched":     len(cubeIDs),
		"textual_results":    len(result.TextMem),
		"activation_results": len(result.ActMem),
		"parametric_results": len(result.ParaMem),
	})

	if mos.metrics != nil {
		mos.metrics.Counter("mos_search_count", 1, map[string]string{
			"user_id": targetUserID,
		})
		mos.metrics.Gauge("mos_search_results_textual", float64(len(result.TextMem)), nil)
		mos.metrics.Gauge("mos_search_results_activation", float64(len(result.ActMem)), nil)
		mos.metrics.Gauge("mos_search_results_parametric", float64(len(result.ParaMem)), nil)
	}

	return result, nil
}

// Add adds memory to a specific cube
func (mos *MOSCore) Add(ctx context.Context, request *types.AddMemoryRequest) error {
	if !mos.initialized {
		return errors.NewMemoryError("MOS core not initialized")
	}

	// Validate request
	if request.Messages == nil && request.MemoryContent == nil && request.DocPath == nil {
		return errors.NewValidationError("at least one of messages, memory_content, or doc_path must be provided")
	}

	targetUserID := mos.userID
	if request.UserID != "" {
		targetUserID = request.UserID
	}

	// Determine target cube
	cubeID := ""
	if request.MemCubeID != "" {
		cubeID = request.MemCubeID
	} else {
		// TODO: Get default cube for user
		cubes := mos.cubeRegistry.List()
		if len(cubes) == 0 {
			return errors.NewValidationError("no memory cubes available and no cube_id specified")
		}
		cubeID = cubes[0].GetID()
	}

	// Get the cube
	cube, err := mos.cubeRegistry.Get(cubeID)
	if err != nil {
		return fmt.Errorf("failed to get memory cube: %w", err)
	}

	// Add memory based on request type
	if request.Messages != nil && mos.config.EnableTextualMemory && cube.GetTextualMemory() != nil {
		// Convert []map[string]interface{} to types.MessageList
		messageList := make(types.MessageList, len(request.Messages))
		for i, msg := range request.Messages {
			if role, ok := msg["role"].(string); ok {
				if content, ok := msg["content"].(string); ok {
					messageList[i] = types.MessageDict{
						Role:    types.MessageRole(role),
						Content: content,
					}
				}
			}
		}
		if err := mos.addMessagesToTextualMemory(ctx, cube.GetTextualMemory(), messageList, targetUserID); err != nil {
			return fmt.Errorf("failed to add messages to textual memory: %w", err)
		}
	}

	if request.MemoryContent != nil && mos.config.EnableTextualMemory && cube.GetTextualMemory() != nil {
		if err := mos.addContentToTextualMemory(ctx, cube.GetTextualMemory(), *request.MemoryContent, targetUserID); err != nil {
			return fmt.Errorf("failed to add content to textual memory: %w", err)
		}
	}

	if request.DocPath != nil && mos.config.EnableTextualMemory && cube.GetTextualMemory() != nil {
		if err := mos.addDocumentToTextualMemory(ctx, cube.GetTextualMemory(), *request.DocPath, targetUserID); err != nil {
			return fmt.Errorf("failed to add document to textual memory: %w", err)
		}
	}

	mos.logger.Info("Added memory to cube", map[string]interface{}{
		"cube_id": cubeID,
		"user_id": targetUserID,
	})

	if mos.metrics != nil {
		mos.metrics.Counter("mos_add_memory_count", 1, map[string]string{
			"cube_id": cubeID,
			"user_id": targetUserID,
		})
	}

	return nil
}

// addMessagesToTextualMemory adds messages to textual memory
func (mos *MOSCore) addMessagesToTextualMemory(ctx context.Context, textMem interfaces.TextualMemory, messages types.MessageList, userID string) error {
	items := make([]*types.TextualMemoryItem, 0, len(messages))

	for _, msg := range messages {
		item := &types.TextualMemoryItem{
			ID:     uuid.New().String(),
			Memory: msg.Content,
			Metadata: map[string]interface{}{
				"user_id":    userID,
				"session_id": mos.sessionID,
				"source":     "conversation",
				"role":       string(msg.Role),
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		items = append(items, item)
	}

	return textMem.AddTextual(ctx, items)
}

// addContentToTextualMemory adds content to textual memory
func (mos *MOSCore) addContentToTextualMemory(ctx context.Context, textMem interfaces.TextualMemory, content, userID string) error {
	item := &types.TextualMemoryItem{
		ID:     uuid.New().String(),
		Memory: content,
		Metadata: map[string]interface{}{
			"user_id":    userID,
			"session_id": mos.sessionID,
			"source":     "manual",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return textMem.AddTextual(ctx, []*types.TextualMemoryItem{item})
}

// addDocumentToTextualMemory adds document content to textual memory
func (mos *MOSCore) addDocumentToTextualMemory(ctx context.Context, textMem interfaces.TextualMemory, docPath, userID string) error {
	// TODO: Implement document parsing and chunking
	// For now, just add the document path as a placeholder
	item := &types.TextualMemoryItem{
		ID:     uuid.New().String(),
		Memory: fmt.Sprintf("Document: %s", docPath),
		Metadata: map[string]interface{}{
			"user_id":    userID,
			"session_id": mos.sessionID,
			"source":     "document",
			"doc_path":   docPath,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return textMem.AddTextual(ctx, []*types.TextualMemoryItem{item})
}

// Get retrieves a specific memory item
func (mos *MOSCore) Get(ctx context.Context, cubeID, memoryID, userID string) (types.MemoryItem, error) {
	if !mos.initialized {
		return nil, errors.NewMemoryError("MOS core not initialized")
	}

	cube, err := mos.cubeRegistry.Get(cubeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get memory cube: %w", err)
	}

	// Try textual memory first
	if cube.GetTextualMemory() != nil {
		if item, err := cube.GetTextualMemory().Get(ctx, memoryID); err == nil {
			return item, nil
		}
	}

	// Try activation memory
	if cube.GetActivationMemory() != nil {
		if item, err := cube.GetActivationMemory().Get(ctx, memoryID); err == nil {
			return item, nil
		}
	}

	// Try parametric memory
	if cube.GetParametricMemory() != nil {
		if item, err := cube.GetParametricMemory().Get(ctx, memoryID); err == nil {
			return item, nil
		}
	}

	return nil, errors.NewMemoryNotFoundError(memoryID)
}

// GetAll retrieves all memories from a cube
func (mos *MOSCore) GetAll(ctx context.Context, cubeID, userID string) (*types.MOSSearchResult, error) {
	if !mos.initialized {
		return nil, errors.NewMemoryError("MOS core not initialized")
	}

	cube, err := mos.cubeRegistry.Get(cubeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get memory cube: %w", err)
	}

	result := &types.MOSSearchResult{
		TextMem: make([]types.CubeMemoryResult, 0),
		ActMem:  make([]types.CubeMemoryResult, 0),
		ParaMem: make([]types.CubeMemoryResult, 0),
	}

	// Get textual memories
	if mos.config.EnableTextualMemory && cube.GetTextualMemory() != nil {
		memories, err := cube.GetTextualMemory().GetAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get textual memories: %w", err)
		}
		if len(memories) > 0 {
			result.TextMem = append(result.TextMem, types.CubeMemoryResult{
				CubeID:   cubeID,
				Memories: memories,
			})
		}
	}

	// Get activation memories
	if mos.config.EnableActivationMemory && cube.GetActivationMemory() != nil {
		memories, err := cube.GetActivationMemory().GetAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get activation memories: %w", err)
		}
		if len(memories) > 0 {
			result.ActMem = append(result.ActMem, types.CubeMemoryResult{
				CubeID:   cubeID,
				Memories: memories,
			})
		}
	}

	// Get parametric memories
	if mos.config.EnableParametricMemory && cube.GetParametricMemory() != nil {
		memories, err := cube.GetParametricMemory().GetAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get parametric memories: %w", err)
		}
		if len(memories) > 0 {
			result.ParaMem = append(result.ParaMem, types.CubeMemoryResult{
				CubeID:   cubeID,
				Memories: memories,
			})
		}
	}

	return result, nil
}

// Update updates a memory item
func (mos *MOSCore) Update(ctx context.Context, cubeID, memoryID, userID string, memory types.MemoryItem) error {
	if !mos.initialized {
		return errors.NewMemoryError("MOS core not initialized")
	}

	cube, err := mos.cubeRegistry.Get(cubeID)
	if err != nil {
		return fmt.Errorf("failed to get memory cube: %w", err)
	}

	// Try to update in the appropriate memory type
	switch mem := memory.(type) {
	case *types.TextualMemoryItem:
		if cube.GetTextualMemory() != nil {
			return cube.GetTextualMemory().Update(ctx, memoryID, mem)
		}
	case *types.ActivationMemoryItem:
		if cube.GetActivationMemory() != nil {
			return cube.GetActivationMemory().Update(ctx, memoryID, mem)
		}
	case *types.ParametricMemoryItem:
		if cube.GetParametricMemory() != nil {
			return cube.GetParametricMemory().Update(ctx, memoryID, mem)
		}
	}

	return errors.NewValidationError("unsupported memory type or memory backend not available")
}

// Delete deletes a memory item
func (mos *MOSCore) Delete(ctx context.Context, cubeID, memoryID, userID string) error {
	if !mos.initialized {
		return errors.NewMemoryError("MOS core not initialized")
	}

	cube, err := mos.cubeRegistry.Get(cubeID)
	if err != nil {
		return fmt.Errorf("failed to get memory cube: %w", err)
	}

	// Try to delete from all memory types
	deleted := false

	if cube.GetTextualMemory() != nil {
		if err := cube.GetTextualMemory().Delete(ctx, memoryID); err == nil {
			deleted = true
		}
	}

	if !deleted && cube.GetActivationMemory() != nil {
		if err := cube.GetActivationMemory().Delete(ctx, memoryID); err == nil {
			deleted = true
		}
	}

	if !deleted && cube.GetParametricMemory() != nil {
		if err := cube.GetParametricMemory().Delete(ctx, memoryID); err == nil {
			deleted = true
		}
	}

	if !deleted {
		return errors.NewMemoryNotFoundError(memoryID)
	}

	return nil
}

// DeleteAll deletes all memories from a cube
func (mos *MOSCore) DeleteAll(ctx context.Context, cubeID, userID string) error {
	if !mos.initialized {
		return errors.NewMemoryError("MOS core not initialized")
	}

	cube, err := mos.cubeRegistry.Get(cubeID)
	if err != nil {
		return fmt.Errorf("failed to get memory cube: %w", err)
	}

	// Delete from all memory types
	if cube.GetTextualMemory() != nil {
		if err := cube.GetTextualMemory().DeleteAll(ctx); err != nil {
			return fmt.Errorf("failed to delete textual memories: %w", err)
		}
	}

	if cube.GetActivationMemory() != nil {
		if err := cube.GetActivationMemory().DeleteAll(ctx); err != nil {
			return fmt.Errorf("failed to delete activation memories: %w", err)
		}
	}

	if cube.GetParametricMemory() != nil {
		if err := cube.GetParametricMemory().DeleteAll(ctx); err != nil {
			return fmt.Errorf("failed to delete parametric memories: %w", err)
		}
	}

	return nil
}

// Chat processes a chat request with memory integration
func (mos *MOSCore) Chat(ctx context.Context, request *types.ChatRequest) (*types.ChatResponse, error) {
	if !mos.initialized {
		return nil, errors.NewMemoryError("MOS core not initialized")
	}

	start := time.Now()
	defer func() {
		if mos.metrics != nil {
			mos.metrics.Timer("mos_chat_duration", float64(time.Since(start).Milliseconds()), nil)
		}
	}()

	targetUserID := mos.userID
	if request.UserID != "" {
		targetUserID = request.UserID
	}

	// Validate user access
	if mos.userManager != nil {
		valid, err := mos.userManager.ValidateUser(ctx, targetUserID)
		if err != nil {
			return nil, fmt.Errorf("failed to validate user: %w", err)
		}
		if !valid {
			return nil, errors.NewUnauthorizedError(fmt.Sprintf("user '%s' does not exist or is inactive", targetUserID))
		}
	}

	// Get accessible cubes for user
	accessibleCubes, err := mos.getAccessibleCubes(ctx, targetUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get accessible cubes: %w", err)
	}

	// Initialize chat history if needed
	chatHistory, err := mos.initializeChatHistoryIfNeeded(ctx, targetUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize chat history: %w", err)
	}

	// Build context with memories if textual memory is enabled
	var memories []types.MemoryItem
	if mos.config.EnableTextualMemory {
		memories, err = mos.searchMemoriesForChat(ctx, request.Query, accessibleCubes)
		if err != nil {
			mos.logger.Warn("Failed to search memories for chat", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Build system prompt with memories
	systemPrompt := mos.buildSystemPrompt(memories)

	// Prepare messages for LLM
	currentMessages := []types.MessageDict{
		{Role: types.MessageRoleSystem, Content: systemPrompt},
	}

	// Add chat history
	if chatHistory != nil {
		currentMessages = append(currentMessages, chatHistory.ChatHistory...)
	}

	// Add current query
	currentMessages = append(currentMessages, types.MessageDict{
		Role:    types.MessageRoleUser,
		Content: request.Query,
	})

	// Generate response using LLM
	response, err := mos.generateChatResponse(ctx, currentMessages, accessibleCubes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate chat response: %w", err)
	}

	// Update chat history
	err = mos.updateChatHistory(ctx, targetUserID, request.Query, response)
	if err != nil {
		mos.logger.Warn("Failed to update chat history", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Submit to scheduler if enabled
	mos.submitToSchedulerIfEnabled(ctx, targetUserID, accessibleCubes, response)

	chatResponse := &types.ChatResponse{
		Response:  response,
		SessionID: mos.sessionID,
		UserID:    targetUserID,
		Timestamp: time.Now(),
	}

	mos.logger.Info("Chat response generated", map[string]interface{}{
		"user_id":      targetUserID,
		"query_len":    len(request.Query),
		"response_len": len(response),
	})

	if mos.metrics != nil {
		mos.metrics.Counter("mos_chat_count", 1, map[string]string{
			"user_id": targetUserID,
		})
	}

	return chatResponse, nil
}

// CreateUser creates a new user
func (mos *MOSCore) CreateUser(ctx context.Context, userName string, role types.UserRole, userID string) (string, error) {
	if !mos.initialized {
		return "", errors.NewMemoryError("MOS core not initialized")
	}

	if mos.userManager != nil {
		return mos.userManager.CreateUser(ctx, userName, role, userID)
	}

	return "", errors.NewInternalError("user manager not available")
}

// GetUserInfo retrieves user information
func (mos *MOSCore) GetUserInfo(ctx context.Context, userID string) (map[string]interface{}, error) {
	if !mos.initialized {
		return nil, errors.NewMemoryError("MOS core not initialized")
	}

	targetUserID := userID
	if targetUserID == "" {
		targetUserID = mos.userID
	}

	if mos.userManager != nil {
		user, err := mos.userManager.GetUser(ctx, targetUserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}

		cubes, err := mos.userManager.GetUserCubes(ctx, targetUserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user cubes: %w", err)
		}

		cubeInfos := make([]map[string]interface{}, len(cubes))
		for i, cube := range cubes {
			cubeInfos[i] = map[string]interface{}{
				"cube_id":   cube.ID,
				"cube_name": cube.Name,
				"cube_path": cube.Path,
				"owner_id":  cube.OwnerID,
				"is_loaded": true, // TODO: Check if cube is actually loaded
			}
		}

		return map[string]interface{}{
			"user_id":          user.UserID,
			"user_name":        user.UserName,
			"role":             string(user.Role),
			"created_at":       user.CreatedAt,
			"accessible_cubes": cubeInfos,
		}, nil
	}

	return map[string]interface{}{
		"user_id": targetUserID,
		"message": "user manager not available",
	}, nil
}

// ListUsers lists all users
func (mos *MOSCore) ListUsers(ctx context.Context) ([]*types.User, error) {
	if !mos.initialized {
		return nil, errors.NewMemoryError("MOS core not initialized")
	}

	if mos.userManager != nil {
		return mos.userManager.ListUsers(ctx)
	}

	return nil, errors.NewInternalError("user manager not available")
}

// ShareCubeWithUser shares a cube with another user
func (mos *MOSCore) ShareCubeWithUser(ctx context.Context, cubeID, targetUserID string) (bool, error) {
	if !mos.initialized {
		return false, errors.NewMemoryError("MOS core not initialized")
	}

	if mos.userManager != nil {
		err := mos.userManager.ShareCube(ctx, cubeID, mos.userID, targetUserID)
		if err != nil {
			return false, fmt.Errorf("failed to share cube: %w", err)
		}
		return true, nil
	}

	return false, errors.NewInternalError("user manager not available")
}

// Close closes the MOS core
func (mos *MOSCore) Close() error {
	mos.mu.Lock()
	defer mos.mu.Unlock()

	if mos.closed {
		return nil
	}

	var errs []error

	// Close scheduler
	if mos.scheduler != nil {
		if err := mos.scheduler.Stop(context.Background()); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop scheduler: %w", err))
		}
	}

	// Close cube registry
	if err := mos.cubeRegistry.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close cube registry: %w", err))
	}

	// Close LLM
	if mos.chatLLM != nil {
		if err := mos.chatLLM.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close LLM: %w", err))
		}
	}

	mos.closed = true

	mos.logger.Info("MOS Core closed")

	if len(errs) > 0 {
		return fmt.Errorf("errors occurred while closing MOS core: %v", errs)
	}

	return nil
}

// Helper methods

// getAccessibleCubes gets cubes accessible by a user
func (mos *MOSCore) getAccessibleCubes(ctx context.Context, userID string) ([]*types.MemCube, error) {
	if mos.userManager == nil {
		return []*types.MemCube{}, nil
	}
	return mos.userManager.GetUserCubes(ctx, userID)
}

// initializeChatHistoryIfNeeded initializes chat history if needed
func (mos *MOSCore) initializeChatHistoryIfNeeded(ctx context.Context, userID string) (*types.ChatHistory, error) {
	if mos.chatManager == nil {
		return nil, nil
	}
	return mos.chatManager.GetChatHistory(ctx, userID, mos.sessionID)
}

// searchMemoriesForChat searches memories across all accessible cubes for chat context
func (mos *MOSCore) searchMemoriesForChat(ctx context.Context, query string, accessibleCubes []*types.MemCube) ([]types.MemoryItem, error) {
	var allMemories []types.MemoryItem

	for _, cubeInfo := range accessibleCubes {
		cube, err := mos.cubeRegistry.Get(cubeInfo.ID)
		if err != nil {
			mos.logger.Warn("Failed to get cube for chat search", map[string]interface{}{
				"cube_id": cubeInfo.ID,
				"error":   err.Error(),
			})
			continue
		}

		if cube.GetTextualMemory() != nil {
			memories, err := cube.GetTextualMemory().Search(ctx, query, mos.config.TopK)
			if err != nil {
				mos.logger.Warn("Failed to search textual memory", map[string]interface{}{
					"cube_id": cubeInfo.ID,
					"error":   err.Error(),
				})
				continue
			}
			allMemories = append(allMemories, memories...)
		}
	}

	return allMemories, nil
}

// buildSystemPrompt builds the system prompt with optional memories
func (mos *MOSCore) buildSystemPrompt(memories []types.MemoryItem) string {
	basePrompt := "You are a knowledgeable and helpful AI assistant. " +
		"You have access to conversation memories that help you provide more personalized responses. " +
		"Use the memories to understand the user's context, preferences, and past interactions. " +
		"If memories are provided, reference them naturally when relevant, but don't explicitly mention having memories."

	if len(memories) == 0 {
		return basePrompt
	}

	memoryContext := "\n\n## Memories:\n"
	for i, memory := range memories {
		if i >= mos.config.TopK {
			break
		}
		memoryContext += fmt.Sprintf("%d. %s\n", i+1, memory.GetContent())
	}

	return basePrompt + memoryContext
}

// generateChatResponse generates response using LLM
func (mos *MOSCore) generateChatResponse(ctx context.Context, messages []types.MessageDict, accessibleCubes []*types.MemCube) (string, error) {
	if mos.chatLLM == nil {
		return "Chat LLM not available", nil
	}

	// Convert to MessageList
	messageList := types.MessageList(messages)

	// Handle activation memory if enabled
	if mos.config.EnableActivationMemory {
		for _, cubeInfo := range accessibleCubes {
			cube, err := mos.cubeRegistry.Get(cubeInfo.ID)
			if err != nil {
				continue
			}

			if cube.GetActivationMemory() != nil {
				// Try to get cached activations
				// This would involve KV cache integration
				break
			}
		}
	}

	return mos.chatLLM.Generate(ctx, messageList)
}

// updateChatHistory updates the chat history with new query and response
func (mos *MOSCore) updateChatHistory(ctx context.Context, userID, query, response string) error {
	if mos.chatManager == nil {
		return nil
	}

	chatHistory, err := mos.chatManager.GetChatHistory(ctx, userID, mos.sessionID)
	if err != nil {
		return err
	}

	if chatHistory == nil {
		chatHistory = &types.ChatHistory{
			UserID:        userID,
			SessionID:     mos.sessionID,
			CreatedAt:     time.Now(),
			TotalMessages: 0,
			ChatHistory:   []types.MessageDict{},
		}
	}

	chatHistory.ChatHistory = append(chatHistory.ChatHistory,
		types.MessageDict{Role: types.MessageRoleUser, Content: query},
		types.MessageDict{Role: types.MessageRoleAssistant, Content: response},
	)
	chatHistory.TotalMessages = len(chatHistory.ChatHistory)

	return mos.chatManager.SaveChatHistory(ctx, chatHistory)
}

// submitToSchedulerIfEnabled submits message to scheduler if enabled
func (mos *MOSCore) submitToSchedulerIfEnabled(ctx context.Context, userID string, accessibleCubes []*types.MemCube, response string) {
	if !mos.config.EnableMemScheduler || mos.scheduler == nil || len(accessibleCubes) != 1 {
		return
	}

	// Create scheduled task for response
	task := &types.ScheduledTask{
		ID:        uuid.New().String(),
		UserID:    userID,
		CubeID:    accessibleCubes[0].ID,
		TaskType:  "chat_response",
		Priority:  types.TaskPriorityMedium,
		Payload:   map[string]interface{}{"response": response},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    "pending",
	}

	if err := mos.scheduler.Schedule(ctx, task); err != nil {
		mos.logger.Warn("Failed to submit to scheduler", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// isLocalPath checks if a path is a local file system path
func (mos *MOSCore) isLocalPath(path string) bool {
	// Simple heuristic: if it doesn't start with http/https, consider it local
	return !strings.HasPrefix(strings.ToLower(path), "http://") &&
		!strings.HasPrefix(strings.ToLower(path), "https://")
}
