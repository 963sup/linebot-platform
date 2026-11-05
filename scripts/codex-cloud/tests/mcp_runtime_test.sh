#!/usr/bin/env bash
set -Eeuo pipefail

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)"
if grep -E 'trap .*RETURN' "$repo_root/scripts/codex-cloud/lib/mcp_runtime.sh"; then
  echo 'RETURN trap can delete the staged runtime when npm subshell returns' >&2
  exit 1
fi
test_root="$(mktemp -d /workspace/mcp-runtime-test.XXXXXX)"
trap 'rm -rf -- "$test_root"' EXIT
fake_bin="$test_root/fake-bin"
mkdir -p "$fake_bin"

cat >"$fake_bin/node" <<'EOF'
#!/usr/bin/env bash
[[ "${1:-}" == --version ]] && { echo v24.0.0; exit 0; }
exit 2
EOF
cat >"$fake_bin/npm" <<'EOF'
#!/usr/bin/env bash
if [[ "${1:-}" == --version ]]; then echo 11.0.0; exit 0; fi
[[ "${1:-}" == ci && " $* " == *" --ignore-scripts "* ]] || exit 3
while IFS=$'\t' read -r package version; do
  mkdir -p "node_modules/$package"
  jq -n --arg version "$version" '{version:$version}' >"node_modules/$package/package.json"
done < <(jq -r '.dependencies | to_entries[] | [.key,.value] | @tsv' package.json)
mkdir -p node_modules/.bin node_modules/.bin-targets
for binary in repomix line-bot-mcp-server openapi-mcp-server toolbox; do
  printf '#!/usr/bin/env bash\nexit 0\n' >"node_modules/.bin-targets/$binary"
  chmod 755 "node_modules/.bin-targets/$binary"
  ln -s "../.bin-targets/$binary" "node_modules/.bin/$binary"
done
EOF
chmod 755 "$fake_bin/node" "$fake_bin/npm"

export CODEX_CLOUD_TOOLS="$test_root/tools"
export CODEX_CLOUD_BIN="$CODEX_CLOUD_TOOLS/bin"
export CODEX_CLOUD_NODE_BIN="$fake_bin"
source "$repo_root/scripts/codex-cloud/lib/common.sh"
source "$repo_root/scripts/codex-cloud/lib/paths.sh"
source "$repo_root/scripts/codex-cloud/lib/mcp_runtime.sh"

codex_cloud_mcp_runtime_install "$repo_root"
codex_cloud_mcp_runtime_check "$repo_root"
test -L "$CODEX_CLOUD_TOOLS/mcp-runtime"
release="$(readlink -f "$CODEX_CLOUD_TOOLS/mcp-runtime")"
# Simulate another installer winning the release-name race. The losing staged
# directory must be removed, never nested inside the immutable release.
race_temp="$(mktemp -d "$CODEX_CLOUD_TOOLS/mcp-runtime-releases/.race.XXXXXX")"
printf 'loser\n' >"$race_temp/marker"
codex_cloud_mcp_publish_release "$race_temp" "$release"
test ! -e "$race_temp"
test ! -e "$release/$(basename "$race_temp")"
codex_cloud_mcp_runtime_install "$repo_root"
codex_cloud_mcp_runtime_check "$repo_root"
test "$(readlink -f "$CODEX_CLOUD_TOOLS/mcp-runtime")" = "$release"

original_repomix_target="$(readlink "$release/node_modules/.bin/repomix")"
ln -sfn /bin/true "$release/node_modules/.bin/repomix"
if codex_cloud_mcp_runtime_check "$repo_root"; then
  echo 'absolute out-of-release symlink target accepted' >&2
  exit 1
fi
ln -sfn "$original_repomix_target" "$release/node_modules/.bin/repomix"
codex_cloud_mcp_runtime_check "$repo_root"

chmod u+w "$release/node_modules/repomix/package.json"
printf '\ntampered\n' >>"$release/node_modules/repomix/package.json"
if codex_cloud_mcp_runtime_check "$repo_root"; then
  echo 'tampered transitive release file accepted' >&2
  exit 1
fi

rm -f "$CODEX_CLOUD_TOOLS/mcp-runtime"
mkdir "$CODEX_CLOUD_TOOLS/mcp-runtime"
if codex_cloud_mcp_runtime_install "$repo_root"; then
  echo 'legacy non-symlink runtime target accepted' >&2
  exit 1
fi
echo 'MCP runtime installer tests passed'
