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
	welcomeText := "🤖 *Welcome to the JiraF Bot!* 🤖\n\n" +
		"I help collect discussions and turn them into Todoist tasks.\n\n" +
		"Workflow:\n" +
		"1. First set your project with `/set_project <id or URL>`\n" +
		"2. Start a discussion with `/start_discussion`\n" +
		"3. Send messages that will be collected as context\n" +
		"4. Create a task with `/create_task` or cancel with `/cancel`\n\n" +
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

func (c *HelpCommand) Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig {
	helpText := "🤖 *JiraF Bot Commands* 🤖\n\n"

	helpText += "*Discussion Workflow:*\n"
	helpText += "• `/set_project <id|url>` - Set Todoist project for this chat\n"
	helpText += "• `/start_discussion` - Start collecting messages for task creation\n"
	helpText += "• `/cancel` - Cancel current discussion\n"
	helpText += "• `/create_task` - Create task from discussion context\n\n"

	helpText += "Type `/help` anytime to see this list again."

	helpText += "• `/analyze` - AI-analyze discussion and create smart task\n\n"

	msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
	msg.ParseMode = "Markdown"
	return &msg
}
