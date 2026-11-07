#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
script="${script_dir}/preflight.sh"
promotion="${script_dir}/promote.sh"

if grep -Eiq 'gcloud[[:space:]].*(create|deploy|update|delete|add-iam-policy-binding|remove-iam-policy-binding)' "${script}"; then
  echo "preflight contains a mutating gcloud command" >&2
  exit 1
fi
for command in 'artifacts docker images describe' 'tasks queues describe' 'firestore databases describe' 'run services describe' 'iam service-accounts describe' 'secrets describe' 'projects get-iam-policy' 'secrets get-iam-policy' 'firestore indexes composite list' 'firestore fields ttls list'; do
  grep -Fq "${command}" "${script}"
done
grep -Fq 'approved_for_production == true' "${script}"
grep -Fq 'worker_capacity' "${script}"
grep -Fq 'STAGING_EVIDENCE_SHA256' "${script}"
grep -Fq 'authoritative_worker_origin' "${script}"
grep -Fq '.image_digests.miniapp' "${script}"
grep -Fq '.miniapp_config.endpoint_url' "${script}"
grep -Fq 'evidence_miniapp_endpoint_url' "${script}"
grep -Fq 'evidence_miniapp_liff_url' "${script}"
grep -Fq 'evidence_miniapp_channel_id' "${script}"
grep -Fq 'MINI App deployment configuration does not match approved staging evidence' "${script}"
grep -Fq 'ATTENDANCE_LOCATION_KEY_VERSIONS must contain pinned attendance-location-key resources' "${script}"
grep -Fq 'versions/([1-9][0-9]*)$' "${script}"
grep -Fq 'Attendance encryption configuration does not match approved staging evidence' "${script}"
grep -Fq 'require_project_role "${MINIAPP_SERVICE_ACCOUNT}" roles/datastore.user' "${script}"
grep -Fq 'require_secret_role attendance-location-key "${MINIAPP_SERVICE_ACCOUNT}"' "${script}"
grep -Fq 'miniapp_identity' "${script}"
grep -Fq 'attendance_enrollment' "${script}"
grep -Fq 'attendance_concurrency' "${script}"
grep -Fq 'attendance_review' "${script}"
grep -Fq 'attendance_retention' "${script}"
grep -Fq 'reject_secret_role line-bot-channel-secret "${MINIAPP_SERVICE_ACCOUNT}"' "${script}"

grep -Fq 'TARGET_PERCENT must be one of 10, 50, 100' "${promotion}"
grep -Fq '.smoke_passed == true' "${promotion}"
grep -Fq '.expires_at_utc' "${promotion}"
if grep -Fq 'latestReadyRevisionName' "${promotion}"; then
  echo 'promotion must not infer a latest revision' >&2
  exit 1
fi

echo 'preflight static checks passed'
