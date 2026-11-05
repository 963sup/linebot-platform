#!/usr/bin/env bash
# Read-only checks that establish the execution contract before mutation.
set -Eeuo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$script_dir/lib/common.sh"
source "$script_dir/lib/paths.sh"

case "$PATH" in
  "$CODEX_CLOUD_BIN:$CODEX_CLOUD_TOOLS/go/bin:"*) ;;
  *) codex_cloud_fail "unexpected PATH precedence: $PATH"; exit 2 ;;
esac

for command in bash git; do
  command -v "$command" >/dev/null 2>&1 || { codex_cloud_fail "missing command: $command"; exit 2; }
done

test -d /workspace || { codex_cloud_fail "/workspace is missing"; exit 2; }
test -w /workspace || { codex_cloud_fail "/workspace is not writable"; exit 2; }
case "$CODEX_CLOUD_TOOLS:$CODEX_CLOUD_BIN:$GOCACHE:$GOPATH:$XDG_CACHE_HOME" in
  *"$HOME"*) codex_cloud_fail "mutable path resolves under read-only HOME"; exit 2 ;;
esac
for path in "$CODEX_CLOUD_TOOLS" "$CODEX_CLOUD_BIN" "$GOCACHE" "$GOPATH" "$XDG_CACHE_HOME"; do
  case "$path" in
    /workspace/*) ;;
    *) codex_cloud_fail "mutable path must stay under /workspace: $path"; exit 2 ;;
  esac
done

printf 'PATH=%s\n' "$PATH"
printf 'tools=%s\ngocache=%s\ngopath=%s\nxdg-cache=%s\n' "$CODEX_CLOUD_TOOLS" "$GOCACHE" "$GOPATH" "$XDG_CACHE_HOME"
codex_cloud_log "preflight READY"
