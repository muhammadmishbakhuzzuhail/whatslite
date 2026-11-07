# 02 · Component registry

54 reusable masters. Count = number of `ref` instances in the file.

| Component | Size | Instances | Key children |
|---|---|---|---|
| `Wallpaper / Doodle Tile` | 800×800 | 68 | 2×2 proof of a seamless 400pt pattern; image fill `wallpaper-doodle.png`; vector source `desktop/assets/wallpaper/doodle.svg` |
| `App / Conversation / Bubble / In` | 360×None | 39 | Message, Meta |
| `App / Titlebar / Windows` | fill_container×32 | 27 | Title Group, Window Controls |
| `App / Sidebar / Windows` | 200×796 | 24 | App / Sidebar / Nav Item / Chats / Selected, App / Sidebar / Nav Item / Updates, App / Sidebar / Nav Item / Calls, App / Sidebar / Nav Item / Communities, App / Sidebar / Nav Item / Media, App / Sidebar / Nav Item / AI |
| `App / Sidebar / Nav Item` | 184×40 | 22 | Left, Badge |
| `App / Conversation / Bubble / Out` | 360×None | 22 | Message, Meta |
| `App / Conversation / Composer` | fill_container×56 | 15 | Plus Button, Input Pill, Emoji Button, Send Button |
| `App / Conversation / Header` | fill_container×58 | 14 | Identity, Actions |
| `App / Chat List / Header Master` | fill_container×64 | 11 | More Menu, List Title, New Chat |
| `Logo / App Icon` | 114×114 | 8 | Mark Glyph |
| `App / Conversation / Date Pill` | None×None | 8 | Label |
| `App / Overlay / Toast / Coming Soon` | None×None | 8 | Icon, Text |
| `App / Skeleton / Chat Row` | 300×80 | 8 | Avatar, Body, Right |
| `App / Skeleton / Media Tile` | 189×140 | 8 | Scrim, Duration Bar, Placeholder Icon |
| `App / Conversation / Bubble / Media Out` | 288×None | 4 | Media |
| `App / Conversation / Notice / Encrypted` | 600×None | 4 | Lock, Body |
| `App / Skeleton / Bubble` | 300×None | 4 | Line 1, Line 2, Line 3 |
| `App / Conversation / Bubble / Voice Out` | 300×None | 3 | Play, Wave |
| `App / Conversation / Notice / System` | 600×None | 3 | Timer, Body |
| `App / Sidebar / Icon Rail` | 64×668 | 3 | Rail Item / chat, Rail Item / donut_large, Rail Item / call, Rail Item / image, Rail Item / auto_awesome, Rail Item / lock |
| `App / Coming Soon / Section` | 640×420 | 3 | Disc, Title, Body, CTA, Note |
| `App / Info Panel / Contact` | 380×868 | 3 | Head, Hero, Actions, Media Block, Row Mute notifications, Row Disappearing messages |
| `App / Sidebar / Profile Row` | 184×40 | 2 | Left, Edit |
| `App / Composer / Attachment Menu` | 380×None | 2 | Grid |
| `App / Overlay / Context Menu` | 200×None | 2 | Item / Reply, Item / Forward, Item / Copy, Item / Star message, Item / Delete |
| `App / Skeleton / Conversation` | 520×340 | 2 | Bubble Skeleton 1, Row End, Bubble Skeleton 3, Row End 2 |
| `App / Conversation / Empty State` | 640×420 | 2 | Logo Mark, Title, Body, Divider, Encrypted |
| `App / Info Panel / Community` | 380×868 | 2 | Head, Hero, Actions, Description, Announcements Row, Groups Label |
| `Logo / Primary Lockup` | None×None | 1 | Mark, Wordmark |
| `Logo / Monochrome` | None×None | 1 | Mark, Wordmark |
| `App / Conversation / Bubble / Document Out` | 300×None | 1 | File Row, Meta |
| `App / Conversation / Bubble / Link In` | 320×None | 1 | Message, Link Card, Meta |
| `App / Conversation / Bubble / Video Out` | 268×None | 1 | Inner, Meta Wrap |
| `App / Conversation / Bubble / Reply In` | 340×None | 1 | Quote, Message, Meta |
| `App / Conversation / Bubble / Deleted In` | 300×None | 1 | Message, Meta |
| `App / Overlay / Modal / Coming Soon` | 300×None | 1 | Title, Body, Actions |
| `App / Drawer / Contact Info` | 360×640 | 1 | Head, Hero, Actions, Divider, Media Block, Divider |
| `App / Sidebar / macOS` | 192×688 | 1 | Traffic Lights, App / Sidebar / Nav Item / Chats / Selected, App / Sidebar / Nav Item / Updates, App / Sidebar / Nav Item / Calls, App / Sidebar / Nav Item / Media, App / Sidebar / Nav Item / AI |
| `App / Info Panel / Group` | 380×868 | 1 | Head, Hero, Actions, Members, Media Block, Settings Rows |
| `App / Status / Filmstrip` | None×84 | 1 | Item Frame 1, Item Frame 2, Item Frame 3, Item Frame 4, Item Frame 5 |
| `App / Tooltip` | None×None | 1 | Label |
| `App / Select` | 180×None | 1 | Field, Menu |
| `App / Checkbox` | None×None | 1 | Box, Label |
| `App / Radio` | None×None | 1 | Dot, Label |
| `App / Progress` | 120×6 | 1 | Fill |
| `App / Avatar Group` | None×None | 1 | Avatar RP, Avatar MA, Avatar +9 |
| `App / Badge` | None×None | 1 | Count |
| `App / Keycap` | None×None | 1 | Keys |
| `App / Call Bar` | 420×None | 1 | Avatar, Body, Action mic, Action call |
| `App / Toast Stack` | 360×None | 1 | Toast info, Toast check_circle, Toast error |
| `App / Accounts Switcher` | 200×None | 1 | Stack, Body |
| `App / Chat List / Chat Row` **unused** | 300×80 | 0 | Avatar, Body, Right |
| `App / Chat List / Search Field` **unused** | 276×32 | 0 | Search Icon, Placeholder |
| `App / Chat List / Filter Chip` **unused** | 72×34 | 0 | Label |
