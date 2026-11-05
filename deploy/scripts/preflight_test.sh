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

grep -Fq 'TARGET_PERCENT must be one of 10, 50, 100' "${promotion}"
grep -Fq '.smoke_passed == true' "${promotion}"
grep -Fq '.expires_at_utc' "${promotion}"
if grep -Fq 'latestReadyRevisionName' "${promotion}"; then
  echo 'promotion must not infer a latest revision' >&2
  exit 1
fi

echo 'preflight static checks passed'
