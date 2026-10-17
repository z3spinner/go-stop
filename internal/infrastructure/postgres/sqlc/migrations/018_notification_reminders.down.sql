DROP INDEX IF EXISTS idx_notifications_reminder_due;
ALTER TABLE notifications DROP COLUMN IF EXISTS reminder_count;
ALTER TABLE notifications DROP COLUMN IF EXISTS reminded_at;
