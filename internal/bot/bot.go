package bot

import (
	"log"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/user/telegram-bot/internal/commands"
	"github.com/user/telegram-bot/internal/todoist"
)

// Bot represents the Telegram bot
type Bot struct {
	api             *tgbotapi.BotAPI
	commandRegistry *commands.Registry
	wg              sync.WaitGroup
	stopCh          chan struct{}
}

// New creates a new Bot instance
func New(telegramToken string, todoistToken string) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		return nil, err
	}

	// Create Todoist client
	todoistClient := todoist.NewClient(todoistToken)

	// Initialize command registry
	registry := commands.NewRegistry()

	// Create and register commands
	startCmd := commands.NewStartCommand(registry)
	registry.Register(startCmd)

	helpCmd := commands.NewHelpCommand(registry)
	registry.Register(helpCmd)

	// Task management commands
	createCmd := commands.NewCreateCommand(todoistClient)
	registry.Register(createCmd)

	listCmd := commands.NewListCommand(todoistClient)
	registry.Register(listCmd)

	completeCmd := commands.NewCompleteCommand(todoistClient)
	registry.Register(completeCmd)

	viewCmd := commands.NewViewCommand(todoistClient)
	registry.Register(viewCmd)

	updateCmd := commands.NewUpdateCommand(todoistClient)
	registry.Register(updateCmd)

	deleteCmd := commands.NewDeleteCommand(todoistClient)
	registry.Register(deleteCmd)

	deleteConfirmCmd := commands.NewDeleteConfirmCommand(todoistClient)
	registry.Register(deleteConfirmCmd)

	return &Bot{
		api:             api,
		commandRegistry: registry,
		stopCh:          make(chan struct{}),
	}, nil
}

// Start begins listening for updates from Telegram
func (b *Bot) Start() error {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := b.api.GetUpdatesChan(updateConfig)

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		b.handleUpdates(updates)
	}()

	return nil
}

// Stop gracefully shuts down the bot
func (b *Bot) Stop() {
	close(b.stopCh)
	b.api.StopReceivingUpdates()
	b.wg.Wait()
}

// handleUpdates processes incoming updates from Telegram
func (b *Bot) handleUpdates(updates tgbotapi.UpdatesChannel) {
	for {
		select {
		case <-b.stopCh:
			return
		case update, ok := <-updates:
			if !ok {
				return
			}
			b.handleUpdate(update)
		}
	}
}

// handleUpdate processes a single update from Telegram
func (b *Bot) handleUpdate(update tgbotapi.Update) {
	if update.Message != nil {
		b.handleMessage(update.Message)
		return
	}
}

// handleMessage processes a single message from a user
func (b *Bot) handleMessage(message *tgbotapi.Message) {
	log.Printf("[%s] %s", message.From.UserName, message.Text)

	if !message.IsCommand() {
		// Send generic help message for non-commands
		msg := tgbotapi.NewMessage(message.Chat.ID, "Hello! I'm a Todoist assistant bot. Use /help to see available commands.")
		b.api.Send(msg)
		return
	}

	commandName := message.Command()
	command, exists := b.commandRegistry.Get(commandName)

	if !exists {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Unknown command. Use /help to see available commands.")
		b.api.Send(msg)
		return
	}

	// Execute the command
	responseMsg := command.Execute(message)
	_, err := b.api.Send(responseMsg)
	if err != nil {
		log.Printf("Error sending message: %v", err)
	}
}
