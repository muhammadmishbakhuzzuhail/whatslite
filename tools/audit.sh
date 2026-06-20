#!/usr/bin/env bash
# Objective visual audit: render Svelte + QML with the SAME mock scenario and
# language, then compare with NUMBERS — not eyeballing a side-by-side.
#
#   tools/audit.sh [theme]      # dark (default) | light
#
# How it works (and why, after researching cross-engine visual testing):
#  - The QML mock engine mirrors frontend/src/lib/data/mock.js, so both sides
#    show identical chats -> the diff measures rendering fidelity, not content.
#  - QML renders under Xvfb + the real xcb backend (tools/snap-qml.sh), NOT the
#    software backend, which silently drops ShaderEffect/OpacityMask. So the PNG
#    is faithful to a real display.
#  - Comparison uses tools/layout-diff.py (grayscale + blur + coarse block grid),
#    NOT a raw DSSIM/RMSE scalar. Chrome and Qt rasterise text with different
#    anti-aliasing, which pins raw DSSIM at a ~0.18 noise floor on any text and
#    swamps real regressions. The blurred block-diff is font-AA-immune: it scores
#    layout/position/colour drift (lower = closer) and a heatmap localises it.
#  - tools/probe.py measures chat-row pitch in px (catches "off by N px").
#
#  KNOWN LIMITATION: Svelte's conversation pane renders empty under headless
#  Chrome, so only the SIDEBAR is comparable here; conversation parity is
#  verified against the app.css spec instead.
set -e
cd "$(dirname "$0")/.."
THEME="${1:-dark}"
./tools/snap-svelte.sh /tmp/au_sv.png  "$THEME"        >/dev/null
./tools/snap-qml.sh    /tmp/au_qml.png "$THEME" "" id  | sed 's/^/  /'

# Sidebar = left 460px (rail + chat list), the region both engines render.
magick /tmp/au_sv.png  -crop 460x740+0+0 +repage /tmp/au_sv_sb.png
magick /tmp/au_qml.png -crop 460x740+0+0 +repage /tmp/au_qml_sb.png

echo "── layout-diff (font-AA-immune; lower = closer) ──"
python3 tools/layout-diff.py /tmp/au_sv_sb.png /tmp/au_qml_sb.png /tmp/au_heat.png

echo "── row-pitch geometry ──"
python3 tools/probe.py /tmp/au_sv_sb.png  | sed 's/^/  svelte /'
python3 tools/probe.py /tmp/au_qml_sb.png | sed 's/^/  qml    /'

# RGB overlay: R=svelte, G/B=qml -> drift shows as red/cyan fringes.
magick /tmp/au_sv_sb.png  -colorspace Gray /tmp/au_gs.png
magick /tmp/au_qml_sb.png -colorspace Gray /tmp/au_gq.png
magick /tmp/au_gs.png /tmp/au_gq.png /tmp/au_gq.png -combine /tmp/au_overlay.png
echo "── artifacts ──"
echo "  heatmap : /tmp/au_heat.png       overlay : /tmp/au_overlay.png"
echo "            (heatmap red = structural drift; overlay red=svelte-only cyan=qml-only)"
