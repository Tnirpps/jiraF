package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/user/telegram-bot/internal/db"
)

func TestStartDiscussion_NoProjectID(t *testing.T) {
	// Test data
	chatID := int64(123456789)

	// Create and configure mock with fluent API
	mockDBManager := new(MockDBManager)
	ConfigureMockDB(mockDBManager).
		WithProjectID(chatID, "", db.ErrProjectIDNotSet)

	// Create command
	cmd := NewStartDiscussionCommand(mockDBManager)

	// Create test message using helper function
	message := CreateCommandMessage(chatID, "/start_discussion")

	// Execute command
	response := cmd.Execute(message)

	// Assert response
	assert.Contains(t, response.Text, "Please set a project ID first")

	// Verify mock
	mockDBManager.AssertExpectations(t)
}

func TestStartDiscussion_Success(t *testing.T) {
	// Test data
	chatID := int64(123456789)
	projectID := "12345"
	sessionID := 1

	// Create and configure mock with fluent API
	mockDBManager := new(MockDBManager)
	ConfigureMockDB(mockDBManager).
		WithProjectID(chatID, projectID, nil).
		WithStartSession(chatID, sessionID, nil)

	// Create command
	cmd := NewStartDiscussionCommand(mockDBManager)

	// Create test message using helper function
	message := CreateCommandMessage(chatID, "/start_discussion")

	// Execute command
	response := cmd.Execute(message)

	// Assert response
	assert.Contains(t, response.Text, "Discussion started")
	assert.Contains(t, response.Text, projectID)

	// Verify mock
	mockDBManager.AssertExpectations(t)
}

func TestStartDiscussion_AlreadyActive(t *testing.T) {
	// Test data
	chatID := int64(123456789)
	projectID := "12345"

	// Create and configure mock with fluent API
	mockDBManager := new(MockDBManager)
	ConfigureMockDB(mockDBManager).
		WithProjectID(chatID, projectID, nil).
		WithStartSession(chatID, 0, db.ErrSessionAlreadyExists)

	// Create command
	cmd := NewStartDiscussionCommand(mockDBManager)

	// Create test message using helper function
	message := CreateCommandMessage(chatID, "/start_discussion")

	// Execute command
	response := cmd.Execute(message)

	// Assert response
	assert.Contains(t, response.Text, "already in progress")

	// Verify mock
	mockDBManager.AssertExpectations(t)
}
