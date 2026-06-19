#!/usr/bin/env bash
# Audit visual: render Svelte + QML, potong per-region (sidebar/conversation)
# biar terbaca jelas, + montase berlabel. Opsi: arg1 = gambar referensi WhatsApp.
#   tools/audit.sh [whatsapp_ref.png]
set -e
cd "$(dirname "$0")/.."
REF="${1:-}"
./tools/snap-svelte.sh /tmp/aud_svelte.png >/dev/null
./tools/snap-qml.sh    /tmp/aud_qml.png   >/dev/null
lbl(){ magick "$2" -resize 460x740 -gravity North -background '#2a2a2a' -splice 0x26 \
        -fill white -pointsize 18 -annotate +0+4 "$1" "$3"; }
# crop sidebar (kiri) + conversation (kanan) tiap render → resize seragam → label
for n in svelte qml; do
  magick /tmp/aud_$n.png -crop 460x740+0+0   +repage /tmp/aud_${n}_side_raw.png
  magick /tmp/aud_$n.png -crop 640x740+460+0 +repage /tmp/aud_${n}_conv_raw.png
  lbl "${n^^} sidebar" /tmp/aud_${n}_side_raw.png /tmp/aud_${n}_side.png
done
# side-by-side sidebar (≈920 lebar → terbaca)
magick /tmp/aud_svelte_side.png /tmp/aud_qml_side.png +append /tmp/audit_sidebar.png
echo "sidebar compare → /tmp/audit_sidebar.png"
if [ -n "$REF" ] && [ -f "$REF" ]; then
  lbl "WHATSAPP ref" "$REF" /tmp/aud_ref_side.png
  magick /tmp/aud_ref_side.png /tmp/aud_svelte_side.png /tmp/aud_qml_side.png +append /tmp/audit_3way.png
  echo "3-way compare → /tmp/audit_3way.png"
fi
