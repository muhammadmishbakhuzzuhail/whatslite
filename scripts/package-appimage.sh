#!/usr/bin/env bash
# Bangun AppImage portabel dari binary Gio (go build ./cmd/whatslite-gio).
# Butuh: linuxdeploy + appimagetool (otomatis diunduh ke ./build/tools bila tak ada).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/build/bin/whatslite"
TOOLS="$ROOT/build/tools"
APPDIR="$ROOT/build/AppDir"

if [[ ! -x "$BIN" ]]; then
  echo "Binary belum ada. Build dulu: go build -o build/bin/whatslite ./cmd/whatslite-gio" >&2
  exit 1
fi

mkdir -p "$TOOLS"
fetch() { # url dest
  if [[ ! -x "$2" ]]; then
    echo "Unduh $(basename "$2")…"
    curl -sSL -o "$2" "$1" && chmod +x "$2"
  fi
}
ARCH="$(uname -m)"
fetch "https://github.com/linuxdeploy/linuxdeploy/releases/download/continuous/linuxdeploy-${ARCH}.AppImage" "$TOOLS/linuxdeploy"
fetch "https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-${ARCH}.AppImage" "$TOOLS/appimagetool"

rm -rf "$APPDIR"
mkdir -p "$APPDIR/usr/bin"
install -m755 "$BIN" "$APPDIR/usr/bin/whatslite"

export APPIMAGE_EXTRACT_AND_RUN=1
"$TOOLS/linuxdeploy" --appdir "$APPDIR" \
  -d "$ROOT/build/linux/whatslite.desktop" \
  -i "$ROOT/build/linux/whatslite.svg"

ARCH="$ARCH" "$TOOLS/appimagetool" "$APPDIR" "$ROOT/build/bin/WhatsLite-${ARCH}.AppImage"
echo "Selesai: build/bin/WhatsLite-${ARCH}.AppImage"
