package commands

import (
	"context"

	"github.com/user/telegram-bot/internal/db"
)

type DBManager interface {
	// Methods needed for the start_discussion command
	GetTodoistProjectID(ctx context.Context, chatID int64) (string, error)
	HasActiveSession(ctx context.Context, chatID int64) (bool, error)
	StartSession(ctx context.Context, chatID int64) (int, error)

	// Methods needed for the set_project command
	SetTodoistProjectID(ctx context.Context, chatID int64, projectID string) error

	// Methods needed for other commands
	GetActiveSession(ctx context.Context, chatID int64) (*db.Session, error)
	CloseSession(ctx context.Context, chatID int64) error
	SaveMessage(ctx context.Context, chatID int64, messageID int, userID int64, username, text string) error
	GetSessionMessages(ctx context.Context, sessionID int) ([]db.Message, error)
	
	// ДОБАВЬТЕ этот метод для команды analyze
	SaveCreatedTask(ctx context.Context, sessionID int, todoistTaskID, url string) error
}