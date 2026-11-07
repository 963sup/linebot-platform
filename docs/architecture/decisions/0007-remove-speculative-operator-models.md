# ADR 0007: Remove speculative operator models

- Status: Accepted
- Date: 2026-11-07
- Supersedes: ADR 0003 tactical models for Audience, Experience Presentation and Promotion.

## Decision

The affected context IDs are `audience`, `experience-presentation` and `promotion`;
they remain candidate Application process models, not strategic promotions.

Keep native LINE audience, rich-menu and coupon operations behind their existing
Application contracts. Remove the independent AudienceDefinition,
InteractionSurface and Promotion lifecycles, their Firestore repositories and
the audience-definitions, interaction-surfaces and promotions CLI commands.
They are not required by the current MINI App, Messaging API, Firestore,
OpenAI, GenAI and MCP client skeleton. Reintroducing product-owned lifecycles
requires a concrete use case rather than an SDK endpoint inventory.

Remove the unused Secret Manager application client factory and direct SDK
dependency. Cloud Run secret references, secrets and IAM are unchanged.

## Compatibility

This removes local CLI commands and code; it does not delete or migrate stored
documents. Native LINE CLI operations, token verification, delivery inbox/outbox,
Cloud Tasks and AI/MCP factories remain. Tenant routing, event-key compatibility,
Room support and local synchronous Webhook mode require separate decisions and
are not changed in this slice.

## Verification and rollback

Verify native LINE API contracts, retired-command rejection, architectural
dependencies, generated Domain documentation, build, vet and race tests.
Restore retired code from Git only if a corresponding product requirement is
approved. No deployment, cloud data deletion or IAM change is part of this ADR.
