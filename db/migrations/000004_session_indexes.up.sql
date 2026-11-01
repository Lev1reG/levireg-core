-- Serves the per-User session cap on login and the ON DELETE CASCADE from auth.users.
CREATE INDEX sessions_username_created_at_idx ON auth.sessions (username, created_at DESC);
