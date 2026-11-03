# Firestore durable LINE event schema (staging only)

Two top-level collections use opaque SHA-256 IDs. `line_inbox_v1` is retained for 7 days, and `line_outbox_v1` for 1 hour. Configure Firestore TTL for the field `expires_at` in **both** collections before staging use. TTL deletion is asynchronous; do not rely on it as an immediate secure purge.

The inbox/outbox transaction persists the original reply token only in the outbox, together with the derived fixed response and lease state, **not** the original user message or sender ID. A worker commits `sending` before calling LINE; terminal states are `sent`, `expired` or `unknown`. An abandoned send becomes unknown, not automatically retryable; exactly-once delivery is not claimed.

Before staging, create an isolated Firestore database, configure least-privilege service identities, review retention/privacy policy, and arrange a scheduled, OIDC-authenticated POST to `/tasks/dispatch` to repair committed but unscheduled work. Bounded state queries in this implementation are not suitable for high-volume unpaginated operation; a later iteration needs indexed expiry queries, pagination, metrics and operator reconciliation.

These schema/retention notes and the merged PR #8 implementation do **not** provision Firestore, Cloud Tasks, Cloud Run, IAM, TTL policies or Cloud Scheduler. An isolated, explicitly authorized staging setup is required.

## Reconcile indexes and bounded progress (proposed)

The file [indexes.json](indexes.json) defines the required three ordered
composite queries for `line_outbox_v1`:
`(state, reply_until)`, `(state, lease_until)` and
`(state, next_dispatch_at)`, plus an optional `(state, updated_at)`
for operator inspection.

The adapter writes `next_dispatch_at` at first ingestion and refreshes it on
a signed redelivery that can legally reopen `pending` / `expired` work.
Reconcile selects only overdue Pending and abandoned Sending rows for
terminal transitions, then reserves each eligible Pending job with a
conditional Firestore transaction. Reserved jobs advance
`next_dispatch_at` by 2 minutes **before** Cloud Tasks enqueue: a crash or
ambiguous task publication delays a retry but cannot lose the durable item.
The webhook's original CreateTask attempt remains the fast path.

**Migration warning**: Firestore queries using `next_dispatch_at` do not
return older documents missing that field. Before switching an existing
database to the new reconciler, perform an authorized, restartable backfill
of `next_dispatch_at` for Pending records, verify the counts, and only then
cut over. No live database has been updated by this repository change.

Deploying these indexes or the two `expires_at` TTL policies is a separate
reviewed staging operation. The emulator can accept queries without enforcing
cloud index availability; **CI passing cannot establish production index or
TTL readiness**. TTL deletion is asynchronous and must not replace a domain
unsend/erasure tombstone.
