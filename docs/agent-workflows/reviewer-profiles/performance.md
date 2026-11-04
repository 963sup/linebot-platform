# Reviewer Profile: performance

## Activate when

Concurrency, goroutines, HTTP clients, Firestore queries, Cloud Tasks, Cloud Run settings, caching, batching, logging, or hot paths change.

## Required checks

- Context cancellation, deadlines, connection reuse, and resource closure are correct.
- Retry/backoff cannot create storms or duplicate side effects.
- Firestore reads use intentional indexes, limits, pagination, and transaction scope.
- Client reuse is safe and per-user credentials are isolated.
- Telemetry uses low-cardinality labels and avoids sensitive data.
- Claims identify workload, baseline, method, variance, and cost.

## Commands

Run `go test ./... -race`, relevant benchmarks, and `go test ./... -count=1`; use profiles only when authorized.

## Evidence and restrictions

Separate measurements, estimates, and unknowns. Do not add speculative caches, worker pools, scaling changes, or paid load tests. Production throughput, quotas, cold starts, and cost are `Not Verified` without authorized measurements.
