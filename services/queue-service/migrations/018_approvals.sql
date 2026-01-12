CREATE TABLE approval_requests (
  approval_id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  request_type TEXT NOT NULL,
  payload JSONB NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  created_by UUID NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  approved_by UUID NULL,
  approved_at TIMESTAMPTZ NULL
);
