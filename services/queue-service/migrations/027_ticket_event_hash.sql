ALTER TABLE ticket_events
ADD COLUMN ticket_seq INT,
ADD COLUMN prev_hash TEXT,
ADD COLUMN hash TEXT;

WITH ranked AS (
  SELECT seq, ROW_NUMBER() OVER (PARTITION BY ticket_id ORDER BY seq) AS rn
  FROM ticket_events
)
UPDATE ticket_events
SET ticket_seq = ranked.rn
FROM ranked
WHERE ticket_events.seq = ranked.seq;

ALTER TABLE ticket_events
ALTER COLUMN ticket_seq SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ticket_events_ticket_seq_idx
ON ticket_events (ticket_id, ticket_seq);
