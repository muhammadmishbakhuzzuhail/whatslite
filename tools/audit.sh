#!/usr/bin/env bash
# Objective visual audit: render Svelte + QML with the SAME mock scenario and
# language, then compare with NUMBERS — not eyeballing a side-by-side.
#
#   tools/audit.sh [theme]      # dark (default) | light
#
# Why the pieces:
#  - The QML mock engine mirrors frontend/src/lib/data/mock.js, so both sides
#    show identical chats → the diff measures rendering fidelity, not content.
#  - DSSIM (tools/diff.sh) gives a perceptual distance + a red heatmap.
#  - The RGB overlay reveals positional drift (red = Svelte-only, cyan = QML-only).
#  - probe.py measures row pitch in px (catches "off by N px" the eye misses).
#
#  KNOWN LIMITATION: Svelte's conversation pane renders empty under headless
#  Chrome (messages don't paint), so only the SIDEBAR is pixel-comparable here.
#  Conversation parity is verified via the app.css spec + probe geometry instead.
set -e
cd "$(dirname "$0")/.."
THEME="${1:-dark}"
./tools/snap-svelte.sh /tmp/au_sv.png  "$THEME"        >/dev/null
./tools/snap-qml.sh    /tmp/au_qml.png "$THEME" "" id  >/dev/null

# Sidebar = left 460px (rail + chat list), the region both engines render.
magick /tmp/au_sv.png  -crop 460x740+0+0 +repage /tmp/au_sv_sb.png
magick /tmp/au_qml.png -crop 460x740+0+0 +repage /tmp/au_qml_sb.png

echo "── sidebar DSSIM (lower = closer) ──"
./tools/diff.sh /tmp/au_sv_sb.png /tmp/au_qml_sb.png /tmp/au_sb | grep -E 'DSSIM|heatmap'

echo "── row-pitch geometry ──"
python3 tools/probe.py /tmp/au_sv_sb.png  | sed 's/^/  svelte /'
python3 tools/probe.py /tmp/au_qml_sb.png | sed 's/^/  qml    /'

# RGB overlay: R=svelte, G/B=qml → drift shows as red/cyan fringes.
magick /tmp/au_sv_sb.png  -colorspace Gray /tmp/au_gs.png
magick /tmp/au_qml_sb.png -colorspace Gray /tmp/au_gq.png
magick /tmp/au_gs.png /tmp/au_gq.png /tmp/au_gq.png -combine /tmp/au_overlay.png
echo "── artifacts ──"
echo "  heatmap : /tmp/au_sb-heat.png    montage : /tmp/au_sb-sxs.png"
echo "  overlay : /tmp/au_overlay.png    (red=svelte-only, cyan=qml-only, grey=match)"
