package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
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
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	DueDate        string   `json:"due_date"`
	Priority       int      `json:"priority"`
	PriorityText   string   `json:"priority_text"`
	Labels         []string `json:"labels,omitempty"`
	TaskType       string   `json:"task_type,omitempty"`
	MissingDetails []string `json:"missing_details,omitempty"`
}

// AIClient клиент для работы с OpenRouter AI
type AIClient struct {
	httpClient          *httpclient.Client
	model               string
	createTaskPrompt    string
	editTaskPrompt      string
	taskTemplates       []TaskTemplate
	taskTemplatesPrompt string
}

// NewClient создает новый AI клиент (OpenRouter)
// Принимает конфигурацию как аргумент для упрощения тестирования
func NewClient(config *httpclient.ClientConfig) (Client, error) {
	// Загружаем настройки AI
	aiSettings, err := LoadAiSettings("configs/ai_settings.yaml")
	if err != nil {
		log.Printf("Error loading AI settings: %v. Using default settings.", err)
		return nil, fmt.Errorf("failed to load AI settings: %w", err)
	}

	// Создаем HTTP клиент из переданной конфигурации
	client, err := config.CreateClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Получаем модель из env (или используем gpt-4o-mini по умолчанию)
	model := os.Getenv("OPENROUTER_MODEL")
	if model == "" {
		model = "openai/gpt-4o-mini"
	}

	taskTemplates, err := LoadTaskTemplates(aiSettings.TaskTemplatesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load task templates: %w", err)
	}

	return &AIClient{
		httpClient:          client,
		model:               model,
		createTaskPrompt:    aiSettings.CreateTaskPrompt,
		editTaskPrompt:      aiSettings.EditTaskPrompt,
		taskTemplates:       taskTemplates,
		taskTemplatesPrompt: BuildTaskTemplatesPromptSection(taskTemplates),
	}, nil
}

// OpenRouter запрос
type OpenRouterRequest struct {
	Model    string              `json:"model"`
	Messages []OpenRouterMessage `json:"messages"`
	Stream   bool                `json:"stream"`
	Options  *OpenRouterOptions  `json:"options,omitempty"`
}

type OpenRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenRouterOptions struct {
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
	TopP        float64 `json:"top_p,omitempty"`
}

type OpenRouterResponse struct {
	Choices []OpenRouterChoice `json:"choices"`
	Usage   OpenRouterUsage    `json:"usage"`
	Model   string             `json:"model"`
}

type OpenRouterChoice struct {
	Message      OpenRouterMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

type OpenRouterUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// AnalyzeDiscussion анализирует сообщения используя OpenRouter AI
func (c *AIClient) AnalyzeDiscussion(ctx context.Context, messages []string) (*AnalyzedTask, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages to analyze")
	}

	discussionText := strings.Join(messages, "\n")
	fullPrompt := c.createTaskPrompt +
		"\n\n" + c.taskTemplatesPrompt +
		"\n\nДиалог для анализа:\n" + discussionText +
		"\n\nОтвет в JSON формате:"

	request := OpenRouterRequest{
		Model: c.model,
		Messages: []OpenRouterMessage{
			{
				Role:    "user",
				Content: fullPrompt,
			},
		},
		Stream: false,
		Options: &OpenRouterOptions{
			Temperature: 0.3,
			MaxTokens:   2000,
			TopP:        0.9,
		},
	}

	var response OpenRouterResponse
	err := c.httpClient.Post(ctx, "chat/completions", request, &response)
	if err != nil {
		return nil, fmt.Errorf("OpenRouter API error: %w", err)
	}

	return c.parseOpenRouterResponse(&response)
}

// EditTask редактирует задачу используя OpenRouter AI
func (c *AIClient) EditTask(ctx context.Context, task *AnalyzedTask, userFeedback string) (*AnalyzedTask, error) {
	if task == nil {
		return nil, fmt.Errorf("no task to edit")
	}

	if userFeedback == "" {
		return nil, fmt.Errorf("no feedback provided for editing")
	}

	taskJSON, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task: %w", err)
	}

	fullPrompt := fmt.Sprintf(c.editTaskPrompt, c.taskTemplatesPrompt, string(taskJSON), userFeedback)
	log.Printf("[OpenRouter edit prompt]: %s", fullPrompt)

	request := OpenRouterRequest{
		Model: c.model,
		Messages: []OpenRouterMessage{
			{
				Role:    "user",
				Content: fullPrompt,
			},
		},
		Stream: false,
		Options: &OpenRouterOptions{
			Temperature: 0.3,
			MaxTokens:   2000,
			TopP:        0.9,
		},
	}

	var response OpenRouterResponse
	err = c.httpClient.Post(ctx, "chat/completions", request, &response)
	if err != nil {
		return nil, fmt.Errorf("OpenRouter API error: %w", err)
	}

	return c.parseOpenRouterResponse(&response)
}

// parseOpenRouterResponse парсит ответ OpenRouter
func (c *AIClient) parseOpenRouterResponse(response *OpenRouterResponse) (*AnalyzedTask, error) {
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	text := response.Choices[0].Message.Content
	log.Printf("OpenRouter raw response: %s", text)

	jsonStart := strings.Index(text, "{")
	jsonEnd := strings.LastIndex(text, "}")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no valid JSON found in response")
	}

	jsonStr := text[jsonStart : jsonEnd+1]

	var task AnalyzedTask
	if err := json.Unmarshal([]byte(jsonStr), &task); err != nil {
		log.Printf("Failed to parse JSON: %s, error: %v", jsonStr, err)
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return c.validateAndCompleteTask(&task), nil
}

// validateAndCompleteTask валидирует и заполняет значения по умолчанию
func (c *AIClient) validateAndCompleteTask(task *AnalyzedTask) *AnalyzedTask {
	if task.Title == "" {
		task.Title = "Без названия"
	}

	if task.Description == "" {
		task.Description = "Описание не предоставлено"
	}

	if task.PriorityText == "" {
		priorityMap := map[int]string{
			1: "Низкий",
			2: "Средний",
			3: "Высокий",
			4: "Срочный",
		}
		if text, ok := priorityMap[task.Priority]; ok {
			task.PriorityText = text
		} else {
			task.Priority = 1
			task.PriorityText = "Низкий"
		}
	}

	if task.Priority < 1 || task.Priority > 4 {
		task.Priority = 1
	}

	if task.Labels == nil {
		task.Labels = []string{}
	}

	task.TaskType = normalizeTaskType(task.TaskType)

	if task.MissingDetails == nil {
		task.MissingDetails = []string{}
	}

	log.Printf("Parsed task: Title=%s, Priority=%d, Due=%s",
		task.Title, task.Priority, task.DueDate)

	return task
}

func normalizeTaskType(taskType string) string {
	taskType = strings.ToLower(strings.TrimSpace(taskType))
	taskType = strings.ReplaceAll(taskType, " ", "_")
	taskType = strings.ReplaceAll(taskType, "-", "_")

	switch taskType {
	case "bug", "баг":
		return "bug"
	case "epic", "эпик":
		return "epic"
	case "":
		return "task"
	default:
		return taskType
	}
}
