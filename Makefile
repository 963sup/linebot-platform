.PHONY: bootstrap check fmt-check vet test build run-webhook run-worker run-ops

bootstrap:
	go mod tidy
	go mod download

check: fmt-check vet test build

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
