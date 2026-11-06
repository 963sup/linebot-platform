#!/usr/bin/env bash
set -Eeuo pipefail
repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$repo_root/scripts/codex-cloud/lib/common.sh"
source "$repo_root/scripts/codex-cloud/lib/paths.sh"
source "$repo_root/scripts/codex-cloud/lib/mise.sh"
mkdir -p "$CODEX_CLOUD_BIN" "$MISE_DATA_DIR" "$MISE_CACHE_DIR" "$MISE_CONFIG_DIR" "$MISE_STATE_DIR"
codex_cloud_mise_install "$repo_root"
cd "$repo_root"
if (($#)); then exec "$CODEX_CLOUD_BIN/mise" "$@"; fi
printf 'Ready: bash bootstrap.sh install; bash bootstrap.sh run setup\n'
