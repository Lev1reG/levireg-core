-- name: CreateUser :one
INSERT INTO auth.users (username, email, name, password_hash, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

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
