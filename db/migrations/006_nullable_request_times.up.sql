-- Allow date and departure_at to be NULL for day-mode and anytime alerts.
ALTER TABLE requests
  ALTER COLUMN date DROP NOT NULL,
  ALTER COLUMN departure_at DROP NOT NULL;
