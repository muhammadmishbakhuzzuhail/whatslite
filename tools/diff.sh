#!/usr/bin/env bash
# Objective image diff: DSSIM (perceptual, 0=identical) + RMSE + a red heatmap of
# where the two renders differ. Replaces eyeballing side-by-side montages.
#
#   tools/diff.sh A.png B.png [out-prefix]
#
# Prints:  DSSIM <n>  RMSE <n>   (lower = closer)
# Writes:  <prefix>-heat.png  (B with differences highlighted)
#          <prefix>-sxs.png   (A | B | heatmap, labelled)
set -e
A="$1"; B="$2"; OUT="${3:-/tmp/diff}"
[ -f "$A" ] && [ -f "$B" ] || { echo "usage: diff.sh A.png B.png [out-prefix]"; exit 2; }

# Normalise B to A's geometry so the metric is meaningful even if sizes drift.
W=$(magick identify -format '%w' "$A"); H=$(magick identify -format '%h' "$A")
magick "$B" -resize "${W}x${H}!" /tmp/_diffB.png

# Perceptual + pixel metrics (magick prints metric to stderr; capture it).
DSSIM=$(magick compare -metric DSSIM "$A" /tmp/_diffB.png /tmp/_diffheat_raw.png 2>&1 || true)
RMSE=$(magick compare -metric RMSE  "$A" /tmp/_diffB.png null: 2>&1 | sed -E 's/ .*//' || true)
printf 'DSSIM %-10s RMSE %s\n' "$DSSIM" "$RMSE"

# Heatmap: differing pixels in red over a dimmed B (easy to read where it drifts).
magick compare -metric DSSIM -highlight-color red -lowlight-color '#00000040' \
  "$A" /tmp/_diffB.png "${OUT}-heat.png" 2>/dev/null || cp /tmp/_diffheat_raw.png "${OUT}-heat.png"

lbl(){ magick "$1" -resize 380x -gravity North -background '#222' -splice 0x22 \
        -fill white -pointsize 15 -annotate +0+3 "$2" miff:-; }
{ lbl "$A" "A (reference)"; lbl /tmp/_diffB.png "B (subject)"; lbl "${OUT}-heat.png" "diff heatmap"; } |
  magick - +append "${OUT}-sxs.png"
echo "heatmap → ${OUT}-heat.png   montage → ${OUT}-sxs.png"
