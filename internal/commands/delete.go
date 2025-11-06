package commands

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/user/telegram-bot/internal/todoist"
)

// DeleteCommand handles the /delete command to delete a task
type DeleteCommand struct {
	todoistClient todoist.Client
}

// NewDeleteCommand creates a new delete command handler
func NewDeleteCommand(todoistClient todoist.Client) *DeleteCommand {
	return &DeleteCommand{
		todoistClient: todoistClient,
	}
}

// Name returns the command name
func (c *DeleteCommand) Name() string {
	return "delete"
}

// Description returns the command description
func (c *DeleteCommand) Description() string {
	return "Delete a task permanently (usage: /delete task_id)"
}

// Execute handles the command execution
func (c *DeleteCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	// Extract task ID from the message
	taskID := strings.TrimSpace(message.CommandArguments())
	if taskID == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"⚠️ *Error:* Task ID is required\n\n"+
				"Usage: `/delete task_id`")
		msg.ParseMode = "Markdown"
		return &msg
	}

	// First, get the task to show the user what they're deleting
	task, err := c.todoistClient.GetTask(context.Background(), taskID)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("❌ *Failed to find task:* %v", err))
		msg.ParseMode = "Markdown"
		return &msg
	}

	// Confirm details before deletion
	taskInfo := fmt.Sprintf("*Task:* %s\n*ID:* %s", task.Content, task.ID)

	// Show due date if exists
	if task.Due != nil {
		taskInfo += fmt.Sprintf("\n*Due:* %s", task.Due.Date)
	}

	// Add confirmation instructions
	confirmationText := fmt.Sprintf("🗑️ *Confirm Delete*\n\n%s\n\n"+
		"To confirm deletion, reply with `/delete_confirm %s`\n"+
		"To cancel, simply ignore this message.", taskInfo, task.ID)

	msg := tgbotapi.NewMessage(message.Chat.ID, confirmationText)
	msg.ParseMode = "Markdown"
	return &msg
}

// DeleteConfirmCommand handles the /delete_confirm command to confirm task deletion
type DeleteConfirmCommand struct {
	todoistClient todoist.Client
}

// NewDeleteConfirmCommand creates a new delete confirm command handler
func NewDeleteConfirmCommand(todoistClient todoist.Client) *DeleteConfirmCommand {
	return &DeleteConfirmCommand{
		todoistClient: todoistClient,
	}
}

// Name returns the command name
func (c *DeleteConfirmCommand) Name() string {
	return "delete_confirm"
}

// Description returns the command description
func (c *DeleteConfirmCommand) Description() string {
	return "Confirm task deletion (usage: /delete_confirm task_id)"
}

// Execute handles the command execution
func (c *DeleteConfirmCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	// Extract task ID from the message
	taskID := strings.TrimSpace(message.CommandArguments())
	if taskID == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"⚠️ *Error:* Task ID is required\n\n"+
				"Usage: `/delete_confirm task_id`")
		msg.ParseMode = "Markdown"
		return &msg
	}

	// Call the Todoist API to delete the task
	err := c.todoistClient.DeleteTask(context.Background(), taskID)
	if err != nil {
		// Handle API error
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("❌ *Failed to delete task:* %v", err))
		msg.ParseMode = "Markdown"
		return &msg
	}

	// Success response
	responseText := fmt.Sprintf("✅ *Task deleted successfully!*\n\n"+
		"Task with ID `%s` has been permanently deleted.", taskID)

	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	msg.ParseMode = "Markdown"
	return &msg
}
