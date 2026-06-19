#!/usr/bin/env bash
# Render the QML app (mock engine) offscreen → PNG. arg2=dark for dark theme.
set -e
cd "$(dirname "$0")/.."
OUT="${1:-/tmp/qml_shot.png}"; DARK="${2:-}"
go build -o /tmp/mockengine ./qt-poc/mockengine
cmake --build qt-app/build -j >/dev/null 2>&1
S=/tmp/snap.sock; rm -f "$S" /tmp/walite.png
/tmp/mockengine "$S" >/dev/null 2>&1 & M=$!; sleep 0.4
timeout 15 env QT_QPA_PLATFORM=offscreen QT_QUICK_BACKEND=software WALITE_SELFTEST=1 WALITE_SHOT=1 ${DARK:+WALITE_DARK=1} \
  qt-app/build/walite-qt "$S" >/dev/null 2>&1
kill $M 2>/dev/null || true
cp /tmp/walite.png "$OUT"; echo "qml → $OUT"
