# whatslite-handoff

Design package for **WhatsLite** — an independent WhatsApp-Desktop-style client.
Drop this folder into the project (e.g. `./design/`) and point the coding agent at `00-START-HERE.md`.

```
00-START-HERE.md        entry point: rules + index + folder map
01-SCREENS.md           38 pages, generated from the .pen
02-COMPONENTS.md        54 masters + instance counts, generated
03-BEHAVIOUR-RULES.md   12 rule groups (grouping, media, states, keyboard, a11y)
04-COMING-SOON.md       feature → status → exact fallback copy
05-BUILD-ORDER.md       11 steps with acceptance checks + DoD
06-PROMPTS.md           copy-paste prompts A–H for the coding agent
urutan-judul.md         title order · header hover menu · group profile · avatar + doodle fixes · clickable demo · backend decision
design/whatslite.pen    the design file (source of truth)
machine/                tokens.json · tokens.css · theme.json · components.json · screens.json · strings.json · wallpaper-doodle.png
brand/                  logo: 3 SVG + 3 PNG + ico + README + export.sh
docs/                   deep specs written during design
qa/                     native app screenshots + reproduction flags
```

Verify: `machine/tokens.json` has 28 tokens, `machine/screens.json` has 38 pages, `machine/components.json` has 54 masters.

The `.pen` references `./wallpaper-doodle-400.png` for the tiled wallpaper. Keep that PNG next to the `.pen` file; `machine/wallpaper-doodle.png` is the handoff copy. The vector source is `desktop/assets/wallpaper/doodle.svg`. The canvas uses an 800pt 2×2 proof; the app repeats the 400pt pattern.

Licences: Lucide (ISC) · Material Symbols Rounded (Apache 2.0) · Inter (OFL 1.1).
WhatsLite is an independent project. Not affiliated with WhatsApp or Meta.
