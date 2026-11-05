#!/usr/bin/env bash

# Single repository-owned PATH contract for every Codex Cloud phase.
# Task-phase HOME is read-only, so mutable tools and caches live in /workspace.
export CODEX_CLOUD_TOOLS="${CODEX_CLOUD_TOOLS:-/workspace/.linebot-tools}"
export CODEX_CLOUD_BIN="${CODEX_CLOUD_BIN:-$CODEX_CLOUD_TOOLS/bin}"
if [[ -z "${CODEX_CLOUD_NODE_BIN:-}" ]]; then
  CODEX_CLOUD_NODE_BIN="$(dirname -- "$(command -v node 2>/dev/null || printf '/usr/local/bin/node')")"
fi
export CODEX_CLOUD_NODE_BIN
export PATH="$CODEX_CLOUD_BIN:$CODEX_CLOUD_TOOLS/go/bin:$CODEX_CLOUD_NODE_BIN:/opt/codex/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
export GOCACHE="${GOCACHE:-/workspace/.gocache}"
export GOPATH="${GOPATH:-/workspace/.gopath}"
export XDG_CACHE_HOME="${XDG_CACHE_HOME:-/workspace/.cache}"
export GOFLAGS="${GOFLAGS:--mod=readonly}"
export GOMAXPROCS="${GOMAXPROCS:-5}"
hash -r
