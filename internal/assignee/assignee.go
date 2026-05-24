package assignee

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"

	"github.com/user/telegram-bot/internal/ai"
	"github.com/user/telegram-bot/internal/db"
	"github.com/user/telegram-bot/internal/todoist"
)

var mentionPattern = regexp.MustCompile(`(?i)@[\p{L}\p{N}_\.]+`)

type YAMLFile struct {
	Version   int            `yaml:"version"`
	Assignees []YAMLAssignee `yaml:"assignees"`
}

type YAMLAssignee struct {
	TodoistEmail    string   `yaml:"todoist_email"`
	TelegramAliases []string `yaml:"telegram_aliases"`
}

type ImportSummary struct {
	CollaboratorsCount int
	AliasesCount       int
	Warnings           []string
}

type Resolved struct {
	TodoistID   string
	Name        string
	Email       string
	MatchSource string
}

func NormalizeAlias(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "@")
	if value == "" {
		return ""
	}

	fields := strings.FieldsFunc(value, unicode.IsSpace)
	for i := range fields {
		fields[i] = strings.ToLower(strings.TrimSpace(fields[i]))
	}

	return strings.Join(fields, " ")
}

func ExtractMentions(text string) []string {
	matches := mentionPattern.FindAllString(text, -1)
	seen := make(map[string]struct{}, len(matches))
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		normalized := NormalizeAlias(match)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func ParseAndValidateYAML(chatID int64, projectID string, raw []byte, collaborators []todoist.Collaborator) ([]db.AssigneeMapping, ImportSummary, error) {
	var payload YAMLFile
	if err := yaml.Unmarshal(raw, &payload); err != nil {
		return nil, ImportSummary{}, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if payload.Version != 1 {
		return nil, ImportSummary{}, fmt.Errorf("unsupported mapping version %d", payload.Version)
	}
	if len(payload.Assignees) == 0 {
		return nil, ImportSummary{}, fmt.Errorf("assignees list must not be empty")
	}

	collaboratorsByEmail := make(map[string]todoist.Collaborator, len(collaborators))
	for _, collaborator := range collaborators {
		email := strings.ToLower(strings.TrimSpace(collaborator.Email))
		if email == "" {
			continue
		}
		collaboratorsByEmail[email] = collaborator
	}

	seenAliases := make(map[string]string)
	mappings := make([]db.AssigneeMapping, 0)
	summary := ImportSummary{CollaboratorsCount: len(collaborators)}

	for idx, entry := range payload.Assignees {
		email := strings.ToLower(strings.TrimSpace(entry.TodoistEmail))
		if email == "" {
			return nil, summary, fmt.Errorf("в assignees[%d] не указан todoist_email", idx)
		}

		collaborator, ok := collaboratorsByEmail[email]
		if !ok {
			return nil, summary, fmt.Errorf(
				"email %q не найден среди участников выбранного Todoist-проекта (%s). Проверьте, что в чате выбран правильный проект и что этот человек добавлен в проект Todoist",
				entry.TodoistEmail,
				projectID,
			)
		}

		if len(entry.TelegramAliases) == 0 {
			return nil, summary, fmt.Errorf("для %q список telegram_aliases пустой", entry.TodoistEmail)
		}

		for _, alias := range entry.TelegramAliases {
			rawAlias := strings.TrimSpace(alias)
			normalizedAlias := NormalizeAlias(rawAlias)
			if normalizedAlias == "" {
				summary.Warnings = append(summary.Warnings, fmt.Sprintf("empty alias skipped for %s", entry.TodoistEmail))
				continue
			}

			if existingEmail, exists := seenAliases[normalizedAlias]; exists && existingEmail != email {
				return nil, summary, fmt.Errorf("alias %q конфликтует: после нормализации он указывает на нескольких Todoist-пользователей", rawAlias)
			}
			if existingEmail, exists := seenAliases[normalizedAlias]; exists && existingEmail == email {
				summary.Warnings = append(
					summary.Warnings,
					fmt.Sprintf("alias %q пропущен: после нормализации он дублирует другой alias этого же пользователя", rawAlias),
				)
				continue
			}
			seenAliases[normalizedAlias] = email

			mappings = append(mappings, db.AssigneeMapping{
				ChatID:           chatID,
				TodoistProjectID: projectID,
				AliasRaw:         rawAlias,
				AliasNormalized:  normalizedAlias,
				TodoistUserID:    collaborator.ID,
				TodoistUserName:  collaborator.Name,
				TodoistUserEmail: collaborator.Email,
			})
			summary.AliasesCount++
		}
	}

	if len(mappings) == 0 {
		return nil, summary, fmt.Errorf("в файле не осталось валидных alias после проверки")
	}

	sort.Slice(mappings, func(i, j int) bool {
		if mappings[i].TodoistUserID == mappings[j].TodoistUserID {
			return mappings[i].AliasNormalized < mappings[j].AliasNormalized
		}
		return mappings[i].TodoistUserID < mappings[j].TodoistUserID
	})

	return mappings, summary, nil
}

func BuildAICandidates(mappings []db.AssigneeMapping, collaborators []todoist.Collaborator) []ai.AssigneeCandidate {
	collaboratorsByID := make(map[string]todoist.Collaborator, len(collaborators))
	for _, collaborator := range collaborators {
		collaboratorsByID[collaborator.ID] = collaborator
	}

	grouped := make(map[string]*ai.AssigneeCandidate)
	for _, mapping := range mappings {
		collaborator, ok := collaboratorsByID[mapping.TodoistUserID]
		if !ok {
			continue
		}
		candidate := grouped[mapping.TodoistUserID]
		if candidate == nil {
			candidate = &ai.AssigneeCandidate{
				TodoistUserID:    collaborator.ID,
				TodoistUserName:  collaborator.Name,
				TodoistUserEmail: collaborator.Email,
			}
			grouped[mapping.TodoistUserID] = candidate
		}
		candidate.Aliases = append(candidate.Aliases, mapping.AliasRaw)
	}

	ids := make([]string, 0, len(grouped))
	for id := range grouped {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	result := make([]ai.AssigneeCandidate, 0, len(ids))
	for _, id := range ids {
		candidate := grouped[id]
		sort.Strings(candidate.Aliases)
		result = append(result, *candidate)
	}
	return result
}

func Resolve(ctx context.Context, client ai.Client, messages []db.Message, messageTexts []string, assigneeNote string, mappings []db.AssigneeMapping, collaborators []todoist.Collaborator, preferManual bool) (Resolved, error) {
	activeCollaborators := make(map[string]todoist.Collaborator, len(collaborators))
	for _, collaborator := range collaborators {
		activeCollaborators[collaborator.ID] = collaborator
	}

	candidates := BuildAICandidates(mappings, collaborators)
	if len(candidates) == 0 {
		return Resolved{}, nil
	}

	selection, err := client.AnalyzeAssignee(ctx, messageTexts, assigneeNote, candidates)
	if err != nil {
		return Resolved{}, err
	}
	if selection == nil || strings.TrimSpace(selection.TodoistUserID) == "" {
		return Resolved{}, nil
	}

	collaborator, ok := activeCollaborators[selection.TodoistUserID]
	if !ok {
		return Resolved{}, nil
	}

	return Resolved{
		TodoistID:   collaborator.ID,
		Name:        collaborator.Name,
		Email:       collaborator.Email,
		MatchSource: "ai_guess",
	}, nil
}
