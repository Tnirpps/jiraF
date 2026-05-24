package commands

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/user/telegram-bot/internal/todoist"
)

func TestSetProjectCommand_Execute_ShowsProjects(t *testing.T) {
	mockTodoistClient := new(MockTodoistClient)
	mockDBManager := new(MockDBManager)

	cmd := NewSetProjectCommand(mockTodoistClient, mockDBManager)

	chatID := int64(123456789)
	projects := []todoist.Project{
		{ID: "12345", Name: "Backend"},
		{ID: "67890", Name: "Frontend"},
	}

	mockTodoistClient.On("GetProjects", mock.Anything).Return(projects, nil)

	message := CreateCommandMessage(chatID, "/set_project")
	response := cmd.Execute(message)

	assert.Contains(t, response.Text, "Выберите проект Todoist")
	markup, ok := response.ReplyMarkup.(tgbotapi.InlineKeyboardMarkup)
	assert.True(t, ok)
	assert.Len(t, markup.InlineKeyboard, 2)
	assert.Equal(t, "Backend", markup.InlineKeyboard[0][0].Text)
	if assert.NotNil(t, markup.InlineKeyboard[0][0].CallbackData) {
		assert.Equal(t, "select_project:12345", *markup.InlineKeyboard[0][0].CallbackData)
	}
	assert.Equal(t, "Frontend", markup.InlineKeyboard[1][0].Text)
	if assert.NotNil(t, markup.InlineKeyboard[1][0].CallbackData) {
		assert.Equal(t, "select_project:67890", *markup.InlineKeyboard[1][0].CallbackData)
	}

	mockTodoistClient.AssertExpectations(t)
	mockDBManager.AssertNotCalled(t, "SetTodoistProjectID", mock.Anything, mock.Anything, mock.Anything)
}
