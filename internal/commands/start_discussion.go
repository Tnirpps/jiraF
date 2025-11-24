package commands

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/user/telegram-bot/internal/db"
)

type StartDiscussionCommand struct {
	dbManager DBManager
}

func NewStartDiscussionCommand(dbManager DBManager) *StartDiscussionCommand {
	return &StartDiscussionCommand{
		dbManager: dbManager,
	}
}

func (c *StartDiscussionCommand) Name() string {
	return "start_discussion"
}

func (c *StartDiscussionCommand) Description() string {
	return "Start a new discussion session"
}

func (c *StartDiscussionCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	ctx := context.Background()

	projectID, err := c.dbManager.GetTodoistProjectID(ctx, message.Chat.ID)
	if err != nil {
		if err == db.ErrProjectIDNotSet {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Please set a project ID first using /set_project command.")
			return &msg
		}
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Error getting project ID: %v", err))
		return &msg
	}

	sessionID, err := c.dbManager.StartSession(ctx, message.Chat.ID)
	if err != nil {
		if err == db.ErrSessionAlreadyExists {
			msg := tgbotapi.NewMessage(message.Chat.ID, "A discussion is already in progress. Please finish or /cancel it first.")
			return &msg
		}
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Error starting discussion: %v", err))
		return &msg
	}

	responseText := fmt.Sprintf(
		"Discussion started for project ID: %s\nSession ID: %d\n\nAll messages will be collected until you type /create_task or /cancel.",
		projectID,
		sessionID,
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	return &msg
}
