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
