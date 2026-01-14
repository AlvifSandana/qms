package postgres

import (
	"context"
	"errors"
	"time"

	"qms/realtime-service/internal/store"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) ListOutboxEvents(ctx context.Context, offset store.OutboxOffset, limit int) ([]store.OutboxEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT event_id, tenant_id, type, payload_json, created_at
		FROM outbox_events
		WHERE created_at > $1 OR (created_at = $1 AND event_id > $2)
		ORDER BY created_at ASC, event_id ASC
		LIMIT $3
	`, offset.LastEventTime, offset.LastEventID, limit)
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

func (s *Store) GetOffset(ctx context.Context) (store.OutboxOffset, error) {
	var offset store.OutboxOffset
	row := s.pool.QueryRow(ctx, `
		SELECT last_event_time, last_event_id
		FROM realtime_offsets
		WHERE id = 1
	`)
	if err := row.Scan(&offset.LastEventTime, &offset.LastEventID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.OutboxOffset{}, nil
		}
		return store.OutboxOffset{}, err
	}
	return offset, nil
}

func (s *Store) UpdateOffset(ctx context.Context, offset store.OutboxOffset) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO realtime_offsets (id, last_event_time, last_event_id)
		VALUES (1, $1, $2)
		ON CONFLICT (id) DO UPDATE SET last_event_time = EXCLUDED.last_event_time, last_event_id = EXCLUDED.last_event_id
	`, offset.LastEventTime, offset.LastEventID)
	return err
}

func (s *Store) GetNotificationOffset(ctx context.Context) (time.Time, error) {
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

func (s *Store) CleanupOutbox(ctx context.Context, before time.Time) error {
	if before.IsZero() {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
		DELETE FROM outbox_events
		WHERE created_at < $1
	`, before)
	return err
}

func (s *Store) GetSession(ctx context.Context, sessionID string) (store.Session, error) {
	var session store.Session
	row := s.pool.QueryRow(ctx, `
		SELECT s.session_id, s.user_id, s.expires_at, u.tenant_id, r.name
		FROM sessions s
		JOIN users u ON u.user_id = s.user_id
		JOIN roles r ON r.role_id = u.role_id
		WHERE s.session_id = $1 AND s.expires_at > NOW()
	`, sessionID)
	if err := row.Scan(&session.SessionID, &session.UserID, &session.ExpiresAt, &session.TenantID, &session.Role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.Session{}, store.ErrSessionNotFound
		}
		return store.Session{}, err
	}
	return session, nil
}

func (s *Store) GetAccess(ctx context.Context, userID string) ([]string, []string, error) {
	branchRows, err := s.pool.Query(ctx, `
		SELECT branch_id
		FROM user_branch_access
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, nil, err
	}
	defer branchRows.Close()

	var branches []string
	for branchRows.Next() {
		var branchID string
		if err := branchRows.Scan(&branchID); err != nil {
			return nil, nil, err
		}
		branches = append(branches, branchID)
	}
	if err := branchRows.Err(); err != nil {
		return nil, nil, err
	}

	serviceRows, err := s.pool.Query(ctx, `
		SELECT service_id
		FROM user_service_access
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, nil, err
	}
	defer serviceRows.Close()

	var services []string
	for serviceRows.Next() {
		var serviceID string
		if err := serviceRows.Scan(&serviceID); err != nil {
			return nil, nil, err
		}
		services = append(services, serviceID)
	}
	if err := serviceRows.Err(); err != nil {
		return nil, nil, err
	}

	return branches, services, nil
}
