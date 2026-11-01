-- NULL means Pending User. Existing Users stay able to sign in.
ALTER TABLE auth.users ADD COLUMN approved_at TIMESTAMPTZ;
UPDATE auth.users SET approved_at = created_at;
