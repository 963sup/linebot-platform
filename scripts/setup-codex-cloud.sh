#!/usr/bin/env bash
# Codex Cloud Install script. Prepares tools; never deploys or authorizes MCPs.
set -Eeuo pipefail
umask 077

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

if ! command -v go >/dev/null 2>&1; then
  echo "Go is missing. Configure the toolchain required by go.mod first." >&2
  exit 1
fi
for setting in CODEX_CLOUD_INSTALL_SKILLS CODEX_CLOUD_REGISTER_GOOGLE_MARKETPLACE CODEX_CLOUD_WARM_TESTS; do
  value="${!setting:-0}"
  case "$value" in
    0|1) ;;
    *) echo "$setting must be 0 or 1." >&2; exit 1 ;;
  esac
done
if [ "${CODEX_CLOUD_REGISTER_GOOGLE_MARKETPLACE:-0}" = "1" ] && [ "${CODEX_CLOUD_INSTALL_SKILLS:-0}" != "1" ]; then
  echo "Marketplace registration requires CODEX_CLOUD_INSTALL_SKILLS=1." >&2
  exit 1
fi

# Do not rely on setup-shell PATH persisting into the agent phase.
export GOTOOLCHAIN=auto
printf 'Go toolchain: %s\n' "$(go version)"
required_go="$(awk '$1 == "go" { print $2; exit }' go.mod)"
test -n "$required_go" || { echo "go.mod has no Go version" >&2; exit 1; }
printf 'Module requires Go %s\n' "$required_go"

go mod download
go mod verify
go run ./cmd/codexcheck

gopls_version="v0.24.0"
bin_dir="$HOME/.local/bin"
mkdir -p "$bin_dir"
need_install=1
if [ -x "$bin_dir/gopls" ] && "$bin_dir/gopls" version 2>/dev/null | grep -Eq "(^|[[:space:]])${gopls_version//./\\.}([[:space:]]|$)"; then
  need_install=0
fi
if [ "$need_install" -eq 1 ]; then
  GOBIN="$bin_dir" go install "golang.org/x/tools/gopls@$gopls_version"
fi
"$bin_dir/gopls" version

if [ "${CODEX_CLOUD_INSTALL_SKILLS:-0}" = "1" ]; then
  plugin_args=()
  if [ "${CODEX_CLOUD_REGISTER_GOOGLE_MARKETPLACE:-0}" = "1" ]; then
    plugin_args+=(--register-google-marketplace)
  fi
  bash scripts/setup-codex-plugins.sh "${plugin_args[@]}"
else
  status=0
  bash scripts/check-codex-plugins.sh || status=$?
  case "$status" in
    0) ;;
    2) echo "Optional skills missing or blocked; no installation was requested." ;;
    *) echo "Plugin preflight failed." >&2; exit "$status" ;;
  esac
fi

if [ "${CODEX_CLOUD_WARM_TESTS:-0}" = "1" ]; then
  go test ./...
fi

echo "Go and gopls preparation completed. Host role/skill loading and MCP authentication remain unverified."
