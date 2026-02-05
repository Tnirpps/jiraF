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
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	DueDate      string   `json:"due_date"`
	Priority     int      `json:"priority"`
	PriorityText string   `json:"priority_text"`
	Labels       []string `json:"labels,omitempty"`
}

// ProviderType тип AI провайдера
type ProviderType string

const (
	ProviderYandexGPT  ProviderType = "yandex"
	ProviderOpenRouter ProviderType = "openrouter"
)

// AIClient универсальный клиент для работы с разными AI провайдерами
type AIClient struct {
	provider         ProviderType
	httpClient       *httpclient.Client
	modelURI         string // для YandexGPT
	model            string // для OpenRouter
	createTaskPrompt string
	editTaskPrompt   string
}

// NewClient создает новый AI клиент в зависимости от конфигурации
func NewClient() (Client, error) {
	// Загружаем настройки AI
	aiSettings, err := LoadAiSettings("configs/ai_settings.yaml")
	if err != nil {
		log.Printf("Error loading AI settings: %v. Using default settings.", err)
		return nil, fmt.Errorf("failed to load AI settings: %w", err)
	}

	// Определяем провайдера
	provider := strings.ToLower(os.Getenv("AI_PROVIDER"))
	if provider == "" {
		provider = "openrouter" // по умолчанию OpenRouter
	}

	// Определяем какой провайдер использовать
	var aiClient *AIClient
	switch ProviderType(provider) {
	case ProviderYandexGPT:
		aiClient, err = newYandexGPTClient(&aiSettings)
		if err != nil {
			return nil, err
		}
	case ProviderOpenRouter:
		aiClient, err = newOpenRouterClient(&aiSettings)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", provider)
	}

	// Устанавливаем промпты
	aiClient.createTaskPrompt = aiSettings.CreateTaskPrompt
	aiClient.editTaskPrompt = aiSettings.EditTaskPrompt

	return aiClient, nil
}

// newYandexGPTClient создает клиент для YandexGPT
func newYandexGPTClient(aiSettings *AiSettings) (*AIClient, error) {
	// Загружаем конфигурацию
	configs, err := httpclient.LoadConfig("configs/api.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to load API configuration: %w", err)
	}

	// Получаем конфигурацию YandexGPT
	clientConfig, err := configs.GetClientConfig("yandex_gpt")
	if err != nil {
		return nil, fmt.Errorf("failed to get YandexGPT client configuration: %w", err)
	}

	// Создаем HTTP клиент
	client, err := clientConfig.CreateClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	folderID := os.Getenv("YANDEX_FOLDER_ID")
	if folderID == "" {
		return nil, fmt.Errorf("YANDEX_FOLDER_ID environment variable is required for YandexGPT")
	}

	return &AIClient{
		provider:   ProviderYandexGPT,
		httpClient: client,
		modelURI:   fmt.Sprintf(aiSettings.ModelURLTemplate, folderID),
	}, nil
}

// newOpenRouterClient создает клиент для OpenRouter
func newOpenRouterClient(aiSettings *AiSettings) (*AIClient, error) {
	// Загружаем конфигурацию
	configs, err := httpclient.LoadConfig("configs/api.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to load API configuration: %w", err)
	}

	// Получаем конфигурацию OpenRouter
	clientConfig, err := configs.GetClientConfig("openrouter")
	if err != nil {
		return nil, fmt.Errorf("failed to get OpenRouter client configuration: %w", err)
	}

	// Создаем HTTP клиент
	client, err := clientConfig.CreateClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Получаем модель из env (или используем gpt-4o-mini по умолчанию)
	model := os.Getenv("OPENROUTER_MODEL")
	if model == "" {
		model = "openai/gpt-4o-mini"
	}

	return &AIClient{
		provider:   ProviderOpenRouter,
		httpClient: client,
		model:      model,
	}, nil
}

// Структуры для YandexGPT
type YandexGPTRequest struct {
	ModelURI          string             `json:"modelUri"`
	CompletionOptions *CompletionOptions `json:"completionOptions"`
	Messages          []YandexGPTMessage `json:"messages"`
}

type YandexGPTMessage struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type CompletionOptions struct {
	Stream      bool    `json:"stream"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"maxTokens"`
}

type YandexGPTResponse struct {
	Result *GPTResult `json:"result"`
}

type GPTResult struct {
	Alternatives []Alternative `json:"alternatives"`
	Usage        Usage         `json:"usage"`
	ModelVersion string        `json:"modelVersion"`
}

type Alternative struct {
	Message MessageContent `json:"message"`
	Status  string         `json:"status"`
}

type MessageContent struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type CompletionTokensDetails struct {
	ReasoningTokens string `json:"reasoningTokens"`
}

type Usage struct {
	InputTextTokens         string                  `json:"inputTextTokens"`
	CompletionTokens        string                  `json:"completionTokens"`
	TotalTokens             string                  `json:"totalTokens"`
	CompletionTokensDetails CompletionTokensDetails `json:"completionTokensDetails"`
}

// Структуры для OpenRouter
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

// AnalyzeDiscussion анализирует сообщения используя выбранный AI провайдер
func (c *AIClient) AnalyzeDiscussion(ctx context.Context, messages []string) (*AnalyzedTask, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages to analyze")
	}

	switch c.provider {
	case ProviderYandexGPT:
		return c.analyzeWithYandexGPT(ctx, messages)
	case ProviderOpenRouter:
		return c.analyzeWithOpenRouter(ctx, messages)
	default:
		return nil, fmt.Errorf("unsupported provider for analysis: %v", c.provider)
	}
}

// analyzeWithYandexGPT анализирует с помощью YandexGPT
func (c *AIClient) analyzeWithYandexGPT(ctx context.Context, messages []string) (*AnalyzedTask, error) {
	discussionText := strings.Join(messages, "\n")
	fullPrompt := c.createTaskPrompt + "\n" + discussionText + "\n\nResponse in JSON format:"

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

	var response YandexGPTResponse
	err := c.httpClient.Post(ctx, "completion", request, &response)
	if err != nil {
		return nil, fmt.Errorf("YandexGPT API error: %w", err)
	}

	return c.parseYandexGPTResponse(&response)
}

// analyzeWithOpenRouter анализирует с помощью OpenRouter
func (c *AIClient) analyzeWithOpenRouter(ctx context.Context, messages []string) (*AnalyzedTask, error) {
	discussionText := strings.Join(messages, "\n")
	fullPrompt := c.createTaskPrompt + "\nДиалог для анализа:\n" + discussionText + "\n\nОтвет в JSON формате:"

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

// EditTask редактирует задачу используя выбранный AI провайдер
func (c *AIClient) EditTask(ctx context.Context, task *AnalyzedTask, userFeedback string) (*AnalyzedTask, error) {
	if task == nil {
		return nil, fmt.Errorf("no task to edit")
	}

	if userFeedback == "" {
		return nil, fmt.Errorf("no feedback provided for editing")
	}

	switch c.provider {
	case ProviderYandexGPT:
		return c.editWithYandexGPT(ctx, task, userFeedback)
	case ProviderOpenRouter:
		return c.editWithOpenRouter(ctx, task, userFeedback)
	default:
		return nil, fmt.Errorf("unsupported provider for editing: %v", c.provider)
	}
}

// editWithYandexGPT редактирует с помощью YandexGPT
func (c *AIClient) editWithYandexGPT(ctx context.Context, task *AnalyzedTask, userFeedback string) (*AnalyzedTask, error) {
	taskJSON, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task: %w", err)
	}

	fullPrompt := fmt.Sprintf(c.editTaskPrompt, string(taskJSON), userFeedback)
	log.Printf("[YandexGPT edit prompt]: %s", fullPrompt)

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

	var response YandexGPTResponse
	err = c.httpClient.Post(ctx, "completion", request, &response)
	if err != nil {
		return nil, fmt.Errorf("YandexGPT API error: %w", err)
	}

	return c.parseYandexGPTResponse(&response)
}

// editWithOpenRouter редактирует с помощью OpenRouter
func (c *AIClient) editWithOpenRouter(ctx context.Context, task *AnalyzedTask, userFeedback string) (*AnalyzedTask, error) {
	taskJSON, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task: %w", err)
	}

	fullPrompt := fmt.Sprintf(c.editTaskPrompt, string(taskJSON), userFeedback)
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

// parseYandexGPTResponse парсит ответ YandexGPT
func (c *AIClient) parseYandexGPTResponse(response *YandexGPTResponse) (*AnalyzedTask, error) {
	if response.Result == nil || len(response.Result.Alternatives) == 0 {
		return nil, fmt.Errorf("no alternatives in response")
	}

	text := response.Result.Alternatives[0].Message.Text
	log.Printf("YandexGPT raw response: %s", text)

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

	log.Printf("Parsed task: Title=%s, Priority=%d, Due=%s",
		task.Title, task.Priority, task.DueDate)

	return task
}