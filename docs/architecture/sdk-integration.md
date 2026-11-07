# SDK Integration & Codex Zero-Context Development

This design extends the existing Go modular monolith and Hexagonal Architecture. It is not a new SDK framework. The **default** webhook/CLI runtime uses LINE Messaging API. The opt-in durable event mode additionally integrates Firestore and Cloud Tasks with typed adapters and a private worker. Other providers have native factories but are not yet connected to business workflows.

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

The catalog intentionally contains no pinned version fields, no credentials and no exhaustive method inventory. `declared` is dependency-only; `client_ready` requires native Factory source but does not imply credentials or business integration; `integrated` requires both Factory and Adapter source (as with LINE). None of these statuses is a production-readiness certificate.

## Development sequence

Natural-language request → read `AGENTS.md` → discover provider via catalog → confirm version in `go.mod` → inspect actual symbols using Go/gopls → corroborate behavior with upstream docs → reuse or implement Port/Adapter/Factory → write offline tests → run checks → report evidence and synchronize status/docs.

Load only the relevant provider's sources; do not eagerly ingest every SDK or duplicate the official documentation in the repository. No generic dynamic `SDK.Call(provider, method, payload)` API.

## Composition、runtime call 與 source dependency

Composition ownership：`cmd/{webhook,worker,ops}` 將程序組裝委派給 explicit composition root `internal/bootstrap`；Bootstrap 建立所需的 Application use case、Adapter 與 Infrastructure typed client，並注入具體實作。

Runtime call 可以由 Application 經 capability port 呼叫 Adapter，也可以由 Application 協調使用 Domain repository contract；這不表示 Application 或 Domain 在 source code 中 import 具體 Adapter。

Compile-time source dependency 指向 contract owner：Adapter import consuming Application capability 的 port 或 Domain repository contract；Application 與 Domain contract 不 import Adapter。Infrastructure 提供技術 client 與 lifecycle primitive，不 import Application use case、Domain policy 或 Adapter。

- Application: use cases and provider-neutral capability ports; no provider SDK types/imports.
- Domain: business invariants, models and necessary repository contracts; no external provider SDK.
- Adapter: maps SDK-native data and errors to the Application capability port or Domain repository contract it implements.
- Infrastructure: typed SDK factories, credential wiring, transport, telemetry and provider-specific policies. Prefer SDK-native retry/transport mechanisms and avoid double retry.
- Bootstrap: construct only required clients, inject them explicitly, and close owned resources during shutdown or partial-startup failure.

For LINE, existing `internal/infrastructure/line/client_factory.go` and `internal/adapter/outbound/line/messages.go` are the current reference slice. No other Factory is considered implemented merely because its directory exists.

## Phase 2: implemented SDK runtime foundation

The webhook entrypoint calls `bootstrap.NewWebhookRuntime` to compose Firestore and Cloud Tasks, without a LINE sending client. The worker and Push CLI each own their managed LINE HTTP transport and close it after their work drains. No synchronous webhook/router compatibility layer remains.

- `internal/infrastructure/httpclient`: clones Go's default transport per process/SDK Client, reuses idle connections, applies explicit timeout, and supports optional transport decorators. It does not override TLS defaults or automatically retry.
- `internal/infrastructure/observability`: logs provider name, numeric status, duration and transport error flag at debug level. It does **not** log URL, headers, body, credentials, user ID or raw SDK errors.
- `internal/infrastructure/resilience`: categorizes HTTP failure and unknown outcomes without leaking SDK response data in `Error()`, and preserves underlying errors via `Unwrap()` for controlled code inspection. It provides **no global retry loops**.
- `internal/infrastructure/line`: typed native LINE SDK Client factory with injected HTTP Client; the caller owns HTTP transport resources.
- `internal/adapter/outbound/line`: SDK v8.23.0's `WithContext` mutates `MessagingApiAPI.ctx`, so the adapter creates a **per-call shallow copy** before calling `WithContext`. This works for the pinned client structure, prevents shared mutation under parallel requests, and must be revalidated on any SDK upgrade. The HTTP timeout is retained as an upper bound. Reply and Push use the native `WithHttpInfo` endpoints to obtain HTTP status for safe classification.
- `internal/bootstrap/config.go`: optional `LINE_API_TIMEOUT` defaults to 8s, accepts 1s–12s for bounded worker-side LINE calls; public webhook does not send LINE messages.

**No cloud client is constructed by the credential-free local health process.** Enabling webhook messages requires the complete durable configuration and constructs Firestore and Cloud Tasks clients. The Push CLI only constructs its LINE client. The durable path is **not production-ready** without authorized IAM/OIDC, TTL, scheduler and staging validation.

## Phase 3: native provider Client Factories (merged via PR #8)

- `internal/infrastructure/openai` requires explicit API Key and caller-owned HTTP client, and returns `openai.Client` with the native SDK Responses API and retry behavior.
- `internal/infrastructure/google/genai` separates Gemini Developer API key authentication from Vertex AI ADC/workload identity. Vertex does not inject an unauthenticated custom HTTP client.
- `internal/infrastructure/google/cloud/{firestore,cloudtasks}` returns native clients accepting ADC/client options and caller-owned `Close()`. No resources are provisioned by construction.
- Google Tasks is `google.golang.org/api/tasks/v1`, included in the existing `google.golang.org/api`; Google Cloud Tasks is the separately pinned background queue SDK.
- OpenAI, Gemini and MCP remain `client_ready`. Firestore and Cloud Tasks now have explicit outbound Adapters for the optional Stage-4 event pipeline and are listed `integrated` in the Catalog, **not** deployed or staging-verified.

## Phase 4: gated durable LINE webhook and private worker (merged via PR #8)

- `internal/application/usecases/messaging-delivery` defines provider-neutral atomic receipt, SHA-256 delivery IDs, duplicate handling, single-use send claim, terminal `sent`/`expired`/`unknown` outcomes, and recovery orchestration.
- `internal/adapter/outbound/google/cloud/firestore` stores the Inbox and Outbox in one transaction. The original user message is not persisted; the short-lived Reply Token is temporarily stored, cleared at terminal state and subject to separately configured Firestore TTL.
- `internal/adapter/outbound/google/cloud/cloudtasks` enqueues only opaque event keys using an OIDC service account. Because Cloud Tasks retains completed task names for up to 24 hours, task names use an event key plus a deterministic **10-second scheduling bucket**, while Firestore Claim provides the actual send-side idempotency guard.
- Durable mode verifies the LINE raw signature and destination, persists before acknowledgement, and schedules tasks. `cmd/worker` hosts OIDC-authenticated `/tasks/v2/line-reply` and `/tasks/v2/dispatch` endpoints. Worker and scheduled reconcilers must additionally be protected by Cloud Run IAM and private ingress, configured only in an authorized environment.
- Cloud Scheduler cadence may exceed the conservative 45-second LINE Reply window. Expired events are marked `expired`; this is not guaranteed reply delivery or end-to-end exactly-once semantics. Bounded recovery queries still require high-volume pagination/indexing, telemetry, and authorized emulator/staging validation.
- See [event processing](event-processing.md), [staging constraints](../operations/durable-worker.md), and [Firestore retention](../../deploy/firestore/README.md). No GCP resource was deployed or provisioned in this PR.

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
3. **Native provider factories (merged via PR #8)**: Google Cloud, OpenAI, Gemini and MCP native Client Factories have offline tests. Firestore and Cloud Tasks also have opt-in runtime adapters; remaining business adapters and authorized live integrations are pending.
4. **Durable event handling (merged via PR #8, code implemented)**: Firestore inbox/outbox, Cloud Tasks worker, single-use Claim, unknown outcomes and bounded reconciliation. Authorized IAM/OIDC/TTL/scheduler setup and staging proof remain pending.
5. **Authorized staging & production hardening (pending)**: verify IAM/OIDC, TTL, Cloud Scheduler, LINE Sandbox, Codex fresh-session behavior in an independently authorized environment. Prior offline validation is not deployment authorization.

For the current module-by-module code readiness versus deployment readiness, see [main infrastructure status](infrastructure-status.md). Historical references to PR #8 document the merged changes, not a pending pull request.

## Definition of done

`go run ./cmd/codexcheck` validates exact direct-dependency coverage without reading remote documentation; `make check` enforces formatting, vet, tests and build. For every newly activated service, verify its typed client, adapter, safe errors, cancellation, cleanup, failure modes, and any necessary OAuth/IAM. Report unverified external actions explicitly. A passed CI job is not proof of successful Codex Cloud MCP connection or production readiness.
