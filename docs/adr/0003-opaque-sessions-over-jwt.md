# Opaque database sessions instead of JWT

Logging in creates a random opaque token; only its SHA-256 hash is stored in `auth.sessions`, and every authenticated request looks the hash up in Postgres. We chose this over stateless JWTs because there is one service and one database, so the per-request lookup costs nothing meaningful, while JWTs cannot be revoked before they expire without adding a denylist that makes them stateful anyway. Logout is a row delete, there is no signing secret to leak or rotate, and the client learns nothing from the token. Clients send it as `Authorization: Bearer`; the dashboard (TanStack Start) holds it in its own `HttpOnly` cookie and calls the API only from server functions, so the API needs no CORS or cookie handling.

Revisit if a second service must verify tokens without access to this database.
