#!/usr/bin/env bash
# Render the QML app (mock engine) to a PNG for visual auditing.
#   tools/snap-qml.sh [out.png] [theme] [panel] [lang]
#     theme : dark | light            (default dark)
#     panel : WALITE_OPEN value (sticker|gif|detail|settings|forward|...) or ""
#     lang  : en | id | es            (default en; use id to match the Svelte mock)
#
# Renders under Xvfb with the real xcb/RHI backend, NOT QT_QUICK_BACKEND=software.
# WHY: the software backend silently DROPS ShaderEffect-based items — that means
# Qt5Compat OpacityMask/DropShadow (the Avatar photo mask, any shadows) render as
# no-ops, so a software-backend PNG is NOT faithful to a real display. Xvfb + xcb
# (Mesa llvmpipe) exercises the real GPU code path so effects actually render.
# Falls back to offscreen+software if Xvfb/xcb is unavailable (less faithful).
set -e
cd "$(dirname "$0")/.."
OUT="${1:-/tmp/qml_shot.png}"; THEME="${2:-dark}"; PANEL="${3:-}"; LANG_="${4:-}"
go build -o /tmp/mockengine ./qt-poc/mockengine
cmake --build qt-app/build -j >/dev/null 2>&1
S=/tmp/snap.sock; rm -f "$S" /tmp/walite.png
/tmp/mockengine "$S" >/dev/null 2>&1 & M=$!; sleep 0.4
trap 'kill $M 2>/dev/null || true' EXIT

COMMON_ENV=(WALITE_SELFTEST=1 WALITE_SHOT=1)
[ "$THEME" = dark ] && COMMON_ENV+=(WALITE_DARK=1)
[ -n "$PANEL" ] && COMMON_ENV+=("WALITE_OPEN=$PANEL")
[ -n "$LANG_" ] && COMMON_ENV+=("WALITE_LANG=$LANG_")

if command -v xvfb-run >/dev/null && [ -f /usr/lib/qt6/plugins/platforms/libqxcb.so ]; then
  BACKEND="xcb (faithful)"
  xvfb-run -a -s "-screen 0 1100x740x24" \
    env QT_QPA_PLATFORM=xcb "${COMMON_ENV[@]}" \
    timeout 25 qt-app/build/walite-qt "$S" >/dev/null 2>&1 || true
else
  BACKEND="offscreen+software (effects dropped!)"
  env QT_QPA_PLATFORM=offscreen QT_QUICK_BACKEND=software "${COMMON_ENV[@]}" \
    timeout 20 qt-app/build/walite-qt "$S" >/dev/null 2>&1 || true
fi
kill $M 2>/dev/null || true
[ -f /tmp/walite.png ] || { echo "render FAILED (no /tmp/walite.png)"; exit 1; }
cp /tmp/walite.png "$OUT"
echo "qml → $OUT  [$BACKEND theme=$THEME panel=${PANEL:-none} lang=${LANG_:-en}]"
