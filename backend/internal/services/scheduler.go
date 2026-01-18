package services

import (
	"context"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/impelixx/habbit-tracker/backend/internal/db"
	"github.com/impelixx/habbit-tracker/backend/internal/models"
	"github.com/rs/zerolog/log"
)

// SchedulerService handles background reminder scheduling
type SchedulerService struct {
	db           *db.MongoDB
	bot          *tgbotapi.BotAPI
	stopChan     chan struct{}
	checkInterval time.Duration
}

// NewSchedulerService creates a new scheduler service
func NewSchedulerService(database *db.MongoDB, bot *tgbotapi.BotAPI) *SchedulerService {
	return &SchedulerService{
		db:           database,
		bot:          bot,
		stopChan:     make(chan struct{}),
		checkInterval: 1 * time.Minute, // Check every minute
	}
}

// Start begins the scheduler
func (s *SchedulerService) Start() {
	log.Info().Msg("Starting reminder scheduler...")

	ticker := time.NewTicker(s.checkInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := s.checkAndSendReminders(); err != nil {
					log.Error().Err(err).Msg("Failed to check reminders")
				}
			case <-s.stopChan:
				ticker.Stop()
				log.Info().Msg("Reminder scheduler stopped")
				return
			}
		}
	}()

	log.Info().Msg("Reminder scheduler started")
}

// Stop stops the scheduler
func (s *SchedulerService) Stop() {
	close(s.stopChan)
}

// checkAndSendReminders checks for users who need reminders and sends them
func (s *SchedulerService) checkAndSendReminders() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get all enabled reminders
	reminders, err := s.db.GetEnabledReminders(ctx)
	if err != nil {
		return fmt.Errorf("failed to get reminders: %w", err)
	}

	now := time.Now().UTC()
	sentCount := 0

	for _, reminder := range reminders {
		// Check if it's time to send reminder
		if s.shouldSendReminder(reminder, now) {
			// Atomically update the lastSent field to prevent race conditions
			// This ensures only one scheduler instance sends the reminder
			updated, err := s.db.AtomicUpdateReminderLastSent(ctx, reminder.ID, reminder.LastSent, now)
			if err != nil {
				log.Error().
					Err(err).
					Str("userId", reminder.UserID.Hex()).
					Msg("Failed to atomically update reminder last sent time")
				continue
			}
			
			// If update failed, another instance already sent the reminder
			if !updated {
				log.Debug().
					Str("userId", reminder.UserID.Hex()).
					Msg("Reminder already sent by another instance")
				continue
			}
			
			// Only send if we successfully updated the lastSent field
			if err := s.sendReminder(ctx, reminder); err != nil {
				log.Error().
					Err(err).
					Str("userId", reminder.UserID.Hex()).
					Msg("Failed to send reminder")
				
				// Revert the lastSent update so the reminder can be retried
				if revertErr := s.db.UpdateReminderLastSent(ctx, reminder.ID, reminder.LastSent); revertErr != nil {
					log.Error().
						Err(revertErr).
						Str("userId", reminder.UserID.Hex()).
						Msg("Failed to revert lastSent after send failure")
				}
				continue
			}
			sentCount++
		}
	}

	if sentCount > 0 {
		log.Info().Int("count", sentCount).Msg("Reminders sent")
	}

	return nil
}

// shouldSendReminder checks if a reminder should be sent
func (s *SchedulerService) shouldSendReminder(reminder *models.Reminder, now time.Time) bool {
	// Parse user's timezone
	location, err := time.LoadLocation(reminder.Timezone)
	if err != nil {
		log.Warn().
			Err(err).
			Str("timezone", reminder.Timezone).
			Msg("Invalid timezone, using UTC")
		location = time.UTC
	}

	// Convert current time to user's timezone
	userTime := now.In(location)

	// Parse reminder time (HH:MM format)
	reminderHour, reminderMin := 0, 0
	if _, err := fmt.Sscanf(reminder.Time, "%d:%d", &reminderHour, &reminderMin); err != nil {
		log.Error().
			Err(err).
			Str("time", reminder.Time).
			Msg("Failed to parse reminder time")
		return false
	}

	// Check if current hour and minute match
	if userTime.Hour() != reminderHour || userTime.Minute() != reminderMin {
		return false
	}

	// Check if we already sent today
	if reminder.LastSent != nil {
		lastSentUserTime := reminder.LastSent.In(location)
		// If last sent was today, don't send again
		if lastSentUserTime.Year() == userTime.Year() &&
			lastSentUserTime.YearDay() == userTime.YearDay() {
			return false
		}
	}

	return true
}

// sendReminder sends a reminder to a user
func (s *SchedulerService) sendReminder(ctx context.Context, reminder *models.Reminder) error {
	// Get user
	user, err := s.db.GetUserByID(ctx, reminder.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Get today's tasks for user
	now := time.Now()
	tasks, err := s.db.GetTasksByUserIDAndDate(ctx, reminder.UserID, now)
	if err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	// Build reminder message
	messageText := s.buildReminderMessage(user, tasks)

	// Create inline keyboard for quick actions
	keyboard := s.buildReminderKeyboard(tasks)

	msg := tgbotapi.NewMessage(user.TelegramID, messageText)
	msg.ParseMode = "Markdown"
	if len(keyboard) > 0 {
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	}

	if _, err := s.bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	log.Info().
		Int64("telegramId", user.TelegramID).
		Int("taskCount", len(tasks)).
		Msg("Reminder sent to user")

	return nil
}

// buildReminderMessage creates the reminder message text
func (s *SchedulerService) buildReminderMessage(user *models.User, tasks []*models.Task) string {
	message := fmt.Sprintf("🔔 *Daily Reminder*\n\n")
	message += fmt.Sprintf("Good morning, %s! 👋\n\n", user.FirstName)

	if len(tasks) == 0 {
		message += "📋 You have no tasks for today.\n\n"
		message += "Use /add to create tasks or open the Mini App!"
		return message
	}

	// Count completed and incomplete tasks
	completed := 0
	incomplete := 0
	for _, task := range tasks {
		if task.Completed {
			completed++
		} else {
			incomplete++
		}
	}

	message += fmt.Sprintf("📋 *Today's Tasks:* %d total\n", len(tasks))
	message += fmt.Sprintf("✅ Completed: %d\n", completed)
	message += fmt.Sprintf("⭕ Remaining: %d\n\n", incomplete)

	// List incomplete tasks
	if incomplete > 0 {
		message += "*Tasks to complete:*\n"
		count := 0
		for i, task := range tasks {
			if !task.Completed && count < 10 {
				priorityEmoji := "🔵"
				if task.Priority == models.PriorityHigh {
					priorityEmoji = "🔴"
				} else if task.Priority == models.PriorityMedium {
					priorityEmoji = "🟡"
				}
				message += fmt.Sprintf("%d. %s %s\n", i+1, priorityEmoji, task.Title)
				count++
			}
		}
		if incomplete > 10 {
			message += fmt.Sprintf("... and %d more\n", incomplete-10)
		}
	}

	message += "\n💪 Let's make it a productive day!"

	return message
}

// buildReminderKeyboard creates inline keyboard for quick actions
func (s *SchedulerService) buildReminderKeyboard(tasks []*models.Task) [][]tgbotapi.InlineKeyboardButton {
	var keyboard [][]tgbotapi.InlineKeyboardButton

	// Add buttons for first 3 incomplete tasks
	count := 0
	for _, task := range tasks {
		if !task.Completed && count < 3 {
			button := tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("✓ %s", task.Title),
				fmt.Sprintf("complete:%s", task.ID.Hex()),
			)
			keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{button})
			count++
		}
	}

	// Add "View All" button
	if len(tasks) > 3 {
		viewAllButton := tgbotapi.NewInlineKeyboardButtonData(
			"📋 View All Tasks",
			"cmd:today",
		)
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{viewAllButton})
	}

	return keyboard
}
