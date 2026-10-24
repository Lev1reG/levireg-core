APP := levireg-core

# .env must be plain KEY=value (no quotes, no multi-line): make parses it, not
# the shell, so quotes end up inside the value.
# shortcut: no multi-line secrets via .env, switch to godotenv in main.go then.
# Values here win over the caller's shell, so `PORT=9000 make run` still uses .env.
-include .env
export

.PHONY: run build test test-v fmt check clean

run:
	go run .

build:
	go build -o $(APP) .

test:
	go test ./...

test-v:
	go test $(PKG) -run $(RUN) -v

fmt:
	gofmt -w .

check:
	gofmt -l .
	go vet ./...
	go test ./...

clean:
	rm -f $(APP)
