# Design System v1 — WhatsApp macOS-inspired GPUI Kit

**Status:** Proposed reconstruction. This is not an official WhatsApp design system or evidence of exact app dimensions. Source-grounded macro patterns (macOS sidebar with labels, Liquid Glass materials in selected surfaces) are separated from chosen implementation measurements.

## 1. Canvas & 100% scale

- **Reference logical canvas:** 1512 × 982 pt (14-inch MacBook Pro example with default logical scaling). This is a *reference preset*, not a universal physical MacBook resolution.
- **macOS menu bar:** reserved 25 pt in our reference composition; application region begins at y=25, height 957 pt. Real menu-bar height differs by OS and display settings; **query the actual work area** at runtime.
- **Full-window:** the app fills its available desktop work area, not a floating screenshot centered on a wallpaper. macOS true fullscreen may hide its menu bar; our mockup targets maximized desktop window, not necessarily native fullscreen mode.
- **Pixel vs point:** logical 1512 × 982 pt may export to 1512 × 982 at 1x or 3024 × 1964 physical px at 2x. Never put a scaled illustration of a laptop inside a screen reference.
- **Reference exports:** one flat image per theme, `1512x982@1x.png`; if exporting Retina use `3024x1964@2x.png`, with exact same component point sizes.

## 2. Shell grid at the reference width

| Region | X (pt) | Width (pt) | Notes |
| --- | ---: | ---: | --- |
| Navigation | 0 | 208 | Can collapse to 72 on compact widths |
| Chat list | 208 | 360 | Resizable, min 300 max 440 |
| Conversation | 568 | 944 | Fills remaining content width |

The 3 panels occupy the **content area below the macOS menu bar**. Panel heights use the available application window; no artificial bottom dock inside the app. Chat list header=72, filter row=52, chat row=76, conversation header=72, bottom composer min=72. All dimensions are **selected design values**, not reverse engineered measurements of official WhatsApp.

## 3. Color and materials

Use semantic keys in `design-tokens.json`; never hard-code hex values in view components. Use `navigation`, `list`, `conversation`, `bubble_in`, `bubble_out`, `text_primary`, `accent`, etc. The wallpaper is a separate repeatable texture (optional asset), not text content. Light uses warm chat background; dark uses near-black teal background. Prioritize contrast and message readability.

Liquid Glass design language is documented for WhatsApp Mac in 2026 but was gradually rolled out. Use translucent toolbar/sidebar/composer surfaces **only if** host compositor and accessibility settings permit; otherwise fallback to opaque surfaces. Do not implement translucency by screenshotting the desktop. Respect reduce-transparency settings.

## 4. Typography

Default font stack: SF Pro Text / SF Pro Display where available as OS system fonts. Do not bundle proprietary font binaries. Body 14 pt, small 12 pt, heading 20 pt; weights 400/500/600/700. Long contact names truncate. Set min sizes and a11y scaling consciously; on real macOS, app text may scale independently from screenshot reference.

## 5. Components and required states

See `component-registry.csv` for 16 primitives and composites. Each requires hover, focus, pressed, selected, disabled where applicable, plus theme parity. Meaningful action states: loading, error, empty, reconnecting, recording, uploading and retry. State must come from model data; visual-only buttons are not an acceptable implementation.

## 6. Interaction & accessibility rules

- All icon-only buttons have tooltips and accessibility labels.
- Keyboard focus is visible; tab order follows navigation → list → chat header → messages → composer.
- Context menus open from click/right click, close on Escape/outside click, and preserve focus.
- Inbound/outbound bubbles have different alignment, colors, and accessible sent/read labels; don't encode message status only by color.
- List and message timeline use virtualization for long histories; restore scroll positions as conversations switch.
- State ownership: `AppState` owns active route/selected chat; `ConversationState` owns composer draft/reply/media state; `ThemeState` resolves semantic tokens.
- Avoid animated effects while Reduced Motion is enabled. Support contrast and reduce-transparency modes.

## 7. GPUI Kit handoff

Use `gpui-kit` and its component/base layers for layouts, controls, focus, menus, and scrolling when they satisfy the visual spec. Build custom chat bubble/timeline renderer where necessary. Follow https://gpui-kit.com/docs/ for the installed version. `src/design_tokens.rs` is intentionally **framework-agnostic Rust token data** (no unverified GPUI API calls). Integrate by mapping token values into GPUI styles with verified APIs from your pinned version.

Do not confuse a visual mock with working WhatsApp connectivity. Messaging, login pairing, and calling require independent service/protocol evaluation.

## 8. Review gate before wireframes

- Verify shell dimensions at 1512×982 1x and 3024×1964 2x render exports.
- Light/dark show the **same** content and component measurements.
- Audit sidebar layout versus 2026 Mac actual screenshots; mark deviations as reconstruction.
- Inspect header, tab, message row, composer and interactions at 100% with no synthetic zoom.
- Do not release 120-state rendering until this gate passes.

## 9. Reference provenance

- WABetaInfo, 2026-07-18, WhatsApp Mac Liquid Glass rollout: https://wabetainfo.com/whatsapp-is-rolling-out-liquid-glass-design-for-the-mac-app/
- 9to5Mac, 2026-07-20, Liquid Glass for Mac: https://9to5mac.com/2026/07/20/whatsapp-rolling-out-liquid-glass-revamp-to-its-mac-app/
- GPUI Kit official documentation: https://gpui-kit.com/docs/

**Source level:** macro UI direction and technology details are supported; exact token values, dimensions, and custom screen compositions are deliberate project decisions requiring visual audit.
