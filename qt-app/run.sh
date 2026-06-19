#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Build + run WhatsLite (Qt6/QML frontend + headless Go engine).
#
#   ./qt-app/run.sh          # build engine + UI, launch both against the real engine
#   ./qt-app/run.sh mock     # run against the mock engine (no WhatsApp session)
set -euo pipefail
cd "$(dirname "$0")/.."

echo "[run] building engine…"
go build -o whatslite-engine ./cmd/whatslite-engine

echo "[run] building Qt UI…"
cmake -B qt-app/build -S qt-app >/dev/null
cmake --build qt-app/build -j >/dev/null

if [[ "${1:-}" == "mock" ]]; then
    SOCK=/tmp/walite-mock.sock
    go build -o /tmp/mockengine ./qt-poc/mockengine
    rm -f "$SOCK"; /tmp/mockengine "$SOCK" &
else
    SOCK="$HOME/.local/share/whatslite/bridge.sock"
    ./whatslite-engine &
fi
ENGINE=$!
trap 'kill $ENGINE 2>/dev/null || true' EXIT

# wait for the bridge socket
for _ in $(seq 1 50); do [[ -S "$SOCK" ]] && break; sleep 0.1; done

echo "[run] launching UI → $SOCK"
qt-app/build/walite-qt "$SOCK"
