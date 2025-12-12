package commands

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/user/telegram-bot/internal/ai"
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
	return "Create task from discussion context"
}

// Execute handles the command execution
func (c *CreateTaskCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	ctx := context.Background()

	// Check if there's an active session
	hasActive, err := c.dbManager.HasActiveSession(ctx, message.Chat.ID)
	if err != nil {
		log.Printf("Error checking session: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Error checking session: %v", err))
		return &msg
	}

	if !hasActive {
		msg := tgbotapi.NewMessage(message.Chat.ID, "No active discussion. Start with /start_discussion first.")
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
		msg := tgbotapi.NewMessage(message.Chat.ID, "Only the user who started this discussion can create a task from it.")
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
		msg := tgbotapi.NewMessage(message.Chat.ID, "No messages in discussion to create task from.")
		return &msg
	}

	// Get project ID for this chat (will be used in the confirm stage)
	_, err = c.dbManager.GetTodoistProjectID(ctx, message.Chat.ID)
	if err != nil {
		log.Printf("Error getting project: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Error getting project: %v", err))
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

	// Analyze with AI using our structured prompt
	log.Printf("Calling AI client to analyze discussion with %d messages", len(messageTexts))

	analyzedTask, err := c.aiClient.AnalyzeDiscussion(ctx, messageTexts)
	if err != nil {
		log.Printf("AI analysis failed: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ AI analysis failed: %v", err))
		return &msg
	}

	log.Printf("AI analysis successful: Title: %s, Priority: %d, Due: %s",
		analyzedTask.Title, analyzedTask.Priority, analyzedTask.DueDate)

	// Extract assignee from messages
	assigneeNote := c.extractAssignee(strings.Join(messageTexts, " "))

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
		assigneeNote,
	)
	if err != nil {
		log.Printf("Failed to save draft task: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Error saving draft: %v", err))
		return &msg
	}

	// Create preview message
	return c.createPreviewMessage(message.Chat.ID, session.ID, analyzedTask, dueISO, assigneeNote)
}

func CreateInlineKeyboard(sessionID int) tgbotapi.InlineKeyboardMarkup {
	sessionIDStr := fmt.Sprintf("%d", sessionID)
	confirmButton := tgbotapi.NewInlineKeyboardButtonData("✅ Confirm", CallbackConfirm+CallbackDataSeparator+sessionIDStr)
	editButton := tgbotapi.NewInlineKeyboardButtonData("✏️ Edit", CallbackEdit+CallbackDataSeparator+sessionIDStr)
	cancelButton := tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", CallbackCancel+CallbackDataSeparator+sessionIDStr)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(confirmButton, editButton, cancelButton),
	)
	return keyboard
}

// createPreviewMessage creates a task preview with buttons
func (c *CreateTaskCommand) createPreviewMessage(chatID int64, sessionID int, task *ai.AnalyzedTask, dueISO, assigneeNote string) *tgbotapi.MessageConfig {
	// Format due date for display (MSK timezone)
	dueDisplay := c.formatDueDateForDisplay(dueISO)

	// Create response message with task details
	responseText := fmt.Sprintf("📝 *Draft Task Preview*\n\n"+
		"*Title:* %s\n\n"+
		"*Description:* %s\n\n",
		task.Title, task.Description)

	if dueDisplay != "" {
		responseText += fmt.Sprintf("*Due:* %s\n\n", dueDisplay)
	}

	responseText += fmt.Sprintf("*Priority:* %s\n\n", task.PriorityText)

	if assigneeNote != "" {
		responseText += fmt.Sprintf("*Assigned to:* %s\n\n", assigneeNote)
	}

	responseText += "Please confirm to create this task in Todoist."

	// Create message with inline buttons
	msg := tgbotapi.NewMessage(chatID, responseText)
	msg.ParseMode = "Markdown"

	// Add inline keyboard
	msg.ReplyMarkup = CreateInlineKeyboard(sessionID)

	return &msg
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
func (c *CreateTaskCommand) formatDueDateForDisplay(dueISO string) string {
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
