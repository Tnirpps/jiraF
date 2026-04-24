package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/user/telegram-bot/internal/httpclient"
	"github.com/user/telegram-bot/internal/taskfields"
	"github.com/user/telegram-bot/internal/tasklinks"
)

// Client defines the interface for interacting with AI models
type Client interface {
	AnalyzeLinks(ctx context.Context, messages []string, candidates []tasklinks.LinkCandidate) ([]tasklinks.TaskLink, error)
	AnalyzeDiscussion(ctx context.Context, messages []string, selectedLinks []tasklinks.TaskLink) (*AnalyzedTask, error)
	EditTask(ctx context.Context, task *AnalyzedTask, userFeedback string) (*AnalyzedTask, error)
}

// AnalyzedTask represents the structured task from AI analysis
type AnalyzedTask struct {
	Title          string               `json:"title"`
	Description    string               `json:"description"`
	DueDate        string               `json:"due_date"`
	Priority       int                  `json:"priority"`
	PriorityText   string               `json:"priority_text,omitempty"`
	AssigneeNote   string               `json:"assignee_note,omitempty"`
	Labels         []string             `json:"labels,omitempty"`
	TaskType       string               `json:"task_type,omitempty"`
	MissingDetails []string             `json:"-"`
	SelectedLinks  []tasklinks.TaskLink `json:"selected_links,omitempty"`
	taskfields.TaskFields
}

func (t *AnalyzedTask) UnmarshalJSON(data []byte) error {
	type analyzedTaskAlias struct {
		Title         string               `json:"title"`
		Description   string               `json:"description"`
		DueDate       string               `json:"due_date"`
		Priority      any                  `json:"priority"`
		PriorityText  string               `json:"priority_text,omitempty"`
		AssigneeNote  string               `json:"assignee_note,omitempty"`
		Labels        []string             `json:"labels,omitempty"`
		TaskType      string               `json:"task_type,omitempty"`
		SelectedLinks []tasklinks.TaskLink `json:"selected_links,omitempty"`
		taskfields.TaskFields
	}

	var raw analyzedTaskAlias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	parsedPriority, err := parsePriorityValue(raw.Priority)
	if err != nil {
		return fmt.Errorf("parse priority: %w", err)
	}

	t.Title = raw.Title
	t.Description = raw.Description
	t.DueDate = raw.DueDate
	t.Priority = parsedPriority
	t.PriorityText = raw.PriorityText
	t.AssigneeNote = raw.AssigneeNote
	t.Labels = raw.Labels
	t.TaskType = raw.TaskType
	t.SelectedLinks = raw.SelectedLinks
	t.TaskFields = raw.TaskFields

	return nil
}

func parsePriorityValue(value any) (int, error) {
	switch v := value.(type) {
	case nil:
		return 0, nil
	case float64:
		return int(v), nil
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, nil
		}

		switch trimmed {
		case "1", "2", "3", "4":
			return int(trimmed[0] - '0'), nil
		}

		switch strings.ToLower(trimmed) {
		case "low", "низкий", "normal", "обычный":
			return 1, nil
		case "medium", "mid", "средний":
			return 2, nil
		case "high", "высокий":
			return 3, nil
		case "urgent", "critical", "срочный", "критичный":
			return 4, nil
		default:
			return 0, fmt.Errorf("unsupported priority value %q", v)
		}
	default:
		return 0, fmt.Errorf("unsupported priority type %T", value)
	}
}

func isKnownTaskField(key string) bool {
	switch key {
	case "title", "description", "due_date", "priority", "priority_text",
		"assignee_note", "labels", "task_type", "selected_links":
		return true
	default:
		return taskfields.IsKnownKey(key)
	}
}

// AIClient клиент для работы с OpenRouter AI
type AIClient struct {
	httpClient          *httpclient.Client
	model               string
	createTaskPrompt    string
	editTaskPrompt      string
	analyzeLinksPrompt  string
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
		analyzeLinksPrompt:  aiSettings.AnalyzeLinksPrompt,
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

func (c *AIClient) AnalyzeLinks(ctx context.Context, messages []string, candidates []tasklinks.LinkCandidate) ([]tasklinks.TaskLink, error) {
	if len(candidates) == 0 {
		return []tasklinks.TaskLink{}, nil
	}

	requestPayload, err := json.MarshalIndent(struct {
		Messages   []string                  `json:"messages"`
		Candidates []tasklinks.LinkCandidate `json:"candidates"`
	}{
		Messages:   messages,
		Candidates: candidates,
	}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal link candidates: %w", err)
	}

	fullPrompt := c.analyzeLinksPrompt + "\n\nInput:\n" + string(requestPayload)

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
			Temperature: 0.2,
			MaxTokens:   1200,
			TopP:        0.9,
		},
	}

	var response OpenRouterResponse
	err = c.httpClient.Post(ctx, "chat/completions", request, &response)
	if err != nil {
		return nil, fmt.Errorf("OpenRouter API error: %w", err)
	}

	return c.parseLinkAnalysisResponse(&response, candidates)
}

// AnalyzeDiscussion анализирует сообщения используя OpenRouter AI
func (c *AIClient) AnalyzeDiscussion(ctx context.Context, messages []string, selectedLinks []tasklinks.TaskLink) (*AnalyzedTask, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages to analyze")
	}

	discussionText := strings.Join(messages, "\n")
	selectedLinksJSON, err := json.MarshalIndent(selectedLinks, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal selected links: %w", err)
	}
	fullPrompt := c.createTaskPrompt +
		"\n\n" + c.taskTemplatesPrompt +
		"\n\nSelected materials. Use these as task materials, but do not decide link usefulness again:\n" + string(selectedLinksJSON) +
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
	err = c.httpClient.Post(ctx, "chat/completions", request, &response)
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

func (c *AIClient) parseLinkAnalysisResponse(response *OpenRouterResponse, candidates []tasklinks.LinkCandidate) ([]tasklinks.TaskLink, error) {
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	text := response.Choices[0].Message.Content
	log.Printf("OpenRouter raw link response: %s", text)

	jsonStart := strings.Index(text, "{")
	jsonEnd := strings.LastIndex(text, "}")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no valid JSON found in link response")
	}

	var payload struct {
		Links []tasklinks.TaskLink `json:"links"`
	}
	if err := json.Unmarshal([]byte(text[jsonStart:jsonEnd+1]), &payload); err != nil {
		return nil, fmt.Errorf("failed to parse link response: %w", err)
	}

	return tasklinks.NormalizeSelectedLinks(candidates, payload.Links), nil
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
	task.TaskFields = task.TaskFields.Clean()

	if task.SelectedLinks == nil {
		task.SelectedLinks = []tasklinks.TaskLink{}
	} else {
		candidates := make([]tasklinks.LinkCandidate, 0, len(task.SelectedLinks))
		for _, link := range task.SelectedLinks {
			candidates = append(candidates, tasklinks.LinkCandidate{URL: link.URL})
		}
		task.SelectedLinks = tasklinks.NormalizeSelectedLinks(candidates, task.SelectedLinks)
	}
	task.MissingDetails = c.missingDetailsForTask(task)

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

func (c *AIClient) missingDetailsForTask(task *AnalyzedTask) []string {
	if task == nil {
		return []string{}
	}

	fields := c.fieldsForType(task.TaskType)
	if len(fields) == 0 {
		return []string{}
	}

	result := make([]string, 0, len(fields))
	for _, field := range fields {
		if isFieldFilled(task, field.Key) {
			continue
		}
		label := taskfields.LowerLabelForKey(field.Key)
		if label == "" {
			label = strings.ToLower(strings.TrimSpace(field.Label))
		}
		if label != "" {
			result = append(result, label)
		}
	}
	return result
}

func isFieldFilled(task *AnalyzedTask, key string) bool {
	switch strings.TrimSpace(key) {
	case taskfields.DueDate:
		return strings.TrimSpace(task.DueDate) != ""
	case taskfields.SelectedLinks:
		return len(task.SelectedLinks) > 0
	case taskfields.DesignOrDocsLinks, taskfields.UsefulLinks:
		return task.TaskFields.Value(key) != "" || len(task.SelectedLinks) > 0
	default:
		return task.TaskFields.Value(key) != ""
	}
}

func (c *AIClient) fieldsForType(taskType string) []taskfields.FieldDefinition {
	taskType = normalizeTaskType(taskType)
	for _, template := range c.taskTemplates {
		if template.Type == taskType {
			return template.Fields
		}
	}

	return nil
}
