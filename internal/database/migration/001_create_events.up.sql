CREATE TABLE events (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    type      TEXT        NOT NULL,
    payload         JSONB       NOT NULL,
    received_at     TIMESTAMPTZ NOT NULL DEFAULT now()
)