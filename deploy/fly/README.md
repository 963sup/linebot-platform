# Fly.io staging: LINE webhook + Cloud Tasks worker

**Status: deployment configuration only.** No Fly.io app, billing resource, Google IAM,
Google Cloud Tasks queue, LINE callback, secret, or GitHub environment is created by
committing this directory. No production release is configured.

The Go binaries, DDD/Ports/Adapters code, and Google SDK integration are reused
unchanged. The staging topology has **two separate Fly apps**:

| App | Config / image | Inbound endpoint | Credentials |
| --- | --- | --- | --- |
| Public webhook | `fly.toml` / `Dockerfile` | `POST /webhook/line` | LINE Channel Secret only; GCP read/write + enqueue identity |
| OIDC-authenticated worker | `fly.worker.toml` / `Dockerfile.worker` | `POST /tasks/line-reply`, `POST /tasks/dispatch` | LINE Access Token only; GCP read/write + enqueue identity |

Both expose `GET /healthz` (liveness only). The worker's Fly HTTPS hostname is
**publicly routable** so Cloud Tasks/Scheduler can reach it; its `/tasks/*`
handlers reject requests without a Google-signed ID token whose audience and
service account email match the runtime configuration. It is **not equivalent**
to Cloud Run IAM private ingress. Protect it with the existing verifier, least
privilege, rate limits / monitoring, and staging penetration and abuse tests.

## 1. Prerequisites (manual and potentially billable)

- An authorized Fly.io account/org and `flyctl`, and an isolated Google Cloud
  **staging** project; decide and document a cost limit. This guide makes no
  paid API calls by itself.
- A *separate* Firestore database/collections and Cloud Tasks queue for staging,
  Firestore `expires_at` TTL, and Google Cloud Scheduler for
  `POST /tasks/dispatch`. Configure least-privilege GCP permissions as in
  [durable-worker.md](../../docs/operations/durable-worker.md).
- A dedicated **staging** LINE Messaging API Channel Secret / Access Token.
  Never point the initial release at a production audience.
- Separate GCP runtime identities for webhook and worker. Webhook needs
  narrowly scoped Firestore access, `cloudtasks.tasks.create` on its queue,
  and `iam.serviceAccounts.actAs` on the task-invoker identity where required.
  Worker needs the minimal Firestore and queue permissions needed for processing
  and dispatch. Ensure the Cloud Tasks service agent can mint OIDC tokens, and
  Cloud Scheduler uses the intended invoker identity.
- Google Application Default Credentials (ADC) **do not appear automatically**
  on Fly.io. Google recommends Workload Identity Federation (WIF) from outside
  Google Cloud, but WIF needs a verified external OIDC issuer and a refreshing
  subject-token source; **do not assume GitHub Actions OIDC is a Fly Machine
  identity**. Until that end-to-end integration is designed and verified, these
  staging TOMLs use a narrow, revocable Google service-account JSON key via
  Fly's secret-backed file injection. Review and rotate these keys; do not use
  the key bootstrap as an unreviewed production solution. On a verified keyless
  transition remove `[[files]]` and adjust `GOOGLE_APPLICATION_CREDENTIALS`
  in both configs for the supported ADC credential source.

## 2. Create two Fly apps (once, with explicit approval)

Choose globally unique app names ending with `-staging`. For example:

```sh
WEBHOOK_APP='<unique-webhook-staging>'
WORKER_APP='<unique-worker-staging>'
fly auth login
fly apps create "$WEBHOOK_APP"
fly apps create "$WORKER_APP"
```

Update the `app` values in `fly.toml` and `fly.worker.toml` to the chosen
names (the deploy workflow also passes `--app` explicitly).
The sample `primary_region = "nrt"` is a starting point, not a claim of
lowest latency: place Fly Machines near the actual GCP queue/Firestore and
measure network latency. Staging defaults to a **single always-on Machine**
per app (`--ha=false`); expect uptime limits and continuous costs. A production
design needs explicit HA/scaling/capacity review and separate configs.

## 3. Set per-app runtime settings and secrets

Required by `internal/bootstrap/config_contract.go` (do not invent a
second parser). Supply these on **each app**, with actual staging values:

- `LINE_BOT_DESTINATION`
- `GCP_PROJECT_ID`
- `TASK_QUEUE_NAME=projects/PROJECT/locations/REGION/queues/QUEUE`
- `TASK_WORKER_URL=https://WORKER_APP.fly.dev/tasks/line-reply`
- `TASK_OIDC_AUDIENCE=https://WORKER_APP.fly.dev` (origin only, no trailing path)
- `TASK_INVOKER_EMAIL=invoker@PROJECT.iam.gserviceaccount.com`

Supply the webhook app **only** `LINE_CHANNEL_SECRET`. Supply the worker app
**only** `LINE_CHANNEL_ACCESS_TOKEN`. Neither app should receive the other's
LINE credential. Fly `[env]` already contains public runtime constants:
`APP_ENV=staging`, `LINE_EVENT_MODE=durable`, `PORT=8080`,
`LOG_FORMAT=json`. Do not set `K_SERVICE` or weaken the role checks.

Provide these user-specific values with Fly app-scoped secrets or approved
protected CI provisioning (the non-sensitive settings may later be moved into
a reviewed environment-specific config). Prefer `fly secrets import --app
"$WEBHOOK_APP"` and `fly secrets import --app "$WORKER_APP"` from separate
local, **untracked, tightly permissioned** input files over entering tokens on
the command line. Never commit, echo, or paste real credentials into issues or
Actions output; do not upload those files to GitHub.

For the staging ADC bootstrap, each app must also have its **own**
`GCP_ADC_JSON_B64` secret: base64 of *that app's restricted Google service
account JSON*. Fly decodes that secret and places the file at
`/gcp-credentials.json` using `[[files]]` (it must never appear in an image).
The `GOOGLE_APPLICATION_CREDENTIALS` path is already configured. Base64 is
**encoding, not encryption**; only Fly's encrypted secret storage protects it.
Validate that the nonroot distroless processes can read the injected files
without disclosing their contents.

## 4. Validate without deployment

```sh
go run ./cmd/codexcheck
make check
fly config validate --config fly.toml
fly config validate --config fly.worker.toml
```

The Go contract tests check the role/image/secret boundaries. These commands
do **not** validate Fly billing, GCP authentication, OIDC integration,
Firestore permissions, LINE webhooks, or end-to-end latency.

## 5. First staging deployment (manual)

Once secrets, IAM, networking and cost authorization are in place:

```sh
fly deploy --app "$WORKER_APP" --config fly.worker.toml --remote-only --ha=false
fly deploy --app "$WEBHOOK_APP" --config fly.toml --remote-only --ha=false
fly status --app "$WORKER_APP"
fly status --app "$WEBHOOK_APP"
```

Deploy the worker first. HTTPS endpoints are then:
`https://WEBHOOK_APP.fly.dev/webhook/line` and
`https://WORKER_APP.fly.dev/tasks/line-reply`.
Configure the LINE Developers webhook URL only after validating signature and
GCP integration on staging. Configure Cloud Scheduler with `POST` to
`https://WORKER_APP.fly.dev/tasks/dispatch` using a Google-signed OIDC token
with the same audience and service account email; never remove Worker
authorization or accept `X-CloudTasks-*` headers as authentication.

A successful `/healthz` only confirms the process is listening; do **not**
interpret it as a valid Firestore, Cloud Tasks, OIDC or LINE smoke test. Perform
the acceptance tests in [staging-acceptance.md](../../docs/operations/staging-acceptance.md):
inbox/outbox transactions, redelivery/duplication, worker OIDC rejection,
latency and reply deadline, retry/unknown reconciliation, TTL, no PII in logs,
and secrets isolation. The Cloud Scheduler 1-minute cadence cannot guarantee
replies before the project's conservative 45-second deadline.

## 6. Opt-in GitHub Actions deployment

The new `.github/workflows/fly-deploy.yml` is **workflow_dispatch only**,
restricted to `main` and to GitHub Environment `fly-staging`. Configure that
Environment with required reviewer approval and restrict to `main`. Define
Environment (or repository) variables:

- `FLY_WEBHOOK_APP` = actual webhook app name ending with `-staging`
- `FLY_WORKER_APP` = actual worker app name ending with `-staging`

Define separate Environment secrets:

- `FLY_WEBHOOK_DEPLOY_TOKEN` = app-scoped Fly deploy token for webhook
- `FLY_WORKER_DEPLOY_TOKEN` = app-scoped Fly deploy token for worker

Generate tokens according to the current [Fly token docs](https://fly.io/docs/security/tokens/);
do not use an all-apps org token. Use Actions → Fly.io staging deploy →
Run workflow → `both` (worker first), `worker`, or `webhook`.
The workflow first runs Codex/catalog verification, Go tests, and Fly TOML
validation. Deployment does not create apps or assign GCP permissions.

## 7. Operational guardrails

- Keep the existing Cloud Run files/workflows unchanged; Fly is an **optional**
  alternate hosting target. No Cloud Build triggers are created.
- Review logs with `fly logs --app "$WEBHOOK_APP"` and `fly logs --app
  "$WORKER_APP"`; ensure secrets, LINE user IDs, reply tokens, and raw
  webhook bodies are never logged.
- Tune Fly availability, outbound network/firewall rules, Cloud Tasks retries,
  GCP quotas and alerting after staging measurement.
- Never add LINE Access Token to public Webhook, nor allow unauthenticated task
  processing; both are validated fail-closed in Go runtime.
- No Fly volume/database is required: Firestore remains the durable store.
- Production requires its own approval, access controls, keyless identity
  verification, rollout/rollback and HA design; `fly-staging` cannot deploy
  to production.

Official references:
[Fly app configuration](https://fly.io/docs/reference/configuration/),
[Fly GitHub Actions](https://fly.io/docs/launch/continuous-deployment-with-github-actions/),
[Cloud Tasks HTTP targets](https://docs.cloud.google.com/tasks/docs/creating-http-target-tasks),
[Google WIF](https://docs.cloud.google.com/iam/docs/workload-identity-federation).
