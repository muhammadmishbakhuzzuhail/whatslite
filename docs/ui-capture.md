# ui-capture

`tools/ui-capture.sh` captures the **live, running** WhatsLite (Gio) window's
actual on-screen pixels to a PNG under `docs/ui-shots/` (named
`wlive-<UTC-timestamp>.png`). Unlike `tools/snap-gio.sh`, which renders the UI
headless from mock data, this grabs the real app exactly as it appears on
screen — the input for the analyze → fix loop. It auto-detects a capture
backend, preferring whole-window accuracy: `grim` on Wayland (using
`hyprctl`/`swaymsg` + `jq` to find the window geometry, else the full focused
output), `import` (ImageMagick) on X11/XWayland (window via `xdotool`, else
root), or `gnome-screenshot` as a last resort.

```sh
tools/ui-capture.sh             # single capture; prints the saved PNG path
tools/ui-capture.sh --watch 5   # loop every N seconds (default 5) until Ctrl-C
tools/ui-capture.sh --help      # usage
```
