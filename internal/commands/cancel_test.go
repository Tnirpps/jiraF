package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/user/telegram-bot/internal/db"
)

func TestCancelCommand_Execute_Success(t *testing.T) {
	chatID := int64(123456789)

	mockDBManager := new(MockDBManager)
	mockDBManager.On("GetActiveSession", mock.Anything, chatID).Return(&db.Session{
		ID:      1,
		ChatID:  chatID,
		OwnerID: chatID,
		Status:  "open",
	}, nil)

	cmd := NewCancelCommand(mockDBManager)
	message := CreateCommandMessage(chatID, "/cancel")

	response := cmd.Execute(message)

	assert.Contains(t, response.Text, "Завершить обсуждение без создания задачи")
	assert.NotNil(t, response.ReplyMarkup)
	mockDBManager.AssertExpectations(t)
}

func TestCancelCommand_Execute_NotOwner(t *testing.T) {
	chatID := int64(123456789)

	mockDBManager := new(MockDBManager)
	mockDBManager.On("GetActiveSession", mock.Anything, chatID).Return(&db.Session{
		ID:      1,
		ChatID:  chatID,
		OwnerID: 999999,
		Status:  "open",
	}, nil)

	cmd := NewCancelCommand(mockDBManager)
	message := CreateCommandMessage(chatID, "/cancel")

	response := cmd.Execute(message)

	assert.Contains(t, response.Text, "Только автор обсуждения может завершить его")
	mockDBManager.AssertExpectations(t)
}
