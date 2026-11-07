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
export XDG_CONFIG_HOME="$BOOTSTRAP_WORKSPACE/.config"
export XDG_DATA_HOME="$BOOTSTRAP_WORKSPACE/.local/share"
export XDG_STATE_HOME="$BOOTSTRAP_WORKSPACE/.local/state"
export NPM_CONFIG_CACHE="$BOOTSTRAP_WORKSPACE/.cache/npm"
export GOCACHE="$BOOTSTRAP_WORKSPACE/.cache/go-build"
export GOPATH="$BOOTSTRAP_WORKSPACE/.cache/go"
export GOMODCACHE="$GOPATH/pkg/mod"
export GOBIN="$GOPATH/bin"
export UV_CACHE_DIR="$BOOTSTRAP_WORKSPACE/.cache/uv"
export UV_TOOL_DIR="$BOOTSTRAP_WORKSPACE/.local/share/uv/tools"
export UV_TOOL_BIN_DIR="$BOOTSTRAP_WORKSPACE/.local/bin"
export UV_PYTHON_INSTALL_DIR="$BOOTSTRAP_WORKSPACE/.local/share/uv/python"
export PYTHONUSERBASE="$BOOTSTRAP_WORKSPACE/.local/share/python"
# Preserve host-provided credentials/configuration for authenticated commands.
export CLOUDSDK_CONFIG="${CLOUDSDK_CONFIG:-$BOOTSTRAP_WORKSPACE/.cache/gcloud-install}"
export MISE_AUTO_INSTALL=0
export MISE_EXEC_AUTO_INSTALL=0
mise="$BOOTSTRAP_WORKSPACE/.tools/bin/mise"
cd "$BOOTSTRAP_REPO"

# Downloads happen only in the explicit setup phase, never on MCP startup.
if [[ "${1:-setup}" == setup ]]; then
  mkdir -p "$(dirname "$mise")" "$MISE_DATA_DIR" "$MISE_CACHE_DIR" "$MISE_CONFIG_DIR" "$MISE_STATE_DIR"
  for tool_dir in "$XDG_CACHE_HOME" "$XDG_CONFIG_HOME" "$XDG_DATA_HOME" "$XDG_STATE_HOME" \
    "$NPM_CONFIG_CACHE" "$GOCACHE" "$GOPATH" "$GOMODCACHE" "$GOBIN" \
    "$UV_CACHE_DIR" "$UV_TOOL_DIR" "$UV_TOOL_BIN_DIR" "$UV_PYTHON_INSTALL_DIR" "$PYTHONUSERBASE"; do
    mkdir -p "$tool_dir"
    [[ -w "$tool_dir" ]] || { printf 'Tool directory is not writable: %s\n' "$tool_dir" >&2; exit 1; }
  done
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
