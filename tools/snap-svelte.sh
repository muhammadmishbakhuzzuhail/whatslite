#!/usr/bin/env bash
# Render the Svelte frontend (mock data, LIVE=false) headlessly → reference PNG.
set -e
cd "$(dirname "$0")/../frontend"
OUT="${1:-/tmp/svelte_ref.png}"
THEME="${2:-dark}"   # dark|light|system — diteruskan via ?theme= (stores.js)
node_modules/.bin/vite --port 5173 >/tmp/vite.log 2>&1 & V=$!
trap "kill $V 2>/dev/null || true" EXIT
for _ in $(seq 1 60); do grep -q "Local:" /tmp/vite.log && break; sleep 0.2; done
sleep 1
google-chrome-stable --headless=new --disable-gpu --no-sandbox --hide-scrollbars \
  --screenshot="$OUT" --window-size=1100,740 --virtual-time-budget=6000 "http://localhost:5173/?theme=$THEME" >/dev/null 2>&1
echo "svelte → $OUT"
