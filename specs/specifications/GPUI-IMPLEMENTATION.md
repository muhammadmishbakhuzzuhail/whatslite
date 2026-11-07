# GPUI Kit Implementation Specification

## Current API reference
- https://gpui-kit.com/docs/ (checked 2026-10-29). Official getting started documents `gpui-kit = "0.8.1"`, `application()`, `assets::Assets`, `init(cx)`, `open_window(...)`, and `Render`. Pin/test version rather than copy unpublished APIs.
- Use `gpui-kit` entry point; `gpui-base` for behaviors and state, `gpui-component` for styled controls when suitable. Do not add `gpui-shell` unless extensions are explicitly required.

## Architecture (minimal, no overengineering)
`AppModel` owns `Theme`, `Route`, `SessionState`, `NavigationState`, `ChatListState`, `ConversationState`, `ComposerState`, `CallState`, `SettingsState`.
`AppWindow` is root GPUI view. `NavigationRail`, `ChatListPanel`, `ConversationPanel`, `AppToolbar`, `DialogHost`, `ToastHost` are reusable views. Feature screens compose these. Keep domain models (Contact, Chat, Message, Attachment, Call) in `domain/` with stable IDs.

## Rendering and layout
Reference viewport 1512x982 pt. Menu bar 25 pt is OS-owned, not manually drawn in shipped app. Maximize window beneath menu bar. Three-pane shell: rail 208 pt (184–240), list 360 pt (300–440), conversation flex min 440. Root min 1024x700. At 1024 show adaptive behavior (hide rail labels or collapse chat list based on active route); preserve navigation/accessibility. Reuse tokens rather than CSS pixel guessing. Make vertical list scroll virtualized if data grows.

## Interaction model
Actions → state update → notification/invalidation → render. Typed actions: `Navigate(Route)`, `SelectChat(ChatId)`, `SetFilter(Filter)`, `OpenDialog(DialogKind)`, `ReplyTo(MessageId)`, `SendDraft`, `RetrySend(MessageId)`, `OpenAttachmentPicker`, `ToggleTheme`, `CallAction(CallCommand)`. Keep focus handles for search/composer/dialog and keyboard bindings (Cmd+K find, Cmd+N new chat, Esc dismiss, Enter send with configurable multiline shortcut).

## Ownership + services
Introduce trait boundaries (`AuthService`, `ChatService`, `ContactService`, `MediaService`, `CallService`, `SettingsStore`, `NotificationService`) backed by deterministic in-memory fakes. UI must never block on service work. Abstract asynchronous calls and surface pending/error states. Use stable IDs; do not generate random sample content per render. Fake QR only (visual pattern) and fake linking transitions.

## Components
NavigationRail, NavigationItem, ChatRow, FilterChip, SearchField, Avatar, PresenceBadge, ConversationHeader, MessageBubble, MessageActionMenu, Composer, ReplyPreview, AttachmentMenu, MediaGallery, CallSurface, SettingsSection, Modal, Toast, ProgressBanner. Reuse components by props/state; avoid independent screen-specific variants.

## Theme and styling freedom
AI agent may create modern polished high-fidelity styling, but must retain semantically named Light/Dark tokens and state styling (hover, pressed, selected, focus, disabled, error). Maintain 4/8 pt spacing rhythm, contrast, keyboard reachability, hover vs selected distinction, reduced-motion support. No brand logo copies necessary.

## Build steps / gates
1. Compile GPUI starter window with pinned dependency on target macOS.
2. Implement tokens/theme + shell with mock data; capture screenshot.
3. Build auth screens and actions.
4. Build chats, messages, composer, attachments.
5. Groups/contacts, calls, gallery.
6. Settings/updates/special chats and error states.
7. Validate all 120 state IDs through route/state test harness, screenshot visual regression at 1512x982 pt, and input accessibility.

## Honest limits
Specs are implementation-ready design guidance, not a compiled or tested Rust application. Some GPUI APIs evolve; verify latest signatures against pinned docs. macOS titlebar and OS notifications are system surfaces, not arbitrary SVG windows.
