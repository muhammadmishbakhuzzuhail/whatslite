#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
#
# ui-capture.sh — capture the LIVE running WhatsLite (Gio) window's actual
# on-screen pixels to PNG, for UI/UX analysis. Unlike tools/snap-gio.sh (which
# renders headless from mock data), this grabs the REAL running app as the user
# sees it — the input for the analyze -> fix loop: shoot, eyeball/diff, fix,
# reshoot.
#
# Backends are auto-detected, preferring whole-window accuracy:
#   1. grim     (Wayland/wlroots) — window geometry via hyprctl/swaymsg+jq,
#                else full focused output.
#   2. import   (ImageMagick, X11/XWayland) — window via xdotool, else root.
#   3. gnome-screenshot — window (-w), else fullscreen (-f).
#
# Usage:
#   tools/ui-capture.sh            # single capture; prints saved PNG path
#   tools/ui-capture.sh --watch [N] # loop every N seconds (default 5) until Ctrl-C
#   tools/ui-capture.sh --help     # this help
set -euo pipefail

APP_TITLE="WhatsLite"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR="$REPO_ROOT/docs/ui-shots"

usage() {
	cat <<'EOF'
ui-capture.sh — capture the LIVE running WhatsLite window to PNG.

Usage:
  tools/ui-capture.sh             Single capture; prints the saved PNG path.
  tools/ui-capture.sh --watch [N] Loop every N seconds (default 5) until Ctrl-C.
  tools/ui-capture.sh --help      Show this help.

Output: docs/ui-shots/wlive-<UTC-timestamp>.png
Backends (auto-detected): grim (Wayland) > import (X11) > gnome-screenshot.
EOF
}

have() { command -v "$1" >/dev/null 2>&1; }
ok() { [ -s "$1" ]; } # capture succeeded only if file exists and is non-empty

# Echo "X,Y WxH" for the WhatsLite window on a wlroots compositor, or nothing.
wl_window_geom() {
	have jq || return 0
	if have hyprctl; then
		hyprctl clients -j 2>/dev/null | jq -r --arg t "$APP_TITLE" '
			.[] | select((.title // "" | test($t; "i")) or (.class // "" | test($t; "i")))
			| "\(.at[0]),\(.at[1]) \(.size[0])x\(.size[1])"' 2>/dev/null | head -1
		return 0
	fi
	if have swaymsg; then
		swaymsg -t get_tree 2>/dev/null | jq -r --arg t "$APP_TITLE" '
			.. | objects | select(.pid? and .rect?)
			| select(((.name // "") | test($t; "i")) or ((.app_id // "") | test($t; "i")) or ((.window_properties?.class // "") | test($t; "i")))
			| "\(.rect.x),\(.rect.y) \(.rect.width)x\(.rect.height)"' 2>/dev/null | head -1
		return 0
	fi
}

# Each backend RETURNS NON-ZERO on failure so capture_one falls through. grim
# aborts on GNOME/Mutter ("compositor doesn't support screen capture"), so we must
# verify the file was actually written rather than trusting exit status alone.
try_gnome() {
	have gnome-screenshot || return 1
	gnome-screenshot -w -f "$1" 2>/dev/null && ok "$1" && return 0 # focused window
	gnome-screenshot -f "$1" 2>/dev/null && ok "$1" && return 0     # whole screen
	return 1
}
try_grim() { # wlroots (Hyprland/sway); window geometry if hyprctl/swaymsg present
	have grim || return 1
	local geom
	geom="$(wl_window_geom || true)"
	if [ -n "${geom:-}" ]; then
		grim -g "$geom" "$1" 2>/dev/null && ok "$1" && return 0
	fi
	grim "$1" 2>/dev/null && ok "$1" && return 0
	return 1
}
try_import() { # X11/XWayland; window via xdotool, else root
	have import || return 1
	local wid=""
	have xdotool && wid="$(xdotool search --name "$APP_TITLE" 2>/dev/null | head -1 || true)"
	if [ -n "$wid" ]; then
		import -window "$wid" "$1" 2>/dev/null && ok "$1" && return 0
	fi
	import -window root "$1" 2>/dev/null && ok "$1" && return 0
	return 1
}

capture_one() {
	local out="$1"
	# Order by what actually targets a WINDOW on each environment. On GNOME/Mutter
	# grim is non-functional, so prefer gnome-screenshot there; on wlroots prefer
	# grim. Every backend falls through on failure (file-existence checked).
	case "${XDG_CURRENT_DESKTOP:-}" in
	*[Gg][Nn][Oo][Mm][Ee]*) try_gnome "$out" && return 0 ;;
	esac
	try_grim "$out" && return 0
	try_import "$out" && return 0
	try_gnome "$out" && return 0
	echo "ui-capture.sh: no working capture backend (tried gnome-screenshot, grim, import)" >&2
	echo "  Install one, or use the in-app live capture: WLGIO_SHOTDIR=docs/ui-shots ./cmd/whatslite-gio/run.sh" >&2
	return 1
}

shoot() {
	mkdir -p "$OUT_DIR"
	local ts out
	ts="$(date -u +%Y%m%d-%H%M%S)"
	out="$OUT_DIR/wlive-$ts.png"
	capture_one "$out"
	echo "$out"
}

main() {
	local mode="single" interval=5
	case "${1:-}" in
		--help | -h)
			usage
			exit 0
			;;
		--watch)
			mode="watch"
			if [ "${2:-}" != "" ]; then interval="$2"; fi
			;;
		"")
			mode="single"
			;;
		*)
			echo "ui-capture.sh: unknown argument '$1'" >&2
			usage >&2
			exit 2
			;;
	esac

	if [ "$mode" = "watch" ]; then
		echo "Watching: capturing every ${interval}s into $OUT_DIR (Ctrl-C to stop)" >&2
		while true; do
			shoot
			sleep "$interval"
		done
	else
		shoot
	fi
}

main "$@"
