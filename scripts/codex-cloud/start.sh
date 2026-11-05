#!/usr/bin/env bash
# Start phase: make the checkout exactly match remote main, then verify readiness.
set -Eeuo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$script_dir/lib/common.sh"
source "$script_dir/lib/paths.sh"
repo_root="$(codex_cloud_repo_root)"
cd "$repo_root"
codex_cloud_require git
git fetch --prune origin main
git switch --quiet main
git reset --hard origin/main
git clean -ffdx
test "$(git rev-parse HEAD)" = "$(git rev-parse origin/main)"
codex_cloud_log "synchronized main at $(git rev-parse --short HEAD)"
exec bash "$repo_root/scripts/codex-cloud/verify.sh"
