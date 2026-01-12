CREATE TYPE ticket_status AS ENUM (
  'waiting',
  'called',
  'serving',
  'done',
  'no_show',
  'cancelled',
  'held'
);

CREATE TABLE tenants (
  tenant_id UUID PRIMARY KEY,
  name TEXT NOT NULL
);

CREATE TABLE branches (
  branch_id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
  name TEXT NOT NULL
);

CREATE TABLE services (
  service_id UUID PRIMARY KEY,
  branch_id UUID NOT NULL REFERENCES branches(branch_id),
  name TEXT NOT NULL,
  code TEXT NOT NULL,
  active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE ticket_sequences (
  branch_id UUID NOT NULL,
  service_id UUID NOT NULL,
  next_number BIGINT NOT NULL,
  PRIMARY KEY (branch_id, service_id)
);

CREATE TABLE tickets (
  ticket_id UUID PRIMARY KEY,
  request_id UUID UNIQUE NOT NULL,
  ticket_number TEXT NOT NULL,
  tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
  branch_id UUID NOT NULL REFERENCES branches(branch_id),
  service_id UUID NOT NULL REFERENCES services(service_id),
  area_id UUID NULL,
  status ticket_status NOT NULL,
  channel TEXT NOT NULL,
  priority_class TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  called_at TIMESTAMPTZ NULL,
  served_at TIMESTAMPTZ NULL,
  completed_at TIMESTAMPTZ NULL,
  counter_id UUID NULL
);

CREATE TABLE outbox_events (
  event_id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  type TEXT NOT NULL,
  payload_json JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  processed_at TIMESTAMPTZ NULL,
  attempts INTEGER NOT NULL DEFAULT 0,
  last_error TEXT NULL
);
