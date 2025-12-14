package commands

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CancelCommand struct {
	dbManager DBManager
}

func NewCancelCommand(dbManager DBManager) *CancelCommand {
	return &CancelCommand{
		dbManager: dbManager,
	}
}

func (c *CancelCommand) Name() string {
	return "cancel"
}

func (c *CancelCommand) Description() string {
	return "Отменить текущее обсуждение"
}

func (c *CancelCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	ctx := context.Background()

	// Get the active session
	session, err := c.dbManager.GetActiveSession(ctx, message.Chat.ID)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Нет активного обсуждения для отмены.")
		return &msg
	}

	// Check if the user is the session owner
	senderID := int64(message.From.ID)
	if session.OwnerID != senderID {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Только автор текущего обсуждение может отменить его.")
		return &msg
	}

	// Proceed with cancellation
	err = c.dbManager.CloseSession(ctx, message.Chat.ID)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Error canceling discussion: %v", err))
		return &msg
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "Обсуждение отменено. Все собранные сообщения удалены.")
	return &msg
}
