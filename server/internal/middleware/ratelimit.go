package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// RateLimiter creates a rate limiting middleware
func RateLimiter(max int, expiration time.Duration) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: expiration,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Use user ID if authenticated, otherwise use IP
			userID := c.Locals("userID")
			if userID != nil {
				return userID.(string)
			}
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error":   "Too many requests, please try again later",
			})
		},
	})
}

// StrictRateLimiter for sensitive endpoints (e.g., auth)
func StrictRateLimiter() fiber.Handler {
	return RateLimiter(5, 15*time.Minute) // 5 requests per 15 minutes
}

// ModerateRateLimiter for regular API calls
func ModerateRateLimiter() fiber.Handler {
	return RateLimiter(30, 1*time.Minute) // 30 requests per minute
}

// RelaxedRateLimiter for read-only endpoints
func RelaxedRateLimiter() fiber.Handler {
	return RateLimiter(100, 1*time.Minute) // 100 requests per minute
}

// UploadRateLimiter for file uploads
func UploadRateLimiter() fiber.Handler {
	return RateLimiter(10, 5*time.Minute) // 10 uploads per 5 minutes
}
