// Package chat - SimpleMemChat implementation
package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/llm"
	"github.com/memtensor/memgos/pkg/logger"
	"github.com/memtensor/memgos/pkg/types"
)

// SimpleMemChat implements a basic memory-augmented chat system
type SimpleMemChat struct {
	// Configuration
	config *MemChatConfig
	logger interfaces.Logger
	
	// Core components
	chatLLM  interfaces.LLM
	memCube  interfaces.MemCube
	
	// Session state
	context     *ChatContext
	metrics     *ChatMetrics
	mode        ConversationMode
	isRunning   bool
	
	// Memory management
	memoryExtractor MemoryExtractor
	memoryMutex     sync.RWMutex
	
	// Control
	stopChan chan struct{}
	mutex    sync.RWMutex
}

// MemoryExtractor handles extraction of memories from conversations
type MemoryExtractor struct {
	config           *MemoryExtractionConfig
	extractionRules  []ExtractionRule
	summaryGenerator SummaryGenerator
}

// ExtractionRule defines rules for extracting memories
type ExtractionRule struct {
	Name        string   `json:"name"`
	Keywords    []string `json:"keywords"`
	Patterns    []string `json:"patterns"`
	MinLength   int      `json:"min_length"`
	MaxLength   int      `json:"max_length"`
	Importance  float64  `json:"importance"`
	Category    string   `json:"category"`
	Enabled     bool     `json:"enabled"`
}

// SummaryGenerator generates summaries for memory storage
type SummaryGenerator struct {
	llm         interfaces.LLM
	maxLength   int
	temperature float64
}

// NewSimpleMemChat creates a new SimpleMemChat instance
func NewSimpleMemChat(config *MemChatConfig) (*SimpleMemChat, error) {
	if err := ValidateMemChatConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	
	// Create logger
	chatLogger := logger.NewLogger()
	
	// Create LLM
	llmConfig := &llm.LLMConfig{
		Provider:    config.ChatLLM.Provider,
		Model:       config.ChatLLM.Model,
		APIKey:      config.ChatLLM.APIKey,
		BaseURL:     config.ChatLLM.BaseURL,
		MaxTokens:   config.ChatLLM.MaxTokens,
		Temperature: config.ChatLLM.Temperature,
		TopP:        config.ChatLLM.TopP,
		Timeout:     config.ChatLLM.Timeout,
		Extra:       config.ChatLLM.Extra,
	}
	
	chatLLM, err := llm.NewFromConfig(llmConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM: %w", err)
	}
	
	// Create memory extractor
	extractorConfig := &MemoryExtractionConfig{
		ExtractionMode:       "automatic",
		KeywordFilters:      []string{"important", "remember", "note", "key", "fact"},
		MinimumLength:       10,
		MaximumLength:       500,
		ImportanceScore:     0.5,
		EnableSummarization: true,
		EnableCategorization: true,
	}
	
	extractor := &MemoryExtractor{
		config: extractorConfig,
		extractionRules: []ExtractionRule{
			{
				Name:       "important_facts",
				Keywords:   []string{"important", "key", "fact", "remember"},
				Patterns:   []string{".*important.*", ".*remember.*", ".*key.*"},
				MinLength:  10,
				MaxLength:  200,
				Importance: 0.8,
				Category:   "fact",
				Enabled:    true,
			},
			{
				Name:       "preferences",
				Keywords:   []string{"prefer", "like", "dislike", "favorite", "hate"},
				Patterns:   []string{".*prefer.*", ".*like.*", ".*favorite.*"},
				MinLength:  5,
				MaxLength:  100,
				Importance: 0.7,
				Category:   "preference",
				Enabled:    true,
			},
			{
				Name:       "personal_info",
				Keywords:   []string{"name", "age", "work", "job", "location", "family"},
				Patterns:   []string{".*name.*", ".*work.*", ".*job.*"},
				MinLength:  5,
				MaxLength:  150,
				Importance: 0.9,
				Category:   "personal",
				Enabled:    true,
			},
		},
		summaryGenerator: SummaryGenerator{
			llm:         chatLLM,
			maxLength:   100,
			temperature: 0.3,
		},
	}
	
	chat := &SimpleMemChat{
		config:          config,
		logger:          chatLogger,
		chatLLM:         chatLLM,
		context:         NewChatContext(config.UserID, config.SessionID, ConversationModeDefault),
		metrics:         NewChatMetrics(),
		mode:           ConversationModeDefault,
		isRunning:      false,
		memoryExtractor: *extractor,
		stopChan:       make(chan struct{}),
	}
	
	return chat, nil
}

// Chat processes a single chat query and returns the response
func (s *SimpleMemChat) Chat(ctx context.Context, query string) (*ChatResult, error) {
	start := time.Now()
	
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Check for special commands
	if isCommand, result := s.handleCommand(ctx, query); isCommand {
		return result, nil
	}
	
	// Retrieve relevant memories
	var memories []*types.TextualMemoryItem
	var err error
	
	if s.config.EnableTextualMemory && s.memCube != nil {
		memories, err = s.RetrieveMemories(ctx, query, s.config.TopK)
		if err != nil {
			s.logger.Warn("Failed to retrieve memories", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
	
	// Build system prompt with memories
	systemPrompt := s.buildSystemPrompt(memories)
	
	// Prepare messages for LLM
	currentMessages := types.MessageList{
		{Role: types.MessageRoleSystem, Content: systemPrompt},
	}
	
	// Add recent conversation history
	if len(s.context.Messages) > 0 {
		startIdx := len(s.context.Messages) - s.config.MaxTurnsWindow*2
		if startIdx < 0 {
			startIdx = 0
		}
		currentMessages = append(currentMessages, s.context.Messages[startIdx:]...)
	}
	
	// Add current user query
	currentMessages = append(currentMessages, types.MessageDict{
		Role:    types.MessageRoleUser,
		Content: query,
	})
	
	// Generate response
	response, err := s.chatLLM.Generate(ctx, currentMessages)
	if err != nil {
		s.metrics.ErrorCount++
		return nil, NewChatErrorWithCause("llm_generation_failed", "Failed to generate response", err)
	}
	
	// Update conversation context
	s.context.Messages = append(s.context.Messages, 
		types.MessageDict{Role: types.MessageRoleUser, Content: query},
		types.MessageDict{Role: types.MessageRoleAssistant, Content: response},
	)
	
	// Apply context window limit
	if len(s.context.Messages) > s.config.MaxTurnsWindow*2 {
		s.context.Messages = s.context.Messages[len(s.context.Messages)-s.config.MaxTurnsWindow*2:]
	}
	
	// Extract and store new memories
	var newMemories []*types.TextualMemoryItem
	if s.config.EnableTextualMemory && s.memCube != nil {
		lastTwoMessages := s.context.Messages[len(s.context.Messages)-2:]
		newMemories, err = s.ExtractMemories(ctx, lastTwoMessages)
		if err != nil {
			s.logger.Warn("Failed to extract memories", map[string]interface{}{
				"error": err.Error(),
			})
		} else if len(newMemories) > 0 {
			err = s.StoreMemories(ctx, newMemories)
			if err != nil {
				s.logger.Warn("Failed to store memories", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}
	}
	
	// Generate follow-up questions
	followUpQuestions := s.generateFollowUpQuestions(response, query)
	
	// Update metrics
	duration := time.Since(start)
	s.updateMetrics(len(memories), len(newMemories), duration)
	
	// Build result
	result := &ChatResult{
		Response:          response,
		Memories:          memories,
		NewMemories:       newMemories,
		Context:           s.context,
		Metrics:           s.metrics,
		IsCommand:         false,
		FollowUpQuestions: followUpQuestions,
		TokensUsed:        s.estimateTokens(currentMessages, response),
		Duration:          duration,
		Timestamp:         time.Now(),
	}
	
	return result, nil
}

// ChatStream implements streaming chat with callback
func (s *SimpleMemChat) ChatStream(ctx context.Context, query string, callback StreamingCallback) (*ChatResult, error) {
	// For now, fall back to regular chat
	// TODO: Implement actual streaming when LLM supports it
	result, err := s.Chat(ctx, query)
	if err != nil {
		return nil, err
	}
	
	// Simulate streaming by calling callback with complete response
	err = callback(result.Response, true, map[string]interface{}{
		"tokens_used": result.TokensUsed,
		"duration":    result.Duration,
	})
	if err != nil {
		return nil, NewChatErrorWithCause("streaming_callback_failed", "Streaming callback failed", err)
	}
	
	return result, nil
}

// Run starts the interactive chat mode
func (s *SimpleMemChat) Run(ctx context.Context) error {
	s.mutex.Lock()
	s.isRunning = true
	s.mutex.Unlock()
	
	defer func() {
		s.mutex.Lock()
		s.isRunning = false
		s.mutex.Unlock()
	}()
	
	s.logger.Info("🎯 SimpleMemChat is starting...")
	fmt.Println("\n📢 [System] Simple MemChat is running.")
	fmt.Println("Commands: 'bye' to quit, 'clear' to clear chat history, 'mem' to show all memories, 'export' to export chat history")
	fmt.Println("Commands: 'help' for help, 'stats' for statistics, 'mode <mode>' to change conversation mode\n")
	
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Context cancelled, stopping chat")
			return ctx.Err()
		case <-s.stopChan:
			s.logger.Info("Stop signal received, stopping chat")
			return nil
		default:
			// Get user input
			fmt.Print("👤 [You] ")
			var userInput string
			if _, err := fmt.Scanln(&userInput); err != nil {
				continue
			}
			
			userInput = strings.TrimSpace(userInput)
			fmt.Println()
			
			if userInput == "" {
				continue
			}
			
			// Process chat
			result, err := s.Chat(ctx, userInput)
			if err != nil {
				fmt.Printf("❌ [Error] %s\n\n", err.Error())
				continue
			}
			
			// Handle command results
			if result.IsCommand {
				if result.CommandResult != nil {
					switch cmd := result.CommandResult.(type) {
					case string:
						if cmd == "bye" {
							fmt.Println("📢 [System] MemChat has stopped.")
							return nil
						}
						fmt.Printf("📢 [System] %s\n\n", cmd)
					case map[string]interface{}:
						if jsonData, err := json.MarshalIndent(cmd, "", "  "); err == nil {
							fmt.Printf("📊 [Stats]\n%s\n\n", string(jsonData))
						}
					}
				}
				continue
			}
			
			// Display memories retrieved
			if len(result.Memories) > 0 && s.config.EnableTextualMemory {
				fmt.Printf("🧠 [Memory] Retrieved %d memories:\n", len(result.Memories))
				for i, memory := range result.Memories {
					fmt.Printf("  %d. %s\n", i+1, s.truncateText(memory.Memory, 100))
				}
				fmt.Println()
			}
			
			// Display response
			fmt.Printf("🤖 [Assistant] %s\n", result.Response)
			
			// Display new memories stored
			if len(result.NewMemories) > 0 && s.config.EnableTextualMemory {
				fmt.Printf("\n🧠 [Memory] Stored %d new memory(ies):\n", len(result.NewMemories))
				for i, memory := range result.NewMemories {
					fmt.Printf("  %d. %s\n", i+1, s.truncateText(memory.Memory, 100))
				}
			}
			
			// Display follow-up questions
			if len(result.FollowUpQuestions) > 0 {
				fmt.Println("\n💡 [Follow-up Questions]")
				for i, question := range result.FollowUpQuestions {
					fmt.Printf("  %d. %s\n", i+1, question)
				}
			}
			
			fmt.Println()
		}
	}
}

// Memory-related implementations
func (s *SimpleMemChat) GetMemCube() interfaces.MemCube {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.memCube
}

func (s *SimpleMemChat) SetMemCube(cube interfaces.MemCube) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.memCube = cube
	return nil
}

func (s *SimpleMemChat) RetrieveMemories(ctx context.Context, query string, topK int) ([]*types.TextualMemoryItem, error) {
	if s.memCube == nil {
		return nil, NewChatError("no_memory_cube", "No memory cube configured")
	}
	
	textMem := s.memCube.GetTextualMemory()
	if textMem == nil {
		return nil, NewChatError("no_textual_memory", "No textual memory available")
	}
	
	memories, err := textMem.SearchSemantic(ctx, query, topK, nil)
	if err != nil {
		return nil, NewChatErrorWithCause("memory_search_failed", "Failed to search memories", err)
	}
	
	s.metrics.MemoriesRetrieved += len(memories)
	return memories, nil
}

func (s *SimpleMemChat) StoreMemories(ctx context.Context, memories []*types.TextualMemoryItem) error {
	if s.memCube == nil {
		return NewChatError("no_memory_cube", "No memory cube configured")
	}
	
	textMem := s.memCube.GetTextualMemory()
	if textMem == nil {
		return NewChatError("no_textual_memory", "No textual memory available")
	}
	
	// Add metadata
	for _, memory := range memories {
		if memory.Metadata == nil {
			memory.Metadata = make(map[string]interface{})
		}
		memory.Metadata["user_id"] = s.config.UserID
		memory.Metadata["session_id"] = s.config.SessionID
		memory.Metadata["status"] = "activated"
		memory.Metadata["extracted_at"] = time.Now()
	}
	
	err := textMem.AddTextual(ctx, memories)
	if err != nil {
		return NewChatErrorWithCause("memory_storage_failed", "Failed to store memories", err)
	}
	
	s.metrics.MemoriesStored += len(memories)
	return nil
}

func (s *SimpleMemChat) ExtractMemories(ctx context.Context, messages types.MessageList) ([]*types.TextualMemoryItem, error) {
	var extractedMemories []*types.TextualMemoryItem
	
	for _, message := range messages {
		if message.Role == types.MessageRoleUser || message.Role == types.MessageRoleAssistant {
			memories := s.memoryExtractor.extractFromText(message.Content)
			extractedMemories = append(extractedMemories, memories...)
		}
	}
	
	return extractedMemories, nil
}

// Session and context management
func (s *SimpleMemChat) GetSessionID() string {
	return s.config.SessionID
}

func (s *SimpleMemChat) GetUserID() string {
	return s.config.UserID
}

func (s *SimpleMemChat) GetChatHistory(ctx context.Context) (*types.ChatHistory, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	return &types.ChatHistory{
		UserID:        s.config.UserID,
		SessionID:     s.config.SessionID,
		CreatedAt:     s.config.CreatedAt,
		TotalMessages: len(s.context.Messages),
		ChatHistory:   s.context.Messages,
	}, nil
}

func (s *SimpleMemChat) ClearChatHistory(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.context.Messages = make(types.MessageList, 0)
	s.context.TurnCount = 0
	s.context.TokensUsed = 0
	s.context.LastUpdated = time.Now()
	
	return nil
}

func (s *SimpleMemChat) SaveChatHistory(ctx context.Context) error {
	// TODO: Implement chat history persistence
	return nil
}

func (s *SimpleMemChat) ExportChatHistory(ctx context.Context, outputPath string) (string, error) {
	history, err := s.GetChatHistory(ctx)
	if err != nil {
		return "", err
	}
	
	if outputPath == "" {
		outputPath = "chat_exports"
	}
	
	// Create output directory
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return "", NewChatErrorWithCause("export_failed", "Failed to create export directory", err)
	}
	
	// Generate filename
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s_chat_history.json", s.config.UserID, timestamp)
	filepath := filepath.Join(outputPath, filename)
	
	// Write to file
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return "", NewChatErrorWithCause("export_failed", "Failed to marshal chat history", err)
	}
	
	err = os.WriteFile(filepath, data, 0644)
	if err != nil {
		return "", NewChatErrorWithCause("export_failed", "Failed to write chat history file", err)
	}
	
	s.logger.Info("Chat history exported", map[string]interface{}{
		"file_path": filepath,
		"messages":  len(history.ChatHistory),
	})
	
	return filepath, nil
}

// Mode and configuration
func (s *SimpleMemChat) GetMode() ConversationMode {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.mode
}

func (s *SimpleMemChat) SetMode(mode ConversationMode) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.mode = mode
	s.context.Mode = mode
	
	// Update system prompt based on mode
	s.config.SystemPrompt = s.getSystemPromptForMode(mode)
	
	return nil
}

func (s *SimpleMemChat) GetConfig() *MemChatConfig {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.config
}

func (s *SimpleMemChat) UpdateConfig(config *MemChatConfig) error {
	if err := ValidateMemChatConfig(config); err != nil {
		return err
	}
	
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.config = config
	
	return nil
}

// Metrics and monitoring
func (s *SimpleMemChat) GetMetrics(ctx context.Context) (*ChatMetrics, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	// Update session duration
	s.metrics.SessionDuration = time.Since(s.config.CreatedAt)
	s.metrics.LastActive = time.Now()
	
	return s.metrics, nil
}

func (s *SimpleMemChat) ResetMetrics(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.metrics = NewChatMetrics()
	return nil
}

// Control and lifecycle
func (s *SimpleMemChat) Stop(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if s.isRunning {
		close(s.stopChan)
		s.stopChan = make(chan struct{})
	}
	
	return nil
}

func (s *SimpleMemChat) IsRunning() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.isRunning
}

func (s *SimpleMemChat) HealthCheck(ctx context.Context) error {
	// Check LLM health
	if s.chatLLM == nil {
		return NewChatError("llm_not_initialized", "LLM is not initialized")
	}
	
	// Check memory cube if enabled
	if s.config.EnableTextualMemory && s.memCube == nil {
		return NewChatError("memory_cube_not_set", "Memory cube is not set but textual memory is enabled")
	}
	
	return nil
}

func (s *SimpleMemChat) Close() error {
	if err := s.Stop(context.Background()); err != nil {
		return err
	}
	
	if s.chatLLM != nil {
		return s.chatLLM.Close()
	}
	
	return nil
}

// Context management
func (s *SimpleMemChat) GetContext(ctx context.Context) (*ChatContext, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.context, nil
}

func (s *SimpleMemChat) UpdateContext(ctx context.Context, context *ChatContext) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.context = context
	return nil
}

func (s *SimpleMemChat) GetContextWindow() int {
	return s.config.MaxTurnsWindow
}

func (s *SimpleMemChat) SetContextWindow(window int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.config.MaxTurnsWindow = window
}

// Helper methods
func (s *SimpleMemChat) handleCommand(ctx context.Context, input string) (bool, *ChatResult) {
	input = strings.ToLower(strings.TrimSpace(input))
	
	switch {
	case input == "bye":
		return true, &ChatResult{
			Response:      "bye",
			IsCommand:     true,
			CommandResult: "bye",
			Timestamp:     time.Now(),
		}
		
	case input == "clear":
		s.ClearChatHistory(ctx)
		return true, &ChatResult{
			Response:      "Chat history cleared.",
			IsCommand:     true,
			CommandResult: "Chat history cleared.",
			Timestamp:     time.Now(),
		}
		
	case input == "mem":
		if !s.config.EnableTextualMemory || s.memCube == nil {
			return true, &ChatResult{
				Response:      "Textual memory is not enabled.",
				IsCommand:     true,
				CommandResult: "Textual memory is not enabled.",
				Timestamp:     time.Now(),
			}
		}
		
		textMem := s.memCube.GetTextualMemory()
		memories, err := textMem.GetAll(ctx)
		if err != nil || len(memories) == 0 {
			return true, &ChatResult{
				Response:      "No memories found.",
				IsCommand:     true,
				CommandResult: "No memories found.",
				Timestamp:     time.Now(),
			}
		}
		
		result := fmt.Sprintf("Found %d memories:\n", len(memories))
		for i, mem := range memories {
			if textMem, ok := mem.(*types.TextualMemoryItem); ok {
				result += fmt.Sprintf("%d. %s\n", i+1, s.truncateText(textMem.Memory, 100))
			}
		}
		
		return true, &ChatResult{
			Response:      result,
			IsCommand:     true,
			CommandResult: result,
			Timestamp:     time.Now(),
		}
		
	case input == "export":
		filepath, err := s.ExportChatHistory(ctx, "")
		var result string
		if err != nil {
			result = fmt.Sprintf("Export failed: %s", err.Error())
		} else {
			result = fmt.Sprintf("Chat history exported to: %s", filepath)
		}
		
		return true, &ChatResult{
			Response:      result,
			IsCommand:     true,
			CommandResult: result,
			Timestamp:     time.Now(),
		}
		
	case input == "stats":
		metrics, _ := s.GetMetrics(ctx)
		return true, &ChatResult{
			Response:      "Statistics retrieved.",
			IsCommand:     true,
			CommandResult: metrics,
			Timestamp:     time.Now(),
		}
		
	case input == "help":
		help := `Available commands:
• bye - Exit the chat
• clear - Clear chat history
• mem - Show all stored memories
• export - Export chat history to JSON
• stats - Show chat statistics
• mode <mode> - Change conversation mode (default, analytical, creative, helpful, explainer, summarizer)
• help - Show this help message`
		
		return true, &ChatResult{
			Response:      help,
			IsCommand:     true,
			CommandResult: help,
			Timestamp:     time.Now(),
		}
		
	case strings.HasPrefix(input, "mode "):
		mode := strings.TrimSpace(strings.TrimPrefix(input, "mode "))
		switch ConversationMode(mode) {
		case ConversationModeDefault, ConversationModeAnalytical, ConversationModeCreative,
			 ConversationModeHelpful, ConversationModeExplainer, ConversationModeSummarizer:
			s.SetMode(ConversationMode(mode))
			result := fmt.Sprintf("Conversation mode changed to: %s", mode)
			return true, &ChatResult{
				Response:      result,
				IsCommand:     true,
				CommandResult: result,
				Timestamp:     time.Now(),
			}
		default:
			result := "Invalid mode. Available modes: default, analytical, creative, helpful, explainer, summarizer"
			return true, &ChatResult{
				Response:      result,
				IsCommand:     true,
				CommandResult: result,
				Timestamp:     time.Now(),
			}
		}
	}
	
	return false, nil
}

func (s *SimpleMemChat) buildSystemPrompt(memories []*types.TextualMemoryItem) string {
	basePrompt := s.config.SystemPrompt
	
	// Add mode-specific instructions
	modePrompt := s.getSystemPromptForMode(s.mode)
	if modePrompt != basePrompt {
		basePrompt = modePrompt
	}
	
	if len(memories) == 0 {
		return basePrompt
	}
	
	memoryContext := "\n\n## Relevant Memories:\n"
	for i, memory := range memories {
		timestamp := ""
		if createdAt, ok := memory.Metadata["memory_time"]; ok {
			timestamp = fmt.Sprintf(" (%v)", createdAt)
		} else if !memory.CreatedAt.IsZero() {
			timestamp = fmt.Sprintf(" (%s)", memory.CreatedAt.Format("2006-01-02 15:04"))
		}
		
		memoryContext += fmt.Sprintf("%d.%s %s\n", i+1, timestamp, memory.Memory)
	}
	
	return basePrompt + memoryContext
}

func (s *SimpleMemChat) getSystemPromptForMode(mode ConversationMode) string {
	switch mode {
	case ConversationModeAnalytical:
		return "You are an analytical AI assistant. Focus on logical reasoning, data analysis, and structured thinking. Break down complex problems systematically and provide evidence-based insights."
		
	case ConversationModeCreative:
		return "You are a creative AI assistant. Think outside the box, suggest innovative solutions, and help with creative endeavors. Be imaginative and encourage exploration of new ideas."
		
	case ConversationModeHelpful:
		return "You are a helpful AI assistant. Focus on being supportive, practical, and solution-oriented. Provide clear, actionable advice and step-by-step guidance."
		
	case ConversationModeExplainer:
		return "You are an explanatory AI assistant. Focus on clear, educational explanations. Break down complex concepts into understandable parts and use examples and analogies."
		
	case ConversationModeSummarizer:
		return "You are a summarizing AI assistant. Focus on condensing information into key points, creating concise overviews, and highlighting the most important aspects."
		
	default:
		return s.config.SystemPrompt
	}
}

func (s *SimpleMemChat) generateFollowUpQuestions(response, originalQuery string) []string {
	// Simple heuristic-based follow-up question generation
	var questions []string
	
	// Check for topics that might warrant follow-up
	if strings.Contains(strings.ToLower(response), "more information") ||
	   strings.Contains(strings.ToLower(response), "details") {
		questions = append(questions, "Would you like me to elaborate on any specific aspect?")
	}
	
	if strings.Contains(strings.ToLower(response), "example") {
		questions = append(questions, "Would you like to see more examples?")
	}
	
	if strings.Contains(strings.ToLower(response), "alternative") ||
	   strings.Contains(strings.ToLower(response), "option") {
		questions = append(questions, "Are you interested in exploring other alternatives?")
	}
	
	// General follow-up based on query type
	if strings.Contains(strings.ToLower(originalQuery), "how") {
		questions = append(questions, "Do you need help implementing this?")
	}
	
	if strings.Contains(strings.ToLower(originalQuery), "what") {
		questions = append(questions, "Would you like to know more about this topic?")
	}
	
	// Limit to max 3 questions
	if len(questions) > 3 {
		questions = questions[:3]
	}
	
	return questions
}

func (s *SimpleMemChat) updateMetrics(retrievedCount, storedCount int, duration time.Duration) {
	s.metrics.TotalMessages++
	s.metrics.MemoriesRetrieved += retrievedCount
	s.metrics.MemoriesStored += storedCount
	
	// Update average response time
	if s.metrics.TotalMessages == 1 {
		s.metrics.AverageResponseTime = duration
	} else {
		s.metrics.AverageResponseTime = time.Duration(
			(int64(s.metrics.AverageResponseTime)*(int64(s.metrics.TotalMessages)-1) + int64(duration)) /
			int64(s.metrics.TotalMessages),
		)
	}
	
	s.metrics.LastActive = time.Now()
}

func (s *SimpleMemChat) estimateTokens(messages types.MessageList, response string) int {
	totalText := response
	for _, msg := range messages {
		totalText += msg.Content
	}
	// Simple heuristic: ~4 characters per token
	return len(totalText) / 4
}

func (s *SimpleMemChat) truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength] + "..."
}

// Memory extraction implementation
func (e *MemoryExtractor) extractFromText(text string) []*types.TextualMemoryItem {
	var memories []*types.TextualMemoryItem
	
	// Apply extraction rules
	for _, rule := range e.extractionRules {
		if !rule.Enabled {
			continue
		}
		
		if e.matchesRule(text, rule) {
			memory := &types.TextualMemoryItem{
				Memory:    text,
				Metadata:  make(map[string]interface{}),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			
			memory.Metadata["category"] = rule.Category
			memory.Metadata["importance"] = rule.Importance
			memory.Metadata["extraction_rule"] = rule.Name
			memory.Metadata["memory_time"] = time.Now()
			
			memories = append(memories, memory)
			break // Only apply first matching rule
		}
	}
	
	return memories
}

func (e *MemoryExtractor) matchesRule(text string, rule ExtractionRule) bool {
	if len(text) < rule.MinLength || len(text) > rule.MaxLength {
		return false
	}
	
	text = strings.ToLower(text)
	
	// Check keywords
	for _, keyword := range rule.Keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}
	
	return false
}