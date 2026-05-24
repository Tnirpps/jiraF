package commands

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const ReplyKindAssigneeMapUpload = "assignee_map_upload"

type SetAssigneeMapCommand struct {
	dbManager DBManager
}

func NewSetAssigneeMapCommand(dbManager DBManager) *SetAssigneeMapCommand {
	return &SetAssigneeMapCommand{dbManager: dbManager}
}

func (c *SetAssigneeMapCommand) Name() string {
	return "set_assignee_map"
}

func (c *SetAssigneeMapCommand) Description() string {
	return "загрузить YAML-маппинг Telegram исполнителей в Todoist"
}

func (c *SetAssigneeMapCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	projectID, err := c.dbManager.GetTodoistProjectID(context.Background(), message.Chat.ID)
	if err != nil || projectID == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Сначала выберите проект Todoist через /set_project, затем загрузите YAML-маппинг исполнителей.")
		return &msg
	}

	text := "Отправьте YAML-файл маппинга в ответ на это сообщение.\n\nПример:\n```yaml\nversion: 1\nassignees:\n  - todoist_email: \"alice@example.com\"\n    telegram_aliases: [\"@alice\", \"alice\", \"Алиса\"]\n```\n\nФайл заменит текущий маппинг для выбранного проекта."
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true}
	return &msg
}

func (c *SetAssigneeMapCommand) WaitingReply(message *tgbotapi.Message) (string, string, bool) {
	projectID, err := c.dbManager.GetTodoistProjectID(context.Background(), message.Chat.ID)
	if err != nil || projectID == "" {
		return "", "", false
	}
	return ReplyKindAssigneeMapUpload, fmt.Sprintf("%d:%s", message.Chat.ID, projectID), true
}
