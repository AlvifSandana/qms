CREATE TABLE tenant_approval_prefs (
  tenant_id UUID PRIMARY KEY,
  approvals_enabled BOOLEAN NOT NULL DEFAULT FALSE
);
