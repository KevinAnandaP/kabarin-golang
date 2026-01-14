package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gofiber/contrib/websocket"
)

// Client represents a WebSocket client connection
type Client struct {
	ID       string // User ID
	UniqueID string // User's unique ID (#WORD-123)
	Conn     *websocket.Conn
	Hub      *Hub
	Send     chan []byte
}

// NewClient creates a new WebSocket client
func NewClient(userID, uniqueID string, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		ID:       userID,
		UniqueID: uniqueID,
		Conn:     conn,
		Hub:      hub,
		Send:     make(chan []byte, 256),
	}
}

// ReadPump handles incoming messages from the client
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Parse incoming message
		var incoming IncomingMessage
		if err := json.Unmarshal(message, &incoming); err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		// Handle different event types
		c.handleIncomingMessage(incoming)
	}
}

// WritePump handles outgoing messages to the client
func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Write error: %v", err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleIncomingMessage processes different types of incoming messages
func (c *Client) handleIncomingMessage(msg IncomingMessage) {
	switch msg.Type {
	case EventTypingStart:
		c.handleTypingStart(msg.Payload)
	case EventTypingStop:
		c.handleTypingStop(msg.Payload)
	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// handleTypingStart broadcasts typing start event
func (c *Client) handleTypingStart(payload map[string]interface{}) {
	chatID, _ := payload["chatId"].(string)
	groupID, _ := payload["groupId"].(string)

	typingPayload := TypingPayload{
		UserID:  c.ID,
		ChatID:  chatID,
		GroupID: groupID,
	}

	message := WSMessage{
		Type:      EventTypingStart,
		Payload:   typingPayload,
		Timestamp: time.Now(),
	}

	// Broadcast to relevant users
	if groupID != "" {
		c.Hub.BroadcastToGroup(groupID, message, c.ID)
	} else if chatID != "" {
		// Extract receiver ID from chatId and send to that user
		c.Hub.BroadcastToUsers([]string{chatID}, message)
	}
}

// handleTypingStop broadcasts typing stop event
func (c *Client) handleTypingStop(payload map[string]interface{}) {
	chatID, _ := payload["chatId"].(string)
	groupID, _ := payload["groupId"].(string)

	typingPayload := TypingPayload{
		UserID:  c.ID,
		ChatID:  chatID,
		GroupID: groupID,
	}

	message := WSMessage{
		Type:      EventTypingStop,
		Payload:   typingPayload,
		Timestamp: time.Now(),
	}

	// Broadcast to relevant users
	if groupID != "" {
		c.Hub.BroadcastToGroup(groupID, message, c.ID)
	} else if chatID != "" {
		c.Hub.BroadcastToUsers([]string{chatID}, message)
	}
}

// SendMessage sends a message to the client
func (c *Client) SendMessage(msg WSMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case c.Send <- data:
	default:
		close(c.Send)
		c.Hub.Unregister <- c
	}

	return nil
}
