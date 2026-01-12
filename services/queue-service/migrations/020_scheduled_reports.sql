CREATE TABLE scheduled_reports (
  report_id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  branch_id UUID NOT NULL,
  service_id UUID NOT NULL,
  cron TEXT NOT NULL,
  channel TEXT NOT NULL,
  recipient TEXT NOT NULL,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  last_sent_at TIMESTAMPTZ NULL
);
