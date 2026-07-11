DROP INDEX IF EXISTS idx_events_pending;
ALTER TABLE events DROP COLUMN IF EXISTS status;
DROP TYPE IF EXISTS publish_status;