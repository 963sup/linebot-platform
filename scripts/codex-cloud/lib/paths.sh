#!/usr/bin/env bash

# Single repository-owned PATH contract for every Codex Cloud phase.
# Task-phase HOME is read-only, so mutable tools and caches live in /workspace.
linebot_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)"
export CODEX_CLOUD_TOOLS="${CODEX_CLOUD_TOOLS:-$linebot_root/.cache/tools}"
export CODEX_CLOUD_BIN="${CODEX_CLOUD_BIN:-$CODEX_CLOUD_TOOLS/bin}"
export MISE_DATA_DIR="${MISE_DATA_DIR:-$CODEX_CLOUD_TOOLS/mise}"
export MISE_CACHE_DIR="${MISE_CACHE_DIR:-$CODEX_CLOUD_TOOLS/cache}"
export MISE_CONFIG_DIR="${MISE_CONFIG_DIR:-$CODEX_CLOUD_TOOLS/config}"
export MISE_STATE_DIR="${MISE_STATE_DIR:-$CODEX_CLOUD_TOOLS/state}"
export PATH="$CODEX_CLOUD_BIN:$MISE_DATA_DIR/shims:$PATH"
export GOCACHE="${GOCACHE:-$linebot_root/.cache/go-build}"
export GOPATH="${GOPATH:-$linebot_root/.cache/go}"
export XDG_CACHE_HOME="${XDG_CACHE_HOME:-$linebot_root/.cache}"
export GOFLAGS="${GOFLAGS:--mod=readonly}"
export GOTOOLCHAIN="${GOTOOLCHAIN:-local}"
hash -r
