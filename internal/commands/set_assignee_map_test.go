package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/user/telegram-bot/internal/db"
)

func TestSetAssigneeMapCommand_Execute(t *testing.T) {
	mockDB := new(MockDBManager)
	cmd := NewSetAssigneeMapCommand(mockDB)

	t.Run("project required", func(t *testing.T) {
		mockDB.On("GetTodoistProjectID", mock.Anything, int64(100)).Return("", db.ErrProjectIDNotSet).Once()

		msg := CreateCommandMessage(100, "/set_assignee_map")
		response := cmd.Execute(msg)

		assert.Contains(t, response.Text, "Сначала выберите проект Todoist")
	})

	t.Run("returns force reply", func(t *testing.T) {
		mockDB.On("GetTodoistProjectID", mock.Anything, int64(200)).Return("project-1", nil).Twice()

		msg := CreateCommandMessage(200, "/set_assignee_map")
		response := cmd.Execute(msg)
		assert.Contains(t, response.Text, "YAML-файл")

		kind, value, ok := cmd.WaitingReply(msg)
		assert.True(t, ok)
		assert.Equal(t, ReplyKindAssigneeMapUpload, kind)
		assert.Equal(t, "200:project-1", value)
	})
}
