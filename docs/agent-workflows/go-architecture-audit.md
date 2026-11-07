# Repository-wide Go architecture audit

Trigger this workflow for a request to align or refactor all production Go against DDD and Hexagonal Architecture. The primary orchestrator owns the decision and delegates read-only analysis to `architecture_auditor` and the smallest useful specialist set. Localized changes do not trigger a full audit.

## Canonical natural-language trigger

可直接使用以下指令：

> 使用 `architecture_auditor` 透過 Repomix MCP 完整分析 production `.go`（排除 `*_test.go`），對照 `docs/assets/architecture/**`、Blueprint 與 Domain manifest，一次找出所有架構、領域語意、抽象、型別、介面、package、函數與命名不一致。先完整盤點與規劃，不要邊讀邊改；將既有項目分類為 `Keep`、`Rename`、`Merge`、`Delete` 或 `Exception`，產出統一且可執行的修正計畫，再由主代理依計畫實施、驗證，並以測試、gopls與同範圍 Repomix review 確保儲存庫完全對齊。

This trigger covers architecture, domain semantics, abstractions, types, interfaces, packages, functions, and naming. The auditor remains read-only; only the primary orchestrator implements the reviewed plan.

## Capability orchestration

Read `.agents/skills/repository-engineering-method/SKILL.md` before the audit. Repomix MCP owns complete snapshot evidence; Graphify explores package and symbol relationships, callers, tests and blast radius; gopls and current source confirm symbol truth. Select and fully read applicable pinned `samber/cc-skills-golang` procedures for Go Idioms and applicable `samber/cc-skills` procedures for general engineering decisions. Installation or Plugin registration is not evidence of use. Context7 verifies version-specific third-party Go or SDK claims, while provider MCPs or first-party documentation verify provider contracts; neither source decides Domain ownership.

Apply DDD, Hexagonal Architecture, Go Idioms, KISS, YAGNI and DRY to every non-local decision, marking an irrelevant lens `Not Applicable` rather than inventing a finding. Assign Agents and subagents by Bounded Context, layer or operational owner only after shared facts and boundaries are established. Use the smallest non-overlapping role set and retain an independent reviewer.

## Full-read gate

### Accepted execution fallbacks

Repomix MCP and the mise-pinned Repomix CLI are accepted snapshot transports for this workflow. All references below to MCP packs also allow CLI packs with the same normalized include/exclude patterns. Record the tool version, command, repository HEAD, dirty-state fingerprint, artifact path and SHA-256; verify the packed path set against the intended source inventory, including any files omitted by ignore or security rules. Do not disable security checks to force completeness. Record MCP evidence as `Not Verified` when using the CLI. Use the same patterns for the final review pack and account for added or deleted files.

When delegation is unavailable or prohibited by the host, the primary agent performs the auditor's full analysis and decision matrix, prepares the plan, reviews that plan in a separate pass, then implements and reviews the final diff sequentially. References to `architecture_auditor`, `architect`, implementation agents and an independent `reviewer` below describe the preferred delegated route, not prerequisites in this fallback. Explicitly report that self-review is not independent review. Preserve all source-reading, compatibility and verification gates; never claim an agent or MCP was used when it was not.

1. Record HEAD, dirty-state fingerprint, normalized patterns and baseline output IDs. Call Repomix MCP `pack_codebase` with `includePatterns: **/*.go` and `ignorePatterns: **/*_test.go`; verify the output contains no test files. Separately inventory and read `docs/assets/architecture/**`, `docs/architecture/blueprint.md`, domain manifests and relevant ADRs. The user-facing `/assets/architecture/**` means the repository path `docs/assets/architecture/**`.
2. Read every packed production Go file in bounded contiguous ranges. Record range ownership, total lines, first/last offsets and truncation; repair gaps before proceeding. When agents divide ranges, the orchestrator checks their union covers the full snapshot exactly. Confirm changing facts against current files and gopls; packed output is a snapshot.
3. No code or architecture asset edits before the read and analysis gates complete. If neither accepted snapshot transport can produce a complete pack, or the output cannot be fully read, report the audit `BLOCKED` with the missing evidence. A verified CLI pack satisfies this gate without being reported as MCP evidence.

## First-principles decision

Derive business purpose, invariants, identity and lifecycle, transaction boundary, bounded-context ownership, and required dependencies. Compare code with architecture assets and the normative blueprint. Visuals and conceptual Mermaid files are non-normative; correct them when they conflict with verified business rules or the blueprint. Use Context7 for relevant version-specific claims and approved Skills only when they add evidence.

## Mandatory architecture decision matrix

The auditor applies all six groups below to every production package and reported difference. These are available building blocks, not a scaffold checklist. A missing component is a defect only when a verified invariant, owner, production consumer and acceptance test require it. A component without necessary behavior or a production consumer is a removal candidate.

| Group | Composition to classify | Constraints to enforce |
| --- | --- | --- |
| Domain | Aggregate, Aggregate Root, Entity, Value Object, Domain Service, Policy, Specification, Factory, Repository Interface, Domain Event, Domain Error | Own business meaning, invariants, transitions and consistency. Do not depend on Application, Adapters, Infrastructure, transports, databases or provider SDKs. Do not create a Repository, Factory or Service for every Entity. |
| Application | Use Case, Application Service, Command, Query, input/output DTO, Input Port, Output Port, Unit of Work, Transaction Boundary, Event Handler, Authorization | Orchestrate Domain behavior without duplicating it. Depend on Domain and minimal consumer-owned ports, not concrete databases, frameworks or SDKs. Do not force CRUD through buses or handler layers. |
| Adapters | HTTP, webhook, messaging, CLI, scheduler/job handlers; Repository, API client and event publisher/consumer adapters; mapper, transport DTO and error translator | Inbound adapters call Application. Outbound adapters implement a consuming Application capability port or Domain repository. Keep SDK, storage and wire types outside the core. Do not add mapper layers for direct data transfer. |
| Infrastructure and Bootstrap | Database/client configuration, persistence technology, messaging, cache, external SDK, HTTP server/client, configuration, logger, metrics/tracing, authentication integration, transaction implementation, runtime/deployment, dependency assembly | Technical primitives do not decide business policy. In this repository `cmd/*` and `internal/bootstrap` own composition roots; `internal/infrastructure` must not absorb Bootstrap, Application, Adapter or Domain responsibilities. Do not wrap a directly usable standard-library primitive without lifecycle, policy or test value. |
| Strategic DDD | Domain, Subdomain, Bounded Context, Ubiquitous Language, Context Map, Integration Contract, Anti-Corruption Layer | Establish a business boundary and model owner before changing modules. Do not divide by provider API or table. Contexts do not share mutable models; cross-context behavior uses explicit contracts. Candidate Contexts are not promoted by packages, diagrams, SDKs or tests alone. |
| Global | Dependency Rule, high cohesion, low coupling, encapsulation, explicit ownership, Go idioms, KISS, YAGNI, DRY | Dependencies point inward; interfaces stay small and consumer-owned; errors and ownership are explicit. Remove accidental duplication and unused indirection, but do not abstract incidental similarity or optimize for line count. |

Repository-specific interpretation is mandatory:

- An unused Factory, Repository or Domain Event with no production consumer or necessary behavior must be proposed for removal; do not add components to complete a pattern.
- Application capability ports remain inside `internal/application/usecases/<capability>/`; never propose a global Application ports bucket.
- A Domain repository expresses Aggregate persistence semantics. An Application workflow store or external capability remains owned by its consuming capability.
- Repository implementations are outbound adapters; database connections, SDK clients, configuration and lifecycle are Infrastructure concerns.
- Provider authentication and token verification are integration boundaries. Use-case authorization remains an Application decision expressed in business terms.
- Authority order is: `docs/architecture/blueprint.md` and architecture tests for dependency rules; `docs/domain/model.yaml`, accepted ADRs and executable domain tests for model semantics; API/persistence contracts and verified production behavior for compatibility; `docs/assets/architecture/**` last as non-normative views. Correct the lower-authority source when evidence conflicts.

For each finding report: layer/component, business invariant, current evidence, mismatch, root cause, owner, smallest correction, compatibility impact, acceptance tests and residual risk. Explicitly report `Not Applicable` when a building block is unnecessary; absence alone is not a finding.

The orchestrator merges specialist findings into one decision-complete plan, orders shared contracts before consumers, assigns one writer per file, and has `architect` review the plan. Only then may `domain`, `application` and `integration` writers implement in dependency order. Preserve API, persistence, security, consistency and existing valid behavior unless a change is explicitly approved.

## Semantic implementation and acceptance

Discover available gopls MCP/LSP tools before relying on them. Use workspace, package API, diagnostics, search and references for impact analysis; use rename only after checking affected references. Unexposed LSP actions are not assumed available. Use the normal editor for changes gopls cannot perform. After each batch run focused compilation, tests and diagnostics.

Finish with gofmt, `go test ./...`, `go vet ./...`, `go build ./...`, architecture/dependency and contract tests, relevant race and integration checks, gopls diagnostics/references, and a same-pattern Repomix MCP review pack. Compare production LOC, file/package/interface counts, changed-function complexity and measured performance only where a benchmark exists. An independent `reviewer` checks the authorized diff and evidence. Report the difference/root-cause register, unified design, actual changes, before/after metrics, passed and failed checks, and remaining risks. Do not claim external verification that was not performed.

Final evidence records Graphify graph freshness and material queries when relationship claims were made, selected Skill provenance and fingerprints, the six design-lens outcomes, external documentation sources, delegated ownership and accepted fallbacks. A configured Plugin, MCP or Agent is never reported as used without an observed call or handoff.
