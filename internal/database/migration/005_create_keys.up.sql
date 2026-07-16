CREATE TYPE key_role AS ENUM ('emitter', 'subscriber');

CREATE TABLE keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    key_hash    TEXT NOT NULL,
    role        key_role NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL
);

CREATE TABLE key_permissions (
     id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
     key_id      UUID NOT NULL REFERENCES keys(id) ON DELETE CASCADE,
     event_type  TEXT NOT NULL
);