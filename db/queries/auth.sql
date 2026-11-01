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

-- name: CreateSession :exec
INSERT INTO auth.sessions (token_hash, username, expires_at)
VALUES ($1, $2, $3);

-- name: GetUserBySessionToken :one
SELECT u.* FROM auth.sessions s
JOIN auth.users u ON u.username = s.username
WHERE s.token_hash = $1 AND s.expires_at > now();

-- name: DeleteSession :exec
DELETE FROM auth.sessions WHERE token_hash = $1;
