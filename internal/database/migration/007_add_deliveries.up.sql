CREATE TYPE delivery_status AS ENUM ('processing', 'delivered', 'failed', 'dead');

CREATE TABLE deliveries (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id        UUID            NOT NULL REFERENCES events(id),
    subscriber_id   UUID            NOT NULL REFERENCES subscribers(id),
    payload         JSONB           NOT NULL,
    endpoint_url    TEXT            NOT NULL,
    status          delivery_status NOT NULL,
    attempts        INTEGER         NOT NULL,
    claimed_at      TIMESTAMPTZ     NOT NULL DEFAULT now()
)