#!/usr/bin/env bash
# Compatibility entrypoint; canonical workflow lives in bootstrap.sh and mise.toml.
set -Eeuo pipefail
repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
exec bash "$repo_root/bootstrap.sh" setup
