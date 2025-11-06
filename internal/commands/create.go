package commands

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/user/telegram-bot/internal/todoist"
)

// CreateCommand handles the /create command to create a task in Todoist
type CreateCommand struct {
	todoistClient todoist.Client
}

// NewCreateCommand creates a new create command handler
func NewCreateCommand(todoistClient todoist.Client) *CreateCommand {
	return &CreateCommand{
		todoistClient: todoistClient,
	}
}

// Name returns the command name
func (c *CreateCommand) Name() string {
	return "create"
}

// Description returns the command description
func (c *CreateCommand) Description() string {
	return "Create a new task in Todoist (usage: /create Task name)"
}

// Execute handles the command execution
func (c *CreateCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	// Extract task title from the message
	args := message.CommandArguments()
	if args == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"⚠️ *Error:* Task title is required\n\n"+
				"Usage: `/create Task name`")
		msg.ParseMode = "Markdown"
		return &msg
	}

	// Create the task in Todoist
	taskTitle := strings.TrimSpace(args)
	task := &todoist.TaskRequest{
		Content: taskTitle,
	}

	// Call the Todoist API
	createdTask, err := c.todoistClient.CreateTask(context.Background(), task)
	if err != nil {
		// Handle API error
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("❌ *Failed to create task:* %v", err))
		msg.ParseMode = "Markdown"
		return &msg
	}

	// Success response
	responseText := fmt.Sprintf("✅ *Task created successfully!*\n\n"+
		"*Title:* %s\n"+
		"*ID:* %s\n"+
		"*Project:* %s",
		createdTask.Content, createdTask.ID, createdTask.ProjectID)

	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	msg.ParseMode = "Markdown"
	return &msg
}
