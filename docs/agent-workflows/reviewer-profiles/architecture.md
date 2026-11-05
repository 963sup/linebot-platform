# Reviewer Profile: architecture

## Activate when

Packages, ports, DTOs, aggregates, repositories, directory structure, or cross-context contracts change.

## Required checks

- Domain does not import transport, SDKs, databases, adapters, or infrastructure.
- Application owns use-case orchestration and consumer-owned ports.
- Adapters isolate external protocols and SDK types.
- Aggregates preserve invariants and cross-context communication uses explicit contracts.
- No duplicate entry point or abstraction without a consumer.
- For cross-file changes, verify the orchestrator-owned baseline and writer review pack use exactly the same normalized extension-scoped `includePatterns`, and that the review pack was created after implementation, without replacing either artifact. Scope expansion requires a new baseline before further edits. For a read-only task with no change lifecycle, accept one scoped snapshot and `Not Applicable` review fields.
- Reject empty matches, silent full-repository widening, stale HEAD/scoped fingerprints, or a narrower review pattern that can hide new dependencies. Confirm every cited finding against current source or gopls.

## Commands

`go list ./...`, `go test ./tests/architecture/... ./...`, `go run ./cmd/codexcheck`; use gopls references and diagnostics when available.

## Evidence and restrictions

Distinguish target architecture from implemented wiring. Report severity, file/line, violated boundary, consumer impact, correction, and regression test. A reviewer does not modify or move packages, delete files, merge, push, deploy, or change IAM; obsolete empty scaffolds may be identified for an implementation owner to remove. Unavailable references or runtime composition are `Not Verified`.
