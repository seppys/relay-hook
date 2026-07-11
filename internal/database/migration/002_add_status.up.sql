CREATE TYPE publish_status AS ENUM ('pending', 'published');

ALTER TABLE events ADD COLUMN status publish_status NOT NULL DEFAULT 'pending';

CREATE INDEX idx_events_pending ON events (received_at) WHERE status = 'pending';