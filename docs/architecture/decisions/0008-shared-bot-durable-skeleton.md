# ADR 0008: Shared Bot, conversation isolation and durable-only ingress

Status: Accepted for the explicitly requested skeleton cleanup. Supersedes ADR 0006.

## Decision

- One Messaging API Bot and one MINI App configuration serve every company. A verified `groupId` identifies a group conversation; `userId` identifies a direct conversation or an actor within a group. The Bot destination is not a company/tenant identifier.
- Remove Tenant/ChannelConnection aggregates, IDs, repositories, registration CLI and per-request resolver. Do not add replacement registries or membership machinery.
- Keep the LINE ACL and asynchronous DeliveryContext with conversation kind and SHA-256 conversation/actor fingerprints. These values express isolation, not proof of membership or permission for MINI App group operations. There is no user-profile or group-registration write on page load/login.
- Durable is the only webhook processing path. The public process verifies signatures/destination and commits Inbox/Outbox before queueing. Only the worker owns LINE sending credentials. A credential-free local process serves health/readiness and returns 503 on the webhook; it is not a synchronous fallback.
- Remove Room application, CLI and SDK adapter branches. Unsupported Room webhooks are acknowledged after signature/destination checks without persistence or enqueueing. Previously queued Room work remains readable for draining.
- Firestore uses only the default database constructor. Keep context/project validation, transactional delivery invariants, typed persistence failures and cancellation; remove unused repository error constants.

## Persistence compatibility exception

Owner: Firestore delivery adapter, wired by Webhook bootstrap.

The old event key is SHA-256(`tenantID + NUL + connectionID + NUL + eventID`). Changing it blindly would permit duplicate deliveries. Startup reads the old destination-hashed document in `line_channel_connections_v1` once, solely to reuse that namespace. No document is created or updated. Only NotFound selects the configured Bot destination namespace; malformed data, permission failures and network errors fail startup. Application code receives an opaque stable namespace and has no Tenant/Connection concepts.

Do not delete or change this old metadata while its namespace is in use. Removal requires an explicit namespace cutover with stopped ingress, drained pending tasks, expired redelivery eligibility and retained dedup evidence; it is not part of this source cleanup. This read-only shim is a storage migration constraint, not a business registry.

New outbox records are schema v4 without Tenant/Connection fields. Readers accept legacy unversioned, v2 and v3 records; updates do not silently promote legacy schemas. Existing v3 conversation fingerprints still prevent changed-source redelivery. Unknown schemas fail closed.

## Rollout and verification

1. Upgrade the worker first so it can read v4 while draining old jobs.
2. Switch the public webhook without mixing old/new public revisions; old public code rejects v4 work.
3. Do not revert to a pre-v4 worker/public image while v4 work can still be received or processed. Use a schema-compatible rollback image or an explicitly coordinated drain.
4. Verify legacy key equality, safe Room ignoring, source isolation, default database validation, replay/claim behavior and failure handling. Emulator tests cover real stored legacy fields as well as new records. Offline tests are not staging acceptance.

No cloud resources, permissions, data or traffic are changed by this cleanup.
