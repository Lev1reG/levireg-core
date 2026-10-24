# levireg-core

Backend service for the personal website (`personal-website-static` is the frontend repo, a sibling directory). Go + chi router + zap logger, single binary, no database or third-party services yet.

## Commands

Use the Makefile targets, not raw `go` commands — they load `.env`:

```bash
make run          # go run . with .env exported
make check        # gofmt -l . && go vet ./... && go test ./...  (run before finishing)
make test-v PKG=./internal/server/ RUN=TestHealthz   # single test
```

- `make fmt` rewrites files; `gofmt -l .` in `check` only reports.
- `PORT=9000 make run` does **not** work: `.env` is `-include`d and `export`ed after, so `.env` wins. Edit `.env` to change it.
- `.env` must be plain `KEY=value` (no quotes/multiline) because `make` parses it, not the shell.
- The committed binary `levireg-core` is `make build` output — gitignored, never edit or commit it.

## Layout

- `main.go` — wiring only: config → logger → server, listen, wait for SIGINT/SIGTERM, 10s graceful shutdown.
- `internal/config` — env loading (`PORT` default 8021, `APP_ENV=production` switches to JSON logs).
- `internal/logger` — zap setup plus `RequestLogger` middleware (zap has no HTTP adapter, don't swap in chi's own logger).
- `internal/server` — router assembly and `http.Server` timeouts; `New()` returns the server so tests exercise `srv.Handler` directly.
- `internal/handler` — HTTP handlers.

## Conventions

- New routes go under `r.Route("/api/v1", ...)`. `/healthz` stays outside `/api/v1` so probes don't inherit future API middleware (auth, rate limiting).
- `/healthz` is liveness only; dependency checks belong in a separate `/readyz`.
- Dependencies are passed in as parameters (`New(log, addr)`); no globals or package-level singletons. Tests pass `zap.NewNop()`.
- Handlers are plain `func(w, r)`; keep them testable via `httptest`.

## Agent skills

### Issue tracker

GitHub Issues via `gh` (repo `Lev1reG/levireg-core`); no PR-as-request surface. See `docs/agents/issue-tracker.md`.

### Triage labels

Default five canonical roles. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `GLOSSARY.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.
