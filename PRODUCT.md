# Product

<!-- impeccable:product-schema 1 -->

## Platform

adaptive

## Users
- Desktop users on macOS and Windows seeking a fast, lightweight, native desktop messaging experience.
- Developers and evaluators building and testing WhatsApp-inspired desktop messaging workflows, interaction states, and GPUI UI architecture.

## Product Purpose
An independent, WhatsApp-inspired desktop messenger frontend built with Rust and GPUI Kit.
Provides a responsive desktop messaging experience across 15 functional modules and 120 discoverable interaction states using deterministic fixtures and typed service interfaces.
Success means fluid navigation, fast rendering, tokenized Light and Dark themes, keyboard accessibility, and adherence to layout contracts without backend network dependencies.

## Positioning
Unlike Electron- or browser-based wrappers, this is an ultra-fast, GPU-accelerated native desktop messenger frontend written in Rust + GPUI Kit. It provides complete interaction fidelity across 15 core messaging domains (chat, media, calls, contacts, groups, communities, settings, notifications) driven entirely by deterministic service contracts and mock fixtures.

## Operating Context
- Desktop window environment on macOS (maximized application window below OS menu bar at 1512×982 logical points) and Windows (Visual Studio C++ toolchain / GPUI Kit).
- Three-column desktop layout: Navigation rail (64–208pt), Chat / List pane (360pt), Conversation / Detail flexible workspace.
- Dark and Light appearance modes toggled via settings or system theme.
- Full keyboard navigation, shortcuts, and native desktop window ergonomics.

## Capabilities and Constraints
- Frontend only: strictly mock data and typed service interfaces. No live WhatsApp Web connection, no real authentication or QR pairing, no background network calls to WhatsApp servers.
- 15 functional modules: Onboarding/Pairing, Main Window Shell, Chat List, Conversation View, Message Composer, Attachments, Group Chats, Contacts & Profiles, Calls, Media Detail, Special Chats (Locked/Archived), Updates & Communities, Settings, Notifications, System & Error handling.
- 120 unique registered interaction states specified in `specifications/screen-map.json`.
- Canvas dimensions: 1512×982 logical points; wireframe SVGs are structural layout contracts, not pixel-perfect visual art.
- Reusable component architecture: composable GPUI views rather than 120 rigid page components.
- Zero hardcoded theme colors in UI implementation; all styling must use tokenized definitions (`design-system/design-tokens.json`).

## Brand Commitments
- Independent product inspired by WhatsApp; not an official WhatsApp or Meta product.
- Name: WhatsLite Desktop / WhatsApp-inspired macOS Desktop Messenger.
- Neutral, modern messenger identity respecting macOS desktop interface aesthetics and WhatsApp-inspired functional ergonomics.

## Evidence on Hand
- Screen and component specs: `reference/design/01-SCREENS.md`, `reference/design/02-COMPONENTS.md`, and `reference/design/design/whatslite.pen`.
- Interaction and state rules: `reference/design/03-BEHAVIOUR-RULES.md` and `reference/design/machine/screens.json`.
- Design token system: `reference/design/machine/theme.json` and `reference/design/machine/tokens.json`.
- GPUI architecture and service contracts: `crates/whatslite-services/src/lib.rs` and `desktop/src/`.
- Desktop application code: `desktop/src/`; UI uses deterministic mock data and does not connect to WhatsApp.

## Product Principles
- **Native Speed & Responsiveness:** Leverage GPUI's GPU-accelerated rendering and Rust's low overhead for instantaneous navigation, fluid list scrolling, and zero lag.
- **Contract-Driven Mock Architecture:** Decouple all presentation from backend state via strictly typed service interfaces and deterministic fixtures.
- **Token-Bound Visual Consistency:** Theme colors, typography, radii, and spacing derive exclusively from the design system tokens across Light and Dark themes.
- **Desktop Ergonomics:** Prioritize scanability, keyboard shortcuts, multi-column navigation, and window resizing behavior expected of first-class desktop software.

## Accessibility & Inclusion
- Support both high-contrast Light and Dark themes via semantic design tokens.
- Native keyboard navigation and focus management across sidebar, chat list, message list, and composer.
- Accessible interactive affordances, semantic labeling, and clear typography hierarchy.
