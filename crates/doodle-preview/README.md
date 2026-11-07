# Doodle Preview

A tiny, app-free playground for the chat doodle wallpaper. It renders three
columns: the light tile, the dark tile, and an optional reference image. Use it
to compare composition and density before touching the main WhatsLite app.

```bash
# no reference: light + dark panels
cargo run -p doodle-preview

# compare against a reference wallpaper
DOODLE_REFERENCE=/tmp/wa-doodle-ref/471f864a.jpg cargo run -p doodle-preview

# try another repeat / line colour without editing code
DOODLE_TILE=800 DOODLE_LINE=C9B9A3 cargo run -p doodle-preview
```

Environment:

| Variable | Default | Meaning |
|---|---|---|
| `DOODLE_REFERENCE` | unset | Absolute path to a local reference image. Never bundled. |
| `DOODLE_TILE` | `800` | Logical repeat size in points. |
| `DOODLE_LINE` | `C9B9A3` | Light-theme line colour. |
| `DOODLE_LIGHT_OPACITY` | `0.60` | Light wallpaper layer opacity. |
| `DOODLE_DARK_OPACITY` | `0.20` | Dark wallpaper layer opacity. |

The artwork lives in `../../design/machine/wallpaper-doodle-source.svg`;
`build.rs` rasterises it at 800/1600/2400 for the preview. The main app uses the
same SVG source, rasterises its own 800/1600/2400 alpha masks at build time, and
repeats the tile at 800 pt. Light ink is `#C9B9A3` at 60%; dark ink is `#D8CCBC`
at 20%.

Why a separate app: it compiles and links without the WhatsLite app surface, so
the design loop is just this small crate instead of the full desktop binary.
