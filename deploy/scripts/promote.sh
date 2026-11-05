#!/usr/bin/env bash
set -euo pipefail

# Promote exactly one service by one explicitly approved traffic step.
required=(PROJECT_ID REGION SERVICE CANDIDATE_REVISION BASE_REVISION TARGET_PERCENT PROMOTION_EVIDENCE_FILE PROMOTION_EVIDENCE_SHA256)
for name in "${required[@]}"; do
  [[ -n "${!name:-}" ]] || { echo "missing required promotion input: ${name}" >&2; exit 2; }
done
case "${TARGET_PERCENT}" in 10|50|100) ;; *) echo 'TARGET_PERCENT must be one of 10, 50, 100' >&2; exit 2;; esac
if [[ ! "${PROMOTION_EVIDENCE_SHA256}" =~ ^[0-9a-f]{64}$ ]] || [[ ! -f "${PROMOTION_EVIDENCE_FILE}" ]]; then
  echo 'promotion evidence file and lowercase SHA-256 are required' >&2
  exit 2
fi
actual_digest="$(sha256sum "${PROMOTION_EVIDENCE_FILE}" | awk '{print $1}')"
[[ "${actual_digest}" == "${PROMOTION_EVIDENCE_SHA256}" ]] || { echo 'promotion evidence digest mismatch' >&2; exit 2; }

jq -e --arg project "${PROJECT_ID}" --arg region "${REGION}" --arg service "${SERVICE}" \
  --arg candidate "${CANDIDATE_REVISION}" --arg base "${BASE_REVISION}" --argjson percent "${TARGET_PERCENT}" '
  .schema_version == 1 and .approved == true and .smoke_passed == true and
  .project_id == $project and .region == $region and .service == $service and
  .candidate_revision == $candidate and .base_revision == $base and
  .approved_percent == $percent and
  (.reviewer | type == "string" and length > 0) and
  (.observed_at_utc | type == "string" and length > 0) and
  (.expires_at_utc | type == "string" and length > 0)
' "${PROMOTION_EVIDENCE_FILE}" >/dev/null

expires_at="$(jq -r '.expires_at_utc' "${PROMOTION_EVIDENCE_FILE}")"
if ! expires_epoch="$(date -u -d "${expires_at}" +%s 2>/dev/null)" || (( expires_epoch <= $(date -u +%s) )); then
  echo 'promotion evidence is expired or has an invalid expires_at_utc' >&2
  exit 2
fi

gcloud run revisions describe "${CANDIDATE_REVISION}" --region="${REGION}" --project="${PROJECT_ID}" --quiet >/dev/null
gcloud run revisions describe "${BASE_REVISION}" --region="${REGION}" --project="${PROJECT_ID}" --quiet >/dev/null
if [[ "${TARGET_PERCENT}" == 100 ]]; then
  allocation="${CANDIDATE_REVISION}=100"
else
  allocation="${CANDIDATE_REVISION}=${TARGET_PERCENT},${BASE_REVISION}=$((100 - TARGET_PERCENT))"
fi
gcloud run services update-traffic "${SERVICE}" --region="${REGION}" --project="${PROJECT_ID}" --to-revisions="${allocation}" --quiet
echo "explicit ${TARGET_PERCENT}% promotion completed for ${SERVICE}"
