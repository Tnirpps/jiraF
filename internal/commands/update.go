package commands

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/user/telegram-bot/internal/todoist"
)

// UpdateCommand handles the /update command to update task properties
type UpdateCommand struct {
	todoistClient todoist.Client
}

// NewUpdateCommand creates a new update command handler
func NewUpdateCommand(todoistClient todoist.Client) *UpdateCommand {
	return &UpdateCommand{
		todoistClient: todoistClient,
	}
}

// Name returns the command name
func (c *UpdateCommand) Name() string {
	return "update"
}

// Description returns the command description
func (c *UpdateCommand) Description() string {
	return "Update a task (usage: /update task_id field=value [field2=value2...])"
}

// Execute handles the command execution
func (c *UpdateCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	// Parse arguments: first arg is task ID, rest are field=value pairs
	args := strings.Fields(message.CommandArguments())
	if len(args) < 2 {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"⚠️ *Error:* Missing arguments\n\n"+
				"Usage: `/update task_id field=value [field2=value2...]`\n\n"+
				"Supported fields: content, description, due_string, priority")
		msg.ParseMode = "Markdown"
		return &msg
	}

	taskID := args[0]

	// First, check if the task exists
	_, err := c.todoistClient.GetTask(context.Background(), taskID)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("❌ *Failed to find task:* %v", err))
		msg.ParseMode = "Markdown"
		return &msg
	}

	// Create update request
	updateReq := &todoist.TaskRequest{}

	// Parse field=value pairs
	var updatedFields []string
	for _, arg := range args[1:] {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			continue // Skip invalid format
		}

		field := strings.ToLower(parts[0])
		value := parts[1]

		switch field {
		case "content", "title":
			updateReq.Content = value
			updatedFields = append(updatedFields, "content")
		case "description", "desc":
			updateReq.Description = value
			updatedFields = append(updatedFields, "description")
		case "due", "due_string":
			updateReq.DueString = value
			updatedFields = append(updatedFields, "due date")
		case "priority", "prio":
			// Priority is 1-4, with 4 being highest
			switch strings.ToLower(value) {
			case "1", "normal", "p1":
				updateReq.Priority = 1
			case "2", "medium", "p2":
				updateReq.Priority = 2
			case "3", "high", "p3":
				updateReq.Priority = 3
			case "4", "urgent", "p4":
				updateReq.Priority = 4
			default:
				updateReq.Priority = 1 // Default to normal priority
			}
			updatedFields = append(updatedFields, "priority")
		case "labels", "label":
			// Split comma-separated labels
			labels := strings.Split(value, ",")
			for i, label := range labels {
				labels[i] = strings.TrimSpace(label)
			}
			updateReq.Labels = labels
			updatedFields = append(updatedFields, "labels")
		}
	}

	if len(updatedFields) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"⚠️ *Error:* No valid fields to update\n\n"+
				"Supported fields: content, description, due_string, priority, labels")
		msg.ParseMode = "Markdown"
		return &msg
	}

	// Call the Todoist API to update the task
	updatedTask, err := c.todoistClient.UpdateTask(context.Background(), taskID, updateReq)
	if err != nil {
		// Handle API error
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("❌ *Failed to update task:* %v", err))
		msg.ParseMode = "Markdown"
		return &msg
	}

	// Success response
	// Escape special Markdown characters in task content
	content := updatedTask.Content
	content = strings.ReplaceAll(content, "[", "\\[")
	content = strings.ReplaceAll(content, "]", "\\]")
	content = strings.ReplaceAll(content, "*", "\\*")
	content = strings.ReplaceAll(content, "_", "\\_")
	content = strings.ReplaceAll(content, "`", "\\`")

	responseText := fmt.Sprintf("✅ *Task updated successfully!*\n\n"+
		"*Task:* %s\n"+
		"*Updated fields:* %s\n\n"+
		"View details with: /view %s",
		content, strings.Join(updatedFields, ", "), updatedTask.ID)

	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	msg.ParseMode = "Markdown"
	return &msg
}
