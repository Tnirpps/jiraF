package commands

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/mock"
	"github.com/user/telegram-bot/internal/ai"
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

	// Default user ID is the same as chat ID for simplicity in testing
	userID := chatID

	return &tgbotapi.Message{
		Chat: &tgbotapi.Chat{
			ID: chatID,
		},
		From: &tgbotapi.User{
			ID: userID,
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

func (m *MockDBManager) StartSession(ctx context.Context, chatID int64, ownerID int64) (int, error) {
	args := m.Called(ctx, chatID, ownerID)
	return args.Int(0), args.Error(1)
}

func (m *MockDBManager) IsSessionOwner(ctx context.Context, sessionID int, userID int64) (bool, error) {
	args := m.Called(ctx, sessionID, userID)
	return args.Bool(0), args.Error(1)
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

func (m *MockDBManager) SaveDraftTask(ctx context.Context, sessionID int, title, description, dueISO string, priority int, assigneeNote string) error {
	args := m.Called(ctx, sessionID, title, description, dueISO, priority, assigneeNote)
	return args.Error(0)
}

func (m *MockDBManager) GetDraftTask(ctx context.Context, sessionID int) (db.DraftTask, error) {
	args := m.Called(ctx, sessionID)
	if v := args.Get(0); v != nil {
		if dt, ok := v.(db.DraftTask); ok {
			return dt, args.Error(1)
		}
	}
	return db.DraftTask{}, args.Error(1)
}

func (m *MockDBManager) SaveCreatedTask(ctx context.Context, sessionID int, todoistTaskID, url string) error {
	args := m.Called(ctx, sessionID, todoistTaskID, url)
	return args.Error(0)
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
func (h *MockDBHelper) WithStartSession(chatID int64, ownerID int64, sessionID int, err error) *MockDBHelper {
	h.mock.On("StartSession", mock.Anything, chatID, ownerID).Return(sessionID, err)
	return h
}

// WithIsSessionOwner sets up the mock to expect and respond to IsSessionOwner calls
func (h *MockDBHelper) WithIsSessionOwner(sessionID int, userID int64, isOwner bool, err error) *MockDBHelper {
	h.mock.On("IsSessionOwner", mock.Anything, sessionID, userID).Return(isOwner, err)
	return h
}

// WithCloseSession sets up the mock to expect and respond to CloseSession calls
func (h *MockDBHelper) WithCloseSession(chatID int64, err error) *MockDBHelper {
	h.mock.On("CloseSession", mock.Anything, chatID).Return(err)
	return h
}

// Mock AI model client
type AIClientMock struct {
	mock.Mock
}

func (m *AIClientMock) AnalyzeDiscussion(ctx context.Context, messages []string) (*ai.AnalyzedTask, error) {
	args := m.Called(ctx, messages)
	if v := args.Get(0); v != nil {
		return v.(*ai.AnalyzedTask), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *AIClientMock) EditTask(ctx context.Context, task *ai.AnalyzedTask, userFeedback string) (*ai.AnalyzedTask, error) {
	args := m.Called(ctx, task, userFeedback)
	if v := args.Get(0); v != nil {
		return v.(*ai.AnalyzedTask), args.Error(1)
	}
	return nil, args.Error(1)
}

type AIClientMockMockHelper struct {
	m *AIClientMock
}

func ConfigureClientMock(m *AIClientMock) *AIClientMockMockHelper {
	return &AIClientMockMockHelper{m: m}
}

func (h *AIClientMockMockHelper) AnalyzeDiscussionExact(msgs []string, res *ai.AnalyzedTask, err error) *AIClientMockMockHelper {
	h.m.On("AnalyzeDiscussion", mock.Anything, msgs).Return(res, err)
	return h
}

func (h *AIClientMockMockHelper) EditTaskExact(task *ai.AnalyzedTask, feedback string, res *ai.AnalyzedTask, err error) *AIClientMockMockHelper {
	h.m.On("EditTask", mock.Anything, task, feedback).Return(res, err)
	return h
}
