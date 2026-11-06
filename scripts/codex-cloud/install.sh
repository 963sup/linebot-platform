#!/usr/bin/env bash
# Codex Cloud install phase: pinned local preparation only.
set -Eeuo pipefail
umask 077

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$script_dir/lib/common.sh"
source "$script_dir/lib/paths.sh"
source "$script_dir/lib/skills.sh"
source "$script_dir/lib/mcp.sh"
source "$script_dir/lib/mcp_runtime.sh"
repo_root="$(codex_cloud_repo_root)"
cd "$repo_root"
bash "$script_dir/preflight.sh"
mkdir -p "$CODEX_CLOUD_BIN" "$GOCACHE" "$GOPATH" "$XDG_CACHE_HOME"
for command in go sha256sum docker; do
  codex_cloud_require "$command"
done
codex_cloud_ensure_project_trusted

for setting in CODEX_CLOUD_INSTALL_SKILLS CODEX_CLOUD_REGISTER_GOOGLE_MARKETPLACE CODEX_CLOUD_WARM_TESTS; do
  value="${!setting:-0}"
  case "$value" in 0|1) ;; *) codex_cloud_fail "$setting must be 0 or 1"; exit 1 ;; esac
done
if [[ "${CODEX_CLOUD_REGISTER_GOOGLE_MARKETPLACE:-0}" == 1 && "${CODEX_CLOUD_INSTALL_SKILLS:-0}" != 1 ]]; then
  codex_cloud_fail "marketplace registration requires CODEX_CLOUD_INSTALL_SKILLS=1"
  exit 1
fi

export GOTOOLCHAIN=auto
codex_cloud_log "Go toolchain: $(go version)"
required_go="$(awk '$1 == "go" { print $2; exit }' go.mod)"
test -n "$required_go" || { codex_cloud_fail "go.mod has no Go version"; exit 1; }
go mod download
go mod verify
go run ./cmd/codexcheck

gopls_version=v0.24.0
if [[ ! -x "$CODEX_CLOUD_BIN/gopls" ]] || ! "$CODEX_CLOUD_BIN/gopls" version 2>/dev/null | grep -Eq "(^|[[:space:]])${gopls_version//./\\.}([[:space:]]|$)"; then
  GOBIN="$CODEX_CLOUD_BIN" go install "golang.org/x/tools/gopls@$gopls_version"
fi
"$CODEX_CLOUD_BIN/gopls" version

codex_cloud_mcp_runtime_install "$repo_root"
codex_cloud_mcp_runtime_check "$repo_root"
docker pull hashicorp/terraform-mcp-server:1.4.1

if [[ "${CODEX_CLOUD_INSTALL_SKILLS:-0}" == 1 ]]; then
  plugin_args=()
  [[ "${CODEX_CLOUD_REGISTER_GOOGLE_MARKETPLACE:-0}" == 1 ]] && plugin_args+=(--register-google-marketplace)
  codex_cloud_skills_install "$repo_root" "${plugin_args[@]}"
  if [[ "${CODEX_CLOUD_REGISTER_GOOGLE_MARKETPLACE:-0}" == 1 ]]; then
    codex_cloud_google_plugin_check || codex_cloud_google_plugin_install
  fi
else
  codex_cloud_skills_check "$repo_root" || codex_cloud_log "DEGRADED: optional approved skills are not fully verified"
fi
codex_cloud_mcp_registration_check
[[ "${CODEX_CLOUD_WARM_TESTS:-0}" == 1 ]] && go test ./...
codex_cloud_log "install completed; host skill discovery and live MCP calls require a fresh task"
