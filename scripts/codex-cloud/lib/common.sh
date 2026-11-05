#!/usr/bin/env bash

codex_cloud_log() { printf '[codex-cloud] %s\n' "$*"; }
codex_cloud_fail() { codex_cloud_log "BLOCKED: $*" >&2; return 1; }
codex_cloud_require() { command -v "$1" >/dev/null 2>&1 || codex_cloud_fail "missing command: $1"; }
codex_cloud_repo_root() { cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd; }
