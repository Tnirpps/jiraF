package ai

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/user/telegram-bot/internal/httpclient"
)

// Client defines the interface for interacting with AI models
type Client interface {
	AnalyzeDiscussion(ctx context.Context, messages []string) (*AnalyzedTask, error)
	EditTask(ctx context.Context, task *AnalyzedTask, userFeedback string) (*AnalyzedTask, error)
}

// AnalyzedTask represents the structured task from AI analysis
type AnalyzedTask struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	DueDate      string   `json:"due_date"`
	Priority     int      `json:"priority"`
	PriorityText string   `json:"priority_text"`
	Labels       []string `json:"labels,omitempty"`
}

// AIClient is the implementation for AI analysis
type AIClient struct {
	httpClient       *httpclient.Client
	createTaskPrompt string
	editTaskPrompt   string
}

// NewClient creates a new AI client
func NewClient() (Client, error) {
	// Load configuration from YAML file
	configs, err := httpclient.LoadConfig("configs/api.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to load API configuration: %w", err)
	}

	// Get client configuration
	clientConfig, err := configs.GetClientConfig("ai_service")
	if err != nil {
		return nil, fmt.Errorf("failed to get AI client configuration: %w", err)
	}

	// Create the HTTP client
	client, err := clientConfig.CreateClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Prompt for creating a task from discussion
	createTaskPrompt := `
	Analyze this discussion and extract task information:
	- Extract a concise, descriptive title
	- Generate a comprehensive description
	- Identify any due date mentioned
	- Determine priority (1=Normal, 2=Medium, 3=High, 4=Urgent)
	- Extract relevant labels/tags

	Discussion:
	%s
	`

	// Prompt for editing a task based on user feedback
	editTaskPrompt := `
	The user wants to edit this task based on their feedback.
	Modify only the fields mentioned in the feedback, keeping all other fields the same.

	Current task:
	Title: %s
	Description: %s
	Due Date: %s
	Priority: %s
	Labels: %s

	User feedback:
	%s

	Return the updated task with all fields.
	`

	return &AIClient{
		httpClient:       client,
		createTaskPrompt: createTaskPrompt,
		editTaskPrompt:   editTaskPrompt,
	}, nil
}

// AnalyzeDiscussion analyzes messages using AI to extract task information
func (c *AIClient) AnalyzeDiscussion(ctx context.Context, messages []string) (*AnalyzedTask, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages to analyze")
	}

	// Join all messages into a single text
	discussionText := strings.Join(messages, "\n")

	// Format the prompt with our template for task creation
	prompt := fmt.Sprintf(c.createTaskPrompt, discussionText)

	// Log the prompt for debugging purposes
	log.Printf("AI Prompt generated (not used in placeholder implementation): %s", prompt)

	// In a production environment, this would call the real AI API:
	// return c.callAIAPI(ctx, prompt)

	// For this implementation, we'll return a placeholder task
	// but we're keeping the structure in place for real API integration
	return c.generatePlaceholderTask(discussionText), nil
}

// EditTask edits an existing task based on user feedback
func (c *AIClient) EditTask(ctx context.Context, task *AnalyzedTask, userFeedback string) (*AnalyzedTask, error) {
	if task == nil {
		return nil, fmt.Errorf("no task to edit")
	}

	if userFeedback == "" {
		return nil, fmt.Errorf("no feedback provided for editing")
	}

	// Format labels for prompt in a real implementation
	// This would be used in the prompt like:
	// labels := strings.Join(task.Labels, ", ")

	// Format the prompt with task details and user feedback
	// In a real implementation, this would call the AI API
	// For now, return a modified task based on simple rules

	// STUB: This is where we would make the real API call
	// Return a modified task as a placeholder
	return c.generatePlaceholderEditedTask(task, userFeedback), nil
}

// generatePlaceholderTask creates a simple placeholder task for new task creation
// This would be replaced with real AI API call implementation
func (c *AIClient) generatePlaceholderTask(text string) *AnalyzedTask {
	// Create a basic placeholder task with minimal processing
	// In a real implementation, this would parse the AI response

	// Extract a simple title (first line or truncated text)
	title := "Task from discussion"
	lines := strings.Split(text, "\n")
	if len(lines) > 0 && len(lines[0]) > 0 {
		title = lines[0]
		if len(title) > 50 {
			title = title[:47] + "..."
		}
	}

	return &AnalyzedTask{
		Title:        title,
		Description:  fmt.Sprintf("Discussion content:\n\n%s", text),
		DueDate:      "",
		Priority:     1,
		PriorityText: "Normal",
		Labels:       []string{},
	}
}

// generatePlaceholderEditedTask creates a placeholder for edited task
// This would be replaced with real AI API call implementation
func (c *AIClient) generatePlaceholderEditedTask(task *AnalyzedTask, feedback string) *AnalyzedTask {
	// For demonstration purposes, make simple changes based on feedback
	// In a real implementation, this would be replaced with AI analysis

	editedTask := &AnalyzedTask{
		Title:        task.Title,
		Description:  task.Description,
		DueDate:      task.DueDate,
		Priority:     task.Priority,
		PriorityText: task.PriorityText,
		Labels:       task.Labels,
	}

	// Very simple keyword-based changes (just for placeholder functionality)
	lowerFeedback := strings.ToLower(feedback)

	// Update title if requested
	if strings.Contains(lowerFeedback, "change title") ||
		strings.Contains(lowerFeedback, "rename") ||
		strings.Contains(lowerFeedback, "изменить название") {
		parts := strings.Split(feedback, ":")
		if len(parts) > 1 {
			editedTask.Title = strings.TrimSpace(parts[1])
		}
	}

	// Update priority if requested
	if strings.Contains(lowerFeedback, "high priority") ||
		strings.Contains(lowerFeedback, "высокий приоритет") {
		editedTask.Priority = 3
		editedTask.PriorityText = "High"
	} else if strings.Contains(lowerFeedback, "urgent") ||
		strings.Contains(lowerFeedback, "срочно") {
		editedTask.Priority = 4
		editedTask.PriorityText = "Urgent"
	}

	// Add placeholder for due date changes, etc.

	return editedTask
}

/*
// Example of what a real API call implementation might look like
func (c *AIClient) callAIAPI(ctx context.Context, prompt string) (*AnalyzedTask, error) {
	type AIRequest struct {
		Prompt string `json:"prompt"`
	}

	type AIResponse struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		DueDate     string   `json:"due_date"`
		Priority    int      `json:"priority"`
		Labels      []string `json:"labels"`
	}

	// Prepare the request
	request := AIRequest{Prompt: prompt}
	var response AIResponse

	// Make the API call
	err := c.httpClient.Post(ctx, "analyze", request, &response)
	if err != nil {
		return nil, fmt.Errorf("error calling AI API: %w", err)
	}

	// Map the API response to our AnalyzedTask structure
	priorityText := "Normal"
	switch response.Priority {
	case 2:
		priorityText = "Medium"
	case 3:
		priorityText = "High"
	case 4:
		priorityText = "Urgent"
	}

	return &AnalyzedTask{
		Title:        response.Title,
		Description:  response.Description,
		DueDate:      response.DueDate,
		Priority:     response.Priority,
		PriorityText: priorityText,
		Labels:       response.Labels,
	}, nil
}
*/
