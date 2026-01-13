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

type Notification struct {
	NotificationID string
	TenantID       string
	Channel        string
	Recipient      string
	Status         string
	Attempts       int
	LastError      string
	Message        string
	NextAttemptAt  *time.Time
	CreatedAt      time.Time
}

type Store interface {
	ListOutboxEvents(ctx context.Context, after time.Time, limit int) ([]OutboxEvent, error)
	GetLastOffset(ctx context.Context) (time.Time, error)
	UpdateOffset(ctx context.Context, value time.Time) error
	IsNotificationsEnabled(ctx context.Context, tenantID string) (bool, error)
	GetQueuePosition(ctx context.Context, tenantID, branchID, serviceID, ticketID string) (int, error)
	GetTemplate(ctx context.Context, tenantID, templateID, lang, channel string) (string, error)
	InsertNotification(ctx context.Context, notification Notification) error
	ListDueNotifications(ctx context.Context, limit int) ([]Notification, error)
	MarkNotificationSent(ctx context.Context, notificationID string) error
	MarkNotificationRetry(ctx context.Context, notificationID, lastError string, nextAttemptAt time.Time) (int, error)
	MarkNotificationFailed(ctx context.Context, notificationID, lastError string) (int, error)
	InsertDLQ(ctx context.Context, notificationID, reason string) error
}
