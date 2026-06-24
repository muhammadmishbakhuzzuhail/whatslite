#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
#
# memmon.sh — pantau pemakaian memori resident (RSS) proses whatslite-gio
# dari waktu ke waktu, lalu cetak tabel CSV-ish ke stdout.
#
#   tools/memmon.sh [interval_detik] [nama_proses]
#
# Default: interval 2 detik, nama proses "whatslite-gio".
# Berhenti dengan Ctrl-C atau otomatis saat proses hilang.

set -euo pipefail

# --- argumen --------------------------------------------------------------
# $1 = interval sampling (detik), $2 = nama proses yang dicari.
INTERVAL="${1:-2}"
PROC_NAME="${2:-whatslite-gio}"

# --- helper: kB (integer) → MB (satu desimal) -----------------------------
# /proc/<pid>/status memberi nilai dalam kB. Pakai awk supaya tidak butuh bc.
kb_to_mb() {
	# $1 = nilai kB; cetak misal "123.4"
	awk -v kb="${1:-0}" 'BEGIN { printf "%.1f", kb / 1024 }'
}

# --- helper: cari PID terbaru yang cocok ----------------------------------
# pgrep -f mencocokkan seluruh command line. Jika ada beberapa proses,
# ambil yang PID-nya paling besar (umumnya yang paling baru dijalankan).
find_pid() {
	# Kembalikan PID lewat stdout, atau string kosong bila tak ada.
	pgrep -f -- "$PROC_NAME" 2>/dev/null | sort -n | tail -n1 || true
}

# --- tunggu sampai proses muncul ------------------------------------------
PID=""
while :; do
	PID="$(find_pid)"
	if [ -n "$PID" ]; then
		break
	fi
	echo "menunggu proses ${PROC_NAME}…"
	sleep "$INTERVAL"
done

echo "memantau ${PROC_NAME} (pid=${PID}), interval ${INTERVAL}s — tekan Ctrl-C untuk berhenti"

# --- header tabel (dicetak sekali) ----------------------------------------
printf '%-9s  %-8s  %-8s  %s\n' "elapsed_s" "RSS_MB" "VSZ_MB" "peak"

# --- loop sampling ---------------------------------------------------------
START="$(date +%s)"   # detak awal; elapsed = sekarang - START (plain bash)
PEAK_KB=0             # puncak RSS dalam kB, dipertahankan selama proses jalan

while :; do
	STATUS="/proc/${PID}/status"

	# Proses hilang? Cetak puncak terakhir lalu keluar dengan bersih.
	if [ ! -r "$STATUS" ]; then
		echo "proses ${PROC_NAME} (pid=${PID}) hilang — puncak RSS = $(kb_to_mb "$PEAK_KB") MB"
		exit 0
	fi

	# Baca VmRSS (resident) dan VmSize (virtual) dari /proc/<pid>/status.
	# Format baris: "VmRSS:\t  123456 kB" — ambil kolom angka (field $2).
	RSS_KB="$(awk '/^VmRSS:/  { print $2 }' "$STATUS" 2>/dev/null || true)"
	VSZ_KB="$(awk '/^VmSize:/ { print $2 }' "$STATUS" 2>/dev/null || true)"

	# Jika gagal baca (proses baru saja mati saat dibaca), perlakukan sebagai hilang.
	if [ -z "${RSS_KB:-}" ]; then
		echo "proses ${PROC_NAME} (pid=${PID}) hilang — puncak RSS = $(kb_to_mb "$PEAK_KB") MB"
		exit 0
	fi

	# Perbarui puncak RSS bila sampel ini lebih tinggi.
	if [ "$RSS_KB" -gt "$PEAK_KB" ]; then
		PEAK_KB="$RSS_KB"
	fi

	NOW="$(date +%s)"
	ELAPSED=$(( NOW - START ))

	# Satu baris per sampel: elapsed, RSS, VSZ, dan puncak berjalan.
	printf '%-9s  %-8s  %-8s  peak=%s\n' \
		"$ELAPSED" \
		"$(kb_to_mb "$RSS_KB")" \
		"$(kb_to_mb "$VSZ_KB")" \
		"$(kb_to_mb "$PEAK_KB")"

	sleep "$INTERVAL"
done
