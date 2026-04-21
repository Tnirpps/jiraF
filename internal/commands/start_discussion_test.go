package commands

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/user/telegram-bot/internal/db"
	"github.com/user/telegram-bot/internal/todoist"
)

// Tests StartDiscussionCommand behavior when no project ID is configured for the chat
// Verifies that the command returns an error message asking the user to set a project first
func TestStartDiscussion_NoProjectID(t *testing.T) {
	// Test data
	chatID := int64(123456789)

	// Create and configure mock with fluent API
	mockDBManager := new(MockDBManager)
	ConfigureMockDB(mockDBManager).
		WithProjectID(chatID, "", db.ErrProjectIDNotSet)
	mockTodoistClient := new(MockTodoistClient)
	mockTodoistClient.On("GetProjects", mock.Anything).Return([]todoist.Project{
		{ID: "12345", Name: "Backend"},
	}, nil)

	// Create command
	cmd := NewStartDiscussionCommand(mockDBManager, mockTodoistClient)

	// Create test message using helper function
	message := CreateCommandMessage(chatID, "/start_discussion")

	// Execute command
	response := cmd.Execute(message)

	assert.Contains(t, response.Text, "Сначала выберите проект Todoist")
	_, ok := response.ReplyMarkup.(tgbotapi.InlineKeyboardMarkup)
	assert.True(t, ok)

	// Verify mock
	mockDBManager.AssertExpectations(t)
	mockTodoistClient.AssertExpectations(t)
}

// Tests StartDiscussionCommand successful execution when project ID is configured
// Verifies that a new discussion session is created and confirmation message is returned
func TestStartDiscussion_Success(t *testing.T) {
	// Test data
	chatID := int64(123456789)
	projectID := "12345"
	sessionID := 1

	// Create and configure mock with fluent API
	mockDBManager := new(MockDBManager)
	ConfigureMockDB(mockDBManager).
		WithProjectID(chatID, projectID, nil).
		WithStartSession(chatID, chatID, sessionID, nil)

	// Create command
	mockTodoistClient := new(MockTodoistClient)
	cmd := NewStartDiscussionCommand(mockDBManager, mockTodoistClient)

	// Create test message using helper function
	message := CreateCommandMessage(chatID, "/start_discussion")

	// Execute command
	response := cmd.Execute(message)

	assert.Contains(t, response.Text, "Обсуждение началось")

	// Verify mock
	mockDBManager.AssertExpectations(t)
}

// Tests StartDiscussionCommand behavior when a discussion session already exists for the chat
// Verifies that the command returns an error message indicating an active discussion is in progress
func TestStartDiscussion_AlreadyActive(t *testing.T) {
	// Test data
	chatID := int64(123456789)
	projectID := "12345"

	// Create and configure mock with fluent API
	mockDBManager := new(MockDBManager)
	ConfigureMockDB(mockDBManager).
		WithProjectID(chatID, projectID, nil).
		WithStartSession(chatID, chatID, 0, db.ErrSessionAlreadyExists)

	// Create command
	mockTodoistClient := new(MockTodoistClient)
	cmd := NewStartDiscussionCommand(mockDBManager, mockTodoistClient)

	// Create test message using helper function
	message := CreateCommandMessage(chatID, "/start_discussion")

	// Execute command
	response := cmd.Execute(message)

	assert.Contains(t, response.Text, "Обсуждение уже идёт")

	// Verify mock
	mockDBManager.AssertExpectations(t)
}
