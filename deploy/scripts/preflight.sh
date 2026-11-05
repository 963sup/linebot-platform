#!/usr/bin/env bash
set -euo pipefail

# Read-only deployment preflight. This script only calls describe/get commands;
# it never creates, updates, deploys, grants, or deletes cloud resources.

required=(
  PROJECT_ID REGION QUEUE_ID PUBLIC_IMAGE_URI PRIVATE_IMAGE_URI
  PUBLIC_SERVICE_ACCOUNT WORKER_SERVICE_ACCOUNT INVOKER_SERVICE_ACCOUNT
  WORKER_SERVICE TASK_WORKER_ORIGIN
  STAGING_EVIDENCE_FILE STAGING_EVIDENCE_SHA256
)
for name in "${required[@]}"; do
  if [[ -z "${!name:-}" ]]; then
    echo "missing required preflight input: ${name}" >&2
    exit 2
  fi
done

if [[ ! "${TASK_WORKER_ORIGIN}" =~ ^https://[a-z0-9-]+\.[a-z0-9-]+\.run\.app$ ]]; then
  echo "TASK_WORKER_ORIGIN must be a Cloud Run HTTPS origin without a path" >&2
  exit 2
fi

if [[ ! "${STAGING_EVIDENCE_SHA256}" =~ ^[0-9a-f]{64}$ ]]; then
  echo "STAGING_EVIDENCE_SHA256 must be a lowercase SHA-256 digest" >&2
  exit 2
fi
if [[ ! -f "${STAGING_EVIDENCE_FILE}" ]]; then
  echo "staging evidence file does not exist" >&2
  exit 2
fi
actual_digest="$(sha256sum "${STAGING_EVIDENCE_FILE}" | awk '{print $1}')"
if [[ "${actual_digest}" != "${STAGING_EVIDENCE_SHA256}" ]]; then
  echo "staging evidence digest mismatch" >&2
  exit 2
fi

# The evidence is deliberately non-secret and machine-checkable. jq -e fails
# closed when any required gate is absent or not explicitly passed.
jq -e '
  .schema_version == 1 and
  .environment == "staging" and
  .approved_for_production == true and
  (.observed_at_utc | type == "string" and length > 0) and
  (.expires_at_utc | type == "string" and length > 0) and
  (.reviewer | type == "string" and length > 0) and
  (.image_digests.public | type == "string" and contains("@sha256:")) and
  (.image_digests.private | type == "string" and contains("@sha256:")) and
  (["iam","credentials","firestore","worker_capacity","redelivery","reply_deadline","reconcile","indexes_ttl","monitoring","rollback"] - (.passed_gates // [] | unique) | length == 0)
' "${STAGING_EVIDENCE_FILE}" >/dev/null

evidence_public="$(jq -r '.image_digests.public' "${STAGING_EVIDENCE_FILE}")"
evidence_private="$(jq -r '.image_digests.private' "${STAGING_EVIDENCE_FILE}")"
expires_at="$(jq -r '.expires_at_utc' "${STAGING_EVIDENCE_FILE}")"
if ! expires_epoch="$(date -u -d "${expires_at}" +%s 2>/dev/null)" || (( expires_epoch <= $(date -u +%s) )); then
  echo "staging evidence is expired or has an invalid expires_at_utc" >&2
  exit 2
fi
evidence_public_digest="${evidence_public##*@sha256:}"
evidence_private_digest="${evidence_private##*@sha256:}"
public_digest="${PUBLIC_IMAGE_URI##*@sha256:}"
private_digest="${PRIVATE_IMAGE_URI##*@sha256:}"
if [[ "${evidence_public_digest}" != "${public_digest}" || "${evidence_private_digest}" != "${private_digest}" ]]; then
  echo "deployment images do not match approved staging evidence" >&2
  exit 2
fi

for image in "${PUBLIC_IMAGE_URI}" "${PRIVATE_IMAGE_URI}"; do
  case "${image}" in
    "${REGION}-docker.pkg.dev/${PROJECT_ID}/"*"@sha256:"*) ;;
    *) echo "image must be in the target project/region and pinned by digest" >&2; exit 2 ;;
  esac
  gcloud artifacts docker images describe "${image}" --project="${PROJECT_ID}" --quiet >/dev/null
done

gcloud tasks queues describe "${QUEUE_ID}" --location="${REGION}" --project="${PROJECT_ID}" --quiet >/dev/null
gcloud firestore databases describe --database='(default)' --project="${PROJECT_ID}" --quiet >/dev/null
authoritative_worker_origin="$(gcloud run services describe "${WORKER_SERVICE}" --region="${REGION}" --project="${PROJECT_ID}" --format='value(status.url)')"
if [[ "${authoritative_worker_origin}" != "${TASK_WORKER_ORIGIN}" ]]; then
  echo "TASK_WORKER_ORIGIN does not match the authoritative Cloud Run service URL" >&2
  exit 2
fi

for account in "${PUBLIC_SERVICE_ACCOUNT}" "${WORKER_SERVICE_ACCOUNT}" "${INVOKER_SERVICE_ACCOUNT}"; do
  gcloud iam service-accounts describe "${account}" --project="${PROJECT_ID}" --quiet >/dev/null
done
for secret in line-bot-channel-secret line-bot-channel-access-token; do
  gcloud secrets describe "${secret}" --project="${PROJECT_ID}" --quiet >/dev/null
done

require_project_role() {
  local account="$1" role="$2"
  gcloud projects get-iam-policy "${PROJECT_ID}" \
    --flatten='bindings[].members' \
    --filter="bindings.role=${role} AND bindings.members=serviceAccount:${account}" \
    --format='value(bindings.role)' | grep -Fxq "${role}"
}
require_secret_role() {
  local secret="$1" account="$2"
  gcloud secrets get-iam-policy "${secret}" --project="${PROJECT_ID}" \
    --flatten='bindings[].members' \
    --filter="bindings.role=roles/secretmanager.secretAccessor AND bindings.members=serviceAccount:${account}" \
    --format='value(bindings.role)' | grep -Fxq 'roles/secretmanager.secretAccessor'
}
reject_project_role() {
  local account="$1" role="$2"
  if gcloud projects get-iam-policy "${PROJECT_ID}" \
    --flatten='bindings[].members' \
    --filter="bindings.role=${role} AND bindings.members=serviceAccount:${account}" \
    --format='value(bindings.role)' | grep -Fxq "${role}"; then
    echo "forbidden project-wide role ${role} on ${account}" >&2
    exit 2
  fi
}
reject_secret_role() {
  local secret="$1" account="$2"
  if gcloud secrets get-iam-policy "${secret}" --project="${PROJECT_ID}" \
    --flatten='bindings[].members' \
    --filter="bindings.role=roles/secretmanager.secretAccessor AND bindings.members=serviceAccount:${account}" \
    --format='value(bindings.role)' | grep -Fxq 'roles/secretmanager.secretAccessor'; then
    echo "forbidden secret access: ${account} -> ${secret}" >&2
    exit 2
  fi
}

require_project_role "${PUBLIC_SERVICE_ACCOUNT}" roles/datastore.user
require_project_role "${PUBLIC_SERVICE_ACCOUNT}" roles/cloudtasks.enqueuer
require_project_role "${WORKER_SERVICE_ACCOUNT}" roles/datastore.user
require_project_role "${WORKER_SERVICE_ACCOUNT}" roles/cloudtasks.enqueuer
require_secret_role line-bot-channel-secret "${PUBLIC_SERVICE_ACCOUNT}"
require_secret_role line-bot-channel-access-token "${WORKER_SERVICE_ACCOUNT}"
reject_project_role "${PUBLIC_SERVICE_ACCOUNT}" roles/secretmanager.secretAccessor
reject_project_role "${WORKER_SERVICE_ACCOUNT}" roles/secretmanager.secretAccessor
reject_secret_role line-bot-channel-access-token "${PUBLIC_SERVICE_ACCOUNT}"
reject_secret_role line-bot-channel-secret "${WORKER_SERVICE_ACCOUNT}"

# Listing is read-only. The signed staging evidence is the authoritative record
# that the required definitions are Ready, not merely declared by the API.
gcloud firestore indexes composite list --database='(default)' --project="${PROJECT_ID}" --format=json >/dev/null
gcloud firestore fields ttls list --database='(default)' --project="${PROJECT_ID}" --format=json >/dev/null

echo "read-only preflight passed for ${PROJECT_ID}/${REGION}"
