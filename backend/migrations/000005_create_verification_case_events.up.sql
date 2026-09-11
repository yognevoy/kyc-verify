CREATE TABLE verification_case_events (
    id          UUID PRIMARY KEY,
    case_id     UUID NOT NULL REFERENCES verification_cases (id) ON DELETE CASCADE,
    from_status TEXT,
    to_status   TEXT NOT NULL,
    actor_type  TEXT NOT NULL CHECK (actor_type IN ('system', 'user', 'reviewer', 'provider')),
    actor_id    UUID REFERENCES users (id),
    comment     TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_verification_case_events_case_id ON verification_case_events (case_id);
