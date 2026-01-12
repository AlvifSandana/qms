CREATE TABLE areas (
  area_id UUID PRIMARY KEY,
  branch_id UUID NOT NULL REFERENCES branches(branch_id),
  name TEXT NOT NULL
);

CREATE TABLE counters (
  counter_id UUID PRIMARY KEY,
  branch_id UUID NOT NULL REFERENCES branches(branch_id),
  area_id UUID NULL REFERENCES areas(area_id),
  name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active'
);

CREATE TABLE counter_services (
  counter_id UUID NOT NULL REFERENCES counters(counter_id),
  service_id UUID NOT NULL REFERENCES services(service_id),
  PRIMARY KEY (counter_id, service_id)
);
