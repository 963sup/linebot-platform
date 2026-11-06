# Repository-wide Go architecture audit

Trigger this workflow for a request to align or refactor all production Go against DDD and Hexagonal Architecture. The primary orchestrator owns the decision and delegates read-only analysis to `architecture_auditor` and the smallest useful specialist set. Localized changes do not trigger a full audit.

## Full-read gate

1. Record HEAD, dirty-state fingerprint, normalized patterns and baseline output IDs. Call Repomix MCP `pack_codebase` with `includePatterns: **/*.go` and `ignorePatterns: **/*_test.go`; verify the output contains no test files. Separately inventory and read `docs/assets/architecture/**`, `docs/architecture/blueprint.md`, domain manifests and relevant ADRs. The user-facing `/assets/architecture/**` means the repository path `docs/assets/architecture/**`.
2. Read every packed production Go file in bounded contiguous ranges. Record range ownership, total lines, first/last offsets and truncation; repair gaps before proceeding. When agents divide ranges, the orchestrator checks their union covers the full snapshot exactly. Confirm changing facts against current files and gopls; packed output is a snapshot.
3. No code or architecture asset edits before the read and analysis gates complete. If Repomix MCP is unavailable or its output cannot be fully read, report the audit `BLOCKED`; a CLI pack does not count as MCP evidence for this workflow.

## First-principles decision

Derive business purpose, invariants, identity and lifecycle, transaction boundary, bounded-context ownership, and required dependencies. Compare code with architecture assets and the normative blueprint. Visuals and conceptual Mermaid files are non-normative; correct them when they conflict with verified business rules or the blueprint. Classify Aggregate, Entity, VO, Policy, Specification, Factory, Repository and Domain Event by necessary behavior, not by diagram labels. Record each difference, root cause, affected consumers, proposed correction, compatibility impact and evidence. Use Context7 for relevant version-specific claims and approved Skills only when they add evidence.

The orchestrator merges specialist findings into one decision-complete plan, orders shared contracts before consumers, assigns one writer per file, and has `architect` review the plan. Only then may `domain`, `application` and `integration` writers implement in dependency order. Preserve API, persistence, security, consistency and existing valid behavior unless a change is explicitly approved.

## Semantic implementation and acceptance

Discover available gopls MCP/LSP tools before relying on them. Use workspace, package API, diagnostics, search and references for impact analysis; use rename only after checking affected references. Unexposed LSP actions are not assumed available. Use the normal editor for changes gopls cannot perform. After each batch run focused compilation, tests and diagnostics.

Finish with gofmt, `go test ./...`, `go vet ./...`, `go build ./...`, architecture/dependency and contract tests, relevant race and integration checks, gopls diagnostics/references, and a same-pattern Repomix MCP review pack. Compare production LOC, file/package/interface counts, changed-function complexity and measured performance only where a benchmark exists. An independent `reviewer` checks the authorized diff and evidence. Report the difference/root-cause register, unified design, actual changes, before/after metrics, passed and failed checks, and remaining risks. Do not claim external verification that was not performed.
