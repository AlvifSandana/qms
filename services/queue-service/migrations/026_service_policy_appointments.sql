ALTER TABLE service_policies
ADD COLUMN appointment_ratio_percent INT NOT NULL DEFAULT 0,
ADD COLUMN appointment_window_size INT NOT NULL DEFAULT 10,
ADD COLUMN appointment_boost_minutes INT NOT NULL DEFAULT 15;
