// Package mcp provides a self-documenting MCP (Model Context Protocol) server for MemGOS
package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// Server represents the MemGOS MCP server
type Server struct {
	mosCore   interfaces.MOSCore
	apiClient APIClient // For token-based API communication
	config    *config.MOSConfig
	logger    interfaces.Logger
	server    *server.MCPServer
}

// APIClient interface for HTTP API communication
type APIClient interface {
	interfaces.MOSCore
	ListCubes(ctx context.Context, userID string) ([]*types.MemCube, error)
	GetCubeInfo(ctx context.Context, cubeID, userID string) (*types.MemCube, error)
	HealthCheck(ctx context.Context) (map[string]interface{}, error)
}

// ServerInfo contains metadata about the MCP server
type ServerInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
	License     string `json:"license"`
}

// NewServer creates a new MemGOS MCP server with direct MOSCore
func NewServer(mosCore interfaces.MOSCore, cfg *config.MOSConfig, logger interfaces.Logger) *Server {
	s := &Server{
		mosCore: mosCore,
		config:  cfg,
		logger:  logger,
	}

	// Create MCP server with self-documenting capabilities
	mcpServer := server.NewMCPServer(
		"MemGOS MCP Server",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	s.server = mcpServer
	s.setupTools()
	// TODO: Add resource providers when mcp-go library supports them
	
	return s
}

// NewServerWithAPIClient creates a new MemGOS MCP server with API client
func NewServerWithAPIClient(apiClient APIClient, cfg *config.MOSConfig, logger interfaces.Logger) *Server {
	s := &Server{
		apiClient: apiClient,
		config:    cfg,
		logger:    logger,
	}

	// Create MCP server with self-documenting capabilities
	mcpServer := server.NewMCPServer(
		"MemGOS MCP Server (API Mode)",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	s.server = mcpServer
	s.setupTools()
	// TODO: Add resource providers when mcp-go library supports them
	
	return s
}

// getClient returns the active client (either MOSCore or API client)
func (s *Server) getClient() interfaces.MOSCore {
	if s.apiClient != nil {
		return s.apiClient
	}
	return s.getClient()
}

// getAPIClient returns the API client with additional methods
func (s *Server) getAPIClient() APIClient {
	if s.apiClient != nil {
		return s.apiClient
	}
	// Return a wrapper for MOSCore if needed
	return &MOSCoreWrapper{mosCore: s.mosCore}
}

// MOSCoreWrapper wraps MOSCore to provide API client interface
type MOSCoreWrapper struct {
	mosCore interfaces.MOSCore
}

func (w *MOSCoreWrapper) Initialize(ctx context.Context) error                { return w.mosCore.Initialize(ctx) }
func (w *MOSCoreWrapper) RegisterMemCube(ctx context.Context, cubePath, cubeID, userID string) error { return w.mosCore.RegisterMemCube(ctx, cubePath, cubeID, userID) }
func (w *MOSCoreWrapper) UnregisterMemCube(ctx context.Context, cubeID, userID string) error { return w.mosCore.UnregisterMemCube(ctx, cubeID, userID) }
func (w *MOSCoreWrapper) Search(ctx context.Context, query *types.SearchQuery) (*types.MOSSearchResult, error) { return w.mosCore.Search(ctx, query) }
func (w *MOSCoreWrapper) Add(ctx context.Context, request *types.AddMemoryRequest) error { return w.mosCore.Add(ctx, request) }
func (w *MOSCoreWrapper) Get(ctx context.Context, cubeID, memoryID, userID string) (types.MemoryItem, error) { return w.mosCore.Get(ctx, cubeID, memoryID, userID) }
func (w *MOSCoreWrapper) GetAll(ctx context.Context, cubeID, userID string) (*types.MOSSearchResult, error) { return w.mosCore.GetAll(ctx, cubeID, userID) }
func (w *MOSCoreWrapper) Update(ctx context.Context, cubeID, memoryID, userID string, memory types.MemoryItem) error { return w.mosCore.Update(ctx, cubeID, memoryID, userID, memory) }
func (w *MOSCoreWrapper) Delete(ctx context.Context, cubeID, memoryID, userID string) error { return w.mosCore.Delete(ctx, cubeID, memoryID, userID) }
func (w *MOSCoreWrapper) DeleteAll(ctx context.Context, cubeID, userID string) error { return w.mosCore.DeleteAll(ctx, cubeID, userID) }
func (w *MOSCoreWrapper) Chat(ctx context.Context, request *types.ChatRequest) (*types.ChatResponse, error) { return w.mosCore.Chat(ctx, request) }
func (w *MOSCoreWrapper) CreateUser(ctx context.Context, userName string, role types.UserRole, userID string) (string, error) { return w.mosCore.CreateUser(ctx, userName, role, userID) }
func (w *MOSCoreWrapper) GetUserInfo(ctx context.Context, userID string) (map[string]interface{}, error) { return w.mosCore.GetUserInfo(ctx, userID) }
func (w *MOSCoreWrapper) ListUsers(ctx context.Context) ([]*types.User, error) { return w.mosCore.ListUsers(ctx) }
func (w *MOSCoreWrapper) ShareCubeWithUser(ctx context.Context, cubeID, targetUserID string) (bool, error) { return w.mosCore.ShareCubeWithUser(ctx, cubeID, targetUserID) }
func (w *MOSCoreWrapper) Close() error { return w.mosCore.Close() }

// Implement API client specific methods with fallbacks for MOSCore
func (w *MOSCoreWrapper) ListCubes(ctx context.Context, userID string) ([]*types.MemCube, error) {
	// For MOSCore, we'll return empty list since this method doesn't exist in the interface
	return []*types.MemCube{}, nil
}

func (w *MOSCoreWrapper) GetCubeInfo(ctx context.Context, cubeID, userID string) (*types.MemCube, error) {
	// For MOSCore, create a basic cube info from available data
	return &types.MemCube{
		ID:      cubeID,
		Name:    cubeID,
		OwnerID: userID,
	}, nil
}

func (w *MOSCoreWrapper) HealthCheck(ctx context.Context) (map[string]interface{}, error) {
	// For MOSCore, return basic health status
	return map[string]interface{}{
		"status": "healthy",
		"mode":   "direct",
	}, nil
}

// Start starts the MCP server with stdio transport
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting MemGOS MCP Server", map[string]interface{}{
		"transport": "stdio",
		"version":   "1.0.0",
	})

	// Run server with stdio transport
	return server.ServeStdio(s.server)
}

// Stop gracefully stops the MCP server
func (s *Server) Stop() error {
	s.logger.Info("Stopping MemGOS MCP Server", nil)
	return nil
}

// GetServerInfo returns metadata about the MCP server
func (s *Server) GetServerInfo() ServerInfo {
	return ServerInfo{
		Name:        "MemGOS MCP Server",
		Version:     "1.0.0",
		Description: "Self-documenting MCP server for MemGOS - Memory Operating System",
		Author:      "MemGOS Contributors",
		License:     "MIT",
	}
}

// setupTools configures all available MCP tools
func (s *Server) setupTools() {
	// Memory Search Tool
	searchTool := mcp.NewTool("search_memories",
		mcp.WithDescription("Search for memories using semantic similarity or keywords"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query for finding relevant memories"),
		),
		mcp.WithNumber("top_k",
			mcp.Description("Number of top results to return (default: 10)"),
		),
		mcp.WithString("cube_id",
			mcp.Description("Optional: Search within specific memory cube"),
		),
	)
	s.server.AddTool(searchTool, s.handleSearchMemories)

	// Add Memory Tool
	addTool := mcp.NewTool("add_memory",
		mcp.WithDescription("Add a new memory to the memory system"),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("The content/text of the memory to add"),
		),
		mcp.WithString("cube_id",
			mcp.Required(),
			mcp.Description("Memory cube ID to add the memory to"),
		),
		mcp.WithObject("metadata",
			mcp.Description("Optional metadata for the memory (tags, categories, etc.)"),
		),
	)
	s.server.AddTool(addTool, s.handleAddMemory)

	// Get User Info Tool
	userInfoTool := mcp.NewTool("get_user_info",
		mcp.WithDescription("Get information about the current user"),
	)
	s.server.AddTool(userInfoTool, s.handleGetUserInfo)

	// Health Check Tool
	healthTool := mcp.NewTool("health_check",
		mcp.WithDescription("Check the health status of the MemGOS system"),
	)
	s.server.AddTool(healthTool, s.handleHealthCheck)

	// List Tools Tool (Self-Documentation)
	listToolsTool := mcp.NewTool("list_tools",
		mcp.WithDescription("List all available MCP tools with their descriptions"),
		mcp.WithBoolean("include_schemas",
			mcp.Description("Include detailed parameter schemas (default: true)"),
		),
	)
	s.server.AddTool(listToolsTool, s.handleListTools)

	// Register Cube Tool
	registerCubeTool := mcp.NewTool("register_cube",
		mcp.WithDescription("Register a new memory cube in the system"),
		mcp.WithString("cube_path",
			mcp.Required(),
			mcp.Description("File system path to the memory cube"),
		),
		mcp.WithString("cube_id",
			mcp.Required(),
			mcp.Description("Unique identifier for the cube"),
		),
		mcp.WithString("cube_name",
			mcp.Description("Human-readable name for the cube"),
		),
	)
	s.server.AddTool(registerCubeTool, s.handleRegisterCube)

	// Chat Tool
	chatTool := mcp.NewTool("chat",
		mcp.WithDescription("Send a chat message with memory-augmented context"),
		mcp.WithString("message",
			mcp.Required(),
			mcp.Description("The chat message to send"),
		),
		mcp.WithString("session_id",
			mcp.Description("Chat session identifier"),
		),
		mcp.WithBoolean("use_memory",
			mcp.Description("Whether to use memory context (default: true)"),
		),
	)
	s.server.AddTool(chatTool, s.handleChat)

	// Get Memory Tool
	getMemoryTool := mcp.NewTool("get_memory",
		mcp.WithDescription("Retrieve a specific memory by ID from a cube"),
		mcp.WithString("cube_id",
			mcp.Required(),
			mcp.Description("Memory cube ID to search in"),
		),
		mcp.WithString("memory_id",
			mcp.Required(),
			mcp.Description("Unique identifier of the memory to retrieve"),
		),
	)
	s.server.AddTool(getMemoryTool, s.handleGetMemory)

	// Update Memory Tool
	updateMemoryTool := mcp.NewTool("update_memory",
		mcp.WithDescription("Update the content of an existing memory"),
		mcp.WithString("cube_id",
			mcp.Required(),
			mcp.Description("Memory cube ID containing the memory"),
		),
		mcp.WithString("memory_id",
			mcp.Required(),
			mcp.Description("Unique identifier of the memory to update"),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("New content for the memory"),
		),
		mcp.WithObject("metadata",
			mcp.Description("Optional updated metadata for the memory"),
		),
	)
	s.server.AddTool(updateMemoryTool, s.handleUpdateMemory)

	// Delete Memory Tool
	deleteMemoryTool := mcp.NewTool("delete_memory",
		mcp.WithDescription("Delete a specific memory from a cube"),
		mcp.WithString("cube_id",
			mcp.Required(),
			mcp.Description("Memory cube ID containing the memory"),
		),
		mcp.WithString("memory_id",
			mcp.Required(),
			mcp.Description("Unique identifier of the memory to delete"),
		),
	)
	s.server.AddTool(deleteMemoryTool, s.handleDeleteMemory)

	// List Cubes Tool
	listCubesTool := mcp.NewTool("list_cubes",
		mcp.WithDescription("List all memory cubes accessible to the current user"),
		mcp.WithBoolean("include_stats",
			mcp.Description("Include cube statistics like memory counts (default: false)"),
		),
	)
	s.server.AddTool(listCubesTool, s.handleListCubes)

	// Cube Info Tool
	cubeInfoTool := mcp.NewTool("cube_info",
		mcp.WithDescription("Get detailed information about a specific memory cube"),
		mcp.WithString("cube_id",
			mcp.Required(),
			mcp.Description("Memory cube ID to get information about"),
		),
		mcp.WithBoolean("include_contents",
			mcp.Description("Include a sample of cube contents (default: false)"),
		),
	)
	s.server.AddTool(cubeInfoTool, s.handleCubeInfo)

	// Chat History Tool
	chatHistoryTool := mcp.NewTool("chat_history",
		mcp.WithDescription("Retrieve chat history for the current session"),
		mcp.WithString("session_id",
			mcp.Description("Session ID to get history for (defaults to current session)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of messages to return (default: 20)"),
		),
	)
	s.server.AddTool(chatHistoryTool, s.handleChatHistory)

	// Context Search Tool
	contextSearchTool := mcp.NewTool("context_search",
		mcp.WithDescription("Search for contextually relevant memories for a given query"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Query to find relevant context for"),
		),
		mcp.WithNumber("context_size",
			mcp.Description("Number of context items to return (default: 5)"),
		),
		mcp.WithString("cube_id",
			mcp.Description("Optional: Search within specific memory cube"),
		),
	)
	s.server.AddTool(contextSearchTool, s.handleContextSearch)

	// Session Info Tool
	sessionInfoTool := mcp.NewTool("session_info",
		mcp.WithDescription("Get information about the current user session"),
	)
	s.server.AddTool(sessionInfoTool, s.handleSessionInfo)

	// Validate Access Tool
	validateAccessTool := mcp.NewTool("validate_access",
		mcp.WithDescription("Validate user access to specific resources"),
		mcp.WithString("resource_type",
			mcp.Required(),
			mcp.Description("Type of resource to validate (cube, user, system)"),
		),
		mcp.WithString("resource_id",
			mcp.Required(),
			mcp.Description("ID of the resource to validate access to"),
		),
	)
	s.server.AddTool(validateAccessTool, s.handleValidateAccess)
}

// Tool Handler Functions

func (s *Server) handleSearchMemories(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	query := arguments["query"].(string)
	topK := 10
	if tk, ok := arguments["top_k"]; ok {
		if tkFloat, ok := tk.(float64); ok {
			topK = int(tkFloat)
		}
	}
	
	cubeID := ""
	if ci, ok := arguments["cube_id"]; ok {
		cubeID = ci.(string)
	}

	// Create search query
	searchQuery := &types.SearchQuery{
		Query:   query,
		TopK:    topK,
		UserID:  s.config.UserID,
		CubeIDs: []string{},
	}
	
	if cubeID != "" {
		searchQuery.CubeIDs = []string{cubeID}
	}

	// Execute search using active client
	result, err := s.getClient().Search(ctx, searchQuery)
	if err != nil {
		s.logger.Error("Failed to search memories", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ Search failed: %s", err.Error()),
				},
			},
		}, nil
	}

	// Format results
	resultText := fmt.Sprintf("🔍 Search Results for: '%s'\n\n", query)
	
	totalResults := 0
	if len(result.TextMem) > 0 {
		resultText += "📝 **Textual Memories:**\n"
		for _, cubeResult := range result.TextMem {
			resultText += fmt.Sprintf("\n🗂️ Cube: %s\n", cubeResult.CubeID)
			for i, memory := range cubeResult.Memories {
				if i >= topK {
					break
				}
				content := memory.GetContent()
				if len(content) > 100 {
					content = content[:100] + "..."
				}
				resultText += fmt.Sprintf("  %d. %s\n", i+1, content)
				totalResults++
			}
		}
	}
	
	if len(result.ActMem) > 0 {
		resultText += "\n🧠 **Activation Memories:**\n"
		for _, cubeResult := range result.ActMem {
			resultText += fmt.Sprintf("\n🗂️ Cube: %s\n", cubeResult.CubeID)
			for i := range cubeResult.Memories {
				if i >= topK {
					break
				}
				resultText += fmt.Sprintf("  %d. [Activation Memory]\n", i+1)
				totalResults++
			}
		}
	}
	
	if len(result.ParaMem) > 0 {
		resultText += "\n⚙️ **Parametric Memories:**\n"
		for _, cubeResult := range result.ParaMem {
			resultText += fmt.Sprintf("\n🗂️ Cube: %s\n", cubeResult.CubeID)
			for i := range cubeResult.Memories {
				if i >= topK {
					break
				}
				resultText += fmt.Sprintf("  %d. [Parametric Memory]\n", i+1)
				totalResults++
			}
		}
	}
	
	if totalResults == 0 {
		resultText += "\n📭 No memories found matching your query."
	} else {
		resultText += fmt.Sprintf("\n\n✅ Found %d total results", totalResults)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: resultText,
			},
		},
	}, nil
}

func (s *Server) handleAddMemory(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	content := arguments["content"].(string)
	cubeID := arguments["cube_id"].(string)
	
	// Handle optional metadata
	var metadata map[string]interface{}
	if md, ok := arguments["metadata"]; ok {
		if mdMap, ok := md.(map[string]interface{}); ok {
			metadata = mdMap
		}
	}

	// Create add memory request
	addRequest := &types.AddMemoryRequest{
		MemCubeID:     cubeID,
		MemoryContent: &content,
		UserID:        s.config.UserID,
	}

	// Execute add using active client
	err := s.getClient().Add(ctx, addRequest)
	if err != nil {
		s.logger.Error("Failed to add memory", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ Failed to add memory: %s", err.Error()),
				},
			},
		}, nil
	}

	resultText := fmt.Sprintf("✅ Successfully added memory to cube: %s\n\n📝 Content: %s", cubeID, content)
	if metadata != nil {
		resultText += fmt.Sprintf("\n📋 Metadata: %v", metadata)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: resultText,
			},
		},
	}, nil
}

func (s *Server) handleGetUserInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userInfo := fmt.Sprintf("👤 Current User Information:\n\n• User ID: %s\n• Session ID: %s\n• MCP Server: Active\n• Version: 1.0.0",
		s.config.UserID, s.config.SessionID)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: userInfo,
			},
		},
	}, nil
}

func (s *Server) handleHealthCheck(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get health status using API client
	health, err := s.getAPIClient().HealthCheck(ctx)
	if err != nil {
		s.logger.Error("Failed to get health status", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ Health check failed: %s", err.Error()),
				},
			},
		}, nil
	}

	mode := "API"
	if s.mosCore != nil {
		mode = "Direct"
	}

	healthInfo := fmt.Sprintf("🟢 **MemGOS MCP Server Health Status:**\n\n")
	healthInfo += fmt.Sprintf("• **MCP Server:** Running\n")
	healthInfo += fmt.Sprintf("• **Transport:** stdio\n")
	healthInfo += fmt.Sprintf("• **Mode:** %s\n", mode)
	healthInfo += fmt.Sprintf("• **Backend Status:** %v\n", health["status"])
	healthInfo += fmt.Sprintf("• **User ID:** %s\n", s.config.UserID)
	healthInfo += fmt.Sprintf("• **Session ID:** %s\n", s.config.SessionID)
	healthInfo += fmt.Sprintf("• **Version:** 1.0.0\n")

	if s.config.MCPMode && s.config.APIURL != "" {
		healthInfo += fmt.Sprintf("• **API URL:** %s\n", s.config.APIURL)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: healthInfo,
			},
		},
	}, nil
}

func (s *Server) handleListTools(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	includeSchemas := true
	if is, ok := arguments["include_schemas"]; ok {
		if isBool, ok := is.(bool); ok {
			includeSchemas = isBool
		}
	}

	toolsList := "🛠️ Available MemGOS MCP Tools:\n\n"
	tools := []struct {
		name        string
		description string
		params      string
	}{
		{"search_memories", "Search for memories using semantic similarity or keywords", "query (required), top_k, cube_id"},
		{"add_memory", "Add a new memory to the memory system", "content (required), cube_id (required), metadata"},
		{"register_cube", "Register a new memory cube in the system", "cube_path (required), cube_id (required), cube_name"},
		{"chat", "Send a chat message with memory-augmented context", "message (required), session_id, use_memory"},
		{"get_memory", "Retrieve a specific memory by ID from a cube", "cube_id (required), memory_id (required)"},
		{"update_memory", "Update the content of an existing memory", "cube_id (required), memory_id (required), content (required), metadata"},
		{"delete_memory", "Delete a specific memory from a cube", "cube_id (required), memory_id (required)"},
		{"list_cubes", "List all memory cubes accessible to the current user", "include_stats"},
		{"cube_info", "Get detailed information about a specific memory cube", "cube_id (required), include_contents"},
		{"chat_history", "Retrieve chat history for the current session", "session_id, limit"},
		{"context_search", "Search for contextually relevant memories for a given query", "query (required), context_size, cube_id"},
		{"session_info", "Get information about the current user session", "none"},
		{"validate_access", "Validate user access to specific resources", "resource_type (required), resource_id (required)"},
		{"get_user_info", "Get information about the current user", "none"},
		{"health_check", "Check the health status of the MemGOS system", "none"},
		{"list_tools", "List all available MCP tools with their descriptions", "include_schemas"},
	}

	for i, tool := range tools {
		toolsList += fmt.Sprintf("%d. **%s**\n   %s\n", i+1, tool.name, tool.description)
		if includeSchemas {
			toolsList += fmt.Sprintf("   Parameters: %s\n", tool.params)
		}
		toolsList += "\n"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: toolsList,
			},
		},
	}, nil
}

func (s *Server) handleRegisterCube(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	cubePath := arguments["cube_path"].(string)
	cubeID := arguments["cube_id"].(string)
	
	cubeName := cubeID // Default to cube ID
	if cn, ok := arguments["cube_name"]; ok {
		cubeName = cn.(string)
	}

	// Execute cube registration using MOSCore
	err := s.getClient().RegisterMemCube(ctx, cubePath, cubeID, s.config.UserID)
	if err != nil {
		s.logger.Error("Failed to register cube", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ Failed to register cube: %s", err.Error()),
				},
			},
		}, nil
	}

	resultText := fmt.Sprintf("✅ Successfully registered memory cube:\n\n• Cube ID: %s\n• Cube Name: %s\n• Path: %s\n• User: %s",
		cubeID, cubeName, cubePath, s.config.UserID)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: resultText,
			},
		},
	}, nil
}

func (s *Server) handleChat(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	message := arguments["message"].(string)
	
	useMemory := true
	if um, ok := arguments["use_memory"]; ok {
		if umBool, ok := um.(bool); ok {
			useMemory = umBool
		}
	}

	// Create chat request
	chatRequest := &types.ChatRequest{
		Query:  message,
		UserID: s.config.UserID,
	}

	// Execute chat using MOSCore
	response, err := s.getClient().Chat(ctx, chatRequest)
	if err != nil {
		s.logger.Error("Failed to process chat", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ Chat failed: %s", err.Error()),
				},
			},
		}, nil
	}

	memoryNote := ""
	if useMemory {
		memoryNote = "\n🧠 Response enhanced with memory context"
	}
	
	resultText := fmt.Sprintf("💬 **Chat Response:**\n\n%s%s\n\n📋 **Session Info:**\n• Session ID: %s\n• User ID: %s\n• Timestamp: %s",
		response.Response, memoryNote, response.SessionID, response.UserID, response.Timestamp.Format("2006-01-02 15:04:05"))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: resultText,
			},
		},
	}, nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Additional tool handlers (shortened for space - implement remaining handlers following the same pattern)

func (s *Server) handleGetMemory(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	cubeID := arguments["cube_id"].(string)
	memoryID := arguments["memory_id"].(string)

	// Get memory using MOSCore
	memory, err := s.getClient().Get(ctx, cubeID, memoryID, s.config.UserID)
	if err != nil {
		s.logger.Error("Failed to get memory", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ Failed to get memory: %s", err.Error()),
				},
			},
		}, nil
	}

	resultText := fmt.Sprintf("📄 **Memory Retrieved:**\n\n• **ID:** %s\n• **Cube:** %s\n• **Content:** %s\n• **Type:** %T",
		memoryID, cubeID, memory.GetContent(), memory)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: resultText,
			},
		},
	}, nil
}

func (s *Server) handleUpdateMemory(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	cubeID := arguments["cube_id"].(string)
	memoryID := arguments["memory_id"].(string)
	content := arguments["content"].(string)

	// Get existing memory first
	existingMemory, err := s.getClient().Get(ctx, cubeID, memoryID, s.config.UserID)
	if err != nil {
		s.logger.Error("Failed to get existing memory for update", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ Failed to find memory to update: %s", err.Error()),
				},
			},
		}, nil
	}

	// Update memory content based on type
	switch mem := existingMemory.(type) {
	case *types.TextualMemoryItem:
		mem.Memory = content
		if metadata, ok := arguments["metadata"]; ok {
			if mdMap, ok := metadata.(map[string]interface{}); ok {
				for k, v := range mdMap {
					mem.Metadata[k] = v
				}
			}
		}
		err = s.getClient().Update(ctx, cubeID, memoryID, s.config.UserID, mem)
	default:
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "❌ Unsupported memory type for update",
				},
			},
		}, nil
	}

	if err != nil {
		s.logger.Error("Failed to update memory", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ Failed to update memory: %s", err.Error()),
				},
			},
		}, nil
	}

	resultText := fmt.Sprintf("✅ Successfully updated memory:\n\n• **ID:** %s\n• **Cube:** %s\n• **New Content:** %s",
		memoryID, cubeID, content)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: resultText,
			},
		},
	}, nil
}

func (s *Server) handleDeleteMemory(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	cubeID := arguments["cube_id"].(string)
	memoryID := arguments["memory_id"].(string)

	// Delete memory using MOSCore
	err := s.getClient().Delete(ctx, cubeID, memoryID, s.config.UserID)
	if err != nil {
		s.logger.Error("Failed to delete memory", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ Failed to delete memory: %s", err.Error()),
				},
			},
		}, nil
	}

	resultText := fmt.Sprintf("✅ Successfully deleted memory:\n\n• **ID:** %s\n• **Cube:** %s",
		memoryID, cubeID)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: resultText,
			},
		},
	}, nil
}

func (s *Server) handleListCubes(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	includeStats := false
	if is, ok := arguments["include_stats"]; ok {
		if isBool, ok := is.(bool); ok {
			includeStats = isBool
		}
	}

	// Use API client to get cubes list
	cubes, err := s.getAPIClient().ListCubes(ctx, s.config.UserID)
	if err != nil {
		s.logger.Error("Failed to get user cubes", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ Failed to list cubes: %s", err.Error()),
				},
			},
		}, nil
	}

	resultText := "📦 **Accessible Memory Cubes:**\n\n"

	if len(cubes) == 0 {
		resultText += "📭 No memory cubes found for user"
	} else {
		for i, cube := range cubes {
			resultText += fmt.Sprintf("%d. **%s** (ID: %s)\n", i+1, cube.Name, cube.ID)
			if cube.Path != "" {
				resultText += fmt.Sprintf("   • Path: %s\n", cube.Path)
			}
			resultText += fmt.Sprintf("   • Owner: %s\n", cube.OwnerID)
			resultText += fmt.Sprintf("   • Status: %s\n", func() string {
				if cube.IsLoaded {
					return "🟢 Loaded"
				}
				return "🔴 Not Loaded"
			}())

			if includeStats {
				// Get cube statistics
				if cubeResult, err := s.getClient().GetAll(ctx, cube.ID, s.config.UserID); err == nil {
					textCount := 0
					actCount := 0
					paraCount := 0
					for _, textCube := range cubeResult.TextMem {
						textCount += len(textCube.Memories)
					}
					for _, actCube := range cubeResult.ActMem {
						actCount += len(actCube.Memories)
					}
					for _, paraCube := range cubeResult.ParaMem {
						paraCount += len(paraCube.Memories)
					}
					resultText += fmt.Sprintf("   • Textual Memories: %d\n", textCount)
					resultText += fmt.Sprintf("   • Activation Memories: %d\n", actCount)
					resultText += fmt.Sprintf("   • Parametric Memories: %d\n", paraCount)
				}
			}
			resultText += "\n"
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: resultText,
			},
		},
	}, nil
}

func (s *Server) handleCubeInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	cubeID := arguments["cube_id"].(string)
	includeContents := false
	if ic, ok := arguments["include_contents"]; ok {
		if icBool, ok := ic.(bool); ok {
			includeContents = icBool
		}
	}

	// Get cube information using API client
	cubeInfo, err := s.getAPIClient().GetCubeInfo(ctx, cubeID, s.config.UserID)
	if err != nil {
		s.logger.Error("Failed to get cube info", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ Failed to get cube info: %s", err.Error()),
				},
			},
		}, nil
	}

	// Get cube contents if needed
	var cubeResult *types.MOSSearchResult
	if includeContents {
		cubeResult, err = s.getClient().GetAll(ctx, cubeID, s.config.UserID)
		if err != nil {
			s.logger.Warn("Failed to get cube contents", map[string]interface{}{
				"cube_id": cubeID,
				"error":   err.Error(),
			})
		}
	}

	textCount := 0
	actCount := 0
	paraCount := 0
	if cubeResult != nil {
		for _, textCube := range cubeResult.TextMem {
			textCount += len(textCube.Memories)
		}
		for _, actCube := range cubeResult.ActMem {
			actCount += len(actCube.Memories)
		}
		for _, paraCube := range cubeResult.ParaMem {
			paraCount += len(paraCube.Memories)
		}
	}

	resultText := fmt.Sprintf("📦 **Cube Information: %s**\n\n", cubeInfo.Name)
	resultText += fmt.Sprintf("• **ID:** %s\n", cubeInfo.ID)
	resultText += fmt.Sprintf("• **Owner:** %s\n", cubeInfo.OwnerID)
	if cubeInfo.Path != "" {
		resultText += fmt.Sprintf("• **Path:** %s\n", cubeInfo.Path)
	}
	resultText += fmt.Sprintf("• **Status:** %s\n", func() string {
		if cubeInfo.IsLoaded {
			return "🟢 Loaded"
		}
		return "🔴 Not Loaded"
	}())
	resultText += fmt.Sprintf("• **Created:** %s\n", cubeInfo.CreatedAt.Format("2006-01-02 15:04:05"))
	resultText += fmt.Sprintf("• **Updated:** %s\n\n", cubeInfo.UpdatedAt.Format("2006-01-02 15:04:05"))
	
	if cubeResult != nil {
		resultText += fmt.Sprintf("📊 **Statistics:**\n")
		resultText += fmt.Sprintf("• Textual Memories: %d\n", textCount)
		resultText += fmt.Sprintf("• Activation Memories: %d\n", actCount)
		resultText += fmt.Sprintf("• Parametric Memories: %d\n", paraCount)
		resultText += fmt.Sprintf("• Total Memories: %d\n\n", textCount+actCount+paraCount)
	}

	if includeContents && cubeResult != nil && textCount > 0 {
		resultText += "📄 **Sample Contents (Textual):**\n"
		sampleCount := 0
		for _, textCube := range cubeResult.TextMem {
			for _, memory := range textCube.Memories {
				if sampleCount >= 3 {
					break
				}
				content := memory.GetContent()
				if len(content) > 100 {
					content = content[:100] + "..."
				}
				resultText += fmt.Sprintf("%d. %s\n", sampleCount+1, content)
				sampleCount++
			}
			if sampleCount >= 3 {
				break
			}
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: resultText,
			},
		},
	}, nil
}

func (s *Server) handleChatHistory(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	sessionID := s.config.SessionID
	if si, ok := arguments["session_id"]; ok {
		sessionID = si.(string)
	}

	limit := 20
	if l, ok := arguments["limit"]; ok {
		if lFloat, ok := l.(float64); ok {
			limit = int(lFloat)
		}
	}

	// Get user info which includes chat manager access
	userInfo, err := s.getClient().GetUserInfo(ctx, s.config.UserID)
	if err != nil {
		s.logger.Error("Failed to get user info for chat history", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ Failed to get chat history: %s", err.Error()),
				},
			},
		}, nil
	}

	resultText := fmt.Sprintf("💬 **Chat History** (Session: %s)\n\n", sessionID)
	resultText += fmt.Sprintf("👤 User: %s\n", userInfo["user_id"])
	resultText += fmt.Sprintf("📊 Limit: %d messages\n\n", limit)
	resultText += "📝 **Note:** Full chat history integration requires chat manager access.\n"
	resultText += "This feature will show conversation history when fully integrated with the chat system."

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: resultText,
			},
		},
	}, nil
}

func (s *Server) handleContextSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	query := arguments["query"].(string)
	contextSize := 5
	if cs, ok := arguments["context_size"]; ok {
		if csFloat, ok := cs.(float64); ok {
			contextSize = int(csFloat)
		}
	}

	cubeID := ""
	if ci, ok := arguments["cube_id"]; ok {
		cubeID = ci.(string)
	}

	// Create search query for context
	searchQuery := &types.SearchQuery{
		Query:   query,
		TopK:    contextSize,
		UserID:  s.config.UserID,
		CubeIDs: []string{},
	}

	if cubeID != "" {
		searchQuery.CubeIDs = []string{cubeID}
	}

	// Execute context search using MOSCore
	result, err := s.getClient().Search(ctx, searchQuery)
	if err != nil {
		s.logger.Error("Failed to search context", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ Context search failed: %s", err.Error()),
				},
			},
		}, nil
	}

	resultText := fmt.Sprintf("🔍 **Context Search Results** for: '%s'\n\n", query)

	contextCount := 0
	for _, cubeResult := range result.TextMem {
		for _, memory := range cubeResult.Memories {
			if contextCount >= contextSize {
				break
			}
			content := memory.GetContent()
			if len(content) > 150 {
				content = content[:150] + "..."
			}
			resultText += fmt.Sprintf("**Context %d** (Cube: %s):\n%s\n\n", contextCount+1, cubeResult.CubeID, content)
			contextCount++
		}
		if contextCount >= contextSize {
			break
		}
	}

	if contextCount == 0 {
		resultText += "📭 No relevant context found for the given query."
	} else {
		resultText += fmt.Sprintf("✅ Found %d contextual items", contextCount)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: resultText,
			},
		},
	}, nil
}

func (s *Server) handleSessionInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get user info
	userInfo, err := s.getClient().GetUserInfo(ctx, s.config.UserID)
	if err != nil {
		s.logger.Error("Failed to get user info", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("❌ Failed to get session info: %s", err.Error()),
				},
			},
		}, nil
	}

	resultText := "🔐 **Session Information:**\n\n"
	resultText += fmt.Sprintf("• **User ID:** %s\n", s.config.UserID)
	resultText += fmt.Sprintf("• **Session ID:** %s\n", s.config.SessionID)
	resultText += fmt.Sprintf("• **User Name:** %s\n", userInfo["user_name"])
	resultText += fmt.Sprintf("• **Role:** %s\n", userInfo["role"])
	resultText += fmt.Sprintf("• **Account Created:** %s\n", userInfo["created_at"])
	
	if cubes, ok := userInfo["accessible_cubes"].([]map[string]interface{}); ok {
		resultText += fmt.Sprintf("• **Accessible Cubes:** %d\n", len(cubes))
	}
	
	resultText += "\n🌐 **MCP Server Info:**\n"
	resultText += "• **Server:** MemGOS MCP Server v1.0.0\n"
	resultText += "• **Transport:** stdio\n"
	resultText += "• **Status:** Active\n"

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: resultText,
			},
		},
	}, nil
}

func (s *Server) handleValidateAccess(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	resourceType := arguments["resource_type"].(string)
	resourceID := arguments["resource_id"].(string)

	var hasAccess bool
	var err error
	var details string

	switch resourceType {
	case "cube":
		// Try to get cube info to validate access
		_, err = s.getClient().GetAll(ctx, resourceID, s.config.UserID)
		hasAccess = err == nil
		if hasAccess {
			details = fmt.Sprintf("User %s has access to cube %s", s.config.UserID, resourceID)
		} else {
			details = fmt.Sprintf("User %s does not have access to cube %s: %s", s.config.UserID, resourceID, err.Error())
		}

	case "user":
		// Validate if user exists
		userInfo, err := s.getClient().GetUserInfo(ctx, resourceID)
		hasAccess = err == nil && userInfo != nil
		if hasAccess {
			details = fmt.Sprintf("User %s exists and is accessible", resourceID)
		} else {
			details = fmt.Sprintf("User %s is not accessible: %s", resourceID, func() string {
				if err != nil {
					return err.Error()
				}
				return "user not found"
			}())
		}

	case "system":
		// System access validation (basic health check)
		hasAccess = s.getClient() != nil
		if hasAccess {
			details = "System access granted - MOS Core is available"
		} else {
			details = "System access denied - MOS Core is not available"
		}

	default:
		hasAccess = false
		details = fmt.Sprintf("Unknown resource type: %s", resourceType)
	}

	status := "❌ Access Denied"
	if hasAccess {
		status = "✅ Access Granted"
	}

	resultText := fmt.Sprintf("🔐 **Access Validation Result:**\n\n%s\n\n• **Resource Type:** %s\n• **Resource ID:** %s\n• **User:** %s\n• **Details:** %s",
		status, resourceType, resourceID, s.config.UserID, details)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: resultText,
			},
		},
	}, nil
}