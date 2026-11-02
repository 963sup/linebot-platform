#!/usr/bin/env bash
# Codex Cloud environment Install script:
#     bash scripts/setup-codex-cloud.sh
# This script installs build prerequisites only; it does NOT deploy or authorize MCPs.
set -Eeuo pipefail
umask 077

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

if ! command -v go >/dev/null 2>&1; then
  echo "Go is missing. Configure Go 1.27.3 in the Codex Cloud environment first." >&2
  exit 1
fi

# Codex Cloud has a separate setup shell: do not rely on exported PATH or tokens
# remaining in the agent phase. The MCP config uses an explicit HOME-based path.
export GOTOOLCHAIN=auto
printf 'Go toolchain: %s\n' "$(go version)"
required_go="$(awk '$1 == "go" { print $2; exit }' go.mod)"
test -n "$required_go" || { echo "go.mod has no Go version" >&2; exit 1; }
printf 'Module requires Go %s\n' "$required_go"

# Keep go.mod/go.sum unchanged: CI verifies the committed lockfiles.
go mod download
go mod verify

# Validate Codex TOML, agents and setup shell syntax with Go (no Python).
go run ./cmd/codexcheck

# gopls MCP is a standalone stdio process. Pin it independently of go.mod.
gopls_version="v0.24.0"
bin_dir="$HOME/.local/bin"
mkdir -p "$bin_dir"
need_install=1
if [ -x "$bin_dir/gopls" ] && "$bin_dir/gopls" version 2>/dev/null | grep -Fq "$gopls_version"; then
  need_install=0
fi
if [ "$need_install" -eq 1 ]; then
  GOBIN="$bin_dir" go install "golang.org/x/tools/gopls@$gopls_version"
fi
"$bin_dir/gopls" version

# An optional warm-up for published environment snapshots; normally avoid
# rebuilding everything during environment preparation.
if [ "${CODEX_CLOUD_WARM_TESTS:-0}" = "1" ]; then
  go test ./...
fi

# Never run codex mcp login, npx toolbox, gcloud, or mutate ~/.codex/config.toml.
echo "Codex Cloud Go dependencies and gopls MCP binary are ready."
