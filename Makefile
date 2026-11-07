.PHONY: bootstrap check config-check config-docs domain-check domain-docs architecture-assets-check dev fmt-check vet test integration-compile integration-test build run-webhook run-worker run-ops

bootstrap:
	go mod tidy
	go mod download

check: config-check domain-check fmt-check vet test integration-compile build

architecture-assets-check:
	./scripts/render-architecture-assets.sh --check

config-check:
	go run ./cmd/configcheck --check

config-docs:
	go run ./cmd/configcheck --write-doc

domain-check:
	go run ./cmd/domainmodel -mode=check

domain-docs:
	go run ./cmd/domainmodel -mode=write

# No implicit .env loading: make dev starts the safe, credential-free local MVP.
dev:
	APP_ENV=local LINE_EVENT_MODE=durable LINE_MESSAGING_API_CHANNEL_SECRET= LINE_MESSAGING_API_CHANNEL_ACCESS_TOKEN= go run ./cmd/webhook

fmt-check:
	@command -v gofmt >/dev/null 2>&1 || { printf '%s\n' 'gofmt is required for fmt-check' >&2; exit 1; }; \
	files="$$(gofmt -l cmd/ internal/ tests/)" || { printf '%s\n' 'gofmt failed during fmt-check' >&2; exit 1; }; \
	if [ -n "$$files" ]; then printf 'Go files require gofmt:\n%s\n' "$$files"; gofmt -d cmd/ internal/ tests/; exit 1; fi

vet:
	go vet ./...

test:
	go test -race -count=1 ./...

# Compile build-tagged Firestore integration tests without requiring an emulator.
integration-compile:
	go test -tags=integration -run '^_compile_only_' ./tests/integration

# Runs the real Firestore integration suite. The caller must provide a local
# emulator through FIRESTORE_EMULATOR_HOST; unlike integration-compile this
# target must execute test bodies and fails when the emulator is unavailable.
integration-test:
	@test -n "$$FIRESTORE_EMULATOR_HOST" || (printf '%s\n' 'FIRESTORE_EMULATOR_HOST is required' >&2; exit 1)
	go test -tags=integration -race -count=1 ./tests/integration

build:
	go build ./...

run-webhook:
	go run ./cmd/webhook

run-ops:
	go run ./cmd/ops $(ARGS)

run-worker:
	go run ./cmd/worker
