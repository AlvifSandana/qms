ALTER TABLE services
ADD COLUMN priority_policy TEXT NOT NULL DEFAULT 'fifo',
ADD COLUMN hours_json JSONB NULL;
