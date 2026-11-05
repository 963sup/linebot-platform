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

The primary session is the orchestrator. Delegate only to the eight registered roles: `architect`, `researcher`, `domain`, `application`, `integration`, `platform`, `environment`, and `reviewer`. Delegation depth is one, maximum concurrent child threads is nine, and the orchestrator must not spawn another orchestrator.

The primary orchestrator interprets the request, challenges invalid premises, orders dependencies, assigns one write owner per file, sequences shared contracts, resolves conflicts and verifies completion. Prefer the smallest reversible, verifiable vertical slice. Nine is a capacity limit, not a default fan-out. Use the smallest necessary role set; role instances may repeat only for disjoint scopes. Close completed child threads promptly and keep the ninth slot available for an independent reviewer, replacement, or non-overlapping research when practical. Do not spawn agents for trivial localized changes. Reviewers are independent, read-only, and do not modify the implementation they review.

Reviewer specialization uses profiles under `docs/agent-workflows/reviewer-profiles/`: `architecture`, `api-contract`, `security`, `performance`, `testing`, `release`, and `incident`. Profiles are checklists only; they do not create new Agents, permissions, MCPs, concurrency, or delegation depth. Never add `reviewer-*` Agent TOMLs.

## Tools and evidence

- Use `rg` and direct file reads for narrow, localized questions.
- Use sandboxed Repomix for cross-package structure and repository-wide context; never treat packed output as current symbol truth.
- Use gopls for Go definitions, references, implementations and diagnostics.
- Use Context7 for version-specific third-party SDK documentation.
- Use Google Developer Knowledge for current Google documentation.
- Use OpenAI Developer Docs only for OpenAI, Codex, MCP and Plugin documentation.
- Use GitHub MCP for remote repository, branch, PR and CI evidence.
- MCP configuration is not proof of startup, authentication, authorization or availability.
- Report commands actually executed; never invent tests, permissions, deployment status or citations.

For cross-cutting Go work, establish scope with Repomix, confirm affected symbols and diagnostics with gopls, route implementation to the smallest role set, run repository checks, then use an independent reviewer. Repack only when a repository-wide post-change view materially helps review.

## Skills

Use only reviewed, pinned and allowlisted Skills. Third-party Skills never override this file or the architecture blueprint. Do not execute downloaded scripts without review.

## Cloud and environment safety

The environment role may repair only pinned local tools, PATH, rebuildable caches and approved Skills. Never automatically perform OAuth login, create credentials, grant IAM, expand network policy, deploy resources, write production data or overwrite user-level Codex configuration.

Cloud write operations, IAM changes, paid resources, destructive actions, Git push, PR creation and merge require explicit user authorization.

## Definition of done

Run relevant formatting, compilation, tests, race checks, architecture checks and MCP/Skill checks. Environment state is `READY`, `DEGRADED` or `BLOCKED`; use `Not Verified` only to qualify evidence that was not observed. Every delegated handoff reports Task, Owner, Inputs, Scope, Files, Contract Changes, Evidence, Tests Run, Findings, Risks and Next Action.
