# Gio UI audit — remaining gaps (2026-06-22)

Headline: the Gio UI is still largely a **render-only mockup**. Of ~125 exported
`App` engine methods, only ~9 are wired (`GetState`, `QRCode`, `GetChats`,
`GetMessages`, `OpenChat`, `Logout`, `LinkWithPhone`, `SendText`; `AddViaQR`/`MyQR`
exposed but uncalled). Most view funcs are `(gtx, th, t)` — no controller, no
engine ref — so they paint demo data and do nothing on interaction. "Interactivity"
is mostly `u.overlay = "..."` toggling, not real actions.

Use the capture helpers to drive the analyze→fix loop:
- In-app live shots (real WA data): `WLGIO_SHOTDIR=docs/ui-shots ./cmd/whatslite-gio/run.sh`
- On-screen pixels: `tools/ui-capture.sh` / `--watch`

## TOP 8 to build next (ranked)

1. **Message context-menu → engine** — `ui.go` ctxMenu items (Reply/React/Star/
   Delete/Forward/Info) are no-ops; wire against `ctxIdx`.
2. **Chat-row actions menu** — Archive/Pin/Mute/MarkRead/Delete/Clear unreachable.
3. **Sidebar SearchBar + Filters** — no search input, no filter chips at all.
4. **Attach picker → send media** — `PickerView` decorative; wire SendMedia/Contact/
   Location/Poll + file pick.
5. **Reaction picker applies reaction** — emojis not clickable; call `React`, close.
6. **Scroll-to-bottom + day separators + optimistic send** — core chat UX.
7. **Real avatars & media bytes** — `AvatarBytes`/`MediaBytes` never called; avatars
   are initials, media bubbles show "[type]".
8. **Forward modal with chat selection** — static card; build recipient list + `Forward`.

Runner-ups: presence/typing (`SubscribePresence`/`SendTyping`), settings rows
(Privacy/Storage/Profile/Notifications), pagination (`LoadOlderHistory`).

## By category (severity | file:line | problem | fix)

### Interactivity gaps
- HIGH | ui.go sidebar | no SearchBar → can't filter chats | editor + filter u.chats
- HIGH | reactionpickerview.go | reaction emojis not clickable | clickables → `React`
- HIGH | ModalsView forward | static card, no chat-pick/send | list → `Forward`
- HIGH | ctxMenu Reply/Star/Hapus/Forward/Info | only close overlay | wire engine
- HIGH | pickerview.go | attach picker decorative | wire Send* + file pick
- HIGH | settingsview rows 1–6 | only Tema/Keluar wired | Privacy/Retention/KeepDeleted/Lang/Storage
- MED  | infodrawerview.go | fully static | feed GetGroupInfo + Mute/Block/Leave/Clear
- MED  | composer mic | nil clickable | voice record → SendMedia(audio)
- MED  | lightboxview.go | never opened from a bubble | open on media tap + Download

### Engine methods exposed but unwired
- Chats: MarkRead/Unread, Archive, Pin, Mute, DeleteChat, ClearChat, ExportChat, GetPinned
- Send: Reply, Forward, React, EditMessage, DeleteMessage, StarMessage, PinMessage, GetStarred
- Media send: SendMedia, SendSticker, SendGif, SendPoll, SendLocation, SendContact, ScheduleMessage
- Presence/history: SendTyping, SubscribePresence, GetMessagesBefore/LoadOlderHistory
- Connect: AddViaQR, MyQR
- Panes still static: Status, Channels, Calls, Contacts, Scheduled, Groups, Communities

### Missing components (present in Svelte)
SearchBar, Filters, NewChatModal, GifPicker, StickerPicker, ProfilePane, PrivacyPane,
StoragePane, StarredPane, CommunitiesPane, MediaPreviewModal, IncomingCall, ConfirmDialog/
Toast. Dead code (defined, never dispatched): searchview.go, archivedpaneview.go,
scheduledpaneview.go.

### UX behaviors missing
scroll-to-bottom on send, day separators, unread divider, optimistic send echo,
presence/typing subtitle, delivery ticks on out bubbles, reply-quote rendering,
real media thumbnails/waveform, localized timestamp grouping.
