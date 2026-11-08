# Wallpaper generation

The conversation wallpaper is built from two MIT-licensed icon sets:

- **Sketchy Icon Library** — https://github.com/Downshift/sketchy-icons
- **Oreo Doodle Icons** — https://github.com/oreo-design/doodle-icons

Clone them locally, then run the generator:

```bash
git clone --depth 1 https://github.com/Downshift/sketchy-icons /tmp/sketchy-icons
git clone --depth 1 https://github.com/oreo-design/doodle-icons /tmp/oreo-icons
python3 reference/design/wallpaper/build_gapfilled.py
```

Output:

- `reference/design/machine/wallpaper-doodle-source.svg` — the 1024 x 1024 seamless master.
  Current composition: 300 objects, 846 small icons, 520 dots.
- The script also renders the light/dark page-38 panels into the local `.pencil`
  document folder; adjust the `PEN` constant at the top of the script if your
  document lives elsewhere.

The app does **not** parse this SVG at runtime. `desktop/build.rs` uses resvg to
rasterise it to 800/1600/2400 alpha masks at build time; `desktop/src/wallpaper.rs`
tints the mask and repeats it at an **800 pt** logical step. Light ink is
`#C9B9A3` at 60%, dark ink is `#D8CCBC` at 20%.
