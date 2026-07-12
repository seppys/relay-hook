CREATE TABLE subscribers (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_url  TEXT NOT NULL
);

CREATE TABLE subscriptions (
    id            UUID NOT NULL DEFAULT gen_random_uuid(),
    subscriber_id UUID NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
    event_type    TEXT NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (subscriber_id, event_type)
);

CREATE INDEX idx_subscriptions_event_type ON subscriptions (event_type);