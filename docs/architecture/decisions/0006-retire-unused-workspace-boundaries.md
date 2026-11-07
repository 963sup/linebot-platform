# ADR 0006: Retire unused Workspace boundaries

- Status: Accepted
- Date: 2026-11-07
- Supersedes: Workspace/Google account-link scope in ADRs 0001–0003; Cloud Storage factory retention in ADR 0005.

## Decision

The supported scope is LINE MINI App, LINE Messaging API, Firestore, OpenAI,
Google GenAI and an MCP client. Remove Google Workspace automation, its Google
account-link Domain/Application/OAuth chain, CLI commands, endpoint inventories
and unused Cloud Storage factory. Workspace may be integrated later against a
concrete use case; do not retain speculative code for that possibility.

Remove unused Firestore tenant/convenience helpers. Inject the LINE session
Verifier directly instead of wrapping it in a pass-through Service. Compose
each retained operator capability independently; no command initializes all
operator repositories.

## Boundaries and consequences

LINE ID-token verification, webhook signatures, Firestore delivery state and
Cloud Tasks reliability remain. OpenAI, Gemini/Vertex and MCP native factories
remain supported but are not thereby claimed to be runtime-integrated.

Remove obsolete local Identity outbox index/TTL declarations. This change does
not migrate or delete any cloud data, indexes, secrets or IAM resources. Review
an infrastructure plan separately before applying the reduced declarations.

## Verification and rollback

Verify CLI composition, MINI App authentication, module/catalog consistency,
architecture, documentation links and ordinary/race tests. Restore retired
code from Git only with an explicit product requirement and fresh tests.
