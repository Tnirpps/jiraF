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

// setupTestServer creates a mock HTTP server that simulates the Todoist API for testing
func setupTestServer(t *testing.T) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, `{"error":"Invalid token"}`)
			return
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/tasks":
			handleCreateTask(t, w, r)
		case r.Method == http.MethodGet && r.URL.Path == "/tasks":
			handleGetTasks(t, w, r)
		case r.Method == http.MethodGet && r.URL.Path == "/projects":
			handleGetProjects(t, w, r)
		case r.Method == http.MethodDelete && r.URL.Path == "/tasks/123":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Logf("Unhandled request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return server
}

// handleCreateTask processes mock task creation requests and returns a simulated response
func handleCreateTask(t *testing.T, w http.ResponseWriter, r *http.Request) {
	var taskReq TaskRequest
	err := json.NewDecoder(r.Body).Decode(&taskReq)
	if err != nil {
		t.Logf("Error decoding request body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if taskReq.Content == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"Content is required"}`)
		return
	}

	taskResp := TaskResponse{
		ID:          "123",
		Content:     taskReq.Content,
		Description: taskReq.Description,
		ProjectID:   taskReq.ProjectID,
		Labels:      taskReq.Labels,
		Priority:    taskReq.Priority,
		URL:         "https://todoist.com/showTask?id=123",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(taskResp)
}

// handleGetTasks processes mock task listing requests and returns tasks wrapped in TasksResponse
func handleGetTasks(t *testing.T, w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")

	tasks := TasksResponse{
		Results: []*TaskResponse{
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
		},
		NextCursor: nil,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tasks)
}

// handleGetProjects processes mock project listing requests and returns projects wrapped in ProjectsResponse
func handleGetProjects(t *testing.T, w http.ResponseWriter, r *http.Request) {
	projects := ProjectsResponse{
		Results: []Project{
			{
				ID:   "12345",
				Name: "Test Project",
			},
			{
				ID:   "67890",
				Name: "Another Project",
			},
		},
		NextCursor: nil,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(projects)
}

// createTestConfig creates a temporary configuration file from a template for testing
func createTestConfig(t *testing.T, baseURL string) string {
	tmpl, err := template.ParseFiles("testdata/config.yaml")
	if err != nil {
		t.Fatalf("Error parsing template: %v", err)
	}

	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		t.Fatalf("Error creating temp file: %v", err)
	}
	defer tmpFile.Close()

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

// newTestClient creates a test Todoist client with a mock HTTP server configuration
func newTestClient(t *testing.T, configPath string) Client {
	os.Setenv("TEST_TOKEN", "test-token")
	defer os.Unsetenv("TEST_TOKEN")

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

// Tests that the Todoist client successfully creates a task with valid content
// Verifies that the task ID and content are correctly returned from the API
func TestTodoistClient_CreateTask(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	configPath := createTestConfig(t, server.URL)
	defer os.Remove(configPath)

	client := newTestClient(t, configPath)

	task := &TaskRequest{
		Content:     "Test Task",
		Description: "Test Description",
		Labels:      []string{"backend", "urgent"},
		Priority:    1,
	}

	taskResp, err := client.CreateTask(context.Background(), task)
	if err != nil {
		t.Fatalf("Error creating task: %v", err)
	}

	if taskResp.ID != "123" {
		t.Errorf("Expected task ID '123', got '%s'", taskResp.ID)
	}
	if taskResp.Content != "Test Task" {
		t.Errorf("Expected task content 'Test Task', got '%s'", taskResp.Content)
	}
	if len(taskResp.Labels) != 2 || taskResp.Labels[0] != "backend" || taskResp.Labels[1] != "urgent" {
		t.Errorf("Expected labels to round-trip, got %#v", taskResp.Labels)
	}
}

// Tests that the Todoist client successfully retrieves a list of tasks from a project
// Verifies that the correct number of tasks is returned with proper IDs
func TestTodoistClient_GetTasks(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	configPath := createTestConfig(t, server.URL)
	defer os.Remove(configPath)

	client := newTestClient(t, configPath)

	tasks, err := client.GetTasks(context.Background(), "456")
	if err != nil {
		t.Fatalf("Error getting tasks: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].ID != "123" || tasks[1].ID != "124" {
		t.Errorf("Incorrect task IDs")
	}
}

// Tests that the Todoist client successfully deletes a task by ID
// Verifies that no error is returned on successful deletion
func TestTodoistClient_DeleteTask(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	configPath := createTestConfig(t, server.URL)
	defer os.Remove(configPath)

	client := newTestClient(t, configPath)

	err := client.DeleteTask(context.Background(), "123")
	if err != nil {
		t.Fatalf("Error deleting task: %v", err)
	}
}

// Tests that the Todoist client successfully retrieves a list of projects
// Verifies that the correct number of projects is returned with proper IDs and names
func TestTodoistClient_GetProjects(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	configPath := createTestConfig(t, server.URL)
	defer os.Remove(configPath)

	client := newTestClient(t, configPath)

	projects, err := client.GetProjects(context.Background())
	if err != nil {
		t.Fatalf("Error getting projects: %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}
	if projects[0].ID != "12345" || projects[1].ID != "67890" {
		t.Errorf("Incorrect project IDs")
	}
}
