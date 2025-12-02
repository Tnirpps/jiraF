package todoist

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/user/telegram-bot/internal/httpclient"
)

// TaskRequest represents the request structure for creating a Todoist task
type TaskRequest struct {
	Content      string   `json:"content"` // Required
	Description  string   `json:"description,omitempty"`
	ProjectID    string   `json:"project_id,omitempty"`
	SectionID    string   `json:"section_id,omitempty"`
	ParentID     string   `json:"parent_id,omitempty"`
	Order        int      `json:"order,omitempty"`
	Labels       []string `json:"labels,omitempty"`
	Priority     int      `json:"priority,omitempty"`
	AssigneeID   string   `json:"assignee_id,omitempty"`
	DueString    string   `json:"due_string,omitempty"`
	DueDate      string   `json:"due_date,omitempty"`
	DueDateTime  string   `json:"due_datetime,omitempty"`
	DueLang      string   `json:"due_lang,omitempty"`
	Duration     int      `json:"duration,omitempty"`
	DurationUnit string   `json:"duration_unit,omitempty"`
	DeadlineDate string   `json:"deadline_date,omitempty"`
}

// DueObject represents the due date information for a task
type DueObject struct {
	String      string `json:"string"`
	Date        string `json:"date"`
	IsRecurring bool   `json:"is_recurring"`
	DateTime    string `json:"datetime,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
	Lang        string `json:"lang,omitempty"`
}

// DurationObject represents the duration information for a task
type DurationObject struct {
	Amount int    `json:"amount"`
	Unit   string `json:"unit"`
}

// TaskResponse represents the response from Todoist when creating a task
type TaskResponse struct {
	ID           string            `json:"id"`
	Content      string            `json:"content"`
	Description  string            `json:"description"`
	ProjectID    string            `json:"project_id"`
	SectionID    string            `json:"section_id,omitempty"`
	ParentID     string            `json:"parent_id,omitempty"`
	Order        int               `json:"order"`
	Labels       []string          `json:"labels"`
	Priority     int               `json:"priority"`
	Due          *DueObject        `json:"due,omitempty"`
	Deadline     map[string]string `json:"deadline,omitempty"`
	Duration     *DurationObject   `json:"duration,omitempty"`
	URL          string            `json:"url"`
	CommentCount int               `json:"comment_count"`
	IsCompleted  bool              `json:"is_completed"`
	CreatedAt    string            `json:"created_at"`
	CreatorID    string            `json:"creator_id"`
	AssigneeID   string            `json:"assignee_id,omitempty"`
	AssignerID   string            `json:"assigner_id,omitempty"`
}

// Project represents a Todoist project
type Project struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Color          string `json:"color"`
	CommentCount   int    `json:"comment_count"`
	Order          int    `json:"order"`
	IsShared       bool   `json:"is_shared"`
	IsFavorite     bool   `json:"is_favorite"`
	IsInboxProject bool   `json:"is_inbox_project"`
	IsTeamInbox    bool   `json:"is_team_inbox"`
	ViewStyle      string `json:"view_style"`
	URL            string `json:"url"`
	ParentID       string `json:"parent_id,omitempty"`
}

// Client defines the interface for interacting with the Todoist API
type Client interface {
	// CreateTask creates a new task in Todoist
	CreateTask(ctx context.Context, task *TaskRequest) (*TaskResponse, error)
	// GetProjects returns the list of projects
	GetProjects(ctx context.Context) ([]Project, error)
	// GetTasks returns active tasks, optionally filtered by project ID
	GetTasks(ctx context.Context, projectID string) ([]*TaskResponse, error)
	// GetTask returns a single task by ID
	GetTask(ctx context.Context, taskID string) (*TaskResponse, error)
	// UpdateTask updates an existing task
	UpdateTask(ctx context.Context, taskID string, task *TaskRequest) (*TaskResponse, error)
	// CompleteTask marks a task as complete
	CompleteTask(ctx context.Context, taskID string) error
	// DeleteTask permanently deletes a task
	DeleteTask(ctx context.Context, taskID string) error
}

// TodoistClient is the implementation of the Client interface
type TodoistClient struct {
	httpClient *httpclient.Client
}

// NewClient creates a new Todoist client
func NewClient() (Client, error) {
	// Load configuration from YAML file
	configs, err := httpclient.LoadConfig("configs/api.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to load API configuration: %w", err)
	}

	// Get Todoist client configuration
	clientConfig, err := configs.GetClientConfig("todoist")
	if err != nil {
		return nil, fmt.Errorf("failed to get Todoist client configuration: %w", err)
	}

	// Create the HTTP client
	client, err := clientConfig.CreateClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Add request ID middleware for idempotent operations
	client.WithMiddleware(func(next httpclient.Handler) httpclient.Handler {
		return func(ctx context.Context, req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodPost || req.Method == http.MethodPut {
				req.Header.Set("X-Request-Id", fmt.Sprintf("req-%d", time.Now().UnixNano()))
			}
			return next(ctx, req)
		}
	})

	return &TodoistClient{
		httpClient: client,
	}, nil
}

// CreateTask creates a new task in Todoist
func (c *TodoistClient) CreateTask(ctx context.Context, task *TaskRequest) (*TaskResponse, error) {
	if task.Content == "" {
		return nil, fmt.Errorf("task content is required")
	}

	var createdTask TaskResponse
	err := c.httpClient.Post(ctx, "tasks", task, &createdTask)
	if err != nil {
		return nil, fmt.Errorf("error creating task: %w", err)
	}

	log.Printf("Created Todoist task: %s with ID %s", createdTask.Content, createdTask.ID)
	return &createdTask, nil
}

// GetTasks returns active tasks, optionally filtered by project ID
func (c *TodoistClient) GetTasks(ctx context.Context, projectID string) ([]*TaskResponse, error) {
	path := "tasks"
	if projectID != "" {
		path += "?project_id=" + projectID
	}

	var tasks []*TaskResponse
	err := c.httpClient.Get(ctx, path, &tasks)
	if err != nil {
		return nil, fmt.Errorf("error getting tasks: %w", err)
	}

	return tasks, nil
}

// GetTask returns a single task by ID
func (c *TodoistClient) GetTask(ctx context.Context, taskID string) (*TaskResponse, error) {
	var task TaskResponse
	err := c.httpClient.Get(ctx, fmt.Sprintf("tasks/%s", taskID), &task)
	if err != nil {
		// Check if it's a 404 error
		if httpclient.IsNotFound(err) {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
		return nil, fmt.Errorf("error getting task: %w", err)
	}

	return &task, nil
}

// UpdateTask updates an existing task
func (c *TodoistClient) UpdateTask(ctx context.Context, taskID string, task *TaskRequest) (*TaskResponse, error) {
	var updatedTask TaskResponse
	err := c.httpClient.Post(ctx, fmt.Sprintf("tasks/%s", taskID), task, &updatedTask)
	if err != nil {
		return nil, fmt.Errorf("error updating task: %w", err)
	}

	log.Printf("Updated Todoist task: %s with ID %s", updatedTask.Content, updatedTask.ID)
	return &updatedTask, nil
}

// CompleteTask marks a task as complete
func (c *TodoistClient) CompleteTask(ctx context.Context, taskID string) error {
	err := c.httpClient.Post(ctx, fmt.Sprintf("tasks/%s/close", taskID), nil, nil)
	if err != nil {
		return fmt.Errorf("error completing task: %w", err)
	}

	log.Printf("Marked Todoist task %s as complete", taskID)
	return nil
}

// DeleteTask permanently deletes a task
func (c *TodoistClient) DeleteTask(ctx context.Context, taskID string) error {
	err := c.httpClient.Delete(ctx, fmt.Sprintf("tasks/%s", taskID))
	if err != nil {
		return fmt.Errorf("error deleting task: %w", err)
	}

	log.Printf("Deleted Todoist task %s", taskID)
	return nil
}

// GetProjects returns the list of projects
func (c *TodoistClient) GetProjects(ctx context.Context) ([]Project, error) {
	var projects []Project
	err := c.httpClient.Get(ctx, "projects", &projects)
	if err != nil {
		return nil, fmt.Errorf("error getting projects: %w", err)
	}

	return projects, nil
}
