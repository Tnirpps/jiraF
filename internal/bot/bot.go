package bot

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/user/telegram-bot/internal/ai"
	"github.com/user/telegram-bot/internal/assignee"
	"github.com/user/telegram-bot/internal/commands"
	"github.com/user/telegram-bot/internal/db"
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

	assigneeUploadSessions map[int64]string // map[botMessageID]"chatID:projectID"
	assigneeUploadMutex    sync.RWMutex

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

	setAssigneeMapCmd := commands.NewSetAssigneeMapCommand(dbManager)
	registry.Register(setAssigneeMapCmd)

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
		api:                    api,
		commandRegistry:        registry,
		dbManager:              dbManager,
		callbackHandler:        callbackHandler,
		aiClient:               aiClient,
		todoistClient:          todoistClient,
		stopCh:                 make(chan struct{}),
		editSessions:           make(map[int64]string),
		assigneeUploadSessions: make(map[int64]string),
		pendingActionMessages:  make(map[int64]int),
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
			switch callbackType {
			case commands.CallbackConfirm:
				text = "✅ Задача успешно создана"
			case commands.CallbackCancel:
				text = "❌ Создание задачи отменено. Можете продолжать обсуждение"
			default:
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

	if message.ReplyToMessage != nil && !message.IsCommand() {
		replyToID := int64(message.ReplyToMessage.MessageID)

		b.assigneeUploadMutex.RLock()
		uploadContext, isUploadReply := b.assigneeUploadSessions[replyToID]
		b.assigneeUploadMutex.RUnlock()
		if isUploadReply {
			b.handleAssigneeMapReply(message, uploadContext)
			return
		}

		b.editMutex.RLock()
		sessionID, isEditReply := b.editSessions[replyToID]
		b.editMutex.RUnlock()

		if isEditReply {
			log.Printf("Got reply to edit request for session %s", sessionID)
			b.handleEditReply(message, sessionID)
			return
		}
	}

	if message.Text != "" && !message.IsCommand() {
		if b.handleButtonText(message) {
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
		if waitingCommand, ok := command.(commands.WaitingReplyCommand); ok {
			replyKind, replyValue, shouldWait := waitingCommand.WaitingReply(message)
			if shouldWait {
				b.sendResponseWithTracking(responseMsg, replyKind, replyValue)
				return
			}
		}
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
	b.sendResponseWithTracking(msgConfig, "", "")
}

func (b *Bot) sendResponseWithOptions(msgConfig *tgbotapi.MessageConfig, waitingForReply bool, sessionID string) {
	replyKind := ""
	replyValue := ""
	if waitingForReply && sessionID != "" {
		replyKind = "edit"
		replyValue = sessionID
	}
	b.sendResponseWithTracking(msgConfig, replyKind, replyValue)
}

func (b *Bot) sendResponseWithTracking(msgConfig *tgbotapi.MessageConfig, replyKind, replyValue string) {
	if msgConfig == nil {
		return
	}

	if containsHTTPLink(msgConfig.Text) {
		msgConfig.DisableWebPagePreview = true
	}

	requiresAction := replyKind != "" || hasInlineKeyboard(msgConfig)
	if requiresAction {
		b.deletePendingActionMessage(msgConfig.ChatID)
	}

	sent, err := b.api.Send(msgConfig)
	if err != nil {
		log.Printf("Error sending message: %v", err)
		log.Printf("Message text was: %s", msgConfig.Text)
		return
	}

	if replyKind == "edit" && replyValue != "" {
		b.editMutex.Lock()
		b.editSessions[int64(sent.MessageID)] = replyValue
		b.editMutex.Unlock()

		log.Printf("Added edit session for message ID %d, session %s", sent.MessageID, replyValue)
	}

	if replyKind == commands.ReplyKindAssigneeMapUpload && replyValue != "" {
		b.assigneeUploadMutex.Lock()
		b.assigneeUploadSessions[int64(sent.MessageID)] = replyValue
		b.assigneeUploadMutex.Unlock()
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
		TaskFields:     draftTask.Fields,
	}

	editedTask, err := b.aiClient.EditTask(ctx, aiTask, message.Text)
	if err != nil {
		log.Printf("Error editing task: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Error editing task")
		return
	}

	projectID, err := b.dbManager.GetTodoistProjectID(ctx, message.Chat.ID)
	if err != nil {
		log.Printf("Error getting Todoist project for assignee resolution: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Error retrieving project settings")
		return
	}

	resolvedAssignee := db.AssigneeSnapshot{}
	if mappings, err := b.dbManager.GetAssigneeMappings(ctx, message.Chat.ID, projectID); err == nil && len(mappings) > 0 {
		sessionMessages, messagesErr := b.dbManager.GetSessionMessages(ctx, sessionIDInt)
		if messagesErr != nil {
			log.Printf("Error retrieving session messages for assignee resolution: %v", messagesErr)
		} else if collaborators, collaboratorsErr := b.todoistClient.GetProjectCollaborators(ctx, projectID); collaboratorsErr != nil {
			log.Printf("Error retrieving Todoist collaborators: %v", collaboratorsErr)
		} else {
			messageTexts := buildMessageTexts(sessionMessages)
			manualResolutionText := editedTask.AssigneeNote
			preferManual := shouldPreferManualAssigneeResolution(message.Text, draftTask.AssigneeNote.String, editedTask.AssigneeNote)
			if preferManual {
				manualResolutionText = strings.TrimSpace(message.Text + "\n" + editedTask.AssigneeNote)
			}
			resolved, resolveErr := assignee.Resolve(ctx, b.aiClient, sessionMessages, messageTexts, manualResolutionText, mappings, collaborators, preferManual)
			if resolveErr != nil {
				log.Printf("Error resolving assignee: %v", resolveErr)
			} else {
				resolvedAssignee = db.AssigneeSnapshot{
					TodoistID:   resolved.TodoistID,
					Name:        resolved.Name,
					Email:       resolved.Email,
					MatchSource: resolved.MatchSource,
				}
			}
		}
	}

	err = b.dbManager.SaveDraftTask(ctx, db.DraftTaskInput{
		SessionID:      sessionIDInt,
		Title:          editedTask.Title,
		Description:    editedTask.Description,
		DueISO:         editedTask.DueDate,
		Priority:       editedTask.Priority,
		TaskType:       editedTask.TaskType,
		Labels:         editedTask.Labels,
		MissingDetails: editedTask.MissingDetails,
		SelectedLinks:  editedTask.SelectedLinks,
		AssigneeNote:   editedTask.AssigneeNote,
		Assignee:       resolvedAssignee,
		Fields:         editedTask.TaskFields,
	})
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
		resolvedAssignee,
		"Если хочешь, просто ответь на это сообщение и дополни это в задаче.",
	)
	responseText += "\n\n"

	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true
	msg.ReplyMarkup = commands.CreateInlineKeyboard(sessionIDInt)

	b.sendResponse(&msg)
}

func buildMessageTexts(messages []db.Message) []string {
	result := make([]string, 0, len(messages))
	for _, msg := range messages {
		if msg.Text == "" {
			continue
		}
		username := "Unknown Author"
		if msg.Username.Valid && strings.TrimSpace(msg.Username.String) != "" {
			username = msg.Username.String
		}
		result = append(result, fmt.Sprintf("%s, [%s]: %s", username, msg.Timestamp.Format("2006-01-02 15:04:05"), msg.Text))
	}
	return result
}

func (b *Bot) handleAssigneeMapReply(message *tgbotapi.Message, uploadContext string) {
	b.assigneeUploadMutex.Lock()
	delete(b.assigneeUploadSessions, int64(message.ReplyToMessage.MessageID))
	b.assigneeUploadMutex.Unlock()

	if message.Document == nil {
		b.sendMessage(message.Chat.ID, "❌ Пришлите YAML-файл документом в ответ на сообщение бота.")
		return
	}

	parts := strings.SplitN(uploadContext, ":", 2)
	if len(parts) != 2 {
		b.sendMessage(message.Chat.ID, "❌ Внутренняя ошибка загрузки маппинга.")
		return
	}
	projectID := parts[1]

	fileURL, err := b.api.GetFileDirectURL(message.Document.FileID)
	if err != nil {
		log.Printf("Error getting Telegram file URL: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Не удалось получить файл из Telegram.")
		return
	}

	httpClient := &http.Client{Timeout: 20 * time.Second}
	resp, err := httpClient.Get(fileURL)
	if err != nil {
		log.Printf("Error downloading Telegram file: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Не удалось скачать YAML-файл.")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b.sendMessage(message.Chat.ID, "❌ Telegram вернул ошибку при скачивании файла.")
		return
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading uploaded mapping file: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Не удалось прочитать YAML-файл.")
		return
	}

	ctx := context.Background()
	collaborators, err := b.todoistClient.GetProjectCollaborators(ctx, projectID)
	if err != nil {
		log.Printf("Error loading collaborators for mapping import: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Не удалось загрузить участников Todoist-проекта.")
		return
	}

	mappings, summary, err := assignee.ParseAndValidateYAML(message.Chat.ID, projectID, raw, collaborators)
	if err != nil {
		b.sendMessage(message.Chat.ID, fmt.Sprintf("❌ Не удалось импортировать YAML-маппинг: %v", err))
		return
	}

	if err := b.dbManager.ReplaceAssigneeMappings(ctx, message.Chat.ID, projectID, mappings); err != nil {
		log.Printf("Error saving assignee mappings: %v", err)
		b.sendMessage(message.Chat.ID, userFacingAssigneeMappingSaveError(err))
		return
	}

	text := fmt.Sprintf("✅ Маппинг исполнителей обновлён.\nУчастников Todoist: %d\nЗагружено alias: %d", summary.CollaboratorsCount, summary.AliasesCount)
	if len(summary.Warnings) > 0 {
		log.Printf("Assignee mapping imported with warnings for chat=%d project=%s: %s", message.Chat.ID, projectID, strings.Join(summary.Warnings, "; "))
	}
	b.sendMessage(message.Chat.ID, text)
}

func shouldPreferManualAssigneeResolution(userFeedback, previousAssigneeNote, editedAssigneeNote string) bool {
	feedback := strings.TrimSpace(strings.ToLower(userFeedback))
	if feedback == "" {
		return false
	}

	if strings.TrimSpace(editedAssigneeNote) != "" && editedAssigneeNote != previousAssigneeNote {
		return true
	}

	manualPhrases := []string{
		"исполнитель",
		"ответственный",
		"назначь",
		"назначить",
		"поставь",
		"assignee",
		"assign",
		"responsible",
	}
	for _, phrase := range manualPhrases {
		if strings.Contains(feedback, phrase) {
			return true
		}
	}

	return strings.Contains(userFeedback, "@")
}

func userFacingAssigneeMappingSaveError(err error) string {
	if err == nil {
		return "❌ Не удалось сохранить маппинг исполнителей."
	}

	errText := err.Error()
	if strings.Contains(errText, "duplicate key value violates unique constraint") {
		return "❌ Не удалось сохранить маппинг исполнителей: в YAML есть alias, которые после нормализации совпадают. Уберите дубли вроде `@user` и `user` для одного и того же ключа."
	}

	return fmt.Sprintf("❌ Не удалось сохранить маппинг исполнителей: %s.", errText)
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
