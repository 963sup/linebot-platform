# ADR 0006: Tenant channel and inbound conversation skeleton

Status: Accepted as a hypothesis

## Context

The durable LINE webhook already verifies a configured channel secret and destination, but accepted events had no tenant scope and carried LINE SDK source objects directly into handler decisions. Future customer-care work needs a stable boundary without prematurely implementing cases, assignments, SLA, or a full conversation aggregate.

## Decision

- Authorize `tenant-administration` as a candidate/hypothesis tactical Context; this does not strategically promote it.
- Tenant Administration owns the minimal `ChannelConnection` invariant: one LINE messaging destination is bound to one tenant connection.
- After raw-body signature and configured destination verification, the webhook resolves that binding and creates request-scoped channel context.
- A LINE inbound ACL converts user, group, and room sources into a provider-neutral `InboundTextEvent`.
- Direct, group, and room are address kinds, not separate Bounded Contexts.
- New durable outbox records use schema v3 and retain the local tenant ID, channel-connection ID, address kind, and SHA-256 fingerprints of provider conversation/actor references. Raw provider IDs and message text are not persisted by this boundary.
- Existing schema v2 records remain readable and are not silently promoted to v3 when their missing scope cannot be reconstructed.
- Existing reply policy and Messaging Delivery remain unchanged until a real Customer Care use case defines `ReplyIntent` ownership.
- Provider credentials remain outside the Domain. The current configured secret is transitional; dynamic credential resolution is a later, explicit migration.

## Consequences

Durable webhook processing now requires a registered channel-connection document. The skeleton has a production consumer and preserves the current delivery behavior, while leaving Customer Knowledge and Customer Care unimplemented rather than represented by empty packages.
