package bot

import (
	"context"
	"log"
	"os"
	"sync"
	"strings"

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
	log.Printf("[%s] %s", message.From.UserName, message.Text)

	// Save non-command messages during active sessions
	if message.Text != "" && !message.IsCommand() {
		ctx := context.Background()
		
		hasActive, err := b.dbManager.HasActiveSession(ctx, message.Chat.ID)
		if err != nil {
			log.Printf("Error checking active session: %v", err)
		} else if hasActive {
			err := b.dbManager.SaveMessage(
				ctx,
				message.Chat.ID,
				message.MessageID,
				int64(message.From.ID),
				message.From.UserName,
				message.Text,
			)
			if err != nil {
				log.Printf("Error saving message: %v", err)
			}
		}
	}

	// Process commands
	if message.IsCommand() {
		commandName := message.Command()
		command, exists := b.commandRegistry.Get(commandName)

		if !exists {
			b.sendMessage(message.Chat.ID, "Unknown command. Use /help to see available commands.")
			return
		}

		responseMsg := command.Execute(message)
		b.sendResponse(responseMsg)
	}
}

// sendResponse отправляет сообщение с автоматическим определением ParseMode
func (b *Bot) sendResponse(msgConfig *tgbotapi.MessageConfig) {
	if msgConfig == nil {
		return
	}
	
	// Автоматически устанавливаем Markdown для сообщений с форматированием
	if strings.Contains(msgConfig.Text, "*") || strings.Contains(msgConfig.Text, "`") || 
	   strings.Contains(msgConfig.Text, "[") || strings.Contains(msgConfig.Text, "_") {
		msgConfig.ParseMode = "Markdown"
	}
	
	_, err := b.api.Send(msgConfig)
	if err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

// sendMessage упрощенная отправка текстовых сообщений
func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	b.sendResponse(&msg)
}

// newMarkdownMessage создает сообщение с Markdown форматированием
func (b *Bot) newMarkdownMessage(chatID int64, text string) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	return msg
}