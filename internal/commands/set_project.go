package commands

import (
	"context"
	"fmt"
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
	return "Выбрать или сменить проект Todoist"
}

func (c *SetProjectCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	return buildProjectSelectionMessage(context.Background(), c.todoistClient, message.Chat.ID, "Выберите проект Todoist:")
}

func buildProjectSelectionMessage(ctx context.Context, todoistClient todoist.Client, chatID int64, intro string) *tgbotapi.MessageConfig {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	projects, err := todoistClient.GetProjects(ctx)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Не удалось загрузить проекты Todoist: %v", err))
		return &msg
	}

	if len(projects) == 0 {
		msg := tgbotapi.NewMessage(chatID, "В Todoist не найдено ни одного проекта.")
		return &msg
	}

	msg := tgbotapi.NewMessage(chatID, intro)
	msg.ReplyMarkup = buildProjectSelectionKeyboard(projects)
	return &msg
}

func buildProjectSelectionKeyboard(projects []todoist.Project) tgbotapi.InlineKeyboardMarkup {
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(projects))

	for _, project := range projects {
		button := tgbotapi.NewInlineKeyboardButtonData(project.Name, CallbackSelectProject+CallbackDataSeparator+project.ID)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(button))
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
