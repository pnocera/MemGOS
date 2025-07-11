package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/memtensor/memgos/pkg/types"
)

// healthCheck provides a health check endpoint
// @Summary Health Check
// @Description Check the health status of the MemGOS API server
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (s *Server) healthCheck(c *gin.Context) {
	startTime := time.Now() // This should be stored when server starts
	uptime := time.Since(startTime)

	health := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   "1.0.0", // Should come from build info
		Uptime:    uptime.String(),
		Checks: map[string]string{
			"core":     "ok",
			"database": "ok", // Check actual database status
			"memory":   "ok",
		},
	}

	c.JSON(http.StatusOK, health)
}

// redirectToDocs redirects root to API documentation
func (s *Server) redirectToDocs(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/docs")
}

// setConfig handles configuration updates
func (s *Server) setConfig(c *gin.Context) {
	var req ConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request format",
			Error:   err.Error(),
		})
		return
	}

	// TODO: Apply configuration changes to MOS core
	// For now, just acknowledge receipt

	resp := ConfigResponse{
		Code:    http.StatusOK,
		Message: "Configuration updated successfully",
		Data:    nil,
	}

	c.JSON(http.StatusOK, resp)
}

// createUser handles user creation
func (s *Server) createUser(c *gin.Context) {
	var req UserCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request format",
			Error:   err.Error(),
		})
		return
	}

	userName := ""
	if req.UserName != nil {
		userName = *req.UserName
	}

	userID, err := s.mosCore.CreateUser(c.Request.Context(), userName, req.Role, req.UserID)
	if err != nil {
		s.handleError(c, "Failed to create user", err)
		return
	}

	resp := UserResponse{
		Code:    http.StatusOK,
		Message: "User created successfully",
		Data: &map[string]interface{}{
			"user_id": userID,
		},
	}

	c.JSON(http.StatusOK, resp)
}

// listUsers handles listing all users
func (s *Server) listUsers(c *gin.Context) {
	users, err := s.mosCore.ListUsers(c.Request.Context())
	if err != nil {
		s.handleError(c, "Failed to list users", err)
		return
	}

	// Convert users to response format
	userList := make([]map[string]interface{}, len(users))
	for i, user := range users {
		userList[i] = map[string]interface{}{
			"user_id":   user.ID,
			"user_name": user.Name,
			"role":      user.Role,
			"created_at": user.CreatedAt,
		}
	}

	resp := UserListResponse{
		Code:    http.StatusOK,
		Message: "Users retrieved successfully",
		Data:    &userList,
	}

	c.JSON(http.StatusOK, resp)
}

// getUserInfo handles getting current user information
func (s *Server) getUserInfo(c *gin.Context) {
	userID := s.getUserID(c)

	userInfo, err := s.mosCore.GetUserInfo(c.Request.Context(), userID)
	if err != nil {
		s.handleError(c, "Failed to get user info", err)
		return
	}

	resp := UserResponse{
		Code:    http.StatusOK,
		Message: "User info retrieved successfully",
		Data:    &userInfo,
	}

	c.JSON(http.StatusOK, resp)
}

// registerMemCube handles memory cube registration
func (s *Server) registerMemCube(c *gin.Context) {
	var req MemCubeRegister
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request format",
			Error:   err.Error(),
		})
		return
	}

	userID := s.getUserID(c)
	if req.UserID != nil {
		userID = *req.UserID
	}

	cubeID := ""
	if req.MemCubeID != nil {
		cubeID = *req.MemCubeID
	}

	err := s.mosCore.RegisterMemCube(c.Request.Context(), req.MemCubeNameOrPath, cubeID, userID)
	if err != nil {
		s.handleError(c, "Failed to register memory cube", err)
		return
	}

	resp := SimpleResponse{
		Code:    http.StatusOK,
		Message: "MemCube registered successfully",
	}

	c.JSON(http.StatusOK, resp)
}

// unregisterMemCube handles memory cube unregistration
func (s *Server) unregisterMemCube(c *gin.Context) {
	memCubeID := c.Param("mem_cube_id")
	userID := c.Query("user_id")
	if userID == "" {
		userID = s.getUserID(c)
	}

	err := s.mosCore.UnregisterMemCube(c.Request.Context(), memCubeID, userID)
	if err != nil {
		s.handleError(c, "Failed to unregister memory cube", err)
		return
	}

	resp := SimpleResponse{
		Code:    http.StatusOK,
		Message: "MemCube unregistered successfully",
	}

	c.JSON(http.StatusOK, resp)
}

// shareCube handles sharing a cube with another user
func (s *Server) shareCube(c *gin.Context) {
	cubeID := c.Param("cube_id")

	var req CubeShare
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request format",
			Error:   err.Error(),
		})
		return
	}

	success, err := s.mosCore.ShareCubeWithUser(c.Request.Context(), cubeID, req.TargetUserID)
	if err != nil {
		s.handleError(c, "Failed to share cube", err)
		return
	}

	if !success {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Failed to share cube",
			Error:   "Share operation returned false",
		})
		return
	}

	resp := SimpleResponse{
		Code:    http.StatusOK,
		Message: "Cube shared successfully",
	}

	c.JSON(http.StatusOK, resp)
}

// addMemory handles adding memories
// @Summary Add Memory
// @Description Add new memories to the system
// @Tags memories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body MemoryCreate true "Memory creation parameters"
// @Success 200 {object} SimpleResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /memories [post]
func (s *Server) addMemory(c *gin.Context) {
	var req MemoryCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request format",
			Error:   err.Error(),
		})
		return
	}

	// Validate that at least one type of content is provided
	if len(req.Messages) == 0 && req.MemoryContent == nil && req.DocPath == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Either messages, memory_content, or doc_path must be provided",
		})
		return
	}

	userID := s.getUserID(c)
	if req.UserID != nil {
		userID = *req.UserID
	}

	memCubeID := ""
	if req.MemCubeID != nil {
		memCubeID = *req.MemCubeID
	}

	// Build add request
	addReq := &types.AddMemoryRequest{
		UserID:    userID,
		MemCubeID: memCubeID,
	}

	if len(req.Messages) > 0 {
		// Convert messages to the expected format
		messages := make([]map[string]interface{}, len(req.Messages))
		for i, msg := range req.Messages {
			messages[i] = map[string]interface{}{
				"role":    msg.Role,
				"content": msg.Content,
			}
		}
		addReq.Messages = messages
	} else if req.MemoryContent != nil {
		addReq.MemoryContent = req.MemoryContent
	} else if req.DocPath != nil {
		addReq.DocPath = req.DocPath
	}

	err := s.mosCore.Add(c.Request.Context(), addReq)
	if err != nil {
		s.handleError(c, "Failed to add memory", err)
		return
	}

	resp := SimpleResponse{
		Code:    http.StatusOK,
		Message: "Memories added successfully",
	}

	c.JSON(http.StatusOK, resp)
}

// getAllMemories handles retrieving all memories
// @Summary Get All Memories
// @Description Retrieve all memories from a memory cube
// @Tags memories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param mem_cube_id query string false "Memory cube ID"
// @Param user_id query string false "User ID"
// @Success 200 {object} MemoryResponse
// @Failure 500 {object} ErrorResponse
// @Router /memories [get]
func (s *Server) getAllMemories(c *gin.Context) {
	memCubeID := c.Query("mem_cube_id")
	userID := c.Query("user_id")
	if userID == "" {
		userID = s.getUserID(c)
	}

	// Parameters are extracted directly from query and path

	result, err := s.mosCore.GetAll(c.Request.Context(), memCubeID, userID)
	if err != nil {
		s.handleError(c, "Failed to get memories", err)
		return
	}

	// Convert result to map for response
	responseData := map[string]interface{}{
		"text_memories":       result.TextMem,
		"activation_memories": result.ActMem,
		"parametric_memories": result.ParaMem,
	}

	resp := MemoryResponse{
		Code:    http.StatusOK,
		Message: "Memories retrieved successfully",
		Data:    &responseData,
	}

	c.JSON(http.StatusOK, resp)
}

// getMemory handles retrieving a specific memory
func (s *Server) getMemory(c *gin.Context) {
	memCubeID := c.Param("mem_cube_id")
	memoryID := c.Param("memory_id")
	userID := c.Query("user_id")
	if userID == "" {
		userID = s.getUserID(c)
	}

	// Parameters are extracted directly from query and path

	result, err := s.mosCore.Get(c.Request.Context(), memCubeID, memoryID, userID)
	if err != nil {
		s.handleError(c, "Failed to get memory", err)
		return
	}

	// Convert result to map for response
	responseData := map[string]interface{}{
		"id":         result.GetID(),
		"content":    result.GetContent(),
		"metadata":   result.GetMetadata(),
		"created_at": result.GetCreatedAt(),
		"updated_at": result.GetUpdatedAt(),
	}

	resp := MemoryResponse{
		Code:    http.StatusOK,
		Message: "Memory retrieved successfully",
		Data:    &responseData,
	}

	c.JSON(http.StatusOK, resp)
}

// updateMemory handles updating a memory
func (s *Server) updateMemory(c *gin.Context) {
	memCubeID := c.Param("mem_cube_id")
	memoryID := c.Param("memory_id")
	userID := c.Query("user_id")
	if userID == "" {
		userID = s.getUserID(c)
	}

	var updatedMemory map[string]interface{}
	if err := c.ShouldBindJSON(&updatedMemory); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request format",
			Error:   err.Error(),
		})
		return
	}

	// Parameters are extracted directly from query and path

	// Convert map to MemoryItem - create a textual memory item as placeholder
	// TODO: Implement proper conversion from map to specific memory item type based on memory type
	textualMemory := &types.TextualMemoryItem{
		ID:        memoryID,
		Memory:    fmt.Sprintf("%v", updatedMemory["content"]), // Extract content field
		Metadata:  updatedMemory,
		UpdatedAt: time.Now(),
	}
	err := s.mosCore.Update(c.Request.Context(), memCubeID, memoryID, userID, textualMemory)
	if err != nil {
		s.handleError(c, "Failed to update memory", err)
		return
	}

	resp := SimpleResponse{
		Code:    http.StatusOK,
		Message: "Memory updated successfully",
	}

	c.JSON(http.StatusOK, resp)
}

// deleteMemory handles deleting a specific memory
func (s *Server) deleteMemory(c *gin.Context) {
	memCubeID := c.Param("mem_cube_id")
	memoryID := c.Param("memory_id")
	userID := c.Query("user_id")
	if userID == "" {
		userID = s.getUserID(c)
	}

	// Parameters are extracted directly from query and path

	err := s.mosCore.Delete(c.Request.Context(), memCubeID, memoryID, userID)
	if err != nil {
		s.handleError(c, "Failed to delete memory", err)
		return
	}

	resp := SimpleResponse{
		Code:    http.StatusOK,
		Message: "Memory deleted successfully",
	}

	c.JSON(http.StatusOK, resp)
}

// deleteAllMemories handles deleting all memories from a cube
func (s *Server) deleteAllMemories(c *gin.Context) {
	memCubeID := c.Param("mem_cube_id")
	userID := c.Query("user_id")
	if userID == "" {
		userID = s.getUserID(c)
	}

	// Parameters are extracted directly from query and path

	err := s.mosCore.DeleteAll(c.Request.Context(), memCubeID, userID)
	if err != nil {
		s.handleError(c, "Failed to delete all memories", err)
		return
	}

	resp := SimpleResponse{
		Code:    http.StatusOK,
		Message: "All memories deleted successfully",
	}

	c.JSON(http.StatusOK, resp)
}

// searchMemories handles memory search
// @Summary Search Memories
// @Description Search through memories using semantic search
// @Tags memories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body SearchRequest true "Search parameters"
// @Success 200 {object} SearchResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /search [post]
func (s *Server) searchMemories(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request format",
			Error:   err.Error(),
		})
		return
	}

	userID := s.getUserID(c)
	if req.UserID != nil {
		userID = *req.UserID
	}

	topK := 5
	if req.TopK != nil {
		topK = *req.TopK
	}

	searchQuery := &types.SearchQuery{
		Query:          req.Query,
		UserID:         userID,
		InstallCubeIDs: req.InstallCubeIDs,
		TopK:           topK,
	}

	result, err := s.mosCore.Search(c.Request.Context(), searchQuery)
	if err != nil {
		s.handleError(c, "Failed to search memories", err)
		return
	}

	// Convert result to response format
	responseData := map[string]interface{}{
		"text_memories":       result.TextMem,
		"activation_memories": result.ActMem,
		"parametric_memories": result.ParaMem,
		"query":               req.Query,
		"total_results":       len(result.TextMem) + len(result.ActMem) + len(result.ParaMem),
	}

	resp := SearchResponse{
		Code:    http.StatusOK,
		Message: "Search completed successfully",
		Data:    &responseData,
	}

	c.JSON(http.StatusOK, resp)
}

// chat handles chat requests
// @Summary Chat with AI
// @Description Chat with AI using memory-enhanced responses
// @Tags chat
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ChatRequest true "Chat request"
// @Success 200 {object} ChatResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /chat [post]
func (s *Server) chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request format",
			Error:   err.Error(),
		})
		return
	}

	userID := s.getUserID(c)
	if req.UserID != nil {
		userID = *req.UserID
	}

	chatReq := &types.ChatRequest{
		Query:  req.Query,
		UserID: userID,
	}

	if req.Context != nil {
		// Convert map[string]string to map[string]interface{}
		context := make(map[string]interface{})
		for k, v := range req.Context {
			context[k] = v
		}
		chatReq.Context = context
	}

	if req.MaxTokens != nil {
		chatReq.MaxTokens = req.MaxTokens
	}

	if req.TopK != nil {
		chatReq.TopK = req.TopK
	}

	response, err := s.mosCore.Chat(c.Request.Context(), chatReq)
	if err != nil {
		s.handleError(c, "Failed to chat", err)
		return
	}

	if response.Response == "" {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "No response generated",
		})
		return
	}

	resp := ChatResponse{
		Code:    http.StatusOK,
		Message: "Chat response generated",
		Data:    &response.Response,
	}

	c.JSON(http.StatusOK, resp)
}

// getMetrics handles metrics endpoint
func (s *Server) getMetrics(c *gin.Context) {
	metrics := MetricsResponse{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Uptime:    "unknown", // Should track actual uptime
		Metrics: map[string]interface{}{
			"requests_total":   0,
			"requests_success": 0,
			"requests_error":   0,
			"memory_usage":     "unknown",
			"active_sessions":  0,
		},
	}

	c.JSON(http.StatusOK, metrics)
}

// serveDocs serves API documentation
func (s *Server) serveDocs(c *gin.Context) {
	// TODO: Serve Swagger UI or similar documentation
	c.JSON(http.StatusOK, gin.H{
		"message": "API Documentation",
		"endpoints": gin.H{
			"health":       "GET /health",
			"configure":    "POST /configure",
			"users":        "GET|POST /users",
			"user_info":    "GET /users/me",
			"mem_cubes":    "POST /mem_cubes",
			"memories":     "GET|POST|PUT|DELETE /memories",
			"search":       "POST /search",
			"chat":         "POST /chat",
			"metrics":      "GET /metrics",
			"openapi_spec": "GET /openapi.json",
		},
	})
}

// getOpenAPISpec returns the OpenAPI specification
func (s *Server) getOpenAPISpec(c *gin.Context) {
	// TODO: Generate or serve actual OpenAPI spec
	spec := map[string]interface{}{
		"openapi": "3.1.0",
		"info": map[string]interface{}{
			"title":       "MemGOS REST API",
			"description": "A REST API for managing and searching memories using MemGOS.",
			"version":     "1.0.0",
		},
		"paths": map[string]interface{}{
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Health check",
					"description": "Check API server health",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Server is healthy",
						},
					},
				},
			},
			// Add more endpoints as needed
		},
	}

	c.JSON(http.StatusOK, spec)
}

// Helper functions

// getUserID extracts user ID from context or request
func (s *Server) getUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(string); ok {
			return uid
		}
	}
	return s.config.UserID // fallback to default
}

// handleError provides consistent error handling
func (s *Server) handleError(c *gin.Context, message string, err error) {
	requestID := c.GetString("request_id")
	
	s.logger.Error(message, err, map[string]interface{}{
		"request_id": requestID,
		"path":       c.Request.URL.Path,
		"method":     c.Request.Method,
	})

	errorResp := ErrorResponse{
		Code:    http.StatusInternalServerError,
		Message: message,
		Error:   err.Error(),
		Details: fmt.Sprintf("Request ID: %s", requestID),
	}

	c.JSON(http.StatusInternalServerError, errorResp)
}

// parseIntParam safely parses integer parameters
func (s *Server) parseIntParam(c *gin.Context, param string, defaultValue int) int {
	valueStr := c.Query(param)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}