#!/usr/bin/env bash
# Regenerate the only raster assets we ship, from the SVG masters.
set -e
cd "$(dirname "$0")"
mkdir -p png
for s in 32 128 512; do
  magick -background none svg/app-icon.svg -resize ${s}x${s} png/app-icon-${s}.png
done
magick png/app-icon-32.png png/app-icon-512.png app-icon.ico
echo "brand assets rebuilt: 32, 128, 512 + ico"
