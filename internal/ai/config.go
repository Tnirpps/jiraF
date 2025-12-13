package ai

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type AiSettings struct {
	ModelURLTemplate string `yaml:"model_url_template"`
	CreateTaskPrompt string `yaml:"create_task_prompt"`
	EditTaskPrompt   string `yaml:"edit_task_prompt"`
}

type AiSettingsRoot struct {
	YandexGPT AiSettings `yaml:"gpt"`
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
	if root.YandexGPT.CreateTaskPrompt == "" || root.YandexGPT.EditTaskPrompt == "" {
		return AiSettings{}, fmt.Errorf("prompts missing")
	}
	return root.YandexGPT, nil
}
