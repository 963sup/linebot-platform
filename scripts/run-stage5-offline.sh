#!/usr/bin/env bash
# Explicit offline acceptance; never starts a cloud resource, logs in, or deploys.
set -euo pipefail
cd "$(dirname "$0")/.."

host="$(printenv FIRESTORE_EMULATOR_HOST || true)"
case "$host" in
  localhost:*|127.0.0.1:*|'[::1]':*) ;;
  *)
    echo "Set FIRESTORE_EMULATOR_HOST to a LOCAL running emulator (e.g. 127.0.0.1:8085)." >&2
    echo "Refusing to treat skipped integration tests as a pass." >&2
    exit 2
    ;;
esac

go run ./cmd/codexcheck
make check

# The Go SDK uses FIRESTORE_EMULATOR_HOST to route Firestore requests locally.
# Do not forward real credentials to an untrusted test process.
env -u GOOGLE_APPLICATION_CREDENTIALS \
  go test -tags=integration -race -count=1 -timeout=4m ./tests/integration

echo "Offline Stage-5 checks passed against the local emulator."
echo "This is NOT live Google Cloud, OAuth, Codex fresh-session, or production evidence."
