package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type TaskTemplate struct {
	Type           string
	Path           string
	MissingDetails []string
	Content        string
}

type taskTemplateFrontMatter struct {
	MissingDetails []string `yaml:"missing_details"`
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
			Type:           templateType,
			Path:           path,
			MissingDetails: cleanTemplateFields(metadata.MissingDetails),
			Content:        strings.TrimSpace(body),
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
		if len(template.MissingDetails) > 0 {
			b.WriteString("Fixed follow-up fields for missing_details. Use only these exact field names when they are missing from the dialog:\n")
			for _, field := range template.MissingDetails {
				b.WriteString(fmt.Sprintf("- %s\n", field))
			}
			b.WriteString("\n")
		}
		b.WriteString(template.Content)
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
