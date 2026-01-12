package postgres

import (
	"context"
	"errors"
	"time"

	"qms/notification-service/internal/store"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) ListOutboxEvents(ctx context.Context, after time.Time, limit int) ([]store.OutboxEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT event_id, tenant_id, type, payload_json, created_at
		FROM outbox_events
		WHERE created_at > $1
		ORDER BY created_at ASC
		LIMIT $2
	`, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []store.OutboxEvent
	for rows.Next() {
		var event store.OutboxEvent
		if err := rows.Scan(&event.EventID, &event.TenantID, &event.Type, &event.Payload, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func (s *Store) GetLastOffset(ctx context.Context) (time.Time, error) {
	var value time.Time
	row := s.pool.QueryRow(ctx, `
		SELECT last_event_time
		FROM notification_offsets
		WHERE id = 1
	`)
	if err := row.Scan(&value); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	return value, nil
}

func (s *Store) UpdateOffset(ctx context.Context, value time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO notification_offsets (id, last_event_time)
		VALUES (1, $1)
		ON CONFLICT (id) DO UPDATE SET last_event_time = EXCLUDED.last_event_time
	`, value)
	return err
}

func (s *Store) IsNotificationsEnabled(ctx context.Context, tenantID string) (bool, error) {
	var enabled bool
	row := s.pool.QueryRow(ctx, `
		SELECT enabled
		FROM tenant_notification_prefs
		WHERE tenant_id = $1
	`, tenantID)
	if err := row.Scan(&enabled); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return true, nil
		}
		return false, err
	}
	return enabled, nil
}

func (s *Store) GetTemplate(ctx context.Context, tenantID, templateID, lang, channel string) (string, error) {
	var body string
	row := s.pool.QueryRow(ctx, `
		SELECT body
		FROM notification_templates
		WHERE tenant_id = $1 AND template_id = $2 AND lang = $3 AND channel = $4
	`, tenantID, templateID, lang, channel)
	if err := row.Scan(&body); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return body, nil
}

func (s *Store) InsertNotification(ctx context.Context, notification store.Notification) error {
	if notification.NotificationID == "" {
		notification.NotificationID = uuid.NewString()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO notifications (notification_id, tenant_id, channel, recipient, status, attempts, last_error)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, notification.NotificationID, notification.TenantID, notification.Channel, notification.Recipient, notification.Status, notification.Attempts, notification.LastError)
	return err
}

func (s *Store) MarkNotificationSent(ctx context.Context, notificationID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE notifications
		SET status = 'sent', sent_at = NOW()
		WHERE notification_id = $1
	`, notificationID)
	return err
}

func (s *Store) MarkNotificationFailed(ctx context.Context, notificationID, lastError string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE notifications
		SET status = 'failed', last_error = $2
		WHERE notification_id = $1
	`, notificationID, lastError)
	return err
}

func (s *Store) InsertDLQ(ctx context.Context, notificationID, reason string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO notification_dlq (dlq_id, notification_id, reason)
		VALUES ($1, $2, $3)
	`, uuid.NewString(), notificationID, reason)
	return err
}
