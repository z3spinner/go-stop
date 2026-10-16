-- Reminder pushes for unread notifications about another person's action
-- (contact request, accepted request, shared number, contact offer). The
-- reminders are push-only: they never add inbox rows, they just record when
-- the item was last re-pushed.
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS reminded_at TIMESTAMPTZ;
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS reminder_count INT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_notifications_reminder_due
    ON notifications (COALESCE(reminded_at, created_at))
    WHERE read_at IS NULL AND dismissed_at IS NULL;
