#!/usr/bin/env bash
# Builds a temporary local checker; does not install skills, authenticate, or call remote APIs.
set -Eeuo pipefail
if [ "$#" -ne 0 ]; then
  echo "Check mode accepts no arguments; use setup-codex-plugins.sh for installation." >&2
  exit 64
fi
repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
binary="$(mktemp "${TMPDIR:-/tmp}/codexplugins.XXXXXX")"
trap 'rm -f -- "$binary"' EXIT
go build -o "$binary" ./cmd/codexplugins
"$binary"
