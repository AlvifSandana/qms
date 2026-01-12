package postgres

import (
	"context"
	"time"

	"qms/realtime-service/internal/store"

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
		limit = 100
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
