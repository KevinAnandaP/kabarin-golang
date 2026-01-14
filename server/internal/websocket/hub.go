package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"ngabarin/server/internal/database"
)

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients mapped by user ID
	Clients map[string]*Client

	// Register requests from clients
	Register chan *Client

	// Unregister requests from clients
	Unregister chan *Client

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[string]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.registerClient(client)
		case client := <-h.Unregister:
			h.unregisterClient(client)
		}
	}
}

// registerClient adds a client to the hub
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// If user already has a connection, close the old one
	if existingClient, ok := h.Clients[client.ID]; ok {
		close(existingClient.Send)
	}

	h.Clients[client.ID] = client

	// Update user's online status in database
	_, err := database.Pool.Exec(context.Background(), `
		UPDATE users SET is_online = true, last_seen = $1 WHERE id = $2
	`, time.Now(), client.ID)

	if err != nil {
		log.Printf("Failed to update online status: %v", err)
	}

	// Broadcast user online status to their contacts
	h.broadcastPresence(client.ID, true)

	log.Printf("Client connected: %s (%s)", client.UniqueID, client.ID)
}

// unregisterClient removes a client from the hub
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.Clients[client.ID]; ok {
		delete(h.Clients, client.ID)
		close(client.Send)

		// Update user's offline status in database
		_, err := database.Pool.Exec(context.Background(), `
			UPDATE users SET is_online = false, last_seen = $1 WHERE id = $2
		`, time.Now(), client.ID)

		if err != nil {
			log.Printf("Failed to update offline status: %v", err)
		}

		// Broadcast user offline status to their contacts
		h.broadcastPresence(client.ID, false)

		log.Printf("Client disconnected: %s (%s)", client.UniqueID, client.ID)
	}
}

// broadcastPresence sends user's online/offline status to their contacts
func (h *Hub) broadcastPresence(userID string, isOnline bool) {
	// Get user's contacts
	rows, err := database.Pool.Query(context.Background(), `
		SELECT user_id FROM contacts WHERE contact_id = $1
		UNION
		SELECT contact_id FROM contacts WHERE user_id = $1
	`, userID)

	if err != nil {
		log.Printf("Failed to get contacts: %v", err)
		return
	}
	defer rows.Close()

	// Prepare presence message
	message := WSMessage{
		Type: EventUserOnline,
		Payload: PresencePayload{
			UserID:   userID,
			IsOnline: isOnline,
			LastSeen: time.Now(),
		},
		Timestamp: time.Now(),
	}

	if !isOnline {
		message.Type = EventUserOffline
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal presence message: %v", err)
		return
	}

	// Send to each contact who is online
	for rows.Next() {
		var contactID string
		if err := rows.Scan(&contactID); err != nil {
			continue
		}

		h.mu.RLock()
		if client, ok := h.Clients[contactID]; ok {
			select {
			case client.Send <- data:
			default:
				log.Printf("Failed to send presence to client: %s", contactID)
			}
		}
		h.mu.RUnlock()
	}
}

// BroadcastToUser sends a message to a specific user
func (h *Hub) BroadcastToUser(userID string, message WSMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if client, ok := h.Clients[userID]; ok {
		data, err := json.Marshal(message)
		if err != nil {
			log.Printf("Failed to marshal message: %v", err)
			return
		}

		select {
		case client.Send <- data:
		default:
			log.Printf("Failed to send message to client: %s", userID)
		}
	}
}

// BroadcastToUsers sends a message to multiple users
func (h *Hub) BroadcastToUsers(userIDs []string, message WSMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, userID := range userIDs {
		if client, ok := h.Clients[userID]; ok {
			select {
			case client.Send <- data:
			default:
				log.Printf("Failed to send message to client: %s", userID)
			}
		}
	}
}

// BroadcastToGroup sends a message to all members of a group
func (h *Hub) BroadcastToGroup(groupID string, message WSMessage, excludeUserID string) {
	// Get group members
	rows, err := database.Pool.Query(context.Background(), `
		SELECT user_id FROM group_members WHERE group_id = $1
	`, groupID)

	if err != nil {
		log.Printf("Failed to get group members: %v", err)
		return
	}
	defer rows.Close()

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			continue
		}

		// Skip the sender
		if userID == excludeUserID {
			continue
		}

		if client, ok := h.Clients[userID]; ok {
			select {
			case client.Send <- data:
			default:
				log.Printf("Failed to send message to client: %s", userID)
			}
		}
	}
}

// IsUserOnline checks if a user is currently connected
func (h *Hub) IsUserOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	_, ok := h.Clients[userID]
	return ok
}

// GetOnlineUsers returns a list of currently online user IDs
func (h *Hub) GetOnlineUsers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	userIDs := make([]string, 0, len(h.Clients))
	for userID := range h.Clients {
		userIDs = append(userIDs, userID)
	}

	return userIDs
}

// GetOnlineCount returns the number of currently connected clients
func (h *Hub) GetOnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.Clients)
}
