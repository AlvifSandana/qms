CREATE TABLE holidays (
  holiday_id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  branch_id UUID NOT NULL,
  date DATE NOT NULL,
  name TEXT NOT NULL,
  UNIQUE (tenant_id, branch_id, date)
);
