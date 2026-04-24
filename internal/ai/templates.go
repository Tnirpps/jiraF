package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/user/telegram-bot/internal/taskfields"
	"gopkg.in/yaml.v3"
)

type TaskTemplate struct {
	Type        string
	Path        string
	Description string
	Fields      []taskfields.FieldDefinition
	Content     string
}

type taskTemplateFrontMatter struct {
	Description    string                       `yaml:"description"`
	Fields         []taskfields.FieldDefinition `yaml:"fields"`
	MissingDetails []string                     `yaml:"missing_details"`
}

func LoadTaskTemplates(dir string) ([]TaskTemplate, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read task templates dir: %w", err)
	}

	templates := make([]TaskTemplate, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		templateType := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		templateType = normalizeTaskType(templateType)

		path := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read task template %s: %w", path, err)
		}

		metadata, body, err := parseTaskTemplateContent(content)
		if err != nil {
			return nil, fmt.Errorf("parse task template %s: %w", path, err)
		}

		templates = append(templates, TaskTemplate{
			Type:        templateType,
			Path:        path,
			Description: strings.TrimSpace(metadata.Description),
			Fields:      cleanTemplateFieldDefinitions(metadata.Fields, metadata.MissingDetails),
			Content:     strings.TrimSpace(body),
		})
	}

	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Type < templates[j].Type
	})

	if len(templates) == 0 {
		return nil, fmt.Errorf("no task templates found in %s", dir)
	}

	return templates, nil
}

func BuildTaskTemplatesPromptSection(templates []TaskTemplate) string {
	var b strings.Builder
	b.WriteString("Available task templates:\n")

	for _, template := range templates {
		b.WriteString(fmt.Sprintf("\n=== TEMPLATE: %s ===\n", template.Type))
		if template.Description != "" {
			b.WriteString("When to use this type:\n")
			b.WriteString(template.Description)
			b.WriteString("\n\n")
		}
		if len(template.Fields) > 0 {
			b.WriteString("Fields for this task type. Fill only these exact JSON keys when the dialog contains the information:\n")
			for _, field := range template.Fields {
				b.WriteString(fmt.Sprintf("- %s (%s): %s\n", field.Key, field.Label, field.Description))
			}
		}
		if template.Content != "" {
			b.WriteString("\n")
			b.WriteString(template.Content)
		}
		b.WriteString("\n")
	}

	return strings.TrimSpace(b.String())
}

func parseTaskTemplateContent(content []byte) (taskTemplateFrontMatter, string, error) {
	text := string(content)
	if !strings.HasPrefix(text, "---\n") {
		return taskTemplateFrontMatter{}, text, nil
	}

	rest := strings.TrimPrefix(text, "---\n")
	end := strings.Index(rest, "\n---")
	if end == -1 {
		return taskTemplateFrontMatter{}, "", fmt.Errorf("front matter is not closed")
	}

	frontMatterText := rest[:end]
	body := strings.TrimPrefix(rest[end:], "\n---")
	body = strings.TrimPrefix(body, "\r\n")
	body = strings.TrimPrefix(body, "\n")

	var metadata taskTemplateFrontMatter
	if err := yaml.Unmarshal([]byte(frontMatterText), &metadata); err != nil {
		return taskTemplateFrontMatter{}, "", err
	}

	return metadata, body, nil
}

func cleanTemplateFields(fields []string) []string {
	cleaned := make([]string, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))

	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}

		key := strings.ToLower(field)
		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		cleaned = append(cleaned, field)
	}

	return cleaned
}

func cleanTemplateFieldDefinitions(fields []taskfields.FieldDefinition, legacyFields []string) []taskfields.FieldDefinition {
	if len(fields) == 0 && len(legacyFields) > 0 {
		fields = make([]taskfields.FieldDefinition, 0, len(legacyFields))
		for _, field := range cleanTemplateFields(legacyFields) {
			if key := keyByLegacyLabel(field); key != "" {
				fields = append(fields, taskfields.FieldDefinition{
					Key:   key,
					Label: taskfields.LabelForKey(key),
				})
			}
		}
	}

	cleaned := make([]taskfields.FieldDefinition, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		field.Key = strings.TrimSpace(field.Key)
		field.Label = strings.TrimSpace(field.Label)
		field.Description = strings.TrimSpace(field.Description)
		if !taskfields.IsKnownTemplateKey(field.Key) {
			continue
		}
		if _, ok := seen[field.Key]; ok {
			continue
		}
		if field.Label == "" {
			field.Label = taskfields.LabelForKey(field.Key)
		}
		seen[field.Key] = struct{}{}
		cleaned = append(cleaned, field)
	}

	return cleaned
}

func keyByLegacyLabel(label string) string {
	normalized := strings.ToLower(strings.TrimSpace(label))
	for _, def := range taskfields.KnownDefinitions() {
		if strings.ToLower(def.Label) == normalized {
			return def.Key
		}
	}
	switch normalized {
	case "срок":
		return "due_date"
	case "полезные ссылки":
		return "selected_links"
	default:
		return ""
	}
}
