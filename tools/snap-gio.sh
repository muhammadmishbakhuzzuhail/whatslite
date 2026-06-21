#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Render UI Gio (internal/gioui, data demo) ke PNG headless untuk audit paritas
# vs Svelte. Pakai EGL surfaceless + Mesa software → tak butuh display.
#
#   tools/snap-gio.sh [out.png] [w] [h]
set -e
cd "$(dirname "$0")/.."
OUT="${1:-/tmp/gio_shot.png}"; W="${2:-1000}"; H="${3:-680}"
LIBGL_ALWAYS_SOFTWARE=1 EGL_PLATFORM=surfaceless go run ./cmd/gio-shot "$OUT" "$W" "$H"
