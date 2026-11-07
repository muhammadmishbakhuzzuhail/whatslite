# Interaction states and quality gates

All interactive controls: default, hover, focus-visible, pressed, selected (where applicable), disabled, loading, error. Dialogs restore focus to trigger on close; Escape closes the topmost noncritical overlay. Notifications never steal focus. Every screen must support tab navigation and screen reader labels.

Messages: normal → hover → menu → reply/edit/forward/delete; optimistic send → pending → sent or failed → retry. Attachments: selection → preview → upload → progress → success / failure → retry. Calls: idle → outgoing/incoming → ringing → connected → ended / failed. Auth: welcome → QR or phone link → pending → sync → shell; expired/error → retry.

All visual checks in both themes: no clipping at 1512×982, minimum 1024×700, content scrolls correctly, modal has backdrop and focus trap, unread badge distinguishable from active selection, no inaccessible color-only status.
