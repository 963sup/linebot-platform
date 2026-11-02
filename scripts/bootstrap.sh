#!/usr/bin/env bash
set -euo pipefail

command -v go >/dev/null || { echo "Go is required" >&2; exit 1; }
echo "Using $(go version)"
go mod tidy
go mod download
echo "Dependencies downloaded. Review changes to go.mod and go.sum before committing."
