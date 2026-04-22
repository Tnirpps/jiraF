package bot

import (
	"context"
	"log"
	"strconv"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/user/telegram-bot/internal/ai"
	"github.com/user/telegram-bot/internal/commands"
	"github.com/user/telegram-bot/internal/tasklinks"
	"github.com/user/telegram-bot/internal/todoist"
)

type Bot struct {
	api             *tgbotapi.BotAPI
	commandRegistry *commands.Registry
	dbManager       commands.DBManager
	callbackHandler *commands.CallbackHandler
	aiClient        ai.Client
	todoistClient   todoist.Client
	wg              sync.WaitGroup
	stopCh          chan struct{}

	// Track edit sessions
	editSessions map[int64]string // map[botMessageID]sessionID
	editMutex    sync.RWMutex

	// Track the last bot message in a chat that requires a user action.
	pendingActionMessages map[int64]int
	pendingActionMutex    sync.RWMutex
}

func New(telegramToken string, dbManager commands.DBManager, aiClient ai.Client, todoistClient todoist.Client) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		return nil, err
	}

	// Initialize command registry
	registry := commands.NewRegistry()

	// Create and register commands
	startCmd := commands.NewStartCommand(registry, todoistClient, dbManager)
	registry.Register(startCmd)

	helpCmd := commands.NewHelpCommand(registry)
	registry.Register(helpCmd)

	// Task management commands
	listCmd := commands.NewListCommand(todoistClient)
	registry.Register(listCmd)

	// Register discussion flow commands
	setProjectCmd := commands.NewSetProjectCommand(todoistClient, dbManager)
	registry.Register(setProjectCmd)

	startDiscussionCmd := commands.NewStartDiscussionCommand(dbManager, todoistClient)
	registry.Register(startDiscussionCmd)

	cancelCmd := commands.NewCancelCommand(dbManager)
	registry.Register(cancelCmd)

	// Create task from discussion command
	createTaskCmd := commands.NewCreateTaskCommand(todoistClient, dbManager, aiClient)
	registry.Register(createTaskCmd)

	// Create callback handler
	callbackHandler := commands.NewCallbackHandler(todoistClient, dbManager)

	return &Bot{
		api:                   api,
		commandRegistry:       registry,
		dbManager:             dbManager,
		callbackHandler:       callbackHandler,
		aiClient:              aiClient,
		todoistClient:         todoistClient,
		stopCh:                make(chan struct{}),
		editSessions:          make(map[int64]string),
		pendingActionMessages: make(map[int64]int),
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
		b.clearPendingActionIfMatches(callback.Message.Chat.ID, callback.Message.MessageID)

		// Clear buttons without touching the already rendered message text/formatting.
		editMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{},
		})

		if _, err := b.api.Request(editMarkup); err != nil {
			log.Println("Error clearing reply markup:", err)
			return
		}

		// Check if we need to send the edit message
		if callbackResp.ResponseMessage != nil {
			b.sendResponseWithOptions(callbackResp.ResponseMessage, callbackResp.WaitingForReply, callbackResp.SessionID)
		} else if callbackType != commands.CallbackEdit {
			// Send a confirmation message for non-edit callbacks
			var text string
			if callbackType == commands.CallbackConfirm {
				text = "✅ Задача успешно создана"
			} else if callbackType == commands.CallbackCancel {
				text = "❌ Создание задачи отменено. Можете продолжать обсуждение"
			} else {
				// Unknown callback type
				text = "✅ Создание задачи отменено, продолжайте обсуждение"
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

	if message.Text != "" && !message.IsCommand() {
		if b.handleButtonText(message) {
			return
		}
	}

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
			links := tasklinks.ExtractFromTelegramMessage(message)
			err := b.dbManager.SaveMessage(
				ctx,
				message.Chat.ID,
				message.MessageID,
				int64(message.From.ID),
				message.From.UserName,
				message.Text,
				links,
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

func (b *Bot) handleButtonText(message *tgbotapi.Message) bool {
	buttonCommands := map[string]string{
		"📁 Выбрать проект":       "set_project",
		"💬 Начать обсуждение":    "start_discussion",
		"✅ Создать задачу":       "create_task",
		"🛑 Завершить обсуждение": "cancel",
		"📋 Список задач":         "list",
		"❓ Помощь":               "help",
	}

	commandName, exists := buttonCommands[message.Text]
	if !exists {
		log.Printf("[BUTTON] Кнопка не найдена в мапе: '%s'", message.Text)
		return false
	}

	log.Printf("[BUTTON] %s pressed: %s", message.From.UserName, message.Text)

	command, exists := b.commandRegistry.Get(commandName)
	if !exists {
		b.sendMessage(message.Chat.ID, "Команда недоступна.")
		return true
	}

	responseMsg := command.Execute(message)
	b.sendResponse(responseMsg)
	return true
}

// sendResponse sends a message with debugging logs
func (b *Bot) sendResponse(msgConfig *tgbotapi.MessageConfig) {
	b.sendResponseWithOptions(msgConfig, false, "")
}

func (b *Bot) sendResponseWithOptions(msgConfig *tgbotapi.MessageConfig, waitingForReply bool, sessionID string) {
	if msgConfig == nil {
		return
	}

	if containsHTTPLink(msgConfig.Text) {
		msgConfig.DisableWebPagePreview = true
	}

	requiresAction := waitingForReply || hasInlineKeyboard(msgConfig)
	if requiresAction {
		b.deletePendingActionMessage(msgConfig.ChatID)
	}

	sent, err := b.api.Send(msgConfig)
	if err != nil {
		log.Printf("Error sending message: %v", err)
		log.Printf("Message text was: %s", msgConfig.Text)
		return
	}

	if waitingForReply && sessionID != "" {
		b.editMutex.Lock()
		b.editSessions[int64(sent.MessageID)] = sessionID
		b.editMutex.Unlock()

		log.Printf("Added edit session for message ID %d, session %s", sent.MessageID, sessionID)
	}

	if requiresAction {
		b.pendingActionMutex.Lock()
		b.pendingActionMessages[msgConfig.ChatID] = sent.MessageID
		b.pendingActionMutex.Unlock()
	}
}

// sendMessage simplified method for sending text messages
func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	b.sendResponse(&msg)
}

func containsHTTPLink(text string) bool {
	return strings.Contains(text, "http://") || strings.Contains(text, "https://")
}

// handleEditReply processes a user's reply to an edit request message
func (b *Bot) handleEditReply(message *tgbotapi.Message, sessionID string) {
	log.Printf("Processing edit request for session %s: %s", sessionID, message.Text)

	// Clean up the tracking
	b.editMutex.Lock()
	delete(b.editSessions, int64(message.ReplyToMessage.MessageID))
	b.editMutex.Unlock()

	// Get draft task from database
	sessionIDInt, _ := strconv.Atoi(sessionID)
	ctx := context.Background()
	draftTask, err := b.dbManager.GetDraftTask(ctx, sessionIDInt)
	if err != nil {
		log.Printf("Error retrieving draft task: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Error retrieving task details")
		return
	}
	aiTask := &ai.AnalyzedTask{
		Title:          draftTask.Title.String,
		Description:    draftTask.Description.String,
		DueDate:        draftTask.DueISO.String,
		Priority:       int(draftTask.Priority.Int32),
		PriorityText:   "",
		AssigneeNote:   draftTask.AssigneeNote.String,
		Labels:         []string(draftTask.Labels),
		TaskType:       draftTask.TaskType.String,
		MissingDetails: []string(draftTask.MissingDetails),
		SelectedLinks:  []tasklinks.TaskLink(draftTask.SelectedLinks),
	}

	editedTask, err := b.aiClient.EditTask(ctx, aiTask, message.Text)
	if err != nil {
		log.Printf("Error editing task: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Error editing task")
		return
	}

	err = b.dbManager.SaveDraftTask(
		ctx,
		sessionIDInt,
		editedTask.Title,
		editedTask.Description,
		editedTask.DueDate,
		editedTask.Priority,
		editedTask.TaskType,
		editedTask.Labels,
		editedTask.MissingDetails,
		editedTask.SelectedLinks,
		editedTask.AssigneeNote,
	)
	if err != nil {
		log.Printf("Error saving edited task: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Error saving task")
		return
	}

	responseText := "✅ Задача обновлена!\n\nИзменения сохранены:\n"
	responseText += commands.FormatTaskPreview(
		editedTask,
		editedTask.DueDate,
		editedTask.AssigneeNote,
		"Если хочешь, просто ответь на это сообщение и дополни это в задаче.",
	)
	responseText += "\n\n"

	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true
	msg.ReplyMarkup = commands.CreateInlineKeyboard(sessionIDInt)

	b.sendResponse(&msg)
}

func hasInlineKeyboard(msgConfig *tgbotapi.MessageConfig) bool {
	markup, ok := msgConfig.ReplyMarkup.(tgbotapi.InlineKeyboardMarkup)
	return ok && len(markup.InlineKeyboard) > 0
}

func (b *Bot) clearPendingActionIfMatches(chatID int64, messageID int) {
	b.pendingActionMutex.Lock()
	defer b.pendingActionMutex.Unlock()

	if currentID, ok := b.pendingActionMessages[chatID]; ok && currentID == messageID {
		delete(b.pendingActionMessages, chatID)
	}
}

func (b *Bot) deletePendingActionMessage(chatID int64) {
	b.pendingActionMutex.Lock()
	messageID, ok := b.pendingActionMessages[chatID]
	if ok {
		delete(b.pendingActionMessages, chatID)
	}
	b.pendingActionMutex.Unlock()

	if !ok {
		return
	}

	b.editMutex.Lock()
	delete(b.editSessions, int64(messageID))
	b.editMutex.Unlock()

	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	if _, err := b.api.Request(deleteMsg); err != nil {
		log.Printf("Error deleting previous action message %d in chat %d: %v", messageID, chatID, err)
	}
}
