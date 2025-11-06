package commands

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Command defines the interface for all bot commands
type Command interface {
	// Name returns the command name (without /)
	Name() string
	// Description returns the command description for help text
	Description() string
	// Execute handles the command execution
	Execute(message *tgbotapi.Message) *tgbotapi.MessageConfig
}

// Registry holds all available commands
type Registry struct {
	commands map[string]Command
}

// NewRegistry creates a new command registry
func NewRegistry() *Registry {
	return &Registry{
		commands: make(map[string]Command),
	}
}

// Register adds a command to the registry
func (r *Registry) Register(cmd Command) {
	r.commands[cmd.Name()] = cmd
}

// Get returns a command by name
func (r *Registry) Get(name string) (Command, bool) {
	cmd, exists := r.commands[name]
	return cmd, exists
}

// GetAll returns all registered commands
func (r *Registry) GetAll() []Command {
	cmds := make([]Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		cmds = append(cmds, cmd)
	}
	return cmds
}

// GenerateHelpText generates help text for all commands
func (r *Registry) GenerateHelpText() string {
	helpText := "*Available Commands:*\n\n"
	for _, cmd := range r.GetAll() {
		// Use escape characters for special Markdown characters
		description := cmd.Description()

		// Replace Markdown special characters with their escaped versions
		description = strings.ReplaceAll(description, "[", "\\[")
		description = strings.ReplaceAll(description, "]", "\\]")
		description = strings.ReplaceAll(description, "*", "\\*")
		description = strings.ReplaceAll(description, "_", "\\_")
		description = strings.ReplaceAll(description, "`", "\\`")

		helpText += "/" + cmd.Name() + " - " + description + "\n"
	}
	return helpText
}
