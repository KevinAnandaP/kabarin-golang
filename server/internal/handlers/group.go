package handlers

import (
	"context"
	"strconv"
	"time"

	"ngabarin/server/internal/database"
	"ngabarin/server/internal/models"
	ws "ngabarin/server/internal/websocket"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
)

// CreateGroupRequest represents create group request body
type CreateGroupRequest struct {
	Name      string   `json:"name"`
	Icon      string   `json:"icon,omitempty"`
	MemberIDs []string `json:"memberIds"`
}

// UpdateGroupRequest represents update group request body
type UpdateGroupRequest struct {
	Name string `json:"name,omitempty"`
	Icon string `json:"icon,omitempty"`
}

// AddMembersRequest represents add members request body
type AddMembersRequest struct {
	UserIDs []string `json:"userIds"`
}

// CreateGroup creates a new group
func CreateGroup(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var req CreateGroupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	// Validate input
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Group name is required",
		})
	}

	if len(req.MemberIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "At least one member is required",
		})
	}

	// Start transaction
	tx, err := database.Pool.Begin(context.Background())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Database error",
		})
	}
	defer tx.Rollback(context.Background())

	// Create group
	var group models.Group
	var icon *string
	if req.Icon != "" {
		icon = &req.Icon
	}

	err = tx.QueryRow(context.Background(), `
		INSERT INTO groups (name, icon, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, icon, created_by, created_at, updated_at
	`, req.Name, icon, userID, time.Now(), time.Now()).
		Scan(&group.ID, &group.Name, &group.Icon, &group.CreatedBy, &group.CreatedAt, &group.UpdatedAt)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to create group",
		})
	}

	// Add creator as member
	_, err = tx.Exec(context.Background(), `
		INSERT INTO group_members (group_id, user_id, joined_at)
		VALUES ($1, $2, $3)
	`, group.ID, userID, time.Now())

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to add creator to group",
		})
	}

	// Add other members
	for _, memberID := range req.MemberIDs {
		if memberID == userID {
			continue // Skip creator, already added
		}

		_, err = tx.Exec(context.Background(), `
			INSERT INTO group_members (group_id, user_id, joined_at)
			VALUES ($1, $2, $3)
		`, group.ID, memberID, time.Now())

		if err != nil {
			continue // Skip if user doesn't exist
		}
	}

	// Commit transaction
	if err := tx.Commit(context.Background()); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to commit transaction",
		})
	}

	// Get members
	members, _ := getGroupMembers(group.ID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data": models.GroupWithMembers{
			ID:        group.ID,
			Name:      group.Name,
			Icon:      group.Icon,
			CreatedBy: group.CreatedBy,
			Members:   members,
			CreatedAt: group.CreatedAt,
			UpdatedAt: group.UpdatedAt,
		},
	})
}

// GetGroups returns all groups for current user
func GetGroups(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	// Pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset := (page - 1) * limit

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Get total count
	var total int
	err := database.Pool.QueryRow(context.Background(), `
		SELECT COUNT(DISTINCT g.id)
		FROM groups g
		INNER JOIN group_members gm ON g.id = gm.group_id
		WHERE gm.user_id = $1
	`, userID).Scan(&total)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Database error",
		})
	}

	// Get groups with last message and member count
	rows, err := database.Pool.Query(context.Background(), `
		SELECT 
			g.id, g.name, g.icon, g.created_by, g.created_at, g.updated_at,
			(SELECT COUNT(*) FROM group_members WHERE group_id = g.id) as member_count
		FROM groups g
		INNER JOIN group_members gm ON g.id = gm.group_id
		WHERE gm.user_id = $1
		ORDER BY g.updated_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Database error",
		})
	}
	defer rows.Close()

	var groups []fiber.Map

	for rows.Next() {
		var group models.Group
		var memberCount int

		err := rows.Scan(
			&group.ID, &group.Name, &group.Icon, &group.CreatedBy,
			&group.CreatedAt, &group.UpdatedAt, &memberCount,
		)

		if err != nil {
			continue
		}

		// Get last message (optional, can be slow for many groups)
		var lastMessage *fiber.Map
		var msgContent, senderName string
		var msgCreatedAt time.Time
		err = database.Pool.QueryRow(context.Background(), `
			SELECT m.content, u.name, m.created_at
			FROM messages m
			INNER JOIN users u ON m.sender_id = u.id
			WHERE m.group_id = $1
			ORDER BY m.created_at DESC
			LIMIT 1
		`, group.ID).Scan(&msgContent, &senderName, &msgCreatedAt)

		if err == nil {
			lastMessage = &fiber.Map{
				"content":   msgContent,
				"createdAt": msgCreatedAt,
				"sender": fiber.Map{
					"name": senderName,
				},
			}
		}

		// TODO: Get unread count (requires tracking read status per user per group)
		unreadCount := 0

		groups = append(groups, fiber.Map{
			"id":          group.ID,
			"name":        group.Name,
			"icon":        group.Icon,
			"createdBy":   group.CreatedBy,
			"memberCount": memberCount,
			"lastMessage": lastMessage,
			"unreadCount": unreadCount,
			"createdAt":   group.CreatedAt,
		})
	}

	if groups == nil {
		groups = []fiber.Map{}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    groups,
		"pagination": fiber.Map{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// GetGroupDetails returns group details with members
func GetGroupDetails(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	groupID := c.Params("groupId")

	// Check if user is member
	var isMember bool
	err := database.Pool.QueryRow(context.Background(), `
		SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)
	`, groupID, userID).Scan(&isMember)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Database error",
		})
	}

	if !isMember {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "You are not a member of this group",
		})
	}

	// Get group info
	var group models.Group
	err = database.Pool.QueryRow(context.Background(), `
		SELECT id, name, icon, created_by, created_at, updated_at
		FROM groups WHERE id = $1
	`, groupID).Scan(&group.ID, &group.Name, &group.Icon, &group.CreatedBy, &group.CreatedAt, &group.UpdatedAt)

	if err == pgx.ErrNoRows {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Group not found",
		})
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Database error",
		})
	}

	// Get members
	members, err := getGroupMembers(groupID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to get group members",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": models.GroupWithMembers{
			ID:        group.ID,
			Name:      group.Name,
			Icon:      group.Icon,
			CreatedBy: group.CreatedBy,
			Members:   members,
			CreatedAt: group.CreatedAt,
			UpdatedAt: group.UpdatedAt,
		},
	})
}

// Helper function to get group members
func getGroupMembers(groupID string) ([]models.UserResponse, error) {
	rows, err := database.Pool.Query(context.Background(), `
		SELECT u.id, u.unique_id, u.email, u.name, u.avatar, u.auth_provider, u.is_online, u.last_seen, u.created_at
		FROM users u
		INNER JOIN group_members gm ON u.id = gm.user_id
		WHERE gm.group_id = $1
		ORDER BY gm.joined_at ASC
	`, groupID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []models.UserResponse

	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.UniqueID, &user.Email, &user.Name, &user.Avatar,
			&user.AuthProvider, &user.IsOnline, &user.LastSeen, &user.CreatedAt)

		if err != nil {
			continue
		}

		members = append(members, user.ToResponse())
	}

	if members == nil {
		members = []models.UserResponse{}
	}

	return members, nil
}

// UpdateGroup updates group name or icon
func UpdateGroup(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	groupID := c.Params("groupId")

	var req UpdateGroupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	// Check if user is creator
	var createdBy string
	err := database.Pool.QueryRow(context.Background(), "SELECT created_by FROM groups WHERE id = $1", groupID).Scan(&createdBy)

	if err == pgx.ErrNoRows {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Group not found",
		})
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Database error",
		})
	}

	if createdBy != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "Only group creator can update group",
		})
	}

	// Update group
	query := "UPDATE groups SET updated_at = $1"
	args := []interface{}{time.Now()}
	argCount := 2

	if req.Name != "" {
		query += ", name = $" + strconv.Itoa(argCount)
		args = append(args, req.Name)
		argCount++
	}

	if req.Icon != "" {
		query += ", icon = $" + strconv.Itoa(argCount)
		args = append(args, req.Icon)
		argCount++
	}

	query += " WHERE id = $" + strconv.Itoa(argCount) + " RETURNING id, name, icon, updated_at"
	args = append(args, groupID)

	var group models.Group
	err = database.Pool.QueryRow(context.Background(), query, args...).Scan(&group.ID, &group.Name, &group.Icon, &group.UpdatedAt)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to update group",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"id":        group.ID,
			"name":      group.Name,
			"icon":      group.Icon,
			"updatedAt": group.UpdatedAt,
		},
	})
}

// AddGroupMembers adds new members to group
func AddGroupMembers(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	groupID := c.Params("groupId")

	var req AddMembersRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	// Check if user is member
	var isMember bool
	err := database.Pool.QueryRow(context.Background(), `
		SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)
	`, groupID, userID).Scan(&isMember)

	if err != nil || !isMember {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "You are not a member of this group",
		})
	}

	addedCount := 0
	var addedMembers []models.UserResponse

	for _, memberID := range req.UserIDs {
		// Check if already member
		var exists bool
		database.Pool.QueryRow(context.Background(), `
			SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)
		`, groupID, memberID).Scan(&exists)

		if exists {
			continue
		}

		// Add member
		_, err := database.Pool.Exec(context.Background(), `
			INSERT INTO group_members (group_id, user_id, joined_at)
			VALUES ($1, $2, $3)
		`, groupID, memberID, time.Now())

		if err != nil {
			continue
		}

		// Get user info
		var user models.User
		err = database.Pool.QueryRow(context.Background(), `
			SELECT id, unique_id, name FROM users WHERE id = $1
		`, memberID).Scan(&user.ID, &user.UniqueID, &user.Name)

		if err == nil {
			addedCount++
			addedMembers = append(addedMembers, models.UserResponse{
				ID:       user.ID,
				UniqueID: user.UniqueID,
				Name:     user.Name,
			})
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Members added successfully",
		"data": fiber.Map{
			"addedCount": addedCount,
			"members":    addedMembers,
		},
	})
}

// RemoveGroupMember removes a member from group
func RemoveGroupMember(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	groupID := c.Params("groupId")
	memberID := c.Params("userId")

	// Check if user is creator or removing self
	var createdBy string
	err := database.Pool.QueryRow(context.Background(), "SELECT created_by FROM groups WHERE id = $1", groupID).Scan(&createdBy)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Group not found",
		})
	}

	// Can only remove if creator or removing self
	if createdBy != userID && memberID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "Only group creator can remove members",
		})
	}

	// Remove member
	result, err := database.Pool.Exec(context.Background(), `
		DELETE FROM group_members WHERE group_id = $1 AND user_id = $2
	`, groupID, memberID)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to remove member",
		})
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Member not found in group",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Member removed successfully",
	})
}

// LeaveGroup allows user to leave a group
func LeaveGroup(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	groupID := c.Params("groupId")

	// Get creator info
	var createdBy string
	err := database.Pool.QueryRow(context.Background(), "SELECT created_by FROM groups WHERE id = $1", groupID).Scan(&createdBy)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Group not found",
		})
	}

	// If creator is leaving, transfer ownership to first member
	if createdBy == userID {
		var newCreatorID string
		err = database.Pool.QueryRow(context.Background(), `
			SELECT user_id FROM group_members 
			WHERE group_id = $1 AND user_id != $2 
			ORDER BY joined_at ASC LIMIT 1
		`, groupID, userID).Scan(&newCreatorID)

		if err == nil {
			// Transfer ownership
			database.Pool.Exec(context.Background(), "UPDATE groups SET created_by = $1 WHERE id = $2", newCreatorID, groupID)
		}
	}

	// Remove member
	_, err = database.Pool.Exec(context.Background(), `
		DELETE FROM group_members WHERE group_id = $1 AND user_id = $2
	`, groupID, userID)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to leave group",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "You have left the group",
	})
}

// DeleteGroup deletes a group (creator only)
func DeleteGroup(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	groupID := c.Params("groupId")

	// Check if user is creator
	var createdBy string
	err := database.Pool.QueryRow(context.Background(), "SELECT created_by FROM groups WHERE id = $1", groupID).Scan(&createdBy)

	if err == pgx.ErrNoRows {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Group not found",
		})
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Database error",
		})
	}

	if createdBy != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "Only group creator can delete group",
		})
	}

	// Delete group (cascade will delete members and messages)
	_, err = database.Pool.Exec(context.Background(), "DELETE FROM groups WHERE id = $1", groupID)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to delete group",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Group deleted successfully",
	})
}

// SendGroupMessage sends a message to a group
func SendGroupMessage(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var req struct {
		GroupID string `json:"groupId"`
		Content string `json:"content"`
		Type    string `json:"type"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	// Validate input
	if req.GroupID == "" || req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Group ID and content are required",
		})
	}

	if req.Type == "" {
		req.Type = "text"
	}

	// Check if user is member
	var isMember bool
	err := database.Pool.QueryRow(context.Background(), `
		SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)
	`, req.GroupID, userID).Scan(&isMember)

	if err != nil || !isMember {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "You are not a member of this group",
		})
	}

	// Insert message
	var message models.Message
	err = database.Pool.QueryRow(context.Background(), `
		INSERT INTO messages (sender_id, group_id, content, type, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, sender_id, group_id, content, type, status, created_at, updated_at
	`, userID, req.GroupID, req.Content, req.Type, "sent", time.Now(), time.Now()).
		Scan(&message.ID, &message.SenderID, &message.GroupID, &message.Content,
			&message.Type, &message.Status, &message.CreatedAt, &message.UpdatedAt)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to send message",
		})
	}

	// Broadcast message via WebSocket to all group members
	if WSHub != nil {
		wsMessage := ws.WSMessage{
			Type: ws.EventGroupMessageReceived,
			Payload: ws.GroupMessagePayload{
				ID:        message.ID,
				GroupID:   req.GroupID,
				SenderID:  message.SenderID,
				Content:   message.Content,
				Type:      message.Type,
				CreatedAt: message.CreatedAt,
			},
			Timestamp: time.Now(),
		}
		// Broadcast to all group members except sender
		WSHub.BroadcastToGroup(req.GroupID, wsMessage, userID)

		// Also send to sender for confirmation
		confirmMessage := ws.WSMessage{
			Type: ws.EventGroupMessageSent,
			Payload: ws.GroupMessagePayload{
				ID:        message.ID,
				GroupID:   req.GroupID,
				SenderID:  message.SenderID,
				Content:   message.Content,
				Type:      message.Type,
				CreatedAt: message.CreatedAt,
			},
			Timestamp: time.Now(),
		}
		WSHub.BroadcastToUser(userID, confirmMessage)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    message,
	})
}

// GetGroupMessages returns messages in a group
func GetGroupMessages(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	groupID := c.Params("groupId")

	// Check if user is member
	var isMember bool
	err := database.Pool.QueryRow(context.Background(), `
		SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)
	`, groupID, userID).Scan(&isMember)

	if err != nil || !isMember {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "You are not a member of this group",
		})
	}

	// Pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset := (page - 1) * limit

	// Get total count
	var total int
	err = database.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM messages WHERE group_id = $1
	`, groupID).Scan(&total)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Database error",
		})
	}

	// Get messages with sender info
	rows, err := database.Pool.Query(context.Background(), `
		SELECT 
			m.id, m.sender_id, m.group_id, m.content, m.type, m.created_at,
			u.id, u.unique_id, u.name, u.avatar
		FROM messages m
		INNER JOIN users u ON m.sender_id = u.id
		WHERE m.group_id = $1
		ORDER BY m.created_at DESC
		LIMIT $2 OFFSET $3
	`, groupID, limit, offset)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Database error",
		})
	}
	defer rows.Close()

	var messages []fiber.Map

	for rows.Next() {
		var msgID, senderID, groupID, content, msgType string
		var createdAt time.Time
		var userID, uniqueID, name string
		var avatar *string

		err := rows.Scan(
			&msgID, &senderID, &groupID, &content, &msgType, &createdAt,
			&userID, &uniqueID, &name, &avatar,
		)

		if err != nil {
			continue
		}

		messages = append(messages, fiber.Map{
			"id":      msgID,
			"groupId": groupID,
			"content": content,
			"type":    msgType,
			"sender": fiber.Map{
				"id":       userID,
				"uniqueId": uniqueID,
				"name":     name,
				"avatar":   avatar,
			},
			"createdAt": createdAt,
		})
	}

	if messages == nil {
		messages = []fiber.Map{}
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
