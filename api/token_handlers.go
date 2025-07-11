package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/memtensor/memgos/pkg/users"
)

// APITokenListResponse represents the response for listing API tokens
type APITokenListResponse struct {
	Tokens []APITokenInfo `json:"tokens"`
	Total  int            `json:"total"`
}

// APITokenInfo represents API token information (without the actual token)
type APITokenInfo struct {
	TokenID     string     `json:"token_id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Prefix      string     `json:"prefix"`
	Scopes      []string   `json:"scopes"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	LastUsedIP  string     `json:"last_used_ip,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	IsActive    bool       `json:"is_active"`
}

// CreateAPITokenRequest represents the request to create an API token
type CreateAPITokenRequest struct {
	Name        string     `json:"name" binding:"required"`
	Description string     `json:"description,omitempty"`
	Scopes      []string   `json:"scopes" binding:"required"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// UpdateAPITokenRequest represents the request to update an API token
type UpdateAPITokenRequest struct {
	Name        string     `json:"name,omitempty"`
	Description *string    `json:"description,omitempty"`
	Scopes      []string   `json:"scopes,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	IsActive    *bool      `json:"is_active,omitempty"`
}

// createAPIToken creates a new API token for the authenticated user
// @Summary Create API Token
// @Description Create a new API token for programmatic access
// @Tags tokens
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param token body CreateAPITokenRequest true "Token creation parameters"
// @Success 201 {object} users.APITokenResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tokens [post]
func (s *Server) createAPIToken(c *gin.Context) {
	// Get authenticated user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	var req CreateAPITokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	// Convert to user manager params
	params := users.CreateAPITokenParams{
		Name:        req.Name,
		Description: req.Description,
		Scopes:      req.Scopes,
		ExpiresAt:   req.ExpiresAt,
	}

	// Create token using user manager
	tokenResponseInterface, err := s.authManager.CreateAPIToken(userID.(string), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to create API token",
			Details: err.Error(),
		})
		return
	}

	// Type assert to get the actual response
	tokenResponse, ok := tokenResponseInterface.(*users.APITokenResponse)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Internal server error: invalid response format",
		})
		return
	}

	c.JSON(http.StatusCreated, tokenResponse)
}

// getAPITokens lists all API tokens for the authenticated user
// @Summary List API Tokens
// @Description Get all API tokens for the authenticated user
// @Tags tokens
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} APITokenListResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tokens [get]
func (s *Server) getAPITokens(c *gin.Context) {
	// Get authenticated user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	// Get tokens from user manager
	tokensInterface, err := s.authManager.GetAPITokens(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get API tokens",
			Details: err.Error(),
		})
		return
	}

	// Type assert to get the actual tokens
	tokens, ok := tokensInterface.([]users.APIToken)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Internal server error: invalid token format",
		})
		return
	}

	// Convert to response format
	tokenInfos := make([]APITokenInfo, len(tokens))
	for i, token := range tokens {
		tokenInfos[i] = APITokenInfo{
			TokenID:     token.TokenID,
			Name:        token.Name,
			Description: token.Description,
			Prefix:      token.Prefix,
			Scopes:      token.Scopes,
			ExpiresAt:   token.ExpiresAt,
			LastUsedAt:  token.LastUsedAt,
			LastUsedIP:  token.LastUsedIP,
			CreatedAt:   token.CreatedAt,
			IsActive:    token.IsActive,
		}
	}

	response := APITokenListResponse{
		Tokens: tokenInfos,
		Total:  len(tokenInfos),
	}

	c.JSON(http.StatusOK, response)
}

// getAPIToken retrieves a specific API token by ID
// @Summary Get API Token
// @Description Get details of a specific API token
// @Tags tokens
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Token ID"
// @Success 200 {object} APITokenInfo
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tokens/{id} [get]
func (s *Server) getAPIToken(c *gin.Context) {
	// Get authenticated user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	tokenID := c.Param("id")
	if tokenID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Token ID is required",
		})
		return
	}

	// Get token from user manager
	tokenInterface, err := s.authManager.GetAPIToken(userID.(string), tokenID)
	if err != nil {
		if err.Error() == "API token not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Code:    http.StatusNotFound,
				Message: "API token not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get API token",
			Details: err.Error(),
		})
		return
	}

	// Type assert to get the actual token
	token, ok := tokenInterface.(*users.APIToken)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Internal server error: invalid token format",
		})
		return
	}

	// Convert to response format
	tokenInfo := APITokenInfo{
		TokenID:     token.TokenID,
		Name:        token.Name,
		Description: token.Description,
		Prefix:      token.Prefix,
		Scopes:      token.Scopes,
		ExpiresAt:   token.ExpiresAt,
		LastUsedAt:  token.LastUsedAt,
		LastUsedIP:  token.LastUsedIP,
		CreatedAt:   token.CreatedAt,
		IsActive:    token.IsActive,
	}

	c.JSON(http.StatusOK, tokenInfo)
}

// updateAPIToken updates an API token's metadata
// @Summary Update API Token
// @Description Update API token metadata, scopes, or expiration
// @Tags tokens
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Token ID"
// @Param token body UpdateAPITokenRequest true "Token update parameters"
// @Success 200 {object} APITokenInfo
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tokens/{id} [put]
func (s *Server) updateAPIToken(c *gin.Context) {
	// Get authenticated user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	tokenID := c.Param("id")
	if tokenID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Token ID is required",
		})
		return
	}

	var req UpdateAPITokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	// Convert to user manager params
	params := users.UpdateAPITokenParams{
		Name:        req.Name,
		Description: req.Description,
		Scopes:      req.Scopes,
		ExpiresAt:   req.ExpiresAt,
		IsActive:    req.IsActive,
	}

	// Update token using user manager
	tokenInterface, err := s.authManager.UpdateAPIToken(userID.(string), tokenID, params)
	if err != nil {
		if err.Error() == "API token not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Code:    http.StatusNotFound,
				Message: "API token not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to update API token",
			Details: err.Error(),
		})
		return
	}

	// Type assert to get the actual token
	token, ok := tokenInterface.(*users.APIToken)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Internal server error: invalid token format",
		})
		return
	}

	// Convert to response format
	tokenInfo := APITokenInfo{
		TokenID:     token.TokenID,
		Name:        token.Name,
		Description: token.Description,
		Prefix:      token.Prefix,
		Scopes:      token.Scopes,
		ExpiresAt:   token.ExpiresAt,
		LastUsedAt:  token.LastUsedAt,
		LastUsedIP:  token.LastUsedIP,
		CreatedAt:   token.CreatedAt,
		IsActive:    token.IsActive,
	}

	c.JSON(http.StatusOK, tokenInfo)
}

// revokeAPIToken revokes (deletes) an API token
// @Summary Revoke API Token
// @Description Revoke an API token, making it unusable
// @Tags tokens
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Token ID"
// @Success 204 "Token revoked successfully"
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tokens/{id} [delete]
func (s *Server) revokeAPIToken(c *gin.Context) {
	// Get authenticated user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	tokenID := c.Param("id")
	if tokenID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Token ID is required",
		})
		return
	}

	// Revoke token using user manager
	err := s.authManager.RevokeAPIToken(userID.(string), tokenID)
	if err != nil {
		if err.Error() == "API token not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Code:    http.StatusNotFound,
				Message: "API token not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to revoke API token",
			Details: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}