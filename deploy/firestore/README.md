# Firestore durable LINE event schema (staging only)

Two top-level collections use opaque SHA-256 IDs. `line_inbox_v1` is retained for 7 days, and `line_outbox_v1` for 1 hour. New outbox documents carry `schema_version=2`; the collection names remain stable so an application rollout cannot silently strand pending V1 work. Configure Firestore TTL for the field `expires_at` in **both** collections only after explicit retention approval. TTL deletion is asynchronous; do not rely on it as an immediate secure purge.

The inbox/outbox transaction persists the original reply token only in the outbox, together with the derived fixed response and lease state, **not** the original user message or sender ID. A worker commits `sending` before calling LINE; terminal states are `sent`, `expired` or `unknown`. An abandoned send becomes unknown, not automatically retryable; exactly-once delivery is not claimed.

Before staging, create an isolated Firestore database, configure least-privilege service identities, review retention/privacy policy, and arrange a scheduled, OIDC-authenticated POST to `/tasks/v2/dispatch` to repair committed but unscheduled work. Bounded state queries in this implementation are not suitable for high-volume unpaginated operation; a later iteration needs indexed expiry queries, pagination, metrics and operator reconciliation.

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

The optional read-only Outbox inspection also requires `(state,created_at)`
for oldest Pending age, and `(state,updated_at)` for bounded samples.
`ops events --check` uses Firestore server-side count aggregations and
**never** loads any LINE user text or token. This is a snapshot from several
queries, not a globally consistent read transaction.

Deploying these indexes or the two `expires_at` TTL policies is a separate
reviewed staging operation. The emulator can accept queries without enforcing
cloud index availability; **CI passing cannot establish production index or
TTL readiness**. TTL deletion is asynchronous and must not replace a domain
unsend/erasure tombstone.


## Google OAuth State / Token Collections (planned staging verification)

The OAuth code path uses **separate** collections from LINE inbox/outbox:

| Collection | Document ID | Persisted fields | TTL |
|---|---|---|---|
| `google_oauth_states_v1` | SHA-256 digest of 32-byte random state | AES-GCM encrypted owner / PKCE verifier / nonce / requested scopes, key version and expires_at | 10 minutes |
| `google_oauth_connections_v1` | SHA-256 digest of owner + Google subject | AES-GCM encrypted OAuth credentials / identity / scopes; active, version and updated_at | 30-day inactive tombstone via expires_at |

Records are bound to their Firestore document ID as AES-GCM authenticated associated data. The first callback to atomically consume a state deletes it in a Firestore transaction; further callbacks cannot use that state. Connection updates use versioned transactional CAS and revoked accounts clear their ciphertext so a stale token refresh cannot reactivate them.

**Before authorized staging use:** Configure the `expires_at` Firestore TTL field for both collection groups, provision pinned 32-byte encryption keys in Secret Manager with least-privilege IAM and rotation, validate runtime `APP_ENV` / OAuth callback URL, and test concurrent consumes, refresh CAS and disconnect using Firestore Emulator and then isolated staging. TTL is asynchronous and never substitutes for checking expiry and active flags in code.

No Google OAuth keys, user OAuth tokens, or persistent app credentials are shipped in repository or CI. The new OAuth libraries are opt-in and are not mounted on the public LINE webhook by default.
