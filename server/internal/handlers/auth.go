package handlers

import (
	"context"
	"time"

	"ngabarin/server/internal/database"
	"ngabarin/server/internal/models"
	"ngabarin/server/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
)

// RegisterRequest represents registration request body
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// LoginRequest represents login request body
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register handles user registration
func Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	// Validate input
	if req.Email == "" || req.Password == "" || req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Email, password, and name are required",
		})
	}

	// Check if email already exists
	var exists bool
	err := database.Pool.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", req.Email).Scan(&exists)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Database error",
		})
	}

	if exists {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"success": false,
			"error":   "Email already registered",
		})
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to hash password",
		})
	}

	// Generate unique ID
	uniqueID := utils.GenerateUniqueID(req.Name)

	// Check if uniqueID exists and regenerate if needed
	for {
		var uidExists bool
		err := database.Pool.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM users WHERE unique_id = $1)", uniqueID).Scan(&uidExists)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "Database error",
			})
		}
		if !uidExists {
			break
		}
		uniqueID = utils.GenerateUniqueID(req.Name)
	}

	// Insert user into database
	var user models.User
	err = database.Pool.QueryRow(context.Background(), `
		INSERT INTO users (unique_id, email, name, password_hash, auth_provider, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, unique_id, email, name, auth_provider, is_online, last_seen, created_at, updated_at
	`, uniqueID, req.Email, req.Name, hashedPassword, "email", time.Now(), time.Now()).
		Scan(&user.ID, &user.UniqueID, &user.Email, &user.Name, &user.AuthProvider,
			&user.IsOnline, &user.LastSeen, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to create user",
		})
	}

	// Generate JWT access token
	token, err := utils.GenerateToken(user.ID, user.Email, user.UniqueID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to generate token",
		})
	}

	// Generate refresh token
	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.Email, user.UniqueID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to generate refresh token",
		})
	}

	// Set HTTP-Only Cookie for access token
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    token,
		HTTPOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: "Lax",
		MaxAge:   900, // 15 minutes
	})

	// Set HTTP-Only Cookie for refresh token
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HTTPOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: "Lax",
		MaxAge:   604800, // 7 days
	})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    user.ToResponse(),
	})
}

// Login handles user login
func Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Email and password are required",
		})
	}

	// Get user from database
	var user models.User
	err := database.Pool.QueryRow(context.Background(), `
		SELECT id, unique_id, email, name, password_hash, avatar, auth_provider, is_online, last_seen, created_at, updated_at
		FROM users WHERE email = $1
	`, req.Email).Scan(&user.ID, &user.UniqueID, &user.Email, &user.Name, &user.Password,
		&user.Avatar, &user.AuthProvider, &user.IsOnline, &user.LastSeen, &user.CreatedAt, &user.UpdatedAt)

	if err == pgx.ErrNoRows {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid email or password",
		})
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Database error",
		})
	}

	// Check if user registered with Google OAuth
	if user.AuthProvider == "google" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "This account uses Google login. Please sign in with Google.",
		})
	}

	// Verify password
	if !utils.CheckPassword(user.Password, req.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid email or password",
		})
	}

	// Update user online status
	_, err = database.Pool.Exec(context.Background(), "UPDATE users SET is_online = true, last_seen = $1 WHERE id = $2", time.Now(), user.ID)
	if err != nil {
		// Log error but don't fail the login
	}

	user.IsOnline = true

	// Generate JWT access token
	token, err := utils.GenerateToken(user.ID, user.Email, user.UniqueID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to generate token",
		})
	}

	// Generate refresh token
	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.Email, user.UniqueID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to generate refresh token",
		})
	}

	// Set HTTP-Only Cookie for access token
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    token,
		HTTPOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: "Lax",
		MaxAge:   900, // 15 minutes
	})

	// Set HTTP-Only Cookie for refresh token
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HTTPOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: "Lax",
		MaxAge:   604800, // 7 days
	})

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"user": user.ToResponse(),
		},
	})
}

// GetMe returns current authenticated user
func GetMe(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var user models.User
	err := database.Pool.QueryRow(context.Background(), `
		SELECT id, unique_id, email, name, avatar, auth_provider, is_online, last_seen, created_at, updated_at
		FROM users WHERE id = $1
	`, userID).Scan(&user.ID, &user.UniqueID, &user.Email, &user.Name, &user.Avatar,
		&user.AuthProvider, &user.IsOnline, &user.LastSeen, &user.CreatedAt, &user.UpdatedAt)

	if err == pgx.ErrNoRows {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "User not found",
		})
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Database error",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    user.ToResponse(),
	})
}

// Logout handles user logout
func Logout(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	// Update user offline status
	_, err := database.Pool.Exec(context.Background(), "UPDATE users SET is_online = false, last_seen = $1 WHERE id = $2", time.Now(), userID)
	if err != nil {
		// Log error but don't fail the logout
	}

	// Clear access token cookie
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    "",
		HTTPOnly: true,
		Secure:   false,
		SameSite: "Lax",
		MaxAge:   -1, // Delete cookie
	})

	// Clear refresh token cookie
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HTTPOnly: true,
		Secure:   false,
		SameSite: "Lax",
		MaxAge:   -1, // Delete cookie
	})

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Logged out successfully",
	})
}

// RefreshToken handles token refresh
func RefreshToken(c *fiber.Ctx) error {
	// Get refresh token from cookies
	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Refresh token not found",
		})
	}

	// Validate refresh token
	claims, err := utils.ValidateToken(refreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid refresh token",
		})
	}

	// Check if token type is refresh
	if claims.Type != "refresh" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid token type",
		})
	}

	// Generate new access token
	newAccessToken, err := utils.GenerateToken(claims.UserID, claims.Email, claims.UniqueID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to generate access token",
		})
	}

	// Generate new refresh token
	newRefreshToken, err := utils.GenerateRefreshToken(claims.UserID, claims.Email, claims.UniqueID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to generate refresh token",
		})
	}

	// Set HTTP-Only Cookie for new access token
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    newAccessToken,
		HTTPOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: "Lax",
		MaxAge:   900, // 15 minutes
	})

	// Set HTTP-Only Cookie for new refresh token
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshToken,
		HTTPOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: "Lax",
		MaxAge:   604800, // 7 days
	})

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Tokens refreshed successfully",
	})
}
