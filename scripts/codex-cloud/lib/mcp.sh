#!/usr/bin/env bash

codex_cloud_enabled_mcp_names() {
  printf '%s\n' \
    gopls \
    repomix \
    context7 \
    github \
    openaiDeveloperDocs \
    google_developer_knowledge \
    google_cloud_cli \
    terraform \
    line \
    openapi_line
}

codex_cloud_ensure_project_trusted() {
  local config_dir="${CODEX_HOME:-$HOME/.codex}"
  local config_file="$config_dir/config.toml"
  local project_section='[projects."/workspace/linebot-platform"]'

  mkdir -p "$config_dir"
  if [[ -f "$config_file" ]] && grep -Fqx "$project_section" "$config_file"; then
    if ! awk -v section="$project_section" '
      $0 == section { in_project = 1; next }
      in_project && /^\[/ { exit }
      in_project && $0 ~ /^trust_level[[:space:]]*=[[:space:]]*"trusted"[[:space:]]*$/ { found = 1 }
      END { exit(found ? 0 : 1) }
    ' "$config_file"; then
      codex_cloud_fail "project trust is not trusted in $config_file"
      return 1
    fi
    return 0
  fi

  {
    printf '\n%s\n' "$project_section"
    printf 'trust_level = "trusted"\n'
  } >> "$config_file"
  codex_cloud_log "trusted project config: /workspace/linebot-platform/.codex/config.toml"
}

codex_cloud_mcp_registration_check() {
  local output name status=0
  output="$(codex mcp list 2>&1)" || { printf '%s\n' "$output" >&2; return 1; }
  printf '%s\n' "$output"
  for name in $(codex_cloud_enabled_mcp_names); do
    if ! printf '%s\n' "$output" | awk -v wanted="$name" '$1 == wanted { found = 1 } END { exit(found ? 0 : 1) }'; then
      printf 'MCP registration not visible: %s\n' "$name" >&2
      status=1
    else
      printf 'MCP registration visible: %s\n' "$name"
    fi
  done
  if ! printf '%s\n' "$output" | grep -qE '^firestore_toolbox[[:space:]].*[[:space:]]disabled([[:space:]]|$)'; then
    printf 'MCP disabled state not verified: firestore_toolbox\n' >&2
    status=1
  else
    printf 'MCP disabled as expected: firestore_toolbox\n'
  fi
  printf 'MCP handshake: deferred to fresh task read-only calls\n'
  return "$status"
}
