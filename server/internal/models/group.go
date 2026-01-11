package models

import "time"

// Group represents a chat group
type Group struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Icon      *string   `json:"icon,omitempty" db:"icon"`
	CreatedBy string    `json:"createdBy" db:"created_by"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// GroupMember represents a user's membership in a group
type GroupMember struct {
	GroupID  string    `json:"groupId" db:"group_id"`
	UserID   string    `json:"userId" db:"user_id"`
	JoinedAt time.Time `json:"joinedAt" db:"joined_at"`
}

// GroupWithMembers includes member information
type GroupWithMembers struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Icon      *string        `json:"icon,omitempty"`
	CreatedBy string         `json:"createdBy"`
	Members   []UserResponse `json:"members"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}
