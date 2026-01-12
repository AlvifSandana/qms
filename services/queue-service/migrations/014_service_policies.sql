CREATE TABLE service_policies (
  tenant_id UUID NOT NULL,
  branch_id UUID NOT NULL,
  service_id UUID NOT NULL,
  no_show_grace_seconds INTEGER NOT NULL DEFAULT 300,
  return_to_queue BOOLEAN NOT NULL DEFAULT FALSE,
  PRIMARY KEY (tenant_id, branch_id, service_id)
);
