package models

import "time"

// User represents a user in the system
type User struct {
	ID           string    `json:"id" db:"id"`
	UniqueID     string    `json:"uniqueId" db:"unique_id"` // Format: #GOPRO-882
	Email        string    `json:"email" db:"email"`
	Name         string    `json:"name" db:"name"`
	Password     string    `json:"-" db:"password_hash"` // Never expose in JSON
	Avatar       *string   `json:"avatar,omitempty" db:"avatar"`
	AuthProvider string    `json:"authProvider" db:"auth_provider"` // 'email' or 'google'
	GoogleID     *string   `json:"-" db:"google_id"`
	IsOnline     bool      `json:"isOnline" db:"is_online"`
	LastSeen     time.Time `json:"lastSeen" db:"last_seen"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time `json:"updatedAt" db:"updated_at"`
}

// UserResponse is what we send to clients (without sensitive data)
type UserResponse struct {
	ID           string    `json:"id"`
	UniqueID     string    `json:"uniqueId"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Avatar       *string   `json:"avatar,omitempty"`
	AuthProvider string    `json:"authProvider"`
	IsOnline     bool      `json:"isOnline"`
	LastSeen     time.Time `json:"lastSeen"`
	CreatedAt    time.Time `json:"createdAt"`
}

// ToResponse converts User to UserResponse
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:           u.ID,
		UniqueID:     u.UniqueID,
		Email:        u.Email,
		Name:         u.Name,
		Avatar:       u.Avatar,
		AuthProvider: u.AuthProvider,
		IsOnline:     u.IsOnline,
		LastSeen:     u.LastSeen,
		CreatedAt:    u.CreatedAt,
	}
}
