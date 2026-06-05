#!/usr/bin/env bash
# Autopilot screenshot: data NYATA + UI nyata, tanpa display server.
#   1. build frontend
#   2. export data nyata (offline) -> dist/real-data.json
#   3. Chrome headless file:// ?data=real -> capture light + dark
#
# Pakai: scripts/snap.sh        (build penuh)
#        scripts/snap.sh --fast (lewati npm build; pakai dist yang ada)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/build/bin/whatsapp-lite"
DIST="$ROOT/frontend/dist"
OUT="${SNAP_OUT:-/tmp}"

[ -x "$BIN" ] || { echo "binary tak ada: $BIN (jalankan: wails build -tags \"webkit2_41 netgo\")"; exit 1; }

if [ "${1:-}" != "--fast" ]; then
  ( cd "$ROOT/frontend" && npm run build >/dev/null )
fi

"$BIN" --export-json "$DIST/real-data.json"

rm -rf /tmp/cr-snap
for th in light dark; do
  timeout 30 google-chrome-stable \
    --allow-file-access-from-files --disable-web-security \
    --headless=new --disable-gpu --no-sandbox \
    --user-data-dir=/tmp/cr-snap --window-size=1280,860 --virtual-time-budget=6000 \
    --screenshot="$OUT/wa-real-$th.png" \
    "file://$DIST/index.html?data=real&theme=$th" >/dev/null 2>&1 || true
  echo "  $OUT/wa-real-$th.png"
done
echo "done."
