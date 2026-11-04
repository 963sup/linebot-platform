# Reviewer Profile: api-contract

## Activate when

Routes, handlers, raw request processing, external payloads, DTOs, headers, status codes, or public behavior change.

## Required checks

- Method, path, status, content type, and error shape are intentional.
- Webhook raw body is preserved for signature verification and bounded before processing.
- SDK DTOs do not cross into Domain or Application.
- Retries, idempotency keys, event IDs, timeouts, and partial failures are explicit.
- Tests cover valid, invalid, replayed, oversized, cancelled, and unknown inputs.

## Commands

`go test ./tests/contract/... ./internal/adapter/...`, `go vet ./...`, `go run ./cmd/codexcheck`.

## Evidence and restrictions

Record source version, payload, observed response, and untested consumer. Do not send real messages, call production OAuth, change channel settings, or modify external schemas. Live provider behavior is `Not Verified` unless authorized and executed.
