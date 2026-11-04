#!/usr/bin/env bash
set -Eeuo pipefail

printf '[cloud-repair] precheck\n'
bash scripts/cloud-verify.sh || true

if [[ "${ALLOW_LOCAL_TOOL_REPAIR:-}" != "1" ]]; then
  printf '%s\n' '[cloud-repair] no repair performed; set ALLOW_LOCAL_TOOL_REPAIR=1 for pinned local tool repair only.'
  exit 0
fi

printf '%s\n' '[cloud-repair] change: running approved local bootstrap only'
bash scripts/cloud-bootstrap.sh
printf '%s\n' '[cloud-repair] postcheck'
bash scripts/cloud-verify.sh
printf '%s\n' '[cloud-repair] evidence: local tool repair completed; no OAuth, IAM, deployment, or cloud data write was attempted.'
