package commands

import (
	"context"
	"database/sql"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/user/telegram-bot/internal/ai"
	"github.com/user/telegram-bot/internal/db"
)

// MockAIClient is a mock implementation of the AI Client interface
type MockAIClient struct {
	mock.Mock
}

func (m *MockAIClient) AnalyzeDiscussion(ctx context.Context, messages []string) (*ai.AnalyzedTask, error) {
	args := m.Called(ctx, messages)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ai.AnalyzedTask), args.Error(1)
}

func (m *MockAIClient) EditTask(ctx context.Context, task *ai.AnalyzedTask, userFeedback string) (*ai.AnalyzedTask, error) {
	args := m.Called(ctx, task, userFeedback)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ai.AnalyzedTask), args.Error(1)
}

func TestCreateTaskCommand_Execute(t *testing.T) {
	// Create mock dependencies
	mockDB := new(MockDBManager)
	mockAI := new(MockAIClient)
	mockTodoist := new(MockTodoistClient)

	// Create command
	cmd := NewCreateTaskCommand(mockTodoist, mockDB, mockAI)

	// Test case: Active session with messages
	t.Run("Create task preview", func(t *testing.T) {
		// Set up mocks
		mockDB.On("HasActiveSession", mock.Anything, int64(123)).Return(true, nil)

		session := &db.Session{ID: 42, ChatID: 123, Status: "open", OwnerID: 456}
		mockDB.On("GetActiveSession", mock.Anything, int64(123)).Return(session, nil)

		// Mock some messages
		messages := []db.Message{
			{
				ID:        1,
				ChatID:    123,
				SessionID: sql.NullInt32{Int32: 42, Valid: true},
				MessageID: 1001,
				Text:      "Let's create a task for implementing the NLP feature",
			},
			{
				ID:        2,
				ChatID:    123,
				SessionID: sql.NullInt32{Int32: 42, Valid: true},
				MessageID: 1002,
				Text:      "It should be done by Friday",
			},
			{
				ID:        3,
				ChatID:    123,
				SessionID: sql.NullInt32{Int32: 42, Valid: true},
				MessageID: 1003,
				Text:      "This is high priority",
			},
		}
		mockDB.On("GetSessionMessages", mock.Anything, 42).Return(messages, nil)

		// Mock project ID
		mockDB.On("GetTodoistProjectID", mock.Anything, int64(123)).Return("project123", nil)

		// Mock AI analysis
		analyzedTask := &ai.AnalyzedTask{
			Title:        "Implement NLP feature",
			Description:  "Task details from discussion",
			DueDate:      "friday",
			Priority:     3,
			PriorityText: "Высокий",
		}
		mockAI.On("AnalyzeDiscussion", mock.Anything, []string{
			"Let's create a task for implementing the NLP feature",
			"It should be done by Friday",
			"This is high priority"}).Return(analyzedTask, nil)

		// Mock saving draft task
		mockDB.On("SaveDraftTask", mock.Anything, 42, "Implement NLP feature", "Task details from discussion",
			mock.Anything, 3, mock.Anything).Return(nil)

		// Create a mock message
		message := &tgbotapi.Message{
			Chat: &tgbotapi.Chat{
				ID: 123,
			},
			From: &tgbotapi.User{
				ID: 456, // Add sender ID for ownership verification
			},
		}

		// Mock ownership verification
		mockDB.On("IsSessionOwner", mock.Anything, 42, int64(456)).Return(true, nil)

		// Execute the command
		result := cmd.Execute(message)

		// Assertions
		assert.NotNil(t, result)
		assert.Contains(t, result.Text, "Draft Task Preview")
		assert.Contains(t, result.Text, "Implement NLP feature")
		assert.Contains(t, result.Text, "*Priority:* Высокий")

		// Check that the message has a reply markup with buttons
		markup, ok := result.ReplyMarkup.(tgbotapi.InlineKeyboardMarkup)
		assert.True(t, ok)
		assert.Len(t, markup.InlineKeyboard, 1)                    // One row
		assert.Len(t, markup.InlineKeyboard[0], 3)                 // Three buttons
		assert.Contains(t, markup.InlineKeyboard[0][0].Text, "✅")  // Confirm button
		assert.Contains(t, markup.InlineKeyboard[0][1].Text, "✏️") // Edit button
		assert.Contains(t, markup.InlineKeyboard[0][2].Text, "❌")  // Cancel button
	})

	// Test case: No active session
	t.Run("No active session", func(t *testing.T) {
		mockDB.On("HasActiveSession", mock.Anything, int64(456)).Return(false, nil)

		message := &tgbotapi.Message{
			Chat: &tgbotapi.Chat{
				ID: 456,
			},
			From: &tgbotapi.User{
				ID: 789, // Add sender ID
			},
		}

		result := cmd.Execute(message)

		assert.NotNil(t, result)
		assert.Contains(t, result.Text, "No active discussion")
	})
}

func TestCreateTaskCommand_ConvertToDueISO(t *testing.T) {
	// Create command with empty mocks
	mockDB := new(MockDBManager)
	mockAI := new(MockAIClient)
	mockTodoist := new(MockTodoistClient)
	cmd := NewCreateTaskCommand(mockTodoist, mockDB, mockAI)

	// Test date conversions
	today := time.Now().Format("2006-01-02")
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "today",
			input:    "today",
			expected: today,
		},
		{
			name:     "tomorrow",
			input:    "tomorrow",
			expected: tomorrow,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "already ISO",
			input:    "2025-12-31",
			expected: "2025-12-31",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := cmd.convertToDueISO(tc.input)
			if tc.name == "already ISO" {
				assert.Contains(t, result, tc.expected)
			} else {
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestCreateTaskCommand_ExtractAssignee(t *testing.T) {
	// Create command with empty mocks
	mockDB := new(MockDBManager)
	mockAI := new(MockAIClient)
	mockTodoist := new(MockTodoistClient)
	cmd := NewCreateTaskCommand(mockTodoist, mockDB, mockAI)

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "mention",
			input:    "Please @ivan handle this task",
			expected: "@ivan",
		},
		{
			name:     "assign phrase",
			input:    "назначить Ивану эту задачу",
			expected: "Ивану",
		},
		{
			name:     "responsible phrase",
			input:    "ответственный Петр",
			expected: "Петр",
		},
		{
			name:     "no assignee",
			input:    "This is a regular task",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := cmd.extractAssignee(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
