package db

import (
	"database/sql"
	"time"
)

type Chat struct {
	ID        int64     `db:"id"`
	CreatedAt time.Time `db:"created_at"`
}

type ChatSettings struct {
	ChatID           int64     `db:"chat_id"`
	TodoistProjectID string    `db:"todoist_project_id"`
	UpdatedAt        time.Time `db:"updated_at"`
}

type Session struct {
	ID        int          `db:"id"`
	ChatID    int64        `db:"chat_id"`
	Status    string       `db:"status"`
	StartedAt time.Time    `db:"started_at"`
	ClosedAt  sql.NullTime `db:"closed_at"`
}

type Message struct {
	ID        int            `db:"id"`
	ChatID    int64          `db:"chat_id"`
	SessionID sql.NullInt32  `db:"session_id"`
	MessageID int            `db:"message_id"`
	UserID    sql.NullInt64  `db:"user_id"`
	Username  sql.NullString `db:"username"`
	Text      string         `db:"text"`
	Timestamp time.Time      `db:"ts"`
}

type DraftTask struct {
	SessionID    int            `db:"session_id"`
	Title        sql.NullString `db:"title"`
	Description  sql.NullString `db:"description"`
	DueISO       sql.NullString `db:"due_iso"`
	Priority     sql.NullInt32  `db:"priority"`
	AssigneeNote sql.NullString `db:"assignee_note"`
	UpdatedAt    time.Time      `db:"updated_at"`
}

type CreatedTask struct {
	ID            int       `db:"id"`
	SessionID     int       `db:"session_id"`
	TodoistTaskID string    `db:"todoist_task_id"`
	URL           string    `db:"url"`
	CreatedAt     time.Time `db:"created_at"`
}

type AuditEdit struct {
	ID              int       `db:"id"`
	SessionID       int       `db:"session_id"`
	InstructionText string    `db:"instruction_text"`
	DiffJSON        []byte    `db:"diff_json"`
	CreatedAt       time.Time `db:"created_at"`
}
