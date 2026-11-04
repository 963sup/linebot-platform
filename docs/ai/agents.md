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

Roles are behavior guidance, not sandbox or IAM boundaries. The host controls actual permissions. The maximum delegated depth is one and the maximum concurrent child threads is four. Do not spawn an orchestrator child; the primary session coordinates work.

Use one write owner per file. Do not delegate trivial localized changes. Every handoff reports Task, Owner, Inputs, Scope, Files, Contract Changes, Evidence, Tests Run, Findings, Risks and Next Action. Unverified states must be reported as `Not Verified`.

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

- Use `gopls` for Go symbols, references and diagnostics.
- Use Context7 for version-specific third-party SDK documentation.
- Use `openaiDeveloperDocs` only for OpenAI/Codex/MCP/Plugin documentation.
- Use `google_developer_knowledge` for Google Cloud and Workspace documentation.
- Use GitHub MCP for repository, branch, PR and CI evidence.
- MCP declarations do not prove startup, authentication, authorization or tool availability.

## Environment policy

`environment` may repair only pinned local tools, PATH, rebuildable caches and approved Skill files. It must never auto-login OAuth, create credentials, grant IAM, change network policy, deploy resources, write production data or overwrite user-level Codex configuration. It reports `READY`, `DEGRADED` or `BLOCKED` with command evidence.

## Reviewer profiles

`reviewer` supports the profiles `architecture`, `api-contract`, `security`, `performance`, `testing`, `release`, and `incident`. Profiles are checklists and routing metadata, not additional Agent registrations. Do not create `reviewer-*` TOML files.
