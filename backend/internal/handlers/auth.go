package handlers

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/impelixx/habbit-tracker/backend/internal/db"
	"github.com/impelixx/habbit-tracker/backend/internal/models"
	"github.com/impelixx/habbit-tracker/backend/internal/services"
	"github.com/impelixx/habbit-tracker/backend/internal/utils"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// AuthHandler handles authentication
type AuthHandler struct {
	authService *services.AuthService
	db          *db.MongoDB
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *services.AuthService, database *db.MongoDB) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		db:          database,
	}
}

// VerifyRequest represents the verify auth request
type VerifyRequest struct {
	InitData string `json:"initData"`
}

// VerifyResponse represents the verify auth response
type VerifyResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

// Verify handles POST /api/auth/verify
func (h *AuthHandler) Verify(c *fiber.Ctx) error {
	var req VerifyRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse request body")
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// Validate initData
	initDataValues, err := h.authService.ValidateTelegramInitData(req.InitData)
	if err != nil {
		log.Error().Err(err).Msg("Failed to validate initData")
		return utils.ErrorResponse(c, fiber.StatusUnauthorized, "Invalid Telegram data")
	}

	// Parse user data from initData
	userDataStr := initDataValues["user"]
	if userDataStr == "" {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "User data not found in initData")
	}

	var userData struct {
		ID           int64  `json:"id"`
		FirstName    string `json:"first_name"`
		LastName     string `json:"last_name"`
		Username     string `json:"username"`
		LanguageCode string `json:"language_code"`
	}

	if err := json.Unmarshal([]byte(userDataStr), &userData); err != nil {
		log.Error().Err(err).Msg("Failed to parse user data")
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid user data")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get or create user
	user, err := h.db.GetUserByTelegramID(ctx, userData.ID)
	if err != nil {
		if err != mongo.ErrNoDocuments {
			log.Error().Err(err).Msg("Failed to get user")
			return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to get user")
		}

		// Create new user
		user = models.NewUser(
			userData.ID,
			userData.Username,
			userData.FirstName,
			userData.LastName,
		)
		user.LanguageCode = userData.LanguageCode

		if err := h.db.CreateUser(ctx, user); err != nil {
			log.Error().Err(err).Msg("Failed to create user")
			return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to create user")
		}

		log.Info().Int64("telegramId", userData.ID).Msg("New user created via Mini App")
	}

	// Generate JWT
	token, err := h.authService.GenerateJWT(user.ID, user.TelegramID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate JWT")
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to generate token")
	}

	// Return response
	response := VerifyResponse{
		Token: token,
		User:  user,
	}

	return utils.SuccessResponse(c, response)
}

// GetMe handles GET /api/auth/me (requires auth)
func (h *AuthHandler) GetMe(c *fiber.Ctx) error {
	userID := c.Locals("userId")
	if userID == nil {
		return utils.ErrorResponse(c, fiber.StatusUnauthorized, "User not authenticated")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := h.db.GetUserByID(ctx, userID.(primitive.ObjectID))
	if err != nil {
		log.Error().Err(err).Msg("Failed to get user")
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to get user")
	}

	return utils.SuccessResponse(c, fiber.Map{"user": user})
}
