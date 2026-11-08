# whatslite-handoff

Design package for **WhatsLite** — an independent WhatsApp-Desktop-style client.
The handoff lives at `reference/design/`; point the coding agent at `00-START-HERE.md`.

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

The app's wallpaper source is `machine/wallpaper-doodle-source.svg`, an editable 1024 x 1024 seamless tile. `desktop/build.rs` rasterises it with resvg to 800/1600/2400 alpha masks; the runtime tints the mask with the theme and repeats it at an 800pt logical step (light 60%, dark 20%). `machine/wallpaper-doodle.png` is a generated handoff proof, not the build source. Page 38 shows the seamless light/dark crops; the conversation stage repeats the same tile only when the viewport is larger than 800pt.

Licences: Lucide (ISC) · Material Symbols Rounded (Apache 2.0) · Inter (OFL 1.1).
WhatsLite is an independent project. Not affiliated with WhatsApp or Meta.
