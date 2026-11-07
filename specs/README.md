# WhatsApp-inspired macOS — wireframe-first spec v1

**Scope:** Rust + GPUI Kit frontend, entirely mock data with service interfaces; no WhatsApp connectivity promised. 15 editable module SVGs, 15 Mermaid flow diagrams, 120 state records, shared Design System v1. **No high-fidelity assets**: coding agent is free to design polished styling within tokens and layout constraints.

**Display:** 1512×982 logical points at 100%, macOS menu bar 25 pt, content starts y=25; Retina backing pixels may differ. Window maximized, never floating mock-device. **SVGs are monochrome structure guides**, not screenshots or component-perfect production designs. Chat shell navigation 208, chat list 360, conversation flexible. Light and dark are token-driven; wireframes do not need duplicated themes.

## Read order
1. `design-system/design-tokens.json` and `design-system/DESIGN-SYSTEM.md`
2. `wireframes/*.svg` (structural modules)
3. `specifications/screen-map.json`, `feature-inventory.csv`
4. `flows/*.mmd` (coverage graph + key action branches)
5. `specifications/GPUI-IMPLEMENTATION.md`, `specifications/SERVICE-CONTRACTS.md`, `AGENTS.md`

## Caveats
This is WhatsApp-inspired, not endorsed by Meta. Feature inventory includes concepts that may be version/platform-dependent, such as Channels/Communities/Locked Chats. Treat these as **mock screens**, not claims of present feature availability. QR placeholders must not be treated as functional device linking. Do not use unofficial private APIs to promise real WhatsApp sign-in.

## Completion criteria
- 15 SVGs open in browser/Figma-compatible import and all have 1512x982 viewBox.
- 15 .mmd files are Mermaid syntax with IDs from inventory.
- 120 unique screen IDs, exact registry correspondence.
- No per-screen hardcoded theme colors in Rust implementation.
- Keyboard navigation, empty/loading/error, accessibility and snapshot tests included.
