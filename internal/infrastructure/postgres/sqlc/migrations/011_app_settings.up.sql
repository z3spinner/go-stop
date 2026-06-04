-- Generic key/value store for runtime-provisioned configuration
-- (e.g. self-generated VAPID keys). Single source of truth at runtime.
CREATE TABLE app_settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
