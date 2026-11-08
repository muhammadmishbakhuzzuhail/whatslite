---
name: WhatsLite Desktop Design System
description: WhatsApp-inspired native macOS desktop messenger with Liquid Glass aesthetics
colors:
  light-canvas: "#F4F6F8"
  light-navigation: "#EFF2F5"
  light-list: "#FFFFFF"
  light-conversation: "#F7F2EA"
  light-header: "#FFFFFF"
  light-surface: "#FFFFFF"
  light-surface-hover: "#F1F3F5"
  light-surface-selected: "#E8F5EC"
  light-divider: "#DDE2E6"
  light-text-primary: "#182229"
  light-text-secondary: "#65727B"
  light-text-muted: "#849099"
  light-accent: "#17A765"
  light-accent-hover: "#118551"
  light-accent-soft: "#D8F5DF"
  light-bubble-in: "#FFFFFF"
  light-bubble-out: "#D8FBCB"
  light-input: "#F6F7F8"
  dark-canvas: "#11191D"
  dark-navigation: "#172126"
  dark-list: "#172126"
  dark-conversation: "#101A1E"
  dark-header: "#1D282D"
  dark-surface: "#243137"
  dark-surface-hover: "#293940"
  dark-surface-selected: "#253D35"
  dark-divider: "#344247"
  dark-text-primary: "#E9F0F2"
  dark-text-secondary: "#A7B6BC"
  dark-text-muted: "#7D929A"
  dark-accent: "#1DBB73"
  dark-accent-hover: "#2AC884"
  dark-accent-soft: "#194E39"
  dark-bubble-in: "#26363C"
  dark-bubble-out: "#075E48"
  dark-input: "#26343A"
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
Derived from `reference/design/machine/theme.json` and `tokens.json`:
- **Light Theme:**
  - Navigation Rail: `#EFF2F5`
  - Chat List: `#FFFFFF`
  - Conversation Workspace: `#F7F2EA` (warm tint)
  - Incoming Bubble: `#FFFFFF`
  - Outgoing Bubble: `#D8FBCB`
  - Brand Accent: `#17A765` (hover `#118551`, contrast `#075E54`)
  - Text Primary: `#182229`, Secondary: `#65727B`, Muted: `#849099`
  - Divider: `#DDE2E6`
- **Dark Theme:**
  - Navigation Rail: `#172126`
  - Chat List: `#172126`
  - Conversation Workspace: `#101A1E` (deep teal-black)
  - Incoming Bubble: `#26363C`
  - Outgoing Bubble: `#075E48`
  - Brand Accent: `#1DBB73` (hover `#2AC884`, contrast `#075E54`)
  - Text Primary: `#E9F0F2`, Secondary: `#A7B6BC`, Muted: `#7D929A`
  - Divider: `#344247`

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
