# Durable LINE events: staging deployment contract

**Not deployed:** this is a code/configuration specification, not permission to create billable services. Perform all actions only in an isolated, authorized staging project. Production rollout requires its own explicit approval.

## Processes and mode

- **Public webhook**: existing `cmd/webhook`, with raw LINE signature validation. Default is `LINE_EVENT_MODE=sync`; set `LINE_EVENT_MODE=durable` only after Firestore, queue, worker authentication and dispatcher are validated.
- **Private worker**: `cmd/worker`, built separately with `Dockerfile.worker` and optionally `cloudbuild.worker.yaml`. Both the public webhook and private worker need least-privilege GCP identities; do not share a broad service account.
- **Scheduled reconciliation**: Cloud Scheduler invokes POST `/tasks/dispatch` using Google-signed OIDC. The worker independently checks the token Audience and the configured service account Email, in addition to Cloud Run IAM.

## Required configuration

| Variable | Webhook durable | Worker | Meaning |
|---|---|---|---|
| `PORT` | optional | optional | HTTP listener port, default 8080 |
| `LINE_EVENT_MODE` | `durable` | n/a | Explicitly enables durable webhook mode |
| `LINE_CHANNEL_SECRET` | required | n/a | Verify LINE raw-body signature |
| `LINE_CHANNEL_ACCESS_TOKEN` | required | required | LINE SDK credential; not logged |
| `LINE_BOT_DESTINATION` | required | required | Bot destination ID for event isolation and event-key derivation |
| `GCP_PROJECT_ID` | required | required | Dedicated staging project |
| `TASK_QUEUE_NAME` | required | required | Full projects/.../locations/.../queues/... name |
| `TASK_WORKER_URL` | required | required | HTTPS worker URL ending in /tasks/line-reply |
| `TASK_OIDC_AUDIENCE` | required | required | Origin of the worker URL (without task path) |
| `TASK_INVOKER_EMAIL` | required | required | Google service account email authorized to call worker |
| `LINE_API_TIMEOUT` | optional | optional | Defaults to 8s, accepted 1s–12s |

## IAM and networking checks before enabling

- Private Cloud Run worker requires **IAM authentication** and non-public Ingress; no anonymous worker endpoints. Grant the specifically configured OIDC invoker service account permission to invoke this worker. The Cloud Tasks service agent must be allowed to mint OIDC tokens for it; configure and verify the necessary service-account token rights.
- Queue creation identity needs only the necessary Cloud Tasks enqueue rights on the dedicated queue. Worker/reconciliation and webhook identities need restricted Firestore access, not organization-wide roles.
- Firestore needs isolated test data, restricted security policy and TTL configuration for inbox/outbox as documented in `deploy/firestore/README.md`.
- Cloud Tasks service account credentials and LINE credentials must be resolved from trusted runtime configuration/Secret Manager and never committed.
- Cloud Scheduler must invoke `POST /tasks/dispatch` with a matching OIDC identity and Audience. Never expose that endpoint publicly or rely on `X-CloudTasks-*` headers alone for auth.
- Keep the worker URL and Audience bound to the same HTTPS origin. Do not enable `durable` until smoke tests confirm Firestore/Cloud Tasks IAM, OIDC, public LINE callback verification, token expiry, redelivery and unknown-state reconciliation.

## Verification sequence without production traffic

1. Run `go run ./cmd/codexcheck` and `make check`. These are code-level tests and do not prove deployed IAM or storage correctness.
2. In an authorized, isolated Firestore Emulator test, verify atomic first write vs redelivery, claim concurrency, terminal state transitions, TTL serialization and recovery behavior.
3. Use staging identities to test the actual Cloud Tasks CreateTask -> private Worker authentication path and the separate Scheduler dispatch endpoint.
4. Validate idempotency under crashed worker, slow LINE call and unknown network outcome, with only dedicated test LINE accounts.
5. Verify no raw message, reply token or user ID appears in structured logs or Cloud Tasks body. Require staging acceptance before any production readiness statement.

The private Worker image template does **not** modify the existing staging-only deployment safety gate. No Cloud Run, Cloud Build, Firestore, Cloud Tasks, Scheduler or IAM resource is provisioned by committing this document.
