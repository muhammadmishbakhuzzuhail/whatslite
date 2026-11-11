# WhatsLite Desktop

A WhatsApp-inspired desktop UI written in Rust with GPUI Kit. The current
build is frontend-only: screens and the use-case demo run on deterministic
fixtures. The next step is the real backend — **no mock database**. The UI
talks through `whatslite-services` to `whatslite-wa-adapter` → `bridge` →
`wa-rs` and the paired account. See `../reference/design/urutan-judul.md` §6 for the
full decision and boundary.

## Run

```bash
cargo run -p whatslite-desktop
```

Use-case demo: hover the conversation title, open the chevron menu and choose
`Use case demo` — or run with `WHATSLITE_OVERLAY=demo`.

The doodle wallpaper source is the editable seamless SVG at
`../reference/design/machine/wallpaper-doodle-source.svg`. `build.rs` rasterises it to
1x/2x/3x (800/1600/2400) alpha masks; `src/wallpaper.rs` tints the alpha with
`theme::wallpaper_style` and repeats it at an **800pt** logical step. Light ink
is `#C9B9A3` at 60%, dark ink is `#D8CCBC` at 20%; larger windows add more
tiles, the tile itself is never stretched.

On Linux, GPUI runs natively on Wayland. To use the X11 backend for screenshot
capture:

```bash
env -u WAYLAND_DISPLAY cargo run -p whatslite-desktop
```

`desktop/build.rs` creates a local linker symlink for `libxkbcommon-x11.so`
when a distribution only ships `libxkbcommon-x11.so.0`. This keeps the build
working without changing system packages.

## Design source

The implementation follows the handoff package in `../reference/design/`:

- `reference/design/machine/theme.json` — 28 light/dark colour tokens
- `reference/design/urutan-judul.md` — title order, hover menu, group profile, avatar and doodle fixes, demo, backend rule
- `reference/design/machine/strings.json` — copy deck
- `reference/design/03-BEHAVIOUR-RULES.md` — grouping, media, states, keyboard rules
- `reference/design/04-COMING-SOON.md` — fallback copy for unwired controls
- `reference/design/05-BUILD-ORDER.md` — layout and acceptance rules
- `reference/design/01-SCREENS.md` page 38 — final doodle wallpaper proof (light/dark v19, 1024 × 1024 SVG, 800 pt repeat)

The only colour literals live in `src/theme.rs`; all UI surfaces read the
palette. Dark mode is a value swap, not a second UI.

## Structure

- `src/theme.rs` — tokens, palette, GPUI Kit theme bridge, layout constants
- `src/model.rs` — deterministic fixtures for chats, messages, calls, updates,
  communities, media, contacts, broadcast lists and accounts
- `src/components.rs` — reusable visual primitives (avatars, badges, chips,
  cards, empty states, skeletons, media tiles, menu shells)
- `src/app.rs` — shell, routing, state machine, all screens and overlays
- `src/assets.rs` — bundled Inter and Noto Emoji
- `src/wallpaper.rs` — tiled doodle wallpaper (build-time 1x/2x/3x textures)
- `src/main.rs` — window bootstrap, font loading and theme initialisation

## Implemented surfaces

- Group identity: top-aligned sender avatars, clickable sender names and member
  rows → contact profile, hover chevron → header dropdown beside the title
- List states: skeleton rows, specific empty copy, failed state with red alert + Retry; conversation bubble skeleton/error
- Use-case demo sheet: every key flow reachable; add/delete/edit stay local
- WhatsApp-style hand-drawn doodle wallpaper (tiled, dark-mode aware)
- Windows and macOS shell variants
- Onboarding QR states, pairing sync and account switcher / add-account sheet
- Chat list with filters, search, unread, pinned, muted, typing, draft, 99+,
  long-name truncation and empty / loading / failed states
- Conversation with grouped bubbles, replies, reactions, receipts, edited and
  deleted messages, media states, voice speed/transcript, albums, view-once,
  upload states, poll states, event states, contact cards, static and live
  location states, stickers and missing packs, GIF failure, mentions,
  forwarded labels, mixed rich-text formatting, reaction overflow and list,
  calls, pinned-message sheet and unread dividers
- Composer with multiline growth, reply/edit modes, attachment preview and
  staging, emoji and sticker pickers, poll/event create sheets, draft restore,
  offline queue, outbox count, call bar, voice recording and a blocked-contact
  read-only state
- Contact, group and community info panels with group member management,
  clear/export chat, security code, channel info, community members and
  member approvals
- Calls list, missed-call banner, incoming call accept/decline, voice/video/
  group call surfaces with 2/3/4/6+ layouts and every call state
- Updates list, status viewer (pause, seen-by, viewers list, filmstrip),
  status composer with a 30-item cap, drag-to-reorder sequence, status privacy
  and font pickers, disappearing picker, channels and channel info
- In-chat search with All / Media / Links / Docs filters and hit navigation,
  message selection mode with type-aware actions, scroll-to-quoted-message
- Communities list/view/info
- Media hub with per-chat filtering, select mode, star/forward/download/delete
  actions, lightbox, paging and zoom; media cache and drag-file-to-chat-row
- AI panel with idle, suggested-reply, streaming and error states, plus the
  draft-reply sheet
- Contacts list/detail/edit sheet
- Archived, starred and locked sections
- Broadcast list, thread, interactive recipient picker, create flow, naming
  step and forward sheet
- Settings: six categories plus the full set of sub-pages, including linked
  devices, visibility, change number, recovery email, profile photo,
  auto-lock, notification overrides, sound, quiet hours, blocked contacts,
  storage per chat, wallpaper, text size, two-step verification, account
  info, delete account, help, system status, call states, network states,
  primitives, an editable shortcut page, disappearing messages, language,
  proxy, about and an in-app design reference covering foundations, components,
  handoff and audit
- Overlays: attachment menu, context menu, emoji/sticker pickers, delete and
  clear-chat confirmations, new chat/group/community, poll/event composers,
  file dialog, group members, export chat, security code, channel info,
  community members, member approvals, forward sheet, poll votes, locked
  unlock, status viewers, accounts, add account, AI draft, OS notification,
  permission, tray, update, error, drag & drop, paste and toast stack
- Light/dark/System theme, live OS appearance switching, 1024/1366/1440/1920,
  RTL layout, 125/150% text scaling and reduced motion
- Desktop window settings: autostart, always on top, close to tray, taskbar
  badge, spellcheck, update channel and print thread

## QA environment flags

These are only for local screenshots and state review:

```bash
WHATSLITE_ROUTE=calls|updates|communities|media|ai|contacts|settings|archived|starred|locked|broadcast|onboarding
WHATSLITE_CHAT=0..6
WHATSLITE_HEADER_HOVER=1
WHATSLITE_INFO=contact|group|community
WHATSLITE_MEDIA_SELECT=1
WHATSLITE_THEME=dark
WHATSLITE_SHELL=macos
WHATSLITE_LIST_STATE=loading|failed
WHATSLITE_NETWORK=connecting|offline|reconnecting
WHATSLITE_CALL=1|group|video
WHATSLITE_CALL_STATE=dialing|ringing|connecting|reconnecting|busy|no-answer|ended-by-them|declined
WHATSLITE_STATUS=1
WHATSLITE_SEARCH=1
WHATSLITE_SEARCH_QUERY=foto
WHATSLITE_SEARCH_FILTER=media|links|docs
WHATSLITE_MESSAGE_SELECT=1
WHATSLITE_GIF_NO_RESULTS=1
WHATSLITE_OVERLAY=demo|header-menu|chat-list-menu|contact-profile|attachments|context|emoji|delete|newchat|newgroup|new-community|lightbox|status|status-font|status-privacy|status-viewers|pinned|accounts|add-account|ai-draft|shortcut-editor|poll|event|disappearing|sticker|gif|file-dialog|group-members|clear-chat|export-chat|security-code|channel-info|community-members|member-approvals|forward|poll-votes|reactions|locked-unlock|system-os|system-permission|system-tray|system-update|system-error|system-drop|system-paste|system-toasts|new-broadcast
WHATSLITE_SUBPAGE=linked-devices|visibility|change-number|recovery-email|profile-photo|auto-lock|overrides|sound|preview|quiet-hours|blocked|block-picker|wallpaper|font-size|two-step|account-info|delete-account|storage-per-chat|help-centre|design|contact-us|report|system-status|call-states|network|primitives|shortcuts|language|proxy|about
WHATSLITE_PROFILE_PHOTO=empty|cropping|uploaded|saved
WHATSLITE_STATUS_ITEMS=30
WHATSLITE_STATUS_DRAG=1
WHATSLITE_RECORDING=1
WHATSLITE_STAGED=1|3
WHATSLITE_FILE_SELECTED=1
WHATSLITE_CALL_INCOMING=1
WHATSLITE_GROUP_CALL_SIZE=2|3|4|6
WHATSLITE_COMMUNITY_MEMBER=1
WHATSLITE_OUTBOX=3
WHATSLITE_MEDIA_CACHE=0
WHATSLITE_DRAG_FILE=2
WHATSLITE_SYSTEM_THEME=1
WHATSLITE_OS_THEME=dark|light
WHATSLITE_AI=idle|suggested|streaming|error
WHATSLITE_REDUCED_MOTION=1
WHATSLITE_TEXT_SCALE=1.25
WHATSLITE_RTL=1
```
