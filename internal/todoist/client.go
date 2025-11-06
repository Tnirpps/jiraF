package todoist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

const (
	// BaseURL is the base URL for Todoist API v2
	BaseURL = "https://api.todoist.com/rest/v2"
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
	apiToken   string
	httpClient *http.Client
}

// NewClient creates a new Todoist client
func NewClient(apiToken string) Client {
	return &TodoistClient{
		apiToken:   apiToken,
		httpClient: http.DefaultClient,
	}
}

// CreateTask creates a new task in Todoist
func (c *TodoistClient) CreateTask(ctx context.Context, task *TaskRequest) (*TaskResponse, error) {
	if task.Content == "" {
		return nil, fmt.Errorf("task content is required")
	}

	reqBody, err := json.Marshal(task)
	if err != nil {
		return nil, fmt.Errorf("error marshaling task: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, BaseURL+"/tasks", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("X-Request-Id", fmt.Sprintf("task-%d", time.Now().UnixNano()))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return nil, fmt.Errorf("API error (status %d): %v", resp.StatusCode, errResp)
		}
		return nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	var createdTask TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&createdTask); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	log.Printf("Created Todoist task: %s with ID %s", createdTask.Content, createdTask.ID)
	return &createdTask, nil
}

// GetTasks returns active tasks, optionally filtered by project ID
func (c *TodoistClient) GetTasks(ctx context.Context, projectID string) ([]*TaskResponse, error) {
	url := BaseURL + "/tasks"
	if projectID != "" {
		url += "?project_id=" + projectID
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return nil, fmt.Errorf("API error (status %d): %v", resp.StatusCode, errResp)
		}
		return nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	var tasks []*TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return tasks, nil
}

// GetTask returns a single task by ID
func (c *TodoistClient) GetTask(ctx context.Context, taskID string) (*TaskResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/tasks/%s", BaseURL, taskID), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return nil, fmt.Errorf("API error (status %d): %v", resp.StatusCode, errResp)
		}
		return nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	var task TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &task, nil
}

// UpdateTask updates an existing task
func (c *TodoistClient) UpdateTask(ctx context.Context, taskID string, task *TaskRequest) (*TaskResponse, error) {
	reqBody, err := json.Marshal(task)
	if err != nil {
		return nil, fmt.Errorf("error marshaling task: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/tasks/%s", BaseURL, taskID), bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("X-Request-Id", fmt.Sprintf("update-%d", time.Now().UnixNano()))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return nil, fmt.Errorf("API error (status %d): %v", resp.StatusCode, errResp)
		}
		return nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	var updatedTask TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&updatedTask); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	log.Printf("Updated Todoist task: %s with ID %s", updatedTask.Content, updatedTask.ID)
	return &updatedTask, nil
}

// CompleteTask marks a task as complete
func (c *TodoistClient) CompleteTask(ctx context.Context, taskID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/tasks/%s/close", BaseURL, taskID), nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("X-Request-Id", fmt.Sprintf("close-%d", time.Now().UnixNano()))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		var errResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return fmt.Errorf("API error (status %d): %v", resp.StatusCode, errResp)
		}
		return fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	log.Printf("Marked Todoist task %s as complete", taskID)
	return nil
}

// DeleteTask permanently deletes a task
func (c *TodoistClient) DeleteTask(ctx context.Context, taskID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fmt.Sprintf("%s/tasks/%s", BaseURL, taskID), nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		var errResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return fmt.Errorf("API error (status %d): %v", resp.StatusCode, errResp)
		}
		return fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	log.Printf("Deleted Todoist task %s", taskID)
	return nil
}

// GetProjects returns the list of projects
func (c *TodoistClient) GetProjects(ctx context.Context) ([]Project, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, BaseURL+"/projects", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return nil, fmt.Errorf("API error (status %d): %v", resp.StatusCode, errResp)
		}
		return nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	var projects []Project
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return projects, nil
}
