package commands

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/user/telegram-bot/internal/todoist"
)

// CompleteCommand handles the /complete command to mark a task as complete
type CompleteCommand struct {
	todoistClient todoist.Client
}

// NewCompleteCommand creates a new complete command handler
func NewCompleteCommand(todoistClient todoist.Client) *CompleteCommand {
	return &CompleteCommand{
		todoistClient: todoistClient,
	}
}

// Name returns the command name
func (c *CompleteCommand) Name() string {
	return "complete"
}

// Description returns the command description
func (c *CompleteCommand) Description() string {
	return "Mark a task as complete (usage: /complete task_id)"
}

// Execute handles the command execution
func (c *CompleteCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	// Extract task ID from the message
	taskID := strings.TrimSpace(message.CommandArguments())
	if taskID == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"⚠️ *Error:* Task ID is required\n\n"+
				"Usage: `/complete task_id`")
		msg.ParseMode = "Markdown"
		return &msg
	}

	// First, get the task to show the user what they're completing
	task, err := c.todoistClient.GetTask(context.Background(), taskID)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("❌ *Failed to find task:* %v", err))
		msg.ParseMode = "Markdown"
		return &msg
	}

	// Call the Todoist API to complete the task
	err = c.todoistClient.CompleteTask(context.Background(), taskID)
	if err != nil {
		// Handle API error
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("❌ *Failed to complete task:* %v", err))
		msg.ParseMode = "Markdown"
		return &msg
	}

	// Success response
	responseText := fmt.Sprintf("✅ *Task completed successfully!*\n\n"+
		"*Task:* %s", task.Content)

	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	msg.ParseMode = "Markdown"
	return &msg
}
