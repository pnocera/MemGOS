package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

// OpenAPISpec returns the complete OpenAPI 3.1.0 specification for the MemGOS API
func (s *Server) getCompleteOpenAPISpec(c *gin.Context) {
	spec := map[string]interface{}{
		"openapi": "3.1.0",
		"info": map[string]interface{}{
			"title":       "MemGOS REST API",
			"description": "A REST API for managing and searching memories using MemGOS.",
			"version":     "1.0.0",
			"contact": map[string]interface{}{
				"name": "MemGOS Team",
				"url":  "https://github.com/memtensor/memgos",
			},
			"license": map[string]interface{}{
				"name": "MIT",
				"url":  "https://opensource.org/licenses/MIT",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url":         "http://localhost:8080",
				"description": "Development server",
			},
		},
		"paths": s.getOpenAPIPaths(),
		"components": s.getOpenAPIComponents(),
	}

	c.JSON(http.StatusOK, spec)
}

// getOpenAPIPaths returns all API paths for OpenAPI spec
func (s *Server) getOpenAPIPaths() map[string]interface{} {
	return map[string]interface{}{
		"/health": map[string]interface{}{
			"get": map[string]interface{}{
				"summary":     "Health Check",
				"description": "Check the health status of the API server",
				"tags":        []string{"Health"},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Server is healthy",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/HealthResponse",
								},
							},
						},
					},
				},
			},
		},
		"/configure": map[string]interface{}{
			"post": map[string]interface{}{
				"summary":     "Configure MemGOS",
				"description": "Update MemGOS configuration",
				"tags":        []string{"Configuration"},
				"requestBody": map[string]interface{}{
					"required": true,
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": "#/components/schemas/ConfigRequest",
							},
						},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Configuration updated successfully",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/ConfigResponse",
								},
							},
						},
					},
					"400": s.getErrorResponse("Invalid request format"),
				},
			},
		},
		"/users": map[string]interface{}{
			"get": map[string]interface{}{
				"summary":     "List Users",
				"description": "Get a list of all users",
				"tags":        []string{"Users"},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Users retrieved successfully",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/UserListResponse",
								},
							},
						},
					},
				},
			},
			"post": map[string]interface{}{
				"summary":     "Create User",
				"description": "Create a new user",
				"tags":        []string{"Users"},
				"requestBody": map[string]interface{}{
					"required": true,
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": "#/components/schemas/UserCreate",
							},
						},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "User created successfully",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/UserResponse",
								},
							},
						},
					},
					"400": s.getErrorResponse("Invalid request format"),
				},
			},
		},
		"/users/me": map[string]interface{}{
			"get": map[string]interface{}{
				"summary":     "Get Current User Info",
				"description": "Get current user information including accessible cubes",
				"tags":        []string{"Users"},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "User info retrieved successfully",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/UserResponse",
								},
							},
						},
					},
				},
			},
		},
		"/mem_cubes": map[string]interface{}{
			"post": map[string]interface{}{
				"summary":     "Register MemCube",
				"description": "Register a new memory cube",
				"tags":        []string{"Memory Cubes"},
				"requestBody": map[string]interface{}{
					"required": true,
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": "#/components/schemas/MemCubeRegister",
							},
						},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "MemCube registered successfully",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/SimpleResponse",
								},
							},
						},
					},
					"400": s.getErrorResponse("Invalid request format"),
				},
			},
		},
		"/memories": map[string]interface{}{
			"get": map[string]interface{}{
				"summary":     "Get All Memories",
				"description": "Retrieve all memories from a MemCube",
				"tags":        []string{"Memories"},
				"parameters": []map[string]interface{}{
					{
						"name":        "mem_cube_id",
						"in":          "query",
						"required":    false,
						"description": "Memory cube ID to filter by",
						"schema": map[string]interface{}{
							"type": "string",
						},
					},
					{
						"name":        "user_id",
						"in":          "query",
						"required":    false,
						"description": "User ID to filter by",
						"schema": map[string]interface{}{
							"type": "string",
						},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Memories retrieved successfully",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/MemoryResponse",
								},
							},
						},
					},
				},
			},
			"post": map[string]interface{}{
				"summary":     "Create Memories",
				"description": "Store new memories in a MemCube",
				"tags":        []string{"Memories"},
				"requestBody": map[string]interface{}{
					"required": true,
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": "#/components/schemas/MemoryCreate",
							},
						},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Memories added successfully",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/SimpleResponse",
								},
							},
						},
					},
					"400": s.getErrorResponse("Invalid request format"),
				},
			},
		},
		"/search": map[string]interface{}{
			"post": map[string]interface{}{
				"summary":     "Search Memories",
				"description": "Search for memories across MemCubes",
				"tags":        []string{"Search"},
				"requestBody": map[string]interface{}{
					"required": true,
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": "#/components/schemas/SearchRequest",
							},
						},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Search completed successfully",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/SearchResponse",
								},
							},
						},
					},
					"400": s.getErrorResponse("Invalid request format"),
				},
			},
		},
		"/chat": map[string]interface{}{
			"post": map[string]interface{}{
				"summary":     "Chat with MemGOS",
				"description": "Send a chat message to MemGOS and get a response",
				"tags":        []string{"Chat"},
				"requestBody": map[string]interface{}{
					"required": true,
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": "#/components/schemas/ChatRequest",
							},
						},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Chat response generated",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/ChatResponse",
								},
							},
						},
					},
					"400": s.getErrorResponse("Invalid request format"),
				},
			},
		},
		"/metrics": map[string]interface{}{
			"get": map[string]interface{}{
				"summary":     "Get Metrics",
				"description": "Get API server metrics and statistics",
				"tags":        []string{"Monitoring"},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Metrics retrieved successfully",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/MetricsResponse",
								},
							},
						},
					},
				},
			},
		},
	}
}

// getOpenAPIComponents returns all schema components
func (s *Server) getOpenAPIComponents() map[string]interface{} {
	return map[string]interface{}{
		"schemas": map[string]interface{}{
			"BaseRequest": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]interface{}{
						"type":        "string",
						"description": "User ID for the request",
						"example":     "user123",
					},
				},
			},
			"BaseResponse": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"code": map[string]interface{}{
						"type":        "integer",
						"description": "Response status code",
						"example":     200,
					},
					"message": map[string]interface{}{
						"type":        "string",
						"description": "Response message",
						"example":     "Operation successful",
					},
					"data": map[string]interface{}{
						"description": "Response data",
					},
				},
				"required": []string{"code", "message"},
			},
			"SimpleResponse": map[string]interface{}{
				"allOf": []map[string]interface{}{
					{"$ref": "#/components/schemas/BaseResponse"},
					{
						"type": "object",
						"properties": map[string]interface{}{
							"data": map[string]interface{}{
								"type": "null",
							},
						},
					},
				},
			},
			"HealthResponse": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{
						"type":        "string",
						"description": "Health status",
						"example":     "healthy",
					},
					"timestamp": map[string]interface{}{
						"type":        "string",
						"format":      "date-time",
						"description": "Timestamp of health check",
					},
					"version": map[string]interface{}{
						"type":        "string",
						"description": "API version",
						"example":     "1.0.0",
					},
					"uptime": map[string]interface{}{
						"type":        "string",
						"description": "Server uptime",
						"example":     "2h30m45s",
					},
					"checks": map[string]interface{}{
						"type":        "object",
						"description": "Component health checks",
						"additionalProperties": map[string]interface{}{
							"type": "string",
						},
					},
				},
				"required": []string{"status", "timestamp", "version"},
			},
			"ErrorResponse": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"code": map[string]interface{}{
						"type":        "integer",
						"description": "Error code",
						"example":     400,
					},
					"message": map[string]interface{}{
						"type":        "string",
						"description": "Error message",
						"example":     "Bad request",
					},
					"error": map[string]interface{}{
						"type":        "string",
						"description": "Detailed error information",
					},
					"details": map[string]interface{}{
						"type":        "string",
						"description": "Additional error details",
					},
				},
				"required": []string{"code", "message"},
			},
			"Message": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"role": map[string]interface{}{
						"type":        "string",
						"description": "Role of the message sender",
						"enum":        []string{"user", "assistant"},
						"example":     "user",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Message content",
						"example":     "Hello, how can I help you?",
					},
				},
				"required": []string{"role", "content"},
			},
			"MemoryCreate": map[string]interface{}{
				"allOf": []map[string]interface{}{
					{"$ref": "#/components/schemas/BaseRequest"},
					{
						"type": "object",
						"properties": map[string]interface{}{
							"messages": map[string]interface{}{
								"type":        "array",
								"description": "List of messages to store",
								"items": map[string]interface{}{
									"$ref": "#/components/schemas/Message",
								},
							},
							"mem_cube_id": map[string]interface{}{
								"type":        "string",
								"description": "Memory cube ID",
								"example":     "cube123",
							},
							"memory_content": map[string]interface{}{
								"type":        "string",
								"description": "Content to store as memory",
								"example":     "This is important information",
							},
							"doc_path": map[string]interface{}{
								"type":        "string",
								"description": "Path to document to store",
								"example":     "/path/to/document.txt",
							},
						},
					},
				},
			},
			"SearchRequest": map[string]interface{}{
				"allOf": []map[string]interface{}{
					{"$ref": "#/components/schemas/BaseRequest"},
					{
						"type": "object",
						"properties": map[string]interface{}{
							"query": map[string]interface{}{
								"type":        "string",
								"description": "Search query",
								"example":     "How to implement a feature?",
							},
							"install_cube_ids": map[string]interface{}{
								"type":        "array",
								"description": "List of cube IDs to search in",
								"items": map[string]interface{}{
									"type": "string",
								},
								"example": []string{"cube123", "cube456"},
							},
							"top_k": map[string]interface{}{
								"type":        "integer",
								"description": "Maximum number of results to return",
								"example":     5,
								"minimum":     1,
								"maximum":     100,
							},
						},
						"required": []string{"query"},
					},
				},
			},
			"ChatRequest": map[string]interface{}{
				"allOf": []map[string]interface{}{
					{"$ref": "#/components/schemas/BaseRequest"},
					{
						"type": "object",
						"properties": map[string]interface{}{
							"query": map[string]interface{}{
								"type":        "string",
								"description": "Chat query message",
								"example":     "What is the latest update?",
							},
							"context": map[string]interface{}{
								"type":        "object",
								"description": "Additional context for the chat",
								"additionalProperties": map[string]interface{}{
									"type": "string",
								},
							},
							"max_tokens": map[string]interface{}{
								"type":        "integer",
								"description": "Maximum tokens in response",
								"example":     1000,
								"minimum":     1,
							},
							"top_k": map[string]interface{}{
								"type":        "integer",
								"description": "Number of memories to retrieve",
								"example":     5,
							},
						},
						"required": []string{"query"},
					},
				},
			},
			// Add more schema definitions as needed
			"ConfigResponse": map[string]interface{}{
				"allOf": []map[string]interface{}{
					{"$ref": "#/components/schemas/BaseResponse"},
				},
			},
			"MemoryResponse": map[string]interface{}{
				"allOf": []map[string]interface{}{
					{"$ref": "#/components/schemas/BaseResponse"},
					{
						"type": "object",
						"properties": map[string]interface{}{
							"data": map[string]interface{}{
								"type":        "object",
								"description": "Memory data",
							},
						},
					},
				},
			},
			"SearchResponse": map[string]interface{}{
				"allOf": []map[string]interface{}{
					{"$ref": "#/components/schemas/BaseResponse"},
					{
						"type": "object",
						"properties": map[string]interface{}{
							"data": map[string]interface{}{
								"type":        "object",
								"description": "Search results",
								"properties": map[string]interface{}{
									"text_memories": map[string]interface{}{
										"type":        "array",
										"description": "Textual memory results",
									},
									"activation_memories": map[string]interface{}{
										"type":        "array",
										"description": "Activation memory results",
									},
									"parametric_memories": map[string]interface{}{
										"type":        "array",
										"description": "Parametric memory results",
									},
									"total_results": map[string]interface{}{
										"type":        "integer",
										"description": "Total number of results",
									},
								},
							},
						},
					},
				},
			},
			"ChatResponse": map[string]interface{}{
				"allOf": []map[string]interface{}{
					{"$ref": "#/components/schemas/BaseResponse"},
					{
						"type": "object",
						"properties": map[string]interface{}{
							"data": map[string]interface{}{
								"type":        "string",
								"description": "Chat response text",
							},
						},
					},
				},
			},
			"MetricsResponse": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"timestamp": map[string]interface{}{
						"type":        "string",
						"format":      "date-time",
						"description": "Metrics timestamp",
					},
					"uptime": map[string]interface{}{
						"type":        "string",
						"description": "Server uptime",
					},
					"metrics": map[string]interface{}{
						"type":        "object",
						"description": "Performance metrics",
						"additionalProperties": true,
					},
				},
				"required": []string{"timestamp", "metrics"},
			},
		},
		"securitySchemes": map[string]interface{}{
			"BearerAuth": map[string]interface{}{
				"type":         "http",
				"scheme":       "bearer",
				"bearerFormat": "JWT",
				"description":  "JWT token authentication",
			},
			"ApiKeyAuth": map[string]interface{}{
				"type":        "apiKey",
				"in":          "header",
				"name":        "X-API-Key",
				"description": "API key authentication",
			},
		},
	}
}

// getErrorResponse creates a standard error response schema
func (s *Server) getErrorResponse(description string) map[string]interface{} {
	return map[string]interface{}{
		"description": description,
		"content": map[string]interface{}{
			"application/json": map[string]interface{}{
				"schema": map[string]interface{}{
					"$ref": "#/components/schemas/ErrorResponse",
				},
			},
		},
	}
}

// generateOpenAPIJSON generates and saves the OpenAPI spec to a file
func (s *Server) generateOpenAPIJSON() ([]byte, error) {
	spec := map[string]interface{}{
		"openapi": "3.1.0",
		"info": map[string]interface{}{
			"title":       "MemGOS REST API",
			"description": "A REST API for managing and searching memories using MemGOS.",
			"version":     "1.0.0",
		},
		"paths":      s.getOpenAPIPaths(),
		"components": s.getOpenAPIComponents(),
	}

	return json.MarshalIndent(spec, "", "  ")
}