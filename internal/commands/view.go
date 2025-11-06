package commands

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/user/telegram-bot/internal/todoist"
)

// ViewCommand handles the /view command to show task details
type ViewCommand struct {
	todoistClient todoist.Client
}

// NewViewCommand creates a new view command handler
func NewViewCommand(todoistClient todoist.Client) *ViewCommand {
	return &ViewCommand{
		todoistClient: todoistClient,
	}
}

// Name returns the command name
func (c *ViewCommand) Name() string {
	return "view"
}

// Description returns the command description
func (c *ViewCommand) Description() string {
	return "View task details (usage: /view task_id)"
}

// Execute handles the command execution
func (c *ViewCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	// Extract task ID from the message
	taskID := strings.TrimSpace(message.CommandArguments())
	if taskID == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"⚠️ *Error:* Task ID is required\n\n"+
				"Usage: `/view task_id`")
		msg.ParseMode = "Markdown"
		return &msg
	}

	// Call the Todoist API to get the task
	task, err := c.todoistClient.GetTask(context.Background(), taskID)
	if err != nil {
		// Handle API error
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("❌ *Failed to get task:* %v", err))
		msg.ParseMode = "Markdown"
		return &msg
	}

	// Format detailed task view
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📝 *Task Details*\n\n"))
	sb.WriteString(fmt.Sprintf("*Title:* %s\n", task.Content))
	sb.WriteString(fmt.Sprintf("*ID:* `%s`\n", task.ID))

	// Show description if exists
	if task.Description != "" {
		sb.WriteString(fmt.Sprintf("*Description:* %s\n", task.Description))
	}

	// Show due date if exists
	if task.Due != nil {
		sb.WriteString(fmt.Sprintf("*Due:* %s", task.Due.Date))
		if task.Due.DateTime != "" {
			sb.WriteString(fmt.Sprintf(" at %s", task.Due.DateTime))
		}
		sb.WriteString("\n")

		if task.Due.IsRecurring {
			sb.WriteString("*Recurring:* Yes\n")
		}
	}

	// Show priority
	priorityNames := map[int]string{
		1: "Normal",
		2: "Medium",
		3: "High",
		4: "Urgent",
	}
	priorityName, ok := priorityNames[task.Priority]
	if !ok {
		priorityName = "Normal"
	}
	sb.WriteString(fmt.Sprintf("*Priority:* %s\n", priorityName))

	// Show project ID
	sb.WriteString(fmt.Sprintf("*Project:* %s\n", task.ProjectID))

	// Show labels if any
	if len(task.Labels) > 0 {
		sb.WriteString(fmt.Sprintf("*Labels:* %s\n", strings.Join(task.Labels, ", ")))
	}

	// Show URL to open in Todoist
	sb.WriteString(fmt.Sprintf("\n[Open in Todoist](%s)", task.URL))

	msg := tgbotapi.NewMessage(message.Chat.ID, sb.String())
	msg.ParseMode = "Markdown"
	return &msg
}
