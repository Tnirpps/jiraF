package commands

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/user/telegram-bot/internal/todoist"
)

const defaultTimeout = 10 * time.Second

type SetProjectCommand struct {
	todoistClient todoist.Client
	dbManager     DBManager
}

func NewSetProjectCommand(todoistClient todoist.Client, dbManager DBManager) *SetProjectCommand {
	return &SetProjectCommand{
		todoistClient: todoistClient,
		dbManager:     dbManager,
	}
}

func (c *SetProjectCommand) Name() string {
	return "set_project"
}

func (c *SetProjectCommand) Description() string {
	return "Set Todoist project ID for this chat (usage: /set_project <id or URL>)"
}

func (c *SetProjectCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	ctx := context.Background()
	args := strings.TrimSpace(message.CommandArguments())

	if args == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Please provide a project ID or URL. Usage: /set_project <id or URL>")
		return &msg
	}

	projectID := args
	urlRegex := regexp.MustCompile(`todoist.com/app/projects/(\d+)`)
	if matches := urlRegex.FindStringSubmatch(args); len(matches) > 1 {
		projectID = matches[1]
	}

	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	projects, err := c.todoistClient.GetProjects(ctx)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Error fetching Todoist projects: %v", err))
		return &msg
	}

	validProject := false
	for _, project := range projects {
		if project.ID == projectID {
			validProject = true
			break
		}
	}

	if !validProject {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Invalid project ID. Please check and try again.")
		return &msg
	}

	err = c.dbManager.SetTodoistProjectID(ctx, message.Chat.ID, projectID)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Error saving project ID: %v", err))
		return &msg
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Project ID set to: %s", projectID))
	return &msg
}
