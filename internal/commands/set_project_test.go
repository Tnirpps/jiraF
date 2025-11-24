package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/user/telegram-bot/internal/todoist"
)

// Mock TodoistClient for testing
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

// MockDBManager is now defined in test_helpers.go

func TestSetProjectCommand_Execute_Success(t *testing.T) {
	// Create mocks
	mockTodoistClient := new(MockTodoistClient)
	mockDBManager := new(MockDBManager)

	// Create command
	// Create command with the interface
	cmd := NewSetProjectCommand(mockTodoistClient, mockDBManager)

	// Test data
	chatID := int64(123456789)
	projectID := "12345"

	// Set up expectations
	mockTodoistClient.On("GetProjects", mock.Anything).Return([]todoist.Project{
		{ID: projectID, Name: "Test Project"},
	}, nil)

	// Configure DBManager with fluent API
	ConfigureMockDB(mockDBManager).
		WithSetProjectID(chatID, projectID, nil)

	// Create test message using helper function
	message := CreateCommandMessage(chatID, "/set_project", projectID)

	// Execute command
	response := cmd.Execute(message)

	// Assert response
	assert.Contains(t, response.Text, "Project ID set to: "+projectID)

	// Verify mocks
	mockTodoistClient.AssertExpectations(t)
	mockDBManager.AssertExpectations(t)
}

func TestSetProjectCommand_Execute_InvalidProject(t *testing.T) {
	// Create mocks
	mockTodoistClient := new(MockTodoistClient)
	mockDBManager := new(MockDBManager)

	// Create command
	cmd := NewSetProjectCommand(mockTodoistClient, mockDBManager)

	// Test data
	chatID := int64(123456789)
	projectID := "12345"

	// Set up expectations
	mockTodoistClient.On("GetProjects", mock.Anything).Return([]todoist.Project{
		{ID: "98765", Name: "Different Project"},
	}, nil)

	// Create test message using helper function
	message := CreateCommandMessage(chatID, "/set_project", projectID)

	// Execute command
	response := cmd.Execute(message)

	// Assert response
	assert.Contains(t, response.Text, "Invalid project ID")

	// Verify mocks
	mockTodoistClient.AssertExpectations(t)
	mockDBManager.AssertNotCalled(t, "SetTodoistProjectID")
}

func TestSetProjectCommand_Execute_ExtractProjectIDFromURL(t *testing.T) {
	// Create mocks
	mockTodoistClient := new(MockTodoistClient)
	mockDBManager := new(MockDBManager)

	// Create command
	cmd := NewSetProjectCommand(mockTodoistClient, mockDBManager)

	// Test data
	chatID := int64(123456789)
	projectID := "12345"
	projectURL := "https://todoist.com/app/projects/12345"

	// Set up expectations
	mockTodoistClient.On("GetProjects", mock.Anything).Return([]todoist.Project{
		{ID: projectID, Name: "Test Project"},
	}, nil)

	// Configure DBManager with fluent API
	ConfigureMockDB(mockDBManager).
		WithSetProjectID(chatID, projectID, nil)

	// Create test message using helper function
	message := CreateCommandMessage(chatID, "/set_project", projectURL)

	// Execute command
	response := cmd.Execute(message)

	// Assert response
	assert.Contains(t, response.Text, "Project ID set to: "+projectID)

	// Verify mocks
	mockTodoistClient.AssertExpectations(t)
	mockDBManager.AssertExpectations(t)
}
