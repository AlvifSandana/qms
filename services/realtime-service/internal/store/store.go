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

type Store interface {
	ListOutboxEvents(ctx context.Context, after time.Time, limit int) ([]OutboxEvent, error)
}
