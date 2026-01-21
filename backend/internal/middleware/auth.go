package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/impelixx/habbit-tracker/backend/internal/services"
	"github.com/impelixx/habbit-tracker/backend/internal/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AuthMiddleware creates a middleware for JWT authentication
func AuthMiddleware(authService *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return utils.ErrorResponse(c, fiber.StatusUnauthorized, "Missing authorization header")
		}

		// Extract token (format: "Bearer <token>")
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return utils.ErrorResponse(c, fiber.StatusUnauthorized, "Invalid authorization header format")
		}

		tokenString := parts[1]

		// Validate token
		claims, err := authService.ValidateJWT(tokenString)
		if err != nil {
			return utils.ErrorResponse(c, fiber.StatusUnauthorized, "Invalid or expired token")
		}

		// Parse user ID
		userID, err := primitive.ObjectIDFromHex(claims.UserID)
		if err != nil {
			return utils.ErrorResponse(c, fiber.StatusUnauthorized, "Invalid user ID in token")
		}

		// Store user info in context
		c.Locals("userId", userID)
		c.Locals("telegramId", claims.TelegramID)

		return c.Next()
	}
}
