#!/usr/bin/env bash

codex_cloud_mcp_registration_check() {
  local output name status=0
  output="$(codex mcp list 2>&1)" || { printf '%s\n' "$output" >&2; return 1; }
  printf '%s\n' "$output"
  for name in gopls repomix context7 github openaiDeveloperDocs; do
    if ! printf '%s\n' "$output" | grep -qE "(^|[[:space:]])${name}([[:space:]]|$)"; then
      printf 'MCP registration not visible: %s\n' "$name" >&2
      status=1
    fi
  done
  return "$status"
}
