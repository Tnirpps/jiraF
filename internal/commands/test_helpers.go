package commands

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/mock"
	"github.com/user/telegram-bot/internal/db"
)

// CreateCommandMessage is a helper function to create a Telegram message with a command
// for testing purposes. It properly sets up message entities required for commands.
func CreateCommandMessage(chatID int64, commandText string, args ...string) *tgbotapi.Message {
	var fullText string
	if len(args) > 0 {
		fullText = commandText + " " + args[0]
	} else {
		fullText = commandText
	}

	// Command entity length is the length of the command, including the /
	commandLength := len(commandText)

	return &tgbotapi.Message{
		Chat: &tgbotapi.Chat{
			ID: chatID,
		},
		Text: fullText,
		Entities: []tgbotapi.MessageEntity{
			{
				Type:   "bot_command",
				Offset: 0,
				Length: commandLength,
			},
		},
	}
}

// Mock DBManager for testing - move this from set_project_test.go to have it in one place
type MockDBManager struct {
	mock.Mock
}

func (m *MockDBManager) EnsureChatExists(ctx context.Context, chatID int64) error {
	args := m.Called(ctx, chatID)
	return args.Error(0)
}

func (m *MockDBManager) SetTodoistProjectID(ctx context.Context, chatID int64, projectID string) error {
	args := m.Called(ctx, chatID, projectID)
	return args.Error(0)
}

func (m *MockDBManager) GetTodoistProjectID(ctx context.Context, chatID int64) (string, error) {
	args := m.Called(ctx, chatID)
	return args.String(0), args.Error(1)
}

func (m *MockDBManager) HasActiveSession(ctx context.Context, chatID int64) (bool, error) {
	args := m.Called(ctx, chatID)
	return args.Bool(0), args.Error(1)
}

func (m *MockDBManager) GetActiveSession(ctx context.Context, chatID int64) (*db.Session, error) {
	args := m.Called(ctx, chatID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.Session), args.Error(1)
}

func (m *MockDBManager) StartSession(ctx context.Context, chatID int64) (int, error) {
	args := m.Called(ctx, chatID)
	return args.Int(0), args.Error(1)
}

func (m *MockDBManager) CloseSession(ctx context.Context, chatID int64) error {
	args := m.Called(ctx, chatID)
	return args.Error(0)
}

func (m *MockDBManager) SaveMessage(ctx context.Context, chatID int64, messageID int, userID int64, username, text string) error {
	args := m.Called(ctx, chatID, messageID, userID, username, text)
	return args.Error(0)
}

func (m *MockDBManager) GetSessionMessages(ctx context.Context, sessionID int) ([]db.Message, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).([]db.Message), args.Error(1)
}

// Helper functions for fluent API style mock configuration
func ConfigureMockDB(m *MockDBManager) *MockDBHelper {
	return &MockDBHelper{mock: m}
}

// MockDBHelper provides a fluent interface for configuring mock expectations
type MockDBHelper struct {
	mock *MockDBManager
}

// WithProjectID sets up the mock to expect and respond to GetTodoistProjectID calls
func (h *MockDBHelper) WithProjectID(chatID int64, projectID string, err error) *MockDBHelper {
	h.mock.On("GetTodoistProjectID", mock.Anything, chatID).Return(projectID, err)
	return h
}

// WithSetProjectID sets up the mock to expect and respond to SetTodoistProjectID calls
func (h *MockDBHelper) WithSetProjectID(chatID int64, projectID string, err error) *MockDBHelper {
	h.mock.On("SetTodoistProjectID", mock.Anything, chatID, projectID).Return(err)
	return h
}

// WithActiveSession sets up the mock to expect and respond to HasActiveSession calls
func (h *MockDBHelper) WithActiveSession(chatID int64, hasActive bool, err error) *MockDBHelper {
	h.mock.On("HasActiveSession", mock.Anything, chatID).Return(hasActive, err)
	return h
}

// WithStartSession sets up the mock to expect and respond to StartSession calls
func (h *MockDBHelper) WithStartSession(chatID int64, sessionID int, err error) *MockDBHelper {
	h.mock.On("StartSession", mock.Anything, chatID).Return(sessionID, err)
	return h
}

// WithCloseSession sets up the mock to expect and respond to CloseSession calls
func (h *MockDBHelper) WithCloseSession(chatID int64, err error) *MockDBHelper {
	h.mock.On("CloseSession", mock.Anything, chatID).Return(err)
	return h
}
