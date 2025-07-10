# MemGOS Examples

This directory contains comprehensive examples demonstrating how to use MemGOS in various scenarios. Each example includes detailed explanations, code samples, and best practices.

## Quick Start Examples

### 1. Basic Memory Operations

**File**: `basic_operations.go`

Demonstrates fundamental MemGOS operations:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/core"
	"github.com/memtensor/memgos/pkg/logger"
	"github.com/memtensor/memgos/pkg/metrics"
	"github.com/memtensor/memgos/pkg/types"
)

func main() {
	ctx := context.Background()
	
	// Initialize MemGOS
	cfg := config.NewMOSConfig()
	cfg.UserID = "example-user"
	cfg.EnableTextualMemory = true
	
	logger := logger.NewConsoleLogger("info")
	metrics := metrics.NewPrometheusMetrics()
	
	mosCore, err := core.NewMOSCore(cfg, logger, metrics)
	if err != nil {
		log.Fatal(err)
	}
	defer mosCore.Close()
	
	err = mosCore.Initialize(ctx)
	if err != nil {
		log.Fatal(err)
	}
	
	// Register a memory cube
	err = mosCore.RegisterMemCube(ctx, "./data/knowledge", "kb", cfg.UserID)
	if err != nil {
		log.Fatal(err)
	}
	
	// Add memory
	addReq := &types.AddMemoryRequest{
		MemoryContent: stringPtr("Go is a statically typed programming language"),
		CubeID:        stringPtr("kb"),
		UserID:        stringPtr(cfg.UserID),
	}
	
	err = mosCore.Add(ctx, addReq)
	if err != nil {
		log.Fatal(err)
	}
	
	// Search memories
	searchQuery := &types.SearchQuery{
		Query:   "programming language",
		TopK:    5,
		CubeIDs: []string{"kb"},
		UserID:  cfg.UserID,
	}
	
	results, err := mosCore.Search(ctx, searchQuery)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Found %d results\n", len(results.TextMem))
	for _, mem := range results.TextMem {
		fmt.Printf("- %s\n", mem.Memory)
	}
}

func stringPtr(s string) *string {
	return &s
}
```

### 2. Chat Integration

**File**: `chat_example.go`

Shows how to use MemGOS for AI chat with memory context:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/core"
	"github.com/memtensor/memgos/pkg/logger"
	"github.com/memtensor/memgos/pkg/metrics"
	"github.com/memtensor/memgos/pkg/types"
)

func main() {
	ctx := context.Background()
	
	// Configure with OpenAI for chat
	cfg := config.NewMOSConfig()
	cfg.UserID = "chat-user"
	cfg.EnableTextualMemory = true
	cfg.ChatModel = &config.LLMConfig{
		Backend: "openai",
		Model:   "gpt-3.5-turbo",
		APIKey:  "your-openai-api-key",
	}
	
	logger := logger.NewConsoleLogger("info")
	metrics := metrics.NewPrometheusMetrics()
	
	mosCore, err := core.NewMOSCore(cfg, logger, metrics)
	if err != nil {
		log.Fatal(err)
	}
	defer mosCore.Close()
	
	err = mosCore.Initialize(ctx)
	if err != nil {
		log.Fatal(err)
	}
	
	// Register knowledge base
	err = mosCore.RegisterMemCube(ctx, "./data/docs", "docs", cfg.UserID)
	if err != nil {
		log.Fatal(err)
	}
	
	// Add some knowledge
	knowledge := []string{
		"MemGOS is a memory operating system for AI applications",
		"Go channels enable communication between goroutines",
		"REST APIs provide stateless communication between services",
	}
	
	for _, k := range knowledge {
		addReq := &types.AddMemoryRequest{
			MemoryContent: &k,
			CubeID:        stringPtr("docs"),
			UserID:        stringPtr(cfg.UserID),
		}
		err = mosCore.Add(ctx, addReq)
		if err != nil {
			log.Fatal(err)
		}
	}
	
	// Chat with memory context
	chatReq := &types.ChatRequest{
		Query:  "What is MemGOS and how does it relate to Go?",
		UserID: stringPtr(cfg.UserID),
	}
	
	response, err := mosCore.Chat(ctx, chatReq)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("AI Response: %s\n", response.Response)
}
```

### 3. Multi-Cube Operations

**File**: `multi_cube_example.go`

Demonstrates working with multiple memory cubes:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/core"
	"github.com/memtensor/memgos/pkg/logger"
	"github.com/memtensor/memgos/pkg/memory"
	"github.com/memtensor/memgos/pkg/metrics"
	"github.com/memtensor/memgos/pkg/types"
)

func main() {
	ctx := context.Background()
	
	cfg := config.NewMOSConfig()
	cfg.UserID = "multi-user"
	cfg.EnableTextualMemory = true
	
	logger := logger.NewConsoleLogger("info")
	metrics := metrics.NewPrometheusMetrics()
	
	mosCore, err := core.NewMOSCore(cfg, logger, metrics)
	if err != nil {
		log.Fatal(err)
	}
	defer mosCore.Close()
	
	err = mosCore.Initialize(ctx)
	if err != nil {
		log.Fatal(err)
	}
	
	// Create and register multiple cubes
	cubes := map[string][]string{
		"programming": {
			"Go is compiled and statically typed",
			"Python is interpreted and dynamically typed",
			"Rust focuses on memory safety without garbage collection",
		},
		"databases": {
			"PostgreSQL is a powerful relational database",
			"MongoDB is a popular NoSQL document database",
			"Redis is an in-memory key-value store",
		},
		"architecture": {
			"Microservices enable independent deployment and scaling",
			"Event-driven architecture promotes loose coupling",
			"CQRS separates read and write operations",
		},
	}
	
	// Register cubes and add content
	for cubeID, memories := range cubes {
		err = mosCore.RegisterMemCube(ctx, fmt.Sprintf("./data/%s", cubeID), cubeID, cfg.UserID)
		if err != nil {
			log.Fatal(err)
		}
		
		for _, memory := range memories {
			addReq := &types.AddMemoryRequest{
				MemoryContent: &memory,
				CubeID:        &cubeID,
				UserID:        stringPtr(cfg.UserID),
			}
			err = mosCore.Add(ctx, addReq)
			if err != nil {
				log.Fatal(err)
			}
		}
		fmt.Printf("Added %d memories to cube '%s'\n", len(memories), cubeID)
	}
	
	// Search across all cubes
	searchQuery := &types.SearchQuery{
		Query:   "database",
		TopK:    10,
		CubeIDs: []string{"programming", "databases", "architecture"},
		UserID:  cfg.UserID,
	}
	
	results, err := mosCore.Search(ctx, searchQuery)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("\nCross-cube search results for 'database':\n")
	for _, mem := range results.TextMem {
		fmt.Printf("- [%s] %s\n", mem.CubeID, mem.Memory)
	}
	
	// Search specific cube
	specificQuery := &types.SearchQuery{
		Query:   "typed",
		TopK:    5,
		CubeIDs: []string{"programming"},
		UserID:  cfg.UserID,
	}
	
	results, err = mosCore.Search(ctx, specificQuery)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("\nProgramming-specific search results for 'typed':\n")
	for _, mem := range results.TextMem {
		fmt.Printf("- %s\n", mem.Memory)
	}
}
```

## Advanced Examples

### 4. Semantic Search with Embeddings

**File**: `semantic_search_example.go`

Demonstrates advanced semantic search capabilities:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/core"
	"github.com/memtensor/memgos/pkg/logger"
	"github.com/memtensor/memgos/pkg/metrics"
	"github.com/memtensor/memgos/pkg/types"
)

func main() {
	ctx := context.Background()
	
	// Configure with OpenAI embeddings
	cfg := config.NewMOSConfig()
	cfg.UserID = "semantic-user"
	cfg.EnableTextualMemory = true
	cfg.MemReader = &config.MemoryConfig{
		Backend:        "general",
		MemoryFilename: "semantic_memory.json",
		Embedder: &config.EmbedderConfig{
			Backend: "openai",
			Model:   "text-embedding-ada-002",
			APIKey:  "your-openai-api-key",
		},
	}
	
	logger := logger.NewConsoleLogger("info")
	metrics := metrics.NewPrometheusMetrics()
	
	mosCore, err := core.NewMOSCore(cfg, logger, metrics)
	if err != nil {
		log.Fatal(err)
	}
	defer mosCore.Close()
	
	err = mosCore.Initialize(ctx)
	if err != nil {
		log.Fatal(err)
	}
	
	err = mosCore.RegisterMemCube(ctx, "./data/concepts", "concepts", cfg.UserID)
	if err != nil {
		log.Fatal(err)
	}
	
	// Add conceptually diverse content
	concepts := []string{
		"Machine learning algorithms learn patterns from data",
		"Neural networks are inspired by biological brain structure",
		"Deep learning uses multiple layers for feature extraction",
		"Natural language processing helps computers understand text",
		"Computer vision enables machines to interpret visual information",
		"Reinforcement learning uses rewards to train agents",
		"Supervised learning requires labeled training data",
		"Unsupervised learning finds hidden patterns in data",
	}
	
	for _, concept := range concepts {
		addReq := &types.AddMemoryRequest{
			MemoryContent: &concept,
			CubeID:        stringPtr("concepts"),
			UserID:        stringPtr(cfg.UserID),
		}
		err = mosCore.Add(ctx, addReq)
		if err != nil {
			log.Fatal(err)
		}
	}
	
	// Semantic search queries
	semanticQueries := []string{
		"artificial intelligence training methods",
		"computer understanding of human language",
		"visual recognition systems",
		"learning without explicit instruction",
	}
	
	for _, query := range semanticQueries {
		searchQuery := &types.SearchQuery{
			Query:   query,
			TopK:    3,
			CubeIDs: []string{"concepts"},
			UserID:  cfg.UserID,
		}
		
		results, err := mosCore.Search(ctx, searchQuery)
		if err != nil {
			log.Fatal(err)
		}
		
		fmt.Printf("\nSemantic search for: '%s'\n", query)
		for i, mem := range results.TextMem {
			fmt.Printf("%d. %s (score: %.3f)\n", i+1, mem.Memory, mem.Score)
		}
	}
}
```

### 5. API Server Integration

**File**: `api_server_example.go`

Shows how to integrate with the MemGOS API server:

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type MemGOSClient struct {
	BaseURL string
	UserID  string
	Client  *http.Client
}

type AddMemoryRequest struct {
	MemoryContent string                 `json:"memory_content"`
	CubeID        string                 `json:"cube_id"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type SearchResponse struct {
	Success bool `json:"success"`
	Data    struct {
		TextualMemories []struct {
			ID       string                 `json:"id"`
			Memory   string                 `json:"memory"`
			Metadata map[string]interface{} `json:"metadata"`
			CubeID   string                 `json:"cube_id"`
			Score    float64                `json:"score"`
		} `json:"textual_memories"`
	} `json:"data"`
}

func NewMemGOSClient(baseURL, userID string) *MemGOSClient {
	return &MemGOSClient{
		BaseURL: baseURL,
		UserID:  userID,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *MemGOSClient) AddMemory(memory, cubeID string, metadata map[string]interface{}) error {
	req := AddMemoryRequest{
		MemoryContent: memory,
		CubeID:        cubeID,
		Metadata:      metadata,
	}
	
	jsonData, err := json.Marshal(req)
	if err != nil {
		return err
	}
	
	httpReq, err := http.NewRequest("POST", c.BaseURL+"/api/v1/memories", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-User-ID", c.UserID)
	
	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add memory: %s", resp.Status)
	}
	
	return nil
}

func (c *MemGOSClient) Search(query string, topK int) (*SearchResponse, error) {
	url := fmt.Sprintf("%s/api/v1/search?q=%s&top_k=%d", c.BaseURL, query, topK)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("X-User-ID", c.UserID)
	
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	var searchResp SearchResponse
	err = json.Unmarshal(body, &searchResp)
	if err != nil {
		return nil, err
	}
	
	return &searchResp, nil
}

func main() {
	// Initialize client
	client := NewMemGOSClient("http://localhost:8000", "api-user")
	
	// Add memories
	memories := map[string]map[string]interface{}{
		"Go is excellent for building APIs and microservices": {
			"category": "programming",
			"language": "go",
		},
		"Docker containers provide consistent deployment environments": {
			"category": "devops",
			"tool":     "docker",
		},
		"Kubernetes orchestrates containerized applications at scale": {
			"category": "devops",
			"tool":     "kubernetes",
		},
	}
	
	for memory, metadata := range memories {
		err := client.AddMemory(memory, "tech-stack", metadata)
		if err != nil {
			log.Printf("Failed to add memory: %v", err)
			continue
		}
		fmt.Printf("Added: %s\n", memory)
	}
	
	// Search memories
	queries := []string{"microservices", "containers", "deployment"}
	
	for _, query := range queries {
		results, err := client.Search(query, 3)
		if err != nil {
			log.Printf("Search failed: %v", err)
			continue
		}
		
		fmt.Printf("\nSearch results for '%s':\n", query)
		for _, mem := range results.Data.TextualMemories {
			fmt.Printf("- %s (score: %.3f)\n", mem.Memory, mem.Score)
		}
	}
}
```

## Integration Examples

### 6. Document Processing Pipeline

**File**: `document_pipeline_example.go`

Shows how to build a document processing pipeline:

```go
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/core"
	"github.com/memtensor/memgos/pkg/logger"
	"github.com/memtensor/memgos/pkg/metrics"
	"github.com/memtensor/memgos/pkg/types"
)

type DocumentProcessor struct {
	mosCore interfaces.MOSCore
	userID  string
}

func NewDocumentProcessor(mosCore interfaces.MOSCore, userID string) *DocumentProcessor {
	return &DocumentProcessor{
		mosCore: mosCore,
		userID:  userID,
	}
}

func (dp *DocumentProcessor) ProcessDirectory(ctx context.Context, dirPath, cubeID string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.IsDir() {
			return nil
		}
		
		// Process text files
		if strings.HasSuffix(strings.ToLower(path), ".txt") ||
			strings.HasSuffix(strings.ToLower(path), ".md") {
			return dp.processTextFile(ctx, path, cubeID)
		}
		
		return nil
	})
}

func (dp *DocumentProcessor) processTextFile(ctx context.Context, filePath, cubeID string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	content, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	
	// Split content into chunks
	chunks := dp.chunkContent(string(content), 500)
	
	for i, chunk := range chunks {
		if strings.TrimSpace(chunk) == "" {
			continue
		}
		
		metadata := map[string]interface{}{
			"source":    filePath,
			"chunk_id":  i,
			"total_chunks": len(chunks),
			"processed_at": time.Now().Format(time.RFC3339),
		}
		
		addReq := &types.AddMemoryRequest{
			MemoryContent: &chunk,
			CubeID:        &cubeID,
			UserID:        &dp.userID,
			Metadata:      metadata,
		}
		
		err = dp.mosCore.Add(ctx, addReq)
		if err != nil {
			return fmt.Errorf("failed to add chunk %d from %s: %w", i, filePath, err)
		}
	}
	
	fmt.Printf("Processed %s: %d chunks\n", filePath, len(chunks))
	return nil
}

func (dp *DocumentProcessor) chunkContent(content string, maxLength int) []string {
	sentences := strings.Split(content, ".")
	var chunks []string
	var currentChunk strings.Builder
	
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}
		
		if currentChunk.Len()+len(sentence) > maxLength {
			if currentChunk.Len() > 0 {
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
			}
		}
		
		if currentChunk.Len() > 0 {
			currentChunk.WriteString(". ")
		}
		currentChunk.WriteString(sentence)
	}
	
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}
	
	return chunks
}

func main() {
	ctx := context.Background()
	
	cfg := config.NewMOSConfig()
	cfg.UserID = "doc-processor"
	cfg.EnableTextualMemory = true
	
	logger := logger.NewConsoleLogger("info")
	metrics := metrics.NewPrometheusMetrics()
	
	mosCore, err := core.NewMOSCore(cfg, logger, metrics)
	if err != nil {
		log.Fatal(err)
	}
	defer mosCore.Close()
	
	err = mosCore.Initialize(ctx)
	if err != nil {
		log.Fatal(err)
	}
	
	// Register document cube
	err = mosCore.RegisterMemCube(ctx, "./data/documents", "docs", cfg.UserID)
	if err != nil {
		log.Fatal(err)
	}
	
	// Process documents
	processor := NewDocumentProcessor(mosCore, cfg.UserID)
	
	documentDir := "./sample-docs"
	err = processor.ProcessDirectory(ctx, documentDir, "docs")
	if err != nil {
		log.Fatal(err)
	}
	
	// Test search across processed documents
	testQueries := []string{
		"key concepts",
		"implementation details",
		"best practices",
	}
	
	for _, query := range testQueries {
		searchQuery := &types.SearchQuery{
			Query:   query,
			TopK:    5,
			CubeIDs: []string{"docs"},
			UserID:  cfg.UserID,
		}
		
		results, err := mosCore.Search(ctx, searchQuery)
		if err != nil {
			log.Printf("Search failed: %v", err)
			continue
		}
		
		fmt.Printf("\nSearch results for '%s':\n", query)
		for _, mem := range results.TextMem {
			fmt.Printf("- %s... (from %s)\n", 
				mem.Memory[:min(100, len(mem.Memory))], 
				mem.Metadata["source"])
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

### 7. Real-time Chat Bot

**File**: `chatbot_example.go`

Builds a complete chatbot with memory integration:

```go
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/core"
	"github.com/memtensor/memgos/pkg/logger"
	"github.com/memtensor/memgos/pkg/metrics"
	"github.com/memtensor/memgos/pkg/types"
)

type ChatBot struct {
	mosCore        interfaces.MOSCore
	userID         string
	conversationID string
	context        []string
}

func NewChatBot(mosCore interfaces.MOSCore, userID string) *ChatBot {
	return &ChatBot{
		mosCore:        mosCore,
		userID:         userID,
		conversationID: fmt.Sprintf("conv-%d", time.Now().Unix()),
		context:        make([]string, 0),
	}
}

func (cb *ChatBot) Chat(ctx context.Context, message string) (string, error) {
	// Add user message to context
	cb.context = append(cb.context, fmt.Sprintf("User: %s", message))
	
	// Keep context manageable
	if len(cb.context) > 10 {
		cb.context = cb.context[len(cb.context)-10:]
	}
	
	// Build chat request with conversation context
	contextStr := strings.Join(cb.context, "\n")
	chatReq := &types.ChatRequest{
		Query:          fmt.Sprintf("Context: %s\n\nCurrent question: %s", contextStr, message),
		UserID:         &cb.userID,
		ConversationID: &cb.conversationID,
	}
	
	response, err := cb.mosCore.Chat(ctx, chatReq)
	if err != nil {
		return "", err
	}
	
	// Add assistant response to context
	cb.context = append(cb.context, fmt.Sprintf("Assistant: %s", response.Response))
	
	return response.Response, nil
}

func (cb *ChatBot) LearnFromInteraction(ctx context.Context, userInput, response string) error {
	// Store successful interactions as memories
	learning := fmt.Sprintf("User asked: %s. Assistant responded: %s", userInput, response)
	
	addReq := &types.AddMemoryRequest{
		MemoryContent: &learning,
		CubeID:        stringPtr("conversation-history"),
		UserID:        &cb.userID,
		Metadata: map[string]interface{}{
			"type":            "conversation",
			"conversation_id": cb.conversationID,
			"timestamp":       time.Now().Format(time.RFC3339),
			"user_id":         cb.userID,
		},
	}
	
	return cb.mosCore.Add(ctx, addReq)
}

func main() {
	ctx := context.Background()
	
	// Configure with OpenAI
	cfg := config.NewMOSConfig()
	cfg.UserID = "chatbot-user"
	cfg.EnableTextualMemory = true
	cfg.ChatModel = &config.LLMConfig{
		Backend:     "openai",
		Model:       "gpt-3.5-turbo",
		APIKey:      os.Getenv("OPENAI_API_KEY"),
		MaxTokens:   1024,
		Temperature: 0.7,
	}
	
	logger := logger.NewConsoleLogger("info")
	metrics := metrics.NewPrometheusMetrics()
	
	mosCore, err := core.NewMOSCore(cfg, logger, metrics)
	if err != nil {
		log.Fatal(err)
	}
	defer mosCore.Close()
	
	err = mosCore.Initialize(ctx)
	if err != nil {
		log.Fatal(err)
	}
	
	// Register knowledge and conversation cubes
	err = mosCore.RegisterMemCube(ctx, "./data/knowledge", "knowledge", cfg.UserID)
	if err != nil {
		log.Fatal(err)
	}
	
	err = mosCore.RegisterMemCube(ctx, "./data/conversations", "conversation-history", cfg.UserID)
	if err != nil {
		log.Fatal(err)
	}
	
	// Initialize chatbot
	chatbot := NewChatBot(mosCore, cfg.UserID)
	
	// Add some initial knowledge
	initialKnowledge := []string{
		"I am a helpful AI assistant powered by MemGOS",
		"MemGOS is a memory operating system for AI applications",
		"I can remember our conversation and learn from interactions",
		"I have access to a knowledge base that I can search",
	}
	
	for _, knowledge := range initialKnowledge {
		addReq := &types.AddMemoryRequest{
			MemoryContent: &knowledge,
			CubeID:        stringPtr("knowledge"),
			UserID:        &cfg.UserID,
		}
		err = mosCore.Add(ctx, addReq)
		if err != nil {
			log.Printf("Failed to add knowledge: %v", err)
		}
	}
	
	fmt.Println("MemGOS ChatBot - Type 'quit' to exit")
	fmt.Println("=====================================\n")
	
	scanner := bufio.NewScanner(os.Stdin)
	
	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}
		
		userInput := strings.TrimSpace(scanner.Text())
		if userInput == "quit" || userInput == "exit" {
			break
		}
		
		if userInput == "" {
			continue
		}
		
		// Get response from chatbot
		response, err := chatbot.Chat(ctx, userInput)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		
		fmt.Printf("Bot: %s\n\n", response)
		
		// Learn from this interaction
		err = chatbot.LearnFromInteraction(ctx, userInput, response)
		if err != nil {
			log.Printf("Failed to learn from interaction: %v", err)
		}
	}
	
	fmt.Println("\nGoodbye!")
}
```

## Configuration Examples

### 8. Production Configuration

**File**: `config/production.yaml`

```yaml
# Production MemGOS Configuration

# Core settings
user_id: "${USER_ID}"
session_id: "${SESSION_ID}"

# Memory configuration
enable_textual_memory: true
enable_activation_memory: true
enable_parametric_memory: false
enable_mem_scheduler: true

# Chat model (OpenAI)
chat_model:
  backend: "openai"
  model: "gpt-4"
  api_key: "${OPENAI_API_KEY}"
  max_tokens: 2048
  temperature: 0.7
  timeout: 30

# Memory reader with embeddings
mem_reader:
  backend: "general"
  memory_filename: "textual_memory.json"
  top_k: 10
  use_semantic_search: true
  
  embedder:
    backend: "openai"
    model: "text-embedding-ada-002"
    api_key: "${OPENAI_API_KEY}"
    batch_size: 100
    timeout: 30

# Vector database (Qdrant)
vector_db:
  backend: "qdrant"
  host: "${QDRANT_HOST}"
  port: 6333
  collection: "memgos_vectors"
  vector_size: 1536
  distance: "cosine"
  
# Graph database (Neo4j)
graph_db:
  backend: "neo4j"
  uri: "${NEO4J_URI}"
  username: "${NEO4J_USERNAME}"
  password: "${NEO4J_PASSWORD}"
  database: "memgos"
  
# API server
api:
  enabled: true
  host: "0.0.0.0"
  port: 8000
  cors_enabled: true
  rate_limit:
    enabled: true
    requests_per_minute: 100
  auth:
    enabled: true
    jwt_secret: "${JWT_SECRET}"
    
# Caching (Redis)
cache:
  backend: "redis"
  host: "${REDIS_HOST}"
  port: 6379
  password: "${REDIS_PASSWORD}"
  database: 0
  ttl: 3600
  
# Monitoring and logging
log_level: "info"
metrics_enabled: true
health_check_enabled: true

monitoring:
  prometheus:
    enabled: true
    port: 9090
  jaeger:
    enabled: true
    endpoint: "${JAEGER_ENDPOINT}"
    
# Scheduler
scheduler:
  enabled: true
  optimization_interval: "1h"
  cleanup_interval: "24h"
  backup_interval: "12h"
  backup_path: "/data/backups"
  
# Security
security:
  tls:
    enabled: true
    cert_file: "/certs/server.crt"
    key_file: "/certs/server.key"
  encryption:
    enabled: true
    key: "${ENCRYPTION_KEY}"
    
# Performance tuning
performance:
  max_concurrent_requests: 100
  request_timeout: 30
  memory_limit: "2GB"
  gc_percentage: 100
```

### 9. Development Configuration

**File**: `config/development.yaml`

```yaml
# Development MemGOS Configuration

# Core settings
user_id: "dev-user"
session_id: "dev-session"

# Memory configuration
enable_textual_memory: true
enable_activation_memory: false
enable_parametric_memory: false
enable_mem_scheduler: false

# Chat model (local Ollama)
chat_model:
  backend: "ollama"
  model: "llama2:7b"
  endpoint: "http://localhost:11434"
  temperature: 0.7

# Simple memory reader
mem_reader:
  backend: "general"
  memory_filename: "dev_memory.json"
  top_k: 5
  use_semantic_search: false
  
# Local file storage
vector_db:
  backend: "file"
  path: "./data/vectors"
  
# API server
api:
  enabled: true
  host: "localhost"
  port: 8000
  cors_enabled: true
  rate_limit:
    enabled: false
  auth:
    enabled: false
    
# Logging
log_level: "debug"
metrics_enabled: true
health_check_enabled: true

# Development helpers
development:
  hot_reload: true
  debug_mode: true
  mock_llm: false
  sample_data: true
```

## Testing Examples

### 10. Integration Test Suite

**File**: `integration_test_example.go`

```go
package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/core"
	"github.com/memtensor/memgos/pkg/logger"
	"github.com/memtensor/memgos/pkg/metrics"
	"github.com/memtensor/memgos/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type IntegrationTestSuite struct {
	mosCore  interfaces.MOSCore
	tempDir  string
	userID   string
	ctx      context.Context
	logger   interfaces.Logger
	metrics  interfaces.Metrics
}

func NewIntegrationTestSuite(t *testing.T) *IntegrationTestSuite {
	tempDir, err := os.MkdirTemp("", "memgos-integration-test")
	require.NoError(t, err)
	
	cfg := config.NewMOSConfig()
	cfg.UserID = "test-user"
	cfg.SessionID = "test-session"
	cfg.EnableTextualMemory = true
	cfg.EnableActivationMemory = false
	cfg.EnableParametricMemory = false
	
	logger := logger.NewConsoleLogger("error") // Quiet during tests
	metrics := metrics.NewPrometheusMetrics()
	
	mosCore, err := core.NewMOSCore(cfg, logger, metrics)
	require.NoError(t, err)
	
	err = mosCore.Initialize(context.Background())
	require.NoError(t, err)
	
	return &IntegrationTestSuite{
		mosCore: mosCore,
		tempDir: tempDir,
		userID:  cfg.UserID,
		ctx:     context.Background(),
		logger:  logger,
		metrics: metrics,
	}
}

func (suite *IntegrationTestSuite) Cleanup() {
	suite.mosCore.Close()
	os.RemoveAll(suite.tempDir)
}

func TestFullWorkflow(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()
	
	// Test 1: Register memory cube
	cubeID := "test-cube"
	err := suite.mosCore.RegisterMemCube(suite.ctx, suite.tempDir, cubeID, suite.userID)
	assert.NoError(t, err)
	
	// Test 2: Add memories
	memories := []string{
		"Go is a programming language",
		"Python is interpreted",
		"Rust is memory safe",
		"JavaScript runs in browsers",
		"Java is object-oriented",
	}
	
	for i, memory := range memories {
		addReq := &types.AddMemoryRequest{
			MemoryContent: &memory,
			CubeID:        &cubeID,
			UserID:        &suite.userID,
			Metadata: map[string]interface{}{
				"index":    i,
				"category": "programming",
			},
		}
		
		err = suite.mosCore.Add(suite.ctx, addReq)
		assert.NoError(t, err)
	}
	
	// Test 3: Search memories
	searchQuery := &types.SearchQuery{
		Query:   "programming",
		TopK:    10,
		CubeIDs: []string{cubeID},
		UserID:  suite.userID,
	}
	
	results, err := suite.mosCore.Search(suite.ctx, searchQuery)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(results.TextMem), 1)
	
	// Test 4: Specific search
	specificQuery := &types.SearchQuery{
		Query:   "Go",
		TopK:    5,
		CubeIDs: []string{cubeID},
		UserID:  suite.userID,
	}
	
	specificResults, err := suite.mosCore.Search(suite.ctx, specificQuery)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(specificResults.TextMem), 1)
	assert.Contains(t, specificResults.TextMem[0].Memory, "Go")
	
	// Test 5: Memory operations
	if len(results.TextMem) > 0 {
		memoryID := results.TextMem[0].ID
		
		// Get specific memory
		memory, err := suite.mosCore.Get(suite.ctx, cubeID, memoryID, suite.userID)
		assert.NoError(t, err)
		assert.NotNil(t, memory)
		
		// Update memory
		updatedContent := "Updated: Go is a fast programming language"
		updatedMemory := &types.TextualMemoryItem{
			ID:     memoryID,
			Memory: updatedContent,
			Metadata: map[string]interface{}{
				"category": "programming",
				"updated":  true,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		err = suite.mosCore.Update(suite.ctx, cubeID, memoryID, suite.userID, updatedMemory)
		assert.NoError(t, err)
		
		// Verify update
		updatedMem, err := suite.mosCore.Get(suite.ctx, cubeID, memoryID, suite.userID)
		assert.NoError(t, err)
		if textMem, ok := updatedMem.(*types.TextualMemoryItem); ok {
			assert.Equal(t, updatedContent, textMem.Memory)
			assert.True(t, textMem.Metadata["updated"].(bool))
		}
	}
	
	// Test 6: User operations
	userInfo, err := suite.mosCore.GetUserInfo(suite.ctx, suite.userID)
	assert.NoError(t, err)
	assert.Equal(t, suite.userID, userInfo["user_id"])
	
	// Test 7: Cube unregistration
	err = suite.mosCore.UnregisterMemCube(suite.ctx, cubeID, suite.userID)
	assert.NoError(t, err)
	
	// Verify cube is unregistered
	_, err = suite.mosCore.Search(suite.ctx, searchQuery)
	assert.Error(t, err) // Should fail as cube is unregistered
}

func TestConcurrentOperations(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()
	
	cubeID := "concurrent-test"
	err := suite.mosCore.RegisterMemCube(suite.ctx, suite.tempDir, cubeID, suite.userID)
	require.NoError(t, err)
	
	// Concurrent memory additions
	const numGoroutines = 10
	const memoriesPerGoroutine = 5
	
	errors := make(chan error, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < memoriesPerGoroutine; j++ {
				memory := fmt.Sprintf("Memory %d-%d: Concurrent operation test", goroutineID, j)
				addReq := &types.AddMemoryRequest{
					MemoryContent: &memory,
					CubeID:        &cubeID,
					UserID:        &suite.userID,
					Metadata: map[string]interface{}{
						"goroutine": goroutineID,
						"index":     j,
					},
				}
				
				err := suite.mosCore.Add(suite.ctx, addReq)
				if err != nil {
					errors <- err
					return
				}
			}
			errors <- nil
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		err := <-errors
		assert.NoError(t, err)
	}
	
	// Verify all memories were added
	searchQuery := &types.SearchQuery{
		Query:   "Concurrent",
		TopK:    100,
		CubeIDs: []string{cubeID},
		UserID:  suite.userID,
	}
	
	results, err := suite.mosCore.Search(suite.ctx, searchQuery)
	assert.NoError(t, err)
	assert.Len(t, results.TextMem, numGoroutines*memoriesPerGoroutine)
}

func TestErrorHandling(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()
	
	// Test invalid cube registration
	err := suite.mosCore.RegisterMemCube(suite.ctx, "/nonexistent/path", "invalid", suite.userID)
	assert.Error(t, err)
	
	// Test operations on non-existent cube
	addReq := &types.AddMemoryRequest{
		MemoryContent: stringPtr("Test memory"),
		CubeID:        stringPtr("nonexistent"),
		UserID:        &suite.userID,
	}
	
	err = suite.mosCore.Add(suite.ctx, addReq)
	assert.Error(t, err)
	
	// Test invalid search
	searchQuery := &types.SearchQuery{
		Query:   "test",
		TopK:    5,
		CubeIDs: []string{"nonexistent"},
		UserID:  suite.userID,
	}
	
	_, err = suite.mosCore.Search(suite.ctx, searchQuery)
	assert.Error(t, err)
	
	// Test invalid user operations
	_, err = suite.mosCore.GetUserInfo(suite.ctx, "nonexistent-user")
	assert.Error(t, err)
}
```

## Running the Examples

### Prerequisites

1. **Install MemGOS**:
   ```bash
   go install github.com/memtensor/memgos/cmd/memgos@latest
   ```

2. **Set Environment Variables**:
   ```bash
   export OPENAI_API_KEY="your-key-here"
   export QDRANT_URL="http://localhost:6333"  # Optional
   export NEO4J_URI="bolt://localhost:7687"   # Optional
   ```

3. **Create Data Directories**:
   ```bash
   mkdir -p ./data/{knowledge,documents,conversations}
   mkdir -p ./sample-docs
   ```

### Running Examples

```bash
# Basic operations
go run examples/basic_operations.go

# Chat integration
go run examples/chat_example.go

# Multi-cube operations
go run examples/multi_cube_example.go

# Semantic search
go run examples/semantic_search_example.go

# API client
go run examples/api_server_example.go

# Document processing
go run examples/document_pipeline_example.go

# Interactive chatbot
go run examples/chatbot_example.go

# Run tests
go test -v examples/integration_test_example.go
```

### Docker Examples

**File**: `docker-compose.example.yml`

```yaml
version: '3.8'

services:
  memgos:
    image: memtensor/memgos:latest
    ports:
      - "8000:8000"
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - QDRANT_URL=http://qdrant:6333
      - NEO4J_URI=bolt://neo4j:7687
      - NEO4J_USERNAME=neo4j
      - NEO4J_PASSWORD=password
    volumes:
      - ./config:/config
      - ./data:/data
    depends_on:
      - qdrant
      - neo4j
      
  qdrant:
    image: qdrant/qdrant:latest
    ports:
      - "6333:6333"
    volumes:
      - qdrant_data:/qdrant/storage
      
  neo4j:
    image: neo4j:latest
    ports:
      - "7474:7474"
      - "7687:7687"
    environment:
      - NEO4J_AUTH=neo4j/password
    volumes:
      - neo4j_data:/data
      
volumes:
  qdrant_data:
  neo4j_data:
```

## Best Practices from Examples

1. **Resource Management**: Always defer Close() calls
2. **Error Handling**: Check errors from all MemGOS operations
3. **Context Usage**: Pass context for cancellation and timeouts
4. **Configuration**: Use environment variables for sensitive data
5. **Memory Organization**: Use appropriate cube names and metadata
6. **Concurrent Safety**: MemGOS is thread-safe, use goroutines freely
7. **Testing**: Write integration tests for complex workflows
8. **Monitoring**: Enable metrics and logging in production

## Next Steps

After running these examples:

1. **Explore Configuration**: Try different backends and options
2. **Build Your Application**: Use examples as starting points
3. **Performance Testing**: Benchmark with your data sizes
4. **Production Deployment**: Follow production configuration examples
5. **Contribute**: Submit your own examples to help others

For more advanced topics, see:
- [Performance Tuning Guide](../performance-tuning.md)
- [Production Deployment Guide](../production-deployment.md)
- [API Documentation](../api/README.md)
- [Architecture Documentation](../architecture/README.md)
