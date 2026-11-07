# Durable LINE events: staging deployment contract

**Not deployed:** this is a code/configuration specification, not permission to create billable services. Perform all actions only in an isolated, authorized staging project. Production rollout requires its own explicit approval.

## Processes and mode

- **Public webhook**: existing `cmd/webhook`, with raw LINE signature validation. `LINE_EVENT_MODE=sync` is local-development-only; Cloud Run fails closed unless `LINE_EVENT_MODE=durable`. The public durable webhook holds `LINE_MESSAGING_API_CHANNEL_SECRET` but **must not be provided `LINE_MESSAGING_API_CHANNEL_ACCESS_TOKEN`**. Enable durable only after Firestore, queue, worker authentication and dispatcher are validated.
- **Private worker**: `cmd/worker`, packaged with the shared `Dockerfile` by `cloudbuild.yaml`. Both the public webhook and private worker need least-privilege GCP identities; do not share a broad service account.
- **Scheduled reconciliation**: Cloud Scheduler invokes POST `/tasks/v2/dispatch` using Google-signed OIDC. The worker independently checks the token Audience and the configured service account Email, in addition to Cloud Run IAM. This is best-effort pending-work recovery, **not** a guarantee to meet the 45-second LINE Reply window.

## Required configuration

| Variable | Webhook durable | Worker | Meaning |
|---|---|---|---|
| `APP_ENV` | `staging` or `production` on Cloud Run | same | Explicit runtime profile; local is rejected when `K_SERVICE` is present |
| `PORT` | optional | optional | HTTP listener port, default 8080 |
| `LINE_EVENT_MODE` | `durable` | optional `durable` only | Worker always runs in durable mode; explicit `sync` or unknown values fail startup |
| `LINE_MESSAGING_API_CHANNEL_SECRET` | required | n/a | Verify LINE raw-body signature |
| `LINE_MESSAGING_API_CHANNEL_ACCESS_TOKEN` | **forbidden** | required | Only the private worker is permitted to send LINE messages |
| `LINE_MESSAGING_API_BOT_USER_ID` | required | n/a | Webhook destination isolation and event-key derivation; Worker receives resolved delivery context |
| `GCP_PROJECT_ID` | required | required | Dedicated staging project |
| `TASK_QUEUE_NAME` | required | required | Exactly projects/PROJECT/locations/REGION/queues/NAME; PROJECT must equal GCP_PROJECT_ID, QUEUE ID must be 1–100 ASCII letters/digits/hyphens |
| `TASK_WORKER_URL` | required | required | HTTPS worker URL ending in /tasks/v2/line-reply |
| `TASK_OIDC_AUDIENCE` | required | required | Origin of the worker URL (without task path) |
| `TASK_INVOKER_EMAIL` | required | required | Google service account email authorized to call worker |
| `LINE_API_TIMEOUT` | optional | optional | Defaults to 8s, accepted 1s–12s |
| `LOG_LEVEL` | optional | optional | debug/info/warn/error; default info |
| `LOG_FORMAT` | json | json | staging/production require structured JSON |

The checked and generated contract for the complete settings list is [Runtime Configuration](configuration.md). Do not duplicate or drift runtime defaults in deployment templates. In production the Webhook receives its Secret and the Worker receives its Token via separate Secret Manager references, not a shared `.env` file.

## IAM and networking checks before enabling

- Private Cloud Run worker requires **IAM authentication** and non-public Ingress; no anonymous worker endpoints. Grant the specifically configured OIDC invoker service account permission to invoke this worker. The Cloud Tasks service agent must be allowed to mint OIDC tokens for it; configure and verify the necessary service-account token rights.
- Queue creation identity needs only the necessary Cloud Tasks enqueue rights on the dedicated queue. Worker/reconciliation and webhook identities need restricted Firestore access, not organization-wide roles.
- Firestore needs isolated test data, restricted security policy and TTL configuration for inbox/outbox as documented in `deploy/firestore/README.md`.
- Cloud Tasks service account credentials and LINE credentials must be resolved from trusted runtime configuration/Secret Manager and never committed.
- Cloud Scheduler must invoke `POST /tasks/v2/dispatch` with a matching OIDC identity and Audience. Never expose that endpoint publicly or rely on `X-CloudTasks-*` headers alone for auth.
- `/tasks/line-reply` and `/tasks/dispatch` remain OIDC-protected, drain-only compatibility routes for work created before the V2 cutover. New configuration must not target them; remove them only after authorized queue and scheduler drain evidence exists.
- Keep the worker URL and Audience bound to the same HTTPS origin. Do not enable `durable` until smoke tests confirm Firestore/Cloud Tasks IAM, OIDC, public LINE callback verification, token expiry, redelivery and unknown-state reconciliation.

## Cloud Tasks task-name reuse and bounded recovery

Cloud Tasks retains task names for **up to 24 hours** after the task completes or is deleted. A single permanently fixed task ID can therefore prevent recovery of a pending Outbox after the first task was acknowledged. The Queue uses the deterministic event SHA-256 fingerprint combined with a **10-second scheduling bucket**: enqueueing within the same bucket deduplicates, but later buckets may produce a fresh task even if an old task completed. The worker's Firestore single-use claim prevents a second LINE call; task-name dedupe alone does not.

LINE Reply tokens can only be used once. A redelivered webhook carries the same token, and reply eligibility is bounded by the original event timestamp + 20 minutes. A fresh signed redelivery may extend a reply window only when the outbox has **never been sent** (`pending` or `expired`). `sending`, `sent`, and `unknown` are never reopened. The resulting 45-second local window is conservative; LINE does not guarantee actual token validity for the full interval.

Cloud Scheduler's minimum common cadence is one minute, longer than the conservative 45-second Reply deadline. Scheduled reconciliation can expire stale records and repair some pending work, **but does not guarantee an on-time Reply**; review a separate authorized Push/follow-up strategy before production.

Reference: [Google Cloud Tasks task deduplication](https://docs.cloud.google.com/tasks/docs/dual-overview).

## Verification sequence without production traffic

1. Run `go run ./cmd/codexcheck` and `make check`. These are code-level tests and do not prove deployed IAM or storage correctness.
2. In an authorized, isolated Firestore Emulator test, verify atomic first write vs redelivery, claim concurrency, terminal state transitions, TTL serialization and recovery behavior.
3. Use staging identities to test the actual Cloud Tasks CreateTask -> private Worker authentication path and the separate Scheduler dispatch endpoint.
4. Capacity verification is currently `BLOCKED`: the repository has no deployable staging-only blocking fixture. After separately reviewing a fixed-digest test image or fixture that is isolated by Staging IAM and absent from the production image, use at least 17 synchronous blocked requests to verify that no more than 16 enter the handler, the seventeenth receives `503 Service Unavailable` with a valid `Retry-After`, Cloud Tasks retries it later instead of creating an in-process backlog, and shutdown stops admission while draining active handlers. Record the tested Revision's actual `containerConcurrency=16` setting.
5. Validate idempotency under crashed worker, slow LINE call and unknown network outcome, with only dedicated test LINE accounts.
6. Verify no raw message, reply token or user ID appears in structured logs or Cloud Tasks body. Require staging acceptance before any production readiness statement.

The private Worker image template does **not** modify the existing staging-only deployment safety gate. No Cloud Run, Cloud Build, Firestore, Cloud Tasks, Scheduler or IAM resource is provisioned by committing this document.
