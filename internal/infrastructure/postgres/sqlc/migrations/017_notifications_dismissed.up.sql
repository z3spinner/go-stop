-- Lets a user delete a notification from their inbox. It is a soft delete: the
-- row stays as a tombstone so the (phone, dedup_key) uniqueness keeps a push
-- retry or a repeat ping/offer from bringing the same event back. Its content
-- is blanked shortly after (see PurgeInbox), and the row itself goes with the
-- usual cleanup (post expiry/deletion, retention limit).
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS dismissed_at TIMESTAMPTZ;
