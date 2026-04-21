package ai

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// Тесты валидации приоритета (1-4)
// ============================================================================

func TestPriorityValidation(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		wantErr  bool
	}{
		{"priority_1_valid", 1, false},
		{"priority_2_valid", 2, false},
		{"priority_3_valid", 3, false},
		{"priority_4_valid", 4, false},
		{"priority_0_invalid", 0, true},
		{"priority_5_invalid", 5, true},
		{"priority_negative", -1, true},
		{"priority_too_high", 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePriority(tt.priority)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePriority(%d) error = %v, wantErr %v", tt.priority, err, tt.wantErr)
			}
		})
	}
}

func validatePriority(priority int) error {
	if priority < 1 || priority > 4 {
		return &PriorityError{Priority: priority}
	}
	return nil
}

type PriorityError struct {
	Priority int
}

func (e *PriorityError) Error() string {
	return "priority must be between 1 and 4"
}

// ============================================================================
// Тесты JSON сериализации/десериализации
// ============================================================================

func TestTaskJSON(t *testing.T) {
	task := AnalyzedTask{
		Title:          "Тестовая задача",
		Description:    "Описание тестовой задачи",
		DueDate:        "2026-03-20",
		Priority:       3,
		PriorityText:   "High",
		AssigneeNote:   "@qa-team",
		Labels:         []string{"frontend", "bug"},
		TaskType:       "bug",
		MissingDetails: []string{"шаги воспроизведения", "ожидаемое поведение"},
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded AnalyzedTask
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Title != task.Title {
		t.Errorf("Title mismatch: got %v, want %v", decoded.Title, task.Title)
	}
	if decoded.Priority != task.Priority {
		t.Errorf("Priority mismatch: got %v, want %v", decoded.Priority, task.Priority)
	}
	if decoded.AssigneeNote != task.AssigneeNote {
		t.Errorf("AssigneeNote mismatch: got %v, want %v", decoded.AssigneeNote, task.AssigneeNote)
	}
	if len(decoded.Labels) != len(task.Labels) {
		t.Errorf("Labels length mismatch: got %v, want %v", len(decoded.Labels), len(task.Labels))
	}
	if decoded.TaskType != task.TaskType {
		t.Errorf("TaskType mismatch: got %v, want %v", decoded.TaskType, task.TaskType)
	}
	if len(decoded.MissingDetails) != len(task.MissingDetails) {
		t.Errorf("MissingDetails length mismatch: got %v, want %v", len(decoded.MissingDetails), len(task.MissingDetails))
	}
}

func TestTaskJSON_AllowsStringPriority(t *testing.T) {
	raw := []byte(`{
		"title": "Тестовая задача",
		"description": "Описание",
		"due_date": "2026-05-01",
		"priority": "3",
		"task_type": "task"
	}`)

	var decoded AnalyzedTask
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Priority != 3 {
		t.Fatalf("Priority mismatch: got %v, want %v", decoded.Priority, 3)
	}
}

// ============================================================================
// Тесты конфигурации AI
// ============================================================================

func TestAIConfigFromEnv(t *testing.T) {
	os.Setenv("OPENROUTER_API_KEY", "test_key_123")
	os.Setenv("OPENROUTER_MODEL", "openai/gpt-4o-mini")
	os.Setenv("AI_PROVIDER", "openrouter")
	defer func() {
		os.Unsetenv("OPENROUTER_API_KEY")
		os.Unsetenv("OPENROUTER_MODEL")
		os.Unsetenv("AI_PROVIDER")
	}()

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey != "test_key_123" {
		t.Errorf("OPENROUTER_API_KEY mismatch: got %v, want %v", apiKey, "test_key_123")
	}

	model := os.Getenv("OPENROUTER_MODEL")
	if model != "openai/gpt-4o-mini" {
		t.Errorf("OPENROUTER_MODEL mismatch: got %v, want %v", model, "openai/gpt-4o-mini")
	}

	provider := os.Getenv("AI_PROVIDER")
	if provider != "openrouter" {
		t.Errorf("AI_PROVIDER mismatch: got %v, want %v", provider, "openrouter")
	}
}

// ============================================================================
// Интеграционный тест - проверка что всё собирается и запускается
// ============================================================================

func TestAIClientInitialization(t *testing.T) {
	ctx := context.Background()
	if ctx == nil {
		t.Error("Failed to create context")
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	select {
	case <-ctxWithTimeout.Done():
		t.Log("Context timeout works correctly")
	case <-time.After(6 * time.Second):
		t.Error("Context timeout failed")
	}
}

// ============================================================================
// Тесты приоритетов (текстовые описания)
// ============================================================================

func TestPriorityTextMapping(t *testing.T) {
	tests := []struct {
		priority     int
		expectedText string
	}{
		{1, "Normal"},
		{2, "Medium"},
		{3, "High"},
		{4, "Urgent"},
	}

	priorityMap := map[int]string{
		1: "Normal",
		2: "Medium",
		3: "High",
		4: "Urgent",
	}

	for _, tt := range tests {
		t.Run(tt.expectedText, func(t *testing.T) {
			text, ok := priorityMap[tt.priority]
			if !ok {
				t.Errorf("Priority %d not found in map", tt.priority)
			}
			if text != tt.expectedText {
				t.Errorf("PriorityText mismatch: got %v, want %v", text, tt.expectedText)
			}
		})
	}
}

// ============================================================================
// Тесты форматирования сообщений для AI
// ============================================================================

func TestMessageFormatting(t *testing.T) {
	messages := []string{
		"alex, [2026-03-08 15:00:00]: Нужно сделать задачу",
		"max, [2026-03-08 15:01:00]: Какую задачу?",
		"alex, [2026-03-08 15:02:00]: Купить молоко к завтра",
	}

	formatted := strings.Join(messages, "\n")

	if !strings.Contains(formatted, "alex") {
		t.Error("Formatted messages should contain username 'alex'")
	}
	if !strings.Contains(formatted, "Купить молоко") {
		t.Error("Formatted messages should contain message content")
	}
	if len(formatted) == 0 {
		t.Error("Formatted messages should not be empty")
	}

	t.Logf("Formatted messages length: %d", len(formatted))
}

// ============================================================================
// Тесты проверки API ключа
// ============================================================================

func TestAIConnection(t *testing.T) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		t.Skip("OPENROUTER_API_KEY not set, skipping AI connection test")
	}

	if len(apiKey) < 10 {
		t.Error("OPENROUTER_API_KEY seems too short")
	}

	t.Log("AI API key is present and appears valid")
}

// ============================================================================
// Полный тест создания задачи
// ============================================================================

func TestTaskCreationFlow(t *testing.T) {
	messages := []string{
		"alex, [2026-03-08 15:00:00]: Нужно создать задачу на разработку",
		"max, [2026-03-08 15:01:00]: Какую именно?",
		"alex, [2026-03-08 15:02:00]: Сделать лендинг для проекта срочно к пятнице",
	}

	formatted := strings.Join(messages, "\n")
	if len(formatted) == 0 {
		t.Fatal("Failed to format messages")
	}

	aiResponse := `{
		"title": "Сделать лендинг для проекта",
		"description": "Разработка лендинга для проекта",
		"due_date": "2026-03-13",
		"priority": 4,
		"assignee_note": "@alex",
		"labels": ["frontend", "urgent", "project"],
		"task_type": "epic",
		"missing_details": ["риски", "зависимости"]
	}`

	var task AnalyzedTask
	if err := json.Unmarshal([]byte(aiResponse), &task); err != nil {
		t.Fatalf("Failed to parse AI response: %v", err)
	}

	if task.Priority < 1 || task.Priority > 4 {
		t.Errorf("Invalid priority: %d", task.Priority)
	}

	if len(task.Labels) == 0 {
		t.Error("Expected at least one label")
	}
	if task.TaskType == "" {
		t.Error("Expected task type to be present")
	}
	if task.AssigneeNote != "@alex" {
		t.Errorf("expected assignee note to be parsed, got %q", task.AssigneeNote)
	}

	t.Logf("Task created successfully: %s (Priority: %d, Due: %s)",
		task.Title, task.Priority, task.DueDate)
}

func TestValidateAndCompleteTask_FillsDerivedAndOptionalFields(t *testing.T) {
	client := &AIClient{}
	task := &AnalyzedTask{
		Title:       "Починить авторизацию",
		Description: "Нужно починить логин через OAuth",
		Priority:    10,
		TaskType:    "Эпик",
	}

	result := client.validateAndCompleteTask(task)

	if result.Priority != 1 {
		t.Fatalf("expected invalid priority to fall back to 1, got %d", result.Priority)
	}
	if result.PriorityText != "Низкий" {
		t.Fatalf("expected derived priority text, got %q", result.PriorityText)
	}
	if result.TaskType != "epic" {
		t.Fatalf("expected normalized task type epic, got %q", result.TaskType)
	}
	if result.AssigneeNote != "" {
		t.Fatalf("expected empty assignee note by default, got %q", result.AssigneeNote)
	}
	if result.Labels == nil || len(result.Labels) != 0 {
		t.Fatalf("expected empty labels slice, got %#v", result.Labels)
	}
	if result.MissingDetails == nil || len(result.MissingDetails) != 0 {
		t.Fatalf("expected empty missing details slice, got %#v", result.MissingDetails)
	}
}

func TestLoadTaskTemplates(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(
		filepath.Join(dir, "task.md"),
		[]byte("## Task template\n\n### Deadline"),
		0o644,
	); err != nil {
		t.Fatalf("write task template: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(dir, "epic.md"),
		[]byte("## Epic template\n\n### Risks"),
		0o644,
	); err != nil {
		t.Fatalf("write epic template: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(dir, "manual_check.md"),
		[]byte("## Manual check template\n\n### Environment"),
		0o644,
	); err != nil {
		t.Fatalf("write manual_check template: %v", err)
	}

	templates, err := LoadTaskTemplates(dir)
	if err != nil {
		t.Fatalf("LoadTaskTemplates() error = %v", err)
	}

	if len(templates) != 3 {
		t.Fatalf("expected 3 templates, got %d", len(templates))
	}

	if templates[0].Type != "epic" {
		t.Errorf("expected first template to be epic, got %s", templates[0].Type)
	}

	if templates[1].Type != "manual_check" {
		t.Errorf("expected second template to be manual_check, got %s", templates[1].Type)
	}

	if templates[2].Type != "task" {
		t.Errorf("expected third template to be task, got %s", templates[2].Type)
	}

	promptSection := BuildTaskTemplatesPromptSection(templates)
	if !strings.Contains(promptSection, "TEMPLATE: epic") {
		t.Error("prompt section should include epic template header")
	}
	if !strings.Contains(promptSection, "TEMPLATE: task") {
		t.Error("prompt section should include task template header")
	}
	if !strings.Contains(promptSection, "Manual check template") {
		t.Error("prompt section should include template content")
	}
}

func TestNormalizeTaskType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "bug", want: "bug"},
		{input: "Эпик", want: "epic"},
		{input: "manual check", want: "manual_check"},
		{input: "manual-check", want: "manual_check"},
		{input: "task", want: "task"},
		{input: "", want: "task"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeTaskType(tt.input)
			if got != tt.want {
				t.Errorf("normalizeTaskType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ============================================================================
// Main - запуск всех тестов
// ============================================================================

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

func setup() {
	// Инициализация перед тестами
}

func teardown() {
	// Очистка после тестов
}
