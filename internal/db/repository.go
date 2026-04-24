package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/user/telegram-bot/internal/taskfields"
	"github.com/user/telegram-bot/internal/tasklinks"
)

var ErrNoActiveSession = errors.New("no active session found")
var ErrSessionAlreadyExists = errors.New("active session already exists for this chat")
var ErrProjectIDNotSet = errors.New("todoist project ID not set for this chat")

type nullableTaskFields struct {
	TaskContext                sql.NullString
	WhatToDo                   sql.NullString
	ConstraintsAndDependencies sql.NullString
	ReadinessCriteria          sql.NullString
	WhatIsBroken               sql.NullString
	ReproductionSteps          sql.NullString
	ExpectedBehavior           sql.NullString
	ActualBehavior             sql.NullString
	Environment                sql.NullString
	ImpactAndRisks             sql.NullString
	SuspectedCause             sql.NullString
	FixScope                   sql.NullString
	VerificationCriteria       sql.NullString
	DesignOrDocsLinks          sql.NullString
	Prerequisites              sql.NullString
	ProblemToSolve             sql.NullString
	BriefSolution              sql.NullString
	Risks                      sql.NullString
	Approvers                  sql.NullString
	ProjectParticipants        sql.NullString
	AcceptanceCriteria         sql.NullString
	UsefulLinks                sql.NullString
}

func nullableString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func nullableTaskFieldsFrom(fields taskfields.TaskFields) nullableTaskFields {
	fields = fields.Clean()
	return nullableTaskFields{
		TaskContext:                nullableString(fields.TaskContext),
		WhatToDo:                   nullableString(fields.WhatToDo),
		ConstraintsAndDependencies: nullableString(fields.ConstraintsAndDependencies),
		ReadinessCriteria:          nullableString(fields.ReadinessCriteria),
		WhatIsBroken:               nullableString(fields.WhatIsBroken),
		ReproductionSteps:          nullableString(fields.ReproductionSteps),
		ExpectedBehavior:           nullableString(fields.ExpectedBehavior),
		ActualBehavior:             nullableString(fields.ActualBehavior),
		Environment:                nullableString(fields.Environment),
		ImpactAndRisks:             nullableString(fields.ImpactAndRisks),
		SuspectedCause:             nullableString(fields.SuspectedCause),
		FixScope:                   nullableString(fields.FixScope),
		VerificationCriteria:       nullableString(fields.VerificationCriteria),
		DesignOrDocsLinks:          nullableString(fields.DesignOrDocsLinks),
		Prerequisites:              nullableString(fields.Prerequisites),
		ProblemToSolve:             nullableString(fields.ProblemToSolve),
		BriefSolution:              nullableString(fields.BriefSolution),
		Risks:                      nullableString(fields.Risks),
		Approvers:                  nullableString(fields.Approvers),
		ProjectParticipants:        nullableString(fields.ProjectParticipants),
		AcceptanceCriteria:         nullableString(fields.AcceptanceCriteria),
		UsefulLinks:                nullableString(fields.UsefulLinks),
	}
}

func (f *nullableTaskFields) scanTargets() []any {
	return []any{
		&f.TaskContext,
		&f.WhatToDo,
		&f.ConstraintsAndDependencies,
		&f.ReadinessCriteria,
		&f.WhatIsBroken,
		&f.ReproductionSteps,
		&f.ExpectedBehavior,
		&f.ActualBehavior,
		&f.Environment,
		&f.ImpactAndRisks,
		&f.SuspectedCause,
		&f.FixScope,
		&f.VerificationCriteria,
		&f.DesignOrDocsLinks,
		&f.Prerequisites,
		&f.ProblemToSolve,
		&f.BriefSolution,
		&f.Risks,
		&f.Approvers,
		&f.ProjectParticipants,
		&f.AcceptanceCriteria,
		&f.UsefulLinks,
	}
}

func (f nullableTaskFields) values() []any {
	return []any{
		f.TaskContext,
		f.WhatToDo,
		f.ConstraintsAndDependencies,
		f.ReadinessCriteria,
		f.WhatIsBroken,
		f.ReproductionSteps,
		f.ExpectedBehavior,
		f.ActualBehavior,
		f.Environment,
		f.ImpactAndRisks,
		f.SuspectedCause,
		f.FixScope,
		f.VerificationCriteria,
		f.DesignOrDocsLinks,
		f.Prerequisites,
		f.ProblemToSolve,
		f.BriefSolution,
		f.Risks,
		f.Approvers,
		f.ProjectParticipants,
		f.AcceptanceCriteria,
		f.UsefulLinks,
	}
}

func (f nullableTaskFields) taskFields() taskfields.TaskFields {
	return taskfields.TaskFields{
		TaskContext:                f.TaskContext.String,
		WhatToDo:                   f.WhatToDo.String,
		ConstraintsAndDependencies: f.ConstraintsAndDependencies.String,
		ReadinessCriteria:          f.ReadinessCriteria.String,
		WhatIsBroken:               f.WhatIsBroken.String,
		ReproductionSteps:          f.ReproductionSteps.String,
		ExpectedBehavior:           f.ExpectedBehavior.String,
		ActualBehavior:             f.ActualBehavior.String,
		Environment:                f.Environment.String,
		ImpactAndRisks:             f.ImpactAndRisks.String,
		SuspectedCause:             f.SuspectedCause.String,
		FixScope:                   f.FixScope.String,
		VerificationCriteria:       f.VerificationCriteria.String,
		DesignOrDocsLinks:          f.DesignOrDocsLinks.String,
		Prerequisites:              f.Prerequisites.String,
		ProblemToSolve:             f.ProblemToSolve.String,
		BriefSolution:              f.BriefSolution.String,
		Risks:                      f.Risks.String,
		Approvers:                  f.Approvers.String,
		ProjectParticipants:        f.ProjectParticipants.String,
		AcceptanceCriteria:         f.AcceptanceCriteria.String,
		UsefulLinks:                f.UsefulLinks.String,
	}
}

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
func (m *Manager) SaveMessage(ctx context.Context, chatID int64, messageID int, userID int64, username, text string, links []tasklinks.TaskLink) error {
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
		INSERT INTO messages (chat_id, session_id, message_id, user_id, username, text, links)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
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
		tasklinks.TaskLinkSlice(links),
	)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	return nil
}

// GetSessionMessages gets all messages for a session
func (m *Manager) GetSessionMessages(ctx context.Context, sessionID int) ([]Message, error) {
	query := `
		SELECT id, chat_id, session_id, message_id, user_id, username, text, links, ts
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
			&msg.Links,
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
func (m *Manager) SaveDraftTask(
	ctx context.Context,
	sessionID int,
	title, description, dueISO string,
	priority int,
	taskType string,
	labels, missingDetails []string,
	selectedLinks []tasklinks.TaskLink,
	assigneeNote string,
	fields taskfields.TaskFields,
) error {
	query := `
		INSERT INTO draft_tasks (
			session_id, title, description, due_iso, priority, task_type, labels, missing_details, selected_links, assignee_note,
			task_context, what_to_do, constraints_and_dependencies, readiness_criteria,
			what_is_broken, reproduction_steps, expected_behavior, actual_behavior, environment, impact_and_risks, suspected_cause, fix_scope, verification_criteria,
			design_or_docs_links, prerequisites, problem_to_solve, brief_solution, risks, approvers, project_participants, acceptance_criteria, useful_links,
			updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23,
			$24, $25, $26, $27, $28, $29, $30, $31, $32,
			$33
		)
		ON CONFLICT (session_id) DO UPDATE
		SET title = $2, description = $3, due_iso = $4, priority = $5, task_type = $6,
		    labels = $7, missing_details = $8, selected_links = $9, assignee_note = $10,
		    task_context = $11, what_to_do = $12, constraints_and_dependencies = $13, readiness_criteria = $14,
		    what_is_broken = $15, reproduction_steps = $16, expected_behavior = $17, actual_behavior = $18, environment = $19,
		    impact_and_risks = $20, suspected_cause = $21, fix_scope = $22, verification_criteria = $23,
		    design_or_docs_links = $24, prerequisites = $25, problem_to_solve = $26, brief_solution = $27, risks = $28,
		    approvers = $29, project_participants = $30, acceptance_criteria = $31, useful_links = $32,
		    updated_at = $33
	`

	fieldValues := nullableTaskFieldsFrom(fields).values()
	args := []any{
		sessionID,
		nullableString(title),
		nullableString(description),
		nullableString(dueISO),
		sql.NullInt32{Int32: int32(priority), Valid: priority > 0},
		nullableString(taskType),
		StringSlice(labels),
		StringSlice(missingDetails),
		tasklinks.TaskLinkSlice(selectedLinks),
		nullableString(assigneeNote),
	}
	args = append(args, fieldValues...)
	args = append(args, time.Now())

	_, err := m.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to save draft task: %w", err)
	}

	return nil
}

func (m *Manager) GetDraftTask(ctx context.Context, sessionID int) (DraftTask, error) {
	const query = `
        SELECT session_id, title, description, due_iso, priority, task_type, labels, missing_details, selected_links, assignee_note,
               task_context, what_to_do, constraints_and_dependencies, readiness_criteria,
               what_is_broken, reproduction_steps, expected_behavior, actual_behavior, environment, impact_and_risks, suspected_cause, fix_scope, verification_criteria,
               design_or_docs_links, prerequisites, problem_to_solve, brief_solution, risks, approvers, project_participants, acceptance_criteria, useful_links,
               updated_at
        FROM draft_tasks
        WHERE session_id = $1
    `

	var t DraftTask
	var fields nullableTaskFields
	targets := []any{
		&t.SessionID,
		&t.Title,
		&t.Description,
		&t.DueISO,
		&t.Priority,
		&t.TaskType,
		&t.Labels,
		&t.MissingDetails,
		&t.SelectedLinks,
		&t.AssigneeNote,
	}
	targets = append(targets, fields.scanTargets()...)
	targets = append(targets, &t.UpdatedAt)

	err := m.db.QueryRowContext(ctx, query, sessionID).Scan(targets...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DraftTask{}, fmt.Errorf("draft task not found: %w", err)
		}
		return DraftTask{}, fmt.Errorf("failed to get draft task: %w", err)
	}
	t.Fields = fields.taskFields()

	return t, nil
}

// DeleteDraftTask removes the current draft task for a session.
func (m *Manager) DeleteDraftTask(ctx context.Context, sessionID int) error {
	const query = `
		DELETE FROM draft_tasks
		WHERE session_id = $1
	`

	if _, err := m.db.ExecContext(ctx, query, sessionID); err != nil {
		return fmt.Errorf("failed to delete draft task: %w", err)
	}

	return nil
}

// SaveCreatedTask saves a created Todoist task and a snapshot of the fields used to create it.
func (m *Manager) SaveCreatedTask(ctx context.Context, task DraftTask, todoistTaskID, url string) error {
	query := `
		INSERT INTO created_tasks (
			session_id, todoist_task_id, url, title, description, due_iso, priority, task_type, labels, selected_links, assignee_note,
			task_context, what_to_do, constraints_and_dependencies, readiness_criteria,
			what_is_broken, reproduction_steps, expected_behavior, actual_behavior, environment, impact_and_risks, suspected_cause, fix_scope, verification_criteria,
			design_or_docs_links, prerequisites, problem_to_solve, brief_solution, risks, approvers, project_participants, acceptance_criteria, useful_links
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24,
			$25, $26, $27, $28, $29, $30, $31, $32, $33
		)
	`
	args := []any{
		task.SessionID,
		todoistTaskID,
		url,
		task.Title,
		task.Description,
		task.DueISO,
		task.Priority,
		task.TaskType,
		task.Labels,
		task.SelectedLinks,
		task.AssigneeNote,
	}
	args = append(args, nullableTaskFieldsFrom(task.Fields).values()...)
	_, err := m.db.ExecContext(ctx, query, args...)
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
