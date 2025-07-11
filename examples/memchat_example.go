// Package main demonstrates comprehensive usage of the MemChat system
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/memtensor/memgos/pkg/chat"
	"github.com/memtensor/memgos/pkg/types"
	"github.com/memtensor/memgos/pkg/interfaces"
)

func memChatMain() {
	fmt.Println("🤖 MemChat System Demo")
	fmt.Println("====================================================")
	
	ctx := context.Background()
	
	// Run different examples
	examples := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"Basic Chat", runBasicChatExample},
		{"Memory-Augmented Chat", runMemoryAugmentedExample},
		{"Chat Manager", runChatManagerExample},
		{"Configuration Profiles", runConfigurationExample},
		{"Conversation Modes", runConversationModesExample},
		{"Advanced Features", runAdvancedFeaturesExample},
	}
	
	for _, example := range examples {
		fmt.Printf("\n📋 Running: %s\n", example.name)
		fmt.Println("------------------------------")
		
		if err := example.fn(ctx); err != nil {
			log.Printf("❌ Error in %s: %v", example.name, err)
		} else {
			fmt.Printf("✅ %s completed successfully\n", example.name)
		}
		
		// Small delay between examples
		time.Sleep(time.Second)
	}
	
	fmt.Println("\n🎉 MemChat Demo completed!")
}

// runBasicChatExample demonstrates basic chat functionality
func runBasicChatExample(ctx context.Context) error {
	fmt.Println("Creating a simple MemChat instance...")
	
	// Create basic chat
	memChat, err := chat.CreateSimpleChat("demo_user", "basic_session")
	if err != nil {
		return fmt.Errorf("failed to create chat: %w", err)
	}
	defer memChat.Close()
	
	// Test basic interactions
	queries := []string{
		"Hello! What can you help me with?",
		"Tell me about Go programming language",
		"What are some best practices for memory management?",
	}
	
	for i, query := range queries {
		fmt.Printf("\n💬 Query %d: %s\n", i+1, query)
		
		result, err := memChat.Chat(ctx, query)
		if err != nil {
			return fmt.Errorf("chat failed: %w", err)
		}
		
		fmt.Printf("🤖 Response: %s\n", truncateString(result.Response, 100))
		fmt.Printf("📊 Tokens used: %d, Duration: %v\n", 
			result.TokensUsed, result.Duration)
		
		if len(result.FollowUpQuestions) > 0 {
			fmt.Printf("💡 Follow-up: %s\n", result.FollowUpQuestions[0])
		}
	}
	
	// Test commands
	fmt.Println("\n🔧 Testing commands...")
	commands := []string{"help", "stats", "clear"}
	
	for _, cmd := range commands {
		result, err := memChat.Chat(ctx, cmd)
		if err != nil {
			continue
		}
		
		fmt.Printf("Command '%s': %s\n", cmd, 
			truncateString(result.Response, 80))
	}
	
	return nil
}

// runMemoryAugmentedExample demonstrates memory integration
func runMemoryAugmentedExample(ctx context.Context) error {
	fmt.Println("Creating MemChat with memory integration...")
	
	// Create memory cube (simplified for demo)
	memCube, err := createDemoMemCube()
	if err != nil {
		return fmt.Errorf("failed to create memory cube: %w", err)
	}
	defer memCube.Close()
	
	// Create chat with memory
	memChat, err := chat.CreateChatWithMemory("memory_user", "memory_session", memCube, 5)
	if err != nil {
		return fmt.Errorf("failed to create memory chat: %w", err)
	}
	defer memChat.Close()
	
	// Conversation with information storage
	conversations := []struct {
		message string
		intent  string
	}{
		{
			message: "My name is Alex and I'm a software engineer working on distributed systems.",
			intent:  "Personal information",
		},
		{
			message: "I prefer using Go for backend services because of its concurrency model.",
			intent:  "Technical preference",
		},
		{
			message: "I'm currently working on a microservices project with Kubernetes deployment.",
			intent:  "Current project",
		},
		{
			message: "What programming languages should I consider for my next project?",
			intent:  "Query using stored information",
		},
	}
	
	for i, conv := range conversations {
		fmt.Printf("\n💬 Turn %d (%s):\n", i+1, conv.intent)
		fmt.Printf("User: %s\n", conv.message)
		
		result, err := memChat.Chat(ctx, conv.message)
		if err != nil {
			return fmt.Errorf("conversation failed: %w", err)
		}
		
		fmt.Printf("🤖 Assistant: %s\n", truncateString(result.Response, 150))
		
		if len(result.NewMemories) > 0 {
			fmt.Printf("🧠 Stored %d memories:\n", len(result.NewMemories))
			for _, mem := range result.NewMemories {
				category := mem.Metadata["category"]
				importance := mem.Metadata["importance"]
				fmt.Printf("  - %s (category: %s, importance: %.2f)\n", 
					truncateString(mem.Memory, 60), category, importance)
			}
		}
		
		if len(result.Memories) > 0 {
			fmt.Printf("🔍 Retrieved %d relevant memories\n", len(result.Memories))
		}
	}
	
	// Test memory recall
	fmt.Println("\n🧠 Memory Recall Test:")
	recallQuery := "What do you know about my background and preferences?"
	result, err := memChat.Chat(ctx, recallQuery)
	if err != nil {
		return fmt.Errorf("memory recall failed: %w", err)
	}
	
	fmt.Printf("Query: %s\n", recallQuery)
	fmt.Printf("🤖 Response: %s\n", result.Response)
	fmt.Printf("📊 Used %d memories for context\n", len(result.Memories))
	
	return nil
}

// runChatManagerExample demonstrates session management
func runChatManagerExample(ctx context.Context) error {
	fmt.Println("Creating and managing multiple chat sessions...")
	
	// Create chat manager
	config := chat.DefaultChatManagerConfig()
	config.MaxConcurrentChats = 5
	
	manager, err := chat.NewChatManager(config)
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}
	defer manager.Stop(ctx)
	
	// Start manager
	if err := manager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start manager: %w", err)
	}
	
	// Create multiple sessions
	users := []string{"alice", "bob", "charlie"}
	sessions := make(map[string]string)
	
	for _, user := range users {
		sessionID, err := manager.CreateSession(ctx, user, nil)
		if err != nil {
			return fmt.Errorf("failed to create session for %s: %w", user, err)
		}
		sessions[user] = sessionID
		fmt.Printf("✅ Created session for %s: %s\n", user, sessionID[:8]+"...")
	}
	
	// Send different messages to each session
	messages := map[string]string{
		"alice":   "I'm interested in machine learning algorithms",
		"bob":     "Can you help me with database optimization?",
		"charlie": "I need advice on API design patterns",
	}
	
	fmt.Println("\n💬 Routing messages to sessions:")
	for user, message := range messages {
		sessionID := sessions[user]
		result, err := manager.RouteChat(ctx, sessionID, message)
		if err != nil {
			fmt.Printf("❌ Failed to route message for %s: %v\n", user, err)
			continue
		}
		
		fmt.Printf("👤 %s: %s\n", user, message)
		fmt.Printf("🤖 Response: %s\n", truncateString(result.Response, 100))
	}
	
	// Get session information
	fmt.Println("\n📊 Session Information:")
	sessionInfo, err := manager.GetSessionInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get session info: %w", err)
	}
	
	for _, info := range sessionInfo {
		fmt.Printf("Session %s: User=%s, Messages=%d, Active=%v\n",
			info.SessionID[:8]+"...", info.UserID, info.MessageCount, info.IsActive)
	}
	
	// Test broadcasting
	fmt.Println("\n📢 Broadcasting message to Alice's sessions:")
	err = manager.BroadcastMessage(ctx, "alice", "System maintenance will begin in 10 minutes")
	if err != nil {
		fmt.Printf("❌ Broadcast failed: %v\n", err)
	} else {
		fmt.Println("✅ Broadcast sent successfully")
	}
	
	return nil
}

// runConfigurationExample demonstrates configuration profiles
func runConfigurationExample(ctx context.Context) error {
	fmt.Println("Managing configuration profiles...")
	
	// Create configuration manager
	configManager, err := chat.NewChatConfigManager("")
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}
	
	// Create different profiles
	profiles := map[string]string{
		"developer":  "standard",
		"researcher": "research", 
		"creative":   "creative",
		"minimal":    "minimal",
	}
	
	fmt.Println("🔧 Creating configuration profiles:")
	for name, template := range profiles {
		config, err := configManager.CreateConfigFromTemplate(template, "demo_user", "session_"+name)
		if err != nil {
			fmt.Printf("❌ Failed to create config from template %s: %v\n", template, err)
			continue
		}
		
		description := fmt.Sprintf("Configuration optimized for %s work", name)
		err = configManager.CreateProfile(name, description, config)
		if err != nil {
			fmt.Printf("❌ Failed to create profile %s: %v\n", name, err)
			continue
		}
		
		fmt.Printf("✅ Created profile '%s' from template '%s'\n", name, template)
		fmt.Printf("   LLM: %s/%s, Memory: %v, TopK: %d\n", 
			config.ChatLLM.Provider, config.ChatLLM.Model, 
			config.EnableTextualMemory, config.TopK)
	}
	
	// List and compare profiles
	fmt.Println("\n📋 Available profiles:")
	profileNames := configManager.ListProfiles()
	for _, name := range profileNames {
		config, err := configManager.GetProfile(name)
		if err != nil {
			continue
		}
		
		fmt.Printf("• %s: Temp=%.1f, Tokens=%d, Memory=%v\n",
			name, config.ChatLLM.Temperature, config.ChatLLM.MaxTokens, config.EnableTextualMemory)
	}
	
	// Test using a profile
	fmt.Println("\n🧪 Testing researcher profile:")
	researchConfig, err := configManager.GetProfile("researcher")
	if err != nil {
		return fmt.Errorf("failed to get researcher profile: %w", err)
	}
	
	memChat, err := chat.CreateFromConfig(researchConfig)
	if err != nil {
		return fmt.Errorf("failed to create chat from profile: %w", err)
	}
	defer memChat.Close()
	
	result, err := memChat.Chat(ctx, "Explain the latest developments in neural network architectures")
	if err != nil {
		return fmt.Errorf("chat with researcher profile failed: %w", err)
	}
	
	fmt.Printf("🤖 Research response: %s\n", truncateString(result.Response, 120))
	
	return nil
}

// runConversationModesExample demonstrates different conversation modes
func runConversationModesExample(ctx context.Context) error {
	fmt.Println("Testing different conversation modes...")
	
	// Create chat instance
	memChat, err := chat.CreateSimpleChat("modes_user", "modes_session")
	if err != nil {
		return fmt.Errorf("failed to create chat: %w", err)
	}
	defer memChat.Close()
	
	// Test different modes
	modes := []struct {
		mode        chat.ConversationMode
		description string
	}{
		{chat.ConversationModeAnalytical, "Logical and structured thinking"},
		{chat.ConversationModeCreative, "Imaginative and innovative"},
		{chat.ConversationModeHelpful, "Practical and solution-oriented"},
		{chat.ConversationModeExplainer, "Educational and clear explanations"},
		{chat.ConversationModeSummarizer, "Concise and key points focused"},
	}
	
	query := "How can I improve team productivity in a remote work environment?"
	
	for i, modeInfo := range modes {
		fmt.Printf("\n🎯 Mode %d: %s (%s)\n", i+1, modeInfo.mode, modeInfo.description)
		
		// Set conversation mode
		err := memChat.SetMode(modeInfo.mode)
		if err != nil {
			fmt.Printf("❌ Failed to set mode: %v\n", err)
			continue
		}
		
		result, err := memChat.Chat(ctx, query)
		if err != nil {
			fmt.Printf("❌ Chat failed in %s mode: %v\n", modeInfo.mode, err)
			continue
		}
		
		fmt.Printf("💬 Query: %s\n", query)
		fmt.Printf("🤖 Response: %s\n", truncateString(result.Response, 150))
		
		if len(result.FollowUpQuestions) > 0 {
			fmt.Printf("💡 Follow-up: %s\n", result.FollowUpQuestions[0])
		}
	}
	
	return nil
}

// runAdvancedFeaturesExample demonstrates advanced MemChat features
func runAdvancedFeaturesExample(ctx context.Context) error {
	fmt.Println("Exploring advanced MemChat features...")
	
	// Create advanced configuration
	config := chat.DefaultMemChatConfig()
	config.UserID = "advanced_user"
	config.SessionID = "advanced_session"
	config.StreamingEnabled = true
	config.TopK = 10
	config.CustomSettings = map[string]interface{}{
		"enable_citations":     true,
		"enable_follow_ups":    true,
		"enable_context_aware": true,
	}
	
	memChat, err := chat.CreateFromConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create advanced chat: %w", err)
	}
	defer memChat.Close()
	
	// Test streaming (simulated)
	fmt.Println("🌊 Testing streaming responses:")
	
	streamingCallback := func(chunk string, isComplete bool, metadata map[string]interface{}) error {
		if isComplete {
			fmt.Printf("\n✅ Stream complete (tokens: %v)\n", metadata["tokens_used"])
		} else {
			fmt.Print(".")
		}
		return nil
	}
	
	result, err := memChat.ChatStream(ctx, "Explain the benefits of microservices architecture", streamingCallback)
	if err != nil {
		return fmt.Errorf("streaming chat failed: %w", err)
	}
	
	fmt.Printf("🤖 Final response: %s\n", truncateString(result.Response, 120))
	
	// Test context management
	fmt.Println("\n🧠 Testing context management:")
	
	// Get current context
	context, err := memChat.GetContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to get context: %w", err)
	}
	
	fmt.Printf("Context: %d messages, %d tokens used\n", 
		len(context.Messages), context.TokensUsed)
	
	// Test multiple turns
	fmt.Println("\n🔄 Multi-turn conversation:")
	turns := []string{
		"I'm building a web application with Go",
		"What database would you recommend?",
		"How about for caching?",
		"Can you compare Redis and Memcached?",
	}
	
	for i, turn := range turns {
		result, err := memChat.Chat(ctx, turn)
		if err != nil {
			fmt.Printf("❌ Turn %d failed: %v\n", i+1, err)
			continue
		}
		
		fmt.Printf("Turn %d: %s\n", i+1, truncateString(turn, 50))
		fmt.Printf("Response: %s\n", truncateString(result.Response, 80))
	}
	
	// Test metrics
	fmt.Println("\n📊 Performance metrics:")
	metrics, err := memChat.GetMetrics(ctx)
	if err != nil {
		return fmt.Errorf("failed to get metrics: %w", err)
	}
	
	fmt.Printf("Total messages: %d\n", metrics.TotalMessages)
	fmt.Printf("Total tokens: %d\n", metrics.TotalTokens)
	fmt.Printf("Average response time: %v\n", metrics.AverageResponseTime)
	fmt.Printf("Memories retrieved: %d\n", metrics.MemoriesRetrieved)
	fmt.Printf("Session duration: %v\n", metrics.SessionDuration)
	
	// Test export
	fmt.Println("\n💾 Exporting chat history:")
	exportPath, err := memChat.ExportChatHistory(ctx, "")
	if err != nil {
		fmt.Printf("❌ Export failed: %v\n", err)
	} else {
		fmt.Printf("✅ Chat history exported to: %s\n", exportPath)
	}
	
	return nil
}

// Helper functions
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func createDemoMemCube() (interfaces.MemCube, error) {
	// In a real implementation, this would create an actual memory cube
	// For demo purposes, we'll create a mock or simplified version
	
	// This is a placeholder - in real usage you would:
	// 1. Create a memory configuration
	// 2. Initialize vector database
	// 3. Set up embeddings
	// 4. Create the memory cube with proper persistence
	
	return &demoMemCube{}, nil
}

// Demo memory cube implementation (simplified)
type demoMemCube struct {
	memories map[string]*types.TextualMemoryItem
}

func (d *demoMemCube) GetID() string { return "demo-cube" }
func (d *demoMemCube) GetName() string { return "Demo Memory Cube" }
func (d *demoMemCube) GetTextualMemory() interfaces.TextualMemory {
	return &demoTextualMemory{memories: d.memories}
}
func (d *demoMemCube) GetActivationMemory() interfaces.ActivationMemory { return nil }
func (d *demoMemCube) GetParametricMemory() interfaces.ParametricMemory { return nil }
func (d *demoMemCube) Load(ctx context.Context, dir string) error { return nil }
func (d *demoMemCube) Dump(ctx context.Context, dir string) error { return nil }
func (d *demoMemCube) Close() error { return nil }

type demoTextualMemory struct {
	memories map[string]*types.TextualMemoryItem
}

func (d *demoTextualMemory) SearchSemantic(ctx context.Context, query string, topK int, filters map[string]string) ([]*types.TextualMemoryItem, error) {
	var results []*types.TextualMemoryItem
	count := 0
	for _, memory := range d.memories {
		if count >= topK {
			break
		}
		results = append(results, memory)
		count++
	}
	return results, nil
}

func (d *demoTextualMemory) AddTextual(ctx context.Context, items []*types.TextualMemoryItem) error {
	if d.memories == nil {
		d.memories = make(map[string]*types.TextualMemoryItem)
	}
	for _, item := range items {
		d.memories[item.ID] = item
	}
	return nil
}

func (d *demoTextualMemory) GetTextual(ctx context.Context, ids []string) ([]*types.TextualMemoryItem, error) {
	var results []*types.TextualMemoryItem
	for _, id := range ids {
		if memory, exists := d.memories[id]; exists {
			results = append(results, memory)
		}
	}
	return results, nil
}

// Implement remaining TextualMemory interface methods (simplified)
func (d *demoTextualMemory) Load(ctx context.Context, dir string) error { return nil }
func (d *demoTextualMemory) Dump(ctx context.Context, dir string) error { return nil }
func (d *demoTextualMemory) Add(ctx context.Context, memories []types.MemoryItem) error { return nil }
func (d *demoTextualMemory) Get(ctx context.Context, id string) (types.MemoryItem, error) { return nil, nil }
func (d *demoTextualMemory) GetAll(ctx context.Context) ([]types.MemoryItem, error) {
	var results []types.MemoryItem
	for _, memory := range d.memories {
		results = append(results, memory)
	}
	return results, nil
}
func (d *demoTextualMemory) Update(ctx context.Context, id string, memory types.MemoryItem) error { return nil }
func (d *demoTextualMemory) Delete(ctx context.Context, id string) error { return nil }
func (d *demoTextualMemory) DeleteAll(ctx context.Context) error { return nil }
func (d *demoTextualMemory) Search(ctx context.Context, query string, topK int) ([]types.MemoryItem, error) { return nil, nil }
func (d *demoTextualMemory) Close() error { return nil }