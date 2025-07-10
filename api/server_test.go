package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/memtensor/memgos/pkg/config"
	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMOSCore is a mock implementation of MOSCore for testing
type MockMOSCore struct {
	mock.Mock
}

func (m *MockMOSCore) Initialize(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockMOSCore) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockMOSCore) Add(ctx context.Context, req *types.AddMemoryRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockMOSCore) Search(ctx context.Context, query *types.SearchQuery) (*types.MOSSearchResult, error) {
	args := m.Called(ctx, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.MOSSearchResult), args.Error(1)
}

func (m *MockMOSCore) Chat(ctx context.Context, req *types.ChatRequest) (*types.ChatResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*types.ChatResponse), args.Error(1)
}

func (m *MockMOSCore) RegisterMemCube(ctx context.Context, cubePath, cubeID, userID string) error {
	args := m.Called(ctx, cubePath, cubeID, userID)
	return args.Error(0)
}

func (m *MockMOSCore) UnregisterMemCube(ctx context.Context, cubeID, userID string) error {
	args := m.Called(ctx, cubeID, userID)
	return args.Error(0)
}

func (m *MockMOSCore) CreateUser(ctx context.Context, userName string, role types.UserRole, userID string) (string, error) {
	args := m.Called(ctx, userName, role, userID)
	return args.String(0), args.Error(1)
}

// ListUsers is not in MOSCore interface, remove this method

func (m *MockMOSCore) GetUserInfo(ctx context.Context, userID string) (map[string]interface{}, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// ShareCubeWithUser is not in MOSCore interface, remove this method

func (m *MockMOSCore) GetAll(ctx context.Context, cubeID, userID string) (*types.MOSSearchResult, error) {
	args := m.Called(ctx, cubeID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.MOSSearchResult), args.Error(1)
}

func (m *MockMOSCore) Get(ctx context.Context, cubeID, memoryID, userID string) (types.MemoryItem, error) {
	args := m.Called(ctx, cubeID, memoryID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(types.MemoryItem), args.Error(1)
}

func (m *MockMOSCore) Update(ctx context.Context, cubeID, memoryID, userID string, memory types.MemoryItem) error {
	args := m.Called(ctx, cubeID, memoryID, userID, memory)
	return args.Error(0)
}

func (m *MockMOSCore) Delete(ctx context.Context, cubeID, memoryID, userID string) error {
	args := m.Called(ctx, cubeID, memoryID, userID)
	return args.Error(0)
}

func (m *MockMOSCore) DeleteAll(ctx context.Context, cubeID, userID string) error {
	args := m.Called(ctx, cubeID, userID)
	return args.Error(0)
}

// MockLogger is a mock implementation of Logger for testing
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(message string, fields ...map[string]interface{}) {
	m.Called(message, fields)
}

func (m *MockLogger) Info(message string, fields ...map[string]interface{}) {
	m.Called(message, fields)
}

func (m *MockLogger) Warn(message string, fields ...map[string]interface{}) {
	m.Called(message, fields)
}

func (m *MockLogger) Error(message string, err interface{}, fields ...map[string]interface{}) {
	m.Called(message, err, fields)
}

// Test setup helpers

func setupTestServer() (*Server, *MockMOSCore, *MockLogger) {
	mockCore := &MockMOSCore{}
	mockLogger := &MockLogger{}

	// Allow Info calls for server setup
	mockLogger.On("Info", mock.AnythingOfType("string"), mock.Anything).Maybe()
	mockLogger.On("Debug", mock.AnythingOfType("string"), mock.Anything).Maybe()
	mockLogger.On("Error", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Maybe()

	cfg := &config.MOSConfig{
		UserID:    "test-user",
		SessionID: "test-session",
		APIPort:   8080,
	}

	server := NewServer(mockCore, cfg, mockLogger)
	return server, mockCore, mockLogger
}

func performRequest(router *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(bodyBytes)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// Test cases

func TestHealthCheck(t *testing.T) {
	server, _, _ := setupTestServer()

	w := performRequest(server.router, "GET", "/health", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var response HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response.Status)
	assert.NotEmpty(t, response.Timestamp)
	assert.NotEmpty(t, response.Version)
}

func TestCreateUser(t *testing.T) {
	server, mockCore, _ := setupTestServer()

	// Setup mock expectations
	mockCore.On("CreateUser", mock.Anything, "john_doe", types.UserRoleUser, "user123").Return("user123", nil)

	reqBody := UserCreate{
		UserName: stringPtr("john_doe"),
		Role:     types.UserRoleUser,
		UserID:   "user123",
	}

	w := performRequest(server.router, "POST", "/users", reqBody)

	assert.Equal(t, http.StatusOK, w.Code)

	var response UserResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "User created successfully", response.Message)
	assert.NotNil(t, response.Data)

	mockCore.AssertExpectations(t)
}

func TestListUsers(t *testing.T) {
	server, mockCore, _ := setupTestServer()

	// Setup mock expectations
	mockUsers := []types.User{
		{
			ID:        "user1",
			Name:      "John Doe",
			Role:      types.UserRoleUser,
			CreatedAt: time.Now(),
		},
		{
			ID:        "user2",
			Name:      "Jane Smith",
			Role:      types.UserRoleAdmin,
			CreatedAt: time.Now(),
		},
	}
	mockCore.On("ListUsers", mock.Anything).Return(mockUsers, nil)

	w := performRequest(server.router, "GET", "/users", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var response UserListResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "Users retrieved successfully", response.Message)
	assert.NotNil(t, response.Data)
	assert.Len(t, *response.Data, 2)

	mockCore.AssertExpectations(t)
}

func TestAddMemory(t *testing.T) {
	server, mockCore, _ := setupTestServer()

	// Setup mock expectations
	mockCore.On("Add", mock.Anything, mock.AnythingOfType("*types.AddMemoryRequest")).Return(nil)

	reqBody := MemoryCreate{
		MemoryContent: stringPtr("This is a test memory"),
		MemCubeID:     stringPtr("cube123"),
	}

	w := performRequest(server.router, "POST", "/memories", reqBody)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SimpleResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "Memories added successfully", response.Message)

	mockCore.AssertExpectations(t)
}

func TestSearchMemories(t *testing.T) {
	server, mockCore, _ := setupTestServer()

	// Setup mock expectations
	mockResult := &types.MOSSearchResult{
		TextMemories: []types.MemoryItem{
			&types.TextualMemoryItem{ID: "mem1", Memory: "test content"},
		},
		ActivationMemories: []types.MemoryItem{},
		ParametricMemories: []types.MemoryItem{},
	}
	mockCore.On("Search", mock.Anything, mock.AnythingOfType("*types.SearchQuery")).Return(mockResult, nil)

	reqBody := SearchRequest{
		Query: "test query",
		TopK:  intPtr(5),
	}

	w := performRequest(server.router, "POST", "/search", reqBody)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SearchResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "Search completed successfully", response.Message)
	assert.NotNil(t, response.Data)

	mockCore.AssertExpectations(t)
}

func TestChat(t *testing.T) {
	server, mockCore, _ := setupTestServer()

	// Setup mock expectations
	mockResponse := &types.ChatResponse{
		Response: "Hello! How can I help you?",
	}
	mockCore.On("Chat", mock.Anything, mock.AnythingOfType("*types.ChatRequest")).Return(mockResponse, nil)

	reqBody := ChatRequest{
		Query: "Hello",
	}

	w := performRequest(server.router, "POST", "/chat", reqBody)

	assert.Equal(t, http.StatusOK, w.Code)

	var response ChatResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "Chat response generated", response.Message)
	assert.NotNil(t, response.Data)
	assert.Equal(t, "Hello! How can I help you?", *response.Data)

	mockCore.AssertExpectations(t)
}

func TestRegisterMemCube(t *testing.T) {
	server, mockCore, _ := setupTestServer()

	// Setup mock expectations
	mockCore.On("RegisterMemCube", mock.Anything, "/path/to/cube", "cube123", "test-user").Return(nil)

	reqBody := MemCubeRegister{
		MemCubeNameOrPath: "/path/to/cube",
		MemCubeID:         stringPtr("cube123"),
	}

	w := performRequest(server.router, "POST", "/mem_cubes", reqBody)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SimpleResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "MemCube registered successfully", response.Message)

	mockCore.AssertExpectations(t)
}

func TestUnregisterMemCube(t *testing.T) {
	server, mockCore, _ := setupTestServer()

	// Setup mock expectations
	mockCore.On("UnregisterMemCube", mock.Anything, "cube123", "test-user").Return(nil)

	w := performRequest(server.router, "DELETE", "/mem_cubes/cube123", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SimpleResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "MemCube unregistered successfully", response.Message)

	mockCore.AssertExpectations(t)
}

func TestGetMetrics(t *testing.T) {
	server, _, _ := setupTestServer()

	w := performRequest(server.router, "GET", "/metrics", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var response MetricsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.Timestamp)
	assert.NotNil(t, response.Metrics)
}

func TestOpenAPISpec(t *testing.T) {
	server, _, _ := setupTestServer()

	w := performRequest(server.router, "GET", "/openapi.json", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var spec map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &spec)
	assert.NoError(t, err)
	assert.Equal(t, "3.1.0", spec["openapi"])
	assert.NotNil(t, spec["info"])
	assert.NotNil(t, spec["paths"])
	assert.NotNil(t, spec["components"])
}

// Test error cases

func TestAddMemoryBadRequest(t *testing.T) {
	server, _, _ := setupTestServer()

	// Request with no content
	reqBody := MemoryCreate{}

	w := performRequest(server.router, "POST", "/memories", reqBody)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, response.Code)
}

func TestSearchMemoriesBadRequest(t *testing.T) {
	server, _, _ := setupTestServer()

	// Request with empty query
	reqBody := SearchRequest{}

	w := performRequest(server.router, "POST", "/search", reqBody)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, response.Code)
}

// Benchmark tests

func BenchmarkHealthCheck(b *testing.B) {
	server, _, _ := setupTestServer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := performRequest(server.router, "GET", "/health", nil)
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

func BenchmarkAddMemory(b *testing.B) {
	server, mockCore, _ := setupTestServer()
	mockCore.On("Add", mock.Anything, mock.AnythingOfType("*types.AddMemoryRequest")).Return(nil)

	reqBody := MemoryCreate{
		MemoryContent: stringPtr("Benchmark test memory"),
		MemCubeID:     stringPtr("benchmark-cube"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := performRequest(server.router, "POST", "/memories", reqBody)
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// Helper functions

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}