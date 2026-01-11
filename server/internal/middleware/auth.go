package middleware

import (
	"ngabarin/server/internal/utils"

	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware validates JWT token from cookie
func AuthMiddleware(c *fiber.Ctx) error {
	// Get token from cookie
	tokenString := c.Cookies("token")
	if tokenString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized - No token provided",
		})
	}

	// Validate token
	claims, err := utils.ValidateToken(tokenString)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized - Invalid token",
		})
	}

	// Store user info in context
	c.Locals("userID", claims.UserID)
	c.Locals("email", claims.Email)
	c.Locals("uniqueID", claims.UniqueID)

	return c.Next()
}

// GetUserID gets user ID from context
func GetUserID(c *fiber.Ctx) string {
	userID, ok := c.Locals("userID").(string)
	if !ok {
		return ""
	}
	return userID
}

// GetUserEmail gets user email from context
func GetUserEmail(c *fiber.Ctx) string {
	email, ok := c.Locals("email").(string)
	if !ok {
		return ""
	}
	return email
}

// GetUniqueID gets unique ID from context
func GetUniqueID(c *fiber.Ctx) string {
	uniqueID, ok := c.Locals("uniqueID").(string)
	if !ok {
		return ""
	}
	return uniqueID
}
