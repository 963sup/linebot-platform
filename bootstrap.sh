#!/usr/bin/env bash
# One entry point for setup and for launching installed tools. Never resets Git.
set -Eeuo pipefail
repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
export MISE_DATA_DIR=/workspace/.linebot-tools/mise
export MISE_CACHE_DIR=/workspace/.cache/mise
export MISE_CONFIG_DIR=/workspace/.config/mise
export MISE_STATE_DIR=/workspace/.local/state/mise
export XDG_CACHE_HOME=/workspace/.cache
export MISE_AUTO_INSTALL=0
export MISE_EXEC_AUTO_INSTALL=0
mise=/workspace/.linebot-tools/bin/mise
cd "$repo_root"

# Downloads happen only in the explicit setup phase, never on MCP startup.
if [[ "${1:-setup}" == setup ]]; then
  mkdir -p "$(dirname "$mise")" "$MISE_DATA_DIR" "$MISE_CACHE_DIR" "$MISE_CONFIG_DIR" "$MISE_STATE_DIR"
  if [[ ! -x "$mise" ]]; then
    archive="$(mktemp /workspace/.linebot-tools/mise.XXXXXX)"
    trap 'rm -f -- "$archive"' EXIT
    curl -fsSL --connect-timeout 15 --max-time 300 --retry 2 https://github.com/jdx/mise/releases/download/v2026.11.3/mise-v2026.11.3-linux-x64 -o "$archive"
    printf '%s  %s\n' c2f7e814cb39f60b7f67a0ff36cb805e94128bf2dff09d175309d0f095b8cc3b "$archive" | sha256sum -c - >&2
    install -m 0755 "$archive" "$mise"
    rm -f -- "$archive"
    trap - EXIT
  fi
  [[ "$("$mise" --version)" == '2026.11.3 '* ]] || { echo 'Expected mise 2026.11.3' >&2; exit 1; }
  "$mise" trust "$repo_root/mise.toml"
  "$mise" install node npm:repomix go go:golang.org/x/tools/gopls
  exec "$mise" run setup
fi
[[ -x "$mise" ]] || { echo 'Run bash bootstrap.sh setup before starting Codex.' >&2; exit 1; }
exec "$mise" "$@"
