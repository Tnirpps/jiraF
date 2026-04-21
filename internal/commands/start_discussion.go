package commands

import (
	"context"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/user/telegram-bot/internal/db"
	"github.com/user/telegram-bot/internal/todoist"
)

type StartDiscussionCommand struct {
	dbManager     DBManager
	todoistClient todoist.Client
}

func NewStartDiscussionCommand(dbManager DBManager, todoistClient todoist.Client) *StartDiscussionCommand {
	return &StartDiscussionCommand{
		dbManager:     dbManager,
		todoistClient: todoistClient,
	}
}

func (c *StartDiscussionCommand) Name() string {
	return "start_discussion"
}

func (c *StartDiscussionCommand) Description() string {
	return "Начать сбор сообщений для создания задачи"
}

func (c *StartDiscussionCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	ctx := context.Background()

	projectID, err := c.dbManager.GetTodoistProjectID(ctx, message.Chat.ID)
	if err != nil {
		if err == db.ErrProjectIDNotSet {
			return buildProjectSelectionMessage(ctx, c.todoistClient, message.Chat.ID, "Сначала выберите проект Todoist:")
		}
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Error getting project ID: %v", err))
		return &msg
	}

	sessionID, err := c.dbManager.StartSession(ctx, message.Chat.ID, int64(message.From.ID))
	if err != nil {
		if err == db.ErrSessionAlreadyExists {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Обсуждение уже идёт! Прежде, чем начать новое завершите текущее.")
			return &msg
		}
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Error starting discussion: %v", err))
		return &msg
	}

	log.Printf("Start for id: %s session: %d\n", projectID, sessionID)

	responseText := "Обсуждение началось.\nСообщения будут сохраняться, пока вы не создадите задачу (/create_task) или не завершите обсуждение (/cancel)."

	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	return &msg
}
