# API responses use a `{code, message, data}` envelope

Every `/api/v1` response is wrapped in `{code, message, data, errors}`, where `code` is a stable machine-readable string, `message` is human-facing prose, `data` is the typed payload, and `errors` carries per-field validation messages. We chose this over returning raw payloads with the HTTP status alone because the frontend needs to distinguish outcomes ("email taken" vs "rate limited") without parsing translated strings, and over a single `{apiMsg, apiData}` field pair because one string cannot serve both machines and humans. `code` is always present; `message`, `data` and `errors` are omitted when empty. Helpers in `internal/handler` are the only writers of this shape.

`/healthz` is deliberately excluded: probes need the smallest possible body and are a different consumer from the API client.
