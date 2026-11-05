# Codex V3 Agent Governance

The primary Codex session is the `orchestrator`. It may delegate to exactly these eight project roles:

| Role | Responsibility |
| --- | --- |
| `architect` | Go DDD, Hexagonal boundaries, ports and ADRs |
| `researcher` | Current Go, SDK, CLI, Google, OpenAI and Codex documentation |
| `domain` | Entities, aggregates, value objects, events and invariants |
| `application` | Use cases, commands, queries, DTOs and consumer-owned ports |
| `integration` | External APIs, persistence, messaging and SDK adapters |
| `platform` | Cloud Run, IAM, configuration, observability and release planning |
| `environment` | Toolchain, MCP, approved Skills and Codex Cloud health |
| `reviewer` | Independent Go, architecture, security, performance and test review |

Roles are behavior guidance, not sandbox or IAM boundaries. The host controls actual permissions. The maximum delegated depth is one and the maximum concurrent child threads is nine. Do not spawn an orchestrator child; the primary session coordinates work.

The primary orchestrator interprets requests, orders dependencies, assigns one write owner per file, sequences shared contracts, resolves conflicts and verifies completion. Nine is a capacity limit, not a required fan-out. Use the smallest necessary role set; repeated role instances require disjoint scopes. Keep reviewers independent and read-only, close completed threads promptly, and reserve the ninth slot for review, replacement, or non-overlapping research when practical. Do not delegate trivial localized changes. Every handoff reports Task, Owner, Inputs, Scope, Files, Contract Changes, Evidence, Tests Run, Findings, Risks and Next Action. Unverified states must be reported as `Not Verified`.

## Routing

| Task | Primary role | Review |
| --- | --- | --- |
| Domain rule or aggregate | `domain` | `architect`, `reviewer` |
| Use case or port | `application` | `architect`, `reviewer` |
| SDK, Firestore, LINE, Google or task adapter | `integration` | `researcher`, `reviewer` |
| Cloud or environment configuration | `platform` / `environment` | `reviewer` |
| Unknown API or version | `researcher` | implementing role |
| Cross-package semantic change | `architect` then `application`/`integration` | `reviewer` |

## Evidence policy

- Use direct search for local questions and sandboxed Repomix only when cross-package structure or repository-wide context is needed.
- For extension-focused work, call Repomix MCP `pack_codebase` with comma-separated fast-glob `includePatterns`: Markdown uses `**/*.md`, Go uses `**/*.{go,mod,sum}`, shell uses `**/*.sh`, and mixed contracts add each required pattern explicitly. Do not pack unrelated extensions by default.
- Patterns must be repository-relative POSIX globs. Empty matches, absolute paths, `..`, leading negation, symlink escape or files outside the declared extension set are `BLOCKED`; never silently widen to a full-repository pack. When scope expands, establish a new baseline before additional edits.
- Use `gopls` for Go symbols, references and diagnostics.
- Use Context7 for version-specific third-party SDK documentation.
- Use `openaiDeveloperDocs` only for OpenAI/Codex/MCP/Plugin documentation.
- Use `google_developer_knowledge` for Google Cloud and Workspace documentation.
- Use GitHub MCP for repository, branch, PR and CI evidence.
- MCP declarations do not prove startup, authentication, authorization or tool availability.

Repomix evidence follows the task lifecycle rather than forcing every role to create its own pair:

- The orchestrator creates and owns the immutable task-scoped baseline identity and normalized `includePatterns`, then shares them with delegated roles.
- A writer confirms current source, changes only its assigned scope, runs repository checks, and creates or requests a post-change review pack with exactly the baseline patterns. Scope expansion requires a new orchestrator-owned baseline before further edits.
- A reviewer does not replace either artifact. It checks ordering, identity, pattern equality, freshness and the changed contract, then confirms every finding against current source or gopls.
- A read-only role consumes the shared baseline. If its task has no change lifecycle, one scoped snapshot is sufficient; it must not create a no-op before/after pair merely to fill handoff fields. A role that can be read-only or a writer declares which lifecycle applied.

Handoffs record the applicable status, patterns, output IDs, metrics, HEAD, dirty-state or scoped fingerprint, artifact hash and stale status. If Repomix or gopls is unavailable, use direct search or pinned CLI/Go fallbacks and report that tool as `Not Verified`; a packed snapshot never overrides current source or gopls. Full-repository packs are reserved for genuinely cross-extension repository audits.

## Environment policy

`environment` may repair only pinned local tools, PATH, rebuildable caches and approved Skill files. It must never auto-login OAuth, create credentials, grant IAM, change network policy, deploy resources, write production data or overwrite user-level Codex configuration. It reports `READY`, `DEGRADED` or `BLOCKED` with command evidence.

## Reviewer profiles

`reviewer` supports the profiles `architecture`, `api-contract`, `security`, `performance`, `testing`, `release`, and `incident`. Profiles are checklists and routing metadata, not additional Agent registrations. Do not create `reviewer-*` TOML files.
