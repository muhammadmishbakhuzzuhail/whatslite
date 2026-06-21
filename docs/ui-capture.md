# ui-capture

Two ways to screenshot the **running** WhatsLite (Gio) app with **real WhatsApp
data** — the input for the analyze → fix loop. Both write PNGs to
`docs/ui-shots/`.

## 1. In-app capture (recommended — real data, deterministic)

The app itself renders frames to an off-screen headless GL window (same ops as
on-screen, same goroutine — no race, compositor-independent) and saves PNGs.

```sh
# whole live frame every 3s (whatever screen you're on)
WLGIO_SHOTDIR=docs/ui-shots ./cmd/whatslite-gio/run.sh

# tune cadence
WLGIO_SHOTDIR=docs/ui-shots WLGIO_SHOT_EVERY=2 ./cmd/whatslite-gio/run.sh

# capture SPECIFIC named screens with REAL data (drives the UI to each, shoots,
# restores your view) → wlive-<ts>-<screen>.png
WLGIO_SHOTDIR=docs/ui-shots \
  WLGIO_SHOT_SCREENS=settings,chats,calls,contacts,status,channels,forward,attach,reaction,chatctx \
  ./cmd/whatslite-gio/run.sh
```

Screen names: views `chats calls contacts status channels settings`; overlays
`info forward picker attach reaction msgctx chatctx lightbox msginfo`.

## 2. External window grab (real on-screen pixels)

`tools/ui-capture.sh` grabs the actual on-screen window via the OS. Backend is
auto-detected and **verified** (falls through if one fails): on GNOME/Mutter it
uses `gnome-screenshot -w` (grim does not work there); on wlroots (Hyprland/sway)
`grim` with window geometry via `hyprctl`/`swaymsg`+`jq`; on X11 `import` via
`xdotool`.

```sh
tools/ui-capture.sh             # single capture; prints the saved PNG path
tools/ui-capture.sh --watch 5   # loop every N seconds (default 5) until Ctrl-C
tools/ui-capture.sh --help      # usage
```

`tools/snap-gio.sh` / `cmd/gio-shot` render the same screens headless from **mock
data** (no running app needed) — use those for quick layout checks, the above for
real-data fidelity.
