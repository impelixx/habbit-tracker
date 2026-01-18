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
	settingsText := "⚙️ *Settings*\n\n" +
		"Use the Mini App to configure:\n" +
		"• Reminder time\n" +
		"• Timezone\n" +
		"• Theme preferences\n\n" +
		"(Settings UI coming soon in bot)"

	msg := tgbotapi.NewMessage(message.Chat.ID, settingsText)
	msg.ParseMode = "Markdown"
	_, err := s.bot.Send(msg)
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

	// Parse callback data
	// Format: "complete:<taskID>"
	var taskIDHex string
	fmt.Sscanf(query.Data, "complete:%s", &taskIDHex)

	if taskIDHex == "" {
		return s.answerCallbackQuery(query.ID, "Invalid task ID")
	}

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
