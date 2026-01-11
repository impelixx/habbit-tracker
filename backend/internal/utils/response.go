package utils

import "github.com/gofiber/fiber/v2"

// SuccessResponse sends a success JSON response
func SuccessResponse(c *fiber.Ctx, data interface{}) error {
	return c.JSON(fiber.Map{
		"success": true,
		"data":    data,
	})
}

// ErrorResponse sends an error JSON response
func ErrorResponse(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{
		"success": false,
		"error":   message,
	})
}

// MessageResponse sends a simple message response
func MessageResponse(c *fiber.Ctx, message string) error {
	return c.JSON(fiber.Map{
		"success": true,
		"message": message,
	})
}
