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

**Pagination meta**:
The envelope's optional `meta` object on list responses: the current `page`, the `per_page` size, and the `total` number of matching items.
_Avoid_: page info, paging, cursor

**Liveness probe**:
`/healthz`, which reports only that the process is up. Outside the API envelope on purpose.
_Avoid_: health check, readiness, heartbeat

**Readiness probe**:
`/readyz`, which reports whether the service can reach its dependencies (the database). Outside the API envelope, like the liveness probe.
_Avoid_: health check, deep health

**User**:
A person known to the service. Identified by a unique username; also has an email and a display name. Is either a **Pending User** or an **Approved User**; only an Approved User can sign in.
_Avoid_: account, akun, member (as a synonym)

**Registration**:
Anyone, without signing in, asking to become a User. Always produces a Pending User with the Member Role. Distinct from a Superadmin creating an Approved User directly.
_Avoid_: sign-up, account request

**Pending User**:
A User created by a Registration who has not yet been approved. Cannot sign in. Holds their username and email until approved or rejected.
_Avoid_: registration request, applicant, inactive user

**Approved User**:
A User who can sign in: either approved by a Superadmin, or created directly by one.
_Avoid_: active user, verified user

**Approval**:
A Superadmin turning a Pending User into an Approved User. One-way: an Approved User never returns to pending.
_Avoid_: activation, verification

**Rejection**:
A Superadmin discarding a Pending User. The User ceases to exist; their username and email become free again.
_Avoid_: denial, ban

**Role**:
The permission level a User holds. Exactly one of **Superadmin** or **Member**.
_Avoid_: permission, group, level

**Superadmin**:
A User with full control, including Approval and Rejection of Pending Users and creating Approved Users directly.
_Avoid_: admin, root, owner

**Member**:
A User without administrative rights. Every publicly registered User is a Member.
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
