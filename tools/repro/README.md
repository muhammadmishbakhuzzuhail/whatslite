# Reaction dead-zone repro (RESOLVED)

Standalone QML reproductions built to isolate the message-reaction rendering bug
(chips never appeared in the conversation; see the `qml-reactions-repeater-bug`
notes for the multi-session investigation).

`reaction-deadzone.qml` mirrors the app's structure (RowLayout rail+sidebar+
conversation, ColumnLayout header+wallpaper+composer, doodle Image + wash,
ListView with reuseItems+spacing+header, content ColumnLayout with hidden
children + AlignRight meta, ApplicationWindow, ListModel roles). It RENDERS the
chip correctly for both in/out — i.e. the bug does NOT reproduce here. That ruled
out every Qt primitive and the whole layout; the only thing the repro could not
replicate was the real C++ `JsonListModel`.

## Root cause (found)

`JsonListModel::data()` returned a BRAND-NEW `QVariantMap` on every call for the
`m` role (`m_items.at(row).toObject().toVariantMap()`). A `QVariantList` nested
inside a map that is re-created on every access is not reliably iterable by a
QML `Repeater` — `model.m.reactions` read as length-correct but produced 0
delegates, and chips simply never appeared.

## Fix

Cache the per-row `QVariantMap` (`m_cache`, rebuilt in `setItems`/`prepend`) and
return the STABLE instance from `data()`. With a stable map the nested
`reactions` QVariantList becomes a usable Repeater model and the existing inline
reaction Flow renders the chip at the bubble's bottom-left, no QML change needed.
This is also a perf win (no QVariantMap reconstruction per binding read).

Run the repro:
  xvfb-run -a -s "-screen 0 1100x740x24" env QT_QPA_PLATFORM=xcb qml6 tools/repro/reaction-deadzone.qml
  # writes /tmp/repro9.png
