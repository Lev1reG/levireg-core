# Postgres via pgx and sqlc, no ORM

Persistence is PostgreSQL accessed through `pgx/v5` (`pgxpool`) with queries written as plain SQL and turned into typed Go by sqlc. We chose this over an ORM (GORM, ent) because the SQL stays visible and reviewable, there is no runtime query builder to learn or debug, and compile-time generated code catches schema drift. The cost is writing each query by hand.

Schema is owned by golang-migrate files in `db/migrations/`; sqlc reads that same directory, so there is one source of truth for the schema. sqlc and migrate run as pinned Docker images through Makefile targets, so no contributor installs either binary. Generated code in `internal/db/` is committed.
