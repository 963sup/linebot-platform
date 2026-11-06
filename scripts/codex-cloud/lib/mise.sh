#!/usr/bin/env bash

codex_cloud_mise_install() {
  local repo_root="$1" version checksum target temp actual
  version="$(jq -r '.mise.version' "$repo_root/config/toolchain.lock")"
  checksum="$(jq -r '.mise.linux_x64_sha256' "$repo_root/config/toolchain.lock")"
  target="$CODEX_CLOUD_BIN/mise"

  if [[ -x "$target" ]] && [[ "$($target --version | awk '{print $1}')" == "$version" ]]; then
    return 0
  fi

  temp="$(mktemp "$CODEX_CLOUD_TOOLS/.mise.XXXXXX")"
  if ! curl --fail --silent --show-error --location \
    "https://github.com/jdx/mise/releases/download/v${version}/mise-v${version}-linux-x64" \
    --output "$temp"; then
    rm -f -- "$temp"
    return 1
  fi
  actual="$(sha256sum "$temp" | awk '{print $1}')"
  if [[ "$actual" != "$checksum" ]]; then
    codex_cloud_fail "mise v${version} checksum mismatch"
    rm -f -- "$temp"
    return 1
  fi
  chmod 0755 "$temp"
  mv -Tf -- "$temp" "$target"
}
