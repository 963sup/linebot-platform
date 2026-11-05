# ADR 0003: Tactical models without strategic promotion

- Status: Accepted
- Date: 2026-11-04

## Context

The dependency-informed context map identifies several useful business-boundary hypotheses. Code is now needed to make their proposed invariants executable, but an implementation artifact alone cannot prove that a Bounded Context, Subdomain classification, owner, or Ubiquitous Language matches the business.

Messaging Delivery also needs a hard wire and persistence boundary because the previous task payload and Firestore documents do not carry an explicit schema version.

## Decision

Implement provider-independent tactical models for Audience, Experience Presentation, Promotion, Workspace Automation, and Messaging Delivery. Implement Membership, Contact Directory, Insights Reporting, and Content Intelligence as Application-owned read/process models rather than inventing Aggregates.

Keep these Contexts at `candidate/hypothesis` until product and domain review validates their language, ownership, and boundaries. Their tests are implementation evidence, not strategic validation evidence. Identity remains the only strategically promoted `implemented` Context.

Use versioned Messaging Delivery boundaries:

- Existing Firestore collections remain `line_inbox_v1` and `line_outbox_v1`; new outbox documents carry `schema_version=2`, while a bounded compatibility reader accepts structurally valid legacy documents.
- Worker contracts `/tasks/v2/line-reply` and `/tasks/v2/dispatch`.
- Task payload `{schema_version, delivery_id}` with strict decoding.

A production rollout still requires an explicit drain, retention, compatibility, and rollback decision; offline tests do not authorize that rollout.

## Consequences

- Candidate invariants can be tested without binding the Domain to LINE, Google, Firestore, Cloud Tasks, OpenAI, or Gemini types.
- Consumer-owned Application ports define provider-independent integration contracts.
- Cross-Context Domain imports are rejected by architecture tests.
- Provider adapters and composition wiring for candidate Contexts remain separate delivery work.
- Existing durable work remains addressable by the stable collection names, but staging and production compatibility remain `Not Verified` and blocked on an explicit rollout procedure.
