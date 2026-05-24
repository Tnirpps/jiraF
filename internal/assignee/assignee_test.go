package assignee

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/user/telegram-bot/internal/ai"
	"github.com/user/telegram-bot/internal/db"
	"github.com/user/telegram-bot/internal/tasklinks"
	"github.com/user/telegram-bot/internal/todoist"
)

type aiStub struct {
	selection *ai.AssigneeSelection
	err       error
}

func (s aiStub) AnalyzeLinks(ctx context.Context, messages []string, candidates []tasklinks.LinkCandidate) ([]tasklinks.TaskLink, error) {
	return nil, nil
}

func (s aiStub) AnalyzeDiscussion(ctx context.Context, messages []string, selectedLinks []tasklinks.TaskLink) (*ai.AnalyzedTask, error) {
	return nil, nil
}

func (s aiStub) EditTask(ctx context.Context, task *ai.AnalyzedTask, userFeedback string) (*ai.AnalyzedTask, error) {
	return task, nil
}

func (s aiStub) AnalyzeAssignee(ctx context.Context, messages []string, assigneeNote string, candidates []ai.AssigneeCandidate) (*ai.AssigneeSelection, error) {
	return s.selection, s.err
}

func TestParseAndValidateYAML(t *testing.T) {
	collaborators := []todoist.Collaborator{
		{ID: "u1", Name: "Alice", Email: "alice@example.com"},
		{ID: "u2", Name: "Bob", Email: "bob@example.com"},
	}

	t.Run("valid file", func(t *testing.T) {
		raw := []byte(`
version: 1
assignees:
  - todoist_email: "alice@example.com"
    telegram_aliases: ["@alice", "Alice Doe"]
`)

		mappings, summary, err := ParseAndValidateYAML(10, "project-1", raw, collaborators)
		if err != nil {
			t.Fatalf("ParseAndValidateYAML() error = %v", err)
		}
		if len(mappings) != 2 {
			t.Fatalf("expected 2 mappings, got %d", len(mappings))
		}
		if summary.AliasesCount != 2 {
			t.Fatalf("expected 2 aliases, got %d", summary.AliasesCount)
		}
		if mappings[0].TodoistUserID != "u1" {
			t.Fatalf("unexpected mapping: %#v", mappings[0])
		}
	})

	t.Run("unknown collaborator", func(t *testing.T) {
		raw := []byte(`
version: 1
assignees:
  - todoist_email: "nobody@example.com"
    telegram_aliases: ["@ghost"]
`)
		if _, _, err := ParseAndValidateYAML(10, "project-1", raw, collaborators); err == nil {
			t.Fatal("expected error for unknown collaborator")
		}
	})

	t.Run("duplicate alias", func(t *testing.T) {
		raw := []byte(`
version: 1
assignees:
  - todoist_email: "alice@example.com"
    telegram_aliases: ["@shared"]
  - todoist_email: "bob@example.com"
    telegram_aliases: ["shared"]
`)
		if _, _, err := ParseAndValidateYAML(10, "project-1", raw, collaborators); err == nil {
			t.Fatal("expected duplicate alias error")
		}
	})

	t.Run("normalized duplicate for same user is skipped with warning", func(t *testing.T) {
		raw := []byte(`
version: 1
assignees:
  - todoist_email: "alice@example.com"
    telegram_aliases: ["@alice", "alice", " Alice "]
`)

		mappings, summary, err := ParseAndValidateYAML(10, "project-1", raw, collaborators)
		if err != nil {
			t.Fatalf("ParseAndValidateYAML() error = %v", err)
		}
		if len(mappings) != 1 {
			t.Fatalf("expected 1 unique mapping, got %d", len(mappings))
		}
		if len(summary.Warnings) == 0 {
			t.Fatal("expected duplicate warning")
		}
	})
}

func TestResolve(t *testing.T) {
	now := time.Now()
	messages := []db.Message{
		{
			Username:  sql.NullString{String: "alice", Valid: true},
			Text:      "Нужно это сделать",
			Timestamp: now,
		},
	}
	messageTexts := []string{"alice, [2026-04-25 10:00:00]: Нужно это сделать"}
	mappings := []db.AssigneeMapping{
		{AliasRaw: "@alice", AliasNormalized: "alice", TodoistUserID: "u1", TodoistUserName: "Alice", TodoistUserEmail: "alice@example.com"},
		{AliasRaw: "@backend", AliasNormalized: "backend", TodoistUserID: "u2", TodoistUserName: "Backend Person", TodoistUserEmail: "backend@example.com"},
	}
	collaborators := []todoist.Collaborator{
		{ID: "u1", Name: "Alice", Email: "alice@example.com"},
		{ID: "u2", Name: "Backend Person", Email: "backend@example.com"},
	}

	t.Run("author alias", func(t *testing.T) {
		resolved, err := Resolve(context.Background(), aiStub{}, messages, messageTexts, "", mappings, collaborators, false)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if resolved.TodoistID != "" || resolved.MatchSource != "" {
			t.Fatalf("expected no deterministic author match, got %#v", resolved)
		}
	})

	t.Run("manual edit", func(t *testing.T) {
		resolved, err := Resolve(context.Background(), aiStub{
			selection: &ai.AssigneeSelection{TodoistUserID: "u2"},
		}, messages, messageTexts, "@backend", mappings, collaborators, true)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if resolved.TodoistID != "u2" || resolved.MatchSource != "ai_guess" {
			t.Fatalf("unexpected manual resolution: %#v", resolved)
		}
	})

	t.Run("manual edit phrase uses ai decision", func(t *testing.T) {
		resolved, err := Resolve(context.Background(), aiStub{
			selection: &ai.AssigneeSelection{TodoistUserID: "u2"},
		}, messages, messageTexts, "Исполнителем должен быть Backend Person", mappings, collaborators, true)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if resolved.TodoistID != "u2" || resolved.MatchSource != "ai_guess" {
			t.Fatalf("unexpected ai-driven manual resolution: %#v", resolved)
		}
	})

	t.Run("explicit mention alone does not auto-assign without ai choice", func(t *testing.T) {
		resolved, err := Resolve(
			context.Background(),
			aiStub{},
			[]db.Message{{Text: "Нужно, чтобы @backend это сделал", Timestamp: now}},
			[]string{"unknown, [2026-04-25 10:00:00]: Нужно, чтобы @backend это сделал"},
			"",
			mappings,
			collaborators,
			false,
		)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if resolved.TodoistID != "" || resolved.MatchSource != "" {
			t.Fatalf("expected no deterministic mention fallback: %#v", resolved)
		}
	})

	t.Run("ai guess", func(t *testing.T) {
		resolved, err := Resolve(context.Background(), aiStub{
			selection: &ai.AssigneeSelection{TodoistUserID: "u2"},
		}, []db.Message{{Text: "Передайте Бэкенду", Timestamp: now}}, []string{"unknown, [2026-04-25 10:00:00]: Передайте Бэкенду"}, "", mappings, collaborators, false)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if resolved.TodoistID != "u2" || resolved.MatchSource != "ai_guess" {
			t.Fatalf("unexpected ai guess: %#v", resolved)
		}
	})
}
