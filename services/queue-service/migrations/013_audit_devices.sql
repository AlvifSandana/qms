CREATE TABLE audit_logs (
  audit_id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  actor_user_id UUID NULL,
  action_type TEXT NOT NULL,
  target_type TEXT NOT NULL,
  target_id UUID NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ip TEXT NULL,
  user_agent TEXT NULL
);

CREATE TABLE devices (
  device_id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  branch_id UUID NOT NULL REFERENCES branches(branch_id),
  area_id UUID NULL REFERENCES areas(area_id),
  type TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'offline',
  last_seen TIMESTAMPTZ NULL
);
