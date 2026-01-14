package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"ngabarin/server/internal/database"
	"ngabarin/server/internal/models"

	"github.com/gofiber/fiber/v2"
)

// ChatListItem represents a chat in the list with contact status
type ChatListItem struct {
	ID          string              `json:"id"`
	User        models.UserResponse `json:"user"`
	IsContact   bool                `json:"isContact"`
	IsOnline    bool                `json:"isOnline"`
	LastMessage *struct {
		Content   string `json:"content"`
		CreatedAt string `json:"createdAt"`
	} `json:"lastMessage,omitempty"`
}

// GetChats returns all chats (contacts + people who messaged the user)
func GetChats(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	// Query to get all unique users: contacts + people who messaged the user
	rows, err := database.Pool.Query(context.Background(), `
		WITH chat_list AS (
			-- Get contacts with their latest messages
			SELECT 
				c.contact_id as user_id,
				m.created_at as last_message_at,
				m.content as last_message_content,
				TRUE as is_contact
			FROM contacts c
			LEFT JOIN LATERAL (
				SELECT created_at, content
				FROM messages
				WHERE (sender_id = $1 AND receiver_id = c.contact_id)
				   OR (sender_id = c.contact_id AND receiver_id = $1)
				ORDER BY created_at DESC
				LIMIT 1
			) m ON TRUE
			WHERE c.user_id = $1
			
			UNION
			
			-- Get non-contacts who have messaged the user
			SELECT 
				msg.user_id,
				msg.last_message_at,
				msg.last_message_content,
				FALSE as is_contact
			FROM (
				SELECT DISTINCT ON (
					CASE 
						WHEN sender_id = $1 THEN receiver_id 
						ELSE sender_id 
					END
				)
					CASE 
						WHEN sender_id = $1 THEN receiver_id 
						ELSE sender_id 
					END as user_id,
					created_at as last_message_at,
					content as last_message_content
				FROM messages
				WHERE sender_id = $1 OR receiver_id = $1
				ORDER BY 
					CASE 
						WHEN sender_id = $1 THEN receiver_id 
						ELSE sender_id 
					END,
					created_at DESC
			) msg
			WHERE NOT EXISTS (
				SELECT 1 FROM contacts WHERE user_id = $1 AND contact_id = msg.user_id
			)
		)
		SELECT 
			u.id, u.unique_id, u.email, u.name, u.avatar, u.auth_provider, 
			u.is_online, u.last_seen, u.created_at, u.updated_at,
			cl.is_contact,
			cl.last_message_at,
			cl.last_message_content
		FROM chat_list cl
		INNER JOIN users u ON cl.user_id = u.id
		ORDER BY cl.last_message_at DESC NULLS LAST
	`, userID)

	if err != nil {
		log.Printf("GetChats query error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   fmt.Sprintf("Database error: %v", err),
		})
	}
	defer rows.Close()

	var chats []ChatListItem

	for rows.Next() {
		var user models.User
		var isContact bool
		var lastMessageAt *time.Time
		var lastMessageContent *string

		err := rows.Scan(
			&user.ID, &user.UniqueID, &user.Email, &user.Name, &user.Avatar,
			&user.AuthProvider, &user.IsOnline, &user.LastSeen, &user.CreatedAt, &user.UpdatedAt,
			&isContact, &lastMessageAt, &lastMessageContent,
		)

		if err != nil {
			continue
		}

		chatItem := ChatListItem{
			ID:        user.ID,
			User:      user.ToResponse(),
			IsContact: isContact,
			IsOnline:  user.IsOnline,
		}

		// Add last message if exists
		if lastMessageContent != nil && lastMessageAt != nil {
			chatItem.LastMessage = &struct {
				Content   string `json:"content"`
				CreatedAt string `json:"createdAt"`
			}{
				Content:   *lastMessageContent,
				CreatedAt: lastMessageAt.Format(time.RFC3339),
			}
		}

		chats = append(chats, chatItem)
	}

	if chats == nil {
		chats = []ChatListItem{}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    chats,
	})
}
