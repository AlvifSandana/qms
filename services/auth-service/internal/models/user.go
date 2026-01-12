package models

import "time"

type User struct {
	UserID   string    `json:"user_id"`
	TenantID string    `json:"tenant_id"`
	RoleName string    `json:"role"`
	Email    string    `json:"email"`
	Created  time.Time `json:"created_at"`
}

type Session struct {
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}
