package ai

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type AiSettings struct {
	Model              string `yaml:"model"`
	ModelURLTemplate   string `yaml:"model_url_template"`
	CreateTaskPrompt   string `yaml:"create_task_prompt"`
	EditTaskPrompt     string `yaml:"edit_task_prompt"`
	AnalyzeLinksPrompt string `yaml:"analyze_links_prompt"`
	TaskTemplatesDir   string `yaml:"task_templates_dir"`
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

	if root.OpenRouter.AnalyzeLinksPrompt == "" {
		root.OpenRouter.AnalyzeLinksPrompt = defaultAnalyzeLinksPrompt
	}

	if root.OpenRouter.TaskTemplatesDir == "" {
		root.OpenRouter.TaskTemplatesDir = "configs/task_templates"
	}

	return root.OpenRouter, nil
}

const defaultAnalyzeLinksPrompt = `You are a task assistant. Select only links that are useful materials for creating, understanding, implementing, or verifying the task.
Return only raw JSON:
{
  "links": [
    {
      "url": "one of input URLs",
      "role": "logs | metrics | docs | design | chat | other",
      "reason": "brief Russian phrase, 4-8 words, why this link is useful"
    }
  ]
}
Rules:
- Use only URLs from the input candidates.
- Do not fetch or assume page contents.
- Select at most 10 links.
- Keep reason compact: 4-8 words, no long sentences.
- If no link is useful, return {"links":[]}.`
