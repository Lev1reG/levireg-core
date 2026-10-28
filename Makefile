APP := levireg-core

# .env must be plain KEY=value (no quotes, no multi-line): make parses it, not
# the shell, so quotes end up inside the value.
# shortcut: no multi-line secrets via .env, switch to godotenv in main.go then.
# Values here win over the caller's shell, so `PORT=9000 make run` still uses .env.
-include .env
export

DATABASE_URL ?= postgres://levireg:levireg@localhost:5432/levireg?sslmode=disable

SQLC_IMAGE := sqlc/sqlc:1.31.1
MIGRATE_IMAGE := migrate/migrate:v4.20.1
# --network host so the migrate container reaches the compose Postgres on localhost.
MIGRATE := docker run --rm --network host -v $(CURDIR)/db/migrations:/migrations $(MIGRATE_IMAGE) -path /migrations -database "$(DATABASE_URL)"

.PHONY: run build test test-v fmt check clean up down sqlc migrate-up migrate-down migrate-new

run:
	go run .

build:
	go build -o $(APP) .

test:
	go test -p 1 ./...

test-v:
	go test $(PKG) -run $(RUN) -v

fmt:
	gofmt -w .

check:
	gofmt -l .
	go vet ./...
	go test -p 1 ./...

clean:
	rm -f $(APP)

up:
	docker compose up -d --wait

down:
	docker compose down

sqlc:
	docker run --rm --user $(shell id -u):$(shell id -g) -v $(CURDIR):/src -w /src $(SQLC_IMAGE) generate

migrate-up:
	$(MIGRATE) up

migrate-down:
	$(MIGRATE) down 1

migrate-new:
	docker run --rm --user $(shell id -u):$(shell id -g) -v $(CURDIR)/db/migrations:/migrations $(MIGRATE_IMAGE) create -ext sql -dir /migrations -seq $(NAME)
