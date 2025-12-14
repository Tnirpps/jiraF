package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
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
	callbackHandler *commands.CallbackHandler
	aiClient        ai.Client
	wg              sync.WaitGroup
	stopCh          chan struct{}

	// Track edit sessions
	editSessions map[int64]string // map[botMessageID]sessionID
	editMutex    sync.RWMutex
}

func New(telegramToken string, dbManager commands.DBManager) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		return nil, err
	}

	// Create Todoist client
	todoistClient, err := todoist.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Todoist client: %w", err)
	}

	// Create AI client
	aiClient, err := ai.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create AI client: %w", err)
	}

	// Initialize command registry
	registry := commands.NewRegistry()

	// Create and register commands
	startCmd := commands.NewStartCommand(registry)
	registry.Register(startCmd)

	helpCmd := commands.NewHelpCommand(registry)
	registry.Register(helpCmd)

	// Task management commands
	listCmd := commands.NewListCommand(todoistClient)
	registry.Register(listCmd)

	// Register discussion flow commands
	setProjectCmd := commands.NewSetProjectCommand(todoistClient, dbManager)
	registry.Register(setProjectCmd)

	startDiscussionCmd := commands.NewStartDiscussionCommand(dbManager)
	registry.Register(startDiscussionCmd)

	cancelCmd := commands.NewCancelCommand(dbManager)
	registry.Register(cancelCmd)

	// Create task from discussion command
	createTaskCmd := commands.NewCreateTaskCommand(todoistClient, dbManager, aiClient)
	registry.Register(createTaskCmd)

	// Create callback handler
	callbackHandler := commands.NewCallbackHandler(todoistClient, dbManager)

	return &Bot{
		api:             api,
		commandRegistry: registry,
		dbManager:       dbManager,
		callbackHandler: callbackHandler,
		aiClient:        aiClient,
		stopCh:          make(chan struct{}),
		editSessions:    make(map[int64]string),
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

	if update.CallbackQuery != nil {
		b.handleCallback(update.CallbackQuery)
		return
	}
}

// handleCallback processes callback queries from inline buttons
func (b *Bot) handleCallback(callback *tgbotapi.CallbackQuery) {
	log.Printf("[CALLBACK] %s: %s", callback.From.UserName, callback.Data)

	// Extract command type from callback data
	parts := strings.Split(callback.Data, "_")
	if len(parts) < 1 {
		log.Printf("Invalid callback data format: %s", callback.Data)
		return
	}

	callbackType := parts[0]
	log.Printf("Parsed callback type: %s, original data: %s", callbackType, callback.Data)

	// Use our dedicated callback handler for all callback types
	callbackResp := b.callbackHandler.HandleCallback(callback)
	if callbackResp != nil && callbackResp.CallbackConfig != nil {
		_, err := b.api.Request(callbackResp.CallbackConfig)
		if err != nil {
			log.Printf("Error sending callback response: %v", err)
		}
	}

	// Only delete buttons if the user is the session owner
	if callbackResp.IsOwner {
		// Delete buttons from the original message

		editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, callback.Message.Text)
		editMsg.ParseMode = "Markdown"
		// Очищение разметки с кнопками
		editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{},
		}

		// Отправка отредактированного сообщения без кнопок
		if _, err := b.api.Send(editMsg); err != nil {
			log.Println("Error editing message:", err)
			return
		}

		// Check if we need to send the edit message
		if callbackResp.ResponseMessage != nil {
			// Send the message from the callback
			sent, err := b.api.Send(callbackResp.ResponseMessage)
			if err != nil {
				log.Printf("Error sending callback response message: %v", err)
			} else if callbackResp.WaitingForReply && callbackResp.SessionID != "" {
				// If this is an edit message waiting for reply, track it
				b.editMutex.Lock()
				b.editSessions[int64(sent.MessageID)] = callbackResp.SessionID
				b.editMutex.Unlock()

				log.Printf("Added edit session for message ID %d, session %s",
					sent.MessageID, callbackResp.SessionID)
			}
		} else if callbackType != commands.CallbackEdit {
			// Send a confirmation message for non-edit callbacks
			var text string
			if callbackType == commands.CallbackConfirm {
				text = "✅ Задача успешно создана"
			} else if callbackType == commands.CallbackCancel {
				text = "❌ Создание задачи отменено. Можете продолжать обсуждение"
			} else {
				// Unknown callback type
				text = "🔄 Action processed"
			}

			msg := tgbotapi.NewMessage(callback.Message.Chat.ID, text)
			_, err := b.api.Send(msg)
			if err != nil {
				log.Printf("Error sending confirmation message: %v", err)
			}
		}
	}
}

// handleMessage processes a single message from a user
func (b *Bot) handleMessage(message *tgbotapi.Message) {
	log.Printf("[%s] %s", message.From.UserName, message.Text)

	// Check if this is a reply to an edit request
	if message.ReplyToMessage != nil && !message.IsCommand() {
		replyToID := int64(message.ReplyToMessage.MessageID)

		// Check if this is a reply to our edit instruction message
		b.editMutex.RLock()
		sessionID, isEditReply := b.editSessions[replyToID]
		b.editMutex.RUnlock()

		if isEditReply {
			log.Printf("Got reply to edit request for session %s", sessionID)
			b.handleEditReply(message, sessionID)
			return
		}
	}

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
		log.Printf("[COMMAND] %s: %s", message.From.UserName, commandName)
		command, exists := b.commandRegistry.Get(commandName)

		if !exists {
			b.sendMessage(message.Chat.ID, "Unknown command. Use /help to see available commands.")
			return
		}

		responseMsg := command.Execute(message)
		b.sendResponse(responseMsg)
	}
}

// sendResponse sends a message with debugging logs
func (b *Bot) sendResponse(msgConfig *tgbotapi.MessageConfig) {
	if msgConfig == nil {
		return
	}

	_, err := b.api.Send(msgConfig)
	if err != nil {
		log.Printf("Error sending message: %v", err)
		log.Printf("Message text was: %s", msgConfig.Text)
	}
}

// sendMessage simplified method for sending text messages
func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	b.sendResponse(&msg)
}

// handleEditReply processes a user's reply to an edit request message
func (b *Bot) handleEditReply(message *tgbotapi.Message, sessionID string) {
	log.Printf("Processing edit request for session %s: %s", sessionID, message.Text)

	// Clean up the tracking
	b.editMutex.Lock()
	delete(b.editSessions, int64(message.ReplyToMessage.MessageID))
	b.editMutex.Unlock()

	// In a real implementation, we would:
	// 1. Get the draft task from the database
	// 2. Call the AI client's EditTask method
	// 3. Update the task in the database
	// 4. Show a new preview to the user

	// PLACEHOLDER: Get draft task from database
	// Example:
	sessionIDInt, _ := strconv.Atoi(sessionID)
	ctx := context.Background()
	draftTask, err := b.dbManager.GetDraftTask(ctx, sessionIDInt)
	if err != nil {
		log.Printf("Error retrieving draft task: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Error retrieving task details")
		return
	}
	aiTask := &ai.AnalyzedTask{
		Title:        draftTask.Title.String,
		Description:  draftTask.Description.String,
		DueDate:      draftTask.DueISO.String,
		Priority:     int(draftTask.Priority.Int32),
		PriorityText: "",
	}

	editedTask, err := b.aiClient.EditTask(ctx, aiTask, message.Text)
	if err != nil {
		log.Printf("Error editing task: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Error editing task")
		return
	}

	err = b.dbManager.SaveDraftTask(ctx, sessionIDInt, editedTask.Title, editedTask.Description, editedTask.DueDate, editedTask.Priority, "")
	if err != nil {
		log.Printf("Error saving edited task: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Error saving task")
		return
	}

	// Send back a confirmation message with the changes
	responseText := fmt.Sprintf("✅ *Task Updated!*\n\n"+
		"New details:\n"+
		"*Title:* %s\n"+
		"*Description:* %s\n"+
		"*Due:* %s\n"+
		"*Priority:* %s\n"+
		"*Labels:* %s\n\n",
		editedTask.Title,
		editedTask.Description,
		editedTask.DueDate,
		editedTask.PriorityText,
		strings.Join(editedTask.Labels, ", "))

	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = commands.CreateInlineKeyboard(sessionIDInt)

	b.sendResponse(&msg)
}
