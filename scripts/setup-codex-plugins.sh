#!/usr/bin/env bash
# Approved pinned skill files only. No OAuth, IAM, runtime deployment, or downloaded scripts.
set -Eeuo pipefail
repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
binary="$(mktemp "${TMPDIR:-/tmp}/codexplugins.XXXXXX")"
trap 'rm -f -- "$binary"' EXIT
go build -o "$binary" ./cmd/codexplugins
"$binary" --install "$@"
