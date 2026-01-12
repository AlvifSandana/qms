CREATE TABLE device_configs (
  config_id UUID PRIMARY KEY,
  device_id UUID NOT NULL,
  version INTEGER NOT NULL,
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_device_configs_device_version ON device_configs (device_id, version);
