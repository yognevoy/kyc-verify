CREATE TABLE users (
    id            UUID PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'applicant' CHECK (role IN ('applicant', 'reviewer')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
