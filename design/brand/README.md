# WhatsLite — brand assets

## Mark
`messages-square` from **Lucide** (ISC). Used unmodified as the product symbol inside our own squircle tile.

## Tile
- Fill `#00AD5A` (token `$wl-accent`) · radius 28% of the tile
- Glyph white, 56% of the tile, optically centred
- No gradient, border or shadow

## Files (only these)
| File | Use |
|---|---|
| `svg/app-icon.svg` | master vector 114×114 — source for everything |
| `svg/lockup-primary.svg` | mark + `WhatsLite`, light backgrounds |
| `svg/lockup-monochrome.svg` | single ink, dark backgrounds |
| `png/app-icon-32.png` | title bar / tray |
| `png/app-icon-128.png` | dev / favicon / about |
| `png/app-icon-512.png` | store, web, docs |
| `app-icon.ico` | Windows packaging (32 + 512) |
| `export.sh` | rebuild the three PNGs + ico from the SVG |

## Rules
1. Never place the mark on the accent without its tile.
2. Clear space: one stroke width (≈6.4px at 114).
3. Minimum size 16px; below 24px the same glyph is used.
4. Do not rotate, outline, recolour or add a shadow.
5. Always start from `svg/app-icon.svg`, never upscale a PNG.

## Licence
Lucide — ISC · Material Symbols Rounded — Apache 2.0 · Inter — OFL 1.1.
WhatsLite is an independent project. Not affiliated with WhatsApp or Meta.
