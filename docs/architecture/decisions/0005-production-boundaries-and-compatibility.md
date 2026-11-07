# ADR 0005: Production boundaries and additive reporting compatibility

- Status: Accepted
- Date: 2026-11-06
- Scope: repository implementation and compatibility; no deployment authorization

## Context

The complete production architecture audit found candidate Domain event queues with no production readers, test-only Application wrappers, and LINE report mappings that discarded nested provider data. Existing CLI JSON fields are externally observable even where their names or values are incomplete. SDK client readiness is also an explicit catalog responsibility distinct from a deployed business workflow.

## Decision

Keep the tactical aggregate behavior authorized by ADR 0003 and its candidate strategic status. Remove unconsumed candidate event queues and test-only orchestration. This does not remove live transitions, Identity events or Identity's transactional outbox. A Domain aggregate does not require a separate repository unless production persistence semantics require it; declared repository contracts must still resolve and express Domain ownership.

Keep consumer-owned reporting DTOs and method signatures. Correct missing provider data through additive fields. The following compatibility exceptions are owned by the indicated Application capabilities:

| Owner | Retained contract | Accurate additive representation | Exit condition and acceptance |
| --- | --- | --- | --- |
| insights-reporting | Follower `CountReport.Accepted`, `Delivered`, `Failed` continue to mean targeted reaches, followers, blocks respectively. | `TargetedReaches`, `Followers`, `Blocks`, `CalculationStatus`. | Remove legacy projections only with an explicitly versioned CLI migration; nonzero pinned-SDK fixtures assert both old values and new fields. |
| membership | `SubscriptionView.Membership`, `Status`, `ValidUntil` retain their previous zero values; recipient and observation time remain. | `Subscriptions` contains nullable plan and subscriber projections for every returned subscription. | Remove misleading singular fields only with an explicitly versioned CLI migration; zero/one/multiple subscription fixtures assert old fields, optional nulls and all plan/subscriber data. |
| insights-reporting | Existing message-event top-level fields retain their previous values. | Nested provider counters are represented by metric points with date, metric and area/index identity as applicable. | A versioned CLI migration is required to remove old fields; nested overview/message/click fixtures verify preservation and added data. |

Provider subscription plans carry observed price/currency, not a financial calculation contract. Joined timestamps use UNIX seconds; next billing dates retain the provider's calendar date string in UTC+9 and are not reinterpreted as expiry instants. Provider data contains no subscription status or validity deadline; the adapter must not invent either. Demographic ratios retain fractional provider percentages and availability; absence does not mean a measured zero.

Catalog-ready technical factories remain explicit supported integration boundaries even where no current business workflow consumes them. Their owner is Infrastructure and the SDK catalog; readiness is not evidence of integration or deployment. Remove such a factory only when the catalog capability is deliberately retired or replaced, with constructor/configuration and lifecycle tests updated. This exception does not authorize test-only convenience wrappers that discard lifecycle ownership, generic provider routers or unused Domain scaffolding.

The retained factory exceptions have these concrete owners and acceptance evidence. The common exit condition is an approved retirement or replacement of the corresponding SDK catalog capability; a missing business consumer alone does not trigger retirement.

| Infrastructure owner / factory | Compatibility reason | Acceptance evidence |
| --- | --- | --- |
| `internal/infrastructure/google/cloud/storage.NewClient` | Preserve the catalog's Cloud Storage client-ready integration boundary. | `tests/contract/google_cloud_clients_test.go` and SDK catalog/retention checks. |
| `internal/infrastructure/google/genai.NewGeminiClient`, `NewVertexClient` | Preserve explicitly configured Gemini/Vertex backends. | `client_factory_test.go`: `TestNewGeminiClientValidatesAndKeepsBackendExplicit`, `TestNewVertexClientRequiresProjectAndLocation`; SDK catalog checks. |
| `internal/infrastructure/openai.NewClient` | Preserve the configured OpenAI client-ready integration boundary. | `client_factory_test.go`: `TestNewClientValidatesConfigurationWithoutNetwork`; SDK catalog checks. |
| `internal/infrastructure/mcp.NewClient` | Preserve the catalog's native MCP client integration boundary. | `cmd/codexcheck`: `TestClientReadyRequiresFactorySourceNotAdapter`, `TestReservedSDKInventoryMatchesRepository`, `TestReservedSDKModulesCannotBeSilentlyPruned`; package compilation. No MCP runtime/lifecycle integration test is claimed. |

## Alternatives and consequences

Breaking report return types or silently reassigning legacy fields would force an unplanned CLI migration. Retaining only the legacy fields would continue losing provider facts. Additive projection preserves current consumers while making accurate data available; consumers requiring a strict fixed JSON field set must tolerate the documented additive evolution.

Conceptual architecture assets remain non-normative hypotheses. They cannot promote contexts, require new authentication sessions, or establish event/outbox contracts. Current ownership and invariants remain in the Blueprint and Domain manifest.

## Rollback

Revert additive mappings and DTO fields together if needed, preserving existing JSON projections and storage schemas. Restore removed abstractions only with a real production consumer and acceptance tests. No storage migration, collection rename, credential change or deployment is authorized by this ADR.
