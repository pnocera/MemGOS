package core

import (
	"context"
	"fmt"

	"github.com/memtensor/memgos/pkg/chat"
	"github.com/memtensor/memgos/pkg/interfaces"
	"github.com/memtensor/memgos/pkg/types"
)

// ChatManagerAdapter adapts the chat.BaseMemChat to implement interfaces.ChatManager
type ChatManagerAdapter struct {
	baseChat chat.BaseMemChat
}

// NewChatManagerAdapter creates a new ChatManagerAdapter
func NewChatManagerAdapter(baseChat chat.BaseMemChat) interfaces.ChatManager {
	return &ChatManagerAdapter{
		baseChat: baseChat,
	}
}

// Chat processes a chat request and returns a response
func (c *ChatManagerAdapter) Chat(ctx context.Context, request *types.ChatRequest) (*types.ChatResponse, error) {
	// Use the base chat to process the query
	result, err := c.baseChat.Chat(ctx, request.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to process chat: %w", err)
	}
	
	// Convert chat.ChatResult to types.ChatResponse
	response := &types.ChatResponse{
		Response:  result.Response,
		SessionID: c.baseChat.GetSessionID(),
		UserID:    request.UserID,
		Timestamp: result.Timestamp,
	}
	
	return response, nil
}

// GetChatHistory retrieves chat history for a user
func (c *ChatManagerAdapter) GetChatHistory(ctx context.Context, userID, sessionID string) (*types.ChatHistory, error) {
	// Check if the session matches
	if c.baseChat.GetSessionID() != sessionID {
		return nil, fmt.Errorf("session mismatch")
	}
	
	// Check if the user matches
	if c.baseChat.GetUserID() != userID {
		return nil, fmt.Errorf("user mismatch")
	}
	
	// Get chat history from base chat
	history, err := c.baseChat.GetChatHistory(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat history: %w", err)
	}
	
	return history, nil
}

// ClearChatHistory clears chat history for a user
func (c *ChatManagerAdapter) ClearChatHistory(ctx context.Context, userID, sessionID string) error {
	// Check if the session matches
	if c.baseChat.GetSessionID() != sessionID {
		return fmt.Errorf("session mismatch")
	}
	
	// Check if the user matches
	if c.baseChat.GetUserID() != userID {
		return fmt.Errorf("user mismatch")
	}
	
	// Clear chat history
	err := c.baseChat.ClearChatHistory(ctx)
	if err != nil {
		return fmt.Errorf("failed to clear chat history: %w", err)
	}
	
	return nil
}

// SaveChatHistory saves chat history
func (c *ChatManagerAdapter) SaveChatHistory(ctx context.Context, history *types.ChatHistory) error {
	// Check if the session matches
	if c.baseChat.GetSessionID() != history.SessionID {
		return fmt.Errorf("session mismatch")
	}
	
	// Check if the user matches
	if c.baseChat.GetUserID() != history.UserID {
		return fmt.Errorf("user mismatch")
	}
	
	// Save chat history
	err := c.baseChat.SaveChatHistory(ctx)
	if err != nil {
		return fmt.Errorf("failed to save chat history: %w", err)
	}
	
	return nil
}