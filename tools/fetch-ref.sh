#!/usr/bin/env bash
# Unduh gambar referensi (WhatsApp asli dari URL Google/web) → file lokal.
#   tools/fetch-ref.sh <url> [out.png]
# Atau cukup salin screenshot WhatsApp-mu ke /tmp/whatsapp_ref.png
set -e
URL="$1"; OUT="${2:-/tmp/whatsapp_ref.png}"
[ -z "$URL" ] && { echo "usage: fetch-ref.sh <image-url> [out]"; exit 2; }
curl -fsSL -A "Mozilla/5.0" "$URL" -o "$OUT"
file "$OUT" | grep -qiE "image|PNG|JPEG" && echo "ref → $OUT ($(magick identify -format '%wx%h' "$OUT" 2>/dev/null))" || { echo "BUKAN gambar (hotlink-protected?)"; rm -f "$OUT"; exit 1; }
