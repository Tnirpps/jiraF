package commands

import (
	"log"
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
func (h *CallbackHandler) HandleCallback(callback *tgbotapi.CallbackQuery) *tgbotapi.CallbackConfig {
	// Extract callback type and session ID
	parts := strings.Split(callback.Data, "_")
	if len(parts) < 2 {
		callbackCfg := tgbotapi.NewCallback(callback.ID, "Invalid callback data")
		return &callbackCfg
	}

	callbackType := parts[0]
	// The rest is the session ID
	sessionIDStr := strings.Join(parts[1:], "_")

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
		return &callbackCfg
	}
}

// handleConfirmCallback handles confirming a task
func (h *CallbackHandler) handleConfirmCallback(callback *tgbotapi.CallbackQuery, sessionIDStr string) *tgbotapi.CallbackConfig {
	// PLACEHOLDER: This will be implemented in the next phase
	// Will connect to Todoist API and create actual task
	log.Printf("PLACEHOLDER: Confirming task from session %s", sessionIDStr)

	// In the next phase, this will:
	// 1. Fetch the draft task from the database
	// 2. Create a real task in Todoist
	// 3. Save the created task ID and URL
	// 4. Close the session

	callbackCfg := tgbotapi.NewCallback(callback.ID, "✅ Got it! This will create a task in the next phase.")
	return &callbackCfg
}

// handleEditCallback handles editing a task
func (h *CallbackHandler) handleEditCallback(callback *tgbotapi.CallbackQuery, sessionIDStr string) *tgbotapi.CallbackConfig {
	// PLACEHOLDER: This will be implemented in the next phase
	// Will prompt user for edit instructions and apply them to the draft
	log.Printf("PLACEHOLDER: Editing task from session %s", sessionIDStr)

	// In the next phase, this will:
	// 1. Ask the user for edit instructions
	// 2. Send them to the ML service
	// 3. Update the draft with the changes
	// 4. Show a new preview

	callbackCfg := tgbotapi.NewCallback(callback.ID, "✏️ Got it! This will allow editing in the next phase.")
	return &callbackCfg
}

// handleCancelCallback handles canceling a task
func (h *CallbackHandler) handleCancelCallback(callback *tgbotapi.CallbackQuery, sessionIDStr string) *tgbotapi.CallbackConfig {
	// PLACEHOLDER: This will be implemented in the next phase
	// Will cancel task creation and possibly close the session
	log.Printf("PLACEHOLDER: Canceling task from session %s", sessionIDStr)

	// In the next phase, this will:
	// 1. Delete the draft task if needed
	// 2. Possibly close the session

	callbackCfg := tgbotapi.NewCallback(callback.ID, "❌ Got it! Task creation canceled.")
	return &callbackCfg
}
