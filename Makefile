.PHONY: bootstrap check config-check config-docs dev fmt-check vet test build run-webhook run-worker run-ops

bootstrap:
	go mod tidy
	go mod download

check: config-check fmt-check vet test build

config-check:
	go run ./cmd/configcheck --check

config-docs:
	go run ./cmd/configcheck --write-doc

# No implicit .env loading: make dev starts the safe, credential-free local MVP.
dev:
	APP_ENV=local LINE_EVENT_MODE=sync LINE_CHANNEL_SECRET= LINE_CHANNEL_ACCESS_TOKEN= go run ./cmd/webhook

fmt-check:
	@if [ -n "`gofmt -l cmd/ internal/ tests/`" ]; then printf "Go files require gofmt:\n"; gofmt -d cmd/ internal/ tests/; exit 1; fi

vet:
	go vet ./...

test:
	go test -race -count=1 ./...

build:
	go build ./...

run-webhook:
	go run ./cmd/webhook

run-ops:
	go run ./cmd/ops $(ARGS)

run-worker:
	go run ./cmd/worker
