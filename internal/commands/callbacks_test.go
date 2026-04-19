package commands

import (
	"database/sql"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/user/telegram-bot/internal/db"
	"github.com/user/telegram-bot/internal/todoist"
)

// Tests that a valid session owner can successfully confirm and create a task from a draft
func TestCallbackHandler_HandleCallback_ParsesSessionIDCorrectly(t *testing.T) {
	mockDB := new(MockDBManager)
	mockTodoist := new(MockTodoistClient)

	sessionID := 123
	chatID := int64(789)
	userID := int64(456)

	mockDB.On("IsSessionOwner", mock.Anything, sessionID, userID).Return(true, nil)
	mockDB.On("GetDraftTask", mock.Anything, sessionID).Return(db.DraftTask{
		SessionID:   sessionID,
		Title:       sql.NullString{String: "Test Task", Valid: true},
		Description: sql.NullString{String: "Test Description", Valid: true},
		DueISO:      sql.NullString{String: "2026-04-01", Valid: true},
		Priority:    sql.NullInt32{Int32: 3, Valid: true},
		UpdatedAt:   time.Now(),
	}, nil)
	mockDB.On("GetTodoistProjectID", mock.Anything, chatID).Return("project123", nil)
	mockTodoist.On("CreateTask", mock.Anything, mock.Anything).Return(&todoist.TaskResponse{
		ID:      "todoist123",
		Content: "Test Task",
		URL:     "https://todoist.com/showTask?id=todoist123",
	}, nil)
	mockDB.On("SaveCreatedTask", mock.Anything, sessionID, "todoist123", mock.Anything).Return(nil)
	mockDB.On("CloseSession", mock.Anything, chatID).Return(nil)

	handler := NewCallbackHandler(mockTodoist, mockDB)

	callback := &tgbotapi.CallbackQuery{
		ID:   "test_callback_id",
		From: &tgbotapi.User{ID: userID},
		Message: &tgbotapi.Message{
			Chat:      &tgbotapi.Chat{ID: chatID},
			MessageID: 101,
		},
		Data: "confirm_task:123",
	}

	response := handler.HandleCallback(callback)

	assert.NotNil(t, response)
	assert.True(t, response.IsOwner)
	assert.NotNil(t, response.CallbackConfig)

	mockDB.AssertExpectations(t)
	mockTodoist.AssertExpectations(t)
}

// Tests that a user who is not the session owner cannot manage or cancel the discussion
func TestCallbackHandler_HandleCallback_NonOwner(t *testing.T) {
	mockDB := new(MockDBManager)
	mockTodoist := new(MockTodoistClient)

	sessionID := 123
	userID := int64(456)

	mockDB.On("IsSessionOwner", mock.Anything, sessionID, userID).Return(false, nil)

	handler := NewCallbackHandler(mockTodoist, mockDB)

	callback := &tgbotapi.CallbackQuery{
		ID:   "test_callback_id",
		From: &tgbotapi.User{ID: userID},
		Message: &tgbotapi.Message{
			Chat:      &tgbotapi.Chat{ID: 789},
			MessageID: 101,
		},
		Data: "cancel_task:123",
	}

	response := handler.HandleCallback(callback)

	assert.NotNil(t, response)
	assert.False(t, response.IsOwner)
	assert.NotNil(t, response.CallbackConfig)
	assert.Contains(t, response.CallbackConfig.Text, "Только автор обсуждения может отменить задачу")

	mockDB.AssertExpectations(t)
}

func TestCallbackHandler_HandleCallback_CancelKeepsSessionOpen(t *testing.T) {
	mockDB := new(MockDBManager)
	mockTodoist := new(MockTodoistClient)

	sessionID := 123
	chatID := int64(789)
	userID := int64(456)

	mockDB.On("IsSessionOwner", mock.Anything, sessionID, userID).Return(true, nil)
	mockDB.On("DeleteDraftTask", mock.Anything, sessionID).Return(nil)

	handler := NewCallbackHandler(mockTodoist, mockDB)

	callback := &tgbotapi.CallbackQuery{
		ID:   "test_callback_id",
		From: &tgbotapi.User{ID: userID},
		Message: &tgbotapi.Message{
			Chat:      &tgbotapi.Chat{ID: chatID},
			MessageID: 101,
		},
		Data: "cancel_task:123",
	}

	response := handler.HandleCallback(callback)

	assert.NotNil(t, response)
	assert.True(t, response.IsOwner)
	assert.NotNil(t, response.CallbackConfig)
	assert.NotNil(t, response.ResponseMessage)
	assert.Contains(t, response.ResponseMessage.Text, "Обсуждение продолжается")
	mockDB.AssertNotCalled(t, "CloseSession", mock.Anything, chatID)
	mockDB.AssertExpectations(t)
}

// Tests that malformed callback data without proper separator is handled gracefully
func TestCallbackHandler_HandleCallback_InvalidCallbackData(t *testing.T) {
	mockDB := new(MockDBManager)
	mockTodoist := new(MockTodoistClient)

	handler := NewCallbackHandler(mockTodoist, mockDB)

	callback := &tgbotapi.CallbackQuery{
		ID:   "test_callback_id",
		From: &tgbotapi.User{ID: 456},
		Data: "invalid_format",
	}

	response := handler.HandleCallback(callback)

	assert.NotNil(t, response)
	assert.False(t, response.IsOwner)
	assert.NotNil(t, response.CallbackConfig)
	assert.Contains(t, response.CallbackConfig.Text, "Invalid callback data")

	mockDB.AssertNotCalled(t, "IsSessionOwner", mock.Anything, mock.Anything, mock.Anything)
	mockDB.AssertExpectations(t)
}

// Tests that unknown callback action types are rejected with an appropriate error message
func TestCallbackHandler_HandleCallback_UnknownCallbackType(t *testing.T) {
	mockDB := new(MockDBManager)
	mockTodoist := new(MockTodoistClient)

	handler := NewCallbackHandler(mockTodoist, mockDB)

	callback := &tgbotapi.CallbackQuery{
		ID:   "test_callback_id",
		From: &tgbotapi.User{ID: 456},
		Data: "unknown_action:123",
	}

	response := handler.HandleCallback(callback)

	assert.NotNil(t, response)
	assert.False(t, response.IsOwner)
	assert.NotNil(t, response.CallbackConfig)
	assert.Contains(t, response.CallbackConfig.Text, "Unknown callback type")

	mockDB.AssertExpectations(t)
}

// Tests that non-numeric session IDs in callback data are handled without errors
func TestCallbackHandler_HandleCallback_InvalidSessionID(t *testing.T) {
	mockDB := new(MockDBManager)
	mockTodoist := new(MockTodoistClient)

	handler := NewCallbackHandler(mockTodoist, mockDB)

	callback := &tgbotapi.CallbackQuery{
		ID:   "test_callback_id",
		From: &tgbotapi.User{ID: 456},
		Message: &tgbotapi.Message{
			Chat:      &tgbotapi.Chat{ID: 789},
			MessageID: 101,
		},
		Data: "confirm_task:abc",
	}

	response := handler.HandleCallback(callback)

	assert.NotNil(t, response)
	assert.False(t, response.IsOwner)

	mockDB.AssertExpectations(t)
}
