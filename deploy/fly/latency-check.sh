#!/usr/bin/env bash
# Public health-only probe. Runs from the caller's actual network.
# Not a LINE-platform or end-to-end Worker latency benchmark.
set -euo pipefail

if [[ ${1:-} == "" || ! ${1} =~ ^[a-z0-9][a-z0-9-]*[a-z0-9]$ ]]; then
  echo "usage: bash deploy/fly/latency-check.sh WEBHOOK_APP" >&2
  exit 2
fi

url="https://${1}.fly.dev/healthz"
printf 'Measuring %s from this network (5 samples)\n' "$url"
for _ in 1 2 3 4 5; do
  curl --fail --silent --show-error --output /dev/null \
    --connect-timeout 3 --max-time 10 \
    --write-out 'http=%{http_code} dns=%{time_namelookup}s connect=%{time_connect}s tls=%{time_appconnect}s total=%{time_total}s\n' \
    "$url"
done
