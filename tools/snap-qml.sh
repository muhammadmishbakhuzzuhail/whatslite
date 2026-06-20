#!/usr/bin/env bash
# Render the QML app (mock engine) offscreen → PNG.
#   tools/snap-qml.sh [out.png] [theme] [panel] [lang]
#     theme : dark | light            (default dark)
#     panel : WALITE_OPEN value (sticker|gif|detail|settings|forward|...) or ""
#     lang  : en | id | es            (default en; use id to match the Svelte mock)
# Env passthrough means no more raw inline `env ... walite-qt` invocations.
set -e
cd "$(dirname "$0")/.."
OUT="${1:-/tmp/qml_shot.png}"; THEME="${2:-dark}"; PANEL="${3:-}"; LANG_="${4:-}"
go build -o /tmp/mockengine ./qt-poc/mockengine
cmake --build qt-app/build -j >/dev/null 2>&1
S=/tmp/snap.sock; rm -f "$S" /tmp/walite.png
/tmp/mockengine "$S" >/dev/null 2>&1 & M=$!; sleep 0.4
timeout 20 env QT_QPA_PLATFORM=offscreen QT_QUICK_BACKEND=software \
  WALITE_SELFTEST=1 WALITE_SHOT=1 \
  ${THEME:+$([ "$THEME" = dark ] && echo WALITE_DARK=1)} \
  ${PANEL:+WALITE_OPEN=$PANEL} \
  ${LANG_:+WALITE_LANG=$LANG_} \
  qt-app/build/walite-qt "$S" >/dev/null 2>&1
kill $M 2>/dev/null || true
cp /tmp/walite.png "$OUT"; echo "qml → $OUT (theme=$THEME panel=${PANEL:-none} lang=${LANG_:-en})"
