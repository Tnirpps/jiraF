package commands

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/user/telegram-bot/internal/todoist"
)

// CreateTaskCommand handles the /create_task command
type CreateTaskCommand struct {
	todoistClient todoist.Client
	dbManager     DBManager
}

// NewCreateTaskCommand creates a new create_task command handler
func NewCreateTaskCommand(todoistClient todoist.Client, dbManager DBManager) *CreateTaskCommand {
	return &CreateTaskCommand{
		todoistClient: todoistClient,
		dbManager:     dbManager,
	}
}

// Name returns the command name
func (c *CreateTaskCommand) Name() string {
	return "create_task"
}

// Description returns the command description
func (c *CreateTaskCommand) Description() string {
	return "Create task from discussion context"
}

// Execute handles the command execution
func (c *CreateTaskCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	ctx := context.Background()

	// Check if there's an active session
	hasActive, err := c.dbManager.HasActiveSession(ctx, message.Chat.ID)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Error checking session: %v", err))
		return &msg
	}

	if !hasActive {
		msg := tgbotapi.NewMessage(message.Chat.ID, "No active discussion. Start with /start_discussion first.")
		return &msg
	}

	// Get active session
	session, err := c.dbManager.GetActiveSession(ctx, message.Chat.ID)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Error getting session: %v", err))
		return &msg
	}

	// Get all messages from the session
	messages, err := c.dbManager.GetSessionMessages(ctx, session.ID)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Error getting messages: %v", err))
		return &msg
	}

	if len(messages) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "No messages in discussion to create task from.")
		return &msg
	}

	// Get project ID for this chat
	projectID, err := c.dbManager.GetTodoistProjectID(ctx, message.Chat.ID)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Error getting project: %v", err))
		return &msg
	}

	// Use first message as task title
	taskTitle := "Task from discussion"
	if len(messages) > 0 && messages[0].Text != "" {
		taskTitle = messages[0].Text
		if len(taskTitle) > 100 {
			taskTitle = taskTitle[:97] + "..."
		}
	}

	// Create the task in Todoist
	taskRequest := &todoist.TaskRequest{
		Content:   taskTitle,
		ProjectID: projectID,
	}

	createdTask, err := c.todoistClient.CreateTask(ctx, taskRequest)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Failed to create task: %v", err))
		return &msg
	}

	// Save created task to database
	err = c.dbManager.SaveCreatedTask(ctx, session.ID, createdTask.ID, createdTask.URL)
	if err != nil {
		fmt.Printf("Failed to save created task: %v\n", err)
	}

	// Close the session
	err = c.dbManager.CloseSession(ctx, message.Chat.ID)
	if err != nil {
		fmt.Printf("Failed to close session: %v\n", err)
	}

	// Success response
	responseText := fmt.Sprintf("✅ *Task Created from Discussion!*\n\n"+
		"*Title:* %s\n"+
		"[Open in Todoist](%s)",
		taskTitle, createdTask.URL)

	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	msg.ParseMode = "Markdown"
	return &msg
}