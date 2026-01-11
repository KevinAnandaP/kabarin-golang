package handlers

import (
	"context"

	"ngabarin/server/internal/database"
	"ngabarin/server/internal/models"
	"ngabarin/server/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
)

// AddContactRequest represents add contact request body
type AddContactRequest struct {
	UniqueID string `json:"uniqueId"`
}

// AddContact adds a new contact by unique ID
func AddContact(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var req AddContactRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	// Validate unique ID format
	if !utils.ValidateUniqueID(req.UniqueID) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid unique ID format. Should be like #WORD-123",
		})
	}

	// Get current user's unique ID to prevent self-adding
	var currentUserUniqueID string
	err := database.Pool.QueryRow(context.Background(), "SELECT unique_id FROM users WHERE id = $1", userID).Scan(&currentUserUniqueID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Database error",
		})
	}

	// Check if trying to add self
	if req.UniqueID == currentUserUniqueID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "You cannot add yourself as a contact",
		})
	}

	// Find user by unique ID
	var contactUser models.User
	err = database.Pool.QueryRow(context.Background(), `
		SELECT id, unique_id, email, name, avatar, auth_provider, is_online, last_seen, created_at, updated_at
		FROM users WHERE unique_id = $1
	`, req.UniqueID).Scan(&contactUser.ID, &contactUser.UniqueID, &contactUser.Email,
		&contactUser.Name, &contactUser.Avatar, &contactUser.AuthProvider,
		&contactUser.IsOnline, &contactUser.LastSeen, &contactUser.CreatedAt, &contactUser.UpdatedAt)

	if err == pgx.ErrNoRows {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "User with this unique ID not found",
		})
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Database error",
		})
	}

	// Check if contact already exists
	var exists bool
	err = database.Pool.QueryRow(context.Background(), `
		SELECT EXISTS(SELECT 1 FROM contacts WHERE user_id = $1 AND contact_id = $2)
	`, userID, contactUser.ID).Scan(&exists)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Database error",
		})
	}

	if exists {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"success": false,
			"error":   "Contact already added",
		})
	}

	// Add contact
	var contact models.Contact
	err = database.Pool.QueryRow(context.Background(), `
		INSERT INTO contacts (user_id, contact_id)
		VALUES ($1, $2)
		RETURNING id, user_id, contact_id, added_at
	`, userID, contactUser.ID).Scan(&contact.ID, &contact.UserID, &contact.ContactID, &contact.AddedAt)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to add contact",
		})
	}

	// Return contact with user info
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data": models.ContactWithUser{
			ID:       contact.ID,
			UserID:   contact.UserID,
			Contact:  contactUser.ToResponse(),
			AddedAt:  contact.AddedAt,
			IsOnline: contactUser.IsOnline,
		},
	})
}

// GetContacts returns all contacts for current user
func GetContacts(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	rows, err := database.Pool.Query(context.Background(), `
		SELECT 
			c.id, c.user_id, c.added_at,
			u.id, u.unique_id, u.email, u.name, u.avatar, u.auth_provider, u.is_online, u.last_seen, u.created_at, u.updated_at
		FROM contacts c
		INNER JOIN users u ON c.contact_id = u.id
		WHERE c.user_id = $1
		ORDER BY c.added_at DESC
	`, userID)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Database error",
		})
	}
	defer rows.Close()

	var contacts []models.ContactWithUser

	for rows.Next() {
		var contact models.Contact
		var user models.User

		err := rows.Scan(
			&contact.ID, &contact.UserID, &contact.AddedAt,
			&user.ID, &user.UniqueID, &user.Email, &user.Name, &user.Avatar,
			&user.AuthProvider, &user.IsOnline, &user.LastSeen, &user.CreatedAt, &user.UpdatedAt,
		)

		if err != nil {
			continue
		}

		contacts = append(contacts, models.ContactWithUser{
			ID:       contact.ID,
			UserID:   contact.UserID,
			Contact:  user.ToResponse(),
			AddedAt:  contact.AddedAt,
			IsOnline: user.IsOnline,
		})
	}

	if contacts == nil {
		contacts = []models.ContactWithUser{}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    contacts,
	})
}

// SearchContacts searches contacts by name or unique ID
func SearchContacts(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	query := c.Query("q", "")

	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Search query is required",
		})
	}

	rows, err := database.Pool.Query(context.Background(), `
		SELECT 
			c.id, c.user_id, c.added_at,
			u.id, u.unique_id, u.email, u.name, u.avatar, u.auth_provider, u.is_online, u.last_seen, u.created_at, u.updated_at
		FROM contacts c
		INNER JOIN users u ON c.contact_id = u.id
		WHERE c.user_id = $1 
		AND (u.name ILIKE $2 OR u.unique_id ILIKE $2)
		ORDER BY c.added_at DESC
	`, userID, "%"+query+"%")

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Database error",
		})
	}
	defer rows.Close()

	var contacts []models.ContactWithUser

	for rows.Next() {
		var contact models.Contact
		var user models.User

		err := rows.Scan(
			&contact.ID, &contact.UserID, &contact.AddedAt,
			&user.ID, &user.UniqueID, &user.Email, &user.Name, &user.Avatar,
			&user.AuthProvider, &user.IsOnline, &user.LastSeen, &user.CreatedAt, &user.UpdatedAt,
		)

		if err != nil {
			continue
		}

		contacts = append(contacts, models.ContactWithUser{
			ID:       contact.ID,
			UserID:   contact.UserID,
			Contact:  user.ToResponse(),
			AddedAt:  contact.AddedAt,
			IsOnline: user.IsOnline,
		})
	}

	if contacts == nil {
		contacts = []models.ContactWithUser{}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    contacts,
	})
}

// RemoveContact removes a contact
func RemoveContact(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	contactID := c.Params("contactId")

	if contactID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Contact ID is required",
		})
	}

	// Check if contact exists
	var exists bool
	err := database.Pool.QueryRow(context.Background(), `
		SELECT EXISTS(SELECT 1 FROM contacts WHERE id = $1 AND user_id = $2)
	`, contactID, userID).Scan(&exists)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Database error",
		})
	}

	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Contact not found",
		})
	}

	// Delete contact
	_, err = database.Pool.Exec(context.Background(), "DELETE FROM contacts WHERE id = $1 AND user_id = $2", contactID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to remove contact",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Contact removed successfully",
	})
}
