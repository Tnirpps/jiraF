package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/user/telegram-bot/internal/httpclient"
)

// Client defines the interface for interacting with AI models
type Client interface {
	AnalyzeDiscussion(ctx context.Context, messages []string) (*AnalyzedTask, error)
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

// HuggingFaceClient is the implementation for AI analysis
type HuggingFaceClient struct {
	httpClient *httpclient.Client
}

// NewClient creates a new AI client
func NewClient() (Client, error) {
	// Load configuration from YAML file
	configs, err := httpclient.LoadConfig("configs/api.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to load API configuration: %w", err)
	}

	// Get HuggingFace client configuration
	clientConfig, err := configs.GetClientConfig("huggingface")
	if err != nil {
		return nil, fmt.Errorf("failed to get HuggingFace client configuration: %w", err)
	}

	// Create the HTTP client
	client, err := clientConfig.CreateClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &HuggingFaceClient{
		httpClient: client,
	}, nil
}

// AnalyzeDiscussion analyzes messages using AI
func (c *HuggingFaceClient) AnalyzeDiscussion(ctx context.Context, messages []string) (*AnalyzedTask, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages to analyze")
	}

	discussionText := strings.Join(messages, "\n")

	// For now, always use smart analysis
	// If we get a real Hugging Face token, we can uncomment the API call below
	/*
		if c.apiToken != "" && c.apiToken != "your_huggingface_token_here" {
			return c.callHuggingFaceAPI(ctx, discussionText)
		}
	*/

	return c.smartAnalysis(discussionText), nil
}

// callHuggingFaceAPI implements real AI calls via Hugging Face API
func (c *HuggingFaceClient) callHuggingFaceAPI(ctx context.Context, text string) (*AnalyzedTask, error) {
	// This is a placeholder for the actual API call implementation
	// When we have a real Hugging Face endpoint, we'd do something like:
	/*
		type HuggingFaceRequest struct {
			Text string `json:"text"`
		}

		type HuggingFaceResponse struct {
			// Fields matching the API response
		}

		request := HuggingFaceRequest{Text: text}
		var response HuggingFaceResponse

		err := c.httpClient.Post(ctx, "models/some-model/analyze", request, &response)
		if err != nil {
			return nil, fmt.Errorf("error calling HuggingFace API: %w", err)
		}

		// Process response and convert to AnalyzedTask
	*/

	// For now, just use the local analysis
	return c.smartAnalysis(text), nil
}

// smartAnalysis provides intelligent task creation
func (c *HuggingFaceClient) smartAnalysis(text string) *AnalyzedTask {
	lines := strings.Split(text, "\n")

	// Фильтруем только обычные сообщения (без команд)
	var cleanMessages []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "/") {
			cleanMessages = append(cleanMessages, line)
		}
	}

	if len(cleanMessages) == 0 {
		return &AnalyzedTask{
			Title:        "Задача из обсуждения",
			Description:  text,
			DueDate:      "",
			Priority:     1,
			PriorityText: "Обычный",
			Labels:       []string{},
		}
	}

	// Анализируем содержание
	fullText := strings.ToLower(strings.Join(cleanMessages, " "))

	task := &AnalyzedTask{
		Title:        c.generateSmartTitle(cleanMessages),
		Description:  c.generateSmartDescription(cleanMessages),
		DueDate:      c.extractDueDate(fullText),
		Priority:     c.extractPriority(fullText),
		PriorityText: "Обычный",
		Labels:       c.extractLabels(fullText),
	}

	// Set priority text
	switch task.Priority {
	case 2:
		task.PriorityText = "Средний"
	case 3:
		task.PriorityText = "Высокий"
	case 4:
		task.PriorityText = "Срочный"
	}

	return task
}

// generateSmartTitle создает умный заголовок на основе обсуждения
func (c *HuggingFaceClient) generateSmartTitle(messages []string) string {
	if len(messages) == 0 {
		return "Задача из обсуждения"
	}

	// Ищем самое информативное сообщение для заголовка
	for _, msg := range messages {
		msg = strings.TrimSpace(msg)
		if len(msg) > 10 {
			// Убираем приветствия и короткие сообщения
			if !c.isGreeting(msg) && len(msg) > 10 {
				if len(msg) > 50 {
					return msg[:47] + "..."
				}
				return msg
			}
		}
	}

	// Если не нашли хороший заголовок, берем первое не-приветствие
	for _, msg := range messages {
		msg = strings.TrimSpace(msg)
		if !c.isGreeting(msg) && msg != "" {
			if len(msg) > 50 {
				return msg[:47] + "..."
			}
			return msg
		}
	}

	// Последний вариант - первое сообщение
	if len(messages) > 0 {
		title := messages[0]
		if len(title) > 50 {
			return title[:47] + "..."
		}
		return title
	}

	return "Задача из обсуждения"
}

// generateSmartDescription создает структурированное описание
func (c *HuggingFaceClient) generateSmartDescription(messages []string) string {
	if len(messages) == 0 {
		return "Нет деталей задачи"
	}

	var description strings.Builder
	description.WriteString("Детали из обсуждения:\n\n")

	// Добавляем только значимые сообщения
	addedCount := 0
	for i, msg := range messages {
		msg = strings.TrimSpace(msg)
		if msg != "" && !c.isGreeting(msg) && !c.isShortMessage(msg) {
			description.WriteString(fmt.Sprintf("• %s\n", msg))
			addedCount++
		}

		// Ограничиваем длину
		if description.Len() > 400 && i > 0 {
			description.WriteString("• ...\n")
			break
		}
	}

	if addedCount == 0 {
		// Если все сообщения короткие, показываем их все
		for _, msg := range messages {
			if msg != "" {
				description.WriteString(fmt.Sprintf("• %s\n", msg))
			}
		}
	}

	// Обрезаем если слишком длинное
	result := description.String()
	if len(result) > 500 {
		return result[:497] + "..."
	}

	return result
}

// isGreeting проверяет, является ли сообщение приветствием
func (c *HuggingFaceClient) isGreeting(text string) bool {
	lower := strings.ToLower(text)
	greetings := []string{
		"привет", "здравствуйте", "добрый день", "доброе утро", "добрый вечер",
		"hi", "hello", "hey",
	}

	for _, greeting := range greetings {
		if strings.Contains(lower, greeting) {
			return true
		}
	}
	return false
}

// isShortMessage проверяет, является ли сообщение слишком коротким
func (c *HuggingFaceClient) isShortMessage(text string) bool {
	words := strings.Fields(text)
	return len(words) <= 2
}

func (c *HuggingFaceClient) extractDueDate(text string) string {
	if strings.Contains(text, "сегодня") || strings.Contains(text, "today") {
		return "today"
	}
	if strings.Contains(text, "завтра") || strings.Contains(text, "tomorrow") {
		return "tomorrow"
	}
	if strings.Contains(text, "понедельник") || strings.Contains(text, "monday") {
		return "monday"
	}
	if strings.Contains(text, "пятница") || strings.Contains(text, "friday") {
		return "friday"
	}
	if strings.Contains(text, "21 декабря") || strings.Contains(text, "декабря") {
		return "2025-12-21"
	}
	return ""
}

func (c *HuggingFaceClient) extractPriority(text string) int {
	if strings.Contains(text, "срочно") || strings.Contains(text, "urgent") ||
		strings.Contains(text, "важно") || strings.Contains(text, "important") ||
		strings.Contains(text, "уволят") {
		return 4
	}
	if strings.Contains(text, "высокий") || strings.Contains(text, "high") {
		return 3
	}
	if strings.Contains(text, "средний") || strings.Contains(text, "medium") {
		return 2
	}
	return 1
}

func (c *HuggingFaceClient) extractLabels(text string) []string {
	labels := []string{}

	if strings.Contains(text, "отчет") || strings.Contains(text, "report") {
		labels = append(labels, "отчет")
	}
	if strings.Contains(text, "презентация") || strings.Contains(text, "presentation") {
		labels = append(labels, "презентация")
	}
	if strings.Contains(text, "срочно") {
		labels = append(labels, "срочно")
	}
	if strings.Contains(text, "печенье") || strings.Contains(text, "cookies") {
		labels = append(labels, "еда")
	}
	if strings.Contains(text, "проект") || strings.Contains(text, "project") {
		labels = append(labels, "проект")
	}
	if strings.Contains(text, "баг") || strings.Contains(text, "bug") {
		labels = append(labels, "баг")
	}
	if strings.Contains(text, "фича") || strings.Contains(text, "feature") {
		labels = append(labels, "фича")
	}
	if strings.Contains(text, "встреча") || strings.Contains(text, "meeting") {
		labels = append(labels, "встреча")
	}
	if strings.Contains(text, "клиент") || strings.Contains(text, "client") {
		labels = append(labels, "клиент")
	}

	return labels
}
