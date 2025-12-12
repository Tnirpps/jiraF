package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrNoActiveSession = errors.New("no active session found")
var ErrSessionAlreadyExists = errors.New("active session already exists for this chat")
var ErrProjectIDNotSet = errors.New("todoist project ID not set for this chat")

// ChatRepository handles database operations for chats
func (m *Manager) EnsureChatExists(ctx context.Context, chatID int64) error {
	query := `
		INSERT INTO chats (id)
		VALUES ($1)
		ON CONFLICT (id) DO NOTHING
	`
	_, err := m.db.ExecContext(ctx, query, chatID)
	if err != nil {
		return fmt.Errorf("failed to ensure chat exists: %w", err)
	}
	return nil
}

// SetTodoistProjectID sets the Todoist project ID for a chat
func (m *Manager) SetTodoistProjectID(ctx context.Context, chatID int64, projectID string) error {
	if err := m.EnsureChatExists(ctx, chatID); err != nil {
		return err
	}

	query := `
		INSERT INTO chat_settings (chat_id, todoist_project_id, updated_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (chat_id) DO UPDATE
		SET todoist_project_id = $2, updated_at = $3
	`
	_, err := m.db.ExecContext(ctx, query, chatID, projectID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set todoist project id: %w", err)
	}
	return nil
}

// GetTodoistProjectID gets the Todoist project ID for a chat
func (m *Manager) GetTodoistProjectID(ctx context.Context, chatID int64) (string, error) {
	query := `
		SELECT todoist_project_id
		FROM chat_settings
		WHERE chat_id = $1
	`
	var projectID sql.NullString
	err := m.db.QueryRowContext(ctx, query, chatID).Scan(&projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrProjectIDNotSet
		}
		return "", fmt.Errorf("failed to get todoist project id: %w", err)
	}

	if !projectID.Valid || projectID.String == "" {
		return "", ErrProjectIDNotSet
	}

	return projectID.String, nil
}

// StartSession creates a new session for a chat with the specified owner
func (m *Manager) StartSession(ctx context.Context, chatID int64, ownerID int64) (int, error) {
	// Check if there's an active session
	active, err := m.HasActiveSession(ctx, chatID)
	if err != nil {
		return 0, err
	}

	if active {
		return 0, ErrSessionAlreadyExists
	}

	// Create a new session with owner
	query := `
		INSERT INTO sessions (chat_id, owner_id, status)
		VALUES ($1, $2, 'open')
		RETURNING id
	`
	var sessionID int
	err = m.db.QueryRowContext(ctx, query, chatID, ownerID).Scan(&sessionID)
	if err != nil {
		return 0, fmt.Errorf("failed to start session: %w", err)
	}

	return sessionID, nil
}

// HasActiveSession checks if a chat has an active session
func (m *Manager) HasActiveSession(ctx context.Context, chatID int64) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM sessions
			WHERE chat_id = $1 AND status = 'open'
		)
	`
	var exists bool
	err := m.db.QueryRowContext(ctx, query, chatID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check active session: %w", err)
	}

	return exists, nil
}

// GetActiveSession returns the active session for a chat
func (m *Manager) GetActiveSession(ctx context.Context, chatID int64) (*Session, error) {
	query := `
		SELECT id, chat_id, owner_id, status, started_at, closed_at
		FROM sessions
		WHERE chat_id = $1 AND status = 'open'
		ORDER BY started_at DESC
		LIMIT 1
	`
	var session Session
	err := m.db.QueryRowContext(ctx, query, chatID).Scan(
		&session.ID,
		&session.ChatID,
		&session.OwnerID,
		&session.Status,
		&session.StartedAt,
		&session.ClosedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoActiveSession
		}
		return nil, fmt.Errorf("failed to get active session: %w", err)
	}

	return &session, nil
}

// IsSessionOwner checks if the given user is the owner of the session
func (m *Manager) IsSessionOwner(ctx context.Context, sessionID int, userID int64) (bool, error) {
	query := `
		SELECT owner_id
		FROM sessions
		WHERE id = $1
	`
	var ownerID sql.NullInt64
	err := m.db.QueryRowContext(ctx, query, sessionID).Scan(&ownerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("session not found")
		}
		return false, fmt.Errorf("failed to get session owner: %w", err)
	}

	// If there's no owner set, return false
	if !ownerID.Valid {
		return false, nil
	}

	return ownerID.Int64 == userID, nil
}

// CloseSession closes an active session
func (m *Manager) CloseSession(ctx context.Context, chatID int64) error {
	session, err := m.GetActiveSession(ctx, chatID)
	if err != nil {
		return err
	}

	query := `
		UPDATE sessions
		SET status = 'closed', closed_at = $1
		WHERE id = $2
	`
	_, err = m.db.ExecContext(ctx, query, time.Now(), session.ID)
	if err != nil {
		return fmt.Errorf("failed to close session: %w", err)
	}

	return nil
}

// SaveMessage saves a message from a chat
func (m *Manager) SaveMessage(ctx context.Context, chatID int64, messageID int, userID int64, username, text string) error {
	if err := m.EnsureChatExists(ctx, chatID); err != nil {
		return err
	}

	// Get active session if exists
	var sessionID sql.NullInt32
	session, err := m.GetActiveSession(ctx, chatID)
	if err == nil {
		sessionID.Int32 = int32(session.ID)
		sessionID.Valid = true
	}

	query := `
		INSERT INTO messages (chat_id, session_id, message_id, user_id, username, text)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	var nullUserID sql.NullInt64
	if userID != 0 {
		nullUserID.Int64 = userID
		nullUserID.Valid = true
	}

	var nullUsername sql.NullString
	if username != "" {
		nullUsername.String = username
		nullUsername.Valid = true
	}

	_, err = m.db.ExecContext(
		ctx,
		query,
		chatID,
		sessionID,
		messageID,
		nullUserID,
		nullUsername,
		text,
	)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	return nil
}

// GetSessionMessages gets all messages for a session
func (m *Manager) GetSessionMessages(ctx context.Context, sessionID int) ([]Message, error) {
	query := `
		SELECT id, chat_id, session_id, message_id, user_id, username, text, ts
		FROM messages
		WHERE session_id = $1
		ORDER BY ts ASC
	`
	rows, err := m.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		err := rows.Scan(
			&msg.ID,
			&msg.ChatID,
			&msg.SessionID,
			&msg.MessageID,
			&msg.UserID,
			&msg.Username,
			&msg.Text,
			&msg.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating message rows: %w", err)
	}

	return messages, nil
}

// SaveDraftTask saves a draft task for a session
func (m *Manager) SaveDraftTask(ctx context.Context, sessionID int, title, description, dueISO string, priority int, assigneeNote string) error {
	query := `
		INSERT INTO draft_tasks (session_id, title, description, due_iso, priority, assignee_note, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (session_id) DO UPDATE
		SET title = $2, description = $3, due_iso = $4, priority = $5, assignee_note = $6, updated_at = $7
	`

	_, err := m.db.ExecContext(
		ctx,
		query,
		sessionID,
		sql.NullString{String: title, Valid: title != ""},
		sql.NullString{String: description, Valid: description != ""},
		sql.NullString{String: dueISO, Valid: dueISO != ""},
		sql.NullInt32{Int32: int32(priority), Valid: priority > 0},
		sql.NullString{String: assigneeNote, Valid: assigneeNote != ""},
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to save draft task: %w", err)
	}

	return nil
}

func (m *Manager) GetDraftTask(ctx context.Context, sessionID int) (DraftTask, error) {
	const query = `
        SELECT session_id, title, description, due_iso, priority, assignee_note, updated_at
        FROM draft_tasks
        WHERE session_id = $1
    `

	var t DraftTask
	err := m.db.QueryRowContext(ctx, query, sessionID).Scan(
		&t.SessionID,
		&t.Title,
		&t.Description,
		&t.DueISO,
		&t.Priority,
		&t.AssigneeNote,
		&t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DraftTask{}, fmt.Errorf("draft task not found: %w", err)
		}
		return DraftTask{}, fmt.Errorf("failed to get draft task: %w", err)
	}

	return t, nil
}

// SaveCreatedTask saves a created Todoist task
func (m *Manager) SaveCreatedTask(ctx context.Context, sessionID int, todoistTaskID, url string) error {
	query := `
		INSERT INTO created_tasks (session_id, todoist_task_id, url)
		VALUES ($1, $2, $3)
	`
	_, err := m.db.ExecContext(ctx, query, sessionID, todoistTaskID, url)
	if err != nil {
		return fmt.Errorf("failed to save created task: %w", err)
	}

	return nil
}

// SaveAuditEdit saves an audit edit record
func (m *Manager) SaveAuditEdit(ctx context.Context, sessionID int, instructionText string, diffJSON []byte) error {
	query := `
		INSERT INTO audit_edits (session_id, instruction_text, diff_json)
		VALUES ($1, $2, $3)
	`
	_, err := m.db.ExecContext(ctx, query, sessionID, instructionText, diffJSON)
	if err != nil {
		return fmt.Errorf("failed to save audit edit: %w", err)
	}

	return nil
}
