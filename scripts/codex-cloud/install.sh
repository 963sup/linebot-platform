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
source "$script_dir/lib/mise.sh"
repo_root="$(codex_cloud_repo_root)"
cd "$repo_root"
bash "$script_dir/preflight.sh"
mkdir -p "$CODEX_CLOUD_BIN" "$MISE_DATA_DIR" "$MISE_CACHE_DIR" "$MISE_CONFIG_DIR" "$MISE_STATE_DIR" "$GOCACHE" "$GOPATH" "$XDG_CACHE_HOME"
for command in curl jq sha256sum docker; do
  codex_cloud_require "$command"
done
codex_cloud_mise_install "$repo_root"
mise install --yes
mise reshim
eval "$(mise env -s bash)"
for command in go node npm gopls; do
  codex_cloud_require "$command"
done

for setting in CODEX_CLOUD_INSTALL_SKILLS CODEX_CLOUD_REGISTER_GOOGLE_MARKETPLACE CODEX_CLOUD_WARM_TESTS; do
  value="${!setting:-0}"
  case "$value" in 0|1) ;; *) codex_cloud_fail "$setting must be 0 or 1"; exit 1 ;; esac
done
if [[ "${CODEX_CLOUD_REGISTER_GOOGLE_MARKETPLACE:-0}" == 1 && "${CODEX_CLOUD_INSTALL_SKILLS:-0}" != 1 ]]; then
  codex_cloud_fail "marketplace registration requires CODEX_CLOUD_INSTALL_SKILLS=1"
  exit 1
fi

required_go="$(awk '$1 == "go" { print $2; exit }' go.mod)"
test -n "$required_go" || { codex_cloud_fail "go.mod has no Go version"; exit 1; }
[[ "$(go version | awk '{print $3}')" == "go${required_go}" ]] || { codex_cloud_fail "Go ${required_go} is required"; exit 1; }
codex_cloud_log "mise: $(mise --version | head -n 1)"
codex_cloud_log "Go toolchain: $(go version)"
go mod download
go mod verify
go run ./cmd/codexcheck
gopls version

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
[[ "${CODEX_CLOUD_WARM_TESTS:-0}" == 1 ]] && go test ./...
codex_cloud_log "install completed; Host project trust, MCP registration, skill discovery and live MCP calls require a fresh task"
