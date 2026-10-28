# levireg-core

Backend service for the personal website. Serves the JSON API consumed by the site frontend.

## Language

**API envelope**:
The fixed shape `{code, message, data}` every `/api/v1` response is wrapped in. `code` is the stable contract, `message` is for humans, `data` is the payload.
_Avoid_: response wrapper, apiMsg, apiData

**Code**:
A stable, machine-readable string identifying the outcome of an API request (e.g. `ok`, `email_taken`). Clients branch on it; it never changes with wording.
_Avoid_: status, error type, apiMsg

**Field error**:
A validation message attached to a single request field, carried in the envelope's optional `errors` map.
_Avoid_: validation error, form error

**Liveness probe**:
`/healthz`, which reports only that the process is up. Outside the API envelope on purpose.
_Avoid_: health check, readiness, heartbeat

**Readiness probe**:
`/readyz`, which reports whether the service can reach its dependencies (the database). Outside the API envelope, like the liveness probe.
_Avoid_: health check, deep health

**User**:
A person who can sign in to the service. Identified by a unique username; also has an email and a display name.
_Avoid_: account, akun, member (as a synonym)

**Role**:
The permission level a User holds. Exactly one of **Superadmin** or **Member**.
_Avoid_: permission, group, level

**Superadmin**:
A User with full control, including registering new Users.
_Avoid_: admin, root, owner

**Member**:
A User without administrative rights.
_Avoid_: normal user, regular user, guest

**Session**:
A User's signed-in state, created by logging in and ended by logging out or by expiring. Proven by presenting its token.
_Avoid_: login, auth token, JWT

**API spec**:
The OpenAPI document that is the authoritative contract for every endpoint: paths, request bodies, responses and the Codes each can return. If the service and the API spec disagree, it is a bug.
_Avoid_: swagger, docs, schema

**API reference**:
The human-readable page rendered from the API spec, for browsing and trying endpoints.
_Avoid_: swagger, docs, API docs
