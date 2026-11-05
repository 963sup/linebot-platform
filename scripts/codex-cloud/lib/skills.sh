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
