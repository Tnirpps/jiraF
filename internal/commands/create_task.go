package commands

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/user/telegram-bot/internal/ai"
	"github.com/user/telegram-bot/internal/db"
	"github.com/user/telegram-bot/internal/taskfields"
	"github.com/user/telegram-bot/internal/tasklinks"
	"github.com/user/telegram-bot/internal/todoist"
)

// CreateTaskCommand handles the /create_task command
type CreateTaskCommand struct {
	todoistClient todoist.Client
	dbManager     DBManager
	aiClient      ai.Client
}

// NewCreateTaskCommand creates a new create_task command handler
func NewCreateTaskCommand(todoistClient todoist.Client, dbManager DBManager, aiClient ai.Client) *CreateTaskCommand {
	return &CreateTaskCommand{
		todoistClient: todoistClient,
		dbManager:     dbManager,
		aiClient:      aiClient,
	}
}

// Name returns the command name
func (c *CreateTaskCommand) Name() string {
	return "create_task"
}

// Description returns the command description
func (c *CreateTaskCommand) Description() string {
	return "Создать задачу на основе обсуждения"
}

// Execute handles the command execution
func (c *CreateTaskCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	ctx := context.Background()

	if _, err := c.dbManager.GetTodoistProjectID(ctx, message.Chat.ID); err != nil {
		if err == db.ErrProjectIDNotSet {
			return buildProjectSelectionMessage(ctx, c.todoistClient, message.Chat.ID, "Сначала выберите проект Todoist:")
		}
		log.Printf("Error getting project: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Error getting project: %v", err))
		return &msg
	}

	// Check if there's an active session
	hasActive, err := c.dbManager.HasActiveSession(ctx, message.Chat.ID)
	if err != nil {
		log.Printf("Error checking session: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Error checking session: %v", err))
		return &msg
	}

	if !hasActive {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Нет активного обсуждения. Начните его командой /start_discussion.")
		return &msg
	}

	// Get active session
	session, err := c.dbManager.GetActiveSession(ctx, message.Chat.ID)
	if err != nil {
		log.Printf("Error getting session: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Error getting session: %v", err))
		return &msg
	}

	// Check if the user is the session owner
	senderID := int64(message.From.ID)
	if session.OwnerID != senderID {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Только автор обсуждения может создать задачу по итогам обсуждения.")
		return &msg
	}

	// Get all messages from the session
	messages, err := c.dbManager.GetSessionMessages(ctx, session.ID)
	if err != nil {
		log.Printf("Error getting messages: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Error getting messages: %v", err))
		return &msg
	}

	if len(messages) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "В обсуждении нет сообщений, чтобы создать задачу.")
		return &msg
	}

	// Extract text from messages
	var messageTexts []string
	for _, msg := range messages {
		if msg.Text != "" {
			var username string
			if msg.Username.Valid {
				username = msg.Username.String
			} else {
				username = "Unknown Author"
			}
			messageTexts = append(
				messageTexts,
				fmt.Sprintf("%s, [%s]: %s", username, msg.Timestamp.Format("2006-01-02 15:04:05"), msg.Text),
			)
		}
	}

	linkCandidates := buildLinkCandidates(messages)
	selectedLinks := []tasklinks.TaskLink{}
	if len(linkCandidates) > 0 {
		selectedLinks, err = c.aiClient.AnalyzeLinks(ctx, messageTexts, linkCandidates)
		if err != nil {
			log.Printf("AI link analysis failed, continuing without selected links: %v", err)
			selectedLinks = []tasklinks.TaskLink{}
		}
	}

	// Analyze with AI using our structured prompt
	log.Printf("Calling AI client to analyze discussion with %d messages", len(messageTexts))

	analyzedTask, err := c.aiClient.AnalyzeDiscussion(ctx, messageTexts, selectedLinks)
	if err != nil {
		log.Printf("AI analysis failed: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ AI суммаризация не удалась(. Попробуйте заново")
		return &msg
	}
	analyzedTask.SelectedLinks = selectedLinks

	log.Printf("AI analysis successful: Title: %s, Priority: %d, Due: %s",
		analyzedTask.Title, analyzedTask.Priority, analyzedTask.DueDate)

	// Keep assignee as part of the canonical task model; fall back to simple extraction.
	assigneeNote := analyzedTask.AssigneeNote
	if assigneeNote == "" {
		assigneeNote = c.extractAssignee(strings.Join(messageTexts, " "))
	}

	// Format due date in ISO
	dueISO := c.convertToDueISO(analyzedTask.DueDate)

	// Save draft task to database
	err = c.dbManager.SaveDraftTask(
		ctx,
		session.ID,
		analyzedTask.Title,
		analyzedTask.Description,
		dueISO,
		analyzedTask.Priority,
		analyzedTask.TaskType,
		analyzedTask.Labels,
		analyzedTask.MissingDetails,
		analyzedTask.SelectedLinks,
		assigneeNote,
		analyzedTask.TaskFields,
	)
	if err != nil {
		log.Printf("Failed to save draft task: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Error saving draft: %v", err))
		return &msg
	}

	// Create preview message
	return c.createPreviewMessage(message.Chat.ID, session.ID, analyzedTask, dueISO, assigneeNote)
}

func buildLinkCandidates(messages []db.Message) []tasklinks.LinkCandidate {
	candidates := make([]tasklinks.LinkCandidate, 0)
	seen := make(map[string]struct{})

	for _, message := range messages {
		for _, link := range message.Links {
			normalizedLinks := tasklinks.NormalizeLinks([]tasklinks.TaskLink{link})
			if len(normalizedLinks) == 0 {
				continue
			}

			url := normalizedLinks[0].URL
			key := strings.ToLower(url)
			if _, ok := seen[key]; ok {
				continue
			}

			username := ""
			if message.Username.Valid {
				username = message.Username.String
			}

			seen[key] = struct{}{}
			candidates = append(candidates, tasklinks.LinkCandidate{
				URL:         url,
				MessageID:   message.MessageID,
				Username:    username,
				Timestamp:   message.Timestamp.Format("2006-01-02 15:04:05"),
				MessageText: message.Text,
			})
		}
	}

	return candidates
}

func CreateInlineKeyboard(sessionID int) tgbotapi.InlineKeyboardMarkup {
	sessionIDStr := fmt.Sprintf("%d", sessionID)
	confirmButton := tgbotapi.NewInlineKeyboardButtonData("✅ Подтвердить", CallbackConfirm+CallbackDataSeparator+sessionIDStr)
	editButton := tgbotapi.NewInlineKeyboardButtonData("✏️ Редактировать", CallbackEdit+CallbackDataSeparator+sessionIDStr)
	cancelButton := tgbotapi.NewInlineKeyboardButtonData("❌ Отменить создание", CallbackCancel+CallbackDataSeparator+sessionIDStr)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(confirmButton, editButton, cancelButton),
	)
	return keyboard
}

// createPreviewMessage creates a task preview with buttons
func (c *CreateTaskCommand) createPreviewMessage(chatID int64, sessionID int, task *ai.AnalyzedTask, dueISO, assigneeNote string) *tgbotapi.MessageConfig {
	responseText := "✅ Черновик задачи готов.\n\n"
	responseText += FormatTaskPreview(task, dueISO, assigneeNote, "Если хочешь, нажми `Редактировать` и дополни это в задаче.")
	responseText += "\n\nПроверь описание и выбери действие:"

	// Create message with inline buttons
	msg := tgbotapi.NewMessage(chatID, responseText)
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true

	// Add inline keyboard
	msg.ReplyMarkup = CreateInlineKeyboard(sessionID)

	return &msg
}

func FormatTaskPreview(task *ai.AnalyzedTask, dueISO, assigneeNote, missingDetailsHint string) string {
	if task == nil {
		return ""
	}

	dueDisplay := FormatDueDateForDisplay(dueISO)
	description := FormatDescriptionForTelegram(task.Description)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("*Название:* %s\n", task.Title))
	if description != "" {
		b.WriteString(fmt.Sprintf("*Описание:*\n%s\n", description))
	}
	if fieldsPreview := FormatTaskFieldsPreview(task.TaskFields); fieldsPreview != "" {
		b.WriteString(fieldsPreview)
		b.WriteString("\n")
	}
	if dueDisplay != "" {
		b.WriteString(fmt.Sprintf("*Срок выполнения:* %s\n", dueDisplay))
	}
	if task.PriorityText != "" {
		b.WriteString(fmt.Sprintf("*Приоритет:* %s\n", task.PriorityText))
	}
	b.WriteString(fmt.Sprintf("*Тип задачи:* %s\n", formatTaskType(task.TaskType)))
	if assigneeNote != "" {
		b.WriteString(fmt.Sprintf("*Исполнитель:* %s\n", assigneeNote))
	}
	labels := cleanLabels(task.Labels)
	if len(labels) > 0 {
		b.WriteString(fmt.Sprintf("*Метки:* %s\n", strings.Join(labels, ", ")))
	}
	if len(task.SelectedLinks) > 0 {
		b.WriteString("\n")
		b.WriteString(FormatSelectedLinksPreview(task.SelectedLinks))
		b.WriteString("\n")
	}
	if len(task.MissingDetails) > 0 {
		b.WriteString("\n")
		b.WriteString(FormatMissingDetailsPrompt(task.MissingDetails))
		b.WriteString("\n")
		if missingDetailsHint != "" {
			b.WriteString(missingDetailsHint)
			b.WriteString("\n")
		}
	}

	return strings.TrimSpace(b.String())
}

func FormatTaskFieldsPreview(fields taskfields.TaskFields) string {
	var b strings.Builder
	for _, field := range fields.FilledDefinitions() {
		value := fields.Value(field.Key)
		if value == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("*%s:* %s\n", field.Label, value))
	}
	return strings.TrimSpace(b.String())
}

func FormatDescriptionForTelegram(description string) string {
	lines := strings.Split(strings.TrimSpace(description), "\n")
	formatted := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "### ") {
			formatted = append(formatted, "*"+strings.TrimSpace(strings.TrimPrefix(trimmed, "### "))+"*")
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			formatted = append(formatted, "*"+strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))+"*")
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			formatted = append(formatted, "*"+strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))+"*")
			continue
		}
		formatted = append(formatted, line)
	}

	return strings.TrimSpace(strings.Join(formatted, "\n"))
}

func cleanLabels(labels []string) []string {
	cleaned := make([]string, 0, len(labels))
	for _, label := range labels {
		label = strings.TrimSpace(label)
		if label != "" {
			cleaned = append(cleaned, label)
		}
	}
	return cleaned
}

func formatTaskType(taskType string) string {
	normalized := strings.ToLower(strings.TrimSpace(taskType))

	switch normalized {
	case "bug":
		return "Баг"
	case "epic":
		return "Эпик"
	}

	parts := strings.FieldsFunc(normalized, func(r rune) bool {
		return r == '_' || r == '-' || unicode.IsSpace(r)
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}

	if len(parts) == 0 {
		return "Задача"
	}

	return strings.Join(parts, " ")
}

func FormatTaskTypeForBot(taskType string) string {
	return formatTaskType(taskType)
}

func FormatMissingDetailsPrompt(details []string) string {
	formattedDetails := formatDetailsList(details)
	if formattedDetails == "" {
		return ""
	}

	return fmt.Sprintf("*Можно ещё уточнить:* похоже, перед созданием задачи стоит обсудить %s.", formattedDetails)
}

func FormatSelectedLinksPreview(links []tasklinks.TaskLink) string {
	if len(links) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("*Полезные материалы:*\n")
	for _, link := range tasklinks.NormalizeLinks(links) {
		b.WriteString(fmt.Sprintf("• %s: %s — %s\n", link.Role, link.URL, link.Reason))
	}

	return strings.TrimSpace(b.String())
}

func AppendSelectedLinksToDescription(description string, links []tasklinks.TaskLink) string {
	links = tasklinks.NormalizeLinks(links)
	if len(links) == 0 {
		return strings.TrimSpace(description)
	}

	var b strings.Builder
	b.WriteString(strings.TrimSpace(description))
	if b.Len() > 0 {
		b.WriteString("\n\n")
	}
	b.WriteString("## Полезные материалы\n")
	for _, link := range links {
		b.WriteString(fmt.Sprintf("- **%s:** %s — %s\n", link.Role, link.URL, link.Reason))
	}

	return strings.TrimSpace(b.String())
}

func BuildTodoistDescription(description string, fields taskfields.TaskFields, links []tasklinks.TaskLink) string {
	var sections []string

	if description = strings.TrimSpace(description); description != "" {
		sections = append(sections, "## Описание\n"+description)
	}
	if fieldsText := formatTaskFieldsForTodoist(fields); fieldsText != "" {
		sections = append(sections, "## Детали задачи\n"+fieldsText)
	}
	if linksText := formatSelectedLinksForTodoist(links); linksText != "" {
		sections = append(sections, "## Полезные материалы\n"+linksText)
	}

	return strings.TrimSpace(strings.Join(sections, "\n\n"))
}

func AppendTaskFieldsToDescription(description string, fields taskfields.TaskFields) string {
	fieldsText := formatTaskFieldsForTodoist(fields)
	if fieldsText == "" {
		return strings.TrimSpace(description)
	}

	var b strings.Builder
	b.WriteString(strings.TrimSpace(description))
	if b.Len() > 0 {
		b.WriteString("\n\n")
	}
	b.WriteString(fieldsText)
	return strings.TrimSpace(b.String())
}

func formatTaskFieldsForTodoist(fields taskfields.TaskFields) string {
	var b strings.Builder
	for _, field := range fields.FilledDefinitions() {
		value := fields.Value(field.Key)
		if value == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("- **%s:** %s\n", field.Label, value))
	}
	return strings.TrimSpace(b.String())
}

func formatSelectedLinksForTodoist(links []tasklinks.TaskLink) string {
	links = tasklinks.NormalizeLinks(links)
	if len(links) == 0 {
		return ""
	}

	var b strings.Builder
	for _, link := range links {
		b.WriteString(fmt.Sprintf("- **%s:** %s — %s\n", link.Role, link.URL, link.Reason))
	}
	return strings.TrimSpace(b.String())
}

func formatDetailsList(details []string) string {
	cleaned := make([]string, 0, len(details))
	seen := make(map[string]struct{}, len(details))

	for _, detail := range details {
		detail = strings.TrimSpace(detail)
		if detail == "" {
			continue
		}

		detail = strings.Trim(detail, "*_`")
		detail = lowerFirstDetailRune(detail)

		key := strings.ToLower(detail)
		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		cleaned = append(cleaned, detail)
	}

	switch len(cleaned) {
	case 0:
		return ""
	case 1:
		return cleaned[0]
	case 2:
		return cleaned[0] + " и " + cleaned[1]
	default:
		return strings.Join(cleaned[:len(cleaned)-1], ", ") + " и " + cleaned[len(cleaned)-1]
	}
}

func lowerFirstDetailRune(detail string) string {
	runes := []rune(detail)
	if len(runes) == 0 {
		return detail
	}

	if len(runes) > 1 && unicode.IsUpper(runes[0]) && unicode.IsUpper(runes[1]) {
		return detail
	}

	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// extractAssignee extracts assignee information from messages
func (c *CreateTaskCommand) extractAssignee(text string) string {
	lowerText := strings.ToLower(text)

	// Look for mentions
	if strings.Contains(text, "@") {
		words := strings.Fields(text)
		for _, word := range words {
			if strings.HasPrefix(word, "@") {
				return word
			}
		}
	}

	// Look for assignee phrases
	assigneePhrases := []string{
		"назначить", "ответственный", "исполнитель", "поручить",
		"assign to", "responsible", "assignee",
	}

	for _, phrase := range assigneePhrases {
		if idx := strings.Index(lowerText, phrase); idx != -1 {
			// Get the text after the phrase - use original text to preserve case
			afterPhraseIdx := idx + len(phrase)
			afterPhrase := text[afterPhraseIdx:]
			words := strings.Fields(afterPhrase)
			if len(words) > 0 {
				return words[0]
			}
		}
	}

	return ""
}

// convertToDueISO converts human-readable due date to ISO format
func (c *CreateTaskCommand) convertToDueISO(dueStr string) string {
	if dueStr == "" {
		return ""
	}

	// Moscow timezone
	moscowLoc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Printf("Error loading timezone: %v", err)
		return dueStr
	}

	// Current time in Moscow
	now := time.Now().In(moscowLoc)

	// Basic conversions
	switch dueStr {
	case "today":
		return now.Format("2006-01-02")
	case "tomorrow":
		return now.AddDate(0, 0, 1).Format("2006-01-02")
	}

	// Handle day names
	switch dueStr {
	case "monday", "понедельник":
		return c.nextWeekday(now, time.Monday)
	case "tuesday", "вторник":
		return c.nextWeekday(now, time.Tuesday)
	case "wednesday", "среда":
		return c.nextWeekday(now, time.Wednesday)
	case "thursday", "четверг":
		return c.nextWeekday(now, time.Thursday)
	case "friday", "пятница":
		return c.nextWeekday(now, time.Friday)
	case "saturday", "суббота":
		return c.nextWeekday(now, time.Saturday)
	case "sunday", "воскресенье":
		return c.nextWeekday(now, time.Sunday)
	}

	// Handle specific dates (for now, just return as is if in proper format)
	if strings.HasPrefix(dueStr, "202") && len(dueStr) >= 10 {
		return dueStr
	}

	return dueStr
}

// nextWeekday returns the date of the next occurrence of the given weekday
func (c *CreateTaskCommand) nextWeekday(now time.Time, weekday time.Weekday) string {
	daysUntil := int(weekday - now.Weekday())
	if daysUntil <= 0 {
		daysUntil += 7
	}
	return now.AddDate(0, 0, daysUntil).Format("2006-01-02")
}

// formatDueDateForDisplay formats ISO date to human-readable form in MSK timezone
func FormatDueDateForDisplay(dueISO string) string {
	if dueISO == "" {
		return ""
	}

	// Try parsing as ISO date
	t, err := time.Parse("2006-01-02", dueISO)
	if err != nil {
		return dueISO // Return original if not parseable
	}

	// Moscow timezone
	moscowLoc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Printf("Error loading timezone: %v", err)
		return dueISO
	}

	// Format in Russian style
	t = t.In(moscowLoc)

	// Get day of week in Russian
	dayOfWeek := []string{
		"Воскресенье",
		"Понедельник",
		"Вторник",
		"Среда",
		"Четверг",
		"Пятница",
		"Суббота",
	}[t.Weekday()]

	// Get month in Russian
	months := []string{
		"января",
		"февраля",
		"марта",
		"апреля",
		"мая",
		"июня",
		"июля",
		"августа",
		"сентября",
		"октября",
		"ноября",
		"декабря",
	}
	month := months[t.Month()-1]

	return fmt.Sprintf("%d %s (%s)", t.Day(), month, dayOfWeek)
}
