#!/usr/bin/env bash

codex_cloud_skills_check() {
	local repo_root="$1" binary status=0
  binary="$(mktemp "${TMPDIR:-/tmp}/codexplugins.XXXXXX")"
  go build -o "$binary" "$repo_root/cmd/codexplugins"
	"$binary" || status=$?
	rm -f -- "$binary"
	return "$status"
}

codex_cloud_skills_install() {
	local repo_root="$1" binary status=0
  shift
  binary="$(mktemp "${TMPDIR:-/tmp}/codexplugins.XXXXXX")"
  go build -o "$binary" "$repo_root/cmd/codexplugins"
	"$binary" --install "$@" || status=$?
	rm -f -- "$binary"
	return "$status"
}

codex_cloud_google_plugin_check() {
	codex plugin list 2>/dev/null |
		grep -qE '^google-cloud-developer@google-plugins[[:space:]]+installed, enabled'
}

codex_cloud_skills_direct_discovery_check() {
  local root="$HOME/.agents/skills"
  test -f "$root/conventional-git/SKILL.md"
  test -f "$root/golang-code-style/SKILL.md"
  test ! -e "$root/cc-skills"
  test ! -e "$root/cc-skills-golang"
}

codex_cloud_skill_source_check() {
  local repo_root="$1" source_name="$2" expected_count="$3"
  local root="$HOME/.agents/skills"
  local source_dir="/workspace/.linebot-tools/skill-sources/$source_name"
  local expected_commit actual_commit marker marker_source count=0

  expected_commit="$(awk -v wanted="$source_name" '
    $0 ~ "\"name\"[[:space:]]*:[[:space:]]*\"" wanted "\"" { found=1 }
    found && $0 ~ /\"commit\"[[:space:]]*:/ {
      line=$0
      sub(/^.*\"commit\"[[:space:]]*:[[:space:]]*\"/, "", line)
      sub(/\".*$/, "", line)
      print line
      exit
    }
  ' "$repo_root/.codex/plugins.lock.json")"
  test -n "$expected_commit" || return 1
  test -d "$source_dir/.git" || return 1
  actual_commit="$(git -C "$source_dir" rev-parse HEAD)"
  test "$actual_commit" = "$expected_commit" || return 1

  while IFS= read -r marker; do
    marker_source="$(cat "$marker")"
    [[ "$marker_source" == "$source_name" ]] && count=$((count + 1))
  done < <(find "$root" -mindepth 2 -maxdepth 2 -type f -name .codex-managed-source -print)
  test "$count" -eq "$expected_count" || return 1
  printf '%s installed=%s commit=%s host-discovery=unverified\n' "$source_name" "$count" "$actual_commit"
}
