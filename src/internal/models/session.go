package models

import "time"

type Session struct {
	SessionID    string     `bson:"session_id"`
	UserID       string     `bson:"user_id"`
	IsActive     bool       `bson:"is_active"`
	ExpiresAt    time.Time  `bson:"expires_at"`
	CreatedAt    time.Time  `bson:"created_at"`
	LogoutAt     *time.Time `bson:"logout_at,omitempty"`
	LastActiveAt time.Time  `bson:"lastActiveAt"`
}
