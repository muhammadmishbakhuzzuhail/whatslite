#!/usr/bin/env bash
# Pasang WhatsLite untuk user saat ini (tanpa root).
# Salin binary → ~/.local/bin, .desktop → applications, icon → hicolor.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/build/bin/whatslite"

if [[ ! -x "$BIN" ]]; then
  echo "Binary belum ada. Build dulu:" >&2
  echo "  go build -o build/bin/whatslite ./cmd/whatslite-gio" >&2
  exit 1
fi

BINDIR="$HOME/.local/bin"
APPDIR="$HOME/.local/share/applications"
ICONDIR="$HOME/.local/share/icons/hicolor/scalable/apps"

mkdir -p "$BINDIR" "$APPDIR" "$ICONDIR"
install -m755 "$BIN" "$BINDIR/whatslite"
install -m644 "$ROOT/build/linux/whatslite.svg" "$ICONDIR/whatslite.svg"
install -m644 "$ROOT/build/linux/whatslite.desktop" "$APPDIR/whatslite.desktop"

command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database "$APPDIR" || true
command -v gtk-update-icon-cache >/dev/null 2>&1 && gtk-update-icon-cache -f -t "$HOME/.local/share/icons/hicolor" || true

echo "Terpasang. Jalankan via menu aplikasi atau: whatslite"
echo "(pastikan ~/.local/bin ada di PATH)"
