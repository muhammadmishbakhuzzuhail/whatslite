#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Jalankan UI WhatsLite-Gio (pure-Go, engine IN-PROCESS — tanpa jembatan IPC).
#
#   ./cmd/whatslite-gio/run.sh         # data WhatsApp asli (sesi headless)
#   ./cmd/whatslite-gio/run.sh demo    # data statis (uji UI tanpa jaringan)
set -euo pipefail
cd "$(dirname "$0")/../.."

# matikan instance lain agar tak rebut DB (engine in-process, satu proses).
# SIGTERM → app shutdown bersih (Disconnect whatsmeow + tutup DB); TUNGGU benar2
# keluar agar tak ada DUA klien tersambung berbarengan (itu merusak sesi Signal →
# "failed to decrypt"). Eskalasi ke -KILL hanya bila ngeyel >5s.
pkill -TERM -f '/whatslite-gio$' 2>/dev/null || true
for _ in $(seq 1 25); do
    pgrep -f '/whatslite-gio$' >/dev/null 2>&1 || break
    sleep 0.2
done
pkill -KILL -f '/whatslite-gio$' 2>/dev/null || true

echo "[gio] building…"
go build -o whatslite-gio ./cmd/whatslite-gio

if [[ "${1:-}" == "demo" ]]; then
    echo "[gio] launching (DEMO data)…"
    WLGIO_DEMO=1 exec ./whatslite-gio
else
    echo "[gio] launching (engine in-process, sesi WA asli)…"
    exec ./whatslite-gio
fi
