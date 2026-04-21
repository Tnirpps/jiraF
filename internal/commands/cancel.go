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
	return "Завершить обсуждение без задачи"
}

func (c *CancelCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	ctx := context.Background()

	// Get the active session
	session, err := c.dbManager.GetActiveSession(ctx, message.Chat.ID)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Нет активного обсуждения.")
		return &msg
	}

	// Check if the user is the session owner
	senderID := int64(message.From.ID)
	if session.OwnerID != senderID {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Только автор обсуждения может завершить его.")
		return &msg
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "Завершить обсуждение без создания задачи?")
	msg.ReplyMarkup = buildCancelDiscussionKeyboard(session.ID)
	return &msg
}

func buildCancelDiscussionKeyboard(sessionID int) tgbotapi.InlineKeyboardMarkup {
	sessionIDStr := fmt.Sprintf("%d", sessionID)
	finishButton := tgbotapi.NewInlineKeyboardButtonData("🛑 Завершить", CallbackFinishDiscussion+CallbackDataSeparator+sessionIDStr)
	continueButton := tgbotapi.NewInlineKeyboardButtonData("↩️ Продолжить", CallbackKeepDiscussion+CallbackDataSeparator+sessionIDStr)

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(finishButton, continueButton),
	)
}
