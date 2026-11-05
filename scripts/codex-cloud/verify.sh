#!/usr/bin/env bash
set -Eeuo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
source "$script_dir/lib/common.sh"
source "$script_dir/lib/paths.sh"
source "$script_dir/lib/skills.sh"
source "$script_dir/lib/mcp.sh"
repo_root="$(codex_cloud_repo_root)"
cd "$repo_root"
bash "$script_dir/preflight.sh"

status=READY
failures=()
warn() { failures+=("$*"); [[ "$status" == READY ]] && status=DEGRADED; }
block() { failures+=("$*"); status=BLOCKED; }
for command in go git codex; do command -v "$command" >/dev/null 2>&1 || block "missing $command"; done
command -v gopls >/dev/null 2>&1 || warn "missing gopls"
command -v go >/dev/null 2>&1 && printf 'go=%s\n' "$(go version)"
command -v gopls >/dev/null 2>&1 && printf 'gopls=%s\n' "$(gopls version | head -n 1)"
if command -v go >/dev/null 2>&1; then
  go run ./cmd/codexcheck || block "repository Codex configuration validation failed"
  codex_cloud_skills_direct_discovery_check || warn "direct samber skill layout is not verified"
  codex_cloud_skill_source_check "$repo_root" cc-skills 21 || warn "cc-skills count or pinned source commit is not verified"
  codex_cloud_skill_source_check "$repo_root" cc-skills-golang 46 || warn "cc-skills-golang count or pinned source commit is not verified"
	codex_cloud_google_plugin_check || warn "Google developer plugin is not installed and enabled"
else
  block "Go-dependent checks skipped"
fi
if command -v codex >/dev/null 2>&1; then
  codex_cloud_mcp_registration_check || warn "one or more MCP registrations are not visible to this CLI context"
fi
(("${#failures[@]}")) && printf '%s\n' "${failures[@]}" >&2
printf '%s\n' "$status"
[[ "$status" == BLOCKED ]] && exit 2
[[ "$status" == DEGRADED ]] && exit 1
