package todoist

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"text/template"

	"github.com/user/telegram-bot/internal/httpclient"
)

// setupTestServer creates a test HTTP server that mimics the Todoist API
func setupTestServer(t *testing.T) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always check for proper auth
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, `{"error":"Invalid token"}`)
			return
		}

		// Handle different API endpoints
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/tasks":
			handleCreateTask(t, w, r)
		case r.Method == http.MethodGet && r.URL.Path == "/tasks":
			handleGetTasks(t, w, r)
		case r.Method == http.MethodDelete && r.URL.Path == "/tasks/123":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Logf("Unhandled request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return server
}

// handleCreateTask processes a task creation request
func handleCreateTask(t *testing.T, w http.ResponseWriter, r *http.Request) {
	var taskReq TaskRequest
	err := json.NewDecoder(r.Body).Decode(&taskReq)
	if err != nil {
		t.Logf("Error decoding request body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Verify task content
	if taskReq.Content == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"Content is required"}`)
		return
	}

	// Return a mock task response
	taskResp := TaskResponse{
		ID:          "123",
		Content:     taskReq.Content,
		Description: taskReq.Description,
		ProjectID:   taskReq.ProjectID,
		Priority:    taskReq.Priority,
		URL:         "https://todoist.com/showTask?id=123",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(taskResp)
}

// handleGetTasks processes a task listing request
func handleGetTasks(t *testing.T, w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")

	// Return mock tasks
	tasks := []*TaskResponse{
		{
			ID:          "123",
			Content:     "Task 1",
			Description: "Description 1",
			ProjectID:   projectID,
			Priority:    1,
		},
		{
			ID:          "124",
			Content:     "Task 2",
			Description: "Description 2",
			ProjectID:   projectID,
			Priority:    2,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tasks)
}

// createTestConfig creates a test configuration file from a template
func createTestConfig(t *testing.T, baseURL string) string {
	// Read the template
	tmpl, err := template.ParseFiles("testdata/config.yaml")
	if err != nil {
		t.Fatalf("Error parsing template: %v", err)
	}

	// Create a temporary file for the test config
	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		t.Fatalf("Error creating temp file: %v", err)
	}
	defer tmpFile.Close()

	// Execute the template with the server URL
	err = tmpl.Execute(tmpFile, struct {
		BaseURL string
	}{
		BaseURL: baseURL,
	})
	if err != nil {
		t.Fatalf("Error executing template: %v", err)
	}

	return tmpFile.Name()
}

// newTestClient creates a test TodoistClient using a custom configuration file
func newTestClient(t *testing.T, configPath string) Client {
	// Set up test token in environment
	os.Setenv("TEST_TOKEN", "test-token")
	defer os.Unsetenv("TEST_TOKEN")

	// Create a custom client
	configs, err := httpclient.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Error loading test config: %v", err)
	}

	clientConfig, err := configs.GetClientConfig("todoist")
	if err != nil {
		t.Fatalf("Error getting client config: %v", err)
	}

	client, err := clientConfig.CreateClient()
	if err != nil {
		t.Fatalf("Error creating HTTP client: %v", err)
	}

	return &TodoistClient{
		httpClient: client,
	}
}

// TestTodoistClient_CreateTask tests task creation
func TestTodoistClient_CreateTask(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	configPath := createTestConfig(t, server.URL)
	defer os.Remove(configPath)

	client := newTestClient(t, configPath)

	// Create a task
	task := &TaskRequest{
		Content:     "Test Task",
		Description: "Test Description",
		Priority:    1,
	}

	taskResp, err := client.CreateTask(context.Background(), task)
	if err != nil {
		t.Fatalf("Error creating task: %v", err)
	}

	// Verify response
	if taskResp.ID != "123" {
		t.Errorf("Expected task ID '123', got '%s'", taskResp.ID)
	}
	if taskResp.Content != "Test Task" {
		t.Errorf("Expected task content 'Test Task', got '%s'", taskResp.Content)
	}
}

// TestTodoistClient_GetTasks tests task listing
func TestTodoistClient_GetTasks(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	configPath := createTestConfig(t, server.URL)
	defer os.Remove(configPath)

	client := newTestClient(t, configPath)

	// Get tasks
	tasks, err := client.GetTasks(context.Background(), "456")
	if err != nil {
		t.Fatalf("Error getting tasks: %v", err)
	}

	// Verify response
	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].ID != "123" || tasks[1].ID != "124" {
		t.Errorf("Incorrect task IDs")
	}
}

// TestTodoistClient_DeleteTask tests task deletion
func TestTodoistClient_DeleteTask(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	configPath := createTestConfig(t, server.URL)
	defer os.Remove(configPath)

	client := newTestClient(t, configPath)

	// Delete a task
	err := client.DeleteTask(context.Background(), "123")
	if err != nil {
		t.Fatalf("Error deleting task: %v", err)
	}
}
