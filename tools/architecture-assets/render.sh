#!/bin/sh
set -eu

source_path=${SOURCE_PATH:-/workspace/docs/assets/architecture/go-ddd-hexagonal-architecture-overview.svg}
output_path=${OUTPUT_PATH:-/out/go-ddd-hexagonal-architecture-overview.png}
width=${RENDER_WIDTH:-2400}

mkdir -p "$(dirname "$output_path")" /tmp/inkscape-profile
export HOME=/tmp/inkscape-profile
inkscape "$source_path" --export-type=png --export-filename="$output_path" --export-width="$width"
