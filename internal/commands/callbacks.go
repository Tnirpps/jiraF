package commands

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/user/telegram-bot/internal/todoist"
)

// Callback data constants for task actions
const (
	// CallbackConfirm is used for confirming and creating a task
	CallbackConfirm = "confirm_task"
	// CallbackEdit is used for editing draft task before creation
	CallbackEdit = "edit_task"
	// CallbackCancel is used for canceling task creation
	CallbackCancel = "cancel_task"
	// CallbackFinishDiscussion is used for confirming discussion finish without task creation
	CallbackFinishDiscussion = "finish_discussion"
	// CallbackKeepDiscussion is used for declining discussion finish and continuing the session
	CallbackKeepDiscussion = "keep_discussion"
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
	dbManager     DBManager
	todoistClient todoist.Client
}

// NewCallbackHandler creates a new callback handler
func NewCallbackHandler(todoistClient todoist.Client, dbManager DBManager) *CallbackHandler {
	return &CallbackHandler{
		dbManager:     dbManager,
		todoistClient: todoistClient,
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
	case CallbackFinishDiscussion:
		return h.handleFinishDiscussionCallback(callback, sessionIDStr)
	case CallbackKeepDiscussion:
		return h.handleKeepDiscussionCallback(callback, sessionIDStr)
	default:
		callbackCfg := tgbotapi.NewCallback(callback.ID, "Unknown callback type")
		return &CallbackResponse{
			CallbackConfig: &callbackCfg,
			IsOwner:        false,
		}
	}
}

func (h *CallbackHandler) parseSessionID(sessionIDStr string) (int, error) {
	sessionID, err := strconv.Atoi(sessionIDStr)
	if err != nil {
		return 0, fmt.Errorf("invalid session ID: %v", err)
	}

	return sessionID, nil
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
		callbackCfg := tgbotapi.NewCallback(callback.ID, "Только автор обсуждения может создать задачу")
		return &CallbackResponse{
			CallbackConfig: &callbackCfg,
			IsOwner:        false,
		}
	}

	sessionID, err := h.parseSessionID(sessionIDStr)
	if err != nil {
		log.Print(fmt.Errorf("invalid session ID: %v", err))
		return nil
	}

	ctx := context.Background()
	task, err := h.dbManager.GetDraftTask(ctx, sessionID)
	if err != nil {
		log.Printf("Error getting draft task: %v", err)
		callbackCfg := tgbotapi.NewCallback(callback.ID, "Error: Failed to get draft task")
		return &CallbackResponse{
			CallbackConfig: &callbackCfg,
			IsOwner:        true,
		}
	}

	projectID, err := h.dbManager.GetTodoistProjectID(ctx, callback.Message.Chat.ID)
	if err != nil {
		log.Printf("Error getting Todoist project ID: %v", err)
		callbackCfg := tgbotapi.NewCallback(callback.ID, "Error: Failed to get Todoist project ID")
		return &CallbackResponse{
			CallbackConfig: &callbackCfg,
			IsOwner:        true,
		}
	}

	todoistRequest := &todoist.TaskRequest{
		Content:     task.Title.String,
		Description: task.Description.String,
		ProjectID:   projectID,
		Priority:    int(task.Priority.Int32),
		DueDate:     task.DueISO.String,
	}

	resp, err := h.todoistClient.CreateTask(ctx, todoistRequest)
	if err != nil {
		log.Printf("Error creating task: %v", err)
		callbackCfg := tgbotapi.NewCallback(callback.ID, "Error: Failed to create task")
		return &CallbackResponse{
			CallbackConfig: &callbackCfg,
			IsOwner:        true,
		}
	}

	err = h.dbManager.SaveCreatedTask(ctx, sessionID, resp.ID, resp.URL)
	if err != nil {
		log.Printf("Error saving created task: %v", err)
	}

	err = h.dbManager.CloseSession(ctx, callback.Message.Chat.ID)
	if err != nil {
		log.Printf("Error closing session: %v", err)
	}

	// ✅ Формируем правильную ссылку на задачу Todoist
	taskURL := fmt.Sprintf("https://app.todoist.com/app/task/%s", resp.ID)

	callbackCfg := tgbotapi.NewCallback(callback.ID, "✅ Отлично! Создаю задачу.")
	messageText := fmt.Sprintf("✅ **Задача создана**: [%s](%s)", task.Title.String, taskURL)
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, messageText)
	msg.ParseMode = "Markdown"

	return &CallbackResponse{
		CallbackConfig:  &callbackCfg,
		IsOwner:         true,
		ResponseMessage: &msg,
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
		callbackCfg := tgbotapi.NewCallback(callback.ID, "Только автор обсуждения может редактировать задачу")
		return &CallbackResponse{
			CallbackConfig: &callbackCfg,
			IsOwner:        false,
		}
	}

	log.Printf("Handling edit task request for session %s", sessionIDStr)

	// Send a message asking for edit instructions
	chatID := callback.Message.Chat.ID

	// Create the message asking for edit instructions
	messageText := `
✏️ Отредактировать задачу
Пожалуйста, ответьте на это сообщение, указав ваши инструкции по редактированию в произвольном формате.
Примеры:
• "Измени заголовок на: Исправление ошибки входа в систему"
• "Установи высокий приоритет"
• "Измени срок выполнения на пятницу"
• "Добавить метку: frontend"
`
	msg := tgbotapi.NewMessage(chatID, messageText)
	msg.ParseMode = "Markdown"

	// Create acknowledgment for the callback
	callbackCfg := tgbotapi.NewCallback(callback.ID, "✏️ Пожалуйста, ответьте на это сообщение с инструкциями по редактированию")

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
		callbackCfg := tgbotapi.NewCallback(callback.ID, "Только автор обсуждения может отменить задачу")
		return &CallbackResponse{
			CallbackConfig: &callbackCfg,
			IsOwner:        false,
		}
	}

	ctx := context.Background()

	sessionID, err := h.parseSessionID(sessionIDStr)
	if err != nil {
		log.Printf("Error parsing session ID on cancel: %v", err)
		callbackCfg := tgbotapi.NewCallback(callback.ID, "Error: Invalid session ID")
		return &CallbackResponse{
			CallbackConfig: &callbackCfg,
			IsOwner:        true,
		}
	}

	err = h.dbManager.DeleteDraftTask(ctx, sessionID)
	if err != nil {
		log.Printf("Error deleting draft task on cancel: %v", err)
	}

	log.Printf("Canceling task from session %s", sessionIDStr)

	callbackCfg := tgbotapi.NewCallback(callback.ID, "❌ Создание задачи отменено")
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "❌ Создание задачи отменено. Обсуждение продолжается.")
	return &CallbackResponse{
		CallbackConfig:  &callbackCfg,
		IsOwner:         true,
		ResponseMessage: &msg,
	}
}

func (h *CallbackHandler) handleFinishDiscussionCallback(callback *tgbotapi.CallbackQuery, sessionIDStr string) *CallbackResponse {
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
		callbackCfg := tgbotapi.NewCallback(callback.ID, "Только автор обсуждения может завершить его")
		return &CallbackResponse{
			CallbackConfig: &callbackCfg,
			IsOwner:        false,
		}
	}

	ctx := context.Background()
	if err := h.dbManager.CloseSession(ctx, callback.Message.Chat.ID); err != nil {
		log.Printf("Error closing session: %v", err)
		callbackCfg := tgbotapi.NewCallback(callback.ID, "Не удалось завершить обсуждение")
		return &CallbackResponse{
			CallbackConfig: &callbackCfg,
			IsOwner:        true,
		}
	}

	callbackCfg := tgbotapi.NewCallback(callback.ID, "🛑 Обсуждение завершено")
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "🛑 Обсуждение завершено без создания задачи.")

	return &CallbackResponse{
		CallbackConfig:  &callbackCfg,
		IsOwner:         true,
		ResponseMessage: &msg,
	}
}

func (h *CallbackHandler) handleKeepDiscussionCallback(callback *tgbotapi.CallbackQuery, sessionIDStr string) *CallbackResponse {
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
		callbackCfg := tgbotapi.NewCallback(callback.ID, "Только автор обсуждения может продолжить обсуждение")
		return &CallbackResponse{
			CallbackConfig: &callbackCfg,
			IsOwner:        false,
		}
	}

	callbackCfg := tgbotapi.NewCallback(callback.ID, "Обсуждение продолжается")
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "↩️ Обсуждение продолжается.")

	return &CallbackResponse{
		CallbackConfig:  &callbackCfg,
		IsOwner:         true,
		ResponseMessage: &msg,
	}
}
