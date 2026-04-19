package commands

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/user/telegram-bot/internal/todoist"
)

func GetMainKeyboard() tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📁 Выбрать проект"),
			tgbotapi.NewKeyboardButton("💬 Начать обсуждение"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("✅ Создать задачу"),
			tgbotapi.NewKeyboardButton("🛑 Завершить обсуждение"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📋 Список задач"),
			tgbotapi.NewKeyboardButton("❓ Помощь"),
		),
	)
	keyboard.ResizeKeyboard = true
	keyboard.OneTimeKeyboard = false
	return keyboard
}

// StartCommand handles the /start command
type StartCommand struct {
	registry      *Registry
	todoistClient todoist.Client
	dbManager     DBManager
}

// NewStartCommand creates a new start command handler
func NewStartCommand(registry *Registry, todoistClient todoist.Client, dbManager DBManager) *StartCommand {
	return &StartCommand{
		registry:      registry,
		todoistClient: todoistClient,
		dbManager:     dbManager,
	}
}

// Name returns the command name
func (c *StartCommand) Name() string {
	return "start"
}

// Description returns the command description
func (c *StartCommand) Description() string {
	return "Start interacting with the bot"
}

func (c *StartCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	welcomeText := `🤖 Привет! Я AI Task Assistant JiraF 🤖

Я помогаю превращать обсуждения в чате в готовые задачи.

🔧 Что я умею:
— анализировать обсуждение
— формировать черновик задачи
— отправлять задачу в Todoist

📋 Как пользоваться:
1️⃣ Выбери проект
2️⃣ Начни обсуждение
3️⃣ Создай задачу из контекста обсуждения

Нажмите на любую кнопку ниже для быстрого доступа:`

	msg := tgbotapi.NewMessage(message.Chat.ID, welcomeText)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = GetMainKeyboard()

	if _, err := c.dbManager.GetTodoistProjectID(context.Background(), message.Chat.ID); err == nil {
		return &msg
	}

	return buildProjectSelectionMessage(context.Background(), c.todoistClient, message.Chat.ID, welcomeText+"\n\nСначала выберите проект Todoist:")
}

// HelpCommand handles the /help command
type HelpCommand struct {
	registry *Registry
}

// NewHelpCommand creates a new help command handler
func NewHelpCommand(registry *Registry) *HelpCommand {
	return &HelpCommand{
		registry: registry,
	}
}

// Name returns the command name
func (c *HelpCommand) Name() string {
	return "help"
}

// Description returns the command description
func (c *HelpCommand) Description() string {
	return "показать список доступных команд"
}

func (c *HelpCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	// ✅ ИСПРАВЛЕНО: Убраны символы < > которые ломают Markdown
	helpText := `🧩 Полный список команд:

📁 /set_project — выбрать проект Todoist для этого чата
💬 /start_discussion — начать сбор сообщений для создания задачи
✅ /create_task — создать задачу на основе обсуждения
🛑 /cancel — завершить обсуждение без задачи
📋 /list — показать список задач
❓ /help — показать эту справку

Используйте кнопки ниже для быстрого доступа:`

	msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
	// ✅ ИСПРАВЛЕНО: Убран ParseMode чтобы не было ошибок парсинга
	// msg.ParseMode = "Markdown"
	msg.ReplyMarkup = GetMainKeyboard()
	return &msg
}
