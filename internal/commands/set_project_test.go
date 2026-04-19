package commands

import (
	"context"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/user/telegram-bot/internal/todoist"
)

// MockTodoistClient for testing
type MockTodoistClient struct {
	mock.Mock
}

func (m *MockTodoistClient) CreateTask(ctx context.Context, task *todoist.TaskRequest) (*todoist.TaskResponse, error) {
	args := m.Called(ctx, task)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*todoist.TaskResponse), args.Error(1)
}

func (m *MockTodoistClient) GetProjects(ctx context.Context) ([]todoist.Project, error) {
	args := m.Called(ctx)
	return args.Get(0).([]todoist.Project), args.Error(1)
}

func (m *MockTodoistClient) GetTasks(ctx context.Context, projectID string) ([]*todoist.TaskResponse, error) {
	args := m.Called(ctx, projectID)
	return args.Get(0).([]*todoist.TaskResponse), args.Error(1)
}

func (m *MockTodoistClient) GetTask(ctx context.Context, taskID string) (*todoist.TaskResponse, error) {
	args := m.Called(ctx, taskID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*todoist.TaskResponse), args.Error(1)
}

func (m *MockTodoistClient) UpdateTask(ctx context.Context, taskID string, task *todoist.TaskRequest) (*todoist.TaskResponse, error) {
	args := m.Called(ctx, taskID, task)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*todoist.TaskResponse), args.Error(1)
}

func (m *MockTodoistClient) CompleteTask(ctx context.Context, taskID string) error {
	args := m.Called(ctx, taskID)
	return args.Error(0)
}

func (m *MockTodoistClient) DeleteTask(ctx context.Context, taskID string) error {
	args := m.Called(ctx, taskID)
	return args.Error(0)
}

func TestSetProjectCommand_Execute_ShowsProjects(t *testing.T) {
	mockTodoistClient := new(MockTodoistClient)
	mockDBManager := new(MockDBManager)

	cmd := NewSetProjectCommand(mockTodoistClient, mockDBManager)

	chatID := int64(123456789)
	projects := []todoist.Project{
		{ID: "12345", Name: "Backend"},
		{ID: "67890", Name: "Frontend"},
	}

	mockTodoistClient.On("GetProjects", mock.Anything).Return(projects, nil)

	message := CreateCommandMessage(chatID, "/set_project")
	response := cmd.Execute(message)

	assert.Contains(t, response.Text, "Выберите проект Todoist")
	markup, ok := response.ReplyMarkup.(tgbotapi.InlineKeyboardMarkup)
	assert.True(t, ok)
	assert.Len(t, markup.InlineKeyboard, 2)
	assert.Equal(t, "Backend", markup.InlineKeyboard[0][0].Text)
	if assert.NotNil(t, markup.InlineKeyboard[0][0].CallbackData) {
		assert.Equal(t, "select_project:12345", *markup.InlineKeyboard[0][0].CallbackData)
	}
	assert.Equal(t, "Frontend", markup.InlineKeyboard[1][0].Text)
	if assert.NotNil(t, markup.InlineKeyboard[1][0].CallbackData) {
		assert.Equal(t, "select_project:67890", *markup.InlineKeyboard[1][0].CallbackData)
	}

	mockTodoistClient.AssertExpectations(t)
	mockDBManager.AssertNotCalled(t, "SetTodoistProjectID", mock.Anything, mock.Anything, mock.Anything)
}
