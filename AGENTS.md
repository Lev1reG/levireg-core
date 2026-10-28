# levireg-core

Backend service for the personal website (`personal-website-static` is the frontend repo, a sibling directory). Go + chi router + zap logger, single binary. PostgreSQL via pgx + sqlc (see `docs/adr/0002-postgres-pgx-sqlc.md`), no other third-party services yet.

## Commands

Use the Makefile targets, not raw `go` commands — they load `.env`:

```bash
make run          # go run . with .env exported
make check        # gofmt -l . && go vet ./... && go test -p 1 ./...  (run before finishing)
make test-v PKG=./internal/server/ RUN=TestHealthz   # single test
```

- `make up` / `make down` start/stop the local Postgres (`compose.yaml`). `DATABASE_URL` defaults to it.
- `make migrate-up`, `make migrate-down` (one step), `make migrate-new NAME=add_x` run golang-migrate in Docker against `db/migrations/`.
- `make sqlc` regenerates `internal/db/` from `db/migrations/` + `db/queries/`. Never edit generated files; commit them.
- DB-backed tests need `make up migrate-up` first; they skip themselves if Postgres is unreachable. `-p 1` because packages share tables. `internal/server/auth_test.go` truncates `auth.users` in whatever `DATABASE_URL` points at: never run it against production.
- `make create-superadmin SA_USERNAME=u SA_EMAIL=e SA_NAME=n` (or `go run . create-superadmin -username u -email e -name n`) creates the first Superadmin (password from stdin, same `DATABASE_URL` as the server).
- `make fmt` rewrites files; `gofmt -l .` in `check` only reports.
- `PORT=9000 make run` does **not** work: `.env` is `-include`d and `export`ed after, so `.env` wins. Edit `.env` to change it.
- `.env` must be plain `KEY=value` (no quotes/multiline) because `make` parses it, not the shell.
- The committed binary `levireg-core` is `make build` output — gitignored, never edit or commit it.

## Layout

- `main.go` — wiring only: config → logger → server, listen, wait for SIGINT/SIGTERM, 10s graceful shutdown. Dispatches the `create-superadmin` subcommand (`superadmin.go`) on `os.Args[1]` before the server starts.
- `internal/config` — env loading (`PORT` default 8021, `APP_ENV=production` switches to JSON logs).
- `internal/logger` — zap setup plus `RequestLogger` middleware (zap has no HTTP adapter, don't swap in chi's own logger).
- `internal/server` — router assembly and `http.Server` timeouts; `New()` returns the server so tests exercise `srv.Handler` directly.
- `internal/handler` — HTTP handlers, auth middleware (`Require`, `RequireSuperadmin`), login rate limit, response codes (constants in `response.go`).
- `internal/auth` — argon2id passwords, user validation/creation, session token generation and hashing.
- `internal/db` — sqlc-generated code. `db/migrations` and `db/queries` hold the SQL.

## Conventions

- New routes go under `r.Route("/api/v1", ...)`. `/healthz` stays outside `/api/v1` so probes don't inherit future API middleware (auth, rate limiting).
- `/healthz` is liveness only; dependency checks live in `/readyz`.
- Dependencies are passed in as parameters (`New(log, addr, pool)`, where `pool` is a `server.DB`); no globals or package-level singletons. Tests pass `zap.NewNop()`.
- Routes needing login go inside the `Require` group; Superadmin-only ones add `RequireSuperadmin`. Add new error codes as constants in `internal/handler/response.go`, not string literals.
- Handlers are plain `func(w, r)`; keep them testable via `httptest`.

## Agent skills

### Issue tracker

GitHub Issues via `gh` (repo `Lev1reG/levireg-core`); no PR-as-request surface. See `docs/agents/issue-tracker.md`.

### Triage labels

Default five canonical roles. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `GLOSSARY.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.
