// Package main provides API client for MemGOS token-based authentication
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// APIClient implements MOSCore interface using HTTP API calls with token authentication
type APIClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
	logger     interfaces.Logger
}

// NewAPIClient creates a new API client for MemGOS backend
func NewAPIClient(baseURL, token string, logger interfaces.Logger) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// TestConnection tests the API connection
func (c *APIClient) TestConnection(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make health check request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// makeRequest is a helper method for making authenticated API requests
func (c *APIClient) makeRequest(ctx context.Context, method, endpoint string, payload interface{}) (*http.Response, error) {
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewReader(jsonData)
	}
	
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	
	return c.httpClient.Do(req)
}

// Initialize implements MOSCore interface
func (c *APIClient) Initialize(ctx context.Context) error {
	return c.TestConnection(ctx)
}

// RegisterMemCube implements MOSCore interface
func (c *APIClient) RegisterMemCube(ctx context.Context, cubePath, cubeID, userID string) error {
	payload := map[string]interface{}{
		"cube_path": cubePath,
		"cube_id":   cubeID,
		"user_id":   userID,
	}
	
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/cubes", payload)
	if err != nil {
		return fmt.Errorf("failed to register cube: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("register cube failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// UnregisterMemCube implements MOSCore interface
func (c *APIClient) UnregisterMemCube(ctx context.Context, cubeID, userID string) error {
	resp, err := c.makeRequest(ctx, "DELETE", fmt.Sprintf("/api/v1/cubes/%s?user_id=%s", cubeID, userID), nil)
	if err != nil {
		return fmt.Errorf("failed to unregister cube: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unregister cube failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// Search implements MOSCore interface
func (c *APIClient) Search(ctx context.Context, query *types.SearchQuery) (*types.MOSSearchResult, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/search", query)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var result types.MOSSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search result: %w", err)
	}
	
	return &result, nil
}

// Add implements MOSCore interface
func (c *APIClient) Add(ctx context.Context, request *types.AddMemoryRequest) error {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/memories", request)
	if err != nil {
		return fmt.Errorf("failed to add memory: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("add memory failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// Get implements MOSCore interface
func (c *APIClient) Get(ctx context.Context, cubeID, memoryID, userID string) (types.MemoryItem, error) {
	endpoint := fmt.Sprintf("/api/v1/memories/%s?cube_id=%s&user_id=%s", memoryID, cubeID, userID)
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get memory: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get memory failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var memory types.TextualMemoryItem
	if err := json.NewDecoder(resp.Body).Decode(&memory); err != nil {
		return nil, fmt.Errorf("failed to decode memory: %w", err)
	}
	
	return &memory, nil
}

// GetAll implements MOSCore interface
func (c *APIClient) GetAll(ctx context.Context, cubeID, userID string) (*types.MOSSearchResult, error) {
	endpoint := fmt.Sprintf("/api/v1/memories?cube_id=%s&user_id=%s", cubeID, userID)
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get all memories: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get all memories failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var result types.MOSSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode memories: %w", err)
	}
	
	return &result, nil
}

// Update implements MOSCore interface
func (c *APIClient) Update(ctx context.Context, cubeID, memoryID, userID string, memory types.MemoryItem) error {
	payload := map[string]interface{}{
		"cube_id":  cubeID,
		"user_id":  userID,
		"memory":   memory,
	}
	
	endpoint := fmt.Sprintf("/api/v1/memories/%s", memoryID)
	resp, err := c.makeRequest(ctx, "PUT", endpoint, payload)
	if err != nil {
		return fmt.Errorf("failed to update memory: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update memory failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// Delete implements MOSCore interface
func (c *APIClient) Delete(ctx context.Context, cubeID, memoryID, userID string) error {
	endpoint := fmt.Sprintf("/api/v1/memories/%s?cube_id=%s&user_id=%s", memoryID, cubeID, userID)
	resp, err := c.makeRequest(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to delete memory: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete memory failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// DeleteAll implements MOSCore interface
func (c *APIClient) DeleteAll(ctx context.Context, cubeID, userID string) error {
	endpoint := fmt.Sprintf("/api/v1/memories?cube_id=%s&user_id=%s", cubeID, userID)
	resp, err := c.makeRequest(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to delete all memories: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete all memories failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// Chat implements MOSCore interface
func (c *APIClient) Chat(ctx context.Context, request *types.ChatRequest) (*types.ChatResponse, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/chat", request)
	if err != nil {
		return nil, fmt.Errorf("failed to chat: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("chat failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var response types.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode chat response: %w", err)
	}
	
	return &response, nil
}

// CreateUser implements MOSCore interface
func (c *APIClient) CreateUser(ctx context.Context, userName string, role types.UserRole, userID string) (string, error) {
	payload := map[string]interface{}{
		"user_name": userName,
		"role":      role,
		"user_id":   userID,
	}
	
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/users", payload)
	if err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create user failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode create user response: %w", err)
	}
	
	if id, ok := result["user_id"].(string); ok {
		return id, nil
	}
	
	return userID, nil
}

// GetUserInfo implements MOSCore interface
func (c *APIClient) GetUserInfo(ctx context.Context, userID string) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("/api/v1/users/%s", userID)
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get user info failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}
	
	return userInfo, nil
}

// ListUsers implements MOSCore interface
func (c *APIClient) ListUsers(ctx context.Context) ([]*types.User, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/users", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list users failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var users []*types.User
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}
	
	return users, nil
}

// ShareCubeWithUser implements MOSCore interface
func (c *APIClient) ShareCubeWithUser(ctx context.Context, cubeID, targetUserID string) (bool, error) {
	payload := map[string]interface{}{
		"cube_id":        cubeID,
		"target_user_id": targetUserID,
	}
	
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/cubes/share", payload)
	if err != nil {
		return false, fmt.Errorf("failed to share cube: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("share cube failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return true, nil
}

// Close implements MOSCore interface
func (c *APIClient) Close() error {
	// HTTP client doesn't need explicit closing
	return nil
}

// Additional methods needed by MCP server but not in MOSCore interface

// ListCubes gets list of available cubes for user
func (c *APIClient) ListCubes(ctx context.Context, userID string) ([]*types.MemCube, error) {
	endpoint := fmt.Sprintf("/api/v1/cubes?user_id=%s", userID)
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list cubes: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list cubes failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var cubes []*types.MemCube
	if err := json.NewDecoder(resp.Body).Decode(&cubes); err != nil {
		return nil, fmt.Errorf("failed to decode cubes: %w", err)
	}
	
	return cubes, nil
}

// GetCubeInfo gets detailed information about a cube
func (c *APIClient) GetCubeInfo(ctx context.Context, cubeID, userID string) (*types.MemCube, error) {
	endpoint := fmt.Sprintf("/api/v1/cubes/%s?user_id=%s", cubeID, userID)
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get cube info: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get cube info failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var cube types.MemCube
	if err := json.NewDecoder(resp.Body).Decode(&cube); err != nil {
		return nil, fmt.Errorf("failed to decode cube info: %w", err)
	}
	
	return &cube, nil
}

// HealthCheck checks system health
func (c *APIClient) HealthCheck(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/health", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check health: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("health check failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode health status: %w", err)
	}
	
	return health, nil
}