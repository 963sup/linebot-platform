#!/usr/bin/env bash
set -Eeuo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
printf '[codex-cloud] precheck\n'
bash "$script_dir/verify.sh" || true
if [[ "${ALLOW_LOCAL_TOOL_REPAIR:-}" != 1 ]]; then
  printf '%s\n' '[codex-cloud] no repair performed; set ALLOW_LOCAL_TOOL_REPAIR=1 for pinned local repair'
  exit 0
fi
printf '%s\n' '[codex-cloud] repairing pinned local tools only'
bash "$script_dir/install.sh"
bash "$script_dir/verify.sh"
