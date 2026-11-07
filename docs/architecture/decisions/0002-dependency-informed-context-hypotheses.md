# ADR 0002: Dependency-informed Context hypotheses

- Status: Accepted
- Date: 2026-11-04
- Superseded in part by: ADR 0003, only for explicitly authorized candidate tactical model packages

> Historical Workspace / Google account-link scope below is retired by [ADR 0006](0006-retire-unused-workspace-boundaries.md). Current capabilities are defined by the Domain manifest.

## Context

The first strategic model used the LINE endpoint inventory conservatively and left every Context except Identity unclassified. That avoided false certainty but omitted useful evidence already present in `go.mod`, the SDK catalog, Application use cases, persistence workflows, and tests.

An external dependency does not define a business boundary. It does reveal a capability that is normally composed with other capabilities: Firestore and Cloud Tasks support durable delivery; OAuth and Google APIs support authorized workspace actions; LINE operations suggest messaging, audience, presentation, promotion, membership, contacts, and analytics; OpenAI and Gemini suggest a replaceable content-generation capability.

## Decision

Record complete Context hypotheses with explicit decision status and confidence. Each hypothesis includes ubiquitous language, model kind, Aggregate consistency boundary where applicable, authoritative data owner, invariants, contracts, evidence, and open validation questions.

Record every direct `go.mod` dependency in `docs/domain/dependency-capabilities.yaml`, cross-check its version and SDK catalog status, and distinguish domain signals from supporting capabilities, generic infrastructure, and tooling. Generic infrastructure cannot own a Context. A `declared` dependency cannot be represented as integrated behavior.

Messaging Delivery is provisionally classified Core with medium confidence. Identity remains the only strategically implemented Context. All other classifications and boundaries remain hypotheses until business review and a promotion ADR. ADR 0003 later permits five of those hypotheses to have tactical model packages; that implementation evidence is not strategic validation.

## Consequences

- Architecture work can start from an explicit complete map without presenting inference as fact.
- Aggregate proposals must name their transactional consistency boundary; read models cannot declare Aggregates.
- Provider resources remain provider-owned and enter through anti-corruption layers or consumer-owned contracts.
- Module changes, SDK status changes, and stale generated documentation fail offline governance checks.
