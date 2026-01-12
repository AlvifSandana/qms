ALTER TABLE notifications
ADD COLUMN next_attempt_at TIMESTAMPTZ NULL;
