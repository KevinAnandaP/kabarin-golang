package handlers

import (
	"context"
	"strconv"
	"time"

	"ngabarin/server/internal/database"
	"ngabarin/server/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
)

// SendMessageRequest represents send message request body
type SendMessageRequest struct {
	ReceiverID string `json:"receiverId"`
	Content    string `json:"content"`
	Type       string `json:"type"` // text, image, file
}

// MarkReadRequest represents mark as read request body
type MarkReadRequest struct {
	MessageIDs []string `json:"messageIds,omitempty"`
	SenderID   string   `json:"senderId,omitempty"`
	ChatID     string   `json:"chatId,omitempty"`
}

// SendMessage sends a direct message
func SendMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var req SendMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	// Validate input
	if req.ReceiverID == "" || req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Receiver ID and content are required",
		})
	}

	// Set default type
	if req.Type == "" {
		req.Type = "text"
	}

	// Validate message type
	if req.Type != "text" && req.Type != "image" && req.Type != "file" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid message type. Must be text, image, or file",
		})
	}

	// Check if receiver exists
	var receiverExists bool
	err := database.Pool.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", req.ReceiverID).Scan(&receiverExists)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Database error",
		})
	}

	if !receiverExists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Receiver not found",
		})
	}

	// Insert message
	var message models.Message
	err = database.Pool.QueryRow(context.Background(), `
		INSERT INTO messages (sender_id, receiver_id, content, type, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, sender_id, receiver_id, content, type, status, created_at, updated_at
	`, userID, req.ReceiverID, req.Content, req.Type, "sent", time.Now(), time.Now()).
		Scan(&message.ID, &message.SenderID, &message.ReceiverID, &message.Content,
			&message.Type, &message.Status, &message.CreatedAt, &message.UpdatedAt)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to send message",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    message,
	})
}

// GetMessages returns message history between two users
func GetMessages(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	chatID := c.Params("chatId") // chatId is the other user's ID

	// Pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset := (page - 1) * limit

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	// Get total count
	var total int
	err := database.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM messages
		WHERE (sender_id = $1 AND receiver_id = $2) OR (sender_id = $2 AND receiver_id = $1)
	`, userID, chatID).Scan(&total)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Database error",
		})
	}

	// Get messages with sender info
	rows, err := database.Pool.Query(context.Background(), `
		SELECT 
			m.id, m.sender_id, m.receiver_id, m.content, m.type, m.status, m.created_at, m.updated_at,
			u.id, u.unique_id, u.email, u.name, u.avatar, u.auth_provider, u.is_online, u.last_seen, u.created_at, u.updated_at
		FROM messages m
		INNER JOIN users u ON m.sender_id = u.id
		WHERE (m.sender_id = $1 AND m.receiver_id = $2) OR (m.sender_id = $2 AND m.receiver_id = $1)
		ORDER BY m.created_at DESC
		LIMIT $3 OFFSET $4
	`, userID, chatID, limit, offset)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Database error",
		})
	}
	defer rows.Close()

	var messages []models.MessageWithSender

	for rows.Next() {
		var message models.Message
		var sender models.User

		err := rows.Scan(
			&message.ID, &message.SenderID, &message.ReceiverID, &message.Content,
			&message.Type, &message.Status, &message.CreatedAt, &message.UpdatedAt,
			&sender.ID, &sender.UniqueID, &sender.Email, &sender.Name, &sender.Avatar,
			&sender.AuthProvider, &sender.IsOnline, &sender.LastSeen, &sender.CreatedAt, &sender.UpdatedAt,
		)

		if err != nil {
			continue
		}

		messages = append(messages, models.MessageWithSender{
			ID:         message.ID,
			Sender:     sender.ToResponse(),
			ReceiverID: message.ReceiverID,
			GroupID:    message.GroupID,
			Content:    message.Content,
			Type:       message.Type,
			Status:     message.Status,
			CreatedAt:  message.CreatedAt,
			UpdatedAt:  message.UpdatedAt,
		})
	}

	if messages == nil {
		messages = []models.MessageWithSender{}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"messages": messages,
			"pagination": fiber.Map{
				"page":  page,
				"limit": limit,
				"total": total,
			},
		},
	})
}

// MarkAsRead marks messages as read
func MarkAsRead(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var req MarkReadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	var result interface{}
	var err error

	// Mark specific messages
	if len(req.MessageIDs) > 0 {
		query := `UPDATE messages SET status = 'read', updated_at = $1 
				  WHERE receiver_id = $2 AND id = ANY($3)`
		result, err = database.Pool.Exec(context.Background(), query, time.Now(), userID, req.MessageIDs)
	} else if req.SenderID != "" {
		// Mark all messages from a specific sender
		query := `UPDATE messages SET status = 'read', updated_at = $1 
				  WHERE receiver_id = $2 AND sender_id = $3 AND status != 'read'`
		result, err = database.Pool.Exec(context.Background(), query, time.Now(), userID, req.SenderID)
	} else if req.ChatID != "" {
		// Mark all messages in a chat (same as SenderID for direct messages)
		query := `UPDATE messages SET status = 'read', updated_at = $1 
				  WHERE receiver_id = $2 AND sender_id = $3 AND status != 'read'`
		result, err = database.Pool.Exec(context.Background(), query, time.Now(), userID, req.ChatID)
	} else {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Either messageIds, senderId, or chatId must be provided",
		})
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to mark messages as read",
		})
	}

	rowsAffected := int64(0)
	if cmdTag, ok := result.(interface{ RowsAffected() int64 }); ok {
		rowsAffected = cmdTag.RowsAffected()
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Messages marked as read",
		"data": fiber.Map{
			"updatedCount": rowsAffected,
		},
	})
}

// UpdateMessageStatus updates a single message status
func UpdateMessageStatus(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	messageID := c.Params("messageId")

	var req struct {
		Status string `json:"status"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	// Validate status
	if req.Status != "sent" && req.Status != "delivered" && req.Status != "read" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid status. Must be sent, delivered, or read",
		})
	}

	// Update message status (only if user is the receiver)
	var message models.Message
	err := database.Pool.QueryRow(context.Background(), `
		UPDATE messages 
		SET status = $1, updated_at = $2
		WHERE id = $3 AND receiver_id = $4
		RETURNING id, status, updated_at
	`, req.Status, time.Now(), messageID, userID).Scan(&message.ID, &message.Status, &message.UpdatedAt)

	if err == pgx.ErrNoRows {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Message not found or you don't have permission",
		})
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to update message status",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"id":        message.ID,
			"status":    message.Status,
			"updatedAt": message.UpdatedAt,
		},
	})
}
