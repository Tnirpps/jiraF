package commands

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Callback data constants for task actions
const (
	// CallbackConfirm is used for confirming and creating a task
	CallbackConfirm = "confirm_task"

	// CallbackEdit is used for editing draft task before creation
	CallbackEdit = "edit_task"

	// CallbackCancel is used for canceling task creation
	CallbackCancel = "cancel_task"
)

// Separator used in callback data
const CallbackDataSeparator = ":"

// CallbackResponse contains the response data for a callback query
type CallbackResponse struct {
	CallbackConfig  *tgbotapi.CallbackConfig
	IsOwner         bool
	ResponseMessage *tgbotapi.MessageConfig // Message to send to the user
	SessionID       string                  // Session ID for context
	WaitingForReply bool                    // Indicates if we're waiting for a reply
}

// CallbackHandler processes callback queries from buttons
type CallbackHandler struct {
	dbManager DBManager
}

// NewCallbackHandler creates a new callback handler
func NewCallbackHandler(dbManager DBManager) *CallbackHandler {
	return &CallbackHandler{
		dbManager: dbManager,
	}
}

// HandleCallback processes callback queries
func (h *CallbackHandler) HandleCallback(callback *tgbotapi.CallbackQuery) *CallbackResponse {
	// Extract callback type and session ID from format "{action}:{session_id}"
	parts := strings.Split(callback.Data, CallbackDataSeparator)
	if len(parts) != 2 {
		log.Printf("Invalid callback data format: %s", callback.Data)
		callbackCfg := tgbotapi.NewCallback(callback.ID, "Invalid callback data")
		return &CallbackResponse{
			CallbackConfig: &callbackCfg,
			IsOwner:        false,
		}
	}

	callbackType := parts[0]
	log.Printf("Callback type: %s", callbackType)

	// The session ID is the second part
	sessionIDStr := parts[1]
	log.Printf("Session ID: %s", sessionIDStr)

	// Process different callback types
	switch callbackType {
	case CallbackConfirm:
		return h.handleConfirmCallback(callback, sessionIDStr)
	case CallbackEdit:
		return h.handleEditCallback(callback, sessionIDStr)
	case CallbackCancel:
		return h.handleCancelCallback(callback, sessionIDStr)
	default:
		callbackCfg := tgbotapi.NewCallback(callback.ID, "Unknown callback type")
		return &CallbackResponse{
			CallbackConfig: &callbackCfg,
			IsOwner:        false,
		}
	}
}

// verifySessionOwner checks if the user is the owner of the session
func (h *CallbackHandler) verifySessionOwner(sessionIDStr string, userID int64) (bool, error) {
	ctx := context.Background()

	// Parse session ID
	sessionID, err := strconv.Atoi(sessionIDStr)
	if err != nil {
		return false, fmt.Errorf("invalid session ID: %v", err)
	}

	// Check if the user is the owner using the DB method
	isOwner, err := h.dbManager.IsSessionOwner(ctx, sessionID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to verify session ownership: %v", err)
	}

	return isOwner, nil
}

// handleConfirmCallback handles confirming a task
func (h *CallbackHandler) handleConfirmCallback(callback *tgbotapi.CallbackQuery, sessionIDStr string) *CallbackResponse {
	// Check if the user is the owner of the session
	isOwner, err := h.verifySessionOwner(sessionIDStr, int64(callback.From.ID))
	if err != nil {
		log.Printf("Error verifying session owner: %v", err)
		callbackCfg := tgbotapi.NewCallback(callback.ID, "Error: Failed to verify session ownership")
		return &CallbackResponse{
			CallbackConfig: &callbackCfg,
			IsOwner:        false,
		}
	}

	if !isOwner {
		callbackCfg := tgbotapi.NewCallback(callback.ID, "Only the user who started this discussion can confirm the task")
		return &CallbackResponse{
			CallbackConfig: &callbackCfg,
			IsOwner:        false,
		}
	}

	// PLACEHOLDER: This will be implemented in the next phase
	// Will connect to Todoist API and create actual task
	log.Printf("PLACEHOLDER: Confirming task from session %s", sessionIDStr)

	// In the next phase, this will:
	// 1. Fetch the draft task from the database
	// 2. Create a real task in Todoist
	// 3. Save the created task ID and URL
	// 4. Close the session

	callbackCfg := tgbotapi.NewCallback(callback.ID, "✅ Got it! This will create a task in the next phase.")
	return &CallbackResponse{
		CallbackConfig: &callbackCfg,
		IsOwner:        true,
	}
}

// handleEditCallback handles editing a task
func (h *CallbackHandler) handleEditCallback(callback *tgbotapi.CallbackQuery, sessionIDStr string) *CallbackResponse {
	// Check if the user is the owner of the session
	isOwner, err := h.verifySessionOwner(sessionIDStr, int64(callback.From.ID))
	if err != nil {
		log.Printf("Error verifying session owner: %v", err)
		callbackCfg := tgbotapi.NewCallback(callback.ID, "Error: Failed to verify session ownership")
		return &CallbackResponse{
			CallbackConfig: &callbackCfg,
			IsOwner:        false,
		}
	}

	if !isOwner {
		callbackCfg := tgbotapi.NewCallback(callback.ID, "Only the user who started this discussion can edit the task")
		return &CallbackResponse{
			CallbackConfig: &callbackCfg,
			IsOwner:        false,
		}
	}

	log.Printf("Handling edit task request for session %s", sessionIDStr)

	// Send a message asking for edit instructions
	chatID := callback.Message.Chat.ID

	// Create the message asking for edit instructions
	messageText := "✏️ *Editing task*\n\nPlease reply to this message with your edit instructions.\n\n" +
		"Examples:\n" +
		"• \"Change title to: Fix login bug\"\n" +
		"• \"Set priority to high\"\n" +
		"• \"Change due date to Friday\"\n" +
		"• \"Add label: frontend\""

	msg := tgbotapi.NewMessage(chatID, messageText)
	msg.ParseMode = "Markdown"

	// Create acknowledgment for the callback
	callbackCfg := tgbotapi.NewCallback(callback.ID, "✏️ Please reply to my next message with your edit instructions")

	// In a real implementation, we would mark in the database that we're waiting for a reply for this session
	// Something like: h.dbManager.SetEditMode(ctx, sessionID, true)

	// Return both the callback acknowledgment and the message to send
	return &CallbackResponse{
		CallbackConfig:  &callbackCfg,
		IsOwner:         true,
		ResponseMessage: &msg,
		SessionID:       sessionIDStr,
		WaitingForReply: true,
	}
}

// handleCancelCallback handles canceling a task
func (h *CallbackHandler) handleCancelCallback(callback *tgbotapi.CallbackQuery, sessionIDStr string) *CallbackResponse {
	// Check if the user is the owner of the session
	isOwner, err := h.verifySessionOwner(sessionIDStr, int64(callback.From.ID))
	if err != nil {
		log.Printf("Error verifying session owner: %v", err)
		callbackCfg := tgbotapi.NewCallback(callback.ID, "Error: Failed to verify session ownership")
		return &CallbackResponse{
			CallbackConfig: &callbackCfg,
			IsOwner:        false,
		}
	}

	if !isOwner {
		callbackCfg := tgbotapi.NewCallback(callback.ID, "Only the user who started this discussion can cancel the task")
		return &CallbackResponse{
			CallbackConfig: &callbackCfg,
			IsOwner:        false,
		}
	}

	// PLACEHOLDER: This will be implemented in the next phase
	// Will cancel task creation and possibly close the session
	log.Printf("PLACEHOLDER: Canceling task from session %s", sessionIDStr)

	// In the next phase, this will:
	// 1. Delete the draft task if needed
	// 2. Possibly close the session

	callbackCfg := tgbotapi.NewCallback(callback.ID, "❌ Got it! Task creation canceled.")
	return &CallbackResponse{
		CallbackConfig: &callbackCfg,
		IsOwner:        true,
	}
}
