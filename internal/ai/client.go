package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

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

// YandexGPTRequest structure for YandexGPT API request
type YandexGPTRequest struct {
	ModelURI          string                  `json:"modelUri"`
	CompletionOptions *CompletionOptions      `json:"completionOptions"`
	Messages          []YandexGPTMessage      `json:"messages"`
}

// YandexGPTMessage structure for YandexGPT message
type YandexGPTMessage struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

// CompletionOptions completion options for YandexGPT
type CompletionOptions struct {
	Stream      bool    `json:"stream"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"maxTokens"`
}

// YandexGPTResponse structure for YandexGPT API response
type YandexGPTResponse struct {
	Result *GPTResult `json:"result"`
}

// GPTResult result of processing
type GPTResult struct {
	Alternatives []Alternative `json:"alternatives"`
	Usage        Usage         `json:"usage"`
	ModelVersion string        `json:"modelVersion"`
}

// Alternative alternative response
type Alternative struct {
	Message MessageContent `json:"message"`
	Status  string         `json:"status"`
}

// MessageContent message content
type MessageContent struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

// Usage usage statistics
type Usage struct {
	InputTextTokens  int `json:"inputTextTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// AIClient is the implementation for AI analysis with YandexGPT
type AIClient struct {
	httpClient       *httpclient.Client
	apiKey           string
	folderID         string
	modelURI         string
	createTaskPrompt string
	editTaskPrompt   string
}

// NewClient creates a new AI client for YandexGPT
func NewClient() (Client, error) {
	// Load configuration from YAML file
	configs, err := httpclient.LoadConfig("configs/api.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to load API configuration: %w", err)
	}

	// Get client configuration
	clientConfig, err := configs.GetClientConfig("yandex_gpt")
	if err != nil {
		return nil, fmt.Errorf("failed to get YandexGPT client configuration: %w", err)
	}

	// Create the HTTP client
	client, err := clientConfig.CreateClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Get API key from environment variable
	apiKey := os.Getenv("YANDEX_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("YANDEX_API_KEY environment variable is required")
	}

	// Get folder ID from environment variable
	folderID := os.Getenv("YANDEX_FOLDER_ID")
	if folderID == "" {
		return nil, fmt.Errorf("YANDEX_FOLDER_ID environment variable is required")
	}

	// Validate API key format
	if len(apiKey) < 20 {
		return nil, fmt.Errorf("YANDEX_API_KEY appears to be invalid (too short)")
	}

	// Model URI for YandexGPT
	modelURI := fmt.Sprintf("gpt://%s/yandexgpt-lite", folderID)

	// Add authorization header to the client
	client.WithMiddleware(func(next httpclient.Handler) httpclient.Handler {
		return func(ctx context.Context, req *http.Request) (*http.Response, error) {
			req.Header.Set("Authorization", fmt.Sprintf("Api-Key %s", apiKey))
			req.Header.Set("Content-Type", "application/json")
			return next(ctx, req)
		}
	})

	// Test connection to API
	testCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test request to verify API connection
	testRequest := YandexGPTRequest{
		ModelURI: modelURI,
		CompletionOptions: &CompletionOptions{
			Stream:      false,
			Temperature: 0.1,
			MaxTokens:   10,
		},
		Messages: []YandexGPTMessage{
			{
				Role: "user",
				Text: "Hello, reply with one word: ready",
			},
		},
	}

	// Make test API call
	var testResponse YandexGPTResponse
	err = client.Post(testCtx, "completion", testRequest, &testResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to YandexGPT API: %w", err)
	}

	// System prompt for creating a task from discussion
	createTaskPrompt := `You are a task management assistant. Analyze the dialog and extract information for a Todoist task.

Response requirements:
1. Response must be in JSON format
2. All fields are required
3. Use Russian language for fields

JSON format:
{
  "title": "Brief, informative task title (max 100 characters)",
  "description": "Detailed task description, including all important details from the discussion",
  "due_date": "Due date in YYYY-MM-DD format or relative format (today, tomorrow, mon, tue, etc.). If no due date mentioned, leave empty string",
  "priority": "Priority from 1 to 4, where 1 is normal, 4 is urgent",
  "priority_text": "Text description of priority (Normal, Medium, High, Urgent)",
  "labels": ["list", "of", "relevant", "tags"]
}

Rules:
- Title should be specific and informative
- Description should include all technical details mentioned in the discussion
- For priority: use 4 only for truly urgent tasks
- Tags should be relevant to context (e.g.: frontend, backend, bug, feature, meeting)

Dialog to analyze:
`

	// System prompt for editing a task based on user feedback
	editTaskPrompt := `You are a task management assistant. Edit an existing task based on user feedback.

Requirements:
1. Change only the fields mentioned in the feedback
2. Keep all other fields unchanged
3. Response must be in JSON format
4. Use Russian language for fields

Current task:
%s

User feedback:
%s

Return the updated task in JSON format.`

	return &AIClient{
		httpClient:       client,
		apiKey:           apiKey,
		folderID:         folderID,
		modelURI:         modelURI,
		createTaskPrompt: createTaskPrompt,
		editTaskPrompt:   editTaskPrompt,
	}, nil
}

// AnalyzeDiscussion analyzes messages using YandexGPT to extract task information
func (c *AIClient) AnalyzeDiscussion(ctx context.Context, messages []string) (*AnalyzedTask, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages to analyze")
	}

	// Join all messages into a single text
	discussionText := strings.Join(messages, "\n")

	// Create the full prompt
	fullPrompt := c.createTaskPrompt + "\n" + discussionText + "\n\nResponse in JSON format:"

	// Prepare request to YandexGPT
	request := YandexGPTRequest{
		ModelURI: c.modelURI,
		CompletionOptions: &CompletionOptions{
			Stream:      false,
			Temperature: 0.3, // Low temperature for more deterministic responses
			MaxTokens:   2000,
		},
		Messages: []YandexGPTMessage{
			{
				Role: "user",
				Text: fullPrompt,
			},
		},
	}

	// Make API call
	var response YandexGPTResponse
	err := c.httpClient.Post(ctx, "completion", request, &response)
	if err != nil {
		return nil, fmt.Errorf("YandexGPT API error: %w", err)
	}

	// Parse the response
	return c.parseGPTResponse(&response)
}

// EditTask edits an existing task based on user feedback using YandexGPT
func (c *AIClient) EditTask(ctx context.Context, task *AnalyzedTask, userFeedback string) (*AnalyzedTask, error) {
	if task == nil {
		return nil, fmt.Errorf("no task to edit")
	}

	if userFeedback == "" {
		return nil, fmt.Errorf("no feedback provided for editing")
	}

	// Format current task for the prompt
	taskJSON, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task: %w", err)
	}

	// Create the full prompt
	fullPrompt := fmt.Sprintf(c.editTaskPrompt, string(taskJSON), userFeedback)

	// Prepare request to YandexGPT
	request := YandexGPTRequest{
		ModelURI: c.modelURI,
		CompletionOptions: &CompletionOptions{
			Stream:      false,
			Temperature: 0.3,
			MaxTokens:   2000,
		},
		Messages: []YandexGPTMessage{
			{
				Role: "user",
				Text: fullPrompt,
			},
		},
	}

	// Make API call
	var response YandexGPTResponse
	err = c.httpClient.Post(ctx, "completion", request, &response)
	if err != nil {
		return nil, fmt.Errorf("YandexGPT API error: %w", err)
	}

	// Parse the response
	return c.parseGPTResponse(&response)
}

// parseGPTResponse parses YandexGPT response into AnalyzedTask
func (c *AIClient) parseGPTResponse(response *YandexGPTResponse) (*AnalyzedTask, error) {
	if response.Result == nil || len(response.Result.Alternatives) == 0 {
		return nil, fmt.Errorf("no alternatives in response")
	}

	// Get the text from the first alternative
	text := response.Result.Alternatives[0].Message.Text
	log.Printf("YandexGPT raw response: %s", text)

	// Try to extract JSON from the response (model might add extra text)
	jsonStart := strings.Index(text, "{")
	jsonEnd := strings.LastIndex(text, "}")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no valid JSON found in response")
	}

	jsonStr := text[jsonStart : jsonEnd+1]

	// Parse JSON
	var task AnalyzedTask
	if err := json.Unmarshal([]byte(jsonStr), &task); err != nil {
		log.Printf("Failed to parse JSON: %s, error: %v", jsonStr, err)
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Validate required fields
	if task.Title == "" {
		return nil, fmt.Errorf("task title is required")
	}

	// Ensure description is not empty
	if task.Description == "" {
		task.Description = "No description provided"
	}

	// Set priority text if not provided
	if task.PriorityText == "" {
		priorityMap := map[int]string{
			1: "Normal",
			2: "Medium",
			3: "High",
			4: "Urgent",
		}
		if text, ok := priorityMap[task.Priority]; ok {
			task.PriorityText = text
		} else {
			task.Priority = 1
			task.PriorityText = "Normal"
		}
	}

	// Validate priority range
	if task.Priority < 1 || task.Priority > 4 {
		task.Priority = 1
	}

	// Ensure labels is not nil
	if task.Labels == nil {
		task.Labels = []string{}
	}

	log.Printf("Parsed task: Title=%s, Priority=%d, Due=%s",
		task.Title, task.Priority, task.DueDate)

	return &task, nil
}