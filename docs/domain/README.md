# Strategic DDD model

This directory records the repository's strategic Domain-Driven Design model. It distinguishes implemented facts from explicit hypotheses inferred from workflows, dependencies, application capabilities, and external APIs.

## Authority order

When two sources disagree, use this order:

1. `docs/architecture/blueprint.md` defines architecture and dependency rules.
2. `docs/domain/model.yaml` defines subdomains, bounded contexts, language, ownership, invariants, and relationships.
3. `docs/domain/dependency-capabilities.yaml` records how direct `go.mod` modules normally compose capabilities and which Context hypotheses they support.
4. `docs/domain/capability-map.yaml` maps application capabilities to external operations; it does not define domain boundaries.
5. Endpoint inventories and `docs/sdk/catalog.toml` record provider and integration facts.
6. Production Go code and tests provide implementation evidence.
7. `docs/domain/context-map.md` is a generated projection and is never an editable source of truth.

External APIs, SDK groups, package names, and `go.mod` dependencies are discovery evidence. They may support a `hypothesis`, but cannot promote it to `validated` or `implemented` without business language, transactional rules, ownership, and an ADR.

## Status meanings

| Status | Meaning |
| --- | --- |
| `candidate` | A documented hypothesis with language, ownership, model kind, evidence, confidence, and validation questions. It creates no requirement for a Domain package. |
| `validated` | The boundary has an agreed purpose, owner, non-responsibilities, ubiquitous language, non-provider invariant, production capability, and ADR evidence. |
| `implemented` | A validated context also has production Domain behavior and tests at the declared package and evidence paths. |
| `deprecated` | The context is being retired under an explicit migration decision. |

Subdomain classification uses `unclassified`, `core`, `supporting`, or `generic`. A candidate may carry a provisional classification when `decision_status: hypothesis` and its confidence is explicit. This is not a validation claim.

`model_kind` distinguishes `domain-model`, `process-model`, and `read-model`. A domain model requires an Aggregate and its consistency boundary. A read model must not declare an Aggregate. Data ownership distinguishes `local-authoritative`, `provider-authoritative`, and `derived-projection`.

Endpoint mappings use four independent states: `mapped` means the operation belongs to a named Application capability, `provisional` additionally proposes a candidate Context, `unclassified` records unresolved business ownership with a reason, and `not-domain-relevant` keeps provider or transport work at the Application/Adapter boundary. A mapped operation without a Context is intentional and does not imply a missing Domain package.

Changing a status from `candidate` requires an ADR. Creating an empty directory, mapping an endpoint, or matching an SDK name is not sufficient evidence. Package moves and public contract migrations require separate decisions.

## Editing workflow

- Edit `model.yaml` for strategic model changes.
- Edit `dependency-capabilities.yaml` when a direct module, SDK status, inferred capability, or Context signal changes.
- Record every status promotion and boundary change in `docs/architecture/decisions/`.
- Regenerate `context-map.md`; do not edit it directly.
- Preserve evidence paths as repository-relative paths.
- Keep provider vocabulary at adapter boundaries. Names such as LINE, Google, Firestore, Cloud Tasks, OpenAI, OAuth, and ingestion are not bounded-context names.
- Candidate status does not by itself justify or require a Domain package. A candidate without independently reviewed language, lifecycle, invariants, and evidence remains documentation only unless an explicit ADR authorizes a tactical model for hypothesis testing. Such a tactical model does not strategically promote the Context.

## Current confidence

Messaging Delivery is a medium-confidence Core hypothesis because the repository contains explicit claim, unknown-outcome, Inbox/Outbox, and reconciliation behavior. ADR 0003 authorizes provider-independent tactical models for Messaging Delivery, Audience, Experience Presentation and Promotion without promoting those candidate Contexts. A hypothesis never creates a package or persistence migration by itself; either requires explicit decision evidence and separate delivery work.

Browser Geolocation remains an Adapter mechanism and is not physical-presence proof. No current Domain owns or persists its readings; a future use case must define that responsibility explicitly.
