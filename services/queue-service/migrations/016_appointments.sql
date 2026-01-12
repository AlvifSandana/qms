CREATE TABLE appointments (
  appointment_id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  branch_id UUID NOT NULL,
  service_id UUID NOT NULL,
  scheduled_at TIMESTAMPTZ NOT NULL,
  status TEXT NOT NULL DEFAULT 'scheduled',
  customer_ref TEXT NULL
);

ALTER TABLE tickets
ADD COLUMN appointment_id UUID NULL REFERENCES appointments(appointment_id);
