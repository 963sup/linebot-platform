# Reviewer Profile: architecture

## Activate when

Packages, ports, DTOs, aggregates, repositories, directory structure, or cross-context contracts change.

## Required checks

- For cross-boundary or repository-wide changes, verify the proportional method evidence from `.agents/skills/repository-engineering-method/SKILL.md` without replacing writer artifacts. Confirm selected Skills were read; Graphify was used only for relationship or blast-radius claims and checked against current source or gopls; external claims cite Context7, a provider MCP or first-party documentation; and applicable DDD, Hexagonal Architecture, Go Idioms, KISS, YAGNI and DRY decisions are explicit. Treat unused capabilities as `Not Applicable`.

- Domain does not import transport, SDKs, databases, adapters, or infrastructure.
- Application owns use-case orchestration and consumer-owned ports.
- Adapters isolate external protocols and SDK types.
- Aggregates preserve invariants and cross-context communication uses explicit contracts.
- No duplicate entry point or abstraction without a consumer.
- When the authorized lifecycle uses Repomix for snapshot completeness or before/after review, verify the orchestrator-owned baseline and writer review pack use exactly the same normalized extension-scoped `includePatterns`, and that the review pack was created after implementation, without replacing either artifact. Scope expansion requires a new baseline before further edits. When Repomix is unnecessary, accept `Not Applicable`; do not manufacture a baseline merely because multiple files changed.
- Reject empty matches, silent full-repository widening, stale HEAD/scoped fingerprints, or a narrower review pattern that can hide new dependencies. Confirm every cited finding against current source or gopls.

## Commands

`go list ./...`, `go test ./tests/architecture/... ./...`, `go run ./cmd/codexcheck`; use gopls references and diagnostics when available.

## Evidence and restrictions

Distinguish target architecture from implemented wiring. Report severity, file/line, violated boundary, consumer impact, correction, and regression test. A reviewer does not modify or move packages, delete files, merge, push, deploy, or change IAM; obsolete empty scaffolds may be identified for an implementation owner to remove. Unavailable references or runtime composition are `Not Verified`.
