package commands

import (
	"context"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/user/telegram-bot/internal/ai"
	"github.com/user/telegram-bot/internal/todoist"
)

type AnalyzeCommand struct {
	todoistClient todoist.Client
	dbManager     DBManager
	aiClient      ai.Client
}

func NewAnalyzeCommand(todoistClient todoist.Client, dbManager DBManager, aiClient ai.Client) *AnalyzeCommand {
	return &AnalyzeCommand{
		todoistClient: todoistClient,
		dbManager:     dbManager,
		aiClient:      aiClient,
	}
}

func (c *AnalyzeCommand) Name() string {
	return "analyze"
}

func (c *AnalyzeCommand) Description() string {
	return "AI-analyze discussion and create smart task"
}

func (c *AnalyzeCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	ctx := context.Background()

	// Check if there's an active session
	hasActive, err := c.dbManager.HasActiveSession(ctx, message.Chat.ID)
	if err != nil {
		log.Printf("Error checking session: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Unable to check discussion status")
		return &msg
	}

	if !hasActive {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ No active discussion to analyze. Start with /start_discussion first.")
		return &msg
	}

	// Get active session
	session, err := c.dbManager.GetActiveSession(ctx, message.Chat.ID)
	if err != nil {
		log.Printf("Error getting session: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Unable to access discussion data")
		return &msg
	}

	// Get all messages from the session
	messages, err := c.dbManager.GetSessionMessages(ctx, session.ID)
	if err != nil {
		log.Printf("Error getting messages: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Unable to retrieve discussion messages")
		return &msg
	}

	if len(messages) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ No messages in discussion to analyze.")
		return &msg
	}

	// Extract text from messages
	var messageTexts []string
	for _, msg := range messages {
		if msg.Text != "" {
			messageTexts = append(messageTexts, msg.Text)
		}
	}

	// Analyze with AI
	analyzedTask, err := c.aiClient.AnalyzeDiscussion(ctx, messageTexts)
	if err != nil {
		log.Printf("AI analysis failed: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ AI analysis failed. Please try again.")
		return &msg
	}

	// Get project ID
	projectID, err := c.dbManager.GetTodoistProjectID(ctx, message.Chat.ID)
	if err != nil {
		log.Printf("Error getting project: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Unable to access project settings")
		return &msg
	}

	// Create task in Todoist
	taskRequest := &todoist.TaskRequest{
		Content:     analyzedTask.Title,
		Description: analyzedTask.Description,
		ProjectID:   projectID,
		DueString:   analyzedTask.DueDate,
		Priority:    analyzedTask.Priority,
		Labels:      analyzedTask.Labels,
	}

	createdTask, err := c.todoistClient.CreateTask(ctx, taskRequest)
	if err != nil {
		log.Printf("Failed to create task: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Failed to create task in Todoist")
		return &msg
	}

	// Save to database
	err = c.dbManager.SaveCreatedTask(ctx, session.ID, createdTask.ID, createdTask.URL)
	if err != nil {
		log.Printf("Failed to save created task: %v", err)
	}

	// Close session
	err = c.dbManager.CloseSession(ctx, message.Chat.ID)
	if err != nil {
		log.Printf("Failed to close session: %v", err)
	}

	// Success response
	responseText := fmt.Sprintf("🤖 *AI Task Created!*\n\n"+
		"*Title:* %s\n"+
		"*Description:* %s\n"+
		"*Due:* %s\n"+
		"*Priority:* %s\n"+
		"*Labels:* %s\n\n"+
		"[Open in Todoist](%s)",
		analyzedTask.Title,
		analyzedTask.Description,
		analyzedTask.DueDate,
		analyzedTask.PriorityText,
		strings.Join(analyzedTask.Labels, ", "),
		createdTask.URL)

	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	msg.ParseMode = "Markdown"
	return &msg
}
