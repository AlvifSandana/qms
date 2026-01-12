CREATE TABLE call_next_requests (
  request_id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  branch_id UUID NOT NULL,
  service_id UUID NOT NULL,
  counter_id UUID NOT NULL,
  ticket_id UUID NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
