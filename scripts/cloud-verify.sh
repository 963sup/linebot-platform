#!/usr/bin/env bash
set -Eeuo pipefail

status=READY
failures=()
export PATH="${HOME}/.local/bin:/workspace/.linebot-tools/go/bin:${PATH}"
export GOCACHE="${GOCACHE:-/workspace/.gocache}"
mkdir -p "$GOCACHE"
warn() { failures+=("$*"); [[ "$status" == READY ]] && status=DEGRADED; }
block() { failures+=("$*"); status=BLOCKED; }

command -v go >/dev/null 2>&1 || block "missing go"
command -v git >/dev/null 2>&1 || block "missing git"
command -v codex >/dev/null 2>&1 || block "missing codex"
command -v gopls >/dev/null 2>&1 || warn "missing gopls"

if command -v go >/dev/null 2>&1; then printf 'go=%s\n' "$(go version)"; fi
if command -v gopls >/dev/null 2>&1; then printf 'gopls=%s\n' "$(gopls version | head -n 1)"; fi
if command -v codex >/dev/null 2>&1; then codex mcp list || warn "codex mcp list failed"; fi

if command -v go >/dev/null 2>&1; then
  if ! go run ./cmd/codexcheck; then block "codex configuration validation failed"; fi
else
  block "codex configuration validation skipped because go is missing"
fi
if ! bash scripts/check-codex-plugins.sh; then warn "approved Skills are missing or not verified"; fi

if ((${#failures[@]})); then
  printf '%s\n' "${failures[@]}" >&2
fi
printf '%s\n' "$status"
[[ "$status" == BLOCKED ]] && exit 2
[[ "$status" == DEGRADED ]] && exit 1
