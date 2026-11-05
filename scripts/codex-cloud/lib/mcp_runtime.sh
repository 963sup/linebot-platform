#!/usr/bin/env bash

codex_cloud_mcp_runtime_root() { printf '%s\n' "$CODEX_CLOUD_TOOLS/mcp-runtime"; }
codex_cloud_mcp_runtime_releases() { printf '%s\n' "$CODEX_CLOUD_TOOLS/mcp-runtime-releases"; }

codex_cloud_mcp_release_manifest() {
  local release="$1" output="$2"
  (cd "$release" && find . -type f ! -name installed.sha256 -print0 | LC_ALL=C sort -z | xargs -0 sha256sum) >"$output"
}

codex_cloud_mcp_release_symlinks() {
  local release="$1" output="$2"
  (cd "$release" && find . -type l -printf '%p\t%l\n' | LC_ALL=C sort) >"$output"
}

codex_cloud_mcp_release_check() {
  local release="$1" actual actual_links result link_path link_target resolved
  test -f "$release/installed.sha256" && test -f "$release/installed.symlinks" || return 1
  (cd "$release" && sha256sum -c installed.sha256 >/dev/null) || return 1
  actual="$(mktemp "$CODEX_CLOUD_TOOLS/.mcp-manifest.XXXXXX")"
  codex_cloud_mcp_release_manifest "$release" "$actual"
  cmp -s "$release/installed.sha256" "$actual"
  result=$?
  rm -f -- "$actual"
  [[ "$result" == 0 ]] || return "$result"
  actual_links="$(mktemp "$CODEX_CLOUD_TOOLS/.mcp-symlinks.XXXXXX")"
  codex_cloud_mcp_release_symlinks "$release" "$actual_links"
  if ! cmp -s "$release/installed.symlinks" "$actual_links"; then
    rm -f -- "$actual_links"
    return 1
  fi
  rm -f -- "$actual_links"
  while IFS=$'\t' read -r link_path link_target; do
    [[ -n "$link_path" && -n "$link_target" && "$link_target" != /* ]] || return 1
    resolved="$(readlink -f -- "$release/${link_path#./}")" || return 1
    case "$resolved" in "$release"/*) ;; *) return 1;; esac
  done <"$release/installed.symlinks"
}

codex_cloud_mcp_publish_release() {
  local temp="$1" release="$2"
  if mv -T -- "$temp" "$release" 2>/dev/null; then
    return 0
  fi
  rm -rf -- "$temp"
  if codex_cloud_mcp_release_check "$release"; then
    return 0
  fi
  codex_cloud_fail "competing MCP runtime release is absent or corrupt"
  return 1
}

codex_cloud_mcp_runtime_install() {
  local repo_root="$1" target releases release temp pointer lock_hash
  target="$(codex_cloud_mcp_runtime_root)"
  releases="$(codex_cloud_mcp_runtime_releases)"
  lock_hash="$(sha256sum "$repo_root/tools/mcp-runtime/package-lock.json" | awk '{print $1}')"
  release="$releases/$lock_hash"
  pointer="$CODEX_CLOUD_TOOLS/.mcp-runtime.pointer.$$"
  mkdir -p "$CODEX_CLOUD_TOOLS" "$releases" /workspace/.npm-cache
  if [[ -e "$target" && ! -L "$target" ]]; then
    codex_cloud_fail "legacy MCP runtime target is not an atomic symlink: $target"
    return 1
  fi
  if [[ ! -d "$release" ]]; then
    temp="$(mktemp -d "$releases/.install.XXXXXX")"
    cp "$repo_root/tools/mcp-runtime/package.json" "$repo_root/tools/mcp-runtime/package-lock.json" "$temp/"
    if ! (cd "$temp" && npm_config_cache=/workspace/.npm-cache npm ci --ignore-scripts --no-audit --no-fund); then
      rm -rf -- "$temp"; codex_cloud_fail "npm ci failed for the pinned MCP runtime"; return 1
    fi
    for binary in repomix line-bot-mcp-server openapi-mcp-server toolbox; do
      if [[ ! -x "$temp/node_modules/.bin/$binary" ]]; then
        rm -rf -- "$temp"; codex_cloud_fail "MCP runtime binary missing: $binary"; return 1
      fi
    done
    jq -n --arg lock_sha256 "$lock_hash" --arg node "$(node --version)" --arg npm "$(npm --version)" \
      --slurpfile package "$temp/package.json" \
      '{schema_version:1,lock_sha256:$lock_sha256,node_version:$node,npm_version:$npm,dependencies:$package[0].dependencies}' >"$temp/receipt.json"
    codex_cloud_mcp_release_symlinks "$temp" "$temp/installed.symlinks"
    codex_cloud_mcp_release_manifest "$temp" "$temp/installed.sha256"
    if ! codex_cloud_mcp_release_check "$temp"; then
      rm -rf -- "$temp"; codex_cloud_fail "staged MCP runtime manifest verification failed"; return 1
    fi
    find "$temp" -type f -exec chmod a-w {} +
    codex_cloud_mcp_publish_release "$temp" "$release" || return 1
  fi
  codex_cloud_mcp_release_check "$release" || { codex_cloud_fail "immutable MCP runtime release is corrupt"; return 1; }
  ln -s -- "$release" "$pointer"
  if ! mv -Tf -- "$pointer" "$target"; then
    rm -f -- "$pointer"; codex_cloud_fail "could not atomically swap the MCP runtime pointer"; return 1
  fi
}

codex_cloud_mcp_runtime_check() {
  local repo_root="$1" target lock_hash release
  target="$(codex_cloud_mcp_runtime_root)"
  test -L "$target" || { codex_cloud_fail "MCP runtime pointer is not an atomic symlink"; return 1; }
  release="$(readlink -f -- "$target")"
  case "$release" in "$(codex_cloud_mcp_runtime_releases)"/*) ;; *) return 1;; esac
  lock_hash="$(sha256sum "$repo_root/tools/mcp-runtime/package-lock.json" | awk '{print $1}')"
  test "$(basename -- "$release")" = "$lock_hash" || return 1
  codex_cloud_mcp_release_check "$release" || return 1
  test "$(jq -r '.schema_version' "$release/receipt.json")" = 1 || return 1
  test "$(jq -r '.lock_sha256' "$release/receipt.json")" = "$lock_hash" || return 1
  cmp -s "$repo_root/tools/mcp-runtime/package-lock.json" "$release/package-lock.json" || return 1
  while IFS=$'\t' read -r package version; do
    test "$(jq -r '.version' "$release/node_modules/$package/package.json")" = "$version" || return 1
    test "$(jq -r --arg package "$package" '.dependencies[$package]' "$release/receipt.json")" = "$version" || return 1
  done < <(jq -r '.dependencies | to_entries[] | [.key,.value] | @tsv' "$repo_root/tools/mcp-runtime/package.json")
  for binary in repomix line-bot-mcp-server openapi-mcp-server toolbox; do test -x "$release/node_modules/.bin/$binary" || return 1; done
}
