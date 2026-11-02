# SDK Integration & Codex Zero-Context Development

This design extends the existing Go modular monolith and Hexagonal Architecture. It is not a new SDK framework. Current runtime support is limited to LINE Messaging API; the other service SDKs remain declared until real clients, adapters and tests exist.

## Source of truth and workflow

| Artifact | Authority |
|---|---|
| `go.mod` / `go.sum` | Resolved module requirements and integrity; only source of module versions |
| `docs/sdk/catalog.toml` | Direct-dependency discovery, provider routing, representative package imports, capability *hints* and integration status |
| `.agents/skills/sdk-integration/SKILL.md` | Codex's on-demand development process; not runtime code or tool authorization |
| `AGENTS.md` | Architecture, security and tool-routing requirements |
| Go toolchain, gopls, SDK source | Exact exported symbols, signatures and compile-time contracts |
| Official provider documentation | API semantics, permissions, quotas and changes |
| `cmd/codexcheck`, tests and CI | Mechanically enforced repository constraints |

The catalog intentionally contains no pinned version fields, no credentials and no exhaustive method inventory. `declared` only indicates a dependency in `go.mod`—not initialized clients or live API access. `integrated` indicates source-level use; it is **not** a production readiness certificate.

## Development sequence

Natural-language request → read `AGENTS.md` → discover provider via catalog → confirm version in `go.mod` → inspect actual symbols using Go/gopls → corroborate behavior with upstream docs → reuse or implement Port/Adapter/Factory → write offline tests → run checks → report evidence and synchronize status/docs.

Load only the relevant provider's sources; do not eagerly ingest every SDK or duplicate the official documentation in the repository. No generic dynamic `SDK.Call(provider, method, payload)` API.

## Runtime dependency direction

`cmd/{webhook,worker,ops}` → `internal/bootstrap` (explicit composition root) → application use cases and provider adapters → infrastructure-owned typed SDK clients.

- Domain: business invariants and models, no external provider SDK.
- Application: use cases and provider-neutral Ports; no provider SDK types/imports.
- Adapter: maps SDK-native data and errors to application contracts.
- Infrastructure: typed SDK factories, credential wiring, transport, telemetry and provider-specific policies. Prefer SDK-native retry/transport mechanisms and avoid double retry.
- Bootstrap: construct only required clients, inject them explicitly, and close owned resources during shutdown or partial-startup failure.

For LINE, existing `internal/infrastructure/line/client_factory.go` and `internal/adapter/outbound/line/messages.go` are the current reference slice. No other Factory is considered implemented merely because its directory exists.

## Phase 2: implemented SDK runtime foundation

The webhook entrypoint now calls `bootstrap.NewWebhookRuntime` to compose only the LINE provider. It owns a `httpclient.Managed` transport and closes idle connections **after** the inbound HTTP server shuts down. The existing `NewWebhookRouter` signature remains available for compatibility; new long-running entrypoints should use the runtime API for explicit ownership. The Push CLI builds and closes its own managed transport.

- `internal/infrastructure/httpclient`: clones Go's default transport per process/SDK Client, reuses idle connections, applies explicit timeout, and supports optional transport decorators. It does not override TLS defaults or automatically retry.
- `internal/infrastructure/observability`: logs provider name, numeric status, duration and transport error flag at debug level. It does **not** log URL, headers, body, credentials, user ID or raw SDK errors.
- `internal/infrastructure/resilience`: categorizes HTTP failure and unknown outcomes without leaking SDK response data in `Error()`, and preserves underlying errors via `Unwrap()` for controlled code inspection. It provides **no global retry loops**.
- `internal/infrastructure/line`: typed native LINE SDK Client factory with injected HTTP Client; the caller owns HTTP transport resources.
- `internal/adapter/outbound/line`: SDK v8.23.0's `WithContext` mutates `MessagingApiAPI.ctx`, so the adapter creates a **per-call shallow copy** before calling `WithContext`. This works for the pinned client structure, prevents shared mutation under parallel requests, and must be revalidated on any SDK upgrade. The HTTP timeout is retained as an upper bound. Reply and Push use the native `WithHttpInfo` endpoints to obtain HTTP status for safe classification.
- `internal/bootstrap/config.go`: optional `LINE_API_TIMEOUT` defaults to 8s, accepts 1s–12s to remain within the current synchronous webhook HTTP server's 15s write deadline.

**No new providers have been activated.** Google Cloud, Workspace, OpenAI and Gemini are still listed as `declared` in the SDK Catalog. Background queue, durable deduplication and production-grade webhook behavior remain out of scope.

## Security and reliability constraints

- Never commit provider keys or pass production credentials to Codex or CI. Use least-privilege ADC/workload identity, scoped OAuth or SDK-native credentials when a service is actually enabled.
- LINE Webhook must validate the raw request body signature before processing. Its current synchronous handler is an MVP, not durable event processing.
- Remote timeout might mean *unknown outcome*, not failure. Retry only with an operation-appropriate idempotency key and a verified provider policy.
- Do not double-apply generic retries on top of SDK defaults. Treat quotas, concurrency and cancellation per provider.
- Log operational metadata without tokens, raw messages or private prompts.
- Integration tests require isolated, explicitly authorized staging; catalog/Go compilation does not grant API access.

## Phased rollout

1. **Discovery foundation (PR #8, completed slice)**: catalog, one shared skill, automated catalog validation, architecture guard, synchronized docs. No new live SDK calls.
2. **Typed client infrastructure (PR #8, completed LINE slice)**: typed factory injection, owned transport, configuration, logging metadata, safe error semantics and request-context isolation; reusable base for future providers. Verify CI and authorized staging separately.
3. **Provider integrations**: Google Cloud, Google Workspace, OpenAI and Gemini clients and adapters in small independent slices; mark catalog entries integrated only after evidence.
4. **Durable event handling**: Firestore inbox/outbox, Cloud Tasks worker, idempotency and uncertain send outcomes.
5. **Staging acceptance**: run real SDK contract tests with authorized test identities, validate Codex zero-context tasks in a fresh environment, observe and document any gaps.

## Definition of done

`go run ./cmd/codexcheck` validates exact direct-dependency coverage without reading remote documentation; `make check` enforces formatting, vet, tests and build. For every newly activated service, verify its typed client, adapter, safe errors, cancellation, cleanup, failure modes, and any necessary OAuth/IAM. Report unverified external actions explicitly. A passed CI job is not proof of successful Codex Cloud MCP connection or production readiness.
