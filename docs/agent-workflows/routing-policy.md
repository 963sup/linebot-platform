# Agent Routing Policy

The primary session is the orchestrator. Use the smallest role set that can complete the task. Use `architect` for boundaries, `researcher` for current documentation, `domain` for invariants, `application` for use cases and ports, `integration` for external adapters, `platform` for cloud planning, `environment` for local/Codex Cloud tool health, and `reviewer` for independent validation. Maximum delegated depth is one and maximum concurrent child threads is nine.

Nine is a technical capacity limit, not the default fan-out. The eight roles may have multiple instances only when their scopes do not overlap. Assign one write owner per file, sequence waves that share files or contracts, keep reviewers independent and read-only, and close completed child threads promptly. Keep the ninth slot available for review, replacement, or non-overlapping research when practical. Host scheduling and resource limits still require runtime verification.

## Reviewer profiles

Profiles extend the fixed `reviewer` role and are not Agent registrations:

| Profile | Use when |
| --- | --- |
| `architecture` | DDD, Hexagonal boundaries, dependency direction or package structure changes |
| `api-contract` | HTTP, Webhook, LINE, OAuth callback, DTO or compatibility changes |
| `security` | HMAC, OAuth, tokens, secrets, IAM, OIDC or data isolation changes |
| `performance` | Latency, concurrency, throughput, retry, query or observability changes |
| `testing` | Test strategy, regression, race, contract or architecture verification |
| `release` | CI, build, artifact, release readiness, rollback or Git delivery |
| `incident` | Production-like failure triage and recovery planning |

Use `reviewer(profile=security)` or combine only relevant profiles, such as `reviewer(profile=architecture,security,testing)`. Profiles do not add permissions, concurrency, recursion depth or write access. See `reviewer-profiles/` for required checks and evidence rules.

