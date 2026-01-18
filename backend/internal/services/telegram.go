package services

import (
	"context"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/impelixx/habbit-tracker/backend/internal/db"
	"github.com/impelixx/habbit-tracker/backend/internal/models"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
)

// TelegramService handles Telegram bot operations
type TelegramService struct {
	bot       *tgbotapi.BotAPI
	db        *db.MongoDB
	aiService *AIService
}

// NewTelegramService creates a new Telegram service
func NewTelegramService(token string, database *db.MongoDB, aiService *AIService) (*TelegramService, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	log.Info().Str("username", bot.Self.UserName).Msg("Authorized on Telegram bot")

	return &TelegramService{
		bot:       bot,
		db:        database,
		aiService: aiService,
	}, nil
}

// GetBot returns the bot instance
func (s *TelegramService) GetBot() *tgbotapi.BotAPI {
	return s.bot
}

// SetWebhook sets the webhook URL for the bot
func (s *TelegramService) SetWebhook(webhookURL string) error {
	webhook, err := tgbotapi.NewWebhook(webhookURL)
	if err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	_, err = s.bot.Request(webhook)
	if err != nil {
		return fmt.Errorf("failed to set webhook: %w", err)
	}

	log.Info().Str("url", webhookURL).Msg("Webhook set successfully")
	return nil
}

// HandleUpdate processes incoming Telegram updates
func (s *TelegramService) HandleUpdate(update tgbotapi.Update) error {
	if update.Message != nil {
		return s.handleMessage(update.Message)
	}

	if update.CallbackQuery != nil {
		return s.handleCallbackQuery(update.CallbackQuery)
	}

	return nil
}

// handleMessage processes incoming messages
func (s *TelegramService) handleMessage(message *tgbotapi.Message) error {
	ctx := context.Background()

	// Get or create user
	user, err := s.getOrCreateUser(ctx, message.From)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get or create user")
		return err
	}

	// Handle commands
	if message.IsCommand() {
		return s.handleCommand(message, user)
	}

	// Handle voice messages
	if message.Voice != nil {
		return s.handleVoice(message, user)
	}

	// Handle regular text messages
	if message.Text != "" {
		return s.handleText(message, user)
	}

	return nil
}

// handleCommand processes bot commands
func (s *TelegramService) handleCommand(message *tgbotapi.Message, user *models.User) error {
	switch message.Command() {
	case "start":
		return s.handleStart(message, user)
	case "help":
		return s.handleHelp(message, user)
	case "add":
		return s.handleAdd(message, user)
	case "today":
		return s.handleToday(message, user)
	case "done":
		return s.handleDone(message, user)
	case "settings":
		return s.handleSettings(message, user)
	default:
		return s.sendMessage(message.Chat.ID, "Unknown command. Use /help to see available commands.")
	}
}

// handleStart handles the /start command
func (s *TelegramService) handleStart(message *tgbotapi.Message, user *models.User) error {
	welcomeText := fmt.Sprintf(
		"👋 Welcome to Habit Tracker, %s!\n\n"+
			"I'll help you track your tasks and habits.\n\n"+
			"Available commands:\n"+
			"/add <task> - Add a new task\n"+
			"/today - View today's tasks\n"+
			"/done - Mark tasks as completed\n"+
			"/settings - Configure reminders\n"+
			"/help - Show help message\n\n"+
			"You can also open the Mini App for a rich interface!",
		user.FirstName,
	)

	return s.sendMessage(message.Chat.ID, welcomeText)
}

// handleHelp handles the /help command
func (s *TelegramService) handleHelp(message *tgbotapi.Message, user *models.User) error {
	helpText := "📚 *Habit Tracker Help*\n\n" +
		"*Commands:*\n" +
		"/start - Start using the bot\n" +
		"/add <task> - Add a new task\n" +
		"/today - View today's tasks\n" +
		"/done - Mark tasks as completed\n" +
		"/settings - Configure reminders\n" +
		"/help - Show this help message\n\n" +
		"*Features:*\n" +
		"• Create and manage tasks\n" +
		"• Set daily reminders\n" +
		"• Track your progress\n" +
		"• Send voice messages with AI parsing 🎤\n" +
		"• Send natural language text (AI will extract tasks)\n" +
		"• Open the Mini App for full features"

	msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
	msg.ParseMode = "Markdown"
	_, err := s.bot.Send(msg)
	return err
}

// handleAdd handles the /add command
func (s *TelegramService) handleAdd(message *tgbotapi.Message, user *models.User) error {
	ctx := context.Background()

	// Get task text (everything after /add)
	taskText := message.CommandArguments()
	if taskText == "" {
		return s.sendMessage(message.Chat.ID, "Please provide a task name. Example: /add Read 30 pages")
	}

	// Create task
	task := models.NewTask(user.ID, taskText, models.SourceBot)
	if err := s.db.CreateTask(ctx, task); err != nil {
		log.Error().Err(err).Msg("Failed to create task")
		return s.sendMessage(message.Chat.ID, "Failed to create task. Please try again.")
	}

	responseText := fmt.Sprintf("✅ Task added: %s", taskText)
	return s.sendMessage(message.Chat.ID, responseText)
}

// handleToday handles the /today command
func (s *TelegramService) handleToday(message *tgbotapi.Message, user *models.User) error {
	ctx := context.Background()

	// Get today's tasks
	now := time.Now()
	tasks, err := s.db.GetTasksByUserIDAndDate(ctx, user.ID, now)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get tasks")
		return s.sendMessage(message.Chat.ID, "Failed to retrieve tasks. Please try again.")
	}

	if len(tasks) == 0 {
		return s.sendMessage(message.Chat.ID, "📋 No tasks for today. Add one with /add <task>")
	}

	// Build task list
	responseText := "📋 *Today's Tasks:*\n\n"
	for i, task := range tasks {
		status := "⭕"
		if task.Completed {
			status = "✅"
		}
		responseText += fmt.Sprintf("%d. %s %s\n", i+1, status, task.Title)
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	msg.ParseMode = "Markdown"
	_, err = s.bot.Send(msg)
	return err
}

// handleDone handles the /done command
func (s *TelegramService) handleDone(message *tgbotapi.Message, user *models.User) error {
	ctx := context.Background()

	// Get today's incomplete tasks
	now := time.Now()
	tasks, err := s.db.GetTasksByUserIDAndDate(ctx, user.ID, now)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get tasks")
		return s.sendMessage(message.Chat.ID, "Failed to retrieve tasks. Please try again.")
	}

	// Filter incomplete tasks
	incompleteTasks := make([]*models.Task, 0)
	for _, task := range tasks {
		if !task.Completed {
			incompleteTasks = append(incompleteTasks, task)
		}
	}

	if len(incompleteTasks) == 0 {
		return s.sendMessage(message.Chat.ID, "🎉 All tasks completed! Great job!")
	}

	// Create inline keyboard with tasks
	var keyboard [][]tgbotapi.InlineKeyboardButton
	for i, task := range incompleteTasks {
		button := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%d. %s", i+1, task.Title),
			fmt.Sprintf("complete:%s", task.ID.Hex()),
		)
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{button})
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "Select tasks to mark as completed:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	_, err = s.bot.Send(msg)
	return err
}

// handleSettings handles the /settings command
func (s *TelegramService) handleSettings(message *tgbotapi.Message, user *models.User) error {
	ctx := context.Background()

	// Get current reminder settings
	reminder, err := s.db.GetReminderByUserID(ctx, user.ID)

	settingsText := "⚙️ *Settings*\n\n"

	if err != nil || reminder == nil {
		settingsText += "📍 *Current Status:*\n"
		settingsText += "• Daily reminders: ❌ Disabled\n\n"
		settingsText += "Use the buttons below to set up reminders:\n"
	} else {
		statusEmoji := "✅"
		if !reminder.Enabled {
			statusEmoji = "❌"
		}
		settingsText += "📍 *Current Status:*\n"
		settingsText += fmt.Sprintf("• Daily reminders: %s %s\n", statusEmoji,
			map[bool]string{true: "Enabled", false: "Disabled"}[reminder.Enabled])
		settingsText += fmt.Sprintf("• Time: %s\n", reminder.Time)
		settingsText += fmt.Sprintf("• Timezone: %s\n\n", reminder.Timezone)
	}

	settingsText += "💡 *Available Options:*\n"
	settingsText += "• Set reminder time\n"
	settingsText += "• Enable/disable reminders\n"
	settingsText += "• View statistics\n\n"
	settingsText += "Use the Mini App for full settings control!"

	// Create inline keyboard
	var keyboard [][]tgbotapi.InlineKeyboardButton

	if reminder == nil || !reminder.Enabled {
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("🔔 Enable Reminders", "settings:enable"),
		})
	} else {
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("🔕 Disable Reminders", "settings:disable"),
		})
	}

	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("⏰ Set Time", "settings:time"),
	})

	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("📊 View Stats", "settings:stats"),
	})

	msg := tgbotapi.NewMessage(message.Chat.ID, settingsText)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	_, err = s.bot.Send(msg)
	return err
}

// handleText handles regular text messages
func (s *TelegramService) handleText(message *tgbotapi.Message, user *models.User) error {
	ctx := context.Background()

	// If AI service is available, try to parse tasks
	if s.aiService != nil {
		tasks, err := s.aiService.ParseTasksFromText(message.Text)
		if err != nil {
			log.Warn().Err(err).Msg("AI parsing failed, falling back to simple task creation")
		} else if len(tasks) > 0 {
			// Create tasks from AI parsing
			createdCount := 0
			for _, parsedTask := range tasks {
				if parsedTask.Title == "" {
					continue
				}

				task := models.NewTask(user.ID, parsedTask.Title, models.SourceBot)
				task.Description = parsedTask.Description
				task.Priority = parsedTask.Priority
				task.Metadata = &models.TaskMetadata{
					AIModel: s.aiService.primaryModel,
				}

				if err := s.db.CreateTask(ctx, task); err != nil {
					log.Error().Err(err).Msg("Failed to create task")
					continue
				}
				createdCount++
			}

			if createdCount > 0 {
				responseText := fmt.Sprintf("✅ Created %d task(s) from your message:\n", createdCount)
				for i, parsedTask := range tasks {
					if i < 5 { // Show max 5 tasks
						responseText += fmt.Sprintf("• %s (%s priority)\n", parsedTask.Title, parsedTask.Priority)
					}
				}
				if createdCount > 5 {
					responseText += fmt.Sprintf("... and %d more\n", createdCount-5)
				}
				responseText += "\nUse /today to see all tasks."
				return s.sendMessage(message.Chat.ID, responseText)
			}
		}
	}

	// Fallback: treat as simple task
	task := models.NewTask(user.ID, message.Text, models.SourceBot)
	if err := s.db.CreateTask(ctx, task); err != nil {
		log.Error().Err(err).Msg("Failed to create task")
		return s.sendMessage(message.Chat.ID, "Failed to create task. Please try again.")
	}

	responseText := fmt.Sprintf("✅ Task added: %s\n\nUse /today to see all tasks.", message.Text)
	return s.sendMessage(message.Chat.ID, responseText)
}

// handleVoice handles voice messages
func (s *TelegramService) handleVoice(message *tgbotapi.Message, user *models.User) error {
	ctx := context.Background()

	// Check if AI service is available
	if s.aiService == nil {
		return s.sendMessage(message.Chat.ID, "Voice message processing is not available. Please send text instead.")
	}

	// Send typing indicator
	s.bot.Send(tgbotapi.NewChatAction(message.Chat.ID, tgbotapi.ChatTyping))

	// Get file from Telegram
	fileConfig := tgbotapi.FileConfig{FileID: message.Voice.FileID}
	file, err := s.bot.GetFile(fileConfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get voice file")
		return s.sendMessage(message.Chat.ID, "Failed to process voice message. Please try again.")
	}

	// Note: Telegram Bot API doesn't provide direct transcription
	// We would need to download the voice file and use an external STT service
	// For now, we'll return a message indicating this feature needs OpenAI Whisper or similar
	log.Info().Str("fileId", file.FileID).Int("fileSize", file.FileSize).Msg("Voice message received")

	return s.sendMessage(message.Chat.ID,
		"🎤 Voice message received!\n\n"+
			"Voice transcription will be available soon. For now, please:\n"+
			"• Send text messages (I'll parse them with AI)\n"+
			"• Use /add <task> for simple tasks\n"+
			"• Or use the Mini App for full features")
}

// handleCallbackQuery processes callback queries from inline buttons
func (s *TelegramService) handleCallbackQuery(query *tgbotapi.CallbackQuery) error {
	ctx := context.Background()

	// Determine callback type
	data := query.Data

	// Handle different callback types
	if len(data) > 9 && data[:9] == "complete:" {
		return s.handleCompleteCallback(ctx, query, data[9:])
	} else if len(data) > 9 && data[:9] == "settings:" {
		return s.handleSettingsCallback(ctx, query, data[9:])
	} else if len(data) > 4 && data[:4] == "cmd:" {
		return s.handleCommandCallback(ctx, query, data[4:])
	}

	return s.answerCallbackQuery(query.ID, "Unknown action")
}

// handleCompleteCallback handles task completion callbacks
func (s *TelegramService) handleCompleteCallback(ctx context.Context, query *tgbotapi.CallbackQuery, taskIDHex string) error {
	// Get task
	taskID, err := models.ParseObjectID(taskIDHex)
	if err != nil {
		return s.answerCallbackQuery(query.ID, "Invalid task ID")
	}

	task, err := s.db.GetTaskByID(ctx, taskID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return s.answerCallbackQuery(query.ID, "Task not found")
		}
		return s.answerCallbackQuery(query.ID, "Failed to retrieve task")
	}

	// Mark as completed
	task.MarkCompleted()
	if err := s.db.UpdateTask(ctx, task); err != nil {
		log.Error().Err(err).Msg("Failed to update task")
		return s.answerCallbackQuery(query.ID, "Failed to complete task")
	}

	// Update message
	editMsg := tgbotapi.NewEditMessageText(
		query.Message.Chat.ID,
		query.Message.MessageID,
		fmt.Sprintf("✅ Completed: %s", task.Title),
	)
	s.bot.Send(editMsg)

	return s.answerCallbackQuery(query.ID, "Task completed!")
}

// handleSettingsCallback handles settings callbacks
func (s *TelegramService) handleSettingsCallback(ctx context.Context, query *tgbotapi.CallbackQuery, action string) error {
	// Get user from query
	user, err := s.db.GetUserByTelegramID(ctx, query.From.ID)
	if err != nil {
		return s.answerCallbackQuery(query.ID, "User not found")
	}

	switch action {
	case "enable":
		// Enable reminders with default time
		reminder := models.NewReminder(user.ID, "09:00", user.Timezone)
		reminder.Enabled = true
		if err := s.db.UpsertReminder(ctx, reminder); err != nil {
			return s.answerCallbackQuery(query.ID, "Failed to enable reminders")
		}
		s.sendMessage(query.Message.Chat.ID, "✅ Daily reminders enabled at 09:00!\n\nUse /settings to change the time.")
		return s.answerCallbackQuery(query.ID, "Reminders enabled!")

	case "disable":
		reminder, err := s.db.GetReminderByUserID(ctx, user.ID)
		if err == nil {
			reminder.Enabled = false
			s.db.UpdateReminder(ctx, reminder)
		}
		s.sendMessage(query.Message.Chat.ID, "🔕 Daily reminders disabled.")
		return s.answerCallbackQuery(query.ID, "Reminders disabled")

	case "time":
		s.sendMessage(query.Message.Chat.ID,
			"⏰ *Set Reminder Time*\n\n"+
				"To set a custom reminder time, use the Mini App.\n\n"+
				"Default times:\n"+
				"• Morning: 09:00\n"+
				"• Afternoon: 14:00\n"+
				"• Evening: 19:00")
		return s.answerCallbackQuery(query.ID, "")

	case "stats":
		return s.handleStatsCallback(ctx, query, user)
	}

	return s.answerCallbackQuery(query.ID, "Unknown setting")
}

// handleCommandCallback handles command callbacks
func (s *TelegramService) handleCommandCallback(ctx context.Context, query *tgbotapi.CallbackQuery, command string) error {
	user, err := s.db.GetUserByTelegramID(ctx, query.From.ID)
	if err != nil {
		return s.answerCallbackQuery(query.ID, "User not found")
	}

	// Create a fake message to reuse command handlers
	fakeMsg := &tgbotapi.Message{
		From: query.From,
		Chat: query.Message.Chat,
	}

	switch command {
	case "today":
		if err := s.handleToday(fakeMsg, user); err != nil {
			log.Error().Err(err).Msg("failed to handle 'today' command from callback")
			return s.answerCallbackQuery(query.ID, "Failed to send today's tasks")
		}
		return s.answerCallbackQuery(query.ID, "")
	}

	return s.answerCallbackQuery(query.ID, "Unknown command")
}

// handleStatsCallback handles stats display callback
func (s *TelegramService) handleStatsCallback(ctx context.Context, query *tgbotapi.CallbackQuery, user *models.User) error {
	// Get all user tasks
	tasks, err := s.db.GetTasksByUserID(ctx, user.ID)
	if err != nil {
		return s.answerCallbackQuery(query.ID, "Failed to get stats")
	}

	completed := 0
	for _, task := range tasks {
		if task.Completed {
			completed++
		}
	}

	completionRate := 0.0
	if len(tasks) > 0 {
		completionRate = float64(completed) / float64(len(tasks)) * 100
	}

	statsText := fmt.Sprintf(
		"📊 *Your Statistics*\n\n"+
			"📋 Total tasks: %d\n"+
			"✅ Completed: %d\n"+
			"⭕ Remaining: %d\n"+
			"📈 Completion rate: %.1f%%\n\n"+
			"💪 Keep up the great work!",
		len(tasks), completed, len(tasks)-completed, completionRate,
	)

	msg := tgbotapi.NewMessage(query.Message.Chat.ID, statsText)
	msg.ParseMode = "Markdown"
	s.bot.Send(msg)

	return s.answerCallbackQuery(query.ID, "")
}

// Helper methods

// getOrCreateUser retrieves or creates a user
func (s *TelegramService) getOrCreateUser(ctx context.Context, from *tgbotapi.User) (*models.User, error) {
	user, err := s.db.GetUserByTelegramID(ctx, from.ID)
	if err == nil {
		return user, nil
	}

	if err != mongo.ErrNoDocuments {
		return nil, err
	}

	// Create new user
	newUser := models.NewUser(
		from.ID,
		from.UserName,
		from.FirstName,
		from.LastName,
	)

	if err := s.db.CreateUser(ctx, newUser); err != nil {
		return nil, err
	}

	log.Info().Int64("telegramId", from.ID).Msg("New user created")
	return newUser, nil
}

// sendMessage sends a text message to a chat
func (s *TelegramService) sendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := s.bot.Send(msg)
	return err
}

// answerCallbackQuery answers a callback query
func (s *TelegramService) answerCallbackQuery(queryID, text string) error {
	callback := tgbotapi.NewCallback(queryID, text)
	_, err := s.bot.Request(callback)
	return err
}
