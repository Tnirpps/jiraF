package bot

import (
	"context"
	"log"
	"os"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/user/telegram-bot/internal/ai"
	"github.com/user/telegram-bot/internal/commands"
	"github.com/user/telegram-bot/internal/todoist"
)

type Bot struct {
	api             *tgbotapi.BotAPI
	commandRegistry *commands.Registry
	dbManager       commands.DBManager
	wg              sync.WaitGroup
	stopCh          chan struct{}
}

func New(telegramToken string, todoistToken string, dbManager commands.DBManager) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		return nil, err
	}

	// Create Todoist client
	todoistClient := todoist.NewClient(todoistToken)

	// Create AI client
	aiClient := ai.NewClient(os.Getenv("AI_API_TOKEN"))

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

	// Register discussion flow commands
	setProjectCmd := commands.NewSetProjectCommand(todoistClient, dbManager)
	registry.Register(setProjectCmd)

	startDiscussionCmd := commands.NewStartDiscussionCommand(dbManager)
	registry.Register(startDiscussionCmd)

	cancelCmd := commands.NewCancelCommand(dbManager)
	registry.Register(cancelCmd)

	// AI analysis command
	analyzeCmd := commands.NewAnalyzeCommand(todoistClient, dbManager, aiClient)
	registry.Register(analyzeCmd)

	// Create task from discussion command
	createTaskCmd := commands.NewCreateTaskCommand(todoistClient, dbManager)
	registry.Register(createTaskCmd)

	return &Bot{
		api:             api,
		commandRegistry: registry,
		dbManager:       dbManager,
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
	log.Printf("📨 Received message from [%s]: %s", message.From.UserName, message.Text)

	// Only process text messages
	if message.Text != "" {
		ctx := context.Background()

		// Check for active session
		hasActive, err := b.dbManager.HasActiveSession(ctx, message.Chat.ID)
		if err != nil {
			log.Printf("❌ Error checking active session: %v", err)
		} else if hasActive {
			// Check if this is NOT a command
			isCommand := message.IsCommand()
			log.Printf("🔍 Is command: %v", isCommand)
			
			if !isCommand {
				log.Printf("💾 Saving regular message: '%s'", message.Text)
				
				// Save message to database
				err := b.dbManager.SaveMessage(
					ctx,
					message.Chat.ID,
					message.MessageID,
					int64(message.From.ID),
					message.From.UserName,
					message.Text,
				)
				if err != nil {
					log.Printf("❌ Error saving message: %v", err)
				} else {
					log.Printf("✅ Message saved successfully!")
				}
			} else {
				log.Printf("⚡ Command detected, skipping save: %s", message.Text)
			}
		}
	}

	// Process commands
	if message.IsCommand() {
		log.Printf("🔄 Processing command: %s", message.Text)
		
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
}
