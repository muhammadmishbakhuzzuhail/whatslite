#!/usr/bin/env bash
# Side-by-side Svelte vs QML render for visual audit.
set -e
magick /tmp/svelte_ref.png /tmp/qml_shot.png +append /tmp/compare_side.png
echo "side-by-side → /tmp/compare_side.png"
