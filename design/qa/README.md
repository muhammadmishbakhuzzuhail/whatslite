# QA captures — identity, doodle, and clickable demo

Captured from the native app on Linux/X11 at 1440×900 unless noted. These are
review artifacts, not design sources. The doodle is a build-time alpha mask
tinted at runtime by `theme::wallpaper_style`; light and dark share the artwork.
The app repeats the 800pt source at 550pt; the canvas shows an 800pt proof.

| File | What it proves |
|---|---|
| `1440-group-light.png` | Group chat: avatars top-aligned with sender names, dense 550pt doodle repeat (800pt source), 1440 layout. |
| `1440-group-dark.png` | Same screen in dark mode; doodle at 12% opacity. |
| `1440-header-hover.png` | Hover/identity state: chevron visible next to the group name, hover surface. |
| `1440-header-menu.png` | Hover state chevron and the dropdown beside/under the group title. |
| `1440-header-menu-dark.png` | Header dropdown in dark mode. |
| `1440-chat-list-menu.png` | The previously dead ellipsis now opens the Chats menu. |
| `1440-contact-profile.png` | Clicking a sender opens the contact profile overlay (saved name, number, username rule). |
| `1440-group-info.png` | Group info panel: member rows open profiles; settings rows are clickable. |
| `1440-usecase-demo.png` | 12-scenario demo sheet fits at 1440×900. |
| `1440-rtl.png` | RTL layout with the same wallpaper and top-aligned avatars. |
| `1440-text-125.png` | 125% text scale; no clipped bubbles or wallpaper gaps. |
| `1920-group-light.png` | 1920-wide layout. The X11 capture is 1920×1008 because the window frame is capped by the 1080px screen. |

## Reproduce

```bash
# 1440 light group chat
WHATSLITE_CHAT=1 cargo run -p whatslite-desktop

# header menu / chat list menu / profile / demo
WHATSLITE_CHAT=1 WHATSLITE_HEADER_HOVER=1 cargo run -p whatslite-desktop
WHATSLITE_CHAT=1 WHATSLITE_OVERLAY=header-menu cargo run -p whatslite-desktop
WHATSLITE_CHAT=1 WHATSLITE_OVERLAY=chat-list-menu cargo run -p whatslite-desktop
WHATSLITE_CHAT=1 WHATSLITE_OVERLAY=contact-profile cargo run -p whatslite-desktop
WHATSLITE_CHAT=1 WHATSLITE_OVERLAY=demo cargo run -p whatslite-desktop

# dark / RTL / text scale
WHATSLITE_CHAT=1 WHATSLITE_THEME=dark cargo run -p whatslite-desktop
WHATSLITE_CHAT=1 WHATSLITE_RTL=1 cargo run -p whatslite-desktop
WHATSLITE_CHAT=1 WHATSLITE_TEXT_SCALE=1.25 cargo run -p whatslite-desktop
```

The 3× path is covered by `wallpaper::tests::picks_the_1x_2x_3x_texture` and
`wallpaper::tests::pre_rendered_tiles_have_the_expected_sizes` (800/1600/2400).
A 3× screenshot is not captured because this session runs at scale 1.

## Loading / empty / error

| File | What it proves |
|---|---|
| `1440-loading-skeleton.png` | Chat list skeleton rows + conversation bubble skeleton; composer hidden until ready. |
| `1440-error-state.png` | Chat list and conversation error cards, red alert icon, Retry; composer hidden until ready. |
| `1440-media-loading.png` | Media list skeleton rows + media grid skeleton tiles. |
| `1440-media-error.png` | Media list and media detail error states with Retry. |
| `1440-readonly-fields.png` | Form value rows are read-only and have no chevron; the Next button is the action. |

Reproduce:

```bash
WHATSLITE_CHAT=1 WHATSLITE_LIST_STATE=loading cargo run -p whatslite-desktop
WHATSLITE_CHAT=1 WHATSLITE_LIST_STATE=failed  cargo run -p whatslite-desktop
```

Every list uses the same `list_body` contract: skeleton → failed (red alert + Retry) →
empty (specific copy) → ready. The conversation has its own bubble skeleton and error card.
