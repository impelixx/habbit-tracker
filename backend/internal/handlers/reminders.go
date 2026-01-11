package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/impelixx/habbit-tracker/backend/internal/db"
	"github.com/impelixx/habbit-tracker/backend/internal/models"
	"github.com/impelixx/habbit-tracker/backend/internal/utils"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// RemindersHandler handles reminder operations
type RemindersHandler struct {
	db *db.MongoDB
}

// NewRemindersHandler creates a new reminders handler
func NewRemindersHandler(database *db.MongoDB) *RemindersHandler {
	return &RemindersHandler{
		db: database,
	}
}

// SetReminderRequest represents the set reminder request
type SetReminderRequest struct {
	Time     string `json:"time"`     // HH:MM format
	Timezone string `json:"timezone"` // e.g., "America/New_York"
	Enabled  bool   `json:"enabled"`
}

// GetReminder handles GET /api/reminders
func (h *RemindersHandler) GetReminder(c *fiber.Ctx) error {
	userID := c.Locals("userId").(primitive.ObjectID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reminder, err := h.db.GetReminderByUserID(ctx, userID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// No reminder set yet
			return utils.SuccessResponse(c, fiber.Map{
				"reminder": nil,
			})
		}
		log.Error().Err(err).Msg("Failed to get reminder")
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to get reminder")
	}

	return utils.SuccessResponse(c, fiber.Map{
		"reminder": reminder,
	})
}

// SetReminder handles POST /api/reminders/set
func (h *RemindersHandler) SetReminder(c *fiber.Ctx) error {
	userID := c.Locals("userId").(primitive.ObjectID)

	var req SetReminderRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// Validate time format (HH:MM)
	var hour, min int
	if _, err := time.Parse("15:04", req.Time); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid time format. Use HH:MM (e.g., 09:00)")
	}

	// Validate timezone
	if _, err := time.LoadLocation(req.Timezone); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid timezone")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create or update reminder
	reminder := models.NewReminder(userID, req.Time, req.Timezone)
	reminder.Enabled = req.Enabled

	if err := h.db.UpsertReminder(ctx, reminder); err != nil {
		log.Error().Err(err).Msg("Failed to set reminder")
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to set reminder")
	}

	log.Info().
		Str("userId", userID.Hex()).
		Str("time", req.Time).
		Str("timezone", req.Timezone).
		Bool("enabled", req.Enabled).
		Msg("Reminder set")

	return utils.SuccessResponse(c, fiber.Map{
		"reminder": reminder,
	})
}

// DeleteReminder handles DELETE /api/reminders
func (h *RemindersHandler) DeleteReminder(c *fiber.Ctx) error {
	userID := c.Locals("userId").(primitive.ObjectID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get existing reminder
	reminder, err := h.db.GetReminderByUserID(ctx, userID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return utils.MessageResponse(c, "No reminder to delete")
		}
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to get reminder")
	}

	// Delete reminder by setting enabled to false
	reminder.Enabled = false
	if err := h.db.UpdateReminder(ctx, reminder); err != nil {
		log.Error().Err(err).Msg("Failed to delete reminder")
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to delete reminder")
	}

	log.Info().
		Str("userId", userID.Hex()).
		Msg("Reminder disabled")

	return utils.MessageResponse(c, "Reminder disabled successfully")
}
