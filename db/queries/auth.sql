-- name: CreateUser :one
INSERT INTO auth.users (username, email, name, password_hash, role, approved_at)
VALUES ($1, $2, $3, $4, $5, CASE WHEN sqlc.arg(approved)::boolean THEN now() END)
RETURNING *;

-- name: ListUsers :many
SELECT * FROM auth.users
WHERE CASE sqlc.arg(status)::text
    WHEN 'pending' THEN approved_at IS NULL
    WHEN 'approved' THEN approved_at IS NOT NULL
    ELSE true
END
ORDER BY created_at DESC, username ASC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset)::bigint;

-- name: CountUsers :one
SELECT count(*) FROM auth.users
WHERE CASE sqlc.arg(status)::text
    WHEN 'pending' THEN approved_at IS NULL
    WHEN 'approved' THEN approved_at IS NOT NULL
    ELSE true
END;

-- Conditional on still being pending, so concurrent actions cannot both win.
-- name: ApproveUser :one
UPDATE auth.users SET approved_at = now(), updated_at = now()
WHERE username = $1 AND approved_at IS NULL
RETURNING *;


-- name: RejectUser :execrows
DELETE FROM auth.users WHERE username = $1 AND approved_at IS NULL;
-- name: GetUserByUsername :one
SELECT * FROM auth.users WHERE username = $1;

-- Inserts the session and drops the User's oldest ones beyond max_sessions. The
-- DELETE cannot see the new row, so it keeps max_sessions-1 existing ones.
-- Concurrent logins may briefly exceed the cap; the next login trims it.
-- name: CreateSession :exec
WITH new_session AS (
    INSERT INTO auth.sessions (token_hash, username, expires_at)
    VALUES (sqlc.arg(token_hash), sqlc.arg(username), sqlc.arg(expires_at))
)
DELETE FROM auth.sessions AS t
WHERE t.username = sqlc.arg(username) AND t.token_hash NOT IN (
    SELECT s.token_hash FROM auth.sessions s
    WHERE s.username = sqlc.arg(username)
    ORDER BY s.created_at DESC
    LIMIT sqlc.arg(max_sessions)::bigint - 1
);

-- name: DeleteExpiredSessions :execrows
DELETE FROM auth.sessions WHERE expires_at <= now();

-- name: GetUserBySessionToken :one
SELECT u.* FROM auth.sessions s
JOIN auth.users u ON u.username = s.username
WHERE s.token_hash = $1 AND s.expires_at > now();

-- name: DeleteSession :exec
DELETE FROM auth.sessions WHERE token_hash = $1;
