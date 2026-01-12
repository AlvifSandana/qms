CREATE TABLE service_routing_state (
  tenant_id UUID NOT NULL,
  branch_id UUID NOT NULL,
  service_id UUID NOT NULL,
  priority_streak INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (tenant_id, branch_id, service_id)
);
