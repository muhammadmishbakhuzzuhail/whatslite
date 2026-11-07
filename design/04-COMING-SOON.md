# 04 · Coming soon matrix

Every surface below is **fully designed**. The backend wiring is the next step, so the control must answer with the stated fallback. Never hide the control, never leave it dead, never fake data. In the use-case demo, add/delete/edit commits stay local and never touch a real account — see `urutan-judul.md`.

Legend: `LIVE` = wire it · `SOON` = designed, not wired.

## Chat
| Surface | Status | Fallback |
|---|---|---|
| Chats list, conversation, composer, receipts, reply, delete scope, edit | LIVE | — |
| Search in chat | LIVE | — |
| Attachments (file, photos, poll, event, contact) | SOON | toast `Coming soon — attachments are not wired up in this build.` |
| Reactions | LIVE | — |
| Forward | SOON | toast `Coming soon — forwarding is not wired up.` |
| Pinned messages | SOON | toast `Coming soon — pinned messages are not wired up.` |
| Voice recording / playback | SOON | toast `Coming soon — voice messages are not wired up.` |
| Poll / event / location / contact / sticker / GIF | SOON | toast per item, copy in `machine/strings.json` |
| Disappearing messages | SOON | picker opens, `Save` raises the toast |
| Clear chat / export chat | SOON | confirm opens, action raises the toast |
| Chat lock | SOON | Windows Hello prompt, then toast |
| Security code | SOON | rendered, `Verify` raises the toast |

## Lists & sections
| Surface | Status | Fallback |
|---|---|---|
| Archived | SOON | rows render from sample data; `Unarchive` raises the toast |
| Starred | SOON | rows render; `Jump to message` raises the toast |
| Locked | SOON | empty state + `Set up` → toast |
| Calls | SOON | list renders; call / video raise the toast |
| Updates | SOON | list renders; post + viewer raise the toast |
| Channels | SOON | list + view render; follow/mute raise the toast |
| Communities | SOON | list + view + info render; join/exit raise the toast |
| Media hub | SOON | tabs render; download/forward raise the toast |
| AI | SOON | panel + prompt render; every action raises the toast |
| Broadcast lists | SOON | render; `Send` raises the toast |

## Settings
| Surface | Status | Fallback |
|---|---|---|
| Theme, font size, enter-to-send, read receipts, archive rules | LIVE | — |
| Profile name / about | LIVE | — |
| Photo, phone change, email, two-step, linked devices, account info, delete | SOON | form opens, save raises the toast |
| Visibility, disappearing default, blocked contacts, app lock, auto-lock | SOON | sheet opens, save raises the toast |
| Notifications, sound, preview level, quiet hours, overrides | SOON | toggles move locally, save raises the toast |
| Storage, auto-download matrix | SOON | renders, save raises the toast |
| Language, proxy, privacy rows, help links | SOON | row opens, action raises the toast |

## System
| Surface | Status | Fallback |
|---|---|---|
| Theme, dark mode | LIVE | — |
| QR pairing | SOON | renders 4 states; scan raises the toast |
| Accounts (max 2) | SOON | switcher renders; add account raises the toast |
| First sync screen | SOON | render with fake progress, no navigation away |
| Notifications (OS toast) | SOON | rendered in design only |
| Permissions, tray, update, error screen, drag & drop, paste | SOON | design only |
| Offline / reconnecting banner | LIVE | driven by a fake network flag for now |

## Rule of thumb
> If a control exists in the design and the data is not ready, it raises **one** toast with a single sentence. One sentence, no apology, no explanation longer than 12 words.
