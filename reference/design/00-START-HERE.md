# WhatsLite — handoff package

Read this file first. Everything else is reference.

## What this is
A complete UI/UX design for **WhatsLite**, an independent WhatsApp-Desktop-style app for Windows 11 (with a macOS Liquid Glass shell). The design lives in `design/whatslite.pen` (pen.dev format, JSON v2.22). This folder is the only thing you need to build the frontend.

**Not affiliated with WhatsApp or Meta.** Original glyph mark, original wallpaper, open-source icons and fonts.

**Latest identity pass:** the group avatar is top-aligned (fixed), the wallpaper is a hand-drawn doodle tile, the header shows a hover chevron with a menu beside the title, and clicking a group sender opens that person's profile. See `urutan-judul.md` and the `Interactions / Header Hover & Profile` section in the design.

## Read order (agents)
1. `00-START-HERE.md` ← you are here: rules + index
2. `01-SCREENS.md` — what exists, page by page
3. `02-COMPONENTS.md` — the 54 masters you must instance
4. `03-BEHAVIOUR-RULES.md` — rules that are invisible in the file
5. `04-COMING-SOON.md` — what is designed but not wired
6. `05-BUILD-ORDER.md` — the order to build, with acceptance checks
7. `06-PROMPTS.md` — copy-paste prompts for bootstrap and per-screen work
8. `machine/*.json` — same knowledge as data (read this instead of parsing the .pen)
9. `docs/*.md` — deep specs written during design (state matrix, gap audit, etc.)
10. `urutan-judul.md` — title order, hover menu, profile access, avatar/doodle fixes, clickable demo, backend and number/username rules
11. `brand/` — logo SVG/PNG/ICO + usage rules

The `.pen` references `./wallpaper-doodle-550.png`; keep that PNG next to the `.pen` file. The handoff copy is `machine/wallpaper-doodle.png`. The vector source is `machine/wallpaper-doodle-source.svg`. The canvas uses an 800pt proof; the app rasterises the source at build time and repeats it at 800pt.

## Hard rules
1. **Tokens only.** Every colour comes from `machine/tokens.json` (28 tokens, light+dark). No hex literal in view code. Dark mode is a value swap, never a second UI.
2. **Instance, never copy.** Build from the components in `02-COMPONENTS.md`. If something is missing, add it to the registry first.
3. **Coming soon is a feature.** Anything not implemented shows a toast, modal or placeholder panel with the copy from `04-COMING-SOON.md`. Never a dead button, never fake data.
4. **Empty ≠ failed.** Every list has three states: empty (offers one action), loading (skeleton after 200ms), failed (retry). See `03-BEHAVIOUR-RULES.md` §9.
5. **No invented content.** All strings come from `machine/strings.json` or the copy deck in `docs/`. Sample chats are Indonesian; UI labels are English.
6. **Bubble grouping is mandatory.** See `03-BEHAVIOUR-RULES.md` §1. It is the single rule that makes the product read as a chat app.
7. **Keyboard:** focus ring on every control, logical tab order, `Esc` closes the top layer only.
8. **Assets:** icons = Material Symbols Rounded (Apache 2.0) + Lucide (ISC); type = Inter (OFL 1.1); logo = `messages-square` (Lucide) inside our own tile.

## Folder map
```
design/whatslite.pen        the design (source of truth)
machine/tokens.json         28 tokens, light+dark
machine/tokens.css          same, as CSS variables
machine/theme.json          same, as a palette for GPUI/other
machine/components.json     54 masters with ids, sizes, children
machine/screens.json        38 pages with ids and sections
machine/strings.json        every visible string, grouped by page
machine/wallpaper-doodle.png  dense seamless 800 pattern used by the .pen image fill
brand/                      logo: 3 SVG + 3 PNG + ico + README + export.sh
docs/                       deep specs (state matrix, audits, plans)
qa/                         native app QA captures + reproduce flags
```

## Status
- **38 pages** (00–37), **54 components**, **28 tokens**, **≈13k nodes**
- State matrix: **175 rows** (`docs/WHATSITE-*` and page 07 inside the .pen)
- Known debt: 44 chat-row/search/chip frames are detached copies (masters exist and are restored); relink them when you refactor.

## If you only have 10 minutes
Open `machine/tokens.json`, `01-SCREENS.md`, `02-COMPONENTS.md`, `03-BEHAVIOUR-RULES.md`. That is enough to start the shell, the chat list, and the conversation.
