# 03 · Behaviour rules

Rules that are invisible in the .pen file. Implement all of them.

## 1 · Bubble grouping (the most important rule)
A run = consecutive messages from the same sender within 5 minutes.

- Avatar and sender name appear **only on the first bubble** of a run.
- The group avatar is **top-aligned with the sender name** (28px, 6px before the text column). It never sits at the bubble bottom and never overlaps the bubble.
- The sender name and avatar are **buttons**: clicking either opens that person's contact profile (Group info member rows do the same). The thread itself is not renamed.
- The **tail** (4px corner) appears **only on the last bubble** of a run.
- Gap inside a run: **2px**. Gap between runs: **8px**.
- Outgoing runs: no avatar, no name, tail on the last bubble only.
- Max bubble width: **420px** at 1440 (≈68% of the thread); text wraps, never grows the column.
- Emoji-only message (≤3 emoji): renders large, without bubble fill, aligned to the sender side.

## 2 · Media
- **Never auto-download.** Four states per media: `tap to download`, `downloading (progress ring)`, `failed (retry on the bubble)`, `expired (no longer available)`.
- Upload has its own states: `uploading (determinate bar inside the bubble)`, `failed (retry + remove)`.
- Album: 1 = full width, 2 = two columns, 3 = one large + two, 4 = 2×2, 5+ = fourth cell shows `+N`.
- View-once: `sealed → opening → opened`. Cannot be forwarded, no screenshot promise.
- Voice: waveform with progress, speed cycles `1× → 1.5× → 2×`, transcript collapsed under the waveform when available.

## 3 · Conversation states
- Date separators: `Today` / `Yesterday` / `12 September` / `12 Sep 2025`.
- Receipts: one check = sent, two = delivered, two blue = read. Colour only, glyphs never change.
- Edited messages keep `· edited` after the time; deleted shows italic `This message was deleted`.
- Reply quotes clamp to 3 lines; quoting a deleted message shows the same italic line.
- Reactions: chips straddle the bubble edge 4px below; 7+ collapses to `+N` which opens the full list.
- Pinned bar: max 3, header grows 24px, `2 pinned` opens the list.
- In-chat search: field replaces the header, `↑↓` moves between hits, `n of m`, hit ringed for 2s.

## 4 · Composer
- Grows 1 → 5 lines (40 → 108px), then scrolls. Buttons stay pinned.
- `Enter` sends, `Shift+Enter` adds a line — configurable in Settings › Chats.
- Mic swaps to send the moment there is text; recording shows timer + `slide to cancel` + lock.
- Attachment preview before send: thumbnail strip, caption field, size, single send.
- Reply mode, edit mode and disappearing timer are mutually exclusive; **edit > reply > disappearing**.

## 5 · Chat list
- Order: pinned first, then last activity. Unread badge: `8`, `99+`, `128` — width grows, row height fixed at 80.
- Name truncates before the timestamp; preview truncates with ellipsis.
- Muted + unread = dot, not badge; preview stays bold.
- Row hover swaps the timestamp for `⋯` (menu: archive, mute, pin, read, lock, delete).

## 6 · Network
Three states, not one: `Connecting`, `Offline`, `Reconnecting`. Only **Offline** blocks sending; the others queue silently. Offline shows a 32px banner above the composer plus an outbox count.

## 7 · Accounts
Maximum **two** accounts. Switcher sits at the bottom of the sidebar; the inactive account carries its own unread badge. First launch after pairing shows a blocking **sync** screen (`Syncing your messages · 3.240 of 8.112`).

## 8 · Calls
States: dialing, ringing, connecting, reconnecting, busy, no answer, ended by them, declined by you. Minimised call = call bar above the composer. Group call = grid with active-speaker ring and long-press to pin.

## 9 · List states
Every list has all three:
| State | Rule |
|---|---|
| Empty | icon + one sentence + one action. Never a blank panel. |
| Loading | skeleton that mirrors the real row, shown only after 200ms |
| Failed | one line + `Retry`. Never reuse the empty state. |

Exact copy for 18 lists is in `docs/` (search “List States”).

## 10 · Keyboard & focus
- Focus ring: 2px accent, 2px offset, drawn outside the element, **keyboard only**.
- Tab order: sidebar → list → header → thread → composer → send → panel → attachments.
- `Esc` closes exactly one layer (drawer → menu → modal).
- Focus is trapped inside modal and drawer.

## 11 · Accessibility
- Every icon-only button needs an accessible label.
- Reduced motion: all transitions become instant fades.
- Text scaling 125/150% must not clip (allow the composer and rows to grow).
- Minimum contrast: body text 4.5:1, large text 3:1. Re-check `accent-soft` text in dark.

## 12 · Content rules
- UI labels: English. Sample chats: Indonesian.
- Never invent placeholders like “Lorem ipsum” or “John Doe”.
- Time format follows locale; date separators localise.
- RTL: mirror the layout, keep numbers LTR inside RTL text.
