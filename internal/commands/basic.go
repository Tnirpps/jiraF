package commands

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// StartCommand handles the /start command
type StartCommand struct {
	registry *Registry
}

// NewStartCommand creates a new start command handler
func NewStartCommand(registry *Registry) *StartCommand {
	return &StartCommand{
		registry: registry,
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

🔧 Что я умею
— анализировать обсуждение
— формировать черновик задачи
— отправлять задачу в Todoist

Как пользоваться:
1️⃣ Выбери проект
/set_project <id>  — установить проект Todoist для этого чата

2️⃣ Начни обсуждение
/start_discussion — начать сбор сообщений для создания задачи
Продолжайте обсуждать задачу в чате — я всё запомню.

3️⃣ Создай задачу
/create_task — создать задачу из контекста обсуждения
Я проанализирую обсуждение и предложу готовую задачу.

🧩 Полный список команд
/set_project <id> — выбрать проект Todoist для этого чата
/start_discussion — начать сбор сообщений для создания задачи
/cancel — отменить текущее обсуждение
/create_task — создать задачу на основе обсуждения
/help — показать список доступных команд
`

	msg := tgbotapi.NewMessage(message.Chat.ID, welcomeText)
	// msg.ParseMode = "Markdown"
	return &msg
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
	helpText := `🧩 Полный список команд
/set_project <id> — выбрать проект Todoist для этого чата
/start_discussion — начать сбор сообщений для создания задачи
/cancel — отменить текущее обсуждение
/create_task — создать задачу на основе обсуждения
/help — показать список доступных команд`

	msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
	// msg.ParseMode = "Markdown"
	return &msg
}
