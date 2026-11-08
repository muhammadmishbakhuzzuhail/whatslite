# 05 · Build order

Build in this order. Each step has an acceptance check; do not start the next step until it passes.

## 0 · Bootstrap
- Read `machine/tokens.json`, `machine/components.json`, `01-SCREENS.md`, `03-BEHAVIOUR-RULES.md`.
- Create the token layer (`tokens.css` or a palette module). **Acceptance:** switching a single `data-theme` attribute flips the whole UI, no component touched.

## 1 · Shell
Windows title bar 32px (mark 16, name 12/500, controls 46×32) · sidebar 200 · content. macOS variant: floating glass card 192 at 8.5pt, traffic lights inside.
**Acceptance:** both shells render at 1440×900 and 1250×704 with the corner radius and shadows from tokens.

## 2 · Primitives
Button, icon button, input, search, toggle, checkbox, radio, select, chip, badge, avatar, tooltip, keycap, progress, divider.
**Acceptance:** every primitive has default / hover / focus-visible / disabled, and focus ring matches §10.

## 3 · Chat list
Header 64 (⋯, title, +) · search 32 · filter chips 34 · rows 80 (7 states) · dividers.
**Acceptance:** long name truncates, `99+` badge grows, muted shows a dot, empty/loading/failed all exist.

## 4 · Conversation
Header 58 · wallpaper stage · date separators · **bubble grouping** · receipts · reply · reactions · edited/deleted · composer.
**Acceptance:** a run of 3 messages shows avatar+name once and one tail; media shows 4 download states; composer grows to 5 lines.

## 5 · Conversation extras
Pinned bar · in-chat search · scroll-to-bottom · voice · recording · upload states · album · system messages · emoji picker.
**Acceptance:** every item in `04-COMING-SOON.md` §Chat raises exactly one toast.

## 6 · Info panels
Contact 380 · group 380 (members, roles) · community 380. Panel squeezes the thread above 1280, overlays below 1100.
**Acceptance:** 1440 inline, 1366 inline with 486 thread, 1024 overlay + scrim + Esc.

## 7 · Sections
Archived · Starred · Locked · Calls · Updates · Channels · Communities · Media · AI · Broadcast.
**Acceptance:** each has empty / loading / failed, and each action maps to a toast from the matrix.

## 8 · Settings
6 categories, 26 sub-pages, biometric 5 steps, 3 form dialogs.
**Acceptance:** theme applies instantly; every other row raises a toast on save.

## 9 · System states
OS toast · permissions · tray · update · error · drag&drop · paste · toast stack · QR 4 states · accounts · first sync.
**Acceptance:** QR cycles all four states; toast stack shows max 3.

## 10 · Responsive & dark
1920 / 1366 / 1024 · icon rail below 1100 · dark via token swap.
**Acceptance:** no layout break at any width; dark has no hard-coded light surface.

## 11 · Polish
Keyboard navigation (arrow keys in lists), reduced motion, text scaling, ARIA labels, RTL mirroring.

---

## Definition of done
- [ ] Every screen in `01-SCREENS.md` exists as a route or view.
- [ ] Every component in `02-COMPONENTS.md` exists once and is reused.
- [ ] No raw hex outside the token layer.
- [ ] Every non-wired control raises the toast from `04-COMING-SOON.md`.
- [ ] Empty / loading / failed exist for all 18 lists.
- [ ] Keyboard: tab order + focus ring + Esc layers.
- [ ] Dark and 1024 verified.
- [ ] Brand: `messages-square` used everywhere, assets from `brand/`.
