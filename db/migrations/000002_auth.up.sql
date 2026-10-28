CREATE SCHEMA auth;

CREATE TABLE auth.users (
    username      TEXT PRIMARY KEY CHECK (username ~ '^[a-z0-9_]{3,32}$'),
    email         TEXT        NOT NULL,
    name          TEXT        NOT NULL,
    password_hash TEXT        NOT NULL,
    role          TEXT        NOT NULL DEFAULT 'member' CHECK (role IN ('superadmin', 'member')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_email_lower_key ON auth.users (lower(email));

CREATE TABLE auth.sessions (
    token_hash BYTEA       PRIMARY KEY,
    username   TEXT        NOT NULL REFERENCES auth.users (username) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);
