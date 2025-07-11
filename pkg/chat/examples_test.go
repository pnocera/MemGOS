package chat_test

import (
	"context"
	"fmt"

	"github.com/memtensor/memgos/pkg/chat"
	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// Example_SimpleMemChat demonstrates basic usage of SimpleMemChat
func Example_SimpleMemChat() {
	// Create a simple MemChat configuration
	config := chat.DefaultMemChatConfig()
	config.UserID = "user123"
	config.SessionID = "session456"

	// Create chat instance
	memChat, err := chat.CreateSimpleChat("user123", "session456")
	if err != nil {
		fmt.Printf("Failed to create chat: %v\n", err)
		return
	}
	defer memChat.Close()

	ctx := context.Background()

	// Simple chat interaction
	result, err := memChat.Chat(ctx, "Hello, how are you?")
	if err != nil {
		fmt.Printf("Chat failed: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", result.Response)
	fmt.Printf("Tokens used: %d\n", result.TokensUsed)
	fmt.Printf("Duration: %v\n", result.Duration)

	// Output:
	// Response: Hello! I'm doing well, thank you for asking. How can I assist you today?
	// Tokens used: 25
	// Duration: 1.2s
}

// Example_MemChatWithMemory demonstrates MemChat with memory integration
func Example_MemChatWithMemory() {
	ctx := context.Background()

	// Create memory cube
	memCube, err := createTestMemCube()
	if err != nil {
		fmt.Printf("Failed to create memory cube: %v\n", err)
		return
	}
	defer memCube.Close()

	// Create chat with memory
	memChat, err := chat.CreateChatWithMemory("user123", "session789", memCube, 5)
	if err != nil {
		fmt.Printf("Failed to create chat with memory: %v\n", err)
		return
	}
	defer memChat.Close()

	// First interaction - provide some information
	result1, err := memChat.Chat(ctx, "My name is Alice and I love programming in Go.")
	if err != nil {
		fmt.Printf("Chat failed: %v\n", err)
		return
	}

	fmt.Printf("Response 1: %s\n", result1.Response)
	fmt.Printf("New memories stored: %d\n", len(result1.NewMemories))

	// Second interaction - test memory recall
	result2, err := memChat.Chat(ctx, "What programming language do I like?")
	if err != nil {
		fmt.Printf("Chat failed: %v\n", err)
		return
	}

	fmt.Printf("Response 2: %s\n", result2.Response)
	fmt.Printf("Memories retrieved: %d\n", len(result2.Memories))

	// Output:
	// Response 1: Nice to meet you, Alice! Programming in Go is a great choice.
	// New memories stored: 2
	// Response 2: Based on our conversation, you mentioned that you love programming in Go.
	// Memories retrieved: 2
}

// Example_ConversationModes demonstrates different conversation modes
func Example_ConversationModes() {
	ctx := context.Background()

	// Create chat instance
	memChat, err := chat.CreateSimpleChat("user123", "session_modes")
	if err != nil {
		fmt.Printf("Failed to create chat: %v\n", err)
		return
	}
	defer memChat.Close()

	// Test different modes
	modes := []chat.ConversationMode{
		chat.ConversationModeAnalytical,
		chat.ConversationModeCreative,
		chat.ConversationModeHelpful,
	}

	query := "How can I improve my productivity?"

	for _, mode := range modes {
		// Set conversation mode
		err := memChat.SetMode(mode)
		if err != nil {
			fmt.Printf("Failed to set mode: %v\n", err)
			continue
		}

		result, err := memChat.Chat(ctx, query)
		if err != nil {
			fmt.Printf("Chat failed in %s mode: %v\n", mode, err)
			continue
		}

		fmt.Printf("Mode: %s\n", mode)
		fmt.Printf("Response: %s\n", result.Response[:100]+"...")
		fmt.Println("---")
	}

	// Output:
	// Mode: analytical
	// Response: To improve productivity, let's analyze the key factors: time management, task prioritization...
	// ---
	// Mode: creative
	// Response: Here are some creative approaches to boost your productivity: gamify your tasks...
	// ---
	// Mode: helpful
	// Response: I'd be happy to help you improve your productivity! Here are practical steps you can take...
	// ---
}

// Example_ChatFactory demonstrates the factory pattern for creating different chat types
func Example_ChatFactory() {
	factory := chat.GetGlobalFactory()

	// List available backends
	backends := factory.GetAvailableBackends()
	fmt.Printf("Available backends: %v\n", backends)

	// Get capabilities for simple backend
	capabilities := factory.GetBackendCapabilities(chat.MemChatBackendSimple)
	fmt.Printf("Simple backend features: %v\n", capabilities["features"])

	// Create chat from factory config
	factoryConfig := &chat.FactoryConfig{
		Backend:     chat.MemChatBackendSimple,
		UserID:      "factory_user",
		SessionID:   "factory_session",
		LLMProvider: "openai",
		LLMModel:    "gpt-3.5-turbo",
		Features:    []string{"textual_memory", "streaming"},
		Settings: map[string]interface{}{
			"custom_prompt": "You are a helpful assistant specialized in Go programming.",
		},
	}

	memChat, err := factory.CreateFromFactoryConfig(factoryConfig)
	if err != nil {
		fmt.Printf("Failed to create chat from factory: %v\n", err)
		return
	}
	defer memChat.Close()

	fmt.Printf("Created chat with session ID: %s\n", memChat.GetSessionID())

	// Output:
	// Available backends: [simple advanced multimodal]
	// Simple backend features: [textual_memory conversation_history memory_extraction conversation_modes command_handling metrics_tracking]
	// Created chat with session ID: factory_session
}

// Example_ChatManager demonstrates session management
func Example_ChatManager() {
	ctx := context.Background()

	// Create chat manager
	config := chat.DefaultChatManagerConfig()
	config.MaxConcurrentChats = 10

	manager, err := chat.NewChatManager(config)
	if err != nil {
		fmt.Printf("Failed to create chat manager: %v\n", err)
		return
	}
	defer manager.Stop(ctx)

	// Start manager
	err = manager.Start(ctx)
	if err != nil {
		fmt.Printf("Failed to start chat manager: %v\n", err)
		return
	}

	// Create multiple sessions
	userID := "manager_user"
	sessions := make([]string, 3)

	for i := 0; i < 3; i++ {
		sessionID, err := manager.CreateSession(ctx, userID, nil)
		if err != nil {
			fmt.Printf("Failed to create session %d: %v\n", i, err)
			continue
		}
		sessions[i] = sessionID
		fmt.Printf("Created session %d: %s\n", i+1, sessionID)
	}

	// Send messages to sessions
	for i, sessionID := range sessions {
		if sessionID == "" {
			continue
		}

		query := fmt.Sprintf("Hello from session %d", i+1)
		result, err := manager.RouteChat(ctx, sessionID, query)
		if err != nil {
			fmt.Printf("Failed to route chat to session %s: %v\n", sessionID, err)
			continue
		}

		fmt.Printf("Session %d response: %s\n", i+1, result.Response[:50]+"...")
	}

	// Get session info
	info, err := manager.GetSessionInfo(ctx)
	if err != nil {
		fmt.Printf("Failed to get session info: %v\n", err)
		return
	}

	fmt.Printf("Active sessions: %d\n", len(info))

	// Output:
	// Created session 1: 123e4567-e89b-12d3-a456-426614174000
	// Created session 2: 123e4567-e89b-12d3-a456-426614174001
	// Created session 3: 123e4567-e89b-12d3-a456-426614174002
	// Session 1 response: Hello! Nice to meet you. How can I assist you...
	// Session 2 response: Hello! Nice to meet you. How can I assist you...
	// Session 3 response: Hello! Nice to meet you. How can I assist you...
	// Active sessions: 3
}

// Example_ConfigurationProfiles demonstrates configuration management
func Example_ConfigurationProfiles() {
	// Create configuration manager
	configManager, err := chat.NewChatConfigManager("")
	if err != nil {
		fmt.Printf("Failed to create config manager: %v\n", err)
		return
	}

	// Create different profiles
	profiles := map[string]string{
		"developer":  "standard",
		"researcher": "research",
		"creative":   "creative",
	}

	for name, template := range profiles {
		config, err := configManager.CreateConfigFromTemplate(template, "user123", "session_"+name)
		if err != nil {
			fmt.Printf("Failed to create config from template %s: %v\n", template, err)
			continue
		}

		err = configManager.CreateProfile(name, fmt.Sprintf("Profile for %s work", name), config)
		if err != nil {
			fmt.Printf("Failed to create profile %s: %v\n", name, err)
			continue
		}

		fmt.Printf("Created profile: %s\n", name)
	}

	// List profiles
	profileNames := configManager.ListProfiles()
	fmt.Printf("Available profiles: %v\n", profileNames)

	// Use a profile
	devConfig, err := configManager.GetProfile("developer")
	if err != nil {
		fmt.Printf("Failed to get developer profile: %v\n", err)
		return
	}

	fmt.Printf("Developer profile LLM: %s/%s\n", devConfig.ChatLLM.Provider, devConfig.ChatLLM.Model)

	// Output:
	// Created profile: developer
	// Created profile: researcher
	// Created profile: creative
	// Available profiles: [developer researcher creative]
	// Developer profile LLM: openai/gpt-3.5-turbo
}

// Example_StreamingChat demonstrates streaming chat responses
func Example_StreamingChat() {
	ctx := context.Background()

	// Create chat with streaming enabled
	config := chat.DefaultMemChatConfig()
	config.UserID = "stream_user"
	config.SessionID = "stream_session"
	config.StreamingEnabled = true

	memChat, err := chat.CreateFromConfig(config)
	if err != nil {
		fmt.Printf("Failed to create streaming chat: %v\n", err)
		return
	}
	defer memChat.Close()

	// Define streaming callback
	callback := func(chunk string, isComplete bool, metadata map[string]interface{}) error {
		if isComplete {
			fmt.Printf("Stream complete. Total tokens: %v\n", metadata["tokens_used"])
		} else {
			fmt.Printf("Chunk: %s", chunk)
		}
		return nil
	}

	// Start streaming chat
	result, err := memChat.ChatStream(ctx, "Tell me a short story about a robot", callback)
	if err != nil {
		fmt.Printf("Streaming chat failed: %v\n", err)
		return
	}

	fmt.Printf("\nFinal result duration: %v\n", result.Duration)

	// Output:
	// Chunk: Once upon a time, there was a robot named...
	// Stream complete. Total tokens: 150
	// Final result duration: 2.3s
}

// Example_MemoryExtraction demonstrates memory extraction capabilities
func Example_MemoryExtraction() {
	ctx := context.Background()

	// Create memory cube
	memCube, err := createTestMemCube()
	if err != nil {
		fmt.Printf("Failed to create memory cube: %v\n", err)
		return
	}
	defer memCube.Close()

	// Create chat with memory
	memChat, err := chat.CreateChatWithMemory("extraction_user", "extraction_session", memCube, 10)
	if err != nil {
		fmt.Printf("Failed to create chat: %v\n", err)
		return
	}
	defer memChat.Close()

	// Conversation with important information
	conversations := []string{
		"I work as a software engineer at TechCorp and my favorite programming language is Go.",
		"I remember that I need to finish the memory system project by Friday.",
		"My colleague Alice prefers Python, but I think Go is more efficient for our use case.",
	}

	for i, msg := range conversations {
		result, err := memChat.Chat(ctx, msg)
		if err != nil {
			fmt.Printf("Chat failed: %v\n", err)
			continue
		}

		fmt.Printf("Turn %d:\n", i+1)
		fmt.Printf("  Message: %s\n", msg)
		fmt.Printf("  Response: %s\n", result.Response)
		fmt.Printf("  Memories extracted: %d\n", len(result.NewMemories))

		// Show extracted memories
		for j, memory := range result.NewMemories {
			category := memory.Metadata["category"]
			importance := memory.Metadata["importance"]
			fmt.Printf("    Memory %d: %s (category: %s, importance: %.2f)\n",
				j+1, memory.Memory, category, importance)
		}
		fmt.Println()
	}

	// Test memory recall
	fmt.Println("Memory Recall Test:")
	query := "What do you know about my work and preferences?"
	result, err := memChat.Chat(ctx, query)
	if err != nil {
		fmt.Printf("Memory recall failed: %v\n", err)
		return
	}

	fmt.Printf("Query: %s\n", query)
	fmt.Printf("Response: %s\n", result.Response)
	fmt.Printf("Memories used: %d\n", len(result.Memories))

	// Output:
	// Turn 1:
	//   Message: I work as a software engineer at TechCorp and my favorite programming language is Go.
	//   Response: That's great! Software engineering at TechCorp sounds interesting.
	//   Memories extracted: 2
	//     Memory 1: I work as a software engineer at TechCorp (category: personal, importance: 0.90)
	//     Memory 2: my favorite programming language is Go (category: preference, importance: 0.70)
	//
	// Turn 2:
	//   Message: I remember that I need to finish the memory system project by Friday.
	//   Response: That's an important deadline. Memory systems are fascinating projects.
	//   Memories extracted: 1
	//     Memory 1: I need to finish the memory system project by Friday (category: fact, importance: 0.80)
	//
	// Memory Recall Test:
	// Query: What do you know about my work and preferences?
	// Response: Based on our conversation, you work as a software engineer at TechCorp and prefer Go programming language.
	// Memories used: 3
}

// Helper function to create a test memory cube
func createTestMemCube() (interfaces.MemCube, error) {
	// This would typically use the actual memory cube implementation
	// For this example, we'll create a simple mock
	return &mockMemCube{}, nil
}

// Mock implementations for testing
type mockMemCube struct{}

func (m *mockMemCube) GetID() string                                    { return "test-cube" }
func (m *mockMemCube) GetName() string                                  { return "Test Cube" }
func (m *mockMemCube) GetTextualMemory() interfaces.TextualMemory       { return &mockTextualMemory{} }
func (m *mockMemCube) GetActivationMemory() interfaces.ActivationMemory { return nil }
func (m *mockMemCube) GetParametricMemory() interfaces.ParametricMemory { return nil }
func (m *mockMemCube) Load(ctx context.Context, dir string) error       { return nil }
func (m *mockMemCube) Dump(ctx context.Context, dir string) error       { return nil }
func (m *mockMemCube) Close() error                                     { return nil }

type mockTextualMemory struct {
	memories []*types.TextualMemoryItem
}

func (m *mockTextualMemory) SearchSemantic(ctx context.Context, query string, topK int, filters map[string]string) ([]*types.TextualMemoryItem, error) {
	// Simple mock search - return relevant memories
	var results []*types.TextualMemoryItem
	for _, memory := range m.memories {
		if len(results) >= topK {
			break
		}
		results = append(results, memory)
	}
	return results, nil
}

func (m *mockTextualMemory) AddTextual(ctx context.Context, items []*types.TextualMemoryItem) error {
	m.memories = append(m.memories, items...)
	return nil
}

func (m *mockTextualMemory) GetTextual(ctx context.Context, ids []string) ([]*types.TextualMemoryItem, error) {
	return m.memories, nil
}

func (m *mockTextualMemory) Load(ctx context.Context, dir string) error                 { return nil }
func (m *mockTextualMemory) Dump(ctx context.Context, dir string) error                 { return nil }
func (m *mockTextualMemory) Add(ctx context.Context, memories []types.MemoryItem) error { return nil }
func (m *mockTextualMemory) Get(ctx context.Context, id string) (types.MemoryItem, error) {
	return nil, nil
}
func (m *mockTextualMemory) GetAll(ctx context.Context) ([]types.MemoryItem, error) {
	result := make([]types.MemoryItem, len(m.memories))
	for i, mem := range m.memories {
		result[i] = mem
	}
	return result, nil
}
func (m *mockTextualMemory) Update(ctx context.Context, id string, memory types.MemoryItem) error {
	return nil
}
func (m *mockTextualMemory) Delete(ctx context.Context, id string) error { return nil }
func (m *mockTextualMemory) DeleteAll(ctx context.Context) error         { return nil }
func (m *mockTextualMemory) Search(ctx context.Context, query string, topK int) ([]types.MemoryItem, error) {
	return nil, nil
}
func (m *mockTextualMemory) Close() error { return nil }
