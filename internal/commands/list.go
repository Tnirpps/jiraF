package commands

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/user/telegram-bot/internal/todoist"
)

// ListCommand handles the /list command to list tasks or projects
type ListCommand struct {
	todoistClient todoist.Client
}

// NewListCommand creates a new list command handler
func NewListCommand(todoistClient todoist.Client) *ListCommand {
	return &ListCommand{
		todoistClient: todoistClient,
	}
}

// Name returns the command name
func (c *ListCommand) Name() string {
	return "list"
}

// Description returns the command description
func (c *ListCommand) Description() string {
	return "List your tasks or projects (usage: /list [tasks|projects] [project_id])"
}

// Execute handles the command execution
func (c *ListCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	// Parse arguments
	args := strings.Fields(message.CommandArguments())

	// Default to listing tasks
	listType := "tasks"
	var projectID string

	if len(args) > 0 {
		if args[0] == "tasks" || args[0] == "projects" {
			listType = args[0]
		} else {
			// If first arg is not a valid list type, assume it's a project ID
			projectID = args[0]
		}

		// If second arg exists and we're listing tasks, it's a project ID
		if len(args) > 1 && listType == "tasks" {
			projectID = args[1]
		}
	}

	// Handle based on list type
	switch listType {
	case "projects":
		return c.listProjects(message)
	case "tasks":
		return c.listTasks(message, projectID)
	default:
		// Should never reach here
		msg := tgbotapi.NewMessage(message.Chat.ID, "Unknown list type. Use 'tasks' or 'projects'.")
		msg.ParseMode = "Markdown"
		return &msg
	}
}

// listProjects lists all projects
func (c *ListCommand) listProjects(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	projects, err := c.todoistClient.GetProjects(context.Background())
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("❌ *Failed to get projects:* %v", err))
		msg.ParseMode = "Markdown"
		return &msg
	}

	if len(projects) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "No projects found.")
		msg.ParseMode = "Markdown"
		return &msg
	}

	// Format projects list
	var sb strings.Builder
	sb.WriteString("📋 *Your Projects:*\n\n")

	for _, project := range projects {
		sb.WriteString(fmt.Sprintf("• *%s*\n", project.Name))
		sb.WriteString(fmt.Sprintf("  ID: `%s`\n", project.ID))
		sb.WriteString(fmt.Sprintf("  Tasks: Use `/list tasks %s`\n\n", project.ID))
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, sb.String())
	msg.ParseMode = "Markdown"
	return &msg
}

// listTasks lists tasks, optionally filtered by project
func (c *ListCommand) listTasks(message *tgbotapi.Message, projectID string) *tgbotapi.MessageConfig {
	tasks, err := c.todoistClient.GetTasks(context.Background(), projectID)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			fmt.Sprintf("❌ *Failed to get tasks:* %v", err))
		msg.ParseMode = "Markdown"
		return &msg
	}

	// If project ID was specified, get project name
	var projectName string
	if projectID != "" {
		projects, err := c.todoistClient.GetProjects(context.Background())
		if err == nil {
			for _, p := range projects {
				if p.ID == projectID {
					projectName = p.Name
					break
				}
			}
		}
	}

	if len(tasks) == 0 {
		var messageText string
		if projectName != "" {
			messageText = fmt.Sprintf("No tasks found in project \"%s\".", projectName)
		} else if projectID != "" {
			messageText = fmt.Sprintf("No tasks found in project with ID %s.", projectID)
		} else {
			messageText = "No tasks found."
		}

		msg := tgbotapi.NewMessage(message.Chat.ID, messageText)
		msg.ParseMode = "Markdown"
		return &msg
	}

	// Format tasks list
	var sb strings.Builder
	if projectName != "" {
		sb.WriteString(fmt.Sprintf("📝 *Tasks in %s:*\n\n", projectName))
	} else if projectID != "" {
		sb.WriteString(fmt.Sprintf("📝 *Tasks in project %s:*\n\n", projectID))
	} else {
		sb.WriteString("📝 *Your Tasks:*\n\n")
	}

	for _, task := range tasks {
		// Mark completed tasks
		if task.IsCompleted {
			sb.WriteString(fmt.Sprintf("✅ ~%s~\n", task.Content))
		} else {
			sb.WriteString(fmt.Sprintf("⬜ *%s*\n", task.Content))
		}

		sb.WriteString(fmt.Sprintf("  ID: `%s`\n", task.ID))

		// Show due date if exists
		if task.Due != nil {
			sb.WriteString(fmt.Sprintf("  Due: %s\n", task.Due.Date))
		}

		sb.WriteString(fmt.Sprintf("  Project: %s\n\n", task.ProjectID))
	}

	// Add help text for other commands
	sb.WriteString("\n*Use these commands with task IDs:*\n")
	sb.WriteString("/complete [task_id] - Mark a task as complete\n")
	sb.WriteString("/view [task_id] - View task details\n")

	msg := tgbotapi.NewMessage(message.Chat.ID, sb.String())
	msg.ParseMode = "Markdown"
	return &msg
}
