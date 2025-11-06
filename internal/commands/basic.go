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

// Execute handles the command execution
func (c *StartCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	welcomeText := "🤖 *Welcome to the Todoist Assistant Bot!* 🤖\n\n" +
		"I can help you create and manage tasks in Todoist directly from this chat.\n\n" +
		"Here are some things you can do:\n" +
		"• Use `/create Task name` to create a new task\n" +
		"• Use `/list tasks` to see your tasks\n" +
		"• Use `/list projects` to see your projects\n" +
		"• Use `/complete task_id` to mark a task as complete\n\n" +
		"Type `/help` to see all available commands."

	msg := tgbotapi.NewMessage(message.Chat.ID, welcomeText)
	msg.ParseMode = "Markdown"
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
	return "Show available commands"
}

// Execute handles the command execution
func (c *HelpCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	helpText := c.registry.GenerateHelpText()

	msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
	msg.ParseMode = "Markdown"
	return &msg
}
