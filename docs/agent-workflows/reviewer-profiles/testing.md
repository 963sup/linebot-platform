# Reviewer Profile: testing

## Activate when

Tests, fixtures, mocks, fakes, public behavior, persistence, concurrency, or release acceptance change.

## Required checks

- Tests assert behavior and failure modes rather than implementation details.
- Cancellation, timeout, retry, duplicate, malformed, and unknown inputs are covered.
- Security flows cover replay, signature failure, token binding, and redaction.
- Persistence tests cover transactions, conflicts, old data, and idempotency.
- Architecture, contract, and race tests match current boundaries.

## Commands

`go test ./...`, `go test -race ./...`, `go vet ./...`, and `go build ./...`.

## Evidence and restrictions

Record exact commands and pass/fail/skip. A mock is not live provider acceptance. Do not call production services, store real tokens, weaken assertions, or mark skipped tests passed. Unexecuted external scenarios are `Not Verified`.
