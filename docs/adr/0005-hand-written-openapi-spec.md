# Hand-written OpenAPI spec, enforced by tests

The API spec is a hand-written OpenAPI 3.1 YAML file embedded in the binary and served next to a Scalar-rendered API reference. We chose this over generating the spec from comment annotations (swaggo), which only emits Swagger 2.0 and drifts silently because comments are never compiled, and over a code-first framework (huma), which would replace our plain `func(w, r)` handlers and envelope helpers. Hand-writing alone also drifts, so tests fail the build when a chi route has no matching spec operation (or the reverse) and when the spec's `Code` enum differs from the `Code*` constants in `internal/handler`. Request and response schemas are not checked against Go types; reviewers own that.

Revisit if schema drift between Go types and the spec becomes a recurring bug.
