package ai

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type AiSettings struct {
	Model            string `yaml:"model"`
	ModelURLTemplate string `yaml:"model_url_template"`
	CreateTaskPrompt string `yaml:"create_task_prompt"`
	EditTaskPrompt   string `yaml:"edit_task_prompt"`
	TaskTemplatesDir string `yaml:"task_templates_dir"`
}

type AiSettingsRoot struct {
	OpenRouter AiSettings `yaml:"openrouter"`
}

func LoadAiSettings(path string) (AiSettings, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return AiSettings{}, fmt.Errorf("read prompts: %w", err)
	}

	var root AiSettingsRoot
	if err := yaml.Unmarshal(b, &root); err != nil {
		return AiSettings{}, fmt.Errorf("unmarshal prompts: %w", err)
	}

	if root.OpenRouter.Model == "" {
		return AiSettings{}, fmt.Errorf("model is required in AI settings")
	}

	if root.OpenRouter.CreateTaskPrompt == "" || root.OpenRouter.EditTaskPrompt == "" {
		return AiSettings{}, fmt.Errorf("prompts missing in AI settings")
	}

	if root.OpenRouter.TaskTemplatesDir == "" {
		root.OpenRouter.TaskTemplatesDir = "configs/task_templates"
	}

	return root.OpenRouter, nil
}
