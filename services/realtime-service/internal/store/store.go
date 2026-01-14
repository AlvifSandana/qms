package store

import (
	"context"
	"time"
)

type OutboxEvent struct {
	EventID   string
	TenantID  string
	Type      string
	Payload   []byte
	CreatedAt time.Time
}

type OutboxOffset struct {
	LastEventTime time.Time
	LastEventID   string
}

type Session struct {
	SessionID string
	UserID    string
	TenantID  string
	Role      string
	ExpiresAt time.Time
}

type Store interface {
	ListOutboxEvents(ctx context.Context, offset OutboxOffset, limit int) ([]OutboxEvent, error)
	GetOffset(ctx context.Context) (OutboxOffset, error)
	UpdateOffset(ctx context.Context, offset OutboxOffset) error
	GetNotificationOffset(ctx context.Context) (time.Time, error)
	CleanupOutbox(ctx context.Context, before time.Time) error
	GetSession(ctx context.Context, sessionID string) (Session, error)
	GetAccess(ctx context.Context, userID string) ([]string, []string, error)
}
