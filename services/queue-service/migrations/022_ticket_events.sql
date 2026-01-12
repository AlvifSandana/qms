CREATE TABLE ticket_events (
  seq BIGSERIAL PRIMARY KEY,
  ticket_id UUID NOT NULL,
  type TEXT NOT NULL,
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
