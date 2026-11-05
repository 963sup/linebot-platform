#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
source "$repo_root/tools/architecture-assets/render.env"
mode=${1:---check}
case "$mode" in --check|--write) ;; *) printf 'usage: %s [--check|--write]\n' "$0" >&2; exit 2 ;; esac

artifact_dir=${ARCHITECTURE_RENDER_ARTIFACT_DIR:-}
if [[ -n "$artifact_dir" ]]; then
  mkdir -p "$artifact_dir"
  docker build \
    --build-arg "BASE_IMAGE=$BASE_IMAGE" \
    --build-arg "INKSCAPE_PACKAGE=$INKSCAPE_PACKAGE" \
    --build-arg "NOTO_CJK_PACKAGE=$NOTO_CJK_PACKAGE" \
    --tag "$RENDER_IMAGE" \
    "$repo_root/tools/architecture-assets" 2>&1 | tee "$artifact_dir/docker-build.log"
else
  docker build \
    --build-arg "BASE_IMAGE=$BASE_IMAGE" \
    --build-arg "INKSCAPE_PACKAGE=$INKSCAPE_PACKAGE" \
    --build-arg "NOTO_CJK_PACKAGE=$NOTO_CJK_PACKAGE" \
    --tag "$RENDER_IMAGE" \
    "$repo_root/tools/architecture-assets"
fi

render_dir=$(mktemp -d)
trap 'rm -rf "$render_dir"' EXIT
docker run --rm \
  --network none \
  --read-only \
  --tmpfs /tmp:rw,noexec,nosuid,size=64m \
  --volume "$repo_root:/workspace:ro" \
  --volume "$render_dir:/out:rw" \
  --env "SOURCE_PATH=/workspace/$SOURCE" \
  --env "OUTPUT_PATH=/out/overview.png" \
  --env "RENDER_WIDTH=$WIDTH" \
  "$RENDER_IMAGE"

if [[ "$mode" == --write ]]; then
  cp "$render_dir/overview.png" "$repo_root/$OUTPUT"
  exit 0
fi
cmp "$render_dir/overview.png" "$repo_root/$OUTPUT" || {
  if [[ -n "$artifact_dir" ]]; then
    cp "$render_dir/overview.png" "$artifact_dir/generated-overview.png"
    cp "$repo_root/$OUTPUT" "$artifact_dir/canonical-overview.png"
    cmp "$render_dir/overview.png" "$repo_root/$OUTPUT" \
      >"$artifact_dir/first-byte-difference.txt" 2>&1 || true
    sha256sum "$render_dir/overview.png" "$repo_root/$OUTPUT" \
      >"$artifact_dir/sha256sums.txt"
  fi
  printf 'architecture overview PNG is stale; run scripts/render-architecture-assets.sh --write\n' >&2
  exit 1
}
