# WhatsLite Design Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the remaining design-source and in-app Design Reference gaps so the 39-page handoff is internally consistent and fully mirrored in the desktop app.

**Architecture:** The Pencil handoff stays the visual source of truth. `reference/design/machine/*.json` stays the machine-readable source. The desktop app reads `theme.json` at compile time and renders a compact in-app mirror. A JSON consistency test pins page/component/token counts so future edits cannot silently drift.

**Tech Stack:** Rust, GPUI Kit 0.8, `serde_json`, Pencil MCP, Markdown/JSON handoff files.

**Spec:** `reference/design/00-START-HERE.md`, `reference/design/04-COMING-SOON.md`, `PRODUCT.md`, and page 38 in `reference/design/design/whatslite.pen`.

## Global Constraints

- The desktop app is frontend-only: deterministic fixtures and typed service contracts; no live WhatsApp connection (`PRODUCT.md`).
- Design counts are fixed: **39 pages (00–38)**, **54 reusable components**, **28 color tokens**.
- Wallpaper contract: 1024 × 1024 seamless SVG tile, **800 pt** repeat, light `#C9B9A3` at **60%**, dark `#D8CCBC` at **20%**.
- UI colors come only from `reference/design/machine/theme.json`; no new hex literals in view code.
- `DESIGN.md` and `PRODUCT.md` must not duplicate or contradict the machine-readable source.
- No new runtime dependencies; use the existing `serde_json` dev dependency.

## Review Focus

- A duplicated design token drifts from `theme.json` while both documents still look authoritative.
- A documentation link points at a file that does not exist, so a new contributor cannot build the design reference.
- Page/component/token counts drift after a Pencil edit and the in-app reference silently disagrees.
- The new foundation rows overflow or clip at 1024 px width and 125% text scale.
- The page 38 proof file exists but is not discoverable from QA documentation.

---

### Task 1: Normalize design documentation and references

**Files:**
- Modify: `DESIGN.md`
- Modify: `PRODUCT.md`
- Modify: `reference/design/machine/strings.json`
- Modify: `reference/design/docs/WHATSLITE-DEEP-AUDIT.md`
- Modify: `reference/design/qa/PROOF.md`

**Interfaces:**
- Consumes: `reference/design/machine/theme.json`, `tokens.json`, `screens.json`, `components.json`.
- Produces: one source-of-truth statement, valid in-repo references, and the page 38 QA row.

- [ ] **Step 1: Replace duplicated color values in `DESIGN.md`**

Replace the YAML `colors:` frontmatter block and the “Light Theme” / “Dark Theme” bullet lists with one statement:

```markdown
## Colors
The 28 light/dark color tokens live in `reference/design/machine/theme.json`
and are mirrored in `reference/design/machine/tokens.json`. Do not duplicate
hex values here; update the machine-readable source and the Pencil page 01
foundation together.
```

Keep the typography, radius, spacing, layout, and elevation sections unchanged.

- [ ] **Step 2: Fix missing references in `PRODUCT.md`**

Replace `specifications/screen-map.json` with `reference/design/machine/screens.json`, and `design-system/design-tokens.json` with `reference/design/machine/tokens.json`. Keep the sentence’s meaning.

- [ ] **Step 3: Fix the remaining stale wallpaper string**

In `reference/design/machine/strings.json`, replace:

```text
Tile the conversation stage at 800px offsets, clipped by the window
```

with:

```text
Tile the conversation stage at 800 pt offsets, clipped by the window
```

- [ ] **Step 4: Add a page 38 addendum to the historical audit**

Append to `reference/design/docs/WHATSLITE-DEEP-AUDIT.md`:

```markdown
## Addendum · 2026-11-11

Page **38 · Wallpaper Doodle** is now part of the handoff: the final
1024 × 1024 seamless SVG tile, light/dark v19 proofs, and the 800 pt /
60%–20% contract. The handoff is **39 pages (00–38)**; the historical
audit above remains a record of the 38-page pass.
```

- [ ] **Step 5: Add the page 38 proof row to `reference/design/qa/PROOF.md`**

Append after the native-window screenshot table:

```markdown
### Canvas proofs

| File | Size | Proves |
|---|---|---|
| `reference/design/qa/38-wallpaper-doodle.png` | 1440×1267 | Pencil page 38: light/dark v19 proofs, 1024 × 1024 SVG spec, 800 pt repeat |
```

- [ ] **Step 6: Verify**

Run:

```bash
python3 -c "import json; json.load(open('reference/design/machine/strings.json'))"
rg -n "specifications/screen-map.json|design-system/design-tokens.json|800px offsets" DESIGN.md PRODUCT.md reference/design/machine/strings.json
test -f reference/design/qa/38-wallpaper-doodle.png
```

Expected: JSON parses, `rg` prints no matches, and `test` exits 0.

- [ ] **Step 7: Commit**

```bash
git add DESIGN.md PRODUCT.md reference/design/machine/strings.json \
  reference/design/docs/WHATSLITE-DEEP-AUDIT.md reference/design/qa/PROOF.md
git commit -m "docs(design): normalize handoff references"
```

### Task 2: Mirror elevation, iconography, and logo in the in-app Design Reference

**Files:**
- Modify: `desktop/src/app.rs` (`SettingsSubpage::DesignReference`)

**Interfaces:**
- Consumes: `Palette`, `ui::section_label`, `ui::settings_row_field`, `ui::chip`, `IconName`.
- Produces: three static foundation sections between `FOUNDATIONS · SPACING & RADIUS` and `FOUNDATIONS · MOTION`.

- [ ] **Step 1: Insert the elevation section**

After the spacing/radius chips and before `ui::section_label("FOUNDATIONS · MOTION", p)`, add rows with these exact values:

```text
Borders & elevation / hairline: 1px inner #E9E9E9
Focus ring: 2px #00AD5A, 2px offset, keyboard only
Bubble: 0 1 2 / 8%
Popover: 0 8 24 / 12%
Glass: macOS shell; translucent fill, 1px white 50% edge
```

Use `ui::settings_row_field(label, detail, value, p)` for each row. Keep the section label `FOUNDATIONS · BORDERS & ELEVATION`.

- [ ] **Step 2: Insert the iconography section**

Add a `FOUNDATIONS · ICONOGRAPHY` section with:

```text
Material Symbols Rounded · outlined default, filled selected
16 px · inline meta
18 px · row actions
20 px · navigation
24 px · headers and composer
32 px · empty states
```

Render the five sizes as `ui::chip` elements in a wrapping row; keep the first line as a `settings_row_field`.

- [ ] **Step 3: Insert the logo section**

Add a `FOUNDATIONS · LOGO` section with:

```text
messages-square · Lucide · ISC
App icon: squircle radius 28% of tile; clear space one stroke width
Primary lockup: mark + Inter 700 wordmark; gap = half mark width
Monochrome: one ink color; never on the accent
Minimum sizes: 64 / 40 / 28 / 20 / 16 px (app tile only at 16)
```

Use one `settings_row_field` per line; keep the chip row for the five minimum sizes if horizontal space allows, otherwise wrap.

- [ ] **Step 4: Verify layout and tests**

```bash
cargo fmt --all
cargo test -p whatslite-desktop
```

Expected: 21 passed; no layout compile errors. Manually check the Design Reference page at 1024 width and 125% text scale if a display is available.

- [ ] **Step 5: Commit**

```bash
git add desktop/src/app.rs
git commit -m "feat(design): mirror foundation references in app"
```

### Task 3: Pin handoff counts with a desktop consistency test

**Files:**
- Modify: `desktop/src/app.rs` (`#[cfg(test)] mod tests`)

**Interfaces:**
- Consumes: `include_str!` of `reference/design/machine/{screens,components,theme,tokens}.json`.
- Produces: `design_handoff_counts_match_the_rendered_reference` test.

- [ ] **Step 1: Write the failing/locking test**

Add:

```rust
#[test]
fn design_handoff_counts_match_the_rendered_reference() {
    let screens: Vec<serde_json::Value> =
        serde_json::from_str(include_str!("../../reference/design/machine/screens.json"))
            .expect("screens.json must parse");
    assert_eq!(screens.len(), 39, "handoff has 39 pages");
    assert!(
        screens.iter().any(|page| page["name"].as_str() == Some("38 · Wallpaper Doodle")),
        "page 38 must be registered"
    );

    let components: Vec<serde_json::Value> =
        serde_json::from_str(include_str!("../../reference/design/machine/components.json"))
            .expect("components.json must parse");
    assert_eq!(components.len(), 54, "handoff has 54 masters");

    let theme: serde_json::Value =
        serde_json::from_str(include_str!("../../reference/design/machine/theme.json"))
            .expect("theme.json must parse");
    assert_eq!(
        theme["theme"]["colors"].as_object().expect("colors object").len(),
        28,
        "theme has 28 color tokens"
    );

    let tokens: serde_json::Value =
        serde_json::from_str(include_str!("../../reference/design/machine/tokens.json"))
            .expect("tokens.json must parse");
    assert_eq!(tokens.as_object().expect("tokens object").len(), 28);
}
```

- [ ] **Step 2: Run the test**

```bash
cargo test -p whatslite-desktop design_handoff_counts_match_the_rendered_reference
```

Expected: PASS. If it fails, fix the count or the page 38 registration in the JSON source, not the assertion.

- [ ] **Step 3: Commit**

```bash
git add desktop/src/app.rs
git commit -m "test(design): pin handoff page and token counts"
```

### Task 4: Final gate and publish

**Files:**
- No source changes unless a gate fails.

- [ ] **Step 1: Run the gate**

```bash
cargo fmt --all --check
cargo test -p whatslite-desktop
cargo test --workspace -- --test-threads=4
```

Expected: fmt clean; desktop 22 passed; workspace 1954+ passed, 0 failed.

- [ ] **Step 2: Confirm Pencil MCP and source sync**

```bash
opencode mcp list
git status --short | grep -vE 'reference/design/wallpaper/|\.agent/'
```

Expected: `✓ pencil connected`; no tracked changes except the intended ones.

- [ ] **Step 3: Commit and push**

```bash
git add -A
git commit -m "chore(design): complete handoff parity"
git push origin phase-0-foundation
```

## Non-Goals

- Wiring the 29 `SOON` controls to live WhatsApp services. `PRODUCT.md` defines the desktop as frontend-only; that is a separate backend product plan.
- Replacing deterministic fixtures with a database or network layer.
- Redesigning page 38 or the wallpaper artwork; the SVG is the source of truth.
