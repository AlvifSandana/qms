CREATE TABLE ticket_action_requests (
  request_id UUID PRIMARY KEY,
  action TEXT NOT NULL,
  tenant_id UUID NOT NULL,
  branch_id UUID NOT NULL,
  service_id UUID NULL,
  counter_id UUID NULL,
  ticket_id UUID NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ticket_action_requests_action ON ticket_action_requests (action, created_at);
