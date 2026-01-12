INSERT INTO tenant_notification_prefs (tenant_id, enabled)
VALUES ('11111111-1111-1111-1111-111111111111', TRUE)
ON CONFLICT DO NOTHING;

INSERT INTO notification_templates (template_id, tenant_id, lang, channel, body)
VALUES
  ('ticket_created', '11111111-1111-1111-1111-111111111111', 'id', 'sms', 'Tiket {ticket_number} dibuat untuk layanan {service_id}.'),
  ('ticket_called', '11111111-1111-1111-1111-111111111111', 'id', 'sms', 'Tiket {ticket_number} dipanggil di counter {counter_id}.'),
  ('ticket_recalled', '11111111-1111-1111-1111-111111111111', 'id', 'sms', 'Tiket {ticket_number} dipanggil ulang di counter {counter_id}.')
ON CONFLICT DO NOTHING;
