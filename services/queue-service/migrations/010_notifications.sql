CREATE TABLE tenant_notification_prefs (
  tenant_id UUID PRIMARY KEY,
  enabled BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE notification_templates (
  template_id TEXT NOT NULL,
  tenant_id UUID NOT NULL,
  lang TEXT NOT NULL,
  channel TEXT NOT NULL,
  body TEXT NOT NULL,
  PRIMARY KEY (template_id, tenant_id, lang, channel)
);

CREATE TABLE notifications (
  notification_id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  channel TEXT NOT NULL,
  recipient TEXT NOT NULL,
  status TEXT NOT NULL,
  attempts INTEGER NOT NULL DEFAULT 0,
  last_error TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  sent_at TIMESTAMPTZ NULL
);

CREATE TABLE notification_offsets (
  id INTEGER PRIMARY KEY,
  last_event_time TIMESTAMPTZ NOT NULL DEFAULT 'epoch'
);

CREATE TABLE notification_dlq (
  dlq_id UUID PRIMARY KEY,
  notification_id UUID NOT NULL REFERENCES notifications(notification_id),
  reason TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
