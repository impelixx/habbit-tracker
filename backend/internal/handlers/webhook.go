package handlers

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gofiber/fiber/v2"
	"github.com/impelixx/habbit-tracker/backend/internal/services"
	"github.com/impelixx/habbit-tracker/backend/internal/utils"
	"github.com/rs/zerolog/log"
)

// WebhookHandler handles Telegram webhook
type WebhookHandler struct {
	telegramService *services.TelegramService
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(telegramService *services.TelegramService) *WebhookHandler {
	return &WebhookHandler{
		telegramService: telegramService,
	}
}

// HandleWebhook processes incoming Telegram updates
func (h *WebhookHandler) HandleWebhook(c *fiber.Ctx) error {
	var update tgbotapi.Update
	if err := c.BodyParser(&update); err != nil {
		log.Error().Err(err).Msg("Failed to parse webhook body")
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// Process update asynchronously to respond quickly to Telegram
	go func() {
		if err := h.telegramService.HandleUpdate(update); err != nil {
			log.Error().Err(err).Msg("Failed to handle update")
		}
	}()

	// Return 200 OK immediately to Telegram
	return c.SendStatus(fiber.StatusOK)
}
