# ADR 0001: Strategic domain model authority and promotion gates

- Status: Accepted
- Date: 2026-11-04

## Context

The repository contains 102 LINE operations, multiple application capability packages, SDK catalog entries, and `go.mod` dependencies. These are useful implementation evidence, but their taxonomies describe providers and technical integration rather than business language, consistency boundaries, or data ownership.

Treating endpoint groups, package names, or SDK modules as bounded contexts would freeze accidental technical structure into the Domain model. Conversely, maintaining the same model manually in design documents, diagrams, inventories, and code would create conflicting sources of truth.

Identity is currently the only context with production evidence for an aggregate, entity, value objects, policies, specifications, domain events, repository contract, and behavioral tests. Other possible boundaries lack enough verified ownership, language, lifecycle, or provider-independent invariants.

## Decision

The repository uses the following authority order:

1. `docs/architecture/blueprint.md` for architecture and dependency rules.
2. `docs/domain/model.yaml` for the strategic Domain model.
3. `docs/domain/capability-map.yaml` for capability-to-operation mapping.
4. External endpoint inventories and the SDK catalog for provider facts.
5. Production Go code and tests for implementation evidence.
6. Generated context-map documentation as a non-authoritative projection.

Identity is recorded as `implemented/supporting`. Every other initial context is `candidate/unclassified` and creates no package, migration, or ownership commitment.

Context status has these gates:

- `candidate`: a hypothesis that may remain an application capability.
- `validated`: requires an agreed purpose, explicit non-responsibilities, owner, ubiquitous language, at least one provider-independent invariant, production capability, and an ADR approving the boundary.
- `implemented`: additionally requires production Domain behavior, tests, and resolvable package/evidence paths.
- `deprecated`: requires an explicit migration decision.

A status promotion is a reviewed architectural decision. It cannot be achieved solely by editing a status field, mapping endpoints, adding an empty package, or finding a matching dependency in `go.mod`.

Generated views must carry a generated marker and must be reproducible from authoritative manifests. Provider names such as LINE, Google, Firestore, Cloud Tasks, OpenAI, OAuth, and technical process names such as ingestion cannot be promoted directly to Domain context names.

## Consequences

- Strategic modeling can evolve additively without moving production packages or changing public contracts.
- Candidate boundaries remain visible while uncertainty is explicit.
- Endpoint and SDK coverage can be validated independently from Domain completeness.
- Context diagrams cannot silently become a second source of truth.
- Future promotions require product/domain evidence and an ADR, which is intentionally stricter than adding an Application capability.
- Package moves, wire-format changes, persistence migrations, and public API changes remain separate decisions.
