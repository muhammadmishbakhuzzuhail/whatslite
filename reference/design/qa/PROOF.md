# Proof — doodle wallpaper + UI pass

## What was run

```text
$ cargo test -p whatslite-desktop --quiet
running 21 tests
.....................
test result: ok. 21 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out

$ cargo build -p whatslite-desktop
Finished `dev` profile [unoptimized + debuginfo] ...

$ cargo build -p doodle-preview
Finished `dev` profile [unoptimized + debuginfo] ...
```

## What is implemented

- `desktop/src/wallpaper.rs:26` — `TILE_LOGICAL = 800.0`.
- `desktop/src/theme.rs` — `wallpaper_style(dark)`:
  - light line `#C9B9A3`
  - dark line `#D8CCBC`, layer 0.12
- `desktop/build.rs` — alpha-mask PNGs at 800/1600/2400 (1x/2x/3x).
- `desktop/src/wallpaper.rs` — decodes, tints, caches per (scale, line), prewarms.
- `reference/design/machine/wallpaper-doodle-source.svg` — original dense source; edge-wrapped.
- `crates/doodle-preview` — standalone comparison app with light/dark/reference panels.

## Screenshots (native windows, not canvas renders)

| File | Size | Shows |
|---|---:|---|
| `reference/design/qa/1440-group-light.png` | 1440×900 | App, 800pt repeat, light |
| `reference/design/qa/1440-group-dark.png` | 1440×900 | App, 800pt repeat, dark |
| `reference/design/qa/1920-group-light.png` | 1920×1008 | App, 1920 width |
| `reference/design/qa/1440-header-hover.png` | 1440×900 | Hover chevron |
| `reference/design/qa/1440-header-menu.png` | 1440×900 | Dropdown |
| `reference/design/qa/1440-chat-list-menu.png` | 1440×900 | Chat list menu |
| `reference/design/qa/1440-contact-profile.png` | 1440×900 | Profile overlay |
| `reference/design/qa/1440-group-info.png` | 1440×900 | Group info |
| `reference/design/qa/1440-usecase-demo.png` | 1440×900 | 12-scenario demo |
| `reference/design/qa/1440-loading-skeleton.png` | 1440×900 | List + conversation skeleton |
| `reference/design/qa/1440-error-state.png` | 1440×900 | Error + Retry, composer hidden |
| `reference/design/qa/1440-media-loading.png` | 1440×900 | Media skeleton |
| `reference/design/qa/1440-media-error.png` | 1440×900 | Media error |
| `reference/design/qa/1440-readonly-fields.png` | 1440×900 | Read-only fields without chevrons |
| `reference/design/qa/1440-rtl.png` | 1440×900 | RTL |
| `reference/design/qa/1440-text-125.png` | 1440×900 | 125% text |

## How the screenshots were captured

Native GPUI app on Linux/X11, window client area captured with `XGetImage`
(not the pen canvas, not a mock render):

- app window: 1440×900
- 1920 run: resize client to 1920×1080; X11 frame clamps the captured height
  to 1008 because the screen is 1080 high. Width is 1920.
- preview app: 1500×900, three columns.

## Preview comparison

```bash
DOODLE_REFERENCE=/path/to/whatsapp-reference.jpg \
  cargo run -p doodle-preview
```

It renders light, dark, and the reference side by side. `DOODLE_TILE` and
`DOODLE_LINE` allow comparing 800/700/600/550/400 without touching the app.

The reference image is not committed to the repository; it stays local.

## 3× proof

`cargo test -p whatslite-desktop` includes:

- `wallpaper::tests::picks_the_1x_2x_3x_texture`
- `wallpaper::tests::pre_rendered_tiles_tint_at_1x_2x_3x`

They assert texture edges 800/1600/2400, tint channel order, and visible pixels.
A 3× screenshot cannot be captured in this session because the X11 display runs
at scale 1.

## Canvas proofs

| File | Size | Proves |
|---|---|---|
| `reference/design/qa/38-wallpaper-doodle.png` | 1440×1267 | Pencil page 38: light/dark v19 proofs, 1024 × 1024 SVG spec, 800 pt repeat |
