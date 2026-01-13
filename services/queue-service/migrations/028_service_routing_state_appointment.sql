ALTER TABLE service_routing_state
ADD COLUMN appointment_served INT NOT NULL DEFAULT 0,
ADD COLUMN total_served INT NOT NULL DEFAULT 0;
