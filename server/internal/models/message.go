package models

import "time"

// Message represents a chat message
type Message struct {
	ID         string    `json:"id" db:"id"`
	SenderID   string    `json:"senderId" db:"sender_id"`
	ReceiverID *string   `json:"receiverId,omitempty" db:"receiver_id"` // Null for group messages
	GroupID    *string   `json:"groupId,omitempty" db:"group_id"`       // Null for direct messages
	Content    string    `json:"content" db:"content"`
	Type       string    `json:"type" db:"type"`     // 'text', 'image', 'file'
	Status     string    `json:"status" db:"status"` // 'sent', 'delivered', 'read'
	CreatedAt  time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt  time.Time `json:"updatedAt" db:"updated_at"`
}

// MessageWithSender includes sender information
type MessageWithSender struct {
	ID         string       `json:"id"`
	Sender     UserResponse `json:"sender"`
	ReceiverID *string      `json:"receiverId,omitempty"`
	GroupID    *string      `json:"groupId,omitempty"`
	Content    string       `json:"content"`
	Type       string       `json:"type"`
	Status     string       `json:"status"`
	CreatedAt  time.Time    `json:"createdAt"`
	UpdatedAt  time.Time    `json:"updatedAt"`
}
