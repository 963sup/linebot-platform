#!/usr/bin/env bash
# One entry point for setup and for launching installed tools. Never resets Git.
set -Eeuo pipefail
export BOOTSTRAP_WORKSPACE=/workspace
export BOOTSTRAP_REPO=/workspace/linebot-platform
script_repo="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
[[ "$script_repo" == "$BOOTSTRAP_REPO" ]] || { echo 'bootstrap.sh location must match BOOTSTRAP_REPO' >&2; exit 1; }
export MISE_DATA_DIR="$BOOTSTRAP_WORKSPACE/.tools/mise"
export MISE_CACHE_DIR="$BOOTSTRAP_WORKSPACE/.cache/mise"
export MISE_CONFIG_DIR="$BOOTSTRAP_WORKSPACE/.config/mise"
export MISE_STATE_DIR="$BOOTSTRAP_WORKSPACE/.local/state/mise"
export XDG_CACHE_HOME="$BOOTSTRAP_WORKSPACE/.cache"
export NPM_CONFIG_CACHE="$BOOTSTRAP_WORKSPACE/.cache/npm"
export GOCACHE="${GOCACHE:-$BOOTSTRAP_WORKSPACE/.cache/go-build}"
export GOPATH="${GOPATH:-$BOOTSTRAP_WORKSPACE/.cache/go}"
export CLOUDSDK_CONFIG="${CLOUDSDK_CONFIG:-$BOOTSTRAP_WORKSPACE/.cache/gcloud-install}"
export MISE_AUTO_INSTALL=0
export MISE_EXEC_AUTO_INSTALL=0
mise="$BOOTSTRAP_WORKSPACE/.tools/bin/mise"
cd "$BOOTSTRAP_REPO"

# Downloads happen only in the explicit setup phase, never on MCP startup.
if [[ "${1:-setup}" == setup ]]; then
  mkdir -p "$(dirname "$mise")" "$MISE_DATA_DIR" "$MISE_CACHE_DIR" "$MISE_CONFIG_DIR" "$MISE_STATE_DIR"
  if [[ ! -x "$mise" ]]; then
    archive="$(mktemp "$BOOTSTRAP_WORKSPACE/.tools/mise.XXXXXX")"
    trap 'rm -f -- "$archive"' EXIT
    curl -fsSL --connect-timeout 15 --max-time 300 --retry 2 https://github.com/jdx/mise/releases/download/v2026.11.3/mise-v2026.11.3-linux-x64 -o "$archive"
    printf '%s  %s\n' c2f7e814cb39f60b7f67a0ff36cb805e94128bf2dff09d175309d0f095b8cc3b "$archive" | sha256sum -c - >&2
    install -m 0755 "$archive" "$mise"
    rm -f -- "$archive"
    trap - EXIT
  fi
  [[ "$("$mise" --version)" == '2026.11.3 '* ]] || { echo 'Expected mise 2026.11.3' >&2; exit 1; }
  "$mise" trust "$BOOTSTRAP_REPO/mise.toml"
  "$mise" install
  exec "$mise" run setup
fi
[[ -x "$mise" ]] || { echo 'Run bash bootstrap.sh setup before starting Codex.' >&2; exit 1; }
exec "$mise" "$@"
