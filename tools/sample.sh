#!/usr/bin/env bash
# Sistem referensi desain: ukur warna hex render Svelte di titik kunci → dibanding
# token QML. Render dulu via snap-svelte.sh. Usage: tools/sample.sh [img]
IMG="${1:-/tmp/v_svelte.png}"
P(){ printf "%-26s #%s\n" "$3" "$(magick "$IMG" -format "%[hex:p{$1,$2}]" info:)"; }
echo "ukur dari: $IMG"
P 250 30  "header"; P 250 95 "search"; P 250 450 "sidebar-bg"
P 760 400 "conv-wallpaper"; P 1050 30 "conv-header"
