package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type TaskTemplate struct {
	Type    string
	Path    string
	Content string
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

		templates = append(templates, TaskTemplate{
			Type:    templateType,
			Path:    path,
			Content: strings.TrimSpace(string(content)),
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
		b.WriteString(template.Content)
		b.WriteString("\n")
	}

	return strings.TrimSpace(b.String())
}
