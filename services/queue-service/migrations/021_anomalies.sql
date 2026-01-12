CREATE TABLE anomalies (
  anomaly_id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  branch_id UUID NOT NULL,
  service_id UUID NOT NULL,
  type TEXT NOT NULL,
  value NUMERIC NOT NULL,
  threshold NUMERIC NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
