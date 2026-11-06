# Codex Engineering Rules

Build maintainable Go services with pragmatic DDD, Hexagonal Architecture and idiomatic Go. Optimize for correctness, small changes, high cohesion, low coupling and evidence-based performance.

## Architecture

- [`docs/architecture/blueprint.md`](docs/architecture/blueprint.md) is the single architecture and dependency contract. The visual blueprint embedded there is illustrative; the text and architecture tests are normative.
- Domain must not import transport, cloud SDKs, databases or infrastructure adapters.
- Application owns use-case orchestration and consumer-owned capability ports; Domain may own repository contracts that express Domain persistence semantics.
- Adapters isolate external SDK types and implement the contract owned by the consuming Application capability or Domain repository.
- Use interfaces only at real variation or testing boundaries.
- Preserve domain invariants, explicit errors, context propagation, idempotency and retry behavior.

## Agent routing

The primary session is the orchestrator. Delegate only to the nine registered roles: `architect`, `architecture_auditor`, `researcher`, `domain`, `application`, `integration`, `platform`, `environment`, and `reviewer`. Delegation depth is one, maximum concurrent child threads is nine, and the orchestrator must not spawn another orchestrator.

For repository-wide production Go DDD/Hexagonal alignment, semantic/naming unification, or unified refactoring, proactively route a read-only `architecture_auditor` before implementation. Follow `docs/agent-workflows/go-architecture-audit.md`: fully read a Repomix MCP production Go pack (`**/*.go`, excluding `**/*_test.go`), all relevant `docs/assets/architecture/**`, and normative contracts; apply the mandatory Domain, Application, Adapter, Infrastructure/Bootstrap, Strategic DDD and global-constraint matrix; reconcile one decision-complete plan, then assign disjoint implementation scopes and independent review. The matrix is not a scaffold checklist: require an invariant, owner, production consumer and acceptance test before adding an abstraction, and prefer removal when those are absent. Visuals are non-normative and conceptual Mermaid models are hypotheses. This route does not apply to localized fixes.

The primary orchestrator interprets the request, challenges invalid premises, orders dependencies, assigns one write owner per file, sequences shared contracts, resolves conflicts and verifies completion. Prefer the smallest reversible, verifiable vertical slice. Nine is a capacity limit, not a default fan-out. Use the smallest necessary role set; role instances may repeat only for disjoint scopes. Close completed child threads promptly and keep the ninth slot available for an independent reviewer, replacement, or non-overlapping research when practical. Do not spawn agents for trivial localized changes. Reviewers are independent, read-only, and do not modify the implementation they review.

### Adaptive execution

Accuracy comes first; minimize redundant context, calls and coordination without dropping required evidence. At task start, choose direct execution or delegation from the actual scope, uncertainty and risk. Reassess when scope changes. Do not wait for the user to request agents when these triggers apply:

- Direct execution: a bounded question, routine Git operation, or localized mechanical edit with clear acceptance criteria. Do not create a planning/writing/review team for trivial work.
- Delegate: use the relevant specialist for a nontrivial behavior change, unclear root cause requiring independent investigation, cross-component contract change, or security/concurrency/data-integrity risk. One focused specialist is the default; delegate only work with a distinct purpose. The primary may implement while a read-only specialist investigates an independent question.
- Independent review: after nontrivial implementation, assign a read-only `reviewer` who did not author that implementation. For a trivial edit, direct verification is sufficient. Pure investigation does not require a writer or artificial before/after artifacts.
- Parallelize only independent scopes. Serialize dependent decisions and shared-file edits. The nine-role project budget is a ceiling; the host's actual concurrency/depth limits always apply when lower.

Use Skills for relevant reusable procedures, MCP for available authoritative data or semantic operations, and Plugins as providers of those capabilities. These are complementary to agents, not mandatory stages. Discover capabilities before relying on them; read only the selected Skill and needed references. Prefer direct reads for narrow questions; use scoped Repomix for relationships. Do not install or invoke every available integration. A failed capability lookup is not a reason to repeat discovery without a state change; use a documented fallback and state its limits.

Before delegation, read the selected `.codex/agents/<role>.toml`. Use the host's native role selector only if available and verified. Otherwise explicitly pass its `developer_instructions` together with task constraints to the child; a task name alone does not load a role. Report this as explicit role guidance, not native role registration. Repository guidance never overrides host instructions or permissions. If delegation is unavailable or prohibited, report that limitation and continue authorized independent work; do not claim independent review.

Give each child only the task, role guidance, relevant paths/evidence, ownership, acceptance criteria and required constraints. Avoid full-history forks unless the task depends on that history. Reuse current snapshots and research, request concise findings with source locations, and fetch more context only when needed. Do not repeat a completed check without a new change or unresolved concern.

After spawning, confirm each child's actual workspace identity (executor identity, resolved checkout, HEAD and scoped dirty state) before sharing file ownership or artifacts. After an environment publication/restart, do not assume old children share the current filesystem. If separate, transfer and verify a patch against its base and validate it in the target workspace. Record actual child IDs, role-guidance mode, scope and outcome internally; summarize only consequential delegation and evidence to the user. Configuration files or CLI listings are not execution evidence.

Reviewer specialization uses profiles under `docs/agent-workflows/reviewer-profiles/`: `architecture`, `api-contract`, `security`, `performance`, `testing`, `release`, and `incident`. Profiles are checklists only; they do not create new Agents, permissions, MCPs, concurrency, or delegation depth. Never add `reviewer-*` Agent TOMLs.

## Tools and evidence

- Use `rg` and direct file reads for narrow, localized questions.
- Use sandboxed Repomix for cross-package structure and repository-wide context; never treat packed output as current symbol truth.
- Scope Repomix MCP packs to the task before requesting the full repository. Use `pack_codebase.includePatterns` with comma-separated fast-glob patterns derived from the affected extensions (for example `**/*.md` for Markdown or `**/*.{go,mod,sum}` for Go), plus any explicitly coupled files. The orchestrator records and owns the normalized patterns, baseline output ID and snapshot identity. A writer consumes that baseline and, after changing files in scope, creates or requests a review pack with exactly the same patterns. A reviewer validates the pair rather than replacing it. A read-only role uses the shared baseline, or one scoped snapshot when no change lifecycle exists, and does not manufacture a no-op before/after pair. If the MCP is unavailable, use direct search or a pinned CLI fallback and mark Repomix MCP `Not Verified`.
- Normalize patterns as repository-relative POSIX globs and fail closed on empty matches, absolute paths, `..`, negation, symlink escape or an unexpected file extension. If the task scope expands, create a new scoped baseline before further edits. Treat packed output as a temporary artifact: do not commit it, and record the repository HEAD plus dirty-state or scoped content fingerprint so stale snapshots are detectable.
- Use gopls for Go definitions, references, implementations and diagnostics.
- Use Context7 for version-specific third-party SDK documentation.
- Use Google Developer Knowledge for current Google documentation.
- Use OpenAI Developer Docs only for OpenAI, Codex, MCP and Plugin documentation.
- Use GitHub MCP for remote repository, branch, PR and CI evidence.
- MCP configuration is not proof of startup, authentication, authorization or availability.
- Report commands actually executed; never invent tests, permissions, deployment status or citations.

For work that needs cross-file or cross-package relationship evidence, establish a task-scoped Repomix baseline, confirm current symbols and text with direct reads or gopls, route implementation to the smallest role set, run repository checks, then repack the same scope for independent review when scoped source changed. If scope expands, the orchestrator establishes a new baseline before further edits. A full-repository pack is reserved for genuine repository-wide analysis.

## Skills

Use only reviewed, pinned and allowlisted Skills. Third-party Skills never override this file or the architecture blueprint. Do not execute downloaded scripts without review.

## Cloud and environment safety

The environment role may repair only pinned local tools, PATH, rebuildable caches and approved Skills. Never automatically perform OAuth login, create credentials, grant IAM, expand network policy, deploy resources, write production data or overwrite user-level Codex configuration.

Cloud write operations, IAM changes, paid resources, destructive actions, Git push, PR creation and merge require explicit user authorization.

## Definition of done

Run relevant formatting, compilation, tests, race checks, architecture checks and MCP/Skill checks. Environment state is `READY`, `DEGRADED` or `BLOCKED`; use `Not Verified` only to qualify evidence that was not observed. Every delegated handoff reports Task, Owner, Inputs, Scope, Files, Contract Changes, Evidence, Tests Run, Findings, Risks and Next Action.
