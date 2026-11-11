---
name: WhatsLite Desktop Design System
description: WhatsApp-inspired native macOS desktop messenger with Liquid Glass aesthetics
typography:
  family: "SF Pro Text, SF Pro Display, system-ui, -apple-system, sans-serif"
  sizes:
    caption: "11px"
    small: "12px"
    body: "14px"
    body-large: "15px"
    section: "17px"
    heading: "20px"
    page-title: "24px"
rounded:
  xs: "5px"
  sm: "8px"
  md: "12px"
  lg: "16px"
  pill: "999px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "16px"
  lg: "24px"
  xl: "32px"
---

# Design System — WhatsLite Desktop (macOS & Windows)

## Overview
WhatsLite Desktop is an independent, WhatsApp-inspired desktop messenger frontend built with Rust and GPUI Kit.
The design language reflects modern macOS Tahoe aesthetics, Liquid Glass materials, and WhatsApp desktop functional ergonomics.
The visual system strictly avoids generic web dashboard habits, unstyled HTML buttons, and purple/blue AI-slop gradients.

## Colors
The 28 light/dark color tokens live in `reference/design/machine/theme.json`
and are mirrored in `reference/design/machine/tokens.json`. Do not duplicate
hex values here; update the machine-readable source and the Pencil page 01
foundation together.

## Typography
- **Font Stack:** SF Pro Text / SF Pro Display on macOS; Segoe UI on Windows; Arial/system-ui fallback.
- **Hierarchy:**
  - Page Titles: 24pt bold
  - Headings / Chat Titles: 20pt semibold
  - Section Headers: 17pt medium/semibold
  - Body (Messages, Details): 14pt regular (line-height 1.35)
  - Small / Timestamps / Status: 12pt regular
  - Captions / Badges: 11pt medium

## Layout
- **Reference Canvas:** 1512 × 982 logical points at 100% scale.
- **macOS System Area:** Top 25pt reserved for system menu bar; application content starts at y=25, height 957pt.
- **Three-Column Shell Structure:**
  - Navigation Rail: 208pt (min 184pt, max 240pt; collapses to 72pt icon-only at <1170pt width)
  - Chat / Master List: 360pt (min 300pt, max 440pt; collapses to 312pt at <1170pt width)
  - Conversation / Detail Area: Flexible (min 440pt, expands to fill remaining width; 944pt at reference 1512pt)
- **Minimum Window Constraint:** 1024 × 700pt. At the minimum width, collapse the navigation rail to 72pt and chat list to 312pt; keep the conversation panel at least 440pt wide.
- **Vertical Rhythm:**
  - Header height: 72pt
  - Chat filter row: 52pt
  - Chat row height: 76pt
  - Composer minimum height: 72pt (expandable to 180pt multiline)

## Elevation & Depth
- Flat, clean desktop elevation.
- Modals, popovers, and context menus use subtle drop shadows (`0 8px 24px rgba(0,0,0,0.12)` in light mode, `0 8px 24px rgba(0,0,0,0.4)` in dark mode).
- Glass materials used only where host compositor allows blur/vibrancy; fallback to opaque theme colors.

## Shapes
- Window Corner Radius: 12pt (on macOS window container)
- Modals & Cards: 12pt–16pt rounded corners
- Avatar Circles: Fully rounded (`rounded_full`)
- Chat Bubbles: 8pt–12pt rounded corners with directional tail styling
- Filter Pills & Badges: Pill radius (999pt)
- Buttons: 6pt–8pt rounded corners

## Components
- **NavigationRail:** Vertical column holding primary icons (Chats, Calls, Updates, Communities, Settings), badges, and profile avatar.
- **ChatListRow:** 76pt height with 48pt avatar, unread indicator, contact title, timestamp, message preview, and pinned badge.
- **MessageBubble:** Incoming (aligned left, white/dark gray) and Outgoing (aligned right, soft green/deep emerald) with timestamp, delivery ticks, and hover reaction affordances.
- **MessageComposer:** Integrated 72pt bottom bar with attachment trigger, multiline text input, emoji selector, voice recorder, and send button.
- **CallSurface:** Audio/video call layout with participant avatars, video grid, and floating call control toolbar.

## Do's and Don'ts
- **DO:**
  - Use exact semantic colors from design tokens for all view components.
  - Test all components across both Light and Dark modes.
  - Implement visible keyboard focus and accessibility labels on all icon-only buttons.
  - Preserve the 1512×982 reference layout geometry and 1024×700 minimum bounds.
- **DON'T:**
  - Hardcode raw hex colors in UI component code.
  - Introduce generic web/HTML design tropes (purple gradients, generic dashboard cards).
  - Attempt real WhatsApp network calls or real backend authentication.
  - Break three-column desktop ergonomics on full-size screens.
