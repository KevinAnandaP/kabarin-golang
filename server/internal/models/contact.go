package models

import "time"

// Contact represents a friendship connection between two users
type Contact struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"userId" db:"user_id"`
	ContactID string    `json:"contactId" db:"contact_id"`
	AddedAt   time.Time `json:"addedAt" db:"added_at"`
}

// ContactWithUser includes the contact's user information
type ContactWithUser struct {
	ID       string       `json:"id"`
	UserID   string       `json:"userId"`
	Contact  UserResponse `json:"contact"`
	AddedAt  time.Time    `json:"addedAt"`
	IsOnline bool         `json:"isOnline"`
}
