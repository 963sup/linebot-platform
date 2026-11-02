.PHONY: bootstrap check fmt-check vet test build run-webhook run-ops

bootstrap:
	go mod tidy
	go mod download

check: fmt-check vet test build

fmt-check:
	@files="$$(gofmt -l cmd/ internal/ tests/)"; if [ -n "$$files" ]; then printf 'Go files require gofmt:\n%s\n' "$$files"; exit 1; fi

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
