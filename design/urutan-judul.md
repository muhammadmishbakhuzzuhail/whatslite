# urutan-judul.md — title order, click contract, and demo

Status: implementation spec for `desktop/` and the `whatslite.pen` design.
Scope: group chat identity, header affordance, profile access, doodle, and the
clickable use-case demo. Nothing in this file requires a mock database.

---

## 1. Urutan judul (top → bottom)

This is the order in which a title is read on a group-chat screen. Later titles
never replace earlier ones; they only add detail.

| # | Title | Where | Primary content | Click target |
|---|---|---|---|---|
| 1 | App title | window chrome | `whatslite` | window controls only |
| 2 | Section title | sidebar | `Chats`, `Calls`, `Updates`, … | switches route |
| 3 | List title | chat list header | `Chats` + search / new chat / more | search, new chat, chat list menu |
| 4 | Conversation title | header | group name, participant summary, chevron on hover | opens the header dropdown |
| 5 | Sender title | inside a group bubble run | `Rina`, `Bayu`, `Sari` | opens that person's profile |
| 6 | Bubble content | message bubble | text / media / voice | opens the message context menu |
| 7 | Panel title | info panel | `Group info` / `Contact info` | close, member rows |
| 8 | Profile title | profile sheet | saved name or number | profile actions |
| 9 | Menu title | header dropdown | group name + `Group options` | menu items |
| 10 | Sub-page title | settings / flows | e.g. `Create group` | back, commit CTA |

Rules:

- The **group name in the conversation header is the title of the thread**, not
  the title of a person. Clicking a participant's name inside a message must not
  rename or navigate the thread — it opens that participant's profile.
- The **sender title is part of the message run**. It is shown only on the first
  message of a consecutive run from the same sender (`show_identity`).
- A title is always a button only when it has an action. Static section labels
  (`4 MEMBERS`, `MEDIA, LINKS AND DOCS`) stay static.

---

## 2. Header hover + dropdown

### Hover

On mouse enter over the identity block (avatar + group name + status):

1. the block gets the normal hover surface (`p.hover()`);
2. a **down chevron** appears to the right of the group name;
3. the chevron is hidden again on mouse leave.

The chevron keeps its layout slot at rest (`opacity: 0`), so the name does not
jump when the arrow appears.

### Click

Clicking the identity opens a dropdown **beside/under the title**, not in the
middle of the screen:

- anchored under the conversation header,
- left edge aligned with the identity block (16px inset from the conversation
  edge),
- width ~268px, 6px radius, 1px border, small shadow,
- closed by `Esc`, the scrim, or selecting an item.

Menu content (group):

| Item | Result |
|---|---|
| Group info | opens `InfoPanel::Group` |
| Add members | opens the group members sheet |
| Search in chat | opens the in-chat search field |
| Media, links and docs | opens the media route |
| Mute notifications | toast `Muted for 8 hours · local demo` |
| Disappearing messages | opens the timer picker |
| Clear chat | opens the clear-chat confirmation |
| Use case demo | opens the clickable demo sheet |
| Exit group | deferred toast |

Menu content (one-to-one): `View profile`, `Search in chat`,
`Media, links and docs`, `Mute notifications`, `Disappearing messages`,
`Clear chat`, `Use case demo`.

---

## 3. Group avatar bug (the screenshot)

**Symptom:** the 28px avatar circle sat at the bottom of the bubble and looked
clipped/behind it. The screenshot showed `RI` and `BA` vertically centred while
the sender name was above the bubble.

**Cause:** the message row was `align-items: end` (`items_end()` in GPUI), so the
avatar followed the bottom of the tallest child. The avatar also had no stable
column of its own.

**Fix:**

- row is `align-items: start` (`items_start()`);
- avatar sits in its own 28px column, `gap: 6px` before the text column;
- text column is vertical: sender title, then bubble;
- avatar top edge aligns with the sender title, so it reads **above** the bubble;
- avatar is shown only on the first message of a sender run;
- clicking the avatar or the sender title opens the same profile;
- no negative margins, no overlap with the bubble.

Acceptance: at 1024, 1366, 1440 and 1920 widths the avatar is fully visible, the
sender title is not clipped, and bubble left edge = avatar right edge + 6px.

Design file: `paZJL` → rows `Row In Rina`, `Row In Bayu`, `Row In Sari` now use
the horizontal layout described above.

---

## 4. Doodle wallpaper

WhatsApp's doodle is **hand-drawn line art of everyday objects**, not a grid of
UI icons. The old tile was a repeated Material Symbols grid; it read as
"dashboard", not "chat".

Spec:

- 400 × 400 logical pattern, irregular scatter — not a 5 × 5 grid;
- the canvas component is an 800 × 800 2×2 proof of the same pattern;
- edge objects are wrapped to the opposite edge, so the repeat is seamless;
- thin uniform strokes (~2.5px in the 800 source; 1.25 logical at 400),
  rounded caps/joins;
- objects: camera, coffee, plant, headphones, paper plane, bicycle, lightbulb,
  book, gift, umbrella, music note, alarm clock, scissors, sunflower, cloud,
  moon, envelope, pin, pizza, shopping bag, tape, paperclip, compass, globe;
- no fill, no shading, no text, no emoji;
- one stroke `#D8CCBC` on `$wl-bg-conversation`; dark mode keeps the same
  tile at 12% opacity;
- the app is clipped and repeated by the conversation stage at a 400pt
  logical step;
- the canvas tiles at 800pt; each 800 tile contains a 2×2 proof, so the visible
  density matches the 400pt app repeat;
- `desktop/build.rs` rasterises the original SVG at 1×, 2× and 3×
  (400/800/1200); the runtime picks the matching texture, so 1440 and 1920
  stay sharp at 3× scale;
- opacity is 100% in light mode and 12% in dark mode.

Do not re-generate the artwork per screen. It is one reusable `Wallpaper / Doodle
Tile` (`IqUX9`) instance in the canvas and one texture cache in the app. The
vector source is `desktop/assets/wallpaper/doodle.svg`; the canvas uses the
matching `machine/wallpaper-doodle.png` (referenced as
`./wallpaper-doodle-400.png` next to the `.pen`) because the `.pen` stores textures
as image fills. This is original artwork, not a WhatsApp asset.

---

## 5. Use-case demo (all buttons connected)

Goal: a reviewer can walk the product without dead ends. **Add, delete, edit
and send commits are local/demo only** — their surfaces open, and no real
account is touched. A control must never be silently inert.

Launch:

1. hover the conversation title → click the chevron / identity;
2. choose **Use case demo** in the dropdown.

Scenarios in the demo sheet:

1. **Group chat · identity** — select `Tim Produk`; hover the title; click a sender.
2. **Contact profile** — click `Rina`; saved name, number, username rule, actions.
3. **Header dropdown** — open the menu beside the title.
4. **Attachment menu** — file / photo / poll / event / contact / sticker / GIF.
5. **Message actions** — reply, react, edit, forward, copy, star, select, delete.
6. **Search in chat** — field, hit count, previous / next.
7. **Group info** — member rows open profiles; add member opens the members sheet.
8. **Create group** — full flow; the new group stays local until the backend exists.
9. **Delete message** — confirmation is wired; the change stays local until the backend exists.
10. **Edit contact** — form opens; the save stays local until the backend exists.
11. **Chat list menu** — the ellipsis opens create, find and organise.
12. **Theme swap** — light/dark without layout change.

Rules for the FE:

- Every icon-only control has an `aria_label`, a focus ring and a keyboard path.
- A local/deferred commit closes the surface and shows a specific toast, e.g.
  `Delete stays local until the backend is wired` — never a generic dump.
- `Esc` closes only the top layer; the scrim closes the header menu and profile.
- The demo is read-only. It never writes fixtures into the real account.

---

## 6. Mock database or the real backend?

**Decision: no mock database.** Connect to the real `wa-rs` backend and the real
paired account.

Why:

- The protocol, E2E encryption, session store and event pipeline already exist in
  this repository. A second mock schema would duplicate them and hide integration
  bugs (JIDs, LIDs, history sync, app-state, media encryption).
- A mock DB cannot validate the flows that matter: pairing, reconnect, outbox,
  delivery receipts, group metadata.
- `crates/whatslite-mock-backend` is useful **only** as deterministic UI fixtures
  for screenshots and design QA. It must not become a second source of truth.

Target boundary:

```
desktop UI
  → whatslite-services traits (chat, contacts, groups, media, auth)
  → whatslite-wa-adapter (maps wa-rs events and commands)
  → bridge / wa-rs client
  → real WhatsApp account
```

Rules:

- The FE reads from the service traits, never from `model.rs` directly once the
  wiring lands. Fixtures stay behind a `demo`/`fixtures` provider.
- A local store is only a cache of the real session/history (wa-rs SQLite). It is
  not a separate product database.
- Every write goes through the adapter; no optimistic fake success for send,
  delete or edit. Show pending → sent → delivered from real events.
- Do not expose session keys or the raw QR to the UI; the bridge owns pairing.

---

## 7. Number saved by default, or username?

**Never save a number automatically.** The default label is decided in this order:

1. **Saved contact** → saved name is the title; phone is secondary.
2. **Unsaved number** → the formatted phone number is the title, with
   `Not in your contacts` and a `Save contact` action.
3. **Username** → show it only as a secondary line, only when the person chose one
   and privacy allows it. It never replaces the saved name or the number.
4. **Group participant** → saved name if known, otherwise the number. Do not guess
   a display name from the message text.

The profile sheet includes `Phone`, `Username` and `Saved contact` rows so the
rule is visible. `Save contact` is an explicit action, never a side effect of
opening a chat.

---

## 8. Implementation map

| Concern | File / function |
|---|---|
| Header hover, chevron, dropdown | `desktop/src/app.rs` → `render_conversation_header`, `header_menu` |
| Profile sheet | `desktop/src/app.rs` → `contact_profile_sheet` |
| Sender avatar + title | `desktop/src/app.rs` → `render_bubble` |
| Member rows | `desktop/src/app.rs` → `group_member_row`, `info_group`, `group_members_sheet` |
| Demo sheet | `desktop/src/app.rs` → `use_case_demo_sheet`, `Action::StartDemo` |
| Wallpaper | `desktop/src/wallpaper.rs` (tiler), `desktop/build.rs` (1x/2x/3x), `desktop/assets/wallpaper/doodle.svg` |
| Design source | `whatslite.pen` → `IqUX9` (doodle), `paZJL` (group rows), `dYttn` (header) |
| Services / adapter | `crates/whatslite-services`, `crates/whatslite-wa-adapter`, `bridge/` |

---

## 9. Acceptance checklist

- [ ] Avatar is top-aligned with the sender title and never clipped.
- [ ] Hover shows the chevron; click opens the dropdown next to the title.
- [ ] `Esc` and the scrim close the dropdown; only the top layer closes.
- [ ] Clicking `Rina`, `Bayu` or `Sari` opens that person's profile.
- [ ] Unsaved numbers show the number, `Not in your contacts` and `Save contact`.
- [ ] Usernames are secondary and never overwrite a saved name.
- [ ] Doodle is hand-drawn line art, subtle, tiled and readable in dark mode.
- [ ] Demo sheet lists 12 scenarios; every visible button reaches a surface.
- [ ] No mock database was added; the service boundary is the only data path.
- [ ] `cargo check -p whatslite-desktop` passes.
