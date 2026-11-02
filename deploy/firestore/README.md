# Firestore durable LINE event schema (staging only)

Two top-level collections use opaque SHA-256 IDs. `line_inbox_v1` is retained for 7 days, and `line_outbox_v1` for 1 hour. Configure Firestore TTL for the field `expires_at` in **both** collections before staging use. TTL deletion is asynchronous; do not rely on it as an immediate secure purge.

The inbox/outbox transaction persists the original reply token only in the outbox, together with the derived fixed response and lease state, **not** the original user message or sender ID. A worker commits `sending` before calling LINE; terminal states are `sent`, `expired` or `unknown`. An abandoned send becomes unknown, not automatically retryable; exactly-once delivery is not claimed.

Before staging, create an isolated Firestore database, configure least-privilege service identities, review retention/privacy policy, and arrange a scheduled, OIDC-authenticated POST to `/tasks/dispatch` to repair committed but unscheduled work. Bounded state queries in this implementation are not suitable for high-volume unpaginated operation; a later iteration needs indexed expiry queries, pagination, metrics and operator reconciliation.

This PR does not provision any Firestore, Cloud Tasks, Cloud Run, IAM, TTL or Scheduler resources.
