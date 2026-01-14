package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"ngabarin/server/internal/database"
	"ngabarin/server/internal/models"
	"ngabarin/server/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
)

// GoogleOAuthURL generates the Google OAuth URL
func GoogleOAuthURL(c *fiber.Ctx) error {
	// Get OAuth config from environment
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleRedirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

	if googleClientID == "" || googleRedirectURL == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Google OAuth not configured",
		})
	}

	// Generate state token for CSRF protection
	state := generateStateToken()

	// Store state in cookie for verification
	c.Cookie(&fiber.Cookie{
		Name:     "oauth_state",
		Value:    state,
		HTTPOnly: true,
		Secure:   false,
		SameSite: "Lax",
		MaxAge:   300, // 5 minutes
	})

	// Build OAuth URL
	oauthURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=openid email profile&state=%s",
		googleClientID,
		googleRedirectURL,
		state,
	)

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"url": oauthURL,
		},
	})
}

// GoogleOAuthCallback handles the OAuth callback
func GoogleOAuthCallback(c *fiber.Ctx) error {
	// Get state from cookie
	cookieState := c.Cookies("oauth_state")
	queryState := c.Query("state")

	// Verify state token
	if cookieState == "" || cookieState != queryState {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid state parameter",
		})
	}

	// Clear state cookie
	c.Cookie(&fiber.Cookie{
		Name:     "oauth_state",
		Value:    "",
		HTTPOnly: true,
		MaxAge:   -1,
	})

	// Get authorization code
	code := c.Query("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Authorization code not found",
		})
	}

	// Exchange code for access token
	tokenData, err := exchangeCodeForToken(code)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to exchange code for token",
		})
	}

	// Get user info from Google
	googleUser, err := getGoogleUserInfo(tokenData.AccessToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to get user info",
		})
	}

	// Check if user exists
	var user models.User
	selectQuery := `
		SELECT id, unique_id, email, name, avatar, auth_provider, google_id, is_online, last_seen, created_at, updated_at
		FROM users WHERE email = $1
	`
	err = database.Pool.QueryRow(context.Background(), selectQuery, googleUser.Email).
		Scan(&user.ID, &user.UniqueID, &user.Email, &user.Name, &user.Avatar,
			&user.AuthProvider, &user.GoogleID, &user.IsOnline, &user.LastSeen, &user.CreatedAt, &user.UpdatedAt)

	if err == pgx.ErrNoRows {
		// Create new user
		uniqueID := utils.GenerateUniqueID(googleUser.Name)

		// Check if uniqueID exists and regenerate if needed
		// Use a single query with a generated unique_id to avoid prepared statement conflicts
		var uidExists bool
		checkQuery := "SELECT EXISTS(SELECT 1 FROM users WHERE unique_id = $1)"
		for {
			err := database.Pool.QueryRow(context.Background(), checkQuery, uniqueID).Scan(&uidExists)
			if err != nil {
				log.Printf("Failed to check unique_id existence: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"success": false,
					"error":   fmt.Sprintf("Database error: %v", err),
				})
			}
			if !uidExists {
				break
			}
			uniqueID = utils.GenerateUniqueID(googleUser.Name)
		}

		// Insert new user
		insertQuery := `
			INSERT INTO users (unique_id, email, name, avatar, auth_provider, google_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id, unique_id, email, name, avatar, auth_provider, google_id, is_online, last_seen, created_at, updated_at
		`
		err = database.Pool.QueryRow(context.Background(), insertQuery,
			uniqueID, googleUser.Email, googleUser.Name, googleUser.Picture, "google", googleUser.Sub, time.Now(), time.Now()).
			Scan(&user.ID, &user.UniqueID, &user.Email, &user.Name, &user.Avatar,
				&user.AuthProvider, &user.GoogleID, &user.IsOnline, &user.LastSeen, &user.CreatedAt, &user.UpdatedAt)

		if err != nil {
			log.Printf("Failed to create user: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   fmt.Sprintf("Failed to create user: %v", err),
			})
		}
	} else if err != nil {
		log.Printf("Database error while checking user: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   fmt.Sprintf("Database error: %v", err),
		})
	}

	// Update online status
	_, _ = database.Pool.Exec(context.Background(), "UPDATE users SET is_online = true, last_seen = $1 WHERE id = $2", time.Now(), user.ID)
	user.IsOnline = true

	// Generate JWT tokens
	token, err := utils.GenerateToken(user.ID, user.Email, user.UniqueID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to generate token",
		})
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.Email, user.UniqueID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to generate refresh token",
		})
	}

	// Set cookies
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    token,
		HTTPOnly: true,
		Secure:   false,
		SameSite: "Lax",
		MaxAge:   900, // 15 minutes
	})

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HTTPOnly: true,
		Secure:   false,
		SameSite: "Lax",
		MaxAge:   604800, // 7 days
	})

	// Redirect to frontend
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	return c.Redirect(frontendURL + "/chat")
}

// TokenResponse represents Google OAuth token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// GoogleUser represents user info from Google
type GoogleUser struct {
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

// exchangeCodeForToken exchanges authorization code for access token
func exchangeCodeForToken(code string) (*TokenResponse, error) {
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	googleRedirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

	tokenURL := "https://oauth2.googleapis.com/token"

	data := fmt.Sprintf(
		"code=%s&client_id=%s&client_secret=%s&redirect_uri=%s&grant_type=authorization_code",
		code, googleClientID, googleClientSecret, googleRedirectURL,
	)

	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get token, status: %d", resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

// getGoogleUserInfo gets user information from Google
func getGoogleUserInfo(accessToken string) (*GoogleUser, error) {
	userInfoURL := "https://www.googleapis.com/oauth2/v2/userinfo"

	req, err := http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info, status: %d, body: %s", resp.StatusCode, string(body))
	}

	var googleUser GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		return nil, err
	}

	return &googleUser, nil
}

// generateStateToken generates a random state token
func generateStateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
