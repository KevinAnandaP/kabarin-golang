package websocket

import "time"

// EventType represents different WebSocket event types
type EventType string

const (
	// Connection events
	EventConnect    EventType = "connect"
	EventDisconnect EventType = "disconnect"

	// Message events
	EventMessageSent      EventType = "message_sent"
	EventMessageDelivered EventType = "message_delivered"
	EventMessageRead      EventType = "message_read"
	EventMessageReceived  EventType = "message_received"

	// Group message events
	EventGroupMessageSent     EventType = "group_message_sent"
	EventGroupMessageReceived EventType = "group_message_received"

	// Typing events
	EventTypingStart EventType = "typing_start"
	EventTypingStop  EventType = "typing_stop"

	// Presence events
	EventUserOnline  EventType = "user_online"
	EventUserOffline EventType = "user_offline"

	// Error events
	EventError EventType = "error"
)

// WSMessage represents a WebSocket message structure
type WSMessage struct {
	Type      EventType   `json:"type"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
}

// MessagePayload represents message event payload
type MessagePayload struct {
	ID         string    `json:"id"`
	ChatID     string    `json:"chatId"`
	SenderID   string    `json:"senderId"`
	ReceiverID string    `json:"receiverId"`
	Content    string    `json:"content"`
	Type       string    `json:"type"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"createdAt"`
}

// GroupMessagePayload represents group message event payload
type GroupMessagePayload struct {
	ID        string    `json:"id"`
	GroupID   string    `json:"groupId"`
	SenderID  string    `json:"senderId"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
}

// TypingPayload represents typing indicator payload
type TypingPayload struct {
	UserID   string `json:"userId"`
	ChatID   string `json:"chatId,omitempty"`
	GroupID  string `json:"groupId,omitempty"`
	UserName string `json:"userName"`
}

// PresencePayload represents user presence payload
type PresencePayload struct {
	UserID   string    `json:"userId"`
	IsOnline bool      `json:"isOnline"`
	LastSeen time.Time `json:"lastSeen,omitempty"`
}

// MessageStatusPayload represents message status update payload
type MessageStatusPayload struct {
	MessageID string    `json:"messageId"`
	Status    string    `json:"status"` // sent, delivered, read
	UpdatedAt time.Time `json:"updatedAt"`
}

// ErrorPayload represents error event payload
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// IncomingMessage represents messages received from clients
type IncomingMessage struct {
	Type    EventType              `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}
