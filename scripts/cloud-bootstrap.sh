#!/usr/bin/env bash
set -Eeuo pipefail

log() { printf '[cloud-bootstrap] %s\n' "$*"; }
need() { command -v "$1" >/dev/null 2>&1 || { log "BLOCKED: missing $1"; return 1; }; }

export PATH="${HOME}/.local/bin:/workspace/.linebot-tools/go/bin:${PATH}"
GOPLS_VERSION="${GOPLS_VERSION:-v0.20.0}"

need go
need git
need codex

if ! command -v gopls >/dev/null 2>&1; then
  [[ "$GOPLS_VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || { log "BLOCKED: invalid GOPLS_VERSION"; exit 1; }
  mkdir -p "${HOME}/.local/bin"
  GOBIN="${HOME}/.local/bin" go install "golang.org/x/tools/gopls@${GOPLS_VERSION}"
fi

export PATH="${HOME}/.local/bin:${PATH}"
need gopls
log "Go: $(go version)"
log "gopls: $(gopls version | head -n 1)"
log "MCP registrations are configuration evidence only:"
codex mcp list || log "DEGRADED: codex mcp list unavailable"

for path in .agents/skills/agent-tool-preflight/SKILL.md .codex/plugins.lock.json; do
  [[ -f "$path" ]] || { log "BLOCKED: missing $path"; exit 1; }
done
log "Bootstrap completed; live MCP connectivity requires separate verification."
