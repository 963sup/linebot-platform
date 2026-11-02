---
name: sdk-integration
description: Discover, verify and integrate a pinned Go SDK in linebot-platform without guessing APIs or exposing credentials.
---

# SDK integration workflow

Use for tasks that call, add or change any SDK module declared in `go.mod`. This skill gives development steps, not permissions to access live services.

1. **Discover narrowly**: read `docs/sdk/catalog.toml` to find the relevant module, provider, import packages, declared vs integrated status, and proposed Factory/Adapter. Read `go.mod` for the authoritative version. Ignore unrelated SDKs.
2. **Verify symbols**: use the downloaded module source, `go list -m -json <module>`, `go list -f '{{.Dir}}' <package>`, `go doc <package>.<Symbol>`, and gopls diagnostics. Cross-check semantics, authentication, limits, permissions, idempotency and breaking changes against upstream documentation. For OpenAI use `openaiDeveloperDocs` when available; otherwise use Context7 and official sources. Never guess method signatures.
3. **Design the boundary**: prefer an existing application Port and Adapter. Create a Port only when a concrete use case needs one. Keep SDK types within `internal/adapter` and `internal/infrastructure`; build typed factories that expose original SDK clients rather than a universal `Call` API.
4. **Implement a vertical slice**: inject dependencies in `internal/bootstrap`; use `internal/infrastructure/httpclient` for owned HTTP clients when the SDK permits injection, add optional telemetry via `observability`, classify safe errors via `resilience`, share clients within their documented lifetime, close owned resources, respect context cancellation, minimize IAM/OAuth scopes, and do not duplicate the SDK's own retry logic. The LINE v8.23.0 `WithContext` method mutates its receiver: do not mutate shared clients. Preserve existing behavior.
5. **Test offline first**: add fake/contract tests for success, invalid configuration, timeout or cancellation, retry/idempotency and data safety as applicable. Run `go run ./cmd/codexcheck`, `make check`, then isolated authorized integration tests only when credentials and services are explicitly available.
6. **Synchronize and report**: update catalog **status** only after an actual runtime integration is implemented, verify the `go.mod` and `go.sum` diff, and update impacted architecture, README, roadmap and provider matrix. State verified results, untested boundaries and remaining work.

Reference: [verification and evidence contract](references/verification.md).

## Non-negotiable boundaries

- `go.mod` is the only source of module versions. Catalog capability tags are discovery hints, not exhaustive API coverage or guarantees of provider permissions.
- A `//go:build dependencies` blank import and a reserved folder do **not** make a runtime client active.
- Do not add credentials, open real accounts, deploy, send LINE pushes or consume billable AI/Cloud APIs without the required authorization.
- Do not treat an SDK timeout as definite remote failure; for non-idempotent calls a retry may duplicate effects.
- If tools or network are unavailable, state that limitation and do not fabricate successful external tests.
